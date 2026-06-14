import * as React from "react";
import { createFileRoute } from "@tanstack/react-router";
import { apiFetch } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Toggle } from "@/components/ui/toggle";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { CardSkeleton } from "@/components/ui/skeleton";
import { useNotificationStore } from "@/stores/notification";
import type { MitmTool } from "@/lib/types";

export const Route = createFileRoute("/mitm")({
  component: MitmPage,
});

interface MitmStatus {
  enabled: boolean;
  tools: MitmTool[];
}

// MitmPage (PAR-UI-013) renders the MITM proxy config surface: a status panel with
// a global enable toggle (POST /api/mitm/toggle), a per-tool list with enable
// toggles (POST /api/mitm/tools/{id}), and a CA-certificate download. The cert is
// served as raw PEM (NOT a {data} envelope), so the download uses a plain fetch +
// anchor, not apiFetch (§1.2 caveat / §1.3). PARTIAL against the registered mock;
// no Go backend exists yet (§8 ESC-1a).
function MitmPage() {
  const pushToast = useNotificationStore((state) => state.push);
  const [enabled, setEnabled] = React.useState(false);
  const [tools, setTools] = React.useState<MitmTool[]>([]);
  const [loading, setLoading] = React.useState(true);

  const load = React.useCallback(() => {
    setLoading(true);
    apiFetch<MitmStatus>("/api/mitm/status")
      .then((status) => {
        setEnabled(status?.enabled ?? false);
        setTools(status?.tools ?? []);
        setLoading(false);
      })
      .catch(() => {
        setEnabled(false);
        setTools([]);
        setLoading(false);
        pushToast({ message: "Failed to load MITM status" });
      });
  }, [pushToast]);

  React.useEffect(() => {
    load();
  }, [load]);

  async function toggleEnabled(next: boolean) {
    setEnabled(next);
    try {
      await apiFetch("/api/mitm/toggle", { method: "POST" });
    } catch {
      pushToast({ message: "Failed to toggle MITM" });
      load();
    }
  }

  async function toggleTool(tool: MitmTool) {
    const next = !tool.enabled;
    setTools((prev) =>
      prev.map((t) =>
        t.id === tool.id
          ? { ...t, enabled: next, status: next ? "active" : "inactive" }
          : t
      )
    );
    try {
      await apiFetch(`/api/mitm/tools/${tool.id}`, { method: "POST" });
    } catch {
      pushToast({ message: "Failed to toggle tool" });
      load();
    }
  }

  async function downloadCaCert() {
    try {
      // The CA cert is served as raw PEM (application/x-pem-file), NOT a {data}
      // envelope, so this bypasses apiFetch and streams the body directly.
      const response = await fetch(
        `${window.location.origin}/api/mitm/ca-cert`
      );
      if (!response.ok) throw new Error(`HTTP ${response.status}`);
      const pem = await response.text();
      const blob = new Blob([pem], { type: "application/x-pem-file" });
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement("a");
      anchor.href = url;
      anchor.download = "g0router-ca.pem";
      document.body.appendChild(anchor);
      anchor.click();
      anchor.remove();
      URL.revokeObjectURL(url);
    } catch {
      pushToast({ message: "Failed to download CA certificate" });
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <header className="flex items-center justify-between">
        <h1 className="text-2xl font-semibold text-foreground">MITM</h1>
        <Button
          data-testid="mitm-ca-cert-download"
          variant="secondary"
          size="sm"
          onClick={downloadCaCert}
        >
          Download CA certificate
        </Button>
      </header>

      {loading ? (
        <CardSkeleton />
      ) : (
        <>
          <Card>
            <CardHeader>
              <CardTitle>Status</CardTitle>
            </CardHeader>
            <CardContent>
              <label className="flex items-center justify-between text-sm text-foreground">
                <span>
                  MITM proxy is{" "}
                  <Badge variant={enabled ? "success" : "neutral"} size="sm">
                    {enabled ? "enabled" : "disabled"}
                  </Badge>
                </span>
                <Toggle
                  data-testid="mitm-enable-toggle"
                  checked={enabled}
                  onCheckedChange={toggleEnabled}
                  aria-label="Toggle MITM proxy"
                />
              </label>
            </CardContent>
          </Card>

          {tools.length === 0 ? (
            <p className="text-sm text-muted-foreground">No MITM tools.</p>
          ) : (
            <div className="flex flex-col gap-2">
              {tools.map((tool) => (
                <div
                  key={tool.id}
                  data-testid="mitm-tool-row"
                  className="flex items-center justify-between rounded-lg border border-border px-4 py-3"
                >
                  <div className="flex flex-col gap-1">
                    <p className="text-sm font-medium text-foreground">
                      {tool.name}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {tool.dns_override || "no DNS override"}
                    </p>
                  </div>
                  <div className="flex items-center gap-2">
                    <Badge
                      variant={tool.status === "active" ? "success" : "neutral"}
                      size="sm"
                    >
                      {tool.status}
                    </Badge>
                    <Toggle
                      data-testid="mitm-tool-toggle"
                      checked={tool.enabled}
                      onCheckedChange={() => toggleTool(tool)}
                      aria-label={`Toggle ${tool.name}`}
                    />
                  </div>
                </div>
              ))}
            </div>
          )}
        </>
      )}
    </div>
  );
}
