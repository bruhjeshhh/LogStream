import type { ReactNode } from "react";

interface Props {
  title: string;
  right?: ReactNode;
  children: ReactNode;
  className?: string;
}

export function Panel({ title, right, children, className }: Props) {
  return (
    <section
      className={`flex flex-col overflow-hidden rounded border border-edge bg-panel ${className ?? ""}`}
    >
      <header className="flex h-8 shrink-0 items-center justify-between border-b border-edge bg-panel2/60 px-3">
        <h2 className="text-[10px] font-semibold uppercase tracking-wider text-dim">{title}</h2>
        {right}
      </header>
      <div className="min-h-0 flex-1">{children}</div>
    </section>
  );
}
