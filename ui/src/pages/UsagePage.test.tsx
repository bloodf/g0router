import { render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
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
      expect(fetch.mock.calls.map(([path]) => path)).toEqual(expect.arrayContaining(["/api/usage", "/api/logs"]));
    });
    expect(screen.queryByText("req-1092")).not.toBeInTheDocument();
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
});
