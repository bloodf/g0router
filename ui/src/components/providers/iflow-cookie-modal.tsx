import * as React from "react";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { apiFetch } from "@/lib/api";
import { useNotificationStore } from "@/stores/notification";

export interface IFlowCookieModalProps {
  open: boolean;
  onClose: () => void;
  onCreated?: () => void;
}

// IFlowCookieModal (PAR-UI-060) — paste an iFlow session cookie to create an
// iflow connection (port of 9router IFlowCookieModal.js).
function IFlowCookieModal({ open, onClose, onCreated }: IFlowCookieModalProps) {
  const pushToast = useNotificationStore((state) => state.push);
  const [cookie, setCookie] = React.useState("");
  const [busy, setBusy] = React.useState(false);

  React.useEffect(() => {
    if (open) setCookie("");
  }, [open]);

  async function create() {
    setBusy(true);
    try {
      await apiFetch("/api/connections", {
        method: "POST",
        body: JSON.stringify({
          provider_id: "iflow",
          name: "iFlow",
          kind: "api_key",
          secret: cookie,
        }),
      });
      pushToast({ message: "Connected iFlow" });
      onCreated?.();
      onClose();
    } catch {
      pushToast({ message: "Failed to connect iFlow" });
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open={open} onClose={onClose} title="Connect iFlow">
      <div className="flex flex-col gap-4">
        <p className="text-sm text-muted-foreground">
          Paste your iFlow session cookie to authorize the iFlow provider.
        </p>
        <Input
          label="Session cookie"
          type="password"
          value={cookie}
          onChange={(event) => setCookie(event.target.value)}
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

export { IFlowCookieModal };
