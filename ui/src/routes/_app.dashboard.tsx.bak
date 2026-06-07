import { createFileRoute } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { Link } from "@tanstack/react-router";
import { apiFetch } from "@/lib/api/client";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { MetricCard } from "@/components/common/MetricCard";
import { PageHeader } from "@/components/common/PageHeader";
import { ProviderTopology } from "@/components/topology/ProviderTopology";
import { StatusBadge } from "@/components/common/StatusBadge";
import { ProviderIcon } from "@/components/common/ProviderIcon";
import { Icon } from "@/components/common/Icon";
import { MetricsGridSkeleton, ListRowsSkeleton, ErrorState } from "@/components/common/Skeletons";
import { useTrafficStream } from "@/lib/mocks/streams";
import type { Provider } from "@/lib/mocks/types";
import { useTranslation } from "react-i18next";
import { formatDistanceToNow } from "date-fns";

export const Route = createFileRoute("/_app/dashboard")({
  component: DashboardPage,
});

function DashboardPage() {
  const { t } = useTranslation();
  const providersQuery = useQuery<Provider[]>({
    queryKey: ["providers"],
    queryFn: () => apiFetch("/api/providers"),
  });
  const summaryQuery = useQuery({
    queryKey: ["usage", "summary", "today"],
    queryFn: () => apiFetch("/api/usage/summary?period=today"),
  });
  const settingsQuery = useQuery({
    queryKey: ["settings"],
    queryFn: () => apiFetch("/api/settings"),
  });
  const providers = providersQuery.data ?? [];
  const summary = summaryQuery.data;
  const settings = settingsQuery.data;
  const providersLoading = providersQuery.isLoading;
  const summaryLoading = summaryQuery.isLoading;
  const settingsLoading = settingsQuery.isLoading;
  const anyError = providersQuery.isError || summaryQuery.isError || settingsQuery.isError;
  const firstError = providersQuery.error ?? summaryQuery.error ?? settingsQuery.error;
  const retryAll = () => {
    providersQuery.refetch();
    summaryQuery.refetch();
    settingsQuery.refetch();
  };

  const { events } = useTrafficStream({});
  const activeProviders = providers.filter((p) => p.connection_count > 0);

  return (
    <div>
      <PageHeader
        title={t("dashboard.title")}
        description="Real-time view of your LLM gateway."
        icon="dashboard"
      />

      {anyError && (
        <ErrorState
          title="Some dashboard data failed to load"
          error={firstError}
          onRetry={retryAll}
          compact
          className="mb-4"
        />
      )}

      {summaryLoading || providersLoading ? (
        <MetricsGridSkeleton className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-6" />
      ) : (
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
          <MetricCard
            label={t("dashboard.active_connections")}
            value={activeProviders.length}
            icon="link"
            accent="success"
            pulse={activeProviders.length > 0}
          />
          <MetricCard
            label={t("dashboard.requests_today")}
            value={summary?.total_requests ?? "—"}
            icon="bar_chart"
            accent="brand"
            delta={{ value: "+12%", direction: "up" }}
          />
          <MetricCard
            label={t("dashboard.tokens_today")}
            value={
              summary
                ? Intl.NumberFormat("en", { notation: "compact" }).format(
                    summary.total_tokens,
                  )
                : "—"
            }
            icon="memory"
            accent="info"
          />
          <MetricCard
            label={t("dashboard.cost_today")}
            value={summary ? `$${summary.total_cost.toFixed(2)}` : "—"}
            icon="payments"
            accent="warning"
          />
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4 mb-6">
        <Card className="lg:col-span-2 p-4 card-elev border-border">
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-sm font-semibold flex items-center gap-2">
              <Icon name="graph_3" size={18} className="text-brand-500" />
              {t("dashboard.live_topology")}
            </h2>
            <Link to="/traffic">
              <Button variant="ghost" size="sm" className="text-xs">
                Open full view
                <Icon name="arrow_forward" size={14} className="ml-1" />
              </Button>
            </Link>
          </div>
          <ProviderTopology variant="compact" />
        </Card>

        <Card className="p-4 card-elev border-border">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-3">
            <Icon name="monitor_heart" size={18} className="text-success" />
            {t("dashboard.system_status")}
          </h2>
          {settingsLoading ? (
            <ListRowsSkeleton rows={5} />
          ) : (
            <div className="space-y-2.5 text-sm">
              <StatusRow
                label="Cache"
                status={settings?.cache_enabled ? "enabled" : "disabled"}
              />
              <StatusRow
                label="RTK"
                status={settings?.rtk_enabled ? "enabled" : "disabled"}
              />
              <StatusRow
                label="Caveman mode"
                status={settings?.caveman_enabled ? settings.caveman_level : "off"}
              />
              <StatusRow
                label="Request logs"
                status={settings?.enable_request_logs ? "enabled" : "disabled"}
              />
              <StatusRow
                label="Login required"
                status={settings?.require_login ? "enabled" : "disabled"}
              />
            </div>
          )}
          <div className="border-t border-border mt-4 pt-3">
            <h3 className="text-xs font-semibold uppercase text-text-muted mb-2">
              Provider health
            </h3>
            <div className="space-y-1.5">
              {activeProviders.slice(0, 4).map((p) => (
                <div
                  key={p.id}
                  className="flex items-center justify-between text-xs"
                >
                  <div className="flex items-center gap-1.5 truncate">
                    <ProviderIcon provider={p.id} size={18} />
                    <span className="truncate">{p.display_name}</span>
                  </div>
                  <StatusBadge
                    variant={
                      p.status === "active"
                        ? "success"
                        : p.status === "needs_reauth"
                          ? "warning"
                          : p.status === "error"
                            ? "danger"
                            : "muted"
                    }
                    dot
                  >
                    {p.status}
                  </StatusBadge>
                </div>
              ))}
            </div>
          </div>
        </Card>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4 mb-6">
        <Card className="lg:col-span-2 p-4 card-elev border-border overflow-hidden">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-3">
            <Icon name="bolt" size={18} className="text-warning" />
            {t("dashboard.recent_traffic")}
          </h2>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="text-[11px] uppercase tracking-wider text-text-muted text-left">
                <tr className="border-b border-border">
                  <th className="py-2 font-medium">Time</th>
                  <th className="py-2 font-medium">Key</th>
                  <th className="py-2 font-medium">Provider</th>
                  <th className="py-2 font-medium">Model</th>
                  <th className="py-2 font-medium text-right">Tokens</th>
                  <th className="py-2 font-medium text-right">Latency</th>
                  <th className="py-2 font-medium">Status</th>
                </tr>
              </thead>
              <tbody>
                {events.slice(0, 10).map((e) => (
                  <tr key={e.id} className="border-b border-border-subtle">
                    <td className="py-2 text-xs text-text-muted">
                      {formatDistanceToNow(new Date(e.timestamp), { addSuffix: true })}
                    </td>
                    <td className="py-2 truncate max-w-[100px]">{e.api_key_name}</td>
                    <td className="py-2">
                      <div className="flex items-center gap-1.5">
                        <ProviderIcon provider={e.provider} size={18} />
                        <span className="text-xs">{e.provider}</span>
                      </div>
                    </td>
                    <td className="py-2 font-mono text-xs">{e.model}</td>
                    <td className="py-2 text-right tabular-nums text-xs">{e.tokens}</td>
                    <td className="py-2 text-right tabular-nums text-xs">{e.latency_ms}ms</td>
                    <td className="py-2">
                      <StatusBadge
                        variant={e.status === "success" ? "success" : "danger"}
                        dot
                      >
                        {e.status}
                      </StatusBadge>
                    </td>
                  </tr>
                ))}
                {!events.length && (
                  <tr>
                    <td colSpan={7} className="py-8 text-center text-text-muted text-sm">
                      Waiting for traffic…
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </Card>

        <Card className="p-4 card-elev border-border">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-3">
            <Icon name="flash_on" size={18} className="text-info" />
            {t("dashboard.quick_actions")}
          </h2>
          <div className="space-y-2">
            <QuickAction to="/providers" icon="dns" label={t("dashboard.add_provider")} />
            <QuickAction to="/keys" icon="key" label={t("dashboard.create_key")} />
            <QuickAction to="/chat" icon="chat" label={t("dashboard.test_chat")} />
            <QuickAction to="/logs" icon="description" label={t("dashboard.view_logs")} />
          </div>
        </Card>
      </div>
    </div>
  );
}

function StatusRow({ label, status }: { label: string; status: string }) {
  const variant: "success" | "muted" | "info" =
    status === "enabled" || status === "lite" || status === "full" || status === "ultra"
      ? "success"
      : status === "disabled" || status === "off"
        ? "muted"
        : "info";
  return (
    <div className="flex items-center justify-between">
      <span className="text-text-muted">{label}</span>
      <StatusBadge variant={variant} dot>
        {status}
      </StatusBadge>
    </div>
  );
}

function QuickAction({
  to,
  icon,
  label,
}: {
  to: string;
  icon: string;
  label: string;
}) {
  return (
    <Link
      to={to}
      className="flex items-center gap-2.5 p-2.5 rounded-lg border border-border bg-surface hover:border-brand-500 hover:bg-brand-500/5 transition-colors"
    >
      <div className="w-8 h-8 rounded-lg bg-brand-500/10 text-brand-600 flex items-center justify-center">
        <Icon name={icon} size={16} />
      </div>
      <span className="text-sm font-medium">{label}</span>
      <Icon name="chevron_right" size={16} className="ml-auto text-text-muted" />
    </Link>
  );
}
