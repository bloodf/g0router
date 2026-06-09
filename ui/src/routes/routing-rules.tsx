import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/routing-rules")({
  component: RoutingRulesPage,
});

function RoutingRulesPage() {
  return <h1>Routing Rules</h1>;
}
