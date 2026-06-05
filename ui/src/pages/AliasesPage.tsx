import { useCallback, useEffect, useState, type FormEvent } from "react";
import {
  ApiError,
  asyncError,
  asyncSuccess,
  createAlias,
  deleteAlias,
  listAliases,
  updateAlias,
  type AsyncState,
  type ModelAliasResponse
} from "../api";
import { EmptyState, ErrorState, LoadingState, Panel } from "../components/Primitives";

type AliasForm = {
  alias: string;
  provider: string;
  model: string;
};

const emptyForm: AliasForm = { alias: "", model: "", provider: "" };

export function AliasesPage() {
  const [state, setState] = useState<AsyncState<ModelAliasResponse[]>>({ status: "loading" });
  const [form, setForm] = useState<AliasForm>(emptyForm);
  const [editingAlias, setEditingAlias] = useState("");
  const [mutationError, setMutationError] = useState<ApiError | null>(null);
  const [busyAlias, setBusyAlias] = useState("");

  const loadAliases = useCallback(async () => {
    setState({ status: "loading" });
    try {
      setState(asyncSuccess(await listAliases()));
    } catch (error) {
      setState(asyncError(toApiError(error)));
    }
  }, []);

  useEffect(() => {
    void loadAliases();
  }, [loadAliases]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setMutationError(null);
    const alias = form.alias.trim();
    const provider = form.provider.trim();
    const model = form.model.trim();
    if (!alias || !provider || !model) {
      return;
    }
    const targetAlias = editingAlias || alias;
    setBusyAlias(targetAlias);
    try {
      if (editingAlias) {
        await updateAlias(editingAlias, provider, model);
      } else {
        await createAlias(alias, provider, model);
      }
      setForm(emptyForm);
      setEditingAlias("");
      await loadAliases();
    } catch (error) {
      setMutationError(toApiError(error));
    } finally {
      setBusyAlias("");
    }
  }

  function handleEdit(alias: ModelAliasResponse) {
    setMutationError(null);
    setEditingAlias(alias.Alias);
    setForm({ alias: alias.Alias, provider: alias.Provider, model: alias.Model });
  }

  function handleCancelEdit() {
    setMutationError(null);
    setEditingAlias("");
    setForm(emptyForm);
  }

  async function handleDelete(alias: ModelAliasResponse) {
    if (!window.confirm(`Delete alias ${alias.Alias}?`)) {
      return;
    }

    setMutationError(null);
    setBusyAlias(alias.Alias);
    try {
      await deleteAlias(alias.Alias);
      await loadAliases();
    } catch (error) {
      setMutationError(toApiError(error));
    } finally {
      setBusyAlias("");
    }
  }

  const canSubmit = form.alias.trim() !== "" && form.provider.trim() !== "" && form.model.trim() !== "" && busyAlias === "";

  return (
    <Panel title="Model aliases" description="Named model routes that resolve to provider and upstream model pairs.">
      <div className="space-y-5">
        <form className="rounded-md border border-zinc-200 p-4" onSubmit={handleSubmit}>
          <div className="grid gap-3 lg:grid-cols-[1fr_1fr_1.3fr_auto]">
            <TextField label="Alias" value={form.alias} disabled={editingAlias !== ""} onChange={(alias) => setForm((current) => ({ ...current, alias }))} />
            <TextField label="Provider" value={form.provider} onChange={(provider) => setForm((current) => ({ ...current, provider }))} />
            <TextField label="Model" value={form.model} onChange={(model) => setForm((current) => ({ ...current, model }))} />
            <div className="flex items-end gap-2">
              <button className="min-h-10 rounded-md bg-zinc-950 px-3 py-2 text-sm font-semibold text-white disabled:cursor-not-allowed disabled:bg-zinc-300" disabled={!canSubmit} type="submit">
                {editingAlias ? "Update alias" : "Create alias"}
              </button>
              {editingAlias ? (
                <button className="min-h-10 rounded-md border border-zinc-200 px-3 py-2 text-sm font-semibold text-zinc-700" type="button" onClick={handleCancelEdit}>
                  Cancel
                </button>
              ) : null}
            </div>
          </div>
        </form>

        {mutationError ? <ErrorState title={mutationError.authExpired ? "Session expired" : "Could not change alias"} message={mutationError.message} /> : null}
        {renderAliases(state, loadAliases, handleDelete, handleEdit, busyAlias)}
      </div>
    </Panel>
  );
}

function renderAliases(
  state: AsyncState<ModelAliasResponse[]>,
  onRetry: () => void,
  onDelete: (alias: ModelAliasResponse) => void,
  onEdit: (alias: ModelAliasResponse) => void,
  busyAlias: string
) {
  switch (state.status) {
    case "idle":
    case "loading":
      return <LoadingState label="Loading aliases" />;
    case "empty":
      return <EmptyState title="No model aliases" description="Create an alias to expose a stable model name for routing." />;
    case "error":
      return <ErrorState title="Could not load aliases" message={state.error.message} onRetry={onRetry} />;
    case "auth-expired":
      return <ErrorState title="Session expired" message={state.error.message} onRetry={onRetry} />;
    case "success":
      return <AliasesTable aliases={state.data} busyAlias={busyAlias} onDelete={onDelete} onEdit={onEdit} />;
  }
}

function AliasesTable({
  aliases,
  busyAlias,
  onDelete,
  onEdit
}: {
  aliases: ModelAliasResponse[];
  busyAlias: string;
  onDelete: (alias: ModelAliasResponse) => void;
  onEdit: (alias: ModelAliasResponse) => void;
}) {
  return (
    <div className="overflow-x-auto rounded-md border border-zinc-200">
      <table aria-label="Model aliases" className="min-w-[620px] w-full text-left text-sm">
        <thead className="bg-zinc-50 text-xs uppercase text-zinc-500">
          <tr>
            <th className="px-4 py-3 font-semibold">Alias</th>
            <th className="px-4 py-3 font-semibold">Provider</th>
            <th className="px-4 py-3 font-semibold">Model</th>
            <th className="px-4 py-3 font-semibold">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-zinc-200">
          {aliases.map((alias) => (
            <tr key={alias.Alias}>
              <td className="px-4 py-3 font-semibold text-zinc-950">{alias.Alias}</td>
              <td className="px-4 py-3 text-zinc-600">{alias.Provider}</td>
              <td className="px-4 py-3 font-mono text-xs text-zinc-600">{alias.Model}</td>
              <td className="px-4 py-3">
                <div className="flex items-center gap-2">
                  <button className="rounded-md border border-zinc-200 px-3 py-2 text-sm font-semibold text-zinc-700 disabled:cursor-not-allowed disabled:text-zinc-400" disabled={busyAlias === alias.Alias} type="button" aria-label={`Edit ${alias.Alias}`} onClick={() => onEdit(alias)}>
                    Edit
                  </button>
                  <button className="rounded-md border border-zinc-200 px-3 py-2 text-sm font-semibold text-zinc-700 disabled:cursor-not-allowed disabled:text-zinc-400" disabled={busyAlias === alias.Alias} type="button" aria-label={`Delete ${alias.Alias}`} onClick={() => onDelete(alias)}>
                    Delete
                  </button>
                </div>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function TextField({ disabled = false, label, onChange, value }: { disabled?: boolean; label: string; onChange: (value: string) => void; value: string }) {
  return (
    <label className="text-sm font-medium text-zinc-700">
      {label}
      <input className="mt-1 w-full rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-950 disabled:bg-zinc-50 disabled:text-zinc-500" disabled={disabled} value={value} onChange={(event) => onChange(event.target.value)} />
    </label>
  );
}

function toApiError(error: unknown): ApiError {
  if (error instanceof ApiError) {
    return error;
  }
  return new ApiError(0, error instanceof Error ? error.message : "Unknown API error", error);
}
