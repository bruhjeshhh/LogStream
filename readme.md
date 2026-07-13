LogStream is a Log ingestion and searching service. It uses Kafka and Elasticsearch.

## Kubernetes and load testing

The beginner-friendly Kubernetes setup is in [k8s/README.md](k8s/README.md).
It runs the Go services as Deployments and installs Kafka, Elasticsearch, and
PostgreSQL from Helm charts. The consumer exposes `/metrics`, including its
Kafka lag estimate, and has a CPU-based HPA from one to three replicas.

The reproducible Vegeta commands and an honest results table are in
[loadtest/README.md](loadtest/README.md). Run the tests on your machine and
record the values there rather than committing fabricated throughput numbers.

## Failure handling

The consumer provides at-least-once delivery. A Kafka record is committed only after it is indexed successfully, or after a failure record is successfully written to the `logs-dlq` topic. Malformed JSON is DLQ'd immediately. Sink failures are retried five times with jittered exponential backoff (100ms, 200ms, 400ms, 800ms, 1.6s, capped at 5s), then DLQ'd.

During an Elasticsearch or configured PostgreSQL outage, the consumer remains running and retries each affected record. If the dependency returns before retries are exhausted, processing resumes normally. Otherwise the record is sent to `logs-dlq` and its original offset is committed. If Kafka/DLQ is unavailable, the original record is deliberately left uncommitted for redelivery after restart.

Elasticsearch uses the log UUID as its document ID, so replaying a record overwrites the same document. PostgreSQL store implementations must enforce a unique log ID and use an upsert (`ON CONFLICT`) for the same property. This is at-least-once delivery with deduplication, not an exactly-once guarantee.

To exercise recovery locally, start the stack and consumer, then run `docker stop logstream-es` (or `docker stop logstream-postgres`) while producing records. The consumer logs retry attempts; restart the container before the retry budget expires to see the record recover. Keep it stopped past the budget to see the record published to `logs-dlq`; inspect it with `kafka-console-consumer --bootstrap-server localhost:9092 --topic logs-dlq --from-beginning` inside the Kafka container.
