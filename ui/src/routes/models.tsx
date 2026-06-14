import * as React from "react";
import { createFileRoute } from "@tanstack/react-router";
import { apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Toggle } from "@/components/ui/toggle";
import { ProviderIcon } from "@/components/ui/provider-icon";
import { Pagination } from "@/components/ui/pagination";
import { CardSkeleton } from "@/components/ui/skeleton";
import { AddCustomEmbeddingModal } from "@/components/providers/add-custom-embedding-modal";
import { useNotificationStore } from "@/stores/notification";
import type { Model } from "@/lib/types";

export const Route = createFileRoute("/models")({
  component: ModelsPage,
});

const PAGE_SIZE = 20;

function ModelsPage() {
  const pushToast = useNotificationStore((state) => state.push);
  const [models, setModels] = React.useState<Model[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [page, setPage] = React.useState(1);
  const [addOpen, setAddOpen] = React.useState(false);

  const load = React.useCallback(() => {
    setLoading(true);
    apiFetch<Model[]>("/api/models")
      .then((rows) => {
        setModels(rows ?? []);
        setLoading(false);
      })
      .catch(() => {
        setModels([]);
        setLoading(false);
        pushToast({ message: "Failed to load models" });
      });
  }, [pushToast]);

  React.useEffect(() => {
    load();
  }, [load]);

  async function setDisabled(model: Model, disabled: boolean) {
    setModels((prev) =>
      prev.map((m) =>
        m.id === model.id ? { ...m, is_disabled: disabled } : m
      )
    );
    try {
      await apiFetch("/api/models/disabled", {
        method: disabled ? "POST" : "DELETE",
        body: JSON.stringify({ model_id: model.id }),
      });
    } catch {
      pushToast({ message: "Failed to update the model" });
      load();
    }
  }

  const totalPages = Math.max(1, Math.ceil(models.length / PAGE_SIZE));
  const pageModels = models.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE);

  return (
    <div className="flex flex-col gap-6">
      <header className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-foreground">Models</h1>
        <Button variant="primary" size="sm" onClick={() => setAddOpen(true)}>
          Add custom model
        </Button>
      </header>

      {loading ? (
        <CardSkeleton />
      ) : models.length === 0 ? (
        <p className="text-sm text-muted-foreground">No models available.</p>
      ) : (
        <>
          <div className="flex flex-col gap-2">
            {pageModels.map((model) => (
              <div
                key={`${model.provider}/${model.id}`}
                data-testid="model-row"
                className="flex items-center justify-between rounded-lg border border-border px-4 py-3"
              >
                <div className="flex items-center gap-3">
                  <ProviderIcon
                    slug={model.provider}
                    name={model.provider}
                    size="sm"
                  />
                  <div>
                    <p className="text-sm font-medium text-foreground">
                      {model.name}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {model.provider}
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-4">
                  <span className="text-xs text-muted-foreground">
                    ${model.input_cost} / ${model.output_cost}
                  </span>
                  <span className="text-xs text-muted-foreground">
                    {model.context_window.toLocaleString()} ctx
                  </span>
                  <Toggle
                    checked={model.is_disabled}
                    onCheckedChange={(checked) => setDisabled(model, checked)}
                    aria-label={`Disable ${model.name}`}
                  />
                </div>
              </div>
            ))}
          </div>
          <Pagination
            page={page}
            totalPages={totalPages}
            onPageChange={setPage}
          />
        </>
      )}

      <AddCustomEmbeddingModal
        open={addOpen}
        onClose={() => setAddOpen(false)}
        onCreated={load}
      />
    </div>
  );
}
