import { fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import App from "./App";
import {
  getApiKeysPath,
  getAliasesPath,
  getCombosPath,
  getConnectionsPath,
  getPricingPath,
  getMcpServersPath,
  getQuotaPath,
  getSettingsPath,
  getUsagePath
} from "./api";

describe("App", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders all control-plane navigation destinations", () => {
    stubDashboardFetch();

    render(<App />);

    expect(screen.getByRole("heading", { name: "g0router" })).toBeInTheDocument();
    const primaryNav = screen.getByRole("navigation", { name: "Primary" });
    expect(primaryNav).toBeInTheDocument();

    for (const label of [
      "Dashboard",
      "Endpoint Setup",
      "API Keys",
      "Providers",
      "Connections/Auth",
      "Aliases",
      "Models",
      "Pricing",
      "Usage",
      "Logs",
      "Quota",
      "Combos/Routing",
      "MCP",
      "MCP Instances",
      "MCP Accounts",
      "MCP Tools",
      "Settings",
      "Settings/Security",
      "Diagnostics"
    ]) {
      expect(within(primaryNav).getByRole("button", { name: label })).toBeInTheDocument();
    }
  });

  it("does not mount every management page on the dashboard view", async () => {
    stubDashboardFetch();

    render(<App />);

    expect(screen.getByRole("heading", { name: "Gateway overview" })).toBeInTheDocument();
    expect(await screen.findByText("No overview data yet")).toBeInTheDocument();
    expect(screen.queryByText("Endpoint controls")).not.toBeInTheDocument();
    expect(screen.queryByText("Provider connections")).not.toBeInTheDocument();
    expect(screen.queryByText("Connections and auth")).not.toBeInTheDocument();
    expect(screen.queryByText("Usage analytics")).not.toBeInTheDocument();
    expect(screen.queryByText("Request logs")).not.toBeInTheDocument();
    expect(screen.queryByText("Quota monitor")).not.toBeInTheDocument();
    expect(screen.queryByText("Combo routing")).not.toBeInTheDocument();
    expect(screen.queryByText("MCP gateway")).not.toBeInTheDocument();
    expect(screen.queryByText("Runtime settings")).not.toBeInTheDocument();
  });

  it("mounts only the selected management page after navigation", async () => {
    const fetch = stubDashboardFetch();

    render(<App />);

    fireEvent.click(within(screen.getByRole("navigation", { name: "Primary" })).getByRole("button", { name: "Providers" }));

    expect(screen.getByRole("heading", { name: "Providers" })).toBeInTheDocument();
    expect(await screen.findByText("No provider records")).toBeInTheDocument();
    expect(screen.queryByText("Endpoint controls")).not.toBeInTheDocument();
    expect(fetch).toHaveBeenCalledWith("/api/providers", expect.objectContaining({ credentials: "same-origin" }));
    expect(fetch).toHaveBeenCalledWith("/api/connections", expect.objectContaining({ credentials: "same-origin" }));
  });

  it("saves a control-plane key and retries API calls with bearer auth", async () => {
    const storage = stubLocalStorage();
    const fetch = stubDashboardFetch();

    render(<App />);

    fireEvent.change(screen.getByLabelText("Control-plane API key"), {
      target: { value: "g0r_dashboard_secret" }
    });
    fireEvent.click(screen.getByRole("button", { name: "Save key" }));

    await screen.findByText("No overview data yet");

    expect(storage.getItem("g0router.controlPlaneKey")).toBe("g0r_dashboard_secret");
    expect(fetch.mock.calls.some(([, options]) => {
      const headers = (options as RequestInit).headers as Record<string, string> | undefined;
      return headers?.Authorization === "Bearer g0r_dashboard_secret";
    })).toBe(true);
  });
});

describe("api helpers", () => {
  it("exposes typed management API paths", () => {
    expect(getConnectionsPath()).toBe("/api/connections");
    expect(getApiKeysPath()).toBe("/api/keys");
    expect(getAliasesPath()).toBe("/api/aliases");
    expect(getPricingPath()).toBe("/api/pricing");
    expect(getUsagePath()).toBe("/api/usage");
    expect(getQuotaPath("openai")).toBe("/api/usage/quota/openai");
    expect(getCombosPath()).toBe("/api/combos");
    expect(getMcpServersPath()).toBe("/api/mcp/instances");
    expect(getSettingsPath()).toBe("/api/settings");
  });
});

function stubDashboardFetch() {
  const fetch = vi.fn(async (input: RequestInfo | URL, _options?: RequestInit) => {
    const path = String(input);
    switch (path) {
      case "/api/connections":
      case "/api/providers":
      case "/api/combos":
      case "/api/aliases":
      case "/api/pricing":
      case "/api/mcp/instances":
      case "/api/mcp/clients":
      case "/api/mcp/tools":
        return jsonResponse({ data: [] });
      case "/api/usage":
      case "/api/logs":
        return jsonResponse({ object: "list", data: [], limit: 0, offset: 0 });
      case "/api/usage/summary":
        return jsonResponse({ request_count: 0, total_tokens: 0, total_cost_usd: 0 });
      case "/api/settings":
        return jsonResponse({
          RequireAPIKey: true,
          RTKEnabled: true,
          CavemanEnabled: false,
          CavemanLevel: "full",
          EnableRequestLogs: false,
          ProxyURL: "",
          DataDir: ""
        });
      case "/api/keys":
        return jsonResponse({ data: [] });
      default:
        if (path.startsWith("/api/mcp/instances/") && path.endsWith("/accounts")) {
          return jsonResponse({ data: [] });
        }
        if (path.startsWith("/api/usage/quota/")) {
          return jsonResponse({ Provider: decodeURIComponent(path.replace("/api/usage/quota/", "")), Limit: 0, Used: 0, Remaining: 0 });
        }
        return jsonResponse({ error: `missing ${path}` }, { status: 404 });
    }
  });
  vi.stubGlobal("fetch", fetch);
  return fetch;
}

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
    ...init
  });
}

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
