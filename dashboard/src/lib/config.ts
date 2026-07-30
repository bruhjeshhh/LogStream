function num(v: string | undefined, fallback: number): number {
  const n = Number(v);
  return v === undefined || v === "" || Number.isNaN(n) ? fallback : n;
}

/** Base URL of the consumer observation hub (hosts /stream and /api/*). */
export const API_BASE: string = import.meta.env.VITE_API_BASE ?? "http://localhost:9090";

/** Fallback polling cadence used only when the SSE stream is unreachable. */
export const POLL_INTERVAL_MS: number = num(import.meta.env.VITE_POLL_INTERVAL_MS, 2000);

/** Rolling window kept for the time-series charts. */
export const HISTORY_SECONDS: number = num(import.meta.env.VITE_HISTORY_SECONDS, 120);

export const MAX_DLQ_ROWS = 200;
export const MAX_LOG_ROWS = 500;
