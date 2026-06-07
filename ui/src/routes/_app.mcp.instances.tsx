import { createFileRoute } from "@tanstack/react-router";
import { ComingSoon } from "@/components/common/ComingSoon";
export const Route = createFileRoute("/_app/mcp/instances")({
  component: () => <ComingSoon title="MCP Instances" description="Local MCP server processes and remote endpoints." icon="memory" />,
});
