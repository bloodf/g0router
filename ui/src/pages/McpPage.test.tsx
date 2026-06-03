import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import {
  getMcpAccountsPath,
  getMcpClientsPath,
  getMcpServersPath,
  getMcpToolsPath
} from "../api";
import { McpPage } from "./McpPage";

type FetchStub = (input: RequestInfo | URL, options?: RequestInit) => Promise<Response>;

function jsonResponse(body: unknown, init: ResponseInit = {}) {
  return new Response(JSON.stringify(body), {
    headers: { "Content-Type": "application/json" },
    ...init
  });
}

describe("McpPage", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("loads MCP clients, instances, tools, and accounts from API contracts without rendering secrets", async () => {
    const fetch = vi.fn(async (input: RequestInfo | URL) => {
      const path = String(input);
      if (path === getMcpClientsPath()) {
        return jsonResponse({
          data: [
            {
              ID: "client-1",
              Name: "filesystem",
              Transport: "stdio",
              Command: "mcp-filesystem",
              Env: { FILES_TOKEN: "super-secret-client-token", MODE: "readonly" },
              IsActive: true,
              HealthStatus: "healthy",
              ToolManifest: { tools: [{ name: "read" }] },
              CreatedAt: "2026-06-03T10:00:00Z"
            }
          ]
        });
      }
      if (path === getMcpServersPath()) {
        return jsonResponse({
          data: [
            {
              ID: "inst-1",
              Name: "atlassian-a",
              ServerKey: "atlassian",
              LaunchType: "http",
              Transport: "streamable-http",
              URL: "https://mcp.atlassian.com/mcp",
              Headers: { Authorization: "Bearer live-header-token", "X-Mode": "readonly" },
              Env: { API_TOKEN: "actual-env-token", MODE: "readonly" },
              AccountLabel: "work",
              IsActive: true,
              HealthStatus: "auth required",
              ToolManifest: { tools: [{ name: "search" }, { name: "issue" }] },
              CreatedAt: "2026-06-03T10:00:00Z",
              UpdatedAt: "2026-06-03T10:00:00Z"
            }
          ]
        });
      }
      if (path === getMcpToolsPath()) {
        return jsonResponse({
          data: [
            { type: "function", function: { name: "inst-1__search", description: "Search issues" } },
            { type: "function", function: { name: "client-1__read", description: "Read files" } }
          ]
        });
      }
      if (path === getMcpAccountsPath("inst-1")) {
        return jsonResponse({
          data: [
            {
              id: "acct-1",
              instance_id: "inst-1",
              account_label: "work",
              email: "ops@example.com",
              resource_uri: "https://mcp.atlassian.com",
              scopes: ["read:jira"],
              expires_at: "2026-06-03T12:00:00Z",
              access_token: "oauth-access-token"
            }
          ]
        });
      }
      return jsonResponse({ error: `unexpected ${path}` }, { status: 500 });
    });
    vi.stubGlobal("fetch", fetch);

    render(<McpPage />);

    expect(screen.getByRole("status")).toHaveTextContent("Loading MCP data");

    const instancesTable = await screen.findByRole("table", { name: "MCP instances" });
    expect(instancesTable.parentElement).toHaveClass("overflow-x-auto");
    const instanceRow = within(instancesTable).getByRole("row", { name: /atlassian-a/i });
    expect(within(instanceRow).getByText("http")).toBeInTheDocument();
    expect(within(instanceRow).getByText("streamable-http")).toBeInTheDocument();
    expect(within(instanceRow).getByText("work")).toBeInTheDocument();
    expect(within(instanceRow).getByText("2")).toBeInTheDocument();
    expect(within(instanceRow).getByText("auth required")).toBeInTheDocument();
    expect(screen.getByText("filesystem")).toBeInTheDocument();
    expect(screen.getByText("inst-1__search")).toBeInTheDocument();
    expect(screen.getByText("ops@example.com")).toBeInTheDocument();
    expect(screen.getAllByText("redacted").length).toBeGreaterThan(0);
    expect(screen.getByRole("heading", { name: "Accounts" }).closest("section")).toHaveClass("overflow-x-auto");
    expect(screen.getByRole("heading", { name: "Tools" }).closest("section")).toHaveClass("overflow-x-auto");

    expect(screen.queryByText("super-secret-client-token")).not.toBeInTheDocument();
    expect(screen.queryByText("Bearer live-header-token")).not.toBeInTheDocument();
    expect(screen.queryByText("actual-env-token")).not.toBeInTheDocument();
    expect(screen.queryByText("oauth-access-token")).not.toBeInTheDocument();
    expect(fetch).toHaveBeenCalledWith(getMcpClientsPath(), expect.any(Object));
    expect(fetch).toHaveBeenCalledWith(getMcpServersPath(), expect.any(Object));
    expect(fetch).toHaveBeenCalledWith(getMcpToolsPath(), expect.any(Object));
    expect(fetch).toHaveBeenCalledWith(getMcpAccountsPath("inst-1"), expect.any(Object));
  });

  it("renders empty, error, and auth-expired async states", async () => {
    const fetch = vi.fn<FetchStub>(async () => jsonResponse({ data: [] }));
    vi.stubGlobal("fetch", fetch);

    const emptyRender = render(<McpPage />);

    expect(await screen.findByText("No MCP data")).toBeInTheDocument();
    emptyRender.unmount();

    fetch.mockImplementation(async (input: RequestInfo | URL) => {
      if (String(input) === getMcpToolsPath()) {
        return jsonResponse({ error: "mcp runtime offline" }, { status: 500, statusText: "offline" });
      }
      return jsonResponse({ data: [] });
    });
    const errorRender = render(<McpPage />);

    expect(await screen.findByText("Could not load MCP gateway")).toBeInTheDocument();
    expect(screen.getByText("mcp runtime offline")).toBeInTheDocument();
    errorRender.unmount();

    fetch.mockImplementation(async () =>
      jsonResponse({ error: "control-plane auth required" }, { status: 401, statusText: "unauthorized" })
    );
    render(<McpPage />);

    expect(await screen.findByText("MCP session expired")).toBeInTheDocument();
    expect(screen.getByText("control-plane auth required")).toBeInTheDocument();
  });

  it("creates MCP instances and starts OAuth through real API paths", async () => {
    const instances = [
      {
        ID: "inst-1",
        Name: "atlassian-a",
        ServerKey: "atlassian",
        LaunchType: "http",
        Transport: "streamable-http",
        URL: "https://mcp.atlassian.com/mcp",
        AccountLabel: "work",
        IsActive: true,
        HealthStatus: "healthy",
        CreatedAt: "2026-06-03T10:00:00Z",
        UpdatedAt: "2026-06-03T10:00:00Z"
      }
    ];
    const fetch = vi.fn(async (input: RequestInfo | URL, options?: RequestInit) => {
      const path = String(input);
      if (path === getMcpClientsPath() || path === getMcpToolsPath()) {
        return jsonResponse({ data: [] });
      }
      if (path === getMcpAccountsPath("inst-1")) {
        return jsonResponse({ data: [] });
      }
      if (path === getMcpAccountsPath("inst-2")) {
        return jsonResponse({ data: [] });
      }
      if (path === getMcpServersPath() && options?.method === "POST") {
        instances.push({
          ID: "inst-2",
          Name: "linear-work",
          ServerKey: "linear",
          LaunchType: "http",
          Transport: "streamable-http",
          URL: "https://mcp.linear.app/mcp",
          AccountLabel: "engineering",
          IsActive: true,
          HealthStatus: "unknown",
          CreatedAt: "2026-06-03T10:05:00Z",
          UpdatedAt: "2026-06-03T10:05:00Z"
        });
        return jsonResponse(instances[1], { status: 201 });
      }
      if (path === getMcpServersPath()) {
        return jsonResponse({ data: instances });
      }
      if (path === "/api/mcp/instances/inst-1/auth/start") {
        return jsonResponse(
          {
            authorization_url: "https://auth.example/authorize?state=abc",
            expires_at: "2026-06-03T10:10:00Z"
          },
          { status: 201 }
        );
      }
      return jsonResponse({ error: `unexpected ${path}` }, { status: 500 });
    });
    vi.stubGlobal("fetch", fetch);

    render(<McpPage />);

    expect(await screen.findByRole("row", { name: /atlassian-a/i })).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Instance name"), { target: { value: "linear-work" } });
    fireEvent.change(screen.getByLabelText("Server key"), { target: { value: "linear" } });
    fireEvent.change(screen.getByLabelText("URL"), { target: { value: "https://mcp.linear.app/mcp" } });
    fireEvent.change(screen.getByLabelText("Account label"), { target: { value: "engineering" } });
    fireEvent.click(screen.getByRole("button", { name: "Create instance" }));

    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith(
        getMcpServersPath(),
        expect.objectContaining({
          body: expect.stringContaining("\"name\":\"linear-work\""),
          method: "POST"
        })
      );
    });
    expect(await screen.findByRole("row", { name: /linear-work/i })).toBeInTheDocument();

    fireEvent.change(screen.getByLabelText("Authorization URL"), {
      target: { value: "https://auth.example/authorize" }
    });
    fireEvent.change(screen.getByLabelText("Resource URI"), {
      target: { value: "https://mcp.atlassian.com" }
    });
    fireEvent.click(screen.getByRole("button", { name: "Start OAuth" }));

    await waitFor(() => {
      expect(fetch).toHaveBeenCalledWith(
        "/api/mcp/instances/inst-1/auth/start",
        expect.objectContaining({
          body: expect.stringContaining("\"resource_uri\":\"https://mcp.atlassian.com\""),
          method: "POST"
        })
      );
    });
    expect(screen.getByRole("link", { name: "Open authorization URL" })).toHaveAttribute(
      "href",
      "https://auth.example/authorize?state=abc"
    );
  });
});
