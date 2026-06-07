import { createFileRoute } from "@tanstack/react-router";
import { ComingSoon } from "@/components/common/ComingSoon";
export const Route = createFileRoute("/_app/mcp/accounts")({
  component: () => <ComingSoon title="MCP Accounts" icon="account_circle" />,
});
