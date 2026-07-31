import type { ElasticsearchHealth } from "../lib/types";
import { fmtInt } from "../lib/format";
import { Sparkline } from "./Sparkline";

interface Props {
  lag: number | undefined;
  lagSeries: number[];
  es: ElasticsearchHealth | undefined;
  bufferFill: number | undefined;
  bufferCapacity: number | undefined;
  inFlight: number | undefined;
  retrying: number | undefined;
}

function esMeta(status: string | undefined): { label: string; color: string } {
  switch (status?.toLowerCase()) {
    case "green":
      return { label: "GREEN", color: "#34d399" };
    case "yellow":
      return { label: "YELLOW", color: "#fbbf24" };
    case "red":
      return { label: "RED", color: "#f87171" };
    default:
      return { label: "UNREACHABLE", color: "#f87171" };
  }
}

function Tile({
  label,
  value,
  color,
  sub,
  spark,
  bar,
}: {
  label: string;
  value: string;
  color: string;
  sub?: string;
  spark?: number[];
  bar?: number;
}) {
  return (
    <div className="relative flex flex-col justify-center overflow-hidden rounded border border-edge bg-panel px-3">
      <span className="text-[9px] font-semibold uppercase tracking-wider text-faint">{label}</span>
      <div className="flex items-baseline gap-2">
        <span className="font-mono text-[15px] font-semibold leading-tight tabular-nums" style={{ color }}>
          {value}
        </span>
        {sub && <span className="truncate font-mono text-[10px] text-faint">{sub}</span>}
      </div>
      {bar != null && (
        <div className="mt-1 h-1 w-full overflow-hidden rounded-full bg-panel2">
          <div
            className="h-full rounded-full transition-all duration-500"
            style={{ width: `${bar}%`, background: color }}
          />
        </div>
      )}
      {spark && spark.length > 1 && (
        <div className="pointer-events-none absolute inset-y-1.5 right-2 w-16 opacity-70">
          <Sparkline data={spark} color={color} height={38} />
        </div>
      )}
    </div>
  );
}

export function HealthStrip({ lag, lagSeries, es, bufferFill, bufferCapacity, inFlight, retrying }: Props) {
  const fill = bufferCapacity && bufferFill != null ? bufferFill / bufferCapacity : null;
  const fillPct = fill != null ? Math.round(fill * 100) : null;
  const fillColor = fill == null ? "#fbbf24" : fill > 0.8 ? "#f87171" : fill > 0.5 ? "#fbbf24" : "#34d399";
  const lagColor = lag == null ? "#fbbf24" : lag > 1000 ? "#f87171" : lag > 100 ? "#fbbf24" : "#34d399";
  const esInfo = esMeta(es?.status);

  return (
    <div className="grid h-16 shrink-0 grid-cols-5 gap-2">
      <Tile
        label="Kafka lag"
        value={lag != null ? fmtInt(lag) : "—"}
        color={lagColor}
        spark={lagSeries}
      />
      <Tile
        label="Elasticsearch"
        value={esInfo.label}
        color={esInfo.color}
        sub={es ? `${es.index} · ${fmtInt(es.docs_total)} docs` : undefined}
      />
      <Tile
        label="Buffer fill"
        value={fillPct != null ? `${fillPct}%` : "—"}
        color={fillColor}
        bar={fillPct != null ? fillPct : 0}
        sub={
          bufferFill != null && bufferCapacity != null
            ? `${fmtInt(bufferFill)} / ${fmtInt(bufferCapacity)}`
            : undefined
        }
      />
      <Tile
        label="In flight"
        value={inFlight != null ? fmtInt(inFlight) : "—"}
        color="#60a5fa"
      />
      <Tile
        label="Retrying"
        value={retrying != null ? fmtInt(retrying) : "—"}
        color={retrying != null && retrying > 0 ? "#fbbf24" : "#34d399"}
      />
    </div>
  );
}
