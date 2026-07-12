package tests

import (
	"LogStream/internal/models"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestIngestionJSONRoundTrip(t *testing.T) {
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	ing := models.Ingestion{
		Service:   "test-service",
		Level:     "info",
		Message:   "test message",
		Timestamp: ts,
		Metadata:  json.RawMessage(`{"key":"value"}`),
	}

	data, err := json.Marshal(ing)
	if err != nil {
		t.Fatalf("failed to marshal Ingestion: %v", err)
	}

	var got models.Ingestion
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("failed to unmarshal Ingestion: %v", err)
	}

	if got.Service != ing.Service {
		t.Errorf("Service = %q, want %q", got.Service, ing.Service)
	}
	if got.Level != ing.Level {
		t.Errorf("Level = %q, want %q", got.Level, ing.Level)
	}
	if got.Message != ing.Message {
		t.Errorf("Message = %q, want %q", got.Message, ing.Message)
	}
	if !got.Timestamp.Equal(ing.Timestamp) {
		t.Errorf("Timestamp = %v, want %v", got.Timestamp, ing.Timestamp)
	}
	if string(got.Metadata) != string(ing.Metadata) {
		t.Errorf("Metadata = %s, want %s", got.Metadata, ing.Metadata)
	}
}

func TestIngestionMetadataOmitEmpty(t *testing.T) {
	ing := models.Ingestion{
		Service: "s",
		Level:   "info",
		Message: "m",
	}

	data, err := json.Marshal(ing)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var result map[string]json.RawMessage
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if _, ok := result["metadata"]; ok {
		t.Error("metadata field should be omitted when empty")
	}
}

func TestLogJSONRoundTrip(t *testing.T) {
	id := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	ts := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	log := models.Log{
		ID:                id,
		Service:           "svc",
		Level:             "warn",
		Message:           "msg",
		EventTimestamp:    ts,
		ReceivedTimestamp: ts,
		Metadata:          json.RawMessage(`{"env":"prod"}`),
	}

	data, err := json.Marshal(log)
	if err != nil {
		t.Fatalf("failed to marshal Log: %v", err)
	}

	var got models.Log
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("failed to unmarshal Log: %v", err)
	}

	if got.ID != log.ID {
		t.Errorf("ID = %v, want %v", got.ID, log.ID)
	}
	if got.Service != log.Service {
		t.Errorf("Service = %q, want %q", got.Service, log.Service)
	}
	if got.Level != log.Level {
		t.Errorf("Level = %q, want %q", got.Level, log.Level)
	}
	if got.Message != log.Message {
		t.Errorf("Message = %q, want %q", got.Message, log.Message)
	}
	if !got.EventTimestamp.Equal(log.EventTimestamp) {
		t.Errorf("EventTimestamp = %v, want %v", got.EventTimestamp, log.EventTimestamp)
	}
	if !got.ReceivedTimestamp.Equal(log.ReceivedTimestamp) {
		t.Errorf("ReceivedTimestamp = %v, want %v", got.ReceivedTimestamp, log.ReceivedTimestamp)
	}
	if string(got.Metadata) != string(log.Metadata) {
		t.Errorf("Metadata = %s, want %s", got.Metadata, log.Metadata)
	}
}

func TestLogMetadataOmitEmpty(t *testing.T) {
	log := models.Log{
		ID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		Service: "s",
		Level:   "info",
		Message: "m",
	}

	data, err := json.Marshal(log)
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	var result map[string]json.RawMessage
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if _, ok := result["metadata"]; ok {
		t.Error("metadata field should be omitted when empty")
	}
}

func TestIngestionZeroValue(t *testing.T) {
	var ing models.Ingestion
	if ing.Service != "" {
		t.Error("zero value Ingestion should have empty Service")
	}
	if ing.Metadata != nil {
		t.Error("zero value Ingestion should have nil Metadata")
	}
}

func TestLogZeroValue(t *testing.T) {
	var log models.Log
	if log.ID != uuid.Nil {
		t.Error("zero value Log should have nil UUID")
	}
}

func TestLogJSONFieldNames(t *testing.T) {
	id := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	log := models.Log{
		ID:                id,
		Service:           "api",
		Level:             "error",
		Message:           "something broke",
		EventTimestamp:    time.Now(),
		ReceivedTimestamp: time.Now(),
	}

	data, err := json.Marshal(log)
	if err != nil {
		t.Fatalf("failed to marshal Log: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	expectedKeys := []string{"id", "service", "level", "message", "timestamp", "receivedtimestamp"}
	for _, key := range expectedKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("expected field %q not found in JSON", key)
		}
	}
}

func TestIngestionJSONFieldNames(t *testing.T) {
	ing := models.Ingestion{
		Service:   "api",
		Level:     "info",
		Message:   "test",
		Timestamp: time.Now(),
		Metadata:  json.RawMessage(`{}`),
	}

	data, err := json.Marshal(ing)
	if err != nil {
		t.Fatalf("failed to marshal Ingestion: %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("failed to unmarshal to map: %v", err)
	}

	expectedKeys := []string{"service", "level", "message", "timestamp", "metadata"}
	for _, key := range expectedKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("expected field %q not found in JSON", key)
		}
	}
}
