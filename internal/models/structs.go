package internal

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
