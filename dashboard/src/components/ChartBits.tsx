import { fmtClock, fmtRate } from "../lib/format";

interface TooltipItem {
  name?: string;
  value?: number | string;
  color?: string;
  dataKey?: string | number;
}

interface ChartTooltipProps {
  active?: boolean;
  payload?: TooltipItem[];
  label?: number;
}

export function ChartTooltip({ active, payload, label }: ChartTooltipProps) {
  if (!active || !payload || payload.length === 0) return null;
  return (
    <div className="rounded-md border border-edge2 bg-panel2 px-2.5 py-1.5 text-[11px] shadow-xl">
      <div className="mb-1 font-mono text-faint">{fmtClock(label ?? 0)}</div>
      {payload.map((p) => (
        <div key={String(p.dataKey)} className="flex items-center gap-2 py-0.5">
          <span className="h-1.5 w-1.5 shrink-0 rounded-full" style={{ background: p.color }} />
          <span className="text-dim">{p.name}</span>
          <span className="ml-auto pl-4 font-mono text-bright">
            {fmtRate(Number(p.value ?? 0))}
          </span>
        </div>
      ))}
    </div>
  );
}

/** Honest placeholder shown while the first samples are still arriving. */
export function EmptyOverlay({ label }: { label: string }) {
  return (
    <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
      <div className="flex flex-col items-center gap-1.5 rounded border border-edge bg-panel/90 px-4 py-2.5">
        <span className="animate-blink h-2 w-2 rounded-full bg-cyan" />
        <span className="text-[11px] text-dim">{label}</span>
      </div>
    </div>
  );
}
