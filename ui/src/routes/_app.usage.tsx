import { createFileRoute } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import {
  AreaChart,
  Area,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { apiFetch } from "@/lib/api/client";
import { Card } from "@/components/ui/card";
import { PageHeader } from "@/components/common/PageHeader";
import { MetricCard } from "@/components/common/MetricCard";
import { MetricsGridSkeleton, ChartSkeleton, ErrorState } from "@/components/common/Skeletons";
import { useState } from "react";
import { format } from "date-fns";

export const Route = createFileRoute("/_app/usage")({ component: UsagePage });

function UsagePage() {
  const [period, setPeriod] = useState("7d");
  const {
    data: summary,
    isLoading: summaryLoading,
    isError: summaryError,
    error: summaryErr,
    refetch: refetchSummary,
  } = useQuery({
    queryKey: ["usage", "summary", period],
    queryFn: () => apiFetch(`/api/usage/summary?period=${period}`),
  });
  const {
    data: chart,
    isLoading: chartLoading,
    isError: chartError,
    error: chartErr,
    refetch: refetchChart,
  } = useQuery({
    queryKey: ["usage", "chart", period],
    queryFn: () => apiFetch(`/api/usage/chart?period=${period}`),
  });

  const data = (chart?.buckets ?? []).map((b: string, i: number) => ({
    label: format(new Date(b), period === "today" || period === "24h" ? "HH:mm" : "MMM d"),
    tokens: (chart?.tokens_input?.[i] ?? 0) + (chart?.tokens_output?.[i] ?? 0),
    cost: chart?.costs?.[i] ?? 0,
    requests: chart?.requests?.[i] ?? 0,
  }));

  return (
    <div>
      <PageHeader
        title="Usage"
        description="Aggregated usage across all keys, providers and models."
        icon="bar_chart"
      />

      <div className="flex items-center gap-1 bg-surface-2 rounded-lg p-1 mb-4 w-fit">
        {["today", "24h", "7d", "30d", "60d"].map((p) => (
          <button
            key={p}
            type="button"
            onClick={() => setPeriod(p)}
            className={
              "px-3 py-1.5 text-xs rounded-md uppercase transition-colors " +
              (period === p
                ? "bg-surface text-foreground shadow-soft font-medium"
                : "text-text-muted hover:text-foreground")
            }
          >
            {p}
          </button>
        ))}
      </div>

      {summaryLoading ? (
        <MetricsGridSkeleton />
      ) : summaryError ? (
        <ErrorState
          title="Couldn’t load usage summary"
          error={summaryErr}
          onRetry={() => refetchSummary()}
          compact
          className="mb-4"
        />
      ) : (
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-4">
          <MetricCard label="Requests" value={summary?.total_requests ?? "—"} icon="bar_chart" />
          <MetricCard
            label="Tokens"
            value={
              summary
                ? Intl.NumberFormat("en", { notation: "compact" }).format(summary.total_tokens)
                : "—"
            }
            icon="memory"
            accent="info"
          />
          <MetricCard
            label="Cost"
            value={summary ? `$${summary.total_cost.toFixed(2)}` : "—"}
            icon="payments"
            accent="warning"
          />
          <MetricCard
            label="Avg latency"
            value={summary ? `${summary.avg_latency_ms}ms` : "—"}
            icon="speed"
            accent="success"
          />
        </div>
      )}

      <Card className="p-4 card-elev border-border">
        <h3 className="text-sm font-semibold mb-3">Tokens over time</h3>
        {chartLoading ? (
          <ChartSkeleton height={280} />
        ) : chartError ? (
          <ErrorState
            title="Couldn’t load chart"
            error={chartErr}
            onRetry={() => refetchChart()}
            compact
          />
        ) : (
          <div style={{ width: "100%", height: 280 }}>
            <ResponsiveContainer>
              <AreaChart data={data}>
                <defs>
                  <linearGradient id="tg" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="var(--color-brand)" stopOpacity={0.4} />
                    <stop offset="95%" stopColor="var(--color-brand)" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border)" opacity={0.6} />
                <XAxis dataKey="label" tick={{ fontSize: 11, fill: "var(--color-text-muted)" }} stroke="var(--color-border)" />
                <YAxis tick={{ fontSize: 11, fill: "var(--color-text-muted)" }} stroke="var(--color-border)" />
                <Tooltip
                  contentStyle={{
                    background: "var(--color-surface)",
                    border: "1px solid var(--color-border)",
                    borderRadius: 8,
                    fontSize: 12,
                  }}
                />
                <Area
                  type="monotone"
                  dataKey="tokens"
                  stroke="var(--color-brand)"
                  strokeWidth={2}
                  fill="url(#tg)"
                />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        )}
      </Card>
    </div>
  );
}
