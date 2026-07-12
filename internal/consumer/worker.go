package consumer

import (
	"LogStream/internal/models"
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/segmentio/kafka-go"
)

func Process(ctx context.Context, msg kafka.Message) error {
	var entry models.Log
	if err := json.Unmarshal(msg.Value, &entry); err != nil {
		return fmt.Errorf("%w: %v", ErrMalformedMessage, err)
	}

	log.Printf(
		"pushing log id=%s service=%s level=%s message=%q",
		entry.ID,
		entry.Service,
		entry.Level,
		entry.Message,
	)
	if err := index(ctx, entry); err != nil {
		return fmt.Errorf("elasticsearch write: %w", err)
	}

	if err := writePostgres(ctx, entry); err != nil {
		return fmt.Errorf("postgres write: %w", err)
	}

	return nil
}
