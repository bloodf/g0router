import type { ReactNode } from "react";
import { Card } from "@/components/ui/card";
import { Icon } from "./Icon";
import { cn } from "@/lib/utils";

export function MetricCard({
  label,
  value,
  icon,
  delta,
  pulse,
  accent,
  hint,
}: {
  label: string;
  value: ReactNode;
  icon?: string;
  delta?: { value: string; direction: "up" | "down" };
  pulse?: boolean;
  accent?: "brand" | "success" | "warning" | "danger" | "info";
  hint?: string;
}) {
  const accentColor: Record<string, string> = {
    brand: "text-brand-500 bg-brand-500/10",
    success: "text-success bg-success/10",
    warning: "text-warning bg-warning/10",
    danger: "text-destructive bg-destructive/10",
    info: "text-info bg-info/10",
  };
  return (
    <Card className="p-5 card-elev border-border">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="text-xs uppercase tracking-wider text-text-muted font-medium">
            {label}
          </div>
          <div className="mt-2 text-3xl font-semibold tracking-tight flex items-baseline gap-2">
            <span>{value}</span>
            {pulse && <span className="pulse-dot" />}
          </div>
          {(delta || hint) && (
            <div className="mt-1.5 text-xs flex items-center gap-1 text-text-muted">
              {delta && (
                <span
                  className={cn(
                    "inline-flex items-center gap-0.5",
                    delta.direction === "up" ? "text-success" : "text-destructive",
                  )}
                >
                  <Icon
                    name={delta.direction === "up" ? "trending_up" : "trending_down"}
                    size={14}
                  />
                  {delta.value}
                </span>
              )}
              {hint && <span>{hint}</span>}
            </div>
          )}
        </div>
        {icon && (
          <div
            className={cn(
              "p-2.5 rounded-xl flex-shrink-0",
              accentColor[accent ?? "brand"],
            )}
          >
            <Icon name={icon} size={22} />
          </div>
        )}
      </div>
    </Card>
  );
}
