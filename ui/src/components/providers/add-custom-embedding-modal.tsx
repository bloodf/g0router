import * as React from "react";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { apiFetch } from "@/lib/api";
import { useNotificationStore } from "@/stores/notification";

export interface AddCustomEmbeddingModalProps {
  open: boolean;
  onClose: () => void;
  onCreated?: () => void;
}

// AddCustomEmbeddingModal (PAR-UI-063) — register a custom embedding/model via
// POST /api/models/custom (port of 9router AddCustomEmbeddingModal.js).
function AddCustomEmbeddingModal({
  open,
  onClose,
  onCreated,
}: AddCustomEmbeddingModalProps) {
  const pushToast = useNotificationStore((state) => state.push);
  const [provider, setProvider] = React.useState("");
  const [name, setName] = React.useState("");
  const [busy, setBusy] = React.useState(false);

  React.useEffect(() => {
    if (open) {
      setProvider("");
      setName("");
    }
  }, [open]);

  async function create() {
    setBusy(true);
    try {
      await apiFetch("/api/models/custom", {
        method: "POST",
        body: JSON.stringify({
          provider,
          name,
          input_cost: 0,
          output_cost: 0,
          context_window: 0,
        }),
      });
      pushToast({ message: "Added custom model" });
      onCreated?.();
      onClose();
    } catch {
      pushToast({ message: "Failed to add the custom model" });
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open={open} onClose={onClose} title="Add custom model">
      <div className="flex flex-col gap-4">
        <Input
          label="Provider"
          value={provider}
          onChange={(event) => setProvider(event.target.value)}
        />
        <Input
          label="Model name"
          value={name}
          onChange={(event) => setName(event.target.value)}
        />
        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button variant="primary" loading={busy} onClick={create}>
            Add model
          </Button>
        </div>
      </div>
    </Modal>
  );
}

export { AddCustomEmbeddingModal };
