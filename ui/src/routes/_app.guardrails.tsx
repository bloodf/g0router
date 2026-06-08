import { createFileRoute } from "@tanstack/react-router";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { apiFetch } from "@/lib/api/client";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { Label } from "@/components/ui/label";
import { PageHeader } from "@/components/common/PageHeader";
import { Icon } from "@/components/common/Icon";
import { CardSkeleton, ErrorState } from "@/components/common/Skeletons";
import { toast } from "sonner";
import type { Guardrails as GuardrailsConfig } from "@/lib/types";

export const Route = createFileRoute("/_app/guardrails")({
  component: GuardrailsPage,
});

function GuardrailsPage() {
  const qc = useQueryClient();
  const [form, setForm] = useState<GuardrailsConfig | null>(null);
  const [testPrompt, setTestPrompt] = useState("");
  const [testResult, setTestResult] = useState<{
    blocked: boolean;
    redacted_prompt: string;
    matches: string[];
  } | null>(null);

  const { data, isLoading, isError, error, refetch } = useQuery<GuardrailsConfig>({
    queryKey: ["guardrails"],
    queryFn: () => apiFetch("/api/guardrails"),
  });

  useEffect(() => {
    if (data) setForm(data);
  }, [data]);

  const save = useMutation({
    mutationFn: (body: GuardrailsConfig) =>
      apiFetch<GuardrailsConfig>("/api/guardrails", { method: "PUT", body }),
    onSuccess: (saved) => {
      qc.setQueryData(["guardrails"], saved);
      toast.success("Guardrails updated");
    },
    onError: (e: any) => toast.error(e?.message || "Failed to save"),
  });

  const test = useMutation({
    mutationFn: (prompt: string) =>
      apiFetch<{
        blocked: boolean;
        redacted_prompt: string;
        matches: string[];
      }>("/api/guardrails/test", { method: "POST", body: { prompt } }),
    onSuccess: setTestResult,
    onError: (e: any) => toast.error(e?.message || "Test failed"),
  });

  if (isLoading || !form) {
    return (
      <div className="space-y-6">
        <PageHeader
          title="Guardrails"
          description="Blocklist and PII redaction settings."
          icon="shield"
        />
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
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
          title="Guardrails"
          description="Blocklist and PII redaction settings."
          icon="shield"
        />
        <ErrorState title="Couldn’t load guardrails" error={error} onRetry={refetch} />
      </div>
    );
  }

  const update = <K extends keyof GuardrailsConfig>(key: K, value: GuardrailsConfig[K]) => {
    setForm((prev) => (prev ? { ...prev, [key]: value } : prev));
  };

  const parseList = (raw: string) =>
    raw
      .split("\n")
      .map((s) => s.trim())
      .filter(Boolean);

  return (
    <div className="space-y-6">
      <PageHeader
        title="Guardrails"
        description="Blocklist and PII redaction settings."
        icon="shield"
        actions={
          <Button onClick={() => save.mutate(form)} disabled={save.isPending}>
            <Icon name={save.isPending ? "hourglass_empty" : "save"} size={16} className="mr-1.5" />
            {save.isPending ? "Saving…" : "Save changes"}
          </Button>
        }
      />

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <Card className="card-elev border-border p-5 space-y-4">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-2">
            <Icon name="block" size={18} className="text-destructive" />
            Blocklist
          </h2>
          <ToggleField
            label="Enable blocklist"
            checked={form.guardrails_enabled}
            onCheckedChange={(v) => update("guardrails_enabled", v)}
          />
          <div className="space-y-1.5">
            <Label className="text-sm font-medium">Blocklist terms (one per line)</Label>
            <textarea
              className="w-full bg-surface-2 border border-border rounded-lg px-3 py-2 text-sm focus:border-brand-500 outline-none min-h-[120px]"
              value={form.guardrails_blocklist?.join("\n") ?? ""}
              onChange={(e) => update("guardrails_blocklist", parseList(e.target.value))}
              placeholder="badword&#8203;"
            />
          </div>
        </Card>

        <Card className="card-elev border-border p-5 space-y-4">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-2">
            <Icon name="admin_panel_settings" size={18} className="text-warning" />
            PII Redaction
          </h2>
          <ToggleField
            label="Enable PII redaction"
            checked={form.pii_redaction_enabled}
            onCheckedChange={(v) => update("pii_redaction_enabled", v)}
          />
          <div className="space-y-1.5">
            <Label className="text-sm font-medium">PII types (one per line)</Label>
            <textarea
              className="w-full bg-surface-2 border border-border rounded-lg px-3 py-2 text-sm focus:border-brand-500 outline-none min-h-[120px]"
              value={form.pii_redaction_types?.join("\n") ?? ""}
              onChange={(e) => update("pii_redaction_types", parseList(e.target.value))}
              placeholder="email\nphone\nssn"
            />
          </div>
        </Card>

        <Card className="card-elev border-border p-5 lg:col-span-2 space-y-4">
          <h2 className="text-sm font-semibold flex items-center gap-2 mb-2">
            <Icon name="science" size={18} className="text-info" />
            Test prompt
          </h2>
          <div className="flex gap-2">
            <input
              className="flex-1 bg-surface-2 border border-border rounded-lg px-3 py-2 text-sm focus:border-brand-500 outline-none"
              value={testPrompt}
              onChange={(e) => setTestPrompt(e.target.value)}
              placeholder="Enter a prompt to test blocklist and PII redaction"
            />
            <Button
              onClick={() => test.mutate(testPrompt)}
              disabled={test.isPending || !testPrompt}
            >
              {test.isPending ? "Testing…" : "Test"}
            </Button>
          </div>
          {testResult && (
            <div className="space-y-2 text-sm">
              <div className="flex items-center gap-2">
                <span className="text-text-muted">Blocked:</span>
                <span
                  className={
                    testResult.blocked ? "text-destructive font-medium" : "text-success font-medium"
                  }
                >
                  {testResult.blocked ? "Yes" : "No"}
                </span>
              </div>
              {testResult.matches.length > 0 && (
                <div>
                  <span className="text-text-muted">Matches:</span> {testResult.matches.join(", ")}
                </div>
              )}
              <div>
                <span className="text-text-muted">Redacted:</span>{" "}
                <span className="font-mono text-xs bg-surface-2 px-1.5 py-0.5 rounded">
                  {testResult.redacted_prompt}
                </span>
              </div>
            </div>
          )}
        </Card>
      </div>
    </div>
  );
}

function ToggleField({
  label,
  checked,
  onCheckedChange,
}: {
  label: string;
  checked: boolean;
  onCheckedChange: (v: boolean) => void;
}) {
  return (
    <div className="flex items-start justify-between gap-4">
      <div className="text-sm font-medium">{label}</div>
      <Switch checked={checked} onCheckedChange={onCheckedChange} className="shrink-0" />
    </div>
  );
}
