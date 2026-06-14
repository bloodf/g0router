import * as React from "react";
import { apiFetch } from "@/lib/api";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ProviderIcon } from "@/components/ui/provider-icon";
import { CardSkeleton } from "@/components/ui/skeleton";
import type { Connection, Model, Provider } from "@/lib/types";

export interface ProviderDetailPanelProps {
  provider: Provider;
  onClose: () => void;
  onAddConnection: (provider: Provider) => void;
}

// ProviderDetailPanel is the in-page provider detail (PAR-UI-009): it loads the
// provider's connections + models from the provider-shaped read API and lists
// them. New/detail are in-page state, not nested routes (plan §1.5).
function ProviderDetailPanel({
  provider,
  onClose,
  onAddConnection,
}: ProviderDetailPanelProps) {
  const [connections, setConnections] = React.useState<Connection[]>([]);
  const [models, setModels] = React.useState<Model[]>([]);
  const [loading, setLoading] = React.useState(true);

  React.useEffect(() => {
    let active = true;
    setLoading(true);
    Promise.all([
      apiFetch<Connection[]>(`/api/providers/${provider.id}/connections`).catch(
        () => [] as Connection[]
      ),
      apiFetch<Model[]>(`/api/providers/${provider.id}/models`).catch(
        () => [] as Model[]
      ),
    ]).then(([conns, mods]) => {
      if (!active) return;
      setConnections(conns ?? []);
      setModels(mods ?? []);
      setLoading(false);
    });
    return () => {
      active = false;
    };
  }, [provider.id]);

  return (
    <Card data-testid="provider-detail-panel" className="flex flex-col gap-4">
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <ProviderIcon slug={provider.id} name={provider.display_name} size="lg" />
          <div>
            <h2 className="text-lg font-semibold text-foreground">
              {provider.display_name}
            </h2>
            <p className="text-xs text-muted-foreground">{provider.description}</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Button
            data-testid="add-connection-action"
            variant="primary"
            size="sm"
            onClick={() => onAddConnection(provider)}
          >
            Add connection
          </Button>
          <Button variant="ghost" size="sm" onClick={onClose}>
            Close
          </Button>
        </div>
      </div>

      {loading ? (
        <CardSkeleton />
      ) : (
        <>
          <section className="flex flex-col gap-2">
            <h3 className="text-sm font-semibold text-foreground">Connections</h3>
            {connections.length === 0 ? (
              <p className="text-xs text-muted-foreground">No connections yet.</p>
            ) : (
              connections.map((conn) => (
                <div
                  key={conn.id}
                  data-testid="connection-row"
                  className="flex items-center justify-between rounded-md border border-border px-3 py-2"
                >
                  <span className="text-sm text-foreground">{conn.name}</span>
                  <div className="flex items-center gap-2">
                    <Badge variant="default" size="sm">
                      {conn.auth_type}
                    </Badge>
                    {conn.needs_reauth ? (
                      <Badge variant="error" size="sm">
                        needs reauth
                      </Badge>
                    ) : (
                      <Badge variant="success" size="sm" dot>
                        active
                      </Badge>
                    )}
                  </div>
                </div>
              ))
            )}
          </section>

          <section className="flex flex-col gap-2">
            <h3 className="text-sm font-semibold text-foreground">Models</h3>
            {models.length === 0 ? (
              <p className="text-xs text-muted-foreground">No models listed.</p>
            ) : (
              models.map((model) => (
                <div
                  key={model.id}
                  data-testid="model-row-detail"
                  className="flex items-center justify-between rounded-md border border-border px-3 py-1.5"
                >
                  <span className="text-sm text-foreground">{model.name}</span>
                  <span className="text-xs text-muted-foreground">
                    ${model.input_cost} / ${model.output_cost}
                  </span>
                </div>
              ))
            )}
          </section>
        </>
      )}
    </Card>
  );
}

export { ProviderDetailPanel };
