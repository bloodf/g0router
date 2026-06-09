# Phase 7: Dashboard Shell + Providers UI

**Phase:** 07  
**Goal:** Port the dashboard shell and providers/connections pages from 9router's WebUI.  
**Requirements:** UI-01..05  
**Estimated duration:** 4–5 days  
**Wave:** 2 — Core Providers + Admin

---

## Why

Users need a visual control plane. This phase brings the first working dashboard pages so administrators can manage providers without using curl.

---

## Scope

### In scope
- `ui/src/` rebuilt with Vite + React 19 + Tailwind 4 + shadcn/ui:
  - `routes/login.tsx`
  - `components/layout/` (Header, Sidebar, Shell)
  - `routes/_app.dashboard.tsx`
  - `routes/_app.providers.index.tsx`
  - `routes/_app.providers.$id.tsx`
  - `routes/_app.connections.tsx`
- TanStack Query hooks for `/api/login`, `/api/providers`, `/api/connections`, `/api/settings`.
- g0router branding and color palette applied.

### Out of scope
- Models, aliases, combos pages (Phase 9).
- Usage, logs pages (Phase 10).
- Advanced features like MCP, proxy pools, nodes (Phases 14–15).

---

## Verification

### Tests
1. E2E: login flow works with mock API.
2. E2E: providers list page loads and shows seeded providers.
3. E2E: provider detail page shows connections and models.
4. E2E: connections page supports CRUD.

### Manual verification
1. `npm run dev` and navigate through providers/connections pages.

---

## Tasks

1. Create dashboard shell components.
2. Implement login page.
3. Implement providers list page.
4. Implement provider detail page.
5. Implement connections page.
6. Apply g0router branding.
7. Write Playwright E2E tests against mock API.
8. Verify gates.

---

## Risks

| Risk | Mitigation |
|------|------------|
| UI build errors from shadcn v4 + React 19 | Follow existing project toolchain; don't upgrade dependencies. |
| Mock API drift | Share TypeScript types between mock layer and real API client. |
