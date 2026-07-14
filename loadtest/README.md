# Load testing with Vegeta

Install Vegeta first (`go install github.com/tsenart/vegeta/v12@latest`, or
`winget install tsenart.vegeta`). Start the ingestion service (Kubernetes
port-forward from `k8s/README.md`, or a local `go run ./cmd/ingestion`), then
run:

```powershell
.\loadtest\run.ps1 -Rate 25 -Duration 30s -Name single-replica
```

The target file uses `127.0.0.1` so Vegeta does not depend on DNS for
`localhost`. Request bodies come from `ingest-body.json`.

Scale the consumer and repeat the same request rate when running in Kubernetes:

```powershell
kubectl scale deployment/consumer -n logstream --replicas=3
.\loadtest\run.ps1 -Rate 25 -Duration 30s -Name three-consumers
```

Before increasing the rate, watch consumer lag and HPA status in separate
terminals:

```powershell
kubectl get hpa,pods -n logstream -w
kubectl port-forward -n logstream deployment/consumer 9090:9090
curl.exe http://localhost:9090/metrics
```

Increase `-Rate` gradually until latency grows, errors appear, or lag no longer
returns to zero. Reports are saved under `loadtest/results/`.

## Measured results (2026-07-14)

**Setup:** single local Go ingestion process (`cmd/ingestion` on Windows), Vegeta
hitting `POST /ingest`. Docker/kind were not available, so Kafka, Elasticsearch,
Postgres, and consumer replica scaling were **not** in the path. These numbers
measure HTTP accept + validate + in-memory buffer only — not end-to-end index
latency or consumer lag under load.

| Consumer replicas | Rate | Throughput | p50 | p95 | p99 | Error rate | Notes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| n/a (local ingest only) | 25/s · 30s | 25.03/s | 0.52 ms | 1.38 ms | 2.75 ms | 0% | Roadmap baseline target |
| n/a (local ingest only) | 200/s · 60s | 200.02/s | ~0–0.7 ms\* | 0.78 ms | 0.98 ms | 0% | Sustained soak |
| n/a (local ingest only) | 1000/s · 15s | 1000.09/s | ~0–0.1 ms\* | 0.55 ms | 0.60 ms | 0% | Ramp |
| n/a (local ingest only) | 5000/s · 15s | 5000.04/s | ~0–0.4 ms\* | 0.56 ms | 1.63 ms | 0% | Highest rate tested; max latency 29.8 ms |

\*Vegeta reported `0s` for some low percentiles on Windows at high rates (timer
resolution). Prefer p95/p99 and mean for those runs.

**Roadmap 8.3 (1 vs 3 consumers):** not run — requires a live `logstream`
Kubernetes namespace with Kafka + consumers. Re-run with the `kubectl scale`
commands above once Docker Desktop / kind is available.

**What this implies for resume wording:** you can honestly claim the ingest API
sustained **~5k req/s** with **p99 under ~2 ms** and **0% errors** on a local
process. Do **not** claim K8s consumer-scale improvements until that comparison
is measured.

The HPA in `k8s/base/consumer-hpa.yaml` uses CPU (the roadmap's simple
fallback). `logstream_consumer_lag_messages` is already exposed in Prometheus
format, ready for a later Prometheus Adapter configuration if you want HPA to
scale directly on lag.
