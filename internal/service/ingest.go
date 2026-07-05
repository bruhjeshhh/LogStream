package service

import (
	"LogStream/internal/buffer"
	"LogStream/internal/models"
	"encoding/json"
	"log"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
)

func Ingest(payload []models.Ingestion) {
	// var processedPayloads []models.Log
	var allowedLevels = []string{"trace", "debug", "info", "warn", "error", "fatal"}

	validPayloads := payload[:0]

	for _, pld := range payload {
		if pld.Level != "" && pld.Message != "" && pld.Service != "" && isJSONObject(pld.Metadata) && slices.Contains(allowedLevels, strings.ToLower(pld.Level)) {
			validPayloads = append(validPayloads, pld)
		}
	}

	for _, pld := range validPayloads {
		var vldpld models.Log
		id, err := uuid.NewV7()
		if err != nil {
			log.Fatalf("Failed to generate UUIDv7: %v", err)
		}
		vldpld.ID = id
		vldpld.Level = pld.Level
		vldpld.Service = pld.Service
		vldpld.EventTimestamp = pld.Timestamp
		vldpld.ReceivedTimestamp = time.Now()
		vldpld.Message = pld.Message
		vldpld.Metadata = pld.Metadata
		// processedPayloads = append(processedPayloads, vldpld)

		buffer.Submit(vldpld)
	}
}

func isJSONObject(data []byte) bool {
	var m map[string]any
	// json.Unmarshal returns nil error only if it fits the target type
	return json.Unmarshal(data, &m) == nil
}
