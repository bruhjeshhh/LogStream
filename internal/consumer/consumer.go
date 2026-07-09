package consumer

import (
	"context"
	"log"

	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader *kafka.Reader
	worker *Worker
}

func NewConsumer(reader *kafka.Reader, worker *Worker) *Consumer {
	return &Consumer{
		reader: reader,
		worker: worker,
	}
}

func (c *Consumer) Run(ctx context.Context) error {
	for {
		msg, err := c.reader.FetchMessage(ctx)
		if err != nil {
			return err
		}

		if err := c.worker.Process(ctx, msg); err != nil {
			log.Printf("failed to process message at offset %d: %v", msg.Offset, err)
			continue
		}

		if err := c.reader.CommitMessages(ctx, msg); err != nil {
			return err
		}
	}
}
