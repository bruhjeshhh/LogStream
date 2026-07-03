# LogStream — Architecture & Design

LogStream is a centralized log ingestion and search system, scoped down from the Elastic ELK stack. It addresses the challenge of managing logs across distributed services running on multiple pods, where traditional grep-based debugging becomes impractical. LogStream provides a unified endpoint for log emission and a single interface for log search and analysis.

---

## High-Level Flow

1. **Producers** — Services emit structured log lines in JSON format:
   ```json
   {"service": "auth-api", "level": "ERROR", "timestamp": "...", "message": "..."}
   ```

2. **Ingestion** — Logs are received via a gRPC ingestion endpoint and published to a Kafka cluster. Kafka acts as a durable buffer, decoupling ingestion throughput from downstream processing.

3. **Processing** — A consumer group reads from Kafka and performs:
   - **Parse** — Extract and validate structured fields
   - **Normalize** — Standardize field names, formats, and levels
   - **Enrich** — Augment logs with additional context (e.g., service metadata, cluster info)
   - **Index** — Write enriched logs to Elasticsearch, and lightweight metadata to PostgreSQL for index management

4. **Query** — A query API sits on top of Elasticsearch, enabling searches such as:
   > "Show me all ERROR logs from `auth-api` in the last hour containing 'timeout'"

5. **Deployment** — The entire system runs on Kubernetes. Ingestion and consumer components are separate deployments, allowing independent scaling of consumers when Kafka lag accumulates.

---

## Architecture Diagram

```
+-------------------+
| Log Producers     |
+-------------------+
          |
          |
    HTTP / gRPC
          |
          v
+-------------------+
| Ingestion API     |
+-------------------+
          |
          |
      Kafka Topic
          |
 +--------+--------+
 |                 |
 v                 v
+-----------+   +-----------+
| Consumer  |   | Consumer  |
|     #1    |   |     #2    |
+-----------+   +-----------+
 |                 |
 +--------+--------+
          |
          v
+-------------------+
| Elasticsearch     |
+-------------------+
          |
     Query API
          |
          v
       Client
```

Consumers additionally write metadata to:

```
+-------------------+
| PostgreSQL        |
+-------------------+
```
