import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { DiagnosticsPage } from "./DiagnosticsPage";

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
    ...init
  });
}

describe("DiagnosticsPage", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("summarizes API contract health without exposing secrets", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const path = String(input);
        if (path === "/api/providers") {
          return jsonResponse({ data: [{ id: "openai", public_status: "supported" }] });
        }
        if (path === "/api/settings") {
          return jsonResponse({ RequireAPIKey: true, RTKEnabled: true, CavemanEnabled: false, CavemanLevel: "full", EnableRequestLogs: true, ProxyURL: "", DataDir: "/tmp/g0router" });
        }
        if (path === "/api/connections") {
          return jsonResponse({ data: [{ ID: "conn-1", Provider: "openai", IsActive: true, APIKey: "should-not-render" }] });
        }
        if (path === "/api/mcp/instances") {
          return jsonResponse({ data: [{ ID: "mcp-1", Name: "docs", HealthStatus: "healthy", IsActive: true }] });
        }
        if (path === "/api/logs?limit=1&offset=0") {
          return jsonResponse({ object: "list", data: [], limit: 1, offset: 0 });
        }
        return jsonResponse({ error: `missing ${path}` }, { status: 404 });
      })
    );

    render(<DiagnosticsPage />);

    expect(await screen.findByText("Control plane protected")).toBeInTheDocument();
    expect(screen.getByText("1 providers")).toBeInTheDocument();
    expect(screen.getByText("1 active connections")).toBeInTheDocument();
    expect(screen.getByText("1 MCP instances")).toBeInTheDocument();
    expect(screen.queryByText("should-not-render")).not.toBeInTheDocument();
  });
});
