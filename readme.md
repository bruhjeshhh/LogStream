# LogStream

A distributed log ingestion and search system built with Go, Kafka, Elasticsearch, and PostgreSQL. Designed for centralized log aggregation across distributed services with independent scaling of ingestion and processing tiers.

## Architecture

```
┌─────────────┐     HTTP/JSON      ┌──────────────────┐     Kafka      ┌─────────────┐
│  Producers  │ ─────────────────▶ │  Ingestion API   │ ─────────────▶ │   Kafka     │
│  (services) │                    │  (POST /ingest)  │   Topic:       │  (durable   │
└─────────────┘                    └──────────────────┘   LogStream    │   buffer)   │
                                                                       └──────┬──────┘
                                                                              │
                           ┌──────────────────┐                               │
                           │   Consumer       │                               │
                           │  (consumer group) │ ◀────────────────────────────┘
                           └────────┬─────────┘
                                    │
                      ┌─────────────┴─────────────┐
                      ▼                           ▼
             ┌───────────────┐           ┌──────────────────┐
             │ Elasticsearch │           │  PostgreSQL      │
             │  (full-text   │           │  (metadata:      │
             │   search)     │           │   service names, │
             │               │           │    counts,       │
             └───────┬───────┘           │    time ranges)  │
                     │                   └──────────────────┘
                     ▼
            ┌──────────────────┐
            │  Search API      │
            │  (GET /search)   │
            └──────────────────┘
```

### Components

| Component | Port | Description |
|-----------|------|-------------|
| **Ingestion API** | 8080 | HTTP endpoint (`POST /ingest`) accepting JSON log batches. Validates, assigns UUIDv7 IDs, buffers in memory, flushes to Kafka every 5s or 100 messages. |
| **Consumer** | 9090 (metrics) | Kafka consumer group. Processes each message: parses → indexes to Elasticsearch → writes metadata to PostgreSQL. Implements at-least-once delivery with retry/backoff and dead-letter queue (`logs-dlq`). |
| **Search API** | 8084 | Query endpoint (`GET /search`) over Elasticsearch with filters: service, level, time range, free-text, pagination. |
| **Kafka** | 9092 | Durable message buffer (KRaft mode, no Zookeeper). Topics: `LogStream` (input), `logs-dlq` (dead letters). |
| **Elasticsearch** | 9200 | Full-text search and log storage. Index: `logs`. |
| **PostgreSQL** | 5432 | Lightweight metadata registry (service names, counts, first/last seen timestamps). |

## Data Model

### Ingestion Request (`POST /ingest`)
```json
[
  {
    "service": "auth-api",
    "level": "ERROR",
    "message": "connection timeout to upstream",
    "timestamp": "2026-07-14T10:30:00Z",
    "metadata": {
      "trace_id": "abc-123",
      "region": "us-east-1"
    }
  }
]
```

### Stored Log Document (Elasticsearch)
```json
{
  "id": "0192f3c8-7b4a-7c8d-9e0f-1a2b3c4d5e6f",
  "service": "auth-api",
  "level": "error",
  "message": "connection timeout to upstream",
  "timestamp": "2026-07-14T10:30:00Z",
  "receivedtimestamp": "2026-07-14T10:30:00.123Z",
  "metadata": { "trace_id": "abc-123", "region": "us-east-1" }
}
```

## Quick Start (Docker Compose)

### Prerequisites
- Docker & Docker Compose
- Go 1.22+ (for local development)

### Start Infrastructure
```bash
docker compose up -d
# Kafka (localhost:9092), Elasticsearch (localhost:9200), PostgreSQL (localhost:5432)
```

### Run Services Locally
```bash
# Terminal 1: Ingestion API
go run ./cmd/ingestion

# Terminal 2: Consumer (requires Kafka, ES, Postgres)
go run ./cmd/consumer

# Terminal 3: Search API
go run ./cmd/search
```

### Ingest Logs
```bash
curl -X POST http://localhost:8080/ingest \
  -H "Content-Type: application/json" \
  -d '[{"service":"auth-api","level":"ERROR","message":"timeout","timestamp":"2026-07-14T10:30:00Z","metadata":{"trace_id":"abc-123"}}]'
```

### Search Logs
```bash
# By service + level
curl "http://localhost:8084/search?service=auth-api&level=ERROR"

# Free-text search with time range
curl "http://localhost:8084/search?q=timeout&from=2026-07-14T00:00:00Z&to=2026-07-15T00:00:00Z"

# Pagination
curl "http://localhost:8084/search?page=2&size=50"
```

### Health Checks
```bash
curl http://localhost:8080/api/healthz   # Ingestion
curl http://localhost:9090/healthz      # Consumer
```

## Kubernetes Deployment

### Prerequisites
- Kubernetes cluster (minikube, kind, Docker Desktop, or cloud)
- `kubectl` configured
- Helm (for Kafka/ES/Postgres operators, or use the manifests in `k8s/base`)

### Deploy
```bash
# Apply all manifests via kustomize
kubectl apply -k k8s/base

# Or apply individually
kubectl apply -f k8s/base/namespace.yaml
kubectl apply -f k8s/base/configmap.yaml
kubectl apply -f k8s/base/secret.yaml
kubectl apply -f k8s/base/ingestion.yaml
kubectl apply -f k8s/base/consumer.yaml
kubectl apply -f k8s/base/search.yaml
kubectl apply -f k8s/base/consumer-hpa.yaml
```

### Access Services
```bash
# Ingestion API (port-forward)
kubectl port-forward -n logstream svc/ingestion 8080:8080

# Search API
kubectl port-forward -n logstream svc/search 8084:8084

# Consumer metrics
kubectl port-forward -n logstream deployment/consumer 9090:9090
curl http://localhost:9090/metrics
```

### Scale Consumers
```bash
# Manual scale
kubectl scale deployment/consumer -n logstream --replicas=3

# HPA (CPU-based fallback; lag-based requires Prometheus Adapter)
kubectl get hpa -n logstream -w
```

### Configuration
Environment variables are defined in `k8s/base/configmap.yaml`:
| Variable | Default | Description |
|----------|---------|-------------|
| `KAFKA_BROKERS` | `kafka:9092` | Kafka bootstrap servers |
| `KAFKA_TOPIC` | `LogStream` | Input topic |
| `KAFKA_GROUP_ID` | `consumers-of-logstream` | Consumer group |
| `ELASTICSEARCH_URL` | `http://elasticsearch-master:9200` | ES endpoint |
| `METRICS_ADDR` | `:9090` | Prometheus metrics address |

Secrets (`k8s/base/secret.yaml`): `DATABASE_URL` for PostgreSQL.

## Load Testing

Vegeta load tests are in `loadtest/`.

### Prerequisites
```bash
go install github.com/tsenart/vegeta/v12@latest
# or: winget install tsenart.vegeta
```

### Run Baseline Test
```bash
# Start ingestion (local or port-forwarded)
go run ./cmd/ingestion

# Run test
cd loadtest
./run.ps1 -Rate 25 -Duration 30s -Name baseline
```

### Results (Local Ingestion Only — No Kafka/ES/Postgres)
| Rate | Duration | Throughput | p50 | p95 | p99 | Errors |
|------|----------|------------|-----|-----|-----|--------|
| 25/s | 30s | 25.03/s | 0.52 ms | 1.38 ms | 2.75 ms | 0% |
| 200/s | 60s | 200.02/s | ~0–0.7 ms | 0.78 ms | 0.98 ms | 0% |
| 1000/s | 15s | 1000.09/s | ~0–0.1 ms | 0.55 ms | 0.60 ms | 0% |
| 5000/s | 15s | 5000.04/s | ~0–0.4 ms | 0.56 ms | 1.63 ms | 0% |

> **Note**: These numbers reflect HTTP accept + validate + in-memory buffer flush only. End-to-end pipeline latency (Kafka → consumer → ES/Postgres) and consumer lag under load are **not** measured here. Run against the full K8s stack to get production-relevant numbers.

### Finding the Breaking Point
```bash
# Increase rate until latency spikes or errors appear
./run.ps1 -Rate 10000 -Duration 30s -Name breaking-point
```

## Reliability Guarantees

| Property | Implementation |
|----------|----------------|
| **At-least-once delivery** | Kafka offset committed only after ES + Postgres writes succeed (or DLQ ack) |
| **Dead-letter queue** | Malformed JSON → immediate DLQ. Sink failures → retry (6 attempts, jittered exponential backoff: 100ms → 200ms → 400ms → 800ms → 1.6s → 5s cap) → DLQ |
| **Deduplication** | ES uses log UUID as document ID; Postgres uses `ON CONFLICT (log_id) DO UPDATE` |
| **Graceful degradation** | Consumer stays alive during ES/Postgres outages, retries per policy, recovers when dependencies return |
| **No data loss on Kafka failure** | If DLQ publish fails, original record is left uncommitted for redelivery after restart |

### Testing Failure Scenarios
```bash
# Stop ES while producing logs
docker stop logstream-es
# Observe consumer retry logs; restart ES before 5th retry to see recovery

# Force DLQ by keeping ES down past retry budget
docker stop logstream-es
# Wait >30s (6 retries × max 5s delay)
# Inspect DLQ
docker exec -it kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic logs-dlq --from-beginning
```

## Monitoring & Metrics

Consumer exposes Prometheus metrics on `:9090/metrics`:
- `logstream_consumer_processed_total` — successfully indexed records
- `logstream_consumer_failed_total` — records sent to DLQ
- `logstream_consumer_in_flight` — currently processing
- `logstream_consumer_lag_messages` — estimated Kafka lag (updated every 5s)

HPA in `k8s/base/consumer-hpa.yaml` uses CPU as a simple fallback; switch to lag-based scaling by configuring a Prometheus Adapter.

## Project Structure

```
LogStream/
├── cmd/
│   ├── ingestion/     # HTTP ingestion server (port 8080)
│   ├── consumer/      # Kafka consumer + metrics (port 9090)
│   └── search/        # Search API (port 8084)
├── internal/
│   ├── api/           # HTTP handlers (DecodeIngestions)
│   ├── buffer/        # In-memory channel buffer + periodic Kafka flush
│   ├── consumer/      # Consumer logic: process, retry, DLQ, ES/Postgres sinks
│   ├── kafka/         # Kafka producer (Flush function)
│   ├── models/        # Shared structs (Ingestion, Log)
│   ├── search/        # Elasticsearch query builder + repository
│   └── service/       # Ingestion validation + UUID assignment
├── health/            # /healthz endpoint
├── k8s/
│   └── base/          # Kustomize manifests
├── loadtest/          # Vegeta targets, runner, results
├── tests/             # Integration/unit tests
├── docker-compose.yml # Local Kafka, ES, Postgres
├── Dockerfile.ingestion
├── Dockerfile.consumer
├── Dockerfile.search
├── design.md          # Architecture & design doc
└── LogStream_Build_Roadmap.md  # Phased build plan
```

## Configuration Reference

| Service | Env Var | Default | Required |
|---------|---------|---------|----------|
| **Ingestion** | `KAFKA_BROKERS` | `localhost:9092` | Yes |
| | `KAFKA_TOPIC` | `LogStream` | Yes |
| | `ELASTICSEARCH_URL` | `http://localhost:9200` | Yes (for future) |
| | `METRICS_ADDR` | `:9090` | No |
| **Consumer** | `KAFKA_BROKERS` | `localhost:9092` | Yes |
| | `KAFKA_TOPIC` | `LogStream` | Yes |
| | `KAFKA_GROUP_ID` | `consumers-of-logstream` | Yes |
| | `ELASTICSEARCH_URL` | `http://localhost:9200` | Yes |
| | `DATABASE_URL` | — | Yes (Postgres) |
| | `METRICS_ADDR` | `:9090` | No |
| **Search** | `ELASTICSEARCH_URL` | `http://localhost:9200` | Yes |

## Development

### Run Tests
```bash
go test ./... -v
```

### Build Docker Images
```bash
docker build -t logstream-ingestion:latest -f Dockerfile.ingestion .
docker build -t logstream-consumer:latest -f Dockerfile.consumer .
docker build -t logstream-search:latest -f Dockerfile.search .
```

### Generate UUIDv7
Uses `github.com/google/uuid` for time-ordered, sortable IDs.

## License

MIT