import * as React from "react";
import { ChevronLeft, ChevronRight } from "lucide-react";

import { cn } from "@/lib/utils";

export type PaginationItem = number | "ellipsis";

function range(start: number, end: number): number[] {
  const result: number[] = [];
  for (let i = start; i <= end; i += 1) {
    result.push(i);
  }
  return result;
}

export function paginationRange(
  page: number,
  totalPages: number
): PaginationItem[] {
  const siblingCount = 1;
  // first + last + current + 2*siblings + 2 ellipses
  const totalSlots = siblingCount * 2 + 5;

  if (totalPages <= totalSlots) {
    return range(1, totalPages);
  }

  const leftSibling = Math.max(page - siblingCount, 1);
  const rightSibling = Math.min(page + siblingCount, totalPages);

  const showLeftEllipsis = leftSibling > 2;
  const showRightEllipsis = rightSibling < totalPages - 1;

  const firstPage = 1;
  const lastPage = totalPages;

  if (!showLeftEllipsis && showRightEllipsis) {
    const leftItemCount = 3 + 2 * siblingCount;
    return [...range(1, leftItemCount), "ellipsis", lastPage];
  }

  if (showLeftEllipsis && !showRightEllipsis) {
    const rightItemCount = 3 + 2 * siblingCount;
    return [
      firstPage,
      "ellipsis",
      ...range(totalPages - rightItemCount + 1, totalPages),
    ];
  }

  return [
    firstPage,
    "ellipsis",
    ...range(leftSibling, rightSibling),
    "ellipsis",
    lastPage,
  ];
}

export interface PaginationProps {
  page: number;
  totalPages: number;
  onPageChange: (page: number) => void;
  className?: string;
}

function Pagination({
  page,
  totalPages,
  onPageChange,
  className,
}: PaginationProps) {
  const items = paginationRange(page, totalPages);
  const atFirst = page <= 1;
  const atLast = page >= totalPages;

  return (
    <nav
      aria-label="pagination"
      className={cn("flex items-center gap-1", className)}
    >
      <button
        type="button"
        data-testid="pagination-prev"
        aria-label="Previous page"
        disabled={atFirst}
        onClick={() => onPageChange(page - 1)}
        className="inline-flex size-8 items-center justify-center rounded-md border border-border text-foreground transition-colors hover:bg-muted disabled:pointer-events-none disabled:opacity-50"
      >
        <ChevronLeft className="size-4" />
      </button>

      {items.map((item, index) =>
        item === "ellipsis" ? (
          <span
            key={`ellipsis-${index}`}
            aria-hidden="true"
            className="inline-flex size-8 items-center justify-center text-muted-foreground"
          >
            …
          </span>
        ) : (
          <button
            key={item}
            type="button"
            aria-current={item === page ? "page" : undefined}
            onClick={() => onPageChange(item)}
            className={cn(
              "inline-flex size-8 items-center justify-center rounded-md border text-sm transition-colors",
              item === page
                ? "border-primary bg-primary text-primary-foreground"
                : "border-border text-foreground hover:bg-muted"
            )}
          >
            {item}
          </button>
        )
      )}

      <button
        type="button"
        data-testid="pagination-next"
        aria-label="Next page"
        disabled={atLast}
        onClick={() => onPageChange(page + 1)}
        className="inline-flex size-8 items-center justify-center rounded-md border border-border text-foreground transition-colors hover:bg-muted disabled:pointer-events-none disabled:opacity-50"
      >
        <ChevronRight className="size-4" />
      </button>
    </nav>
  );
}

export { Pagination };
