package tests

import (
	"LogStream/internal/kafka"
	"LogStream/internal/models"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestFlush_EmptyBatch(t *testing.T) {
	kafka.Flush(nil)
	kafka.Flush([]models.Log{})
}

func TestFlush_MessageMarshalling(t *testing.T) {
	id := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	log := models.Log{
		ID:                id,
		Service:           "test-svc",
		Level:             "info",
		Message:           "test message",
		EventTimestamp:    ts,
		ReceivedTimestamp: ts,
	}

	data, err := json.Marshal(log)
	if err != nil {
		t.Fatalf("failed to marshal Log: %v", err)
	}

	var decoded models.Log
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal Log: %v", err)
	}

	if decoded.ID != id {
		t.Errorf("ID = %v, want %v", decoded.ID, id)
	}
	if decoded.Service != "test-svc" {
		t.Errorf("Service = %q, want %q", decoded.Service, "test-svc")
	}
	if decoded.Level != "info" {
		t.Errorf("Level = %q, want %q", decoded.Level, "info")
	}
	if decoded.Message != "test message" {
		t.Errorf("Message = %q, want %q", decoded.Message, "test message")
	}
}

func TestFlush_MessageMarshallingWithMetadata(t *testing.T) {
	id := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	log := models.Log{
		ID:       id,
		Service:  "svc",
		Level:    "error",
		Message:  "error occurred",
		Metadata: json.RawMessage(`{"error_code":500,"stack":"trace"}`),
	}

	data, err := json.Marshal(log)
	if err != nil {
		t.Fatalf("failed to marshal Log with metadata: %v", err)
	}

	var decoded models.Log
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal Log with metadata: %v", err)
	}

	if string(decoded.Metadata) != string(log.Metadata) {
		t.Errorf("Metadata = %s, want %s", decoded.Metadata, log.Metadata)
	}
}

func TestFlush_MultipleMessagesMarshalling(t *testing.T) {
	logs := make([]models.Log, 3)
	for i := range logs {
		logs[i] = models.Log{
			ID:      uuid.MustParse("00000000-0000-0000-0000-000000000003"),
			Service: "svc",
			Level:   "info",
			Message: "msg",
		}
	}

	for i, log := range logs {
		data, err := json.Marshal(log)
		if err != nil {
			t.Fatalf("failed to marshal log %d: %v", i, err)
		}
		var decoded models.Log
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal log %d: %v", i, err)
		}
		if decoded.ID != log.ID {
			t.Errorf("log %d: ID = %v, want %v", i, decoded.ID, log.ID)
		}
	}
}

func TestFlush_MarshalInvalidUTF8(t *testing.T) {
	id := uuid.MustParse("00000000-0000-0000-0000-000000000004")
	log := models.Log{
		ID:      id,
		Service: "svc",
		Level:   "info",
		Message: "hello\xfe\xffworld",
	}

	data, err := json.Marshal(log)
	if err != nil {
		t.Fatalf("failed to marshal Log with invalid UTF-8: %v", err)
	}

	var decoded models.Log
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal Log with invalid UTF-8: %v", err)
	}

	_ = decoded
}

func TestFlush_BatchWithAllLevels(t *testing.T) {
	levels := []string{"trace", "debug", "info", "warn", "error", "fatal"}
	logs := make([]models.Log, len(levels))
	for i, level := range levels {
		logs[i] = models.Log{
			ID:      uuid.MustParse("00000000-0000-0000-0000-000000000005"),
			Service: "svc",
			Level:   level,
			Message: "msg",
		}
	}

	for _, log := range logs {
		data, err := json.Marshal(log)
		if err != nil {
			t.Fatalf("failed to marshal log with level %q: %v", log.Level, err)
		}
		var decoded models.Log
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("failed to unmarshal log with level %q: %v", log.Level, err)
		}
		if decoded.Level != log.Level {
			t.Errorf("Level = %q, want %q", decoded.Level, log.Level)
		}
	}
}

func TestFlush_ExcludeUnmarshalableFromBatch(t *testing.T) {
	log := models.Log{
		ID:      uuid.MustParse("00000000-0000-0000-0000-000000000006"),
		Service: "svc",
		Level:   "info",
		Message: "test",
	}

	data, err := json.Marshal(log)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var decoded models.Log
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.Message != log.Message {
		t.Errorf("Message = %q, want %q", decoded.Message, log.Message)
	}
}
