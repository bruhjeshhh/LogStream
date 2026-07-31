import type { RateSample } from "../hooks/useLiveMetrics";
import { fmtMs, fmtRate } from "../lib/format";
import { MetricCard } from "./MetricCard";

interface Props {
  series: RateSample[];
}

export function MetricCards({ series }: Props) {
  const last = series[series.length - 1];
  const spark = (fn: (s: RateSample) => number) => series.map(fn).slice(-60);
  const failures = (s: RateSample) => s.rejected + s.failed;
  const successRate = last ? last.successRate : null;

  return (
    <div className="grid h-[86px] shrink-0 grid-cols-5 gap-2">
      <MetricCard
        label="Requests"
        value={last ? fmtRate(last.req) : "—"}
        unit="/s"
        sub="ingested logs/s"
        color="#22d3ee"
        spark={spark((s) => s.req)}
      />
      <MetricCard
        label="Drain"
        value={last ? fmtRate(last.processed) : "—"}
        unit="/s"
        sub="consumer"
        color="#34d399"
        spark={spark((s) => s.processed)}
      />
      <MetricCard
        label="p99 latency"
        value={last ? fmtMs(last.p99) : "—"}
        sub="ingestion"
        color="#fbbf24"
        spark={spark((s) => s.p99)}
      />
      <MetricCard
        label="Success rate"
        value={successRate == null ? "—" : `${successRate.toFixed(1)}%`}
        sub="accepted / total"
        color={successRate != null && successRate < 99 ? "#f87171" : "#34d399"}
        spark={spark((s) => s.successRate)}
      />
      <MetricCard
        label="Failures"
        value={last ? fmtRate(failures(last)) : "—"}
        unit="/s"
        sub="rejected + dlq"
        color="#f87171"
        spark={spark(failures)}
      />
    </div>
  );
}
