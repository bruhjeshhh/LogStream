import { Sparkline } from "./Sparkline";

interface Props {
  label: string;
  value: string;
  unit?: string;
  sub?: string;
  color: string;
  spark?: number[];
}

export function MetricCard({ label, value, unit, sub, color, spark }: Props) {
  return (
    <div className="flex flex-col justify-between rounded border border-edge bg-panel px-3 py-2">
      <div className="flex items-baseline justify-between">
        <span className="text-[9px] font-semibold uppercase tracking-wider text-faint">{label}</span>
        {sub && <span className="font-mono text-[9px] text-faint">{sub}</span>}
      </div>
      <div className="flex items-end justify-between gap-2">
        <div className="font-mono text-[22px] font-semibold leading-none tabular-nums" style={{ color }}>
          {value}
          {unit && <span className="ml-1 text-[11px] font-medium text-dim">{unit}</span>}
        </div>
        {spark && spark.length > 1 && (
          <div className="h-6 w-20 shrink-0">
            <Sparkline data={spark} color={color} />
          </div>
        )}
      </div>
    </div>
  );
}
