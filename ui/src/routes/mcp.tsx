import * as React from "react";
import { createFileRoute, Outlet, useMatches } from "@tanstack/react-router";
import { apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { ConfirmModal } from "@/components/ui/confirm-modal";
import { CardSkeleton } from "@/components/ui/skeleton";
import { McpClientCard } from "@/components/mcp/mcp-client-card";
import { McpMarketplaceModal } from "@/components/mcp/mcp-marketplace-modal";
import { useNotificationStore } from "@/stores/notification";
import type { McpClient, McpInstance } from "@/lib/types";

export const Route = createFileRoute("/mcp")({
  component: McpPage,
});

// McpPage (PAR-UI-130 /mcp, g0router-EXTRA) lists MCP clients + instances from
// GET /api/mcp/{clients,instances}, installs via the marketplace modal
// (POST /api/mcp/instances, §1.6), deletes instances (DELETE …/{id}), and starts
// per-instance OAuth (POST …/{id}/auth/start). Reads PascalCase keys (§1.2/§1.4).
// Variant-HAVE against the mcp mock; no Go backend yet (§8 ESC-1a). When the
// nested /mcp/tools route (§1.8) is active it renders the <Outlet> child only.
function McpPage() {
  const matches = useMatches();
  const onTools = matches.some((m) => m.routeId === "/mcp/tools");
  if (onTools) {
    return <Outlet />;
  }
  return <McpClientsView />;
}

function McpClientsView() {
  const pushToast = useNotificationStore((state) => state.push);
  const [clients, setClients] = React.useState<McpClient[]>([]);
  const [instances, setInstances] = React.useState<McpInstance[]>([]);
  const [loading, setLoading] = React.useState(true);
  const [browsing, setBrowsing] = React.useState(false);
  const [deleting, setDeleting] = React.useState<McpInstance | null>(null);
  const [deleteBusy, setDeleteBusy] = React.useState(false);

  const load = React.useCallback(() => {
    setLoading(true);
    Promise.all([
      apiFetch<McpClient[]>("/api/mcp/clients"),
      apiFetch<McpInstance[]>("/api/mcp/instances"),
    ])
      .then(([c, i]) => {
        setClients(c ?? []);
        setInstances(i ?? []);
        setLoading(false);
      })
      .catch(() => {
        setClients([]);
        setInstances([]);
        setLoading(false);
        pushToast({ message: "Failed to load MCP data" });
      });
  }, [pushToast]);

  React.useEffect(() => {
    load();
  }, [load]);

  async function startAuth(instance: McpInstance) {
    try {
      const res = await apiFetch<{ url: string }>(
        `/api/mcp/instances/${instance.ID}/auth/start`,
        { method: "POST" }
      );
      if (res?.url) window.open(res.url, "_blank", "noopener");
    } catch {
      pushToast({ message: "Failed to start authentication" });
    }
  }

  async function confirmDelete() {
    if (!deleting) return;
    setDeleteBusy(true);
    try {
      await apiFetch(`/api/mcp/instances/${deleting.ID}`, { method: "DELETE" });
      setInstances((prev) => prev.filter((i) => i.ID !== deleting.ID));
      pushToast({ message: "Instance deleted" });
      setDeleting(null);
    } catch {
      pushToast({ message: "Failed to delete the instance" });
    } finally {
      setDeleteBusy(false);
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <header className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-foreground">MCP</h1>
        <Button
          data-testid="mcp-marketplace-open"
          variant="primary"
          size="sm"
          onClick={() => setBrowsing(true)}
        >
          Browse marketplace
        </Button>
      </header>

      {loading ? (
        <CardSkeleton />
      ) : (
        <>
          <section className="flex flex-col gap-2">
            <h2 className="text-sm font-semibold text-foreground">Instances</h2>
            {instances.length === 0 ? (
              <p className="text-sm text-muted-foreground">No MCP instances yet.</p>
            ) : (
              instances.map((instance) => (
                <McpClientCard
                  key={instance.ID}
                  testId="mcp-instance-row"
                  name={instance.Name}
                  transport={instance.Transport}
                  healthStatus={instance.HealthStatus}
                  isActive={instance.IsActive}
                  actions={
                    <>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => startAuth(instance)}
                      >
                        Authenticate
                      </Button>
                      <Button
                        data-testid="mcp-instance-delete"
                        variant="danger"
                        size="sm"
                        onClick={() => setDeleting(instance)}
                      >
                        Delete
                      </Button>
                    </>
                  }
                />
              ))
            )}
          </section>

          <section className="flex flex-col gap-2">
            <h2 className="text-sm font-semibold text-foreground">
              Configured servers
            </h2>
            {clients.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                No MCP servers configured.
              </p>
            ) : (
              clients.map((client) => (
                <McpClientCard
                  key={client.ID}
                  testId="mcp-client-row"
                  name={client.Name}
                  transport={client.Transport}
                  healthStatus={client.HealthStatus}
                  isActive={client.IsActive}
                />
              ))
            )}
          </section>
        </>
      )}

      <McpMarketplaceModal
        open={browsing}
        onClose={() => setBrowsing(false)}
        onAdded={load}
      />
      <ConfirmModal
        open={deleting !== null}
        title="Delete instance"
        message={`Delete "${deleting?.Name ?? ""}"? This cannot be undone.`}
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
