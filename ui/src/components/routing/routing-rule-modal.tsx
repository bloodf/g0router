import * as React from "react";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Toggle } from "@/components/ui/toggle";
import { apiFetch } from "@/lib/api";
import { useNotificationStore } from "@/stores/notification";
import type { RoutingRule } from "@/lib/types";

export interface RoutingRuleModalProps {
  open: boolean;
  rule: RoutingRule | null;
  onClose: () => void;
  onSaved?: () => void;
}

// RoutingRuleModal creates/edits a routing rule via POST /api/routing-rules (new)
// or PUT /api/routing-rules/{id} (edit). Variant-HAVE against the mock; no Go
// backend exists yet (§8 ESCALATION-3a).
function RoutingRuleModal({ open, rule, onClose, onSaved }: RoutingRuleModalProps) {
  const pushToast = useNotificationStore((state) => state.push);
  const [name, setName] = React.useState("");
  const [priority, setPriority] = React.useState("1");
  const [condField, setCondField] = React.useState("model");
  const [condOperator, setCondOperator] = React.useState("equals");
  const [condValue, setCondValue] = React.useState("");
  const [targetProvider, setTargetProvider] = React.useState("");
  const [isActive, setIsActive] = React.useState(true);
  const [busy, setBusy] = React.useState(false);

  React.useEffect(() => {
    if (rule) {
      setName(rule.name);
      setPriority(String(rule.priority));
      setCondField(rule.cond_field);
      setCondOperator(rule.cond_operator);
      setCondValue(rule.cond_value);
      setTargetProvider(rule.target_provider);
      setIsActive(rule.is_active);
    } else {
      setName("");
      setPriority("1");
      setCondField("model");
      setCondOperator("equals");
      setCondValue("");
      setTargetProvider("");
      setIsActive(true);
    }
  }, [rule]);

  async function save() {
    setBusy(true);
    const payload = {
      name,
      priority: Number(priority) || 0,
      cond_field: condField,
      cond_operator: condOperator,
      cond_value: condValue,
      target_provider: targetProvider,
      is_active: isActive,
    };
    try {
      if (rule) {
        await apiFetch(`/api/routing-rules/${rule.id}`, {
          method: "PUT",
          body: JSON.stringify(payload),
        });
      } else {
        await apiFetch("/api/routing-rules", {
          method: "POST",
          body: JSON.stringify(payload),
        });
      }
      pushToast({ message: rule ? "Rule updated" : "Rule created" });
      onSaved?.();
      onClose();
    } catch {
      pushToast({ message: "Failed to save the rule" });
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open={open} onClose={onClose} title={rule ? "Edit rule" : "New rule"}>
      <div className="flex flex-col gap-4">
        <Input
          id="routing-rule-name"
          label="Name"
          value={name}
          onChange={(event) => setName(event.target.value)}
        />
        <Input
          id="routing-rule-priority"
          label="Priority"
          type="number"
          value={priority}
          onChange={(event) => setPriority(event.target.value)}
        />
        <Input
          id="routing-rule-cond-field"
          label="Condition field"
          value={condField}
          onChange={(event) => setCondField(event.target.value)}
        />
        <Input
          id="routing-rule-cond-operator"
          label="Condition operator"
          value={condOperator}
          onChange={(event) => setCondOperator(event.target.value)}
        />
        <Input
          id="routing-rule-cond-value"
          label="Condition value"
          value={condValue}
          onChange={(event) => setCondValue(event.target.value)}
        />
        <Input
          id="routing-rule-target-provider"
          label="Target provider"
          value={targetProvider}
          onChange={(event) => setTargetProvider(event.target.value)}
        />
        <label className="flex items-center justify-between text-sm text-foreground">
          Active
          <Toggle checked={isActive} onCheckedChange={setIsActive} />
        </label>
        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button
            data-testid="routing-rule-save"
            variant="primary"
            loading={busy}
            onClick={save}
          >
            Save
          </Button>
        </div>
      </div>
    </Modal>
  );
}

export { RoutingRuleModal };
