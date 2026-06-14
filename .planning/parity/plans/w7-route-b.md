# Micro-plan w7-route-b — Dynamic routing engine: weighted selection + free-conn + proxy-resolve verify + live catalog override + web pseudo-models + upstream detection + project-ID cold-miss + multi-URL fallback (Go)

```
wave: 7
plan: w7-route-b  (split half B of the original w7-route; see §0 SPLIT NOTE)
status: READY (rev 1 — authored against merged Waves 0–6 + the shipped W7
  governance/platform/platnodes plans, live tree @ <base>; WAVE-7-MAP w7-route row
  ~line 174; selection.go micro-serial §234-238 / §267; reconciliation §245)
runs: governance+routing track (engine half). NO routes_admin.go serial slot — all
  surfaces are /v1/models + inference-internal, NOT admin routes. SECONDARY
  micro-serial: ADDITIVE edits to internal/inference/selection.go (weighted
  selection); w7-plat-1 ALSO edited selection.go (proxy hook — MERGED). The
  orchestrator confirms exactly one unmerged selection.go holder before T-weighted
  (MAP §234-238 / §267). Runs ∥ w7-route-a (zero shared Go files — §7).
  Depends on w7-platnodes (merged) + w7-plat-1 (merged — ProxyResolver hook).
branch: main (direct commits, per repo convention)
commit-prefix: phase-1/w7-route-b:
ref-source: 9router frozen @ 827e5c3 — weighted selection (src/sse/services/
  auth.js), free no-auth virtual connection (src/sse/services/auth.js:36-53,
  src/shared/constants/providers.js:14), live model catalog override + web
  search/fetch pseudo-models + upstream-connection detection (src/app/api/v1/
  models/route.js:16-38,46,282-299,386-401), project-ID cold-miss (src/sse/
  handlers/chat.js:191-198), multi-URL fallback (open-sse/services/provider.js:
  155-209). These are INFERENCE-PATH semantics (NO UI contract / NO mock).
base: <base> = git rev-parse HEAD recorded at P0. Substitute the actual SHA
  everywhere §5 says <base>.
selection-micro-serial: this plan's selection.go edit (weighted selection) is
  ADDITIVE-ONLY. w7-plat-1's selection.go proxy hook is MERGED. The orchestrator
  confirms no OTHER unmerged selection.go holder before T-weighted.
new-route: NO UI route files, NO admin routes, NO new mock. The matrix-row flips
  are inference-engine PAR-ROUTE rows proven by HERMETIC Go unit tests.
```

---

## 0. SPLIT NOTE (binding)

w7-route-b is the **dynamic routing engine** half of the original w7-route (see
`w7-route-a.md` §0 for the full split rationale). It is **file-disjoint** from
w7-route-a (admin CRUD + quota): w7-route-b touches ONLY `internal/inference/*`,
`internal/server/routes_openai.go` (`ModelsHandler`), and `internal/providers/*` —
NONE of which w7-route-a touches; w7-route-a touches `internal/admin/*` +
`internal/store/*` admin tables + `routes_admin.go`, NONE of which w7-route-b touches.
The two run **in parallel**. w7-route-b takes **no routes_admin slot** (its surfaces
are not admin routes) but DOES coordinate the **selection.go micro-serial** vs
w7-plat-1 (whose proxy hook is already merged — verify no other unmerged holder).

---

## 1. Scope — the eight PAR-ROUTE engine rows

### Rows this plan closes (each MISSING/PARTIAL → HAVE, hermetic-test-proven)

| Row | Claim (9router ref) | Target after w7-route-b |
|---|---|---|
| **PAR-ROUTE-027** | Weighted provider selection (`auth.js` strategy) | HAVE — additive `weighted` strategy branch in `selection.go` `SelectConnection` (§1.4) |
| **PAR-ROUTE-039** | Free provider no-auth virtual connection injection (`auth.js:36-53`, `providers.js:14`) | HAVE — synthetic connection for `noAuth` providers, additive in the connection-enumeration seam (§1.5) |
| **PAR-ROUTE-055** | Proxy-pool resolution per connection (`chatCore.js:151-182`) | HAVE — VERIFY w7-plat-1's `ResolveProxy` hook covers this; extend only if a gap (§1.6) |
| **PAR-ROUTE-056** | Live model catalog override (Kiro/Qoder dynamic models) (`models/route.js:16-38,289-299`) | HAVE — additive live-catalog adapter seam on `ModelsHandler.List` (§1.7) |
| **PAR-ROUTE-059** | Web search/fetch pseudo-models (`{alias}/search`, `{alias}/fetch`) (`models/route.js:386-401`) | HAVE — additive pseudo-model adapter on `ModelsHandler.List` (§1.8) |
| **PAR-ROUTE-060** | Upstream-connection detection (UUID suffix; skip live fetch) (`models/route.js:46,282-284`) | HAVE — pure `UPSTREAM_CONNECTION_RE` guard gating the live-catalog fetch (§1.9) |
| **PAR-ROUTE-053** | Project-ID cold-miss resolution (antigravity/gemini-cli) (`chat.js:191-198`) | HAVE — additive project-ID resolve+persist seam in the inference request path (§1.10) |
| **PAR-ROUTE-035** | Provider-specific URL building with fallback URLs (PARTIAL) (`provider.js:155-209`) | PARTIAL → HAVE — index-based fallback URL list in the generic chat URL builder (§1.11) |

Matrix flips at closeout (§4 T-close): in `.planning/parity/matrix/9router-routing.md`,
PAR-ROUTE-027/039/053/055/056/059/060 MISSING → HAVE; PAR-ROUTE-035 PARTIAL → HAVE
(each with a cite to the covering hermetic test). NO UI matrix rows (no UI contract).

### 1.1 Preconditions already satisfied by merged waves (evidence)

- **selection.go is NOW the sole unmerged-holder candidate** — w7-platnodes (prefix
  routing) + w7-plat-1 (proxy hook) are MERGED. The current `selection.go` (read at
  authoring) contains:
  - `ProxyResolver` interface + `SetProxyResolver` + `ResolveProxy(conn)`
    (`selection.go:36-80`, w7-plat-1) — PAR-ROUTE-055's machinery already in tree.
  - `SelectionEngine` + `NewSelectionEngine(cs,ss,cd,clock)` (`selection.go:51,61`).
  - `resolveStrategy(providerID) (strategy, stickyLimit, err)` (`selection.go:90`)
    reading `providerStrategies`/`fallbackStrategy`/`stickyRoundRobinLimit` settings;
    default `"fill-first"`; supports `"round-robin"` (sticky) (`selection.go:127-161`).
  - `SelectConnection(providerID, model, exclude, preferredConnID)`
    (`selection.go:159`) — the `switch strategy` block (`selection.go:127`) is the
    ADDITIVE-extension point for `weighted` (§1.4).
  - `WithAccountFallback` (`selection.go:253`) drives the fallback loop.
- **The models-list assembly + additive-adapter seam pattern EXISTS** —
  `internal/server/routes_openai.go` `ModelsHandler` (`routes_openai.go:63`) with
  setter-injected adapters: `SetDisabledChecker`/`SetComboLister`/`SetCustomModelLister`/
  `SetAliasModelLister`/`SetSubConfigModelReader` (`routes_openai.go:68-88`), consumed
  by `List` (`routes_openai.go:93`). This is the EXACT additive pattern for the live
  catalog override (056), web pseudo-models (059), and upstream-detection guard (060)
  — ADD new adapter interfaces + setters; `List` composes them. **The brief's
  "factory.go:104" cite is `providerForModel`/the `catalog.ModelsFor` precedence loop
  (`factory.go`); the models-LIST surface is `ModelsHandler.List` — that is the
  correct hook for 056/059/060.**
- **The generic chat URL builder EXISTS** — `internal/providers/generic/chat.go`
  `chatURL()` (single URL; PAR-ROUTE-035 PARTIAL note: "no index-based fallback URL
  list"). §1.11 adds the additive fallback-URL list.
- **The node prefix resolver EXISTS** — `internal/inference/noderesolve.go`
  `NodeResolver` + `buildNodeProvider` (w7-platnodes) — the precedent for an
  inference-side seam wired in `server.go` to avoid an inference→platform import
  cycle. Reuse this seam pattern for any platform-backed resolver (e.g. proxy/project-
  ID) needed here.
- **No UI / mock contract** — all eight rows are inference-engine semantics with NO
  dashboard surface. The authoritative proof is HERMETIC Go unit tests (the matrix
  rows cite 9router source, not a mock). Determinism is mandatory: any network
  (live-catalog fetch, project-ID fetch, web search) is behind an INJECTABLE seam
  (interface/fake), NEVER a live call in a test (AGENTS.md "No mocks; use interfaces
  and fakes").
- **Admin test harness is NOT the surface here** — these are inference-package unit
  tests (`internal/inference/*_test.go`, `internal/server/routes_openai_test.go`,
  `internal/providers/generic/chat_test.go`). Existing tests must stay green.

### 1.2 Per-row determinism + seam plan (binding — the hermetic core)

Every row that touches I/O gets an injectable seam so tests are network-free:

| Row | I/O? | Injectable seam | Test approach |
|---|---|---|---|
| 027 weighted | none | n/a (pure selection over in-memory conns) | table test over fake `ConnStore`/`SettingStore` (existing test fakes) |
| 039 free-conn | none | a `freeProviderSet` constant + synthetic-conn builder | unit: noAuth provider → synthetic conn injected; non-noAuth → none |
| 055 proxy | already merged | w7-plat-1 `ProxyResolver` | VERIFY existing `selection_test.go` proxy cases; extend if a gap |
| 056 live catalog | network | `LiveCatalogResolver` interface (fake in tests) | List with a fake resolver → dynamic models merged; resolver error → static-only |
| 059 web pseudo-models | none | reads provider config (no net) | provider with search/fetch config → `{alias}/search`+`{alias}/fetch` appear |
| 060 upstream detect | none | pure `UPSTREAM_CONNECTION_RE` | regex table: UUID-suffixed name → skip live fetch; normal → fetch |
| 053 project-ID | network | `ProjectIDResolver` interface (fake) | cold-miss → resolver called + persisted; warm → no call |
| 035 fallback URLs | none | index over a URL list | builder returns url[0], url[1]… by index |

### 1.3 Architecture (binding — additive-only, decision 4/9)

```
027 weighted:     inference/selection.go  (ADDITIVE `case "weighted"` in SelectConnection switch)
039 free-conn:    inference/selection.go OR a new inference/freeconn.go  (synthetic conn in the enumeration seam)
055 proxy:        inference/selection.go  (VERIFY w7-plat-1 ResolveProxy; extend only if gap)
056 live catalog: server/routes_openai.go ModelsHandler (ADDITIVE adapter + setter) + inference/livecatalog.go (resolver, fake-injectable)
059 web pseudo:   server/routes_openai.go ModelsHandler (ADDITIVE adapter + setter) reading provider search/fetch config
060 upstream:     server/routes_openai.go ModelsHandler (ADDITIVE pure-regex guard gating the 056 fetch) + inference/upstream.go (pure UPSTREAM_CONNECTION_RE)
053 project-ID:   inference/projectid.go (resolver, fake-injectable) + an ADDITIVE call in the request path (factory.go/runner.go — the build-provider site, NOT selection logic)
035 fallback URL: providers/generic/chat.go (ADDITIVE index-based URL list in chatURL)
```

All edits are **additive-only**: new functions/files/interfaces + optional setters
(like `SetProxyResolver`); NO existing selection/eligibility/cooldown/URL logic is
rewritten; NO interface or `NewSelectionEngine`/`SetNetworkConfig` signature change
(MAP decision 9 — optional setters only). The selection.go edit is the single
micro-serial-coordinated file (vs w7-plat-1, merged).

### 1.4 PAR-ROUTE-027 weighted provider selection (selection.go ADDITIVE)

9router selects connections by a weight per connection (auth.js strategy). g0router's
`resolveStrategy` already resolves a strategy string (`fill-first`/`round-robin`);
ADD a `"weighted"` branch to the `SelectConnection` `switch strategy` block
(`selection.go:127`). Weight source (DECIDE — §8 ESC-WEIGHT-SRC): RECOMMENDED default
= a per-connection weight read from the connection's `Metadata` JSON (e.g.
`{"weight": N}`), defaulting to 1 when absent — NO schema change (the `Connection`
struct + table are frozen; Metadata is the existing free-form seam,
`connections.go:22`). Selection: deterministic weighted pick over eligible
connections. **Determinism in tests:** seed the engine `clock`/a stable tiebreak so
the weighted pick is reproducible (e.g. weighted round-robin by accumulated weight,
NOT `math/rand` — a pure accumulator is deterministic and testable). Unit cases (fake
`ConnStore`/`SettingStore`): strategy `weighted` + two conns weighted 3:1 → over 4
selects, conn-A picked 3×, conn-B 1×; absent weight → equal; existing
fill-first/round-robin cases UNCHANGED-green.

### 1.5 PAR-ROUTE-039 free no-auth virtual connection (ADDITIVE)

9router injects a synthetic connection for `noAuth` providers (e.g. opencode) so a
request to a free provider routes without a stored credential (`auth.js:36-53`,
`providers.js:14`). ADD: a `freeProviderSet` (the noAuth provider IDs, sourced from
the catalog's noAuth flag if present, else a constant mirroring `providers.js:14` —
§8 ESC-FREE-SET) + a synthetic-connection builder invoked in the connection-
enumeration path when a free provider has NO real eligible connection. The synthetic
conn carries the provider ID, a sentinel name, `Kind:"api_key"` with an empty/none
secret (the free provider needs no auth). **Additive site:** prefer a small helper in
`selection.go` (or a new `inference/freeconn.go`) that `SelectConnection` consults
AFTER the eligible-list build yields empty for a free provider — NOT a rewrite of the
eligibility loop. Unit: free provider, zero stored conns → synthetic conn returned;
non-free provider, zero conns → existing "no eligible connections" error UNCHANGED.

### 1.6 PAR-ROUTE-055 proxy-pool resolution per connection (VERIFY merged)

w7-plat-1 merged `ProxyResolver`/`SetProxyResolver`/`ResolveProxy` into selection.go
and the per-connection proxy wiring (`connections.proxy_pool_id` + `ClientPool`
override). **w7-route-b VERIFIES this covers PAR-ROUTE-055** (the matrix row may
already be flippable by w7-plat-1's PAR-PLAT-009). At T-proxy-verify: run the existing
proxy selection tests; confirm `ResolveProxy` returns the pool URL for a connection
with a `proxy_pool_id`. If a gap remains specific to the ROUTE-055 semantics
(`connectionProxyEnabled`/`vercelRelayUrl` from `providerSpecificData` — a per-
connection flag/relay-URL the proxy hook doesn't read), ADD only the missing additive
read (in `selection.go` `ResolveProxy` extension OR the connection metadata parse) —
NEVER a rewrite. If fully covered, flip PAR-ROUTE-055 → HAVE citing w7-plat-1 +
w7-route-b verification (no code). §8 ESC-055-OVERLAP.

### 1.7 PAR-ROUTE-056 live model catalog override (ModelsHandler ADDITIVE)

Kiro/Qoder expose dynamic per-account model lists fetched live (`models/route.js:
16-38,289-299`). ADD to `ModelsHandler` a `LiveCatalogResolver` adapter
(interface + `SetLiveCatalogResolver` setter, mirroring `SetCustomModelLister`
`routes_openai.go:78`) consulted in `List` AFTER the static/catalog/custom/alias merge.
The resolver fetches per-account dynamic models for providers that support it (Kiro/
Qoder), **behind an injectable interface** so production wires a real fetcher (HTTP)
and tests inject a fake returning fixed models (NO network). Merge dedup follows the
existing seen-set order (`models/route.js:358` precedent already mirrored in `List`).
The live fetch is gated by the upstream-detection guard (060, §1.9) so upstream
connections skip it. Unit (`routes_openai_test.go`): fake resolver returns 2 dynamic
models → `List` includes them; resolver error → static-only, no failure; upstream
connection → resolver NOT called.

### 1.8 PAR-ROUTE-059 web search/fetch pseudo-models (ModelsHandler ADDITIVE)

When a provider has web search/fetch config, 9router exposes `{alias}/search` +
`{alias}/fetch` as pseudo-models (`models/route.js:386-401`). ADD a pseudo-model
adapter to `ModelsHandler.List` (interface + setter) that reads the provider's
search/fetch config (from provider/connection config — NO network) and appends the
two pseudo-model IDs per configured alias. **No live web call here** — this only
EXPOSES the pseudo-models in the list; actually SERVING `{alias}/search`/`{alias}/fetch`
requests (the web-search/fetch execution) is a LARGER, network-bound, fragile concern
(reverse-engineered web endpoints) → **scope w7-route-b to the pseudo-model EXPOSURE +
an injectable execution seam stub; record the live web-execution as an escalation/
follow-up** (§8 ESC-WEB-EXEC — never fabricate live web calls). Unit: provider with
search config → `{alias}/search`+`{alias}/fetch` in the list; provider without →
absent.

### 1.9 PAR-ROUTE-060 upstream-connection detection (pure regex guard)

`UPSTREAM_CONNECTION_RE` (a UUID-suffix pattern) marks upstream connections whose
live model fetch is skipped (`models/route.js:46,282-284`). ADD a pure
`internal/inference/upstream.go` `IsUpstreamConnection(name string) bool` (the regex,
mirroring the ref) + use it in `ModelsHandler.List` to GATE the 056 live-catalog
fetch (skip the resolver for upstream connections). Fully deterministic, no I/O.
Unit table: UUID-suffixed names → true (skip fetch); normal names → false (fetch).

### 1.10 PAR-ROUTE-053 project-ID cold-miss resolution (ADDITIVE seam)

For antigravity/gemini-cli, if the connection lacks a project ID, 9router fetches it
and persists to DB on the first (cold) request (`chat.js:191-198`). ADD an
`internal/inference/projectid.go` `ProjectIDResolver` interface (fake-injectable) +
an ADDITIVE call at the provider-build site (`factory.go`/`runner.go` — where the
provider instance is built and credentials are resolved, NOT inside selection
logic): if the connection's project ID is missing for a provider that needs it,
resolve via the resolver and persist (best-effort) to the connection's Metadata (the
existing free-form seam — NO schema change). Production wires a real resolver; tests
inject a fake. Unit: cold-miss (no project ID) → resolver called once + persisted;
warm (project ID present) → resolver NOT called; non-antigravity/gemini provider →
no resolution. §8 ESC-PROJ-PERSIST (Metadata vs a new column — default Metadata, no
migration).

### 1.11 PAR-ROUTE-035 multi-URL fallback (generic chat.go ADDITIVE)

The generic `chatURL()` returns a single URL (`chat.go:20`); the ref builds an
index-based fallback URL list (`provider.js:155-209`, `base.js:20-42`). ADD an
additive `chatURLs() []string` (or an index param) returning the ordered candidate
URL list (primary + fallbacks from the provider/catalog config); the existing
`chatURL()` stays as `chatURLs()[0]` for callers that want one. The fallback CONSUMPTION
(retry on the next URL when one fails) wires into the existing request/retry path
ADDITIVELY (if the retry loop is in a frozen handler, scope w7-route-b to the
URL-LIST builder + a unit test, and record the live-retry wiring as a follow-up if it
would touch a frozen body — §8 ESC-035-RETRY). Unit (`chat_test.go`): a provider
with N configured URLs → `chatURLs()` returns them in order; single-URL provider → a
one-element list; existing `chatURL()` callers UNCHANGED-green.

### NOT in scope (explicit)

- **No admin CRUD / no admin routes / no routes_admin.go** — that is w7-route-a. No
  `internal/admin/*` edits, no new mock, no UI.
- **No UI src / page / mock / seed edits** — these rows have no dashboard contract.
- **No rewrite of existing selection/eligibility/cooldown/URL/retry logic** — every
  edit is additive (new branch/function/file/setter). selection.go is micro-serial-
  coordinated vs w7-plat-1.
- **No interface / `NewSelectionEngine` / `SetNetworkConfig` signature change** —
  optional setters + new interfaces only (decision 9).
- **No live web search/fetch execution** (059) — pseudo-model EXPOSURE only; live
  execution is an escalation/follow-up (ESC-WEB-EXEC).
- **No live multi-URL retry wiring if it touches a frozen handler body** (035) —
  URL-list builder + unit; retry wiring follow-up if frozen (ESC-035-RETRY).
- **No schema change** — weights/project-ID use the existing `Connection.Metadata`
  seam; NO new columns/tables (decision 2).
- **No network in tests** — all I/O behind injectable fakes (binding).

---

## 2. Precondition checks

```bash
# P0 — clean tree; record <base>
git status --porcelain     # must be empty (explicit git add; ui/dist gitignored)
git rev-parse HEAD         # record as <base> for §5

# P1 — the engine rows are REAL gaps (no existing impl)
grep -nF 'case "weighted"' internal/inference/selection.go ; echo "^ expect EMPTY (027 not yet)"
grep -rniE 'noAuth|freeProvider|free provider|synthetic conn' internal/inference/ ; echo "^ expect EMPTY (039)"
grep -rniE 'LiveCatalog|UPSTREAM_CONNECTION_RE|IsUpstream|projectid|ProjectID|search/fetch|pseudo' internal/inference/ internal/server/routes_openai.go ; echo "^ expect EMPTY (056/059/060/053)"
grep -nF 'func chatURLs' internal/providers/generic/chat.go ; echo "^ expect EMPTY (035)"

# P2 — reused merged seams present (the de-risk)
grep -nF 'ProxyResolver' 'func (e *SelectionEngine) ResolveProxy' 'func (e *SelectionEngine) SetProxyResolver' internal/inference/selection.go   # w7-plat-1, merged
grep -nF 'switch strategy' 'func (e *SelectionEngine) SelectConnection' 'func (e *SelectionEngine) resolveStrategy' internal/inference/selection.go
grep -nF 'SetCustomModelLister' 'SetAliasModelLister' 'func (h *ModelsHandler) List' internal/server/routes_openai.go
grep -nF 'func (s *Store) ListConnections' internal/store/connections.go
grep -nF 'Metadata' internal/store/connections.go   # the free-form weight/project-id seam
grep -nF 'NodeResolver' internal/inference/noderesolve.go   # the inference-side seam precedent
grep -nF 'func chatURL' internal/providers/generic/chat.go

# P3 — selection.go micro-serial FREE
git log --oneline -5 -- internal/inference/selection.go   # last touch = w7-plat-1 (merged)
# Orchestrator MUST confirm no OTHER unmerged selection.go holder before T-weighted.

# P4 — green at base
go test ./... && go vet ./... && go build ./...     # exit 0 (Go untouched-green)
go test ./internal/inference/... -run 'Select|Proxy|Strategy' -v   # existing selection tests pass at base
```

---

## 3. Exclusive file ownership

**CREATE — inference (NEW):**

| File | Contract |
|---|---|
| `internal/inference/freeconn.go` (+_test) | `freeProviderSet` + synthetic-conn builder (039). RED first. |
| `internal/inference/upstream.go` (+_test) | pure `IsUpstreamConnection(name)` + `UPSTREAM_CONNECTION_RE` (060). RED first. |
| `internal/inference/livecatalog.go` (+_test) | `LiveCatalogResolver` interface + a default fetcher (injectable) (056). RED first. |
| `internal/inference/projectid.go` (+_test) | `ProjectIDResolver` interface + cold-miss resolve+persist (053). RED first. |

**EXTEND — inference/server/providers (ADDITIVE only):**

| File | Change (additive ONLY) |
|---|---|
| `internal/inference/selection.go` | ADD `case "weighted"` to the `SelectConnection` switch (027) + the free-conn consult after empty-eligible (039). VERIFY ResolveProxy (055). NO rewrite of existing logic. Micro-serial vs w7-plat-1. |
| `internal/inference/selection_test.go` (EXTEND additively) | weighted picks 3:1; free-conn synthetic; existing cases UNCHANGED-green. RED first. |
| `internal/inference/factory.go` (or `runner.go`) | ADDITIVE project-ID resolve call at the provider-build site (053). NO existing-logic rewrite. |
| `internal/server/routes_openai.go` | ADD `LiveCatalogResolver` adapter+setter (056), web-pseudo-model adapter+setter (059), upstream-gate on the live fetch (060) — all mirroring the existing `Set*Lister` additive pattern. `List` composes them additively. |
| `internal/server/routes_openai_test.go` (EXTEND additively) | live-catalog merge via fake; pseudo-models exposure; upstream skips fetch. RED first. |
| `internal/providers/generic/chat.go` | ADDITIVE `chatURLs() []string` index-based fallback list (035); `chatURL()` preserved as `chatURLs()[0]`. NO rewrite. |
| `internal/providers/generic/chat_test.go` (EXTEND additively) | multi-URL ordered list; single-URL one-element; existing UNCHANGED. RED first. |

**WIRING (server.go — ADDITIVE, only if a new resolver needs construction):**

| File | Change |
|---|---|
| `internal/server/server.go` (CONDITIONAL) | ADDITIVE: construct + `Set*` the new resolvers (live-catalog, project-ID) into `ModelsHandler`/the engine, mirroring how `NodeResolver` is wired (avoiding inference→platform cycles). NO signature change. Only if the resolver needs a real production wiring; tests use fakes directly. |

**FORBIDDEN:** everything else. Explicitly: all `internal/admin/*` + `internal/store/*`
admin tables + `routes_admin.go` (w7-route-a territory); the EXISTING combos engine +
gateway alias resolver; all UI `ui/src/**` + all mocks/seeds/specs (no UI contract);
`internal/inference/selection.go` existing selection/eligibility/cooldown logic
(ADDITIVE branches only); any interface/constructor signature; `ui/dist/**`. Touching
any of these is an automatic REJECT.

---

## 4. TDD tasks

Cadence (strict): **no impl before its `_test.go` is committed RED.**
`go test ./... && go vet ./... && go build ./...` green at EVERY commit. Existing
inference/models/generic tests stay green throughout (all edits additive). Order:
pure rows first (060, 035), then the seam rows (056, 059, 053), then selection.go
(027, 039 — micro-serial), then proxy-verify (055), then closeout.

### T-upstream (060) — pure regex
STEP(a) RED `upstream_test.go`. STEP(b) impl `upstream.go`. Commits:
`phase-1/w7-route-b: failing upstream-connection detection test (TDD red)` /
`phase-1/w7-route-b: upstream-connection detection (UPSTREAM_CONNECTION_RE)`.

### T-fallback (035) — generic chatURLs
STEP(a) RED `chat_test.go` additions. STEP(b) impl `chatURLs()` additive. Commits:
`…: failing multi-URL fallback test (TDD red)` / `…: multi-URL fallback URL list (chatURLs)`.

### T-livecatalog (056) + T-pseudo (059) — ModelsHandler adapters
STEP(a) RED `livecatalog_test.go` + `routes_openai_test.go` additions (fake resolver;
pseudo-models; upstream-gate uses 060). STEP(b) impl `livecatalog.go` + the
`ModelsHandler` adapters/setters + the upstream gate. Commits:
`…: failing live-catalog + web-pseudo-model + upstream-gate tests (TDD red)` /
`…: live model catalog override + web pseudo-models (ModelsHandler adapters)`.

### T-projectid (053) — resolver + build-site call
STEP(a) RED `projectid_test.go` (cold-miss/warm/non-applicable via fake). STEP(b)
impl `projectid.go` + the additive `factory.go`/`runner.go` call. Commits:
`…: failing project-ID cold-miss tests (TDD red)` /
`…: project-ID cold-miss resolution (resolver + persist seam)`.

### T-weighted (027) + T-freeconn (039) — selection.go MICRO-SERIAL
TAKE the selection.go micro-serial slot (orchestrator confirms no other unmerged
holder; w7-plat-1 merged). STEP(a) RED `selection_test.go` additions (weighted 3:1;
free-conn synthetic) + `freeconn_test.go`. STEP(b) impl the `case "weighted"` branch +
the free-conn consult + `freeconn.go`. Existing selection cases UNCHANGED-green.
Commits: `…: failing weighted-selection + free-conn tests (TDD red)` /
`…: weighted provider selection + free no-auth virtual connection (selection.go additive)`.
RELEASE the selection.go micro-serial slot.

### T-proxy-verify (055) — verify w7-plat-1 coverage
Run the existing proxy selection tests; confirm `ResolveProxy` covers ROUTE-055. If a
gap (connectionProxyEnabled/vercelRelayUrl), ADD the minimal additive read + a test;
else flip the matrix row citing w7-plat-1 + verification (no code). Commit (if code):
`…: extend per-connection proxy resolution for ROUTE-055 (connectionProxyEnabled/relay)`.

### T-close — full gates + closeout
```bash
go test ./... && go vet ./... && go build ./...
go test ./internal/inference/... -run 'Weighted|Select|FreeConn|Upstream|LiveCatalog|ProjectID|Proxy|Strategy' -v
go test ./internal/server/ -run 'Models|LiveCatalog|Pseudo|Upstream' -v
go test ./internal/providers/generic/ -run 'ChatURL|Fallback' -v
```
Flip `.planning/parity/matrix/9router-routing.md`: PAR-ROUTE-027/039/053/056/059/060
MISSING → HAVE; 035 PARTIAL → HAVE; 055 → HAVE (w7-plat-1 + verify), each citing the
covering hermetic test. Append the new open items (§8 — ESC-WEB-EXEC live web
execution; ESC-035-RETRY live retry wiring; any 055 gap). Update `docs/WORKFLOW.md`
(P4 base observation; the ESC-WEIGHT-SRC/ESC-FREE-SET/ESC-PROJ-PERSIST decisions; the
selection.go micro-serial take/release). Final commit:
`phase-1/w7-route-b: close — dynamic routing engine (027/035/039/053/055/056/059/060); matrix flips`.

---

## 5. Binary acceptance criteria

`<base>` = the commit recorded at P0. Diff gate is **w7-route-b commit-range-scoped**
(§7). NO routes_admin slot proof (this plan holds none).

**Test gates**
- `go test ./... && go vet ./... && go build ./...` → exit 0.
- `go test ./internal/inference/... -run 'Weighted|FreeConn|Upstream|LiveCatalog|ProjectID|Select|Proxy|Strategy' -v` → exit 0 (weighted 3:1; free-conn synthetic; upstream regex; live-catalog merge + error path; project-ID cold/warm; existing selection UNCHANGED).
- `go test ./internal/server/ -run 'Models|LiveCatalog|Pseudo|Upstream' -v` → exit 0.
- `go test ./internal/providers/generic/ -run 'ChatURL|Fallback' -v` → exit 0.
- NO live network in any test (asserted by hermetic seams; reviewer greps for
  `http.Get`/`http.Client{}` in new test files → none unguarded).

**TDD-order proof**
```bash
for pair in \
  "internal/inference/upstream_test.go:internal/inference/upstream.go" \
  "internal/inference/livecatalog_test.go:internal/inference/livecatalog.go" \
  "internal/inference/projectid_test.go:internal/inference/projectid.go" \
  "internal/inference/freeconn_test.go:internal/inference/freeconn.go" ; do
  tf=${pair%%:*}; im=${pair##*:}
  ct=$(git log --format=%ct --diff-filter=A -1 -- "$tf")
  cf=$(git log --format=%ct --diff-filter=A -1 -- "$im")
  [ "$ct" -le "$cf" ] || echo "TDD VIOLATION: $im"     # prints nothing
done
```

**Grep proofs**
```bash
grep -nF 'case "weighted"' internal/inference/selection.go            # 027
grep -nF 'IsUpstreamConnection' 'UPSTREAM_CONNECTION_RE' internal/inference/upstream.go   # 060
grep -nF 'LiveCatalogResolver' 'SetLiveCatalogResolver' internal/server/routes_openai.go internal/inference/livecatalog.go   # 056
grep -nF 'ProjectIDResolver' internal/inference/projectid.go          # 053
grep -nF 'func chatURLs' internal/providers/generic/chat.go           # 035
grep -niE 'freeProvider|synthetic' internal/inference/freeconn.go     # 039
# additive-only: selection.go has no deletions of existing logic
git diff <base>..HEAD -- internal/inference/selection.go | grep -E '^-' | grep -v '^---' | wc -l   # = 0
# no interface/constructor signature change
git diff <base>..HEAD -- internal/inference/selection.go | grep -E '^[-+].*func NewSelectionEngine' | wc -l   # = 0
# no init()
! grep -rn "func init(" internal/inference/freeconn.go internal/inference/upstream.go internal/inference/livecatalog.go internal/inference/projectid.go && echo "no init() OK"
```

**Negative / freeze proofs (w7-route-b commit-range — §7)**
```bash
R="<first-w7-route-b>^..<last-w7-route-b>"
# Only the sanctioned engine files changed:
git diff $R --name-only -- internal/ | grep -vE \
 'internal/inference/(selection|factory|runner|freeconn|upstream|livecatalog|projectid)(_test)?\.go|internal/server/(routes_openai|server)(_test)?\.go|internal/providers/generic/chat(_test)?\.go' \
 | wc -l                                                                  # = 0
# w7-route-a territory untouched (admin/store admin tables/routes_admin):
git diff $R --name-only -- internal/admin/ internal/server/routes_admin.go internal/store/aliasesadmin.go internal/store/routingrules.go internal/store/modellimits.go internal/store/combosadmin.go | wc -l   # = 0
# frozen combos engine + selection existing logic:
git diff $R --name-only -- internal/admin/combos.go internal/store/combos.go | wc -l   # = 0
git diff $R -- internal/inference/selection.go | grep -E '^-' | grep -v '^---' | wc -l   # = 0 (additive)
# NO UI / mock / seed touched (no UI contract):
git diff $R --name-only -- ui/ | wc -l                                   # = 0
# NO routes_admin.go (this plan holds no slot):
git diff $R --name-only -- internal/server/routes_admin.go | wc -l       # = 0
```

---

## 6. Out of scope (restated, binding)

No admin CRUD / admin routes / routes_admin.go (w7-route-a). No UI / mock / seed
edits (no dashboard contract). No rewrite of existing selection/eligibility/cooldown/
URL/retry logic (additive branches/functions/setters only). No interface or
constructor signature change. No live web search/fetch execution (059 = exposure
only). No live multi-URL retry wiring if it would touch a frozen body (035 = URL-list
builder + unit). No schema change (weights/project-ID via Metadata). No network in
tests (injectable fakes only). selection.go is micro-serial-coordinated vs w7-plat-1
(merged) — orchestrator confirms one unmerged holder before T-weighted.

## 7. Diff-gate scope

Scope to w7-route-b's own commits:
`git log --oneline main | grep "^[0-9a-f]* phase-1/w7-route-b:" | awk '{print $1}'`
then `git diff <first-w7-route-b>^..<last-w7-route-b> -- [file list]`.

`git diff <first>^..<last> --name-only` must be exactly a subset of:
```
internal/inference/selection.go          (ADDITIVE weighted + free-conn; micro-serial; ONE concern)
internal/inference/selection_test.go
internal/inference/freeconn.go (+_test)
internal/inference/upstream.go (+_test)
internal/inference/livecatalog.go (+_test)
internal/inference/projectid.go (+_test)
internal/inference/factory.go            (CONDITIONAL — additive project-ID call)
internal/inference/runner.go             (CONDITIONAL — same)
internal/server/routes_openai.go         (ADDITIVE adapters/setters + upstream gate)
internal/server/routes_openai_test.go
internal/server/server.go                (CONDITIONAL — additive resolver wiring)
internal/providers/generic/chat.go       (ADDITIVE chatURLs)
internal/providers/generic/chat_test.go
.planning/parity/matrix/9router-routing.md
.planning/parity/plans/open-questions.md
docs/WORKFLOW.md
```
Any file outside this list is an automatic review REJECT. All `internal/admin/*`,
`routes_admin.go`, the w7-route-a store tables, the combos engine, and all
`ui/**` are deliberately ABSENT — touching them is an automatic REJECT. selection.go
must be ADDITIVE-only (no deletions) and micro-serial-coordinated; this plan holds NO
routes_admin slot.

## 8. Escalations / decisions (explicit — recommended defaults, do not fabricate)

- **ESC-WEIGHT-SRC (RESOLVED at authoring — weighted-selection weight source, binding
  default).** **Decision: read a per-connection weight from `Connection.Metadata`
  JSON (`{"weight":N}`), default 1** — NO schema change (the Connection struct/table
  are frozen; Metadata is the existing free-form seam). Use a DETERMINISTIC weighted
  accumulator (NOT `math/rand`) so tests are reproducible. Alternative: a new
  `connections.weight` column (additive) if the operator wants a typed field — but
  that touches the frozen connection store; default = Metadata, zero migration. Flag.
- **ESC-FREE-SET (RESOLVED at authoring — free/noAuth provider set, binding default).**
  **Decision: source the noAuth provider IDs from the catalog's noAuth flag if the
  catalog carries one; else a constant mirroring `providers.js:14` (e.g. opencode).**
  Verify the catalog at T-freeconn. RECOMMENDED; flag if the set is ambiguous.
- **ESC-055-OVERLAP (RESOLVED at authoring — ROUTE-055 vs w7-plat-1 PAR-PLAT-009).**
  w7-plat-1 merged the per-connection proxy hook. **Decision: VERIFY coverage; flip
  ROUTE-055 → HAVE citing w7-plat-1 + verification, adding code ONLY if a ROUTE-055-
  specific gap (connectionProxyEnabled/vercelRelayUrl) exists.** Never duplicate the
  proxy machinery. Flag the overlap for the orchestrator.
- **ESC-WEB-EXEC (RESOLVED at authoring — web search/fetch live execution, binding
  default + escalation).** PAR-ROUTE-059 EXPOSES `{alias}/search`+`{alias}/fetch`
  pseudo-models. **Decision: w7-route-b ships the pseudo-model EXPOSURE + an injectable
  execution seam stub; the LIVE web-search/fetch execution (reverse-engineered web
  endpoints, fragile, network-bound) is DEFERRED to a follow-up/escalation** — never
  fabricate live web calls in this plan. Record in open-questions. Flag.
- **ESC-035-RETRY (CONDITIONAL — multi-URL live-retry wiring).** PAR-ROUTE-035 ships
  the `chatURLs()` URL-LIST builder + unit. If wiring the live retry-on-next-URL
  would touch a FROZEN handler body, scope w7-route-b to the builder + unit and record
  the live-retry wiring as a follow-up. If the retry path is additive-safe, wire it.
  Decide at T-fallback. Flag.
- **ESC-PROJ-PERSIST (RESOLVED at authoring — project-ID persistence target, binding
  default).** **Decision: persist the resolved project ID to `Connection.Metadata`
  (existing free-form seam), NO new column.** Alternative: an additive
  `connections.project_id` column (touches the frozen connection store) — default =
  Metadata, zero migration. Flag.
- **ESC-MODELS-HOOK (RESOLVED at authoring — 056/059/060 hook site).** The brief cites
  `factory.go:104`; the models-LIST surface is actually `ModelsHandler.List`
  (`routes_openai.go:93`) with the established `Set*Lister` additive-adapter pattern.
  **Decision: hook 056/059/060 into `ModelsHandler` via new adapters+setters**
  (the correct, precedented additive site); `factory.go` is only touched for the 053
  build-site call. Flag the cite correction for the orchestrator.
- **selection.go micro-serial (§1.3 / P3).** w7-plat-1's selection.go proxy hook is
  MERGED; w7-route-b's weighted/free-conn edit is the next selection.go edit.
  Orchestrator confirms exactly one unmerged holder before T-weighted; the edit is
  ADDITIVE-only.
- **No routes_admin slot.** w7-route-b registers NO admin routes; it holds and
  releases no routes_admin serial slot (distinct from w7-route-a, which does).
```
