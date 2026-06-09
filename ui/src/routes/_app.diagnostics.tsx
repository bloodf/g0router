import { createFileRoute } from "@tanstack/react-router";
import { useState, useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { apiFetch, ApiError } from "@/lib/api/client";
import { Card, CardHeader, CardTitle, CardContent, CardFooter } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Icon } from "@/components/common/Icon";
import { PageHeader } from "@/components/common/PageHeader";
import { StatusBadge } from "@/components/common/StatusBadge";
import { Skeleton } from "@/components/ui/skeleton";
import type { Provider, Connection } from "@/lib/types";

export const Route = createFileRoute("/_app/diagnostics")({
  component: DiagnosticsPage,
});

interface VersionInfo {
  version: string;
  go_version: string;
  build_date: string;
}

interface HealthInfo {
  status: string;
  version: string;
}

interface AuthDiagnostic {
  authenticated: boolean;
  username?: string;
}

function DiagnosticsPage() {
  return (
    <div>
      <PageHeader title="Diagnostics" icon="monitor_heart" />

      <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
        <VersionDiagnostic />
        <HealthDiagnostic />
        <AuthDiagnosticCard />
        <BrowserDiagnostic />
        <ProviderCountDiagnostic />
        <ConnectionCountDiagnostic />
      </div>
    </div>
  );
}

function VersionDiagnostic() {
  const { data, isLoading, isError, error, refetch } = useQuery<VersionInfo>({
    queryKey: ["diagnostics", "version"],
    queryFn: () => apiFetch<VersionInfo>("/api/version"),
  });

  return (
    <DiagnosticCard
      title="Application version"
      icon="deployed_code"
      onRefresh={() => refetch()}
      isLoading={isLoading}
      isError={isError}
      error={error}
    >
      {data && (
        <div className="space-y-1 text-sm">
          <DiagnosticRow label="Version" value={data.version} />
          <DiagnosticRow label="Build date" value={data.build_date} />
          <DiagnosticRow label="Go version" value={data.go_version} />
        </div>
      )}
    </DiagnosticCard>
  );
}

function HealthDiagnostic() {
  const { data, isLoading, isError, error, refetch } = useQuery<HealthInfo & { latency_ms: number }>({
    queryKey: ["diagnostics", "healthz"],
    queryFn: async () => {
      const start = performance.now();
      const response = await fetch("/healthz", { method: "GET", credentials: "same-origin" });
      const latencyMs = Math.round(performance.now() - start);
      if (!response.ok) {
        throw new ApiError(response.status, `Health check failed: ${response.statusText}`);
      }
      const json = await response.json();
      return { ...json, latency_ms: latencyMs };
    },
  });

  return (
    <DiagnosticCard
      title="API connectivity"
      icon="network_check"
      onRefresh={() => refetch()}
      isLoading={isLoading}
      isError={isError}
      error={error}
    >
      {data && (
        <div className="space-y-2">
          <div className="flex items-center gap-2">
            <StatusBadge variant={data.status === "ok" ? "success" : "danger"} dot>
              {data.status === "ok" ? "Online" : "Offline"}
            </StatusBadge>
            <span className="text-xs text-text-muted">{data.latency_ms} ms</span>
          </div>
          <DiagnosticRow label="Version" value={data.version} />
        </div>
      )}
    </DiagnosticCard>
  );
}

function AuthDiagnosticCard() {
  const { data, isLoading, isError, error, refetch } = useQuery<AuthDiagnostic>({
    queryKey: ["diagnostics", "auth"],
    queryFn: async () => {
      const response = await fetch("/api/settings", { method: "GET", credentials: "same-origin" });
      if (response.status === 401) {
        return { authenticated: false };
      }
      if (!response.ok) {
        throw new ApiError(response.status, `Auth check failed: ${response.statusText}`);
      }
      return { authenticated: true };
    },
  });

  return (
    <DiagnosticCard
      title="Authentication status"
      icon="lock_person"
      onRefresh={() => refetch()}
      isLoading={isLoading}
      isError={isError}
      error={error}
    >
      {data && (
        <div className="flex items-center gap-2">
          <StatusBadge variant={data.authenticated ? "success" : "warning"} dot>
            {data.authenticated ? "Logged in" : "Logged out"}
          </StatusBadge>
        </div>
      )}
    </DiagnosticCard>
  );
}

function BrowserDiagnostic() {
  const [screenSize, setScreenSize] = useState("");

  useEffect(() => {
    const update = () => {
      setScreenSize(`${window.innerWidth} × ${window.innerHeight}`);
    };
    update();
    window.addEventListener("resize", update);
    return () => window.removeEventListener("resize", update);
  }, []);

  const refresh = () => setScreenSize(`${window.innerWidth} × ${window.innerHeight}`);

  return (
    <DiagnosticCard
      title="Browser info"
      icon="web"
      onRefresh={refresh}
      isLoading={false}
      isError={false}
    >
      <div className="space-y-1 text-sm">
        <DiagnosticRow label="User agent" value={navigator.userAgent} />
        <DiagnosticRow label="Screen size" value={screenSize} />
        <DiagnosticRow label="Language" value={navigator.language} />
      </div>
    </DiagnosticCard>
  );
}

function ProviderCountDiagnostic() {
  const { data, isLoading, isError, error, refetch } = useQuery<Provider[]>({
    queryKey: ["diagnostics", "providers"],
    queryFn: () => apiFetch<Provider[]>("/api/providers"),
  });

  return (
    <DiagnosticCard
      title="Provider count"
      icon="dns"
      onRefresh={() => refetch()}
      isLoading={isLoading}
      isError={isError}
      error={error}
    >
      {data !== undefined && (
        <div className="flex items-baseline gap-2">
          <span className="text-2xl font-semibold tabular-nums">{data.length}</span>
          <span className="text-sm text-text-muted">configured providers</span>
        </div>
      )}
    </DiagnosticCard>
  );
}

function ConnectionCountDiagnostic() {
  const { data, isLoading, isError, error, refetch } = useQuery<Connection[]>({
    queryKey: ["diagnostics", "connections"],
    queryFn: () => apiFetch<Connection[]>("/api/connections"),
  });

  return (
    <DiagnosticCard
      title="Connection count"
      icon="link"
      onRefresh={() => refetch()}
      isLoading={isLoading}
      isError={isError}
      error={error}
    >
      {data !== undefined && (
        <div className="flex items-baseline gap-2">
          <span className="text-2xl font-semibold tabular-nums">{data.length}</span>
          <span className="text-sm text-text-muted">connections</span>
        </div>
      )}
    </DiagnosticCard>
  );
}

function DiagnosticCard({
  title,
  icon,
  children,
  onRefresh,
  isLoading,
  isError,
  error,
}: {
  title: string;
  icon: string;
  children: React.ReactNode;
  onRefresh: () => void;
  isLoading: boolean;
  isError: boolean;
  error?: unknown;
}) {
  return (
    <Card className="card-elev border-border flex flex-col">
      <CardHeader className="pb-3">
        <CardTitle className="text-sm font-medium flex items-center gap-2">
          <Icon name={icon} size={18} className="text-brand-500" />
          {title}
        </CardTitle>
      </CardHeader>
      <CardContent className="flex-1 min-h-[80px]">
        {isLoading ? (
          <div className="space-y-2">
            <Skeleton className="h-4 w-2/3" />
            <Skeleton className="h-4 w-1/2" />
          </div>
        ) : isError ? (
          <div className="text-sm text-destructive">
            {error instanceof Error ? error.message : "Check failed"}
          </div>
        ) : (
          children
        )}
      </CardContent>
      <CardFooter className="justify-end pt-0">
        <Button variant="outline" size="sm" onClick={onRefresh} disabled={isLoading}>
          <Icon name="refresh" size={14} className="mr-1.5" />
          Refresh
        </Button>
      </CardFooter>
    </Card>
  );
}

function DiagnosticRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex flex-col">
      <span className="text-[10px] uppercase tracking-wider text-text-muted">{label}</span>
      <span className="font-medium truncate" title={value}>
        {value || "—"}
      </span>
    </div>
  );
}
