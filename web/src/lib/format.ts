import type { BlastVerdict, ChangeClass, Severity } from "../api/types";

export const severityColor: Record<Severity, string> = {
  critical: "text-critical",
  high: "text-high",
  medium: "text-medium",
  low: "text-low",
  info: "text-info",
};

export const severityDot: Record<Severity, string> = {
  critical: "bg-critical",
  high: "bg-high",
  medium: "bg-medium",
  low: "bg-low",
  info: "bg-info",
};

export const classLabel: Record<ChangeClass, string> = {
  widened: "Widened",
  narrowed: "Narrowed",
  neutral: "Neutral",
  unknown: "Unknown",
};

export const verdictColor: Record<BlastVerdict, string> = {
  will_break: "text-critical",
  ready: "text-low",
  unknown: "text-muted",
};

export const verdictLabel: Record<BlastVerdict, string> = {
  will_break: "Will break",
  ready: "Ready",
  unknown: "Unknown",
};

/** Human-readable duration from seconds, e.g. 795600 -> "9d 5h". */
export function humanizeDuration(totalSeconds: number): string {
  if (totalSeconds <= 0) return "0s";
  const d = Math.floor(totalSeconds / 86400);
  const h = Math.floor((totalSeconds % 86400) / 3600);
  const m = Math.floor((totalSeconds % 3600) / 60);
  const parts: string[] = [];
  if (d) parts.push(`${d}d`);
  if (h) parts.push(`${h}h`);
  if (m && !d) parts.push(`${m}m`);
  return parts.join(" ") || `${totalSeconds}s`;
}

export function relativeTime(iso: string): string {
  const then = new Date(iso).getTime();
  const diff = Date.now() - then;
  const mins = Math.round(diff / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.round(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  return `${Math.round(hours / 24)}d ago`;
}

export function ttlLabel(sec?: number | null): string {
  if (sec == null) return "—";
  return humanizeDuration(sec);
}
