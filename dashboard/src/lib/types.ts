export interface IngestionMetrics {
  requests_total: number;
  accepted_total: number;
  rejected_total: number;
  buffer_fill: number;
  buffer_capacity: number;
  p50_latency_ms: number;
  p99_latency_ms: number;
}

export interface ElasticsearchHealth {
  status: string;
  index: string;
  docs_total: number;
}

export interface ConsumerMetrics {
  processed_total: number;
  failed_total: number;
  in_flight: number;
  lag_messages: number;
  retrying_total: number;
  elasticsearch: ElasticsearchHealth;
}

/** Merged snapshot served by the consumer hub's /api/metrics and stream. */
export interface MetricsSnapshot {
  ts: number;
  ingestion: IngestionMetrics;
  consumer: ConsumerMetrics;
}

export type DLQState = "retrying" | "dead" | "flushed";

export interface DLQEntry {
  log_id: string;
  service: string;
  level: string;
  message_preview: string;
  retry_count: number;
  max_attempts: number;
  state: DLQState;
  next_retry_at: string | null;
  last_error: string;
  first_failed_at: string;
  last_attempt_at: string;
}

export interface LogEntry {
  id: string;
  service: string;
  level: string;
  message: string;
  timestamp: string;
  received_at: string;
}
