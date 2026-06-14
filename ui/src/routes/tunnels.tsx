import * as React from "react";
import { createFileRoute } from "@tanstack/react-router";
import { apiFetch } from "@/lib/api";
import { Badge } from "@/components/ui/badge";
import { Toggle } from "@/components/ui/toggle";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { CardSkeleton } from "@/components/ui/skeleton";
import { useNotificationStore } from "@/stores/notification";
import type { Tunnel } from "@/lib/types";

export const Route = createFileRoute("/tunnels")({
  component: TunnelsPage,
});

// TunnelsPage (PAR-UI-112/113/114) renders one card per tunnel type
// (cloudflare/tailscale) from GET /api/tunnels with an enable/disable toggle:
// enabling POSTs /api/tunnels/{type}, disabling DELETEs /api/tunnels/{type}. Status
// is read via plain REST (mount + an OPTIONAL health read), NOT a server-sent
// stream (§1.5 REST-poll). PARTIAL vs the registered mock; no Go backend yet
// (§8 ESC-1c).
function TunnelsPage() {
  const pushToast = useNotificationStore((state) => state.push);
  const [tunnels, setTunnels] = React.useState<Tunnel[]>([]);
  const [healthy, setHealthy] = React.useState<boolean | null>(null);
  const [loading, setLoading] = React.useState(true);

  const load = React.useCallback(() => {
    setLoading(true);
    apiFetch<Tunnel[]>("/api/tunnels")
      .then((rows) => {
        setTunnels(rows ?? []);
        setLoading(false);
      })
      .catch(() => {
        setTunnels([]);
        setLoading(false);
        pushToast({ message: "Failed to load tunnels" });
      });
    apiFetch<{ healthy: boolean }>("/api/tunnels/health")
      .then((result) => setHealthy(result?.healthy ?? null))
      .catch(() => setHealthy(null));
  }, [pushToast]);

  React.useEffect(() => {
    load();
  }, [load]);

  async function toggleTunnel(tunnel: Tunnel, next: boolean) {
    setTunnels((prev) =>
      prev.map((t) =>
        t.type === tunnel.type
          ? { ...t, is_enabled: next, status: next ? "active" : "inactive" }
          : t
      )
    );
    try {
      await apiFetch(`/api/tunnels/${tunnel.type}`, {
        method: next ? "POST" : "DELETE",
      });
    } catch {
      pushToast({ message: "Failed to update the tunnel" });
      load();
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <header className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-foreground">Tunnels</h1>
        {healthy !== null ? (
          <Badge variant={healthy ? "success" : "error"} size="sm">
            {healthy ? "healthy" : "unhealthy"}
          </Badge>
        ) : null}
      </header>

      {loading ? (
        <CardSkeleton />
      ) : tunnels.length === 0 ? (
        <p className="text-sm text-muted-foreground">No tunnels configured.</p>
      ) : (
        <div className="flex flex-col gap-4">
          {tunnels.map((tunnel) => (
            <Card key={tunnel.type} data-testid="tunnel-card">
              <CardHeader className="flex flex-row items-center justify-between">
                <CardTitle className="capitalize">{tunnel.type}</CardTitle>
                <div className="flex items-center gap-2">
                  <Badge
                    variant={tunnel.status === "active" ? "success" : "neutral"}
                    size="sm"
                  >
                    {tunnel.status}
                  </Badge>
                  <Toggle
                    data-testid="tunnel-toggle"
                    checked={tunnel.is_enabled}
                    onCheckedChange={(checked) => toggleTunnel(tunnel, checked)}
                    aria-label={`Toggle ${tunnel.type} tunnel`}
                  />
                </div>
              </CardHeader>
              <CardContent>
                <p className="break-all text-sm text-muted-foreground">
                  {tunnel.url}
                </p>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  );
}
