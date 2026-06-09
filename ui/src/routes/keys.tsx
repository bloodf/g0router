import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/keys")({
  component: KeysPage,
});

function KeysPage() {
  return <h1>Keys</h1>;
}
