import { createFileRoute } from "@tanstack/react-router";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api/client";
import { Switch } from "@/components/ui/switch";
import { PageHeader } from "@/components/common/PageHeader";
import { DataTable } from "@/components/common/DataTable";
import { TableSkeleton, ErrorState } from "@/components/common/Skeletons";
import { toast } from "sonner";
import type { FeatureFlag } from "@/lib/types";

export const Route = createFileRoute("/_app/feature-flags")({
  component: FeatureFlagsPage,
});

function FeatureFlagsPage() {
  const qc = useQueryClient();
  const { data, isLoading, isError, error, refetch } = useQuery<FeatureFlag[]>({
    queryKey: ["feature-flags"],
    queryFn: () => apiFetch("/api/feature-flags"),
  });

  const toggle = useMutation({
    mutationFn: ({ id, enabled }: { id: number; enabled: boolean }) =>
      apiFetch<FeatureFlag>(`/api/feature-flags/${id}`, { method: "PUT", body: { enabled } }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["feature-flags"] });
      toast.success("Feature flag updated");
    },
    onError: (e: any) => toast.error(e?.message || "Failed to update"),
  });

  return (
    <div>
      <PageHeader
        title="Feature Flags"
        description="Toggle runtime features controlled by the gateway."
        icon="flag"
      />
      {isLoading ? (
        <TableSkeleton rows={6} columns={3} />
      ) : isError ? (
        <ErrorState title="Couldn’t load feature flags" error={error} onRetry={() => refetch()} />
      ) : (
        <DataTable
          columns={[
            { header: "Key", accessorKey: "key" },
            {
              header: "Description",
              accessorKey: "description",
              cell: ({ row }) => row.original.description || "—",
            },
            {
              header: "Enabled",
              cell: ({ row }) => (
                <Switch
                  checked={row.original.enabled}
                  onCheckedChange={(checked) =>
                    toggle.mutate({ id: row.original.id, enabled: checked })
                  }
                  disabled={toggle.isPending}
                />
              ),
            },
          ]}
          data={data ?? []}
          emptyTitle="No feature flags"
          emptyIcon="flag"
        />
      )}
    </div>
  );
}
