import { Card } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { ProviderIcon } from "@/components/ui/provider-icon";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import type { Provider } from "@/lib/types";

export interface ProviderCardProps {
  provider: Provider;
  onSelect?: (provider: Provider) => void;
  onOAuth?: (provider: Provider) => void;
}

// ProviderCard renders one provider tile in the grouped grid (PAR-UI-007). Its
// root className includes the literal `card-elev` marker the providers e2e spec
// keys on (plan §1.3).
function ProviderCard({ provider, onSelect, onOAuth }: ProviderCardProps) {
  const isActive = provider.status === "active";
  const supportsOAuth = provider.auth_types.includes("oauth");

  return (
    <Card
      data-testid={`provider-card-${provider.id}`}
      role="button"
      tabIndex={0}
      onClick={() => onSelect?.(provider)}
      onKeyDown={(event) => {
        if (event.key === "Enter" || event.key === " ") {
          event.preventDefault();
          onSelect?.(provider);
        }
      }}
      className={cn(
        "card-elev flex cursor-pointer flex-col gap-3 transition-shadow hover:shadow-md"
      )}
    >
      <div className="flex items-start gap-3">
        <ProviderIcon slug={provider.id} name={provider.display_name} size="md" />
        <div className="min-w-0 flex-1">
          <div className="flex items-center justify-between gap-2">
            <span className="truncate font-semibold text-foreground">
              {provider.display_name}
            </span>
            <Badge variant={isActive ? "success" : "neutral"} dot>
              {provider.status}
            </Badge>
          </div>
          {provider.description ? (
            <p className="truncate text-xs text-muted-foreground">
              {provider.description}
            </p>
          ) : null}
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-1.5">
        {provider.auth_types.map((auth) => (
          <Badge key={auth} variant="default" size="sm">
            {auth}
          </Badge>
        ))}
      </div>

      <div className="mt-auto flex items-center justify-between gap-2">
        <span className="text-xs text-muted-foreground">
          {provider.connection_count} connection
          {provider.connection_count === 1 ? "" : "s"}
        </span>
        {supportsOAuth && onOAuth ? (
          <Button
            data-testid="provider-oauth-action"
            variant="outline"
            size="sm"
            onClick={(event) => {
              event.stopPropagation();
              onOAuth(provider);
            }}
          >
            Connect OAuth
          </Button>
        ) : null}
      </div>
    </Card>
  );
}

export { ProviderCard };
