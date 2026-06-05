import { render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { LogsPage } from "./LogsPage";

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
    ...init
  });
}

describe("LogsPage", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("shows loading, empty, error, and auth-expired states", async () => {
    vi.stubGlobal("fetch", vi.fn(() => new Promise<Response>(() => undefined)));

    const { unmount } = render(<LogsPage />);
    expect(screen.getByRole("status")).toHaveTextContent("Loading request logs");

    unmount();
    vi.stubGlobal("fetch", vi.fn(async () => jsonResponse({ object: "list", data: [], limit: 50, offset: 0 })));
    render(<LogsPage />);
    expect(await screen.findByText("No request logs")).toBeInTheDocument();

    unmount();
    vi.stubGlobal("fetch", vi.fn(async () => jsonResponse({ error: "logs unavailable" }, { status: 500, statusText: "Server Error" })));
    render(<LogsPage />);
    expect(await screen.findByText("Could not load logs")).toBeInTheDocument();
    expect(screen.getByText("logs unavailable")).toBeInTheDocument();

    unmount();
    vi.stubGlobal("fetch", vi.fn(async () => jsonResponse({ error: "control-plane auth required" }, { status: 401 })));
    render(<LogsPage />);
    expect(await screen.findByText("Session expired")).toBeInTheDocument();
    expect(screen.getByText("control-plane auth required")).toBeInTheDocument();
  });

  it("renders request logs and uses bounded query defaults", async () => {
    const fetch = vi.fn(async () =>
      jsonResponse({
        object: "list",
        data: [
          {
            id: 1,
            request_id: "req-log",
            timestamp: "2026-06-04T10:00:00Z",
            provider: "openai",
            model: "gpt-4o",
            auth_type: "api_key",
            total_tokens: 44,
            cost_usd: 0.01,
            latency_ms: 100,
            status_code: 200,
            client_tool: "codex"
          }
        ],
        limit: 50,
        offset: 0
      })
    );
    vi.stubGlobal("fetch", fetch);

    render(<LogsPage />);

    expect(await screen.findByText("req-log")).toBeInTheDocument();
    expect(screen.getByText("codex")).toBeInTheDocument();
    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith("/api/logs?limit=50&offset=0", expect.objectContaining({ credentials: "same-origin" }));
    });
  });
});
