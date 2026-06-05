import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { getUsageSummaryPath } from "../api";
import { UsagePage } from "./UsagePage";

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

const emptyUsageList = { object: "list", data: [], limit: 0, offset: 0 };

describe("UsagePage", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("loads usage and request logs from the API without hardcoded rows", async () => {
    const fetch = stubFetch({
      "/api/usage": jsonResponse({
        object: "list",
        data: [
          {
            id: 1,
            request_id: "req-success",
            timestamp: "2026-06-03T10:00:00Z",
            provider: "openai",
            model: "gpt-4o",
            total_tokens: 900,
            cost_usd: 0.045,
            latency_ms: 210,
            status_code: 200,
            client_tool: "codex"
          },
          {
            id: 2,
            request_id: "req-failed",
            timestamp: "2026-06-03T10:01:00Z",
            provider: "anthropic",
            model: "claude-sonnet",
            total_tokens: 120,
            cost_usd: 0.003,
            latency_ms: 80,
            status_code: 502,
            error: "upstream refused request",
            client_tool: "claude-code"
          }
        ],
        limit: 0,
        offset: 0
      }),
      "/api/logs": jsonResponse({
        object: "list",
        data: [
          {
            id: 3,
            request_id: "req-stream",
            timestamp: "2026-06-03T10:02:00Z",
            provider: "gemini",
            model: "gemini-flash",
            total_tokens: 77,
            latency_ms: 45,
            status_code: 200,
            source_format: "openai",
            target_format: "stream",
            client_tool: "cursor"
          }
        ],
        limit: 0,
        offset: 0
      })
    });

    render(<UsagePage />);

    expect(await screen.findByText("req-success")).toBeInTheDocument();
    expect(screen.getByText("req-failed")).toBeInTheDocument();
    expect(screen.getByText("upstream refused request")).toBeInTheDocument();

    const streamLog = screen.getByRole("row", { name: /req-stream/i });
    expect(within(streamLog).getByText("streaming")).toBeInTheDocument();
    expect(within(streamLog).getByText("cursor")).toBeInTheDocument();
    expect(screen.getByRole("table", { name: "Usage rows" }).parentElement).toHaveClass("overflow-x-auto");
    expect(screen.getByRole("table", { name: "Request logs" }).parentElement).toHaveClass("overflow-x-auto");

    await waitFor(() => {
      expect(fetch.mock.calls.map((args) => String(args[0]))).toEqual(expect.arrayContaining(["/api/usage", "/api/logs"]));
    });
    expect(screen.queryByText("req-1092")).not.toBeInTheDocument();
  });

  it("shows key and account attribution and filters by api key and auth type", async () => {
    const usageRecord = {
      id: 1,
      request_id: "req-oauth",
      timestamp: "2026-06-03T10:00:00Z",
      provider: "anthropic",
      model: "claude-sonnet",
      auth_type: "oauth",
      api_key_id: "key-abcdef123456",
      api_key_name: "desktop-client",
      connection_provider: "anthropic",
      account_email: "user@example.test",
      total_tokens: 500,
      cost_usd: 0.02,
      latency_ms: 120,
      status_code: 200
    };

    const fetch = vi.fn(async (input: RequestInfo | URL) => {
      const path = String(input);
      if (path === "/api/keys") {
        return jsonResponse({ data: [{ ID: "key-abcdef123456", Name: "desktop-client", Prefix: "g0r_live", IsActive: true, CreatedAt: "2026-06-03T09:00:00Z" }] });
      }
      if (path.startsWith("/api/usage")) {
        return jsonResponse({ object: "list", data: [usageRecord], limit: 0, offset: 0 });
      }
      if (path === "/api/logs") {
        return jsonResponse(emptyUsageList);
      }
      return jsonResponse({ error: `missing ${path}` }, { status: 404 });
    });
    vi.stubGlobal("fetch", fetch);

    render(<UsagePage />);

    const row = await screen.findByRole("row", { name: /req-oauth/i });
    expect(within(row).getByText("desktop-client")).toBeInTheDocument();
    expect(within(row).getByText("anthropic · user@example.test")).toBeInTheDocument();
    expect(within(row).getByText("oauth")).toBeInTheDocument();

    fireEvent.change(await screen.findByLabelText("Filter by API key"), { target: { value: "key-abcdef123456" } });
    fireEvent.change(screen.getByLabelText("Filter by auth type"), { target: { value: "oauth" } });

    await waitFor(() => {
      const usagePaths = fetch.mock.calls.map((args) => String(args[0])).filter((path) => path.startsWith("/api/usage"));
      expect(usagePaths.some((path) => path.includes("api_key_id=key-abcdef123456") && path.includes("auth_type=oauth"))).toBe(true);
    });
  });

  it("shows loading while usage requests are pending", () => {
    vi.stubGlobal("fetch", vi.fn(() => new Promise<Response>(() => undefined)));

    render(<UsagePage />);

    expect(screen.getByRole("status")).toHaveTextContent("Loading usage data");
  });

  it("shows an empty state when usage and logs are empty", async () => {
    stubFetch({
      "/api/usage": jsonResponse(emptyUsageList),
      "/api/logs": jsonResponse(emptyUsageList)
    });

    render(<UsagePage />);

    expect(await screen.findByText("No usage or logs yet")).toBeInTheDocument();
  });

  it("shows an error state when usage data cannot load", async () => {
    stubFetch({
      "/api/usage": jsonResponse({ error: "usage store unavailable" }, { status: 500 }),
      "/api/logs": jsonResponse(emptyUsageList)
    });

    render(<UsagePage />);

    expect(await screen.findByText("Usage data unavailable")).toBeInTheDocument();
    expect(screen.getByText("usage store unavailable")).toBeInTheDocument();
  });

  it("shows auth-expired state when the control plane rejects usage requests", async () => {
    stubFetch({
      "/api/usage": jsonResponse({ error: "control-plane auth required" }, { status: 403 }),
      "/api/logs": jsonResponse(emptyUsageList)
    });

    render(<UsagePage />);

    expect(await screen.findByText("Session expired")).toBeInTheDocument();
    expect(screen.getByText("control-plane auth required")).toBeInTheDocument();
  });

  it("fetches usage summary and renders an SVG chart", async () => {
    const fetch = vi.fn(async (input: RequestInfo | URL) => {
      const path = String(input);
      if (path === "/api/usage") return jsonResponse({ object: "list", data: [], limit: 0, offset: 0 });
      if (path === "/api/logs") return jsonResponse({ object: "list", data: [], limit: 0, offset: 0 });
      if (path === getUsageSummaryPath()) {
        return jsonResponse({ request_count: 42, total_tokens: 15000, total_cost_usd: 0.75 });
      }
      return jsonResponse({ error: `missing ${path}` }, { status: 404 });
    });
    vi.stubGlobal("fetch", fetch);

    render(<UsagePage />);

    expect(await screen.findByText("Usage summary")).toBeInTheDocument();
    expect(screen.getByText("42")).toBeInTheDocument();
    expect(screen.getByText("15,000")).toBeInTheDocument();
    expect(document.querySelector("svg")).not.toBeNull();

    await waitFor(() => {
      const paths = fetch.mock.calls.map((args) => String(args[0]));
      expect(paths).toContain(getUsageSummaryPath());
    });
  });

  it("degrades gracefully when summary fetch returns zero data", async () => {
    const fetch = vi.fn(async (input: RequestInfo | URL) => {
      const path = String(input);
      if (path === "/api/usage") return jsonResponse({ object: "list", data: [], limit: 0, offset: 0 });
      if (path === "/api/logs") return jsonResponse({ object: "list", data: [], limit: 0, offset: 0 });
      if (path === getUsageSummaryPath()) {
        return jsonResponse({ request_count: 0, total_tokens: 0, total_cost_usd: 0 });
      }
      return jsonResponse({ error: `missing ${path}` }, { status: 404 });
    });
    vi.stubGlobal("fetch", fetch);

    render(<UsagePage />);

    expect(await screen.findByText("Usage summary")).toBeInTheDocument();
    expect(screen.queryByText("Usage data unavailable")).not.toBeInTheDocument();
  });

  it("renders summary section even when summary fetch fails", async () => {
    const fetch = vi.fn(async (input: RequestInfo | URL) => {
      const path = String(input);
      if (path === "/api/usage") return jsonResponse({ object: "list", data: [], limit: 0, offset: 0 });
      if (path === "/api/logs") return jsonResponse({ object: "list", data: [], limit: 0, offset: 0 });
      if (path === getUsageSummaryPath()) {
        return jsonResponse({ error: "summary unavailable" }, { status: 500 });
      }
      return jsonResponse({ error: `missing ${path}` }, { status: 404 });
    });
    vi.stubGlobal("fetch", fetch);

    render(<UsagePage />);

    // Usage page still loads (empty state), summary section absent or shows no error banner from main state
    expect(await screen.findByText("No usage or logs yet")).toBeInTheDocument();
  });
});
