import { createFileRoute } from "@tanstack/react-router";
import { ComingSoon } from "@/components/common/ComingSoon";
export const Route = createFileRoute("/_app/mcp/tools")({
  component: () => <ComingSoon title="MCP Tools" icon="build" />,
});
