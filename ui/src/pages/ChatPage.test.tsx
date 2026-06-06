import { render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ChatPage } from "./ChatPage";

describe("ChatPage", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders playground with provider and model selectors", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const path = String(input);
        if (path === "/api/providers") {
          return new Response(
            JSON.stringify({
              data: [
                { id: "openai", inference: true, public_status: "supported" },
                { id: "anthropic", inference: true, public_status: "supported" }
              ]
            }),
            { headers: { "Content-Type": "application/json" } }
          );
        }
        if (path === "/api/providers/openai/models") {
          return new Response(
            JSON.stringify({
              data: [
                { id: "gpt-4o", object: "model", created: 0, owned_by: "openai" },
                { id: "gpt-4o-mini", object: "model", created: 0, owned_by: "openai" }
              ]
            }),
            { headers: { "Content-Type": "application/json" } }
          );
        }
        return new Response(JSON.stringify({ error: "missing" }), { status: 404 });
      })
    );

    render(<ChatPage />);

    await waitFor(() => {
      expect(screen.getByRole("combobox", { name: "Provider" })).toBeInTheDocument();
    });
    expect(screen.getByRole("combobox", { name: "Model" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Send" })).toBeInTheDocument();
    expect(screen.getByText("Start a conversation")).toBeInTheDocument();
  });
});
