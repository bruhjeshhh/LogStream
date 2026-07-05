package service

import (
	"LogStream/internal/models"
	"log"
	"time"

	"github.com/google/uuid"
)

func Ingest(payload []models.Ingestion) {
	var validPayloads []models.Log

	for _, pld := range payload {
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
		validPayloads = append(validPayloads, vldpld)
	}
}
