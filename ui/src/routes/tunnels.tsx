import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/tunnels")({
  component: TunnelsPage,
});

function TunnelsPage() {
  return <h1>Tunnels</h1>;
}
