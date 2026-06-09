import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/providers")({
  component: ProvidersPage,
});

function ProvidersPage() {
  return <h1>Providers</h1>;
}
