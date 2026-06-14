import * as React from "react";
import { Modal } from "@/components/ui/modal";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { apiFetch } from "@/lib/api";
import { useNotificationStore } from "@/stores/notification";

// PricingModal edits a provider/model's rates (PAR-UI-057). Fields and verbs
// mirror the real Go pricing API (internal/admin/pricing.go:11-17,30,56):
//   save  -> PATCH /api/pricing  {provider:{model:{field:value}}}
//   reset -> DELETE /api/pricing?provider=&model=
export interface PricingRates {
  input: number;
  output: number;
  cached: number;
  reasoning: number;
  cache_creation: number;
}

export interface PricingModalProps {
  open: boolean;
  onClose: () => void;
  provider: string;
  model: string;
  rates: PricingRates;
  onSaved?: () => void;
}

const FIELDS: { key: keyof PricingRates; label: string }[] = [
  { key: "input", label: "Input ($/M)" },
  { key: "output", label: "Output ($/M)" },
  { key: "cached", label: "Cached ($/M)" },
  { key: "reasoning", label: "Reasoning ($/M)" },
  { key: "cache_creation", label: "Cache creation ($/M)" },
];

export function PricingModal({ open, onClose, provider, model, rates, onSaved }: PricingModalProps) {
  const pushToast = useNotificationStore((s) => s.push);
  const [values, setValues] = React.useState<PricingRates>(rates);
  const [saving, setSaving] = React.useState(false);

  React.useEffect(() => {
    setValues(rates);
  }, [rates, provider, model]);

  const setField = (key: keyof PricingRates, raw: string) => {
    const n = Number(raw);
    setValues((v) => ({ ...v, [key]: Number.isFinite(n) ? n : 0 }));
  };

  const save = async () => {
    setSaving(true);
    try {
      await apiFetch("/api/pricing", {
        method: "PATCH",
        body: JSON.stringify({ [provider]: { [model]: values } }),
      });
      pushToast({ message: `Pricing updated for ${provider}/${model}` });
      onSaved?.();
      onClose();
    } catch {
      pushToast({ message: "Failed to update pricing" });
    } finally {
      setSaving(false);
    }
  };

  const reset = async () => {
    setSaving(true);
    try {
      const params = new URLSearchParams({ provider, model });
      await apiFetch(`/api/pricing?${params.toString()}`, { method: "DELETE" });
      pushToast({ message: `Pricing reset for ${provider}/${model}` });
      onSaved?.();
      onClose();
    } catch {
      pushToast({ message: "Failed to reset pricing" });
    } finally {
      setSaving(false);
    }
  };

  return (
    <Modal open={open} onClose={onClose} title={`${provider} / ${model}`} size="md">
      <div className="flex flex-col gap-3">
        {FIELDS.map((f) => (
          <Input
            key={f.key}
            id={`pricing-${f.key}`}
            label={f.label}
            type="number"
            step="0.01"
            min="0"
            value={String(values[f.key])}
            onChange={(e) => setField(f.key, e.target.value)}
          />
        ))}
        <div className="flex items-center justify-between pt-2">
          <Button variant="ghost" onClick={reset} disabled={saving} data-testid="pricing-reset">
            Reset
          </Button>
          <div className="flex gap-2">
            <Button variant="outline" onClick={onClose} disabled={saving}>
              Cancel
            </Button>
            <Button variant="primary" onClick={save} loading={saving} data-testid="pricing-save">
              Save
            </Button>
          </div>
        </div>
      </div>
    </Modal>
  );
}
