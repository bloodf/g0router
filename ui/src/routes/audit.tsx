import { createFileRoute } from "@tanstack/react-router";

export const Route = createFileRoute("/audit")({
  component: AuditPage,
});

function AuditPage() {
  return <h1>Audit</h1>;
}
