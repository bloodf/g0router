import * as React from "react";
import { createFileRoute } from "@tanstack/react-router";
import { apiFetch } from "@/lib/api";
import { CardSkeleton } from "@/components/ui/skeleton";
import { ProviderLimits } from "@/components/usage/provider-limits";
import { useNotificationStore } from "@/stores/notification";
import type { Quota } from "@/lib/types";

export const Route = createFileRoute("/quota")({
  component: QuotaPage,
});

function QuotaPage() {
  const pushToast = useNotificationStore((s) => s.push);
  const [quotas, setQuotas] = React.useState<Quota[]>([]);
  const [loading, setLoading] = React.useState(true);

  // Variant per plan §1.4: provider-limits view rendered from the /api/quota mock
  // contract (the real per-connection aggregation Go is a serial follow-up).
  React.useEffect(() => {
    let cancelled = false;
    apiFetch<Quota[]>("/api/quota")
      .then((data) => {
        if (cancelled) return;
        setQuotas(data ?? []);
        setLoading(false);
      })
      .catch(() => {
        if (cancelled) return;
        setQuotas([]);
        setLoading(false);
        pushToast({ message: "Failed to load quotas" });
      });
    return () => {
      cancelled = true;
    };
  }, [pushToast]);

  return (
    <div className="flex flex-col gap-6">
      <h1 className="text-2xl font-semibold text-foreground">Quota</h1>
      {loading ? (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <CardSkeleton />
          <CardSkeleton />
          <CardSkeleton />
        </div>
      ) : (
        <ProviderLimits quotas={quotas} />
      )}
    </div>
  );
}
