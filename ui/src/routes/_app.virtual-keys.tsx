import { createFileRoute } from "@tanstack/react-router";
import { CrudPage } from "@/components/common/CrudPage";
import { StatusBadge } from "@/components/common/StatusBadge";
import { Progress } from "@/components/ui/progress";
import type { VirtualKey } from "@/lib/types";

export const Route = createFileRoute("/_app/virtual-keys")({
  component: () => (
    <CrudPage<VirtualKey>
      title="Virtual Keys"
      description="Per-team/per-app keys with budgets and rate limits."
      icon="vpn_key"
      endpoint="/api/virtual-keys"
      queryKey={["virtual-keys"]}
      emptyTitle="No virtual keys yet"
      emptyDescription="Issue a virtual key to scope budgets and rate limits per team, app, or environment."
      fields={[
        { name: "name", label: "Name", required: true },
        { name: "budget_usd", label: "Budget ($)", type: "number" },
        {
          name: "budget_period",
          label: "Period",
          type: "select",
          options: [
            { label: "Daily", value: "daily" },
            { label: "Weekly", value: "weekly" },
            { label: "Monthly", value: "monthly" },
          ],
        },
        { name: "rate_limit_rpm", label: "RPM", type: "number" },
        { name: "rate_limit_tpm", label: "TPM", type: "number" },
      ]}
      columns={[
        { header: "Name", accessorKey: "name" },
        {
          header: "Prefix",
          accessorKey: "prefix",
          cell: ({ row }) => (
            <code className="text-xs bg-surface-2 px-1.5 py-0.5 rounded">
              {row.original.prefix}
            </code>
          ),
        },
        {
          header: "Budget",
          cell: ({ row }) => {
            const v = row.original;
            if (!v.budget_usd) return "—";
            const pct = (v.budget_used_usd / v.budget_usd) * 100;
            return (
              <div className="min-w-[140px]">
                <div className="flex justify-between text-xs mb-0.5">
                  <span>
                    ${v.budget_used_usd.toFixed(2)} / ${v.budget_usd}
                  </span>
                  <span className="text-text-muted">{pct.toFixed(0)}%</span>
                </div>
                <Progress value={pct} className="h-1.5" />
              </div>
            );
          },
        },
        { header: "Period", accessorKey: "budget_period" },
        {
          header: "Status",
          cell: ({ row }) => (
            <StatusBadge variant={row.original.is_active ? "success" : "muted"} dot>
              {row.original.is_active ? "active" : "inactive"}
            </StatusBadge>
          ),
        },
      ]}
    />
  ),
});
