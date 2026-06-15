# Micro-plan bf-gov-1 — VK↔Team hierarchy (governance, Go)

```
program: bifrost-parity (Waves 6+; CLI_ORCHESTRATOR.md governs, HANDOFF.md retired)
plan: bf-gov-1
status: READY (rev 1 — authored against merged Waves 0–7 governance tree, live tree;
  BIFROST-MAP.md ledger row bf-gov-1 §300; gov disposition table §238-257;
  architectural decision #7 §140-159; freeze rules §384-399)
runs: governance track. Disjoint from bf-gov-2 (lists.go — runs ∥). bf-gov-3
  extends bf-gov-1's hierarchy shape (internal serial on quota.go: bf-gov-1 → bf-gov-3).
branch: main (direct commits, per AGENTS.md "No PR workflow")
commit-prefix: phase-1/bf-gov-1: (matches the shipped bifrost chain prefix — verified in git log: `phase-1/bf-openai-4: ...`)
ref-source: BLOCKED — ESC-REF-ABSENT (BIFROST-MAP §47-68). The frozen Bifrost ref
  @ ca21298 is ABSENT on this host. The ONLY ground truth is
  .planning/parity/matrix/bifrost-governance.md + g0router's own
  docs/phases/phase-18-bifrost-features.md:64-69. Build to documented matrix
  behavior + g0router conventions; STOP-escalate on any undocumented Bifrost detail.
  NEVER build to a guessed Bifrost wire format.
go-serial-slot: NONE. bf-gov-1 does NOT touch internal/server/routes_admin.go nor
  routes_openai.go for new routes (the VK/Team CRUD routes ALREADY EXIST — §0.3).
  Internal serial: bf-gov-1 holds the FIRST edit to internal/governance/quota.go in
  the bf-gov-1 → bf-gov-3 chain (BIFROST-MAP §394). It RELEASES quota.go to bf-gov-3
  on close.
new-route: NO. Confirmed: bf-gov-1 registers NO new HTTP routes (§0.3).
```

---

## 0. Objective + ground truth

### 0.1 Objective

Upgrade g0router's **flat** virtual-key governance to a **2-level hierarchical**
budget+RPM evaluation (VK-level AND Team-level must BOTH pass), by linking each
virtual key to an optional owning team and threading that link through the
existing quota engine. Additive-only: a `team_id` column on `virtual_keys`, the
nil=true `IsActive` semantic reconciled to g0router's existing column, a
provider-config `AllowAllKeys` field, an inline budget single-owner validation,
and the 2-level check inside `internal/governance/quota.go`. NO new tables (the
`teams` table already exists), NO new routes (VK/Team CRUD already exist), NO
Customer tier, NO in-memory store, NO CAS, NO GORM hooks.

This is the bifrost-governance analogue of g0router's own already-chosen design:
`phase-18-bifrost-features.md:64-69` — *"hierarchical: key limit AND team limit
must both pass"* is the ONLY hierarchy planned. Customer tier is OUT by design.

### 0.2 Rows this plan closes (matrix = ground truth)

| Row | Behavior (matrix) | Disposition | Acceptance after bf-gov-1 |
|---|---|---|---|
| PAR-BF-GOV-001 | VK schema incl. IsActive, TeamID | **BUILD** (VK→Team only; CustomerID = ESC) | `team_id` additive col + `VirtualKey.TeamID *string`; `IsActive` reconciled (D1). CustomerID explicitly omitted (ESC, §3). PARTIAL→PARTIAL-upgraded (HAVE for the VK/Team/IsActive subset; Customer subset stays ESC). |
| PAR-BF-GOV-002 | VK provider config incl. AllowAllKeys | **BUILD** (AllowAllKeys; BlacklistedModels = bf-gov-2; many2many Keys join = ESC) | `ProviderConfig.AllowAllKeys bool` added + consumed by the match path (D5). PARTIAL→PARTIAL-upgraded. |
| PAR-BF-GOV-007 | Team schema (CustomerID excluded) | **BUILD** (Team exists; add nothing structural) | Team store already HAVE (`internal/store/teams.go`); bf-gov-1 only CONSUMES it for the hierarchy check. CalendarAligned/CustomerID = ESC (bf-gov-3 / Customer ESC). MISSING→HAVE (for the budget+RPM subset). |
| PAR-BF-GOV-009 | VK ownership mutual exclusion (TeamID XOR CustomerID) in BeforeSave | **BUILD** (degenerate: Customer tier absent) | With CustomerID ESC, the only owner is TeamID → mutual exclusion is trivially satisfied (D6). Documented; no GORM hook. MISSING→HAVE-by-design. |
| PAR-BF-GOV-010 | Hierarchical eval order Provider/Model → User → VK → Team → Customer | **BUILD** (VK→Team slice only) | `QuotaEngine.Allow` runs VK budget+RPM, then Team budget+RPM; both must pass (D3). Provider/User/Customer tiers = ESC. MISSING→PARTIAL (VK→Team slice HAVE). |
| PAR-BF-GOV-014 | Budget hierarchy eval: VK budgets → Team budgets → Customer budgets | **BUILD** (VK→Team) | 2-level budget check; Team budget enforced via live `SumCostByTeam` aggregate over `request_log` vs team `budget_usd` (D3/D8). Customer = ESC. MISSING→PARTIAL. |
| PAR-BF-GOV-020 | Rate-limit hierarchy eval: VK → provider-config → Team → Customer | **BUILD** (VK→Team) | 2-level RPM check; Team RPM from `teams.rate_limit_rpm` (D3). Provider-config-level RPM + Customer = ESC. MISSING→PARTIAL. |
| PAR-BF-GOV-042 (VK→Team only) | Provider-level budgets/RPM checked before VK-level | **BUILD** (Team-as-owner slice only) | Only the VK→Team ownership precedence is built (D3 precedence). The Provider/Model-level budget tier = ESC. MISSING→PARTIAL (VK→Team subset). |
| PAR-BF-GOV-012 | Budget single-owner mutual-exclusion in BeforeSave | **BUILD** (inline handler check, NOT GORM hook) | Inline validation in the budget-owner assignment path: a budget may name AT MOST one owner among {VK, Team} (D2). The GORM `BeforeSave` *mechanism* is ESC; the *validation behavior* is built inline. MISSING→HAVE (for the VK/Team owner set). |

**Honest scoping note:** every row above is closed only for the **VK→Team
2-level slice**. The Provider/Model/User/Customer tiers, the GORM `BeforeSave`
mechanism, the CAS in-memory store, and the many2many key join are ESC and are
listed explicitly in §3. No row is closed by inventing un-evidenced Bifrost
behavior; where the matrix's full behavior exceeds the VK→Team slice, the row
flips to **PARTIAL** (not HAVE) and the residual is recorded in
`open-questions.md` (§7).

### 0.3 Preconditions already satisfied (evidence — read 3 files, AGENTS.md)

- **VK store EXISTS** (`internal/store/virtualkeys.go`): `VirtualKey` wraps
  `schemas.VirtualKey` + flat `Key`/`IsActive bool`/`CreatedAt`/`UpdatedAt`
  (`virtualkeys.go:13-20`); config nested in a `config_json` blob via
  `virtualKeyConfig{ProviderConfigs,Budget,RateLimitRPM}`
  (`virtualkeys.go:22-51`); CRUD = `CreateVirtualKey`/`ListVirtualKeys`/
  `GetVirtualKeyByID`/`GetVirtualKeyByKey`/`UpdateVirtualKey`/`DeleteVirtualKey`
  (`virtualkeys.go:54-169`); `scanVirtualKey` helper (`:171-189`). Key prefix is
  `g0vk-` (`:78`).
- **`virtual_keys` table** (`migrate.go:74-82`): `id, key, name, config_json,
  is_active INTEGER NOT NULL DEFAULT 1, created_at, updated_at`. **NO `team_id`
  column yet.** `is_active` is a flat NOT-NULL-DEFAULT-1 int (NOT a nil-bool) —
  see D1.
- **Team store EXISTS** (`internal/store/teams.go`): `Team{ID,Name,BudgetUSD,
  BudgetUsedUSD,BudgetPeriod,RateLimitRPM,CreatedAt,UpdatedAt}` (`teams.go:11-20`)
  + full CRUD (`:23-120`) + `scanTeam` (`:122-134`). The `teams` table is
  pre-existing (created via the migrate tables slice; `phase-18:32-41`).
- **VK/Team CRUD ROUTES EXIST** — bf-gov-1 adds NONE. `phase-18:60-62` registers
  `GET/POST /api/virtual-keys`, `PUT/DELETE /api/virtual-keys/:id`, and
  `GET/POST /api/teams`, `PUT/DELETE /api/teams/:id`. **bf-gov-1 is NOT a
  routes_admin.go / routes_openai.go serial holder** (confirmed against
  BIFROST-MAP §300 "routes? NO (CRUD routes exist)").
- **Quota engine EXISTS and is FLAT** (`internal/governance/quota.go`):
  `QuotaEngine{spend SpendReader, clock func() time.Time, rpmHits map[...]}`
  (`:22-29`); `NewQuotaEngine(spend, clock)` (`:32`); `Allow(vk *VirtualKeyInfo,
  model)` runs `checkBudget` then `checkRPM` (`:48-56`); `VirtualKeyInfo{Key,
  BudgetLimit,BudgetPeriod,RateLimitRPM}` (`:14-20`). **Injectable clock is
  ALREADY present** (`clock func() time.Time`, `:25`) — D7 hermetic tests reuse it.
- **Quota test harness EXISTS and is hermetic** (`internal/governance/quota_test.go`):
  `fixedClock(t)` (`:31-33`) + `fakeSpendReader` (`:9-29`, an in-memory fake
  keyed by VK key — NO network, NO sleep, NO real `time.Now`). bf-gov-1 EXTENDS
  this harness; D7 forbids any real-time/sleep test.
- **Gate wiring EXISTS** (`internal/api/vk.go` + `internal/server/routes_openai.go`):
  `VKGate.AllowVK(key,model,providerID)` (`vk.go:55-84`) resolves a VK via the
  `VKResolver` seam, checks active + provider/model match, then calls
  `VKQuotaChecker.Allow(vk *VKInfo, model)` (`vk.go:33-37,76-82`). The store→api
  mapping is `storeVKToAPI` (`routes_openai.go:176-199`); the api→governance
  mapping is `vkQuotaAdapter.Allow` (`routes_openai.go:210-217`). **The team_id
  must flow store.VirtualKey → api.VKInfo → governance.VirtualKeyInfo** for the
  engine to do the 2-level check — these three adapter hops are the in-scope edit
  surface (D3 wiring).
- **Schemas** (`internal/schemas/governance.go`): `VirtualKey{ID,Name,
  ProviderConfigs,Budget *Budget,RateLimitRPM *int}` (`:4-10`);
  `ProviderConfig{Provider,AllowedModels,KeyIDs,Weight}` (`:13-18`);
  `Budget{Limit,Period,Used}` (`:21-25`). **NO TeamID, NO AllowAllKeys yet.**
- **Migrations are additive-only** (`migrate.go`): tables slice with
  `CREATE TABLE IF NOT EXISTS` (`:74-82`); additive `ensureColumn(db,table,column,
  decl)` loop (`:385-401`, e.g. `users.display_name`, `providers.prefix`).
  `ensureColumn` NEVER alters/drops (`:472-474`). bf-gov-1's `team_id` rides this
  exact loop. NO destructive DDL.

---

## 1. Decisions made (and why) — binding

### D1 — `IsActive *bool` (nil=true): reconcile to g0router's existing column (VAR)

The matrix wants `IsActive *bool` where nil resolves to true
(bifrost-governance.md:181, `IsActiveValue()`). g0router ALREADY has
`virtual_keys.is_active INTEGER NOT NULL DEFAULT 1` (`migrate.go:79`) and
`store.VirtualKey.IsActive bool` (`virtualkeys.go:17`), with `CreateVirtualKey`
defaulting `IsActive: true` (`virtualkeys.go:79`).

**Decision:** g0router represents *nil=true* via the **NOT NULL DEFAULT 1**
column — an absent/unspecified active flag persists as `1` (active), which is
behaviorally identical to Bifrost's nil=true. **Do NOT introduce a Go `*bool`**
(that would churn `scanVirtualKey`, the gate, and every caller for zero
behavioral gain and is the opposite of additive). The nil=true SQLite
representation is therefore: **NULL is never stored; the column default `1`
encodes the nil-means-active case.** Record PAR-BF-GOV-001's IsActive subdimension
as **VAR** (variant-by-design, g0router's flat bool over a DEFAULT-1 column),
HAVE. If an operator later requires a true tri-state (active/inactive/unset
distinguishable), that is an additive `*bool` follow-up recorded in
`open-questions.md` — NOT built here.

### D2 — Budget single-owner validation: inline handler check (NOT GORM hook)

PAR-BF-GOV-012's *mechanism* is GORM `BeforeSave` (ESC — g0router has no GORM,
AGENTS.md "No init()/global state"; matrix Go-Port note #1 says replace GORM with
`database/sql`). The *validation behavior* is buildable inline.

**Decision:** Add a pure validation function in the domain layer —
`governance.ValidateBudgetOwner(owner BudgetOwner) error` — that errors if MORE
THAN ONE owner among the *built* owner set {VirtualKeyID, TeamID} is non-empty.
Because the Customer tier is ESC (§3) and Provider/Model-config budget owners are
ESC, the built owner set is exactly {VK, Team}, so the check is: *a budget row may
name at most one of VK or Team.* It is called inline from any handler/store path
that assigns a budget owner. There is NO `BeforeSave` hook and NO new global
state — it is a value-in/error-out function (errors-as-values,
`fmt.Errorf("budget: %w")`). g0router's existing `Budget` schema
(`governance.go:21-25`) has NO owner FKs today, so the owner association is itself
part of the VK→Team linkage (the VK *is* the owner via `team_id`); the validation
guards the additive owner-assignment surface so a future Customer-tier plan
cannot silently create a multi-owner budget. **No-leftovers note:** if, at impl,
no handler path actually assigns a multi-owner budget (because the VK→Team link is
the only owner expression and is single-valued by construction), the function is
still consumed by a unit test asserting the >1-owner case errors — keeping it
guarded-but-live rather than dead. If T-validate finds the function has NO live
production caller, it MUST be folded into the VK-update path that sets `team_id`
(reject assigning a budget to a VK that already has a team owner conflict) so it
is genuinely consumed — else STOP + escalate per §3 no-leftovers.

### D3 — 2-level hierarchical evaluation: precedence + surfaced decision

The engine must evaluate VK-level AND Team-level limits; BOTH must pass
(`phase-18:64-69`). g0router's `QuotaEngine.Allow` (`quota.go:48-56`) currently
checks only VK budget + VK RPM.

**Decision — precedence (deterministic, documented):**
1. **VK budget** (existing `checkBudget`) — deny on exhaustion.
2. **VK RPM** (existing `checkRPM`) — deny on exceed.
3. **Team budget** (NEW) — if the VK has an owning team with a positive budget,
   deny on Team budget exhaustion.
4. **Team RPM** (NEW) — if the owning team has a positive RPM, deny on exceed.

The check is **fail-closed at the first failing level**, evaluated in the order
above (VK before Team — mirrors matrix PAR-BF-GOV-010's "VK → Team" inner order
for the two tiers g0router builds). A VK with no team (`team_id` empty) skips
steps 3–4 (un-teamed VKs are budget/RPM-checked at the VK level only — D4).

**Surfaced decision/error:** denial returns the existing tuple
`(ok=false, status, reason)`. Status codes follow g0router's established mapping
(`vk.go`): **429** for budget-exhausted / rate-limited (the engine's existing
convention, `quota.go:50,53`), preserving the current contract. The `reason`
string names the failing level explicitly so the `{data,error}` envelope is
diagnostic: `"budget exhausted"` (VK), `"rate limit exceeded"` (VK),
`"team budget exhausted"` (Team), `"team rate limit exceeded"` (Team). The
Bifrost `Decision`/`EvaluationResult` enum (PAR-BF-GOV-035/036) is bf-gov-3 scope
(BIFROST-MAP §249) — bf-gov-1 keeps the existing `(bool,int,string)` tuple and
does NOT introduce the enum (no-leftovers; that type belongs to bf-gov-3).

**Wiring (the three adapter hops, in-scope):**
- `governance.VirtualKeyInfo` gains `TeamID string`, `TeamBudgetLimit float64`,
  `TeamBudgetPeriod string`, and `TeamRateLimitRPM int`. The Team budget is
  enforced by aggregating real `request_log` cost for the team via
  `SumCostByTeam(TeamID, windowStart(TeamBudgetPeriod))` — see D8. Adding fields
  to this struct is additive (no signature change to `NewQuotaEngine` or `Allow`).
- `api.VKInfo` gains the same team fields (additive struct fields).
- `routes_openai.go`'s `storeVKToAPI` resolves the VK's `team_id` → loads the
  `Team` via `st.GetTeamByID` → populates the team fields on `VKInfo`; the
  `vkQuotaAdapter.Allow` copies them into `VirtualKeyInfo`. The resolver hop is
  the ONLY place a store read for the team happens (one extra `GetTeamByID` per
  resolve; acceptable — the resolve is already a DB read).

### D4 — `team_id` nullable for un-teamed VKs

**Decision:** `team_id` is **`TEXT NOT NULL DEFAULT ''`** (empty string = no
team), mirroring g0router's existing prefix/role columns
(`migrate.go:390-396`, all `NOT NULL DEFAULT ''`/`'user'`). g0router does NOT use
SQL NULL for "absent" on these additive columns (its convention is a NOT-NULL
default sentinel — consistent with `connections.last_error TEXT NOT NULL
DEFAULT ''`). An empty `team_id` means "un-teamed": the hierarchy check skips the
Team tier (D3 step 3–4 guarded by `team_id != ""`). `store.VirtualKey.TeamID`
is a Go `string` (NOT `*string`) to match the empty-string sentinel and avoid
nil-pointer churn through the adapters. (This is a deliberate VAR vs the matrix's
`TeamID *string` — g0router's additive-column convention is empty-string-sentinel,
matrix Go-Port note #5 already anticipates SQLite divergence.)

### D5 — `AllowAllKeys` placement + semantics

PAR-BF-GOV-002 wants `AllowAllKeys bool` on the provider config; the matrix quirk
(bifrost-governance.md:185) is *"empty Keys with AllowAllKeys=false means no keys
allowed."*

**Decision:** Add `AllowAllKeys bool` to `schemas.ProviderConfig`
(`governance.go:13-18`, additive field with `json:"allow_all_keys"`). It is
serialized into the existing `config_json` blob (no new column — the blob already
carries `ProviderConfigs`, `virtualkeys.go:24`). It is **consumed by the
key-selection path**: when a provider config matches (`vk.go:matchProviderConfig`,
`vk.go:88-104`) and `AllowAllKeys==true`, the gate returns no `keyIDs` pin
(meaning "any key for the provider is allowed" — falls through to normal
selection); when `AllowAllKeys==false` and `KeyIDs` is empty, the matrix quirk
("no keys allowed") is honored. **No-leftovers note:** `AllowAllKeys` MUST be read
in `AllowVK`'s keyIDs-pinning branch (`vk.go:73-75`) — if T-allowkeys cannot wire
it into a live decision, it is a dead field and the plan STOPS + escalates (§3).
The exact "no keys allowed" rejection vs "fall through" must be matrix-evidenced;
the matrix gives the quirk text but not the HTTP surface — if the rejection HTTP
shape is undocumented, default to "fall through to existing selection" (the
conservative, behavior-preserving choice) and record the deny-semantics question
in `open-questions.md` rather than inventing a 403.

### D6 — Ownership mutual exclusion is trivial with Customer tier ESC

PAR-BF-GOV-009 enforces `TeamID XOR CustomerID`. **Decision:** with CustomerID
ESC (§3), a VK can only ever name a TeamID, so the mutual-exclusion invariant is
satisfied by construction (there is no second owner field to conflict with).
Document this explicitly as **HAVE-by-design** rather than building a hook that
checks a field that does not exist. If/when a Customer-tier plan adds
`customer_id`, IT owns reinstating the XOR check — recorded in
`open-questions.md`.

### D7 — Hermetic tests, injectable clock (reuse existing)

`QuotaEngine` already takes `clock func() time.Time` (`quota.go:25,32`) and
`quota_test.go` already has `fixedClock` (`:31-33`) + the in-memory
`fakeSpendReader` (`:9-29`). **Decision:** ALL bf-gov-1 quota tests use
`fixedClock` + an extended fake spend reader; ANY time-window assertion advances
the clock by constructing a new `fixedClock(t2)` engine or by a clock the test
controls — **NO `time.Now`, NO `time.Sleep`, NO network, NO subprocess.** Store
tests use a temp/in-memory SQLite via the existing `store.Open` test pattern.
This is binding (Wave-7 hermetic lesson, BIFROST-MAP §494).

### D8 — Team spend attribution: live aggregate via `SumCostByTeam` (NOT the display accumulator)

`SpendReader.SumCostByAPIKey(key, sinceISO)` (`quota.go:10-11`) attributes spend
by the VK key string, implemented as `SELECT SUM(cost) FROM request_log WHERE
api_key = ? AND timestamp >= ?` (`requestlog.go:249-262`). The Team budget needs
*team-scoped* spend.

**Rejected alternative (inert — do NOT build):** comparing the team's persisted
`teams.budget_used_usd` (`teams.go:15`) against `budget_usd`. Verified against the
live tree: **NOTHING in the request path writes `teams.budget_used_usd`** — grep
shows it is only echoed by Team CRUD (create/update/scan in `store/teams.go` +
`admin/teams.go`). A Team-budget tier built on that column can NEVER deny → an
inert enforcement path (the Wave-5 dead-wiring / no-leftovers violation the gates
must reject). Forbidden.

**Decision (LIVE tier):** add an additive Store method
`SumCostByTeam(teamID, sinceISO string) (float64, error)` that computes team spend
the SAME way the VK tier does — by aggregating real `request_log` cost over every
VK belonging to the team (joining through the new `team_id` column):
```sql
SELECT SUM(cost) FROM request_log
WHERE api_key IN (SELECT key FROM virtual_keys WHERE team_id = ?)
  AND timestamp >= ?
```
Mirror `SumCostByAPIKey`'s `sql.NullFloat64` handling (`requestlog.go:250-261`):
invalid/empty SUM → `0`. Add `SumCostByTeam(teamID, sinceISO string) (float64,
error)` to the `SpendReader` interface (`quota.go:9-12`). The engine's
Team-budget check computes `SumCostByTeam(TeamID, windowStart(TeamBudgetPeriod))`
and denies when it exceeds the team's `budget_usd`, guarded by
`TeamBudgetLimit > 0`. This is a **real enforcement path** — it denies on actual
aggregate team spend, computed identically to and consistently with the VK tier;
no display-only accumulator is trusted. `teams.budget_used_usd` stays the existing
display-only field and is NOT enforced against.

**`VirtualKeyInfo` carries** `TeamID string`, `TeamBudgetLimit float64`
(the team's `budget_usd`), `TeamBudgetPeriod string` (window for the team sum),
and `TeamRateLimitRPM int`, all read at resolve time (via `GetTeamByID`). Team RPM
reuses the engine's in-memory `rpmHits` window keyed by a synthetic
`"team:"+teamID` key.

**Signature safety (verified against live tree):** adding a method to the
`SpendReader` interface does NOT change `NewQuotaEngine(spend, clock)` or
`Allow(vk, model)` (both unchanged — only the interface gains a method). The sole
production implementer is `*store.Store` (it already provides `SumCostByAPIKey`,
`requestlog.go:249`); it gains the additive `SumCostByTeam` method. The only other
implementer is the test fake `fakeSpendReader` (`quota_test.go:9-29`), which gains
a matching in-memory `SumCostByTeam` keyed by a `teamID` map — still hermetic (NO
DB/network/sleep, D7). Every `SpendReader` implementer is updated in this plan.

---

## 2. Target files

### IN-SCOPE — EXTEND (additive only)

| File | Change (additive ONLY) |
|---|---|
| `internal/store/migrate.go` | ADD `{"virtual_keys","team_id","TEXT NOT NULL DEFAULT ''"}` to the `ensureColumn` additive loop (`:385-401`). NOTHING else. |
| `internal/store/migrate_test.go` (EXTEND if present, else rely on store test) | Assert `virtual_keys.team_id` exists post-migrate (additive). |
| `internal/store/virtualkeys.go` | ADD `TeamID string` to `store.VirtualKey` (NOT into the `config_json` blob — it is a queryable column, D4); thread it through INSERT/UPDATE/SELECT statements + `scanVirtualKey`. PRESERVE all existing signatures. |
| `internal/store/virtualkeys_test.go` (EXTEND/CREATE) | RED first: create VK with team_id → get/list round-trips team_id; create without → team_id == "". |
| `internal/schemas/governance.go` | ADD `TeamID string \`json:"team_id,omitempty"\`` to `VirtualKey`; ADD `AllowAllKeys bool \`json:"allow_all_keys"\`` to `ProviderConfig`. Additive fields only. |
| `internal/store/requestlog.go` | ADD `SumCostByTeam(teamID, sinceISO string) (float64, error)` (additive method; subquery over `request_log` joined to `virtual_keys.team_id`, mirroring `SumCostByAPIKey` `:249-262` incl. `sql.NullFloat64` → 0). NOTHING else. |
| `internal/governance/quota.go` | ADD `SumCostByTeam` to the `SpendReader` interface (`:9-12`); ADD team fields to `VirtualKeyInfo`; ADD `checkTeamBudget` (calls `SumCostByTeam`, D8) / `checkTeamRPM` (or fold into `Allow` as steps 3–4, D3); ADD `ValidateBudgetOwner` + `BudgetOwner` value type (D2). PRESERVE `NewQuotaEngine`/`Allow` signatures (interface gains a method only). INTERNAL SERIAL slot. |
| `internal/governance/quota_test.go` (EXTEND) | RED first: 2-level cases (D3) asserting team denial via the fake's `SumCostByTeam` (D8) + `ValidateBudgetOwner` cases (D2). ADD a `SumCostByTeam` method to `fakeSpendReader` (in-memory `teamID`→cost map). `AllowAllKeys` is NOT here (gate-level, lives in vk_test.go). Hermetic (D7). |
| `internal/api/vk.go` | ADD team fields to `api.VKInfo`; ADD `AllowAllKeys` to `api.VKProviderConfig`; consume `AllowAllKeys` in `AllowVK` keyIDs branch (D5). PRESERVE `NewVKGate`/`AllowVK` signatures. |
| `internal/api/vk_test.go` (EXTEND/CREATE) | RED first: `AllowAllKeys` semantics (D5); team-field passthrough to the quota checker (via a fake `VKQuotaChecker`). |
| `internal/server/routes_openai.go` | EXTEND `storeVKToAPI` to load the owning team (`st.GetTeamByID`) and populate team fields; EXTEND `vkQuotaAdapter.Allow` to copy team fields into `VirtualKeyInfo`; map `AllowAllKeys` in the `VKProviderConfig` build (`:191-197`). **NO new route registration.** This is adapter-body edits only, NOT the serial route block. |

### FORBIDDEN (automatic REJECT if touched)

- `internal/server/routes_admin.go` — bf-gov-1 registers NO admin routes.
- Any **new route registration** in `routes_openai.go` (only the existing
  `storeVKToAPI` / `vkQuotaAdapter` / `VKProviderConfig` adapter bodies change).
- `internal/store/teams.go` — CONSUMED read-only (`GetTeamByID`); NOT edited
  (Team CRUD already complete).
- `internal/schemas/lists.go` / any WhiteList/BlackList type — that is **bf-gov-2**.
- The `Decision`/`EvaluationResult` enum, dual-dimension (token+request) rate
  limits, calendar-aligned reset, streaming `UsageUpdate`, 10s sync worker — all
  **bf-gov-3**.
- Any Customer schema/column/table; any `sync.Map` in-memory store; any CAS-spin
  bump; any GORM hook (`BeforeSave`/`AfterFind`); any many2many key join table —
  **ESC** (§3).
- Any UI file (`ui/**`) — bf-gov-1 is Go-only; the VK/Team admin pages already
  consume the existing CRUD (no DTO field these changes surface to the UI beyond
  the additive `team_id`/`allow_all_keys`, which the pages may ignore).

---

## 3. Scope / Non-goals — explicit ESC list

**bf-gov-1 builds ONLY the VK→Team 2-level slice.** The following matrix rows /
behaviors are **ESC** (out of scope; recorded in `open-questions.md` at close):

| ESC item | Matrix row(s) | Why ESC |
|---|---|---|
| **Customer tier** (schema, column, budgets, rate-limits, AfterFind propagation) | 008, 019(customer), 046, 047; CustomerID dimension of 001/009 | `phase-18` design is VK+Team only (BIFROST-MAP §251); a third tier is explicit scope expansion. |
| **`sync.Map` lock-free in-memory governance store** + 40-method `GovernanceStore` interface | 004, 049, 050 | g0router uses `database/sql` reads, not an in-memory mirror (BIFROST-MAP §252, Go-Port note #2). |
| **CAS-spin atomic bumps** (`BumpBudgetUsage`/`BumpRateLimitUsage`) | 013, 018, 041 | Presuppose the in-memory atomic store; g0router accrues via SQL/stored accumulators (BIFROST-MAP §253). |
| **GORM `BeforeSave`/`AfterFind` hooks** (the *mechanism*) | 005, 012(mechanism), 037, 046, 047 | g0router has no GORM (Go-Port note #1). The *budget single-owner validation behavior* (012) IS built inline (D2); the hook mechanism is ESC. |
| **Provider/Model/User tiers** of the full hierarchy chain | 010(P/M/U), 020(provider-config), 042(provider-level), 043 | bf-gov-1 builds the VK→Team slice only; the Provider/Model/User tiers and provider-config-level RPM are not in the `phase-18` 2-level design. |
| **Many2many provider-config↔Keys join table** | 002(Keys) | g0router pins keys via `KeyIDs []string` in the config blob (`governance.go:16`); a join table is a divergent rework. `AllowAllKeys` (the in-scope part of 002) IS built (D5). |
| **VK value SHA-256 hash-index + AES-at-rest** | 006 | g0router uses `g0vk-` random stored keys (`virtualkeys.go:78`); hash-indexed-encrypted lookup is a divergent rework (BIFROST-MAP §254). |
| **BlacklistedModels / WhiteList-BlackList typed semantics** | 026–030, 037, 048 | These are **bf-gov-2** (BIFROST-MAP §245). |
| **Dual-dimension RL / calendar reset / streaming accrual / Decision enum / 10s worker** | 015, 017, 021–023, 035, 036, 038, 039, 044, 045, 016 | These are **bf-gov-3** (BIFROST-MAP §246-249). |
| **Model-catalog cross-provider allowlist / ghost-node recon / governance-context stamping / load-balance weighted selection** | 024, 025, 031, 032, 033, 040, 043 | Cluster/catalog-coupled; 033 weighted selection is already SAT via `inference.SelectionEngine` (BIFROST-MAP §256). |

No-leftovers (binding, §3 CLI_ORCHESTRATOR): bf-gov-1 adds a column/field ONLY if
the hierarchy check (D3) or the gate decision (D5) or a live validation path (D2)
actually consumes it. D2/D5 each carry an explicit STOP-condition if the
added surface has no live consumer.

---

## 4. Task graph (TDD; `N. [step] -> verify: [check]`)

Cadence (AGENTS.md "TDD always"): no impl file/field lands before its covering
`_test.go` is committed RED. `go test ./... && go vet ./... && go build ./...`
green at EVERY commit. Internal serial: bf-gov-1 holds quota.go.

1. **[migration + store team_id, RED]** Write `virtualkeys_test.go` additions
   (create-with/without team_id round-trip) + add the `team_id` `ensureColumn`
   row to `migrate.go`. -> verify: `go test ./internal/store/ -run VirtualKey`
   FAILS (TeamID field/column missing). Commit RED:
   `phase-1/bf-gov-1: failing VK team_id store test (TDD red)`.

2. **[store team_id, GREEN]** Add `TeamID string` to `store.VirtualKey`; thread
   through INSERT/UPDATE/SELECT + `scanVirtualKey`. -> verify:
   `go test ./internal/store/... && go vet ./... && go build ./...` exit 0; the
   round-trip test passes; `team_id` is a queryable column (NOT in config_json).
   Commit: `phase-1/bf-gov-1: VK team_id column (additive)`.

3. **[schema fields, GREEN]** Add `VirtualKey.TeamID` + `ProviderConfig.AllowAllKeys`
   to `schemas/governance.go` (additive). -> verify: `go build ./...` exit 0;
   `grep -n 'team_id\|allow_all_keys' internal/schemas/governance.go` non-empty.
   (Folds into commit 2 or its own micro-commit.)

4. **[SumCostByTeam store method, RED→GREEN]** Write a `requestlog_test.go`
   (EXTEND/CREATE) case: seed `request_log` rows for two VKs, one with
   `team_id=T`, assert `SumCostByTeam(T, since)` sums only T's VKs' cost (and 0
   for an unknown team / pre-window rows). Add `SumCostByTeam` to
   `internal/store/requestlog.go` (subquery join, D8). -> verify:
   `go test ./internal/store/ -run SumCostByTeam` green;
   `go vet ./... && go build ./...` exit 0. Commit RED then GREEN:
   `phase-1/bf-gov-1: failing SumCostByTeam store test (TDD red)` then
   `phase-1/bf-gov-1: SumCostByTeam request_log aggregate (additive)`.

5. **[2-level quota engine, RED]** Extend `quota_test.go` (D7 hermetic): add
   `SumCostByTeam` to `fakeSpendReader`; VK pass + team aggregate spend over
   `budget_usd` → deny 429 "team budget exhausted"; VK RPM pass + Team RPM
   exceeded → deny 429 "team rate limit exceeded"; un-teamed VK skips Team tier;
   both-pass → allow. -> verify: `go test ./internal/governance/ -run Team`
   FAILS. Commit RED: `phase-1/bf-gov-1: failing 2-level hierarchy tests (TDD red)`.

6. **[2-level quota engine, GREEN]** Add `SumCostByTeam` to the `SpendReader`
   interface; add team fields to `VirtualKeyInfo`; implement `checkTeamBudget`
   (calls `SumCostByTeam(TeamID, windowStart(TeamBudgetPeriod))` vs
   `TeamBudgetLimit`, guard `>0`, D8) + `checkTeamRPM` (synthetic `team:<id>` rpm
   window) wired into `Allow` after the VK checks (D3 precedence). -> verify:
   `go test ./internal/governance/... && go vet ./... && go build ./...` exit 0;
   precedence test green (VK denial reported before Team). Commit:
   `phase-1/bf-gov-1: 2-level VK+Team budget/RPM hierarchy in quota engine`.

7. **[budget single-owner validation, RED→GREEN]** Add `ValidateBudgetOwner` +
   `BudgetOwner` + tests: >1 owner among {VK,Team} → error; ≤1 → nil. Wire the
   live consumer per D2 (VK-update path or assignment path). -> verify:
   `go test ./internal/governance/ -run BudgetOwner` green; the function has ≥1
   live production caller (grep proof §5) OR STOP+escalate. Commit:
   `phase-1/bf-gov-1: inline budget single-owner validation`.

8. **[gate AllowAllKeys, RED→GREEN]** Extend `vk_test.go`: `AllowAllKeys=true` →
   no keyID pin / fall-through; `AllowAllKeys=false` + empty KeyIDs → matrix
   quirk (default fall-through, deny only if matrix-evidenced — D5). Add
   `AllowAllKeys` to `api.VKProviderConfig` + consume in `AllowVK`. -> verify:
   `go test ./internal/api/ -run VK` green; `AllowAllKeys` read in a live
   decision branch (grep proof) OR STOP+escalate. Commit:
   `phase-1/bf-gov-1: AllowAllKeys provider-config semantics in VK gate`.

9. **[adapter wiring, RED→GREEN]** Extend the `storeVKToAPI` / `vkQuotaAdapter` /
   `VKProviderConfig` mapping in `routes_openai.go` to load the owning team
   (`GetTeamByID`) and thread `TeamID` + team fields + `AllowAllKeys` end-to-end;
   add a server-level test proving a VK whose team's aggregate `request_log` spend
   exceeds `budget_usd` is denied via the gate. -> verify: `go test
   ./internal/server/... && go test ./... && go vet ./... && go build ./...`
   exit 0; NO new route registered (grep proof §5). Commit:
   `phase-1/bf-gov-1: wire team hierarchy through VK resolver+quota adapter`.

10. **[close]** Run full validation (§6); flip matrix rows (§7); update
    `open-questions.md` (ESC list §3 + D-deferred items); update
    `docs/WORKFLOW.md`; RELEASE the quota.go internal-serial slot to bf-gov-3.
    -> verify: §6 all green; matrix + WORKFLOW + open-questions committed. Commit:
    `phase-1/bf-gov-1: close — VK↔Team hierarchy; matrix flip; serial release`.

---

## 5. Acceptance criteria (binary; file:line where possible)

**Test gates** (each yes/no, exit 0):
- `go test ./... && go vet ./... && go build ./...` → exit 0.
- `go test ./internal/governance/ -run 'Team|BudgetOwner' -v` → all pass
  (≥4 hierarchy cases + ≥2 owner cases).
- `go test ./internal/store/ -run VirtualKey -v` → team_id round-trip passes.
- `go test ./internal/api/ -run VK -v` → AllowAllKeys cases pass.
- `go test ./internal/server/ -v` → team-budget-denial-through-gate passes.

**TDD-order proof** — each impl's covering test is in an earlier-or-equal commit:
```bash
for pair in \
  "internal/store/virtualkeys_test.go:internal/store/virtualkeys.go" \
  "internal/store/requestlog_test.go:internal/store/requestlog.go" \
  "internal/governance/quota_test.go:internal/governance/quota.go" \
  "internal/api/vk_test.go:internal/api/vk.go" ; do
  tf=${pair%%:*}; im=${pair##*:}
  ct=$(git log --format=%ct -1 -- "$tf"); cf=$(git log --format=%ct -1 -- "$im")
  [ "$ct" -le "$cf" ] || echo "TDD VIOLATION: $im"   # prints nothing
done
```

**Grep proofs:**
```bash
# team_id is a queryable COLUMN (not buried in config_json)
grep -n "team_id" internal/store/migrate.go internal/store/virtualkeys.go        # present in both
! grep -n '"team_id"' internal/store/virtualkeys.go | grep -i config_json && echo "team_id not in blob OK"
# 2-level check present and ordered (VK before Team)
grep -n "checkTeamBudget\|checkTeamRPM\|team budget exhausted\|team rate limit" internal/governance/quota.go
grep -n "TeamBudgetLimit\|TeamRateLimitRPM" internal/governance/quota.go internal/api/vk.go internal/server/routes_openai.go
# Team budget tier is LIVE (real aggregate, not the inert display accumulator — D8)
grep -n "func (s \*Store) SumCostByTeam" internal/store/requestlog.go         # additive store method
grep -n "SumCostByTeam" internal/governance/quota.go                          # in SpendReader iface + checkTeamBudget
! grep -n "budget_used_usd" internal/governance/quota.go && echo "team tier does NOT enforce against display accumulator OK"
# AllowAllKeys is CONSUMED in a live decision (no-leftovers D5)
grep -n "AllowAllKeys" internal/api/vk.go                                        # read in AllowVK
grep -n "allow_all_keys" internal/schemas/governance.go                          # additive json field
# budget single-owner validation present + has a live caller (no-leftovers D2)
grep -n "func ValidateBudgetOwner" internal/governance/quota.go
grep -rn "ValidateBudgetOwner(" internal/ --include='*.go' | grep -v _test.go    # ≥1 production caller
# NO new route, NO admin route, NO forbidden mechanisms
! grep -nE 'r\.(GET|POST|PUT|DELETE)\(' internal/server/routes_openai.go | grep -iE 'team|virtual' && echo "no new VK/team route OK"
test ! -e internal/store/customers.go && echo "no Customer tier OK"
! grep -rn "sync.Map\|CompareAndSwap\|BeforeSave\|AfterFind" internal/governance/ internal/store/virtualkeys.go && echo "no GORM/CAS/in-mem-store OK"
! grep -rn "func init(" internal/governance/quota.go internal/api/vk.go internal/store/virtualkeys.go && echo "no init() OK"
# additive migration only
git diff <base>..HEAD -- internal/store/migrate.go | grep -E '^\+' | grep -iE 'DROP|RENAME' | wc -l   # = 0
# hermetic tests (no real time / sleep / net in the new test bodies)
! grep -nE 'time\.Now|time\.Sleep|http\.Get|net\.Dial' internal/governance/quota_test.go && echo "hermetic OK"
```

**Behavioral acceptance (binary):**
- A VK whose owning team's aggregate `request_log` spend (`SumCostByTeam`) within
  the team window exceeds the team's `budget_usd` is DENIED with status 429 and
  reason `"team budget exhausted"`, even when the VK's own budget passes. (Proven
  hermetically via the `fakeSpendReader.SumCostByTeam` map; and at the store layer
  via seeded `request_log` rows.)
- A VK whose owning team's RPM is exceeded is DENIED 429 `"team rate limit
  exceeded"`, even when the VK's own RPM passes.
- A VK with `team_id == ""` is evaluated at the VK level ONLY (Team tier skipped);
  behavior is byte-identical to pre-bf-gov-1 for un-teamed keys.
- `ValidateBudgetOwner` errors when >1 of {VK,Team} owner is set; passes for ≤1.
- A provider config with `AllowAllKeys=true` returns no pinned keyIDs (D5).

---

## 6. Validation commands

```bash
go test ./... && go vet ./... && go build ./...                 # exit 0 (binding)
go test ./internal/governance/ -run 'Team|BudgetOwner' -v
go test ./internal/store/ -run VirtualKey -v
go test ./internal/api/ -run VK -v
go test ./internal/server/ -v
```
No UI build / Playwright needed — bf-gov-1 ships NO UI touch and NO mock
correction (the additive `team_id`/`allow_all_keys` fields surface to no
asserted UI behavior). Hermetic only (D7): no test may hit the network, sleep,
or call real `time.Now`.

---

## 7. Freeze rules + matrix-flip + WORKFLOW + no-leftovers

**Freeze rules (binding):**
- `internal/governance/quota.go` — bf-gov-1 → bf-gov-3 INTERNAL SERIAL
  (BIFROST-MAP §394). bf-gov-1 holds the first edit; RELEASE to bf-gov-3 on the
  close commit (step 10).
- bf-gov-1 is **NOT** a `routes_admin.go` holder and **NOT** a `routes_openai.go`
  route-block holder (it edits only adapter bodies in routes_openai.go, never the
  route-registration block; confirmed §0.3 / §2 forbidden). It takes NO serial
  route slot.
- Migrations: additive `ensureColumn` ONLY; no destructive DDL (`migrate.go`
  ensureColumn never alters/drops, `:472-474`).
- bf-gov-2 (`lists.go`) runs ∥; quota.go (gov-1) vs lists.go (gov-2) are disjoint
  (BIFROST-MAP §330).
- No reverse-engineering of the absent Bifrost ref (ESC-REF-ABSENT) — build to
  matrix + g0router conventions only.

**Matrix-flip (at close, in `.planning/parity/matrix/bifrost-governance.md`):**
- PAR-BF-GOV-001 → **PARTIAL** (VK/Team/IsActive subset HAVE per D1/D4; CustomerID
  subset ESC). Note cite bf-gov-1 + D1/D4.
- PAR-BF-GOV-002 → **PARTIAL** (AllowAllKeys HAVE per D5; BlacklistedModels →
  bf-gov-2; many2many Keys join ESC).
- PAR-BF-GOV-007 → **HAVE** (Team budget+RPM subset; CalendarAligned/CustomerID ESC).
- PAR-BF-GOV-009 → **HAVE** (by-design per D6; Customer XOR re-check deferred).
- PAR-BF-GOV-010 → **PARTIAL** (VK→Team slice HAVE; Provider/User/Customer ESC).
- PAR-BF-GOV-012 → **HAVE** (inline single-owner validation per D2; GORM hook ESC).
- PAR-BF-GOV-014 → **PARTIAL** (VK→Team budget hierarchy HAVE; Customer ESC).
- PAR-BF-GOV-020 → **PARTIAL** (VK→Team RPM hierarchy HAVE; provider-config + Customer ESC).
- PAR-BF-GOV-042 → **PARTIAL** (VK→Team ownership precedence HAVE; Provider/Model tier ESC).

**`open-questions.md` (append at close):**
```
## bf-gov-1 — VK↔Team hierarchy — 2026-06-15
- [ ] Customer tier (gov 008/019/046/047 + CustomerID on 001/009) — ESC; needs operator decision (BIFROST-MAP Escalation §7). Why: third governance tier = scope expansion.
- [ ] teams.budget_used_usd display accumulator — bf-gov-1 does NOT enforce against it (it has no live writer in the request path); the Team budget tier enforces via the live SumCostByTeam request_log aggregate instead (D8). The post-request write to keep budget_used_usd as an accurate DISPLAY figure remains phase-18:69 middleware / bf-gov-3 work. Why: tracked so the display column does not silently diverge from the enforced aggregate.
- [ ] AllowAllKeys deny-semantics ("empty Keys + AllowAllKeys=false → no keys allowed") — HTTP rejection shape undocumented in matrix; bf-gov-1 defaults to fall-through (D5). Why: avoid inventing a 403 the ref can't confirm (ESC-REF-ABSENT).
- [ ] IsActive tri-state (true *bool nil-distinguishable) — bf-gov-1 uses NOT-NULL-DEFAULT-1 column (D1); a true *bool is an additive follow-up if operator needs unset≠active.
- [ ] Decision/EvaluationResult enum (gov 035/036) — deferred to bf-gov-3; bf-gov-1 keeps the (bool,int,string) tuple.
```

**`docs/WORKFLOW.md` (update at close):** add a bf-gov-1 row — VK↔Team 2-level
hierarchy shipped (Go-only, no routes, no UI); rows 001/002/007/009/010/012/014/
020/042 flipped per §7; ESC items recorded in open-questions; quota.go serial
released to bf-gov-3; ESC-REF-ABSENT honored (built to matrix only).

**No-leftovers confirmation (binding):** bf-gov-1 adds `team_id` (consumed by the
D3 Team-tier check), `AllowAllKeys` (consumed by the D5 gate branch — STOP if no
live consumer), `ValidateBudgetOwner` (consumed by a live VK-update/assignment
path — STOP if no live caller), and the team fields on `VirtualKeyInfo`/`VKInfo`
(consumed by the engine + adapters end-to-end). No dead column, field, or method
is introduced; each new surface has a grep-proven live consumer (§5) or the plan
STOPS and escalates.
```
