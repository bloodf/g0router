import * as React from "react";
import { apiFetch } from "@/lib/api";
import { Card } from "@/components/ui/card";
import { SegmentedControl } from "@/components/ui/segmented-control";
import { CardSkeleton } from "@/components/ui/skeleton";
import { ProviderTopology } from "./provider-topology";

// UsageStatsData mirrors the real Go usage.Stats payload
// (internal/usage/stats.go:26-41), snake_case, served by
// GET /api/usage/stats?period= (internal/admin/usage.go:101).
export interface ProviderStat {
  requests: number;
  prompt_tokens: number;
  completion_tokens: number;
  cost: number;
}
export interface ModelStat extends ProviderStat {
  raw_model: string;
  provider: string;
  last_used: string;
}
export interface ActiveRequest {
  model: string;
  provider: string;
  account: string;
  count: number;
}
export interface RecentRequest {
  timestamp: string;
  model: string;
  provider: string;
  prompt_tokens: number;
  completion_tokens: number;
  status: string;
}
export interface UsageStatsData {
  total_requests: number;
  total_prompt_tokens: number;
  total_completion_tokens: number;
  total_cost: number;
  by_provider: Record<string, ProviderStat>;
  by_model: Record<string, ModelStat>;
  active_requests: ActiveRequest[];
  recent_requests: RecentRequest[];
  pending: Record<string, number>;
  error_provider: string;
}

// Live fields delivered over the SSE stream (a subset of UsageStatsData).
export interface UsageStreamFrame {
  active_requests?: ActiveRequest[];
  recent_requests?: RecentRequest[];
  pending?: Record<string, number>;
  error_provider?: string;
  // The stream may also carry the full snapshot; any aggregate fields override.
  total_requests?: number;
  total_prompt_tokens?: number;
  total_completion_tokens?: number;
  total_cost?: number;
  by_provider?: Record<string, ProviderStat>;
  by_model?: Record<string, ModelStat>;
}

const EMPTY_STATS: UsageStatsData = {
  total_requests: 0,
  total_prompt_tokens: 0,
  total_completion_tokens: 0,
  total_cost: 0,
  by_provider: {},
  by_model: {},
  active_requests: [],
  recent_requests: [],
  pending: {},
  error_provider: "",
};

// mergeUsageStats overlays the live SSE frame onto the REST-fetched base stats.
// Only the fields present on the frame are overlaid; absent fields keep the base
// value. This matches the 9router additive-SSE merge (plan §1.3 / ref
// UsageStats.js:256-278): the page is fully functional from REST alone.
export function mergeUsageStats(base: UsageStatsData, frame: UsageStreamFrame): UsageStatsData {
  return {
    ...base,
    ...(frame.total_requests !== undefined ? { total_requests: frame.total_requests } : {}),
    ...(frame.total_prompt_tokens !== undefined ? { total_prompt_tokens: frame.total_prompt_tokens } : {}),
    ...(frame.total_completion_tokens !== undefined ? { total_completion_tokens: frame.total_completion_tokens } : {}),
    ...(frame.total_cost !== undefined ? { total_cost: frame.total_cost } : {}),
    ...(frame.by_provider ? { by_provider: frame.by_provider } : {}),
    ...(frame.by_model ? { by_model: frame.by_model } : {}),
    active_requests: frame.active_requests ?? base.active_requests,
    recent_requests: frame.recent_requests ?? base.recent_requests,
    pending: frame.pending ?? base.pending,
    error_provider: frame.error_provider ?? base.error_provider,
  };
}

export interface UsageStreamHandlers {
  onData: (frame: UsageStreamFrame) => void;
  onError: (ev: unknown) => void;
}

// subscribeUsageStream opens the additive SSE overlay (plan §1.3 / PAR-UI-082).
// It is a pure function (no React) so the SSE contract is unit-provable with a
// stubbed EventSource. Under the e2e harness the MockEventSource fires `open`
// then idles for /api/usage/stream (fixture.ts) — onData is simply never called
// and the page renders from REST. Returns a cleanup that closes the stream.
export function subscribeUsageStream(handlers: UsageStreamHandlers): () => void {
  if (typeof EventSource === "undefined") {
    return () => {};
  }
  const es = new EventSource("/api/usage/stream");
  // Use addEventListener (not .onmessage=) so synthetic dispatchEvent frames
  // from the e2e MockEventSource (fixture.ts) and real EventSource both reach us.
  const onMessage = (ev: MessageEvent) => {
    try {
      const frame = JSON.parse(ev.data) as UsageStreamFrame;
      handlers.onData(frame);
    } catch {
      // Ignore malformed frames; the REST view remains authoritative.
    }
  };
  const onError = (ev: Event) => {
    // No-op / clear-loading — the stream is purely additive (plan §1.3).
    handlers.onError(ev);
  };
  es.addEventListener("message", onMessage as EventListener);
  es.addEventListener("error", onError);
  return () => {
    es.removeEventListener("message", onMessage as EventListener);
    es.removeEventListener("error", onError);
    es.close();
  };
}

export type UsagePeriod = "today" | "24h" | "7d" | "30d" | "60d" | "all";

const PERIOD_OPTIONS: { value: UsagePeriod; label: string }[] = [
  { value: "today", label: "Today" },
  { value: "24h", label: "24h" },
  { value: "7d", label: "7d" },
  { value: "30d", label: "30d" },
  { value: "60d", label: "60d" },
];

function formatNumber(n: number): string {
  return n.toLocaleString("en-US");
}
function formatCost(n: number): string {
  return `$${n.toFixed(2)}`;
}

interface MetricCardProps {
  label: string;
  value: string;
}
function MetricCard({ label, value }: MetricCardProps) {
  return (
    <Card padding="md" className="flex flex-col gap-1">
      <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">{label}</span>
      <span data-testid="usage-metric" className="text-2xl font-semibold text-foreground">
        {value}
      </span>
    </Card>
  );
}

export interface UsageStatsProps {
  period?: UsagePeriod;
  setPeriod?: (p: UsagePeriod) => void;
  hidePeriodSelector?: boolean;
  // initialStats lets SSR / unit tests render synchronously without a fetch.
  initialStats?: UsageStatsData;
}

export function UsageStats({
  period = "all",
  setPeriod,
  hidePeriodSelector,
  initialStats,
}: UsageStatsProps) {
  const [internalPeriod, setInternalPeriod] = React.useState<UsagePeriod>(period);
  const activePeriod = setPeriod ? period : internalPeriod;
  const [stats, setStats] = React.useState<UsageStatsData | null>(initialStats ?? null);
  const [loading, setLoading] = React.useState(!initialStats);

  // REST fetch on mount + whenever the period changes (the authoritative data).
  React.useEffect(() => {
    let cancelled = false;
    setLoading(true);
    apiFetch<UsageStatsData>(`/api/usage/stats?period=${activePeriod}`)
      .then((data) => {
        if (cancelled) return;
        setStats({ ...EMPTY_STATS, ...data });
        setLoading(false);
      })
      .catch(() => {
        if (cancelled) return;
        setStats((prev) => prev ?? EMPTY_STATS);
        setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [activePeriod]);

  // Additive SSE overlay — opens once; merges live fields; never blocks render.
  React.useEffect(() => {
    const cleanup = subscribeUsageStream({
      onData: (frame) => setStats((prev) => mergeUsageStats(prev ?? EMPTY_STATS, frame)),
      onError: () => setLoading(false),
    });
    return cleanup;
  }, []);

  const handlePeriod = (value: string) => {
    const p = value as UsagePeriod;
    if (setPeriod) setPeriod(p);
    else setInternalPeriod(p);
  };

  const view = stats ?? initialStats ?? EMPTY_STATS;

  if (loading && !stats && !initialStats) {
    return (
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <CardSkeleton />
        <CardSkeleton />
        <CardSkeleton />
        <CardSkeleton />
      </div>
    );
  }

  const totalTokens = view.total_prompt_tokens + view.total_completion_tokens;

  return (
    <div className="flex flex-col gap-6">
      {!hidePeriodSelector ? (
        <div data-testid="usage-period">
          <SegmentedControl options={PERIOD_OPTIONS} value={activePeriod} onChange={handlePeriod} />
        </div>
      ) : null}

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <MetricCard label="Requests" value={formatNumber(view.total_requests)} />
        <MetricCard label="Tokens" value={formatNumber(totalTokens)} />
        <MetricCard label="Cost" value={formatCost(view.total_cost)} />
        <MetricCard label="Active" value={formatNumber(view.active_requests.length)} />
      </div>

      <ProviderTopology byProvider={view.by_provider} byModel={view.by_model} />

      <Card padding="none">
        <table className="w-full text-sm" data-testid="usage-provider-table">
          <thead>
            <tr className="border-b border-border text-left text-muted-foreground">
              <th className="px-4 py-2 font-medium">Provider</th>
              <th className="px-4 py-2 font-medium">Requests</th>
              <th className="px-4 py-2 font-medium">Tokens</th>
              <th className="px-4 py-2 font-medium">Cost</th>
            </tr>
          </thead>
          <tbody>
            {Object.entries(view.by_provider).map(([provider, stat]) => (
              <tr key={provider} className="border-b border-border/50">
                <td className="px-4 py-2 font-medium text-foreground">{provider}</td>
                <td className="px-4 py-2">{formatNumber(stat.requests)}</td>
                <td className="px-4 py-2">{formatNumber(stat.prompt_tokens + stat.completion_tokens)}</td>
                <td className="px-4 py-2">{formatCost(stat.cost)}</td>
              </tr>
            ))}
            {Object.keys(view.by_provider).length === 0 ? (
              <tr>
                <td colSpan={4} className="px-4 py-6 text-center text-muted-foreground">
                  No usage in this period.
                </td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </Card>
    </div>
  );
}
