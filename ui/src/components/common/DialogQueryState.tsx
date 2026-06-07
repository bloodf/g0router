import type { ReactNode } from "react";
import { QueryState, FormSkeleton } from "./Skeletons";

/**
 * Shared wrapper for modal/drawer bodies. Renders a skeleton while any
 * dependency query is loading, a consistent error+retry UI on failure,
 * and the children otherwise. Use inside `<DialogContent>` / `<SheetContent>`
 * so every dialog handles loading and errors the same way.
 *
 * Pass any react-query result(s) via `queries` — the wrapper aggregates
 * their `isLoading` / `isError` flags and exposes a single retry handler.
 */
interface QueryLike {
  isLoading: boolean;
  isError: boolean;
  error?: unknown;
  refetch: () => unknown;
}

export function DialogQueryState({
  queries = [],
  skeleton,
  errorTitle = "Couldn’t load this content",
  errorDescription,
  children,
}: {
  queries?: QueryLike[];
  skeleton?: ReactNode;
  errorTitle?: string;
  errorDescription?: string;
  children: ReactNode;
}) {
  const isLoading = queries.some((q) => q.isLoading);
  const isError = queries.some((q) => q.isError);
  const firstError = queries.find((q) => q.isError)?.error;
  const retry = () => queries.forEach((q) => q.refetch());

  return (
    <QueryState
      isLoading={isLoading}
      isError={isError}
      error={firstError}
      onRetry={retry}
      compactError
      errorTitle={errorTitle}
      errorDescription={errorDescription}
      skeleton={skeleton ?? <FormSkeleton fields={4} />}
    >
      {children}
    </QueryState>
  );
}
