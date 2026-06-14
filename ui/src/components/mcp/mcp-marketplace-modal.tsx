import * as React from "react";
import { Modal } from "@/components/ui/modal";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { SegmentedControl } from "@/components/ui/segmented-control";
import { apiFetch } from "@/lib/api";
import { toInstancePayload } from "@/lib/mcp-install";
import { useNotificationStore } from "@/stores/notification";
import type { McpClient } from "@/lib/types";

export interface McpMarketplaceModalProps {
  open: boolean;
  onClose: () => void;
  onAdded?: () => void;
}

// McpMarketplaceModal (PAR-UI-054, §1.6) browses the in-tree mcp contract — the
// ref's cli-tools registry endpoints are absent in g0router, so the browse/install
// flow is REMAPPED to GET /api/mcp/clients (catalog) + POST /api/mcp/instances
// (install). Variant-HAVE; no Go backend yet (§8 ESC-1a).
function McpMarketplaceModal({ open, onClose, onAdded }: McpMarketplaceModalProps) {
  const pushToast = useNotificationStore((state) => state.push);
  const [clients, setClients] = React.useState<McpClient[]>([]);
  const [loading, setLoading] = React.useState(false);
  const [query, setQuery] = React.useState("");
  const [filter, setFilter] = React.useState("all");
  const [installing, setInstalling] = React.useState<string | null>(null);

  React.useEffect(() => {
    if (!open) return;
    setLoading(true);
    apiFetch<McpClient[]>("/api/mcp/clients")
      .then((rows) => {
        setClients(rows ?? []);
        setLoading(false);
      })
      .catch(() => {
        setClients([]);
        setLoading(false);
        pushToast({ message: "Failed to load MCP servers" });
      });
  }, [open, pushToast]);

  const transports = React.useMemo(() => {
    const set = new Set(clients.map((c) => c.Transport));
    return Array.from(set);
  }, [clients]);

  const filterOptions = React.useMemo(
    () => [
      { value: "all", label: "All" },
      ...transports.map((t) => ({ value: t, label: t })),
    ],
    [transports]
  );

  const visible = clients.filter((c) => {
    const matchesQuery = c.Name.toLowerCase().includes(query.toLowerCase());
    const matchesFilter = filter === "all" || c.Transport === filter;
    return matchesQuery && matchesFilter;
  });

  async function install(client: McpClient) {
    setInstalling(client.ID);
    try {
      await apiFetch("/api/mcp/instances", {
        method: "POST",
        body: JSON.stringify(toInstancePayload(client)),
      });
      pushToast({ message: `Installed ${client.Name}` });
      onAdded?.();
      onClose();
    } catch {
      pushToast({ message: "Failed to install the MCP server" });
    } finally {
      setInstalling(null);
    }
  }

  return (
    <Modal open={open} onClose={onClose} title="MCP marketplace" size="lg">
      <div className="flex flex-col gap-4">
        <Input
          id="mcp-marketplace-search"
          placeholder="Search servers"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
        />
        <SegmentedControl
          options={filterOptions}
          value={filter}
          onChange={setFilter}
        />
        {loading ? (
          <p className="text-sm text-muted-foreground">Loading servers…</p>
        ) : visible.length === 0 ? (
          <p className="text-sm text-muted-foreground">No servers found.</p>
        ) : (
          <div className="flex flex-col gap-2">
            {visible.map((client) => (
              <div
                key={client.ID}
                data-testid="mcp-marketplace-row"
                className="flex items-center justify-between rounded-lg border border-border px-4 py-3"
              >
                <div className="flex items-center gap-3">
                  <Badge variant="neutral" size="sm">
                    {client.Transport}
                  </Badge>
                  <div>
                    <p className="text-sm font-medium text-foreground">
                      {client.Name}
                    </p>
                    {client.URL ? (
                      <p className="text-xs text-muted-foreground">{client.URL}</p>
                    ) : client.Command ? (
                      <p className="text-xs text-muted-foreground">
                        {client.Command}
                      </p>
                    ) : null}
                  </div>
                </div>
                <Button
                  data-testid="mcp-marketplace-install"
                  variant="primary"
                  size="sm"
                  loading={installing === client.ID}
                  onClick={() => install(client)}
                >
                  Install
                </Button>
              </div>
            ))}
          </div>
        )}
      </div>
    </Modal>
  );
}

export { McpMarketplaceModal };
