import { createFileRoute } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api/client";
import { PageHeader } from "@/components/common/PageHeader";
import { DataTable } from "@/components/common/DataTable";
import { ProviderIcon } from "@/components/common/ProviderIcon";
import { StatusBadge } from "@/components/common/StatusBadge";
import { TableSkeleton, ErrorState } from "@/components/common/Skeletons";
import type { Model } from "@/lib/types";

export const Route = createFileRoute("/_app/models")({
  component: ModelsPage,
});

function ModelsPage() {
  const { data, isLoading, isError, error, refetch } = useQuery<Model[]>({
    queryKey: ["models"],
    queryFn: () => apiFetch("/api/models"),
  });
  return (
    <div>
      <PageHeader
        title="Models"
        description="Aggregated catalog across every connected provider."
        icon="dataset"
      />
      {isLoading ? (
        <TableSkeleton rows={8} columns={6} />
      ) : isError ? (
        <ErrorState
          title="Couldn’t load models"
          error={error}
          onRetry={() => refetch()}
        />
      ) : (
        <DataTable
          columns={[
            {
              header: "Model",
              accessorKey: "name",
              cell: ({ row }) => (
                <div className="flex items-center gap-2">
                  <ProviderIcon provider={row.original.provider} size={20} />
                  <code className="font-mono text-xs">{row.original.name}</code>
                </div>
              ),
            },
            { header: "Provider", accessorKey: "provider" },
            {
              header: "Input ($/M)",
              accessorKey: "input_cost",
              cell: ({ row }) => `$${row.original.input_cost}`,
            },
            {
              header: "Output ($/M)",
              accessorKey: "output_cost",
              cell: ({ row }) => `$${row.original.output_cost}`,
            },
            {
              header: "Context",
              accessorKey: "context_window",
              cell: ({ row }) =>
                Intl.NumberFormat("en", { notation: "compact" }).format(
                  row.original.context_window,
                ),
            },
            {
              header: "Status",
              cell: ({ row }) => (
                <StatusBadge variant={row.original.is_disabled ? "muted" : "success"} dot>
                  {row.original.is_disabled ? "disabled" : "enabled"}
                </StatusBadge>
              ),
            },
          ]}
          data={data ?? []}
          initialVisibleRows={20}
          emptyTitle="No models"
          emptyDescription="Connect a provider to populate the model catalog."
          emptyIcon="dataset"
        />
      )}
    </div>
  );
}
