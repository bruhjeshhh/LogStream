import { useCallback, useEffect, useRef, useState } from "react";
import { fetchDLQ } from "../lib/api";
import { liveStream } from "../lib/stream";
import type { ConnStatus } from "../lib/stream";
import type { DLQEntry } from "../lib/types";
import { POLL_INTERVAL_MS } from "../lib/config";

function sortEntries(list: DLQEntry[]): DLQEntry[] {
  return [...list].sort(
    (a, b) => new Date(b.last_attempt_at).getTime() - new Date(a.last_attempt_at).getTime(),
  );
}

/**
 * Live DLQ registry. Appends/updates entries as `dlq` events arrive; on first
 * load and on reconnect it backfills from GET /api/dlq; in polling mode it
 * refetches on a fixed cadence.
 */
export function useDLQ() {
  const [entries, setEntries] = useState<DLQEntry[]>([]);
  const [status, setStatus] = useState<ConnStatus>("connecting");
  const mapRef = useRef<Map<string, DLQEntry>>(new Map());
  const prevStatusRef = useRef<ConnStatus>("connecting");

  const applyAll = useCallback((list: DLQEntry[]) => {
    mapRef.current = new Map(list.map((e) => [e.log_id, e]));
    setEntries(sortEntries(list));
  }, []);

  const applyOne = useCallback((entry: DLQEntry) => {
    mapRef.current.set(entry.log_id, entry);
    setEntries(sortEntries([...mapRef.current.values()]));
  }, []);

  useEffect(() => {
    const offs = [liveStream.subscribe("status", setStatus), liveStream.subscribe("dlq", applyOne)];
    fetchDLQ()
      .then(applyAll)
      .catch(() => {});
    return () => offs.forEach((off) => off());
  }, [applyOne, applyAll]);

  useEffect(() => {
    const prev = prevStatusRef.current;
    prevStatusRef.current = status;
    if (status === "live" && prev !== "live") {
      fetchDLQ()
        .then(applyAll)
        .catch(() => {});
    }
  }, [status, applyAll]);

  useEffect(() => {
    if (status !== "polling") return;
    const t = window.setInterval(() => {
      fetchDLQ()
        .then(applyAll)
        .catch(() => {});
    }, POLL_INTERVAL_MS);
    return () => clearInterval(t);
  }, [status, applyAll]);

  return { entries, status };
}
