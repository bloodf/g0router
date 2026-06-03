import { useCallback, useEffect, useState } from "react";
import {
  ApiError,
  asyncError,
  listConnections,
  listProviders,
  type AsyncState,
  type ConnectionResponse,
  type ProviderMatrixEntry
} from "../api";
import { EmptyState, ErrorState, LoadingState, Panel, StatusPill } from "../components/Primitives";

type ProviderData = {
  providers: ProviderMatrixEntry[];
  connections: ConnectionResponse[];
};

const providerStatusTone = {
  active: "good",
  disabled: "bad",
  partial: "warn",
  planned: "neutral",
  supported: "good",
  unsupported: "bad"
} as const;

const connectionStatusTone = {
  active: "good",
  cooldown: "warn",
  degraded: "warn",
  inactive: "bad"
} as const;

export function ProvidersPage() {
  const [state, setState] = useState<AsyncState<ProviderData>>({ status: "loading" });

  const loadProviders = useCallback(async () => {
    setState({ status: "loading" });
    try {
      const [providers, connections] = await Promise.all([listProviders(), listConnections()]);
      const data = { providers, connections };
      setState(providers.length === 0 && connections.length === 0 ? { status: "empty", data } : { status: "success", data });
    } catch (error) {
      setState(asyncError(toApiError(error)));
    }
  }, []);

  useEffect(() => {
    void loadProviders();
  }, [loadProviders]);

  return (
    <Panel title="Provider connections" description="OAuth and API-token provider accounts available to the proxy.">
      {state.status === "loading" || state.status === "idle" ? <LoadingState label="Loading providers" /> : null}
      {state.status === "auth-expired" ? (
        <ErrorState title="Authentication expired" message={state.error.message} onRetry={loadProviders} />
      ) : null}
      {state.status === "error" ? (
        <ErrorState title="Could not load providers" message={state.error.message} onRetry={loadProviders} />
      ) : null}
      {state.status === "empty" ? (
        <EmptyState title="No provider records" description="The management API returned no providers or connections." />
      ) : null}
      {state.status === "success" ? <ProviderTables data={state.data} /> : null}
    </Panel>
  );
}

function ProviderTables({ data }: { data: ProviderData }) {
  return (
    <div className="space-y-5">
      <div>
        <div className="mb-3 flex items-center justify-between gap-3">
          <h4 className="text-sm font-semibold text-zinc-700">Provider contract</h4>
          <span className="text-sm text-zinc-500">{data.providers.length} providers</span>
        </div>
        {data.providers.length === 0 ? (
          <EmptyState title="No providers" description="The provider matrix endpoint returned an empty list." />
        ) : (
          <div className="overflow-hidden rounded-md border border-zinc-200">
            <table className="w-full text-left text-sm">
              <thead className="bg-zinc-50 text-xs uppercase text-zinc-500">
                <tr>
                  <th className="px-4 py-3 font-semibold">Provider</th>
                  <th className="px-4 py-3 font-semibold">Status</th>
                  <th className="px-4 py-3 font-semibold">Auth</th>
                  <th className="px-4 py-3 font-semibold">Capabilities</th>
                  <th className="px-4 py-3 font-semibold">Notes</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-200">
                {data.providers.map((provider) => (
                  <tr key={provider.id}>
                    <td className="px-4 py-3 font-semibold text-zinc-950">{provider.id}</td>
                    <td className="px-4 py-3">
                      <StatusPill tone={providerTone(provider.public_status)}>{provider.public_status || "unknown"}</StatusPill>
                    </td>
                    <td className="px-4 py-3 text-zinc-600">{formatList(provider.auth_types)}</td>
                    <td className="px-4 py-3 text-zinc-600">{formatCapabilities(provider)}</td>
                    <td className="px-4 py-3 text-zinc-600">{provider.notes || "none"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      <div>
        <div className="mb-3 flex items-center justify-between gap-3">
          <h4 className="text-sm font-semibold text-zinc-700">Connections</h4>
          <span className="text-sm text-zinc-500">{data.connections.length} accounts</span>
        </div>
        {data.connections.length === 0 ? (
          <EmptyState title="No connections" description="No provider accounts are stored yet." />
        ) : (
          <div className="overflow-hidden rounded-md border border-zinc-200">
            <table className="w-full text-left text-sm">
              <thead className="bg-zinc-50 text-xs uppercase text-zinc-500">
                <tr>
                  <th className="px-4 py-3 font-semibold">Name</th>
                  <th className="px-4 py-3 font-semibold">Provider</th>
                  <th className="px-4 py-3 font-semibold">Account</th>
                  <th className="px-4 py-3 font-semibold">Auth</th>
                  <th className="px-4 py-3 font-semibold">Status</th>
                  <th className="px-4 py-3 font-semibold">Backoff</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-zinc-200">
                {data.connections.map((connection) => {
                  const status = connectionStatus(connection);
                  return (
                    <tr key={connection.ID}>
                      <td className="px-4 py-3 font-semibold text-zinc-950">{connection.Name || connection.ID}</td>
                      <td className="px-4 py-3 text-zinc-600">{connection.Provider}</td>
                      <td className="px-4 py-3 text-zinc-600">{connection.Email || connection.AccountID || "local"}</td>
                      <td className="px-4 py-3 text-zinc-600">{connection.AuthType}</td>
                      <td className="px-4 py-3">
                        <StatusPill tone={connectionStatusTone[status]}>{status}</StatusPill>
                      </td>
                      <td className="px-4 py-3 text-zinc-600">{connection.BackoffLevel}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}

function providerTone(status: string) {
  return providerStatusTone[status.toLowerCase() as keyof typeof providerStatusTone] ?? "neutral";
}

function connectionStatus(connection: ConnectionResponse): keyof typeof connectionStatusTone {
  if (!connection.IsActive) {
    return "inactive";
  }
  if (connection.UnavailableUntil && connection.UnavailableUntil > Math.floor(Date.now() / 1000)) {
    return "cooldown";
  }
  if (connection.BackoffLevel > 0) {
    return "degraded";
  }
  return "active";
}

function formatCapabilities(provider: ProviderMatrixEntry) {
  const capabilities = [
    provider.inference ? "inference" : "",
    provider.streaming ? "streaming" : "",
    provider.model_catalog ? "models" : "",
    provider.quota ? "quota" : ""
  ].filter(Boolean);
  return formatList(capabilities);
}

function formatList(values: string[]) {
  return values.length === 0 ? "none" : values.join(", ");
}

function toApiError(error: unknown) {
  if (error instanceof ApiError) {
    return error;
  }
  return new ApiError(0, error instanceof Error ? error.message : "request failed", error);
}
