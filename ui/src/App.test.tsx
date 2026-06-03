import { fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import App from "./App";
import {
  getApiKeysPath,
  getCombosPath,
  getConnectionsPath,
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

    for (const label of ["Dashboard", "Endpoint", "Providers", "Usage", "Quota", "Combos", "MCP", "Settings"]) {
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
    expect(screen.queryByText("Usage analytics")).not.toBeInTheDocument();
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
});

describe("api helpers", () => {
  it("exposes typed management API paths", () => {
    expect(getConnectionsPath()).toBe("/api/connections");
    expect(getApiKeysPath()).toBe("/api/keys");
    expect(getUsagePath()).toBe("/api/usage");
    expect(getQuotaPath("openai")).toBe("/api/usage/quota/openai");
    expect(getCombosPath()).toBe("/api/combos");
    expect(getMcpServersPath()).toBe("/api/mcp/instances");
    expect(getSettingsPath()).toBe("/api/settings");
  });
});

function stubDashboardFetch() {
  const fetch = vi.fn(async (input: RequestInfo | URL) => {
    const path = String(input);
    switch (path) {
      case "/api/connections":
      case "/api/providers":
      case "/api/combos":
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
