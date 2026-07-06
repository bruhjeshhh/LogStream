# LogStream — Build Roadmap

A distributed log ingestion and search engine, built with Go, Kafka, PostgreSQL, Elasticsearch, and Kubernetes.

**How to use this doc:** Work phase by phase. Each phase has a "Definition of Done" — don't move to the next phase until you hit it. If you're stuck on something, come back with the specific phase/step and what's breaking, not "it doesn't work."

---

## Phase 0 — Setup & Scope Lock

### 0.1 Environment prep
- Install Go (1.22+), Docker + Docker Compose, `kubectl`, `minikube` or `kind`
- Install a Kafka client CLI (`kcat`/`kafkacat`) for debugging topics from the terminal without writing code
- Set up a GitHub repo now, even empty. Commit early, commit often — you want commit history that shows iteration, not one giant "initial commit."

### 0.2 Write the one-pager
Create `DESIGN.md` in your repo with:
- One paragraph: what problem LogStream solves
- Bullet list: what's in scope (the Phase 1-9 stuff below)
- Bullet list: what's explicitly NOT in scope (multi-region, auth/multi-tenancy, custom wire protocols, log-to-metrics correlation)
- A rough architecture diagram (boxes and arrows is fine — draw.io, excalidraw, or even hand-drawn photographed)

**Definition of done:** DESIGN.md committed. You can explain the whole system in 3 sentences without looking at notes.

---

## Phase 1 — Kafka Fundamentals (hands-on, not theoretical)

### 1.1 Get Kafka running locally
- Use a standard Kafka + KRaft (no separate Zookeeper needed in modern Kafka) `docker-compose.yml` — don't hand-roll the config
- Bring it up, confirm the broker is reachable on `localhost:9092`

### 1.2 Manual topic exploration
- Create a topic manually via CLI: `kafka-topics --create --topic test-logs --partitions 3`
- Use `kcat` to produce a message manually, then consume it manually. Watch it work before you write any Go.

### 1.3 Hello-world producer (Go)
- Use `segmentio/kafka-go` or `confluent-kafka-go` (segmentio is easier to start with, pure Go, no cgo)
- Write a ~20 line producer that pushes a hardcoded string to `test-logs` on a loop

### 1.4 Hello-world consumer (Go)
- Write a ~20 line consumer that reads from `test-logs` and prints to stdout
- Run producer and consumer side by side, confirm messages flow

### 1.5 Understand partitions and consumer groups experientially
- Run 2 consumers in the same consumer group, watch how partitions split between them (add print statements showing which partition each consumer reads)
- Kill one consumer mid-stream, watch the other pick up its partitions (this is the "rebalance" — you don't need to explain the algorithm, just observe the behavior)

**Definition of done:** You can explain, in your own words, what a topic, partition, consumer group, and offset are — using what you *observed*, not a textbook definition. Producer and consumer both work locally.

---

## Phase 2 — Ingestion Service

### 2.1 Define the log schema
- Decide the JSON shape early: `{"service": string, "level": string, "timestamp": string, "message": string, "metadata": object}`
- Write a Go struct for it, put it in a shared `internal/models` package

### 2.2 Build the HTTP ingestion endpoint
- Single route: `POST /ingest` — accepts one log line or a batch (array) of log lines
- Validate the payload (required fields present, level is one of a known enum)
- On success, push each validated line onto the Kafka topic; return 202 Accepted

### 2.3 Batch and buffer (basic backpressure handling)
- Instead of one Kafka write per HTTP request, buffer incoming logs in memory and flush to Kafka every N ms or when batch size hits a threshold — reduces Kafka round trips
- Use a goroutine + channel for this; it's your first real concurrency pattern in the project

### 2.4 Local smoke test
- Send a batch of fake logs via `curl` or a small Go script
- Confirm they land in the Kafka topic using `kcat` to tail it

**Definition of done:** `POST /ingest` accepts logs and they're verifiably sitting in Kafka. No consumer yet — that's fine.

---

## Phase 3 — Processing Pipeline (Consumers)

### 3.1 Stand up Elasticsearch and Postgres locally
- Add both to your `docker-compose.yml`
- Confirm you can hit ES on `localhost:9200` and connect to Postgres with a client

### 3.2 Define the ES index mapping
- Design the index schema explicitly (don't let ES dynamic-map everything — you want control over field types, especially `timestamp` as a date type and `message` as text with a keyword sub-field for exact filtering)

### 3.3 Write the consumer service
- Separate binary/service from the ingestion service (this matters for Phase 5/6 — they scale independently)
- Consumer group reads from the Kafka topic
- For each message: parse → normalize timestamp → validate/extract log level if missing → write to ES

### 3.4 Write metadata to Postgres
- Track which service names exist, log counts per service, first/last seen timestamp
- This is a lightweight registry table, not the log bodies themselves

### 3.5 Scale test with multiple consumer instances
- Run 2-3 consumer instances locally, confirm partition distribution and that no logs are dropped or duplicated (check ES doc count against logs sent)

**Definition of done:** Logs sent to `/ingest` reliably show up in Elasticsearch, searchable by service/level/timestamp. Postgres has accurate metadata. Multiple consumers can run without data loss.

---

## Phase 4 — Query API

### 4.1 Build search endpoint
- `GET /search` with query params: `service`, `level`, `from`, `to`, `q` (free text)
- Translate params into an ES query (bool query with filters + a match query for `q`)

### 4.2 Pagination
- Support `page`/`size` or cursor-based pagination — don't return unbounded result sets

### 4.3 Basic aggregation endpoint (optional but a nice interview talking point)
- `GET /stats` — log count by service, by level, over a time range (ES aggregations)

**Definition of done:** You can curl `/search?service=auth-api&level=ERROR&q=timeout` and get back real, correct results from logs you ingested earlier.

---

## Phase 5 — Reliability

### 5.1 Dead-letter queue
- Malformed messages that fail parsing in the consumer go to a separate `logs-dlq` Kafka topic instead of crashing the consumer or being silently dropped

### 5.2 Retry with backoff
- If ES or Postgres write fails transiently, retry with exponential backoff before giving up and DLQ-ing the message

### 5.3 Graceful degradation test
- Manually stop the ES container while the consumer is running — confirm it doesn't crash, retries, and recovers when ES comes back
- Do the same for Postgres

### 5.4 Idempotency check
- Confirm that if a consumer crashes mid-batch and reprocesses messages after restart, you don't get wildly duplicated data (exactly-once is hard to guarantee fully — at-least-once with dedup on a log ID is a reasonable, honest target)

**Definition of done:** You can kill ES/Postgres mid-run, watch the system survive, and describe exactly what happens to messages during the outage.

---

## Phase 6 — Kubernetes

### 6.1 Local K8s basics warmup (separate from the project)
- Spin up `minikube` or `kind`
- Deploy a plain nginx Deployment + Service, `kubectl port-forward` to it, confirm you understand Pod → Deployment → Service before touching your own app

### 6.2 Containerize your services
- Dockerfile for ingestion service, Dockerfile for consumer service (multi-stage builds to keep images small)

### 6.3 Write K8s manifests
- Deployment + Service for ingestion
- Deployment (no Service needed, consumers don't receive inbound traffic) for consumer
- ConfigMap for shared config (Kafka broker address, ES address, etc.)
- Secret for Postgres credentials

### 6.4 Deploy Kafka, ES, Postgres to the cluster
- For a project like this, using existing Helm charts or simple StatefulSet manifests for these is fine — you don't need to hand-write StatefulSets from scratch to prove understanding, but read what's in them

### 6.5 End-to-end test in-cluster
- Port-forward the ingestion service, send logs, confirm they land in ES (also running in-cluster)

**Definition of done:** Entire pipeline runs inside your local K8s cluster, not just docker-compose. You can `kubectl get pods` and see everything healthy.

---

## Phase 7 — Autoscaling

### 7.1 Expose consumer lag as a metric
- Use a Kafka lag exporter, or compute lag yourself (latest offset − committed offset) and expose it on a `/metrics` endpoint

### 7.2 Set up HPA (Horizontal Pod Autoscaler)
- Configure HPA on the consumer Deployment using the lag metric (via Prometheus adapter) or, as a simpler fallback, CPU-based HPA if custom metrics setup is too heavy for the timeline

### 7.3 Trigger and observe a scale-up
- Push a burst of logs, watch consumer replicas increase, watch lag drop back down

**Definition of done:** You have a screen recording or screenshots showing replica count rising in response to load, and lag dropping as a result. This is one of your best interview stories — don't skip it.

---

## Phase 8 — Load Testing

### 8.1 Install and configure vegeta
- Write a target file hitting `/ingest` with realistic payloads

### 8.2 Baseline test
- Run vegeta at a fixed rate against a single ingestion + single consumer replica, record throughput, latency percentiles (p50/p95/p99), error rate

### 8.3 Scaled test
- Repeat with 3 consumer replicas, same load, compare results

### 8.4 Find the breaking point
- Increase load until something fails (ingestion latency spikes, consumer lag grows unbounded, ES rejects writes). Document exactly what broke first — this is more valuable than "it handled X req/s," because it shows you understand your own system's limits

**Definition of done:** A results table/graph: replicas vs throughput vs latency. This replaces every fabricated metric with real ones you can defend.

---

## Phase 9 — Observability (optional, do if time allows)

### 9.1 Prometheus metrics
- Expose ingestion rate, consumer lag, error rate, ES write latency as Prometheus metrics from your Go services

### 9.2 Grafana dashboard
- One dashboard, 4-5 panels max: ingestion rate, consumer lag over time, error rate, replica count

**Definition of done:** A dashboard screenshot that tells the story of a load test at a glance.

---

## Phase 10 — Docs & Demo Packaging

### 10.1 README
- Architecture diagram (from DESIGN.md, cleaned up)
- Setup instructions (docker-compose for local, then K8s instructions)
- Load test results front and center, with the graph/table from Phase 8

### 10.2 Demo artifact
- A short screen recording (2-3 min): ingest logs → search them → trigger a load spike → show autoscaling → show a graceful failure/recovery
- This single video is what you'll actually show in interviews when they ask "walk me through a project"

### 10.3 Explicitly write your "hardest problem" story
- Pick the one thing that actually broke and took real debugging (probably something in Phase 3, 5, or 7). Write 3-4 sentences on what broke, why, and how you fixed it. This is almost always the actual interview question, so have it ready in your own words.

**Definition of done:** Repo is public, README is the front door, video demo exists, and you have one debugging story memorized cold.

---

## Rough Time Budget

| Phase | Estimated time |
|---|---|
| 0 — Setup | 0.5 day |
| 1 — Kafka basics | 2 days |
| 2 — Ingestion | 2 days |
| 3 — Consumers | 3-4 days |
| 4 — Query API | 1-2 days |
| 5 — Reliability | 2 days |
| 6 — Kubernetes | 3-4 days |
| 7 — Autoscaling | 1-2 days |
| 8 — Load testing | 1-2 days |
| 9 — Observability | 1-2 days (optional) |
| 10 — Docs/demo | 1 day |

**Total: roughly 4-5 weeks at a steady pace, faster if you cut Phase 9.**

---

## When You Come Back With Doubts

Reference the phase and step number (e.g. "stuck on 3.3, consumer isn't committing offsets") rather than "Kafka isn't working" — makes it much faster to actually help.
