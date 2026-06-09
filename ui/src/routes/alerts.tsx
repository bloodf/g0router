import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/alerts")({
  component: AlertsPage,
});

function AlertsPage() {
  return <h1>Alerts</h1>;
}
