# Load testing with Vegeta

Install Vegeta first (for example, `winget install tsenart.vegeta`). Start the
Kubernetes ingestion port-forward from `k8s/README.md`, then run the baseline:

```powershell
.\loadtest\run.ps1 -Rate 25 -Duration 30s -Name single-replica
```

Scale the consumer and repeat the same request rate:

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

Increase `-Rate` gradually (for example 25, 50, 100, 200) until latency grows,
errors appear, or lag no longer returns to zero. Keep the generated `.txt`
reports locally and enter measured values below. The table intentionally has no
made-up metrics.

| Consumer replicas | Rate | Throughput | p50 | p95 | p99 | Error rate | What broke first? |
| --- | --- | --- | --- | --- | --- | --- | --- |
| 1 | _record after running_ |  |  |  |  |  |  |
| 3 | _record after running_ |  |  |  |  |  |  |

The HPA in `k8s/base/consumer-hpa.yaml` uses CPU (the roadmap's simple
fallback). `logstream_consumer_lag_messages` is already exposed in Prometheus
format, ready for a later Prometheus Adapter configuration if you want HPA to
scale directly on lag.
