import { useEffect, useState } from "react";
import {
  getUsageSummary,
  isAuthExpiredError,
  listCombos,
  listConnections,
  listLogs,
  listMCPInstances
} from "../api";
import type {
  ComboResponse,
  ConnectionResponse,
  MCPInstanceResponse,
  UsageLogRecord,
  UsageSummaryResponse
} from "../api";
import { EmptyState, ErrorState, LoadingState, MetricCard, Panel, StatusPill } from "../components/Primitives";

type DashboardData = {
  combos: ComboResponse[];
  connections: ConnectionResponse[];
  logs: UsageLogRecord[];
  mcpInstances: MCPInstanceResponse[];
  summary: UsageSummaryResponse;
};

type DashboardState =
  | { status: "loading" }
  | { status: "success"; data: DashboardData }
  | { status: "empty"; data: DashboardData }
  | { status: "error"; message: string }
  | { status: "auth-expired"; message: string };

export function DashboardPage() {
  const [state, setState] = useState<DashboardState>({ status: "loading" });

  useEffect(() => {
    let cancelled = false;

    async function loadDashboard() {
      try {
        const [connections, summary, logs, combos, mcpInstances] = await Promise.all([
          listConnections(),
          getUsageSummary(),
          listLogs(),
          listCombos(),
          listMCPInstances()
        ]);
        if (cancelled) {
          return;
        }

        const data = { combos, connections, logs: logs.data, mcpInstances, summary };
        setState(hasDashboardData(data) ? { status: "success", data } : { status: "empty", data });
      } catch (error) {
        if (cancelled) {
          return;
        }
        setState({
          status: isAuthExpiredError(error) ? "auth-expired" : "error",
          message: error instanceof Error ? error.message : "dashboard request failed"
        });
      }
    }

    void loadDashboard();

    return () => {
      cancelled = true;
    };
  }, []);

  if (state.status === "loading") {
    return <LoadingState label="Loading dashboard data" />;
  }

  if (state.status === "auth-expired") {
    return <ErrorState title="Session expired" message={state.message} />;
  }

  if (state.status === "error") {
    return <ErrorState title="Dashboard data unavailable" message={state.message} />;
  }

  if (state.status === "empty") {
    return (
      <Panel title="Gateway status" description="Operational snapshot from management APIs.">
        <EmptyState title="No overview data yet" description="No connections, request logs, combos, or MCP instances were returned." />
      </Panel>
    );
  }

  return <DashboardOverview data={state.data} />;
}

function DashboardOverview({ data }: { data: DashboardData }) {
  const activeConnections = data.connections.filter((connection) => connection.IsActive);
  const activeProviderCount = new Set(activeConnections.map((connection) => connection.Provider)).size;
  const failedLogs = data.logs.filter(isFailedLog);
  const successfulLogs = data.logs.filter(isSuccessfulLog);
  const streamingLogs = data.logs.filter(isStreamingRecord);
  const activeCombos = data.combos.filter((combo) => combo.IsActive);
  const activeMcpInstances = data.mcpInstances.filter((instance) => instance.IsActive);
  const healthyMcpInstances = data.mcpInstances.filter((instance) => instance.HealthStatus === "healthy");
  const totalMcpTools = data.mcpInstances.reduce((count, instance) => count + (instance.ToolManifest?.tools?.length ?? 0), 0);

  return (
    <div className="space-y-6">
      <div className="grid gap-4 md:grid-cols-3">
        <MetricCard
          label="Active providers"
          value={formatInteger(activeProviderCount)}
          detail={`${formatInteger(activeConnections.length)} active connections`}
          tone={activeProviderCount > 0 ? "emerald" : "zinc"}
        />
        <MetricCard
          label="Tokens tracked"
          value={formatCompactNumber(data.summary.total_tokens)}
          detail={`${formatUSD(data.summary.total_cost_usd)} tracked cost`}
          tone="sky"
        />
        <MetricCard
          label="Failed logs"
          value={formatInteger(failedLogs.length)}
          detail={`${formatInteger(streamingLogs.length)} streaming rows in recent logs`}
          tone={failedLogs.length > 0 ? "amber" : "emerald"}
        />
      </div>

      <div className="grid gap-6 xl:grid-cols-[1.2fr_0.8fr]">
        <Panel title="Gateway status" description="Operational snapshot from management APIs.">
          <div className="grid gap-4 sm:grid-cols-3">
            <StatusTile label="Provider health" value={`${activeConnections.length}/${data.connections.length} active`} tone="good" />
            <StatusTile
              label="Request flow"
              value={`${successfulLogs.length} ok / ${failedLogs.length} failed`}
              tone={failedLogs.length > 0 ? "warn" : "good"}
            />
            <StatusTile
              label="MCP health"
              value={`${healthyMcpInstances.length}/${activeMcpInstances.length} healthy`}
              tone={healthyMcpInstances.length === activeMcpInstances.length ? "good" : "warn"}
            />
          </div>
        </Panel>

        <Panel title="Operational inventory" description="Combos and MCP instances returned by the control plane.">
          <div className="space-y-4">
            <InventoryGroup
              label={`${activeCombos.length} active combos`}
              rows={activeCombos.map((combo) => ({ key: combo.ID, name: combo.Name, detail: `${combo.Steps.length} steps` }))}
            />
            <InventoryGroup
              label={`${activeMcpInstances.length} active MCP instances`}
              rows={activeMcpInstances.map((instance) => ({
                key: instance.ID,
                name: instance.Name,
                detail: `${instance.HealthStatus || "unknown"} / ${instance.ToolManifest?.tools?.length ?? 0} tools`
              }))}
            />
            <div className="flex items-center justify-between gap-3 rounded-md border border-zinc-200 px-4 py-3">
              <span className="text-sm font-medium text-zinc-700">MCP tools</span>
              <span className="text-sm font-semibold text-zinc-950">{formatInteger(totalMcpTools)}</span>
            </div>
          </div>
        </Panel>
      </div>
    </div>
  );
}

function StatusTile({ label, tone, value }: { label: string; tone: "good" | "warn"; value: string }) {
  return (
    <div className="rounded-md border border-zinc-200 p-4">
      <p className="text-sm font-medium text-zinc-500">{label}</p>
      <div className="mt-3 flex items-center justify-between gap-3">
        <p className="text-lg font-semibold text-zinc-950">{value}</p>
        <StatusPill tone={tone}>api</StatusPill>
      </div>
    </div>
  );
}

function InventoryGroup({ label, rows }: { label: string; rows: Array<{ detail: string; key: string; name: string }> }) {
  return (
    <div>
      <p className="mb-2 text-sm font-semibold text-zinc-700">{label}</p>
      {rows.length === 0 ? (
        <p className="rounded-md border border-dashed border-zinc-300 px-4 py-3 text-sm text-zinc-500">None returned</p>
      ) : (
        <div className="divide-y divide-zinc-200 rounded-md border border-zinc-200">
          {rows.map((row) => (
            <div key={row.key} className="flex items-center justify-between gap-3 px-4 py-3">
              <span className="text-sm font-medium text-zinc-700">{row.name}</span>
              <span className="text-sm text-zinc-500">{row.detail}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function hasDashboardData(data: DashboardData) {
  return (
    data.connections.length > 0 ||
    data.logs.length > 0 ||
    data.combos.length > 0 ||
    data.mcpInstances.length > 0 ||
    data.summary.request_count > 0 ||
    data.summary.total_tokens > 0 ||
    data.summary.total_cost_usd > 0
  );
}

function isFailedLog(record: UsageLogRecord) {
  return Boolean(record.error) || (record.status_code ?? 0) >= 400;
}

function isSuccessfulLog(record: UsageLogRecord) {
  const status = record.status_code ?? 0;
  return !record.error && status >= 200 && status < 400;
}

function isStreamingRecord(record: UsageLogRecord) {
  const source = record.source_format?.toLowerCase() ?? "";
  const target = record.target_format?.toLowerCase() ?? "";
  return source.includes("stream") || source.includes("sse") || target.includes("stream") || target.includes("sse");
}

function formatInteger(value: number) {
  return value.toLocaleString();
}

function formatCompactNumber(value: number) {
  if (Math.abs(value) >= 1000) {
    return `${(value / 1000).toFixed(1)}k`;
  }
  return formatInteger(value);
}

function formatUSD(value: number) {
  return `$${value.toFixed(2)}`;
}
