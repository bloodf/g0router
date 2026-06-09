# Phase 15: Proxy Pools + Provider Nodes

**Phase:** 15  
**Goal:** Implement proxy rotation and remote provider node registration.  
**Requirements:** MGMT-10..11, UI-12..13, PLAT-12..13  
**Estimated duration:** 5–6 days  
**Wave:** 5 — 9router Features

---

## Why

Proxy pools improve egress resilience and quota distribution. Provider nodes allow running the gateway in a distributed topology.

---

## Scope

### In scope
- `internal/platform/proxypool.go` — proxy pool rotation and health tests.
- `internal/platform/nodes.go` — remote node registration and heartbeat.
- `internal/admin/proxy_pools.go` — `/api/proxy-pools` CRUD + batch + test.
- `internal/admin/nodes.go` — `/api/nodes` CRUD + heartbeat.
- Cloudflare Workers deployer integration (#1360).
- Deno Deploy relay integration (#1437).
- Dashboard pages:
  - `routes/_app.proxy-pools.tsx`
  - `routes/_app.nodes.tsx`

### Out of scope
- Full clustering with consensus (future).

---

## Verification

### Tests
1. Proxy pool rotates through configured proxies.
2. Proxy health test marks unhealthy proxies.
3. Node registration accepts heartbeat and marks node online/offline.
4. Requests can route through a registered remote node.
5. Cloudflare/Deno deployment helpers generate valid config.

### Manual verification
1. Configure a proxy pool and observe rotation.
2. Register a remote node and route a request through it.

---

## Tasks

1. Implement proxy pool data model and rotation.
2. Implement proxy health tests.
3. Implement node registration and heartbeat.
4. Implement deployment helpers.
5. Implement admin handlers.
6. Implement dashboard pages.
7. Write tests and E2E coverage.
8. Verify gates.

---

## Risks

| Risk | Mitigation |
|------|------------|
| Proxy failures cascade | Mark proxies unhealthy quickly; use circuit breaker pattern. |
| Node security | Require shared secret for node registration; validate heartbeats. |
