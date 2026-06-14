import { createFileRoute } from "@tanstack/react-router";
import { RequestLogger } from "@/components/usage/request-logger";

export const Route = createFileRoute("/logs")({
  component: LogsPage,
});

function LogsPage() {
  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-semibold text-foreground">Logs</h1>
      <RequestLogger />
    </div>
  );
}
