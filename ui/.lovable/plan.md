## Goal

Two passes:
1. **Audit pass** — walk every route + shared component, log issues (broken links, missing empty/loading/error states, mock-data mismatches, inconsistent layouts, dead code) in a single `AUDIT.md` deliverable.
2. **Pagination pass** — replace the current Prev/Next page controls with **URL-driven infinite scroll** for every table in the app.

## Pagination design

- **Pattern**: infinite scroll. Initial window of N rows; when sentinel enters viewport → window grows by N.
- **URL state**: a single `?visible=<n>` search param per route (TanStack Router `validateSearch` + `zodValidator` + `fallback`). Refreshing / sharing preserves scroll depth.
- **Where state lives**: inside `DataTable` (and `CrudPage` which wraps it), so every table page inherits the behavior automatically. Pages that already define their own `validateSearch` get a merged `visible` field.
- **Mechanism**: IntersectionObserver sentinel row at the bottom. Filter/sort still happen client-side over the full dataset (these are mock-backed pages); only render slicing is windowed. Resetting filter resets `visible` to the page-size default.
- **Search param helper**: new `src/lib/hooks/useVisibleWindow.ts` that reads `visible` via `useSearch({ strict: false })` and writes via `useNavigate` with `replace: true` so back-button isn't polluted on every scroll tick.

## Affected tables

Shared:
- `src/components/common/DataTable.tsx` — primary surface, swap pagination block for sentinel + window.
- `src/components/common/CrudPage.tsx` — already delegates to DataTable, gets pagination for free.
- `src/components/quota/QuotaTable.tsx` — custom; add the same hook + sentinel.

Routes with bespoke tables to convert:
- `_app.audit.tsx`, `_app.endpoint.tsx` (API keys table), `_app.providers.$id.tsx`, `_app.dashboard.tsx` recent-events strip.

CrudPage-backed routes (auto): pricing, aliases, virtual-keys, keys, routing-rules, teams.
DataTable-backed routes (auto): models, logs.

## Audit deliverable

Written to `/mnt/documents/AUDIT.md` (so the user can download), grouped by route, each entry one line: `severity · area · finding · suggested fix`. Severity = blocker / high / medium / low. Cross-cuts (design tokens, deep links, mock router gaps) get their own section.

Audit covers, for every route in `src/routes/_app.*.tsx`:
- Route loads without console errors and matches the 9router reference for layout/density.
- Empty / loading / error states present.
- Deep links (provider id, tunnel hash) resolve.
- Toolbar actions wired to mock router and emit toasts + audit entries where relevant.
- Tokens used (no raw hex/tailwind color literals).
- Tables: have a stable `id` key, sortable headers, and now infinite scroll.

## Implementation order

1. Add `useVisibleWindow` hook + extend `DataTable` and `QuotaTable` with sentinel windowing; drop the Prev/Next footer.
2. Convert bespoke tables (`audit`, `endpoint`, `providers.$id`, `dashboard`) to the same hook.
3. Walk every `_app.*.tsx` route, fix anything cheap inline (missing skeletons, broken tokens, dead imports), record everything else in `AUDIT.md`.
4. Smoke-check the preview for blank pages / runtime errors after the table refactor.

## Out of scope

- No backend changes (mock router stays the source of truth).
- No new pages, no redesigns beyond what the audit flags as blockers.
- Server-side pagination — all current data is mock/in-memory, so windowing happens on the client.
