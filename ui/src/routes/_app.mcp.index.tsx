import { createFileRoute } from "@tanstack/react-router";
import { ComingSoon } from "@/components/common/ComingSoon";
export const Route = createFileRoute("/_app/mcp/")({
  component: () => <ComingSoon title="MCP Overview" description="Connected MCP clients, tools and recent events." icon="hub" />,
});
