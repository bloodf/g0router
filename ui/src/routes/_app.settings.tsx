import { createFileRoute } from "@tanstack/react-router";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api/client";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Checkbox } from "@/components/ui/checkbox";
import { PageHeader } from "@/components/common/PageHeader";
import { Icon } from "@/components/common/Icon";
import { CardSkeleton, ErrorState } from "@/components/common/Skeletons";
import { toast } from "sonner";

export const Route = createFileRoute("/_app/settings")({
  component: SettingsPage,
});

interface BackendSettings {
  require_api_key: boolean;
  require_login: boolean;
  trust_proxy_headers: boolean;
  rtk_enabled: boolean;
  caveman_enabled: boolean;
  caveman_level: "lite" | "full" | "ultra";
  enable_request_logs: boolean;
  proxy_url: string;
  data_dir: string;
  log_retention_days: number;
  allowed_sources: string[];
  notify_webhook_url: string;
  notify_on_reauth: boolean;
  cache_enabled: boolean;
  cache_ttl_seconds: number;
  locale: string;
}

type FormState = BackendSettings & { api_key_secret: string };

const SOURCE_OPTIONS = ["local", "lan", "tailscale", "public"] as const;
const LOCALE_OPTIONS = ["en", "pt", "es", "fr", "de", "ja", "zh"] as const;
const CAVEMAN_LEVELS = ["lite", "full", "ultra"] as const;

const PLACEHOLDER_DOTS = "••••••••";

function generateHex(length: number): string {
  const bytes = new Uint8Array(length);
  crypto.getRandomValues(bytes);
  return Array.from(bytes)
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
}

function SettingsPage() {
  const qc = useQueryClient();
  const [form, setForm] = useState<FormState | null>(null);
  const [apiKeySecret, setApiKeySecret] = useState("");
  const [showSecret, setShowSecret] = useState(false);

  const {
    data: settings,
    isLoading,
    isError,
    error,
    refetch,
  } = useQuery<BackendSettings>({
    queryKey: ["settings"],
    queryFn: () => apiFetch("/api/settings"),
  });

  useEffect(() => {
    if (settings) {
      setForm({
        ...settings,
        api_key_secret: "",
      });
      setApiKeySecret("");
    }
  }, [settings]);

  const saveMutation = useMutation({
    mutationFn: async (body: Partial<FormState>) => {
      return apiFetch<BackendSettings>("/api/settings", {
        method: "PUT",
        body,
      });
    },
    onSuccess: (data) => {
      qc.setQueryData(["settings"], data);
      toast.success("Settings saved");
      setApiKeySecret("");
    },
    onError: (e: any) => {
      const msg = e?.message || "Failed to save settings";
      toast.error(msg);
    },
  });

  const handleSave = () => {
    if (!form) return;
    const body: Partial<FormState> = { ...form };
    delete body.api_key_secret;
    if (apiKeySecret) {
      body.api_key_secret = apiKeySecret;
    }
    saveMutation.mutate(body);
  };

  const updateField = <K extends keyof FormState>(
    key: K,
    value: FormState[K],
  ) => {
    setForm((prev) => (prev ? { ...prev, [key]: value } : prev));
  };

  const toggleSource = (source: string, checked: boolean) => {
    setForm((prev) => {
      if (!prev) return prev;
      const sources = new Set(prev.allowed_sources);
      if (checked) sources.add(source);
      else sources.delete(source);
      return { ...prev, allowed_sources: Array.from(sources) };
    });
  };

  if (isLoading || !form) {
    return (
      <div className="space-y-6">
        <PageHeader
          title="Settings"
          description="Configure gateway behaviour, logging, and security."
          icon="settings"
        />
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <CardSkeleton lines={4} />
          <CardSkeleton lines={4} />
          <CardSkeleton lines={4} />
          <CardSkeleton lines={4} />
        </div>
      </div>
    );
  }

  if (isError) {
    return (
      <div className="space-y-6">
        <PageHeader
          title="Settings"
          description="Configure gateway behaviour, logging, and security."
          icon="settings"
        />
        <ErrorState
          title="Couldn’t load settings"
          error={error}
          onRetry={refetch}
        />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title="Settings"
        description="Configure gateway behaviour, logging, and security."
        icon="settings"
        actions={
          <Button
            onClick={handleSave}
            disabled={saveMutation.isPending}
            className="btn-cta"
          >
            <Icon
              name={saveMutation.isPending ? "hourglass_empty" : "save"}
              size={16}
              className={saveMutation.isPending ? "animate-spin" : ""}
            />
            {saveMutation.isPending ? "Saving…" : "Save changes"}
          </Button>
        }
      />

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* General */}
        <Card className="card-elev border-border p-5">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
            <Icon name="settings" size={18} className="text-brand-500" />
            General
          </h2>
          <div className="space-y-4">
            <ToggleField
              label="Require API key"
              description="Reject requests that don’t include a valid API key."
              checked={form.require_api_key}
              onCheckedChange={(v) => updateField("require_api_key", v)}
            />
            <ToggleField
              label="Require login"
              description="Force users to authenticate before accessing the dashboard."
              checked={form.require_login}
              onCheckedChange={(v) => updateField("require_login", v)}
            />
            <div className="space-y-1.5">
              <Label className="text-sm font-medium">Locale</Label>
              <Select
                value={form.locale}
                onValueChange={(v) => updateField("locale", v)}
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {LOCALE_OPTIONS.map((loc) => (
                    <SelectItem key={loc} value={loc}>
                      {loc.toUpperCase()}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
        </Card>

        {/* Logging */}
        <Card className="card-elev border-border p-5">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
            <Icon name="description" size={18} className="text-info" />
            Logging
          </h2>
          <div className="space-y-4">
            <ToggleField
              label="Enable request logs"
              description="Store every incoming request in the local database."
              checked={form.enable_request_logs}
              onCheckedChange={(v) => updateField("enable_request_logs", v)}
            />
            <div className="space-y-1.5">
              <Label className="text-sm font-medium">Log retention (days)</Label>
              <Input
                type="number"
                min={0}
                max={36500}
                value={form.log_retention_days}
                onChange={(e) =>
                  updateField(
                    "log_retention_days",
                    Math.max(0, Math.min(36500, Number(e.target.value) || 0)),
                  )
                }
              />
            </div>
          </div>
        </Card>

        {/* Features */}
        <Card className="card-elev border-border p-5">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
            <Icon name="bolt" size={18} className="text-warning" />
            Features
          </h2>
          <div className="space-y-4">
            <ToggleField
              label="RTK enabled"
              description="Compress repeated tokens in streamed responses."
              checked={form.rtk_enabled}
              onCheckedChange={(v) => updateField("rtk_enabled", v)}
            />
            <ToggleField
              label="Caveman mode"
              description="Ultra-compressed communication for agent contexts."
              checked={form.caveman_enabled}
              onCheckedChange={(v) => updateField("caveman_enabled", v)}
            />
            {form.caveman_enabled && (
              <div className="space-y-1.5 pl-4 border-l-2 border-border">
                <Label className="text-sm font-medium">Caveman level</Label>
                <Select
                  value={form.caveman_level}
                  onValueChange={(v) =>
                    updateField("caveman_level", v as "lite" | "full" | "ultra")
                  }
                >
                  <SelectTrigger className="w-full">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {CAVEMAN_LEVELS.map((lvl) => (
                      <SelectItem key={lvl} value={lvl}>
                        {lvl.charAt(0).toUpperCase() + lvl.slice(1)}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}
            <ToggleField
              label="Cache enabled"
              description="Cache embeddings and low-volatility responses."
              checked={form.cache_enabled}
              onCheckedChange={(v) => updateField("cache_enabled", v)}
            />
            {form.cache_enabled && (
              <div className="space-y-1.5 pl-4 border-l-2 border-border">
                <Label className="text-sm font-medium">Cache TTL (seconds)</Label>
                <Input
                  type="number"
                  min={0}
                  value={form.cache_ttl_seconds}
                  onChange={(e) =>
                    updateField(
                      "cache_ttl_seconds",
                      Math.max(0, Number(e.target.value) || 0),
                    )
                  }
                />
              </div>
            )}
          </div>
        </Card>

        {/* Network */}
        <Card className="card-elev border-border p-5">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
            <Icon name="network_node" size={18} className="text-success" />
            Network
          </h2>
          <div className="space-y-4">
            <div className="space-y-1.5">
              <Label className="text-sm font-medium">Proxy URL</Label>
              <Input
                type="text"
                value={form.proxy_url}
                onChange={(e) => updateField("proxy_url", e.target.value)}
                placeholder="http://proxy.example.com:8080"
              />
            </div>
            <ToggleField
              label="Trust proxy headers"
              description="Use X-Forwarded-* headers when behind a reverse proxy."
              checked={form.trust_proxy_headers}
              onCheckedChange={(v) => updateField("trust_proxy_headers", v)}
            />
            <div className="space-y-2">
              <Label className="text-sm font-medium">Allowed sources</Label>
              <div className="grid grid-cols-2 gap-2">
                {SOURCE_OPTIONS.map((source) => (
                  <div key={source} className="flex items-center gap-2">
                    <Checkbox
                      id={`source-${source}`}
                      checked={form.allowed_sources.includes(source)}
                      onCheckedChange={(checked) =>
                        toggleSource(source, checked === true)
                      }
                    />
                    <Label
                      htmlFor={`source-${source}`}
                      className="text-sm capitalize cursor-pointer"
                    >
                      {source}
                    </Label>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </Card>

        {/* Notifications */}
        <Card className="card-elev border-border p-5">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
            <Icon name="notifications" size={18} className="text-accent-fuchsia" />
            Notifications
          </h2>
          <div className="space-y-4">
            <ToggleField
              label="Notify on re-auth"
              description="Send a webhook when a provider needs re-authentication."
              checked={form.notify_on_reauth}
              onCheckedChange={(v) => updateField("notify_on_reauth", v)}
            />
            <div className="space-y-1.5">
              <Label className="text-sm font-medium">Webhook URL</Label>
              <Input
                type="url"
                value={form.notify_webhook_url}
                onChange={(e) =>
                  updateField("notify_webhook_url", e.target.value)
                }
                placeholder="https://hooks.example.com/g0router"
              />
            </div>
          </div>
        </Card>

        {/* Security */}
        <Card className="card-elev border-border p-5">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-4">
            <Icon name="lock" size={18} className="text-destructive" />
            Security
          </h2>
          <div className="space-y-3">
            <Label className="text-sm font-medium">API key secret</Label>
            <div className="flex items-center gap-2">
              <Input
                type={showSecret ? "text" : "password"}
                value={apiKeySecret}
                onChange={(e) => setApiKeySecret(e.target.value)}
                placeholder={PLACEHOLDER_DOTS}
                className="font-mono"
              />
              <Button
                variant="outline"
                size="sm"
                onClick={() => setShowSecret((s) => !s)}
                type="button"
              >
                <Icon
                  name={showSecret ? "visibility_off" : "visibility"}
                  size={16}
                  className="mr-1"
                />
                {showSecret ? "Hide" : "Show"}
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => setApiKeySecret(generateHex(32))}
                type="button"
              >
                <Icon name="autorenew" size={16} className="mr-1" />
                Regenerate
              </Button>
            </div>
            <p className="text-xs text-text-muted">
              Write-only. Leave empty to keep the current secret.
            </p>
          </div>
        </Card>
      </div>
    </div>
  );
}

function ToggleField({
  label,
  description,
  checked,
  onCheckedChange,
}: {
  label: string;
  description?: string;
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
}) {
  return (
    <div className="flex items-start justify-between gap-4">
      <div className="space-y-0.5 min-w-0">
        <div className="text-sm font-medium">{label}</div>
        {description && (
          <p className="text-xs text-text-muted">{description}</p>
        )}
      </div>
      <Switch
        checked={checked}
        onCheckedChange={onCheckedChange}
        className="shrink-0 mt-0.5"
      />
    </div>
  );
}
