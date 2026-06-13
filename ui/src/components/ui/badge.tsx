import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";

import { cn } from "@/lib/utils";

const badgeVariants = cva(
  "inline-flex items-center gap-1.5 rounded-md border font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2",
  {
    variants: {
      variant: {
        success: "border-transparent bg-emerald-500/15 text-emerald-600",
        error: "border-transparent bg-destructive/15 text-destructive",
        default: "border-transparent bg-secondary text-secondary-foreground",
        neutral: "border-border bg-muted text-muted-foreground",
        primary: "border-transparent bg-primary/15 text-primary",
      },
      size: {
        sm: "px-2 py-0.5 text-xs",
        md: "px-2.5 py-1 text-sm",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "sm",
    },
  }
);

const dotColors: Record<NonNullable<BadgeProps["variant"]>, string> = {
  success: "bg-emerald-500",
  error: "bg-destructive",
  default: "bg-secondary-foreground",
  neutral: "bg-muted-foreground",
  primary: "bg-primary",
};

export interface BadgeProps
  extends React.HTMLAttributes<HTMLSpanElement>,
    VariantProps<typeof badgeVariants> {
  dot?: boolean;
}

function Badge({ className, variant, size, dot, children, ...props }: BadgeProps) {
  return (
    <span className={cn(badgeVariants({ variant, size }), className)} {...props}>
      {dot ? (
        <span
          data-testid="badge-dot"
          className={cn(
            "inline-block size-1.5 rounded-full",
            dotColors[variant ?? "default"]
          )}
        />
      ) : null}
      {children}
    </span>
  );
}

export { Badge, badgeVariants };
