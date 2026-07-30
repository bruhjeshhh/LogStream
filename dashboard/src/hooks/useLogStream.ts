import { useCallback, useEffect, useRef, useState } from "react";
import { fetchLogs } from "../lib/api";
import { liveStream } from "../lib/stream";
import type { ConnStatus } from "../lib/stream";
import type { LogEntry } from "../lib/types";
import { MAX_LOG_ROWS, POLL_INTERVAL_MS } from "../lib/config";

/**
 * Live tail of processed logs plus the set of services seen so far (used to
 * populate the source filter). Backfills from GET /api/logs on mount and on
 * reconnect; refetches on a cadence when in polling mode.
 */
export function useLogStream() {
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [status, setStatus] = useState<ConnStatus>("connecting");
  const [services, setServices] = useState<string[]>([]);
  const logRef = useRef<LogEntry[]>([]);
  const servicesRef = useRef<Set<string>>(new Set());
  const prevStatusRef = useRef<ConnStatus>("connecting");

  const push = useCallback((entry: LogEntry) => {
    logRef.current = [...logRef.current, entry].slice(-MAX_LOG_ROWS);
    setLogs(logRef.current);
    if (entry.service && !servicesRef.current.has(entry.service)) {
      servicesRef.current.add(entry.service);
      setServices([...servicesRef.current].sort());
    }
  }, []);

  const replace = useCallback((list: LogEntry[]) => {
    logRef.current = list.slice(-MAX_LOG_ROWS);
    setLogs(logRef.current);
    servicesRef.current = new Set(list.map((l) => l.service).filter(Boolean));
    setServices([...servicesRef.current].sort());
  }, []);

  useEffect(() => {
    const offs = [liveStream.subscribe("status", setStatus), liveStream.subscribe("log", push)];
    fetchLogs()
      .then(replace)
      .catch(() => {});
    return () => offs.forEach((off) => off());
  }, [push, replace]);

  useEffect(() => {
    const prev = prevStatusRef.current;
    prevStatusRef.current = status;
    if (status === "live" && prev !== "live") {
      fetchLogs()
        .then(replace)
        .catch(() => {});
    }
  }, [status, replace]);

  useEffect(() => {
    if (status !== "polling") return;
    const t = window.setInterval(() => {
      fetchLogs()
        .then(replace)
        .catch(() => {});
    }, POLL_INTERVAL_MS);
    return () => clearInterval(t);
  }, [status, replace]);

  return { logs, status, services };
}
