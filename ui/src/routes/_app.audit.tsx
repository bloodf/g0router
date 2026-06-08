import { createFileRoute } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { apiFetch, normalizeListResponse } from "@/lib/api/client";
import { PageHeader } from "@/components/common/PageHeader";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Skeleton } from "@/components/ui/skeleton";
import { Icon } from "@/components/common/Icon";
import { StatusBadge } from "@/components/common/StatusBadge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useVisibleWindow } from "@/lib/hooks/useVisibleWindow";
import { ErrorState } from "@/components/common/Skeletons";

export const Route = createFileRoute("/_app/audit")({
  component: AuditPage,
});

interface BackendAuditLog {
  id: string;
  timestamp: string;
  actor_api_key_id: string;
  action: string;
  target: string;
  details?: string;
}

type ActionFilter =
  | "all"
  | "copy_key"
  | "regenerate_key"
  | "revoke_key"
  | "enable_key"
  | "create_key"
  | "delete_key"
  | "export_keys"
  | "other";

type TimeFilter = "all" | "1h" | "24h" | "7d" | "30d";

const ACTION_OPTIONS: { value: ActionFilter; label: string }[] = [
  { value: "all", label: "All actions" },
  { value: "copy_key", label: "Copy key" },
  { value: "regenerate_key", label: "Regenerate key" },
  { value: "revoke_key", label: "Revoke key" },
  { value: "enable_key", label: "Enable key" },
  { value: "create_key", label: "Create key" },
  { value: "delete_key", label: "Delete key" },
  { value: "export_keys", label: "Export keys" },
  { value: "other", label: "Other" },
];

const TIME_OPTIONS: { value: TimeFilter; label: string }[] = [
  { value: "all", label: "All time" },
  { value: "1h", label: "Last hour" },
  { value: "24h", label: "Last 24h" },
  { value: "7d", label: "Last 7 days" },
  { value: "30d", label: "Last 30 days" },
];

const KEY_ACTIONS = new Set<string>([
  "copy_key",
  "regenerate_key",
  "revoke_key",
  "enable_key",
  "create_key",
  "delete_key",
  "export_keys",
]);

function actionVariant(action: string): "success" | "warning" | "danger" | "info" | "muted" {
  if (action === "copy_key" || action === "export_keys") return "info";
  if (action === "regenerate_key" || action === "enable_key" || action === "create_key")
    return "success";
  if (action === "revoke_key" || action === "delete_key") return "danger";
  return "muted";
}

function formatAction(a: string) {
  return a.replace(/_/g, " ");
}

function formatTimestamp(iso: string) {
  const d = new Date(iso);
  return d.toLocaleString(undefined, {
    year: "numeric",
    month: "short",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

function AuditPage() {
  const [query, setQuery] = useState("");
  const [actor, setActor] = useState<string>("all");
  const [action, setAction] = useState<ActionFilter>("all");
  const [time, setTime] = useState<TimeFilter>("all");
  const [keyEventsOnly, setKeyEventsOnly] = useState(false);

  const { data, isLoading, isError, error, refetch } = useQuery<{ items: BackendAuditLog[]; total: number }>({
    queryKey: ["audit", { limit: 500 }],
    queryFn: async () => {
      const raw = await apiFetch("/api/audit?limit=500");
      return normalizeListResponse<BackendAuditLog>(raw);
    },
    refetchInterval: 15_000,
  });

  const items = data?.items ?? [];

  const actors = useMemo(() => {
    const set = new Set<string>();
    items.forEach((i) => {
      if (i.actor_api_key_id) set.add(i.actor_api_key_id);
    });
    return Array.from(set).sort();
  }, [items]);

  const filtered = useMemo(() => {
    const now = Date.now();
    const windowMs =
      time === "1h"
        ? 60 * 60_000
        : time === "24h"
          ? 24 * 60 * 60_000
          : time === "7d"
            ? 7 * 86_400_000
            : time === "30d"
              ? 30 * 86_400_000
              : null;
    const q = query.trim().toLowerCase();
    return items.filter((i) => {
      if (keyEventsOnly && !KEY_ACTIONS.has(i.action)) return false;
      if (actor !== "all" && i.actor_api_key_id !== actor) return false;
      if (action !== "all") {
        if (action === "other") {
          if (KEY_ACTIONS.has(i.action)) return false;
        } else if (i.action !== action) return false;
      }
      if (windowMs !== null) {
        if (now - new Date(i.timestamp).getTime() > windowMs) return false;
      }
      if (q) {
        const hay = `${i.actor_api_key_id} ${i.action} ${i.target} ${i.details ?? ""}`.toLowerCase();
        if (!hay.includes(q)) return false;
      }
      return true;
    });
  }, [items, query, actor, action, time, keyEventsOnly]);

  const { visible, hasMore, sentinelRef, loadMore } = useVisibleWindow(50, filtered.length);
  const visibleRows = filtered.slice(0, visible);

  const exportCsv = () => {
    const escape = (v: unknown) => {
      const s = v == null ? "" : String(v);
      return /[",\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s;
    };
    const headers = ["timestamp", "actor", "action", "target", "details"];
    const rows = filtered.map((i) =>
      [i.timestamp, i.actor_api_key_id, i.action, i.target, i.details ?? ""].map(escape).join(","),
    );
    const csv = [headers.join(","), ...rows].join("\n");
    const blob = new Blob([csv], { type: "text/csv;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `audit-log-${new Date().toISOString().replace(/[:.]/g, "-")}.csv`;
    document.body.appendChild(a);
    a.click();
    a.remove();
    URL.revokeObjectURL(url);
  };

  const resetFilters = () => {
    setQuery("");
    setActor("all");
    setAction("all");
    setTime("all");
    setKeyEventsOnly(false);
  };

  const hasFilters =
    !!query || actor !== "all" || action !== "all" || time !== "all" || keyEventsOnly;

  return (
    <div>
      <PageHeader
        title="Audit Logs"
        description="Every administrative action — including endpoint API key copy, regenerate, revoke, enable, and export events."
        icon="history"
        actions={
          <Button variant="outline" size="sm" onClick={exportCsv} disabled={!filtered.length}>
            <Icon name="download" size={14} className="mr-1.5" />
            Export CSV
          </Button>
        }
      />

      {isError && (
        <ErrorState
          title="Couldn’t load audit logs"
          error={error}
          onRetry={() => refetch()}
          className="mb-4"
        />
      )}

      {/* Filters */}
      <Card className="card-elev border-border p-4 mb-4">
        <div className="grid gap-3 md:grid-cols-[1fr_180px_200px_160px_auto]">
          <div className="relative">
            <Icon
              name="search"
              size={16}
              className="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted"
            />
            <Input
              placeholder="Search actor, action, target, details…"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              className="pl-9"
            />
          </div>

          <Select value={actor} onValueChange={setActor}>
            <SelectTrigger>
              <SelectValue placeholder="Actor" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">All actors</SelectItem>
              {actors.map((a) => (
                <SelectItem key={a} value={a}>
                  {a}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          <Select value={action} onValueChange={(v) => setAction(v as ActionFilter)}>
            <SelectTrigger>
              <SelectValue placeholder="Action" />
            </SelectTrigger>
            <SelectContent>
              {ACTION_OPTIONS.map((o) => (
                <SelectItem key={o.value} value={o.value}>
                  {o.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          <Select value={time} onValueChange={(v) => setTime(v as TimeFilter)}>
            <SelectTrigger>
              <SelectValue placeholder="Time" />
            </SelectTrigger>
            <SelectContent>
              {TIME_OPTIONS.map((o) => (
                <SelectItem key={o.value} value={o.value}>
                  {o.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          <div className="flex items-center gap-2">
            <Button
              variant={keyEventsOnly ? "default" : "outline"}
              size="sm"
              onClick={() => setKeyEventsOnly((v) => !v)}
              title="Toggle to show only endpoint API key events"
            >
              <Icon name="key" size={14} className="mr-1.5" />
              Key events
            </Button>
            {hasFilters && (
              <Button variant="ghost" size="sm" onClick={resetFilters}>
                Reset
              </Button>
            )}
          </div>
        </div>
        <div className="mt-3 text-xs text-text-muted">
          Showing {filtered.length} of {items.length} entries
          {hasFilters && " · filtered"}
        </div>
      </Card>

      {/* Table */}
      <Card className="border-border overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-surface-2 text-[11px] uppercase tracking-wider text-text-muted text-left">
            <tr>
              <th className="px-4 py-2 w-[200px]">Timestamp</th>
              <th className="px-4 py-2 w-[140px]">Actor</th>
              <th className="px-4 py-2 w-[180px]">Action</th>
              <th className="px-4 py-2">Target</th>
              <th className="px-4 py-2">Details</th>
            </tr>
          </thead>
          <tbody>
            {isLoading &&
              Array.from({ length: 6 }).map((_, i) => (
                <tr key={i} className="border-t border-border">
                  <td className="px-4 py-3" colSpan={5}>
                    <Skeleton className="h-4 w-full" />
                  </td>
                </tr>
              ))}

            {!isLoading &&
              visibleRows.map((row) => (
                <tr key={row.id} className="border-t border-border hover:bg-surface-2/40">
                  <td className="px-4 py-2 font-mono text-xs text-text-muted whitespace-nowrap">
                    {formatTimestamp(row.timestamp)}
                  </td>
                  <td className="px-4 py-2">
                    <span className="inline-flex items-center gap-1.5">
                      <span className="w-6 h-6 rounded-full bg-surface-2 flex items-center justify-center text-[10px] font-semibold uppercase">
                        {(row.actor_api_key_id ?? "N/A").slice(0, 2)}
                      </span>
                      {row.actor_api_key_id ?? "N/A"}
                    </span>
                  </td>
                  <td className="px-4 py-2">
                    <StatusBadge variant={actionVariant(row.action)} dot>
                      {formatAction(row.action)}
                    </StatusBadge>
                  </td>
                  <td className="px-4 py-2 font-mono text-xs">{row.target}</td>
                  <td className="px-4 py-2 text-xs text-text-muted">{row.details ?? "—"}</td>
                </tr>
              ))}

            {!isLoading && !filtered.length && (
              <tr>
                <td colSpan={5} className="py-10 text-center text-text-muted text-sm">
                  <Icon name="filter_alt_off" size={28} className="mx-auto mb-2 opacity-60" />
                  <div>No audit entries match the current filters.</div>
                  {hasFilters && (
                    <Button variant="link" size="sm" onClick={resetFilters} className="mt-1">
                      Clear filters
                    </Button>
                  )}
                </td>
              </tr>
            )}
          </tbody>
        </table>
        {hasMore && (
          <div
            ref={sentinelRef}
            className="flex items-center justify-between border-t border-border bg-surface-2/30 px-4 py-2 text-xs text-text-muted"
          >
            <span>
              Showing {visibleRows.length} of {filtered.length}
            </span>
            <Button variant="outline" size="sm" onClick={loadMore}>
              Load more
            </Button>
          </div>
        )}
      </Card>
    </div>
  );
}
