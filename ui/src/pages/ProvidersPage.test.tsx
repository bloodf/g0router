import { fireEvent, render, screen, within } from "@testing-library/react";
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
  omp_id: "openai",
  router9_id: "openai",
  bifrost_id: "openai",
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

    expect(fetch).toHaveBeenCalledWith("/api/providers", expect.objectContaining({ credentials: "same-origin" }));
    expect(fetch).toHaveBeenCalledWith("/api/connections", expect.objectContaining({ credentials: "same-origin" }));
    expect(screen.queryByText(/top-secret|provider-access-token|provider-refresh-token|provider-api-key/i)).not.toBeInTheDocument();
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
