import * as React from "react";
import { createFileRoute } from "@tanstack/react-router";
import { apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Toggle } from "@/components/ui/toggle";
import { Badge } from "@/components/ui/badge";
import { ConfirmModal } from "@/components/ui/confirm-modal";
import { CardSkeleton } from "@/components/ui/skeleton";
import {
  VirtualKeyFormModal,
  type VirtualKeyRecord,
} from "@/components/keys/virtual-key-form-modal";
import { useNotificationStore } from "@/stores/notification";

export const Route = createFileRoute("/virtual-keys")({
  component: VirtualKeysPage,
});

// VirtualKeysPage (PAR-UI-130 subset) lists the virtual keys from the REAL w5-g VK
// CRUD and creates/edits them via the form modal (which embeds the KeyIDs editor).
function VirtualKeysPage() {
  const pushToast = useNotificationStore((state) => state.push);
  const [keys, setKeys] = React.useState<VirtualKeyRecord[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [formOpen, setFormOpen] = React.useState(false);
  const [editing, setEditing] = React.useState<VirtualKeyRecord | null>(null);
  const [deleting, setDeleting] = React.useState<VirtualKeyRecord | null>(null);
  const [deleteBusy, setDeleteBusy] = React.useState(false);

  const load = React.useCallback(() => {
    setLoading(true);
    apiFetch<{ virtual_keys: VirtualKeyRecord[] }>("/api/virtual-keys")
      .then((data) => {
        setKeys(data?.virtual_keys ?? []);
        setLoading(false);
      })
      .catch(() => {
        setKeys([]);
        setLoading(false);
        pushToast({ message: "Failed to load virtual keys" });
      });
  }, [pushToast]);

  React.useEffect(() => {
    load();
  }, [load]);

  function openCreate() {
    setEditing(null);
    setFormOpen(true);
  }

  function openEdit(vk: VirtualKeyRecord) {
    setEditing(vk);
    setFormOpen(true);
  }

  async function setActive(vk: VirtualKeyRecord, active: boolean) {
    setKeys((prev) => prev.map((k) => (k.id === vk.id ? { ...k, is_active: active } : k)));
    try {
      await apiFetch(`/api/virtual-keys/${vk.id}`, {
        method: "PUT",
        body: JSON.stringify({
          name: vk.name,
          provider_configs: vk.provider_configs,
          budget: vk.budget,
          rate_limit_rpm: vk.rate_limit_rpm,
          is_active: active,
        }),
      });
    } catch {
      pushToast({ message: "Failed to update the virtual key" });
      load();
    }
  }

  async function confirmDelete() {
    if (!deleting) return;
    setDeleteBusy(true);
    try {
      await apiFetch(`/api/virtual-keys/${deleting.id}`, { method: "DELETE" });
      setKeys((prev) => prev.filter((k) => k.id !== deleting.id));
      pushToast({ message: "Virtual key deleted" });
      setDeleting(null);
    } catch {
      pushToast({ message: "Failed to delete the virtual key" });
    } finally {
      setDeleteBusy(false);
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <header className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-foreground">Virtual Keys</h1>
        <Button
          data-testid="create-vk-trigger"
          variant="primary"
          size="sm"
          onClick={openCreate}
        >
          Create virtual key
        </Button>
      </header>

      {loading ? (
        <CardSkeleton />
      ) : keys.length === 0 ? (
        <p className="text-sm text-muted-foreground">No virtual keys yet.</p>
      ) : (
        <div className="flex flex-col gap-2">
          {keys.map((vk) => (
            <div
              key={vk.id}
              data-testid="virtual-key-row"
              className="flex items-center justify-between rounded-lg border border-border px-4 py-3"
            >
              <div className="flex flex-col">
                <span className="text-sm font-medium text-foreground">{vk.name}</span>
                <span className="text-xs text-muted-foreground">
                  {vk.budget
                    ? `$${vk.budget.used ?? 0} / $${vk.budget.limit} ${vk.budget.period}`
                    : "No budget"}
                  {vk.rate_limit_rpm != null ? ` · ${vk.rate_limit_rpm} RPM` : ""}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <Badge variant={vk.is_active ? "success" : "neutral"} size="sm">
                  {vk.is_active ? "active" : "inactive"}
                </Badge>
                <Toggle
                  checked={vk.is_active}
                  onCheckedChange={(checked) => setActive(vk, checked)}
                  aria-label={`Toggle ${vk.name}`}
                />
                <Button variant="ghost" size="sm" onClick={() => openEdit(vk)}>
                  Edit
                </Button>
                <Button
                  data-testid="delete-vk"
                  variant="danger"
                  size="sm"
                  onClick={() => setDeleting(vk)}
                >
                  Delete
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}

      <VirtualKeyFormModal
        open={formOpen}
        editing={editing}
        onClose={() => setFormOpen(false)}
        onSaved={load}
      />
      <ConfirmModal
        open={deleting !== null}
        title="Delete virtual key"
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
