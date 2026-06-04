import { useCallback, useEffect, useState, type FormEvent } from "react";
import {
  ApiError,
  asyncError,
  asyncSuccess,
  createPricingOverride,
  deletePricingOverride,
  listPricingOverrides,
  type AsyncState,
  type PricingOverrideResponse
} from "../api";
import { EmptyState, ErrorState, LoadingState, Panel } from "../components/Primitives";

type PricingForm = {
  provider: string;
  model: string;
  inputCost: string;
  outputCost: string;
};

const emptyForm: PricingForm = { inputCost: "", model: "", outputCost: "", provider: "" };

export function PricingPage() {
  const [state, setState] = useState<AsyncState<PricingOverrideResponse[]>>({ status: "loading" });
  const [form, setForm] = useState<PricingForm>(emptyForm);
  const [mutationError, setMutationError] = useState<ApiError | null>(null);
  const [busyKey, setBusyKey] = useState("");

  const loadPricing = useCallback(async () => {
    setState({ status: "loading" });
    try {
      setState(asyncSuccess(await listPricingOverrides()));
    } catch (error) {
      setState(asyncError(toApiError(error)));
    }
  }, []);

  useEffect(() => {
    void loadPricing();
  }, [loadPricing]);

  async function handleCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setMutationError(null);
    const provider = form.provider.trim();
    const model = form.model.trim();
    const inputCost = Number(form.inputCost);
    const outputCost = Number(form.outputCost);
    if (!provider || !model || Number.isNaN(inputCost) || Number.isNaN(outputCost)) {
      return;
    }
    setBusyKey(rowKey(provider, model));
    try {
      await createPricingOverride(provider, model, inputCost, outputCost);
      setForm(emptyForm);
      await loadPricing();
    } catch (error) {
      setMutationError(toApiError(error));
    } finally {
      setBusyKey("");
    }
  }

  async function handleDelete(override: PricingOverrideResponse) {
    setMutationError(null);
    setBusyKey(rowKey(override.Provider, override.Model));
    try {
      await deletePricingOverride(override.Provider, override.Model);
      await loadPricing();
    } catch (error) {
      setMutationError(toApiError(error));
    } finally {
      setBusyKey("");
    }
  }

  const canCreate = form.provider.trim() !== "" && form.model.trim() !== "" && form.inputCost.trim() !== "" && form.outputCost.trim() !== "" && busyKey === "";

  return (
    <Panel title="Pricing overrides" description="Per-provider model costs used by usage and cost accounting.">
      <div className="space-y-5">
        <form className="rounded-md border border-zinc-200 p-4" onSubmit={handleCreate}>
          <div className="grid gap-3 xl:grid-cols-[1fr_1.2fr_1fr_1fr_auto]">
            <TextField label="Provider" value={form.provider} onChange={(provider) => setForm((current) => ({ ...current, provider }))} />
            <TextField label="Model" value={form.model} onChange={(model) => setForm((current) => ({ ...current, model }))} />
            <TextField label="Input cost per token" value={form.inputCost} onChange={(inputCost) => setForm((current) => ({ ...current, inputCost }))} />
            <TextField label="Output cost per token" value={form.outputCost} onChange={(outputCost) => setForm((current) => ({ ...current, outputCost }))} />
            <div className="flex items-end">
              <button className="min-h-10 rounded-md bg-zinc-950 px-3 py-2 text-sm font-semibold text-white disabled:cursor-not-allowed disabled:bg-zinc-300" disabled={!canCreate} type="submit">
                Create override
              </button>
            </div>
          </div>
        </form>

        {mutationError ? <ErrorState title={mutationError.authExpired ? "Session expired" : "Could not change pricing"} message={mutationError.message} /> : null}
        {renderPricing(state, loadPricing, handleDelete, busyKey)}
      </div>
    </Panel>
  );
}

function renderPricing(
  state: AsyncState<PricingOverrideResponse[]>,
  onRetry: () => void,
  onDelete: (override: PricingOverrideResponse) => void,
  busyKey: string
) {
  switch (state.status) {
    case "idle":
    case "loading":
      return <LoadingState label="Loading pricing overrides" />;
    case "empty":
      return <EmptyState title="No pricing overrides" description="Catalog defaults are used until a provider/model override is saved." />;
    case "error":
      return <ErrorState title="Could not load pricing" message={state.error.message} onRetry={onRetry} />;
    case "auth-expired":
      return <ErrorState title="Session expired" message={state.error.message} onRetry={onRetry} />;
    case "success":
      return <PricingTable overrides={state.data} busyKey={busyKey} onDelete={onDelete} />;
  }
}

function PricingTable({ overrides, busyKey, onDelete }: { overrides: PricingOverrideResponse[]; busyKey: string; onDelete: (override: PricingOverrideResponse) => void }) {
  return (
    <div className="overflow-x-auto rounded-md border border-zinc-200">
      <table aria-label="Pricing overrides" className="min-w-[720px] w-full text-left text-sm">
        <thead className="bg-zinc-50 text-xs uppercase text-zinc-500">
          <tr>
            <th className="px-4 py-3 font-semibold">Provider</th>
            <th className="px-4 py-3 font-semibold">Model</th>
            <th className="px-4 py-3 font-semibold">Input</th>
            <th className="px-4 py-3 font-semibold">Output</th>
            <th className="px-4 py-3 font-semibold">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-zinc-200">
          {overrides.map((override) => {
            const key = rowKey(override.Provider, override.Model);
            return (
              <tr key={key}>
                <td className="px-4 py-3 font-semibold text-zinc-950">{override.Provider}</td>
                <td className="px-4 py-3 font-mono text-xs text-zinc-600">{override.Model}</td>
                <td className="px-4 py-3 text-zinc-600">{formatCost(override.InputCostPerToken)}</td>
                <td className="px-4 py-3 text-zinc-600">{formatCost(override.OutputCostPerToken)}</td>
                <td className="px-4 py-3">
                  <button className="rounded-md border border-zinc-200 px-3 py-2 text-sm font-semibold text-zinc-700 disabled:cursor-not-allowed disabled:text-zinc-400" disabled={busyKey === key} type="button" aria-label={`Delete ${override.Provider} ${override.Model}`} onClick={() => onDelete(override)}>
                    Delete
                  </button>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

function TextField({ label, onChange, value }: { label: string; onChange: (value: string) => void; value: string }) {
  return (
    <label className="text-sm font-medium text-zinc-700">
      {label}
      <input className="mt-1 w-full rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-950" value={value} onChange={(event) => onChange(event.target.value)} />
    </label>
  );
}

function formatCost(value: number) {
  return value.toFixed(6).replace(/0+$/, "").replace(/\.$/, ".0");
}

function rowKey(provider: string, model: string) {
  return `${provider}/${model}`;
}

function toApiError(error: unknown): ApiError {
  if (error instanceof ApiError) {
    return error;
  }
  return new ApiError(0, error instanceof Error ? error.message : "Unknown API error", error);
}
