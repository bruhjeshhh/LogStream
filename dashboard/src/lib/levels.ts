export const LEVELS = ["trace", "debug", "info", "warn", "error", "fatal"] as const;

export interface LevelMeta {
  label: string;
  text: string;
  bg: string;
}

export const DEFAULT_LEVEL_META: LevelMeta = {
  label: "LOG",
  text: "#93a1b5",
  bg: "rgba(147,161,181,0.14)",
};

const MAP: Record<string, LevelMeta> = {
  trace: { label: "TRACE", text: "#7c8aa0", bg: "rgba(124,138,160,0.14)" },
  debug: { label: "DEBUG", text: "#60a5fa", bg: "rgba(96,165,250,0.14)" },
  info: { label: "INFO", text: "#38bdf8", bg: "rgba(56,189,248,0.14)" },
  warn: { label: "WARN", text: "#fbbf24", bg: "rgba(251,191,36,0.15)" },
  error: { label: "ERROR", text: "#fb7185", bg: "rgba(251,113,133,0.16)" },
  fatal: { label: "FATAL", text: "#f87171", bg: "rgba(248,113,113,0.2)" },
};

export function levelMeta(level: string): LevelMeta {
  return MAP[level.toLowerCase()] ?? DEFAULT_LEVEL_META;
}
