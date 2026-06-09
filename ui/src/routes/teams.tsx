import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/teams")({
  component: TeamsPage,
});

function TeamsPage() {
  return <h1>Teams</h1>;
}
