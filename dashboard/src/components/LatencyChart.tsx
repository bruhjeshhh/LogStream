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
import { fmtClock, fmtMs } from "../lib/format";
import { ChartTooltip, EmptyOverlay } from "./ChartBits";

interface Props {
  series: RateSample[];
}

export function LatencyChart({ series }: Props) {
  return (
    <div className="relative h-full w-full">
      <ResponsiveContainer width="100%" height="100%">
        <ComposedChart data={series} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
          <defs>
            <linearGradient id="gP99" x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor="#fbbf24" stopOpacity={0.32} />
              <stop offset="100%" stopColor="#fbbf24" stopOpacity={0.02} />
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
            stroke="#2a3a52"
            tick={{ fontSize: 10, fill: "#5c6c83" }}
            tickFormatter={(v: number) => fmtMs(v)}
            tickLine={false}
            axisLine={false}
            width={46}
            domain={[0, "auto"]}
          />
          <Tooltip content={<ChartTooltip />} />
          <Line
            type="monotone"
            dataKey="p50"
            name="p50"
            stroke="#5c6c83"
            strokeWidth={1.2}
            strokeDasharray="3 3"
            dot={false}
            isAnimationActive={false}
          />
          <Area
            type="monotone"
            dataKey="p99"
            name="p99"
            stroke="#fbbf24"
            strokeWidth={1.6}
            fill="url(#gP99)"
            dot={false}
            isAnimationActive={false}
          />
        </ComposedChart>
      </ResponsiveContainer>
      {series.length < 2 && <EmptyOverlay label="waiting for latency samples…" />}
    </div>
  );
}
