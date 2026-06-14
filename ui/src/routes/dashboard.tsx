import { createFileRoute } from "@tanstack/react-router";
import { UsageStats } from "@/components/usage/usage-stats";
import { RequestLogger } from "@/components/usage/request-logger";

export const Route = createFileRoute("/dashboard")({
  component: DashboardPage,
});

function DashboardPage() {
  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-semibold text-foreground">Dashboard</h1>
      <UsageStats period="today" hidePeriodSelector />
      <div className="flex flex-col gap-3">
        <h2 className="text-lg font-medium text-foreground">Recent requests</h2>
        <RequestLogger compact />
      </div>
    </div>
  );
}
