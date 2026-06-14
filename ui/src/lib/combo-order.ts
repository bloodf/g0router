// combo-order.ts — the pure, deterministic reorder helper for the combos member
// list (w6-h §1.3). The combos ComboFormModal's @dnd-kit onDragEnd delegates to
// moveStep; this function is the AUTHORITATIVE reorder correctness proof (unit
// tested in combo-order.test.ts). No DOM, no side effects.

/**
 * Returns a new array with the element at `from` moved to index `to`.
 *
 * - Pure: the input array is never mutated; a new array is always returned.
 * - `from === to`, or an out-of-range `from`/`to`, returns an equivalent copy
 *   with order unchanged.
 * - Untouched elements keep their relative order.
 */
export function moveStep<T>(steps: T[], from: number, to: number): T[] {
  const next = steps.slice();
  if (
    from === to ||
    from < 0 ||
    to < 0 ||
    from >= next.length ||
    to >= next.length
  ) {
    return next;
  }
  const [moved] = next.splice(from, 1);
  next.splice(to, 0, moved);
  return next;
}
