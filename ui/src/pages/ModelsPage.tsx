import { useCallback, useEffect, useMemo, useState } from "react";
import {
  ApiError,
  asyncError,
  listProviderModels,
  listProviders,
  type AsyncState,
  type ProviderMatrixEntry,
  type ProviderModel
} from "../api";
import { EmptyState, ErrorState, LoadingState, Panel, StatusPill } from "../components/Primitives";

type ModelsData = {
  providers: ProviderMatrixEntry[];
  models: ProviderModel[];
  selectedProvider: string;
};

export function ModelsPage() {
  const [providers, setProviders] = useState<ProviderMatrixEntry[]>([]);
  const [selectedProvider, setSelectedProvider] = useState("");
  const [state, setState] = useState<AsyncState<ModelsData>>({ status: "loading" });

  const modelProviders = useMemo(
    () => providers.filter((provider) => provider.model_catalog || provider.list_models || provider.public_inference),
    [providers]
  );

  const loadModels = useCallback(
    async (providerID?: string) => {
      setState({ status: "loading" });
      try {
        const loadedProviders = providers.length > 0 ? providers : await listProviders();
        const candidates = loadedProviders.filter((provider) => provider.model_catalog || provider.list_models || provider.public_inference);
        const nextProvider = providerID || selectedProvider || candidates[0]?.id || "";
        setProviders(loadedProviders);
        setSelectedProvider(nextProvider);
        if (!nextProvider) {
          setState({ status: "empty", data: { models: [], providers: loadedProviders, selectedProvider: "" } });
          return;
        }
        const models = await listProviderModels(nextProvider);
        const data = { models, providers: loadedProviders, selectedProvider: nextProvider };
        setState(models.length === 0 ? { status: "empty", data } : { status: "success", data });
      } catch (error) {
        setState(asyncError<ModelsData>(toApiError(error)));
      }
    },
    [providers, selectedProvider]
  );

  useEffect(() => {
    void loadModels();
  }, []);

  async function handleProviderChange(providerID: string) {
    setSelectedProvider(providerID);
    await loadModels(providerID);
  }

  return (
    <Panel title="Provider models" description="Catalog and upstream model rows exposed by provider management APIs.">
      <div className="space-y-5">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-end">
          <label className="text-sm font-medium text-zinc-700">
            Provider
            <select
              className="mt-1 h-10 w-full rounded-md border border-zinc-300 bg-white px-3 text-sm text-zinc-950 sm:w-64"
              disabled={modelProviders.length === 0}
              onChange={(event) => void handleProviderChange(event.target.value)}
              value={selectedProvider}
            >
              {modelProviders.map((provider) => (
                <option key={provider.id} value={provider.id}>
                  {provider.id}
                </option>
              ))}
            </select>
          </label>
          <button
            className="h-10 rounded-md border border-zinc-300 px-4 text-sm font-semibold text-zinc-700 disabled:cursor-not-allowed disabled:text-zinc-400"
            disabled={!selectedProvider || state.status === "loading"}
            onClick={() => void loadModels(selectedProvider)}
            type="button"
          >
            Refresh models
          </button>
        </div>
        {renderModelsState(state)}
      </div>
    </Panel>
  );
}

function renderModelsState(state: AsyncState<ModelsData>) {
  switch (state.status) {
    case "idle":
    case "loading":
      return <LoadingState label="Loading provider models" />;
    case "auth-expired":
      return <ErrorState title="Session expired" message={state.error.message} />;
    case "error":
      return <ErrorState title="Could not load models" message={state.error.message} />;
    case "empty":
      return (
        <EmptyState
          title={state.data.selectedProvider ? "No models returned" : "No model-capable providers"}
          description={
            state.data.selectedProvider
              ? `${state.data.selectedProvider} did not return catalog or upstream model rows.`
              : "The provider matrix did not return providers with model catalog or listing support."
          }
        />
      );
    case "success":
      return <ModelsTable models={state.data.models} provider={state.data.selectedProvider} />;
  }
}

function ModelsTable({ models, provider }: { models: ProviderModel[]; provider: string }) {
  return (
    <div className="overflow-x-auto rounded-md border border-zinc-200">
      <table aria-label="Provider models" className="min-w-[680px] w-full text-left text-sm">
        <thead className="bg-zinc-50 text-xs uppercase text-zinc-500">
          <tr>
            <th className="px-4 py-3 font-semibold">Model</th>
            <th className="px-4 py-3 font-semibold">Provider</th>
            <th className="px-4 py-3 font-semibold">Owner</th>
            <th className="px-4 py-3 font-semibold">Object</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-zinc-200">
          {models.map((model) => (
            <tr key={model.id}>
              <td className="px-4 py-3 font-mono text-xs text-zinc-800">{model.id}</td>
              <td className="px-4 py-3">
                <StatusPill tone="neutral">{provider}</StatusPill>
              </td>
              <td className="px-4 py-3 text-zinc-600">{model.owned_by || "-"}</td>
              <td className="px-4 py-3 text-zinc-600">{model.object || "model"}</td>
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
