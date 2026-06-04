import { render, screen, within } from "@testing-library/react";
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
    const row = screen.getByRole("row", { name: /OpenAI primary openai operator@example.com oauth active/i });
    expect(within(row).getByText("operator@example.com")).toBeInTheDocument();
    expect(screen.getByRole("table", { name: "Provider connections" })).toBeInTheDocument();
    expect(screen.queryByRole("table", { name: "Provider contract" })).not.toBeInTheDocument();
    expect(screen.queryByText(/top-secret|provider-access-token|provider-api-key/i)).not.toBeInTheDocument();
  });
});
