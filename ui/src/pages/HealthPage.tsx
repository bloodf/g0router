import { useCallback, useEffect, useState } from "react";
import {
  ApiError,
  asyncError,
  asyncSuccess,
  listConnections,
  listProviders,
  type AsyncState,
  type ConnectionResponse,
  type ProviderMatrixEntry
} from "../api";
import { EmptyState, ErrorState, LoadingState, Panel, StatusPill } from "../components/Primitives";

type HealthData = {
  connections: ConnectionResponse[];
  providers: ProviderMatrixEntry[];
};

function toApiError(error: unknown): ApiError {
  if (error instanceof ApiError) return error;
  return new ApiError(0, error instanceof Error ? error.message : "Unknown error", error);
}

export function HealthPage() {
  const [state, setState] = useState<AsyncState<HealthData>>({ status: "loading" });

  const loadHealth = useCallback(async () => {
    setState({ status: "loading" });
    try {
      const [connections, providers] = await Promise.all([listConnections(), listProviders()]);
      const data = { connections, providers };
      if (connections.length === 0) {
        setState({ status: "empty", data });
      } else {
        setState(asyncSuccess(data));
      }
    } catch (error) {
      setState(asyncError<HealthData>(toApiError(error)));
    }
  }, []);

  useEffect(() => {
    void loadHealth();
  }, [loadHealth]);

  return (
    <Panel title="Provider health" description="Per-connection health status, backoff windows, token expiry, and re-auth needs.">
      {renderState(state, loadHealth)}
    </Panel>
  );
}

function renderState(state: AsyncState<HealthData>, onRetry: () => void) {
  switch (state.status) {
    case "idle":
    case "loading":
      return <LoadingState label="Loading health" />;
    case "empty":
      return <EmptyState title="No connections" description="No provider connections are stored yet." />;
    case "error":
      return <ErrorState title="Could not load health" message={state.error.message} onRetry={onRetry} />;
    case "auth-expired":
      return <ErrorState title="Session expired" message={state.error.message} onRetry={onRetry} />;
    case "success":
      return <HealthTable connections={state.data.connections} />;
  }
}

function connectionStatusTone(conn: ConnectionResponse): "good" | "warn" | "bad" {
  if (!conn.IsActive) return "bad";
  if (conn.NeedsReauth) return "bad";
  if (conn.BackoffLevel > 0) return "warn";
  if (conn.UnavailableUntil && conn.UnavailableUntil > Math.floor(Date.now() / 1000)) return "warn";
  return "good";
}

function connectionStatusLabel(conn: ConnectionResponse): string {
  if (!conn.IsActive) return "inactive";
  if (conn.NeedsReauth) return "reauth";
  if (conn.UnavailableUntil && conn.UnavailableUntil > Math.floor(Date.now() / 1000)) return "cooldown";
  if (conn.BackoffLevel > 0) return "degraded";
  return "active";
}

function formatExpiry(expiresAt: number | null | undefined): string {
  if (expiresAt == null) return "—";
  const d = new Date(expiresAt * 1000);
  const now = Date.now();
  if (expiresAt * 1000 < now) return `expired ${d.toLocaleString()}`;
  return d.toLocaleString();
}

function formatUnavailableUntil(ts: number | null | undefined): string {
  if (ts == null) return "—";
  const d = new Date(ts * 1000);
  const now = Date.now();
  if (ts * 1000 < now) return "—";
  return d.toLocaleString();
}

function HealthTable({ connections }: { connections: ConnectionResponse[] }) {
  return (
    <div className="overflow-x-auto rounded-md border border-zinc-200">
      <table aria-label="Connection health" className="min-w-[860px] w-full text-left text-sm">
        <thead className="bg-zinc-50 text-xs uppercase text-zinc-500">
          <tr>
            <th className="px-4 py-3 font-semibold">Name</th>
            <th className="px-4 py-3 font-semibold">Provider</th>
            <th className="px-4 py-3 font-semibold">Account</th>
            <th className="px-4 py-3 font-semibold">Auth</th>
            <th className="px-4 py-3 font-semibold">Status</th>
            <th className="px-4 py-3 font-semibold">Backoff</th>
            <th className="px-4 py-3 font-semibold">Backoff window</th>
            <th className="px-4 py-3 font-semibold">Token expiry</th>
            <th className="px-4 py-3 font-semibold">Last error</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-zinc-200">
          {connections.map((conn) => {
            const tone = connectionStatusTone(conn);
            const label = connectionStatusLabel(conn);
            return (
              <tr key={conn.ID}>
                <td className="px-4 py-3 font-semibold text-zinc-950">{conn.Name || conn.ID}</td>
                <td className="px-4 py-3 text-zinc-600">{conn.Provider}</td>
                <td className="px-4 py-3 text-zinc-600">{conn.Email || conn.AccountID || "local"}</td>
                <td className="px-4 py-3 text-zinc-600">{conn.AuthType}</td>
                <td className="px-4 py-3">
                  <div className="flex flex-wrap items-center gap-1.5">
                    <StatusPill tone={tone}>{label}</StatusPill>
                    {conn.NeedsReauth ? (
                      <StatusPill tone="bad" title={conn.LastRefreshError ?? undefined}>
                        Needs re-auth
                      </StatusPill>
                    ) : null}
                  </div>
                </td>
                <td className="px-4 py-3 text-zinc-600">{conn.BackoffLevel}</td>
                <td className="px-4 py-3 text-zinc-600 font-mono text-xs">{formatUnavailableUntil(conn.UnavailableUntil)}</td>
                <td className="px-4 py-3 text-zinc-600 font-mono text-xs">{formatExpiry(conn.ExpiresAt)}</td>
                <td className="px-4 py-3 text-zinc-600 text-xs">{conn.LastRefreshError || "—"}</td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
