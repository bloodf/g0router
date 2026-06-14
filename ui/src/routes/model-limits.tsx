import * as React from "react";
import { createFileRoute } from "@tanstack/react-router";
import { apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { ConfirmModal } from "@/components/ui/confirm-modal";
import { CardSkeleton } from "@/components/ui/skeleton";
import { ModelLimitModal } from "@/components/routing/model-limit-modal";
import { useNotificationStore } from "@/stores/notification";
import type { ModelLimit } from "@/lib/types";

export const Route = createFileRoute("/model-limits")({
  component: ModelLimitsPage,
});

// ModelLimitsPage (PAR-UI-130 subset) lists model limits from
// GET /api/model-limits and drives create/edit (ModelLimitModal) and delete
// (ConfirmModal). Variant-HAVE against the mock; no Go backend yet (§8
// ESCALATION-3b).
function ModelLimitsPage() {
  const pushToast = useNotificationStore((state) => state.push);
  const [limits, setLimits] = React.useState<ModelLimit[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [editing, setEditing] = React.useState<ModelLimit | null>(null);
  const [creating, setCreating] = React.useState(false);
  const [deleting, setDeleting] = React.useState<ModelLimit | null>(null);
  const [deleteBusy, setDeleteBusy] = React.useState(false);

  const load = React.useCallback(() => {
    setLoading(true);
    apiFetch<ModelLimit[]>("/api/model-limits")
      .then((rows) => {
        setLimits(rows ?? []);
        setLoading(false);
      })
      .catch(() => {
        setLimits([]);
        setLoading(false);
        pushToast({ message: "Failed to load model limits" });
      });
  }, [pushToast]);

  React.useEffect(() => {
    load();
  }, [load]);

  async function confirmDelete() {
    if (!deleting) return;
    setDeleteBusy(true);
    try {
      await apiFetch(`/api/model-limits/${deleting.id}`, { method: "DELETE" });
      setLimits((prev) => prev.filter((l) => l.id !== deleting.id));
      pushToast({ message: "Limit deleted" });
      setDeleting(null);
    } catch {
      pushToast({ message: "Failed to delete the limit" });
    } finally {
      setDeleteBusy(false);
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <header className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-foreground">Model Limits</h1>
        <Button
          data-testid="model-limit-new"
          variant="primary"
          size="sm"
          onClick={() => setCreating(true)}
        >
          New limit
        </Button>
      </header>

      {loading ? (
        <CardSkeleton />
      ) : limits.length === 0 ? (
        <p className="text-sm text-muted-foreground">No model limits yet.</p>
      ) : (
        <div className="flex flex-col gap-2">
          {limits.map((limit) => (
            <div
              key={limit.id}
              data-testid="model-limit-row"
              className="flex items-center justify-between rounded-lg border border-border px-4 py-3"
            >
              <div>
                <p className="text-sm font-medium text-foreground">{limit.model}</p>
                <p className="text-xs text-muted-foreground">
                  {limit.max_tokens} tokens · {limit.max_rpm} RPM
                </p>
              </div>
              <div className="flex items-center gap-2">
                {limit.allowed_key_ids.length > 0 ? (
                  <Badge variant="neutral" size="sm">
                    {limit.allowed_key_ids.length} key
                    {limit.allowed_key_ids.length === 1 ? "" : "s"}
                  </Badge>
                ) : null}
                <Button variant="ghost" size="sm" onClick={() => setEditing(limit)}>
                  Edit
                </Button>
                <Button
                  data-testid="model-limit-delete"
                  variant="danger"
                  size="sm"
                  onClick={() => setDeleting(limit)}
                >
                  Delete
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}

      <ModelLimitModal
        open={creating || editing !== null}
        limit={editing}
        onClose={() => {
          setCreating(false);
          setEditing(null);
        }}
        onSaved={load}
      />
      <ConfirmModal
        open={deleting !== null}
        title="Delete limit"
        message={`Delete the limit for "${deleting?.model ?? ""}"? This cannot be undone.`}
        confirmLabel="Delete"
        cancelLabel="Cancel"
        variant="danger"
        loading={deleteBusy}
        onConfirm={confirmDelete}
        onCancel={() => setDeleting(null)}
      />
    </div>
  );
}
