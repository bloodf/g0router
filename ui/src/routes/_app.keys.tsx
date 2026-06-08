import { createFileRoute } from "@tanstack/react-router";
import { CrudPage } from "@/components/common/CrudPage";
import { StatusBadge } from "@/components/common/StatusBadge";
import { CopyButton } from "@/components/common/CopyButton";
import { format } from "date-fns";
import type { ApiKey } from "@/lib/types";

export const Route = createFileRoute("/_app/keys")({
  component: () => (
    <CrudPage<ApiKey>
      title="API Keys"
      description="OpenAI-compatible keys for /v1 endpoints."
      icon="key"
      endpoint="/api/keys"
      queryKey={["keys"]}
      emptyTitle="No API keys yet"
      emptyDescription="Generate an OpenAI-compatible key to call the /v1 endpoints from your apps."
      fields={[
        { name: "name", label: "Name", required: true },
        { name: "rpm_limit", label: "RPM limit", type: "number" },
        { name: "tpm_limit", label: "TPM limit", type: "number" },
        { name: "daily_spend_cap", label: "Daily spend cap ($)", type: "number" },
      ]}
      columns={[
        { header: "Name", accessorKey: "name" },
        {
          header: "Prefix",
          cell: ({ row }) => (
            <div className="flex items-center gap-1">
              <code className="text-xs bg-surface-2 px-1.5 py-0.5 rounded">
                {row.original.prefix}…
              </code>
              {row.original.full_key && <CopyButton value={row.original.full_key} />}
            </div>
          ),
        },
        {
          header: "RPM",
          accessorKey: "rpm_limit",
          cell: ({ row }) => row.original.rpm_limit ?? "—",
        },
        {
          header: "Cap",
          cell: ({ row }) =>
            row.original.daily_spend_cap ? `$${row.original.daily_spend_cap}/d` : "—",
        },
        {
          header: "Expires",
          cell: ({ row }) =>
            row.original.expires_at
              ? format(new Date(row.original.expires_at), "MMM d, yyyy")
              : "Never",
        },
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
