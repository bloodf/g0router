import { useMemo } from "react";
import { cn } from "@/lib/utils";
import { useVisibleWindow } from "@/lib/hooks/useVisibleWindow";
import {
  formatCountdown,
  formatResetTimeDisplay,
  getColorClasses,
  type QuotaRow,
} from "./quota-utils";

const PAGE_SIZE = 15;

type SortMode = "default" | "remaining-asc" | "remaining-desc";

function sortQuotas(quotas: QuotaRow[], sortMode: SortMode) {
  if (sortMode === "remaining-asc") {
    return [...quotas].sort(
      (a, b) => a.remaining - b.remaining || a.name.localeCompare(b.name),
    );
  }
  if (sortMode === "remaining-desc") {
    return [...quotas].sort(
      (a, b) => b.remaining - a.remaining || a.name.localeCompare(b.name),
    );
  }
  return quotas;
}

export function QuotaTable({
  quotas,
  sortMode = "default",
  showSortLabel = false,
}: {
  quotas: QuotaRow[];
  sortMode?: SortMode;
  showSortLabel?: boolean;
}) {
  const sorted = useMemo(() => sortQuotas(quotas, sortMode), [quotas, sortMode]);
  const { visible, hasMore, sentinelRef, loadMore } = useVisibleWindow(
    PAGE_SIZE,
    sorted.length,
  );

  if (!quotas.length) return null;

  const rows = sorted.slice(0, visible);

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between text-[11px] text-text-muted">
        <span>
          {sorted.length} quota{sorted.length !== 1 ? "s" : ""}
        </span>
        {showSortLabel && <span>Sorted by account remaining</span>}
      </div>

      <div className="rounded-lg border border-border overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-surface-2 text-[10px] uppercase text-text-muted">
            <tr>
              <th className="text-left py-1.5 px-2.5 font-semibold">Account</th>
              <th className="text-left py-1.5 px-2.5 font-semibold">Usage</th>
              <th className="text-right py-1.5 px-2.5 font-semibold">Reset</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((q) => {
              const colors = getColorClasses(q.remaining);
              const countdown = formatCountdown(q.resetAt);
              const resetDisplay = formatResetTimeDisplay(q.resetAt);
              return (
                <tr
                  key={q.name + q.resetAt}
                  className="border-t border-border last:border-b-0 hover:bg-surface-2/50"
                >
                  <td className="py-1.5 px-2.5">
                    <div className="flex items-center gap-1.5">
                      <span>{colors.emoji}</span>
                      <span className="truncate">{q.name}</span>
                    </div>
                  </td>
                  <td className="py-1.5 px-2.5">
                    <div className="flex items-center gap-2">
                      <div className="h-1.5 w-20 rounded-full bg-surface-2 overflow-hidden">
                        <div
                          className={cn("h-full", colors.bg)}
                          style={{ width: `${q.remaining}%` }}
                        />
                      </div>
                      <span className="tabular-nums text-[11px] text-text-muted">
                        {q.used.toLocaleString()}/
                        {q.total > 0 ? q.total.toLocaleString() : "∞"}
                      </span>
                      <span className={cn("text-[11px] font-semibold tabular-nums", colors.text)}>
                        {q.unlimited ? "∞" : `${q.remaining}%`}
                      </span>
                    </div>
                  </td>
                  <td className="py-1.5 px-2.5 text-right text-[11px] text-text-muted">
                    {countdown !== "-" ? `in ${countdown}` : resetDisplay ?? "N/A"}
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      {hasMore ? (
        <div ref={sentinelRef} className="flex items-center justify-between text-[11px] text-text-muted">
          <span>
            Showing {rows.length} of {sorted.length}
          </span>
          <button
            type="button"
            onClick={loadMore}
            className="rounded-md border border-border px-2 py-0.5 hover:bg-surface-2"
          >
            Load more
          </button>
        </div>
      ) : sorted.length > PAGE_SIZE ? (
        <div className="text-right text-[11px] text-text-muted/70">End of list</div>
      ) : null}
    </div>
  );
}
