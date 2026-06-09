import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/guardrails")({
  component: GuardrailsPage,
});

function GuardrailsPage() {
  return <h1>Guardrails</h1>;
}
