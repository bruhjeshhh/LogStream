import { useCallback, useEffect, useRef, useState } from "react";
import { fetchMetrics } from "../lib/api";
import { liveStream } from "../lib/stream";
import type { ConnStatus } from "../lib/stream";
import type { MetricsSnapshot } from "../lib/types";
import { HISTORY_SECONDS } from "../lib/config";
import { normTs } from "../lib/format";

export interface RateSample {
  t: number;
  req: number;
  accepted: number;
  rejected: number;
  processed: number;
  failed: number;
  p99: number;
  successRate: number;
}

const MAX_SAMPLES = HISTORY_SECONDS;
const MAX_LAG_SAMPLES = 90;

function deltaPerSec(prev: number, cur: number, dtMs: number): number {
  if (!(dtMs > 0)) return 0;
  return Math.max(0, (cur - prev) / dtMs) * 1000;
}

/**
 * Derives the rolling per-second rate series (2-min window) and the latest
 * merged metrics snapshot from the live stream, with a one-shot REST backfill.
 */
export function useLiveMetrics() {
  const [status, setStatus] = useState<ConnStatus>("connecting");
  const [latest, setLatest] = useState<MetricsSnapshot | null>(null);
  const [series, setSeries] = useState<RateSample[]>([]);
  const [lagSeries, setLagSeries] = useState<number[]>([]);
  const [lastUpdate, setLastUpdate] = useState(0);

  const prevRef = useRef<MetricsSnapshot | null>(null);
  const lastAtRef = useRef(0);
  const seriesRef = useRef<RateSample[]>([]);
  const lagRef = useRef<number[]>([]);

  const push = useCallback((snapshot: MetricsSnapshot) => {
    const prev = prevRef.current;
    const now = Date.now();
    const dt = prev && lastAtRef.current ? now - lastAtRef.current : 0;
    lastAtRef.current = now;
    prevRef.current = snapshot;

    const ing = snapshot.ingestion;
    const con = snapshot.consumer;

    const dAccepted = prev
      ? Math.max(0, ing.accepted_total - prev.ingestion.accepted_total)
      : 0;
    const dRejected = prev
      ? Math.max(0, ing.rejected_total - prev.ingestion.rejected_total)
      : 0;
    const successRate =
      dAccepted + dRejected > 0 ? (dAccepted / (dAccepted + dRejected)) * 100 : 100;

    const sample: RateSample = {
      t: normTs(snapshot.ts || now),
      req: prev ? deltaPerSec(prev.ingestion.requests_total, ing.requests_total, dt) : 0,
      accepted: prev ? deltaPerSec(prev.ingestion.accepted_total, ing.accepted_total, dt) : 0,
      rejected: prev ? deltaPerSec(prev.ingestion.rejected_total, ing.rejected_total, dt) : 0,
      processed: prev ? deltaPerSec(prev.consumer.processed_total, con.processed_total, dt) : 0,
      failed: prev ? deltaPerSec(prev.consumer.failed_total, con.failed_total, dt) : 0,
      p99: ing.p99_latency_ms,
      successRate,
    };

    seriesRef.current = [...seriesRef.current, sample].slice(-MAX_SAMPLES);
    setSeries(seriesRef.current);

    lagRef.current = [...lagRef.current, con.lag_messages].slice(-MAX_LAG_SAMPLES);
    setLagSeries(lagRef.current);

    setLatest(snapshot);
    setLastUpdate(now);
  }, []);

  useEffect(() => {
    const offs = [liveStream.subscribe("status", setStatus), liveStream.subscribe("metrics", push)];
    fetchMetrics()
      .then(push)
      .catch(() => {});
    return () => offs.forEach((off) => off());
  }, [push]);

  return { status, latest, series, lagSeries, lastUpdate };
}
