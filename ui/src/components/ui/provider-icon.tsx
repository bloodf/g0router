import * as React from "react";

import { cn } from "@/lib/utils";

const PALETTE = [
  "#ef4444",
  "#f97316",
  "#f59e0b",
  "#10b981",
  "#06b6d4",
  "#3b82f6",
  "#6366f1",
  "#8b5cf6",
  "#ec4899",
  "#14b8a6",
];

export function providerInitials(name: string): string {
  return name.slice(0, 2).toUpperCase();
}

export function providerColor(name: string): string {
  let hash = 0;
  for (let i = 0; i < name.length; i += 1) {
    hash = (hash * 31 + name.charCodeAt(i)) >>> 0;
  }
  return PALETTE[hash % PALETTE.length];
}

const sizeStyles = {
  sm: { box: "size-5", text: "text-[8px]" },
  md: { box: "size-8", text: "text-xs" },
  lg: { box: "size-10", text: "text-sm" },
} as const;

export interface ProviderIconProps {
  slug: string;
  name: string;
  size?: keyof typeof sizeStyles;
  className?: string;
}

function ProviderIcon({ slug, name, size = "md", className }: ProviderIconProps) {
  const [errored, setErrored] = React.useState(false);
  const styles = sizeStyles[size];

  if (errored) {
    return (
      <span
        aria-label={name}
        role="img"
        className={cn(
          "inline-flex items-center justify-center rounded-full font-semibold text-white",
          styles.box,
          styles.text,
          className
        )}
        style={{ backgroundColor: providerColor(name) }}
      >
        {providerInitials(name)}
      </span>
    );
  }

  return (
    <img
      src={`/providers/${slug}.png`}
      alt={name}
      onError={() => setErrored(true)}
      className={cn("inline-block rounded-full object-contain", styles.box, className)}
    />
  );
}

export { ProviderIcon };
