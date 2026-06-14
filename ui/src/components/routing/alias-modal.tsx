import * as React from "react";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { apiFetch } from "@/lib/api";
import { useNotificationStore } from "@/stores/notification";
import type { Alias } from "@/lib/types";

export interface AliasModalProps {
  open: boolean;
  alias: Alias | null;
  onClose: () => void;
  onSaved?: () => void;
}

// AliasModal creates/edits a model alias via POST /api/aliases (new) or
// PUT /api/aliases/{id} (edit). Variant-HAVE against the mock; store.ListAliases
// exists but there is no admin endpoint yet (§8 ESCALATION-2).
function AliasModal({ open, alias, onClose, onSaved }: AliasModalProps) {
  const pushToast = useNotificationStore((state) => state.push);
  const [aliasName, setAliasName] = React.useState("");
  const [provider, setProvider] = React.useState("");
  const [model, setModel] = React.useState("");
  const [busy, setBusy] = React.useState(false);

  React.useEffect(() => {
    if (alias) {
      setAliasName(alias.alias);
      setProvider(alias.provider);
      setModel(alias.model);
    } else {
      setAliasName("");
      setProvider("");
      setModel("");
    }
  }, [alias]);

  async function save() {
    setBusy(true);
    const payload = { alias: aliasName, provider, model };
    try {
      if (alias) {
        await apiFetch(`/api/aliases/${alias.id}`, {
          method: "PUT",
          body: JSON.stringify(payload),
        });
      } else {
        await apiFetch("/api/aliases", {
          method: "POST",
          body: JSON.stringify(payload),
        });
      }
      pushToast({ message: alias ? "Alias updated" : "Alias created" });
      onSaved?.();
      onClose();
    } catch {
      pushToast({ message: "Failed to save the alias" });
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open={open} onClose={onClose} title={alias ? "Edit alias" : "New alias"}>
      <div className="flex flex-col gap-4">
        <Input
          id="alias-name"
          label="Alias"
          value={aliasName}
          onChange={(event) => setAliasName(event.target.value)}
        />
        <Input
          id="alias-provider"
          label="Provider"
          value={provider}
          onChange={(event) => setProvider(event.target.value)}
        />
        <Input
          id="alias-model"
          label="Model"
          value={model}
          onChange={(event) => setModel(event.target.value)}
        />
        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button
            data-testid="alias-save"
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

export { AliasModal };
