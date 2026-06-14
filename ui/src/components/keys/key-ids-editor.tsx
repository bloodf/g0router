import * as React from "react";
import { apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Select } from "@/components/ui/select";

// EditorProviderConfig is the real VK provider_configs[] entry
// (schemas.ProviderConfig, governance.go:13-19): {provider, allowed_models[],
// key_ids[], weight?}. key_ids is the pinning field (plan §1.6).
export interface EditorProviderConfig {
  provider: string;
  allowed_models: string[];
  key_ids: string[];
  weight?: number;
}

export function emptyProviderConfig(provider = ""): EditorProviderConfig {
  return { provider, allowed_models: [], key_ids: [] };
}

interface ModelOption {
  id: string;
}

interface ConnectionOption {
  id: string;
  name?: string;
}

export interface KeyIdsEditorProps {
  value: EditorProviderConfig[];
  onChange: (next: EditorProviderConfig[]) => void;
  // providerOptions seeds the provider dropdown (and is used by unit tests). When
  // omitted the editor fetches the provider catalog.
  providerOptions?: string[];
}

// KeyIdsEditor (plan §1.6) lets an operator pin connection key_ids and allowed
// models per provider into a virtual key's provider_configs[]. Allowed models are
// sourced from GET /api/providers/{id}/models (or /api/models); pinnable key_ids
// from GET /api/providers/{id}/connections (both w6-e SHIPPED). It emits the real
// VK provider_configs shape.
export function KeyIdsEditor({ value, onChange, providerOptions }: KeyIdsEditorProps) {
  const [providers, setProviders] = React.useState<string[]>(providerOptions ?? []);
  const [modelsByProvider, setModelsByProvider] = React.useState<Record<string, string[]>>({});
  const [keyIdsByProvider, setKeyIdsByProvider] = React.useState<Record<string, ConnectionOption[]>>({});

  React.useEffect(() => {
    if (providerOptions !== undefined) {
      setProviders(providerOptions);
      return;
    }
    apiFetch<Array<{ id: string }>>("/api/providers/catalog")
      .then((list) => setProviders((list ?? []).map((p) => p.id)))
      .catch(() => setProviders([]));
  }, [providerOptions]);

  const loadProviderData = React.useCallback((provider: string) => {
    if (!provider) return;
    apiFetch<ModelOption[]>(`/api/providers/${provider}/models`)
      .then((list) =>
        setModelsByProvider((prev) => ({ ...prev, [provider]: (list ?? []).map((m) => m.id) }))
      )
      .catch(() => setModelsByProvider((prev) => ({ ...prev, [provider]: [] })));
    apiFetch<ConnectionOption[]>(`/api/providers/${provider}/connections`)
      .then((list) => setKeyIdsByProvider((prev) => ({ ...prev, [provider]: list ?? [] })))
      .catch(() => setKeyIdsByProvider((prev) => ({ ...prev, [provider]: [] })));
  }, []);

  React.useEffect(() => {
    if (providerOptions !== undefined) return;
    for (const config of value) {
      if (config.provider) loadProviderData(config.provider);
    }
  }, [value, providerOptions, loadProviderData]);

  function updateConfig(index: number, patch: Partial<EditorProviderConfig>) {
    onChange(value.map((config, i) => (i === index ? { ...config, ...patch } : config)));
  }

  function setProvider(index: number, provider: string) {
    updateConfig(index, { provider, allowed_models: [], key_ids: [] });
    if (providerOptions === undefined) loadProviderData(provider);
  }

  function toggleInArray(list: string[], item: string): string[] {
    return list.includes(item) ? list.filter((x) => x !== item) : [...list, item];
  }

  function addConfig() {
    onChange([...value, emptyProviderConfig(providers[0] ?? "")]);
  }

  function removeConfig(index: number) {
    onChange(value.filter((_, i) => i !== index));
  }

  const providerSelectOptions = providers.map((p) => ({ value: p, label: p }));

  return (
    <div data-testid="key-ids-editor" className="flex flex-col gap-3">
      {value.map((config, index) => {
        const models = modelsByProvider[config.provider] ?? config.allowed_models;
        const conns = keyIdsByProvider[config.provider] ??
          config.key_ids.map((id) => ({ id }));
        return (
          <div
            key={index}
            data-testid="provider-config-row"
            className="flex flex-col gap-2 rounded-md border border-border p-3"
          >
            <Select
              data-testid="vk-provider-select"
              label="Provider"
              options={[{ value: "", label: "Select a provider" }, ...providerSelectOptions]}
              value={config.provider}
              onChange={(event) => setProvider(index, event.target.value)}
            />

            <div className="flex flex-col gap-1">
              <span className="text-sm font-medium text-foreground">Allowed models</span>
              <div className="flex flex-wrap gap-1">
                {models.length === 0 ? (
                  <span className="text-xs text-muted-foreground">No models available</span>
                ) : (
                  models.map((model) => {
                    const selected = config.allowed_models.includes(model);
                    return (
                      <button
                        key={model}
                        type="button"
                        data-testid="vk-model-option"
                        data-selected={selected}
                        onClick={() =>
                          updateConfig(index, {
                            allowed_models: toggleInArray(config.allowed_models, model),
                          })
                        }
                        className={
                          "rounded border px-2 py-0.5 text-xs " +
                          (selected
                            ? "border-primary bg-primary/15 text-primary"
                            : "border-border text-muted-foreground")
                        }
                      >
                        {model}
                      </button>
                    );
                  })
                )}
              </div>
            </div>

            <div className="flex flex-col gap-1">
              <span className="text-sm font-medium text-foreground">Pinned key IDs</span>
              <div className="flex flex-wrap gap-1">
                {conns.length === 0 ? (
                  <span className="text-xs text-muted-foreground">No connections available</span>
                ) : (
                  conns.map((conn) => {
                    const selected = config.key_ids.includes(conn.id);
                    return (
                      <button
                        key={conn.id}
                        type="button"
                        data-testid="vk-keyid-option"
                        data-selected={selected}
                        onClick={() =>
                          updateConfig(index, {
                            key_ids: toggleInArray(config.key_ids, conn.id),
                          })
                        }
                        className={
                          "rounded border px-2 py-0.5 text-xs " +
                          (selected
                            ? "border-primary bg-primary/15 text-primary"
                            : "border-border text-muted-foreground")
                        }
                      >
                        {conn.name ? `${conn.name} (${conn.id})` : conn.id}
                      </button>
                    );
                  })
                )}
              </div>
            </div>

            <div className="flex justify-end">
              <Button variant="ghost" size="sm" onClick={() => removeConfig(index)}>
                Remove
              </Button>
            </div>
          </div>
        );
      })}
      <Button data-testid="add-provider-config" variant="outline" size="sm" onClick={addConfig}>
        Add provider
      </Button>
    </div>
  );
}
