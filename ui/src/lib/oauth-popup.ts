// Opener-side listener for the w6-c OAuth popup relay (plan §1.4). The popup
// navigates to /callback, which calls relayOAuthCallback (lib/auth.ts) and
// delivers the result via three channels: BroadcastChannel, postMessage, and
// localStorage. This helper subscribes to all three WITHOUT importing or editing
// the frozen lib/auth.ts — it consumes the channel/origin constants by value.

const OAUTH_CHANNEL = "oauth_callback";

export interface OAuthCallbackPayload {
  code?: string;
  state?: string;
  error?: string;
  error_description?: string;
}

interface WindowMessage {
  type?: string;
  code?: string;
  state?: string;
  error?: string;
  error_description?: string;
}

function pickPayload(raw: WindowMessage): OAuthCallbackPayload {
  return {
    code: raw.code,
    state: raw.state,
    error: raw.error,
    error_description: raw.error_description,
  };
}

/**
 * subscribeOAuthPopup wires the three relay channels and invokes `handler` at
 * most once with the received payload, then auto-unsubscribes. The returned
 * function unsubscribes explicitly (e.g. when a modal closes before delivery).
 */
export function subscribeOAuthPopup(
  handler: (payload: OAuthCallbackPayload) => void
): () => void {
  let handled = false;
  let channel: BroadcastChannel | null = null;

  function cleanup() {
    window.removeEventListener("message", onMessage);
    window.removeEventListener("storage", onStorage);
    if (channel) {
      channel.close();
      channel = null;
    }
  }

  function deliver(payload: OAuthCallbackPayload) {
    if (handled) return;
    handled = true;
    cleanup();
    handler(payload);
  }

  function onBroadcast(event: MessageEvent<OAuthCallbackPayload>) {
    deliver(event.data ?? {});
  }

  function onMessage(event: MessageEvent) {
    // Same-origin only (the sender also posts to http://localhost:1455, which we
    // deliberately ignore here, per plan §1.4).
    if (event.origin !== window.location.origin) return;
    const data = event.data as WindowMessage | undefined;
    if (!data || data.type !== OAUTH_CHANNEL) return;
    deliver(pickPayload(data));
  }

  function onStorage(event: StorageEvent) {
    if (event.key !== OAUTH_CHANNEL || !event.newValue) return;
    try {
      deliver(JSON.parse(event.newValue) as OAuthCallbackPayload);
    } catch {
      // Malformed payload — ignore; the other channels may still deliver.
    }
  }

  try {
    channel = new BroadcastChannel(OAUTH_CHANNEL);
    channel.addEventListener("message", onBroadcast);
  } catch {
    // BroadcastChannel unavailable — the message/storage channels still apply.
  }
  window.addEventListener("message", onMessage);
  window.addEventListener("storage", onStorage);

  return cleanup;
}

/** openOAuthPopup opens the provider authorize URL in a popup window. */
export function openOAuthPopup(url: string): Window | null {
  return window.open(url, "oauth", "width=600,height=720");
}
