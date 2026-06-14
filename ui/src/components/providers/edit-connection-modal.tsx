import * as React from "react";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Toggle } from "@/components/ui/toggle";
import { apiFetch } from "@/lib/api";
import { useNotificationStore } from "@/stores/notification";
import type { Connection } from "@/lib/types";

export interface EditConnectionModalProps {
  open: boolean;
  connection: Connection | null;
  onClose: () => void;
  onSaved?: () => void;
}

// EditConnectionModal (PAR-UI-052) edits a connection's name and active state and
// optionally rotates its secret via PUT /api/connections/{id}.
function EditConnectionModal({
  open,
  connection,
  onClose,
  onSaved,
}: EditConnectionModalProps) {
  const pushToast = useNotificationStore((state) => state.push);
  const [name, setName] = React.useState("");
  const [secret, setSecret] = React.useState("");
  const [isActive, setIsActive] = React.useState(true);
  const [busy, setBusy] = React.useState(false);

  React.useEffect(() => {
    if (connection) {
      setName(connection.name);
      setIsActive(connection.is_active);
      setSecret("");
    }
  }, [connection]);

  async function save() {
    if (!connection) return;
    setBusy(true);
    try {
      await apiFetch(`/api/connections/${connection.id}`, {
        method: "PUT",
        body: JSON.stringify({
          provider_id: connection.provider,
          name,
          kind: connection.auth_type,
          is_active: isActive,
          ...(secret ? { secret } : {}),
        }),
      });
      pushToast({ message: "Connection updated" });
      onSaved?.();
      onClose();
    } catch {
      pushToast({ message: "Failed to update the connection" });
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open={open} onClose={onClose} title="Edit connection">
      <div className="flex flex-col gap-4">
        <Input
          label="Name"
          value={name}
          onChange={(event) => setName(event.target.value)}
        />
        <Input
          label="Rotate secret (leave blank to keep current)"
          type="password"
          value={secret}
          onChange={(event) => setSecret(event.target.value)}
        />
        <label className="flex items-center justify-between text-sm text-foreground">
          Active
          <Toggle checked={isActive} onCheckedChange={setIsActive} />
        </label>
        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button variant="primary" loading={busy} onClick={save}>
            Save
          </Button>
        </div>
      </div>
    </Modal>
  );
}

export { EditConnectionModal };
