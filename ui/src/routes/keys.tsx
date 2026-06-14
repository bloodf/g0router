import { createFileRoute } from "@tanstack/react-router";
import { ApiKeysPanel } from "@/components/keys/api-keys-panel";

export const Route = createFileRoute("/keys")({
  component: KeysPage,
});

// KeysPage (PAR-UI-006/115) renders the full API-keys management panel against the
// REAL /api/keys Go CRUD.
function KeysPage() {
  return (
    <div className="flex flex-col gap-6">
      <header>
        <h1 className="text-2xl font-semibold text-foreground">API Keys</h1>
        <p className="text-sm text-muted-foreground">
          Create and manage the API keys clients use to call the gateway.
        </p>
      </header>
      <ApiKeysPanel />
    </div>
  );
}
