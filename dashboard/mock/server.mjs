// Mock LogStream observation hub for developing the dashboard without the Go
// backend. Serves the same contract the consumer service will expose:
//   GET /api/metrics   GET /api/dlq   GET /api/logs   GET /stream (SSE)
// Usage: node mock/server.mjs [port]
import http from "node:http";

const PORT = Number(process.env.PORT ?? process.argv[2] ?? 9090);

let reqs = 0;
let accepted = 0;
let rejected = 0;
let processed = 0;
let failed = 0;
const services = ["auth-api", "orders-svc", "payments-svc", "inventory-svc", "search-svc"];
const levels = ["debug", "info", "info", "info", "warn", "warn", "error", "error", "fatal"];
const messages = [
  "handled request in 12ms",
  "connection timeout to upstream",
  "cache miss for key user:10293",
  "replicating shard 4 → node-2",
  "bulk index flush completed (n=142)",
  "queue depth at 84%, applying backpressure",
  "kafka write succeeded offset=203911",
  "retry scheduled attempt 3/6",
  "auth token refreshed",
  "consumer group rebalance in progress",
];
let lag = 0;
let dlqEntries = new Map();
let logRing = [];
let esUp = true;
const failureErrors = [
  "elasticsearch: connection refused (dial tcp 10.0.0.4:9200)",
  "elasticsearch: bulk flush timed out after 30s",
  "elasticsearch: 429 too many requests, bulk rejected",
  "elasticsearch: index closed, cannot write",
  "kafka: write failed: leader not available for partition",
  "kafka: message too large (serialized size exceeds max)",
  "postgres: connection pool exhausted",
  "postgres: deadlock detected during metadata upsert",
];

const now = () => Date.now();
const rfc = (ms) => new Date(ms).toISOString();
const rnd = (n) => Math.floor(Math.random() * n);

function makeLog() {
  const level = levels[rnd(levels.length)];
  const service = services[rnd(services.length)];
  return {
    id: crypto.randomUUID(),
    service,
    level,
    message: messages[rnd(messages.length)],
    timestamp: rfc(now() - rnd(2000)),
    received_at: rfc(now()),
  };
}

function snapshot() {
  return {
    ts: now(),
    ingestion: {
      requests_total: reqs,
      accepted_total: accepted,
      rejected_total: rejected,
      buffer_fill: rnd(240),
      buffer_capacity: 1000,
      p50_latency_ms: 0.4 + Math.random() * 0.4,
      p99_latency_ms: 0.8 + Math.random() * 1.6,
    },
    consumer: {
      processed_total: processed,
      failed_total: failed,
      in_flight: rnd(4),
      lag_messages: lag,
      retrying_total: [...dlqEntries.values()].filter((e) => e.state === "retrying").length,
      elasticsearch: {
        status: esUp ? "green" : "red",
        index: "logs",
        docs_total: processed,
      },
    },
  };
}

function seedDLQ() {
  for (let i = 0; i < 4; i++) {
    const id = crypto.randomUUID();
    dlqEntries.set(id, {
      log_id: id,
      service: services[rnd(services.length)],
      level: "error",
      message_preview: messages[rnd(messages.length)],
      retry_count: 1 + rnd(5),
      max_attempts: 6,
      state: "retrying",
      next_retry_at: rfc(now() + 1500 + rnd(3000)),
      last_error: failureErrors[rnd(failureErrors.length)],
      first_failed_at: rfc(now() - 30_000),
      last_attempt_at: rfc(now() - 1500),
    });
  }
}
seedDLQ();

setInterval(() => {
  const burst = 20 + rnd(180);
  for (let i = 0; i < burst; i++) {
    reqs += 1;
    if (Math.random() < 0.03) {
      rejected += 1;
      continue;
    }
    accepted += 1;
    const log = makeLog();
    logRing.push(log);
    if (logRing.length > 500) logRing = logRing.slice(-500);
    // small chance a log fails and lands in the DLQ / retry cycle
    if (Math.random() < 0.015) {
      failed += 1;
      const id = crypto.randomUUID();
      const prev = dlqEntries.get(log.id);
      const retry_count = prev ? prev.retry_count + 1 : 1;
      const dead = retry_count >= 6;
      dlqEntries.set(log.id, {
        log_id: log.id,
        service: log.service,
        level: log.level,
        message_preview: log.message,
        retry_count,
        max_attempts: 6,
        state: dead ? "dead" : "retrying",
        next_retry_at: dead ? null : rfc(now() + 800 * Math.pow(2, retry_count) + rnd(400)),
        last_error: failureErrors[rnd(failureErrors.length)],
        first_failed_at: prev ? prev.first_failed_at : rfc(now()),
        last_attempt_at: rfc(now()),
      });
      if (dlqEntries.size > 60) {
        dlqEntries = new Map([...dlqEntries].slice(-60));
      }
    }
  }
  processed += burst;
  lag = Math.max(0, Math.round(500 + (lag - 500) * 0.9 + rnd(81) - 40));
}, 1000);

function sse(res, event, data) {
  res.write(`event: ${event}\ndata: ${JSON.stringify(data)}\n\n`);
}

const server = http.createServer((req, res) => {
  const url = new URL(req.url, `http://localhost:${PORT}`);
  const path = url.pathname;
  res.setHeader("Access-Control-Allow-Origin", "*");

  if (path === "/api/metrics") {
    res.writeHead(200, { "Content-Type": "application/json" });
    res.end(JSON.stringify(snapshot()));
    return;
  }

  if (path === "/api/dlq") {
    const entries = [...dlqEntries.values()].sort(
      (a, b) => new Date(b.last_attempt_at) - new Date(a.last_attempt_at),
    );
    res.writeHead(200, { "Content-Type": "application/json" });
    res.end(JSON.stringify({ entries }));
    return;
  }

  if (path === "/api/logs") {
    res.writeHead(200, { "Content-Type": "application/json" });
    res.end(JSON.stringify({ logs: logRing.slice(-200) }));
    return;
  }

  if (path === "/stream") {
    res.writeHead(200, {
      "Content-Type": "text/event-stream",
      "Cache-Control": "no-cache",
      Connection: "keep-alive",
    });
    res.write(": connected\n\n");
    const t = setInterval(() => sse(res, "metrics", snapshot()), 1000);
    const pushLog = () => {
      const log = makeLog();
      logRing.push(log);
      if (logRing.length > 500) logRing = logRing.slice(-500);
      sse(res, "log", log);
    };
    const logT = setInterval(pushLog, 650);
    req.on("close", () => {
      clearInterval(t);
      clearInterval(logT);
    });
    return;
  }

  res.writeHead(404, { "Content-Type": "application/json" });
  res.end(JSON.stringify({ error: "not found" }));
});

server.listen(PORT, () => {
  console.log(`mock logstream hub on http://localhost:${PORT}`);
});
