import * as React from "react";
import { apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Toggle } from "@/components/ui/toggle";
import { Badge } from "@/components/ui/badge";
import { Modal } from "@/components/ui/modal";
import { ConfirmModal } from "@/components/ui/confirm-modal";
import { useNotificationStore } from "@/stores/notification";

// ApiKeyRow mirrors the REAL Go apiKeyDTO (internal/admin/apikeys.go:11-17):
// {id, key, name, machine_id, is_active, created_at} (plan §1.3 / §8 ESC-2).
export interface ApiKeyRow {
  id: string;
  key?: string;
  name: string;
  machine_id?: string;
  is_active: boolean;
  created_at?: string;
}

export interface ApiKeysPanelProps {
  // initialKeys seeds the list for SSR/unit tests; when omitted the panel fetches
  // from /api/keys on mount.
  initialKeys?: ApiKeyRow[];
  // compact renders a tighter widget for embedding on the endpoint page.
  compact?: boolean;
}

// ApiKeysPanel (PAR-UI-006/115) lists, creates, toggles, and deletes API keys
// against the REAL /api/keys Go CRUD. The created key value is shown once (the Go
// apiKeyDTO returns `key`).
export function ApiKeysPanel({ initialKeys, compact = false }: ApiKeysPanelProps) {
  const pushToast = useNotificationStore((state) => state.push);
  const [keys, setKeys] = React.useState<ApiKeyRow[]>(initialKeys ?? []);
  const [loading, setLoading] = React.useState(initialKeys === undefined);
  const [createOpen, setCreateOpen] = React.useState(false);
  const [createName, setCreateName] = React.useState("");
  const [createBusy, setCreateBusy] = React.useState(false);
  const [createdKey, setCreatedKey] = React.useState<string | null>(null);
  const [deleting, setDeleting] = React.useState<ApiKeyRow | null>(null);
  const [deleteBusy, setDeleteBusy] = React.useState(false);

  const load = React.useCallback(() => {
    setLoading(true);
    apiFetch<{ keys: ApiKeyRow[] }>("/api/keys")
      .then((data) => {
        setKeys(data?.keys ?? []);
        setLoading(false);
      })
      .catch(() => {
        setKeys([]);
        setLoading(false);
        pushToast({ message: "Failed to load API keys" });
      });
  }, [pushToast]);

  React.useEffect(() => {
    if (initialKeys !== undefined) return;
    load();
  }, [initialKeys, load]);

  async function createKey() {
    if (!createName.trim()) return;
    setCreateBusy(true);
    try {
      const created = await apiFetch<{ key: string; name: string; id: string; machine_id: string }>(
        "/api/keys",
        { method: "POST", body: JSON.stringify({ name: createName.trim() }) }
      );
      setCreatedKey(created?.key ?? null);
      setKeys((prev) => [
        ...prev,
        {
          id: created.id,
          key: created.key,
          name: created.name,
          machine_id: created.machine_id,
          is_active: true,
        },
      ]);
      setCreateName("");
      pushToast({ message: "API key created" });
    } catch {
      pushToast({ message: "Failed to create the API key" });
    } finally {
      setCreateBusy(false);
    }
  }

  async function setActive(row: ApiKeyRow, active: boolean) {
    setKeys((prev) => prev.map((k) => (k.id === row.id ? { ...k, is_active: active } : k)));
    try {
      await apiFetch(`/api/keys/${row.id}`, {
        method: "PUT",
        body: JSON.stringify({ is_active: active }),
      });
    } catch {
      pushToast({ message: "Failed to update the API key" });
      load();
    }
  }

  async function confirmDelete() {
    if (!deleting) return;
    setDeleteBusy(true);
    try {
      await apiFetch(`/api/keys/${deleting.id}`, { method: "DELETE" });
      setKeys((prev) => prev.filter((k) => k.id !== deleting.id));
      pushToast({ message: "API key deleted" });
      setDeleting(null);
    } catch {
      pushToast({ message: "Failed to delete the API key" });
    } finally {
      setDeleteBusy(false);
    }
  }

  function closeCreate() {
    setCreateOpen(false);
    setCreatedKey(null);
    setCreateName("");
  }

  const rows = (
    <div className="flex flex-col gap-2">
      {loading ? (
        <p className="text-sm text-muted-foreground">Loading API keys…</p>
      ) : keys.length === 0 ? (
        <p className="text-sm text-muted-foreground">No API keys yet.</p>
      ) : (
        keys.map((row) => (
          <div
            key={row.id}
            data-testid="api-key-row"
            className="flex items-center justify-between rounded-lg border border-border px-4 py-3"
          >
            <div className="flex flex-col">
              <span className="text-sm font-medium text-foreground">{row.name}</span>
              {row.key ? (
                <span className="font-mono text-xs text-muted-foreground">{row.key}</span>
              ) : null}
            </div>
            <div className="flex items-center gap-2">
              <Badge variant={row.is_active ? "success" : "neutral"} size="sm">
                {row.is_active ? "active" : "inactive"}
              </Badge>
              <Toggle
                checked={row.is_active}
                onCheckedChange={(checked) => setActive(row, checked)}
                aria-label={`Toggle ${row.name}`}
              />
              <Button
                data-testid="delete-key"
                variant="danger"
                size="sm"
                onClick={() => setDeleting(row)}
              >
                Delete
              </Button>
            </div>
          </div>
        ))
      )}
    </div>
  );

  const createTrigger = (
    <Button
      data-testid="create-key-trigger"
      variant="primary"
      size="sm"
      onClick={() => setCreateOpen(true)}
    >
      Create key
    </Button>
  );

  const body = (
    <>
      {rows}
      <Modal open={createOpen} onClose={closeCreate} title="Create API key">
        {createdKey ? (
          <div className="flex flex-col gap-4">
            <p className="text-sm text-muted-foreground">
              Copy this key now — it will not be shown again.
            </p>
            <code
              data-testid="created-key-value"
              className="break-all rounded-md border border-border bg-muted px-3 py-2 font-mono text-xs"
            >
              {createdKey}
            </code>
            <div className="flex justify-end">
              <Button variant="primary" onClick={closeCreate}>
                Done
              </Button>
            </div>
          </div>
        ) : (
          <div className="flex flex-col gap-4">
            <Input
              data-testid="create-key-name"
              label="Name"
              value={createName}
              onChange={(event) => setCreateName(event.target.value)}
              placeholder="e.g. Production key"
            />
            <div className="flex justify-end gap-2">
              <Button variant="ghost" onClick={closeCreate}>
                Cancel
              </Button>
              <Button
                data-testid="create-key-submit"
                variant="primary"
                loading={createBusy}
                onClick={createKey}
              >
                Create
              </Button>
            </div>
          </div>
        )}
      </Modal>
      <ConfirmModal
        open={deleting !== null}
        title="Delete API key"
        message={`Delete "${deleting?.name ?? ""}"? This cannot be undone.`}
        confirmLabel="Delete"
        cancelLabel="Cancel"
        variant="danger"
        loading={deleteBusy}
        onConfirm={confirmDelete}
        onCancel={() => setDeleting(null)}
      />
    </>
  );

  if (compact) {
    return (
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>API Keys</CardTitle>
          {createTrigger}
        </CardHeader>
        <CardContent className="mt-4">{body}</CardContent>
      </Card>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <header className="flex items-center justify-between">
        <h2 className="text-lg font-semibold text-foreground">API Keys</h2>
        {createTrigger}
      </header>
      {body}
    </div>
  );
}
