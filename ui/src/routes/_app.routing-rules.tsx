import { createFileRoute } from "@tanstack/react-router";
import { CrudPage } from "@/components/common/CrudPage";
import { StatusBadge } from "@/components/common/StatusBadge";
import type { RoutingRule } from "@/lib/types";

export const Route = createFileRoute("/_app/routing-rules")({
  component: () => (
    <CrudPage<RoutingRule>
      title="Routing Rules"
      description="Match-by-condition routing in priority order."
      icon="alt_route"
      endpoint="/api/routing-rules"
      queryKey={["routing-rules"]}
      emptyTitle="No routing rules yet"
      emptyDescription="Create a rule to route requests by model, key, or header to a specific provider."
      fields={[
        { name: "name", label: "Name", required: true },
        { name: "priority", label: "Priority", type: "number" },
        {
          name: "cond_field",
          label: "Condition field",
          required: true,
          placeholder: "e.g. model, provider, header",
        },
        {
          name: "cond_operator",
          label: "Condition operator",
          type: "select",
          required: true,
          options: [
            { label: "equals", value: "equals" },
            { label: "contains", value: "contains" },
            { label: "starts_with", value: "starts_with" },
            { label: "regex", value: "regex" },
          ],
        },
        { name: "cond_value", label: "Condition value", required: true },
        { name: "target_provider", label: "Target provider", required: true },
        { name: "target_model", label: "Target model" },
      ]}
      columns={[
        { header: "#", accessorKey: "priority" },
        { header: "Name", accessorKey: "name" },
        {
          header: "Condition",
          cell: ({ row }) => (
            <code className="text-xs">
              {row.original.cond_field} {row.original.cond_operator} "
              {row.original.cond_value}"
            </code>
          ),
        },
        { header: "Target", accessorKey: "target_provider" },
        {
          header: "Status",
          cell: ({ row }) => (
            <StatusBadge variant={row.original.is_active ? "success" : "muted"} dot>
              {row.original.is_active ? "on" : "off"}
            </StatusBadge>
          ),
        },
      ]}
    />
  ),
});
