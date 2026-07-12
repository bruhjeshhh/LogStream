package consumer

import (
	"LogStream/internal/models"
	"context"
	"database/sql"
	"fmt"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// PostgresStore is deliberately small so deployments can opt in to metadata
// persistence and tests can use an in-memory implementation. Implementations
// must make Write idempotent by log ID (for example, INSERT ... ON CONFLICT).
type PostgresStore interface {
	Write(context.Context, models.Log) error
}

var postgres PostgresStore

func SetPostgresStore(store PostgresStore) {
	postgres = store
}

func writePostgres(ctx context.Context, entry models.Log) error {
	if postgres == nil {
		return nil
	}
	return postgres.Write(ctx, entry)
}

// SQLPostgresStore persists lightweight metadata. The primary key and upsert
// make a Kafka redelivery update the existing row instead of inserting a copy.
type SQLPostgresStore struct{ db *sql.DB }

func InitPostgres(ctx context.Context) error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://logstream:logstream@localhost:5432/logstream?sslmode=disable"
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open postgres: %w", err)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("ping postgres: %w", err)
	}
	store := &SQLPostgresStore{db: db}
	if err := store.ensureSchema(ctx); err != nil {
		db.Close()
		return err
	}
	SetPostgresStore(store)
	return nil
}

func (s *SQLPostgresStore) ensureSchema(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS log_metadata (
        id UUID PRIMARY KEY,
        service TEXT NOT NULL,
        level TEXT NOT NULL,
        event_timestamp TIMESTAMPTZ NOT NULL,
        received_timestamp TIMESTAMPTZ NOT NULL
    )`)
	if err != nil {
		return fmt.Errorf("create log_metadata schema: %w", err)
	}
	return nil
}

func (s *SQLPostgresStore) Write(ctx context.Context, entry models.Log) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO log_metadata
        (id, service, level, event_timestamp, received_timestamp)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT (id) DO UPDATE SET
            service = EXCLUDED.service,
            level = EXCLUDED.level,
            event_timestamp = EXCLUDED.event_timestamp,
            received_timestamp = EXCLUDED.received_timestamp`,
		entry.ID, entry.Service, entry.Level, entry.EventTimestamp, entry.ReceivedTimestamp)
	if err != nil {
		return fmt.Errorf("postgres write: %w", err)
	}
	return nil
}

func (s *SQLPostgresStore) Close() error { return s.db.Close() }
