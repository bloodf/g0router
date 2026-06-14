import * as React from "react";
import { createFileRoute } from "@tanstack/react-router";
import { apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { ProviderIcon } from "@/components/ui/provider-icon";
import { ConfirmModal } from "@/components/ui/confirm-modal";
import { CardSkeleton } from "@/components/ui/skeleton";
import { AliasModal } from "@/components/routing/alias-modal";
import { useNotificationStore } from "@/stores/notification";
import type { Alias } from "@/lib/types";

export const Route = createFileRoute("/aliases")({
  component: AliasesPage,
});

// AliasesPage (PAR-UI-116) lists model aliases from GET /api/aliases and drives
// create/edit (AliasModal) and delete (ConfirmModal). Variant-HAVE against the
// mock; the alias store exists but there is no admin endpoint yet (§8
// ESCALATION-2).
function AliasesPage() {
  const pushToast = useNotificationStore((state) => state.push);
  const [aliases, setAliases] = React.useState<Alias[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [editing, setEditing] = React.useState<Alias | null>(null);
  const [creating, setCreating] = React.useState(false);
  const [deleting, setDeleting] = React.useState<Alias | null>(null);
  const [deleteBusy, setDeleteBusy] = React.useState(false);

  const load = React.useCallback(() => {
    setLoading(true);
    apiFetch<Alias[]>("/api/aliases")
      .then((rows) => {
        setAliases(rows ?? []);
        setLoading(false);
      })
      .catch(() => {
        setAliases([]);
        setLoading(false);
        pushToast({ message: "Failed to load aliases" });
      });
  }, [pushToast]);

  React.useEffect(() => {
    load();
  }, [load]);

  async function confirmDelete() {
    if (!deleting) return;
    setDeleteBusy(true);
    try {
      await apiFetch(`/api/aliases/${deleting.id}`, { method: "DELETE" });
      setAliases((prev) => prev.filter((a) => a.id !== deleting.id));
      pushToast({ message: "Alias deleted" });
      setDeleting(null);
    } catch {
      pushToast({ message: "Failed to delete the alias" });
    } finally {
      setDeleteBusy(false);
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <header className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-foreground">Aliases</h1>
        <Button
          data-testid="alias-new"
          variant="primary"
          size="sm"
          onClick={() => setCreating(true)}
        >
          New alias
        </Button>
      </header>

      {loading ? (
        <CardSkeleton />
      ) : aliases.length === 0 ? (
        <p className="text-sm text-muted-foreground">No aliases yet.</p>
      ) : (
        <div className="flex flex-col gap-2">
          {aliases.map((alias) => (
            <div
              key={alias.id}
              data-testid="alias-row"
              className="flex items-center justify-between rounded-lg border border-border px-4 py-3"
            >
              <div className="flex items-center gap-3">
                <ProviderIcon slug={alias.provider} name={alias.provider} size="sm" />
                <div>
                  <p className="text-sm font-medium text-foreground">{alias.alias}</p>
                  <p className="text-xs text-muted-foreground">
                    {alias.provider} · {alias.model}
                  </p>
                </div>
              </div>
              <div className="flex items-center gap-2">
                <Button variant="ghost" size="sm" onClick={() => setEditing(alias)}>
                  Edit
                </Button>
                <Button
                  data-testid="alias-delete"
                  variant="danger"
                  size="sm"
                  onClick={() => setDeleting(alias)}
                >
                  Delete
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}

      <AliasModal
        open={creating || editing !== null}
        alias={editing}
        onClose={() => {
          setCreating(false);
          setEditing(null);
        }}
        onSaved={load}
      />
      <ConfirmModal
        open={deleting !== null}
        title="Delete alias"
        message={`Delete "${deleting?.alias ?? ""}"? This cannot be undone.`}
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
