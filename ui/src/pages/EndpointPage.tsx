import { useCallback, useEffect, useState, type FormEvent } from "react";
import {
  ApiError,
  asyncError,
  asyncSuccess,
  createAPIKey,
  deleteAPIKey,
  listAPIKeys,
  type APIKeyResponse,
  type AsyncState,
  type CreateAPIKeyResponse
} from "../api";
import { EmptyState, ErrorState, LoadingState, Panel, StatusPill } from "../components/Primitives";

export function EndpointPage() {
  const [state, setState] = useState<AsyncState<APIKeyResponse[]>>({ status: "loading" });
  const [keyName, setKeyName] = useState("");
  const [createdKey, setCreatedKey] = useState<CreateAPIKeyResponse | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);
  const [deletingID, setDeletingID] = useState<string | null>(null);

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

  return (
    <Panel title="Endpoint controls" description="API key, request transformation, and endpoint protection controls.">
      <div className="grid gap-6 xl:grid-cols-[1.15fr_0.85fr]">
        <div>
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
          {state.status === "success" ? <KeysTable keys={state.data} deletingID={deletingID} onDelete={handleDelete} /> : null}
        </div>

        <div className="space-y-4">
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
  keys: APIKeyResponse[];
  onDelete: (key: APIKeyResponse) => void;
};

function KeysTable({ deletingID, keys, onDelete }: KeysTableProps) {
  return (
    <div className="overflow-hidden rounded-md border border-zinc-200">
      <table className="w-full text-left text-sm">
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
            <tr key={key.ID}>
              <td className="px-4 py-3 font-medium text-zinc-950">{key.Name}</td>
              <td className="px-4 py-3 font-mono text-xs text-zinc-600">{key.Prefix}</td>
              <td className="px-4 py-3">
                <StatusPill tone={key.IsActive ? "good" : "bad"}>{key.IsActive ? "active" : "inactive"}</StatusPill>
              </td>
              <td className="px-4 py-3 text-zinc-600">{formatTimestamp(key.LastUsedAt)}</td>
              <td className="px-4 py-3 text-zinc-600">{formatTimestamp(key.CreatedAt)}</td>
              <td className="px-4 py-3">
                <button
                  aria-label={`Delete ${key.Name}`}
                  className="rounded-md border border-zinc-200 px-3 py-2 text-sm font-semibold text-zinc-700 disabled:cursor-not-allowed disabled:text-zinc-400"
                  type="button"
                  disabled={deletingID === key.ID}
                  onClick={() => onDelete(key)}
                >
                  {deletingID === key.ID ? "Deleting" : "Delete"}
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
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
