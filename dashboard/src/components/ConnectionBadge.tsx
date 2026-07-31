import type { ConnStatus } from "../lib/stream";

const META: Record<ConnStatus, { label: string; text: string; dot: string; pulse: boolean }> = {
  connecting: { label: "CONNECTING", text: "text-dim", dot: "bg-dim", pulse: false },
  live: { label: "LIVE", text: "text-mint", dot: "bg-mint", pulse: true },
  polling: { label: "POLLING", text: "text-amber", dot: "bg-amber", pulse: true },
  error: { label: "OFFLINE", text: "text-red", dot: "bg-red", pulse: false },
};

export function ConnectionBadge({ status }: { status: ConnStatus }) {
  const m = META[status];
  return (
    <span
      className={`flex items-center gap-1.5 rounded border border-edge bg-panel2 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider ${m.text}`}
    >
      <span className={`h-1.5 w-1.5 rounded-full ${m.dot} ${m.pulse ? "animate-blink" : ""}`} />
      {m.label}
    </span>
  );
}
