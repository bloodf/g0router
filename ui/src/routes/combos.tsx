import * as React from "react";
import { createFileRoute } from "@tanstack/react-router";
import { apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { ConfirmModal } from "@/components/ui/confirm-modal";
import { CardSkeleton } from "@/components/ui/skeleton";
import { ComboList } from "@/components/combos/combo-list";
import { ComboFormModal } from "@/components/combos/combo-form-modal";
import { useNotificationStore } from "@/stores/notification";
import type { Combo } from "@/lib/types";

export const Route = createFileRoute("/combos")({
  component: CombosPage,
});

// CombosPage (PAR-UI-010 / PAR-PR-339) lists combos from GET /api/combos and
// drives create/edit (ComboFormModal, PAR-UI-050 with @dnd-kit member reorder)
// and delete (ConfirmModal → DELETE /api/combos/{id}). Variant-HAVE against the
// mock; the real Go combos DTO/key divergence is a serial follow-up (§1.2 / §8
// ESCALATION-1).
function CombosPage() {
  const pushToast = useNotificationStore((state) => state.push);
  const [combos, setCombos] = React.useState<Combo[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [editing, setEditing] = React.useState<Combo | null>(null);
  const [creating, setCreating] = React.useState(false);
  const [deleting, setDeleting] = React.useState<Combo | null>(null);
  const [deleteBusy, setDeleteBusy] = React.useState(false);

  const load = React.useCallback(() => {
    setLoading(true);
    apiFetch<Combo[]>("/api/combos")
      .then((rows) => {
        setCombos(rows ?? []);
        setLoading(false);
      })
      .catch(() => {
        setCombos([]);
        setLoading(false);
        pushToast({ message: "Failed to load combos" });
      });
  }, [pushToast]);

  React.useEffect(() => {
    load();
  }, [load]);

  async function setActive(combo: Combo, active: boolean) {
    setCombos((prev) =>
      prev.map((c) => (c.id === combo.id ? { ...c, is_active: active } : c))
    );
    try {
      await apiFetch(`/api/combos/${combo.id}`, {
        method: "PUT",
        body: JSON.stringify({ ...combo, is_active: active }),
      });
    } catch {
      pushToast({ message: "Failed to update the combo" });
      load();
    }
  }

  async function confirmDelete() {
    if (!deleting) return;
    setDeleteBusy(true);
    try {
      await apiFetch(`/api/combos/${deleting.id}`, { method: "DELETE" });
      setCombos((prev) => prev.filter((c) => c.id !== deleting.id));
      pushToast({ message: "Combo deleted" });
      setDeleting(null);
    } catch {
      pushToast({ message: "Failed to delete the combo" });
    } finally {
      setDeleteBusy(false);
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <header className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-foreground">Combos</h1>
        <Button
          data-testid="combo-new"
          variant="primary"
          size="sm"
          onClick={() => setCreating(true)}
        >
          New combo
        </Button>
      </header>

      {loading ? (
        <CardSkeleton />
      ) : combos.length === 0 ? (
        <p className="text-sm text-muted-foreground">No combos yet.</p>
      ) : (
        <ComboList
          combos={combos}
          onToggle={setActive}
          onEdit={setEditing}
          onDelete={setDeleting}
        />
      )}

      <ComboFormModal
        open={creating || editing !== null}
        combo={editing}
        onClose={() => {
          setCreating(false);
          setEditing(null);
        }}
        onSaved={load}
      />
      <ConfirmModal
        open={deleting !== null}
        title="Delete combo"
        message={`Delete "${deleting?.name ?? ""}"? This cannot be undone.`}
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
