import * as React from "react";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { apiFetch } from "@/lib/api";
import { useNotificationStore } from "@/stores/notification";
import type { ModelLimit } from "@/lib/types";

export interface ModelLimitModalProps {
  open: boolean;
  limit: ModelLimit | null;
  onClose: () => void;
  onSaved?: () => void;
}

// ModelLimitModal creates/edits a model limit via POST /api/model-limits (new)
// or PUT /api/model-limits/{id} (edit). Variant-HAVE against the mock; no Go
// backend exists yet (§8 ESCALATION-3b).
function ModelLimitModal({ open, limit, onClose, onSaved }: ModelLimitModalProps) {
  const pushToast = useNotificationStore((state) => state.push);
  const [model, setModel] = React.useState("");
  const [maxTokens, setMaxTokens] = React.useState("0");
  const [maxRpm, setMaxRpm] = React.useState("0");
  const [allowedKeyIds, setAllowedKeyIds] = React.useState("");
  const [busy, setBusy] = React.useState(false);

  React.useEffect(() => {
    if (limit) {
      setModel(limit.model);
      setMaxTokens(String(limit.max_tokens));
      setMaxRpm(String(limit.max_rpm));
      setAllowedKeyIds((limit.allowed_key_ids ?? []).join(", "));
    } else {
      setModel("");
      setMaxTokens("0");
      setMaxRpm("0");
      setAllowedKeyIds("");
    }
  }, [limit]);

  async function save() {
    setBusy(true);
    const payload = {
      model,
      max_tokens: Number(maxTokens) || 0,
      max_rpm: Number(maxRpm) || 0,
      allowed_key_ids: allowedKeyIds
        .split(",")
        .map((s) => s.trim())
        .filter(Boolean),
    };
    try {
      if (limit) {
        await apiFetch(`/api/model-limits/${limit.id}`, {
          method: "PUT",
          body: JSON.stringify(payload),
        });
      } else {
        await apiFetch("/api/model-limits", {
          method: "POST",
          body: JSON.stringify(payload),
        });
      }
      pushToast({ message: limit ? "Limit updated" : "Limit created" });
      onSaved?.();
      onClose();
    } catch {
      pushToast({ message: "Failed to save the limit" });
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open={open} onClose={onClose} title={limit ? "Edit limit" : "New limit"}>
      <div className="flex flex-col gap-4">
        <Input
          id="model-limit-model"
          label="Model"
          value={model}
          onChange={(event) => setModel(event.target.value)}
        />
        <Input
          id="model-limit-max-tokens"
          label="Max tokens"
          type="number"
          value={maxTokens}
          onChange={(event) => setMaxTokens(event.target.value)}
        />
        <Input
          id="model-limit-max-rpm"
          label="Max RPM"
          type="number"
          value={maxRpm}
          onChange={(event) => setMaxRpm(event.target.value)}
        />
        <Input
          id="model-limit-allowed-key-ids"
          label="Allowed key IDs (comma separated)"
          value={allowedKeyIds}
          onChange={(event) => setAllowedKeyIds(event.target.value)}
        />
        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button
            data-testid="model-limit-save"
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

export { ModelLimitModal };
