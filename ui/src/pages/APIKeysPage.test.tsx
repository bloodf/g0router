import { render, screen, within } from "@testing-library/react";
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
  });
});
