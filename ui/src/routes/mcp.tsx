import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/mcp")({
  component: McpPage,
});

function McpPage() {
  return <h1>MCP</h1>;
}
