import * as React from "react";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { apiFetch } from "@/lib/api";
import { useNotificationStore } from "@/stores/notification";

export interface KiroAuthModalProps {
  open: boolean;
  onClose: () => void;
  onCreated?: () => void;
}

// KiroAuthModal (PAR-UI-059) — paste a Kiro access token to create a kiro
// connection (port of 9router KiroAuthModal.js).
function KiroAuthModal({ open, onClose, onCreated }: KiroAuthModalProps) {
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
          provider_id: "kiro",
          name: "Kiro",
          kind: "api_key",
          secret: token,
        }),
      });
      pushToast({ message: "Connected Kiro" });
      onCreated?.();
      onClose();
    } catch {
      pushToast({ message: "Failed to connect Kiro" });
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open={open} onClose={onClose} title="Connect Kiro">
      <div className="flex flex-col gap-4">
        <p className="text-sm text-muted-foreground">
          Paste your Kiro access token to authorize the Kiro provider.
        </p>
        <Input
          label="Access token"
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

export { KiroAuthModal };
