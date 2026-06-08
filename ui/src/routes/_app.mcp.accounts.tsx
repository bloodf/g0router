import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api/client";
import { PageHeader } from "@/components/common/PageHeader";
import { DataTable } from "@/components/common/DataTable";
import { TableSkeleton, ErrorState } from "@/components/common/Skeletons";
import type { McpInstance, McpAccount } from "@/lib/types";

export const Route = createFileRoute("/_app/mcp/accounts")({
  component: McpAccountsPage,
});

function McpAccountsPage() {
  const [instanceId, setInstanceId] = useState<string>("");

  const { data: instances, isLoading: instancesLoading } = useQuery<McpInstance[]>({
    queryKey: ["mcp-instances"],
    queryFn: () => apiFetch("/api/mcp/instances"),
  });

  const {
    data: accounts,
    isLoading: accountsLoading,
    isError,
    error,
    refetch,
  } = useQuery<McpAccount[]>({
    queryKey: ["mcp-accounts", instanceId],
    queryFn: () => apiFetch(`/api/mcp/instances/${instanceId}/accounts`),
    enabled: !!instanceId,
  });

  return (
    <div>
      <PageHeader
        title="MCP Accounts"
        description="Linked OAuth accounts per MCP instance."
        icon="account_circle"
      />
      <div className="mb-4">
        <label className="text-xs font-medium block mb-1 text-text-muted">Instance</label>
        <select
          className="w-full max-w-sm bg-surface-2 border border-border rounded-lg px-3 py-2 text-sm focus:border-brand-500 outline-none"
          value={instanceId}
          onChange={(e) => setInstanceId(e.target.value)}
        >
          <option value="">— select instance —</option>
          {instances?.map((i) => (
            <option key={i.ID} value={i.ID}>
              {i.Name} ({i.Transport})
            </option>
          ))}
        </select>
      </div>
      {!instanceId ? (
        <div className="text-center py-16 text-text-muted text-sm border border-dashed border-border rounded-xl">
          Select an instance to view linked accounts.
        </div>
      ) : accountsLoading ? (
        <TableSkeleton rows={5} columns={5} />
      ) : isError ? (
        <ErrorState title="Couldn’t load accounts" error={error} onRetry={refetch} />
      ) : (
        <DataTable
          columns={[
            { header: "Label", accessorKey: "account_label" },
            { header: "Subject", accessorKey: "subject" },
            { header: "Email", accessorKey: "email" },
            { header: "Issuer", accessorKey: "issuer" },
            {
              header: "Scopes",
              cell: ({ row }) => row.original.scopes?.join(", ") || "—",
            },
            {
              header: "Expires",
              cell: ({ row }) => row.original.expires_at || "—",
            },
          ]}
          data={accounts ?? []}
          emptyTitle="No accounts"
          emptyDescription="This instance has no linked accounts yet."
          emptyIcon="account_circle"
        />
      )}
    </div>
  );
}
