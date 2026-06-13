import * as React from "react";

import { cn } from "@/lib/utils";

const spinnerSizes = {
  sm: "size-4",
  md: "size-6",
  lg: "size-8",
} as const;

export interface SpinnerProps
  extends React.SVGAttributes<SVGSVGElement> {
  size?: keyof typeof spinnerSizes;
}

function Spinner({ className, size = "md", ...props }: SpinnerProps) {
  return (
    <svg
      role="status"
      aria-label="Loading"
      viewBox="0 0 24 24"
      fill="none"
      className={cn("animate-spin text-muted-foreground", spinnerSizes[size], className)}
      {...props}
    >
      <circle
        className="opacity-25"
        cx="12"
        cy="12"
        r="10"
        stroke="currentColor"
        strokeWidth="4"
      />
      <path
        className="opacity-75"
        fill="currentColor"
        d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z"
      />
    </svg>
  );
}

export interface LoadingProps {
  message?: string;
  size?: keyof typeof spinnerSizes;
  className?: string;
}

function Loading({ message, size = "md", className }: LoadingProps) {
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center gap-2 p-6",
        className
      )}
    >
      <Spinner size={size} />
      {message ? (
        <p className="text-sm text-muted-foreground">{message}</p>
      ) : null}
    </div>
  );
}

export { Spinner, Loading };
