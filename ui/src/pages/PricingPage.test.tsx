import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { getPricingPath } from "../api";
import { PricingPage } from "./PricingPage";

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
    ...init
  });
}

function emptyResponse(init: ResponseInit = {}) {
  return new Response(null, init);
}

describe("PricingPage", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("shows the loading state while pricing overrides are fetched", () => {
    vi.stubGlobal("fetch", vi.fn(() => new Promise<Response>(() => undefined)));

    render(<PricingPage />);

    expect(screen.getByRole("status")).toHaveTextContent("Loading pricing overrides");
  });

  it("renders empty, error, and auth-expired states", async () => {
    const fetch = vi
      .fn()
      .mockResolvedValueOnce(jsonResponse({ data: [] }))
      .mockResolvedValueOnce(jsonResponse({ error: "pricing unavailable" }, { status: 500, statusText: "Server Error" }))
      .mockResolvedValueOnce(jsonResponse({ error: "control-plane auth required" }, { status: 403, statusText: "Forbidden" }));
    vi.stubGlobal("fetch", fetch);

    const { unmount } = render(<PricingPage />);
    expect(await screen.findByText("No pricing overrides")).toBeInTheDocument();

    unmount();
    render(<PricingPage />);
    expect(await screen.findByText("Could not load pricing")).toBeInTheDocument();
    expect(screen.getByText("pricing unavailable")).toBeInTheDocument();

    unmount();
    render(<PricingPage />);
    expect(await screen.findByText("Session expired")).toBeInTheDocument();
    expect(screen.getByText("control-plane auth required")).toBeInTheDocument();
  });

  it("lists pricing overrides from the management API", async () => {
    const fetch = vi.fn(async () =>
      jsonResponse({
        data: [{ Provider: "openai", Model: "gpt-4o-mini", InputCostPerToken: 0.000001, OutputCostPerToken: 0.000002 }]
      })
    );
    vi.stubGlobal("fetch", fetch);

    render(<PricingPage />);

    const row = await screen.findByRole("row", { name: /openai gpt-4o-mini/i });
    expect(within(row).getByText("0.000001")).toBeInTheDocument();
    expect(within(row).getByText("0.000002")).toBeInTheDocument();
  });

  it("creates and deletes pricing overrides through the API", async () => {
    let overrides: unknown[] = [];
    const fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = String(input);
      const method = init?.method ?? "GET";
      if (path === getPricingPath() && method === "GET") {
        return jsonResponse({ data: overrides });
      }
      if (path === getPricingPath() && method === "POST") {
        overrides = [{ Provider: "anthropic", Model: "claude-sonnet", InputCostPerToken: 0.000003, OutputCostPerToken: 0.000015 }];
        return jsonResponse(overrides[0], { status: 201 });
      }
      if (path === `${getPricingPath()}/anthropic/claude-sonnet` && method === "DELETE") {
        overrides = [];
        return emptyResponse({ status: 204 });
      }
      throw new Error(`unexpected ${method} ${path}`);
    });
    vi.stubGlobal("fetch", fetch);
    const confirm = vi.fn(() => true);
    vi.stubGlobal("confirm", confirm);

    render(<PricingPage />);

    await screen.findByText("No pricing overrides");
    fireEvent.change(screen.getByLabelText("Provider"), { target: { value: "anthropic" } });
    fireEvent.change(screen.getByLabelText("Model"), { target: { value: "claude-sonnet" } });
    fireEvent.change(screen.getByLabelText("Input cost per token"), { target: { value: "0.000003" } });
    fireEvent.change(screen.getByLabelText("Output cost per token"), { target: { value: "0.000015" } });
    fireEvent.click(screen.getByRole("button", { name: "Create override" }));

    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith(
        getPricingPath(),
        expect.objectContaining({
          body: JSON.stringify({
            provider: "anthropic",
            model: "claude-sonnet",
            input_cost_per_token: 0.000003,
            output_cost_per_token: 0.000015
          }),
          method: "POST"
        })
      );
    });
    const row = await screen.findByRole("row", { name: /anthropic claude-sonnet/i });
    fireEvent.click(within(row).getByRole("button", { name: "Delete anthropic claude-sonnet" }));

    expect(confirm).toHaveBeenCalledWith("Delete pricing override anthropic/claude-sonnet?");
    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith(`${getPricingPath()}/anthropic/claude-sonnet`, expect.objectContaining({ method: "DELETE" }));
    });
    expect(await screen.findByText("No pricing overrides")).toBeInTheDocument();
  });

  it("updates pricing overrides through the documented PUT endpoint", async () => {
    let overrides = [{ Provider: "openai", Model: "gpt-4o-mini", InputCostPerToken: 0.000001, OutputCostPerToken: 0.000002 }];
    const fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = String(input);
      const method = init?.method ?? "GET";
      if (path === getPricingPath() && method === "GET") {
        return jsonResponse({ data: overrides });
      }
      if (path === `${getPricingPath()}/openai/gpt-4o-mini` && method === "PUT") {
        overrides = [{ Provider: "openai", Model: "gpt-4o-mini", InputCostPerToken: 0.000004, OutputCostPerToken: 0.000008 }];
        return jsonResponse(overrides[0]);
      }
      throw new Error(`unexpected ${method} ${path}`);
    });
    vi.stubGlobal("fetch", fetch);

    render(<PricingPage />);

    const row = await screen.findByRole("row", { name: /openai gpt-4o-mini/i });
    fireEvent.click(within(row).getByRole("button", { name: "Edit openai gpt-4o-mini" }));
    fireEvent.change(screen.getByLabelText("Input cost per token"), { target: { value: "0.000004" } });
    fireEvent.change(screen.getByLabelText("Output cost per token"), { target: { value: "0.000008" } });
    fireEvent.click(screen.getByRole("button", { name: "Update override" }));

    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith(
        `${getPricingPath()}/openai/gpt-4o-mini`,
        expect.objectContaining({
          body: JSON.stringify({ input_cost_per_token: 0.000004, output_cost_per_token: 0.000008 }),
          method: "PUT"
        })
      );
    });
    const updatedRow = await screen.findByRole("row", { name: /openai gpt-4o-mini/i });
    expect(within(updatedRow).getByText("0.000004")).toBeInTheDocument();
    expect(within(updatedRow).getByText("0.000008")).toBeInTheDocument();
  });
});
