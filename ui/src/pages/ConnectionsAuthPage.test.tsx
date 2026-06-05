import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ConnectionsAuthPage } from "./ConnectionsAuthPage";

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
    ...init
  });
}

describe("ConnectionsAuthPage", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders connection auth rows without provider contract details or credentials", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const path = String(input);
        if (path === "/api/providers") {
          return jsonResponse({
            data: [
              {
                id: "openai",
                auth_types: ["oauth", "api_key"],
                model_catalog: true,
                list_models: true,
                public_inference: true,
                public_status: "supported"
              }
            ]
          });
        }
        if (path === "/api/connections") {
          return jsonResponse({
            data: [
              {
                ID: "conn-openai",
                Provider: "openai",
                Name: "OpenAI primary",
                AuthType: "oauth",
                IsActive: true,
                Email: "operator@example.com",
                AccountID: "acct-1",
                UnavailableUntil: null,
                BackoffLevel: 0,
                NeedsReauth: false,
                LastRefreshError: null,
                ProviderSpecificData: { access_token: "provider-access-token" },
                AccessToken: "top-secret-access-token",
                RefreshToken: "top-secret-refresh-token",
                APIKey: "provider-api-key",
                CreatedAt: "2026-06-04T00:00:00Z",
                UpdatedAt: "2026-06-04T00:00:00Z"
              }
            ]
          });
        }
        return jsonResponse({ error: `missing ${path}` }, { status: 404 });
      })
    );

    render(<ConnectionsAuthPage />);

    expect(await screen.findByRole("heading", { level: 3, name: "Connections and auth" })).toBeInTheDocument();
    const row = await screen.findByRole("row", { name: /OpenAI primary openai operator@example.com oauth active/i });
    expect(within(row).getByText("operator@example.com")).toBeInTheDocument();
    expect(screen.getByRole("table", { name: "Provider connections" })).toBeInTheDocument();
    expect(screen.queryByRole("table", { name: "Provider contract" })).not.toBeInTheDocument();
    expect(screen.queryByText(/top-secret|provider-access-token|provider-api-key/i)).not.toBeInTheDocument();
  });

  it("shows Needs re-auth badge on stale connections and omits it on healthy ones", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const path = String(input);
        if (path === "/api/providers") {
          return new Response(JSON.stringify({ data: [{ id: "openai", auth_types: ["oauth", "api_key"], model_catalog: true, list_models: true, public_inference: true, public_status: "supported" }] }), { headers: { "Content-Type": "application/json" } });
        }
        if (path === "/api/connections") {
          return new Response(
            JSON.stringify({
              data: [
                {
                  ID: "conn-stale",
                  Provider: "openai",
                  Name: "Stale OAuth",
                  AuthType: "oauth",
                  IsActive: true,
                  Email: "stale@example.com",
                  AccountID: null,
                  UnavailableUntil: null,
                  BackoffLevel: 0,
                  NeedsReauth: true,
                  LastRefreshError: "refresh token revoked",
                  CreatedAt: "2026-06-04T00:00:00Z",
                  UpdatedAt: "2026-06-04T00:00:00Z"
                },
                {
                  ID: "conn-ok",
                  Provider: "openai",
                  Name: "Healthy API",
                  AuthType: "api_key",
                  IsActive: true,
                  Email: null,
                  AccountID: null,
                  UnavailableUntil: null,
                  BackoffLevel: 0,
                  NeedsReauth: false,
                  LastRefreshError: null,
                  CreatedAt: "2026-06-04T00:00:00Z",
                  UpdatedAt: "2026-06-04T00:00:00Z"
                }
              ]
            }),
            { headers: { "Content-Type": "application/json" } }
          );
        }
        return new Response(JSON.stringify({ error: `missing ${path}` }), { status: 404, headers: { "Content-Type": "application/json" } });
      })
    );

    render(<ConnectionsAuthPage />);

    const staleRow = await screen.findByRole("row", { name: /Stale OAuth openai stale@example.com oauth/i });
    expect(within(staleRow).getByText("Needs re-auth")).toBeInTheDocument();
    expect(within(staleRow).getByTitle("refresh token revoked")).toBeInTheDocument();

    const okRow = screen.getByRole("row", { name: /Healthy API openai local api_key/i });
    expect(within(okRow).queryByText("Needs re-auth")).not.toBeInTheDocument();
  });

  it("shows Re-authenticate button for needs_reauth connections and triggers authorize call", async () => {
    const fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = String(input);
      const method = (init?.method ?? "GET").toUpperCase();
      if (path === "/api/providers") {
        return jsonResponse({
          data: [{ id: "openai", auth_types: ["oauth", "api_key"], model_catalog: true, list_models: true, public_inference: true, public_status: "supported" }]
        });
      }
      if (path === "/api/connections") {
        return jsonResponse({
          data: [
            {
              ID: "conn-stale",
              Provider: "openai",
              Name: "Stale OAuth",
              AuthType: "oauth",
              IsActive: true,
              Email: "stale@example.com",
              AccountID: null,
              UnavailableUntil: null,
              BackoffLevel: 0,
              NeedsReauth: true,
              LastRefreshError: "refresh token revoked",
              CreatedAt: "2026-06-04T00:00:00Z",
              UpdatedAt: "2026-06-04T00:00:00Z"
            },
            {
              ID: "conn-ok",
              Provider: "openai",
              Name: "Healthy API",
              AuthType: "api_key",
              IsActive: true,
              Email: null,
              AccountID: null,
              UnavailableUntil: null,
              BackoffLevel: 0,
              NeedsReauth: false,
              LastRefreshError: null,
              CreatedAt: "2026-06-04T00:00:00Z",
              UpdatedAt: "2026-06-04T00:00:00Z"
            }
          ]
        });
      }
      if (path === "/api/oauth/openai/authorize" && method === "POST") {
        return jsonResponse({ provider: "openai", auth_url: "https://openai.com/auth?code=abc", session_id: "sess-reauth-123" });
      }
      return jsonResponse({ error: `missing ${path}` }, { status: 404 });
    });
    vi.stubGlobal("fetch", fetch);

    render(<ConnectionsAuthPage />);

    // Stale row has Re-authenticate button; healthy row does not
    const staleRow = await screen.findByRole("row", { name: /Stale OAuth openai stale@example.com oauth/i });
    const reAuthBtn = within(staleRow).getByRole("button", { name: /Re-authenticate/i });
    expect(reAuthBtn).toBeInTheDocument();

    const okRow = screen.getByRole("row", { name: /Healthy API openai local api_key/i });
    expect(within(okRow).queryByRole("button", { name: /Re-authenticate/i })).not.toBeInTheDocument();

    // Clicking Re-authenticate calls the authorize endpoint
    fireEvent.click(reAuthBtn);

    await waitFor(() => {
      const authCalls = fetch.mock.calls.filter((args) => String(args[0]) === "/api/oauth/openai/authorize");
      expect(authCalls.length).toBeGreaterThanOrEqual(1);
    });
  });

  it("exposes provider OAuth controls on the dedicated Connections/Auth route", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const path = String(input);
        if (path === "/api/providers") {
          return jsonResponse({
            data: [
              {
                id: "openai",
                auth_types: ["oauth", "api_key"],
                model_catalog: true,
                list_models: true,
                public_inference: true,
                public_status: "supported"
              }
            ]
          });
        }
        if (path === "/api/connections") {
          return jsonResponse({ data: [] });
        }
        return jsonResponse({ error: `missing ${path}` }, { status: 404 });
      })
    );

    render(<ConnectionsAuthPage />);

    expect(await screen.findByRole("combobox", { name: "OAuth provider" })).toBeInTheDocument();
    expect(screen.getByLabelText("OAuth account label")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Start OAuth" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add connection" })).toBeInTheDocument();
    expect(screen.queryByRole("table", { name: "Provider contract" })).not.toBeInTheDocument();
  });
});
