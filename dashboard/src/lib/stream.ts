import { API_BASE, POLL_INTERVAL_MS } from "./config";
import { fetchMetrics } from "./api";
import type { DLQEntry, LogEntry, MetricsSnapshot } from "./types";

export type ConnStatus = "connecting" | "live" | "polling" | "error";

export type StreamEventMap = {
  metrics: (s: MetricsSnapshot) => void;
  log: (e: LogEntry) => void;
  dlq: (e: DLQEntry) => void;
  status: (s: ConnStatus) => void;
};

type EventName = keyof StreamEventMap;

const MAX_SSE_ERRORS = 3;
const SSE_RETRY_MS = 10_000;

/**
 * Live data source for the dashboard. Primary transport is a single SSE
 * connection to `${API_BASE}/stream`; if it cannot be established, we fall
 * back to polling the REST metrics endpoint every POLL_INTERVAL_MS and
 * periodically try to upgrade back to SSE.
 */
class LiveStream {
  private handlers: { [K in EventName]: Set<StreamEventMap[K]> } = {
    metrics: new Set(),
    log: new Set(),
    dlq: new Set(),
    status: new Set(),
  };
  private es: EventSource | null = null;
  private pollTimer: number | null = null;
  private retryTimer: number | null = null;
  private sseErrors = 0;
  private started = false;
  private closed = false;

  subscribe<K extends EventName>(event: K, fn: StreamEventMap[K]): () => void {
    this.handlers[event].add(fn);
    return () => {
      this.handlers[event].delete(fn);
    };
  }

  private emit<K extends EventName>(event: K, payload: Parameters<StreamEventMap[K]>[0]): void {
    for (const fn of this.handlers[event]) {
      (fn as (p: Parameters<StreamEventMap[K]>[0]) => void)(payload);
    }
  }

  start(): void {
    if (this.started) return;
    this.started = true;
    this.closed = false;
    this.connectSSE();
  }

  close(): void {
    this.closed = true;
    this.started = false;
    this.sseErrors = 0;
    this.teardownSSE();
    this.stopPolling();
  }

  private setStatus(s: ConnStatus): void {
    this.emit("status", s);
  }

  private teardownSSE(): void {
    if (this.es) {
      this.es.close();
      this.es = null;
    }
  }

  private connectSSE(): void {
    if (this.closed || this.es) return;

    let es: EventSource;
    try {
      es = new EventSource(`${API_BASE}/stream`);
    } catch {
      this.startPolling();
      return;
    }
    this.es = es;

    es.onopen = () => {
      this.sseErrors = 0;
      this.setStatus("live");
    };

    es.onerror = () => {
      this.sseErrors += 1;
      if (this.sseErrors >= MAX_SSE_ERRORS) {
        this.teardownSSE();
        this.startPolling();
      }
    };

    const onFrame =
      (name: "metrics" | "log" | "dlq") => (ev: MessageEvent) => {
        try {
          this.emit(name, JSON.parse(ev.data) as Parameters<StreamEventMap[typeof name]>[0]);
        } catch {
          /* skip malformed frame */
        }
      };

    es.addEventListener("metrics", onFrame("metrics"));
    es.addEventListener("log", onFrame("log"));
    es.addEventListener("dlq", onFrame("dlq"));
  }

  private stopPolling(): void {
    if (this.pollTimer) {
      clearInterval(this.pollTimer);
      this.pollTimer = null;
    }
    if (this.retryTimer) {
      clearTimeout(this.retryTimer);
      this.retryTimer = null;
    }
  }

  private startPolling(): void {
    if (this.closed || this.pollTimer) return;
    this.setStatus("polling");

    const tick = () => {
      if (this.closed) return;
      fetchMetrics()
        .then((m) => this.emit("metrics", m))
        .catch(() => {
          /* backend down; wait for next tick */
        });
    };

    tick();
    this.pollTimer = window.setInterval(tick, POLL_INTERVAL_MS);
    this.retryTimer = window.setTimeout(() => {
      this.stopPolling();
      this.connectSSE();
    }, SSE_RETRY_MS);
  }
}

/** Singleton shared by all dashboard panels. */
export const liveStream = new LiveStream();
