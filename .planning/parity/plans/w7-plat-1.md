# Micro-plan w7-plat-1 — Proxy-pools backend + outbound proxy / SSRF mitigation (Go)

```
wave: 7
plan: w7-plat-1
status: READY (rev 1 — authored against merged Waves 0–6 + the shipped W7
  governance plans (w7-gov-1/2/3, gate-verified), live tree @ <base>;
  WAVE-7-MAP w7-plat-1 row ~line 183; serial chain §219-224; micro-serial
  §234-238 (selection.go); reconciliation §245; freeze rules §267)
runs: platform track. Disjoint domain/store/admin files from w7-plat-2 (tunnels),
  w7-plat-3 (mitm) — run ∥. TAKES the internal/server/routes_admin.go SERIAL SLOT
  in chain order (… → w7-mcp-3 → **w7-plat-1** → w7-plat-2 → w7-plat-3 → w7-misc;
  MAP §219-224). SECONDARY micro-serial: this plan adds an ADDITIVE
  proxy-resolution hook to internal/inference/selection.go; w7-route also edits
  selection.go (weighted selection) — the orchestrator serializes the two
  selection.go edit windows (MAP §234-238 / §267).
branch: main (direct commits, per repo convention)
commit-prefix: phase-1/w7-plat-1:
ref-source: 9router frozen @ 827e5c3 — proxy-pools + outbound-proxy/SSRF surfaces;
  the BINDING contract for W7 is the W6 e2e mock (decision 1: real Go wins, mock
  corrected to mirror it). Mock sources:
    ui/e2e/mocks/handlers/proxy-pools.ts + seed/proxy-pools.ts.
base: <base> = git rev-parse HEAD recorded at P0. Substitute the actual SHA
  everywhere §5 says <base>.
go-serial-slot: this plan holds the SINGLE in-flight edit to
  internal/server/routes_admin.go while live (W3/W4/W5/W6 lesson; MAP decision 3).
  Slot must be FREE at P-check before T-routes; RELEASE to w7-plat-2 on close.
selection-micro-serial: this plan's selection.go edit is ADDITIVE-ONLY (a new
  proxy-resolution helper + an additive call site). It must NOT overlap w7-route's
  selection.go edit window. Orchestrator confirms exactly one unmerged selection.go
  holder before T-proxywire (MAP §267).
new-route: NO UI route files. The /proxy-pools page ALREADY SHIPPED in w6-m
  (PARTIAL) against the mock; this plan builds the REAL Go so the page flips
  PARTIAL→HAVE and corrects the mock body to mirror the Go DTOs.
```

---

## 1. Scope — PAR rows + the two surfaces

### Rows this plan closes

| Row / item | Claim | Target state after w7-plat-1 |
|---|---|---|
| PAR-PLAT-001 | proxy-pool list/read | HAVE (Go — `GET /api/proxy-pools`, `GET /api/proxy-pools/{id}`) |
| PAR-PLAT-002 | proxy-pool create | HAVE (`POST /api/proxy-pools`) |
| PAR-PLAT-003 | proxy-pool update | HAVE (`PUT /api/proxy-pools/{id}`) |
| PAR-PLAT-004 | proxy-pool delete (+ bound-connection guard) | HAVE (`DELETE /api/proxy-pools/{id}`, 409 if a connection references the pool) |
| PAR-PLAT-005 | proxy-pool connectivity test | HAVE (`POST /api/proxy-pools/{id}/test`, real HTTP-via-proxy probe; deterministic in tests) |
| PAR-PLAT-009 | per-connection proxy resolution | HAVE (selection.go additive hook → `NetworkConfig.ProxyURL` injection) |
| PAR-AUTH-020 | outbound proxy + SSRF mitigation | HAVE (`internal/platform/outboundproxy.go` SSRF guard + ClientPool wiring) |
| open-questions w6-m **ESC-1b** (proxy-pools backend absent) | real `/api/proxy-pools*` CRUD + connectivity test | RESOLVED (cite this plan) |
| PAR-UI-019 | proxy-pools page (PARTIAL) | PARTIAL → HAVE |
| PAR-UI-104 | proxy-pool form/edit (PARTIAL) | PARTIAL → HAVE |
| PAR-UI-105 | proxy-pool test/status (PARTIAL) | PARTIAL → HAVE |

Matrix flips at closeout (§4 T-close): in `.planning/parity/matrix/`, PAR-PLAT-001..005,
PAR-PLAT-009, PAR-AUTH-020 → HAVE (real Go); PAR-UI-019/104/105 PARTIAL → HAVE.
Mark `open-questions.md` w6-m ESC-1b RESOLVED with a cite to this plan; append any new
open items (§8).

### 1.1 Preconditions already satisfied by merged waves (evidence)

- **W6-m UI is SHIPPED (PARTIAL) and FROZEN (consume-only, MAP decision 8 / §267).**
  The `/proxy-pools` page renders against the registered mock; the UI type is
  `ProxyPool` (`ui/src/lib/types.ts:207-217`):
  `{id,name,protocol,host,port,username,is_active,last_check_at,last_check_status}`.
  The binding acceptance contract is the existing spec (must stay green at closeout):
  `ui/e2e/proxy-pools.spec.ts`.
- **The outbound HTTP client + proxy plumbing ALREADY EXIST (the big de-risk)**
  — `internal/providers/utils/client.go`:
  - `ClientPool` wraps a `*fasthttp.Client` and ALREADY supports per-proxy clients
    (`ClientPool.proxies map[string]*fasthttp.Client`, `client.go:18`); `Do`
    resolves a proxy via `proxyFunc` and lazily builds a proxied client via
    `clientForProxy(proxyURL)` using `fasthttpproxy.FasthttpHTTPDialer`
    (`client.go:38-81`).
  - **GAP:** `proxyFunc` is `httpproxy.FromEnvironment().ProxyFunc()`
    (`client.go:32`) — it resolves proxies from ENV ONLY, NOT from a per-connection
    `NetworkConfig.ProxyURL`. So per-connection proxy is NOT honored today even
    though the dial machinery exists. This plan ADDS that wiring (§1.6).
- **`NetworkConfig.ProxyURL` exists but is INERT** — `schemas.NetworkConfig{Timeout,
  ProxyURL,MaxRetries}` (`internal/schemas/provider.go:38-42`); `SetNetworkConfig`
  is on the `schemas.Provider` interface (`provider.go:71`) and every provider
  STORES it (`generic/provider.go:44-45`, `openai/provider.go:33-34`) — but the
  stored `ProxyURL` is never read by `ClientPool.Do`. This plan makes the
  ClientPool HONOR a per-instance proxy override (§1.6). The interface + field
  already exist → NO interface change, NO `SetNetworkConfig` signature change.
- **An SSRF / loopback precedent helper exists** — `internal/server/guard.go:232`
  `isLoopbackHostname(h)` + `loopbackHosts` map (an INBOUND host-access guard). It
  is the precedent for an analogous OUTBOUND SSRF check, but it only covers
  loopback hostnames, not private/link-local CIDR ranges; this plan builds a
  PURPOSE-BUILT outbound SSRF evaluator in `internal/platform/outboundproxy.go`
  (§1.7) — it does NOT edit guard.go.
- **Connection→proxy linkage seam exists** — `store.Connection.Metadata string`
  (`connections.go:22`) is a free-form JSON field already plumbed through
  create/list/update (`connections.go:43-94`) and exposed in the connection DTO
  (`admin/connections.go:21,34`). It is the additive linkage candidate (§8
  ESC-CONN-LINK; default = `ensureColumn connections.proxy_pool_id`).
- **Envelope + handler patterns** (`internal/admin/respond.go`): `writeData(ctx,
  status, data)` / `writeError(ctx, status, message)` → `{data,error:{message}}`
  snake_case (`respond.go:19,23`). `pathID(ctx.UserValue("id"))` extracts `{id}`
  (`handlers.go:84`). CRUD template = `internal/admin/virtualkeys.go`
  (List/Create/Get/Update/Delete + DTO + request structs + validate +
  ErrNotFound→404).
- **Store CRUD template** (`internal/store/virtualkeys.go`): `newID()`,
  `time.Now().Unix()` timestamps, `boolToInt` for SQLite bools, `scanX` helper,
  `ErrNotFound` on `sql.ErrNoRows`, JSON-blob config column for nested data.
- **Migrations are additive-only** (`internal/store/migrate.go`): new tables via the
  `tables []struct{name,create}` slice with `CREATE TABLE IF NOT EXISTS`
  (`migrate.go:15-191`); new columns via the `ensureColumn` loop
  (`migrate.go:235-247`); secret-at-rest precedent = the `*_enc` reversible columns
  written/read via `s.cipher.Encrypt/Decrypt` (`connections.go:118-147`).
- **Admin test harness** (`internal/admin/admin_test.go:24` `newTestEnv`): real
  `store.Open(tempDB, secret)` + `auth.NewSessions` + `SeedAdmin("admin","123456")`
  + `New(...)`. NO mocks. `call(...)` drives a handler + decodes the envelope. This
  is the authoritative proof surface.
- **The audit seam shipped in w7-gov-1** — `internal/admin/audit.go:64`
  `func (h *Handlers) recordAudit(ctx, action, target, details string)` (resolves
  the actor from `ctx.UserValue(userKey).(*store.User)`, best-effort, logs on
  failure). REUSE `h.recordAudit` on every proxy-pool mutation (NO audit-write
  retrofit into other files).
- **Handlers injection** — the `Handlers` struct composes `h.store` directly; new
  domains use `h.store` with NO new global state and NO `New(...)` signature change
  (MAP decision 9). `internal/admin/handlers.go` already holds `audit
  *governance.AuditService` constructed in `New` (w7-gov-1).

### 1.2 The mock contract this flip must mirror (binding — decision 1)

**Decision 1 (MAP §36, §245):** real Go wins; the W6 mock body + seed are corrected
IN THIS PLAN to mirror the real Go `{data,error}` snake_case DTO. The page is FROZEN
(decision 8); prefer matching the mock's existing field names in the Go DTO (they were
modeled to match 9router); only ESCALATE if impossible.

**Proxy-pools** (`ui/e2e/mocks/handlers/proxy-pools.ts` + `seed/proxy-pools.ts`):
- Routes the page consumes (`proxy-pools.ts`):
  - `GET /api/proxy-pools` → bare JSON array under `{data}` (`Array.from(store.proxyPools.values())`).
  - `POST /api/proxy-pools` → returns the created pool object (`{id, last_check_at, ...body}`).
  - `POST /api/proxy-pools/batch` → bulk-import (`{items:[...]}` → `{}`).
  - `GET|PUT|DELETE /api/proxy-pools/{id}` (regex `\/api\/proxy-pools\/[^/]+$`) →
    get-or-404 / merge-update / delete→`{}`.
  - `POST /api/proxy-pools/{id}/test` (regex `…\/test$`) → `{ok:true, latency_ms:N}`.
- Mock/seed entry shape = the UI `ProxyPool` type (`types.ts:207-217`):
  `{id,name,protocol,host,port,username,is_active,last_check_at,last_check_status}`
  — **this is the canonical Go DTO** (verified against the brief). `protocol` is a
  string (e.g. `"https"`); `port` is a number; `is_active` is a bool;
  `last_check_status` is a string (`"ok"` in the seed); `last_check_at` is ISO-8601.
- **Mock divergences to reconcile (mock mirrors Go):**
  - **POST shape:** the mock spreads `{id, last_check_at, ...body}` so it echoes
    whatever the body carries plus an auto `last_check_at`. The Go `POST` returns
    the canonical proxyPoolDTO (9 fields) with `is_active` defaulting to `true`,
    `last_check_status` defaulting to `""` (or `"unknown"`), and `last_check_at`
    EMPTY/`""` (a freshly-created pool has not been tested — the mock's
    auto-stamping of `last_check_at` on create is cosmetic and the spec does not
    assert it). **Reconciliation:** correct the mock POST to mirror the Go default
    (no auto `last_check_at` stamp on create; defaults as the Go sets them). Confirm
    the spec does not assert a non-empty `last_check_at` immediately after create;
    if it does → ESC-MOCK.
  - **`/test` response:** the mock returns `{ok:true, latency_ms:N}`. **DECIDE the
    Go `/test` response shape (§8 ESC-TEST-SHAPE).** RECOMMENDED default: Go returns
    `{data:{ok:bool, latency_ms:int, status:string}}` and ALSO persists
    `last_check_status`/`last_check_at` on the pool row; the corrected mock mirrors
    `{ok, latency_ms}` (+ `status` if the page reads it — VERIFY the page's
    consumption before adding fields the page ignores).
  - **`/batch`:** the brief's SCOPE lists the core CRUD + `/test` but does NOT list
    `/batch`. **DECIDE (§8 ESC-BATCH).** RECOMMENDED default: SHIP `POST
    /api/proxy-pools/batch` too (it is in the registered mock and the w6-m page may
    call it on import) — a thin loop over `CreateProxyPool`, returning `{data:{}}`
    or `{data:{created:N}}`. If the page never calls `/batch`, it can be deferred
    to a follow-up; VERIFY the page at T-mocks. Default = ship it (cheap, mirrors
    the mock).
- Mock DELETE returns `{}`; Go returns `{data:{message:"Proxy pool deleted
  successfully"}}` (mirrors `DeleteVirtualKey` shape); the page ignores the body on
  delete. **NEW Go behavior NOT in the mock:** the bound-connection guard — DELETE
  returns **409** if any connection references the pool (§1.5). The mock has no such
  branch; the corrected mock MAY add a 409 branch ONLY if a spec exercises it
  (default: leave the mock delete as `{}` since the spec deletes an unbound pool;
  the 409 is proven by the Go admin test, not the e2e — §8 ESC-409-MOCK).

### 1.3 Architecture (binding — layered DDD, decision 4)

Two surfaces, layered transport → domain → repository:

```
proxy-pools:    admin/proxypools.go  → platform/proxypools.go   → store/proxypools.go (NEW table proxy_pools)
outbound/SSRF:  (consumed by inference) → platform/outboundproxy.go → (no store; pure policy + ClientPool wiring)
per-conn proxy: inference/selection.go (ADDITIVE hook) → resolves a connection's proxy_pool_id
                  → platform/proxypools.go (lookup) → schemas.NetworkConfig.ProxyURL on the provider
                  → utils/client.go ClientPool honors the per-instance ProxyURL (ADDITIVE)
                  → platform/outboundproxy.go SSRF-guards the resolved dial target
```

- **proxy-pools** is a CRUD domain. Per the phase-12B arch test
  (transport→domain→repository enforced), build a `platform.ProxyPoolService` over
  `*store.Store` (the `internal/platform` package exists as a placeholder —
  `doc.go` already names "the proxy pool" as a platform feature). Mirror the
  governance-domain seam (w7-gov-* use `internal/governance/*`); platform is the
  parallel home for these features. If the arch test ALLOWS handler→store directly
  for pure CRUD (as virtualkeys does), the domain wrapper is OPTIONAL — but because
  the connectivity-test + per-connection resolution logic warrants a domain seam
  (it is reused by selection.go, NOT only by the handler), **build
  `platform/proxypools.go` as the domain service** (it holds the connectivity-test
  + the `ResolveProxyForConnection` lookup). Decide the thin-vs-full split at
  T-proxypools against the arch test (§8 ESC-ARCH).
- **outbound/SSRF** is a pure policy package — `platform/outboundproxy.go` holds the
  SSRF evaluator (`IsBlockedTarget(host string) (bool, reason)`) + the ClientPool
  wiring helper. NO store. It is consumed at two points: (a) the connectivity-test
  (so the test itself cannot be abused for SSRF), and (b) the live outbound dial
  path (so a user-configured proxy/target cannot reach internal addresses).
- **per-connection proxy** is wired via an ADDITIVE hook in
  `inference/selection.go`: after a connection is selected, resolve its
  `proxy_pool_id` → the pool's proxy URL → set `NetworkConfig.ProxyURL` on the
  provider instance (via the existing `SetNetworkConfig`). The ClientPool then
  honors that per-instance proxy (§1.6). This is ADDITIVE — no existing selection
  logic changes (micro-serial vs w7-route).

### 1.4 Proxy-pools Go contract (NEW, TDD)

Table `proxy_pools` (additive, `migrate.go` tables slice). **DECIDE typed-vs-JSON
columns (§8 ESC-SCHEMA).** RECOMMENDED default = **typed columns** (the DTO is a
fixed 9-field shape; typed columns enable the `?isActive` filter + the
bound-connection `WHERE proxy_pool_id=?` guard cleanly; the gov plans used typed
columns for fixed-shape domains and a JSON blob only for nested data). If proxy
credentials are stored, the password is `*_enc` at rest (§1.4 secret note):

```sql
CREATE TABLE IF NOT EXISTS proxy_pools (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  protocol TEXT NOT NULL DEFAULT 'http',     -- http|https|socks5
  host TEXT NOT NULL,
  port INTEGER NOT NULL DEFAULT 0,
  username TEXT NOT NULL DEFAULT '',
  password_enc TEXT NOT NULL DEFAULT '',     -- encrypted at rest (s.cipher); NEVER echoed
  is_active INTEGER NOT NULL DEFAULT 1,
  last_check_status TEXT NOT NULL DEFAULT '',
  last_check_at TEXT NOT NULL DEFAULT '',    -- ISO-8601 (RFC3339), mirrors mock
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
)
```

`internal/store/proxypools.go` (NEW): `ProxyPool` struct
`{ID,Name,Protocol,Host,Port,Username,Password,IsActive,LastCheckStatus,LastCheckAt,
CreatedAt,UpdatedAt}` (`Password` plaintext in memory, encrypted at rest via
`s.cipher`, mirroring `connections.go:118-147`) + methods:
- `CreateProxyPool(p)` / `ListProxyPools(filterActive *bool)` / `GetProxyPoolByID(id)`
  / `UpdateProxyPool(p)` / `DeleteProxyPool(id)` (mirror `virtualkeys.go`: `newID()`,
  unix ts, `ErrNotFound`, `scanProxyPool`, `boolToInt`).
- `SetProxyPoolCheck(id, status, atRFC3339)` — updates `last_check_status` +
  `last_check_at` after a connectivity test.
- `CountConnectionsUsingProxyPool(id)` — for the delete guard (`SELECT COUNT(*) …
  WHERE proxy_pool_id=?`; depends on ESC-CONN-LINK landing the linkage column).
- `ListProxyPools` honors the `filterActive` arg for the `?isActive` query.

**Secret note:** if `password` is present it is encrypted at rest (`password_enc`)
and the read DTO MASKS it (`password_set bool`, never the cleartext) — mirrors the
connection DTO `secret_set` pattern (`admin/connections.go:15`). The mock seed has
no password field; the corrected mock seed need not add one. (§8 ESC-PROXY-CRED:
if the operator wants proxy auth, this is the shape; if not, drop password entirely
— RECOMMENDED: include `password` support since proxies commonly require auth, and
the UI `ProxyPool` type carries `username`.)

`internal/admin/proxypools.go` (NEW):

| Handler | Route | Shape (snake_case, `{data}`) | Notes |
|---|---|---|---|
| `ListProxyPools` | `GET /api/proxy-pools[?isActive=true][&includeUsage=true]` | bare array under `{data}` of `proxyPoolDTO` (mirror `proxy-pools.ts` GET). `?isActive` filters; `?includeUsage` adds a `usage_count` field per pool (count of bound connections) IF the page reads it — VERIFY (§8 ESC-USAGE) | `proxyPoolDTO{id,name,protocol,host,port,username,is_active,last_check_status,last_check_at}` (+ `password_set` if creds; NEVER `password`) |
| `CreateProxyPool` | `POST /api/proxy-pools` | body `{name,protocol,host,port,username?,password?,is_active?}`; `is_active` defaults true; returns `{data:proxyPoolDTO}`; 400 on empty name/host or invalid port | drop the mock's cosmetic create-time `last_check_at` stamp |
| `BatchProxyPools` | `POST /api/proxy-pools/batch` | body `{items:[poolReq]}`; loop `CreateProxyPool`; returns `{data:{created:N}}` | §8 ESC-BATCH (default: ship) |
| `GetProxyPool` | `GET /api/proxy-pools/{id}` | `{data:proxyPoolDTO}` or 404 | |
| `UpdateProxyPool` | `PUT /api/proxy-pools/{id}` | body = create body; returns updated `{data:proxyPoolDTO}` or 404 | password unchanged if omitted (mirror connection update) |
| `DeleteProxyPool` | `DELETE /api/proxy-pools/{id}` | `{data:{message:"Proxy pool deleted successfully"}}` or 404; **409 if `CountConnectionsUsingProxyPool(id) > 0`** | bound-connection guard (§1.5) |
| `TestProxyPool` | `POST /api/proxy-pools/{id}/test` | `{data:{ok:bool, latency_ms:int, status:string}}`; persists `last_check_status`/`last_check_at` | connectivity test (§1.6); deterministic in unit tests |

### 1.5 Bound-connection delete guard (NEW)

On `DELETE /api/proxy-pools/{id}`: BEFORE deleting, call
`CountConnectionsUsingProxyPool(id)`; if `> 0`, return **409** with
`{error:{message:"Proxy pool is in use by N connection(s)"}}` and do NOT delete.
This depends on the connection→proxy linkage column (§8 ESC-CONN-LINK). The Go
admin test asserts both branches: delete-unbound→200, delete-bound→409 +
pool-still-exists. The e2e spec only deletes an unbound pool (the mock has no
linkage), so the 409 is an admin-test-only proof (§8 ESC-409-MOCK).

### 1.6 Connectivity test + per-connection proxy wiring (NEW — the deterministic-test core)

**Connectivity test (`platform.ProxyPoolService.TestConnectivity`).** Performs a
real HTTP request THROUGH the proxy to a probe target and reports reachability +
latency. **Determinism (binding — AGENTS.md "No mocks; use interfaces and fakes"):**
the service takes an injectable HTTP-via-proxy seam (e.g. a `ProxyProber` interface
or a `func(proxyURL, target string) (latencyMs int, err error)` field) so tests
inject a deterministic fake (no network); production wires the real `ClientPool`
proxied dial. The probe target is SSRF-guarded BEFORE dialing (§1.7) — a test
target that resolves to a private/loopback/link-local IP is REFUSED with a
deterministic error. Unit tests (in `internal/platform/`) cover:
- proxy reachable → `{ok:true, latency_ms:>0}` + `last_check_status="ok"` persisted.
- proxy unreachable / fake returns error → `{ok:false}` + `last_check_status="error"`.
- probe target is private/loopback/link-local → REFUSED (SSRF), `ok:false`,
  status reflects the block.

**Per-connection proxy wiring (the inert-ProxyURL fix).** Today
`NetworkConfig.ProxyURL` is stored but never honored (§1.1). This plan makes the
ClientPool honor a per-instance proxy override, ADDITIVELY in
`internal/providers/utils/client.go`:
- ADD a per-`ClientPool` proxy override (e.g. a `SetProxyURL(string)` method or a
  `proxyOverride *url.URL` field) so a provider instance configured via
  `SetNetworkConfig{ProxyURL:...}` routes its `Do` through the configured proxy
  (using the EXISTING `clientForProxy` machinery, `client.go:61-81`) INSTEAD of the
  env `proxyFunc`. When no override is set, behavior is UNCHANGED (env `proxyFunc`
  path) — fully backward-compatible, additive. **This is the single edit to
  `utils/client.go`; it is additive and is NOT a handler body / NOT routes_admin /
  NOT selection.go — it is the sanctioned outbound-injection point.**
- The provider already plumbs `SetNetworkConfig` → `p.networkConfig`
  (`generic/provider.go:44`); ADD an additive call inside the provider's request
  path (or in `New`/`SetNetworkConfig`) that pushes `networkConfig.ProxyURL` into
  the ClientPool override. **DECIDE the exact injection site (§8 ESC-INJECT):**
  RECOMMENDED = have `SetNetworkConfig` call `p.client.SetProxyURL(config.ProxyURL)`
  (one additive line in each provider's existing `SetNetworkConfig` — generic +
  openai; these are NOT frozen handler bodies, they are the provider-client
  injection point named in the brief). If touching multiple provider files is
  undesirable, the alternative is to honor the override entirely inside ClientPool
  by reading a field set at construction — decide at T-proxywire. KEEP the edit
  ADDITIVE + minimal.

**selection.go additive hook (`internal/inference/selection.go`).** After a
connection is selected (the `SelectConnection` / `WithAccountFallback` result,
`selection.go:132,226`), resolve the connection's `proxy_pool_id` (§8 ESC-CONN-LINK)
→ look up the active pool via `platform.ProxyPoolService` → build the proxy URL
(`protocol://[user:pass@]host:port`) → set it on the provider's `NetworkConfig`
before the outbound call. **ADDITIVE ONLY** — a new helper + a single additive call
site; NO change to the existing selection/eligibility/cooldown logic. Because
w7-route ALSO edits selection.go (weighted selection), the orchestrator serializes
the two edit windows (selection-micro-serial, MAP §267). **DECIDE the resolution
seam (§8 ESC-RESOLVE):** RECOMMENDED = a small additive function
`resolveConnectionProxy(conn *store.Connection) (proxyURL string, ok bool)` that
the runner/provider-build path calls; the SelectionEngine gains an additive
dependency on a `ProxyResolver` interface (constructor stays compatible via an
optional setter, NO required `NewSelectionEngine` signature change — mirror the
gov-1 "no New() signature change" rule). If the cleanest additive site is actually
`internal/inference/factory.go` / `runner.go` (where the provider instance is built
and `SetNetworkConfig` is called) rather than selection.go itself, prefer that —
but the brief + MAP name selection.go as the coordination point, so the additive
hook lands there and the orchestrator serializes it regardless. Confirm the exact
additive site at T-proxywire; do NOT touch existing selection logic.

### 1.7 SSRF mitigation (PAR-AUTH-020) — exact policy (NEW)

`internal/platform/outboundproxy.go` (NEW) holds the SSRF evaluator. **Policy
(binding default — §8 ESC-SSRF-POLICY for confirmation):** for any
user-controllable outbound target (the connectivity-test probe target AND, where
applicable, a resolved dial host), RESOLVE the host to IPs and BLOCK if ANY resolved
IP falls in a disallowed range; ALLOW only public/global-unicast addresses.

Blocked ranges (deterministic, unit-tested):
- **Loopback:** `127.0.0.0/8`, `::1` (`net.IP.IsLoopback()`).
- **Private:** `10.0.0.0/8`, `172.16.0.0/12`, `192.168.0.0/16`, `fc00::/7`
  (`net.IP.IsPrivate()`).
- **Link-local:** `169.254.0.0/16`, `fe80::/10`
  (`net.IP.IsLinkLocalUnicast()` / `IsLinkLocalMulticast()`).
- **Unspecified / multicast / cloud-metadata:** `0.0.0.0`, `::`,
  `IsUnspecified()`, `IsMulticast()`, and explicitly `169.254.169.169`/
  `169.254.169.254` (cloud metadata — already covered by link-local but call it out).

API (deterministic, no network where possible):
- `IsBlockedIP(ip net.IP) (blocked bool, reason string)` — pure function over the
  range set; FULLY deterministic; the core unit-test surface.
- `IsBlockedTarget(host string) (blocked bool, reason string, err error)` — parses
  host; if it's a literal IP, calls `IsBlockedIP`; if a hostname, resolves via an
  injectable resolver seam (so tests are deterministic — inject a fake resolver
  returning fixed IPs) and blocks if ANY resolved IP is blocked.

**Deterministic unit tests (binding):** `IsBlockedIP` blocks `127.0.0.1`, `10.1.2.3`,
`172.16.0.1`, `192.168.1.1`, `169.254.0.1`, `169.254.169.254`, `::1`, `fe80::1`,
`fc00::1`; ALLOWS `8.8.8.8`, `1.1.1.1`, `93.184.216.34`, a public IPv6. The
connectivity-test refuses a probe target resolving to any blocked IP.

**Where it hooks:** (a) the connectivity-test guards its probe target before dialing
(§1.6); (b) DECIDE whether to also guard the LIVE outbound dial path (§8
ESC-SSRF-SCOPE). RECOMMENDED default for w7-plat-1: guard the connectivity-test
probe target (the directly user-triggered SSRF vector) AND the per-connection proxy
URL host (a user-configured proxy host that points at an internal address is the
PAR-AUTH-020 vector). Guarding ALL provider outbound targets (catalog base URLs) is
broader and risks blocking legitimate self-hosted backends — leave that as a tracked
follow-up unless the operator wants it (record in open-questions). The SSRF evaluator
is reusable for that future scope.

### 1.8 routes_admin.go registration (serial-slot additive, §3)

Add (additive appends; static/deeper-before-`{id}` precedence honored by the file —
see the `/api/providers/{id}/catalog` deeper-route precedent `routes_admin.go:101-106`
and `/api/connections/{id}/refresh:116`):
```go
// Proxy-pools CRUD (static collection + batch before {id}; {id}/test deepest).
r.GET("/api/proxy-pools", h.RequireSession(h.ListProxyPools))
r.POST("/api/proxy-pools", h.RequireSession(h.CreateProxyPool))
r.POST("/api/proxy-pools/batch", h.RequireSession(h.BatchProxyPools))   // §8 ESC-BATCH
r.GET("/api/proxy-pools/{id}", h.RequireSession(h.GetProxyPool))
r.PUT("/api/proxy-pools/{id}", h.RequireSession(h.UpdateProxyPool))
r.DELETE("/api/proxy-pools/{id}", h.RequireSession(h.DeleteProxyPool))
r.POST("/api/proxy-pools/{id}/test", h.RequireSession(h.TestProxyPool))
```
Route-precedence note: `/api/proxy-pools/batch` (static) vs `/api/proxy-pools/{id}`
and `/api/proxy-pools/{id}/test` (deeper) follow the file's existing
static/deeper-before-param ordering. A genuine `fasthttp/router` collision is §8
ESC-ROUTE, not a silent path change. Diff bound §5: the route block is ONE commit,
additive only.

### NOT in scope (explicit)

- **No UI page/component/route/store edits** — `/proxy-pools` and all w6-m
  components are FROZEN consume-only (decision 8). The ONLY UI-tree touches are the
  mock-body + seed corrections (§1.2 / §3).
- **No edits to other platform domains** — tunnels (w7-plat-2), mitm (w7-plat-3) are
  disjoint; do not touch their mocks/seeds/Go.
- **No edits to pre-existing admin handlers' bodies** — apikeys, virtualkeys,
  providers*, connections, combos, auth, version, usage are FORBIDDEN. EXCEPTIONS
  (NOT handler bodies, sanctioned by the brief): the ADDITIVE selection.go proxy
  hook (§1.6), the ADDITIVE `utils/client.go` proxy-override (§1.6), and the
  ADDITIVE one-line `SetNetworkConfig` injection in the provider files (§1.6,
  ESC-INJECT). Plus the additive `connections` linkage column + scan (ESC-CONN-LINK)
  — store-layer, not a handler body.
- **No edits to inference selection/eligibility/cooldown logic** — the selection.go
  hook is ADDITIVE only (new helper + one call site), serialized vs w7-route.
- **No interface change** — `schemas.Provider`/`SetNetworkConfig`/`NewSelectionEngine`
  signatures PRESERVED (additive setters / optional deps only; MAP decision 9).
- **No destructive DDL / column renames** — additive `ensureTable`/`ensureColumn`
  ONLY (decision 2).
- **No new global state** — handlers compose `h.store`; the platform service is
  constructed over `h.store`.
- **No secret exposure** — proxy `password` (if any) `*_enc` at rest + masked in
  responses (`password_set`); SSRF/test responses carry no secrets (§5 grep proofs).
- **No broad outbound SSRF retrofit** across all provider targets (tracked
  follow-up; §1.7 ESC-SSRF-SCOPE).

---

## 2. Precondition checks

Run all before any edit; abort and report to orchestrator on any failure.

```bash
# P0 — clean tree; record <base>
git status --porcelain     # must be empty (explicit `git add <file>`, never -A;
                           # ui/dist/** gitignored — never stage it)
git rev-parse HEAD         # record as <base> for §5

# P1 — the gaps are REAL (no Go for proxy-pools / no honored per-conn proxy)
grep -nE '/api/proxy-pools' internal/server/routes_admin.go ; echo "^ expect EMPTY"
test ! -e internal/store/proxypools.go && test ! -e internal/admin/proxypools.go && echo "proxy admin/store gap OK"
test ! -e internal/platform/proxypools.go && test ! -e internal/platform/outboundproxy.go && echo "platform gap OK"
grep -nE 'proxy_pool_id' internal/store/connections.go ; echo "^ expect EMPTY (linkage not yet added)"

# P2 — reused surfaces present (the de-risk)
grep -n "type ClientPool\|func NewClientPool\|clientForProxy\|proxyFunc" internal/providers/utils/client.go
grep -n "ProxyURL\|type NetworkConfig\|SetNetworkConfig" internal/schemas/provider.go
grep -n "s.cipher.Encrypt\|s.cipher.Decrypt" internal/store/connections.go
grep -n "func writeData\|func writeError" internal/admin/respond.go
grep -n "func (h \*Handlers) recordAudit" internal/admin/audit.go
grep -n "func newTestEnv\|func.*call(" internal/admin/admin_test.go

# P3 — migrate pattern + secret-at-rest precedent
grep -n "tables := \|CREATE TABLE IF NOT EXISTS\|ensureColumn(db, col" internal/store/migrate.go | head
grep -n "isLoopbackHostname\|net.SplitHostPort" internal/server/guard.go   # SSRF precedent (not edited)

# P4 — the W6-m UI + spec present (consume-only) and the mock to correct
test -f ui/e2e/proxy-pools.spec.ts && echo "spec present"
test -f ui/e2e/mocks/handlers/proxy-pools.ts && test -f ui/e2e/mocks/seed/proxy-pools.ts && echo "mock+seed present"
grep -n "batch\|/test\|last_check_at\|last_check_status\|includeUsage\|isActive" ui/e2e/mocks/handlers/proxy-pools.ts
grep -nE "includeUsage|isActive|/batch|/test|proxy-pools" ui/src/**/*.ts* 2>/dev/null | head   # what the PAGE actually calls (resolves ESC-USAGE/ESC-BATCH)

# P5 — routes_admin.go serial slot FREE + selection.go micro-serial FREE
git log --oneline -5 -- internal/server/routes_admin.go    # last touch = prior chain holder (merged)
git log --oneline -5 -- internal/inference/selection.go    # confirm no unmerged w7-route selection.go edit in flight
# Orchestrator MUST confirm: (a) no concurrent W7 plan holds an unmerged routes_admin.go edit;
# (b) no concurrent w7-route holds an unmerged selection.go edit, before T-proxywire.

# P6 — green at base
go test ./... && go vet ./... && go build ./...     # exit 0 (Go untouched-green)
cd ui && npm run build                               # exit 0 (build BEFORE e2e — e2e-hygiene)
cd ui && npx playwright test e2e/proxy-pools.spec.ts # PASS at base against the W6 mock; record in WORKFLOW.md
```

---

## 3. Exclusive file ownership

After w7-plat-1 merges, all CREATE files are owned by w7-plat-1; later plans consume,
never edit (MAP decision 7).

**CREATE — store (NEW):**

| File | Contract |
|---|---|
| `internal/store/proxypools.go` | `ProxyPool` struct + `CreateProxyPool`/`ListProxyPools(filterActive *bool)`/`GetProxyPoolByID`/`UpdateProxyPool`/`DeleteProxyPool`/`SetProxyPoolCheck`/`CountConnectionsUsingProxyPool` + `scanProxyPool`; `newID()`, unix ts, `ErrNotFound`, `boolToInt`; `password_enc` via `s.cipher` (if creds). Mirrors `virtualkeys.go`/`connections.go`. |
| `internal/store/proxypools_test.go` | Table-driven, temp `store.Open`: create→get→list(+isActive filter)→update→delete→404; password round-trips encrypted (raw column ≠ cleartext); `CountConnectionsUsingProxyPool` returns 0/N. RED first. |

**EXTEND — store (additive only):**

| File | Change (additive ONLY) |
|---|---|
| `internal/store/migrate.go` | ADD the `proxy_pools` table to the `tables` slice; ADD `ensureColumn("connections","proxy_pool_id","TEXT NOT NULL DEFAULT ''")` to the additive-column loop (ESC-CONN-LINK). ADDITIVE ONLY. |
| `internal/store/connections.go` | ADDITIVE: extend `Connection` struct with `ProxyPoolID string`; add it to the INSERT/SELECT/UPDATE column lists + `scanConnection`. Existing signatures PRESERVED. (Smallest linkage; ESC-CONN-LINK.) |
| `internal/store/connections_test.go` (EXTEND additively) | RED first: create a connection with `ProxyPoolID` → round-trips; `CountConnectionsUsingProxyPool` reflects it. |

**CREATE — domain (NEW):**

| File | Contract |
|---|---|
| `internal/platform/proxypools.go` | `ProxyPoolService` over `*store.Store`: `List/Create/Get/Update/Delete` thin wrappers (if arch test requires the seam), `TestConnectivity(id) (result, error)` (injectable prober — deterministic), `ResolveProxyForConnection(conn) (proxyURL string, ok bool)`. Constructor `NewProxyPoolService(st, prober)`. No `init()`; errors-as-values. |
| `internal/platform/proxypools_test.go` | TestConnectivity ok/error/SSRF-refused via a fake prober (no network); ResolveProxyForConnection builds the correct URL for an active pool, returns ok=false for an inactive/missing pool. RED first. |
| `internal/platform/outboundproxy.go` | `IsBlockedIP(ip) (bool,string)` (pure) + `IsBlockedTarget(host) (bool,string,error)` (injectable resolver). Blocks loopback/private/link-local/unspecified/multicast/metadata (§1.7). No `init()`. |
| `internal/platform/outboundproxy_test.go` | Deterministic table: blocks 127.0.0.1/10.x/172.16.x/192.168.x/169.254.x/169.254.169.254/::1/fe80::/fc00::; allows 8.8.8.8/1.1.1.1/public-IPv6. RED first. |

**EXTEND — outbound client (additive only — the sanctioned injection point):**

| File | Change (additive ONLY) |
|---|---|
| `internal/providers/utils/client.go` | ADD a per-`ClientPool` proxy override (`SetProxyURL(string)` or a `proxyOverride` field) so a configured proxy is honored by `Do` via the EXISTING `clientForProxy` machinery; no override → UNCHANGED env-proxyFunc behavior. Additive; backward-compatible. |
| `internal/providers/utils/client_test.go` (EXTEND/CREATE additively) | RED first: with an override set, `Do` routes via the proxied client; without, via the base client (assert via the proxies map / a fake dial seam, no real network). |
| `internal/providers/generic/provider.go` (+ `internal/providers/openai/provider.go`) | ADDITIVE one line in the EXISTING `SetNetworkConfig`: push `config.ProxyURL` into the ClientPool override (ESC-INJECT). NO signature change; NOT a handler body — this is the named provider-client injection point. |

**EXTEND — inference (ADDITIVE hook, selection-micro-serial):**

| File | Change (additive ONLY, serialized vs w7-route) |
|---|---|
| `internal/inference/selection.go` | ADD a proxy-resolution hook: a `ProxyResolver` interface dep on the SelectionEngine (set via an optional additive setter — NO `NewSelectionEngine` signature change) + an additive call site that resolves the selected connection's proxy and sets it on the provider's `NetworkConfig`. NO change to existing selection/eligibility/cooldown logic. (If `factory.go`/`runner.go` is the cleaner additive site, prefer it — ESC-RESOLVE — but the orchestrator serializes selection.go regardless.) |
| `internal/inference/selection_test.go` (EXTEND additively) OR a new `internal/inference/proxyresolve_test.go` | RED first: a selected connection with an active proxy_pool_id yields the expected `NetworkConfig.ProxyURL`; no pool → no proxy; existing selection tests UNCHANGED-green. |

**CREATE — transport (NEW):**

| File | Contract |
|---|---|
| `internal/admin/proxypools.go` | `ListProxyPools`/`CreateProxyPool`/`BatchProxyPools`/`GetProxyPool`/`UpdateProxyPool`/`DeleteProxyPool`/`TestProxyPool` + `proxyPoolDTO` + request/validate; `writeData`/`writeError`; 409 bound-connection guard on delete; `h.recordAudit` after each mutation (best-effort). NEVER echoes `password`. |
| `internal/admin/proxypools_test.go` | via `newTestEnv`: create→list(≥1, +isActive filter)→get→update→delete→404; create empty-name/bad-port→400; **delete-bound→409 + pool-still-exists**; test→persists last_check_status; **no response leaks `password`/`password_enc`**; an audit entry is written on create. RED first. |

**MODIFY — serial-slot route registration (additive only):**

| File | Change |
|---|---|
| `internal/server/routes_admin.go` | ADD the 7 route lines (§1.8). NOTHING else. ONE commit. SERIAL SLOT — only holder while live; RELEASE to w7-plat-2 on close. |

**MODIFY — handlers wiring (additive only, IF a service field is chosen):**

| File | Change |
|---|---|
| `internal/admin/handlers.go` | OPTIONAL additive: add a `proxyPools *platform.ProxyPoolService` field constructed in `New` (NO signature change — build over the existing `st`). If a free accessor is cleaner, skip. Decide at T-proxypools. |

**MODIFY — e2e mock corrections (mirror real Go, decision 1):**

| File | Change |
|---|---|
| `ui/e2e/mocks/handlers/proxy-pools.ts` (BODY) | POST: mirror the Go default (no cosmetic create-time `last_check_at`; defaults as Go). `/test`: mirror Go `{ok,latency_ms[,status]}`. `/batch`: keep iff the page calls it (ESC-BATCH). GET/PUT/DELETE/`{id}` already mirror; verify field names. Do NOT add a 409 delete branch unless a spec needs it (ESC-409-MOCK). |
| `ui/e2e/mocks/seed/proxy-pools.ts` (BODY) | Already the 9-field `ProxyPool` shape — verify; correct only if a field name diverges. No password field needed. |

**FORBIDDEN:** everything else. Explicitly: all pre-existing `internal/admin/*.go`
except the NEW proxypools files + the OPTIONAL handlers.go additive field; all other
`internal/store/*.go` except proxypools (NEW) + migrate/connections (additive); all
other `internal/platform/*` is NEW-by-this-plan only; `internal/inference/*` except
the ADDITIVE selection.go (or factory/runner) hook; `internal/providers/*` except the
ADDITIVE `utils/client.go` override + the one-line `SetNetworkConfig` injection in
generic/openai providers; `internal/server/guard.go` (REUSE the SSRF precedent, no
edit); all UI `ui/src/**` (FROZEN, decision 8); all other mocks/seeds/specs;
`ui/package.json` + lockfile; `ui/vite.config.ts`; `ui/playwright.config.ts`;
`ui/dist/**` (gitignored — NEVER stage, NEVER revert `ui/dist/index.html`). Touching
any of these is an automatic REJECT.

---

## 4. TDD tasks

Cadence (strict, AGENTS.md "TDD always"): **no Go impl file may exist before its
`_test.go` is committed RED.** `go test ./... && go vet ./... && go build ./...` green
at EVERY commit. The e2e spec stays green throughout (real Go is additive; mock
corrections mirror it). Order: SSRF policy → proxy-pools store+admin → connectivity
test → connection linkage + selection.go proxy wiring (micro-serial) → routes serial
slot → mock corrections → closeout.

### T-ssrf — STEP(a) RED, STEP(b) impl
STEP(a): write `internal/platform/outboundproxy_test.go` (deterministic IP table).
`go test ./internal/platform/ -run 'SSRF|Block'` → FAIL. Commit RED:
`phase-1/w7-plat-1: failing SSRF outbound-policy tests (TDD red)`.
STEP(b): implement `internal/platform/outboundproxy.go`. Gates green. Commit:
`phase-1/w7-plat-1: outbound SSRF policy (block private/loopback/link-local)`.

### T-proxypools — STEP(a) RED store+admin, STEP(b) impl
STEP(a): write `internal/store/proxypools_test.go` + `internal/admin/proxypools_test.go`;
add the `proxy_pools` table to `migrate.go` (so tests compile + the table exists).
`go test ./internal/store/ -run Proxy` and `go test ./internal/admin/ -run Proxy` →
FAIL. Commit RED:
`phase-1/w7-plat-1: failing proxy-pools store+admin tests (TDD red)`.
STEP(b): implement `internal/store/proxypools.go` + `internal/admin/proxypools.go`
(CRUD + 409 guard wiring stubbed against CountConnectionsUsingProxyPool) +
`internal/platform/proxypools.go` service. Gates green. Commit:
`phase-1/w7-plat-1: proxy-pools store + platform service + admin CRUD`.

### T-conntest — STEP(a) RED, STEP(b) impl
STEP(a): write `internal/platform/proxypools_test.go` connectivity-test cases
(fake prober: ok/error/SSRF-refused) + the `TestProxyPool` admin assertion. → FAIL.
Commit RED: `phase-1/w7-plat-1: failing proxy connectivity-test tests (TDD red)`.
STEP(b): implement `TestConnectivity` (injectable prober + SSRF guard on the probe
target) + the `TestProxyPool` handler (persists last_check_status/at). Gates green.
Commit: `phase-1/w7-plat-1: proxy connectivity-test endpoint (SSRF-guarded)`.

### T-proxywire — connection linkage + per-connection proxy (selection-micro-serial)
TAKE the selection.go micro-serial slot (orchestrator confirms FREE at P5 — no
unmerged w7-route selection.go edit). STEP(a): write the additive
`connections_test.go` linkage case + `client_test.go` override case +
`selection_test.go`/`proxyresolve_test.go` resolution case → FAIL. Commit RED:
`phase-1/w7-plat-1: failing per-connection proxy + linkage tests (TDD red)`.
STEP(b): ADD `connections.proxy_pool_id` (migrate + struct + scan); ADD the
`ClientPool` proxy override (`utils/client.go`) + the one-line `SetNetworkConfig`
injection (generic/openai); ADD the additive selection.go proxy-resolution hook +
`ResolveProxyForConnection`. Existing selection tests UNCHANGED-green. Gates green.
Commit: `phase-1/w7-plat-1: per-connection proxy resolution (selection hook + client wiring)`.
RELEASE the selection.go micro-serial slot.

### T-routes — serial-slot route registration
TAKE the routes_admin.go serial slot (orchestrator confirms FREE at P5). Add the 7
route lines (§1.8). Gates green. Commit (ONE commit touches the serial file):
`phase-1/w7-plat-1: register proxy-pools admin routes (serial slot)`.

### T-mocks — mock-body corrections (mirror real Go, decision 1)
Correct `proxy-pools.ts` (POST defaults, `/test` shape, `/batch` per ESC-BATCH);
verify the seed. Gates: `cd ui && npm run build` green (BEFORE playwright);
`npx playwright test e2e/proxy-pools.spec.ts` green (still). If a correction reds a
non-w7-plat-1 spec, STOP + ESCALATE (§8 ESC-MOCK). Commit:
`phase-1/w7-plat-1: correct proxy-pools mock to mirror real Go DTOs`.

### T-close — full gates + closeout
```bash
go test ./... && go vet ./... && go build ./...
go test ./internal/platform/... ./internal/admin/ -run 'Proxy' -v
go test ./internal/store/ -run 'Proxy|Connection' -v
go test ./internal/inference/ -run 'Proxy|Select' -v        # existing selection tests still green
go test ./internal/providers/utils/ -run 'Proxy|Client' -v
cd ui && npm run build                                       # BEFORE playwright (e2e-hygiene)
cd ui && npx playwright test e2e/proxy-pools.spec.ts        # green (ISOLATED, no concurrent playwright)
cd ui && npx playwright test                                # full suite green (no regressions)
```
Flip the matrix: PAR-PLAT-001..005, PAR-PLAT-009, PAR-AUTH-020 → HAVE (real Go, cite
§1.4-1.7); PAR-UI-019/104/105 PARTIAL → HAVE. Mark `open-questions.md` w6-m ESC-1b
RESOLVED with a cite; append any new open items (§8 — broad-SSRF-retrofit follow-up,
proxy-auth scope). Update `docs/WORKFLOW.md` (P6 base observation; the ESC-SCHEMA /
ESC-CONN-LINK / ESC-SSRF-POLICY / ESC-INJECT / ESC-RESOLVE decisions; the
serial-slot take/release; the selection.go micro-serial take/release; the mock
corrections). Final commit:
`phase-1/w7-plat-1: close — proxy-pools + outbound proxy/SSRF Go; matrix flip; mock mirror`.
**On the close commit, RELEASE the routes_admin.go serial slot to w7-plat-2.**

---

## 5. Binary acceptance criteria

All must hold; each is yes/no. `<base>` = the commit recorded at P0. Diff gate is
**w7-plat-1 commit-range-scoped** (§7).

**Test gates**
- `go test ./... && go vet ./... && go build ./...` → exit 0.
- `go test ./internal/platform/... ./internal/admin/ -run 'Proxy' -v` → exit 0, all
  pass (SSRF policy ≥12 cases; proxy CRUD ≥6; connectivity-test ≥3 incl SSRF-refuse;
  delete-bound→409; no-password-leak).
- `go test ./internal/store/ -run 'Proxy|Connection' -v` → exit 0 (incl password
  encrypted-at-rest + CountConnectionsUsingProxyPool).
- `go test ./internal/inference/ -run 'Proxy|Select' -v` → exit 0 (proxy resolution +
  existing selection tests unchanged-green).
- `go test ./internal/providers/utils/ -run 'Proxy|Client' -v` → exit 0.
- `cd ui && npm run build` → exit 0 (BEFORE playwright).
- `cd ui && npx playwright test e2e/proxy-pools.spec.ts` → exit 0, all pass, 0 skipped
  (ISOLATED; no concurrent playwright; `ui/dist/index.html` NEVER reverted).
- `cd ui && npx playwright test` → exit 0, no spec green-at-base goes red.

**TDD-order proof** — each impl file's covering test appears in an earlier-or-equal
commit:
```bash
for pair in \
  "internal/platform/outboundproxy_test.go:internal/platform/outboundproxy.go" \
  "internal/platform/proxypools_test.go:internal/platform/proxypools.go" \
  "internal/store/proxypools_test.go:internal/store/proxypools.go" \
  "internal/admin/proxypools_test.go:internal/admin/proxypools.go" ; do
  tf=${pair%%:*}; im=${pair##*:}
  ct=$(git log --format=%ct --diff-filter=A -1 -- "$tf")
  cf=$(git log --format=%ct --diff-filter=A -1 -- "$im")
  [ "$ct" -le "$cf" ] || echo "TDD VIOLATION: $im"     # prints nothing
done
```

**Grep proofs**
```bash
# proxy-pools transport + store
grep -n "func (h \*Handlers) ListProxyPools\|CreateProxyPool\|GetProxyPool\|UpdateProxyPool\|DeleteProxyPool\|TestProxyPool" internal/admin/proxypools.go
grep -n "protocol\|host\|port\|is_active\|last_check_status\|last_check_at" internal/admin/proxypools.go   # canonical DTO
grep -n "func (s \*Store) CreateProxyPool\|ListProxyPools\|GetProxyPoolByID\|UpdateProxyPool\|DeleteProxyPool\|CountConnectionsUsingProxyPool\|SetProxyPoolCheck" internal/store/proxypools.go
grep -n "writeData\|writeError\|recordAudit" internal/admin/proxypools.go      # envelope + audit
grep -n "409\|StatusConflict\|CountConnectionsUsingProxyPool" internal/admin/proxypools.go   # bound-connection guard
# SSRF policy block ranges (binding)
grep -nE "127\.|10\.|172\.16|192\.168|169\.254|IsLoopback|IsPrivate|IsLinkLocal|fe80|fc00|169\.254\.169\.254" internal/platform/outboundproxy.go
grep -n "func IsBlockedIP\|func IsBlockedTarget" internal/platform/outboundproxy.go
# per-connection proxy wiring
grep -n "SetProxyURL\|proxyOverride\|ProxyURL" internal/providers/utils/client.go
grep -n "ProxyPoolID\|proxy_pool_id" internal/store/connections.go internal/store/migrate.go
grep -n "ResolveProxyForConnection\|NetworkConfig\|ProxyResolver" internal/inference/selection.go internal/platform/proxypools.go
# routes
grep -nE '/api/proxy-pools' internal/server/routes_admin.go
# no init(); no global state
! grep -rn "func init(" internal/admin/proxypools.go internal/store/proxypools.go internal/platform/proxypools.go internal/platform/outboundproxy.go && echo "no init() OK"
```

**No-secret-exposure proofs (binding)**
```bash
# password/cleartext never appears in any proxy DTO/response field
! grep -nE 'json:"password"' internal/admin/proxypools.go && echo "no password json field OK"
grep -nA12 'type proxyPoolDTO struct' internal/admin/proxypools.go ; echo "^ must NOT contain password (only password_set, if any)"
# password encrypted at rest (if creds supported)
grep -n "password_enc\|s.cipher.Encrypt\|s.cipher.Decrypt" internal/store/proxypools.go
# runtime no-leak: marshal every proxy-pool response, assert it contains neither the
# cleartext password nor any ciphertext prefix (asserted in proxypools_test.go).
# additive migrations only (no DROP/RENAME introduced by this plan)
git diff <base>..HEAD -- internal/store/migrate.go | grep -E '^\+' | grep -iE 'DROP COLUMN|RENAME COLUMN|DROP TABLE' | wc -l   # = 0
```

**Negative / freeze proofs (w7-plat-1 commit-range — §7)**
```bash
R="<first-w7-plat-1>^..<last-w7-plat-1>"
# Only the sanctioned Go files changed:
git diff $R --name-only -- internal/ | grep -vE \
 'internal/store/(proxypools|connections|migrate)(_test)?\.go|internal/platform/(proxypools|outboundproxy)(_test)?\.go|internal/admin/(proxypools)(_test)?\.go|internal/admin/handlers\.go|internal/inference/(selection|factory|runner|proxyresolve)(_test)?\.go|internal/providers/utils/client(_test)?\.go|internal/providers/(generic|openai)/provider(_test)?\.go|internal/server/routes_admin\.go' \
 | wc -l                                                                  # = 0
# Frozen admin handlers untouched:
git diff $R --name-only -- internal/admin/auth.go internal/admin/apikeys.go internal/admin/virtualkeys.go internal/admin/providers.go internal/admin/connections.go internal/admin/combos.go | wc -l   # = 0
# Frozen guard / SSRF precedent untouched (reused, not edited):
git diff $R --name-only -- internal/server/guard.go | wc -l              # = 0
# selection.go = additive only (no deletions):
git diff $R -- internal/inference/selection.go | grep -E '^-' | grep -v '^---' | wc -l   # = 0
# utils/client.go = additive only (no deletions of existing logic):
git diff $R -- internal/providers/utils/client.go | grep -E '^-' | grep -v '^---' | wc -l   # = 0
# UI frozen except the sanctioned mock/seed bodies:
git diff $R --name-only -- ui/ | grep -vE \
 'ui/e2e/mocks/handlers/proxy-pools\.ts|ui/e2e/mocks/seed/proxy-pools\.ts' | wc -l   # = 0
git diff $R --name-only -- ui/src/ | wc -l                               # = 0 (src frozen)
git diff $R --name-only -- ui/dist/ | wc -l                             # = 0 (dist gitignored/never staged)
# routes_admin.go = exactly ONE commit, additive:
git log --oneline $R -- internal/server/routes_admin.go | wc -l          # = 1
git diff $R -- internal/server/routes_admin.go | grep -E '^-' | grep -v '^---' | wc -l   # = 0 (no deletions)
```

---

## 6. Out of scope (restated, binding)

No UI src edits (decision 8 — pages/components/routes/stores frozen); only the
sanctioned proxy-pools mock-body + seed corrections. No edits to pre-existing admin
handler bodies (auth, apikeys, virtualkeys, providers*, connections, combos,
version, usage). No edits to existing inference selection/eligibility/cooldown logic
(selection.go hook is ADDITIVE only). No `guard.go` edit (SSRF precedent reused, not
modified). No interface / `New(...)` / `SetNetworkConfig` / `NewSelectionEngine`
signature changes (additive setters/optional deps only). No JWT. No destructive DDL —
additive `ensureTable`/`ensureColumn` only. No new global state. No other platform
domains (w7-plat-2 tunnels, w7-plat-3 mitm). No broad SSRF retrofit across all
provider outbound targets (tracked follow-up). No secret exposure (proxy password
`*_enc` + masked; SSRF/test responses carry no secrets). Mock-vs-Go contradiction →
escalate (§8), never fudge a mock or edit a frozen handler. NEVER revert
`ui/dist/index.html`; NEVER run concurrent playwright; `npm run build` before e2e
(e2e-hygiene).

## 7. Diff-gate scope

W7 platform plans (plat-1/2/3) commit to main concurrently, so a broad
`<base>..HEAD` range sweeps in sibling commits. The diff gate MUST be scoped to
w7-plat-1's own commits. Isolate them:
`git log --oneline main | grep "^[0-9a-f]* phase-1/w7-plat-1:" | awk '{print $1}'`
then `git diff <first-w7-plat-1>^..<last-w7-plat-1> -- [file list]`.

`git diff <first>^..<last> --name-only` must be exactly a subset of:
```
internal/store/proxypools.go
internal/store/proxypools_test.go
internal/store/connections.go            (additive: ProxyPoolID field + scan)
internal/store/connections_test.go
internal/store/migrate.go                (additive table + ensureColumn; ONE commit per concern ok)
internal/platform/proxypools.go
internal/platform/proxypools_test.go
internal/platform/outboundproxy.go
internal/platform/outboundproxy_test.go
internal/admin/proxypools.go
internal/admin/proxypools_test.go
internal/admin/handlers.go               (OPTIONAL additive proxyPools field; no New() sig change)
internal/inference/selection.go          (ADDITIVE proxy hook; serialized vs w7-route; ONE concern)
internal/inference/selection_test.go     (or internal/inference/proxyresolve_test.go)
internal/inference/factory.go            (CONDITIONAL — only if ESC-RESOLVE moves the hook here; additive)
internal/inference/runner.go             (CONDITIONAL — same)
internal/providers/utils/client.go       (additive proxy override)
internal/providers/utils/client_test.go
internal/providers/generic/provider.go   (additive one-line SetNetworkConfig injection)
internal/providers/openai/provider.go    (additive one-line SetNetworkConfig injection)
internal/server/routes_admin.go          (serial-slot additive routes; ONE commit)
ui/e2e/mocks/handlers/proxy-pools.ts     (body only — POST defaults, /test shape, /batch per ESC-BATCH)
ui/e2e/mocks/seed/proxy-pools.ts         (verify; correct only on divergence)
.planning/parity/matrix/*                 (row flips)
.planning/parity/plans/open-questions.md
docs/WORKFLOW.md
```
Any file outside this list in the scoped diff is an automatic review REJECT.
`internal/server/guard.go`, the pre-existing admin handlers, and all `ui/src/**`
are deliberately ABSENT — touching them is an automatic REJECT. The `routes_admin.go`
edit must appear in exactly ONE commit (§5); selection.go must be ADDITIVE-only and
serialized vs w7-route; the serial slot is released to w7-plat-2 on close.

## 8. Escalations / decisions (explicit — recommended defaults, do not fabricate)

- **ESC-SCHEMA (RESOLVED at authoring — typed vs JSON columns, binding default).**
  The `proxy_pools` row is a fixed 9-field DTO with a `?isActive` filter + a
  `WHERE proxy_pool_id=?` delete guard. **Decision: typed columns** (clean filter +
  join; matches the gov-plan precedent of typed columns for fixed-shape domains,
  JSON-blob only for nested data). A JSON `data` column is the zero-typed-migration
  alternative but blocks SQL-level filtering. RECOMMENDED: typed columns. Flag for
  orchestrator confirmation.
- **ESC-CONN-LINK (RESOLVED at authoring — connection→proxy linkage, binding
  default).** Two options: (a) an additive `ensureColumn
  connections.proxy_pool_id` + a `Connection.ProxyPoolID` field, or (b) a key inside
  the existing `Connection.Metadata` JSON. **Decision: (a) the typed column** — it
  enables the `CountConnectionsUsingProxyPool` delete guard via a clean
  `WHERE proxy_pool_id=?` and the per-connection resolution without parsing
  metadata. Additive, cheap. RECOMMENDED. Alternative (metadata key) avoids a
  migration but makes the guard a full scan + parse. Flag for confirmation.
- **ESC-SSRF-POLICY (RESOLVED at authoring — exact blocked ranges, binding
  default).** Block loopback (`127.0.0.0/8`, `::1`), private (`10/8`, `172.16/12`,
  `192.168/16`, `fc00::/7`), link-local (`169.254/16`, `fe80::/10`), unspecified,
  multicast, and cloud-metadata (`169.254.169.254`); ALLOW public/global-unicast.
  Implemented via `net.IP.IsLoopback/IsPrivate/IsLinkLocalUnicast/IsUnspecified/
  IsMulticast` + explicit metadata check. RECOMMENDED as stated; flag for
  confirmation (e.g. if the operator runs legitimate self-hosted proxies on a
  private range, an allowlist override is a follow-up — record in open-questions).
- **ESC-SSRF-SCOPE (RESOLVED at authoring — where SSRF guards, binding default).**
  Guard (a) the connectivity-test probe target and (b) the per-connection proxy URL
  host (the PAR-AUTH-020 vectors a user directly controls). Do NOT (yet) guard all
  provider outbound targets (catalog base URLs) — that risks blocking legitimate
  self-hosted backends; it is a tracked follow-up reusing the same evaluator.
  RECOMMENDED; flag for confirmation.
- **ESC-INJECT (RESOLVED at authoring — where the ClientPool honors per-conn proxy,
  binding default).** `NetworkConfig.ProxyURL` is stored but inert. **Decision:**
  ADD a `ClientPool.SetProxyURL` override (additive, uses the existing
  `clientForProxy` machinery) and have each provider's EXISTING `SetNetworkConfig`
  push `config.ProxyURL` into it (one additive line in generic + openai providers —
  the named injection point, NOT a frozen handler body). Alternative: honor entirely
  inside ClientPool via a construction-time field (avoids touching provider files).
  RECOMMENDED: the `SetNetworkConfig` push (minimal, explicit). Decide finally at
  T-proxywire; keep additive.
- **ESC-RESOLVE (RESOLVED at authoring — selection.go vs factory/runner hook site,
  binding default + the w7-route coordination).** The brief + MAP name selection.go
  as the per-connection proxy coordination point (shared with w7-route). **Decision:**
  add the additive `ProxyResolver` hook so the resolved proxy lands on the provider's
  `NetworkConfig` before the outbound call; if the cleanest additive site is actually
  `factory.go`/`runner.go` (where the provider is built + `SetNetworkConfig` is
  already called), use that — but the orchestrator SERIALIZES selection.go vs
  w7-route regardless (selection-micro-serial, MAP §267). The edit is ADDITIVE-ONLY;
  `NewSelectionEngine` signature is PRESERVED (optional setter for the resolver).
  Flag the coordination for the orchestrator; do NOT begin T-proxywire until the
  selection.go window is confirmed free of an unmerged w7-route edit.
- **ESC-BATCH (CONDITIONAL — ship `/batch` or defer).** The registered mock has
  `POST /api/proxy-pools/batch`; the brief SCOPE does not list it. **Default: SHIP
  it** (thin loop over Create, cheap, mirrors the mock) UNLESS the w6-m page never
  calls it (VERIFY at P4/T-mocks via `grep -nE 'proxy-pools/batch' ui/src`). If
  unused, defer + record in open-questions. Flag for confirmation.
- **ESC-USAGE (CONDITIONAL — `?includeUsage` field).** The brief mentions a possible
  `?includeUsage` filter. SHIP a `usage_count` per-pool field on
  `GET /api/proxy-pools?includeUsage=true` ONLY if the page reads it (VERIFY at P4).
  Default: implement the query param as a no-op-safe additive field gated on the
  query; if the page never sends it, the param is harmless. Flag.
- **ESC-PROXY-CRED (RESOLVED at authoring — proxy password support, binding
  default).** Proxies commonly need auth and the UI `ProxyPool` carries `username`.
  **Decision:** support an optional `password`, stored `password_enc` at rest,
  MASKED in responses (`password_set`), used to build the proxy URL
  (`protocol://user:pass@host:port`). The mock seed has no password — no seed change.
  RECOMMENDED; if the operator wants no proxy auth, drop the password column. Flag.
- **ESC-TEST-SHAPE (RESOLVED at authoring — `/test` response, binding default).** Go
  returns `{data:{ok,latency_ms,status}}` and persists `last_check_status`/
  `last_check_at`; the mock is corrected to mirror `{ok,latency_ms}` (+`status` iff
  the page reads it — VERIFY). RECOMMENDED; flag.
- **ESC-409-MOCK (CONDITIONAL — delete-bound 409 in the mock).** The 409
  bound-connection guard is NEW Go behavior the mock lacks. Default: leave the mock
  delete as `{}` (the e2e spec deletes an UNBOUND pool); prove the 409 in the Go
  admin test only. Add a mock 409 branch ONLY if a spec exercises a bound delete
  (none expected). Flag if the spec surprises.
- **ESC-ARCH (CONDITIONAL — arch test on the proxy-pools layer).** The phase-12B arch
  test enforces transport→domain→repository. virtualkeys/apikeys call `h.store`
  directly. Because the connectivity-test + per-connection resolution warrant a
  domain seam (reused beyond the handler), build `platform/proxypools.go`. If the
  arch test ALSO requires the plain CRUD to route through the service (not
  handler→store), route all CRUD through `ProxyPoolService`; if it allows
  handler→store for CRUD, keep CRUD thin and the service holds only test/resolve.
  Decide at T-proxypools by running the arch test; do NOT pre-guess.
- **ESC-ROUTE (CONDITIONAL — fasthttp/router precedence).** `/api/proxy-pools/batch`
  (static) vs `/api/proxy-pools/{id}` and `/api/proxy-pools/{id}/test` (deeper)
  follow the file's static/deeper-before-param precedent
  (`routes_admin.go:101-106,116`). If the matcher mis-disambiguates or panics on a
  conflict, STOP and ESCALATE for a path arrangement — never silently diverge
  page/mock/Go.
- **ESC-MOCK (CONDITIONAL — mock ripple).** `proxy-pools.ts` is w6-m-owned and
  proxy-pools-only (not shared). If a body correction reds a non-w7-plat-1 spec or a
  seed correction ripples, STOP and ESCALATE for orchestrator serialization — no
  fudge, no frozen-branch edit.
- **Serial-slot dependency (§1.8 / P5).** w7-plat-1 TAKES the routes_admin.go slot
  in chain order (… → w7-mcp-3 → **w7-plat-1** → w7-plat-2 → …) and RELEASES it to
  w7-plat-2 on close. SEPARATELY, the selection.go micro-serial is taken at
  T-proxywire and released after. Orchestrator confirms exactly one unmerged holder
  of each (decision 3 / MAP §267) before the respective task.
```
