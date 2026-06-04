import { useCallback, useEffect, useState, type FormEvent } from "react";
import { ApiError, apiFetch, asyncError, asyncSuccess, getCombosPath, listCombos, type AsyncState, type ComboResponse } from "../api";
import { EmptyState, ErrorState, LoadingState, Panel, StatusPill } from "../components/Primitives";

type ComboForm = {
  isActive: boolean;
  model: string;
  name: string;
  provider: string;
};

const emptyForm: ComboForm = {
  isActive: true,
  model: "",
  name: "",
  provider: ""
};

export function CombosPage() {
  const [combosState, setCombosState] = useState<AsyncState<ComboResponse[]>>({ status: "loading" });
  const [form, setForm] = useState<ComboForm>(emptyForm);
  const [isCreating, setIsCreating] = useState(false);
  const [deletingID, setDeletingID] = useState<string | null>(null);
  const [mutationError, setMutationError] = useState<ApiError | null>(null);

  const loadCombos = useCallback(async () => {
    setCombosState({ status: "loading" });
    try {
      const combos = await listCombos();
      setCombosState(asyncSuccess(combos));
    } catch (error) {
      setCombosState(asyncError<ComboResponse[]>(toApiError(error)));
    }
  }, []);

  useEffect(() => {
    void loadCombos();
  }, [loadCombos]);

  async function handleCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setMutationError(null);
    setIsCreating(true);
    try {
      await apiFetch<ComboResponse>(getCombosPath(), {
        method: "POST",
        body: {
          name: form.name.trim(),
          steps: [{ provider: form.provider.trim(), model: form.model.trim() }],
          is_active: form.isActive
        }
      });
      setForm(emptyForm);
      await loadCombos();
    } catch (error) {
      setMutationError(toApiError(error));
    } finally {
      setIsCreating(false);
    }
  }

  async function handleDelete(combo: ComboResponse) {
    if (!window.confirm(`Delete combo ${combo.Name}?`)) {
      return;
    }

    setMutationError(null);
    setDeletingID(combo.ID);
    try {
      await apiFetch<void>(`${getCombosPath()}/${encodeURIComponent(combo.ID)}`, { method: "DELETE" });
      await loadCombos();
    } catch (error) {
      setMutationError(toApiError(error));
    } finally {
      setDeletingID(null);
    }
  }

  const canCreate = form.name.trim() !== "" && form.provider.trim() !== "" && form.model.trim() !== "" && !isCreating;

  return (
    <Panel title="Combo routing" description="Reusable routing chains for fallback, round-robin, and account selection.">
      <div className="space-y-5">
        <form className="rounded-md border border-zinc-200 p-4" onSubmit={handleCreate}>
          <div className="grid gap-3 lg:grid-cols-[1.1fr_1fr_1.3fr_auto]">
            <label className="text-sm font-medium text-zinc-700">
              Combo name
              <input
                className="mt-1 w-full rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-950"
                value={form.name}
                onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))}
              />
            </label>
            <label className="text-sm font-medium text-zinc-700">
              Step provider
              <input
                className="mt-1 w-full rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-950"
                value={form.provider}
                onChange={(event) => setForm((current) => ({ ...current, provider: event.target.value }))}
              />
            </label>
            <label className="text-sm font-medium text-zinc-700">
              Step model
              <input
                className="mt-1 w-full rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-950"
                value={form.model}
                onChange={(event) => setForm((current) => ({ ...current, model: event.target.value }))}
              />
            </label>
            <div className="flex items-end gap-3">
              <label className="flex min-h-10 items-center gap-2 text-sm font-medium text-zinc-700">
                <input
                  checked={form.isActive}
                  className="h-4 w-4 accent-zinc-950"
                  type="checkbox"
                  onChange={(event) => setForm((current) => ({ ...current, isActive: event.target.checked }))}
                />
                Active
              </label>
              <button
                className="min-h-10 rounded-md bg-zinc-950 px-3 py-2 text-sm font-semibold text-white disabled:cursor-not-allowed disabled:bg-zinc-300"
                disabled={!canCreate}
                type="submit"
              >
                {isCreating ? "Creating" : "Create combo"}
              </button>
            </div>
          </div>
        </form>

        {mutationError ? (
          <ErrorState title={mutationError.authExpired ? "Session expired" : "Could not change combo"} message={mutationError.message} />
        ) : null}

        {renderCombosState(combosState, loadCombos, handleDelete, deletingID)}
      </div>
    </Panel>
  );
}

function renderCombosState(
  state: AsyncState<ComboResponse[]>,
  onRetry: () => void,
  onDelete: (combo: ComboResponse) => void,
  deletingID: string | null
) {
  switch (state.status) {
    case "idle":
    case "loading":
      return <LoadingState label="Loading combos" />;
    case "empty":
      return (
        <EmptyState
          title="No combo routes configured"
          description="Create a combo to expose a combo/<name> fallback chain to the proxy."
        />
      );
    case "error":
      return <ErrorState title="Could not load combos" message={state.error.message} onRetry={onRetry} />;
    case "auth-expired":
      return <ErrorState title="Session expired" message={state.error.message} onRetry={onRetry} />;
    case "success":
      return <CombosTable combos={state.data} deletingID={deletingID} onDelete={onDelete} />;
  }
}

function CombosTable({
  combos,
  deletingID,
  onDelete
}: {
  combos: ComboResponse[];
  deletingID: string | null;
  onDelete: (combo: ComboResponse) => void;
}) {
  return (
    <div className="overflow-x-auto rounded-md border border-zinc-200">
      <table aria-label="Combo routes" className="min-w-[680px] w-full text-left text-sm">
        <thead className="bg-zinc-50 text-xs uppercase text-zinc-500">
          <tr>
            <th className="px-4 py-3 font-semibold">Name</th>
            <th className="px-4 py-3 font-semibold">Steps</th>
            <th className="px-4 py-3 font-semibold">Status</th>
            <th className="px-4 py-3 font-semibold">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-zinc-200">
          {combos.map((combo) => (
            <tr key={combo.ID}>
              <td className="px-4 py-3 font-semibold text-zinc-950">{combo.Name}</td>
              <td className="px-4 py-3 text-zinc-600">
                <div className="flex flex-wrap gap-2">
                  {combo.Steps.map((step) => (
                    <span key={`${step.provider}/${step.model}`} className="rounded-md bg-zinc-50 px-2 py-1 font-mono text-xs">
                      {step.provider} / {step.model}
                    </span>
                  ))}
                </div>
              </td>
              <td className="px-4 py-3">
                <StatusPill tone={combo.IsActive ? "good" : "neutral"}>{combo.IsActive ? "active" : "inactive"}</StatusPill>
              </td>
              <td className="px-4 py-3">
                <button
                  aria-label={`Delete ${combo.Name}`}
                  className="rounded-md border border-zinc-200 px-3 py-2 text-sm font-semibold text-zinc-700 disabled:cursor-not-allowed disabled:text-zinc-400"
                  disabled={deletingID === combo.ID}
                  type="button"
                  onClick={() => onDelete(combo)}
                >
                  {deletingID === combo.ID ? "Deleting" : "Delete"}
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function toApiError(error: unknown): ApiError {
  if (error instanceof ApiError) {
    return error;
  }
  return new ApiError(0, error instanceof Error ? error.message : "Unknown API error", error);
}
