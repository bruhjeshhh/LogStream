# LogStream Dashboard

A single-screen, real-time ops dashboard for the LogStream log pipeline (Go +
Kafka + Elasticsearch). Dark theme, dense layout, live-updating — throughput,
latency, failures/retries, dead-letter queue, and a tailing log feed.

## Run it locally

```bash
cd dashboard
npm install
npm run dev        # http://localhost:5173
```

Point it at your backend by setting `VITE_API_BASE` (default `http://localhost:9090`):

```bash
VITE_API_BASE=http://localhost:9090 npm run dev
```

### No backend handy? Use the bundled mock

```bash
node mock/server.mjs 9091          # serves the same contract the Go side will expose
VITE_API_BASE=http://localhost:9091 npm run dev
```

### Production build

```bash
npm run build      # typecheck + bundle → dist/
npm run preview    # serve the built app
```

For the GitHub Pages layout (repo is served at `/LogStream/`), build with the
base path and a public backend URL baked in:

```bash
GH_PAGES=true VITE_API_BASE=https://<your-hub>.onrender.com npm run build
```

## Deploying

The frontend is a static SPA; it reads no secrets. It just needs a public URL
for the observation hub (the backend behind `VITE_API_BASE`).

### Frontend → GitHub Pages

1. Enable Pages: repo **Settings → Pages → Source: GitHub Actions**.
2. Set the backend URL as a repo variable: **Settings → Secrets and variables →
   Actions → Variables** → `VITE_API_BASE` = `https://<your-hub>.onrender.com`.
3. Push to `main`. The workflow in `.github/workflows/pages.yml` builds
   `dashboard/` (with `GH_PAGES=true`, so assets resolve under `/LogStream/`)
   and deploys to
   `https://<user>.github.io/LogStream/`.

### Backend → Render (mock hub)

The mock (`mock/server.mjs`) implements the same contract as the real Go
consumer and can be hosted anywhere Node runs. Two options on Render:

1. **Blueprint (recommended):** Dashboard → **New + → Blueprint** → select the
   LogStream repo. The `render.yaml` at the repo root provisions a free Node
   web service rooted at `dashboard/mock` (start command `npm start`; Render
   sets `$PORT` automatically).
2. **Manual:** **New + → Web Service** → select the repo → **Root Directory**
   `dashboard/mock` → Runtime *Node* → **Start Command** `npm start` → Free
   plan.

Then copy the generated `.onrender.com` URL into `VITE_API_BASE` (step 2 in the
Pages section above).

Free-tier note: Render's free web services **sleep after 15 minutes of
inactivity** and take ~30–60s to wake on the next request; the dashboard
reconnects by itself, so it just looks slow on first open. The mock also resets
its counters whenever the instance restarts.

The mock shows **simulated** data. To show **real** pipeline data instead, point
`VITE_API_BASE` at the deployed Go consumer (any VPS or an always-on tunnel).

## Environment variables

| Var | Default | Purpose |
|-----|---------|---------|
| `VITE_API_BASE` | `http://localhost:9090` | Base URL of the consumer observation hub (hosts `/api/*` and `/stream`) |
| `VITE_POLL_INTERVAL_MS` | `2000` | REST polling cadence used only when the SSE stream is unreachable |
| `VITE_HISTORY_SECONDS` | `120` | Rolling window for the time-series charts |

## Backend API contract (what the consumer hub must expose)

The dashboard talks to **one** base URL only. Everything below is served by the
consumer service (`:9090`), which internally polls the ingestion service
(`:8080`) once per second and merges the two metric sets.

### REST

**`GET /api/metrics`** — merged snapshot (also the initial backfill):

```json
{
  "ts": 1785588727946,
  "ingestion": {
    "requests_total": 123456, "accepted_total": 123400, "rejected_total": 56,
    "buffer_fill": 320, "buffer_capacity": 1000,
    "p50_latency_ms": 0.45, "p99_latency_ms": 1.23
  },
  "consumer": {
    "processed_total": 123400, "failed_total": 12, "in_flight": 2,
    "lag_messages": 345, "retrying_total": 3,
    "elasticsearch": { "status": "green", "index": "logs", "docs_total": 123400 }
  }
}
```

**`GET /api/dlq?limit=200`** — live retry/DLQ registry (in-memory on the consumer):

```json
{
  "entries": [{
    "log_id": "0192f3c8-7b4a-...", "service": "auth-api", "level": "error",
    "message_preview": "connection timeout to upstream",
    "retry_count": 4, "max_attempts": 6,
    "state": "retrying",
    "next_retry_at": "2026-08-01T10:00:03.000Z",
    "last_error": "elasticsearch: connection refused",
    "first_failed_at": "2026-08-01T09:59:55.000Z",
    "last_attempt_at": "2026-08-01T10:00:02.000Z"
  }]
}
```

`state` ∈ `retrying | dead | flushed`. `next_retry_at` is `null` when not retrying.

**`GET /api/logs?tail=200`** — recent processed logs (bounded ring buffer):

```json
{ "logs": [ { "id": "0192f3c8-...", "service": "auth-api", "level": "info",
              "message": "handled request in 12ms",
              "timestamp": "2026-08-01T10:00:00.000Z",
              "received_at": "2026-08-01T10:00:00.120Z" } ] }
```

### SSE stream — `GET /stream`

Single multiplexed connection; named events, one JSON payload per line:

```
event: metrics   # every ~1s — merged { ts, ingestion, consumer } (same shape as /api/metrics)
event: log       # as each log is processed — a single log object
event: dlq       # on retry schedule / backoff / final death — a single entry object
```

No event-ID/replay needed: the dashboard backfills from the three REST
endpoints on connect and on reconnect. If the stream is unreachable it falls
back to polling `/api/metrics` (and the DLQ/log feeds refetch their endpoints)
every 2s, retrying SSE every 10s.

### CORS

The consumer hub must send `Access-Control-Allow-Origin: *` on all responses
(the app runs from a different origin during development). This matches the
`mock/server.mjs` reference implementation.

## Component structure

```
dashboard/
├── mock/server.mjs            # dependency-free mock hub (node)
├── index.html
├── vite.config.ts
└── src/
    ├── App.tsx                # screen shell: header, health strip, cards, charts, tables
    ├── index.css              # Tailwind v4 theme tokens + custom animations
    ├── lib/
    │   ├── config.ts          # env-var config (VITE_*)
    │   ├── types.ts           # contract types (MetricsSnapshot, DLQEntry, LogEntry…)
    │   ├── api.ts             # REST client
    │   ├── stream.ts          # singleton SSE client + polling fallback
    │   ├── format.ts          # number/ms/relative-time/id/color helpers
    │   └── levels.ts          # log-level color/label map
    ├── hooks/
    │   ├── useLiveMetrics.ts  # per-second rate series (2-min window) from counter deltas
    │   ├── useDLQ.ts          # live DLQ registry
    │   ├── useLogStream.ts    # live log tail + seen-services
    │   └── useNow.ts          # 1s ticker for relative timestamps
    └── components/
        ├── Header.tsx, ConnectionBadge.tsx
        ├── HealthStrip.tsx    # lag, ES status, buffer fill, in-flight, retrying
        ├── MetricCards.tsx    # 5 big numbers w/ sparklines
        ├── ThroughputChart.tsx / LatencyChart.tsx   # Recharts
        ├── ChartBits.tsx      # shared dark tooltip + honest empty state
        ├── DLQTable.tsx       # failed/retrying entries, row-flash on change
        ├── LogFeed.tsx        # tailing feed w/ level + service filters
        ├── Panel.tsx, Sparkline.tsx, MetricCard.tsx
        └── main.tsx
```

## Behavior notes

- Rates (`/s`) are computed client-side as deltas of the backend's monotonic
  counters — the backend only needs to expose cumulative totals plus a
  timestamp.
- Charts keep the last `HISTORY_SECONDS` of 1-second samples in a ring buffer.
- Everything shows an honest empty state until real data arrives; the
  connection badge reflects SSE (`LIVE`), fallback polling (`POLLING`), or
  nothing reachable (`OFFLINE`).
