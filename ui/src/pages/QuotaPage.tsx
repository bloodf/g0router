import { useEffect, useState } from "react";
import { getQuota, isAuthExpiredError, listProviders } from "../api";
import type { ProviderMatrixEntry, QuotaResponse } from "../api";
import { EmptyState, ErrorState, LoadingState, Panel, ProgressBar, StatusPill } from "../components/Primitives";

type ProviderQuota = {
  provider: ProviderMatrixEntry;
  quota: QuotaResponse;
};

type QuotaState =
  | { status: "loading" }
  | { status: "success"; data: ProviderQuota[] }
  | { status: "empty" }
  | { status: "error"; message: string }
  | { status: "auth-expired"; message: string };

export function QuotaPage() {
  const [state, setState] = useState<QuotaState>({ status: "loading" });

  useEffect(() => {
    let cancelled = false;

    async function loadQuotas() {
      try {
        const providers = await listProviders();
        const quotaProviders = providers.filter((provider) => provider.quota);

        if (quotaProviders.length === 0) {
          if (!cancelled) {
            setState({ status: "empty" });
          }
          return;
        }

        const data = await Promise.all(
          quotaProviders.map(async (provider) => ({
            provider,
            quota: await getQuota(provider.id)
          }))
        );

        if (!cancelled) {
          setState({ status: "success", data });
        }
      } catch (error) {
        if (cancelled) {
          return;
        }
        setState({
          status: isAuthExpiredError(error) ? "auth-expired" : "error",
          message: error instanceof Error ? error.message : "quota request failed"
        });
      }
    }

    void loadQuotas();

    return () => {
      cancelled = true;
    };
  }, []);

  return (
    <Panel title="Quotas monitor" description="Provider limit usage returned by quota-capable API contracts.">
      {renderQuotaContent(state)}
    </Panel>
  );
}

function renderQuotaContent(state: QuotaState) {
  switch (state.status) {
    case "loading":
      return <LoadingState label="Loading quota data" />;
    case "empty":
      return <EmptyState title="No quota-capable providers" description="The provider matrix did not report any quota-capable entries." />;
    case "error":
      return <ErrorState title="Quota data unavailable" message={state.message} />;
    case "auth-expired":
      return <ErrorState title="Session expired" message={state.message} />;
    case "success":
      return <QuotaList quotas={state.data} />;
  }
}

function QuotaList({ quotas }: { quotas: ProviderQuota[] }) {
  return (
    <div className="space-y-5">
      {quotas.map(({ provider, quota }) => {
        const providerID = quota.Provider || provider.id;
        const percent = quota.Limit > 0 ? Math.round((quota.Used / quota.Limit) * 100) : 0;
        const clampedPercent = Math.max(0, Math.min(percent, 100));

        return (
          <article key={provider.id} aria-label={`${providerID} quota`} className="rounded-md border border-zinc-200 p-4">
            <div className="mb-4 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
              <div>
                <h4 className="font-semibold text-zinc-950">{providerID}</h4>
                <p className="mt-1 text-sm text-zinc-500">
                  {quota.Used.toLocaleString()} of {quota.Limit.toLocaleString()} used
                </p>
              </div>
              <StatusPill tone={quota.Remaining <= 0 ? "bad" : clampedPercent >= 85 ? "warn" : "good"}>
                {quota.Remaining.toLocaleString()} remaining
              </StatusPill>
            </div>
            <ProgressBar label="Quota used" value={clampedPercent} />
          </article>
        );
      })}
    </div>
  );
}
