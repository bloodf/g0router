import { Skeleton } from "@/components/ui/skeleton";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Icon } from "./Icon";
import { cn } from "@/lib/utils";
import type { ReactNode } from "react";

/**
 * Shared skeleton primitives. Use these instead of bespoke
 * `Array.from(...).map(...).Skeleton` blocks so every page
 * loads with consistent placeholders.
 */

export function CardSkeleton({
  className,
  lines = 2,
}: {
  className?: string;
  lines?: number;
}) {
  return (
    <Card className={cn("card-elev border-border p-4 space-y-2.5", className)}>
      <Skeleton className="h-4 w-1/3" />
      {Array.from({ length: lines }).map((_, i) => (
        <Skeleton key={i} className="h-3 w-full" />
      ))}
    </Card>
  );
}

export function MetricsGridSkeleton({
  count = 4,
  className,
  height = "h-24",
}: {
  count?: number;
  className?: string;
  height?: string;
}) {
  return (
    <div
      className={cn(
        "grid grid-cols-2 lg:grid-cols-4 gap-4 mb-4",
        className,
      )}
    >
      {Array.from({ length: count }).map((_, i) => (
        <Skeleton key={i} className={height} />
      ))}
    </div>
  );
}

export function CardsGridSkeleton({
  count = 6,
  height = "h-36",
  className,
}: {
  count?: number;
  height?: string;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "grid sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3",
        className,
      )}
    >
      {Array.from({ length: count }).map((_, i) => (
        <Skeleton key={i} className={height} />
      ))}
    </div>
  );
}

export function StackedListSkeleton({
  count = 4,
  height = "h-32",
}: {
  count?: number;
  height?: string;
}) {
  return (
    <div className="space-y-3">
      {Array.from({ length: count }).map((_, i) => (
        <Skeleton key={i} className={height} />
      ))}
    </div>
  );
}

export function TableSkeleton({
  rows = 6,
  columns = 5,
  showHeader = true,
}: {
  rows?: number;
  columns?: number;
  showHeader?: boolean;
}) {
  const gridStyle = {
    gridTemplateColumns: `repeat(${columns}, minmax(0, 1fr))`,
  };
  return (
    <div className="rounded-xl border border-border bg-surface overflow-hidden">
      {showHeader && (
        <div
          className="grid gap-3 p-3 border-b border-border bg-surface-2"
          style={gridStyle}
        >
          {Array.from({ length: columns }).map((_, i) => (
            <Skeleton key={i} className="h-3" />
          ))}
        </div>
      )}
      <div className="divide-y divide-border">
        {Array.from({ length: rows }).map((_, r) => (
          <div key={r} className="grid gap-3 p-3" style={gridStyle}>
            {Array.from({ length: columns }).map((_, c) => (
              <Skeleton key={c} className="h-4" />
            ))}
          </div>
        ))}
      </div>
    </div>
  );
}

export function ChartSkeleton({
  height = 280,
  className,
}: {
  height?: number;
  className?: string;
}) {
  return (
    <Skeleton
      className={cn("w-full", className)}
      style={{ height }}
    />
  );
}

export function FormSkeleton({ fields = 4 }: { fields?: number }) {
  return (
    <div className="space-y-3">
      {Array.from({ length: fields }).map((_, i) => (
        <div key={i} className="space-y-1.5">
          <Skeleton className="h-3 w-24" />
          <Skeleton className="h-9 w-full" />
        </div>
      ))}
      <div className="flex justify-end gap-2 pt-2">
        <Skeleton className="h-9 w-20" />
        <Skeleton className="h-9 w-20" />
      </div>
    </div>
  );
}

export function ListRowsSkeleton({ rows = 5 }: { rows?: number }) {
  return (
    <div className="space-y-2">
      {Array.from({ length: rows }).map((_, i) => (
        <Skeleton key={i} className="h-5 w-full" />
      ))}
    </div>
  );
}

export function DetailHeaderSkeleton() {
  return (
    <div className="space-y-3 mb-6">
      <Skeleton className="h-4 w-40" />
      <div className="flex items-start gap-4">
        <Skeleton className="h-14 w-14 rounded-2xl" />
        <div className="flex-1 space-y-2">
          <Skeleton className="h-6 w-64" />
          <Skeleton className="h-4 w-96" />
        </div>
      </div>
    </div>
  );
}

export function PageHeaderSkeleton() {
  return (
    <div className="mb-5 space-y-2">
      <div className="flex items-center gap-2">
        <Skeleton className="h-8 w-8 rounded-lg" />
        <Skeleton className="h-6 w-48" />
      </div>
      <Skeleton className="h-4 w-80" />
    </div>
  );
}

/**
 * Generic route-transition placeholder. Rendered by the router's
 * defaultPendingComponent so any navigation that suspends shows a
 * consistent shell instead of blank space.
 */
export function RouteTransitionSkeleton() {
  return (
    <div className="p-4 md:p-6 max-w-[1600px] mx-auto space-y-4">
      <PageHeaderSkeleton />
      <MetricsGridSkeleton />
      <TableSkeleton rows={6} columns={5} />
    </div>
  );
}

/**
 * Consistent error placeholder. Use whenever a fetch fails so the user
 * gets a clear retry path instead of a blank screen.
 */
export function ErrorState({
  title = "Couldn’t load data",
  description,
  error,
  onRetry,
  compact,
  className,
}: {
  title?: string;
  description?: string;
  error?: unknown;
  onRetry?: () => void;
  compact?: boolean;
  className?: string;
}) {
  const message =
    description ??
    (error instanceof Error
      ? error.message
      : typeof error === "string"
        ? error
        : "Something went wrong while fetching this content.");
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center text-center rounded-xl border border-dashed border-destructive/40 bg-destructive/5",
        compact ? "py-6 px-4" : "py-12 px-6",
        className,
      )}
    >
      <div
        className={cn(
          "rounded-2xl bg-destructive/10 flex items-center justify-center mb-3",
          compact ? "w-10 h-10" : "w-14 h-14",
        )}
      >
        <Icon
          name="error"
          size={compact ? 20 : 28}
          className="text-destructive"
        />
      </div>
      <h3 className={cn("font-semibold", compact ? "text-sm" : "text-base")}>
        {title}
      </h3>
      <p
        className={cn(
          "mt-1 text-text-muted max-w-md",
          compact ? "text-xs" : "text-sm",
        )}
      >
        {message}
      </p>
      {onRetry && (
        <Button
          variant="outline"
          size={compact ? "sm" : "default"}
          onClick={onRetry}
          className="mt-4"
        >
          <Icon name="refresh" size={14} className="mr-1.5" />
          Try again
        </Button>
      )}
    </div>
  );
}

/**
 * Renders loading skeleton, error state, or children depending on query
 * status. Use to standardize the loading/error/content trio.
 */
export function QueryState({
  isLoading,
  isError,
  error,
  onRetry,
  skeleton,
  errorTitle,
  errorDescription,
  compactError,
  children,
}: {
  isLoading?: boolean;
  isError?: boolean;
  error?: unknown;
  onRetry?: () => void;
  skeleton: ReactNode;
  errorTitle?: string;
  errorDescription?: string;
  compactError?: boolean;
  children: ReactNode;
}) {
  if (isLoading) return <>{skeleton}</>;
  if (isError)
    return (
      <ErrorState
        title={errorTitle}
        description={errorDescription}
        error={error}
        onRetry={onRetry}
        compact={compactError}
      />
    );
  return <>{children}</>;
}
