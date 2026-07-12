package tests

import (
	"LogStream/internal/buffer"
	"LogStream/internal/models"
	"LogStream/internal/service"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestIngest_ValidEntrySubmitted(t *testing.T) {
	drainIngestChan()

	payload := []models.Ingestion{
		{
			Service:  "test-svc",
			Level:    "info",
			Message:  "hello",
			Metadata: json.RawMessage(`{}`),
		},
	}

	service.Ingest(payload)

	select {
	case log := <-buffer.IngestChan:
		if log.Service != "test-svc" {
			t.Errorf("Service = %q, want %q", log.Service, "test-svc")
		}
		if log.Level != "info" {
			t.Errorf("Level = %q, want %q", log.Level, "info")
		}
		if log.Message != "hello" {
			t.Errorf("Message = %q, want %q", log.Message, "hello")
		}
		if log.ID == uuid.Nil {
			t.Error("ID should not be nil UUID")
		}
		if log.ReceivedTimestamp.IsZero() {
			t.Error("ReceivedTimestamp should be set")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for log on IngestChan")
	}
}

func TestIngest_ValidEntryAllLevels(t *testing.T) {
	levels := []string{"trace", "debug", "info", "warn", "error", "fatal"}
	for _, level := range levels {
		drainIngestChan()
		service.Ingest([]models.Ingestion{{
			Service:  "svc",
			Level:    level,
			Message:  "msg",
			Metadata: json.RawMessage(`{}`),
		}})
		select {
		case log := <-buffer.IngestChan:
			if log.Level != level {
				t.Errorf("Level = %q, want %q", log.Level, level)
			}
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for log with level %q", level)
		}
	}
}

func TestIngest_LevelCaseInsensitive(t *testing.T) {
	drainIngestChan()
	service.Ingest([]models.Ingestion{{
		Service:  "svc",
		Level:    "INFO",
		Message:  "msg",
		Metadata: json.RawMessage(`{}`),
	}})
	select {
	case log := <-buffer.IngestChan:
		if log.Level != "INFO" {
			t.Errorf("Level = %q, want %q", log.Level, "INFO")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for log")
	}
}

func TestIngest_RejectsEmptyService(t *testing.T) {
	drainIngestChan()
	service.Ingest([]models.Ingestion{{
		Service:  "",
		Level:    "info",
		Message:  "msg",
		Metadata: json.RawMessage(`{}`),
	}})
	select {
	case <-buffer.IngestChan:
		t.Error("expected no log submitted for empty Service")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestIngest_RejectsEmptyLevel(t *testing.T) {
	drainIngestChan()
	service.Ingest([]models.Ingestion{{
		Service:  "svc",
		Level:    "",
		Message:  "msg",
		Metadata: json.RawMessage(`{}`),
	}})
	select {
	case <-buffer.IngestChan:
		t.Error("expected no log submitted for empty Level")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestIngest_RejectsEmptyMessage(t *testing.T) {
	drainIngestChan()
	service.Ingest([]models.Ingestion{{
		Service:  "svc",
		Level:    "info",
		Message:  "",
		Metadata: json.RawMessage(`{}`),
	}})
	select {
	case <-buffer.IngestChan:
		t.Error("expected no log submitted for empty Message")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestIngest_RejectsInvalidLevel(t *testing.T) {
	drainIngestChan()
	service.Ingest([]models.Ingestion{{
		Service:  "svc",
		Level:    "critical",
		Message:  "msg",
		Metadata: json.RawMessage(`{}`),
	}})
	select {
	case <-buffer.IngestChan:
		t.Error("expected no log submitted for invalid level")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestIngest_RejectsNonObjectMetadata(t *testing.T) {
	drainIngestChan()
	service.Ingest([]models.Ingestion{{
		Service:  "svc",
		Level:    "info",
		Message:  "msg",
		Metadata: json.RawMessage(`"string"`),
	}})
	select {
	case <-buffer.IngestChan:
		t.Error("expected no log submitted for non-object Metadata")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestIngest_RejectsArrayMetadata(t *testing.T) {
	drainIngestChan()
	service.Ingest([]models.Ingestion{{
		Service:  "svc",
		Level:    "info",
		Message:  "msg",
		Metadata: json.RawMessage(`[1,2,3]`),
	}})
	select {
	case <-buffer.IngestChan:
		t.Error("expected no log submitted for array Metadata")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestIngest_NilMetadataRejected(t *testing.T) {
	drainIngestChan()
	service.Ingest([]models.Ingestion{{
		Service: "svc",
		Level:   "info",
		Message: "msg",
	}})
	select {
	case <-buffer.IngestChan:
		t.Error("expected no log submitted for nil Metadata")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestIngest_GeneratedUUIDIsNotNil(t *testing.T) {
	drainIngestChan()
	service.Ingest([]models.Ingestion{{
		Service:  "svc",
		Level:    "info",
		Message:  "msg",
		Metadata: json.RawMessage(`{}`),
	}})
	select {
	case log := <-buffer.IngestChan:
		if log.ID == uuid.Nil {
			t.Error("UUID should not be nil")
		}
		if log.ID[6]>>4 != 7 {
			t.Errorf("expected UUIDv7, got version %d", log.ID[6]>>4)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for log")
	}
}

func TestIngest_ReceivedTimestampIsRecent(t *testing.T) {
	drainIngestChan()
	before := time.Now()
	service.Ingest([]models.Ingestion{{
		Service:  "svc",
		Level:    "info",
		Message:  "msg",
		Metadata: json.RawMessage(`{}`),
	}})
	after := time.Now()
	select {
	case log := <-buffer.IngestChan:
		if log.ReceivedTimestamp.Before(before) || log.ReceivedTimestamp.After(after) {
			t.Errorf("ReceivedTimestamp %v should be between %v and %v", log.ReceivedTimestamp, before, after)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for log")
	}
}

func TestIngest_MultipleValidEntries(t *testing.T) {
	drainIngestChan()
	count := 5
	payload := make([]models.Ingestion, count)
	for i := 0; i < count; i++ {
		payload[i] = models.Ingestion{
			Service:  "svc",
			Level:    "info",
			Message:  "msg",
			Metadata: json.RawMessage(`{}`),
		}
	}

	service.Ingest(payload)

	for i := 0; i < count; i++ {
		select {
		case <-buffer.IngestChan:
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for log %d", i)
		}
	}

	select {
	case <-buffer.IngestChan:
		t.Errorf("expected exactly %d logs, got more", count)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestIngest_MixedValidAndInvalid(t *testing.T) {
	drainIngestChan()
	payload := []models.Ingestion{
		{Service: "svc", Level: "info", Message: "valid1", Metadata: json.RawMessage(`{}`)},
		{Service: "", Level: "info", Message: "invalid", Metadata: json.RawMessage(`{}`)},
		{Service: "svc", Level: "info", Message: "valid2", Metadata: json.RawMessage(`{}`)},
	}

	service.Ingest(payload)

	select {
	case log := <-buffer.IngestChan:
		if log.Message != "valid1" {
			t.Errorf("first valid = %q, want %q", log.Message, "valid1")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for first valid log")
	}

	select {
	case log := <-buffer.IngestChan:
		if log.Message != "valid2" {
			t.Errorf("second valid = %q, want %q", log.Message, "valid2")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for second valid log")
	}

	select {
	case <-buffer.IngestChan:
		t.Error("expected no more logs")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestIngest_ForwardedTimestamp(t *testing.T) {
	drainIngestChan()
	ts := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	service.Ingest([]models.Ingestion{{
		Service:   "svc",
		Level:     "info",
		Message:   "msg",
		Timestamp: ts,
		Metadata:  json.RawMessage(`{}`),
	}})
	select {
	case log := <-buffer.IngestChan:
		if !log.EventTimestamp.Equal(ts) {
			t.Errorf("EventTimestamp = %v, want %v", log.EventTimestamp, ts)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for log")
	}
}

func TestIngest_EmptyPayload(t *testing.T) {
	drainIngestChan()
	service.Ingest([]models.Ingestion{})
	select {
	case <-buffer.IngestChan:
		t.Error("expected no logs for empty payload")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestIngest_AllInvalid(t *testing.T) {
	drainIngestChan()
	service.Ingest([]models.Ingestion{
		{Service: "", Level: "info", Message: "msg", Metadata: json.RawMessage(`{}`)},
		{Service: "svc", Level: "", Message: "msg", Metadata: json.RawMessage(`{}`)},
		{Service: "svc", Level: "info", Message: "", Metadata: json.RawMessage(`{}`)},
	})
	select {
	case <-buffer.IngestChan:
		t.Error("expected no logs when all entries are invalid")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestIngest_MetadataForwarded(t *testing.T) {
	drainIngestChan()
	meta := json.RawMessage(`{"key":"value","n":42}`)
	service.Ingest([]models.Ingestion{{
		Service:  "svc",
		Level:    "info",
		Message:  "msg",
		Metadata: meta,
	}})
	select {
	case log := <-buffer.IngestChan:
		if string(log.Metadata) != string(meta) {
			t.Errorf("Metadata = %s, want %s", log.Metadata, meta)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for log")
	}
}
