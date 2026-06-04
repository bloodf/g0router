import { afterEach, describe, expect, it, vi } from "vitest";
import {
  ApiError,
  apiFetch,
  getApiKeysPath,
  getAliasesPath,
  getCombosPath,
  getConnectionsPath,
  getLogsPath,
  getMcpAccountsPath,
  getMcpClientsPath,
  getMcpServersPath,
  getMcpToolsPath,
  getProviderModelsPath,
  getProvidersPath,
  getPricingPath,
  getQuotaPath,
  getSettingsPath,
  getUsagePath,
  getUsageSummaryPath,
  isAuthExpiredError,
  listProviders
} from "./api";

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
    ...init
  });
}

describe("api path helpers", () => {
  it("matches the management API contract", () => {
    expect(getProvidersPath()).toBe("/api/providers");
    expect(getProviderModelsPath("github/copilot")).toBe("/api/providers/github%2Fcopilot/models");
    expect(getConnectionsPath()).toBe("/api/connections");
    expect(getSettingsPath()).toBe("/api/settings");
    expect(getApiKeysPath()).toBe("/api/keys");
    expect(getAliasesPath()).toBe("/api/aliases");
    expect(getPricingPath()).toBe("/api/pricing");
    expect(getCombosPath()).toBe("/api/combos");
    expect(getUsagePath()).toBe("/api/usage");
    expect(getUsageSummaryPath()).toBe("/api/usage/summary");
    expect(getQuotaPath("openai")).toBe("/api/usage/quota/openai");
    expect(getLogsPath()).toBe("/api/logs");
    expect(getMcpClientsPath()).toBe("/api/mcp/clients");
    expect(getMcpServersPath()).toBe("/api/mcp/instances");
    expect(getMcpAccountsPath("inst 1")).toBe("/api/mcp/instances/inst%201/accounts");
    expect(getMcpToolsPath()).toBe("/api/mcp/tools");
  });
});

describe("apiFetch", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("sends JSON bodies and parses JSON responses", async () => {
    const fetch = vi.fn(async () => jsonResponse({ RequireAPIKey: false }));
    vi.stubGlobal("fetch", fetch);

    const response = await apiFetch<{ RequireAPIKey: boolean }>(getSettingsPath(), {
      method: "PUT",
      body: { RequireAPIKey: false }
    });

    expect(response.RequireAPIKey).toBe(false);
    expect(fetch).toHaveBeenCalledWith(
      "/api/settings",
      expect.objectContaining({
        body: "{\"RequireAPIKey\":false}",
        credentials: "same-origin",
        headers: expect.objectContaining({ "Content-Type": "application/json" }),
        method: "PUT"
      })
    );
  });

  it("unwraps list endpoints through typed helpers", async () => {
    const fetch = vi.fn(async () =>
      jsonResponse({
        data: [
          {
            id: "openai",
            auth_types: ["api_key"],
            inference: true,
            public_status: "supported"
          }
        ]
      })
    );
    vi.stubGlobal("fetch", fetch);

    const providers = await listProviders();

    expect(providers).toHaveLength(1);
    expect(providers[0].id).toBe("openai");
    expect(fetch).toHaveBeenCalledWith("/api/providers", expect.any(Object));
  });

  it("sends the saved control-plane key as a bearer token", async () => {
    const storage = stubLocalStorage();
    storage.setItem("g0router.controlPlaneKey", "g0r_test_secret");
    const fetch = vi.fn(async () => jsonResponse({ RequireAPIKey: true }));
    vi.stubGlobal("fetch", fetch);

    await apiFetch(getSettingsPath());

    expect(fetch).toHaveBeenCalledWith(
      "/api/settings",
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: "Bearer g0r_test_secret"
        })
      })
    );
  });

  it("does not override explicit auth headers", async () => {
    const storage = stubLocalStorage();
    storage.setItem("g0router.controlPlaneKey", "g0r_saved");
    const fetch = vi.fn(async () => jsonResponse({ ok: true }));
    vi.stubGlobal("fetch", fetch);

    await apiFetch("/api/settings", {
      headers: { Authorization: "Bearer explicit" }
    });

    expect(fetch).toHaveBeenCalledWith(
      "/api/settings",
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: "Bearer explicit"
        })
      })
    );
  });

  it("returns undefined for empty 204 responses", async () => {
    const fetch = vi.fn(async () => new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetch);

    await expect(apiFetch<void>("/api/keys/key-1", { method: "DELETE" })).resolves.toBeUndefined();
  });

  it("marks 401 and 403 responses as auth-expired API errors", async () => {
    const fetch = vi.fn(async () => jsonResponse({ error: "control-plane auth required" }, { status: 401 }));
    vi.stubGlobal("fetch", fetch);

    try {
      await apiFetch("/api/settings");
      throw new Error("expected apiFetch to fail");
    } catch (error) {
      expect(error).toBeInstanceOf(ApiError);
      expect(isAuthExpiredError(error)).toBe(true);
      expect(error).toMatchObject({
        authExpired: true,
        message: "control-plane auth required",
        status: 401
      });
    }
  });
});

function stubLocalStorage() {
  const values = new Map<string, string>();
  const storage = {
    getItem: vi.fn((key: string) => values.get(key) ?? null),
    setItem: vi.fn((key: string, value: string) => {
      values.set(key, value);
    }),
    removeItem: vi.fn((key: string) => {
      values.delete(key);
    })
  };
  vi.stubGlobal("localStorage", storage);
  return storage;
}
