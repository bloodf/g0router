import { useState } from "react";
import { Link } from "@tanstack/react-router";
import { Card } from "@/components/ui/card";
import { ProviderIcon } from "@/components/common/ProviderIcon";
import { StatusBadge } from "@/components/common/StatusBadge";
import { Icon } from "@/components/common/Icon";
import { Button } from "@/components/ui/button";
import { QuotaProgressBar } from "./QuotaProgressBar";
import { QuotaTable } from "./QuotaTable";
import { Skeleton } from "@/components/ui/skeleton";
import type { QuotaRow } from "./quota-utils";

interface Props {
  provider: string;
  name: string;
  plan?: "free" | "pro" | "ultra" | "enterprise";
  quotas: QuotaRow[];
  message?: string;
  error?: string;
  loading?: boolean;
  onRefresh?: () => Promise<void> | void;
}

const planVariant = {
  free: "muted",
  pro: "primary",
  ultra: "success",
  enterprise: "info",
} as const;

export function ProviderLimitCard({
  provider,
  name,
  plan,
  quotas,
  message,
  error,
  loading,
  onRefresh,
}: Props) {
  const [refreshing, setRefreshing] = useState(false);
  const handleRefresh = async () => {
    if (!onRefresh || refreshing) return;
    setRefreshing(true);
    try {
      await onRefresh();
    } finally {
      setRefreshing(false);
    }
  };

  return (
    <Card className="card-elev border-border p-4 flex flex-col gap-3">
      {/* Header */}
      <div className="flex items-start gap-2.5">
        <Link
          to="/providers/$id"
          params={{ id: provider }}
          className="flex items-start gap-2.5 flex-1 min-w-0 group"
          title="Open provider page"
        >
          <ProviderIcon provider={provider} size={36} />
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <h3 className="font-semibold text-sm truncate group-hover:text-brand-600 transition-colors">
                {name || provider}
              </h3>
              {plan && (
                <StatusBadge variant={planVariant[plan]} className="uppercase">
                  {plan}
                </StatusBadge>
              )}
            </div>
            <div className="text-[11px] text-text-muted capitalize">{provider}</div>
          </div>
        </Link>
        <Button
          variant="ghost"
          size="icon"
          className="h-7 w-7"
          onClick={handleRefresh}
          disabled={refreshing}
          aria-label="Refresh quota"
        >
          <Icon
            name="refresh"
            size={16}
            className={refreshing ? "animate-spin" : ""}
          />
        </Button>
      </div>

      {/* Body */}
      {loading && (
        <div className="space-y-3">
          <Skeleton className="h-12" />
          <Skeleton className="h-12" />
        </div>
      )}

      {!loading && error && (
        <div className="flex items-start gap-2 rounded-md bg-destructive/10 text-destructive p-2.5 text-xs">
          <Icon name="error" size={16} />
          <span className="flex-1">{error}</span>
        </div>
      )}

      {!loading && !error && message && (
        <div className="flex items-start gap-2 rounded-md bg-info/10 text-info p-2.5 text-xs">
          <Icon name="info" size={16} />
          <span className="flex-1">{message}</span>
        </div>
      )}

      {!loading && !error && !message && quotas.length > 0 && (
        quotas.length > 3 ? (
          <QuotaTable quotas={quotas} sortMode="remaining-asc" showSortLabel />
        ) : (
          <div className="space-y-4">
            {quotas.map((q) => (
              <QuotaProgressBar key={q.name} row={q} />
            ))}
          </div>
        )
      )}

      {!loading && !error && !message && quotas.length === 0 && (
        <div className="flex flex-col items-center justify-center py-8 text-text-muted text-xs gap-1.5">
          <Icon name="data_usage" size={28} />
          <span>No quota data available</span>
        </div>
      )}
    </Card>
  );
}
