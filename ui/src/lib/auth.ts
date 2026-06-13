import { apiFetch } from "@/lib/api";
import type { User } from "@/stores/user";

export type AuthMode = "password" | "oidc" | "both";

export interface AuthStatusResult {
  auth_mode: AuthMode;
}

export interface LoginResult {
  token: string;
  user: User;
}

/**
 * LoginError carries the HTTP status and (for 429 lockouts) the retry_after
 * seconds from the `{error:{message,retry_after,reset_hint}}` envelope. We read
 * these sibling fields here — NOT in the frozen lib/api.ts — by performing the
 * POST directly so the login page can drive its rate-limit countdown.
 */
export class LoginError extends Error {
  constructor(
    message: string,
    public status: number,
    public retryAfter?: number
  ) {
    super(message);
    this.name = "LoginError";
  }
}

interface LoginEnvelope {
  data?: { token?: string; user?: User } | null;
  error?: {
    message?: string;
    retry_after?: number;
    reset_hint?: string;
  } | null;
}

/**
 * getAuthStatus reads GET /api/auth/status, which (per the real Go contract,
 * internal/admin/auth.go:177-179) returns only `{ auth_mode }`. On any error it
 * degrades gracefully to password mode (mirrors 9router login/page.js:50-56).
 */
export async function getAuthStatus(): Promise<AuthStatusResult> {
  try {
    const data = await apiFetch<AuthStatusResult>("/api/auth/status");
    const mode = data?.auth_mode;
    if (mode === "oidc" || mode === "both" || mode === "password") {
      return { auth_mode: mode };
    }
    return { auth_mode: "password" };
  } catch {
    return { auth_mode: "password" };
  }
}

/**
 * loginWithPassword POSTs /api/auth/login. It does its own fetch (instead of
 * apiFetch) so the 429 lockout's `retry_after` sibling field — which the frozen
 * apiFetch discards — reaches the caller via LoginError.retryAfter.
 */
export async function loginWithPassword(
  username: string,
  password: string
): Promise<LoginResult> {
  const response = await fetch(`${window.location.origin}/api/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ username, password }),
  });

  const text = await response.text();
  let envelope: LoginEnvelope = {};
  if (text) {
    try {
      envelope = JSON.parse(text) as LoginEnvelope;
    } catch {
      throw new LoginError(`non-JSON response: ${text}`, response.status);
    }
  }

  if (!response.ok || envelope.error) {
    const message = envelope.error?.message ?? `HTTP ${response.status}`;
    throw new LoginError(message, response.status, envelope.error?.retry_after);
  }

  const token = envelope.data?.token;
  const user = envelope.data?.user;
  if (!token || !user) {
    throw new LoginError("malformed login response", response.status);
  }
  return { token, user };
}

/** logout POSTs /api/auth/logout; benign errors are ignored (best-effort). */
export async function logout(): Promise<void> {
  try {
    await apiFetch("/api/auth/logout", { method: "POST" });
  } catch {
    // Session may already be gone server-side — clearing local state still proceeds.
  }
}

/** startOidc hands the browser to the Go-driven OIDC start endpoint (302 to IdP). */
export function startOidc(): void {
  window.location.href = "/api/auth/oidc/start";
}

export interface OAuthCallbackPayload {
  code?: string;
  state?: string;
  error?: string;
  error_description?: string;
}

const OAUTH_CHANNEL = "oauth_callback";
// Origins permitted to receive the postMessage relay: this app + the 9router
// CLI loopback that opened the popup (ports 9router/src/app/callback/page.js).
const OAUTH_TARGET_ORIGINS = [window.location.origin, "http://localhost:1455"];

/**
 * relayOAuthCallback delivers a provider-OAuth popup result to its opener via
 * three channels (postMessage to an origin allowlist, BroadcastChannel, and
 * localStorage), per plan §1.3 / 9router callback/page.js:44-83.
 */
export function relayOAuthCallback(payload: OAuthCallbackPayload): void {
  const message = { type: OAUTH_CHANNEL, ...payload };

  if (typeof window !== "undefined" && window.opener) {
    for (const origin of OAUTH_TARGET_ORIGINS) {
      try {
        window.opener.postMessage(message, origin);
      } catch {
        // Cross-origin opener may reject a given target origin — try the rest.
      }
    }
  }

  try {
    const channel = new BroadcastChannel(OAUTH_CHANNEL);
    channel.postMessage(payload);
    channel.close();
  } catch {
    // BroadcastChannel unavailable — localStorage fallback still fires below.
  }

  try {
    localStorage.setItem(OAUTH_CHANNEL, JSON.stringify(payload));
  } catch {
    // Storage may be blocked; the other channels already attempted delivery.
  }
}
