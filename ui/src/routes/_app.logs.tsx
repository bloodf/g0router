import { createFileRoute } from "@tanstack/react-router";
import { useQuery } from "@tanstack/react-query";
import { apiFetch, normalizeListResponse } from "@/lib/api/client";
import { PageHeader } from "@/components/common/PageHeader";
import { DataTable } from "@/components/common/DataTable";
import { StatusBadge } from "@/components/common/StatusBadge";
import { ProviderIcon } from "@/components/common/ProviderIcon";
import { TableSkeleton, ErrorState } from "@/components/common/Skeletons";
import { format } from "date-fns";
import type { UsageLog } from "@/lib/types";

function LogsPage() {
  const { data, isLoading, isError, error, refetch } = useQuery<{
    items: UsageLog[];
    total: number;
  }>({
    queryKey: ["logs"],
    queryFn: async () => {
      const raw = await apiFetch("/api/logs?limit=100");
      return normalizeListResponse<UsageLog>(raw);
    },
  });
  return (
    <div>
      <PageHeader
        title="Logs"
        description="Request-level history with filters and JSON view."
        icon="receipt_long"
      />
      {isLoading ? (
        <TableSkeleton rows={8} columns={8} />
      ) : isError ? (
        <ErrorState
          title="Couldn't load logs"
          error={error}
          onRetry={() => refetch()}
        />
      ) : (
        <DataTable
          columns={[
            { header: "Time", cell: ({ row }) => <span className="text-xs text-text-muted">{format(new Date(row.original.timestamp), "MMM d HH:mm:ss")}</span> },
            { header: "Provider", cell: ({ row }) => <div className="flex items-center gap-1.5"><ProviderIcon provider={row.original.provider} size={18} /><span className="text-xs">{row.original.provider}</span></div> },
            { header: "Model", cell: ({ row }) => <code className="text-xs">{row.original.model}</code> },
            { header: "Key", accessorKey: "api_key_name" },
            { header: "Status", cell: ({ row }) => <StatusBadge variant={row.original.status === "success" ? "success" : "danger"} dot>{row.original.status_code}</StatusBadge> },
            { header: "Tokens", cell: ({ row }) => <span className="tabular-nums text-xs">{row.original.total_tokens.toLocaleString()}</span> },
            { header: "Cost", cell: ({ row }) => <span className="tabular-nums text-xs">${row.original.cost_usd.toFixed(4)}</span> },
            { header: "Latency", cell: ({ row }) => <span className="tabular-nums text-xs">{row.original.latency_ms}ms</span> },
          ]}
          data={data?.items ?? []}
          initialVisibleRows={25}
          emptyTitle="No log entries"
          emptyDescription="Requests will appear here once traffic flows through the gateway."
          emptyIcon="receipt_long"
        />
      )}
    </div>
  );
}

export const Route = createFileRoute("/_app/logs")({
  component: LogsPage,
});
