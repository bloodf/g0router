import * as React from "react";
import { createFileRoute } from "@tanstack/react-router";
import { apiFetch } from "@/lib/api";
import { ProviderCard } from "@/components/providers/provider-card";
import { ProviderDetailPanel } from "@/components/providers/provider-detail-panel";
import { CardSkeleton } from "@/components/ui/skeleton";
import { useNotificationStore } from "@/stores/notification";
import type { Provider } from "@/lib/types";

export const Route = createFileRoute("/providers")({
  component: ProvidersPage,
});

type GroupKey = "oauth" | "api_key" | "free" | "compatible";

const GROUP_LABELS: Record<GroupKey, string> = {
  oauth: "OAuth",
  api_key: "API Key",
  free: "Free / No-auth",
  compatible: "Compatible",
};

const GROUP_ORDER: GroupKey[] = ["oauth", "api_key", "free", "compatible"];

// groupProvider buckets a provider per its auth_types (plan §1.5 variant): OAuth
// when it supports oauth, Free/No-auth for noauth providers, API-Key for api_key,
// and Compatible for anything else (custom/OpenAI-compatible).
function groupProvider(provider: Provider): GroupKey {
  const auth = provider.auth_types;
  if (auth.includes("oauth")) return "oauth";
  if (auth.includes("noauth")) return "free";
  if (auth.includes("api_key")) return "api_key";
  return "compatible";
}

function ProvidersPage() {
  const pushToast = useNotificationStore((state) => state.push);
  const [providers, setProviders] = React.useState<Provider[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [selected, setSelected] = React.useState<Provider | null>(null);

  React.useEffect(() => {
    let active = true;
    setLoading(true);
    apiFetch<Provider[]>("/api/providers/catalog")
      .then((list) => {
        if (!active) return;
        setProviders(list ?? []);
        setLoading(false);
      })
      .catch(() => {
        if (!active) return;
        setProviders([]);
        setLoading(false);
        pushToast({ message: "Failed to load providers" });
      });
    return () => {
      active = false;
    };
  }, [pushToast]);

  const groups = React.useMemo(() => {
    const buckets: Record<GroupKey, Provider[]> = {
      oauth: [],
      api_key: [],
      free: [],
      compatible: [],
    };
    for (const provider of providers) {
      buckets[groupProvider(provider)].push(provider);
    }
    return buckets;
  }, [providers]);

  function handleAddConnection(provider: Provider) {
    // The auth/config modal flow lands in T5; surface the intent for now.
    pushToast({ message: `Add a connection for ${provider.display_name}` });
  }

  function handleOAuth(provider: Provider) {
    pushToast({ message: `Connect ${provider.display_name} via OAuth` });
  }

  return (
    <div className="flex flex-col gap-6">
      <header className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-foreground">Providers</h1>
      </header>

      {selected ? (
        <ProviderDetailPanel
          provider={selected}
          onClose={() => setSelected(null)}
          onAddConnection={handleAddConnection}
        />
      ) : null}

      {loading ? (
        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <CardSkeleton />
          <CardSkeleton />
          <CardSkeleton />
        </div>
      ) : (
        GROUP_ORDER.map((key) => {
          const list = groups[key];
          if (list.length === 0) return null;
          return (
            <section
              key={key}
              data-testid="provider-group"
              data-group={key}
              className="flex flex-col gap-3"
            >
              <h2 className="text-sm font-semibold uppercase tracking-wide text-muted-foreground">
                {GROUP_LABELS[key]}
              </h2>
              <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
                {list.map((provider) => (
                  <ProviderCard
                    key={provider.id}
                    provider={provider}
                    onSelect={setSelected}
                    onOAuth={handleOAuth}
                  />
                ))}
              </div>
            </section>
          );
        })
      )}
    </div>
  );
}
