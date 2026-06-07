import { cn } from "@/lib/utils";
import {
  formatCountdown,
  formatResetTimeDisplay,
  getColorClasses,
  type QuotaRow,
} from "./quota-utils";

export function QuotaProgressBar({ row }: { row: QuotaRow }) {
  const colors = getColorClasses(row.remaining);
  const countdown = formatCountdown(row.resetAt);
  const resetDisplay = formatResetTimeDisplay(row.resetAt);
  const remaining = row.remaining;

  return (
    <div className="space-y-1.5">
      <div className="flex items-center justify-between gap-2">
        <span className="text-sm font-medium truncate">{row.name}</span>
        <span className={cn("text-xs font-semibold tabular-nums flex items-center gap-1", colors.text)}>
          <span>{colors.emoji}</span>
          {row.unlimited ? "∞" : `${remaining}%`}
        </span>
      </div>

      {!row.unlimited && (
        <div className="h-2 rounded-full bg-surface-2 overflow-hidden">
          <div
            className={cn("h-full rounded-full transition-all", colors.bg)}
            style={{ width: `${remaining}%` }}
          />
        </div>
      )}

      <div className="flex items-center justify-between text-[11px] text-text-muted">
        <span className="tabular-nums">
          {row.used.toLocaleString()} /{" "}
          {row.total > 0 ? row.total.toLocaleString() : "∞"} {row.unit ?? "requests"}
        </span>
        {countdown !== "-" && <span>• Reset in {countdown}</span>}
      </div>

      {resetDisplay && (
        <div className="text-[10px] text-text-subtle">Reset at {resetDisplay}</div>
      )}
    </div>
  );
}
