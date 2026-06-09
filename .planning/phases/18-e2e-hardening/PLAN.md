# Phase 18: E2E Hardening

**Phase:** 18  
**Goal:** Build the mocked API layer and achieve full Playwright coverage.  
**Requirements:** TEST-05..07  
**Estimated duration:** 5–6 days  
**Wave:** 6 — Hardening + Ship

---

## Why

The user's objective is to test every screen, button, form, input, modal, and dialog via a mocked API layer that is a 1:1 copy of the real backend.

---

## Scope

### In scope
- `ui/e2e/mocks/` implements the full `/api/*` and `/v1/*` contract.
- Shared TypeScript types between mock layer and real frontend API client.
- Playwright tests for every dashboard route and interaction.
- Mock data seeding helpers.

### Out of scope
- Visual regression tests (future).

---

## Verification

### Tests
1. Every dashboard route has at least one Playwright test.
2. Every form has create + edit + validation tests.
3. Every table has sort/filter/pagination tests where applicable.
4. Every modal/dialog has open + close + submit tests.
5. `npx playwright test` passes with zero failures.

### Manual verification
1. Run `npx playwright test --ui` and spot-check critical flows.

---

## Tasks

1. Extend mock handlers to cover full API surface.
2. Add shared TypeScript contract types.
3. Write Playwright tests for all routes.
4. Add mock data seeding fixtures.
5. Run and fix E2E failures.
6. Verify gates.

---

## Risks

| Risk | Mitigation |
|------|------------|
| Mock layer drifts from real API | Auto-generate types from Go structs or validate mock handlers against admin test fixtures. |
| Flaky E2E tests | Use explicit locators and wait states; avoid timing-dependent assertions. |
