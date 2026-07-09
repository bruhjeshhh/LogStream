package consumer

import (
	"context"
	"log"

	"github.com/segmentio/kafka-go"
)

func Run(ctx context.Context, reader *kafka.Reader) error {
	for {
		msg, err := reader.FetchMessage(ctx)
		if err != nil {
			return err
		}

		if err := Process(ctx, msg); err != nil {
			log.Printf("failed to process message offset=%d: %v", msg.Offset, err)
			continue
		}

		if err := reader.CommitMessages(ctx, msg); err != nil {
			return err
		}
	}
}
