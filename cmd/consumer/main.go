package main

import (
	"context"
	"fmt"

	"github.com/segmentio/kafka-go"
)

const (
	broker = "localhost:9092"
	topic  = "LogStream"
)

var reader = kafka.NewReader(kafka.ReaderConfig{
	Brokers: []string{broker},
	Topic:   topic,
	GroupID: "consumers-of-logStream",
})

func main() {
	for {
		m, err := reader.ReadMessage(context.Background())
		if err != nil {
			break
		}
		fmt.Printf("message at offset %d: %s = %s\n", m.Offset, string(m.Key), string(m.Value)) //need to change this to maybe calling a func
	}

}
