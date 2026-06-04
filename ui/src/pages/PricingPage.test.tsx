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

    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith(`${getPricingPath()}/anthropic/claude-sonnet`, expect.objectContaining({ method: "DELETE" }));
    });
    expect(await screen.findByText("No pricing overrides")).toBeInTheDocument();
  });
});
