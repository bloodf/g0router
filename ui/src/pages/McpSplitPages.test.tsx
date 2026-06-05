import { fireEvent, render, screen, waitFor } from "@testing-library/react";
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
    expect(await screen.findByRole("table", { name: "MCP instances" })).toHaveTextContent("linear-tools");
    expect(screen.queryByRole("heading", { name: "Start OAuth" })).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Execute tool" })).not.toBeInTheDocument();
  });

  it("renders only account OAuth controls on MCP Accounts", async () => {
    stubMCPFetch();
    render(<McpAccountsPage />);

    expect(await screen.findByRole("heading", { level: 3, name: "MCP accounts" })).toBeInTheDocument();
    expect(await screen.findByRole("heading", { name: "Start OAuth" })).toBeInTheDocument();
    expect(await screen.findByText("ops@example.com")).toBeInTheDocument();
    expect(screen.queryByRole("table", { name: "MCP instances" })).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Execute tool" })).not.toBeInTheDocument();
  });

  it("starts MCP OAuth with resource URI discovery when Authorization URL is empty", async () => {
    const postBodies: unknown[] = [];
    const fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = String(input);
      if (path === getMcpClientsPath()) {
        return jsonResponse({ data: [] });
      }
      if (path === getMcpServersPath()) {
        return jsonResponse({ data: [instance] });
      }
      if (path === getMcpToolsPath()) {
        return jsonResponse({ data: [] });
      }
      if (path === getMcpAccountsPath("inst-1")) {
        return jsonResponse({ data: [] });
      }
      if (path === `${getMcpServersPath()}/inst-1/auth/start` && init?.method === "POST") {
        postBodies.push(JSON.parse(String(init.body)));
        return jsonResponse(
          {
            authorization_url: "https://auth.linear.example/authorize?state=unit",
            expires_at: "2026-06-04T12:00:00Z"
          },
          { status: 201 }
        );
      }
      return jsonResponse({ error: `missing ${path}` }, { status: 404 });
    });
    vi.stubGlobal("fetch", fetch);

    render(<McpAccountsPage />);

    expect(await screen.findByRole("heading", { level: 3, name: "MCP accounts" })).toBeInTheDocument();
    fireEvent.change(await screen.findByLabelText("Resource URI"), { target: { value: "https://mcp.linear.app" } });

    expect(screen.getByLabelText("Authorization URL")).not.toBeRequired();
    fireEvent.click(await screen.findByRole("button", { name: "Start OAuth" }));

    await waitFor(() => expect(postBodies).toHaveLength(1));
    expect(postBodies[0]).toMatchObject({
      authorization_url: "",
      redirect_uri: "http://localhost:3000/api/mcp/oauth/callback",
      resource_uri: "https://mcp.linear.app"
    });
  });

  it("blocks MCP OAuth start when Authorization URL and Resource URI are empty", async () => {
    const postBodies: unknown[] = [];
    const fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = String(input);
      if (path === getMcpClientsPath()) {
        return jsonResponse({ data: [] });
      }
      if (path === getMcpServersPath()) {
        return jsonResponse({ data: [instance] });
      }
      if (path === getMcpToolsPath()) {
        return jsonResponse({ data: [] });
      }
      if (path === getMcpAccountsPath("inst-1")) {
        return jsonResponse({ data: [] });
      }
      if (path === `${getMcpServersPath()}/inst-1/auth/start` && init?.method === "POST") {
        postBodies.push(JSON.parse(String(init.body)));
        return jsonResponse({ authorization_url: "https://auth.linear.example/authorize", expires_at: "2026-06-04T12:00:00Z" }, { status: 201 });
      }
      return jsonResponse({ error: `missing ${path}` }, { status: 404 });
    });
    vi.stubGlobal("fetch", fetch);

    render(<McpAccountsPage />);

    expect(await screen.findByRole("heading", { level: 3, name: "MCP accounts" })).toBeInTheDocument();
    fireEvent.click(await screen.findByRole("button", { name: "Start OAuth" }));

    expect(await screen.findByText("Authorization URL or Resource URI is required.")).toBeInTheDocument();
    expect(postBodies).toHaveLength(0);
  });

  it("renders only tool controls on MCP Tools", async () => {
    stubMCPFetch();
    render(<McpToolsPage />);

    expect(await screen.findByRole("heading", { level: 3, name: "MCP tools" })).toBeInTheDocument();
    expect(await screen.findByRole("heading", { name: "Execute tool" })).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Tools" }).closest("section")).toHaveTextContent("inst-1__search");
    expect(screen.queryByRole("table", { name: "MCP instances" })).not.toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Start OAuth" })).not.toBeInTheDocument();
  });

  it("submits parsed MCP instance launch fields and blocks malformed JSON", async () => {
    const postBodies: unknown[] = [];
    const fetch = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const path = String(input);
      if (path === getMcpClientsPath()) {
        return jsonResponse({ data: [] });
      }
      if (path === getMcpToolsPath()) {
        return jsonResponse({ data: [] });
      }
      if (path === getMcpAccountsPath("inst-1")) {
        return jsonResponse({ data: [] });
      }
      if (path === getMcpServersPath() && init?.method === "POST") {
        postBodies.push(JSON.parse(String(init.body)));
        return jsonResponse(
          {
            ...instance,
            ID: "inst-created",
            Name: "filesystem-tools",
            ServerKey: "filesystem",
            LaunchType: "command",
            Transport: "stdio"
          },
          { status: 201 }
        );
      }
      if (path === getMcpServersPath()) {
        return jsonResponse({ data: [instance] });
      }
      return jsonResponse({ error: `missing ${path}` }, { status: 404 });
    });
    vi.stubGlobal("fetch", fetch);

    render(<McpInstancesPage />);

    expect(await screen.findByRole("heading", { level: 3, name: "MCP instances" })).toBeInTheDocument();
    expect(screen.getByLabelText("Args JSON")).toBeInTheDocument();
    expect(screen.getByLabelText("Headers JSON")).toBeInTheDocument();
    expect(screen.getByLabelText("Env JSON")).toBeInTheDocument();
    expect(screen.getByLabelText("Working directory")).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Instance name"), { target: { value: "filesystem-tools" } });
    fireEvent.change(screen.getByLabelText("Server key"), { target: { value: "filesystem" } });
    fireEvent.change(screen.getByLabelText("Launch type"), { target: { value: "command" } });
    fireEvent.change(screen.getByLabelText("Command"), { target: { value: "node" } });
    fireEvent.change(screen.getByLabelText("Args JSON"), { target: { value: "[\"server.js\", \"--stdio\"]" } });
    fireEvent.change(screen.getByLabelText("Headers JSON"), { target: { value: "{\"Authorization\":\"Bearer secret-token\"}" } });
    fireEvent.change(screen.getByLabelText("Env JSON"), { target: { value: "{\"API_KEY\":\"secret-value\"}" } });
    fireEvent.change(screen.getByLabelText("Working directory"), { target: { value: "/srv/mcp" } });
    fireEvent.click(screen.getByRole("button", { name: "Create instance" }));

    await waitFor(() => expect(postBodies).toHaveLength(1));
    expect(postBodies[0]).toMatchObject({
      args: ["server.js", "--stdio"],
      command: "node",
      cwd: "/srv/mcp",
      env: { API_KEY: "secret-value" },
      headers: { Authorization: "Bearer secret-token" },
      launch_type: "command",
      name: "filesystem-tools",
      server_key: "filesystem",
      transport: "stdio"
    });

    fireEvent.change(screen.getByLabelText("Instance name"), { target: { value: "broken-tools" } });
    fireEvent.change(screen.getByLabelText("Server key"), { target: { value: "broken" } });
    fireEvent.change(screen.getByLabelText("Args JSON"), { target: { value: "[\"server.js\"]" } });
    fireEvent.change(screen.getByLabelText("Headers JSON"), { target: { value: "{\"Authorization\":" } });
    fireEvent.click(screen.getByRole("button", { name: "Create instance" }));

    expect(await screen.findByText("Headers JSON is invalid.")).toBeInTheDocument();
    expect(postBodies).toHaveLength(1);
  });
});
