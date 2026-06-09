import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/models")({
  component: ModelsPage,
});

function ModelsPage() {
  return <h1>Models</h1>;
}
