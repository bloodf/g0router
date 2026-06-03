import { render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { QuotaPage } from "./QuotaPage";

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

describe("QuotaPage", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("fetches quota-capable providers through /api/usage/quota/:provider", async () => {
    const fetch = stubFetch({
      "/api/providers": jsonResponse({
        data: [
          { id: "openai", quota: true, public_status: "supported", auth_types: ["api_key"] },
          { id: "anthropic", quota: false, public_status: "supported", auth_types: ["oauth"] },
          { id: "openai/team", quota: true, public_status: "supported", auth_types: ["api_key"] }
        ]
      }),
      "/api/usage/quota/openai": jsonResponse({ Provider: "openai", Limit: 1000, Used: 250, Remaining: 750 }),
      "/api/usage/quota/openai%2Fteam": jsonResponse({
        Provider: "openai/team",
        Limit: 2000,
        Used: 1900,
        Remaining: 100
      })
    });

    render(<QuotaPage />);

    expect(await screen.findByText("openai")).toBeInTheDocument();

    const teamRow = screen.getByRole("article", { name: /openai\/team/i });
    expect(within(teamRow).getByText("1,900 of 2,000 used")).toBeInTheDocument();
    expect(within(teamRow).getByText("100 remaining")).toBeInTheDocument();

    await waitFor(() => {
      const paths = fetch.mock.calls.map(([path]) => path);
      expect(paths).toContain("/api/providers");
      expect(paths).toContain("/api/usage/quota/openai");
      expect(paths).toContain("/api/usage/quota/openai%2Fteam");
      expect(paths).not.toContain("/api/quota");
      expect(paths).not.toContain("/api/usage/quota/anthropic");
    });
  });

  it("shows loading while provider quota discovery is pending", () => {
    vi.stubGlobal("fetch", vi.fn(() => new Promise<Response>(() => undefined)));

    render(<QuotaPage />);

    expect(screen.getByRole("status")).toHaveTextContent("Loading quota data");
  });

  it("shows an empty state when no providers expose quota", async () => {
    const fetch = stubFetch({
      "/api/providers": jsonResponse({
        data: [
          { id: "anthropic", quota: false, public_status: "supported", auth_types: ["oauth"] },
          { id: "ollama", quota: false, public_status: "local", auth_types: ["noauth"] }
        ]
      })
    });

    render(<QuotaPage />);

    expect(await screen.findByText("No quota-capable providers")).toBeInTheDocument();
    expect(fetch).toHaveBeenCalledTimes(1);
  });

  it("shows an error state when quota fetch fails", async () => {
    stubFetch({
      "/api/providers": jsonResponse({
        data: [{ id: "openai", quota: true, public_status: "supported", auth_types: ["api_key"] }]
      }),
      "/api/usage/quota/openai": jsonResponse({ error: "quota fetcher not found" }, { status: 404 })
    });

    render(<QuotaPage />);

    expect(await screen.findByText("Quota data unavailable")).toBeInTheDocument();
    expect(screen.getByText("quota fetcher not found")).toBeInTheDocument();
  });

  it("shows auth-expired state when provider discovery is rejected", async () => {
    stubFetch({
      "/api/providers": jsonResponse({ error: "control-plane auth required" }, { status: 401 })
    });

    render(<QuotaPage />);

    expect(await screen.findByText("Session expired")).toBeInTheDocument();
    expect(screen.getByText("control-plane auth required")).toBeInTheDocument();
  });
});
