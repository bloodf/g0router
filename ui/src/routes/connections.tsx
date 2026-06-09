import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/connections")({
  component: ConnectionsPage,
});

function ConnectionsPage() {
  return <h1>Connections</h1>;
}
