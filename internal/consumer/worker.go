package consumer

import (
	"LogStream/internal/models"
	"context"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
)

func Process(ctx context.Context, msg kafka.Message) error {
	var entry models.Log
	if err := json.Unmarshal(msg.Value, &entry); err != nil {
		return err
	}

	log.Printf(
		"pushing log id=%s service=%s level=%s message=%q",
		entry.ID,
		entry.Service,
		entry.Level,
		entry.Message,
	)
	if elasticerr := index(ctx, entry); elasticerr != nil {
		return elasticerr
	}

	return nil
}
