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

	if err := consumer.InitElastic(); err != nil {
		log.Fatalf("failed to init elasticsearch: %v", err)
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{"localhost:9092"},
		Topic:   "LogStream",
		GroupID: "consumers-of-logstream",
	})
	defer reader.Close()

	log.Println("consumer started")

	if err := consumer.Run(ctx, reader); err != nil && err != context.Canceled {
		log.Fatalf("consumer stopped with error: %v", err)
	}

	log.Println("consumer stopped")
}
