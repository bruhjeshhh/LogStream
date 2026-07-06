package api

import (
	"LogStream/internal/buffer"
	"LogStream/internal/models"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func drainIngestChan() {
	for {
		select {
		case <-buffer.IngestChan:
		default:
			return
		}
	}
}

func TestDecodeIngestions_ValidPayload(t *testing.T) {
	drainIngestChan()

	body := []models.Ingestion{
		{
			Service:  "test-svc",
			Level:    "info",
			Message:  "test message",
			Metadata: json.RawMessage(`{}`),
		},
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(payload))
	rec := httptest.NewRecorder()

	DecodeIngestions(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	select {
	case log := <-buffer.IngestChan:
		if log.Service != "test-svc" {
			t.Errorf("Service = %q, want %q", log.Service, "test-svc")
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for log on IngestChan")
	}
}

func TestDecodeIngestions_WrongMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/ingest", nil)
	rec := httptest.NewRecorder()

	DecodeIngestions(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestDecodeIngestions_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader([]byte(`not json`)))
	rec := httptest.NewRecorder()

	DecodeIngestions(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestDecodeIngestions_EmptyBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader([]byte{}))
	rec := httptest.NewRecorder()

	DecodeIngestions(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestDecodeIngestions_NonArrayJSON(t *testing.T) {
	drainIngestChan()

	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader([]byte(`{"service":"svc"}`)))
	rec := httptest.NewRecorder()

	DecodeIngestions(rec, req)

	// Single object should fail because we decode into []Ingestion
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestDecodeIngestions_MultipleEntries(t *testing.T) {
	drainIngestChan()

	body := []models.Ingestion{
		{Service: "svc1", Level: "info", Message: "msg1", Metadata: json.RawMessage(`{}`)},
		{Service: "svc2", Level: "warn", Message: "msg2", Metadata: json.RawMessage(`{}`)},
	}
	payload, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader(payload))
	rec := httptest.NewRecorder()

	DecodeIngestions(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}

	for i := 0; i < 2; i++ {
		select {
		case <-buffer.IngestChan:
		case <-time.After(time.Second):
			t.Fatalf("timed out waiting for log %d", i)
		}
	}
}

func TestDecodeIngestions_ResponseBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader([]byte(`[]`)))
	rec := httptest.NewRecorder()

	DecodeIngestions(rec, req)

	body := strings.TrimSpace(rec.Body.String())
	if body != "" {
		t.Errorf("response body = %q, want empty", body)
	}
}

func TestDecodeIngestions_ContentType(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/ingest", bytes.NewReader([]byte(`[]`)))
	rec := httptest.NewRecorder()

	DecodeIngestions(rec, req)

	ct := rec.Header().Get("Content-Type")
	if ct != "" {
		t.Errorf("Content-Type = %q, want empty", ct)
	}
}
