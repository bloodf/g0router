# Phase 9: Models + Aliases + Combos

**Phase:** 09  
**Goal:** Build catalog-backed model management with aliases and combos.  
**Requirements:** CATALOG-05..07, MGMT-06..07, UI-06..07  
**Estimated duration:** 4–5 days  
**Wave:** 3 — Catalog + Usage

---

## Why

Aliases and combos let users create friendly names and resilient fallback chains without memorizing provider-specific model IDs.

---

## Scope

### In scope
- `internal/admin/models.go` — `/api/models/aliases` CRUD + `/api/models/disabled`.
- `internal/admin/combos.go` — `/api/combos` CRUD.
- `internal/catalog/` extensions:
  - Custom pricing overrides.
  - Disabled model tracking.
- Dashboard pages:
  - `routes/_app.models.tsx`
  - `routes/_app.models.aliases.tsx`
  - `routes/_app.combos.tsx`

### Out of scope
- Provider-specific model metadata editing beyond aliases/pricing.

---

## Verification

### Tests
1. Alias CRUD persists and resolves in catalog lookups.
2. Custom pricing overrides apply in cost calculation.
3. Disabled models are excluded from `/v1/models`.
4. Combo CRUD defines ordered fallback chains.
5. Combos appear in `/v1/models` with combo IDs.

### Manual verification
1. Create an alias `my-gpt-4o` → `openai/gpt-4o` and use it in a chat request.
2. Create a combo and observe fallback behavior.

---

## Tasks

1. Extend catalog for aliases, overrides, and disabled models.
2. Implement models admin handlers.
3. Implement combos admin handlers.
4. Implement dashboard pages.
5. Write tests and E2E coverage.
6. Verify gates.

---

## Risks

| Risk | Mitigation |
|------|------------|
| Alias loops | Validate aliases do not form cycles. |
| Combo fallback ordering is unclear | Dashboard shows explicit numbered fallback order. |
