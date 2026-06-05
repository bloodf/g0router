import { useCallback, useEffect, useState, type ChangeEvent, type FormEvent } from "react";
import { ApiError, asyncError, getSettings, updateSettings, type AsyncState, type SettingsResponse } from "../api";
import { EmptyState, ErrorState, LoadingState, Panel, StatusPill } from "../components/Primitives";

export function SettingsPage({ title = "Runtime settings", description = "Gateway defaults that affect proxy behavior and local control-plane access." }: { title?: string; description?: string } = {}) {
  const [loadState, setLoadState] = useState<AsyncState<SettingsResponse | null>>({ status: "loading" });
  const [form, setForm] = useState<SettingsResponse | null>(null);
  const [isSaving, setIsSaving] = useState(false);
  const [saveError, setSaveError] = useState<ApiError | null>(null);
  const [saved, setSaved] = useState(false);

  const loadSettings = useCallback(async () => {
    setLoadState({ status: "loading" });
    setSaveError(null);
    setSaved(false);
    try {
      const settings = await getSettings();
      if (!settings) {
        setForm(null);
        setLoadState({ status: "empty", data: null });
        return;
      }
      setForm(settings);
      setLoadState({ status: "success", data: settings });
    } catch (error) {
      setForm(null);
      setLoadState(asyncError<SettingsResponse | null>(toApiError(error)));
    }
  }, []);

  useEffect(() => {
    void loadSettings();
  }, [loadSettings]);

  async function handleSave(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!form) {
      return;
    }

    setIsSaving(true);
    setSaveError(null);
    setSaved(false);
    try {
      const savedSettings = await updateSettings(form);
      setForm(savedSettings);
      setLoadState({ status: "success", data: savedSettings });
      setSaved(true);
    } catch (error) {
      setSaveError(toApiError(error));
    } finally {
      setIsSaving(false);
    }
  }

  return (
    <Panel title={title} description={description}>
      {renderSettingsState(loadState, loadSettings, form, setForm, handleSave, isSaving, saveError, saved)}
    </Panel>
  );
}

function renderSettingsState(
  state: AsyncState<SettingsResponse | null>,
  onRetry: () => void,
  form: SettingsResponse | null,
  setForm: (settings: SettingsResponse) => void,
  onSave: (event: FormEvent<HTMLFormElement>) => void,
  isSaving: boolean,
  saveError: ApiError | null,
  saved: boolean
) {
  switch (state.status) {
    case "idle":
    case "loading":
      return <LoadingState label="Loading settings" />;
    case "empty":
      return <EmptyState title="No runtime settings returned" description="The settings endpoint responded without a body." />;
    case "error":
      return <ErrorState title="Could not load settings" message={state.error.message} onRetry={onRetry} />;
    case "auth-expired":
      return <ErrorState title="Session expired" message={state.error.message} onRetry={onRetry} />;
    case "success":
      if (!form) {
        return <EmptyState title="No runtime settings returned" description="The settings endpoint responded without a body." />;
      }
      return (
        <SettingsForm
          form={form}
          isSaving={isSaving}
          saveError={saveError}
          saved={saved}
          setForm={setForm}
          onSave={onSave}
        />
      );
  }
}

function SettingsForm({
  form,
  isSaving,
  saveError,
  saved,
  setForm,
  onSave
}: {
  form: SettingsResponse;
  isSaving: boolean;
  saveError: ApiError | null;
  saved: boolean;
  setForm: (settings: SettingsResponse) => void;
  onSave: (event: FormEvent<HTMLFormElement>) => void;
}) {
  return (
    <form className="space-y-5" onSubmit={onSave}>
      <div className="grid gap-4 xl:grid-cols-2">
        <div className="space-y-3">
          <ToggleRow
            checked={form.require_api_key}
            label="Require API key"
            onChange={(value) => setForm({ ...form, require_api_key: value })}
          />
          <ToggleRow checked={form.rtk_enabled} label="RTK enabled" onChange={(value) => setForm({ ...form, rtk_enabled: value })} />
          <ToggleRow
            checked={form.caveman_enabled}
            label="Caveman enabled"
            onChange={(value) => setForm({ ...form, caveman_enabled: value })}
          />
          <ToggleRow
            checked={form.enable_request_logs}
            label="Enable request logs"
            onChange={(value) => setForm({ ...form, enable_request_logs: value })}
          />
          <LogRetentionRow value={form.log_retention_days} onChange={(value) => setForm({ ...form, log_retention_days: value })} />
          <ConnectionSourcesRow
            value={form.allowed_sources ?? []}
            onChange={(value) => setForm({ ...form, allowed_sources: value })}
          />
        </div>

        <div className="space-y-3">
          <label className="block text-sm font-medium text-zinc-700">
            Caveman level
            <input
              className="mt-1 w-full rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-950"
              value={form.caveman_level}
              onChange={(event) => setForm({ ...form, caveman_level: event.target.value })}
            />
          </label>
          <label className="block text-sm font-medium text-zinc-700">
            Proxy URL
            <input
              className="mt-1 w-full rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-950"
              value={form.proxy_url}
              onChange={(event) => setForm({ ...form, proxy_url: event.target.value })}
            />
          </label>
          <label className="block text-sm font-medium text-zinc-700">
            Data directory
            <input
              className="mt-1 w-full rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-950"
              value={form.data_dir}
              onChange={(event) => setForm({ ...form, data_dir: event.target.value })}
            />
          </label>
        </div>
      </div>

      {saveError ? <ErrorState title={saveError.authExpired ? "Session expired" : "Could not save settings"} message={saveError.message} /> : null}

      <div className="flex items-center justify-between gap-3 rounded-md border border-zinc-200 px-4 py-3">
        <div className="flex items-center gap-2">
          <StatusPill tone={form.require_api_key ? "good" : "warn"}>{form.require_api_key ? "protected" : "open"}</StatusPill>
          {saved ? <span className="text-sm font-semibold text-emerald-700">Settings saved</span> : null}
        </div>
        <button
          className="rounded-md bg-zinc-950 px-3 py-2 text-sm font-semibold text-white disabled:cursor-not-allowed disabled:bg-zinc-300"
          disabled={isSaving}
          type="submit"
        >
          {isSaving ? "Saving" : "Save settings"}
        </button>
      </div>
    </form>
  );
}

const retentionPresets = [5, 15, 30, 60, 90, 180];

function LogRetentionRow({ value, onChange }: { value: number; onChange: (value: number) => void }) {
  const isPreset = value === 0 || retentionPresets.includes(value);
  const [custom, setCustom] = useState(!isPreset);

  function handleSelect(event: ChangeEvent<HTMLSelectElement>) {
    const next = event.target.value;
    if (next === "custom") {
      setCustom(true);
      return;
    }
    setCustom(false);
    onChange(Number(next));
  }

  return (
    <div className="space-y-2 rounded-md border border-zinc-200 px-4 py-3">
      <label className="block text-sm font-medium text-zinc-700">
        Log retention
        <select
          aria-label="Log retention"
          className="mt-1 w-full rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-950"
          value={custom ? "custom" : String(value)}
          onChange={handleSelect}
        >
          {retentionPresets.map((days) => (
            <option key={days} value={String(days)}>
              {days} days
            </option>
          ))}
          <option value="0">Keep forever</option>
          <option value="custom">Custom…</option>
        </select>
      </label>
      {custom ? (
        <label className="block text-sm font-medium text-zinc-700">
          Custom retention days
          <input
            aria-label="Custom retention days"
            className="mt-1 w-full rounded-md border border-zinc-200 px-3 py-2 text-sm text-zinc-950"
            min={0}
            type="number"
            value={value}
            onChange={(event) => onChange(Math.max(0, Number(event.target.value) || 0))}
          />
        </label>
      ) : null}
    </div>
  );
}

const connectionSources: { token: string; label: string }[] = [
  { token: "local", label: "Local (loopback)" },
  { token: "lan", label: "LAN (private network)" },
  { token: "tailscale", label: "Tailscale" },
  { token: "public", label: "Public web" }
];

function ConnectionSourcesRow({ value, onChange }: { value: string[]; onChange: (value: string[]) => void }) {
  function toggle(token: string, checked: boolean) {
    const next = checked ? [...value.filter((item) => item !== token), token] : value.filter((item) => item !== token);
    onChange(connectionSources.map((source) => source.token).filter((token) => next.includes(token)));
  }

  return (
    <fieldset className="space-y-2 rounded-md border border-zinc-200 px-4 py-3">
      <legend className="text-sm font-medium text-zinc-700">Connection sources</legend>
      <div className="space-y-2">
        {connectionSources.map((source) => (
          <label key={source.token} className="flex items-center justify-between gap-3 text-sm text-zinc-700">
            <span>{source.label}</span>
            <input
              aria-label={source.label}
              checked={value.includes(source.token)}
              className="h-4 w-4 accent-zinc-950"
              type="checkbox"
              onChange={(event) => toggle(source.token, event.target.checked)}
            />
          </label>
        ))}
      </div>
      <p className="text-xs leading-5 text-zinc-500">
        Requests from unchecked sources are rejected (403). Enabling “Public web” allows any source.
      </p>
    </fieldset>
  );
}

function ToggleRow({ checked, label, onChange }: { checked: boolean; label: string; onChange: (value: boolean) => void }) {
  return (
    <label className="flex items-center justify-between gap-3 rounded-md border border-zinc-200 px-4 py-3">
      <span className="text-sm font-medium text-zinc-700">{label}</span>
      <input checked={checked} className="h-4 w-4 accent-zinc-950" type="checkbox" onChange={(event) => onChange(event.target.checked)} />
    </label>
  );
}

function toApiError(error: unknown): ApiError {
  if (error instanceof ApiError) {
    return error;
  }
  return new ApiError(0, error instanceof Error ? error.message : "Unknown API error", error);
}
