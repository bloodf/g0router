import * as React from "react";
import { createFileRoute } from "@tanstack/react-router";
import { apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Toggle } from "@/components/ui/toggle";
import { ConfirmModal } from "@/components/ui/confirm-modal";
import { CardSkeleton } from "@/components/ui/skeleton";
import { ProxyPoolFormModal } from "@/components/platform/proxy-pool-form-modal";
import { useNotificationStore } from "@/stores/notification";
import type { ProxyPool } from "@/lib/types";

export const Route = createFileRoute("/proxy-pools")({
  component: ProxyPoolsPage,
});

// ProxyPoolsPage (PAR-UI-019/104/105) lists proxy pools from GET /api/proxy-pools
// and drives create/edit (ProxyPoolFormModal), per-pool connectivity test
// (POST /api/proxy-pools/{id}/test) and delete (ConfirmModal, DELETE). PARTIAL
// against the registered mock; no Go backend exists yet (§1.4/§8 ESC-1b).
function ProxyPoolsPage() {
  const pushToast = useNotificationStore((state) => state.push);
  const [pools, setPools] = React.useState<ProxyPool[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [editing, setEditing] = React.useState<ProxyPool | null>(null);
  const [creating, setCreating] = React.useState(false);
  const [deleting, setDeleting] = React.useState<ProxyPool | null>(null);
  const [deleteBusy, setDeleteBusy] = React.useState(false);
  const [testing, setTesting] = React.useState<string | null>(null);

  const load = React.useCallback(() => {
    setLoading(true);
    apiFetch<ProxyPool[]>("/api/proxy-pools")
      .then((rows) => {
        setPools(rows ?? []);
        setLoading(false);
      })
      .catch(() => {
        setPools([]);
        setLoading(false);
        pushToast({ message: "Failed to load proxy pools" });
      });
  }, [pushToast]);

  React.useEffect(() => {
    load();
  }, [load]);

  async function setActive(pool: ProxyPool, active: boolean) {
    setPools((prev) =>
      prev.map((p) => (p.id === pool.id ? { ...p, is_active: active } : p))
    );
    try {
      await apiFetch(`/api/proxy-pools/${pool.id}`, {
        method: "PUT",
        body: JSON.stringify({ ...pool, is_active: active }),
      });
    } catch {
      pushToast({ message: "Failed to update the pool" });
      load();
    }
  }

  async function testPool(pool: ProxyPool) {
    setTesting(pool.id);
    try {
      const result = await apiFetch<{ ok: boolean; latency_ms: number }>(
        `/api/proxy-pools/${pool.id}/test`,
        { method: "POST" }
      );
      pushToast({
        message: result?.ok
          ? `Pool reachable (${result.latency_ms} ms)`
          : "Pool unreachable",
      });
    } catch {
      pushToast({ message: "Failed to test the pool" });
    } finally {
      setTesting(null);
    }
  }

  async function confirmDelete() {
    if (!deleting) return;
    setDeleteBusy(true);
    try {
      await apiFetch(`/api/proxy-pools/${deleting.id}`, { method: "DELETE" });
      setPools((prev) => prev.filter((p) => p.id !== deleting.id));
      pushToast({ message: "Pool deleted" });
      setDeleting(null);
    } catch {
      pushToast({ message: "Failed to delete the pool" });
    } finally {
      setDeleteBusy(false);
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <header className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-foreground">Proxy Pools</h1>
        <Button
          data-testid="proxy-pool-new"
          variant="primary"
          size="sm"
          onClick={() => setCreating(true)}
        >
          New pool
        </Button>
      </header>

      {loading ? (
        <CardSkeleton />
      ) : pools.length === 0 ? (
        <p className="text-sm text-muted-foreground">No proxy pools yet.</p>
      ) : (
        <div className="flex flex-col gap-2">
          {pools.map((pool) => (
            <div
              key={pool.id}
              data-testid="proxy-pool-row"
              className="flex items-center justify-between rounded-lg border border-border px-4 py-3"
            >
              <div className="flex flex-col gap-1">
                <p className="text-sm font-medium text-foreground">{pool.name}</p>
                <p className="text-xs text-muted-foreground">
                  {pool.protocol}://{pool.host}:{pool.port}
                </p>
              </div>
              <div className="flex items-center gap-2">
                <Badge
                  variant={pool.last_check_status === "ok" ? "success" : "neutral"}
                  size="sm"
                >
                  {pool.last_check_status || "unknown"}
                </Badge>
                <Toggle
                  checked={pool.is_active}
                  onCheckedChange={(checked) => setActive(pool, checked)}
                  aria-label={`Toggle ${pool.name}`}
                />
                <Button
                  data-testid="proxy-pool-test"
                  variant="ghost"
                  size="sm"
                  loading={testing === pool.id}
                  onClick={() => testPool(pool)}
                >
                  Test
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setEditing(pool)}
                >
                  Edit
                </Button>
                <Button
                  data-testid="proxy-pool-delete"
                  variant="danger"
                  size="sm"
                  onClick={() => setDeleting(pool)}
                >
                  Delete
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}

      <ProxyPoolFormModal
        open={creating || editing !== null}
        pool={editing}
        onClose={() => {
          setCreating(false);
          setEditing(null);
        }}
        onSaved={load}
      />
      <ConfirmModal
        open={deleting !== null}
        title="Delete pool"
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
