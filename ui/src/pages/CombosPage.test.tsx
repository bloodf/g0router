import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { getCombosPath } from "../api";
import { CombosPage } from "./CombosPage";

function makeCombo(overrides: Record<string, unknown> = {}) {
  return {
    ID: "combo-1",
    Name: "research-chain",
    Steps: [{ provider: "anthropic", model: "claude-sonnet-4" }],
    Strategy: "fallback" as const,
    IsActive: true,
    CreatedAt: "2026-06-03T05:00:00Z",
    UpdatedAt: "2026-06-03T05:00:00Z",
    ...overrides
  };
}

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
              Strategy: "round_robin",
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
    const confirm = vi.fn(() => true);
    vi.stubGlobal("confirm", confirm);

    render(<CombosPage />);

    const row = await screen.findByRole("row", { name: /research-chain/i });
    expect(within(row).getByText("anthropic / claude-sonnet-4")).toBeInTheDocument();
    expect(within(row).getByText("openai / gpt-4o")).toBeInTheDocument();
    expect(within(row).getByText("active")).toBeInTheDocument();
    expect(screen.getByRole("table", { name: "Combo routes" }).parentElement).toHaveClass("overflow-x-auto");

    fireEvent.click(within(row).getByRole("button", { name: "Delete research-chain" }));

    expect(confirm).toHaveBeenCalledWith("Delete combo research-chain?");
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
            Strategy: "fallback",
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
              Strategy: "fallback",
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
    fireEvent.change(screen.getByLabelText("Step 1 provider"), { target: { value: "gemini" } });
    fireEvent.change(screen.getByLabelText("Step 1 model"), { target: { value: "gemini-2.5-pro" } });
    fireEvent.click(screen.getByRole("button", { name: "Create combo" }));

    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith(
        getCombosPath(),
        expect.objectContaining({
          body: JSON.stringify({
            name: "fast-fallback",
            steps: [{ provider: "gemini", model: "gemini-2.5-pro" }],
            is_active: true,
            strategy: "fallback"
          }),
          credentials: "same-origin",
          method: "POST"
        })
      );
    });
    expect(await screen.findByRole("row", { name: /fast-fallback/i })).toBeInTheDocument();
  });

  it("updates combos through the documented combo PUT endpoint", async () => {
    let combos = [
      {
        ID: "combo-1",
        Name: "research-chain",
        Steps: [{ provider: "anthropic", model: "claude-sonnet-4" }],
        Strategy: "fallback",
        IsActive: true,
        CreatedAt: "2026-06-03T05:00:00Z",
        UpdatedAt: "2026-06-03T05:00:00Z"
      }
    ];
    const fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = String(input);
      const method = init?.method ?? "GET";
      if (path === getCombosPath() && method === "GET") {
        return jsonResponse({ data: combos });
      }
      if (path === `${getCombosPath()}/combo-1` && method === "PUT") {
        combos = [
          {
            ...combos[0],
            Name: "research-fallback",
            Steps: [{ provider: "openai", model: "gpt-4o" }],
            IsActive: false
          }
        ];
        return jsonResponse(combos[0]);
      }
      throw new Error(`unexpected ${method} ${path}`);
    });
    vi.stubGlobal("fetch", fetch);

    render(<CombosPage />);

    const row = await screen.findByRole("row", { name: /research-chain/i });
    fireEvent.click(within(row).getByRole("button", { name: "Edit research-chain" }));
    fireEvent.change(screen.getByLabelText("Combo name"), { target: { value: "research-fallback" } });
    fireEvent.change(screen.getByLabelText("Step 1 provider"), { target: { value: "openai" } });
    fireEvent.change(screen.getByLabelText("Step 1 model"), { target: { value: "gpt-4o" } });
    fireEvent.click(screen.getByLabelText("Active"));
    fireEvent.click(screen.getByRole("button", { name: "Update combo" }));

    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith(
        `${getCombosPath()}/combo-1`,
        expect.objectContaining({
          body: JSON.stringify({
            name: "research-fallback",
            steps: [{ provider: "openai", model: "gpt-4o" }],
            is_active: false,
            strategy: "fallback"
          }),
          credentials: "same-origin",
          method: "PUT"
        })
      );
    });
    const updatedRow = await screen.findByRole("row", { name: /research-fallback/i });
    expect(within(updatedRow).getByText("openai / gpt-4o")).toBeInTheDocument();
    expect(within(updatedRow).getByText("inactive")).toBeInTheDocument();
  });

  it("editing a 2-step combo preserves both steps on save", async () => {
    let combos = [
      {
        ID: "combo-multi",
        Name: "multi-chain",
        Steps: [
          { provider: "anthropic", model: "claude-sonnet-4" },
          { provider: "openai", model: "gpt-4o" }
        ],
        Strategy: "round_robin",
        IsActive: true,
        CreatedAt: "2026-06-03T05:00:00Z",
        UpdatedAt: "2026-06-03T05:00:00Z"
      }
    ];
    const fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = String(input);
      const method = init?.method ?? "GET";
      if (path === getCombosPath() && method === "GET") {
        return jsonResponse({ data: combos });
      }
      if (path === `${getCombosPath()}/combo-multi` && method === "PUT") {
        combos = [{ ...combos[0], Name: "multi-chain-updated" }];
        return jsonResponse(combos[0]);
      }
      throw new Error(`unexpected ${method} ${path}`);
    });
    vi.stubGlobal("fetch", fetch);

    render(<CombosPage />);

    const row = await screen.findByRole("row", { name: /multi-chain/i });
    fireEvent.click(within(row).getByRole("button", { name: "Edit multi-chain" }));

    // Both steps loaded into form
    expect(screen.getByLabelText("Step 1 provider")).toHaveValue("anthropic");
    expect(screen.getByLabelText("Step 1 model")).toHaveValue("claude-sonnet-4");
    expect(screen.getByLabelText("Step 2 provider")).toHaveValue("openai");
    expect(screen.getByLabelText("Step 2 model")).toHaveValue("gpt-4o");

    fireEvent.change(screen.getByLabelText("Combo name"), { target: { value: "multi-chain-updated" } });
    fireEvent.click(screen.getByRole("button", { name: "Update combo" }));

    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith(
        `${getCombosPath()}/combo-multi`,
        expect.objectContaining({
          body: JSON.stringify({
            name: "multi-chain-updated",
            steps: [
              { provider: "anthropic", model: "claude-sonnet-4" },
              { provider: "openai", model: "gpt-4o" }
            ],
            is_active: true,
            strategy: "round_robin"
          }),
          credentials: "same-origin",
          method: "PUT"
        })
      );
    });
  });

  it("user can add a second step to a new combo", async () => {
    const fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = String(input);
      const method = init?.method ?? "GET";
      if (path === getCombosPath() && method === "GET" && fetch.mock.calls.length === 1) {
        return jsonResponse({ data: [] });
      }
      if (path === getCombosPath() && method === "POST") {
        return jsonResponse(
          {
            ID: "combo-new",
            Name: "two-step",
            Steps: [
              { provider: "anthropic", model: "claude-haiku-4" },
              { provider: "openai", model: "gpt-4o-mini" }
            ],
            Strategy: "fallback",
            IsActive: true,
            CreatedAt: "2026-06-03T06:00:00Z",
            UpdatedAt: "2026-06-03T06:00:00Z"
          },
          { status: 201 }
        );
      }
      if (path === getCombosPath() && method === "GET") {
        return jsonResponse({
          data: [
            {
              ID: "combo-new",
              Name: "two-step",
              Steps: [
                { provider: "anthropic", model: "claude-haiku-4" },
                { provider: "openai", model: "gpt-4o-mini" }
              ],
              Strategy: "fallback",
              IsActive: true,
              CreatedAt: "2026-06-03T06:00:00Z",
              UpdatedAt: "2026-06-03T06:00:00Z"
            }
          ]
        });
      }
      throw new Error(`unexpected ${method} ${path}`);
    });
    vi.stubGlobal("fetch", fetch);

    render(<CombosPage />);

    await screen.findByText("No combo routes configured");
    fireEvent.change(screen.getByLabelText("Combo name"), { target: { value: "two-step" } });
    fireEvent.change(screen.getByLabelText("Step 1 provider"), { target: { value: "anthropic" } });
    fireEvent.change(screen.getByLabelText("Step 1 model"), { target: { value: "claude-haiku-4" } });

    fireEvent.click(screen.getByRole("button", { name: "+ Add step" }));

    expect(screen.getByLabelText("Step 2 provider")).toBeInTheDocument();
    fireEvent.change(screen.getByLabelText("Step 2 provider"), { target: { value: "openai" } });
    fireEvent.change(screen.getByLabelText("Step 2 model"), { target: { value: "gpt-4o-mini" } });

    fireEvent.click(screen.getByRole("button", { name: "Create combo" }));

    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith(
        getCombosPath(),
        expect.objectContaining({
          body: JSON.stringify({
            name: "two-step",
            steps: [
              { provider: "anthropic", model: "claude-haiku-4" },
              { provider: "openai", model: "gpt-4o-mini" }
            ],
            is_active: true,
            strategy: "fallback"
          }),
          credentials: "same-origin",
          method: "POST"
        })
      );
    });
    const resultRow = await screen.findByRole("row", { name: /two-step/i });
    expect(within(resultRow).getByText("anthropic / claude-haiku-4")).toBeInTheDocument();
    expect(within(resultRow).getByText("openai / gpt-4o-mini")).toBeInTheDocument();
  });

  it("selecting a strategy submits it on create", async () => {
    const fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = String(input);
      const method = init?.method ?? "GET";
      if (path === getCombosPath() && method === "GET" && fetch.mock.calls.length === 1) {
        return jsonResponse({ data: [] });
      }
      if (path === getCombosPath() && method === "POST") {
        return jsonResponse(
          { ID: "combo-rr", Name: "rr-combo", Steps: [{ provider: "openai", model: "gpt-4o" }], Strategy: "round_robin", IsActive: true, CreatedAt: "2026-06-05T00:00:00Z", UpdatedAt: "2026-06-05T00:00:00Z" },
          { status: 201 }
        );
      }
      if (path === getCombosPath() && method === "GET") {
        return jsonResponse({ data: [{ ID: "combo-rr", Name: "rr-combo", Steps: [{ provider: "openai", model: "gpt-4o" }], Strategy: "round_robin", IsActive: true, CreatedAt: "2026-06-05T00:00:00Z", UpdatedAt: "2026-06-05T00:00:00Z" }] });
      }
      throw new Error(`unexpected ${method} ${path}`);
    });
    vi.stubGlobal("fetch", fetch);

    render(<CombosPage />);

    await screen.findByText("No combo routes configured");
    fireEvent.change(screen.getByLabelText("Combo name"), { target: { value: "rr-combo" } });
    fireEvent.change(screen.getByLabelText("Step 1 provider"), { target: { value: "openai" } });
    fireEvent.change(screen.getByLabelText("Step 1 model"), { target: { value: "gpt-4o" } });
    fireEvent.change(screen.getByLabelText("Strategy"), { target: { value: "round_robin" } });
    fireEvent.click(screen.getByRole("button", { name: "Create combo" }));

    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith(
        getCombosPath(),
        expect.objectContaining({
          body: JSON.stringify({ name: "rr-combo", steps: [{ provider: "openai", model: "gpt-4o" }], is_active: true, strategy: "round_robin" }),
          method: "POST"
        })
      );
    });
    const row = await screen.findByRole("row", { name: /rr-combo/i });
    expect(within(row).getByText("round_robin")).toBeInTheDocument();
  });

  it("editing a combo preserves and loads its strategy", async () => {
    const combo = makeCombo({ Strategy: "least_used" });
    const fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = String(input);
      const method = init?.method ?? "GET";
      if (path === getCombosPath() && method === "GET") {
        return jsonResponse({ data: [combo] });
      }
      if (path === `${getCombosPath()}/combo-1` && method === "PUT") {
        return jsonResponse({ ...combo, Strategy: "least_used" });
      }
      throw new Error(`unexpected ${method} ${path}`);
    });
    vi.stubGlobal("fetch", fetch);

    render(<CombosPage />);

    const row = await screen.findByRole("row", { name: /research-chain/i });
    expect(within(row).getByText("least_used")).toBeInTheDocument();

    fireEvent.click(within(row).getByRole("button", { name: "Edit research-chain" }));
    expect(screen.getByLabelText("Strategy")).toHaveValue("least_used");

    fireEvent.click(screen.getByRole("button", { name: "Update combo" }));

    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith(
        `${getCombosPath()}/combo-1`,
        expect.objectContaining({
          body: JSON.stringify({ name: "research-chain", steps: [{ provider: "anthropic", model: "claude-sonnet-4" }], is_active: true, strategy: "least_used" }),
          method: "PUT"
        })
      );
    });
  });

  it("the list shows strategy for each combo", async () => {
    const fetch = vi.fn(async () =>
      jsonResponse({
        data: [
          makeCombo({ ID: "c1", Name: "chain-a", Strategy: "fallback" }),
          makeCombo({ ID: "c2", Name: "chain-b", Strategy: "auto" })
        ]
      })
    );
    vi.stubGlobal("fetch", fetch);

    render(<CombosPage />);

    const rowA = await screen.findByRole("row", { name: /chain-a/i });
    const rowB = await screen.findByRole("row", { name: /chain-b/i });
    expect(within(rowA).getByText("fallback")).toBeInTheDocument();
    expect(within(rowB).getByText("auto")).toBeInTheDocument();
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
