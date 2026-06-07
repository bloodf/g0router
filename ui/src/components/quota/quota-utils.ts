export function calculatePercentage(used: number, total: number) {
  if (!total || total <= 0) return 0;
  return Math.max(0, Math.min(100, Math.round((used / total) * 100)));
}

export function getRemainingPercentage(q: { used: number; limit: number }) {
  if (!q.limit || q.limit <= 0) return 100;
  const used = calculatePercentage(q.used, q.limit);
  return Math.max(0, 100 - used);
}

export function getColorClasses(remainingPct: number) {
  if (remainingPct > 70) {
    return {
      text: "text-success",
      bg: "bg-success",
      bgLight: "bg-success/10",
      emoji: "🟢",
    } as const;
  }
  if (remainingPct >= 30) {
    return {
      text: "text-warning",
      bg: "bg-warning",
      bgLight: "bg-warning/15",
      emoji: "🟡",
    } as const;
  }
  return {
    text: "text-destructive",
    bg: "bg-destructive",
    bgLight: "bg-destructive/10",
    emoji: "🔴",
  } as const;
}

export function formatCountdown(resetAt?: string | null) {
  if (!resetAt) return "-";
  const ms = new Date(resetAt).getTime() - Date.now();
  if (ms <= 0) return "now";
  const s = Math.floor(ms / 1000);
  const d = Math.floor(s / 86400);
  const h = Math.floor((s % 86400) / 3600);
  const m = Math.floor((s % 3600) / 60);
  if (d > 0) return `${d}d ${h}h`;
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m`;
}

export function formatResetTimeDisplay(resetAt?: string | null) {
  if (!resetAt) return null;
  try {
    const date = new Date(resetAt);
    const now = new Date();
    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
    const tomorrow = new Date(today);
    tomorrow.setDate(tomorrow.getDate() + 1);
    let dayStr = "";
    if (date >= today && date < tomorrow) dayStr = "Today";
    else if (date >= tomorrow && date.getTime() < tomorrow.getTime() + 86_400_000)
      dayStr = "Tomorrow";
    else dayStr = date.toLocaleDateString("en-US", { month: "short", day: "numeric" });
    const timeStr = date.toLocaleTimeString("en-US", {
      hour: "numeric",
      minute: "2-digit",
      hour12: true,
    });
    return `${dayStr}, ${timeStr}`;
  } catch {
    return null;
  }
}

export interface QuotaRow {
  name: string;
  used: number;
  total: number;
  unlimited?: boolean;
  resetAt?: string;
  remaining: number; // remaining percentage 0-100
  unit?: string;
}
