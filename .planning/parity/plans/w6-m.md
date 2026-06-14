# Micro-plan w6-m — Platform pages cluster (mitm/proxy-pools/tunnels — THE designated PARTIAL plan; UI-only, ZERO Go)

```
wave: 6
plan: w6-m
status: READY (rev 1 — authored against merged w6-a + w6-b + page waves, live
  tree @ 2978d2bc)
runs: page wave 2, AFTER w6-b MERGE (consumes frozen ui/src/components/ui/*) and
  AFTER w6-a MERGE (consumes apiFetch/ApiError, stores, the e2e mock harness + the
  ALREADY-REGISTERED mitm/proxy-pools/tunnels handlers/seeds). Disjoint from every
  other wave-6 plan (three unique routes, a unique ui/src/components/platform/ dir,
  three unique specs). Holds NO Go serial slot (zero new Go — see §1.2 /
  go-serial-slot).
branch: main (direct commits, per repo convention)
commit-prefix: phase-1/w6-m:
ref-source: 9router frozen @ 827e5c3 —
  src/app/(dashboard)/dashboard/mitm/MitmPageClient.js (MITM proxy config: status
  panel + CA-cert download + per-tool enable/disable; ref fetches /api/providers,
  /api/keys, /api/models/alias, /api/settings — those auxiliary reads are NOT ported,
  §1.3; the g0router surface is the in-tree mitm MOCK contract),
  src/app/(dashboard)/dashboard/proxy-pools/page.js (proxy pool management:
  list + create/edit modal + bulk/batch ops + per-pool test; ref hits
  /api/proxy-pools?includeUsage=true + POST /api/proxy-pools — REMAPPED to the
  in-tree proxy-pools mock paths, §1.4),
  src/app/(dashboard)/dashboard/endpoint/EndpointPageClient.js (tunnel status via
  setInterval REST POLL on /api/tunnel/status — NOT SSE, §1.5; the g0router /tunnels
  page is a g0router-grouping of the ref's endpoint-embedded tunnel controls into a
  dedicated page, consuming the in-tree /api/tunnels REST mock).
  For the list/table/modal/CRUD conventions also mirror the SHIPPED g0router
  routing-rules page (ui/src/routes/routing-rules.tsx — apiFetch+useState
  list/Toggle/edit-modal/ConfirmModal-delete, §1.1), exactly the pattern this plan
  follows.
base: <base> = git rev-parse HEAD recorded at P0 (expected 2978d2bc at authoring;
  if main advanced, record the actual SHA and substitute everywhere §5 says <base>)
freeze-exception: NONE. The header.tsx / __root.tsx / main.tsx exceptions are
  SPENT (w6-a/w6-b/w6-c). w6-m touches no frozen w6-a/w6-b/page-wave file.
go-serial-slot: NONE. w6-m adds ZERO Go (every platform domain is mock-only, §1.2).
  The routes_admin.go serial chain CLOSED on w6-j (MAP §Cross-cutting; the chain was
  w6-pre→w6-d→w6-e→w6-j). w6-m does NOT take it and NEVER touches
  internal/server/routes_admin.go. The MAP w6-m row already states the Go backends
  for mitm/proxy-pools/tunnels remain W7 — VERIFIED in §1.2: every domain's Go
  backend is ABSENT today; w6-m ships the UI halves against the registered mocks.
new-route: NO. All three routes (`/mitm`, `/proxy-pools`, `/tunnels`) ALREADY exist
  as registered stubs (§1.1); rewriting their component bodies does NOT change the
  route tree. `ui/src/routeTree.gen.ts` is UNCHANGED by w6-m (§1.7) — w6-m is a
  same-route-rewrite plan (like w6-f/j/k), NOT a new-route plan (w6-l is the only
  wave-2 new-route plan).
disposition: PARTIAL (binding). This is THE designated PARTIAL plan of wave 6. Unlike
  the variant-HAVE page plans (w6-k/w6-l), w6-m flips its seven matrix rows to
  **PARTIAL** (UI half done, mock-backed specs green; Go backends are explicit W7
  follow-ups), NOT HAVE. The MAP w6-m row is explicit: "Go backends for ALL of these
  remain W7 — rows … flip PARTIAL in W6 (UI half done, mock-backed specs green), HAVE
  in W7 when Go backends land." See §1.2 / §1.6 / §8.
```

---

## 1. Scope — PAR rows

### Rows this plan flips to PARTIAL

| Row | Claim | Target state after w6-m |
|---|---|---|
| PAR-UI-013 | Route `/mitm` MITM proxy config page (g0router route; ref `/dashboard/mitm`) | **PARTIAL** (UI half: REWRITE stub → status panel + CA-cert download + per-tool toggle against the `/api/mitm/*` MOCK; NO Go — §1.2 / §1.3 / §8 ESC-1a; HAVE in W7 when Go lands) |
| PAR-UI-019 | Route `/proxy-pools` proxy pool management with bulk ops (ref `/dashboard/proxy-pools`) | **PARTIAL** (UI half: REWRITE stub → list + create/edit modal + batch + per-pool test against the `/api/proxy-pools*` MOCK; NO Go — §1.2 / §1.4 / §8 ESC-1b; HAVE in W7) |
| PAR-UI-104 | API endpoint: `GET /api/proxy-pools` list pools (consumed by `/proxy-pools`) | **PARTIAL** (UI consumes the registered `GET /api/proxy-pools` MOCK; NO Go endpoint — §1.2 / §8 ESC-1b; HAVE in W7) |
| PAR-UI-105 | API endpoint: `POST /api/proxy-pools` create pool | **PARTIAL** (UI consumes the registered `POST /api/proxy-pools` MOCK; NO Go endpoint — §1.2 / §8 ESC-1b; HAVE in W7) |
| PAR-UI-112 | API endpoint: tunnel STATUS (ref `GET /api/tunnel/status`) | **PARTIAL** (UI consumes the registered `GET /api/tunnels` + `/api/tunnels/health` MOCK — path REMAPPED to the in-tree g0router contract, §1.5; NO Go — §1.2 / §8 ESC-1c; HAVE in W7) |
| PAR-UI-113 | API endpoint: ENABLE/DISABLE Cloudflare tunnel (ref `POST /api/tunnel/enable|disable`) | **PARTIAL** (UI consumes `POST /api/tunnels/cloudflare` enable / `DELETE /api/tunnels/cloudflare` disable MOCK — REMAPPED, §1.5; NO Go — §1.2 / §8 ESC-1c; HAVE in W7) |
| PAR-UI-114 | API endpoint: ENABLE/DISABLE Tailscale (ref `POST /api/tunnel/tailscale-enable|disable`) | **PARTIAL** (UI consumes `POST /api/tunnels/tailscale` enable / `DELETE /api/tunnels/tailscale` disable MOCK — REMAPPED, §1.5; NO Go — §1.2 / §8 ESC-1c; HAVE in W7) + the `/tunnels` page itself (g0router-grouped, §1.5) |

7 row-entries: PAR-UI-013 (`/mitm`) + PAR-UI-019 (`/proxy-pools`) + PAR-UI-104/105
(proxy-pool list/create APIs) + PAR-UI-112/113/114 (tunnel status/enable/disable
APIs) + the `/tunnels` page. Matches WAVE-6-MAP w6-m row (~line 139: scope "mitm
page, proxy-pools page, tunnels page; UI wires up to mock-backed e2e for all API
contracts (PAR-UI-104/105 proxy-pool list/create, PAR-UI-112..114 tunnel
status/enable/disable); Go backends for ALL of these remain W7 — rows
PAR-UI-013/019/104/105/112/113/114 flip PARTIAL in W6 … HAVE in W7"; rows
"PAR-UI-013→PARTIAL, PAR-UI-019→PARTIAL, PAR-UI-104→PARTIAL, PAR-UI-105→PARTIAL,
PAR-UI-112→PARTIAL, PAR-UI-113→PARTIAL, PAR-UI-114→PARTIAL + tunnels page") and
§Ownership w6-m (~line 202-204: `ui/src/routes/{mitm,proxy-pools,tunnels}.tsx`,
`ui/src/components/platform/**`, `ui/e2e/{mitm,proxy-pools,tunnels}.spec.ts` +
mocks, "No Go").

> **Matrix-row note (binding).** PAR-UI-013, PAR-UI-019, PAR-UI-104, PAR-UI-105,
> PAR-UI-112, PAR-UI-113, PAR-UI-114 are all currently **MISSING** in
> `.planning/parity/matrix/9router-ui.md` (lines 24, 30, 115, 116, 123, 124, 125 —
> verified at authoring) and are **w6-m-exclusive** (no sibling wave-6 plan owns
> them). w6-m flips each MISSING → **PARTIAL** (NOT HAVE) with the variant note + the
> W7 Go follow-up citation (§4 T5). This is the ONE wave-6 plan that lands rows as
> PARTIAL by design — every other page plan lands HAVE/variant-HAVE.

### 1.1 Preconditions already satisfied by merged waves (evidence)

- Route STUBS exist, must be REWRITTEN (not created — already registered, so
  `routeTree.gen.ts` does NOT change; MAP decision 6 / §1.7). All three render only
  an `<h1>`:
  `ui/src/routes/mitm.tsx:1-9` (`createFileRoute("/mitm")`, `<h1>MITM</h1>`),
  `ui/src/routes/proxy-pools.tsx:1-9` (`createFileRoute("/proxy-pools")`,
  `<h1>Proxy Pools</h1>`),
  `ui/src/routes/tunnels.tsx:1-9` (`createFileRoute("/tunnels")`,
  `<h1>Tunnels</h1>`). All three are ALREADY in the route tree:
  `TunnelsRoute`/`'/tunnels'` (`routeTree.gen.ts:14,57-59`),
  `ProxyPoolsRoute`/`'/proxy-pools'` (`routeTree.gen.ts:22,97-99`),
  `MitmRoute`/`'/mitm'` (`routeTree.gen.ts:28,127`). **Rewriting their component
  bodies does NOT change the tree (§1.7); w6-m is NOT a new-route plan.**
- Frozen primitives this plan CONSUMES (w6-b, never edited; 16 present): `Button`
  `ui/src/components/ui/button.tsx`; `Input` `ui/src/components/ui/input.tsx`;
  `Select` `ui/src/components/ui/select.tsx`; `Card`/`CardHeader`/`CardTitle`/
  `CardContent` `ui/src/components/ui/card.tsx`; `Modal`
  `ui/src/components/ui/modal.tsx` (controlled `open`/`onClose`, traffic lights,
  Escape, overlay, scroll-lock — the proxy-pool form-modal base); `ConfirmModal`
  `ui/src/components/ui/confirm-modal.tsx`; `Badge` `ui/src/components/ui/badge.tsx`
  (status/health badges); `Toggle` `ui/src/components/ui/toggle.tsx` (mitm enable,
  tool toggles, tunnel enable, pool is_active); `Loading`/`Spinner`/`Skeleton`/
  `CardSkeleton` `ui/src/components/ui/{loading,skeleton}.tsx`; `Tooltip`
  `ui/src/components/ui/tooltip.tsx`. (`SegmentedControl`/`Pagination`/`ProviderIcon`
  available but likely unused by this cluster.)
- Frozen foundation this plan CONSUMES (w6-a, never edited): `apiFetch`
  `ui/src/lib/api.ts:19` + `ApiError` `ui/src/lib/api.ts:3`; toast via
  `useNotificationStore.push` `ui/src/stores/notification.ts`; Material Symbols
  `ui/src/index.css:3` (status icons + the CA-cert download glyph).
- Shipped-page pattern this plan MIRRORS (consume-as-template, never edited): the
  w6-h routing-rules page `ui/src/routes/routing-rules.tsx` — `apiFetch<T[]>` in a
  `useCallback` `load`, `useState` list/loading/editing/creating/deleting,
  `CardSkeleton` while loading, empty-state copy, list rows with
  `data-testid="…-row"`, a `Toggle` for active-state with optimistic update +
  reload-on-failure, Edit→modal, Delete→`ConfirmModal`, `onSaved={load}`. The
  `/proxy-pools` page follows this shape exactly; `/mitm` + `/tunnels` are
  status-panel + toggle variants of it (no TanStack Query; plain apiFetch+useState).
- UI types this plan CONSUMES (`ui/src/lib/types.ts`, never edited):
  `MitmTool` (`types.ts:151-157`, `{id:string, name, enabled:boolean,
  dns_override:string, status:"active"|"inactive"}`);
  `ProxyPool` (`types.ts:207-217`, `{id:string, name, protocol, host, port:number,
  username, is_active:boolean, last_check_at, last_check_status}`);
  `Tunnel` (`types.ts:284-289`, `{type:string, is_enabled:boolean, url, status}`).
  The mitm `/api/mitm/status` response is `{enabled:boolean, tools:MitmTool[]}`
  (handler `mitm.ts:7`); the page also reads `/api/mitm/ca-cert` (PEM text).
- **No new dependency needed (VERIFIED):** every w6-m surface is status-panel/list/
  table/modal/toggle/cert-download — built from frozen primitives only. NO charting,
  NO DnD, NO editor, **NO SSE/EventSource** (tunnels is REST-poll, §1.5). (If any
  surface unexpectedly needed a new dep it would be an ESCALATION, §8; it is not.)
- **e2e mock harness present + ALREADY REGISTERED (CONSUME-ONLY — like w6-l, NOT
  like w6-i; §1.4):** all three handlers AND seeds are in-tree AND wired:
  - `ui/e2e/mocks/handlers/mitm.ts` (`registerMitmHandlers`, serving
    `GET /api/mitm/status` → `{enabled,tools}`, `POST /api/mitm/toggle`,
    `GET /api/mitm/ca-cert` → PEM, `POST /api/mitm/tools/{id}` toggle —
    `mitm.ts:6-39`).
  - `ui/e2e/mocks/handlers/proxy-pools.ts` (`registerProxyPoolsHandlers`, serving
    `GET/POST /api/proxy-pools`, `POST /api/proxy-pools/batch`,
    `GET/PUT/DELETE /api/proxy-pools/{id}`, `POST /api/proxy-pools/{id}/test` —
    `proxy-pools.ts:6-58`).
  - `ui/e2e/mocks/handlers/tunnels.ts` (`registerTunnelsHandlers`, serving
    `GET /api/tunnels` → `Tunnel[]`, `GET /api/tunnels/health` → `{healthy}`,
    `POST /api/tunnels/{type}` enable, `DELETE /api/tunnels/{type}` disable —
    `tunnels.ts:6-39`).
  - All three registered in `ui/e2e/mocks/handlers/index.ts`:
    `registerTunnelsHandlers` (`index.ts:19` import, `index.ts:54` call),
    `registerProxyPoolsHandlers` (`index.ts:31` import, `index.ts:66` call),
    `registerMitmHandlers` (`index.ts:32` import, `index.ts:67` call).
  - Seeds present + exported: `seed/tunnels.ts` (`seedTunnels` → 2 tunnels:
    cloudflare/tailscale, `seed/index.ts:12`), `seed/proxy-pools.ts`
    (`seedProxyPools` → 1 pool "US East", `seed/index.ts:22`), `seed/mitm.ts`
    (`seedMitmStatus` → `{enabled:false, ca_cert, tools:[2]}`, `seed/index.ts:23`).
  - Store fields seeded: `tunnels` (`store.ts:95`, seeded keyed by `type`
    `store.ts:192`), `proxyPools` (`store.ts:114`, seeded keyed by `id`
    `store.ts:213`), `mitmEnabled`/`mitmTools` (`store.ts:115-116`, seeded
    `store.ts:214-216`).
  - **CONSEQUENCE (binding):** w6-m needs ZERO `handlers/index.ts` edit, ZERO
    `seed/index.ts` edit, ZERO `store.ts` edit — every mock route + seed it needs is
    already registered. Like w6-l, w6-m is purely a CONSUMER of the mock layer; the
    sanctioned-index-edit exception that w6-i used is NOT INVOKED here.
- Existing acceptance specs (the contract — thin smoke only):
  `ui/e2e/mitm.spec.ts:9-12` (1 test: `/mitm` body contains "MITM"),
  `ui/e2e/proxy-pools.spec.ts:9-12` (1 test: `/proxy-pools` body contains
  "Proxy Pools"), `ui/e2e/tunnels.spec.ts:9-12` (1 test: `/tunnels` body contains
  "Tunnels"). Each `beforeEach` logs in via `ui/e2e/helpers.ts:3` (`#username`/
  `#password`, admin/123456). w6-m EXTENDS these three (RED first, §3/§4) — NO new
  spec files (all three already exist).

### 1.2 Go contract for mitm + proxy-pools + tunnels (file:line evidence — w6-m adds ZERO Go)

**VERIFICATION RESULT (the prompt's required per-domain check).** The MAP w6-m row
already states the Go backends remain W7. This was VERIFIED: every platform domain's
Go backend is ABSENT today. Authoritative evidence:

- `internal/server/routes_admin.go` registers NO mitm/proxy-pools/tunnels routes
  (`grep -nE 'mitm|proxy-pool|proxy_pool|ProxyPool|tunnel|Tunnel|Mitm|MITM'
  internal/server/routes_admin.go` → EMPTY).
- `grep -rniE '"/api/mitm|"/api/proxy-pools|"/api/tunnels|MitmHandler|
  ProxyPoolHandler|TunnelHandler|api/tunnels|api/mitm' internal/ cmd/` (excluding
  `_test`) → **ZERO matches.** No store layer, no admin handler, no route for any of
  the three.
- `internal/admin/` has NO `mitm.go`/`proxy_pools.go`/`tunnels.go`
  (`ls internal/admin/ | grep -iE 'mitm|proxy|tunnel'` → EMPTY).
- The ONLY in-tree tunnel reference is `internal/server/guard.go:135-141`, a
  settings-driven `tunnelDashboardAccess`/`tunnelUrl`/`tailscaleHost` host-access
  guard (reads `settings[...]` to decide dashboard access from a tunnel host) — a
  forward-looking guard, NOT an admin CRUD route for tunnel status/enable/disable
  (the same class as w6-l's `guard.go` MCP `LOCAL_ONLY` entry: a guard with no live
  CRUD route behind it). w6-m NEVER touches `guard.go`.

**Per-domain backend verification table:**

| Domain | Real Go endpoint? | Evidence | Disposition |
|---|---|---|---|
| mitm | **ABSENT** | no `/api/mitm/*` in routes_admin.go; grep ZERO; no `internal/admin/mitm.go` | **PARTIAL** vs the `/api/mitm/{status,toggle,ca-cert,tools/{id}}` MOCK; W7 Go follow-up (§8 ESC-1a) |
| proxy-pools | **ABSENT** | no `/api/proxy-pools*`; grep ZERO; no `internal/admin/proxy_pools.go` | **PARTIAL** vs the `/api/proxy-pools[/{id}][/batch][/test]` MOCK; W7 Go follow-up (§8 ESC-1b) |
| tunnels | **ABSENT** | no `/api/tunnels*`; grep ZERO; no `internal/admin/tunnels.go`; only `guard.go:135-141` settings-guard (no CRUD route) | **PARTIAL** vs the `/api/tunnels[/health][/{type}]` MOCK; W7 Go follow-up (§8 ESC-1c) |

**Binding consequence:** w6-m adds ZERO Go. ALL three pages + their API contracts
ship as **PARTIAL against the registered/consumed MOCK contract** (the mocks are the
binding capability contract for the UI half; the Go runtime is the explicit W7
follow-up). The PARTIAL disposition is itself the binding call (the MAP designates
w6-m as the PARTIAL plan) — unlike w6-k/w6-l which recorded variant-HAVE, w6-m
records **PARTIAL** so the matrix accurately reflects "UI done, backend pending W7."
The runtime Go gaps are recorded as W7 follow-ups in §8 and appended to
`.planning/parity/plans/open-questions.md`.

Envelope: all platform mock handlers use the `json`/`error` helpers
(`ui/e2e/mocks/handlers/utils.ts`) which mirror the real admin
`writeData`/`writeError` (`internal/admin/respond.go`) → snake_case `{data}` /
`{error:{message}}`. apiFetch (`ui/src/lib/api.ts:19`) unwraps `{data}`. **Caveat
(binding):** the mitm `/api/mitm/ca-cert` handler returns a RAW PEM body
(`Content-Type: application/x-pem-file`), NOT a `{data}` envelope (`mitm.ts:18-24`).
The CA-cert download therefore does NOT go through `apiFetch` (which expects
`{data}`); the page fetches the cert via a plain `fetch`/anchor download or reads the
raw text directly (§1.3 point 3). All OTHER reads go through `apiFetch`.

### 1.3 The MITM page — status panel + CA-cert + per-tool toggle (binding decision)

**The reference.** 9router's `MitmPageClient.js` renders a MITM proxy config surface:
a running/cert status panel, a CA-certificate download, and a per-tool list (DNS
override hosts) with enable/disable. The ref also fetches `/api/providers`,
`/api/keys`, `/api/models/alias`, `/api/settings` for auxiliary context
(`MitmPageClient.js:24-56`).

**The g0router constraints (evidence):**
- The g0router mitm surface is the registered `/api/mitm/*` MOCK
  (`handlers/mitm.ts`): `GET /api/mitm/status` → `{enabled:boolean, tools:MitmTool[]}`
  (`mitm.ts:7`); `POST /api/mitm/toggle` flips global enable (`mitm.ts:11-15`);
  `GET /api/mitm/ca-cert` → raw PEM (`mitm.ts:18-24`); `POST /api/mitm/tools/{id}`
  flips a single tool's `enabled`/`status` (`mitm.ts:26-37`). Seed: `enabled:false`,
  2 tools (`Request Inspector` active, `Response Modifier` inactive),
  `seedMitmStatus()` (`seed/mitm.ts:3-14`).
- The ref's auxiliary `/api/providers|keys|models/alias|settings` reads are NOT
  ported — they belong to other pages (w6-e/w6-f/w6-j) and are out of w6-m scope. The
  g0router mitm page consumes ONLY the `/api/mitm/*` mock contract.

**Decision (binding):**
1. **REWRITE `ui/src/routes/mitm.tsx`** → `apiFetch<{enabled:boolean,
   tools:MitmTool[]}>("/api/mitm/status")` on mount; render a status panel (global
   enable `Toggle` POSTing `/api/mitm/toggle`, optimistic + reload-on-fail) + the
   tool list (each row: name, dns_override, status `Badge`, enable `Toggle` POSTing
   `/api/mitm/tools/{id}`). Body contains "MITM".
2. **No POST/PUT/DELETE create for tools** — the mock exposes only status + toggles
   (no tool CRUD), so the page is a status+toggle surface, not a CRUD list. Accepted
   (PARTIAL, §1.6).
3. **CA-cert download** — the page renders a "Download CA certificate" control that
   GETs `/api/mitm/ca-cert`. Because that route returns a RAW PEM body (NOT a `{data}`
   envelope, §1.2 caveat), the download uses a plain `fetch`/anchor (NOT `apiFetch`).
   The e2e asserts the control is present (a click triggering a download is finicky
   under Playwright — the binding e2e proof is that the control RENDERS + the status/
   tool surface works; the cert fetch path is covered by a unit on the pure
   download-href/blob helper if extracted, §3).
4. **No Go.** `/api/mitm/*` has no backend (§1.2); PAR-UI-013 ships **PARTIAL** vs the
   registered `/api/mitm/*` mock; the real Go mitm endpoints are a W7 follow-up
   (§8 ESC-1a).

### 1.4 The proxy-pools page — list + create/edit + batch + test (binding decision)

**The reference.** 9router's `proxy-pools/page.js` is a full proxy-pool manager:
fetch `/api/proxy-pools?includeUsage=true` (`page.js:71`), a create/edit modal POSTing
`/api/proxy-pools` (`page.js:122`) / PUTting `/api/proxy-pools/{id}`, bulk/batch
import, and per-pool connectivity test.

**The g0router constraints (evidence):**
- The g0router proxy-pools surface is the registered `/api/proxy-pools*` MOCK
  (`handlers/proxy-pools.ts`): `GET /api/proxy-pools` → `ProxyPool[]`
  (`proxy-pools.ts:8`); `POST /api/proxy-pools` create (fabricates `{id:nextId,
  last_check_at, ...body}`, `proxy-pools.ts:10-15`); `POST /api/proxy-pools/batch`
  (bulk create from `body.items`, `proxy-pools.ts:18-27`);
  `GET/PUT/DELETE /api/proxy-pools/{id}` (`proxy-pools.ts:29-47`);
  `POST /api/proxy-pools/{id}/test` → `{ok, latency_ms}` (`proxy-pools.ts:49-51`).
  Seed: 1 pool "US East" (`seedProxyPools()`, `seed/proxy-pools.ts:3-19`).
- The ref's `?includeUsage=true` query param is NOT served by the mock (the mock
  ignores query and returns all pools) — the page reads plain `/api/proxy-pools`
  (the in-tree contract). The `vercel/deno/cloudflare-deploy` ref sub-routes are NOT
  ported (deploy automation is out of w6-m scope; the mock has no equivalent).

**Decision (binding):**
1. **REWRITE `ui/src/routes/proxy-pools.tsx`** → `apiFetch<ProxyPool[]>
   ("/api/proxy-pools")` (the routing-rules.tsx template, §1.1); list rows (name,
   protocol/host:port, is_active `Toggle`, last_check_status `Badge`); New/Edit via
   `<ProxyPoolFormModal>` (POST `/api/proxy-pools` new / PUT `/api/proxy-pools/{id}`
   edit); per-pool "Test" → POST `/api/proxy-pools/{id}/test` (render `{ok,
   latency_ms}`); Delete via `ConfirmModal` (DELETE `/api/proxy-pools/{id}`). Body
   contains "Proxy Pools".
2. **Batch import (optional within-page)** — the mock serves `POST /api/proxy-pools/
   batch`; the page MAY surface a bulk-add control. The binding e2e proof is the
   list + create-modal + test + delete; batch is a nice-to-have whose presence is
   asserted only if implemented (the mock supports it; no extra mock work).
3. **The form-payload mapping is the pure/unit-tested seam:** extract the
   create/edit payload builder into a PURE helper
   `ui/src/lib/proxy-pool-form.ts` (e.g. `toProxyPoolPayload(form): ProxyPoolCreate`),
   unit-tested without a DOM (the w6-h `combo-order.ts` / w6-l `mcp-install.ts`
   precedent). The modal wiring is e2e-proven; the payload logic is the unit.
4. **No Go.** `/api/proxy-pools*` has no backend (§1.2); PAR-UI-019/104/105 ship
   **PARTIAL** vs the registered mocks; the real Go proxy-pool endpoints are a W7
   follow-up (§8 ESC-1b).

### 1.5 The tunnels page — REST-poll status (NO SSE) + enable/disable (binding decision + the live-status harness call)

**The reference.** In 9router the tunnel controls live INSIDE the endpoint page
(`EndpointPageClient.js`): tunnel STATUS is fetched via `fetch("/api/tunnel/status",
{cache:"no-store"})` on a `setInterval` POLL (`EndpointPageClient.js:184,244,
265-267`), and enable/disable hit `/api/tunnel/enable|disable` +
`/api/tunnel/tailscale-enable|disable`. **The ref uses REST polling, NOT SSE/
EventSource** — verified: `grep -nE '/api/tunnel|EventSource|setInterval'
EndpointPageClient.js` shows `setInterval` poll loops and `fetch` calls, ZERO
`EventSource`.

**The g0router constraints (evidence):**
- The g0router tunnels surface is the registered `/api/tunnels*` MOCK
  (`handlers/tunnels.ts`): `GET /api/tunnels` → `Tunnel[]` (`tunnels.ts:7`);
  `GET /api/tunnels/health` → `{healthy:boolean}` (`tunnels.ts:11`);
  `POST /api/tunnels/{type}` enable (sets `is_enabled:true, status:"active"`,
  `tunnels.ts:18-21`); `DELETE /api/tunnels/{type}` disable (`tunnels.ts:22-25`),
  where `{type}` is `cloudflare` | `tailscale`. Seed: 2 tunnels
  (cloudflare/tailscale, both `is_enabled:false, status:"inactive"`, `seedTunnels()`
  `seed/tunnels.ts:3-9`).
- The ref's `/api/tunnel/*` paths (singular, `enable`/`disable`/`tailscale-enable`)
  DIVERGE from the in-tree mock `/api/tunnels/{type}` (plural, type-parameterized).
  **The page consumes the in-tree g0router mock paths (the binding contract), NOT the
  ref paths** (path REMAP, like w6-l's marketplace remap, §1.5 there).

**Tunnels live-status harness decision (RESOLVED — the prompt's required call):**
- **The tunnels page is REST-POLL, NOT SSE.** Both signals agree: (a) the 9router ref
  uses `setInterval` REST polling on `/api/tunnel/status`, never `EventSource`; (b) the
  in-tree mock serves `GET /api/tunnels` + `GET /api/tunnels/health` as plain REST —
  there is NO `/api/tunnels/stream` mock route. **VERIFIED:** the e2e
  `MockEventSource` (`fixture.ts:35-111`) has branches ONLY for `/api/traffic/stream`
  (`fixture.ts:60`) and `/api/console-logs/stream` (`fixture.ts:78`) — there is NO
  tunnels-stream branch, and none is needed. The page reads tunnel status on mount via
  `apiFetch<Tunnel[]>("/api/tunnels")` + an OPTIONAL `apiFetch<{healthy:boolean}>
  ("/api/tunnels/health")`, and MAY refresh on interaction (after enable/disable) or a
  light `setInterval` re-fetch. **NO SSE, NO `EventSource`, NO `fixture.ts` edit.**
  This is the established harness strategy applied (w6-g §1.3 / w6-i §1.5 said: if a
  surface is REST not SSE, keep it REST-deterministic and do NOT touch `fixture.ts`) —
  here the surface IS REST by both ref + mock, so the SSE branch question is moot and
  resolved REST-poll. (If a future design adds a real `/api/tunnels/stream`, that is a
  W7 follow-up requiring a sanctioned `fixture.ts` branch — NOT w6-m.)

**Decision (binding):**
1. **REWRITE `ui/src/routes/tunnels.tsx`** → `apiFetch<Tunnel[]>("/api/tunnels")` on
   mount (REST-poll, §above); render one card/row per tunnel `type` (cloudflare /
   tailscale) showing `url`, `status` `Badge`, and an enable/disable `Toggle`:
   enabling POSTs `/api/tunnels/{type}`, disabling DELETEs `/api/tunnels/{type}`
   (optimistic + reload-on-fail); an optional health indicator from
   `/api/tunnels/health`. Body contains "Tunnels". This is the **g0router-grouped
   tunnels page** (the ref kept tunnel controls inside endpoint; g0router gives them a
   dedicated `/tunnels` route — the MAP/`tunnels.spec.ts` mandate it).
2. **No POST create / no tunnel deletion-of-record** — the mock enable/disable only
   flips `is_enabled`/`status` on the two seeded tunnels (no add/remove tunnel
   records), so the page is an enable/disable surface over a fixed cloudflare/tailscale
   pair. Accepted (PARTIAL, §1.6).
3. **No Go.** `/api/tunnels*` has no backend (§1.2; only the `guard.go:135-141`
   settings-guard, not a CRUD route); PAR-UI-112/113/114 + the `/tunnels` page ship
   **PARTIAL** vs the registered mocks; the real Go tunnel status/enable/disable
   endpoints are a W7 follow-up (§8 ESC-1c).

### 1.6 PARTIAL disposition (recorded — the binding call for the WHOLE plan)

This plan is PARTIAL by design (the MAP designates w6-m as THE PARTIAL plan). Each
row lands PARTIAL, NOT HAVE:

- **PAR-UI-013 `/mitm`**: PARTIAL — status panel + CA-cert + per-tool toggle against
  the `/api/mitm/*` mock; NO Go backend (§1.2/§1.3). HAVE in W7 (§8 ESC-1a).
- **PAR-UI-019 `/proxy-pools`**: PARTIAL — list + create/edit + batch + test against
  the `/api/proxy-pools*` mock; NO Go backend (§1.2/§1.4). HAVE in W7 (§8 ESC-1b).
- **PAR-UI-104/105 (proxy-pool list/create APIs)**: PARTIAL — UI consumes the
  registered `GET/POST /api/proxy-pools` mocks; NO Go endpoints (§1.2). HAVE in W7.
- **PAR-UI-112/113/114 (tunnel status/enable/disable APIs) + `/tunnels` page**:
  PARTIAL — UI consumes `GET /api/tunnels` (+`/health`), `POST/DELETE
  /api/tunnels/{type}` mocks (REST-poll, REMAPPED paths); NO Go endpoints
  (§1.2/§1.5). HAVE in W7 (§8 ESC-1c).
- **Why PARTIAL not HAVE:** w6-k/w6-l recorded *variant-HAVE* because the MAP claimed
  their backends existed and verification merely corrected the runtime gap to a serial
  follow-up; w6-m's MAP row EXPLICITLY pre-declares the Go backends as W7 work, so the
  honest matrix state is PARTIAL (the half that exists — the UI — is done; the other
  half — Go — is a tracked W7 follow-up). The mock-backed specs are GREEN; the UI is
  fully functional against the mocks.
- **Mock-only surfaces** (all of them) → PARTIAL by definition (this whole plan is
  the PARTIAL plan).

### 1.7 `routeTree.gen.ts` is NOT touched

All three routes already exist as registered stubs (§1.1); rewriting their component
bodies does not change the route tree, and no new route file is added (the form
modal, the platform components are in-page components, not routes). Therefore
`ui/src/routeTree.gen.ts` is UNCHANGED by w6-m (MAP decision 6; w6-m is NOT a
new-route-file plan — w6-l is the only wave-2 one). If a build incidentally reformats
it, that is an ESCALATION (§8), not an in-plan edit. This is the diff-gate difference
from w6-l (where routeTree.gen.ts IS allowed) — for w6-m it must be UNCHANGED (like
w6-f/j/k).

### Variant / PARTIAL notes (recorded rationale)

- **All three routes (PAR-UI-013/019 + `/tunnels`)**: REWRITE existing stubs;
  status/list/toggle + modal against the registered mocks; NO Go for any (§1.2).
  Recorded **PARTIAL**; Go deferred to W7 (§8 ESC-1a/1b/1c).
- **PAR-UI-104/105/112/113/114 (API contracts)**: served by the registered e2e
  mocks; NO Go. Recorded **PARTIAL** (§8).
- **Tunnel paths REMAPPED**: ref `/api/tunnel/{enable,disable,tailscale-*}` →
  in-tree `/api/tunnels/{type}` POST/DELETE (§1.5). Accepted constraint.
- **Tunnels live-status = REST-poll, NOT SSE** (§1.5): ref + mock both REST; no
  `fixture.ts` SSE branch exists or is needed. Accepted, resolved.
- **mitm CA-cert is raw PEM (not `{data}`)** (§1.2 caveat): the download uses plain
  `fetch`/anchor, not `apiFetch`. Accepted constraint.
- **mitm tools are toggle-only; tunnels enable/disable-only**: the mocks expose no
  tool-CRUD / no tunnel add-remove, so those surfaces are status+toggle, not full
  CRUD. Accepted, not a gap.
- **Pages render inside app chrome**: `__root.tsx` wraps every route in
  Sidebar+Header; pages render in `<Outlet>`; specs assert page content with chrome
  present (w6-c/w6-e/w6-k/w6-l precedent). Accepted constraint, not a gap.
- **Data layer = plain `apiFetch` + React state, NOT TanStack Query**: `QueryClient`
  is NOT mounted (`__root.tsx`/`main.tsx` FROZEN, w6-a; verified
  `grep -rn QueryClientProvider ui/src/routes/__root.tsx ui/src/main.tsx` → EMPTY).
  PAR-UI-081 already HAVE from w6-a (`open-questions.md:6`); w6-m consumes `apiFetch`
  and does NOT mount a provider, NOT edit any frozen file. Accepted constraint.

### NOT in scope (explicit)

- **No Go changes.** ALL of `internal/` is FORBIDDEN. Every platform domain backend
  is ABSENT (§1.2) — those are W7 FOLLOW-UPS (§8), NEVER an in-plan Go edit (the MAP
  assigns no Go to w6-m and the serial slot closed on w6-j). No new
  `internal/admin/{mitm,proxy_pools,tunnels}.go`, no `routes_admin.go` platform
  routes, no `guard.go` edit.
- **No new route FILES** — only the three existing stubs are rewritten;
  `routeTree.gen.ts` untouched (§1.7). All form modals/panels are in-page components.
- **No dependency additions** — every surface uses frozen primitives only (§1.1).
  NO `package.json` / lockfile edit.
- **No edits to any frozen w6-a/w6-b/page-wave file** — no `__root.tsx`,
  `main.tsx`, layout components, `ui/src/components/ui/*`, `ui/src/stores/*`,
  `ui/src/lib/api.ts`, `ui/src/lib/utils.ts`, `ui/src/lib/auth.ts`,
  `ui/src/providers/*`, `ui/src/routes/{login,callback}.tsx`, nor any sibling
  page-plan's routes/components. No header exception remains (SPENT).
- **No `QueryClientProvider` mount** — plain `apiFetch`; PAR-UI-081 already HAVE
  (w6-a).
- **No mock-layer edits at all** (§1.1) — `ui/e2e/mocks/handlers/index.ts`,
  `ui/e2e/mocks/seed/index.ts`, `ui/e2e/mocks/store.ts`, `ui/e2e/mocks/fixture.ts`,
  and the mitm/proxy-pools/tunnels handler BODIES + seed files are CONSUMED unchanged
  (already registered). NO new mock handler, NO seed correction, NO `fixture.ts` SSE
  branch (tunnels is REST, §1.5).
- **No SSE/streaming/charts/DnD/editor** — all three surfaces are request/response
  status + list/toggle CRUD + a cert download + a test POST. Tunnels is REST-poll.
- **No new spec files** — only the three existing
  `{mitm,proxy-pools,tunnels}.spec.ts` are extended.
- **No providers/connections/models (w6-e), no keys/VK/endpoint (w6-f), no usage/
  quota/pricing (w6-g), no combos/routing (w6-h), no chat/console/translator (w6-i),
  no settings/version (w6-j), no governance pages (w6-k), no mcp/skills (w6-l).**
- **No real outbound network** — all reads are mock-intercepted.

---

## 2. Precondition checks

Run all before any edit; abort and report to orchestrator on any failure.

```bash
# P0 — clean tree; record <base>
git status --porcelain     # must be empty (untracked tooling artifacts must be
                           # gitignored; worker uses explicit `git add <file>`,
                           # never `git add -A`; ui/dist/** is gitignored — do not
                           # stage build artifacts)
git rev-parse HEAD         # record as <base> for §5 (expected 2978d2bc)

# P1 — w6-b primitives present and frozen (consumed)
ls ui/src/components/ui/*.tsx | grep -v test | wc -l    # = 16 (w6-b set intact)
grep -n "export { Modal }\|export function Modal" ui/src/components/ui/modal.tsx
grep -n "ConfirmModal" ui/src/components/ui/confirm-modal.tsx
grep -n "export function Toggle\|export { Toggle }" ui/src/components/ui/toggle.tsx
grep -n "Badge" ui/src/components/ui/badge.tsx

# P2 — w6-a foundation present and frozen (consumed; PAR-UI-081 already HAVE)
grep -n "export async function apiFetch" ui/src/lib/api.ts
grep -n "export class ApiError" ui/src/lib/api.ts
grep -n "push:" ui/src/stores/notification.ts
grep -rn "QueryClientProvider\|QueryClient" ui/src/routes/__root.tsx ui/src/main.tsx ; echo "^ expect EMPTY (not mounted; do NOT mount)"

# P3 — the shipped-page template is present (consume-as-template, never edit)
grep -n "apiFetch\|ConfirmModal\|createFileRoute" ui/src/routes/routing-rules.tsx

# P4 — the three route stubs are still bare (safe to rewrite); no new dirs yet
grep -n "<h1>MITM</h1>"        ui/src/routes/mitm.tsx
grep -n "<h1>Proxy Pools</h1>" ui/src/routes/proxy-pools.tsx
grep -n "<h1>Tunnels</h1>"     ui/src/routes/tunnels.tsx
test ! -d ui/src/components/platform && echo "platform components dir absent (good)"
test ! -e ui/src/lib/proxy-pool-form.ts && echo "proxy-pool-form helper absent (good)"
grep -nE "MitmRoute|ProxyPoolsRoute|TunnelsRoute|'/mitm'|'/proxy-pools'|'/tunnels'" ui/src/routeTree.gen.ts | head
echo "^ all three ALREADY registered (stubs, §1.1) — rewrites do NOT change the tree §1.7"

# P5 — e2e mock harness present + ALREADY REGISTERED (CONSUME unchanged; §1.1/§1.4)
grep -n "registerMitmHandlers\|registerProxyPoolsHandlers\|registerTunnelsHandlers" ui/e2e/mocks/handlers/index.ts
grep -n "seedMitmStatus\|seedProxyPools\|seedTunnels" ui/e2e/mocks/seed/index.ts
grep -n "mitmEnabled\|mitmTools\|proxyPools\|tunnels" ui/e2e/mocks/store.ts | head
grep -n "/api/mitm/status\|/api/mitm/toggle\|/api/mitm/ca-cert\|/api/mitm/tools" ui/e2e/mocks/handlers/mitm.ts
grep -n "/api/proxy-pools" ui/e2e/mocks/handlers/proxy-pools.ts
grep -n "/api/tunnels" ui/e2e/mocks/handlers/tunnels.ts

# P6 — tunnels live-status is REST, NOT SSE (§1.5): no tunnels-stream mock; fixture has no tunnels branch
grep -nE "tunnel.*stream|stream.*tunnel" ui/e2e/mocks/ -r ; echo "^ expect EMPTY (no /api/tunnels/stream — REST-poll §1.5)"
grep -nE "/api/traffic/stream|/api/console-logs/stream|tunnel" ui/e2e/mocks/fixture.ts ; echo "^ MockEventSource has ONLY traffic+console branches, NO tunnels (REST §1.5)"

# P7 — Go reality: ALL three platform domains ABSENT (§1.2/§8)
grep -nE 'mitm|proxy-pool|proxy_pool|ProxyPool|tunnel|Tunnel|Mitm|MITM' internal/server/routes_admin.go ; echo "^ expect EMPTY (no Go platform routes)"
grep -rniE '"/api/mitm|"/api/proxy-pools|"/api/tunnels|MitmHandler|ProxyPoolHandler|TunnelHandler' internal/ cmd/ | grep -v _test ; echo "^ expect EMPTY (no Go platform handlers/stores)"
ls internal/admin/ | grep -iE 'mitm|proxy|tunnel' ; echo "^ expect EMPTY (no admin platform handler)"
grep -nE 'tunnelDashboardAccess|tunnelUrl|tailscaleHost' internal/server/guard.go ; echo "^ ONLY guard.go:135-141 settings-guard (forward-looking, NOT a CRUD route; never edited)"

# P8 — routes_admin.go serial slot: w6-m does NOT take it (zero Go; chain closed on w6-j)
echo "w6-m adds ZERO Go → does NOT touch internal/server/routes_admin.go; the serial chain (w6-pre→w6-d→w6-e→w6-j) CLOSED on w6-j."

# P9 — harness green at base
cd ui && npx playwright test e2e/mitm.spec.ts e2e/proxy-pools.spec.ts e2e/tunnels.spec.ts
# Record base result: stubs render only <h1> (which already contains the asserted
# text), so the three text-contains smoke assertions PASS at base (the <h1> + sidebar
# chrome carry the page names). The RED arc is the ADDED list/status/toggle/modal
# assertions in §4 T1. Record exact pass/fail per spec in WORKFLOW.md.
cd ui && npm run build                               # exit 0
cd ui && npx vitest run src/                         # exit 0 (existing units green)
go test ./... && go vet ./...                        # exit 0 (Go untouched-green)
```

---

## 3. Exclusive file ownership

After w6-m merges, all CREATE files below are owned by w6-m; later plans consume,
never edit (MAP decision 7).

**CREATE — routes (REWRITE existing stubs; no new route files, §1.7):**

| File | Exports / contract |
|---|---|
| `ui/src/routes/mitm.tsx` (REWRITE) | `Route=createFileRoute("/mitm")`; `MitmPage`: `apiFetch<{enabled:boolean,tools:MitmTool[]}>("/api/mitm/status")` → status panel (global enable `Toggle` POSTing `/api/mitm/toggle`) + tool rows (name, dns_override, status `Badge`, enable `Toggle` POSTing `/api/mitm/tools/{id}`) + a "Download CA certificate" control GETting `/api/mitm/ca-cert` (raw PEM via plain `fetch`/anchor, §1.2/§1.3). Body contains "MITM". |
| `ui/src/routes/proxy-pools.tsx` (REWRITE) | `Route=createFileRoute("/proxy-pools")`; `ProxyPoolsPage`: `apiFetch<ProxyPool[]>("/api/proxy-pools")` → list rows (name, protocol/host:port, is_active `Toggle`, last_check_status `Badge`); New/Edit via `<ProxyPoolFormModal>` (POST `/api/proxy-pools` / PUT `/api/proxy-pools/{id}`); per-pool "Test" → POST `/api/proxy-pools/{id}/test` (render `{ok,latency_ms}`); Delete via `ConfirmModal` (DELETE `/api/proxy-pools/{id}`); optional batch via `POST /api/proxy-pools/batch`. Body contains "Proxy Pools". |
| `ui/src/routes/tunnels.tsx` (REWRITE) | `Route=createFileRoute("/tunnels")`; `TunnelsPage`: `apiFetch<Tunnel[]>("/api/tunnels")` (REST-poll, §1.5) + optional `apiFetch<{healthy:boolean}>("/api/tunnels/health")` → one card per tunnel `type` (cloudflare/tailscale) with url, status `Badge`, enable/disable `Toggle` (POST `/api/tunnels/{type}` enable / DELETE `/api/tunnels/{type}` disable). NO SSE, NO `EventSource`. Body contains "Tunnels". |

**CREATE — platform components (`ui/src/components/platform/`):**

| File | Exports / contract |
|---|---|
| `proxy-pool-form-modal.tsx` | `ProxyPoolFormModal` — `Modal`+`Input`/`Select`; fields name, protocol, host, port, username, is_active; save → POST `/api/proxy-pools` (new) / PUT `/api/proxy-pools/{id}` (edit) using the `proxy-pool-form` helper payload; `onSaved` reload. |
| `mitm-tool-row.tsx` | `MitmToolRow` (OPTIONAL extract) — renders one `MitmTool` row (name, dns_override, status `Badge`, enable `Toggle` POSTing `/api/mitm/tools/{id}`). MAY be inlined into `mitm.tsx`. |
| `tunnel-card.tsx` | `TunnelCard` (OPTIONAL extract) — renders one `Tunnel` (type, url, status `Badge`, enable/disable `Toggle`). MAY be inlined into `tunnels.tsx`. |

(The list-page components / form modals are DOM-heavy; their coverage is the e2e
render/open-modal/save/toggle assertions, not units — same disposition as
w6-e/w6-k/w6-l modal components.)

**CREATE — lib (pure helper, §1.4):**

| Export | Contract |
|---|---|
| `ui/src/lib/proxy-pool-form.ts` — `toProxyPoolPayload(form): ProxyPoolCreate` (and/or a `caCertDownloadHref(pem)` accessor if the mitm cert download is extracted) | Pure, deterministic, no DOM. The AUTHORITATIVE proxy-pool-create-contract proof (§1.4 point 3). |

**CREATE — unit tests (vitest — pure logic reachable without a live backend):**

| File | Contents |
|---|---|
| `ui/src/lib/proxy-pool-form.test.ts` | ≥3 tests: `toProxyPoolPayload` maps name/protocol/host/port/username/is_active from a form object; coerces `port` to number; omits/handles empty optional fields. (If the mitm cert-href helper is extracted, ≥1 test for it.) Committed RED before `proxy-pool-form.ts`. |

(If `proxy-pool-form.ts` stays inlined, move its assertions into a page-logic helper
test the executor extracts; the e2e create-modal spec remains the binding contract
regardless.)

**MODIFY — e2e specs (the acceptance contract):**

| File | Change |
|---|---|
| `ui/e2e/mitm.spec.ts` | KEEP the 1 existing test (body "MITM"). ADD RED: status panel renders (global enable `Toggle`); tool rows render from seed (≥2: "Request Inspector"/"Response Modifier", status visible); toggling a tool fires POST `/api/mitm/tools/{id}`; a "Download CA certificate" control is present. |
| `ui/e2e/proxy-pools.spec.ts` | KEEP the 1 existing test (body "Proxy Pools"). ADD RED: pool rows render from seed (≥1: "US East", host visible); open ProxyPoolFormModal (`[data-testid="modal-traffic-lights"]`); save fires POST `/api/proxy-pools`; per-pool "Test" fires POST `/api/proxy-pools/{id}/test`; delete via ConfirmModal fires DELETE. |
| `ui/e2e/tunnels.spec.ts` | KEEP the 1 existing test (body "Tunnels"). ADD RED: tunnel cards render from seed (≥2: cloudflare/tailscale, url + status visible); enabling a tunnel fires POST `/api/tunnels/{type}`; disabling fires DELETE `/api/tunnels/{type}`. (REST-only; NO SSE assertion, §1.5.) |

**CONSUME (NO edit) — mock layer:**

| File | Disposition |
|---|---|
| `ui/e2e/mocks/handlers/{mitm,proxy-pools,tunnels}.ts`, `ui/e2e/mocks/seed/{mitm,proxy-pools,tunnels}.ts` | CONSUME as-is (§1.1/§1.4). NO edit. Every route + seed is already registered. If a within-mock inconsistency breaks this cluster's specs, ESCALATE (§8 ESC-2) — never edit a shared/foundation mock, never add Go. |

**FORBIDDEN:** everything else. Explicitly: ALL of `internal/` (every platform
backend ABSENT — W7 FOLLOW-UPS §8, never an in-plan Go edit; w6-m holds NO serial
slot; NO `guard.go` edit); ALL of `ui/src/components/ui/*` (w6-b frozen);
`ui/src/stores/*`, `ui/src/lib/api.ts`, `ui/src/lib/utils.ts`,
`ui/src/lib/auth.ts`, `ui/src/providers/*` (w6-a frozen);
`ui/src/routes/__root.tsx`, `ui/src/main.tsx`, `ui/src/components/layout/*`,
`ui/src/routes/{login,callback}.tsx`, `ui/src/components/auth/*`; ALL sibling
page-plan routes/components (`ui/src/routes/{providers,connections,models,keys,
virtual-keys,endpoint,dashboard,usage,logs,traffic,quota,pricing,combos,
routing-rules,model-limits,aliases,chat,console,translator,settings,teams,audit,
feature-flags,guardrails,prompts,alerts,mcp,mcp.tools,skills}.tsx` and their
component dirs); `ui/package.json` + lockfile; `ui/vite.config.ts`;
`ui/playwright.config.ts`; `ui/components.json`; `ui/src/index.css`;
`ui/src/routeTree.gen.ts` (generated; UNCHANGED §1.7 — NOT the w6-l exception);
`ui/e2e/mocks/fixture.ts` (NO tunnels SSE branch — REST §1.5),
`ui/e2e/mocks/store.ts`, `ui/e2e/mocks/handlers/index.ts`,
`ui/e2e/mocks/seed/index.ts` (foundation/wiring — all three platform handlers/seeds
ALREADY registered); the mitm/proxy-pools/tunnels handler BODIES + seed files
(CONSUME unchanged — NO edit); all other `ui/e2e/*.spec.ts`; NO new spec files; NO
new mock files. `ui/dist/**` is gitignored — never stage it.

---

## 4. TDD tasks

Cadence (strict): **no route/component/lib file may exist (or be rewritten beyond its
stub) before the failing test that covers it is committed.** `cd ui && npm run build`
green at EVERY commit (test files + red specs are never imported by production code —
w6-b/w6-c/w6-e/w6-k/w6-l rationale). `go test ./... && go vet ./...` stays
untouched-green at EVERY commit (ZERO new Go). The three e2e specs stay RED (on the
ADDED assertions) from T1 until the implementation tasks green them; that is the arc.

### T1 — STEP(a): extend all three e2e specs (commit RED)

Add the RED tests in §3 to `mitm/proxy-pools/tunnels.spec.ts` (names are the
acceptance contract, §5). CONSUME all three mock handlers/seeds unchanged (§1.1/§1.4)
— NO mock edit, NO registration edit (already wired), NO `fixture.ts` SSE branch
(tunnels is REST, §1.5).

STEP(b): run all three specs — **record failure output** (no status panel, no tool
rows, no pool rows/modal, no tunnel cards/toggles). Commit RED:
`phase-1/w6-m: failing mitm/proxy-pools/tunnels e2e (TDD red)`.

**Mock-vs-reality gate**: re-confirm the §1.2 Go reality (ALL three domains ABSENT).
These are W7 FOLLOW-UPS (§8), NOT in-plan Go. CONSUME the mocks as the capability
contract; w6-m makes NO mock-side edit. If a within-mock inconsistency breaks a spec,
ESCALATE (§8 ESC-2) rather than editing a shared/foundation mock; NEVER add Go,
NEVER edit `index.ts`/`store.ts`/`fixture.ts`/`seed/index.ts`.

### T2 — STEP(a): unit test for proxy-pool-form (commit RED)

Write `ui/src/lib/proxy-pool-form.test.ts` (§3, §1.4 — the create-contract proof).
Pure logic; no DOM (the w6-h `combo-order.ts` / w6-l `mcp-install.ts` precedent). Run
`cd ui && npx vitest run src/lib/proxy-pool-form.test.ts` → FAILS (module missing).
Record failure. Commit RED:
`phase-1/w6-m: failing unit test for proxy-pool-form (TDD red)`.

### T3 — STEP(b): proxy-pools page + form modal + helper

Implement `ui/src/lib/proxy-pool-form.ts` (greens its unit), `proxy-pool-form-modal.tsx`;
rewrite `proxy-pools.tsx` (list + modal CRUD + test + delete). Gates:
`npx vitest run src/lib/proxy-pool-form.test.ts` green; `proxy-pools.spec.ts` green
(rows, create-modal, test, delete); `npm run build` green; `go test ./... && go vet
./...` untouched green. Commit:
`phase-1/w6-m: proxy-pools page (list/create/test/delete) + form modal + helper`.

### T4 — STEP(b): mitm page + tunnels page

Rewrite `mitm.tsx` (status panel + tool toggles + CA-cert download) and `tunnels.tsx`
(REST-poll tunnel cards + enable/disable toggles). Optionally extract
`mitm-tool-row.tsx` / `tunnel-card.tsx`. Gates: `mitm.spec.ts`, `tunnels.spec.ts`
green; all three specs green; `npx vitest run src/` green; `npm run build` green;
`go test ./... && go vet ./...` untouched green. Commit:
`phase-1/w6-m: mitm page (status/tools/ca-cert) + tunnels page (rest-poll status/enable/disable)`.

### T5 — full gates + closeout

```bash
cd ui && npm run build
cd ui && npx playwright test e2e/mitm.spec.ts e2e/proxy-pools.spec.ts e2e/tunnels.spec.ts   # all green
cd ui && npx playwright test                             # full suite: no spec green-at-base may be red
cd ui && npx vitest run src/                             # all green incl new platform unit
go test ./... && go vet ./...                            # untouched-green (ZERO new Go)
grep -nE "MitmRoute|ProxyPoolsRoute|TunnelsRoute" ui/src/routeTree.gen.ts   # unchanged (§1.7)
```
Flip §1 matrix rows in `.planning/parity/matrix/9router-ui.md` (MISSING → PARTIAL,
NOT HAVE — §1 note): PAR-UI-013 → PARTIAL (cite §1.3/§8 ESC-1a, "UI half; Go W7");
PAR-UI-019 → PARTIAL (cite §1.4/§8 ESC-1b); PAR-UI-104 → PARTIAL (mock-served list,
§8 ESC-1b); PAR-UI-105 → PARTIAL (mock-served create, §8 ESC-1b); PAR-UI-112 →
PARTIAL (mock-served status, REST-poll §1.5, §8 ESC-1c); PAR-UI-113 → PARTIAL
(mock-served cloudflare enable/disable, §8 ESC-1c); PAR-UI-114 → PARTIAL
(mock-served tailscale enable/disable + `/tunnels` page, §8 ESC-1c). Update
`docs/WORKFLOW.md` (record the P9 base spec observations, the **PARTIAL disposition**
for all seven rows with the W7 Go follow-up list §8, and the **tunnels live-status =
REST-poll, NO SSE** harness decision §1.5). Append the §8 open items to
`.planning/parity/plans/open-questions.md`. Final commit:
`phase-1/w6-m: close — platform cluster (mitm/proxy-pools/tunnels) PARTIAL; matrix flips to PARTIAL`.
**w6-m holds NO serial slot — nothing to release.**

---

## 5. Binary acceptance criteria

All must hold; each is yes/no. `<base>` = the commit recorded at P0 (2978d2bc at
authoring). Diff gate is **w6-m commit-range-scoped** (§7) because page-wave plans
commit to main concurrently.

**Test gates**
- `cd ui && npx playwright test e2e/mitm.spec.ts` → exit 0, all pass (1 original +
  added: status, tools, ca-cert), 0 skipped.
- `cd ui && npx playwright test e2e/proxy-pools.spec.ts` → exit 0, all pass (1
  original + added: list, create-modal, test, delete).
- `cd ui && npx playwright test e2e/tunnels.spec.ts` → exit 0, all pass (1 original +
  added: cards, enable, disable — REST-only, NO SSE).
- `cd ui && npx vitest run src/lib/proxy-pool-form.test.ts` → exit 0, ≥3 cases pass.
- `cd ui && npx vitest run src/` → exit 0 (all prior + new units green).
- `cd ui && npm run build` → exit 0 (routeTree.gen.ts NOT regenerated — §1.7).
- `go test ./... && go vet ./...` → exit 0 (Go untouched — ZERO new Go).

**TDD-order proof** — each impl file's covering test appears in an earlier-or-equal
commit:
```bash
ct=$(git log --format=%ct --diff-filter=A -1 -- ui/src/lib/proxy-pool-form.test.ts)
cf=$(git log --format=%ct --diff-filter=A -1 -- ui/src/lib/proxy-pool-form.ts)
[ "$ct" -le "$cf" ] || echo "TDD VIOLATION: proxy-pool-form.ts"           # prints nothing
# the e2e RED-extension commit precedes the page rewrites
sa=$(git log --format=%ct -1 --grep="failing mitm/proxy-pools/tunnels e2e")
pp=$(git log --format=%ct --diff-filter=M -1 -- ui/src/routes/proxy-pools.tsx)
[ "$sa" -le "$pp" ] || echo "TDD VIOLATION: proxy-pools.tsx before red spec"  # nothing
mi=$(git log --format=%ct --diff-filter=M -1 -- ui/src/routes/mitm.tsx)
[ "$sa" -le "$mi" ] || echo "TDD VIOLATION: mitm.tsx before red spec"      # nothing
tu=$(git log --format=%ct --diff-filter=M -1 -- ui/src/routes/tunnels.tsx)
[ "$sa" -le "$tu" ] || echo "TDD VIOLATION: tunnels.tsx before red spec"   # nothing
```

**Grep proofs**
```bash
grep -rn "/api/mitm/status\|/api/mitm/toggle" ui/src/routes/mitm.tsx                          # PAR-UI-013 status/toggle
grep -rn "/api/mitm/tools" ui/src/routes/mitm.tsx ui/src/components/platform/                 # PAR-UI-013 per-tool toggle
grep -rn "/api/mitm/ca-cert" ui/src/routes/mitm.tsx                                           # §1.3 CA-cert download
grep -rn "/api/proxy-pools" ui/src/routes/proxy-pools.tsx ui/src/components/platform/proxy-pool-form-modal.tsx  # PAR-UI-019/104/105
grep -rn "/api/proxy-pools/.*test\|/test" ui/src/routes/proxy-pools.tsx                       # per-pool test
grep -rn "toProxyPoolPayload\|export " ui/src/lib/proxy-pool-form.ts                          # §1.4 pure helper
grep -rn "/api/tunnels" ui/src/routes/tunnels.tsx ui/src/components/platform/                 # PAR-UI-112/113/114
grep -rn 'POST\|DELETE\|/api/tunnels/' ui/src/routes/tunnels.tsx                              # enable (POST) / disable (DELETE) §1.5
# Tunnels is REST-poll, NOT SSE — NO EventSource introduced:
! grep -rniE 'EventSource|/api/tunnels/stream' ui/src/routes/tunnels.tsx ui/src/components/platform/ && echo "tunnels REST-poll (no SSE) OK §1.5"
# No QueryClientProvider introduced anywhere by w6-m:
! grep -rn "QueryClientProvider" ui/src/components/platform ui/src/routes/mitm.tsx ui/src/routes/proxy-pools.tsx ui/src/routes/tunnels.tsx && echo "no QueryClientProvider added OK"
# No editor/charts/DnD dep:
! grep -rniE 'monaco|codemirror|recharts|dnd-kit|xyflow' ui/src/routes/mitm.tsx ui/src/routes/proxy-pools.tsx ui/src/routes/tunnels.tsx ui/src/components/platform/ && echo "no heavy dep OK"
```

**Negative / freeze proofs (w6-m commit-range — see §7)**
```bash
R="<first-w6-m>^..<last-w6-m>"
git diff $R --name-only -- internal/ | wc -l                            # = 0 (ZERO new Go; routes_admin.go/guard.go untouched, no serial slot)
git diff $R --name-only -- ui/package.json ui/package-lock.json ui/vite.config.ts ui/playwright.config.ts ui/components.json ui/src/index.css | wc -l   # = 0 (no deps/config)
git diff $R --name-only -- ui/src/components/ui/ | wc -l                # = 0 (w6-b frozen)
git diff $R --name-only -- ui/src/stores/ ui/src/providers/ ui/src/lib/api.ts ui/src/lib/utils.ts ui/src/lib/auth.ts | wc -l   # = 0 (w6-a frozen)
git diff $R --name-only -- ui/src/routes/__root.tsx ui/src/main.tsx ui/src/components/layout/ ui/src/routes/login.tsx ui/src/routes/callback.tsx ui/src/components/auth/ | wc -l   # = 0
git diff $R --name-only -- ui/src/routeTree.gen.ts | wc -l             # = 0 (§1.7 UNCHANGED — NOT the w6-l exception)
git diff $R --name-only -- ui/e2e/mocks/fixture.ts ui/e2e/mocks/store.ts ui/e2e/mocks/handlers/index.ts ui/e2e/mocks/seed/index.ts | wc -l   # = 0 (foundation/wiring untouched — platform mocks already registered; NO tunnels SSE branch §1.5)
git diff $R --name-only -- ui/e2e/mocks/handlers/ | wc -l               # = 0 (NO platform handler body edited; mocks consumed as-is §1.4)
git diff $R --name-only -- ui/e2e/mocks/seed/ | wc -l                   # = 0 (platform seeds consumed unchanged)
git diff $R --name-only -- 'ui/src/routes/' | grep -vE 'mitm\.tsx|proxy-pools\.tsx|tunnels\.tsx' | wc -l   # = 0 (only the three stubs rewritten)
git diff $R --name-only -- ui/e2e/ | grep -vE 'mitm\.spec\.ts|proxy-pools\.spec\.ts|tunnels\.spec\.ts' | wc -l   # = 0 (only the three specs; no mock edit)
```

---

## 6. Out of scope (restated, binding)

ZERO Go changes — ALL three platform domains' backends are ABSENT (§1.2); ALL three
pages + their seven API-contract rows ship **PARTIAL** (NOT variant-HAVE — w6-m is
THE designated PARTIAL plan, the MAP pre-declares the Go backends as W7) against the
registered/consumed MOCK contract with the runtime Go gaps deferred to W7 follow-ups
(§8), NEVER an in-plan Go edit (MAP assigns no Go to w6-m; the serial chain closed on
w6-j) and w6-m holds NO serial slot; no `QueryClientProvider` mount (PAR-UI-081
already HAVE from w6-a); no new route files / no `routeTree.gen.ts` change (§1.7 — the
SOLE diff-gate difference from w6-l, where it IS allowed); no dependency additions
(frozen primitives only); no edits to any frozen w6-a/w6-b/page-wave file (no header
exception remains — SPENT); NO mock-layer edits at all —
`index.ts`/`seed/index.ts`/`store.ts`/`fixture.ts` and the platform handler bodies +
seeds are CONSUMED unchanged (already registered); **NO SSE — tunnels live-status is
REST-poll (§1.5), no `fixture.ts` tunnels-stream branch exists or is added**; no new
spec files; no charts/DnD/editor. Mock-vs-Go divergence, an absent backend, or a
shared/foundation-mock edit that would ripple to a non-w6-m spec → escalate (§8),
never patch Go, never fudge a mock, never hand-edit routeTree.gen.ts.

## 7. Diff-gate scope

Page-wave plans (w6-f/j/k/l/m) commit to main concurrently, so a broad
`<base>..HEAD` range sweeps in sibling commits. The diff gate MUST be scoped to
w6-m's own commits. The orchestrator isolates them with:
`git log --oneline main | grep "^[0-9a-f]* phase-1/w6-m:" | awk '{print $1}'`
and runs `git diff <first-w6-m>^..<last-w6-m> -- [file list]` (same commit-range
scoping as w6-c §7 / w6-e §7 / w6-k §7 / w6-l §7).

`git diff <first-w6-m>^..<last-w6-m> --name-only` must be exactly a subset of:

```
ui/src/routes/mitm.tsx
ui/src/routes/proxy-pools.tsx
ui/src/routes/tunnels.tsx
ui/src/components/platform/proxy-pool-form-modal.tsx
ui/src/components/platform/mitm-tool-row.tsx          (OPTIONAL — may be inlined)
ui/src/components/platform/tunnel-card.tsx            (OPTIONAL — may be inlined)
ui/src/lib/proxy-pool-form.ts
ui/src/lib/proxy-pool-form.test.ts
ui/e2e/mitm.spec.ts
ui/e2e/proxy-pools.spec.ts
ui/e2e/tunnels.spec.ts
.planning/parity/matrix/9router-ui.md
.planning/parity/plans/open-questions.md
docs/WORKFLOW.md
```

Any file outside this list in the scoped diff is an automatic review REJECT.
`internal/**` (incl. `routes_admin.go`/`guard.go` — w6-m holds NO serial slot and
adds no platform Go), `ui/package.json`, `ui/src/routeTree.gen.ts` (UNCHANGED §1.7 —
NOT the w6-l exception), `ui/e2e/mocks/fixture.ts`, `ui/e2e/mocks/store.ts`,
`ui/e2e/mocks/handlers/index.ts`, `ui/e2e/mocks/seed/index.ts`, the
mitm/proxy-pools/tunnels handler bodies + seeds (CONSUME unchanged), any sibling
page-plan file, and any frozen w6-a/b file are deliberately ABSENT — touching them is
an automatic REJECT. `ui/dist/**` is gitignored and must never appear. After merge,
the three pages, `ui/src/components/platform/**`, and `ui/src/lib/proxy-pool-form.ts`
become consume-only for later plans.

## 8. Escalations / cross-track dependencies

- **No blocking dependency at authoring.** w6-a + w6-b + the page waves are merged
  (live tree @ 2978d2bc: 16 primitives present, `apiFetch`/stores/mock harness + the
  mitm + proxy-pools + tunnels handlers/seeds ALREADY REGISTERED per §1.1, the
  routing-rules.tsx page template shipped). w6-m holds NO Go serial slot (ZERO new
  Go) and no frozen exception. Fully unblocked for page wave 2.
- **ESCALATION-1 (RESOLVED at authoring — ALL three platform backends ABSENT; the MAP
  pre-declares them W7, §1.2). w6-m flips the seven rows to PARTIAL and schedules the
  W7 Go follow-ups:**
  - **1a (mitm, PAR-UI-013):** no Go `/api/mitm/*`. Ship `/mitm` PARTIAL vs the
    `/api/mitm/{status,toggle,ca-cert,tools/{id}}` MOCK. **W7 Go follow-up:** real
    mitm proxy config — `GET /api/mitm/status`, `POST /api/mitm/toggle`,
    `GET /api/mitm/ca-cert` (serve the generated CA PEM), `POST /api/mitm/tools/{id}`
    (per-tool enable + DNS override) over the MITM proxy subsystem. Flips PAR-UI-013
    PARTIAL → HAVE. NOT in w6-m.
  - **1b (proxy-pools, PAR-UI-019 + PAR-UI-104 + PAR-UI-105):** no Go
    `/api/proxy-pools*`. Ship `/proxy-pools` PARTIAL vs the MOCK
    (`GET/POST /api/proxy-pools`, `POST /api/proxy-pools/batch`,
    `GET/PUT/DELETE /api/proxy-pools/{id}`, `POST /api/proxy-pools/{id}/test`).
    **W7 Go follow-up:** proxy-pool store + admin CRUD + batch import + a real
    connectivity-test endpoint over the outbound-proxy subsystem. Flips PAR-UI-019/
    104/105 PARTIAL → HAVE. NOT in w6-m.
  - **1c (tunnels, PAR-UI-112 + PAR-UI-113 + PAR-UI-114 + the `/tunnels` page):** no
    Go `/api/tunnels*` (only the `guard.go:135-141` settings-guard, not a CRUD
    route). Ship `/tunnels` PARTIAL vs the MOCK (`GET /api/tunnels`,
    `GET /api/tunnels/health`, `POST/DELETE /api/tunnels/{type}` for
    cloudflare/tailscale). **W7 Go follow-up:** real tunnel status/enable/disable
    over the Cloudflare + Tailscale tunnel subsystems (REST status endpoint +
    enable/disable actions; if a live-status STREAM is later wanted, add a real
    `/api/tunnels/stream` SSE endpoint + a sanctioned `fixture.ts` branch — but the
    current contract is REST-poll, §1.5). Flips PAR-UI-112/113/114 PARTIAL → HAVE.
    NOT in w6-m.
- **ESCALATION-2 (CONDITIONAL — shared/foundation mock pressure):** if any surface
  genuinely needs an `index.ts`/`seed/index.ts`/`store.ts`/`fixture.ts`/platform
  handler-body/seed edit (it must not — every route + seed is already registered,
  §1.1, and the mocks are consumed unchanged, §1.4), OR if a within-mock
  inconsistency breaks THIS cluster's specs in a way that would ripple to a non-w6-m
  spec, STOP and ESCALATE (orchestrator serializes the shared change) — do not fudge
  the mock, do not add Go, do not edit any foundation mock.
- **Tunnels live-status harness (RESOLVED — REST-poll, NO SSE, §1.5):** both the
  9router ref (`setInterval` poll on `/api/tunnel/status`, no `EventSource`) and the
  in-tree mock (`GET /api/tunnels` + `/api/tunnels/health`, no `/stream`) are REST.
  The e2e `MockEventSource` (`fixture.ts:35-111`) has NO tunnels branch and needs
  none. **Decision:** the tunnels page reads status via `apiFetch` REST (mount +
  optional interval/interaction refresh); NO `EventSource`, NO `fixture.ts` edit. A
  real streaming status endpoint is a W7 follow-up, NOT w6-m.
- **PARTIAL-vs-HAVE disposition (RESOLVED — the binding call, §1.6):** unlike
  w6-k/w6-l (variant-HAVE), w6-m records all seven rows as **PARTIAL** because the
  MAP w6-m row explicitly pre-declares the Go backends as W7 work. The UI half is
  done + mock-backed specs green; the matrix honestly reflects "UI done, backend
  pending W7." The orchestrator schedules the three W7 Go follow-ups (ESC-1a/1b/1c)
  which flip the rows PARTIAL → HAVE when the backends land.
- **`routeTree.gen.ts` (CONDITIONAL):** if a build reformats it (no new route is
  added, so it must not), that is an ESCALATION (§1.7), not an in-plan edit; resolve
  by regeneration, never manual. (Distinct from w6-l, where the route addition
  legitimately regenerates it.)
- **PAR-UI-081 dependency (RESOLVED):** already HAVE from w6-a (apiFetch = queryFn
  adapter, `open-questions.md:6`). w6-m consumes `apiFetch`; no `QueryClientProvider`
  mount; the MAP-decision-2 TanStack-Query provider wiring (if ever wanted) is an
  orchestrator serial follow-up, NOT w6-m.
- **MAP-disposition follow-up (record, non-blocking):** the WAVE-6-MAP w6-m row
  already correctly states the Go backends remain W7; §1.2 VERIFIED all three ABSENT.
  Record in WORKFLOW.md + open-questions so the orchestrator tracks the three W7 Go
  follow-ups (ESC-1a..1c) that flip PAR-UI-013/019/104/105/112/113/114 PARTIAL → HAVE.
```
