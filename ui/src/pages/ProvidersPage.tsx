import { useCallback, useEffect, useState, type FormEvent } from "react";
import {
  ApiError,
  asyncError,
  createConnection,
  deleteConnection,
  exchangeProviderOAuth,
  listConnections,
  listProviders,
  pollProviderOAuth,
  startProviderOAuth,
  testConnection,
  type AsyncState,
  type ConnectionResponse,
  type ProviderOAuthStartResponse,
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
  return <ProviderConnectionsControlPlane showProviderContract />;
}

export function ProviderConnectionsControlPlane({ showProviderContract = false }: { showProviderContract?: boolean }) {
  const [state, setState] = useState<AsyncState<ProviderData>>({ status: "loading" });

  const loadProviders = useCallback(async (showLoading = true) => {
    if (showLoading) {
      setState({ status: "loading" });
    }
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
    <Panel
      title={showProviderContract ? "Provider connections" : "Connections and auth"}
      description={
        showProviderContract
          ? "OAuth and API-token provider accounts available to the proxy."
          : "Provider accounts, OAuth-backed rows, API-token rows, and connection actions available to the proxy."
      }
    >
      {state.status === "loading" || state.status === "idle" ? <LoadingState label="Loading providers" /> : null}
      {state.status === "auth-expired" ? (
        <ErrorState title="Authentication expired" message={state.error.message} onRetry={() => void loadProviders()} />
      ) : null}
      {state.status === "error" ? (
        <ErrorState title="Could not load providers" message={state.error.message} onRetry={() => void loadProviders()} />
      ) : null}
      {state.status === "empty" ? (
        <EmptyState title="No provider records" description="The management API returned no providers or connections." />
      ) : null}
      {state.status === "success" ? (
        <ProviderTables data={state.data} onReload={() => loadProviders(false)} showProviderContract={showProviderContract} />
      ) : null}
    </Panel>
  );
}

function ProviderTables({
  data,
  onReload,
  showProviderContract
}: {
  data: ProviderData;
  onReload: () => Promise<void>;
  showProviderContract: boolean;
}) {
  const apiKeyProviders = data.providers.filter((provider) => provider.auth_types?.includes("api_key"));
  const oauthProviders = data.providers.filter((provider) => provider.auth_types?.includes("oauth"));
  const [provider, setProvider] = useState(apiKeyProviders[0]?.id ?? "");
  const [oauthProvider, setOAuthProvider] = useState(oauthProviders[0]?.id ?? "");
  const [oauthAccountLabel, setOAuthAccountLabel] = useState("");
  const [oauthCallback, setOAuthCallback] = useState("");
  const [oauthSession, setOAuthSession] = useState<ProviderOAuthStartResponse | null>(null);
  const [name, setName] = useState("");
  const [apiKey, setApiKey] = useState("");
  const [isCreating, setIsCreating] = useState(false);
  const [isStartingOAuth, setIsStartingOAuth] = useState(false);
  const [isExchangingOAuth, setIsExchangingOAuth] = useState(false);
  const [isPollingOAuth, setIsPollingOAuth] = useState(false);
  const [busyConnectionID, setBusyConnectionID] = useState("");
  const [mutationError, setMutationError] = useState("");
  const [mutationMessage, setMutationMessage] = useState("");

  const selectedProvider = provider || apiKeyProviders[0]?.id || "";
  const selectedOAuthProvider = oauthProvider || oauthProviders[0]?.id || "";

  async function handleCreateConnection(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedProvider || !name.trim() || !apiKey.trim()) {
      setMutationError("Provider, connection name, and API key are required.");
      setMutationMessage("");
      return;
    }
    setIsCreating(true);
    setMutationError("");
    setMutationMessage("");
    try {
      await createConnection({
        provider: selectedProvider,
        name: name.trim(),
        auth_type: "api_key",
        api_key: apiKey.trim(),
        is_active: true
      });
      setName("");
      setApiKey("");
      setMutationMessage(`${name.trim()} was added`);
      await onReload();
    } catch (error) {
      setMutationError(toApiError(error).message);
    } finally {
      setIsCreating(false);
    }
  }

  async function handleStartOAuth(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedOAuthProvider) {
      setMutationError("OAuth provider is required.");
      setMutationMessage("");
      return;
    }
    setIsStartingOAuth(true);
    setMutationError("");
    setMutationMessage("");
    setOAuthSession(null);
    try {
      const session = await startProviderOAuth(selectedOAuthProvider, oauthAccountLabel.trim());
      setOAuthSession(session);
      setOAuthCallback("");
      setMutationMessage(`OAuth started for ${selectedOAuthProvider}`);
    } catch (error) {
      setMutationError(toApiError(error).message);
    } finally {
      setIsStartingOAuth(false);
    }
  }

  async function handleExchangeOAuth(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedOAuthProvider) {
      setMutationError("OAuth provider is required.");
      setMutationMessage("");
      return;
    }
    const callback = oauthCallback.trim();
    const parsed = parseOAuthCallback(callback, oauthSession?.session_id ?? "");
    if (!parsed.code || !parsed.state) {
      setMutationError("OAuth callback code and state are required.");
      setMutationMessage("");
      return;
    }
    setIsExchangingOAuth(true);
    setMutationError("");
    setMutationMessage("");
    try {
      const connection = await exchangeProviderOAuth(selectedOAuthProvider, parsed.state, parsed.code);
      setOAuthCallback("");
      setOAuthSession(null);
      setMutationMessage(`OAuth connected ${connection.name || connection.id}`);
      await onReload();
    } catch (error) {
      setMutationError(toApiError(error).message);
    } finally {
      setIsExchangingOAuth(false);
    }
  }

  async function handlePollOAuth() {
    if (!selectedOAuthProvider || !oauthSession?.session_id) {
      setMutationError("OAuth session is required.");
      setMutationMessage("");
      return;
    }
    setIsPollingOAuth(true);
    setMutationError("");
    setMutationMessage("");
    try {
      const result = await pollProviderOAuth(selectedOAuthProvider, oauthSession.session_id);
      if (result.status !== "complete" || !result.connection) {
        setMutationMessage(`OAuth ${result.status}`);
        return;
      }
      setOAuthCallback("");
      setOAuthSession(null);
      setMutationMessage(`OAuth connected ${result.connection.name || result.connection.id}`);
      await onReload();
    } catch (error) {
      setMutationError(toApiError(error).message);
    } finally {
      setIsPollingOAuth(false);
    }
  }

  async function handleTestConnection(connection: ConnectionResponse) {
    const label = connection.Name || connection.ID;
    setBusyConnectionID(connection.ID);
    setMutationError("");
    setMutationMessage("");
    try {
      const result = await testConnection(connection.ID);
      setMutationMessage(`${result.name || label} is ${result.ok ? "active" : "inactive"}`);
    } catch (error) {
      setMutationError(toApiError(error).message);
    } finally {
      setBusyConnectionID("");
    }
  }

  async function handleDeleteConnection(connection: ConnectionResponse) {
    const label = connection.Name || connection.ID;
    if (!window.confirm(`Delete provider connection ${label}?`)) {
      return;
    }
    setBusyConnectionID(connection.ID);
    setMutationError("");
    setMutationMessage("");
    try {
      await deleteConnection(connection.ID);
      setMutationMessage(`${label} was deleted`);
      await onReload();
    } catch (error) {
      setMutationError(toApiError(error).message);
    } finally {
      setBusyConnectionID("");
    }
  }

  return (
    <div className="space-y-5">
      {apiKeyProviders.length > 0 ? (
        <form onSubmit={handleCreateConnection} className="grid gap-3 rounded-md border border-zinc-200 bg-zinc-50 p-4 md:grid-cols-[1fr_1fr_1.5fr_auto]">
          <label className="grid gap-1 text-sm font-medium text-zinc-700">
            Provider
            <select
              value={selectedProvider}
              onChange={(event) => setProvider(event.target.value)}
              className="h-10 rounded-md border border-zinc-300 bg-white px-3 text-sm text-zinc-950"
            >
              {apiKeyProviders.map((entry) => (
                <option key={entry.id} value={entry.id}>
                  {entry.id}
                </option>
              ))}
            </select>
          </label>
          <label className="grid gap-1 text-sm font-medium text-zinc-700">
            Connection name
            <input
              value={name}
              onChange={(event) => setName(event.target.value)}
              className="h-10 rounded-md border border-zinc-300 bg-white px-3 text-sm text-zinc-950"
              type="text"
            />
          </label>
          <label className="grid gap-1 text-sm font-medium text-zinc-700">
            Provider API key
            <input
              value={apiKey}
              onChange={(event) => setApiKey(event.target.value)}
              className="h-10 rounded-md border border-zinc-300 bg-white px-3 text-sm text-zinc-950"
              type="password"
            />
          </label>
          <button
            type="submit"
            disabled={isCreating}
            className="h-10 self-end rounded-md bg-zinc-950 px-4 text-sm font-semibold text-white disabled:cursor-not-allowed disabled:bg-zinc-400"
          >
            Add connection
          </button>
        </form>
      ) : null}
      {oauthProviders.length > 0 ? (
        <form onSubmit={handleStartOAuth} className="grid gap-3 rounded-md border border-zinc-200 bg-zinc-50 p-4 md:grid-cols-[1fr_1.5fr_auto]">
          <label className="grid gap-1 text-sm font-medium text-zinc-700">
            OAuth provider
            <select
              value={selectedOAuthProvider}
              onChange={(event) => {
                setOAuthProvider(event.target.value);
                setOAuthSession(null);
                setOAuthCallback("");
              }}
              className="h-10 rounded-md border border-zinc-300 bg-white px-3 text-sm text-zinc-950"
            >
              {oauthProviders.map((entry) => (
                <option key={entry.id} value={entry.id}>
                  {entry.id}
                </option>
              ))}
            </select>
          </label>
          <label className="grid gap-1 text-sm font-medium text-zinc-700">
            OAuth account label
            <input
              value={oauthAccountLabel}
              onChange={(event) => setOAuthAccountLabel(event.target.value)}
              className="h-10 rounded-md border border-zinc-300 bg-white px-3 text-sm text-zinc-950"
              type="text"
            />
          </label>
          <button
            type="submit"
            disabled={isStartingOAuth}
            className="h-10 self-end rounded-md bg-zinc-950 px-4 text-sm font-semibold text-white disabled:cursor-not-allowed disabled:bg-zinc-400"
          >
            Start OAuth
          </button>
        </form>
      ) : null}
      {oauthSession ? (
        <form onSubmit={handleExchangeOAuth} className="grid gap-3 rounded-md border border-zinc-200 bg-white p-4 md:grid-cols-[1fr_auto]">
          <div className="space-y-2 md:col-span-2">
            {oauthSession.auth_url ? (
              <a className="text-sm font-semibold text-zinc-950 underline" href={oauthSession.auth_url} target="_blank" rel="noreferrer">
                Open authorization URL
              </a>
            ) : null}
            {oauthSession.session_id ? <p className="text-sm text-zinc-600">Session state: {oauthSession.session_id}</p> : null}
            {oauthSession.user_code ? <p className="text-sm text-zinc-600">Device code: {oauthSession.user_code}</p> : null}
            {oauthSession.poll_interval ? <p className="text-sm text-zinc-600">Poll interval: {oauthSession.poll_interval}s</p> : null}
            {oauthSession.verification ? (
              <a className="text-sm text-zinc-600 underline" href={oauthSession.verification} target="_blank" rel="noreferrer">
                Verification URL
              </a>
            ) : null}
          </div>
          <label className="grid gap-1 text-sm font-medium text-zinc-700">
            Callback URL or code
            <input
              value={oauthCallback}
              onChange={(event) => setOAuthCallback(event.target.value)}
              className="h-10 rounded-md border border-zinc-300 bg-white px-3 text-sm text-zinc-950"
              type="text"
            />
          </label>
          <button
            type="submit"
            disabled={isExchangingOAuth}
            className="h-10 self-end rounded-md bg-zinc-950 px-4 text-sm font-semibold text-white disabled:cursor-not-allowed disabled:bg-zinc-400"
          >
            Complete OAuth
          </button>
          {oauthSession.user_code || oauthSession.poll_interval ? (
            <button
              type="button"
              disabled={isPollingOAuth}
              onClick={handlePollOAuth}
              className="h-10 rounded-md border border-zinc-300 bg-white px-4 text-sm font-semibold text-zinc-950 disabled:cursor-not-allowed disabled:text-zinc-400 md:col-start-2"
            >
              Poll OAuth
            </button>
          ) : null}
        </form>
      ) : null}
      {mutationMessage ? <p className="text-sm font-medium text-emerald-700">{mutationMessage}</p> : null}
      {mutationError ? <p className="text-sm font-medium text-red-700">{mutationError}</p> : null}
      {showProviderContract ? (
        <div>
        <div className="mb-3 flex items-center justify-between gap-3">
          <h4 className="text-sm font-semibold text-zinc-700">Provider contract</h4>
          <span className="text-sm text-zinc-500">{data.providers.length} providers</span>
        </div>
        {data.providers.length === 0 ? (
          <EmptyState title="No providers" description="The provider matrix endpoint returned an empty list." />
        ) : (
          <div className="overflow-x-auto rounded-md border border-zinc-200">
            <table aria-label="Provider contract" className="min-w-[780px] w-full text-left text-sm">
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
      ) : null}

      <div>
        <div className="mb-3 flex items-center justify-between gap-3">
          <h4 className="text-sm font-semibold text-zinc-700">Connections</h4>
          <span className="text-sm text-zinc-500">{data.connections.length} accounts</span>
        </div>
        {data.connections.length === 0 ? (
          <EmptyState title="No connections" description="No provider accounts are stored yet." />
        ) : (
          <div className="overflow-x-auto rounded-md border border-zinc-200">
            <table aria-label="Provider connections" className="min-w-[780px] w-full text-left text-sm">
              <thead className="bg-zinc-50 text-xs uppercase text-zinc-500">
                <tr>
                  <th className="px-4 py-3 font-semibold">Name</th>
                  <th className="px-4 py-3 font-semibold">Provider</th>
                  <th className="px-4 py-3 font-semibold">Account</th>
                  <th className="px-4 py-3 font-semibold">Auth</th>
                  <th className="px-4 py-3 font-semibold">Status</th>
                  <th className="px-4 py-3 font-semibold">Backoff</th>
                  <th className="px-4 py-3 font-semibold">Actions</th>
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
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-2">
                          <button
                            type="button"
                            onClick={() => void handleTestConnection(connection)}
                            disabled={busyConnectionID === connection.ID}
                            className="rounded-md border border-zinc-300 px-3 py-1.5 text-xs font-semibold text-zinc-700 disabled:cursor-not-allowed disabled:text-zinc-400"
                            aria-label={`Test ${connection.Name || connection.ID}`}
                          >
                            Test
                          </button>
                          <button
                            type="button"
                            onClick={() => void handleDeleteConnection(connection)}
                            disabled={busyConnectionID === connection.ID}
                            className="rounded-md border border-red-200 px-3 py-1.5 text-xs font-semibold text-red-700 disabled:cursor-not-allowed disabled:text-red-300"
                            aria-label={`Delete ${connection.Name || connection.ID}`}
                          >
                            Delete
                          </button>
                        </div>
                      </td>
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

function formatList(values?: string[] | null) {
  return values == null || values.length === 0 ? "none" : values.join(", ");
}

function parseOAuthCallback(value: string, fallbackState: string) {
  if (!value) {
    return { code: "", state: fallbackState };
  }
  try {
    const parsed = new URL(value);
    return {
      code: parsed.searchParams.get("code")?.trim() ?? "",
      state: parsed.searchParams.get("state")?.trim() || fallbackState
    };
  } catch {
    return { code: value, state: fallbackState };
  }
}

function toApiError(error: unknown) {
  if (error instanceof ApiError) {
    return error;
  }
  return new ApiError(0, error instanceof Error ? error.message : "request failed", error);
}
