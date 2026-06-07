import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import type { ReactNode } from "react";

type Variant =
  | "default"
  | "primary"
  | "success"
  | "warning"
  | "danger"
  | "info"
  | "muted";

const variantClass: Record<Variant, string> = {
  default: "bg-surface-2 text-foreground border-transparent",
  primary: "bg-brand-500/10 text-brand-600 dark:text-brand-300 border-transparent",
  success: "bg-success/10 text-success border-transparent",
  warning: "bg-warning/10 text-warning border-transparent",
  danger: "bg-destructive/10 text-destructive border-transparent",
  info: "bg-info/10 text-info border-transparent",
  muted: "bg-surface-2 text-text-muted border-transparent",
};

export function StatusBadge({
  variant = "default",
  children,
  dot,
  className,
}: {
  variant?: Variant;
  children: ReactNode;
  dot?: boolean;
  className?: string;
}) {
  const dotColor: Record<Variant, string> = {
    default: "bg-text-muted",
    primary: "bg-brand-500",
    success: "bg-success",
    warning: "bg-warning",
    danger: "bg-destructive",
    info: "bg-info",
    muted: "bg-text-muted",
  };
  return (
    <Badge
      variant="outline"
      className={cn(
        "rounded-full px-2.5 py-0.5 font-medium text-[11px] gap-1.5",
        variantClass[variant],
        className,
      )}
    >
      {dot && (
        <span
          className={cn("w-1.5 h-1.5 rounded-full", dotColor[variant])}
        />
      )}
      {children}
    </Badge>
  );
}
