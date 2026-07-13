package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"os"
	"time"

	"github.com/segmentio/kafka-go"
)

const DLQTopic = "logs-dlq"

var ErrMalformedMessage = errors.New("malformed kafka message")

type messageReader interface {
	FetchMessage(context.Context) (kafka.Message, error)
	CommitMessages(context.Context, ...kafka.Message) error
}

type messageWriter interface {
	WriteMessages(context.Context, ...kafka.Message) error
}

type RetryPolicy struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
}

// MaxAttempts includes the initial write; the default therefore makes five
// retries after a failure.
var DefaultRetryPolicy = RetryPolicy{MaxAttempts: 6, InitialDelay: 100 * time.Millisecond, MaxDelay: 5 * time.Second}

// DLQMessage preserves the failed Kafka record together with enough context to
// diagnose or replay it later.
type DLQMessage struct {
	Topic     string    `json:"topic"`
	Partition int       `json:"partition"`
	Offset    int64     `json:"offset"`
	Key       []byte    `json:"key,omitempty"`
	Value     []byte    `json:"value"`
	Error     string    `json:"error"`
	FailedAt  time.Time `json:"failed_at"`
}

func Run(ctx context.Context, reader *kafka.Reader) error {
	broker := os.Getenv("KAFKA_BROKERS")
	if broker == "" {
		broker = "localhost:9092"
	}
	dlq := &kafka.Writer{Addr: kafka.TCP(broker), Topic: DLQTopic, Balancer: &kafka.LeastBytes{}}
	defer dlq.Close()
	return RunWithDependencies(ctx, reader, dlq, DefaultRetryPolicy)
}

// RunWithDependencies consumes records without losing failed messages. A record
// is committed only once its sinks succeed or its DLQ record is acknowledged.
func RunWithDependencies(ctx context.Context, reader messageReader, dlq messageWriter, policy RetryPolicy) error {
	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			return err
		}

		if err := processWithRetry(ctx, msg, policy); err != nil {
			metrics.failed.Add(1)
			log.Printf("processing failed offset=%d: %v; sending to DLQ", msg.Offset, err)
			if dlqErr := publishDLQ(ctx, dlq, msg, err); dlqErr != nil {
				// Do not commit: Kafka will redeliver this record after a restart.
				log.Printf("DLQ publish failed offset=%d: %v; leaving uncommitted", msg.Offset, dlqErr)
				// Stop rather than commit a later offset from the same partition,
				// which could otherwise skip this unprotected record.
				return dlqErr
			}
		}

		if err := reader.CommitMessages(ctx, msg); err != nil {
			return err
		}
	}
}

func processWithRetry(ctx context.Context, msg kafka.Message, policy RetryPolicy) error {
	if policy.MaxAttempts < 1 {
		policy.MaxAttempts = 1
	}
	if policy.InitialDelay <= 0 {
		policy.InitialDelay = 100 * time.Millisecond
	}
	if policy.MaxDelay <= 0 {
		policy.MaxDelay = 5 * time.Second
	}
	var err error
	for attempt := 1; attempt <= policy.MaxAttempts; attempt++ {
		err = Process(ctx, msg)
		if err == nil || errors.Is(err, ErrMalformedMessage) {
			return err
		}
		if attempt == policy.MaxAttempts {
			break
		}
		delay := policy.InitialDelay << (attempt - 1)
		if delay > policy.MaxDelay {
			delay = policy.MaxDelay
		}
		// Small jitter prevents a fleet of consumers retrying in lockstep.
		delay = delay/2 + time.Duration(rand.Int64N(int64(delay/2)+1))
		log.Printf("retrying offset=%d attempt=%d/%d in %s: %v", msg.Offset, attempt+1, policy.MaxAttempts, delay, err)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	return err
}

func publishDLQ(ctx context.Context, writer messageWriter, msg kafka.Message, cause error) error {
	payload, err := json.Marshal(DLQMessage{Topic: msg.Topic, Partition: msg.Partition, Offset: msg.Offset, Key: msg.Key, Value: msg.Value, Error: cause.Error(), FailedAt: time.Now().UTC()})
	if err != nil {
		return fmt.Errorf("marshal DLQ message: %w", err)
	}
	if err := writer.WriteMessages(ctx, kafka.Message{Key: msg.Key, Value: payload}); err != nil {
		return fmt.Errorf("write DLQ message: %w", err)
	}
	return nil
}
