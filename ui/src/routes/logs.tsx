import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/logs")({
  component: LogsPage,
});

function LogsPage() {
  return <h1>Logs</h1>;
}
