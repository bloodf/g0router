import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { ProviderIcon } from "@/components/ui/provider-icon";
import type { Provider } from "@/lib/types";

export interface ProviderInfoCardProps {
  provider: Provider;
}

// ProviderInfoCard (PAR-UI-064) — a read-only summary of a provider's identity
// and capabilities (port of 9router ProviderInfoCard.js).
function ProviderInfoCard({ provider }: ProviderInfoCardProps) {
  return (
    <Card data-testid="provider-info-card" className="flex flex-col gap-3">
      <div className="flex items-center gap-3">
        <ProviderIcon slug={provider.id} name={provider.display_name} size="md" />
        <div>
          <h3 className="font-semibold text-foreground">
            {provider.display_name}
          </h3>
          <p className="text-xs text-muted-foreground">{provider.description}</p>
        </div>
      </div>
      {provider.capabilities.length > 0 ? (
        <div className="flex flex-wrap gap-1.5">
          {provider.capabilities.map((cap) => (
            <Badge key={cap} variant="primary" size="sm">
              {cap}
            </Badge>
          ))}
        </div>
      ) : null}
    </Card>
  );
}

export { ProviderInfoCard };
