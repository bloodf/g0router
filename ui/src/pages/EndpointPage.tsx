import { useCallback, useEffect, useState, type FormEvent } from "react";
import {
  ApiError,
  asyncError,
  asyncSuccess,
  createAPIKey,
  deleteAPIKey,
  listAPIKeys,
  updateAPIKeyPolicy,
  type APIKeyPolicy,
  type APIKeyResponse,
  type AsyncState,
  type CreateAPIKeyResponse
} from "../api";
import { EmptyState, ErrorState, LoadingState, Panel, StatusPill } from "../components/Primitives";

export function EndpointPage() {
  return <APIKeysControlPlane showEndpointControls />;
}

export function APIKeysControlPlane({ showEndpointControls = false }: { showEndpointControls?: boolean }) {
  const [state, setState] = useState<AsyncState<APIKeyResponse[]>>({ status: "loading" });
  const [keyName, setKeyName] = useState("");
  const [createdKey, setCreatedKey] = useState<CreateAPIKeyResponse | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [copiedEndpoint, setCopiedEndpoint] = useState("");
  const [creating, setCreating] = useState(false);
  const [deletingID, setDeletingID] = useState<string | null>(null);
  const [editingPolicyKey, setEditingPolicyKey] = useState<APIKeyResponse | null>(null);

  const loadKeys = useCallback(async () => {
    setState({ status: "loading" });
    try {
      setState(asyncSuccess(await listAPIKeys()));
    } catch (error) {
      setState(asyncError(toApiError(error)));
    }
  }, []);

  useEffect(() => {
    void loadKeys();
  }, [loadKeys]);

  async function handleCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const name = keyName.trim();
    if (name === "") {
      return;
    }

    setCreating(true);
    setActionError(null);
    try {
      const response = await createAPIKey(name);
      setCreatedKey(response);
      setKeyName("");
      setState((current) => appendCreatedKey(current, response.key));
    } catch (error) {
      const apiError = toApiError(error);
      if (apiError.authExpired) {
        setState(asyncError(apiError));
      } else {
        setActionError(apiError.message);
      }
    } finally {
      setCreating(false);
    }
  }

  async function handleDelete(key: APIKeyResponse) {
    if (!window.confirm(`Delete API key ${key.Name}?`)) {
      return;
    }

    setDeletingID(key.ID);
    setActionError(null);
    setCreatedKey(null);
    try {
      await deleteAPIKey(key.ID);
      await loadKeys();
    } catch (error) {
      const apiError = toApiError(error);
      if (apiError.authExpired) {
        setState(asyncError(apiError));
      } else {
        setActionError(apiError.message);
      }
    } finally {
      setDeletingID(null);
    }
  }

  async function handleCopyEndpoint(path: string) {
    const endpoint = `${window.location.origin}${path}`;
    await navigator.clipboard.writeText(endpoint);
    setCopiedEndpoint(endpoint);
  }

  async function handleSavePolicy(key: APIKeyResponse, policy: APIKeyPolicy) {
    setActionError(null);
    try {
      await updateAPIKeyPolicy(key.ID, policy);
      setEditingPolicyKey(null);
      await loadKeys();
    } catch (error) {
      const apiError = toApiError(error);
      if (apiError.authExpired) {
        setState(asyncError(apiError));
      } else {
        setActionError(apiError.message);
      }
    }
  }

  return (
    <Panel
      title={showEndpointControls ? "Endpoint controls" : "API keys"}
      description={showEndpointControls ? "API key, request transformation, and endpoint protection controls." : "Gateway API keys for authenticated client traffic."}
    >
      <div className="mb-5 rounded-md border border-sky-200 bg-sky-50 p-4">
        <p className="text-sm font-semibold text-sky-900">An API key is required to call the proxy</p>
        <p className="mt-1 text-sm leading-6 text-sky-800">
          Every request to <code className="font-mono">/v1/*</code> must send a gateway API key. Generate one below; the full key
          is shown once at creation and cannot be retrieved later.
        </p>
      </div>

      {showEndpointControls ? (
        <div className="mb-5 rounded-md border border-zinc-200 p-4">
          <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
            <div>
              <h4 className="text-sm font-semibold text-zinc-700">OpenAI-compatible endpoints</h4>
              <p className="mt-1 text-sm text-zinc-500">Use these local URLs in OpenAI-compatible clients.</p>
            </div>
            <div className="flex flex-wrap gap-2">
              <button
                className="rounded-md border border-zinc-200 px-3 py-2 text-sm font-semibold text-zinc-700"
                type="button"
                onClick={() => void handleCopyEndpoint("/v1/chat/completions")}
              >
                Copy chat completions endpoint
              </button>
              <button
                className="rounded-md border border-zinc-200 px-3 py-2 text-sm font-semibold text-zinc-700"
                type="button"
                onClick={() => void handleCopyEndpoint("/v1/models")}
              >
                Copy models endpoint
              </button>
            </div>
          </div>
          {copiedEndpoint ? (
            <p className="mt-3 text-sm font-semibold text-emerald-700">Endpoint copied</p>
          ) : null}
        </div>
      ) : null}

      <div className="grid gap-6 xl:grid-cols-[1.15fr_0.85fr]">
        <div className="min-w-0">
          <div className="mb-3 flex items-center justify-between gap-3">
            <h4 className="text-sm font-semibold text-zinc-700">API keys</h4>
            {state.status === "success" || state.status === "empty" ? (
              <span className="text-sm text-zinc-500">{state.data.length} keys</span>
            ) : null}
          </div>
          {state.status === "loading" || state.status === "idle" ? <LoadingState label="Loading API keys" /> : null}
          {state.status === "auth-expired" ? (
            <ErrorState title="Authentication expired" message={state.error.message} onRetry={loadKeys} />
          ) : null}
          {state.status === "error" ? <ErrorState title="Could not load API keys" message={state.error.message} onRetry={loadKeys} /> : null}
          {state.status === "empty" ? (
            <EmptyState title="No API keys" description="Create a gateway key before routing protected client requests." />
          ) : null}
          {state.status === "success" ? (
            <KeysTable
              keys={state.data}
              deletingID={deletingID}
              editingPolicyKey={editingPolicyKey}
              onDelete={handleDelete}
              onEditPolicy={setEditingPolicyKey}
              onSavePolicy={handleSavePolicy}
              onCancelPolicy={() => setEditingPolicyKey(null)}
            />
          ) : null}
        </div>

        <div className="min-w-0 space-y-4">
          <form className="rounded-md border border-zinc-200 p-4" onSubmit={handleCreate}>
            <label className="block text-sm font-semibold text-zinc-700" htmlFor="endpoint-key-name">
              Key name
            </label>
            <div className="mt-3 flex gap-2">
              <input
                id="endpoint-key-name"
                className="min-w-0 flex-1 rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-950 outline-none focus:border-zinc-400"
                value={keyName}
                onChange={(event) => setKeyName(event.target.value)}
              />
              <button
                className="rounded-md bg-zinc-950 px-3 py-2 text-sm font-semibold text-white disabled:cursor-not-allowed disabled:bg-zinc-400"
                type="submit"
                disabled={creating || keyName.trim() === ""}
              >
                {creating ? "Creating" : "Create key"}
              </button>
            </div>
          </form>

          {createdKey ? (
            <div className="rounded-md border border-amber-200 bg-amber-50 p-4">
              <p className="text-sm font-semibold text-amber-900">New gateway key</p>
              <code className="mt-3 block overflow-x-auto rounded-md bg-white px-3 py-2 text-xs font-semibold text-zinc-950">
                {createdKey.raw}
              </code>
              <p className="mt-2 text-sm leading-6 text-amber-800">Copy it now. It is not available from stored key data.</p>
              <button className="mt-3 rounded-md border border-amber-200 bg-white px-3 py-2 text-sm font-semibold text-amber-800" type="button" onClick={() => setCreatedKey(null)}>
                Dismiss
              </button>
            </div>
          ) : null}

          {actionError ? <ErrorState title="API key action failed" message={actionError} /> : null}
        </div>
      </div>
    </Panel>
  );
}

type KeysTableProps = {
  deletingID: string | null;
  editingPolicyKey: APIKeyResponse | null;
  keys: APIKeyResponse[];
  onCancelPolicy: () => void;
  onDelete: (key: APIKeyResponse) => void;
  onEditPolicy: (key: APIKeyResponse) => void;
  onSavePolicy: (key: APIKeyResponse, policy: APIKeyPolicy) => void;
};

function KeysTable({ deletingID, editingPolicyKey, keys, onCancelPolicy, onDelete, onEditPolicy, onSavePolicy }: KeysTableProps) {
  return (
    <div className="overflow-x-auto rounded-md border border-zinc-200">
      <table aria-label="API keys" className="min-w-[760px] w-full text-left text-sm">
        <thead className="bg-zinc-50 text-xs uppercase text-zinc-500">
          <tr>
            <th className="px-4 py-3 font-semibold">Name</th>
            <th className="px-4 py-3 font-semibold">Prefix</th>
            <th className="px-4 py-3 font-semibold">Status</th>
            <th className="px-4 py-3 font-semibold">Last used</th>
            <th className="px-4 py-3 font-semibold">Created</th>
            <th className="px-4 py-3 font-semibold">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-zinc-200">
          {keys.map((key) => (
            <>
              <tr key={key.ID}>
                <td className="px-4 py-3 font-medium text-zinc-950">{key.Name}</td>
                <td className="px-4 py-3 font-mono text-xs text-zinc-600">{key.Prefix}</td>
                <td className="px-4 py-3">
                  <StatusPill tone={key.IsActive ? "good" : "bad"}>{key.IsActive ? "active" : "inactive"}</StatusPill>
                </td>
                <td className="px-4 py-3 text-zinc-600">{formatTimestamp(key.LastUsedAt)}</td>
                <td className="px-4 py-3 text-zinc-600">{formatTimestamp(key.CreatedAt)}</td>
                <td className="px-4 py-3">
                  <div className="flex items-center gap-2">
                    <button
                      aria-label={`Edit policy ${key.Name}`}
                      className="rounded-md border border-zinc-200 px-3 py-2 text-sm font-semibold text-zinc-700 disabled:cursor-not-allowed disabled:text-zinc-400"
                      type="button"
                      disabled={deletingID === key.ID}
                      onClick={() => onEditPolicy(key)}
                    >
                      Edit policy
                    </button>
                    <button
                      aria-label={`Delete ${key.Name}`}
                      className="rounded-md border border-zinc-200 px-3 py-2 text-sm font-semibold text-zinc-700 disabled:cursor-not-allowed disabled:text-zinc-400"
                      type="button"
                      disabled={deletingID === key.ID}
                      onClick={() => onDelete(key)}
                    >
                      {deletingID === key.ID ? "Deleting" : "Delete"}
                    </button>
                  </div>
                </td>
              </tr>
              {editingPolicyKey?.ID === key.ID ? (
                <tr key={`${key.ID}-policy`}>
                  <td colSpan={6} className="px-4 py-3">
                    <PolicyForm key={key.ID} apiKey={key} onSave={onSavePolicy} onCancel={onCancelPolicy} />
                  </td>
                </tr>
              ) : null}
            </>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function PolicyForm({ apiKey, onSave, onCancel }: { apiKey: APIKeyResponse; onSave: (key: APIKeyResponse, policy: APIKeyPolicy) => void; onCancel: () => void }) {
  const [expiresAt, setExpiresAt] = useState<number | null>(apiKey.expires_at ?? null);
  const [scopes, setScopes] = useState<string>((apiKey.scopes ?? []).join(", "));
  const [rateLimitRpm, setRateLimitRpm] = useState<number | null>(apiKey.rate_limit_rpm ?? null);
  const [rateLimitTpm, setRateLimitTpm] = useState<number | null>(apiKey.rate_limit_tpm ?? null);
  const [dailySpendCapUsd, setDailySpendCapUsd] = useState<number | null>(apiKey.daily_spend_cap_usd ?? null);

  function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const scopeList = scopes.split(",").map((s) => s.trim()).filter(Boolean);
    onSave(apiKey, {
      expires_at: expiresAt,
      scopes: scopeList,
      rate_limit_rpm: rateLimitRpm,
      rate_limit_tpm: rateLimitTpm,
      daily_spend_cap_usd: dailySpendCapUsd
    });
  }

  return (
    <form className="space-y-3 rounded-md border border-zinc-200 bg-zinc-50 p-4" onSubmit={handleSubmit}>
      <p className="text-sm font-semibold text-zinc-700">Edit policy: {apiKey.Name}</p>
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <label className="block text-sm font-medium text-zinc-700">
          Rate limit RPM
          <input
            aria-label="Rate limit RPM"
            className="mt-1 w-full rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-950"
            min={0}
            type="number"
            value={rateLimitRpm ?? ""}
            onChange={(e) => setRateLimitRpm(e.target.value === "" ? null : Number(e.target.value))}
          />
        </label>
        <label className="block text-sm font-medium text-zinc-700">
          Rate limit TPM
          <input
            aria-label="Rate limit TPM"
            className="mt-1 w-full rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-950"
            min={0}
            type="number"
            value={rateLimitTpm ?? ""}
            onChange={(e) => setRateLimitTpm(e.target.value === "" ? null : Number(e.target.value))}
          />
        </label>
        <label className="block text-sm font-medium text-zinc-700">
          Daily spend cap (USD)
          <input
            aria-label="Daily spend cap (USD)"
            className="mt-1 w-full rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-950"
            min={0}
            step="0.01"
            type="number"
            value={dailySpendCapUsd ?? ""}
            onChange={(e) => setDailySpendCapUsd(e.target.value === "" ? null : Number(e.target.value))}
          />
        </label>
        <label className="block text-sm font-medium text-zinc-700">
          Expires at (unix seconds)
          <input
            aria-label="Expires at (unix seconds)"
            className="mt-1 w-full rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-950"
            min={0}
            type="number"
            value={expiresAt ?? ""}
            onChange={(e) => setExpiresAt(e.target.value === "" ? null : Number(e.target.value))}
          />
        </label>
        <label className="block text-sm font-medium text-zinc-700 sm:col-span-2">
          Scopes (comma-separated, empty = all models)
          <input
            aria-label="Scopes"
            className="mt-1 w-full rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-950"
            placeholder="e.g. openai, anthropic"
            type="text"
            value={scopes}
            onChange={(e) => setScopes(e.target.value)}
          />
        </label>
      </div>
      <div className="flex gap-2">
        <button
          className="rounded-md bg-zinc-950 px-3 py-2 text-sm font-semibold text-white"
          type="submit"
        >
          Save policy
        </button>
        <button
          className="rounded-md border border-zinc-200 px-3 py-2 text-sm font-semibold text-zinc-700"
          type="button"
          onClick={onCancel}
        >
          Cancel
        </button>
      </div>
    </form>
  );
}

function appendCreatedKey(state: AsyncState<APIKeyResponse[]>, key: APIKeyResponse): AsyncState<APIKeyResponse[]> {
  if (state.status !== "success" && state.status !== "empty") {
    return { status: "success", data: [key] };
  }
  const existing = state.data.filter((apiKey) => apiKey.ID !== key.ID);
  return { status: "success", data: [...existing, key] };
}

function formatTimestamp(value?: string | null) {
  return value && value !== "" ? value : "never";
}

function toApiError(error: unknown) {
  if (error instanceof ApiError) {
    return error;
  }
  return new ApiError(0, error instanceof Error ? error.message : "request failed", error);
}
