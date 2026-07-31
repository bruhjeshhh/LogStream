import { useEffect, useRef, useState } from "react";
import type { DLQEntry, DLQState } from "../lib/types";
import { fmtRelUntil, shortId, truncate } from "../lib/format";
import { colorFor } from "../lib/format";
import { levelMeta } from "../lib/levels";
import { useNow } from "../hooks/useNow";

const STATE_META: Record<DLQState, { label: string; color: string; bg: string; pulse?: boolean }> = {
  retrying: { label: "RETRYING", color: "#fbbf24", bg: "rgba(251,191,36,0.14)", pulse: true },
  dead: { label: "DEAD", color: "#f87171", bg: "rgba(248,113,113,0.16)" },
  flushed: { label: "FLUSHED", color: "#93a1b5", bg: "rgba(147,161,181,0.14)" },
};

function StateBadge({ state }: { state: DLQState }) {
  const m = STATE_META[state];
  return (
    <span
      className="inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-[9px] font-bold tracking-wider"
      style={{ color: m.color, background: m.bg }}
    >
      {m.pulse && <span className="animate-blink h-1 w-1 rounded-full" style={{ background: m.color }} />}
      {m.label}
    </span>
  );
}

/** Flashes rows whose retry count/state/last attempt changed since last render. */
function useRowFlash(entries: DLQEntry[]) {
  const prevRef = useRef<Map<string, string>>(new Map());
  const [flashIds, setFlashIds] = useState<Set<string>>(new Set());

  useEffect(() => {
    const cur = new Map<string, string>();
    const flashed = new Set<string>();
    for (const e of entries) {
      const sig = `${e.retry_count}|${e.state}|${e.last_attempt_at}`;
      cur.set(e.log_id, sig);
      const prev = prevRef.current.get(e.log_id);
      if (prev === undefined || prev !== sig) flashed.add(e.log_id);
    }
    prevRef.current = cur;
    if (flashed.size) {
      setFlashIds(flashed);
      const t = window.setTimeout(() => setFlashIds(new Set()), 1600);
      return () => clearTimeout(t);
    }
  }, [entries]);

  return flashIds;
}

export function DLQTable({ entries }: { entries: DLQEntry[] }) {
  const now = useNow(1000);
  const flashIds = useRowFlash(entries);

  if (entries.length === 0) {
    return (
      <div className="flex h-full items-center justify-center">
        <div className="text-center">
          <div className="mx-auto mb-2 flex h-8 w-8 items-center justify-center rounded-full border border-mint/40 bg-mint/10">
            <svg viewBox="0 0 16 16" className="h-4 w-4 text-mint" fill="none">
              <path d="M3 8.5 6.5 12 13 4.5" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
            </svg>
          </div>
          <p className="text-xs text-dim">No dead letters — every write is flowing.</p>
          <p className="mt-1 font-mono text-[10px] text-faint">writes that exhaust 6 retries land here</p>
        </div>
      </div>
    );
  }

  return (
    <div className="h-full overflow-auto">
      <table className="w-full border-collapse text-left text-[11px]">
        <thead className="sticky top-0 z-10 bg-panel">
          <tr className="text-[9px] uppercase tracking-wider text-faint">
            <th className="px-2 py-1.5 font-semibold">state</th>
            <th className="px-2 py-1.5 font-semibold">log id</th>
            <th className="px-2 py-1.5 font-semibold">service</th>
            <th className="px-2 py-1.5 font-semibold">level</th>
            <th className="px-2 py-1.5 font-semibold">message</th>
            <th className="px-2 py-1.5 font-semibold">retries</th>
            <th className="px-2 py-1.5 font-semibold">next retry</th>
            <th className="px-2 py-1.5 font-semibold">last error</th>
          </tr>
        </thead>
        <tbody>
          {entries.map((e) => {
            const lm = levelMeta(e.level);
            return (
              <tr
                key={e.log_id}
                className={`border-t border-edge/60 hover:bg-panel2/50 ${flashIds.has(e.log_id) ? "animate-flash" : ""}`}
              >
                <td className="px-2 py-1.5">
                  <StateBadge state={e.state} />
                </td>
                <td className="px-2 py-1.5 font-mono text-faint" title={e.log_id}>
                  {shortId(e.log_id)}
                </td>
                <td className="max-w-[100px] truncate px-2 py-1.5 font-mono" style={{ color: colorFor(e.service) }}>
                  {e.service}
                </td>
                <td className="px-2 py-1.5">
                  <span
                    className="rounded px-1 py-0.5 text-[9px] font-semibold"
                    style={{ color: lm.text, background: lm.bg }}
                  >
                    {lm.label}
                  </span>
                </td>
                <td className="max-w-[180px] truncate px-2 py-1.5 text-dim" title={e.message_preview}>
                  {truncate(e.message_preview, 36)}
                </td>
                <td className="px-2 py-1.5">
                  <div className="flex items-center gap-1.5">
                    <span className="font-mono text-faint">
                      {e.retry_count}/{e.max_attempts}
                    </span>
                    <div className="h-1 w-10 overflow-hidden rounded-full bg-panel2">
                      <div
                        className="h-full rounded-full bg-amber"
                        style={{
                          width: `${Math.min(100, (e.retry_count / Math.max(1, e.max_attempts)) * 100)}%`,
                        }}
                      />
                    </div>
                  </div>
                </td>
                <td className="px-2 py-1.5 font-mono">
                  {e.state === "retrying" && e.next_retry_at ? (
                    <span className="text-amber" title={new Date(e.next_retry_at).toLocaleString()}>
                      in {fmtRelUntil(new Date(e.next_retry_at).getTime(), now)}
                    </span>
                  ) : (
                    <span className="text-faint">—</span>
                  )}
                </td>
                <td className="max-w-[220px] truncate px-2 py-1.5 font-mono text-rose" title={e.last_error}>
                  {truncate(e.last_error, 40)}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
