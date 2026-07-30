export function normTs(ts: number): number {
  return ts < 1e12 ? ts * 1000 : ts;
}

export function fmtInt(n: number): string {
  if (!Number.isFinite(n)) return "—";
  return n.toLocaleString("en-US");
}

export function fmtRate(n: number): string {
  if (!Number.isFinite(n)) return "—";
  if (n >= 10000) return `${Math.round(n / 1000)}k`;
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k`;
  if (n >= 100) return n.toFixed(0);
  return n.toFixed(1);
}

export function fmtMs(n: number): string {
  if (!Number.isFinite(n)) return "—";
  if (n >= 1000) return `${(n / 1000).toFixed(2)}s`;
  if (n >= 100) return `${n.toFixed(0)}ms`;
  return `${n.toFixed(n >= 10 ? 1 : 2)}ms`;
}

export function fmtClock(ts: number): string {
  const d = new Date(ts);
  const p = (x: number) => x.toString().padStart(2, "0");
  return `${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`;
}

export function fmtClockFull(ts: number): string {
  const d = new Date(ts);
  const p = (x: number) => x.toString().padStart(2, "0");
  return `${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}.${d
    .getMilliseconds()
    .toString()
    .padStart(3, "0")}`;
}

/** Relative duration like "2s ago" (handles future timestamps too). */
export function fmtRel(ts: number, now: number): string {
  const a = Math.abs(now - ts);
  const suffix = now >= ts ? "ago" : "ahead";
  if (a < 1000) return "now";
  if (a < 60_000) return `${Math.round(a / 1000)}s ${suffix}`;
  if (a < 3_600_000) return `${Math.round(a / 60_000)}m ${suffix}`;
  return `${Math.round(a / 3_600_000)}h ${suffix}`;
}

/** Countdown until the given future timestamp, e.g. "in 2.3s". */
export function fmtRelUntil(ts: number, now: number): string {
  const d = ts - now;
  if (d <= 0) return "now";
  if (d < 1000) return `${(d / 1000).toFixed(1)}s`;
  if (d < 60_000) return `${(d / 1000).toFixed(1)}s`;
  if (d < 3_600_000) return `${Math.round(d / 60_000)}m`;
  return `${Math.round(d / 3_600_000)}h`;
}

export function shortId(id: string): string {
  return id.length > 13 ? `${id.slice(0, 13)}…` : id;
}

export function truncate(s: string, n: number): string {
  if (s.length <= n) return s;
  return `${s.slice(0, n - 1)}…`;
}

/** Stable deterministic color for a service name, for the dark theme. */
export function colorFor(name: string): string {
  let h = 0;
  for (let i = 0; i < name.length; i++) h = (h * 31 + name.charCodeAt(i)) >>> 0;
  return `hsl(${h % 360} 62% 62%)`;
}
