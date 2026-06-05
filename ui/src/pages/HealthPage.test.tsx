import { render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { getConnectionsPath, getProvidersPath } from "../api";
import { HealthPage } from "./HealthPage";

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
    ...init
  });
}

function makeConnection(overrides: Record<string, unknown> = {}) {
  return {
    ID: "conn-1",
    Provider: "openai",
    Name: "openai-main",
    AuthType: "oauth",
    IsActive: true,
    ExpiresAt: null,
    Email: "user@example.com",
    AccountID: "acct-1",
    UnavailableUntil: null,
    BackoffLevel: 0,
    NeedsReauth: false,
    LastRefreshError: null,
    CreatedAt: "2026-06-01T00:00:00Z",
    UpdatedAt: "2026-06-01T00:00:00Z",
    ...overrides
  };
}

function stubBothApis(connections: unknown[]) {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const path = String(input);
      if (path === getProvidersPath()) {
        return jsonResponse({
          data: [
            { id: "openai", auth_types: ["oauth", "api_key"], public_status: "supported" },
            { id: "anthropic", auth_types: ["api_key"], public_status: "supported" }
          ]
        });
      }
      if (path === getConnectionsPath()) {
        return jsonResponse({ data: connections });
      }
      return jsonResponse({ error: `missing ${path}` }, { status: 404 });
    })
  );
}

describe("HealthPage", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("shows loading state while data is fetched", () => {
    vi.stubGlobal("fetch", vi.fn(() => new Promise<Response>(() => undefined)));

    render(<HealthPage />);

    expect(screen.getByRole("status")).toHaveTextContent("Loading health");
  });

  it("renders empty state when no connections exist", async () => {
    stubBothApis([]);

    render(<HealthPage />);

    expect(await screen.findByText("No connections")).toBeInTheDocument();
  });

  it("renders connection health rows with status, backoff, and expiry", async () => {
    stubBothApis([makeConnection()]);

    render(<HealthPage />);

    const table = await screen.findByRole("table", { name: "Connection health" });
    const rows = within(table).getAllByRole("row");
    expect(rows.length).toBeGreaterThanOrEqual(2);

    const dataRow = rows[1];
    expect(within(dataRow).getByText("openai-main")).toBeInTheDocument();
    expect(within(dataRow).getByText("openai")).toBeInTheDocument();
  });

  it("shows needs-reauth indicator for connections requiring re-authentication", async () => {
    stubBothApis([
      makeConnection({
        ID: "conn-stale",
        Name: "stale-conn",
        NeedsReauth: true,
        LastRefreshError: "token revoked"
      }),
      makeConnection({
        ID: "conn-ok",
        Name: "healthy-conn",
        NeedsReauth: false
      })
    ]);

    render(<HealthPage />);

    await screen.findByRole("table", { name: "Connection health" });

    const staleRow = screen.getByRole("row", { name: /stale-conn/i });
    expect(within(staleRow).getByText("Needs re-auth")).toBeInTheDocument();

    const healthyRow = screen.getByRole("row", { name: /healthy-conn/i });
    expect(within(healthyRow).queryByText("Needs re-auth")).not.toBeInTheDocument();
  });

  it("shows backoff level and unavailable_until when present", async () => {
    const unavailableUntil = Math.floor(Date.now() / 1000) + 3600;
    stubBothApis([
      makeConnection({
        ID: "conn-backoff",
        Name: "backed-off",
        BackoffLevel: 3,
        UnavailableUntil: unavailableUntil
      })
    ]);

    render(<HealthPage />);

    await screen.findByRole("table", { name: "Connection health" });
    const row = screen.getByRole("row", { name: /backed-off/i });
    expect(within(row).getByText("3")).toBeInTheDocument();
  });

  it("shows last_refresh_error when present", async () => {
    stubBothApis([
      makeConnection({
        ID: "conn-err",
        Name: "err-conn",
        LastRefreshError: "oauth token expired"
      })
    ]);

    render(<HealthPage />);

    await screen.findByRole("table", { name: "Connection health" });
    expect(screen.getByText("oauth token expired")).toBeInTheDocument();
  });

  it("renders error state on fetch failure", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse({ error: "store unavailable" }, { status: 500 }))
    );

    render(<HealthPage />);

    expect(await screen.findByText("Could not load health")).toBeInTheDocument();
    expect(screen.getByText("store unavailable")).toBeInTheDocument();
  });

  it("renders auth-expired state on 401", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse({ error: "auth required" }, { status: 401 }))
    );

    render(<HealthPage />);

    expect(await screen.findByText("Session expired")).toBeInTheDocument();
  });

  it("calls the connections and providers endpoints", async () => {
    const fetch = vi.fn(async (input: RequestInfo | URL) => {
      const path = String(input);
      if (path === getProvidersPath()) return jsonResponse({ data: [] });
      if (path === getConnectionsPath()) return jsonResponse({ data: [makeConnection()] });
      return jsonResponse({ error: `missing ${path}` }, { status: 404 });
    });
    vi.stubGlobal("fetch", fetch);

    render(<HealthPage />);

    await screen.findByRole("table", { name: "Connection health" });

    const paths = fetch.mock.calls.map((args) => String(args[0]));
    expect(paths).toContain(getConnectionsPath());
    expect(paths).toContain(getProvidersPath());
  });
});
