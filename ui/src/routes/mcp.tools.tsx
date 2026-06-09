import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/mcp/tools")({
  component: McpToolsPage,
});

function McpToolsPage() {
  return <h1>MCP Tools</h1>;
}
