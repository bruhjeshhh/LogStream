package main

import (
	"LogStream/internal/consumer"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/segmentio/kafka-go"
)

func main() {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "LogStream",
		GroupID: "consumers-of-logstream",
	})
	defer reader.Close()

	worker := consumer.NewWorker()

	c := consumer.NewConsumer(reader, worker)

	log.Println("consumer started")

	if err := c.Run(ctx); err != nil && err != context.Canceled {
		log.Fatalf("consumer stopped with error: %v", err)
	}

	log.Println("consumer stopped")
}
