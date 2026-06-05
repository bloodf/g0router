import { useCallback, useEffect, useState, type FormEvent } from "react";
import { ApiError, asyncError, asyncSuccess, createCombo, deleteCombo, listCombos, updateCombo, type AsyncState, type ComboResponse, type ComboStepResponse, type ComboStrategy } from "../api";
import { EmptyState, ErrorState, LoadingState, Panel, StatusPill } from "../components/Primitives";

type ComboForm = {
  isActive: boolean;
  name: string;
  steps: ComboStepResponse[];
  strategy: ComboStrategy;
};

const emptyStep: ComboStepResponse = { provider: "", model: "" };

const emptyForm: ComboForm = {
  isActive: true,
  name: "",
  steps: [{ ...emptyStep }],
  strategy: "fallback"
};

const strategyOptions: { value: ComboStrategy; label: string; hint: string }[] = [
  { value: "fallback", label: "Fallback", hint: "Try steps in order until one succeeds (default)" },
  { value: "round_robin", label: "Round-robin", hint: "Rotate the starting step across calls to spread load" },
  { value: "least_used", label: "Least-used", hint: "Start with the least-used step (by call count)" },
  { value: "auto", label: "Auto", hint: "Pick the best step per request (capable model for tools/large context, cheaper otherwise)" }
];

export function CombosPage() {
  const [combosState, setCombosState] = useState<AsyncState<ComboResponse[]>>({ status: "loading" });
  const [form, setForm] = useState<ComboForm>(emptyForm);
  const [isSaving, setIsSaving] = useState(false);
  const [editingComboID, setEditingComboID] = useState("");
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

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setMutationError(null);
    setIsSaving(true);
    const name = form.name.trim();
    const steps = form.steps.map((s) => ({ provider: s.provider.trim(), model: s.model.trim() }));
    try {
      if (editingComboID) {
        await updateCombo(editingComboID, name, steps, form.isActive, form.strategy);
      } else {
        await createCombo(name, steps, form.isActive, form.strategy);
      }
      setForm(emptyForm);
      setEditingComboID("");
      await loadCombos();
    } catch (error) {
      setMutationError(toApiError(error));
    } finally {
      setIsSaving(false);
    }
  }

  function handleEdit(combo: ComboResponse) {
    setMutationError(null);
    setEditingComboID(combo.ID);
    setForm({
      isActive: combo.IsActive,
      name: combo.Name,
      steps: combo.Steps.length > 0 ? combo.Steps.map((s) => ({ ...s })) : [{ ...emptyStep }],
      strategy: combo.Strategy ?? "fallback"
    });
  }

  function handleStepChange(index: number, field: keyof ComboStepResponse, value: string) {
    setForm((current) => ({
      ...current,
      steps: current.steps.map((s, i) => (i === index ? { ...s, [field]: value } : s))
    }));
  }

  function handleAddStep() {
    setForm((current) => ({ ...current, steps: [...current.steps, { ...emptyStep }] }));
  }

  function handleRemoveStep(index: number) {
    setForm((current) => ({
      ...current,
      steps: current.steps.length > 1 ? current.steps.filter((_, i) => i !== index) : current.steps
    }));
  }

  function handleMoveStep(index: number, direction: -1 | 1) {
    const next = index + direction;
    setForm((current) => {
      if (next < 0 || next >= current.steps.length) return current;
      const steps = current.steps.map((s) => ({ ...s }));
      [steps[index], steps[next]] = [steps[next], steps[index]];
      return { ...current, steps };
    });
  }

  function handleCancelEdit() {
    setMutationError(null);
    setEditingComboID("");
    setForm(emptyForm);
  }

  async function handleDelete(combo: ComboResponse) {
    if (!window.confirm(`Delete combo ${combo.Name}?`)) {
      return;
    }

    setMutationError(null);
    setDeletingID(combo.ID);
    try {
      await deleteCombo(combo.ID);
      await loadCombos();
    } catch (error) {
      setMutationError(toApiError(error));
    } finally {
      setDeletingID(null);
    }
  }

  const canSave =
    form.name.trim() !== "" &&
    form.steps.length > 0 &&
    form.steps.every((s) => s.provider.trim() !== "" && s.model.trim() !== "") &&
    !isSaving;

  return (
    <Panel title="Combo routing" description="Reusable routing chains for fallback, round-robin, and account selection.">
      <div className="space-y-5">
        <form className="rounded-md border border-zinc-200 p-4" onSubmit={handleSubmit}>
          <div className="mb-3 space-y-3">
            <div className="grid gap-3 lg:grid-cols-2">
              <label className="text-sm font-medium text-zinc-700">
                Combo name
                <input
                  aria-label="Combo name"
                  className="mt-1 w-full rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-950"
                  value={form.name}
                  onChange={(event) => setForm((current) => ({ ...current, name: event.target.value }))}
                />
              </label>
              <label className="text-sm font-medium text-zinc-700">
                Strategy
                <select
                  aria-label="Strategy"
                  className="mt-1 w-full rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-950"
                  value={form.strategy}
                  onChange={(event) => setForm((current) => ({ ...current, strategy: event.target.value as ComboStrategy }))}
                >
                  {strategyOptions.map((opt) => (
                    <option key={opt.value} value={opt.value}>{opt.label}</option>
                  ))}
                </select>
                <p className="mt-1 text-xs text-zinc-500">
                  {strategyOptions.find((opt) => opt.value === form.strategy)?.hint}
                </p>
              </label>
            </div>
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
                disabled={!canSave}
                type="submit"
              >
                {isSaving ? "Saving" : editingComboID ? "Update combo" : "Create combo"}
              </button>
              {editingComboID ? (
                <button className="min-h-10 rounded-md border border-zinc-200 px-3 py-2 text-sm font-semibold text-zinc-700" type="button" onClick={handleCancelEdit}>
                  Cancel
                </button>
              ) : null}
            </div>
          </div>
          <div className="space-y-2">
            <p className="text-xs font-semibold uppercase text-zinc-500">Steps (ordered)</p>
            {form.steps.map((step, index) => (
              <div key={index} className="grid gap-2 lg:grid-cols-[1fr_1.3fr_auto]">
                <label className="text-sm font-medium text-zinc-700">
                  {index === 0 ? "Provider" : ""}
                  <input
                    aria-label={`Step ${index + 1} provider`}
                    className="mt-1 w-full rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-950"
                    placeholder="provider"
                    value={step.provider}
                    onChange={(event) => handleStepChange(index, "provider", event.target.value)}
                  />
                </label>
                <label className="text-sm font-medium text-zinc-700">
                  {index === 0 ? "Model" : ""}
                  <input
                    aria-label={`Step ${index + 1} model`}
                    className="mt-1 w-full rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-950"
                    placeholder="model"
                    value={step.model}
                    onChange={(event) => handleStepChange(index, "model", event.target.value)}
                  />
                </label>
                <div className="flex items-end gap-1">
                  <button
                    aria-label={`Move step ${index + 1} up`}
                    className="min-h-10 rounded-md border border-zinc-200 px-2 py-2 text-sm text-zinc-700 disabled:cursor-not-allowed disabled:text-zinc-300"
                    disabled={index === 0}
                    type="button"
                    onClick={() => handleMoveStep(index, -1)}
                  >
                    ↑
                  </button>
                  <button
                    aria-label={`Move step ${index + 1} down`}
                    className="min-h-10 rounded-md border border-zinc-200 px-2 py-2 text-sm text-zinc-700 disabled:cursor-not-allowed disabled:text-zinc-300"
                    disabled={index === form.steps.length - 1}
                    type="button"
                    onClick={() => handleMoveStep(index, 1)}
                  >
                    ↓
                  </button>
                  <button
                    aria-label={`Remove step ${index + 1}`}
                    className="min-h-10 rounded-md border border-zinc-200 px-2 py-2 text-sm text-zinc-700 disabled:cursor-not-allowed disabled:text-zinc-300"
                    disabled={form.steps.length === 1}
                    type="button"
                    onClick={() => handleRemoveStep(index)}
                  >
                    ✕
                  </button>
                </div>
              </div>
            ))}
            <button
              className="mt-1 rounded-md border border-zinc-200 px-3 py-1.5 text-sm font-semibold text-zinc-700"
              type="button"
              onClick={handleAddStep}
            >
              + Add step
            </button>
          </div>
        </form>

        {mutationError ? (
          <ErrorState title={mutationError.authExpired ? "Session expired" : "Could not change combo"} message={mutationError.message} />
        ) : null}

        {renderCombosState(combosState, loadCombos, handleDelete, handleEdit, deletingID)}
      </div>
    </Panel>
  );
}

function renderCombosState(
  state: AsyncState<ComboResponse[]>,
  onRetry: () => void,
  onDelete: (combo: ComboResponse) => void,
  onEdit: (combo: ComboResponse) => void,
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
      return <CombosTable combos={state.data} deletingID={deletingID} onDelete={onDelete} onEdit={onEdit} />;
  }
}

function CombosTable({
  combos,
  deletingID,
  onDelete,
  onEdit
}: {
  combos: ComboResponse[];
  deletingID: string | null;
  onDelete: (combo: ComboResponse) => void;
  onEdit: (combo: ComboResponse) => void;
}) {
  return (
    <div className="overflow-x-auto rounded-md border border-zinc-200">
      <table aria-label="Combo routes" className="min-w-[680px] w-full text-left text-sm">
        <thead className="bg-zinc-50 text-xs uppercase text-zinc-500">
          <tr>
            <th className="px-4 py-3 font-semibold">Name</th>
            <th className="px-4 py-3 font-semibold">Steps</th>
            <th className="px-4 py-3 font-semibold">Strategy</th>
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
              <td className="px-4 py-3 text-zinc-600 text-sm">{combo.Strategy ?? "fallback"}</td>
              <td className="px-4 py-3">
                <StatusPill tone={combo.IsActive ? "good" : "neutral"}>{combo.IsActive ? "active" : "inactive"}</StatusPill>
              </td>
              <td className="px-4 py-3">
                <div className="flex items-center gap-2">
                  <button
                    aria-label={`Edit ${combo.Name}`}
                    className="rounded-md border border-zinc-200 px-3 py-2 text-sm font-semibold text-zinc-700 disabled:cursor-not-allowed disabled:text-zinc-400"
                    disabled={deletingID === combo.ID}
                    type="button"
                    onClick={() => onEdit(combo)}
                  >
                    Edit
                  </button>
                  <button
                    aria-label={`Delete ${combo.Name}`}
                    className="rounded-md border border-zinc-200 px-3 py-2 text-sm font-semibold text-zinc-700 disabled:cursor-not-allowed disabled:text-zinc-400"
                    disabled={deletingID === combo.ID}
                    type="button"
                    onClick={() => onDelete(combo)}
                  >
                    {deletingID === combo.ID ? "Deleting" : "Delete"}
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

function toApiError(error: unknown): ApiError {
  if (error instanceof ApiError) {
    return error;
  }
  return new ApiError(0, error instanceof Error ? error.message : "Unknown API error", error);
}
