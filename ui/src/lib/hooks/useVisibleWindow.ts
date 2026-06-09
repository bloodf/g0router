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
  const [visible, setVisibleState] = useState<number>(() => {
    if (typeof window === "undefined") return pageSize;
    const v = Number(new URLSearchParams(window.location.search).get("visible"));
    if (!Number.isFinite(v) || v <= 0) return pageSize;
    return Math.max(v, pageSize);
  });

  const sentinelRef = useRef<HTMLDivElement | null>(null);

  // Clamp during render instead of copying into state in an effect.
  const clampedVisible = Math.min(visible, totalRows || pageSize);

  // Wrap state updates so URL stays in sync without an effect.
  const setVisible = useCallback(
    (next: number | ((prev: number) => number)) => {
      setVisibleState((prev) => {
        const resolved =
          typeof next === "function"
            ? (next as (prev: number) => number)(prev)
            : next;
        if (typeof window === "undefined") return resolved;
        const url = new URL(window.location.href);
        if (resolved <= pageSize) url.searchParams.delete("visible");
        else url.searchParams.set("visible", String(resolved));
        window.history.replaceState({}, "", url.toString());
        return resolved;
      });
    },
    [pageSize],
  );

  const loadMore = useCallback(() => {
    setVisible((v) => Math.min(v + pageSize, Math.max(totalRows, pageSize)));
  }, [pageSize, setVisible, totalRows]);

  const reset = useCallback(() => {
    setVisible(pageSize);
  }, [pageSize, setVisible]);

  // IntersectionObserver: grow window when sentinel enters viewport.
  useEffect(() => {
    const el = sentinelRef.current;
    if (!el || clampedVisible >= totalRows) return;
    const io = new IntersectionObserver(
      (entries) => {
        if (entries.some((e) => e.isIntersecting)) loadMore();
      },
      { rootMargin: "200px 0px" },
    );
    io.observe(el);
    return () => io.disconnect();
  }, [loadMore, clampedVisible, totalRows]);

  return {
    visible: clampedVisible,
    hasMore: clampedVisible < totalRows,
    sentinelRef,
    loadMore,
    reset,
  };
}
