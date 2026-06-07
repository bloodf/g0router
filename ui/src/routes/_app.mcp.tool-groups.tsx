import { createFileRoute } from "@tanstack/react-router";
import { ComingSoon } from "@/components/common/ComingSoon";
export const Route = createFileRoute("/_app/mcp/tool-groups")({
  component: () => <ComingSoon title="MCP Tool Groups" icon="workspaces" />,
});
