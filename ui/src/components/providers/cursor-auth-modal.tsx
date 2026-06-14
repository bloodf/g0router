import * as React from "react";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { apiFetch } from "@/lib/api";
import { useNotificationStore } from "@/stores/notification";

export interface CursorAuthModalProps {
  open: boolean;
  onClose: () => void;
  onCreated?: () => void;
}

// CursorAuthModal (PAR-UI-058) — paste a Cursor session token to create a
// cursor-agent connection (port of 9router CursorAuthModal.js).
function CursorAuthModal({ open, onClose, onCreated }: CursorAuthModalProps) {
  const pushToast = useNotificationStore((state) => state.push);
  const [token, setToken] = React.useState("");
  const [busy, setBusy] = React.useState(false);

  React.useEffect(() => {
    if (open) setToken("");
  }, [open]);

  async function create() {
    setBusy(true);
    try {
      await apiFetch("/api/connections", {
        method: "POST",
        body: JSON.stringify({
          provider_id: "cursor-agent",
          name: "Cursor",
          kind: "api_key",
          secret: token,
        }),
      });
      pushToast({ message: "Connected Cursor" });
      onCreated?.();
      onClose();
    } catch {
      pushToast({ message: "Failed to connect Cursor" });
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open={open} onClose={onClose} title="Connect Cursor">
      <div className="flex flex-col gap-4">
        <p className="text-sm text-muted-foreground">
          Paste your Cursor session token to authorize the cursor-agent provider.
        </p>
        <Input
          label="Session token"
          type="password"
          value={token}
          onChange={(event) => setToken(event.target.value)}
        />
        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button variant="primary" loading={busy} onClick={create}>
            Connect
          </Button>
        </div>
      </div>
    </Modal>
  );
}

export { CursorAuthModal };
