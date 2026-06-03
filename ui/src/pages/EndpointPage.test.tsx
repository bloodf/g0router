import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { EndpointPage } from "./EndpointPage";

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

const existingKey = {
  ID: "key-1",
  Name: "local-admin",
  Prefix: "g0_live_3fb2",
  IsActive: true,
  LastUsedAt: "2026-06-03T00:00:00Z",
  CreatedAt: "2026-06-02T23:00:00Z",
  raw: "stored-raw-api-key",
  key: "stored-gateway-key"
};

const createdKey = {
  ID: "key-2",
  Name: "ops-key",
  Prefix: "g0_live_a917",
  IsActive: true,
  LastUsedAt: null,
  CreatedAt: "2026-06-03T00:10:00Z"
};

describe("EndpointPage", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("shows a loading state while API keys are loading", async () => {
    const keys = deferredResponse();
    const fetch = vi.fn(() => keys.promise);
    vi.stubGlobal("fetch", fetch);

    render(<EndpointPage />);

    expect(screen.getByRole("status")).toHaveTextContent("Loading API keys");

    keys.resolve(jsonResponse({ data: [] }));
    await screen.findByText("No API keys");
  });

  it("renders stored API keys from /api/keys without displaying raw key material", async () => {
    const fetch = vi.fn(async (path: string) => {
      if (path === "/api/keys") {
        return jsonResponse({ data: [existingKey] });
      }
      throw new Error(`unexpected path ${path}`);
    });
    vi.stubGlobal("fetch", fetch);

    render(<EndpointPage />);

    const row = await screen.findByRole("row", { name: /local-admin g0_live_3fb2 active/i });
    expect(within(row).getByText("g0_live_3fb2")).toBeInTheDocument();
    expect(fetch).toHaveBeenCalledWith("/api/keys", expect.objectContaining({ credentials: "same-origin" }));
    expect(screen.queryByText(/stored-raw-api-key|stored-gateway-key/i)).not.toBeInTheDocument();
  });

  it("creates a key through /api/keys and shows only the new raw value as transient output", async () => {
    let keys: unknown[] = [existingKey];
    const fetch = vi.fn(async (path: string, init?: RequestInit) => {
      if (path === "/api/keys" && init?.method === "POST") {
        keys = [existingKey, createdKey];
        return jsonResponse({ key: createdKey, raw: "g0_test_created_once" });
      }
      if (path === "/api/keys") {
        return jsonResponse({ data: keys });
      }
      throw new Error(`unexpected path ${path}`);
    });
    vi.stubGlobal("fetch", fetch);

    render(<EndpointPage />);

    await screen.findByRole("row", { name: /local-admin/i });
    fireEvent.change(screen.getByLabelText("Key name"), { target: { value: "ops-key" } });
    fireEvent.click(screen.getByRole("button", { name: "Create key" }));

    expect(await screen.findByText("g0_test_created_once")).toBeInTheDocument();
    expect(screen.getByText("Copy it now. It is not available from stored key data.")).toBeInTheDocument();
    expect(screen.queryByText(/stored-raw-api-key|stored-gateway-key/i)).not.toBeInTheDocument();
    expect(fetch).toHaveBeenCalledWith(
      "/api/keys",
      expect.objectContaining({
        body: "{\"name\":\"ops-key\"}",
        method: "POST"
      })
    );
  });

  it("deletes keys through /api/keys/:id and refreshes the list", async () => {
    let keys: unknown[] = [existingKey];
    const fetch = vi.fn(async (path: string, init?: RequestInit) => {
      if (path === "/api/keys/key-1" && init?.method === "DELETE") {
        keys = [];
        return new Response(null, { status: 204 });
      }
      if (path === "/api/keys") {
        return jsonResponse({ data: keys });
      }
      throw new Error(`unexpected path ${path}`);
    });
    vi.stubGlobal("fetch", fetch);

    render(<EndpointPage />);

    await screen.findByRole("row", { name: /local-admin/i });
    fireEvent.click(screen.getByRole("button", { name: "Delete local-admin" }));

    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith("/api/keys/key-1", expect.objectContaining({ method: "DELETE" }));
    });
    expect(await screen.findByText("No API keys")).toBeInTheDocument();
  });

  it("renders an empty state when /api/keys returns no keys", async () => {
    const fetch = vi.fn(async () => jsonResponse({ data: [] }));
    vi.stubGlobal("fetch", fetch);

    render(<EndpointPage />);

    expect(await screen.findByText("No API keys")).toBeInTheDocument();
    expect(screen.getByText("Create a gateway key before routing protected client requests.")).toBeInTheDocument();
  });

  it("renders an error state and retries key loading", async () => {
    const responses = [
      jsonResponse({ error: "keys unavailable" }, { status: 500, statusText: "Internal Server Error" }),
      jsonResponse({ data: [existingKey] })
    ];
    const fetch = vi.fn(async () => responses.shift());
    vi.stubGlobal("fetch", fetch);

    render(<EndpointPage />);

    expect(await screen.findByText("Could not load API keys")).toBeInTheDocument();
    expect(screen.getByText("keys unavailable")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: "Retry" }));

    await screen.findByRole("row", { name: /local-admin/i });
    expect(fetch).toHaveBeenCalledTimes(2);
  });

  it("renders an auth-expired state for protected key APIs", async () => {
    const fetch = vi.fn(async () => jsonResponse({ error: "control-plane auth required" }, { status: 403 }));
    vi.stubGlobal("fetch", fetch);

    render(<EndpointPage />);

    expect(await screen.findByText("Authentication expired")).toBeInTheDocument();
    expect(screen.getByText("control-plane auth required")).toBeInTheDocument();
  });
});
