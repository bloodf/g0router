import { createFileRoute } from "@tanstack/react-router";
import { CrudPage } from "@/components/common/CrudPage";
import { Progress } from "@/components/ui/progress";
import type { Team } from "@/lib/types";

const BUDGET_PERIOD_OPTIONS = [
  { label: "Daily", value: "daily" },
  { label: "Weekly", value: "weekly" },
  { label: "Monthly", value: "monthly" },
];

export const Route = createFileRoute("/_app/teams")({
  component: () => (
    <CrudPage<Team>
      title="Teams"
      description="Group virtual keys and members by team with shared budgets."
      icon="groups"
      endpoint="/api/teams"
      queryKey={["teams"]}
      emptyTitle="No teams yet"
      emptyDescription="Create a team to group virtual keys and members under a shared budget."
      fields={[
        { name: "name", label: "Name", required: true },
        { name: "budget_usd", label: "Budget ($)", type: "number" },
        {
          name: "budget_period",
          label: "Budget period",
          type: "select",
          options: BUDGET_PERIOD_OPTIONS,
        },
        { name: "rate_limit_rpm", label: "Rate limit (RPM)", type: "number" },
      ]}
      columns={[
        { header: "Name", accessorKey: "name" },
        {
          header: "Budget",
          cell: ({ row }) => {
            const t = row.original;
            if (!t.budget_usd) return "—";
            const pct = (t.budget_used_usd / t.budget_usd) * 100;
            return (
              <div className="min-w-[140px]">
                <div className="flex justify-between text-xs mb-0.5">
                  <span>
                    ${t.budget_used_usd.toFixed(2)} / ${t.budget_usd}
                  </span>
                </div>
                <Progress value={pct} className="h-1.5" />
              </div>
            );
          },
        },
      ]}
    />
  ),
});
