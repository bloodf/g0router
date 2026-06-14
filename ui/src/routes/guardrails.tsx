import * as React from "react";
import { createFileRoute } from "@tanstack/react-router";
import { apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Toggle } from "@/components/ui/toggle";
import { CardSkeleton } from "@/components/ui/skeleton";
import { GuardrailsTester } from "@/components/governance/guardrails-tester";
import { useNotificationStore } from "@/stores/notification";
import type { Guardrails } from "@/lib/types";

export const Route = createFileRoute("/guardrails")({
  component: GuardrailsPage,
});

// GuardrailsPage (PAR-UI-130 subset) edits the guardrails singleton config via
// GET/PUT /api/guardrails and embeds the prompt tester (§1.3). Variant-HAVE
// against the mock; no Go /api/guardrails exists yet (§8 ESCALATION-1d).
function GuardrailsPage() {
  const pushToast = useNotificationStore((state) => state.push);
  const [config, setConfig] = React.useState<Guardrails | null>(null);
  const [loading, setLoading] = React.useState(true);
  const [blocklistText, setBlocklistText] = React.useState("");
  const [piiTypesText, setPiiTypesText] = React.useState("");
  const [saving, setSaving] = React.useState(false);

  const load = React.useCallback(() => {
    setLoading(true);
    apiFetch<Guardrails>("/api/guardrails")
      .then((data) => {
        setConfig(data);
        setBlocklistText((data?.guardrails_blocklist ?? []).join(", "));
        setPiiTypesText((data?.pii_redaction_types ?? []).join(", "));
        setLoading(false);
      })
      .catch(() => {
        setLoading(false);
        pushToast({ message: "Failed to load guardrails" });
      });
  }, [pushToast]);

  React.useEffect(() => {
    load();
  }, [load]);

  function splitList(value: string): string[] {
    return value
      .split(",")
      .map((entry) => entry.trim())
      .filter(Boolean);
  }

  async function save() {
    if (!config) return;
    setSaving(true);
    const payload: Guardrails = {
      guardrails_enabled: config.guardrails_enabled,
      guardrails_blocklist: splitList(blocklistText),
      pii_redaction_enabled: config.pii_redaction_enabled,
      pii_redaction_types: splitList(piiTypesText),
    };
    try {
      await apiFetch("/api/guardrails", {
        method: "PUT",
        body: JSON.stringify(payload),
      });
      setConfig(payload);
      pushToast({ message: "Guardrails saved" });
    } catch {
      pushToast({ message: "Failed to save guardrails" });
    } finally {
      setSaving(false);
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <header>
        <h1 className="text-2xl font-semibold text-foreground">Guardrails</h1>
      </header>

      {loading || !config ? (
        <CardSkeleton />
      ) : (
        <div className="flex flex-col gap-4 rounded-lg border border-border px-4 py-4">
          <label className="flex items-center justify-between text-sm text-foreground">
            Guardrails enabled
            <Toggle
              data-testid="guardrails-enabled"
              checked={config.guardrails_enabled}
              onCheckedChange={(checked) =>
                setConfig({ ...config, guardrails_enabled: checked })
              }
              aria-label="Guardrails enabled"
            />
          </label>
          <Input
            id="guardrails-blocklist"
            data-testid="guardrails-blocklist"
            label="Blocklist (comma-separated)"
            value={blocklistText}
            onChange={(event) => setBlocklistText(event.target.value)}
          />
          <label className="flex items-center justify-between text-sm text-foreground">
            PII redaction enabled
            <Toggle
              checked={config.pii_redaction_enabled}
              onCheckedChange={(checked) =>
                setConfig({ ...config, pii_redaction_enabled: checked })
              }
              aria-label="PII redaction enabled"
            />
          </label>
          <Input
            id="guardrails-pii-types"
            label="PII types (comma-separated)"
            value={piiTypesText}
            onChange={(event) => setPiiTypesText(event.target.value)}
          />
          <div className="flex justify-end">
            <Button
              data-testid="guardrails-save"
              variant="primary"
              loading={saving}
              onClick={save}
            >
              Save
            </Button>
          </div>
        </div>
      )}

      <div className="flex flex-col gap-3 rounded-lg border border-border px-4 py-4">
        <h2 className="text-lg font-semibold text-foreground">Prompt tester</h2>
        <GuardrailsTester />
      </div>
    </div>
  );
}
