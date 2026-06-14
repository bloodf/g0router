import { describe, it, expect, beforeEach, vi } from "vitest";
import {
  subscribeOAuthPopup,
  type OAuthCallbackPayload,
} from "./oauth-popup";

// The w6-c relay delivers via three channels. oauth-popup.ts is the opener-side
// listener. These tests hand-stub BroadcastChannel / window / localStorage in
// plain node (the w6-a theme.test.ts precedent — no jsdom), then drive each
// channel and assert the handler fires at most once.

type MessageListener = (event: { origin: string; data: unknown }) => void;
type StorageListener = (event: { key: string | null; newValue: string | null }) => void;

interface Harness {
  emitBroadcast: (payload: OAuthCallbackPayload) => void;
  emitMessage: (origin: string, data: unknown) => void;
  emitStorage: (key: string | null, newValue: string | null) => void;
  channelClosed: () => boolean;
}

function setupHarness(origin = "https://app.test"): Harness {
  const messageListeners: MessageListener[] = [];
  const storageListeners: StorageListener[] = [];
  const broadcastListeners: Array<(ev: { data: OAuthCallbackPayload }) => void> = [];
  let closed = false;

  class FakeBroadcastChannel {
    onmessage: ((ev: { data: OAuthCallbackPayload }) => void) | null = null;
    constructor(public name: string) {}
    addEventListener(_type: string, cb: (ev: { data: OAuthCallbackPayload }) => void) {
      broadcastListeners.push(cb);
    }
    removeEventListener() {}
    postMessage() {}
    close() {
      closed = true;
    }
  }
  (globalThis as unknown as { BroadcastChannel: unknown }).BroadcastChannel =
    FakeBroadcastChannel as unknown;

  (globalThis as unknown as { window: unknown }).window = {
    location: { origin },
    addEventListener: (type: string, cb: unknown) => {
      if (type === "message") messageListeners.push(cb as MessageListener);
      if (type === "storage") storageListeners.push(cb as StorageListener);
    },
    removeEventListener: () => {},
  };

  return {
    emitBroadcast: (payload) => {
      for (const cb of broadcastListeners) cb({ data: payload });
    },
    emitMessage: (msgOrigin, data) => {
      for (const cb of messageListeners) cb({ origin: msgOrigin, data });
    },
    emitStorage: (key, newValue) => {
      for (const cb of storageListeners) cb({ key, newValue });
    },
    channelClosed: () => closed,
  };
}

describe("subscribeOAuthPopup", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("fires the handler once for a BroadcastChannel message", () => {
    const h = setupHarness();
    const handler = vi.fn();
    subscribeOAuthPopup(handler);
    h.emitBroadcast({ code: "abc", state: "xyz" });
    expect(handler).toHaveBeenCalledTimes(1);
    expect(handler).toHaveBeenCalledWith({ code: "abc", state: "xyz" });
  });

  it("fires the handler for a same-origin window message", () => {
    const h = setupHarness("https://app.test");
    const handler = vi.fn();
    subscribeOAuthPopup(handler);
    h.emitMessage("https://app.test", { type: "oauth_callback", code: "abc", state: "xyz" });
    expect(handler).toHaveBeenCalledTimes(1);
    expect(handler).toHaveBeenCalledWith(
      expect.objectContaining({ code: "abc", state: "xyz" })
    );
  });

  it("ignores a cross-origin window message", () => {
    const h = setupHarness("https://app.test");
    const handler = vi.fn();
    subscribeOAuthPopup(handler);
    h.emitMessage("https://evil.test", { type: "oauth_callback", code: "abc" });
    expect(handler).not.toHaveBeenCalled();
  });

  it("ignores a same-origin message without the oauth_callback type", () => {
    const h = setupHarness("https://app.test");
    const handler = vi.fn();
    subscribeOAuthPopup(handler);
    h.emitMessage("https://app.test", { type: "something-else", code: "abc" });
    expect(handler).not.toHaveBeenCalled();
  });

  it("fires the handler for a storage event under the oauth_callback key", () => {
    const h = setupHarness();
    const handler = vi.fn();
    subscribeOAuthPopup(handler);
    h.emitStorage("oauth_callback", JSON.stringify({ code: "abc", state: "xyz" }));
    expect(handler).toHaveBeenCalledTimes(1);
    expect(handler).toHaveBeenCalledWith(
      expect.objectContaining({ code: "abc", state: "xyz" })
    );
  });

  it("fires the handler only once when two channels deliver the same payload", () => {
    const h = setupHarness();
    const handler = vi.fn();
    subscribeOAuthPopup(handler);
    h.emitBroadcast({ code: "abc", state: "xyz" });
    h.emitStorage("oauth_callback", JSON.stringify({ code: "abc", state: "xyz" }));
    expect(handler).toHaveBeenCalledTimes(1);
  });
});
