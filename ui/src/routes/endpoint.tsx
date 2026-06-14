import * as React from "react";
import { createFileRoute } from "@tanstack/react-router";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { ApiKeysPanel } from "@/components/keys/api-keys-panel";
import { ProviderNodeModal } from "@/components/keys/provider-node-modal";
import { useNotificationStore } from "@/stores/notification";

export const Route = createFileRoute("/endpoint")({
  component: EndpointPage,
});

// EndpointPage (PAR-UI-006) is the operator's "how to call the gateway" panel: it
// shows the OpenAI-compatible base URL with a copy action and a sample request,
// embeds a compact API-keys widget, and exposes the custom provider-node flow.
function EndpointPage() {
  const pushToast = useNotificationStore((state) => state.push);
  const [nodeOpen, setNodeOpen] = React.useState(false);
  const origin =
    typeof window !== "undefined" ? window.location.origin : "http://localhost";
  const baseUrl = `${origin}/v1`;

  async function copyBaseUrl() {
    try {
      await navigator.clipboard.writeText(baseUrl);
      pushToast({ message: "Base URL copied" });
    } catch {
      pushToast({ message: "Copy failed" });
    }
  }

  const sample = `curl ${baseUrl}/chat/completions \\
  -H "Authorization: Bearer $G0ROUTER_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{"model":"gpt-4o","messages":[{"role":"user","content":"hi"}]}'`;

  return (
    <div className="flex flex-col gap-6">
      <header className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-foreground">Endpoint</h1>
          <p className="text-sm text-muted-foreground">
            Point any OpenAI-compatible client at the gateway base URL.
          </p>
        </div>
        <Button
          data-testid="add-node-trigger"
          variant="outline"
          size="sm"
          onClick={() => setNodeOpen(true)}
        >
          Add custom node
        </Button>
      </header>

      <Card>
        <CardHeader>
          <CardTitle>Base URL</CardTitle>
        </CardHeader>
        <CardContent className="mt-4 flex flex-col gap-3">
          <div className="flex items-center gap-2">
            <code
              data-testid="base-url"
              className="break-all rounded-md border border-border bg-muted px-3 py-2 font-mono text-sm"
            >
              {baseUrl}
            </code>
            <Button variant="ghost" size="sm" onClick={copyBaseUrl}>
              Copy
            </Button>
          </div>
          <pre className="overflow-x-auto rounded-md border border-border bg-muted px-3 py-2 font-mono text-xs text-muted-foreground">
            {sample}
          </pre>
        </CardContent>
      </Card>

      <ApiKeysPanel compact />

      <ProviderNodeModal open={nodeOpen} onClose={() => setNodeOpen(false)} />
    </div>
  );
}
