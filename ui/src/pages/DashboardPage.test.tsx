import { render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DashboardPage } from "./DashboardPage";

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
    ...init
  });
}

function stubFetch(routes: Record<string, Response>) {
  const fetch = vi.fn(async (input: RequestInfo | URL) => {
    const path = String(input);
    return routes[path] ?? jsonResponse({ error: `missing route ${path}` }, { status: 404 });
  });
  vi.stubGlobal("fetch", fetch);
  return fetch;
}

const emptyList = { data: [] };
const emptyUsageList = { object: "list", data: [], limit: 0, offset: 0 };

describe("DashboardPage", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("loads overview metrics from real dashboard API contracts", async () => {
    const fetch = stubFetch({
      "/api/connections": jsonResponse({
        data: [
          { ID: "conn-1", Provider: "openai", IsActive: true },
          { ID: "conn-2", Provider: "anthropic", IsActive: true },
          { ID: "conn-3", Provider: "gemini", IsActive: true },
          { ID: "conn-4", Provider: "ollama", IsActive: false }
        ]
      }),
      "/api/usage/summary": jsonResponse({
        request_count: 7,
        total_tokens: 43210,
        total_cost_usd: 1.234
      }),
      "/api/logs": jsonResponse({
        object: "list",
        data: [
          { id: 1, request_id: "req-ok", provider: "openai", model: "gpt-4o", status_code: 200 },
          { id: 2, request_id: "req-failed", provider: "anthropic", model: "claude", status_code: 503 }
        ],
        limit: 0,
        offset: 0
      }),
      "/api/combos": jsonResponse({
        data: [
          { ID: "combo-1", Name: "balanced-chat", Steps: [], IsActive: true },
          { ID: "combo-2", Name: "archived-chat", Steps: [], IsActive: false }
        ]
      }),
      "/api/mcp/instances": jsonResponse({
        data: [
          { ID: "mcp-1", Name: "docs", IsActive: true, HealthStatus: "healthy", ToolManifest: { tools: [{ name: "search" }] } },
          { ID: "mcp-2", Name: "jira", IsActive: true, HealthStatus: "auth required", ToolManifest: { tools: [] } }
        ]
      })
    });

    render(<DashboardPage />);

    expect(await screen.findByText("43.2k")).toBeInTheDocument();
    expect(screen.getByText("$1.23 tracked cost")).toBeInTheDocument();

    const activeProviders = screen.getByText("Active providers").closest("article");
    expect(activeProviders).not.toBeNull();
    expect(within(activeProviders as HTMLElement).getByText("3")).toBeInTheDocument();

    const failedLogs = screen.getByText("Failed logs").closest("article");
    expect(failedLogs).not.toBeNull();
    expect(within(failedLogs as HTMLElement).getByText("1")).toBeInTheDocument();

    expect(screen.getByText("balanced-chat")).toBeInTheDocument();
    expect(screen.getByText("docs")).toBeInTheDocument();

    await waitFor(() => {
      expect(fetch.mock.calls.map(([path]) => path)).toEqual(
        expect.arrayContaining([
          "/api/connections",
          "/api/usage/summary",
          "/api/logs",
          "/api/combos",
          "/api/mcp/instances"
        ])
      );
    });
  });

  it("shows loading while overview requests are pending", () => {
    vi.stubGlobal("fetch", vi.fn(() => new Promise<Response>(() => undefined)));

    render(<DashboardPage />);

    expect(screen.getByRole("status")).toHaveTextContent("Loading dashboard data");
  });

  it("shows an empty state when every overview source is empty", async () => {
    stubFetch({
      "/api/connections": jsonResponse(emptyList),
      "/api/usage/summary": jsonResponse({ request_count: 0, total_tokens: 0, total_cost_usd: 0 }),
      "/api/logs": jsonResponse(emptyUsageList),
      "/api/combos": jsonResponse(emptyList),
      "/api/mcp/instances": jsonResponse(emptyList)
    });

    render(<DashboardPage />);

    expect(await screen.findByText("No overview data yet")).toBeInTheDocument();
  });

  it("shows an error state when overview data cannot load", async () => {
    stubFetch({
      "/api/connections": jsonResponse({ error: "connections unavailable" }, { status: 500 }),
      "/api/usage/summary": jsonResponse({ request_count: 0, total_tokens: 0, total_cost_usd: 0 }),
      "/api/logs": jsonResponse(emptyUsageList),
      "/api/combos": jsonResponse(emptyList),
      "/api/mcp/instances": jsonResponse(emptyList)
    });

    render(<DashboardPage />);

    expect(await screen.findByText("Dashboard data unavailable")).toBeInTheDocument();
    expect(screen.getByText("connections unavailable")).toBeInTheDocument();
  });

  it("shows auth-expired state when the control plane rejects overview requests", async () => {
    stubFetch({
      "/api/connections": jsonResponse({ error: "control-plane auth required" }, { status: 401 }),
      "/api/usage/summary": jsonResponse({ request_count: 0, total_tokens: 0, total_cost_usd: 0 }),
      "/api/logs": jsonResponse(emptyUsageList),
      "/api/combos": jsonResponse(emptyList),
      "/api/mcp/instances": jsonResponse(emptyList)
    });

    render(<DashboardPage />);

    expect(await screen.findByText("Session expired")).toBeInTheDocument();
    expect(screen.getByText("control-plane auth required")).toBeInTheDocument();
  });
});
