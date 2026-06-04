import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { getMcpAccountsPath, getMcpClientsPath, getMcpServersPath, getMcpToolsPath } from "../api";
import { McpAccountsPage } from "./McpAccountsPage";
import { McpInstancesPage } from "./McpInstancesPage";
import { McpToolsPage } from "./McpToolsPage";

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
    ...init
  });
}

const instance = {
  ID: "inst-1",
  Name: "linear-tools",
  ServerKey: "linear",
  LaunchType: "http",
  Transport: "streamable-http",
  URL: "https://mcp.linear.app/mcp",
  AccountLabel: "work",
  IsActive: true,
  HealthStatus: "healthy",
  ToolManifest: { tools: [{ name: "search" }] },
  CreatedAt: "2026-06-04T00:00:00Z",
  UpdatedAt: "2026-06-04T00:00:00Z"
};

function stubMCPFetch() {
  vi.stubGlobal(
    "fetch",
    vi.fn(async (input: RequestInfo | URL) => {
      const path = String(input);
      if (path === getMcpClientsPath()) {
        return jsonResponse({ data: [] });
      }
      if (path === getMcpServersPath()) {
        return jsonResponse({ data: [instance] });
      }
      if (path === getMcpToolsPath()) {
        return jsonResponse({ data: [{ type: "function", function: { name: "inst-1__search", description: "Search issues" } }] });
      }
      if (path === getMcpAccountsPath("inst-1")) {
        return jsonResponse({
          data: [
            {
              id: "acct-1",
              instance_id: "inst-1",
              account_label: "work",
              email: "ops@example.com",
              resource_uri: "https://mcp.linear.app"
            }
          ]
        });
      }
      return jsonResponse({ error: `missing ${path}` }, { status: 404 });
    })
  );
}

describe("split MCP dashboard pages", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("renders only instance controls on MCP Instances", async () => {
    stubMCPFetch();
    render(<McpInstancesPage />);

    expect(await screen.findByRole("heading", { level: 3, name: "MCP instances" })).toBeInTheDocument();
    expect(screen.getByRole("table", { name: "MCP instances" })).toHaveTextContent("linear-tools");
    expect(screen.queryByRole("heading", { name: "Start OAuth" })).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Execute tool" })).not.toBeInTheDocument();
  });

  it("renders only account OAuth controls on MCP Accounts", async () => {
    stubMCPFetch();
    render(<McpAccountsPage />);

    expect(await screen.findByRole("heading", { level: 3, name: "MCP accounts" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Start OAuth" })).toBeInTheDocument();
    expect(screen.getByText("ops@example.com")).toBeInTheDocument();
    expect(screen.queryByRole("table", { name: "MCP instances" })).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Execute tool" })).not.toBeInTheDocument();
  });

  it("renders only tool controls on MCP Tools", async () => {
    stubMCPFetch();
    render(<McpToolsPage />);

    expect(await screen.findByRole("heading", { level: 3, name: "MCP tools" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Execute tool" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Tools" }).closest("section")).toHaveTextContent("inst-1__search");
    expect(screen.queryByRole("table", { name: "MCP instances" })).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Start OAuth" })).not.toBeInTheDocument();
  });
});
