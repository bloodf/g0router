import * as React from "react";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { apiFetch } from "@/lib/api";
import { useNotificationStore } from "@/stores/notification";

export interface GitLabAuthModalProps {
  open: boolean;
  onClose: () => void;
  onCreated?: () => void;
}

// GitLabAuthModal (PAR-UI-062) — paste a GitLab personal access token to create
// a gitlab connection (port of 9router GitLabAuthModal.js).
function GitLabAuthModal({ open, onClose, onCreated }: GitLabAuthModalProps) {
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
          provider_id: "gitlab",
          name: "GitLab",
          kind: "api_key",
          secret: token,
        }),
      });
      pushToast({ message: "Connected GitLab" });
      onCreated?.();
      onClose();
    } catch {
      pushToast({ message: "Failed to connect GitLab" });
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open={open} onClose={onClose} title="Connect GitLab">
      <div className="flex flex-col gap-4">
        <p className="text-sm text-muted-foreground">
          Paste a GitLab personal access token to authorize the GitLab provider.
        </p>
        <Input
          label="Personal access token"
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

export { GitLabAuthModal };
