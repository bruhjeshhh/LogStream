import type { ConnStatus } from "../lib/stream";
import { fmtRel } from "../lib/format";
import { useNow } from "../hooks/useNow";
import { ConnectionBadge } from "./ConnectionBadge";

interface Props {
  status: ConnStatus;
  lastUpdate: number;
}

export function Header({ status, lastUpdate }: Props) {
  const now = useNow(1000);
  return (
    <header className="flex h-11 shrink-0 items-center justify-between rounded border border-edge bg-panel px-4">
      <div className="flex items-baseline gap-3">
        <span className="text-[15px] font-bold tracking-tight text-bright">
          LogStream
          <span className="ml-2 text-[11px] font-semibold text-cyan">λ</span>
        </span>
        <span className="text-[10px] uppercase tracking-[0.18em] text-faint">pipeline monitor</span>
      </div>
      <div className="flex items-center gap-3">
        <span className="font-mono text-[11px] text-faint">
          {lastUpdate ? `updated ${fmtRel(lastUpdate, now)}` : "waiting for data"}
        </span>
        <ConnectionBadge status={status} />
      </div>
    </header>
  );
}
