import * as React from "react";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { apiFetch } from "@/lib/api";
import {
  subscribeOAuthPopup,
  openOAuthPopup,
  type OAuthCallbackPayload,
} from "@/lib/oauth-popup";
import { useNotificationStore } from "@/stores/notification";
import type { Provider } from "@/lib/types";

export interface OAuthModalProps {
  open: boolean;
  provider: Provider | null;
  onClose: () => void;
  onConnected?: () => void;
}

// OAuthModal (PAR-UI-051) drives the provider OAuth popup flow: it starts the
// provider authorize URL, opens it in a popup, subscribes to the w6-c relay
// (oauth-popup.ts, plan §1.4), and finalizes the connection on the relayed code.
function OAuthModal({ open, provider, onClose, onConnected }: OAuthModalProps) {
  const pushToast = useNotificationStore((state) => state.push);
  const [busy, setBusy] = React.useState(false);
  const popupRef = React.useRef<Window | null>(null);
  const unsubscribeRef = React.useRef<(() => void) | null>(null);

  const finalize = React.useCallback(
    async (payload: OAuthCallbackPayload) => {
      if (!provider) return;
      popupRef.current?.close();
      if (payload.error) {
        pushToast({
          message: payload.error_description || payload.error,
        });
        setBusy(false);
        return;
      }
      try {
        await apiFetch(`/api/oauth/${provider.id}/callback`, {
          method: "POST",
          body: JSON.stringify({ code: payload.code, state: payload.state }),
        });
        pushToast({ message: `Connected ${provider.display_name}` });
        onConnected?.();
        onClose();
      } catch {
        pushToast({ message: "Failed to finalize the connection" });
      } finally {
        setBusy(false);
      }
    },
    [provider, pushToast, onConnected, onClose]
  );

  React.useEffect(() => {
    if (!open) {
      unsubscribeRef.current?.();
      unsubscribeRef.current = null;
      return;
    }
    // Subscribe BEFORE opening the popup (plan §1.4 step 2).
    unsubscribeRef.current = subscribeOAuthPopup((payload) => {
      void finalize(payload);
    });
    return () => {
      unsubscribeRef.current?.();
      unsubscribeRef.current = null;
    };
  }, [open, finalize]);

  async function startFlow() {
    if (!provider) return;
    setBusy(true);
    try {
      const data = await apiFetch<{ auth_url?: string }>(
        `/api/oauth/${provider.id}/start`
      );
      const url = data?.auth_url ?? "about:blank";
      popupRef.current = openOAuthPopup(url);
    } catch {
      pushToast({ message: "Failed to start authorization" });
      setBusy(false);
    }
  }

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={provider ? `Connect ${provider.display_name}` : "Connect provider"}
    >
      <div className="flex flex-col gap-4">
        <p className="text-sm text-muted-foreground">
          Authorize {provider?.display_name ?? "this provider"} in the popup
          window. The connection is created once authorization completes.
        </p>
        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button variant="primary" loading={busy} onClick={startFlow}>
            Authorize
          </Button>
        </div>
      </div>
    </Modal>
  );
}

export { OAuthModal };
