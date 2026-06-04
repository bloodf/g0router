import { useEffect, useState } from "react";
import {
  getSettings,
  listConnections,
  listMCPInstances,
  listProviders,
  apiFetch,
  isAuthExpiredError,
  type ConnectionResponse,
  type MCPInstanceResponse,
  type ProviderMatrixEntry,
  type SettingsResponse,
  type UsageListResponse
} from "../api";
import { EmptyState, ErrorState, LoadingState, MetricCard, Panel, StatusPill } from "../components/Primitives";

type DiagnosticsData = {
  connections: ConnectionResponse[];
  logs: UsageListResponse;
  mcpInstances: MCPInstanceResponse[];
  providers: ProviderMatrixEntry[];
  settings: SettingsResponse;
};

type DiagnosticsState =
  | { status: "loading" }
  | { status: "success"; data: DiagnosticsData }
  | { status: "error"; message: string }
  | { status: "auth-expired"; message: string };

export function DiagnosticsPage() {
  const [state, setState] = useState<DiagnosticsState>({ status: "loading" });

  useEffect(() => {
    let cancelled = false;

    async function loadDiagnostics() {
      try {
        const [providers, settings, connections, mcpInstances, logs] = await Promise.all([
          listProviders(),
          getSettings(),
          listConnections(),
          listMCPInstances(),
          apiFetch<UsageListResponse>("/api/logs?limit=1&offset=0")
        ]);
        if (!cancelled) {
          setState({ status: "success", data: { connections, logs, mcpInstances, providers, settings } });
        }
      } catch (error) {
        if (!cancelled) {
          setState({
            status: isAuthExpiredError(error) ? "auth-expired" : "error",
            message: error instanceof Error ? error.message : "diagnostics request failed"
          });
        }
      }
    }

    void loadDiagnostics();
    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <Panel title="Diagnostics" description="Control-plane contract health and release-readiness signals.">
      {renderDiagnostics(state)}
    </Panel>
  );
}

function renderDiagnostics(state: DiagnosticsState) {
  switch (state.status) {
    case "loading":
      return <LoadingState label="Loading diagnostics" />;
    case "error":
      return <ErrorState title="Diagnostics unavailable" message={state.message} />;
    case "auth-expired":
      return <ErrorState title="Session expired" message={state.message} />;
    case "success":
      return <DiagnosticsSummary data={state.data} />;
  }
}

function DiagnosticsSummary({ data }: { data: DiagnosticsData }) {
  const activeConnections = data.connections.filter((connection) => connection.IsActive);
  const healthyInstances = data.mcpInstances.filter((instance) => instance.HealthStatus === "healthy");

  if (data.providers.length === 0 && data.connections.length === 0 && data.mcpInstances.length === 0) {
    return <EmptyState title="No diagnostics data" description="The control plane returned no providers, connections, or MCP instances." />;
  }

  return (
    <div className="space-y-5">
      <div className="grid gap-4 md:grid-cols-3">
        <MetricCard label="Providers" value={`${data.providers.length}`} detail={`${data.providers.length} providers`} tone="sky" />
        <MetricCard label="Connections" value={`${activeConnections.length}`} detail={`${activeConnections.length} active connections`} tone="emerald" />
        <MetricCard label="MCP" value={`${data.mcpInstances.length}`} detail={`${data.mcpInstances.length} MCP instances`} tone={healthyInstances.length === data.mcpInstances.length ? "emerald" : "amber"} />
      </div>

      <div className="grid gap-3 md:grid-cols-2">
        <DiagnosticRow label="Control plane protected" ok={data.settings.RequireAPIKey} />
        <DiagnosticRow label="Request logs endpoint" ok={Array.isArray(data.logs.data)} />
        <DiagnosticRow label="Request logs enabled" ok={data.settings.EnableRequestLogs} />
        <DiagnosticRow label="MCP health known" ok={data.mcpInstances.every((instance) => instance.HealthStatus !== "")} />
      </div>
    </div>
  );
}

function DiagnosticRow({ label, ok }: { label: string; ok: boolean }) {
  return (
    <div className="flex items-center justify-between gap-3 rounded-md border border-zinc-200 px-4 py-3">
      <span className="text-sm font-medium text-zinc-700">{label}</span>
      <StatusPill tone={ok ? "good" : "warn"}>{ok ? "ok" : "check"}</StatusPill>
    </div>
  );
}
