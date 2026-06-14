import * as React from "react";
import { Badge } from "@/components/ui/badge";

export interface McpClientCardProps {
  name: string;
  transport: string;
  healthStatus: string;
  isActive: boolean;
  testId?: string;
  actions?: React.ReactNode;
}

// McpClientCard (PAR-UI-130 /mcp) renders one MCP client/instance row: Name,
// transport Badge, health-status Badge, active state, and optional action slot.
// Reads the PascalCase mcp DTO casing via plain props (§1.2/§1.4).
function McpClientCard({
  name,
  transport,
  healthStatus,
  isActive,
  testId,
  actions,
}: McpClientCardProps) {
  return (
    <div
      data-testid={testId}
      className="flex items-center justify-between rounded-lg border border-border px-4 py-3"
    >
      <div className="flex items-center gap-3">
        <Badge variant="neutral" size="sm">
          {transport}
        </Badge>
        <div>
          <p className="text-sm font-medium text-foreground">{name}</p>
          <p className="text-xs text-muted-foreground">
            {isActive ? "active" : "inactive"}
          </p>
        </div>
      </div>
      <div className="flex items-center gap-2">
        <Badge
          variant={healthStatus === "healthy" ? "success" : "neutral"}
          size="sm"
          dot
        >
          {healthStatus}
        </Badge>
        {actions}
      </div>
    </div>
  );
}

export { McpClientCard };
