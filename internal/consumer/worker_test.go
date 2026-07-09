package consumer

import (
	"LogStream/internal/models"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
)

func TestProcess_ValidJSON(t *testing.T) {
	id := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	ts := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)

	payload := `{
		"id": "00000000-0000-0000-0000-000000000001",
		"service": "api-gateway",
		"level": "error",
		"message": "connection timeout",
		"timestamp": "2025-06-15T10:30:00Z",
		"receivedtimestamp": "2025-06-15T10:30:00Z"
	}`

	msg := kafka.Message{Value: []byte(payload)}
	err := Process(context.Background(), msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var entry models.Log
	if err := json.Unmarshal(msg.Value, &entry); err != nil {
		t.Fatalf("failed to unmarshal payload for verification: %v", err)
	}

	if entry.ID != id {
		t.Errorf("ID = %v, want %v", entry.ID, id)
	}
	if entry.Service != "api-gateway" {
		t.Errorf("Service = %q, want %q", entry.Service, "api-gateway")
	}
	if entry.Level != "error" {
		t.Errorf("Level = %q, want %q", entry.Level, "error")
	}
	if entry.Message != "connection timeout" {
		t.Errorf("Message = %q, want %q", entry.Message, "connection timeout")
	}
	if !entry.EventTimestamp.Equal(ts) {
		t.Errorf("EventTimestamp = %v, want %v", entry.EventTimestamp, ts)
	}
}

func TestProcess_InvalidJSON(t *testing.T) {
	tests := []struct {
		name    string
		payload string
	}{
		{"not json", "this is not json"},
		{"truncated json", `{"id": "abc"`},
		{"json array", `[1, 2, 3]`},
		{"json number", `42`},
		{"json string", `"hello"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := kafka.Message{Value: []byte(tt.payload)}
			err := Process(context.Background(), msg)
			if err == nil {
				t.Error("expected error for invalid JSON, got nil")
			}
		})
	}
}

func TestProcess_EmptyPayload(t *testing.T) {
	msg := kafka.Message{Value: []byte{}}
	err := Process(context.Background(), msg)
	if err == nil {
		t.Error("expected error for empty payload, got nil")
	}
}

func TestProcess_NilPayload(t *testing.T) {
	msg := kafka.Message{Value: nil}
	err := Process(context.Background(), msg)
	if err == nil {
		t.Error("expected error for nil payload, got nil")
	}
}

func TestProcess_MissingRequiredFields(t *testing.T) {
	payload := `{"service": "api"}`
	msg := kafka.Message{Value: []byte(payload)}
	err := Process(context.Background(), msg)

	if err != nil {
		t.Fatalf("unexpected error with metadata: %v", err)
	}
}

func TestProcess_ContextCancellation(t *testing.T) {
	payload := `{"id": "00000000-0000-0000-0000-000000000001"}`
	msg := kafka.Message{Value: []byte(payload)}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Process(ctx, msg)
	if err != nil {
		t.Fatalf("unexpected error with cancelled context: %v", err)
	}
}

func TestProcess_LargeMessage(t *testing.T) {
	bigMessage := make([]byte, 100000)
	for i := range bigMessage {
		bigMessage[i] = 'A'
	}

	payload := `{"id": "00000000-0000-0000-0000-000000000001", "message": "` + string(bigMessage) + `"}`
	msg := kafka.Message{Value: []byte(payload)}

	err := Process(context.Background(), msg)
	if err != nil {
		t.Fatalf("unexpected error with large message: %v", err)
	}
}

func TestProcess_SpecialCharactersInMessage(t *testing.T) {
	payload := `{
		"id": "00000000-0000-0000-0000-000000000001",
		"service": "svc",
		"level": "info",
		"message": "line1\nline2\ttab\"quotes\\backslash"
	}`

	msg := kafka.Message{Value: []byte(payload)}
	err := Process(context.Background(), msg)
	if err != nil {
		t.Fatalf("unexpected error with special characters: %v", err)
	}
}
