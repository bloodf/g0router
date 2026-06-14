# Micro-plan w7-platnodes — Provider-node prefix-routing engine (Go)

```
wave: 7
plan: w7-platnodes
status: READY (rev 1 — authored against merged Waves 0–6 + the shipped W7
  governance/platform/mcp plans, live tree @ <base>; WAVE-7-MAP w7-platnodes row
  ~line 175; serial chain §219-238 (w7-platnodes is FIRST); concurrency §225-238;
  e2e reconciliation §245; freeze rules §267)
runs: governance+routing track. FIRST in the routes_admin.go SERIAL CHAIN
  (w7-platnodes → w7-route → w7-gov-1 → w7-gov-2 → w7-gov-3 → w7-mcp-3 → w7-plat-1
  → w7-plat-2 → w7-plat-3 → w7-misc; MAP §219-238). This plan is the routing
  PREREQUISITE: w7-route depends on it (PAR-ROUTE-009/040 prefix routing). It runs
  ALONE on the routing serial sub-chain before w7-route starts.
branch: main (direct commits, per repo convention)
commit-prefix: phase-1/w7-platnodes:
ref-source: 9router frozen @ 827e5c3 — provider-node prefix-routing surfaces
  (src/sse/services/model.js:35-51 prefix override; src/app/api/provider-nodes/
  route.js:20-104 CRUD + baseUrl sanitization; .../[id]/route.js:61-74 cascade;
  .../validate/route.js:52-201 /models→/chat/completions probe; src/lib/db/
  schema.js:49-58 providerNodes(id,type,name,data,...) JSON-data table). The
  BINDING contract for W7 is the W6 e2e mock (decision 1: real Go wins, mock
  corrected to mirror it). Mock source: ui/e2e/mocks/handlers/nodes.ts.
base: <base> = git rev-parse HEAD recorded at P0. Substitute the actual SHA
  everywhere §5 says <base>. (At authoring, HEAD = 2de2624.)
go-serial-slot: this plan holds the SINGLE in-flight edit to
  internal/server/routes_admin.go while live (W3/W4/W5/W6 lesson; MAP decision 3).
  As the FIRST chain holder the slot is free at P-check; RELEASE to w7-route on close.
inference-resolution-hook: this plan adds an ADDITIVE prefix-override step to the
  model→provider resolution chain (internal/inference/router.go Resolve, or
  factory.go providerForModel). It is ADDITIVE-ONLY (a node-prefix check BEFORE
  static alias/catalog resolution; no rewrite of existing resolution). w7-route
  later edits selection.go (weighted selection) — a DIFFERENT file; no
  selection.go edit here, so no selection-micro-serial conflict with this plan.
new-route: NO UI route files. The provider-nodes admin surface already SHIPPED a
  THIN version in w6-f (ListProviderNodes/CreateProviderNode/ValidateProviderNode
  over the providers table); this plan EXTENDS that surface to the real prefix-
  routing engine (prefix/api_type persistence, update/delete, cascade, real
  /models→/chat/completions probe) and corrects the nodes.ts mock to mirror it.
```

---

## 1. Scope — PAR rows + the engine

### Rows this plan closes

| Row / item | Claim | Target state after w7-platnodes |
|---|---|---|
| PAR-ROUTE-009 | Provider-node prefix matching (openai-/anthropic-compatible, custom-embedding) overrides static alias resolution | HAVE — a node registered with prefix `mn` routes `mn/some-model` to that node's provider, BEFORE static alias/catalog (`internal/inference` additive hook, §1.5) |
| PAR-ROUTE-040 | OpenAI-compatible + Anthropic-compatible provider-node routing | HAVE — node `api_type` (openai/anthropic) selects the adapter; prefix-stripped bare model routed to the node's base URL (§1.5) |
| PAR-PLAT-010 | Provider-node CRUD: types `openai-compatible`/`anthropic-compatible`/`custom-embedding` with prefix + baseUrl | HAVE — `internal/store/providernodes.go` (typed cols on the providers table) + extended `internal/admin/nodes.go` CRUD (list/create/**get/update/delete**, §1.4) |
| PAR-PLAT-011 | baseUrl sanitization: strip trailing `/messages` (anthropic), `/embeddings` (custom-embedding) | HAVE — `platform.SanitizeNodeBaseURL(apiType, raw)` applied on create/update (§1.6) |
| PAR-PLAT-012 | Node update cascades prefix/baseUrl/apiType to bound connections | HAVE — `platform.ProviderNodeService.Update` re-points bound connections' provider base URL (§1.7 cascade) |
| PAR-PLAT-013 | Node validation: `/models` first, fall back to `/chat/completions` if `modelId` given; custom-embedding tests `/embeddings` POST | HAVE — `ValidateProviderNode` runs the real probe via an injectable HTTP seam (hermetic in tests), SSRF-guarded (§1.8) |
| PAR-PLAT-014 | Provider-node SQLite schema | HAVE — additive columns on the providers table (`prefix`, `api_type`), NOT a new JSON-`data` table; justified §1.3 (decision ESC-SCHEMA) |

Matrix flips at closeout (§4 T-close): in `.planning/parity/matrix/9router-routing.md`
PAR-ROUTE-009, PAR-ROUTE-040 → HAVE; in `.planning/parity/matrix/9router-platform.md`
PAR-PLAT-010..014 → HAVE. Append any new open items (§8) to `open-questions.md`.

### 1.1 Preconditions already satisfied by merged waves (evidence)

- **W6-f shipped a THIN provider-nodes admin surface (consume-and-extend).** `internal/
  admin/nodes.go` (live @ <base>) exposes `ListProviderNodes` (filters the providers
  table to `type=="openai-compatible"`, `nodes.go:62-76`), `CreateProviderNode`
  (creates a `providers` row of `providerNodeType="openai-compatible"`, accepts
  camelCase `baseUrl`/`apiType`/`prefix` at decode but **does NOT persist
  prefix/api_type** — `nodes.go:78-104`, comment l.18-20 says "the providers table
  has no such columns"), and `ValidateProviderNode` (best-effort, **URL well-
  formedness only**, `isWellFormedURL`, `nodes.go:106-137`). The decode struct
  `providerNodeRequest` (`nodes.go:42-58`) already accepts `{name,prefix,api_type,
  apiType,base_url,baseUrl,api_key,apiKey}`. The DTO `providerNodeDTO`
  (`nodes.go:21-37`) is `{id,name,base_url,type,enabled}`. **This plan EXTENDS this
  file ADDITIVELY**: persist prefix/api_type, add Get/Update/Delete, apply baseUrl
  sanitization, replace the URL-shape-only validate with a real (injectable) probe,
  and add the cascade. The thin-version function bodies that survive are EXTENDED,
  not rewritten wholesale (scope each body change precisely, §3).
- **The existing nodes tests are the authoritative regression surface** (`internal/
  admin/nodes_test.go`, live @ <base>): `TestListProviderNodesFiltersOpenAICompatible`,
  `TestProviderNodesCreatePersists`, `TestProviderNodesCreateSnakeCaseBaseURL`,
  `TestProviderNodesCreateValidation`, `TestProviderNodesValidateURL`,
  `TestProviderNodesValidateNeverPersistsAPIKey`, `TestNodesRouteDisambiguation`.
  **These must STAY green** — new behavior is additive. (The validate-URL test
  asserts a well-formed URL returns `valid:true` with NO network; the real probe
  must preserve that under the hermetic fake — §1.8.)
- **The providers store is a flat typed-column record** (`internal/store/providers.go`):
  `ProviderRecord{ID,Name,Type,BaseURL,Enabled,CreatedAt,UpdatedAt}` (`providers.go:11-19`)
  with `CreateProvider`/`ListProviders`/`GetProvider`/`UpdateProvider`/`DeleteProvider`
  + `scanProvider` (`providers.go:22-112`). The `providers` table is
  `{id,name,type,base_url,enabled,created_at,updated_at}` (`migrate.go:37-45`). w6-f
  mapped nodes onto THIS table — this plan continues that (additive columns), NOT a
  new table (§1.3 / ESC-SCHEMA).
- **The model→provider resolution chain is `providerForModel`** (`internal/inference/
  factory.go:33-91`), called from `router.go:61`, `runner.go:30,70`. Resolution order
  today: (1) explicit `provider/model` or `alias/model` prefix → `ParseModelPrefix`
  (`alias.go:58-64`) → `catalog.ResolveProviderAlias` → catalog `Lookup` →
  `InferProvider` (name-prefix) → legacy claude-/gemini- heuristics; (3) no-prefix
  catalog lookup over `providerPrecedence`; (4) `InferProvider`; (5) legacy
  heuristics. **The node-prefix override must hook BEFORE this static chain** — i.e.
  a model `myprefix/x` whose `myprefix` matches a registered node prefix routes to
  that node's provider, short-circuiting steps 1-5. The cleanest additive site is
  `Router.Resolve` (`router.go:53-90`) — it already owns an optional
  `aliasStore`/`keyResolver` and runs BEFORE `providerForModel`; add an optional
  `nodeResolver` consulted FIRST (§1.5 / ESC-HOOK).
- **`ResolveModelAlias` runs in `Router.Resolve` BEFORE `providerForModel`**
  (`router.go:55-59`). The node-prefix check must run BEFORE `ResolveModelAlias`
  (a node prefix overrides static alias resolution — PAR-ROUTE-009 "override static
  alias resolution"). So the additive hook is the FIRST step of `Resolve` (§1.5).
- **Router wiring point is `internal/server/server.go:52-55`** — `infRouter.
  SetKeyResolver(resolver)` + `infRouter.SetAliasStore(st)` inside the `if st != nil`
  block. The node resolver wires here too via a new `infRouter.SetNodeResolver(...)`
  ADDITIVE setter (mirrors `SetAliasStore`; NO `NewRouter` signature change, §1.5).
- **`internal/platform` is the live domain home** (NOT a placeholder anymore): it
  holds `ProxyPoolService` (`platform/proxypools.go`, the domain-service template
  with an injectable `Prober` seam, `proxypools.go:18,36`), the SSRF evaluator
  `IsBlockedTarget(host, resolver)` / `IsBlockedIP(ip)` + the injectable
  `IPResolver` seam (`platform/outboundproxy.go:10,27,50`), and `platform/{tunnel,
  mitm}` sub-packages. **The new `platform.ProviderNodeService` follows the
  `ProxyPoolService` template exactly** (constructor over `*store.Store`, injectable
  HTTP-probe seam, errors-as-values, no init()). **REUSE `platform.IsBlockedTarget`**
  for the validation-probe SSRF guard (§1.8) — do NOT reimplement.
- **Envelope + handler patterns** (`internal/admin/respond.go`): `writeData(ctx,
  status, data)` / `writeError(ctx, status, message)` → `{data,error:{message}}`
  snake_case. `pathID(ctx.UserValue("id"))` extracts `{id}` (`handlers.go:142-145`).
  CRUD template incl. ErrNotFound→404 = `internal/admin/virtualkeys.go` /
  `internal/admin/proxypools.go`.
- **The audit seam is live** — `func (h *Handlers) recordAudit(ctx, action, target,
  details string)` (`internal/admin/audit.go:64`) resolves the actor from
  `ctx.UserValue(userKey).(*store.User).Username`, best-effort, logs on failure.
  **REUSE `h.recordAudit` on every node mutation** (create/update/delete). NO audit
  retrofit into other files. (Existing thin `CreateProviderNode` does NOT audit
  today; adding the audit call to the EXTENDED create is sanctioned additive.)
- **Migrations are additive-only** (`internal/store/migrate.go`): new columns via the
  `ensureColumn` loop (`migrate.go:401`), precedent `connections.proxy_pool_id`
  (`migrate.go:321`) + the backoff/rate-limited additive columns
  (`migrate.go:318-321`). New tables via the `tables` slice with
  `CREATE TABLE IF NOT EXISTS` (`migrate.go:37`). secret-at-rest precedent = the
  `*_enc` reversible columns via `s.cipher.Encrypt/Decrypt` (`connections.go:116-151`).
- **Handlers injection** — the `Handlers` struct composes `h.store` directly and
  constructs domain services in `New` (`handlers.go:55-64`: `audit`/`proxyPools`/
  `tunnels`/`mitm`). Add a `providerNodes *platform.ProviderNodeService` field
  constructed in `New` (NO signature change, mirrors `proxyPools`, §3). Injectable
  test seams use additive setters (`SetProxyProber`, `handlers.go:98`); add
  `SetNodeProber` mirroring it (§1.8).

### 1.2 The mock contract this flip must mirror (binding — decision 1)

**Decision 1 (MAP §36, §245):** real Go wins; the W6 mock body is corrected IN THIS
PLAN to mirror the real Go `{data,error}` snake_case DTO. Pages are FROZEN
(decision 8); prefer matching the mock's existing field names in the Go DTO.

**Provider-nodes** (`ui/e2e/mocks/handlers/nodes.ts`, live @ <base>):
- Routes the page consumes:
  - `GET /api/provider-nodes` → `{data:{nodes:[{id,name,base_url,type,enabled}]}}`
    (`nodes.ts:43-49`; filters the providers store to `type=="openai-compatible"`).
  - `POST /api/provider-nodes` → `{data:{node:{...}}}` (`nodes.ts:51-63`; accepts
    `baseUrl`/`base_url`).
  - `POST /api/provider-nodes/validate` → `{data:{valid,error?}}`, api_key NEVER
    stored (`nodes.ts:66-74`; URL-shape check only).
  - `POST /api/models/test` → `{data:{ok,latency_ms}}` (`nodes.ts:75-80`) and
    `GET /api/models/availability` → `{data:{available:[...]}}` (`nodes.ts:81-90`)
    are **mock-only model-test surfaces** belonging to w7-misc (the `/api/models/*`
    backend), NOT w7-platnodes. **DO NOT touch the `/api/models/*` branches** — they
    are out of scope (§ NOT in scope). w7-platnodes owns ONLY the `/api/provider-
    nodes*` branches in this file.
- **Mock divergences to reconcile (mock mirrors Go):**
  - **DTO fields:** the real node DTO gains `prefix` and `api_type` (now persisted).
    **DECIDE whether to add them to the mock `toNode` (§8 ESC-MOCK-DTO).** RECOMMENDED:
    add `prefix` + `api_type` to `toNode` ONLY if the page reads them; the node UI
    page is FROZEN — VERIFY at P4 whether the page consumes prefix/api_type. If the
    page ignores them, the mock may carry them harmlessly to mirror Go (preferred:
    mirror Go's full DTO). If adding breaks a spec, ESC-MOCK.
  - **GET/PUT/DELETE `/api/provider-nodes/{id}`:** the mock has NO `{id}` route
    (w6-f never shipped update/delete). The real Go ADDS them. **The mock MUST add a
    `\/api\/provider-nodes\/[^/]+$` route** (get-or-404 / merge-update / delete→
    `{data:{message}}`) to mirror the new Go surface, ONLY IF a spec exercises it.
    If no spec uses `{id}`, the mock addition is optional (Go admin tests are the
    authoritative proof); VERIFY at P4 (§8 ESC-MOCK-CRUD). Default: add the `{id}`
    route to mirror Go (cheap, future-proofs the page).
  - **Validate:** the mock returns `{valid:true}` for any well-formed URL with NO
    network. The real Go probe is hermetic in tests (injectable seam → deterministic
    success for a well-formed reachable URL). The mock stays as-is (URL-shape →
    `valid:true`); it already mirrors the hermetic success path. NO change unless the
    real Go validate response gains a field the page reads (e.g. `models:[...]` from
    the `/models` probe — VERIFY; default: keep `{valid,error?}`, §8 ESC-MOCK-VALID).
- If a mock correction reds a non-w7-platnodes spec, STOP + ESCALATE (§8 ESC-MOCK).

### 1.3 Schema decision (binding — ESC-SCHEMA, additive columns over a new table)

**DECISION (recommended default, flag for orchestrator confirm).** 9router stores
provider nodes in a dedicated `providerNodes(id,type,name,data,createdAt,updatedAt)`
table with a JSON `data` blob (PAR-PLAT-014, `schema.js:49-58`). g0router's w6-f
already mapped nodes onto the EXISTING `providers` table (a node IS a `providers` row
of `type="openai-compatible"`). **Continue the w6-f mapping with ADDITIVE typed
columns** — `ensureColumn("providers","prefix","TEXT NOT NULL DEFAULT ''")` +
`ensureColumn("providers","api_type","TEXT NOT NULL DEFAULT ''")` — rather than a new
JSON-`data` table. **Justification:**
- It avoids churning w6-f (the thin nodes.go, its tests, and the mock already treat a
  node as a providers row); a new table would orphan w6-f's CRUD and the providers→
  connections FK (`migrate.go:356`).
- Typed columns enable the prefix-routing index/lookup (`WHERE prefix=?`) and the
  cascade (`WHERE provider_id IN (node ids)`) cleanly; a JSON blob would require
  scanning every row to match a prefix at request time (hot path).
- The fixed node shape (`prefix`/`api_type` are scalar) fits typed columns; this
  matches the gov/plat plans' "typed columns for fixed-shape domains, JSON only for
  nested data" precedent.
- Additive-only (MAP decision 2): two new nullable-defaulted columns; existing
  providers rows get `prefix=''`/`api_type=''` (not nodes) — the node filter is
  `type IN (node types) AND prefix != ''` or simply `type` membership (preserve the
  w6-f `type=="openai-compatible"` filter, EXTENDED to the three node types).
- **Node types:** the three 9router node types are `openai-compatible`,
  `anthropic-compatible`, `custom-embedding` (PAR-PLAT-010). w6-f's
  `providerNodeType = "openai-compatible"` const becomes a SET membership. Add a
  `nodeTypes` set; the list filter and the resolver accept all three.

**If the operator prefers the 9router-exact JSON `data` table** (a faithful port),
that is the alternative — but it duplicates the providers/connections wiring and
breaks w6-f. RECOMMENDED: additive columns. Decide at T-schema; do NOT build both.

### 1.4 Architecture (binding — layered DDD, decision 4)

Layered transport → domain → repository, mirroring `ProxyPoolService`:

```
node CRUD:    admin/nodes.go (EXTEND)  → platform/providernodes.go (NEW domain) → store/providernodes.go (NEW repo, additive cols on providers)
baseUrl san:  (create/update)          → platform.SanitizeNodeBaseURL (pure)
cascade:      admin Update             → platform.ProviderNodeService.Update → re-point bound connections' provider base_url
validation:   admin Validate           → platform.ProviderNodeService.Validate (injectable HTTP probe + SSRF guard)
prefix route: inference/router.go Resolve (ADDITIVE first step) → NodeResolver iface → platform.ProviderNodeService.ResolveByPrefix
```

- **`platform.ProviderNodeService`** (NEW, mirrors `ProxyPoolService`): constructor
  `NewProviderNodeService(st)`; injectable HTTP-probe seam `SetProber(NodeProber)`
  (production = real `/models`→`/chat/completions` HTTP; tests = deterministic fake);
  optional `SetResolver(platform.IPResolver)` reusing the SSRF resolver seam. Methods:
  `List`, `Create`, `Get`, `Update` (with cascade), `Delete`, `Validate`,
  `ResolveByPrefix(prefix) (*store.ProviderRecord, bool)`. No init(); errors-as-values.
- **`store/providernodes.go`** (NEW, additive over the providers table): node-scoped
  read/write helpers that set/scan `prefix`/`api_type` ON TOP of the existing
  `ProviderRecord` CRUD. **DECIDE the shape (§8 ESC-STORE-SHAPE):** RECOMMENDED =
  extend `ProviderRecord` ADDITIVELY with `Prefix string` + `APIType string`, update
  the existing `CreateProvider`/`UpdateProvider`/`scanProvider` SELECT/INSERT/UPDATE
  column lists to include the two new columns (ADDITIVE — defaults `''` keep all
  existing callers green), and add node-scoped helpers `ListProviderNodes()` (filter
  to node types), `GetProviderNodeByPrefix(prefix)`, `ListProviderNodePrefixes()` in
  the new `providernodes.go` file. The two-column `ProviderRecord` extension is the
  single additive touch to `providers.go`; everything node-specific lives in
  `providernodes.go`.
- **`admin/nodes.go`** (EXTEND): persist prefix/api_type; add Get/Update/Delete +
  the real validate; apply sanitization; call `h.recordAudit` on mutations; route
  through `h.providerNodes` (the service) for cascade + validate + resolve.
- **prefix route hook** (`internal/inference`): a `NodeResolver` interface
  (`ResolveByPrefix(prefix string) (providerID, baseURL, apiType string, ok bool)`)
  set on the `Router` via `SetNodeResolver`; consulted as the FIRST step of `Resolve`
  (§1.5). ADDITIVE — no rewrite of `providerForModel`.

### 1.5 Prefix-override hook (binding — the precedence decision, ESC-HOOK)

**Where it hooks + precedence.** The override is the FIRST step of `Router.Resolve`
(`internal/inference/router.go:53`), BEFORE `ResolveModelAlias` and BEFORE
`providerForModel`. Precedence (binding): **node-prefix BEFORE static alias/catalog.**

```
Resolve(model):
  0. (NEW, ADDITIVE) if nodeResolver != nil:
       prefix, bare := ParseModelPrefix(model)              // reuse alias.go:58
       if prefix != "" {
         if providerID, baseURL, apiType, ok := nodeResolver.ResolveByPrefix(prefix); ok {
           // route to the node: build/route a generic provider pointed at baseURL,
           // using bare (prefix-stripped) model; api_type selects the adapter.
           return node-routed provider + key   // SHORT-CIRCUITS steps 1-5
         }
       }
  1. (existing) if aliasStore != nil: model = ResolveModelAlias(...)   // UNCHANGED
  2. (existing) providerForModel(model) ...                            // UNCHANGED
```

- **Additive only:** a new `nodeResolver` field on `Router` + a `SetNodeResolver`
  setter (mirror `SetAliasStore`, `router.go:44-48`; NO `NewRouter` signature
  change) + the step-0 block at the top of `Resolve`. When `nodeResolver` is nil OR
  no prefix matches a node, behavior is IDENTICAL to today (falls through to alias→
  `providerForModel`). The existing resolution logic is NOT rewritten.
- **`NodeResolver` interface** (in `internal/inference`, defined where `AliasStore`
  is, `alias.go:11-15`): `ResolveByPrefix(prefix string) (providerID, baseURL,
  apiType string, ok bool)`. The implementation is `platform.ProviderNodeService`
  (adapted via a thin adapter in `server.go` or a method satisfying the interface —
  decide at T-hook to avoid an inference→platform import cycle; RECOMMENDED = define
  the interface in inference, implement it in platform, wire in server.go where both
  are visible — mirrors how `aliasStore = st` is wired at `server.go:55`).
- **Node-routed provider construction (ESC-NODE-PROVIDER).** A node's bare model must
  be served by a generic OpenAI-/Anthropic-compatible provider pointed at the node's
  `base_url`. **DECIDE the construction site (§8):** RECOMMENDED = reuse the existing
  generic-provider build path (`buildProvider` default branch → `generic.New(id)`,
  `factory.go:104-109`) with the node's base URL injected via the provider's existing
  base-URL config seam; for `anthropic-compatible` route through the anthropic
  adapter. The node's provider ID = the node's providers-row ID (already a catalog-
  external ID). If the generic build path cannot accept a runtime base URL without a
  signature change, the node provider is built directly in the resolver path
  (additive helper, NO change to `buildProvider`/provider constructors). Confirm the
  exact construction at T-hook; KEEP it additive — NO interface/constructor signature
  change to `schemas.Provider` or the provider `New()` funcs.
- **Unit-test the precedence (binding, §4 T-hook):**
  - a model `mn/some-model` with a registered node prefix `mn` routes to the node's
    provider + base URL (NOT to static alias/catalog);
  - the SAME bare model name WITHOUT the node prefix falls through to the existing
    static resolution (unchanged);
  - a prefix that matches BOTH a node and a static alias → the NODE wins (override);
  - `nodeResolver == nil` → resolution is byte-identical to the pre-hook behavior
    (the existing inference tests stay green — the proof that the hook is additive).

### 1.6 baseUrl sanitization (PAR-PLAT-011, NEW)

`platform.SanitizeNodeBaseURL(apiType, raw string) string` (pure, unit-tested):
- `anthropic-compatible`: strip a trailing `/messages` (and `/messages/`) →
  the node base URL should be the API root, not the messages endpoint
  (9router `route.js:66-69`).
- `custom-embedding`: strip a trailing `/embeddings` (`route.js:83-87`).
- `openai-compatible`: no strip (or strip a trailing `/chat/completions` if present —
  VERIFY against 9router `route.js`; default: leave openai base URLs untouched per
  the w6-f behavior, only sanitize anthropic/embedding).
- Idempotent; preserves scheme/host/query; trims a single trailing slash.
Applied in `CreateProviderNode` and `UpdateProviderNode` BEFORE persisting. Unit
tests: anthropic `https://x/v1/messages`→`https://x/v1`; embedding
`https://x/embeddings`→`https://x`; openai unchanged; no-trailing-segment unchanged.

### 1.7 Cascade-to-connections (PAR-PLAT-012, NEW)

`ProviderNodeService.Update(node)` cascades prefix/baseUrl/apiType changes to bound
connections (9router `[id]/route.js:61-74`). In g0router a node IS a `providers` row;
connections bind to a provider via `connections.provider_id` (`connections.go:15`).
**The cascade re-points the node's bound state:**
- On node base-URL change, the providers-row `base_url` IS the node's base URL (the
  provider record), so updating the providers row already propagates to every
  connection that resolves via that provider_id — connections store a credential, not
  a base URL (`Connection` has no base-URL field, `connections.go:13-26`). **So the
  cascade primarily = persisting the sanitized base_url/api_type/prefix on the
  providers row** (which connections read transitively at resolve time). **DECIDE
  whether any per-connection field needs rewriting (§8 ESC-CASCADE):** RECOMMENDED =
  the cascade is the providers-row update PLUS, if a node's prefix changes, re-stamp
  any connection metadata that caches the prefix (none today — `Connection.Metadata`
  is free-form; VERIFY no connection caches the node prefix at P-check). If no
  connection field mirrors node data, the cascade is satisfied by the providers-row
  update + a documented note; if a connection field DOES cache it, the cascade
  updates those connection rows (additive helper `RepointConnectionsForNode`).
- **Cascade-to-connections on CREATE (the brief's "creating a node provisions/links a
  connection").** 9router create may auto-provision a connection for the node.
  **DECIDE (§8 ESC-PROVISION):** RECOMMENDED = on `CreateProviderNode`, if the
  request carries an `api_key`, auto-create a bound `api_key` connection
  (`store.CreateConnection{ProviderID: node.ID, Kind:"api_key", Secret: apiKey}`)
  so the node is usable immediately; the api_key is encrypted at rest (`*_enc`,
  `connections.go:119`) and NEVER echoed. If the request has no api_key, create the
  node only (no connection). VERIFY against 9router `route.js` create behavior; if
  9router does NOT auto-provision, default to node-only create + a follow-up note.
  Unit-test both branches (with-key → a connection exists; without-key → none).

### 1.8 Validation probe (PAR-PLAT-013, NEW — the hermetic-test core)

`ValidateProviderNode` runs a REAL reachability probe, hermetic in tests via an
injectable seam (AGENTS.md "no mocks; use interfaces and fakes"; mirrors
`ProxyPoolService.Prober`, `proxypools.go:18,36`, and `SetProxyProber`,
`handlers.go:98`).
- **`NodeProber` seam** (`platform`): `type NodeProber func(req NodeProbeRequest)
  (NodeProbeResult, error)` where the request carries `{apiType, baseURL, apiKey,
  modelID}`. Production = real HTTP; tests inject a deterministic fake (no network).
  Add `SetNodeProber` on `Handlers` mirroring `SetProxyProber`.
- **Probe logic (9router `validate/route.js:52-201`):**
  - openai-/anthropic-compatible: GET `{baseURL}/models` first; if it fails AND a
    `modelId` is provided, POST a minimal `{baseURL}/chat/completions` with that
    model; success if either returns 2xx.
  - custom-embedding: POST `{baseURL}/embeddings` with a minimal body.
  - Response: `{data:{valid:bool, error?:string, models?:[...]}}` (the optional
    `models` list from `/models` — include ONLY if the page reads it, §8 ESC-MOCK-VALID).
- **SSRF guard BEFORE dialing (binding, reuse `platform.IsBlockedTarget`).** The
  node base URL host is user-controllable → guard it with
  `platform.IsBlockedTarget(host, resolver)` (`outboundproxy.go:50`) before invoking
  the prober; a base URL resolving to a private/loopback/link-local/metadata IP is
  REFUSED with `valid:false, error:"target blocked"` and NEVER probed. This closes
  the validate-as-SSRF vector (analogous to PAR-AUTH-020's proxy-host guard,
  `proxypools.go:83`).
- **api_key handling (binding):** the supplied `api_key` is used transiently for the
  probe and NEVER persisted by validate, NEVER echoed (preserve the w6-f
  `TestProviderNodesValidateNeverPersistsAPIKey` invariant, `nodes_test.go:170-187`).
- **Hermetic unit tests:** well-formed reachable (fake → 2xx) → `valid:true`;
  `/models` fails + modelID given + `/chat/completions` 2xx → `valid:true`;
  both fail → `valid:false` + error; base URL → blocked IP → `valid:false`
  (SSRF, prober NOT called); malformed URL → `valid:false` (preserve existing test);
  validate persists NO provider row + leaks NO api_key.

### 1.9 routes_admin.go registration (serial-slot additive, §3)

w6-f already registered the three thin routes (`routes_admin.go:136-138`). This plan
ADDS the new `{id}` CRUD routes (static collection + `/validate` already present;
`{id}` deeper):
```go
// Provider-nodes prefix-routing engine (extends the w6-f thin surface).
// (existing — keep) GET/POST /api/provider-nodes ; POST /api/provider-nodes/validate
r.GET("/api/provider-nodes/{id}", h.RequireSession(h.GetProviderNode))      // NEW
r.PUT("/api/provider-nodes/{id}", h.RequireSession(h.UpdateProviderNode))   // NEW
r.DELETE("/api/provider-nodes/{id}", h.RequireSession(h.DeleteProviderNode))// NEW
```
Route-precedence: the static `/api/provider-nodes/validate` (already at
`routes_admin.go:138`) must remain registered so the matcher disambiguates it from
`/api/provider-nodes/{id}` — the existing `TestNodesRouteDisambiguation`
(`nodes_test.go:189-236`) proves validate-vs-collection; EXTEND it to prove
validate-vs-`{id}`. A genuine `fasthttp/router` collision is §8 ESC-ROUTE, not a
silent path change. Diff bound §5: the route additions are ONE commit, additive only.

### NOT in scope (explicit)

- **No UI page/component/route/store edits** — the provider-nodes page + all w6-f
  components are FROZEN consume-only (decision 8). The ONLY UI-tree touch is the
  `nodes.ts` mock-body correction for the `/api/provider-nodes*` branches (§1.2 / §3).
- **No `/api/models/*` edits** — the `/api/models/test` + `/api/models/availability`
  mock branches (`nodes.ts:75-90`) belong to w7-misc; do NOT touch them or build
  their Go.
- **No edits to w7-route's selection.go weighted-selection logic** — w7-route owns
  selection.go; this plan does NOT touch selection.go (the prefix hook lives in
  router.go, a different file). No selection-micro-serial conflict.
- **No edits to pre-existing admin handlers' bodies** other than the SANCTIONED
  ADDITIVE extension of `internal/admin/nodes.go` (the w6-f file this plan owns going
  forward) — apikeys, virtualkeys, providers* (except the two-column `ProviderRecord`
  store extension), connections, combos, auth, audit, proxypools are FORBIDDEN.
- **No rewrite of the existing resolution chain** — `providerForModel`,
  `ResolveModelAlias`, `InferProvider`, `ParseModelPrefix` are UNCHANGED; the node
  hook is an additive pre-step in `Resolve`.
- **No interface / constructor signature change** — `schemas.Provider`, provider
  `New()` funcs, `NewRouter`, `NewProviderNodeService` (new), `New(...)` (admin)
  signatures PRESERVED (additive setters / optional deps only; MAP decision 9).
- **No destructive DDL / column renames** — additive `ensureColumn` ONLY (decision 2).
- **No new global state** — the node service is constructed over `h.store`.
- **No secret exposure** — node `api_key` (if provisioned as a connection) `*_enc`
  at rest + NEVER echoed; validate uses it transiently; the node DTO carries NO key.

---

## 2. Precondition checks

Run all before any edit; abort and report to orchestrator on any failure.

```bash
# P0 — clean tree; record <base>
git status --porcelain     # must be empty (explicit `git add <file>`, never -A;
                           # ui/dist/** gitignored — never stage it)
git rev-parse HEAD         # record as <base> for §5

# P1 — the gaps are REAL (thin w6-f exists; prefix/api_type/cascade/probe absent)
grep -nE 'prefix|api_type' internal/store/providers.go ; echo "^ expect EMPTY (cols not persisted)"
grep -nE 'prefix|api_type' internal/store/migrate.go ; echo "^ expect EMPTY for providers"
test ! -e internal/store/providernodes.go && test ! -e internal/platform/providernodes.go && echo "node store/domain gap OK"
grep -nE 'GetProviderNode|UpdateProviderNode|DeleteProviderNode' internal/admin/nodes.go ; echo "^ expect EMPTY (no {id} CRUD)"
grep -nE 'SetNodeResolver|NodeResolver|ResolveByPrefix' internal/inference/*.go | grep -v _test ; echo "^ expect EMPTY (no prefix hook)"
grep -nE '/api/provider-nodes/\{id\}' internal/server/routes_admin.go ; echo "^ expect EMPTY"

# P2 — reused surfaces present (the de-risk)
grep -n "type ProviderRecord\|func (s \*Store) CreateProvider\|UpdateProvider\|scanProvider" internal/store/providers.go
grep -n "func (r \*Router) Resolve\|SetAliasStore\|SetKeyResolver\|providerForModel" internal/inference/router.go internal/inference/factory.go
grep -n "func ParseModelPrefix" internal/inference/alias.go
grep -n "infRouter.SetAliasStore\|infRouter.SetKeyResolver" internal/server/server.go
grep -n "func NewProxyPoolService\|type Prober\|func (s \*ProxyPoolService) SetProber\|func IsBlockedTarget\|type IPResolver" internal/platform/proxypools.go internal/platform/outboundproxy.go
grep -n "func (h \*Handlers) recordAudit\|func (h \*Handlers) SetProxyProber" internal/admin/audit.go internal/admin/handlers.go
grep -n "func writeData\|func writeError" internal/admin/respond.go
grep -n "func newTestEnv\|func call(" internal/admin/admin_test.go

# P3 — the w6-f thin nodes surface + its tests are present (consume-and-extend)
grep -n "func (h \*Handlers) ListProviderNodes\|CreateProviderNode\|ValidateProviderNode\|providerNodeType\|providerNodeRequest" internal/admin/nodes.go
go test ./internal/admin/ -run 'ProviderNodes|Nodes' -v 2>&1 | tail -20   # green at base (regression baseline)

# P4 — the W6-f mock present + whether the page reads prefix/api_type/{id}/models-list
test -f ui/e2e/mocks/handlers/nodes.ts && echo "mock present"
grep -nE 'prefix|api_type|provider-nodes/|/models' ui/e2e/mocks/handlers/nodes.ts
grep -rnE 'prefix|api_type|provider-nodes' ui/src 2>/dev/null | head   # what the FROZEN page actually consumes (resolves ESC-MOCK-*)
ls ui/e2e/*node* 2>/dev/null ; echo "^ no dedicated nodes spec at base (mock is consumed by provider/connection specs)"

# P5 — routes_admin.go serial slot FREE (w7-platnodes is FIRST in the chain)
git log --oneline -5 -- internal/server/routes_admin.go   # last touch = a merged W7 plan
# Orchestrator MUST confirm no concurrent W7 plan holds an unmerged routes_admin.go
# edit, AND that w7-route has NOT started (w7-platnodes precedes it).

# P6 — green at base
go test ./... && go vet ./... && go build ./...     # exit 0 (Go untouched-green)
cd ui && npm run build                               # exit 0 (build BEFORE any e2e)
```

---

## 3. Exclusive file ownership

After w7-platnodes merges, all CREATE files are owned by w7-platnodes; later plans
(w7-route especially) consume `ResolveByPrefix` / the node service, never edit
(MAP decision 7).

**CREATE — store (NEW):**

| File | Contract |
|---|---|
| `internal/store/providernodes.go` | Node-scoped helpers over the providers table: `ListProviderNodes()` (filter to the node-type set), `GetProviderNodeByPrefix(prefix) (*ProviderRecord, error)`, `ListProviderNodePrefixes() ([]struct{Prefix, ID, APIType, BaseURL string}, error)`; the `nodeTypes` set (openai-/anthropic-compatible, custom-embedding). Reuses `ProviderRecord` + `scanProvider`. No init(); errors-as-values. |
| `internal/store/providernodes_test.go` | temp `store.Open`: create node rows with prefix/api_type → `ListProviderNodes` returns only node types; `GetProviderNodeByPrefix` returns the right row / ErrNotFound on miss; prefix uniqueness behavior asserted. RED first. |

**EXTEND — store (additive only):**

| File | Change (additive ONLY) |
|---|---|
| `internal/store/migrate.go` | ADD `ensureColumn("providers","prefix","TEXT NOT NULL DEFAULT ''")` + `ensureColumn("providers","api_type","TEXT NOT NULL DEFAULT ''")` to the additive-column loop. ADDITIVE ONLY. |
| `internal/store/providers.go` | ADDITIVE: extend `ProviderRecord` with `Prefix string` + `APIType string`; add them to the INSERT/SELECT/UPDATE column lists + `scanProvider`. Existing method signatures PRESERVED (callers compile unchanged; defaults `''`). |
| `internal/store/providers_test.go` (EXTEND additively, CREATE if absent) | RED first: create/update a provider with Prefix/APIType → round-trips; existing provider tests UNCHANGED-green. |

**CREATE — domain (NEW):**

| File | Contract |
|---|---|
| `internal/platform/providernodes.go` | `ProviderNodeService` over `*store.Store` (mirrors `ProxyPoolService`): `List/Create/Get/Update(+cascade)/Delete`, `Validate(req) (NodeProbeResult,error)` (injectable `NodeProber` + SSRF guard via `IsBlockedTarget`), `ResolveByPrefix(prefix) (providerID, baseURL, apiType string, ok bool)`. `SanitizeNodeBaseURL(apiType, raw) string` (pure). Constructor `NewProviderNodeService(st)`; `SetProber`/`SetResolver`. No init(); errors-as-values. |
| `internal/platform/providernodes_test.go` | SanitizeNodeBaseURL table (anthropic/embedding/openai); Validate ok/fallback/fail/SSRF-blocked via a fake prober (no network); ResolveByPrefix hit/miss/inactive; cascade re-points the providers-row base_url; create-with-key provisions a connection, create-without-key does not. RED first. |

**EXTEND — inference (ADDITIVE prefix hook):**

| File | Change (additive ONLY) |
|---|---|
| `internal/inference/alias.go` (or a new `internal/inference/noderesolve.go`) | ADD the `NodeResolver` interface (`ResolveByPrefix(prefix string) (providerID, baseURL, apiType string, ok bool)`). Prefer a NEW `noderesolve.go` to keep alias.go untouched; decide at T-hook. |
| `internal/inference/router.go` | ADDITIVE: a `nodeResolver NodeResolver` field + `SetNodeResolver` setter (mirror `SetAliasStore`) + the step-0 prefix-override block at the top of `Resolve` (§1.5). NO change to existing `Resolve` logic below step 0; NO `NewRouter` signature change. |
| `internal/inference/router_test.go` OR a new `internal/inference/noderesolve_test.go` | RED first: prefix→node override; same model without prefix→static fallthrough; node-vs-alias collision→node wins; nil nodeResolver→byte-identical (existing inference tests UNCHANGED-green). |

**EXTEND — transport (the w6-f file this plan owns going forward):**

| File | Change (ADDITIVE behavior; scope each body change precisely) |
|---|---|
| `internal/admin/nodes.go` | (1) persist prefix/api_type on `CreateProviderNode` (the thin body currently drops them — extend to set them + apply `SanitizeNodeBaseURL` + optional connection provision + `h.recordAudit`); (2) EXTEND the list filter to the node-type set; (3) ADD `GetProviderNode`/`UpdateProviderNode`(+cascade+audit)/`DeleteProviderNode`(+audit); (4) REPLACE the URL-shape-only `ValidateProviderNode` body with the real injectable probe (preserving the well-formed-URL→valid and never-persist-api_key invariants); (5) extend `providerNodeDTO` with `prefix`/`api_type`; route mutations through `h.providerNodes`. NEVER echo api_key. |
| `internal/admin/nodes_test.go` (EXTEND additively) | RED first for the NEW behavior: prefix/api_type persisted + listed; get/update(+cascade)/delete + 404; sanitization applied; validate probe ok/fallback/blocked (hermetic via `SetNodeProber`); create-with-key provisions a connection + no key leak; **all w6-f tests stay green**. EXTEND `TestNodesRouteDisambiguation` to cover validate-vs-`{id}`. |

**MODIFY — handlers wiring (additive only):**

| File | Change |
|---|---|
| `internal/admin/handlers.go` | ADDITIVE: add `providerNodes *platform.ProviderNodeService` field; construct `platform.NewProviderNodeService(st)` in `New` (mirror `proxyPools`, `handlers.go:61`); add `SetNodeProber(p platform.NodeProber)` mirroring `SetProxyProber` (`handlers.go:98`). NO `New(...)` signature change. |

**MODIFY — server wiring (additive only):**

| File | Change |
|---|---|
| `internal/server/server.go` | ADDITIVE: inside the `if st != nil` block (near `infRouter.SetAliasStore(st)`, l.55), wire `infRouter.SetNodeResolver(<adapter over the node service/store>)`. ONE additive line + (if needed) a thin adapter. NO other change. |

**MODIFY — serial-slot route registration (additive only):**

| File | Change |
|---|---|
| `internal/server/routes_admin.go` | ADD the 3 `{id}` route lines (§1.9). NOTHING else. ONE commit. SERIAL SLOT — first holder; RELEASE to w7-route on close. |

**MODIFY — e2e mock corrections (mirror real Go, decision 1):**

| File | Change |
|---|---|
| `ui/e2e/mocks/handlers/nodes.ts` (BODY — `/api/provider-nodes*` branches ONLY) | `toNode`: add `prefix`/`api_type` to mirror Go (iff page reads them / harmless — ESC-MOCK-DTO). ADD a `\/api\/provider-nodes\/[^/]+$` GET/PUT/DELETE route mirroring the new Go `{id}` surface (iff a spec needs it — ESC-MOCK-CRUD; default add). Validate stays `{valid,error?}` unless the page reads `models` (ESC-MOCK-VALID). **DO NOT touch the `/api/models/test` + `/api/models/availability` branches** (w7-misc). |

**FORBIDDEN:** everything else. Explicitly: all pre-existing `internal/admin/*.go`
except the EXTENDED nodes.go + the additive handlers.go field/setter; all other
`internal/store/*.go` except providernodes.go (NEW) + the additive providers.go/
migrate.go; all other `internal/platform/*` except providernodes.go (NEW);
`internal/inference/*` except the ADDITIVE router.go hook + the NodeResolver
interface (noderesolve.go/alias.go); `internal/inference/selection.go` (w7-route's —
NEVER touch); `internal/inference/factory.go` providerForModel + the catalog
resolution (UNCHANGED); all provider constructors + `schemas.Provider` (no signature
change); all UI `ui/src/**` (FROZEN, decision 8); the `/api/models/*` mock branches;
all other mocks/seeds/specs; `ui/package.json` + lockfile; `ui/vite.config.ts`;
`ui/playwright.config.ts`; `ui/dist/**` (gitignored — NEVER stage, NEVER revert
`ui/dist/index.html`). Touching any of these is an automatic REJECT.

---

## 4. TDD tasks

Cadence (strict, AGENTS.md "TDD always"): **no Go impl behavior may exist before its
`_test.go` is committed RED.** `go test ./... && go vet ./... && go build ./...`
green at EVERY commit. The w6-f nodes tests stay green throughout (new behavior is
additive). Order: schema+store → sanitize+validate+CRUD domain → admin extend →
prefix hook → routes serial slot → mock corrections → closeout.

### T-schema — STEP(a) RED, STEP(b) impl
STEP(a): write `internal/store/providernodes_test.go` + the additive
`providers_test.go` Prefix/APIType case; add the two `ensureColumn`s to `migrate.go`
(so tests compile + columns exist). `go test ./internal/store/ -run 'Provider|Node'`
→ FAIL. Commit RED: `phase-1/w7-platnodes: failing provider-node store tests (TDD red)`.
STEP(b): extend `ProviderRecord` + INSERT/SELECT/UPDATE/scan (additive); implement
`internal/store/providernodes.go`. Gates green. Commit:
`phase-1/w7-platnodes: provider-node store (additive prefix/api_type cols + node helpers)`.

### T-domain — STEP(a) RED, STEP(b) impl
STEP(a): write `internal/platform/providernodes_test.go` (SanitizeNodeBaseURL table;
Validate ok/fallback/fail/SSRF-blocked via fake NodeProber; ResolveByPrefix
hit/miss/inactive; cascade; create-with/without-key provision). → FAIL. Commit RED:
`phase-1/w7-platnodes: failing provider-node domain tests (TDD red)`.
STEP(b): implement `internal/platform/providernodes.go` (service + SanitizeNodeBaseURL
+ injectable NodeProber + IsBlockedTarget SSRF guard + cascade + ResolveByPrefix).
Gates green. Commit:
`phase-1/w7-platnodes: provider-node domain (sanitize + hermetic probe + cascade + resolve)`.

### T-admin — STEP(a) RED, STEP(b) impl
STEP(a): EXTEND `internal/admin/nodes_test.go` with the NEW behavior (prefix/api_type
persist+list; get/update(+cascade)/delete+404; sanitize; hermetic validate via
`SetNodeProber`; create-with-key→connection + no key leak; route disambiguation incl
`{id}`). → FAIL. Commit RED:
`phase-1/w7-platnodes: failing provider-node admin tests (TDD red)`.
STEP(b): EXTEND `internal/admin/nodes.go` (persist + sanitize + provision + audit +
Get/Update/Delete + real validate + DTO prefix/api_type) + the additive handlers.go
field/`SetNodeProber`. Gates green. Commit:
`phase-1/w7-platnodes: provider-node admin CRUD + real validate + cascade`.

### T-hook — STEP(a) RED, STEP(b) impl (prefix-override resolution)
STEP(a): write the inference prefix-override test (`router_test.go` /
`noderesolve_test.go`): prefix→node override; no-prefix→static fallthrough;
node-vs-alias→node wins; nil nodeResolver→identical. → FAIL. Commit RED:
`phase-1/w7-platnodes: failing prefix-override resolution tests (TDD red)`.
STEP(b): add the `NodeResolver` interface + `Router.SetNodeResolver` + the step-0
block in `Resolve` + the node-routed provider construction; wire
`infRouter.SetNodeResolver(...)` in `server.go`. Existing inference tests
UNCHANGED-green. Gates green. Commit:
`phase-1/w7-platnodes: node-prefix override before static alias/catalog (inference hook)`.

### T-routes — serial-slot route registration
TAKE the routes_admin.go serial slot (orchestrator confirms FREE at P5; first chain
holder). Add the 3 `{id}` route lines (§1.9). Gates green. Commit (ONE commit touches
the serial file):
`phase-1/w7-platnodes: register provider-node {id} CRUD routes (serial slot)`.

### T-mocks — mock-body corrections (mirror real Go, decision 1)
Correct `nodes.ts` `/api/provider-nodes*` branches (toNode prefix/api_type per
ESC-MOCK-DTO; add `{id}` route per ESC-MOCK-CRUD; validate per ESC-MOCK-VALID). DO
NOT touch `/api/models/*`. Gates: `cd ui && npm run build` green (BEFORE any
playwright); if a dedicated nodes spec exists run it ISOLATED, else run the spec(s)
that consume nodes.ts; full `npx playwright test` green. If a correction reds a
non-w7-platnodes spec, STOP + ESCALATE (§8 ESC-MOCK). Commit:
`phase-1/w7-platnodes: correct provider-node mock to mirror real Go DTOs`.

### T-close — full gates + closeout
```bash
go test ./... && go vet ./... && go build ./...
go test ./internal/store/ -run 'Provider|Node' -v
go test ./internal/platform/ -run 'Node|Prefix|Sanitize|Validate' -v
go test ./internal/admin/ -run 'Node' -v
go test ./internal/inference/ -run 'Node|Prefix|Resolve' -v     # existing resolution tests still green
cd ui && npm run build                                          # BEFORE playwright (e2e-hygiene)
cd ui && npx playwright test                                    # full suite green (ISOLATED; NEVER revert ui/dist/index.html)
```
Flip the matrix: `9router-routing.md` PAR-ROUTE-009/040 → HAVE; `9router-platform.md`
PAR-PLAT-010..014 → HAVE (cite §1.3-1.8). Append any new open items (§8 —
ESC-PROVISION/ESC-CASCADE follow-ups, /api/models/* w7-misc note) to
`open-questions.md`. Update `docs/WORKFLOW.md` (P6 base observation; the ESC-SCHEMA /
ESC-HOOK / ESC-NODE-PROVIDER / ESC-CASCADE / ESC-PROVISION decisions; the serial-slot
take/release; the mock corrections). Final commit:
`phase-1/w7-platnodes: close — provider-node prefix-routing engine; matrix flip; mock mirror`.
**On the close commit, RELEASE the routes_admin.go serial slot to w7-route**
(w7-platnodes is the routing prerequisite; w7-route may now start).

---

## 5. Binary acceptance criteria

All must hold; each is yes/no. `<base>` = the commit recorded at P0. Diff gate is
**w7-platnodes commit-range-scoped** (§7).

**Test gates**
- `go test ./... && go vet ./... && go build ./...` → exit 0.
- `go test ./internal/platform/ -run 'Node|Prefix|Sanitize|Validate' -v` → exit 0,
  all pass (sanitize ≥4; validate ≥5 incl SSRF-blocked + fallback; resolve ≥3;
  cascade; provision with/without key).
- `go test ./internal/store/ -run 'Provider|Node' -v` → exit 0 (prefix/api_type
  round-trip; node filter; GetProviderNodeByPrefix hit/miss; existing provider tests
  green).
- `go test ./internal/admin/ -run 'Node' -v` → exit 0 (all w6-f tests + new CRUD +
  hermetic validate + no-key-leak + route disambiguation incl `{id}`).
- `go test ./internal/inference/ -run 'Node|Prefix|Resolve' -v` → exit 0 (prefix
  override + existing resolution tests unchanged-green).
- `cd ui && npm run build` → exit 0 (BEFORE playwright).
- `cd ui && npx playwright test` → exit 0, no spec green-at-base goes red,
  `ui/dist/index.html` NEVER reverted.

**TDD-order proof** — each NEW impl file's covering test appears in an earlier-or-equal
commit:
```bash
for pair in \
  "internal/store/providernodes_test.go:internal/store/providernodes.go" \
  "internal/platform/providernodes_test.go:internal/platform/providernodes.go" ; do
  tf=${pair%%:*}; im=${pair##*:}
  ct=$(git log --format=%ct --diff-filter=A -1 -- "$tf")
  cf=$(git log --format=%ct --diff-filter=A -1 -- "$im")
  [ "$ct" -le "$cf" ] || echo "TDD VIOLATION: $im"     # prints nothing
done
# For EXTENDED files (nodes.go, router.go, providers.go), the RED-extension commit
# must precede or equal the impl commit per the §4 cadence (verify via the commit log).
```

**Grep proofs**
```bash
# prefix-override hook is additive + BEFORE static resolution
grep -n "SetNodeResolver\|nodeResolver\|ResolveByPrefix" internal/inference/router.go    # additive hook
grep -n "type NodeResolver" internal/inference/*.go                                       # interface present
grep -n "providerForModel\|ResolveModelAlias" internal/inference/router.go                # UNCHANGED order below step 0
# node store/domain
grep -n "func (s \*Store) ListProviderNodes\|GetProviderNodeByPrefix" internal/store/providernodes.go
grep -n "prefix\|api_type" internal/store/providers.go                                    # additive cols persisted
grep -n "prefix\|api_type" internal/store/migrate.go                                      # additive ensureColumn
grep -n "func NewProviderNodeService\|ResolveByPrefix\|SanitizeNodeBaseURL\|func (s \*ProviderNodeService) Validate" internal/platform/providernodes.go
# injectable probe + SSRF reuse (NOT a real-network test)
grep -n "NodeProber\|SetProber\|IsBlockedTarget" internal/platform/providernodes.go
grep -n "func (h \*Handlers) SetNodeProber" internal/admin/handlers.go
# sanitization (PAR-PLAT-011)
grep -nE "/messages|/embeddings|TrimSuffix" internal/platform/providernodes.go
# admin: audit on mutation + DTO prefix/api_type + never echo api_key
grep -n "recordAudit" internal/admin/nodes.go
grep -n "prefix\|api_type" internal/admin/nodes.go
! grep -niE 'json:"api_key"|"apiKey"' internal/admin/nodes.go | grep -iE 'DTO|response' && echo "no api_key in DTO OK"
# routes additive {id}
grep -nE '/api/provider-nodes/\{id\}' internal/server/routes_admin.go
# no init(); no signature change to NewRouter
! grep -rn "func init(" internal/store/providernodes.go internal/platform/providernodes.go && echo "no init() OK"
grep -n "func NewRouter" internal/inference/router.go    # signature UNCHANGED
```

**No-secret-exposure proofs (binding)**
```bash
# the node DTO has no api_key/secret field
grep -nA10 'type providerNodeDTO struct' internal/admin/nodes.go ; echo "^ must NOT contain api_key/secret"
# a provisioned node connection stores the key encrypted (*_enc), never plaintext column
grep -n "secret_enc\|cipher.Encrypt" internal/store/connections.go    # reused, unchanged path
# additive migrations only (no DROP/RENAME introduced by this plan)
git diff <base>..HEAD -- internal/store/migrate.go | grep -E '^\+' | grep -iE 'DROP COLUMN|RENAME COLUMN|DROP TABLE' | wc -l   # = 0
```
Plus a runtime no-leak assertion in `nodes_test.go`: marshal the validate + node CRUD
responses and assert none contains the supplied api_key.

**Negative / freeze proofs (w7-platnodes commit-range — §7)**
```bash
R="<first-w7-platnodes>^..<last-w7-platnodes>"
# Only the sanctioned Go/UI files changed:
git diff $R --name-only | grep -vE \
 'internal/store/(providernodes|providers|migrate)(_test)?\.go|internal/platform/providernodes(_test)?\.go|internal/inference/(router|alias|noderesolve)(_test)?\.go|internal/admin/(nodes|handlers)(_test)?\.go|internal/server/(server|routes_admin)\.go|ui/e2e/mocks/handlers/nodes\.ts|\.planning/parity/matrix/9router-(routing|platform)\.md|\.planning/parity/plans/open-questions\.md|docs/WORKFLOW\.md' \
 | wc -l                                                                  # = 0
# w7-route's selection.go untouched:
git diff $R --name-only -- internal/inference/selection.go | wc -l       # = 0
# providerForModel / catalog resolution untouched (factory.go not in the list):
git diff $R --name-only -- internal/inference/factory.go | wc -l         # = 0
# UI src frozen:
git diff $R --name-only -- ui/src/ | wc -l                               # = 0
# routes_admin.go = exactly ONE commit, additive:
git log --oneline $R -- internal/server/routes_admin.go | wc -l          # = 1
git diff $R -- internal/server/routes_admin.go | grep -E '^-' | grep -v '^---' | wc -l   # = 0 (no deletions)
# /api/models/* mock branches untouched:
git diff $R -- ui/e2e/mocks/handlers/nodes.ts | grep -E '^[-+]' | grep -iE '/api/models' | wc -l   # = 0
```

---

## 6. Out of scope (restated, binding)

No UI src edits (decision 8 — pages/components/routes/stores frozen); only the
sanctioned `nodes.ts` `/api/provider-nodes*` mock-body correction. No `/api/models/*`
backend or mock edits (w7-misc owns them). No edits to w7-route's selection.go
weighted-selection logic (w7-platnodes touches router.go, a different file). No
rewrite of `providerForModel`/`ResolveModelAlias`/`InferProvider` — the node hook is
an additive pre-step in `Resolve`. No edits to pre-existing admin handlers other than
the sanctioned additive nodes.go extension + the handlers.go field/setter. No
interface/constructor signature change (`schemas.Provider`, provider `New()`,
`NewRouter`, admin `New(...)`). No destructive DDL — additive `ensureColumn` only. No
new global state. No secret exposure (provisioned node api_key `*_enc`/never echoed).
Mock-vs-Go contradiction → escalate (§8), never fudge a mock or edit a frozen handler.

## 7. Diff-gate scope

W7 plans commit to main concurrently, so a broad `<base>..HEAD` range sweeps in
sibling commits. The diff gate MUST be scoped to w7-platnodes's own commits:
`git log --oneline main | grep "^[0-9a-f]* phase-1/w7-platnodes:" | awk '{print $1}'`
then `git diff <first-w7-platnodes>^..<last-w7-platnodes> -- [file list]`.

`git diff <first>^..<last> --name-only` must be exactly a subset of:
```
internal/store/providernodes.go
internal/store/providernodes_test.go
internal/store/providers.go              (additive Prefix/APIType + col lists)
internal/store/providers_test.go
internal/store/migrate.go                (additive ensureColumn prefix/api_type)
internal/platform/providernodes.go
internal/platform/providernodes_test.go
internal/inference/noderesolve.go        (NodeResolver interface — OR alias.go if chosen)
internal/inference/noderesolve_test.go   (OR router_test.go)
internal/inference/router.go             (additive nodeResolver hook; no NewRouter sig change)
internal/admin/nodes.go                  (additive extension of the w6-f file)
internal/admin/nodes_test.go             (additive extension)
internal/admin/handlers.go               (additive providerNodes field + SetNodeProber)
internal/server/server.go                (additive SetNodeResolver wire)
internal/server/routes_admin.go          (serial-slot additive {id} routes; ONE commit)
ui/e2e/mocks/handlers/nodes.ts           (body only — /api/provider-nodes* branches; NOT /api/models/*)
.planning/parity/matrix/9router-routing.md
.planning/parity/matrix/9router-platform.md
.planning/parity/plans/open-questions.md
docs/WORKFLOW.md
```
Any file outside this list in the scoped diff is an automatic review REJECT.
`internal/inference/selection.go`, `internal/inference/factory.go`, the pre-existing
admin handlers, provider constructors, and all `ui/src/**` are deliberately ABSENT —
touching them is an automatic REJECT. The `routes_admin.go` edit must appear in
exactly ONE commit (§5) and the serial slot is released to w7-route on close.

## 8. Escalations / decisions (explicit — recommended defaults, do not fabricate)

- **ESC-SCHEMA (RESOLVED at authoring — additive columns over a new JSON-data table,
  binding default).** Continue w6-f's providers-table mapping with two ADDITIVE typed
  columns (`prefix`, `api_type`) rather than the 9router `providerNodes(...,data)` JSON
  table. Justification §1.3 (avoids churning w6-f + the providers→connections FK;
  enables prefix lookup + cascade; additive-only). Alternative (faithful JSON-data
  table) duplicates wiring and breaks w6-f. RECOMMENDED: additive columns. Flag for
  orchestrator confirm; proceed on the default.
- **ESC-HOOK (RESOLVED at authoring — hook point + precedence, binding default).** The
  node-prefix override is the FIRST step of `Router.Resolve` (`router.go:53`), BEFORE
  `ResolveModelAlias` and `providerForModel`; node-prefix BEFORE static alias/catalog
  (PAR-ROUTE-009 "override static alias resolution"). Additive (`SetNodeResolver` +
  step-0 block; no `NewRouter` sig change; nil resolver → identical behavior).
  Alternative (hook inside `providerForModel`, factory.go) is rejected — it runs
  AFTER `ResolveModelAlias`, which would let a static alias pre-empt a node prefix,
  violating the override semantics. RECOMMENDED: Resolve step-0. Flag for confirm.
- **ESC-NODE-PROVIDER (CONDITIONAL — node-routed provider construction).** A node's
  bare model must be served by a generic/anthropic provider pointed at the node's
  base URL. RECOMMENDED = reuse the generic build path with the node base URL injected
  via the provider's existing base-URL seam; if no runtime base-URL seam exists
  without a signature change, build the node provider directly in the resolver
  (additive helper, NO change to `buildProvider`/provider constructors). Decide at
  T-hook by inspecting the generic provider's base-URL config; do NOT change any
  provider `New()` signature.
- **ESC-STORE-SHAPE (RESOLVED at authoring).** Extend `ProviderRecord` additively with
  `Prefix`/`APIType` + update the shared INSERT/SELECT/UPDATE/scan; keep node-specific
  reads in the new `providernodes.go`. Defaults `''` keep all existing providers
  callers green. RECOMMENDED as stated.
- **ESC-CASCADE (CONDITIONAL — what the update cascade rewrites).** A node IS a
  providers row; connections bind by `provider_id` and store NO base URL
  (`connections.go:13-26`), so a base-URL change propagates transitively via the
  providers row — the cascade is primarily the providers-row update. VERIFY at
  P-check that no `Connection.Metadata` caches the node prefix/base URL; if one does,
  add an additive `RepointConnectionsForNode`. RECOMMENDED: providers-row update +
  documented note; escalate only if a connection field mirrors node data.
- **ESC-PROVISION (CONDITIONAL — create auto-provisions a connection).** The brief's
  "creating a node provisions/links a connection." RECOMMENDED = on create, if the
  request carries `api_key`, auto-create a bound `api_key` connection
  (`provider_id=node.ID`, encrypted at rest, never echoed); without a key, node-only.
  VERIFY against 9router `route.js` create behavior at T-admin; if 9router does NOT
  auto-provision, default to node-only + a follow-up note in open-questions. Unit-test
  both branches.
- **ESC-MOCK-DTO / ESC-MOCK-CRUD / ESC-MOCK-VALID (CONDITIONAL — mirror Go, decision 1).**
  Resolve at P4 by grepping `ui/src` for what the FROZEN node page actually consumes:
  add `prefix`/`api_type` to the mock `toNode` (default: mirror Go), add the `{id}`
  GET/PUT/DELETE mock route (default: add to mirror the new Go surface), and decide
  whether validate carries a `models` list (default: keep `{valid,error?}`). If a
  correction reds a non-w7-platnodes spec → ESC-MOCK.
- **ESC-MOCK (CONDITIONAL — shared mock ripple).** `nodes.ts` is consumed by node/
  provider specs. w7-platnodes edits ONLY its `/api/provider-nodes*` branches (NEVER
  `/api/models/*`). If a correction reds a non-w7-platnodes spec, STOP and ESCALATE
  for orchestrator serialization — no fudge, no frozen-branch edit.
- **ESC-ROUTE (CONDITIONAL — fasthttp/router precedence).** `/api/provider-nodes/
  validate` (static) vs `/api/provider-nodes/{id}` (param) follow the file's existing
  static-before-param ordering (the providers `/{id}/catalog` precedent
  `routes_admin.go:131`). The EXTENDED `TestNodesRouteDisambiguation` proves it. If
  the matcher mis-disambiguates, STOP and ESCALATE for a path arrangement — never
  silently diverge page/mock/Go.
- **Serial-slot dependency (§1.9 / P5).** w7-platnodes is FIRST in the routes_admin.go
  chain (MAP §219-238); the slot is free at P-check. RELEASE it to w7-route on close.
  Orchestrator confirms exactly one unmerged holder (decision 3) and that w7-route
  has NOT started before T-routes.
- **No other blocking dependency.** All reused surfaces (providers.go, router.go
  Resolve, ParseModelPrefix, platform ProxyPoolService/IsBlockedTarget templates,
  recordAudit, respond.go, newTestEnv, migrate additive pattern) are in-tree at
  <base>. w7-platnodes is unblocked immediately; it is the routing prerequisite for
  w7-route.
```
