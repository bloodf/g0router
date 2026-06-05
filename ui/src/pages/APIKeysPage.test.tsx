import { fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
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
    fireEvent.click(screen.getByRole("button", { name: "Create key" }));

    expect(await screen.findByText("g0r_live_secret_value")).toBeInTheDocument();
  });
});
