interface Props {
  data: number[];
  color: string;
  height?: number;
}

/** Tiny zero-dependency area sparkline (no chart library overhead). */
export function Sparkline({ data, color, height = 24 }: Props) {
  const w = 100;
  const h = height;
  const max = Math.max(...data, 1e-9);
  const min = Math.min(...data, 0);
  const range = max - min || 1;
  const pts = data.map(
    (v, i) => `${(i / Math.max(1, data.length - 1)) * w},${h - ((v - min) / range) * h}`,
  );
  const line = "M" + pts.join(" L");
  const area = `${line} L ${w},${h} L 0,${h} Z`;
  return (
    <svg viewBox={`0 0 ${w} ${h}`} className="h-full w-full" preserveAspectRatio="none">
      <path d={area} fill={color} opacity={0.14} />
      <path d={line} fill="none" stroke={color} strokeWidth={1.5} vectorEffect="non-scaling-stroke" />
    </svg>
  );
}
