import * as React from "react";
import { createFileRoute } from "@tanstack/react-router";
import { apiFetch } from "@/lib/api";
import { Toggle } from "@/components/ui/toggle";
import { CardSkeleton } from "@/components/ui/skeleton";
import { useNotificationStore } from "@/stores/notification";
import type { FeatureFlag } from "@/lib/types";

export const Route = createFileRoute("/feature-flags")({
  component: FeatureFlagsPage,
});

// FeatureFlagsPage (PAR-UI-130 subset) lists feature flags from
// GET /api/feature-flags and toggles each via PUT /api/feature-flags/{id}
// {enabled}. The mock exposes only GET + PUT-by-id, so there is no create/delete
// UI (plan §1.4). Variant-HAVE against the mock; no Go /api/feature-flags exists
// yet (§8 ESCALATION-1c).
function FeatureFlagsPage() {
  const pushToast = useNotificationStore((state) => state.push);
  const [flags, setFlags] = React.useState<FeatureFlag[]>([]);
  const [loading, setLoading] = React.useState(true);

  const load = React.useCallback(() => {
    setLoading(true);
    apiFetch<FeatureFlag[]>("/api/feature-flags")
      .then((rows) => {
        setFlags(rows ?? []);
        setLoading(false);
      })
      .catch(() => {
        setFlags([]);
        setLoading(false);
        pushToast({ message: "Failed to load feature flags" });
      });
  }, [pushToast]);

  React.useEffect(() => {
    load();
  }, [load]);

  async function setEnabled(flag: FeatureFlag, enabled: boolean) {
    setFlags((prev) =>
      prev.map((f) => (f.id === flag.id ? { ...f, enabled } : f))
    );
    try {
      await apiFetch(`/api/feature-flags/${flag.id}`, {
        method: "PUT",
        body: JSON.stringify({ enabled }),
      });
    } catch {
      pushToast({ message: "Failed to update the flag" });
      load();
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <header>
        <h1 className="text-2xl font-semibold text-foreground">Feature Flags</h1>
      </header>

      {loading ? (
        <CardSkeleton />
      ) : flags.length === 0 ? (
        <p className="text-sm text-muted-foreground">No feature flags yet.</p>
      ) : (
        <div className="flex flex-col gap-2">
          {flags.map((flag) => (
            <div
              key={flag.id}
              data-testid="feature-flag-row"
              className="flex items-center justify-between rounded-lg border border-border px-4 py-3"
            >
              <div>
                <p className="text-sm font-medium text-foreground">{flag.key}</p>
                <p className="text-xs text-muted-foreground">{flag.description}</p>
              </div>
              <Toggle
                checked={flag.enabled}
                onCheckedChange={(checked) => setEnabled(flag, checked)}
                aria-label={`Toggle ${flag.key}`}
              />
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
