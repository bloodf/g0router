import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/combos")({
  component: CombosPage,
});

function CombosPage() {
  return <h1>Combos</h1>;
}
