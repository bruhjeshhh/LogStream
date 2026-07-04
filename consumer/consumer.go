package main

import (
	"context"
	"fmt"
	"log"
	_ "time"

	"github.com/segmentio/kafka-go"
)

func main() {
	// to consume messages
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{"localhost:9092"},
		Topic:    "test-logs",
		GroupID:  "logsteam-ke-majdoor",
		MaxBytes: 10e6, // 10MB
	})

	for {
		m, err := r.ReadMessage(context.Background())
		if err != nil {
			break
		}
		fmt.Printf("message at offset %d: %s = %s\n", m.Offset, string(m.Key), string(m.Value))
	}

	if err := r.Close(); err != nil {
		log.Fatal("failed to close reader:", err)
	}
}
