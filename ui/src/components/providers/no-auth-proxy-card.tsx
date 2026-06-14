import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { ProviderIcon } from "@/components/ui/provider-icon";
import type { Provider } from "@/lib/types";

export interface NoAuthProxyCardProps {
  provider: Provider;
  onConnect?: (provider: Provider) => void;
}

// NoAuthProxyCard (PAR-UI-064) — the card for no-auth / local proxy providers
// (ollama, lm-studio, vllm) that connect without credentials (port of 9router
// NoAuthProxyCard.js).
function NoAuthProxyCard({ provider, onConnect }: NoAuthProxyCardProps) {
  return (
    <Card data-testid="no-auth-proxy-card" className="flex flex-col gap-3">
      <div className="flex items-center gap-3">
        <ProviderIcon slug={provider.id} name={provider.display_name} size="md" />
        <div>
          <h3 className="font-semibold text-foreground">
            {provider.display_name}
          </h3>
          <p className="text-xs text-muted-foreground">
            No credentials required — connects directly to a local or proxied
            endpoint.
          </p>
        </div>
      </div>
      <Button
        variant="primary"
        size="sm"
        onClick={() => onConnect?.(provider)}
      >
        Connect
      </Button>
    </Card>
  );
}

export { NoAuthProxyCard };
