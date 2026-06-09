import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/traffic")({
  component: TrafficPage,
});

function TrafficPage() {
  return <h1>Traffic</h1>;
}
