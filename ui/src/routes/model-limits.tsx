import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/model-limits")({
  component: ModelLimitsPage,
});

function ModelLimitsPage() {
  return <h1>Model Limits</h1>;
}
