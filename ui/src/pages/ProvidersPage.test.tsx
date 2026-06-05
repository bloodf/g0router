import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ProvidersPage } from "./ProvidersPage";

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
    ...init
  });
}

function deferredResponse() {
  let resolve!: (value: Response) => void;
  const promise = new Promise<Response>((resolver) => {
    resolve = resolver;
  });
  return { promise, resolve };
}

const providerEntry = {
  id: "openai",
  auth_types: ["oauth", "api_key"],
  refresh: true,
  registered_adapter: true,
  public_inference: true,
  direct_dispatch: true,
  inference: true,
  streaming: true,
  model_catalog: true,
  list_models: true,
  quota: false,
  public_status: "supported",
  notes: "public direct-dispatch provider"
};

const connectionEntry = {
  ID: "conn-openai",
  Provider: "openai",
  Name: "primary",
  AuthType: "oauth",
  ExpiresAt: 1_799_000_000,
  IsActive: true,
  ProviderSpecificData: {
    access_token: "provider-access-token",
    refresh_token: "provider-refresh-token"
  },
  AccountID: "acct-1",
  Email: "operator@example.com",
  UnavailableUntil: null,
  BackoffLevel: 0,
  ModelLocks: {},
  NeedsReauth: false,
  LastRefreshError: null,
  CreatedAt: "2026-06-03T00:00:00Z",
  UpdatedAt: "2026-06-03T00:05:00Z",
  AccessToken: "top-secret-access-token",
  RefreshToken: "top-secret-refresh-token",
  APIKey: "provider-api-key"
};

describe("ProvidersPage", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("shows a loading state while provider and connection contracts are loading", async () => {
    const providers = deferredResponse();
    const connections = deferredResponse();
    const fetch = vi.fn((path: string) => {
      if (path === "/api/providers") {
        return providers.promise;
      }
      return connections.promise;
    });
    vi.stubGlobal("fetch", fetch);

    render(<ProvidersPage />);

    expect(screen.getByRole("status")).toHaveTextContent("Loading providers");

    providers.resolve(jsonResponse({ data: [] }));
    connections.resolve(jsonResponse({ data: [] }));
    await screen.findByText("No provider records");
  });

  it("renders providers and connections from the management API without leaking credentials", async () => {
    const fetch = vi.fn(async (path: string) => {
      if (path === "/api/providers") {
        return jsonResponse({ data: [providerEntry] });
      }
      if (path === "/api/connections") {
        return jsonResponse({ data: [connectionEntry] });
      }
      throw new Error(`unexpected path ${path}`);
    });
    vi.stubGlobal("fetch", fetch);

    render(<ProvidersPage />);

    const providerRow = await screen.findByRole("row", { name: /openai supported oauth, api_key/i });
    expect(within(providerRow).getByText("public direct-dispatch provider")).toBeInTheDocument();

    const connectionRow = screen.getByRole("row", { name: /primary openai operator@example.com oauth active/i });
    expect(within(connectionRow).getByText("operator@example.com")).toBeInTheDocument();
    expect(within(connectionRow).getByText("active")).toBeInTheDocument();
    expect(screen.getByRole("table", { name: "Provider contract" }).parentElement).toHaveClass("overflow-x-auto");
    expect(screen.getByRole("table", { name: "Provider connections" }).parentElement).toHaveClass("overflow-x-auto");

    expect(fetch).toHaveBeenCalledWith("/api/providers", expect.objectContaining({ credentials: "same-origin" }));
    expect(fetch).toHaveBeenCalledWith("/api/connections", expect.objectContaining({ credentials: "same-origin" }));
    expect(screen.queryByText(/top-secret|provider-access-token|provider-refresh-token|provider-api-key/i)).not.toBeInTheDocument();
  });

  it("creates, tests, and deletes API-key connections without rendering secrets", async () => {
    const connections = [connectionEntry];
    const fetch = vi.fn(async (path: string, options?: RequestInit) => {
      if (path === "/api/providers") {
        return jsonResponse({ data: [providerEntry] });
      }
      if (path === "/api/connections" && options?.method === "POST") {
        expect(options.body).toContain("\"api_key\":\"sk-created-secret\"");
        connections.push({
          ...connectionEntry,
          ID: "conn-created",
          Provider: "openai",
          Name: "created",
          AuthType: "api_key",
          Email: "",
          AccountID: ""
        });
        return jsonResponse(connections[1], { status: 201 });
      }
      if (path === "/api/connections/conn-created/test") {
        return jsonResponse({ ok: true, provider: "openai", name: "created" });
      }
      if (path === "/api/connections/conn-created" && options?.method === "DELETE") {
        connections.splice(1, 1);
        return new Response(null, { status: 204 });
      }
      if (path === "/api/connections") {
        return jsonResponse({ data: connections });
      }
      throw new Error(`unexpected path ${path}`);
    });
    vi.stubGlobal("fetch", fetch);
    vi.spyOn(window, "confirm").mockReturnValue(true);

    render(<ProvidersPage />);

    await screen.findByRole("row", { name: /primary openai/i });
    fireEvent.change(screen.getByLabelText("Provider"), { target: { value: "openai" } });
    fireEvent.change(screen.getByLabelText("Connection name"), { target: { value: "created" } });
    fireEvent.change(screen.getByLabelText("Provider API key"), { target: { value: "sk-created-secret" } });
    fireEvent.click(screen.getByRole("button", { name: "Add connection" }));

    const createdRow = await screen.findByRole("row", { name: /created openai local api_key active/i });
    expect(createdRow).toBeInTheDocument();
    expect(screen.queryByText("sk-created-secret")).not.toBeInTheDocument();

    fireEvent.click(within(createdRow).getByRole("button", { name: "Test created" }));
    expect(await screen.findByText("created is active")).toBeInTheDocument();

    fireEvent.click(within(createdRow).getByRole("button", { name: "Delete created" }));
    await waitFor(() => {
      expect(screen.queryByRole("row", { name: /created openai local api_key active/i })).not.toBeInTheDocument();
    });
    expect(window.confirm).toHaveBeenCalledWith("Delete provider connection created?");
  });

  it("updates connection active state through the documented connection PUT endpoint without sending secrets", async () => {
    const connections = [{ ...connectionEntry }];
    const fetch = vi.fn(async (path: string, options?: RequestInit) => {
      if (path === "/api/providers") {
        return jsonResponse({ data: [providerEntry] });
      }
      if (path === "/api/connections/conn-openai" && options?.method === "PUT") {
        const body = JSON.parse(String(options.body));
        expect(body).toEqual({
          provider: "openai",
          name: "primary",
          auth_type: "oauth",
          expires_at: 1_799_000_000,
          is_active: false,
          provider_specific_data: {},
          account_id: "acct-1",
          email: "operator@example.com",
          unavailable_until: null,
          backoff_level: 0,
          model_locks: {}
        });
        expect(String(options.body)).not.toMatch(/top-secret|provider-access-token|provider-refresh-token|provider-api-key/i);
        connections[0] = { ...connections[0], IsActive: false };
        return jsonResponse(connections[0]);
      }
      if (path === "/api/connections") {
        return jsonResponse({ data: connections });
      }
      throw new Error(`unexpected path ${path}`);
    });
    vi.stubGlobal("fetch", fetch);

    render(<ProvidersPage />);

    const row = await screen.findByRole("row", { name: /primary openai operator@example.com oauth active/i });
    fireEvent.click(within(row).getByRole("button", { name: "Deactivate primary" }));

    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith(
        "/api/connections/conn-openai",
        expect.objectContaining({ credentials: "same-origin", method: "PUT" })
      );
    });
    expect(await screen.findByRole("row", { name: /primary openai operator@example.com oauth inactive/i })).toBeInTheDocument();
  });

  it("creates Cloudflare AI Gateway connections with account ID metadata", async () => {
    const cloudflareProvider = {
      ...providerEntry,
      id: "cloudflare-ai-gateway",
      auth_types: ["api_key"],
      model_catalog: false,
      list_models: false,
      quota: false,
      notes: "requires account_id"
    };
    const connections: unknown[] = [];
    const fetch = vi.fn(async (path: string, options?: RequestInit) => {
      if (path === "/api/providers") {
        return jsonResponse({ data: [cloudflareProvider] });
      }
      if (path === "/api/connections" && options?.method === "POST") {
        expect(JSON.parse(String(options.body))).toEqual({
          provider: "cloudflare-ai-gateway",
          name: "cf-prod",
          auth_type: "api_key",
          api_key: "cf-secret",
          is_active: true,
          account_id: "cf-account-123"
        });
        connections.push({
          ...connectionEntry,
          ID: "conn-cloudflare",
          Provider: "cloudflare-ai-gateway",
          Name: "cf-prod",
          AuthType: "api_key",
          AccountID: "cf-account-123",
          Email: ""
        });
        return jsonResponse(connections[0], { status: 201 });
      }
      if (path === "/api/connections") {
        return jsonResponse({ data: connections });
      }
      throw new Error(`unexpected path ${path}`);
    });
    vi.stubGlobal("fetch", fetch);

    render(<ProvidersPage />);

    await screen.findByRole("combobox", { name: "Provider" });
    fireEvent.change(screen.getByLabelText("Connection name"), { target: { value: "cf-prod" } });
    fireEvent.change(screen.getByLabelText("Provider API key"), { target: { value: "cf-secret" } });
    fireEvent.change(screen.getByLabelText("Cloudflare account ID"), { target: { value: "cf-account-123" } });
    fireEvent.click(screen.getByRole("button", { name: "Add connection" }));

    expect(await screen.findByRole("row", { name: /cf-prod cloudflare-ai-gateway cf-account-123 api_key active/i })).toBeInTheDocument();
    expect(screen.queryByText("cf-secret")).not.toBeInTheDocument();
  });

  it("requires Cloudflare account ID before creating the connection", async () => {
    const cloudflareProvider = {
      ...providerEntry,
      id: "cloudflare-ai-gateway",
      auth_types: ["api_key"],
      notes: "requires account_id"
    };
    const fetch = vi.fn(async (path: string, options?: RequestInit) => {
      if (path === "/api/providers") {
        return jsonResponse({ data: [cloudflareProvider] });
      }
      if (path === "/api/connections" && !options?.method) {
        return jsonResponse({ data: [] });
      }
      throw new Error(`unexpected path ${path}`);
    });
    vi.stubGlobal("fetch", fetch);

    render(<ProvidersPage />);

    await screen.findByRole("combobox", { name: "Provider" });
    fireEvent.change(screen.getByLabelText("Connection name"), { target: { value: "cf-prod" } });
    fireEvent.change(screen.getByLabelText("Provider API key"), { target: { value: "cf-secret" } });
    fireEvent.click(screen.getByRole("button", { name: "Add connection" }));

    expect(await screen.findByText("Cloudflare account ID is required.")).toBeInTheDocument();
    expect(fetch).not.toHaveBeenCalledWith("/api/connections", expect.objectContaining({ method: "POST" }));
    expect(screen.queryByText("cf-secret")).not.toBeInTheDocument();
  });

  it("renders OAuth-capable provider controls separately from API-key creation", async () => {
    const fetch = vi.fn(async (path: string) => {
      if (path === "/api/providers") {
        return jsonResponse({
          data: [
            providerEntry,
            { ...providerEntry, id: "ollama", auth_types: ["noauth"], public_status: "supported" }
          ]
        });
      }
      if (path === "/api/connections") {
        return jsonResponse({ data: [] });
      }
      throw new Error(`unexpected path ${path}`);
    });
    vi.stubGlobal("fetch", fetch);

    render(<ProvidersPage />);

    await screen.findByRole("combobox", { name: "Provider" });
    expect(screen.getByRole("combobox", { name: "OAuth provider" })).toBeInTheDocument();
    expect(screen.getByLabelText("OAuth account label")).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Start OAuth" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Add connection" })).toBeInTheDocument();
    expect(within(screen.getByRole("combobox", { name: "OAuth provider" })).queryByRole("option", { name: "ollama" })).not.toBeInTheDocument();
  });

  it("starts provider OAuth with an account label and renders only redacted session details", async () => {
    const fetch = vi.fn(async (path: string, options?: RequestInit) => {
      if (path === "/api/providers") {
        return jsonResponse({ data: [providerEntry] });
      }
      if (path === "/api/connections") {
        return jsonResponse({ data: [] });
      }
      if (path === "/api/oauth/openai/authorize" && options?.method === "POST") {
        expect(options.body).toBe(JSON.stringify({ account_label: "work-oauth" }));
        return jsonResponse({
          provider: "openai",
          auth_url: "https://auth.example.test/authorize?state=oauth-state",
          session_id: "oauth-state",
          user_code: "ABCD-EFGH",
          verification: "https://auth.example.test/device",
          expires_in: 600,
          access_token: "should-not-render",
          refresh_token: "should-not-render"
        });
      }
      throw new Error(`unexpected path ${path}`);
    });
    vi.stubGlobal("fetch", fetch);

    render(<ProvidersPage />);

    await screen.findByRole("combobox", { name: "OAuth provider" });
    fireEvent.change(screen.getByLabelText("OAuth account label"), { target: { value: "work-oauth" } });
    fireEvent.click(screen.getByRole("button", { name: "Start OAuth" }));

    expect(await screen.findByRole("link", { name: "Open authorization URL" })).toHaveAttribute(
      "href",
      "https://auth.example.test/authorize?state=oauth-state"
    );
    expect(screen.getByText("Session state: oauth-state")).toBeInTheDocument();
    expect(screen.getByText("Device code: ABCD-EFGH")).toBeInTheDocument();
    expect(screen.queryByText(/should-not-render|access_token|refresh_token/i)).not.toBeInTheDocument();
  });

  it("exchanges a provider OAuth callback, reloads the redacted connection, and does not render secrets", async () => {
    const connections: unknown[] = [];
    const fetch = vi.fn(async (path: string, options?: RequestInit) => {
      if (path === "/api/providers") {
        return jsonResponse({ data: [providerEntry] });
      }
      if (path === "/api/connections") {
        return jsonResponse({ data: connections });
      }
      if (path === "/api/oauth/openai/authorize" && options?.method === "POST") {
        return jsonResponse({
          provider: "openai",
          auth_url: "https://auth.example.test/authorize?state=oauth-state",
          session_id: "oauth-state"
        });
      }
      if (path === "/api/oauth/openai/exchange" && options?.method === "POST") {
        expect(options.body).toBe(JSON.stringify({ state: "oauth-state", code: "callback-code" }));
        connections.push({
          ID: "conn-oauth",
          Provider: "openai",
          Name: "work-oauth",
          AuthType: "oauth",
          IsActive: true,
          AccountID: null,
          Email: null,
          BackoffLevel: 0,
          CreatedAt: "2026-06-04T00:00:00Z",
          UpdatedAt: "2026-06-04T00:00:00Z",
          AccessToken: "returned-access-token",
          RefreshToken: "returned-refresh-token"
        });
        return jsonResponse({
          id: "conn-oauth",
          provider: "openai",
          name: "work-oauth",
          auth_type: "oauth",
          scopes: ["read"]
        });
      }
      throw new Error(`unexpected path ${path}`);
    });
    vi.stubGlobal("fetch", fetch);

    render(<ProvidersPage />);

    await screen.findByRole("combobox", { name: "OAuth provider" });
    fireEvent.change(screen.getByLabelText("OAuth account label"), { target: { value: "work-oauth" } });
    fireEvent.click(screen.getByRole("button", { name: "Start OAuth" }));
    await screen.findByRole("link", { name: "Open authorization URL" });
    fireEvent.change(screen.getByLabelText("Callback URL or code"), {
      target: { value: "http://127.0.0.1:8080/api/oauth/callback?state=oauth-state&code=callback-code" }
    });
    fireEvent.click(screen.getByRole("button", { name: "Complete OAuth" }));

    expect(await screen.findByText("OAuth connected work-oauth")).toBeInTheDocument();
    expect(screen.getByRole("row", { name: /work-oauth openai local oauth active/i })).toBeInTheDocument();
    expect(screen.queryByText(/returned-access-token|returned-refresh-token|callback-code/i)).not.toBeInTheDocument();
  });

  it("polls device-style provider OAuth, reloads the redacted connection, and does not render secrets", async () => {
    const connections: unknown[] = [];
    const cursorProvider = { ...providerEntry, id: "cursor", public_status: "auth_only", auth_types: ["oauth"] };
    const fetch = vi.fn(async (path: string, options?: RequestInit) => {
      if (path === "/api/providers") {
        return jsonResponse({ data: [cursorProvider] });
      }
      if (path === "/api/connections") {
        return jsonResponse({ data: connections });
      }
      if (path === "/api/oauth/cursor/authorize" && options?.method === "POST") {
        return jsonResponse({
          provider: "cursor",
          auth_url: "https://cursor.example/loginDeepControl?uuid=cursor-session",
          session_id: "cursor-session",
          user_code: "cursor-session",
          verification: "https://cursor.example/loginDeepControl?uuid=cursor-session",
          poll_interval: 1,
          access_token: "should-not-render",
          refresh_token: "should-not-render"
        });
      }
      if (path === "/api/oauth/cursor/poll?session_id=cursor-session" && options?.method === "GET") {
        connections.push({
          ID: "conn-cursor",
          Provider: "cursor",
          Name: "Cursor OAuth",
          AuthType: "oauth",
          IsActive: true,
          AccountID: null,
          Email: null,
          BackoffLevel: 0,
          CreatedAt: "2026-06-04T00:00:00Z",
          UpdatedAt: "2026-06-04T00:00:00Z",
          AccessToken: "returned-cursor-access-token",
          RefreshToken: "returned-cursor-refresh-token"
        });
        return jsonResponse({
          status: "complete",
          connection: {
            id: "conn-cursor",
            provider: "cursor",
            name: "Cursor OAuth",
            auth_type: "oauth"
          }
        });
      }
      throw new Error(`unexpected path ${path}`);
    });
    vi.stubGlobal("fetch", fetch);

    render(<ProvidersPage />);

    await screen.findByRole("combobox", { name: "OAuth provider" });
    fireEvent.change(screen.getByLabelText("OAuth account label"), { target: { value: "Cursor OAuth" } });
    fireEvent.click(screen.getByRole("button", { name: "Start OAuth" }));
    await screen.findByRole("link", { name: "Open authorization URL" });
    expect(screen.getByText("Device code: cursor-session")).toBeInTheDocument();
    expect(screen.getByText("Poll interval: 1s")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Poll OAuth" }));

    expect(await screen.findByText("OAuth connected Cursor OAuth")).toBeInTheDocument();
    expect(screen.getByRole("row", { name: /Cursor OAuth cursor local oauth active/i })).toBeInTheDocument();
    expect(fetch).toHaveBeenCalledWith("/api/oauth/cursor/poll?session_id=cursor-session", expect.objectContaining({ method: "GET" }));
    expect(screen.queryByText(/should-not-render|returned-cursor-access-token|returned-cursor-refresh-token/i)).not.toBeInTheDocument();
  });

  it("handles provider OAuth exchange failures without leaking callback secrets", async () => {
    const fetch = vi.fn(async (path: string, options?: RequestInit) => {
      if (path === "/api/providers") {
        return jsonResponse({ data: [providerEntry] });
      }
      if (path === "/api/connections") {
        return jsonResponse({ data: [] });
      }
      if (path === "/api/oauth/openai/authorize" && options?.method === "POST") {
        return jsonResponse({
          provider: "openai",
          auth_url: "https://auth.example.test/authorize?state=oauth-state",
          session_id: "oauth-state"
        });
      }
      if (path === "/api/oauth/openai/exchange" && options?.method === "POST") {
        return jsonResponse({ error: "oauth exchange failed" }, { status: 502 });
      }
      throw new Error(`unexpected path ${path}`);
    });
    vi.stubGlobal("fetch", fetch);

    render(<ProvidersPage />);

    await screen.findByRole("combobox", { name: "OAuth provider" });
    fireEvent.click(screen.getByRole("button", { name: "Start OAuth" }));
    await screen.findByRole("link", { name: "Open authorization URL" });
    fireEvent.change(screen.getByLabelText("Callback URL or code"), { target: { value: "callback-secret-code" } });
    fireEvent.click(screen.getByRole("button", { name: "Complete OAuth" }));

    expect(await screen.findByText("oauth exchange failed")).toBeInTheDocument();
    expect(screen.queryByText("callback-secret-code")).not.toBeInTheDocument();
    expect(screen.queryByText(/access_token|refresh_token/i)).not.toBeInTheDocument();
  });

  it("renders providers with null auth_types from the live provider matrix as none", async () => {
    const fetch = vi.fn(async (path: string) => {
      if (path === "/api/providers") {
        return jsonResponse({ data: [{ ...providerEntry, id: "qwen", auth_types: null, public_status: "unsupported" }] });
      }
      if (path === "/api/connections") {
        return jsonResponse({ data: [] });
      }
      throw new Error(`unexpected path ${path}`);
    });
    vi.stubGlobal("fetch", fetch);

    render(<ProvidersPage />);

    const row = await screen.findByRole("row", { name: /qwen unsupported none/i });
    expect(within(row).getByText("none")).toBeInTheDocument();
  });

  it("renders an empty state when both provider contracts are empty", async () => {
    const fetch = vi.fn(async () => jsonResponse({ data: [] }));
    vi.stubGlobal("fetch", fetch);

    render(<ProvidersPage />);

    expect(await screen.findByText("No provider records")).toBeInTheDocument();
    expect(screen.getByText("The management API returned no providers or connections.")).toBeInTheDocument();
  });

  it("renders an error state and retries provider loading", async () => {
    const providerResponses = [
      jsonResponse({ error: "providers unavailable" }, { status: 500, statusText: "Internal Server Error" }),
      jsonResponse({ data: [providerEntry] })
    ];
    const connectionResponses = [jsonResponse({ data: [] }), jsonResponse({ data: [] })];
    const fetch = vi.fn(async (path: string) => {
      if (path === "/api/providers") {
        return providerResponses.shift();
      }
      if (path === "/api/connections") {
        return connectionResponses.shift();
      }
      throw new Error(`unexpected path ${path}`);
    });
    vi.stubGlobal("fetch", fetch);

    render(<ProvidersPage />);

    expect(await screen.findByText("Could not load providers")).toBeInTheDocument();
    expect(screen.getByText("providers unavailable")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Retry" }));

    await screen.findByRole("row", { name: /openai supported oauth, api_key/i });
    expect(fetch).toHaveBeenCalledTimes(4);
  });

  it("shows Needs re-auth badge when needs_reauth is true and hides it when false", async () => {
    const staleEntry = {
      ...connectionEntry,
      ID: "conn-stale",
      Name: "stale",
      NeedsReauth: true,
      LastRefreshError: "token expired"
    };
    const freshEntry = { ...connectionEntry, ID: "conn-fresh", Name: "fresh", NeedsReauth: false, LastRefreshError: null };
    const fetch = vi.fn(async (path: string) => {
      if (path === "/api/providers") {
        return jsonResponse({ data: [providerEntry] });
      }
      if (path === "/api/connections") {
        return jsonResponse({ data: [staleEntry, freshEntry] });
      }
      throw new Error(`unexpected path ${path}`);
    });
    vi.stubGlobal("fetch", fetch);

    render(<ProvidersPage />);

    const staleRow = await screen.findByRole("row", { name: /stale openai operator@example.com oauth/i });
    expect(within(staleRow).getByText("Needs re-auth")).toBeInTheDocument();
    expect(within(staleRow).getByTitle("token expired")).toBeInTheDocument();

    const freshRow = screen.getByRole("row", { name: /fresh openai operator@example.com oauth/i });
    expect(within(freshRow).queryByText("Needs re-auth")).not.toBeInTheDocument();
  });

  it("renders an auth-expired state for protected provider APIs", async () => {
    const fetch = vi.fn(async (path: string) => {
      if (path === "/api/providers") {
        return jsonResponse({ error: "control-plane auth required" }, { status: 401 });
      }
      return jsonResponse({ data: [] });
    });
    vi.stubGlobal("fetch", fetch);

    render(<ProvidersPage />);

    expect(await screen.findByText("Authentication expired")).toBeInTheDocument();
    expect(screen.getByText("control-plane auth required")).toBeInTheDocument();
  });
});
