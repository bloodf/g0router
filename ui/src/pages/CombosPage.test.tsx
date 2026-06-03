import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { getCombosPath } from "../api";
import { CombosPage } from "./CombosPage";

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
    ...init
  });
}

function emptyResponse(init: ResponseInit = {}) {
  return new Response(null, init);
}

describe("CombosPage", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("shows the loading state while combos are fetched", () => {
    vi.stubGlobal("fetch", vi.fn(() => new Promise<Response>(() => undefined)));

    render(<CombosPage />);

    expect(screen.getByRole("status")).toHaveTextContent("Loading combos");
  });

  it("renders the empty state when the combos API has no rows", async () => {
    const fetch = vi.fn(async () => jsonResponse({ data: [] }));
    vi.stubGlobal("fetch", fetch);

    render(<CombosPage />);

    expect(await screen.findByText("No combo routes configured")).toBeInTheDocument();
    expect(screen.getByLabelText("Combo name")).toBeInTheDocument();
    expect(fetch).toHaveBeenCalledWith(getCombosPath(), expect.objectContaining({ credentials: "same-origin" }));
  });

  it("renders API combos and deletes a combo through the real combo endpoint", async () => {
    const fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = String(input);
      const method = init?.method ?? "GET";
      if (path === getCombosPath() && method === "GET" && fetch.mock.calls.length === 1) {
        return jsonResponse({
          data: [
            {
              ID: "combo-1",
              Name: "research-chain",
              Steps: [
                { provider: "anthropic", model: "claude-sonnet-4" },
                { provider: "openai", model: "gpt-4o" }
              ],
              IsActive: true,
              CreatedAt: "2026-06-03T05:00:00Z",
              UpdatedAt: "2026-06-03T05:00:00Z"
            }
          ]
        });
      }
      if (path === `${getCombosPath()}/combo-1` && method === "DELETE") {
        return emptyResponse({ status: 204 });
      }
      if (path === getCombosPath() && method === "GET") {
        return jsonResponse({ data: [] });
      }
      throw new Error(`unexpected ${method} ${path}`);
    });
    vi.stubGlobal("fetch", fetch);

    render(<CombosPage />);

    const row = await screen.findByRole("row", { name: /research-chain/i });
    expect(within(row).getByText("anthropic / claude-sonnet-4")).toBeInTheDocument();
    expect(within(row).getByText("openai / gpt-4o")).toBeInTheDocument();
    expect(within(row).getByText("active")).toBeInTheDocument();

    fireEvent.click(within(row).getByRole("button", { name: "Delete research-chain" }));

    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith(
        `${getCombosPath()}/combo-1`,
        expect.objectContaining({ credentials: "same-origin", method: "DELETE" })
      );
    });
    expect(await screen.findByText("No combo routes configured")).toBeInTheDocument();
  });

  it("creates a combo with the management API request body", async () => {
    const fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = String(input);
      const method = init?.method ?? "GET";
      if (path === getCombosPath() && method === "GET" && fetch.mock.calls.length === 1) {
        return jsonResponse({ data: [] });
      }
      if (path === getCombosPath() && method === "POST") {
        return jsonResponse(
          {
            ID: "combo-2",
            Name: "fast-fallback",
            Steps: [{ provider: "gemini", model: "gemini-2.5-pro" }],
            IsActive: true,
            CreatedAt: "2026-06-03T05:10:00Z",
            UpdatedAt: "2026-06-03T05:10:00Z"
          },
          { status: 201 }
        );
      }
      if (path === getCombosPath() && method === "GET") {
        return jsonResponse({
          data: [
            {
              ID: "combo-2",
              Name: "fast-fallback",
              Steps: [{ provider: "gemini", model: "gemini-2.5-pro" }],
              IsActive: true,
              CreatedAt: "2026-06-03T05:10:00Z",
              UpdatedAt: "2026-06-03T05:10:00Z"
            }
          ]
        });
      }
      throw new Error(`unexpected ${method} ${path}`);
    });
    vi.stubGlobal("fetch", fetch);

    render(<CombosPage />);

    await screen.findByText("No combo routes configured");
    fireEvent.change(screen.getByLabelText("Combo name"), { target: { value: "fast-fallback" } });
    fireEvent.change(screen.getByLabelText("Step provider"), { target: { value: "gemini" } });
    fireEvent.change(screen.getByLabelText("Step model"), { target: { value: "gemini-2.5-pro" } });
    fireEvent.click(screen.getByRole("button", { name: "Create combo" }));

    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith(
        getCombosPath(),
        expect.objectContaining({
          body: JSON.stringify({
            name: "fast-fallback",
            steps: [{ provider: "gemini", model: "gemini-2.5-pro" }],
            is_active: true
          }),
          credentials: "same-origin",
          method: "POST"
        })
      );
    });
    expect(await screen.findByRole("row", { name: /fast-fallback/i })).toBeInTheDocument();
  });

  it("renders recoverable errors and auth-expired errors", async () => {
    const fetch = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ error: "combos unavailable" }, { status: 500, statusText: "Server Error" }))
      .mockResolvedValueOnce(jsonResponse({ error: "control-plane auth required" }, { status: 401, statusText: "Unauthorized" }));
    vi.stubGlobal("fetch", fetch);

    const { unmount } = render(<CombosPage />);

    expect(await screen.findByText("Could not load combos")).toBeInTheDocument();
    expect(screen.getByText("combos unavailable")).toBeInTheDocument();

    unmount();
    render(<CombosPage />);

    expect(await screen.findByText("Session expired")).toBeInTheDocument();
    expect(screen.getByText("control-plane auth required")).toBeInTheDocument();
  });
});
