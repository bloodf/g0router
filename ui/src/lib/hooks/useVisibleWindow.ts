import { useCallback, useEffect, useRef, useState } from "react";

/**
 * Windowed pagination state synced to the URL via ?visible=<n>.
 *
 * - Initial render reads `visible` from window.location.search (SSR-safe).
 * - Writes use history.replaceState so we don't pollute browser history on every
 *   intersection-observer tick and don't fight TanStack Router navigation.
 * - When the underlying dataset shrinks (filter applied), the window is
 *   automatically clamped back down to the page size.
 */
export function useVisibleWindow(pageSize: number, totalRows: number) {
  const readInitial = () => {
    if (typeof window === "undefined") return pageSize;
    const v = Number(new URLSearchParams(window.location.search).get("visible"));
    if (!Number.isFinite(v) || v <= 0) return pageSize;
    return Math.min(Math.max(v, pageSize), Math.max(totalRows, pageSize));
  };

  const [visible, setVisible] = useState<number>(readInitial);
  const sentinelRef = useRef<HTMLDivElement | null>(null);

  // Persist to URL (no history entry per scroll).
  useEffect(() => {
    if (typeof window === "undefined") return;
    const url = new URL(window.location.href);
    if (visible <= pageSize) url.searchParams.delete("visible");
    else url.searchParams.set("visible", String(visible));
    window.history.replaceState({}, "", url.toString());
  }, [visible, pageSize]);

  // Clamp when the dataset shrinks (e.g. user typed a filter).
  useEffect(() => {
    if (totalRows < visible) setVisible(Math.max(pageSize, Math.min(visible, totalRows || pageSize)));
  }, [totalRows, pageSize, visible]);

  const loadMore = useCallback(() => {
    setVisible((v) => Math.min(v + pageSize, Math.max(totalRows, pageSize)));
  }, [pageSize, totalRows]);

  // IntersectionObserver: grow window when sentinel enters viewport.
  useEffect(() => {
    const el = sentinelRef.current;
    if (!el || visible >= totalRows) return;
    const io = new IntersectionObserver(
      (entries) => {
        if (entries.some((e) => e.isIntersecting)) loadMore();
      },
      { rootMargin: "200px 0px" },
    );
    io.observe(el);
    return () => io.disconnect();
  }, [loadMore, visible, totalRows]);

  return {
    visible: Math.min(visible, totalRows || pageSize),
    hasMore: visible < totalRows,
    sentinelRef,
    loadMore,
    reset: () => setVisible(pageSize),
  };
}
