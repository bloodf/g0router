import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { getAliasesPath } from "../api";
import { AliasesPage } from "./AliasesPage";

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
    ...init
  });
}

function emptyResponse(init: ResponseInit = {}) {
  return new Response(null, init);
}

describe("AliasesPage", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("lists aliases from the management API", async () => {
    const fetch = vi.fn(async () =>
      jsonResponse({ data: [{ Alias: "fast", Provider: "openai", Model: "gpt-4o-mini" }] })
    );
    vi.stubGlobal("fetch", fetch);

    render(<AliasesPage />);

    const row = await screen.findByRole("row", { name: /fast openai gpt-4o-mini/i });
    expect(within(row).getByText("fast")).toBeInTheDocument();
    expect(fetch).toHaveBeenCalledWith(getAliasesPath(), expect.objectContaining({ credentials: "same-origin" }));
  });

  it("creates and deletes aliases through the real API contract", async () => {
    let aliases: unknown[] = [];
    const fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = String(input);
      const method = init?.method ?? "GET";
      if (path === getAliasesPath() && method === "GET") {
        return jsonResponse({ data: aliases });
      }
      if (path === getAliasesPath() && method === "POST") {
        aliases = [{ Alias: "cheap", Provider: "groq", Model: "llama-3.3-70b-versatile" }];
        return jsonResponse(aliases[0], { status: 201 });
      }
      if (path === `${getAliasesPath()}/cheap` && method === "DELETE") {
        aliases = [];
        return emptyResponse({ status: 204 });
      }
      throw new Error(`unexpected ${method} ${path}`);
    });
    vi.stubGlobal("fetch", fetch);
    const confirm = vi.fn(() => true);
    vi.stubGlobal("confirm", confirm);

    render(<AliasesPage />);

    await screen.findByText("No model aliases");
    fireEvent.change(screen.getByLabelText("Alias"), { target: { value: "cheap" } });
    fireEvent.change(screen.getByLabelText("Provider"), { target: { value: "groq" } });
    fireEvent.change(screen.getByLabelText("Model"), { target: { value: "llama-3.3-70b-versatile" } });
    fireEvent.click(screen.getByRole("button", { name: "Create alias" }));

    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith(
        getAliasesPath(),
        expect.objectContaining({
          body: JSON.stringify({ alias: "cheap", provider: "groq", model: "llama-3.3-70b-versatile" }),
          method: "POST"
        })
      );
    });
    const row = await screen.findByRole("row", { name: /cheap groq/i });
    fireEvent.click(within(row).getByRole("button", { name: "Delete cheap" }));

    expect(confirm).toHaveBeenCalledWith("Delete alias cheap?");
    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith(`${getAliasesPath()}/cheap`, expect.objectContaining({ method: "DELETE" }));
    });
    expect(await screen.findByText("No model aliases")).toBeInTheDocument();
  });

  it("updates aliases through the documented PUT endpoint", async () => {
    let aliases = [{ Alias: "fast", Provider: "openai", Model: "gpt-4o-mini" }];
    const fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = String(input);
      const method = init?.method ?? "GET";
      if (path === getAliasesPath() && method === "GET") {
        return jsonResponse({ data: aliases });
      }
      if (path === `${getAliasesPath()}/fast` && method === "PUT") {
        aliases = [{ Alias: "fast", Provider: "anthropic", Model: "claude-sonnet-4" }];
        return jsonResponse(aliases[0]);
      }
      throw new Error(`unexpected ${method} ${path}`);
    });
    vi.stubGlobal("fetch", fetch);

    render(<AliasesPage />);

    const row = await screen.findByRole("row", { name: /fast openai gpt-4o-mini/i });
    fireEvent.click(within(row).getByRole("button", { name: "Edit fast" }));
    fireEvent.change(screen.getByLabelText("Provider"), { target: { value: "anthropic" } });
    fireEvent.change(screen.getByLabelText("Model"), { target: { value: "claude-sonnet-4" } });
    fireEvent.click(screen.getByRole("button", { name: "Update alias" }));

    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith(
        `${getAliasesPath()}/fast`,
        expect.objectContaining({
          body: JSON.stringify({ provider: "anthropic", model: "claude-sonnet-4" }),
          method: "PUT"
        })
      );
    });
    expect(await screen.findByRole("row", { name: /fast anthropic claude-sonnet-4/i })).toBeInTheDocument();
  });
});
