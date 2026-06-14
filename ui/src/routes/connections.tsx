import * as React from "react";
import { createFileRoute } from "@tanstack/react-router";
import { apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Toggle } from "@/components/ui/toggle";
import { Badge } from "@/components/ui/badge";
import { ProviderIcon } from "@/components/ui/provider-icon";
import { ConfirmModal } from "@/components/ui/confirm-modal";
import { CardSkeleton } from "@/components/ui/skeleton";
import { EditConnectionModal } from "@/components/providers/edit-connection-modal";
import { useNotificationStore } from "@/stores/notification";
import type { Connection } from "@/lib/types";

export const Route = createFileRoute("/connections")({
  component: ConnectionsPage,
});

// Raw connection rows may arrive in either the UI shape (mock / provider-shaped
// read) or the CRUD DTO shape (provider_id/kind/secret_set). normalizeConnection
// maps the CRUD DTO client-side (plan §8 ESCALATION-2) so the page never needs a
// Go change and stays consistent against the mock.
interface RawConnection {
  id: string;
  provider?: string;
  provider_id?: string;
  name: string;
  auth_type?: string;
  kind?: string;
  is_active?: boolean;
  models?: string[];
  priority?: number;
  needs_reauth?: boolean;
}

function normalizeConnection(raw: RawConnection): Connection {
  return {
    id: raw.id,
    provider: raw.provider ?? raw.provider_id ?? "",
    name: raw.name,
    auth_type: raw.auth_type ?? raw.kind ?? "api_key",
    is_active: raw.is_active ?? true,
    models: raw.models ?? [],
    priority: raw.priority ?? 0,
    needs_reauth: raw.needs_reauth ?? false,
  };
}

function ConnectionsPage() {
  const pushToast = useNotificationStore((state) => state.push);
  const [connections, setConnections] = React.useState<Connection[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [editing, setEditing] = React.useState<Connection | null>(null);
  const [deleting, setDeleting] = React.useState<Connection | null>(null);
  const [deleteBusy, setDeleteBusy] = React.useState(false);

  const load = React.useCallback(() => {
    setLoading(true);
    apiFetch<RawConnection[]>("/api/connections")
      .then((rows) => {
        setConnections((rows ?? []).map(normalizeConnection));
        setLoading(false);
      })
      .catch(() => {
        setConnections([]);
        setLoading(false);
        pushToast({ message: "Failed to load connections" });
      });
  }, [pushToast]);

  React.useEffect(() => {
    load();
  }, [load]);

  async function setActive(conn: Connection, active: boolean) {
    setConnections((prev) =>
      prev.map((c) => (c.id === conn.id ? { ...c, is_active: active } : c))
    );
    try {
      await apiFetch(`/api/connections/${conn.id}`, {
        method: "PUT",
        body: JSON.stringify({
          provider_id: conn.provider,
          name: conn.name,
          kind: conn.auth_type,
          is_active: active,
        }),
      });
    } catch {
      pushToast({ message: "Failed to update the connection" });
      load();
    }
  }

  async function testConnection(conn: Connection) {
    try {
      const result = await apiFetch<{ ok: boolean; latency_ms?: number }>(
        `/api/connections/${conn.id}/test`,
        { method: "POST" }
      );
      pushToast({
        message: result?.ok
          ? `${conn.name} OK (${result.latency_ms ?? 0}ms)`
          : `${conn.name} test failed`,
      });
    } catch {
      pushToast({ message: `${conn.name} test failed` });
    }
  }

  async function confirmDelete() {
    if (!deleting) return;
    setDeleteBusy(true);
    try {
      await apiFetch(`/api/connections/${deleting.id}`, { method: "DELETE" });
      setConnections((prev) => prev.filter((c) => c.id !== deleting.id));
      pushToast({ message: "Connection deleted" });
      setDeleting(null);
    } catch {
      pushToast({ message: "Failed to delete the connection" });
    } finally {
      setDeleteBusy(false);
    }
  }

  async function bulk(active: boolean) {
    try {
      await apiFetch(
        active ? "/api/connections/bulk-enable" : "/api/connections/bulk-disable",
        { method: "POST" }
      );
      setConnections((prev) => prev.map((c) => ({ ...c, is_active: active })));
    } catch {
      pushToast({ message: "Bulk update failed" });
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <header className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-foreground">Connections</h1>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => bulk(true)}>
            Enable all
          </Button>
          <Button variant="outline" size="sm" onClick={() => bulk(false)}>
            Disable all
          </Button>
        </div>
      </header>

      {loading ? (
        <CardSkeleton />
      ) : connections.length === 0 ? (
        <p className="text-sm text-muted-foreground">No connections yet.</p>
      ) : (
        <div className="flex flex-col gap-2">
          {connections.map((conn) => (
            <div
              key={conn.id}
              data-testid="connection-row"
              className="flex items-center justify-between rounded-lg border border-border px-4 py-3"
            >
              <div className="flex items-center gap-3">
                <ProviderIcon slug={conn.provider} name={conn.provider} size="sm" />
                <div>
                  <p className="text-sm font-medium text-foreground">{conn.name}</p>
                  <p className="text-xs text-muted-foreground">{conn.provider}</p>
                </div>
                <Badge variant="default" size="sm">
                  {conn.auth_type}
                </Badge>
                {conn.needs_reauth ? (
                  <Badge variant="error" size="sm">
                    needs reauth
                  </Badge>
                ) : null}
              </div>
              <div className="flex items-center gap-2">
                <Toggle
                  checked={conn.is_active}
                  onCheckedChange={(checked) => setActive(conn, checked)}
                  aria-label={`Toggle ${conn.name}`}
                />
                <Button
                  data-testid="connection-test"
                  variant="ghost"
                  size="sm"
                  onClick={() => testConnection(conn)}
                >
                  Test
                </Button>
                <Button variant="ghost" size="sm" onClick={() => setEditing(conn)}>
                  Edit
                </Button>
                <Button
                  data-testid="connection-delete"
                  variant="danger"
                  size="sm"
                  onClick={() => setDeleting(conn)}
                >
                  Delete
                </Button>
              </div>
            </div>
          ))}
        </div>
      )}

      <EditConnectionModal
        open={editing !== null}
        connection={editing}
        onClose={() => setEditing(null)}
        onSaved={load}
      />
      <ConfirmModal
        open={deleting !== null}
        title="Delete connection"
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
