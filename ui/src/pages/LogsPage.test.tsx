import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { LogsPage } from "./LogsPage";

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
    ...init
  });
}

function makeRow(overrides: Record<string, unknown> = {}) {
  return {
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
    client_tool: "codex",
    combo_name: "research-stack",
    ...overrides
  };
}

function listBody(rows: unknown[], total: number, offset = 0) {
  return { object: "list", data: rows, limit: 50, offset, total };
}

function lastUrl(fetch: ReturnType<typeof vi.fn>): string {
  return String(fetch.mock.calls[fetch.mock.calls.length - 1][0]);
}

describe("LogsPage", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("shows loading, empty, and error states", async () => {
    vi.stubGlobal("fetch", vi.fn(() => new Promise<Response>(() => undefined)));
    const { unmount } = render(<LogsPage />);
    expect(screen.getByRole("status")).toHaveTextContent("Loading request logs");

    unmount();
    vi.stubGlobal("fetch", vi.fn(async () => jsonResponse(listBody([], 0))));
    render(<LogsPage />);
    expect(await screen.findByText("No logs match")).toBeInTheDocument();

    vi.unstubAllGlobals();
    vi.stubGlobal("fetch", vi.fn(async () => jsonResponse({ error: "logs unavailable" }, { status: 500, statusText: "Server Error" })));
    render(<LogsPage />);
    expect(await screen.findByText("Could not load logs")).toBeInTheDocument();

    vi.unstubAllGlobals();
    vi.stubGlobal("fetch", vi.fn(async () => jsonResponse({ error: "control-plane auth required" }, { status: 401 })));
    render(<LogsPage />);
    expect(await screen.findByText("Session expired")).toBeInTheDocument();
  });

  it("renders a row with status, provider, and model", async () => {
    const fetch = vi.fn(async () => jsonResponse(listBody([makeRow()], 1)));
    vi.stubGlobal("fetch", fetch);

    render(<LogsPage />);

    expect(await screen.findByText("openai")).toBeInTheDocument();
    expect(screen.getByText("gpt-4o")).toBeInTheDocument();
    expect(screen.getByText("200")).toBeInTheDocument();
    await waitFor(() => {
      const url = new URL(lastUrl(fetch), "http://localhost");
      expect(url.searchParams.get("limit")).toBe("50");
      expect(url.searchParams.get("offset")).toBe("0");
    });
  });

  it("applies the Kind filter and refetches with status_class", async () => {
    const fetch = vi.fn(async () => jsonResponse(listBody([makeRow()], 1)));
    vi.stubGlobal("fetch", fetch);

    render(<LogsPage />);
    await screen.findByText("openai");

    fireEvent.change(screen.getByLabelText("Kind"), { target: { value: "server_error" } });

    await waitFor(() => {
      const url = new URL(lastUrl(fetch), "http://localhost");
      expect(url.searchParams.get("status_class")).toBe("server_error");
    });
  });

  it("debounces the search input", async () => {
    const fetch = vi.fn(async () => jsonResponse(listBody([makeRow()], 1)));
    vi.stubGlobal("fetch", fetch);

    render(<LogsPage />);
    await screen.findByText("openai");
    const initialCalls = fetch.mock.calls.length;

    fireEvent.change(screen.getByLabelText("Search logs"), { target: { value: "abc" } });
    expect(fetch.mock.calls.length).toBe(initialCalls);

    await waitFor(() => {
      const url = new URL(lastUrl(fetch), "http://localhost");
      expect(url.searchParams.get("search")).toBe("abc");
    });
  });

  it("paginates by advancing the offset and shows the total", async () => {
    const fetch = vi.fn(async (input: RequestInfo | URL) => {
      const url = new URL(String(input), "http://localhost");
      const offset = Number(url.searchParams.get("offset") ?? "0");
      return jsonResponse(listBody([makeRow({ id: offset + 1, request_id: `req-${offset}` })], 120, offset), {});
    });
    vi.stubGlobal("fetch", fetch);

    render(<LogsPage />);
    expect(await screen.findByText("Showing 1–1 of 120")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Next" }));

    await waitFor(() => {
      const url = new URL(lastUrl(fetch), "http://localhost");
      expect(url.searchParams.get("offset")).toBe("50");
    });
    expect(await screen.findByText("Showing 51–51 of 120")).toBeInTheDocument();
  });
});
