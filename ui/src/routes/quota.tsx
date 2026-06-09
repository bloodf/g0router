import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/quota")({
  component: QuotaPage,
});

function QuotaPage() {
  return <h1>Quota</h1>;
}
