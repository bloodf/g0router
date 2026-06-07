import {
  flexRender,
  getCoreRowModel,
  getFilteredRowModel,
  getSortedRowModel,
  useReactTable,
  type ColumnDef,
  type SortingState,
} from "@tanstack/react-table";
import { useState } from "react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Icon } from "./Icon";
import { EmptyState } from "./EmptyState";
import { useVisibleWindow } from "@/lib/hooks/useVisibleWindow";

interface DataTableProps<T> {
  columns: ColumnDef<T, any>[];
  data: T[];
  searchPlaceholder?: string;
  searchColumn?: string;
  /** Number of rows initially rendered; more are revealed as the user scrolls. */
  initialVisibleRows?: number;
  emptyTitle?: string;
  emptyDescription?: string;
  emptyIcon?: string;
  toolbar?: React.ReactNode;
}

export function DataTable<T>({
  columns,
  data,
  searchPlaceholder = "Search…",
  initialVisibleRows = 25,
  emptyTitle = "No records",
  emptyDescription,
  emptyIcon,
  toolbar,
}: DataTableProps<T>) {
  const [sorting, setSorting] = useState<SortingState>([]);
  const [globalFilter, setGlobalFilter] = useState("");

  const table = useReactTable({
    data,
    columns,
    state: { sorting, globalFilter },
    onSortingChange: setSorting,
    onGlobalFilterChange: setGlobalFilter,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    getFilteredRowModel: getFilteredRowModel(),
    globalFilterFn: (row, _col, filterValue) => {
      const v = String(filterValue ?? "").toLowerCase();
      if (!v) return true;
      return Object.values(row.original as Record<string, any>)
        .some((x) => String(x ?? "").toLowerCase().includes(v));
    },
  });

  const filteredRows = table.getRowModel().rows;
  const { visible, hasMore, sentinelRef, loadMore } = useVisibleWindow(
    initialVisibleRows,
    filteredRows.length,
  );
  const renderedRows = filteredRows.slice(0, visible);
  const hasData = filteredRows.length > 0;

  return (
    <div className="space-y-3">
      <div className="flex items-center gap-2 flex-wrap">
        <div className="relative flex-1 min-w-[200px] max-w-md">
          <Icon
            name="search"
            size={16}
            className="absolute left-2.5 top-1/2 -translate-y-1/2 text-text-muted"
          />
          <Input
            value={globalFilter}
            onChange={(e) => setGlobalFilter(e.target.value)}
            placeholder={searchPlaceholder}
            className="pl-9 h-9"
          />
        </div>
        {toolbar}
      </div>

      <div className="rounded-xl border border-border bg-surface overflow-hidden">
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((hg) => (
              <TableRow key={hg.id} className="bg-surface-2 hover:bg-surface-2">
                {hg.headers.map((h) => (
                  <TableHead
                    key={h.id}
                    className="text-[11px] uppercase tracking-wider text-text-muted font-medium"
                  >
                    {h.isPlaceholder ? null : (
                      <button
                        className="inline-flex items-center gap-1 hover:text-foreground"
                        onClick={h.column.getToggleSortingHandler()}
                      >
                        {flexRender(h.column.columnDef.header, h.getContext())}
                        {h.column.getIsSorted() === "asc" && (
                          <Icon name="arrow_upward" size={12} />
                        )}
                        {h.column.getIsSorted() === "desc" && (
                          <Icon name="arrow_downward" size={12} />
                        )}
                      </button>
                    )}
                  </TableHead>
                ))}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {hasData ? (
              renderedRows.map((row) => (
                <TableRow key={row.id} className="hover:bg-surface-2/50">
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id} className="text-sm">
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell colSpan={columns.length} className="p-0">
                  <EmptyState
                    icon={emptyIcon ?? "inbox"}
                    title={emptyTitle}
                    description={emptyDescription}
                  />
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>

      {hasData && (
        <div className="flex items-center justify-between text-xs text-text-muted">
          <div>
            Showing {renderedRows.length} of {filteredRows.length} record(s)
          </div>
          {hasMore ? (
            <div ref={sentinelRef} className="flex items-center gap-2">
              <Button variant="outline" size="sm" onClick={loadMore}>
                Load more
              </Button>
            </div>
          ) : (
            <div className="text-text-muted/70">End of list</div>
          )}
        </div>
      )}
    </div>
  );
}
