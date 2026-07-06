package kafka

import (
	"LogStream/internal/models"
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	broker = "localhost:9092"
	topic  = "LogStream"
)

var writer = &kafka.Writer{
	Addr:     kafka.TCP(broker),
	Topic:    topic,
	Balancer: &kafka.LeastBytes{},
}

func Flush(batch []models.Log) {
	if len(batch) == 0 {
		return
	}

	messages := make([]kafka.Message, 0, len(batch))
	for _, entry := range batch {
		value, err := json.Marshal(entry)
		if err != nil {
			log.Printf("failed to marshal log %s: %v", entry.ID, err)
			continue
		}

		messages = append(messages, kafka.Message{
			Key:   []byte(entry.Service),
			Value: value,
		})
	}

	if len(messages) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := writer.WriteMessages(ctx, messages...); err != nil {
		log.Printf("failed to write logs to kafka: %v", err)
	}
}
