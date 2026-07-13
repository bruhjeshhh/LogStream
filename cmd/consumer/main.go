package main

import (
	"LogStream/internal/consumer"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	if err := consumer.InitPostgres(ctx); err != nil {
		log.Fatalf("failed to initialize postgres: %v", err)
	}

	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		brokers = "localhost:9092"
	}
	topic := os.Getenv("KAFKA_TOPIC")
	if topic == "" {
		topic = "LogStream"
	}
	groupID := os.Getenv("KAFKA_GROUP_ID")
	if groupID == "" {
		groupID = "consumers-of-logstream"
	}
	reader := kafka.NewReader(kafka.ReaderConfig{Brokers: []string{brokers}, Topic: topic, GroupID: groupID})
	defer reader.Close()

	metricsAddr := os.Getenv("METRICS_ADDR")
	if metricsAddr == "" {
		metricsAddr = ":9090"
	}
	metricsMux := http.NewServeMux()
	metricsMux.HandleFunc("GET /metrics", consumer.MetricsHandler)
	metricsMux.HandleFunc("GET /healthz", consumer.MetricsHealthHandler)
	metricsServer := &http.Server{Addr: metricsAddr, Handler: metricsMux}
	go func() {
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("metrics server: %v", err)
		}
	}()
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				consumer.TrackLag(reader)
			}
		}
	}()
	defer metricsServer.Close()

	log.Println("consumer started")

	if err := consumer.Run(ctx, reader); err != nil && err != context.Canceled {
		log.Fatalf("consumer stopped with error: %v", err)
	}

	log.Println("consumer stopped")
}
