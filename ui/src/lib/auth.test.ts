import { describe, it, expect, beforeEach, vi } from "vitest";

// Plain-node unit tests (no jsdom) — hand-stub the globals lib/auth.ts touches,
// following the w6-a stores/theme.test.ts precedent.

function stubWindow(origin = "http://localhost:3000") {
  (globalThis as Record<string, unknown>).window = {
    location: { origin, href: "" },
  };
}

function stubFetch(impl: (url: string, init?: RequestInit) => Promise<Response>) {
  (globalThis as Record<string, unknown>).fetch = vi.fn(impl) as unknown as typeof fetch;
}

function jsonResponse(body: unknown, ok = true, status = 200): Response {
  return {
    ok,
    status,
    text: async () => JSON.stringify(body),
  } as unknown as Response;
}

describe("getAuthStatus", () => {
  beforeEach(() => {
    vi.resetModules();
    stubWindow();
  });

  it("returns the parsed auth_mode from the status endpoint", async () => {
    stubFetch(async () => jsonResponse({ data: { auth_mode: "both" } }));
    const { getAuthStatus } = await import("./auth");
    const status = await getAuthStatus();
    expect(status.auth_mode).toBe("both");
  });

  it("defaults to password when the status fetch fails", async () => {
    stubFetch(async () => {
      throw new Error("network down");
    });
    const { getAuthStatus } = await import("./auth");
    const status = await getAuthStatus();
    expect(status.auth_mode).toBe("password");
  });
});

describe("loginWithPassword", () => {
  beforeEach(() => {
    vi.resetModules();
    stubWindow();
  });

  it("returns token and user on success", async () => {
    stubFetch(async () =>
      jsonResponse({
        data: { token: "tok-1", user: { id: "u1", username: "admin" } },
      })
    );
    const { loginWithPassword } = await import("./auth");
    const result = await loginWithPassword("admin", "123456");
    expect(result.token).toBe("tok-1");
    expect(result.user.username).toBe("admin");
  });

  it("throws a LoginError carrying retryAfter on a 429 lockout", async () => {
    stubFetch(async () =>
      jsonResponse(
        {
          data: null,
          error: {
            message: "Too many failed attempts. Try again in 30s.",
            retry_after: 30,
            reset_hint: "reset",
          },
        },
        false,
        429
      )
    );
    const { loginWithPassword } = await import("./auth");
    await expect(loginWithPassword("admin", "bad")).rejects.toMatchObject({
      status: 429,
      retryAfter: 30,
    });
  });

  it("throws a LoginError with status 401 on invalid credentials", async () => {
    stubFetch(async () =>
      jsonResponse(
        { data: null, error: { message: "invalid username or password" } },
        false,
        401
      )
    );
    const { loginWithPassword } = await import("./auth");
    await expect(loginWithPassword("admin", "bad")).rejects.toMatchObject({
      status: 401,
    });
  });
});

describe("startOidc", () => {
  beforeEach(() => {
    vi.resetModules();
    stubWindow();
  });

  it("navigates the browser to the Go OIDC start endpoint", async () => {
    const { startOidc } = await import("./auth");
    startOidc();
    expect((globalThis as Record<string, { location: { href: string } }>).window.location.href).toBe(
      "/api/auth/oidc/start"
    );
  });
});

describe("relayOAuthCallback", () => {
  beforeEach(() => {
    vi.resetModules();
    stubWindow();
  });

  it("broadcasts on the oauth_callback channel and writes localStorage", () => {
    const posted: unknown[] = [];
    class FakeBroadcastChannel {
      name: string;
      constructor(name: string) {
        this.name = name;
      }
      postMessage(data: unknown) {
        posted.push({ channel: this.name, data });
      }
      close() {}
    }
    (globalThis as Record<string, unknown>).BroadcastChannel =
      FakeBroadcastChannel as unknown as typeof BroadcastChannel;

    const stored: Record<string, string> = {};
    (globalThis as Record<string, unknown>).localStorage = {
      getItem: (k: string) => stored[k] ?? null,
      setItem: (k: string, v: string) => {
        stored[k] = v;
      },
      removeItem: (k: string) => {
        delete stored[k];
      },
    } as Storage;

    return import("./auth").then(({ relayOAuthCallback }) => {
      relayOAuthCallback({ code: "abc", state: "xyz" });

      expect(posted).toHaveLength(1);
      expect(posted[0]).toMatchObject({
        channel: "oauth_callback",
        data: { code: "abc", state: "xyz" },
      });
      expect(stored["oauth_callback"]).toContain("abc");
    });
  });
});
