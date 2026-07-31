import {
  Area,
  CartesianGrid,
  ComposedChart,
  Line,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import type { RateSample } from "../hooks/useLiveMetrics";
import { fmtClock, fmtRate } from "../lib/format";
import { ChartTooltip, EmptyOverlay } from "./ChartBits";

interface Props {
  series: RateSample[];
}

export function ThroughputChart({ series }: Props) {
  return (
    <div className="relative h-full w-full">
      <ResponsiveContainer width="100%" height="100%">
        <ComposedChart data={series} margin={{ top: 8, right: 4, left: 0, bottom: 0 }}>
          <defs>
            <linearGradient id="gAccepted" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#22d3ee" stopOpacity={0.32} />
              <stop offset="100%" stopColor="#22d3ee" stopOpacity={0.02} />
            </linearGradient>
          </defs>
          <CartesianGrid stroke="#1b2534" strokeDasharray="3 3" vertical={false} />
          <XAxis
            dataKey="t"
            type="number"
            domain={["dataMin", "dataMax"]}
            tickFormatter={(v: number) => fmtClock(v)}
            stroke="#2a3a52"
            tick={{ fontSize: 10, fill: "#5c6c83" }}
            tickLine={false}
            axisLine={false}
            minTickGap={48}
          />
          <YAxis
            yAxisId="left"
            stroke="#2a3a52"
            tick={{ fontSize: 10, fill: "#5c6c83" }}
            tickFormatter={(v: number) => fmtRate(v)}
            tickLine={false}
            axisLine={false}
            width={42}
          />
          <YAxis
            yAxisId="right"
            orientation="right"
            stroke="#2a3a52"
            tick={{ fontSize: 10, fill: "#5c6c83" }}
            tickFormatter={(v: number) => fmtRate(v)}
            tickLine={false}
            axisLine={false}
            width={36}
          />
          <Tooltip content={<ChartTooltip />} />
          <Area
            yAxisId="left"
            type="monotone"
            dataKey="accepted"
            name="ingest /s"
            stroke="#22d3ee"
            strokeWidth={1.6}
            fill="url(#gAccepted)"
            dot={false}
            isAnimationActive={false}
          />
          <Line
            yAxisId="left"
            type="monotone"
            dataKey="processed"
            name="drain /s"
            stroke="#34d399"
            strokeWidth={1.6}
            dot={false}
            isAnimationActive={false}
          />
          <Line
            yAxisId="right"
            type="monotone"
            dataKey="rejected"
            name="rejected /s"
            stroke="#f87171"
            strokeWidth={1.2}
            strokeDasharray="4 3"
            dot={false}
            isAnimationActive={false}
          />
        </ComposedChart>
      </ResponsiveContainer>
      {series.length < 2 && <EmptyOverlay label="waiting for the first metrics…" />}
    </div>
  );
}
