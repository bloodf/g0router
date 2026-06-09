import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/prompts")({
  component: PromptsPage,
});

function PromptsPage() {
  return <h1>Prompts</h1>;
}
