import { useEffect, useMemo, useRef, useState } from "react";
import type { LogEntry } from "../lib/types";
import { colorFor, fmtClockFull } from "../lib/format";
import { LEVELS, levelMeta } from "../lib/levels";

interface Props {
  logs: LogEntry[];
  services: string[];
}

export function LogFeed({ logs, services }: Props) {
  const [activeLevels, setActiveLevels] = useState<Set<string>>(new Set(LEVELS));
  const [service, setService] = useState("all");
  const [paused, setPaused] = useState(false);
  const [newCount, setNewCount] = useState(0);
  const containerRef = useRef<HTMLDivElement>(null);
  const prevLenRef = useRef(0);

  const filtered = useMemo(
    () =>
      logs.filter(
        (l) =>
          activeLevels.has(l.level.toLowerCase()) && (service === "all" || l.service === service),
      ),
    [logs, activeLevels, service],
  );

  useEffect(() => {
    const el = containerRef.current;
    if (el && !paused) el.scrollTop = el.scrollHeight;
    const added = Math.max(0, filtered.length - prevLenRef.current);
    prevLenRef.current = filtered.length;
    if (added > 0) setNewCount((c) => (paused ? c + added : 0));
  }, [filtered.length, paused]);

  const handleScroll = () => {
    const el = containerRef.current;
    if (!el) return;
    const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 48;
    setPaused(!nearBottom);
  };

  const toggleLevel = (level: string) => {
    setActiveLevels((prev) => {
      const next = new Set(prev);
      if (next.has(level)) next.delete(level);
      else next.add(level);
      return next;
    });
  };

  const resume = () => {
    setPaused(false);
    setNewCount(0);
  };

  return (
    <div className="flex h-full flex-col">
      <div className="flex shrink-0 items-center gap-1.5 border-b border-edge bg-panel px-2 py-1.5">
        {LEVELS.map((l) => {
          const active = activeLevels.has(l);
          const meta = levelMeta(l);
          return (
            <button
              key={l}
              onClick={() => toggleLevel(l)}
              className={`rounded px-1.5 py-0.5 text-[9px] font-bold uppercase tracking-wider transition-opacity ${
                active ? "" : "opacity-35 hover:opacity-70"
              }`}
              style={{ color: meta.text, background: meta.bg }}
            >
              {l}
            </button>
          );
        })}
        <div className="ml-auto flex items-center gap-1.5">
          <select
            value={service}
            onChange={(e) => setService(e.target.value)}
            className="max-w-[150px] rounded border border-edge bg-panel2 px-1.5 py-0.5 text-[10px] text-dim outline-none"
          >
            <option value="all">all services</option>
            {services.map((s) => (
              <option key={s} value={s}>
                {s}
              </option>
            ))}
          </select>
          {paused ? (
            <button
              onClick={resume}
              className="rounded border border-amber/50 bg-amber/10 px-1.5 py-0.5 text-[10px] font-semibold text-amber"
            >
              {newCount > 0 ? `${newCount} new` : "paused"} · resume
            </button>
          ) : (
            <span className="flex items-center gap-1.5 text-[10px] text-mint">
              <span className="animate-blink h-1.5 w-1.5 rounded-full bg-mint" />
              following
            </span>
          )}
        </div>
      </div>

      <div
        ref={containerRef}
        onScroll={handleScroll}
        className="min-h-0 flex-1 overflow-auto font-mono text-[11px] leading-[1.6]"
      >
        {filtered.length === 0 ? (
          <div className="flex h-full items-center justify-center">
            <p className="text-xs text-faint">
              {logs.length === 0
                ? "No logs yet — waiting for pipeline traffic…"
                : "No rows match the current filters."}
            </p>
          </div>
        ) : (
          filtered.map((l) => {
            const meta = levelMeta(l.level);
            return (
              <div
                key={l.id}
                className="flex items-baseline gap-2 whitespace-nowrap border-b border-edge/40 px-2 py-[1px] hover:bg-panel2/60"
              >
                <span className="shrink-0 text-faint">{fmtClockFull(new Date(l.timestamp).getTime())}</span>
                <span
                  className="shrink-0 rounded px-1 text-[9px] font-bold"
                  style={{ color: meta.text, background: meta.bg }}
                >
                  {meta.label}
                </span>
                <span className="max-w-[140px] shrink-0 truncate" style={{ color: colorFor(l.service) }}>
                  {l.service}
                </span>
                <span className="min-w-0 flex-1 truncate text-dim">{l.message}</span>
              </div>
            );
          })
        )}
      </div>
    </div>
  );
}
