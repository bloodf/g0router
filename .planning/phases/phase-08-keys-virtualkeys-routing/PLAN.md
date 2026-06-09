# Phase 8: Keys + Virtual Keys + Routing

**Phase:** 08  
**Goal:** Implement API keys, virtual keys, weighted routing, and automatic fallback chains.  
**Requirements:** GOV-01..05, MGMT-05, MGMT-08, UI-08..09  
**Estimated duration:** 5–6 days  
**Wave:** 2 — Core Providers + Admin

---

## Why

This is the core value proposition: route requests across multiple providers and accounts with resilience and cost control.

---

## Scope

### In scope
- `internal/governance/`:
  - Virtual key CRUD.
  - Provider config validation.
  - Weighted provider selection.
  - Automatic fallback chain generation.
  - Per-key quota tracking.
- `internal/inference/router.go` updated with governance-aware routing.
- `internal/admin/keys.go` — `/api/keys` CRUD + regenerate.
- `internal/admin/routing.go` — `/api/routing-rules` CRUD.
- Dashboard pages:
  - `routes/_app.keys.tsx`
  - `routes/_app.virtual-keys.tsx`
  - `routes/_app.routing-rules.tsx`

### Out of scope
- Advanced adaptive load balancing (future).
- Semantic caching (future).

---

## Verification

### Tests
1. Virtual key CRUD endpoints work.
2. Weighted routing distributes traffic proportionally across fixtures.
3. Fallback chain retries on simulated provider failure.
4. `x-g0-vk` header routes through the correct virtual key.
5. Quota exhaustion skips exhausted keys.

### Manual verification
1. Create two providers and a virtual key with weights.
2. Send requests and observe routing distribution in logs.

---

## Tasks

1. Implement virtual key data model and store.
2. Implement weighted provider selection.
3. Implement fallback chain generation.
4. Update inference router.
5. Implement keys and routing admin handlers.
6. Implement dashboard pages.
7. Write integration and E2E tests.
8. Verify gates.

---

## Risks

| Risk | Mitigation |
|------|------------|
| Weight normalization edge cases | Explicit tests for zero-weight, all-zero, and single-candidate cases. |
| Fallback loops | Track attempt count and fail after exhausting candidates. |
| Quota race conditions | Use atomic counters or SQLite transactions for quota updates. |
