import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { getProviderModelsPath, getProvidersPath } from "../api";
import { ModelsPage } from "./ModelsPage";

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
    ...init
  });
}

const providers = [
  {
    id: "openai",
    omp_id: "openai",
    router9_id: "openai",
    bifrost_id: "openai",
    auth_types: ["api_key"],
    refresh: false,
    registered_adapter: true,
    public_inference: true,
    direct_dispatch: true,
    inference: true,
    streaming: true,
    model_catalog: true,
    list_models: true,
    quota: false,
    public_status: "supported"
  },
  {
    id: "anthropic",
    omp_id: "anthropic",
    router9_id: "anthropic",
    bifrost_id: "anthropic",
    auth_types: ["api_key"],
    refresh: false,
    registered_adapter: true,
    public_inference: true,
    direct_dispatch: true,
    inference: true,
    streaming: true,
    model_catalog: true,
    list_models: true,
    quota: false,
    public_status: "supported"
  }
];

describe("ModelsPage", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("loads provider models and switches provider using management API routes", async () => {
    const fetch = vi.fn(async (input: RequestInfo | URL) => {
      const path = String(input);
      if (path === getProvidersPath()) {
        return jsonResponse({ data: providers });
      }
      if (path === getProviderModelsPath("openai")) {
        return jsonResponse({ data: [{ id: "gpt-4o", object: "model", created: 0, owned_by: "openai" }] });
      }
      if (path === getProviderModelsPath("anthropic")) {
        return jsonResponse({ data: [{ id: "claude-sonnet-4", object: "model", created: 0, owned_by: "anthropic" }] });
      }
      return jsonResponse({ error: `missing ${path}` }, { status: 404 });
    });
    vi.stubGlobal("fetch", fetch);

    render(<ModelsPage />);

    const row = await screen.findByRole("row", { name: /gpt-4o openai openai model/i });
    expect(within(row).getByText("gpt-4o")).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Provider"), { target: { value: "anthropic" } });

    expect(await screen.findByRole("row", { name: /claude-sonnet-4 anthropic anthropic model/i })).toBeInTheDocument();
    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith(getProviderModelsPath("anthropic"), expect.objectContaining({ credentials: "same-origin" }));
    });
  });

  it("shows an empty state when no model-capable providers are returned", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => jsonResponse({ data: [] })));

    render(<ModelsPage />);

    expect(await screen.findByText("No model-capable providers")).toBeInTheDocument();
  });

  it("shows auth-expired state when model loading is unauthorized", async () => {
    vi.stubGlobal("fetch", vi.fn(async () => jsonResponse({ error: "control-plane auth required" }, { status: 401 })));

    render(<ModelsPage />);

    expect(await screen.findByText("Session expired")).toBeInTheDocument();
    expect(screen.getByText("control-plane auth required")).toBeInTheDocument();
  });
});
