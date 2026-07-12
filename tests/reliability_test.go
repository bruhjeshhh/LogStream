package tests

import (
	"LogStream/internal/consumer"
	"context"
	"encoding/json"
	"testing"

	"github.com/segmentio/kafka-go"
)

type oneMessageReader struct {
	message   kafka.Message
	fetched   bool
	committed []kafka.Message
}

func (r *oneMessageReader) FetchMessage(context.Context) (kafka.Message, error) {
	if r.fetched {
		return kafka.Message{}, context.Canceled
	}
	r.fetched = true
	return r.message, nil
}

func (r *oneMessageReader) CommitMessages(_ context.Context, messages ...kafka.Message) error {
	r.committed = append(r.committed, messages...)
	return nil
}

type recordingWriter struct{ messages []kafka.Message }

func (w *recordingWriter) WriteMessages(_ context.Context, messages ...kafka.Message) error {
	w.messages = append(w.messages, messages...)
	return nil
}

func TestMalformedMessageIsDLQedBeforeCommit(t *testing.T) {
	reader := &oneMessageReader{message: kafka.Message{
		Topic: "LogStream", Partition: 2, Offset: 41, Key: []byte("api"), Value: []byte("not json"),
	}}
	dlq := &recordingWriter{}

	err := consumer.RunWithDependencies(context.Background(), reader, dlq, consumer.DefaultRetryPolicy)
	if err != context.Canceled {
		t.Fatalf("RunWithDependencies() error = %v, want context.Canceled", err)
	}
	if len(dlq.messages) != 1 {
		t.Fatalf("DLQ writes = %d, want 1", len(dlq.messages))
	}
	if len(reader.committed) != 1 {
		t.Fatalf("commits = %d, want 1 after the DLQ write", len(reader.committed))
	}

	var record consumer.DLQMessage
	if err := json.Unmarshal(dlq.messages[0].Value, &record); err != nil {
		t.Fatalf("decode DLQ record: %v", err)
	}
	if record.Topic != "LogStream" || record.Partition != 2 || record.Offset != 41 || string(record.Value) != "not json" {
		t.Errorf("DLQ record did not preserve source message: %#v", record)
	}
}
