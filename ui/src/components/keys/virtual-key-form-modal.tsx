import * as React from "react";
import { apiFetch } from "@/lib/api";
import { Modal } from "@/components/ui/modal";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Toggle } from "@/components/ui/toggle";
import { Button } from "@/components/ui/button";
import { useNotificationStore } from "@/stores/notification";
import {
  KeyIdsEditor,
  emptyProviderConfig,
  type EditorProviderConfig,
} from "./key-ids-editor";

// VirtualKeyRecord mirrors the real Go virtualKeyDTO (virtualkeys.go:13-22).
export interface VirtualKeyRecord {
  id: string;
  key?: string;
  name: string;
  provider_configs?: EditorProviderConfig[];
  budget?: { limit: number; period: string; used?: number };
  rate_limit_rpm?: number;
  is_active: boolean;
}

export interface VirtualKeyFormModalProps {
  open: boolean;
  editing: VirtualKeyRecord | null;
  onClose: () => void;
  onSaved: () => void;
}

const BUDGET_PERIODS = [
  { value: "monthly", label: "Monthly" },
  { value: "weekly", label: "Weekly" },
  { value: "daily", label: "Daily" },
];

// VirtualKeyFormModal (PAR-UI-130) creates/edits a virtual key. It serializes the
// KeyIDs editor selections into provider_configs[{provider,allowed_models,key_ids}]
// and POSTs/PUTs the REAL /api/virtual-keys w5-g CRUD (plan §1.6).
export function VirtualKeyFormModal({ open, editing, onClose, onSaved }: VirtualKeyFormModalProps) {
  const pushToast = useNotificationStore((state) => state.push);
  const [name, setName] = React.useState("");
  const [budgetLimit, setBudgetLimit] = React.useState("");
  const [budgetPeriod, setBudgetPeriod] = React.useState("monthly");
  const [rateLimit, setRateLimit] = React.useState("");
  const [isActive, setIsActive] = React.useState(true);
  const [configs, setConfigs] = React.useState<EditorProviderConfig[]>([emptyProviderConfig()]);
  const [saving, setSaving] = React.useState(false);

  React.useEffect(() => {
    if (!open) return;
    if (editing) {
      setName(editing.name);
      setBudgetLimit(editing.budget ? String(editing.budget.limit) : "");
      setBudgetPeriod(editing.budget?.period || "monthly");
      setRateLimit(editing.rate_limit_rpm != null ? String(editing.rate_limit_rpm) : "");
      setIsActive(editing.is_active);
      setConfigs(
        editing.provider_configs && editing.provider_configs.length > 0
          ? editing.provider_configs.map((pc) => ({
              provider: pc.provider,
              allowed_models: pc.allowed_models ?? [],
              key_ids: pc.key_ids ?? [],
              weight: pc.weight,
            }))
          : [emptyProviderConfig()]
      );
    } else {
      setName("");
      setBudgetLimit("");
      setBudgetPeriod("monthly");
      setRateLimit("");
      setIsActive(true);
      setConfigs([emptyProviderConfig()]);
    }
  }, [open, editing]);

  async function save() {
    if (!name.trim()) {
      pushToast({ message: "Name is required" });
      return;
    }
    const providerConfigs = configs.filter(
      (c) => c.provider && c.allowed_models.length > 0 && c.key_ids.length > 0
    );
    if (providerConfigs.length === 0) {
      pushToast({ message: "Pin at least one model and key ID per provider" });
      return;
    }

    const payload: Record<string, unknown> = {
      name: name.trim(),
      provider_configs: providerConfigs,
      is_active: isActive,
    };
    if (budgetLimit.trim()) {
      payload.budget = { limit: Number(budgetLimit), period: budgetPeriod, used: 0 };
    }
    if (rateLimit.trim()) {
      payload.rate_limit_rpm = Number(rateLimit);
    }

    setSaving(true);
    try {
      const path = editing ? `/api/virtual-keys/${editing.id}` : "/api/virtual-keys";
      await apiFetch(path, {
        method: editing ? "PUT" : "POST",
        body: JSON.stringify(payload),
      });
      pushToast({ message: editing ? "Virtual key updated" : "Virtual key created" });
      onSaved();
      onClose();
    } catch {
      pushToast({ message: "Failed to save the virtual key" });
    } finally {
      setSaving(false);
    }
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={editing ? "Edit virtual key" : "Create virtual key"}
      size="lg"
    >
      <div className="flex flex-col gap-4">
        <Input
          data-testid="vk-name"
          label="Name"
          value={name}
          onChange={(event) => setName(event.target.value)}
          placeholder="Team Alpha"
        />
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
          <Input
            data-testid="vk-budget"
            label="Budget (USD)"
            type="number"
            value={budgetLimit}
            onChange={(event) => setBudgetLimit(event.target.value)}
          />
          <Select
            label="Period"
            options={BUDGET_PERIODS}
            value={budgetPeriod}
            onChange={(event) => setBudgetPeriod(event.target.value)}
          />
          <Input
            data-testid="vk-rpm"
            label="Rate limit (RPM)"
            type="number"
            value={rateLimit}
            onChange={(event) => setRateLimit(event.target.value)}
          />
        </div>

        <div className="flex items-center gap-2">
          <Toggle
            checked={isActive}
            onCheckedChange={setIsActive}
            aria-label="Active"
          />
          <span className="text-sm text-foreground">Active</span>
        </div>

        <div className="flex flex-col gap-2">
          <span className="text-sm font-semibold text-foreground">Provider pinning</span>
          <KeyIdsEditor value={configs} onChange={setConfigs} />
        </div>

        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button data-testid="vk-save" variant="primary" loading={saving} onClick={save}>
            Save
          </Button>
        </div>
      </div>
    </Modal>
  );
}
