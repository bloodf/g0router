import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { getApiKeysPath } from "../api";
import { APIKeysPage } from "./APIKeysPage";

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
    ...init
  });
}

describe("APIKeysPage", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders API key management without endpoint-copy controls", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        jsonResponse({
          data: [
            {
              ID: "key-1",
              Name: "desktop-client",
              Prefix: "g0r_live",
              IsActive: true,
              LastUsedAt: null,
              CreatedAt: "2026-06-03T09:00:00Z"
            }
          ]
        })
      )
    );

    render(<APIKeysPage />);

    const row = await screen.findByRole("row", { name: /desktop-client g0r_live active/i });
    expect(within(row).getByText("g0r_live")).toBeInTheDocument();
    expect(screen.getByRole("heading", { level: 3, name: "API keys" })).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Copy chat completions endpoint" })).not.toBeInTheDocument();
    expect(screen.getByText("An API key is required to call the proxy")).toBeInTheDocument();
  });

  it("generates a key and shows the raw secret once", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (_input: RequestInfo | URL, init?: RequestInit) => {
        const method = init?.method ?? "GET";
        if (method === "POST") {
          return jsonResponse({
            key: { ID: "key-2", Name: "ci", Prefix: "g0r_live", IsActive: true, LastUsedAt: null, CreatedAt: "2026-06-03T09:00:00Z" },
            raw: "g0r_live_secret_value"
          });
        }
        return jsonResponse({ data: [] });
      })
    );

    render(<APIKeysPage />);

    fireEvent.change(await screen.findByLabelText("Key name"), { target: { value: "ci" } });
    fireEvent.click(screen.getByRole("button", { name: "Create key" } ));

    expect(await screen.findByText("g0r_live_secret_value")).toBeInTheDocument();
  });

  it("shows policy fields in key list and allows editing policy via PUT", async () => {
    const keyWithPolicy = {
      ID: "key-3",
      Name: "limited-key",
      Prefix: "g0r_live",
      IsActive: true,
      LastUsedAt: null,
      CreatedAt: "2026-06-03T09:00:00Z",
      expires_at: null,
      scopes: [],
      rate_limit_rpm: 60,
      rate_limit_tpm: null,
      daily_spend_cap_usd: 5.0
    };
    const fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = String(input);
      const method = init?.method ?? "GET";
      if (path === getApiKeysPath() && method === "GET") {
        return jsonResponse({ data: [keyWithPolicy] });
      }
      if (path === `${getApiKeysPath()}/key-3` && method === "PUT") {
        return jsonResponse({
          key: { ...keyWithPolicy, rate_limit_rpm: 120, daily_spend_cap_usd: 10.0 }
        });
      }
      throw new Error(`unexpected ${method} ${path}`);
    });
    vi.stubGlobal("fetch", fetch);

    render(<APIKeysPage />);

    const row = await screen.findByRole("row", { name: /limited-key/i });
    expect(within(row).getByText("limited-key")).toBeInTheDocument();

    // Edit policy
    fireEvent.click(within(row).getByRole("button", { name: /edit policy/i }));

    const rpmInput = await screen.findByLabelText("Rate limit RPM");
    expect(rpmInput).toHaveValue(60);

    fireEvent.change(rpmInput, { target: { value: "120" } });
    fireEvent.change(screen.getByLabelText("Daily spend cap (USD)"), { target: { value: "10" } });
    fireEvent.click(screen.getByRole("button", { name: /save policy/i }));

    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith(
        `${getApiKeysPath()}/key-3`,
        expect.objectContaining({
          method: "PUT",
          body: JSON.stringify({
            expires_at: null,
            scopes: [],
            rate_limit_rpm: 120,
            rate_limit_tpm: null,
            daily_spend_cap_usd: 10
          })
        })
      );
    });
  });
});
