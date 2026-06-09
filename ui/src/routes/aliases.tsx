import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/aliases")({
  component: AliasesPage,
});

function AliasesPage() {
  return <h1>Aliases</h1>;
}
