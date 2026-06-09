import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/usage")({
  component: UsagePage,
});

function UsagePage() {
  return <h1>Usage</h1>;
}
