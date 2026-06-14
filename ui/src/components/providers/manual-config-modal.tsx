import * as React from "react";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { apiFetch } from "@/lib/api";
import { useNotificationStore } from "@/stores/notification";
import type { Provider } from "@/lib/types";

export interface ManualConfigModalProps {
  open: boolean;
  provider: Provider | null;
  onClose: () => void;
  onCreated?: () => void;
}

// ManualConfigModal (PAR-UI-053) creates a connection from a pasted API key or
// manual token/config via POST /api/connections.
function ManualConfigModal({
  open,
  provider,
  onClose,
  onCreated,
}: ManualConfigModalProps) {
  const pushToast = useNotificationStore((state) => state.push);
  const [name, setName] = React.useState("");
  const [secret, setSecret] = React.useState("");
  const [busy, setBusy] = React.useState(false);

  React.useEffect(() => {
    if (open) {
      setName(provider ? `${provider.display_name} key` : "");
      setSecret("");
    }
  }, [open, provider]);

  async function create() {
    if (!provider) return;
    setBusy(true);
    try {
      await apiFetch("/api/connections", {
        method: "POST",
        body: JSON.stringify({
          provider_id: provider.id,
          name,
          kind: "api_key",
          secret,
        }),
      });
      pushToast({ message: `Added a connection for ${provider.display_name}` });
      onCreated?.();
      onClose();
    } catch {
      pushToast({ message: "Failed to add the connection" });
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={provider ? `Add ${provider.display_name} key` : "Add connection"}
    >
      <div className="flex flex-col gap-4">
        <Input
          label="Connection name"
          value={name}
          onChange={(event) => setName(event.target.value)}
        />
        <Input
          label="API key / token"
          type="password"
          value={secret}
          onChange={(event) => setSecret(event.target.value)}
        />
        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button variant="primary" loading={busy} onClick={create}>
            Add connection
          </Button>
        </div>
      </div>
    </Modal>
  );
}

export { ManualConfigModal };
