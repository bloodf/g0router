import { createFileRoute } from "@tanstack/react-router";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useMemo, useState } from "react";
import { apiFetch } from "@/lib/api/client";
import { PageHeader } from "@/components/common/PageHeader";
import { CardsGridSkeleton, ErrorState } from "@/components/common/Skeletons";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Icon } from "@/components/common/Icon";
import { ProviderLimitCard } from "@/components/quota/ProviderLimitCard";
import {
  getRemainingPercentage,
  type QuotaRow,
} from "@/components/quota/quota-utils";
import type { Quota } from "@/lib/types";

type SortMode = "default" | "remaining-asc" | "remaining-desc";

interface ProviderGroup {
  provider: string;
  name: string;
  plan?: Quota["plan"];
  quotas: QuotaRow[];
  message?: string;
  error?: string;
}

function groupByProvider(quotas: Quota[]): ProviderGroup[] {
  const groups = new Map<string, ProviderGroup>();
  for (const q of quotas) {
    const g =
      groups.get(q.provider) ??
      {
        provider: q.provider,
        name: q.connection_name,
        plan: q.plan,
        quotas: [] as QuotaRow[],
        message: q.message,
        error: q.error,
      };
    if (!q.message && !q.error) {
      g.quotas.push({
        name: q.account_label ?? q.connection_name,
        used: q.used,
        total: q.limit,
        unlimited: !q.limit,
        resetAt: q.reset_at,
        remaining: getRemainingPercentage(q),
        unit: q.unit,
      });
    } else if (q.message) {
      g.message = q.message;
    } else if (q.error) {
      g.error = q.error;
    }
    groups.set(q.provider, g);
  }
  return Array.from(groups.values());
}

function sortGroups(groups: ProviderGroup[], mode: SortMode) {
  if (mode === "default") {
    return [...groups].sort((a, b) => a.provider.localeCompare(b.provider));
  }
  return [...groups].sort((a, b) => {
    const minA = Math.min(...(a.quotas.length ? a.quotas.map((q) => q.remaining) : [101]));
    const minB = Math.min(...(b.quotas.length ? b.quotas.map((q) => q.remaining) : [101]));
    return mode === "remaining-asc" ? minA - minB : minB - minA;
  });
}

export const Route = createFileRoute("/_app/quota")({
  component: QuotaPage,
});

function QuotaPage() {
  const qc = useQueryClient();
  const { data = [], isLoading, isFetching, isError, error, refetch } = useQuery<Quota[]>({
    queryKey: ["quotas"],
    queryFn: () => apiFetch("/api/quota"),
    refetchInterval: 30_000,
  });

  const [search, setSearch] = useState("");
  const [providerFilter, setProviderFilter] = useState<string>("all");
  const [sortMode, setSortMode] = useState<SortMode>("remaining-asc");

  const groups = useMemo(() => groupByProvider(data), [data]);

  const providerOptions = useMemo(
    () => Array.from(new Set(groups.map((g) => g.provider))).sort(),
    [groups],
  );

  const visible = useMemo(() => {
    const q = search.trim().toLowerCase();
    const filtered = groups.filter((g) => {
      if (providerFilter !== "all" && g.provider !== providerFilter) return false;
      if (!q) return true;
      return (
        g.provider.toLowerCase().includes(q) ||
        g.name.toLowerCase().includes(q) ||
        g.quotas.some((row) => row.name.toLowerCase().includes(q))
      );
    });
    return sortGroups(filtered, sortMode);
  }, [groups, providerFilter, sortMode, search]);

  const totals = useMemo(() => {
    let healthy = 0;
    let warn = 0;
    let critical = 0;
    let accounts = 0;
    for (const g of groups) {
      for (const r of g.quotas) {
        accounts += 1;
        if (r.remaining > 70) healthy += 1;
        else if (r.remaining >= 30) warn += 1;
        else critical += 1;
      }
    }
    return { healthy, warn, critical, accounts };
  }, [groups]);

  return (
    <div>
      <PageHeader
        title="Quota Tracker"
        description="Per-provider usage with remaining capacity and reset countdown — auto-refreshes every 30s."
        icon="data_usage"
        actions={
          <Button
            variant="outline"
            size="sm"
            onClick={() => refetch()}
            disabled={isFetching}
            className="gap-1.5"
          >
            <Icon
              name="refresh"
              size={14}
              className={isFetching ? "animate-spin" : ""}
            />
            Refresh all
          </Button>
        }
      />

      {/* Summary strip */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-2 mb-4">
        <SummaryStat label="Accounts" value={totals.accounts} tone="muted" />
        <SummaryStat label="Healthy" value={totals.healthy} tone="success" dot />
        <SummaryStat label="Warning" value={totals.warn} tone="warning" dot />
        <SummaryStat label="Critical" value={totals.critical} tone="danger" dot />
      </div>

      {/* Toolbar */}
      <div className="flex flex-col sm:flex-row gap-2 mb-4">
        <div className="relative flex-1">
          <Icon
            name="search"
            size={16}
            className="absolute left-2.5 top-1/2 -translate-y-1/2 text-text-muted"
          />
          <Input
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search providers or accounts…"
            aria-label="Search providers or accounts"
            className="pl-8 h-9"
          />
        </div>
        <Select value={providerFilter} onValueChange={setProviderFilter}>
          <SelectTrigger className="h-9 w-full sm:w-[180px]" aria-label="Provider filter">
            <SelectValue placeholder="All providers" />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">All providers</SelectItem>
            {providerOptions.map((p) => (
              <SelectItem key={p} value={p} className="capitalize">
                {p}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        <Select value={sortMode} onValueChange={(v) => setSortMode(v as SortMode)}>
          <SelectTrigger className="h-9 w-full sm:w-[200px]" aria-label="Sort mode">
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="remaining-asc">Lowest remaining first</SelectItem>
            <SelectItem value="remaining-desc">Highest remaining first</SelectItem>
            <SelectItem value="default">Alphabetical</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {/* Grid */}
      {isLoading ? (
        <CardsGridSkeleton
          count={6}
          height="h-56"
          className="grid sm:grid-cols-2 xl:grid-cols-3 gap-3"
        />
      ) : isError ? (
        <ErrorState
          title="Couldn’t load quotas"
          error={error}
          onRetry={() => refetch()}
        />
      ) : visible.length === 0 ? (
        <div className="border border-dashed border-border rounded-xl p-12 flex flex-col items-center gap-2 text-text-muted">
          <Icon name="data_usage" size={32} />
          <div className="font-medium">No quota data</div>
          <div className="text-xs">Try clearing filters or connect a quota-aware provider.</div>
        </div>
      ) : (
        <div className="grid sm:grid-cols-2 xl:grid-cols-3 gap-3">
          {visible.map((g) => (
            <ProviderLimitCard
              key={g.provider}
              provider={g.provider}
              name={g.name}
              plan={g.plan}
              quotas={g.quotas}
              message={g.message}
              error={g.error}
              onRefresh={async () => {
                // Single endpoint serves every provider, so a per-card refresh
                // refetches the shared list. The card's own spinner is local;
                // other cards stay rendered with their last value.
                await qc.refetchQueries({ queryKey: ["quotas"], exact: true });
              }}
            />
          ))}
        </div>
      )}
    </div>
  );
}

function SummaryStat({
  label,
  value,
  tone,
  dot,
}: {
  label: string;
  value: number;
  tone: "muted" | "success" | "warning" | "danger";
  dot?: boolean;
}) {
  const toneClass = {
    muted: "text-text-muted",
    success: "text-success",
    warning: "text-warning",
    danger: "text-destructive",
  }[tone];
  const dotClass = {
    muted: "bg-text-muted",
    success: "bg-success",
    warning: "bg-warning",
    danger: "bg-destructive",
  }[tone];
  return (
    <div className="card-elev border border-border p-3 flex items-center gap-3">
      {dot && <span className={`w-2 h-2 rounded-full ${dotClass}`} />}
      <div>
        <div className="text-[10px] uppercase tracking-wide text-text-muted">
          {label}
        </div>
        <div className={`text-xl font-semibold tabular-nums ${toneClass}`}>
          {value}
        </div>
      </div>
    </div>
  );
}
