import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api/client";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Icon } from "@/components/common/Icon";
import { PageHeader } from "@/components/common/PageHeader";
import { ProviderIcon } from "@/components/common/ProviderIcon";
import { TrafficSummary } from "@/components/topology/TrafficSummary";
import { ProviderTopology } from "@/components/topology/ProviderTopology";
import { useTrafficStream } from "@/lib/hooks/useTrafficStream";
import { MetricsGridSkeleton, ErrorState } from "@/components/common/Skeletons";
import type { Provider } from "@/lib/types";

export const Route = createFileRoute("/_app/dashboard")({
  component: DashboardPage,
});

function DashboardPage() {
  const [paused, setPaused] = useState(false);
  const {
    data: providers = [],
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery<Provider[]>({
    queryKey: ["providers"],
    queryFn: () => apiFetch<Provider[]>("/api/providers"),
  });

  const { events } = useTrafficStream({ enabled: !paused });

  const activeProviders = providers.filter((p) => p.status === "active").length;
  const errorProviders = providers.filter((p) => p.status === "error").length;

  return (
    <div>
      <PageHeader
        title="Dashboard"
        description="Real-time overview of your gateway."
        icon="dashboard"
        actions={
          <Button variant="outline" onClick={() => setPaused(!paused)}>
            <Icon name={paused ? "play_arrow" : "pause"} size={16} className="mr-1" />
            {paused ? "Resume" : "Pause"}
          </Button>
        }
      />

      {isLoading ? (
        <MetricsGridSkeleton />
      ) : isError ? (
        <ErrorState
          title="Couldn\u2019t load providers"
          error={error}
          onRetry={() => refetch()}
          className="mb-4"
        />
      ) : (
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3 mb-4">
          <StatCard label="Providers" value={providers.length} icon="dns" />
          <StatCard
            label="Active"
            value={activeProviders}
            icon="check_circle"
            tone="success"
          />
          <StatCard
            label="Errors"
            value={errorProviders}
            icon="error"
            tone={errorProviders > 0 ? "danger" : "success"}
          />
          <StatCard label="Events" value={events.length} icon="bolt" tone="info" />
        </div>
      )}

      <TrafficSummary paused={paused} onPausedChange={setPaused} />
      <ProviderTopology paused={paused} onPausedChange={setPaused} />
    </div>
  );
}

function StatCard({
  label,
  value,
  icon,
  tone,
}: {
  label: string;
  value: number;
  icon: string;
  tone?: "info" | "success" | "warning" | "danger";
}) {
  const toneClass: Record<string, string> = {
    info: "text-info bg-info/10",
    success: "text-success bg-success/10",
    warning: "text-warning bg-warning/10",
    danger: "text-destructive bg-destructive/10",
  };
  return (
    <Card className="card-elev border-border p-3">
      <div className="flex items-center gap-2">
        <div
          className={
            "w-8 h-8 rounded-md flex items-center justify-center " +
            (tone ? toneClass[tone] : "bg-surface text-text-muted")
          }
        >
          <Icon name={icon} size={16} />
        </div>
        <div>
          <div className="text-[10px] uppercase tracking-wider text-text-muted">{label}</div>
          <div className="text-lg font-semibold tabular-nums">{value}</div>
        </div>
      </div>
    </Card>
  );
}
