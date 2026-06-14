import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { ProviderIcon } from "@/components/ui/provider-icon";
import type { Quota } from "@/lib/types";

// ProviderLimits renders quota cards (PAR-UI-012): used/limit progress bar,
// plan badge, unit, reset_at; "unlimited" when limit === 0. Variant per plan
// §1.4 — fed by the /api/quota mock contract.
export interface ProviderLimitsProps {
  quotas: Quota[];
}

function formatNumber(n: number): string {
  return n.toLocaleString("en-US");
}

function QuotaCard({ quota }: { quota: Quota }) {
  const unlimited = quota.limit === 0;
  const pct = unlimited ? 0 : Math.min(100, Math.round((quota.used / quota.limit) * 100));
  return (
    <Card data-testid="quota-card" padding="md" className="flex flex-col gap-3">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <ProviderIcon slug={quota.provider.toLowerCase()} name={quota.provider} size="sm" />
          <span className="font-medium text-foreground">{quota.connection_name}</span>
        </div>
        <Badge variant="primary">{quota.plan}</Badge>
      </div>

      <div className="flex flex-col gap-1">
        <div className="flex justify-between text-sm text-muted-foreground">
          <span>
            {formatNumber(quota.used)}
            {unlimited ? "" : ` / ${formatNumber(quota.limit)}`} {quota.unit}
          </span>
          <span>{unlimited ? "Unlimited" : `${pct}%`}</span>
        </div>
        <div
          data-testid="quota-progress"
          role="progressbar"
          aria-valuenow={unlimited ? undefined : pct}
          aria-valuemin={0}
          aria-valuemax={100}
          className="h-2 w-full overflow-hidden rounded-full bg-muted"
        >
          <div
            className={unlimited ? "h-full w-full bg-emerald-500/40" : "h-full bg-primary"}
            style={{ width: unlimited ? "100%" : `${pct}%` }}
          />
        </div>
      </div>

      {quota.account_label ? (
        <span className="text-xs text-muted-foreground">{quota.account_label}</span>
      ) : null}
      <span className="text-xs text-muted-foreground">
        Resets {new Date(quota.reset_at).toLocaleString("en-US")}
      </span>
    </Card>
  );
}

export function ProviderLimits({ quotas }: ProviderLimitsProps) {
  if (quotas.length === 0) {
    return (
      <Card padding="lg" className="text-center text-muted-foreground">
        No provider limits configured.
      </Card>
    );
  }
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
      {quotas.map((q) => (
        <QuotaCard key={q.connection_id} quota={q} />
      ))}
    </div>
  );
}
