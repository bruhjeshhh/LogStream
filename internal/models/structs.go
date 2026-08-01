package models

import (
	"time"

	"encoding/json"

	"github.com/google/uuid"
)

type Ingestion struct {
	Service   string          `json:"service"`
	Level     string          `json:"level"`
	Message   string          `json:"message"`
	Timestamp time.Time       `json:"timestamp"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
}

type Log struct {
	ID                uuid.UUID       `json:"id"`
	Service           string          `json:"service"`
	Level             string          `json:"level"`
	Message           string          `json:"message"`
	EventTimestamp    time.Time       `json:"timestamp"`
	ReceivedTimestamp time.Time       `json:"receivedtimestamp"`
	Metadata          json.RawMessage `json:"metadata,omitempty"`
}

// The structs below back the dashboard observation API. They are the contract
// between the ingestion service, the consumer observation hub, and the web
// dashboard (see dashboard/README.md).

type IngestionMetrics struct {
	RequestsTotal  uint64  `json:"requests_total"`
	AcceptedTotal  uint64  `json:"accepted_total"`
	RejectedTotal  uint64  `json:"rejected_total"`
	BufferFill     int     `json:"buffer_fill"`
	BufferCapacity int     `json:"buffer_capacity"`
	P50LatencyMs   float64 `json:"p50_latency_ms"`
	P99LatencyMs   float64 `json:"p99_latency_ms"`
}

type ElasticsearchHealth struct {
	Status    string `json:"status"`
	Index     string `json:"index"`
	DocsTotal int64  `json:"docs_total"`
}

type ConsumerMetrics struct {
	ProcessedTotal uint64              `json:"processed_total"`
	FailedTotal    uint64              `json:"failed_total"`
	InFlight       int64               `json:"in_flight"`
	LagMessages    int64               `json:"lag_messages"`
	RetryingTotal  int                 `json:"retrying_total"`
	Elasticsearch  ElasticsearchHealth `json:"elasticsearch"`
}

// MetricsSnapshot is the merged view served by the consumer hub's
// GET /api/metrics and the "metrics" SSE event.
type MetricsSnapshot struct {
	TS        int64            `json:"ts"`
	Ingestion IngestionMetrics `json:"ingestion"`
	Consumer  ConsumerMetrics  `json:"consumer"`
}
