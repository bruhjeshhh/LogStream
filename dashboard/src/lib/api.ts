import { API_BASE } from "./config";
import type { DLQEntry, LogEntry, MetricsSnapshot } from "./types";

const MAX = 200;

async function get<T>(path: string): Promise<T> {
  const res = await fetch(`${API_BASE}${path}`);
  if (!res.ok) throw new Error(`${path}: HTTP ${res.status}`);
  return (await res.json()) as T;
}

export const fetchMetrics = () => get<MetricsSnapshot>("/api/metrics");

export const fetchDLQ = (limit = MAX) =>
  get<{ entries: DLQEntry[] }>(`/api/dlq?limit=${limit}`).then((r) => r.entries);

export const fetchLogs = (tail = MAX) =>
  get<{ logs: LogEntry[] }>(`/api/logs?tail=${tail}`).then((r) => r.logs);
