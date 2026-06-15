# Micro-plan bf-gov-4 — VK mandatory mode (governance, Go)

```
program: bifrost-parity (Waves 6+; CLI_ORCHESTRATOR.md governs, HANDOFF.md retired)
plan: bf-gov-4
status: READY (rev 1 — authored against the merged Waves 0–7 + bf-gov-1/2/3 +
  bf-core-2 + bf-mcp-1 tree, live @ HEAD ac878f1; BIFROST-MAP.md bifrost-governance
  disposition row 034 §250; matrix .planning/parity/matrix/bifrost-governance.md:45;
  freeze rules. AUDIT-FOUND GAP: PAR-BF-GOV-034 was BUILD-marked in BIFROST-MAP
  (§250 "VAR/BUILD") but fell through — NOT closed by bf-gov-1/2/3, status still
  MISSING. bf-gov-4 closes the single unassigned row.)
runs: governance track. Disjoint from all open bf-* plans. The bf-gov chain
  (bf-gov-1 → bf-gov-3) is CLOSED and MERGED — vk.go / routes_openai.go gate-wiring /
  admin/mcp.go have NO concurrent holder (all prior holders merged; §7 serial).
branch: main (direct commits, per AGENTS.md "No PR workflow")
commit-prefix: phase-1/bf-gov-4:  (matches the shipped bifrost chain prefix —
  verified in git log: `phase-1/bf-gov-3: …`, `phase-1/bf-mcp-1: …`)
footer: Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>
ref-source: BLOCKED — ESC-REF-ABSENT (BIFROST-MAP §47-68). The frozen Bifrost ref
  @ ca21298 is ABSENT on this host. The ONLY ground truth is
  .planning/parity/matrix/bifrost-governance.md:45 + BIFROST-MAP §250. The matrix
  row cites `plugins/governance/main.go:783-797` — UNREADABLE. Build to the
  documented matrix behavior (reject requests with no resolved VK when mandatory
  mode is on) + g0router conventions ONLY. NEVER build to a guessed Bifrost wire
  format or invent a Bifrost config struct.
go-serial-slot: NONE. bf-gov-4 registers NO new HTTP routes. It edits vk.go (an
  api-layer file: an additive field + setter + the empty-key branch body),
  routes_openai.go (the EXISTING vkGate-wiring block at :103 — NOT a route
  registration), and admin/mcp.go (the admitMCPVK body — additive mandatory
  branch). All three were touched by merged plans (bf-gov-1/2 on vk.go, bf-mcp-1
  on admin/mcp.go) but those are CLOSED — no concurrent holder (§7).
new-route: NO. Confirmed: bf-gov-4 registers NO new HTTP routes; it reuses the
  existing /v1/* gate path and the existing /mcp admission path.
flag: REUSE the bf-core-2 flag-seed mechanism (migrate.go:419-429 INSERT OR IGNORE
  loop) + the shipped IsFeatureEnabled(key) by-key reader (featureflags.go:69).
  bf-gov-4 seeds a NEW `vk_mandatory` flag row OFF by default.
```

---

## 0. Objective + ground truth

### 0.1 Objective

Add an **operator-settable, OFF-by-default, backward-compatible** "VK mandatory
mode" to g0router: when the `vk_mandatory` feature flag is ON, a request that
resolves to **no virtual key** (`key == ""`) is **REJECTED 401 "virtual key
required"** instead of being allowed. When the flag is OFF (the default, and the
only state on a fresh or upgraded store), behavior is **byte-identical to today** —
an absent VK is allowed everywhere it is allowed now.

This is purely additive and needs NO operator decision (it is a runtime flag an
operator may toggle) and NO re-architecture. The intervention is a single injected
predicate consulted in the ONE place an absent VK is currently waved through:
`VKGate.AllowVK`'s empty-key early-return (`internal/api/vk.go:98-100`), mirrored
in the `/mcp` admission path (`internal/admin/mcp.go:843-853`).

The matrix's `x-bf-vk` header maps to g0router's `x-g0-vk` (already the live header,
`vk.go` / `chat.go:365`) and g0router VK values are `g0vk-` prefixed (not `vk_`) —
the header NAME is the documented VAR; the BEHAVIOR (mandatory-VK rejection) is
fully built, so 034 flips to HAVE (variant header).

Additive-only: a `vk_mandatory` flag seeded OFF (reuse the migrate.go:419-429
loop + IsFeatureEnabled by-key), an additive `mandatory func() bool` field +
`SetMandatoryChecker` setter on `VKGate`, the empty-key branch body change in
`AllowVK`, the mirror branch in `admitMCPVK`, and the predicate wiring in the
existing `routes_openai.go:103` gate-wiring block. NO signature change to
`NewVKGate`/`AllowVK`, NO `init()`, NO global state, no destructive DDL.

### 0.2 Rows this plan closes (matrix = ground truth)

| Row | Behavior (matrix:45) | Disposition | Acceptance after bf-gov-4 |
|---|---|---|---|
| PAR-BF-GOV-034 | "Virtual key mandatory mode: rejects requests without `x-bf-vk` header when `isVkMandatory` is true" (`plugins/governance/main.go:783-797`, ABSENT) | **VAR/BUILD** (BIFROST-MAP §250 "BUILD an optional mandatory-VK setting under g0router's header name; record header-name as variant") — **AUDIT-FOUND unassigned gap** (BUILD-marked but NOT closed by bf-gov-1/2/3; still MISSING). | `vk_mandatory` flag (seeded OFF) + injected `mandatory()` predicate consulted in `AllowVK`'s empty-key branch + the `/mcp` `admitMCPVK` mirror. Flag OFF ⇒ absent VK allowed (today's behavior). Flag ON ⇒ absent VK rejected 401 "virtual key required" (JSON-RPC error on /mcp). The `x-bf-vk → x-g0-vk` header name is the VAR. MISSING → **HAVE (variant header)**. |

**Honest scoping note:** 034 closes fully to **HAVE** — the mandatory-VK rejection
BEHAVIOR is built and live on BOTH the `/v1/*` gate and the `/mcp` surface; only the
header NAME diverges (`x-g0-vk` vs `x-bf-vk`), which BIFROST-MAP §250 explicitly
authorizes as a recorded VAR. No row is closed by inventing un-evidenced Bifrost
behavior. The residual (per-VK / per-user mandatory scoping, a config struct vs a
flag) is recorded in `open-questions.md` (§7) and is explicitly out of scope (§3).

### 0.3 Preconditions already satisfied (evidence — read 5 files, AGENTS.md)

- **The empty-key intervention point EXISTS and is the EXACT seam**
  (`internal/api/vk.go:97-100`): `func (g *VKGate) AllowVK(key, model, providerID
  string) (ok bool, status int, reason string, keyIDs []string)`; the first line is
  `if g == nil || g.resolver == nil || key == "" { return true, 0, "", nil }`. The
  `key == ""` disjunct is precisely where an absent VK is waved through today — the
  ONLY change is to split this branch so the `key == ""` case consults the injected
  mandatory predicate (D2). The 4-return shape and the `g == nil`/`g.resolver == nil`
  guards are PRESERVED.
- **`VKGate` is a small struct with an additive-friendly shape**
  (`vk.go:82-90`): `type VKGate struct { resolver VKResolver; quota VKQuotaChecker }`
  + `func NewVKGate(resolver VKResolver, quota VKQuotaChecker) *VKGate { return
  &VKGate{...} }`. There is NO setter pattern on VKGate yet, but the additive
  field + setter is the established api-layer pattern (e.g. `chat.SetVKGate`,
  `chat.SetSemanticCache` from bf-core-2). bf-gov-4 adds a `mandatory func() bool`
  field + `SetMandatoryChecker(func() bool)` setter — `NewVKGate`/`AllowVK`
  signatures UNCHANGED (D2).
- **The by-key, fail-OFF flag reader EXISTS and is shipped**
  (`internal/store/featureflags.go:69`): `func (s *Store) IsFeatureEnabled(key
  string) (bool, error)` — `SELECT enabled FROM feature_flags WHERE key = ?`; a
  MISSING flag returns `(false, nil)` (fail-OFF). bf-gov-4's predicate reads
  `IsFeatureEnabled("vk_mandatory")` — a missing/OFF flag means "not mandatory"
  (today's behavior), so the fail-OFF default is exactly the backward-compat
  guarantee (D1/D3).
- **The flag-seed mechanism EXISTS and is the reuse precedent**
  (`internal/store/migrate.go:416-429`): an idempotent `INSERT OR IGNORE INTO
  feature_flags (key, enabled, description, created_at) VALUES (?, 0, ?, ?)` loop
  seeded by bf-core-2, creating each flag OFF (`enabled = 0`) on a fresh store and
  never overwriting on re-migration (so an admin toggle persists). bf-gov-4 appends
  exactly one entry `{"vk_mandatory", "Reject requests with no resolved virtual key
  (mandatory-VK mode)"}` to this same slice (D1). NO new table, NO new column.
- **The `/mcp` admission seam EXISTS and currently admits an ABSENT VK**
  (`internal/admin/mcp.go:843-853`): `func (h *Handlers) admitMCPVK(ctx
  *fasthttp.RequestCtx) (vk string, admitted bool)` — `key := resolveMCPVK(...)`;
  `if key == "" { return "", true }` (absent VK allowed, the bf-mcp-1 "VK optional"
  posture); a PROVIDED-but-invalid VK is already rejected (`!rec.IsActive → return
  key, false`). The mandatory-mode mirror changes ONLY the `key == ""` branch: under
  mandatory mode, `return "", false` (absent VK rejected too). `resolveMCPVK` is at
  `:53`. The rejection surfaces on `MCPServerPost` (`:867`, a JSON-RPC error via
  `marshalMCPError`) and `MCPServerSSE` (`:932`, a connection reject) — both already
  branch on `admitted` (D4).
- **The gate is constructed/wired in the existing block**
  (`internal/server/routes_openai.go:101-113`): `vkGate := api.NewVKGate(
  newVKResolverAdapter(st), newVKQuotaAdapter(governance.NewQuotaEngine(st,
  time.Now)))` then `chat.SetVKGate(vkGate)` + 9 sibling `SetVKGate` calls. `st`
  (the `*store.Store`) is IN SCOPE here — the mandatory predicate closes over `st`:
  `vkGate.SetMandatoryChecker(func() bool { ok, _ := st.IsFeatureEnabled(
  "vk_mandatory"); return ok })` (D2/D3). This is the gate-wiring block, NOT a route
  registration (no serial-route slot).
- **AllowVK has ~10 live call sites** (verified by grep, production only):
  `internal/api/{chat,completions,responses,audio,images,files,batches,messages,
  embeddings,input_tokens}.go` each call `g.AllowVK(...)`. **CORRECTION (rev 2):** each
  call is guarded by `if vkHeader != ""` (e.g. `chat.go:386`), so `AllowVK("")` is NOT
  reached on `/v1/*` today — the empty-key seam alone does NOT cover the fleet. Per the
  orchestrator ruling (Option A, §2), each guard is REMOVED so `AllowVK` is called
  unconditionally and the empty-key mandatory branch becomes the genuine single `/v1/*`
  seam (behavior- and perf-preserving when OFF; `AllowVK("")` short-circuits at its own
  `key==""` branch without touching the resolver/quota).
- **The flag store is a cheap SQLite read** (`featureflags.go:69` is a single
  indexed-by-UNIQUE-key `QueryRow`). Reading it per-request (fresh) is correct and
  cheap (D3) — no cache, no staleness, an operator toggle takes effect on the next
  request.

---

## 1. Decisions made (and why) — binding

### D1 — `vk_mandatory` feature flag, seeded OFF (backward-compat by construction)

The matrix's `isVkMandatory` is a boolean toggle. **Decision:** represent it as a
g0router feature flag `vk_mandatory`, seeded OFF via the SHIPPED bf-core-2 flag-seed
loop (`migrate.go:419-429`): append `{"vk_mandatory", "Reject requests with no
resolved virtual key (mandatory-VK mode)"}` to the slice. `INSERT OR IGNORE` with
`enabled = 0` means:
- On a FRESH store: the row is created OFF.
- On an UPGRADED store (re-migration): the row is created once OFF and never
  overwritten, so a later admin toggle persists.

Because the flag defaults OFF and the predicate fails OFF on a missing flag
(`IsFeatureEnabled` returns `(false, nil)` for an absent row), **the default and
upgrade behavior is byte-identical to pre-bf-gov-4** — an absent VK is allowed
exactly as today. This is the binding backward-compat guarantee. No new table, no
new column, no operator decision required to ship (the operator OPTIONALLY toggles
it later via the existing feature-flag admin surface — bf-gov-4 adds NO new admin
route).

### D2 — Injected predicate on `VKGate` (no signature break); empty-key branch consults it

`AllowVK` is a pure gate that does not currently read any flag. **Decision:** add an
additive `mandatory func() bool` field to `VKGate` and a `SetMandatoryChecker(check
func() bool)` setter (the established additive-setter pattern — `chat.SetVKGate`,
`chat.SetSemanticCache`). PRESERVE `NewVKGate(resolver, quota)` and the
`AllowVK(key, model, providerID) (ok, status, reason, keyIDs)` 4-return signature
EXACTLY — only the empty-key branch BODY changes and the additive field/setter are
added.

The empty-key branch (`vk.go:98-100`) splits so the `g == nil`/`g.resolver == nil`
guards keep their fast no-op return, and the `key == ""` case consults the predicate:
```go
func (g *VKGate) AllowVK(key, model, providerID string) (ok bool, status int, reason string, keyIDs []string) {
	if g == nil || g.resolver == nil {
		return true, 0, "", nil
	}
	if key == "" {
		if g.mandatory != nil && g.mandatory() {
			return false, 401, "virtual key required", nil
		}
		return true, 0, "", nil
	}
	// ... unchanged: resolve, active check, config match, quota ...
}
```
A nil `mandatory` predicate (gate constructed without the setter, e.g. some tests)
is a clean no-op (absent VK allowed) — two layers of OFF (nil predicate, or predicate
returns false), both byte-identical to today. The status is **401** (auth-shaped:
"a credential is required") with reason `"virtual key required"`, distinct from the
existing `"unknown virtual key"` (401, a PROVIDED bad key) and `"virtual key
inactive"` (403). The 401 choice is g0router's auth convention; if a reviewer
requires a Bifrost-specific status for the absent-mandatory case and it is NOT in
the matrix note, default to 401 and record the status question in open-questions
(ESC-REF-ABSENT — do NOT invent a Bifrost code).

### D3 — Per-request fresh flag read (not cached) — simple, correct, cheap

The predicate must reflect the operator's current toggle. **Decision:** read the
flag FRESH on every gate decision — the predicate is
`func() bool { ok, _ := st.IsFeatureEnabled("vk_mandatory"); return ok }`, closing
over `st` in the `routes_openai.go:103` wiring block. Rationale:
- **Correct:** an operator toggle takes effect on the very next request; no
  staleness, no cache-invalidation bug.
- **Cheap:** `IsFeatureEnabled` is a single `QueryRow` on the UNIQUE `key` index
  (`featureflags.go:69`) — a sub-microsecond SQLite read, dwarfed by the provider
  round-trip the request is about to make.
- **Fail-OFF:** the predicate swallows the error to `false` (the read already
  returns `(false, nil)` for a missing flag; a transient query error degrades to
  "not mandatory" rather than failing-closed and locking everyone out — the
  conservative, backward-compatible choice). The discarded error is acceptable here
  because the ONLY consequence of a read error is "behave as if mandatory mode is
  off", which is the safe default; this is documented so the swallow is intentional,
  not sloppy.

A cached read was REJECTED: it would add a cache + invalidation surface for zero
measurable gain on an already-trivial query, and would delay an operator toggle.
Per-request fresh is the binding choice.

### D4 — `/mcp` mirror: `admitMCPVK` rejects an absent VK under mandatory mode

bf-mcp-1's `admitMCPVK` (`admin/mcp.go:843-853`) currently admits an absent VK
(`key == "" → return "", true`, the "VK optional" posture). **Decision:** mirror the
mandatory check there — under mandatory mode (flag ON), an absent VK on `/mcp` is
REJECTED too:
```go
func (h *Handlers) admitMCPVK(ctx *fasthttp.RequestCtx) (vk string, admitted bool) {
	key := resolveMCPVK(ctxHeaderGetter(ctx))
	if key == "" {
		if h.vkMandatory() { // flag ON ⇒ reject absent VK on /mcp too
			return "", false
		}
		return "", true // absent VK allowed (optional surface, today's behavior).
	}
	// ... unchanged: validate provided VK, reject unknown/inactive ...
}
```
The rejection surfaces through the EXISTING `admitted == false` branches:
`MCPServerPost` (`:867-873`) already writes a JSON-RPC error
(`marshalMCPError("virtual key unknown or inactive")`); bf-gov-4 keeps that path but
the absent-VK rejection message is `"virtual key required"` (a distinct
mandatory-mode message vs the invalid-VK message — implement by threading the reason
or adding a sibling reject branch). `MCPServerSSE` (`:932-938`) already rejects with
`writeError(...StatusUnauthorized...)` — the absent-mandatory case reuses the same
401 close-before-stream path with the "virtual key required" message.

The `h.vkMandatory()` predicate on `*admin.Handlers` reads the SAME
`IsFeatureEnabled("vk_mandatory")` (the handler already holds `h.store`, §0.3 / the
bf-mcp-1 precedent), so the `/v1/*` gate and the `/mcp` surface enforce the IDENTICAL
flag. This is the documented mandatory-mode EXTENSION of bf-mcp-1's "VK optional"
posture (recorded in the matrix note + open-questions). A PROVIDED-but-invalid VK on
`/mcp` is STILL rejected exactly as bf-mcp-1 ships (unchanged); bf-gov-4 only adds
the absent-VK rejection under the flag.

### D5 — Header name VAR; 034 → HAVE (variant header)

The matrix names `x-bf-vk`; g0router's live header is `x-g0-vk` (`vk.go` /
`chat.go:365`) and VK values are `g0vk-` prefixed (`virtualkeys.go:78`), not `vk_`.
**Decision:** record the header NAME (`x-bf-vk → x-g0-vk`) and the key-prefix
(`vk_ → g0vk-`) as the documented VAR. The BEHAVIOR — "reject a request with no
resolved VK when mandatory mode is on" — is fully built on both surfaces, so 034
flips to **HAVE (variant header)**, NOT PARTIAL (BIFROST-MAP §250 explicitly
authorizes "record header-name as variant"). No header rename, no new header — the
mandatory check operates on whether the EXISTING resolution produced a key, so it is
header-name-agnostic by construction.

### D6 — Strict TDD, hermetic (injected predicate as a fake; no real flag store needed)

The mandatory predicate is INJECTED (`func() bool`), so the `VKGate` unit test feeds
a fake predicate directly — NO network, NO sleep, NO real flag store, NO SQLite for
the gate unit test. **Decision (binding):**
- `vk_test.go`: construct `NewVKGate(fakeResolver, fakeQuota)`,
  `gate.SetMandatoryChecker(func() bool { return <true|false> })`, and assert the
  empty-key behavior directly (flag OFF + no VK ⇒ allowed; flag ON + no VK ⇒
  401 "virtual key required"; flag ON + valid VK ⇒ allowed). Fully hermetic.
- `admin/mcp_test.go`: drive `admitMCPVK` with the injected mandatory predicate (or a
  test store with the flag toggled via the existing in-mem `store.Open` pattern) —
  absent VK + flag ON ⇒ not admitted; absent VK + flag OFF ⇒ admitted; provided
  invalid VK ⇒ not admitted (unchanged).
- `featureflags_test.go` (EXTEND, optional): `IsFeatureEnabled("vk_mandatory")`
  returns the seeded OFF default → false; after toggle → true (reuses the shipped
  reader; the seed is the only new bit).
Additive only, no `init()`, errors-as-values, no global state, snake_case. This is
binding (Wave-7 hermetic lesson).

---

## 2. Target files

### IN-SCOPE — EXTEND (additive only)

| File | Change (additive ONLY) |
|---|---|
| `internal/store/migrate.go` | ADD one entry `{"vk_mandatory", "Reject requests with no resolved virtual key (mandatory-VK mode)"}` to the EXISTING flag-seed slice (`:419-421`), seeded OFF via the shipped `INSERT OR IGNORE ... VALUES (?, 0, ?, ?)` (D1). NOTHING else — no table, no column. |
| `internal/api/vk.go` | ADD `mandatory func() bool` field to `VKGate` (`:82-85`); ADD `SetMandatoryChecker(check func() bool)` setter. SPLIT the empty-key early-return (`:98-100`) so the `g == nil`/`g.resolver == nil` guards keep their no-op return and the `key == ""` case consults `g.mandatory` ⇒ `(false, 401, "virtual key required", nil)` when mandatory (D2). PRESERVE `NewVKGate`/`AllowVK` signatures. |
| `internal/api/vk_test.go` | RED first (D6): flag OFF + no VK ⇒ allowed (4-tuple ok=true); flag ON + no VK ⇒ (false, 401, "virtual key required", nil); flag ON + valid VK ⇒ allowed (resolve path unchanged); nil predicate ⇒ allowed (no-op). Injected fake predicate; hermetic. |
| `internal/admin/mcp.go` | ADD the mandatory mirror to `admitMCPVK` (`:843-853`): under the flag, the `key == ""` branch returns `("", false)` (absent VK rejected) instead of `("", true)` (D4); thread a "virtual key required" reason for the absent-mandatory reject (distinct from the invalid-VK message). ADD the `h.vkMandatory()` predicate helper reading `h.store.IsFeatureEnabled("vk_mandatory")`. PRESERVE all existing handler signatures. |
| `internal/admin/mcp_test.go` | RED first (D6): absent VK + flag ON ⇒ not admitted; absent VK + flag OFF ⇒ admitted; provided invalid VK ⇒ not admitted (unchanged regression guard). `MCPServerPost` absent+mandatory ⇒ JSON-RPC error "virtual key required"; `MCPServerSSE` absent+mandatory ⇒ 401 close. Hermetic (injected predicate or in-mem store flag toggle). |
| `internal/server/routes_openai.go` | EXTEND the EXISTING vkGate-wiring block (`:101-113`): after `vkGate := api.NewVKGate(...)` (`:103`), add `vkGate.SetMandatoryChecker(func() bool { ok, _ := st.IsFeatureEnabled("vk_mandatory"); return ok })` (D3, closes over the in-scope `st`). Wiring only — NO new route, NO `NewVKGate` signature change. |
| `internal/server/routes_openai_test.go` (EXTEND/CREATE if a wiring test exists) | Optional server-level test: with the `vk_mandatory` flag ON in the test store, a `/v1/chat/completions` request with NO `x-g0-vk` is rejected 401; with the flag OFF, the same request is admitted (proves the predicate is wired live end-to-end). Hermetic (in-mem store). |
| `internal/store/featureflags_test.go` (EXTEND, optional) | `IsFeatureEnabled("vk_mandatory")` → false on the seeded default; true after toggle. (Mostly covered by the shipped reader; this asserts the seed lands.) |
| `.planning/parity/matrix/bifrost-governance.md` | Flip row 034 → HAVE (variant header) at close (§7). |
| `.planning/parity/plans/open-questions.md` | Append the mandatory-scoping / status-code / config-struct ESC items (§7). |
| `docs/WORKFLOW.md` | Add the bf-gov-4 row at close (§7). |

### FORBIDDEN (automatic REJECT if touched)

- Any signature change to `NewVKGate` or `AllowVK` — additive field + setter ONLY (D2).
- ~~Any per-call-site edit to the 10 AllowVK callers~~ — **CORRECTED (rev 2, orchestrator
  ruling on the executor's escalation):** the original premise was FALSE. The 10 callers
  each guard the gate behind `if vkHeader != ""` (e.g. `chat.go:386`), so `AllowVK("")`
  is NEVER reached on `/v1/*` and the committed empty-key mandatory branch (f1bc048) is
  structurally unreachable there. The AUTHORIZED fix (Option A): at each of the 10 call
  sites, REMOVE the `if vkHeader != ""` guard and call `AllowVK(vkHeader, ...)`
  UNCONDITIONALLY. This is behavior- AND perf-preserving when mandatory is OFF —
  `AllowVK("")` short-circuits at its own `key==""` branch (it never calls the
  resolver/quota) and returns `(true,0,"",nil)`, so an un-keyed request proceeds exactly
  as today; the `len(keyIDs)>0` pin guard already skips when keyIDs is nil. When mandatory
  is ON, `AllowVK("")` returns `(false,401,"virtual key required",nil)` and the existing
  `if !ok { writeError; return }` rejects it. This makes the committed AllowVK seam the
  GENUINE single `/v1/*` decision point (the plan's intent), realized by removing the
  premature guards — NOT churn, the necessary wiring the false premise hid. Add a
  server-level test proving flag-ON + no `x-g0-vk` ⇒ 401 and flag-OFF ⇒ proceeds.
- Any NEW route registration (`routes_openai.go` / `routes_admin.go`) — bf-gov-4 adds
  NO routes; it reuses the existing /v1/* gate and /mcp admission paths.
- Any NEW admin route/handler to TOGGLE the flag — the existing feature-flag admin
  surface (`SetFeatureFlagEnabled`) already toggles flags by id; bf-gov-4 adds no
  toggle UI/route.
- Any per-VK / per-user / per-team mandatory scoping, any "isVkMandatory" config
  struct, any new column/table — a global runtime flag is the whole scope (§3 ESC).
- Any change to `internal/mcp/*` client primitives or `resolveMCPVK` precedence —
  consume as shipped; bf-gov-4 only adds the absent-VK mandatory branch in
  `admitMCPVK`.
- Any `init()`, any new global state, any GORM hook, any destructive DDL
  (DROP/RENAME) in `migrate.go`.
- Any UI file (`ui/**`) — bf-gov-4 is Go-only; the flag is toggled via the existing
  feature-flags surface (no new UI page).

---

## 3. Scope / Non-goals — explicit ESC list

**bf-gov-4 builds ONLY the global, flag-gated, mandatory-VK rejection on the /v1/*
gate + /mcp surface.** The following are **ESC** (recorded in `open-questions.md`):

| ESC item | Why ESC |
|---|---|
| **Per-VK / per-user / per-team mandatory scoping** | bf-gov-4 is a single global runtime flag (BIFROST-MAP §250 "an optional mandatory-VK setting"). Per-entity mandatory mode presupposes the per-VK scoping surface that is bf-mcp-2 / the Customer tier (ESC in bf-gov-1 §3). Defer. |
| **A dedicated `isVkMandatory` config struct / settings key** | g0router represents the toggle as a feature flag (D1), reusing the shipped flag store. A separate config struct would duplicate the flag surface for zero behavioral gain. The flag IS the setting. |
| **Mandatory-mode HTTP status / JSON-RPC error-code parity with Bifrost** | ESC-REF-ABSENT: the matrix note (`main.go:783-797`, UNREADABLE) does not capture Bifrost's exact status/code for the absent-mandatory case. bf-gov-4 uses 401 "virtual key required" (g0router auth convention, D2) and the JSON-RPC `-32600` reject on /mcp; the exact-parity question is recorded, not invented. |
| **The `x-bf-vk` header name / `vk_` key prefix** | VAR (D5): g0router uses `x-g0-vk` / `g0vk-`. The behavior is built; the names diverge by g0router convention (BIFROST-MAP §250 authorizes recording the header as a variant). |

No-leftovers (binding, §3 CLI_ORCHESTRATOR): bf-gov-4 adds the `mandatory` predicate
field + setter ONLY because `AllowVK`'s empty-key branch LIVE-CONSUMES it (a test
proves flag OFF + no VK ⇒ allowed; flag ON + no VK ⇒ 401; flag ON + valid VK ⇒
allowed) and the `routes_openai.go` wiring injects a REAL store-reading predicate.
The `/mcp` mirror MUST actually reject an absent VK under the flag (test-proven). The
`vk_mandatory` flag row is consumed by BOTH predicates. If the predicate cannot be
wired live (e.g. the empty-key branch does not actually reject when the predicate
returns true), the plan STOPS and escalates. No dead field, no dead flag, no
resolved-but-unread predicate.

---

## 4. Task graph (TDD; `N. [step] -> verify: [check]`)

Cadence (AGENTS.md "TDD always"): no impl field/branch lands before its covering
`_test.go` is committed RED. `go test ./... && go vet ./... && go build ./...` green
at EVERY commit. Footer on every commit:
`Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>`.

1. **[flag seed, GREEN]** Append `{"vk_mandatory", "Reject requests with no resolved
   virtual key (mandatory-VK mode)"}` to the migrate.go flag-seed slice (`:419-421`,
   D1). (Optionally) extend `featureflags_test.go`:
   `IsFeatureEnabled("vk_mandatory")` is false on a freshly-migrated store. ->
   verify: `go test ./internal/store/ -run Feature` green; `grep -n vk_mandatory
   internal/store/migrate.go` non-empty; `go vet ./... && go build ./...` exit 0.
   Commit: `phase-1/bf-gov-4: seed vk_mandatory feature flag (OFF by default)`.

2. **[gate predicate, RED]** Write `vk_test.go` additions (D6): flag OFF + no VK ⇒
   allowed; flag ON + no VK ⇒ (false, 401, "virtual key required", nil); flag ON +
   valid VK ⇒ allowed; nil predicate ⇒ allowed. -> verify: `go test ./internal/api/
   -run VK` FAILS (no `SetMandatoryChecker` / branch). Commit RED:
   `phase-1/bf-gov-4: failing VK-mandatory gate tests (TDD red)`.

3. **[gate predicate, GREEN]** Add `mandatory func() bool` + `SetMandatoryChecker` to
   `VKGate`; split the empty-key branch so `key == ""` consults `g.mandatory` (D2).
   PRESERVE `NewVKGate`/`AllowVK` signatures. -> verify: `go test ./internal/api/...
   && go vet ./... && go build ./...` exit 0; the four gate cases pass; signatures
   unchanged (grep §5). Commit:
   `phase-1/bf-gov-4: injected mandatory predicate in VK gate empty-key branch`.

4. **[wiring, GREEN]** Extend the `routes_openai.go` vkGate-wiring block (`:103`) with
   `vkGate.SetMandatoryChecker(func() bool { ok, _ := st.IsFeatureEnabled(
   "vk_mandatory"); return ok })` (D3). (Optionally) add a server-level test: flag ON
   ⇒ no-`x-g0-vk` `/v1/chat/completions` rejected 401; flag OFF ⇒ admitted. ->
   verify: `go test ./internal/server/... && go test ./... && go vet ./... &&
   go build ./...` exit 0; NO new route (grep §5); the predicate is wired (grep §5).
   Commit: `phase-1/bf-gov-4: wire vk_mandatory predicate into the gate (per-request)`.

5. **[/mcp mirror, RED→GREEN]** Extend `admin/mcp_test.go` (D6): absent VK + flag ON
   ⇒ not admitted (MCPServerPost JSON-RPC error "virtual key required"; MCPServerSSE
   401 close); absent VK + flag OFF ⇒ admitted; provided invalid VK ⇒ not admitted
   (regression). Add the mandatory branch to `admitMCPVK` + the `h.vkMandatory()`
   helper (D4). -> verify: `go test ./internal/admin/ -run MCP` green; the /mcp
   absent-VK reject is behavior-proven (grep §5); `go vet ./... && go build ./...`
   exit 0. Commit RED then GREEN:
   `phase-1/bf-gov-4: failing /mcp mandatory-VK mirror tests (TDD red)` /
   `phase-1/bf-gov-4: reject absent VK on /mcp under mandatory mode`.

6. **[close]** Full validation (§6); flip matrix row 034 → HAVE (variant header)
   (§7); append `open-questions.md` (§3 ESC items); update `docs/WORKFLOW.md`. ->
   verify: §6 all green; matrix + WORKFLOW + open-questions committed. Commit:
   `phase-1/bf-gov-4: close — VK mandatory mode; matrix flip 034 → HAVE (variant)`.

---

## 5. Acceptance criteria (binary; file:line / grep where possible)

**Test gates** (each yes/no, exit 0):
- `go test ./... && go vet ./... && go build ./...` → exit 0.
- `go test ./internal/api/ -run VK -v` → the four mandatory gate cases pass.
- `go test ./internal/admin/ -run MCP -v` → the /mcp mirror cases pass.
- `go test ./internal/store/ -run Feature -v` → `vk_mandatory` seed/default pass.
- `go test ./internal/server/ -v` → (if added) end-to-end gate-wiring test passes.

**TDD-order proof** — each impl's covering test is in an earlier-or-equal commit:
```bash
for pair in \
  "internal/api/vk_test.go:internal/api/vk.go" \
  "internal/admin/mcp_test.go:internal/admin/mcp.go" ; do
  tf=${pair%%:*}; im=${pair##*:}
  ct=$(git log --format=%ct -1 -- "$tf"); cf=$(git log --format=%ct -1 -- "$im")
  [ "$ct" -le "$cf" ] || echo "TDD VIOLATION: $im"   # prints nothing
done
```

**Grep proofs:**
```bash
# flag seeded OFF (reuses the bf-core-2 INSERT OR IGNORE loop)
grep -n "vk_mandatory" internal/store/migrate.go                                  # in the seed slice
# injected predicate present + consumed in the empty-key branch (no-leftovers D2)
grep -n "mandatory func() bool\|SetMandatoryChecker" internal/api/vk.go           # field + setter
grep -n "virtual key required" internal/api/vk.go                                 # the mandatory reject
grep -n "g.mandatory" internal/api/vk.go                                          # the empty-key branch reads it
# signatures PRESERVED (additive only)
grep -n "func NewVKGate(resolver VKResolver, quota VKQuotaChecker) \*VKGate" internal/api/vk.go   # unchanged
grep -n "func (g \*VKGate) AllowVK(key, model, providerID string) (ok bool, status int, reason string, keyIDs \[\]byte\|keyIDs \[\]string)" internal/api/vk.go  # 4-return shape kept
# predicate wired LIVE per-request, reading the flag store (no-leftovers D3)
grep -n "SetMandatoryChecker(func() bool" internal/server/routes_openai.go        # wiring
grep -n "IsFeatureEnabled(\"vk_mandatory\")" internal/server/routes_openai.go internal/admin/mcp.go  # both surfaces read the SAME flag
# /mcp mirror rejects an absent VK under mandatory mode (no-leftovers D4)
grep -n "vkMandatory\|virtual key required" internal/admin/mcp.go                 # the mirror branch + reason
grep -niE "mandatory.*absent|absent.*mandatory|no vk.*reject|virtual key required" internal/admin/mcp_test.go  # the reject test exists
# NO new route, NO signature break, NO per-call-site churn, NO init
! grep -nE 'r\.(GET|POST|PUT|DELETE)\(' internal/server/routes_openai.go | grep -iE 'mandatory|vk_mandatory' && echo "no new route OK"
! grep -rn "func init(" internal/api/vk.go internal/admin/mcp.go && echo "no init() OK"
! grep -rn "AllowVK" internal/api/chat.go internal/api/completions.go internal/api/responses.go internal/api/audio.go internal/api/images.go internal/api/files.go internal/api/batches.go internal/api/messages.go internal/api/embeddings.go internal/api/input_tokens.go | grep -iE 'mandatory' && echo "no per-call-site mandatory churn OK"
# additive migration only
git diff <base>..HEAD -- internal/store/migrate.go | grep -E '^\+' | grep -iE 'DROP|RENAME' | wc -l   # = 0
# hermetic gate test (injected predicate; no real flag store / sleep / net)
! grep -nE 'time\.Now|time\.Sleep|http\.Get|net\.Dial' internal/api/vk_test.go && echo "hermetic OK"
```

**Behavioral acceptance (binary):**
- **Flag OFF (default / fresh / upgraded store):** a `/v1/*` request with NO
  `x-g0-vk` is ALLOWED (`AllowVK` returns ok=true), byte-identical to pre-bf-gov-4.
  A `/mcp` request with no VK is admitted. (No operator action needed; the seeded
  default is OFF.)
- **Flag ON + no resolved VK:** `AllowVK` returns `(false, 401, "virtual key
  required", nil)`; the `/mcp` `admitMCPVK` returns `("", false)` ⇒ `MCPServerPost`
  emits a JSON-RPC error "virtual key required" and `MCPServerSSE` rejects 401 before
  streaming.
- **Flag ON + valid VK:** the request is ALLOWED — the mandatory check is bypassed
  (key != ""), and the existing resolve/active/match/quota path runs unchanged.
- **Flag ON + provided invalid (unknown/inactive) VK:** rejected exactly as today
  (`"unknown virtual key"` 401 / `"virtual key inactive"` 403 on /v1/*; invalid-VK
  reject on /mcp) — bf-gov-4 does NOT change the provided-bad-VK path.
- The single gate seam enforces mandatory mode across all 10 `/v1/*` call sites with
  no per-site edit.

---

## 6. Validation commands

```bash
go test ./... && go vet ./... && go build ./...                 # exit 0 (binding)
go test ./internal/api/ -run VK -v
go test ./internal/admin/ -run MCP -v
go test ./internal/store/ -run Feature -v
go test ./internal/server/ -v
```
No UI build / Playwright needed — bf-gov-4 ships NO UI touch (the `vk_mandatory`
flag is toggled via the existing feature-flags admin surface) and NO mock
correction. Hermetic only (D6): the gate test injects a fake predicate; no test hits
the network, sleeps, or requires a real flag store for the unit-level gate case.

---

## 7. Freeze rules + matrix-flip + WORKFLOW + open-questions + no-leftovers

**Freeze rules (binding):**
- bf-gov-4 takes NO serial slot: it registers NO new HTTP routes. It edits `vk.go`
  (additive field + setter + empty-key branch body), `routes_openai.go` (the
  EXISTING vkGate-wiring block — NOT a route registration), and `admin/mcp.go` (the
  `admitMCPVK` body — additive mandatory branch).
- **No concurrent holder:** the bf-gov chain (bf-gov-1/2/3) is CLOSED and MERGED —
  vk.go and the routes_openai.go gate-wiring were touched by those merged plans; no
  open plan holds them. bf-mcp-1 (which authored `admitMCPVK`) is MERGED. Confirm at
  start: all prior holders merged; if any is still open, COORDINATE before editing.
- Migrations: additive flag-seed entry ONLY (the shipped `INSERT OR IGNORE` loop); no
  table, no column, no destructive DDL.
- `NewVKGate`/`AllowVK` signatures FROZEN — additive field + setter only (D2).
- No reverse-engineering of the absent Bifrost ref (ESC-REF-ABSENT) — build to the
  matrix behavior (reject when no resolved VK under the flag) + g0router conventions;
  the status code / JSON-RPC code / header name are g0router VARs, recorded.

**Matrix-flip (at close, in `.planning/parity/matrix/bifrost-governance.md:45`):**
- PAR-BF-GOV-034 → **HAVE (variant header)** — VK mandatory mode built: the
  `vk_mandatory` flag (seeded OFF) gates an injected predicate consulted in
  `AllowVK`'s empty-key branch (`internal/api/vk.go`) and mirrored in the `/mcp`
  `admitMCPVK` (`internal/admin/mcp.go`); flag OFF ⇒ today's behavior, flag ON ⇒
  absent-VK rejected 401 "virtual key required". The `x-bf-vk → x-g0-vk` header name
  is the VAR (D5; BIFROST-MAP §250 authorizes recording header-name as variant). Note
  cite bf-gov-4 + D1/D2/D3/D4/D5. (Was the AUDIT-FOUND unassigned BUILD gap.)

**`open-questions.md` (append at close):**
```
## bf-gov-4 — VK mandatory mode — 2026-06-15
- [ ] Per-VK / per-user / per-team mandatory scoping — bf-gov-4 ships a single GLOBAL runtime flag (vk_mandatory). Per-entity mandatory mode presupposes the per-VK scoping surface (bf-mcp-2 / Customer tier, ESC). Why: matches BIFROST-MAP §250 "an optional mandatory-VK setting"; per-entity is scope expansion.
- [ ] Mandatory-mode status/JSON-RPC error-code parity with Bifrost — bf-gov-4 uses 401 "virtual key required" on /v1/* and a -32600 JSON-RPC error / 401 SSE close on /mcp (g0router conventions). The matrix ref (main.go:783-797) is UNREADABLE (ESC-REF-ABSENT), so the exact Bifrost code is not matched, only the BEHAVIOR. Why: avoid inventing a Bifrost status the ref can't confirm.
- [ ] Header name VAR (x-bf-vk → x-g0-vk) + key prefix (vk_ → g0vk-) — g0router naming; behavior built header-name-agnostically (D5). Why: documented variant per BIFROST-MAP §250.
- [ ] /mcp mandatory-mode extension of bf-mcp-1's "VK optional" posture — bf-mcp-1 admitted an absent VK on /mcp; bf-gov-4 rejects it under the flag (D4). The bf-mcp-1 open item "mandatory-VK enforcement for /mcp" is now CLOSED by bf-gov-4's shared flag. Why: cross-link so the bf-mcp-1 deferral is marked resolved.
```

**`docs/WORKFLOW.md` (update at close):** add a bf-gov-4 row — VK mandatory mode
shipped (Go-only, no routes, no UI): `vk_mandatory` flag (seeded OFF) gates an
injected predicate in the VK gate empty-key branch + the /mcp admitMCPVK mirror;
OFF-by-default backward-compatible (absent VK allowed when off, rejected 401 "virtual
key required" when on); row 034 flipped MISSING → HAVE (variant header); the
AUDIT-FOUND unassigned BUILD gap is closed; no serial slot taken; ESC-REF-ABSENT
honored (built to matrix behavior + g0router conventions only).

**No-leftovers confirmation (binding):** bf-gov-4 adds the `vk_mandatory` flag
(consumed by BOTH predicates), the `mandatory func() bool` field + `SetMandatoryChecker`
setter (consumed by `AllowVK`'s empty-key branch — test-proven: OFF+no-VK ⇒ allowed,
ON+no-VK ⇒ 401, ON+valid-VK ⇒ allowed), the per-request store-reading predicate in
the routes_openai.go wiring (a real `IsFeatureEnabled` read), and the `admitMCPVK`
mandatory branch (test-proven /mcp absent-VK reject). EVERY new surface has a
grep-proven AND behavior-proven live consumer (§5). No dead field, no dead flag, no
resolved-but-unread predicate. If the predicate cannot be wired live (the empty-key
branch does not actually reject when it returns true, or the /mcp mirror does not
reject), the plan STOPS and escalates.
```
