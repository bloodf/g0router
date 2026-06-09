# Phase 10: Usage + Logs

**Phase:** 10  
**Goal:** Implement request logging, cost calculation, and usage/log dashboards.  
**Requirements:** MGMT-09, UI-10..11, REL-01  
**Estimated duration:** 4–5 days  
**Wave:** 3 — Catalog + Usage

---

## Why

Operators need visibility into traffic, costs, and failures. This phase turns the gateway into an observable system.

---

## Scope

### In scope
- `internal/logging/requestlog.go` — write request log after every inference call.
- `internal/admin/usage.go` — `/api/usage` aggregation.
- `internal/admin/logs.go` — `/api/logs` query with filters.
- Cost calculation wired into request log.
- Dashboard pages:
  - `routes/_app.usage.tsx`
  - `routes/_app.logs.tsx`

### Out of scope
- Real-time metrics (future).
- Alerting (future).

---

## Verification

### Tests
1. Every chat completion request writes a log row.
2. Usage aggregation returns correct totals by provider/model/time.
3. Log filters work by provider, status, date range, and model.
4. Cost is nonzero for known models and zero for cache hits (if applicable).

### Manual verification
1. Send a few requests and check usage page.
2. Apply filters on logs page.

---

## Tasks

1. Implement request log schema and writer.
2. Implement usage aggregation queries.
3. Implement logs query with filters.
4. Implement dashboard pages.
5. Write tests and E2E coverage.
6. Verify gates.

---

## Risks

| Risk | Mitigation |
|------|------------|
| Log table grows unbounded | Add retention policy and pagination from day one. |
| Cost calculation wrong | Validate against known fixture costs; warn when pricing missing. |
