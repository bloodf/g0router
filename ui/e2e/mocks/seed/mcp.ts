import type { McpClient, McpInstance, McpTool, McpToolGroup } from "../../src/lib/types";

export function seedMcpClients(): McpClient[] {
  return [
    {
      ID: "mcp-client-1",
      Name: "Filesystem",
      Transport: "stdio",
      Command: "npx",
      Args: ["-y", "@modelcontextprotocol/server-filesystem"],
      Env: { ROOT: "/tmp" },
      IsActive: true,
      HealthStatus: "healthy",
      CreatedAt: new Date(Date.now() - 86400000 * 3).toISOString(),
    },
    {
      ID: "mcp-client-2",
      Name: "GitHub",
      Transport: "sse",
      URL: "https://api.github.com/mcp",
      IsActive: true,
      HealthStatus: "healthy",
      CreatedAt: new Date(Date.now() - 86400000 * 2).toISOString(),
    },
  ];
}

export function seedMcpInstances(): McpInstance[] {
  return [
    {
      ID: "mcp-instance-1",
      Name: "Filesystem Instance",
      Transport: "stdio",
      Command: "npx",
      Args: ["-y", "@modelcontextprotocol/server-filesystem"],
      IsActive: true,
      HealthStatus: "healthy",
      CreatedAt: new Date(Date.now() - 86400000).toISOString(),
    },
  ];
}

export function seedMcpTools(): McpTool[] {
  return [
    {
      type: "function",
      function: {
        name: "read_file",
        description: "Read a file",
        parameters: { type: "object", properties: { path: { type: "string" } }, required: ["path"] },
      },
    },
    {
      type: "function",
      function: {
        name: "write_file",
        description: "Write a file",
        parameters: { type: "object", properties: { path: { type: "string" }, content: { type: "string" } }, required: ["path", "content"] },
      },
    },
  ];
}

export function seedMcpToolGroups(): McpToolGroup[] {
  return [
    { id: 1, name: "File Operations", tool_ids: ["read_file", "write_file"], is_active: true, created_at: new Date(Date.now() - 86400000).toISOString() },
  ];
}
