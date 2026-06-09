import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/console")({
  component: ConsolePage,
});

function ConsolePage() {
  return <h1>Console</h1>;
}
