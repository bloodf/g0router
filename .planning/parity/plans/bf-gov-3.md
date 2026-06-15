# Micro-plan bf-gov-3 — dual-dimension RL (SQL-live) + calendar reset + Decision enum (governance, Go)

```
program: bifrost-parity (Waves 6+; CLI_ORCHESTRATOR.md governs, HANDOFF.md retired)
plan: bf-gov-3
status: READY (rev 1 — authored against the merged bf-gov-1 governance tree, live tree;
  BIFROST-MAP.md ledger row bf-gov-3 §302; gov disposition table §238-257;
  architectural decision #7 §140-159; Go-Port note #7 (time.Ticker) matrix:215;
  freeze rules §384-399). EXTENDS bf-gov-1 (VK↔Team hierarchy, SHIPPED).
runs: governance track. SECOND and FINAL holder of the internal serial on
  internal/governance/quota.go (bf-gov-1 → bf-gov-3). bf-gov-1 RELEASED quota.go on
  its close; bf-gov-3 holds it now and releases to NOBODY (chain end). Disjoint from
  bf-gov-2 (lists.go runs ∥).
branch: main (direct commits, per AGENTS.md "No PR workflow")
commit-prefix: phase-1/bf-gov-3:
commit-footer: Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>
ref-source: BLOCKED — ESC-REF-ABSENT (BIFROST-MAP §47-68). The frozen Bifrost ref
  @ ca21298 is ABSENT on this host. The ONLY ground truth is
  .planning/parity/matrix/bifrost-governance.md + g0router's own
  docs/phases/phase-18-bifrost-features.md. Build to documented matrix behavior +
  g0router conventions; STOP-escalate on any undocumented Bifrost detail. NEVER build
  to a guessed Bifrost wire format (no CAS-spin shape, no sync.Map store, no GORM
  AfterFind, no in-memory→DB sync worker — all ESC/VAR, §3).
go-serial-slot: NONE for routes. bf-gov-3 touches NO routes_admin.go, NO new
  routes_openai.go route registration, and NO internal/server/server.go lifecycle
  change (confirmed §0.3 / §2 FORBIDDEN). Every governance dimension is SQL-live over
  request_log (no in-memory accumulator → no sync worker → no lifecycle wiring → no
  new table).
internal-serial: bf-gov-3 holds internal/governance/quota.go (final holder).
new-route: NO. bf-gov-3 registers NO HTTP routes (§0.3).
```

---

## 0. Objective + ground truth

### 0.1 Objective

EXTEND bf-gov-1's shipped 2-level VK↔Team quota engine with four additive
governance capabilities, all over the EXISTING engine surface (no signature churn),
and ALL fully SQL-live over `request_log` — exactly like bf-gov-1's budget tier
(D8) — so there is NO in-memory governance accumulator to persist and therefore NO
sync worker, NO new table, and NO server-lifecycle change:

1. **Dual-dimension rate limit (BOTH dimensions SQL-live)** — a token-limit AND a
   request-limit, each with its own `max` + `resetDuration`, enforced by a live
   aggregate over `request_log` within the calendar window. Token usage =
   `SUM(prompt_tokens + completion_tokens)` (NEW additive `SumTokensByAPIKey`);
   request usage = `COUNT(*)` (NEW additive `SumRequestsByAPIKey`) — both mirroring
   the SHIPPED `SumCostByAPIKey` (`requestlog.go:249-262`). No in-memory `usage`/
   `lastReset` counter, no CAS, no accrual method.
2. **Calendar-aligned LAZY reset (INHERENT for SQL-live dimensions)** — REUSE the
   SHIPPED `windowStart` (daily/weekly/monthly) + injectable `clock` (bf-gov-1
   already built calendar alignment for the budget window; do NOT re-add it). For an
   SQL-live dimension the lazy reset is INHERENT: the `windowStart(period)` lower
   bound in the WHERE clause means a rolled-over window naturally aggregates only the
   current period's rows (matrix quirk #8) — there is no counter to reset and no
   `lastReset` to track. `windowStart`'s `default` branch is extended additively to
   parse rolling-duration tokens (`1h`/`1d`/`1M`) without touching the 3 shipped
   calendar cases.
3. **`Decision` enum + `EvaluationResult`** — a typed governance outcome
   (Allow/VirtualKeyNotFound/RateLimited/BudgetExceeded/TokenLimited/RequestLimited/
   ModelBlocked/...) surfaced ADDITIVELY via a NEW sibling `Evaluate` method; the
   SHIPPED `Allow(bool,int,string)` is re-expressed as a thin wrapper over `Evaluate`
   (signature PRESERVED). Mapped to g0router's `{data,error}` snake_case envelope and
   surfaced LIVE through the gate `error.code`.
4. **Inline rate-limit validation** — `ValidateRateLimit` (a pure value-in/error-out
   fn, mirror of bf-gov-1's `ValidateBudgetOwner`) requiring a reset duration when a
   max is set + validating its form, called LIVE from the VK admin create/update
   path. Replaces Bifrost's GORM `BeforeSave` mechanism (ESC).

This is the bifrost-governance analogue of g0router's own design
(`phase-18-bifrost-features.md:64-69,72-73` — "hierarchical … both pass" + lazy
reset). The Customer tier, the `sync.Map` in-memory store, CAS-spin bumps, GORM
`AfterFind`, the 10s in-memory→DB **sync worker** (VAR-by-design — g0router has no
in-memory governance state to sync, §3), the streaming `UsageUpdate`/`Accrue` type
(VAR — `SaveUsage` is already the streaming-aware finalized write, §3), and the
cluster remote-baseline rate check are ALL ESC/VAR (§3). **Streaming finalization
(021/022)** is satisfied by VERIFYING the existing write path: `SaveUsage`
(`requestlog.go:36`) writes exactly ONE `request_log` row per COMPLETED request with
final token usage, which IS the matrix finalization rule (tokens counted once with
usage data; one request on completion, never per-chunk) — so the SQL-live aggregates
inherently honor it, with no `UsageUpdate` type built.

### 0.2 Rows this plan closes (matrix = ground truth)

| Row | Behavior (matrix) | Disposition | Acceptance after bf-gov-3 |
|---|---|---|---|
| PAR-BF-GOV-017 | Rate limit schema: dual token/request limits w/ MaxLimit, ResetDuration, CurrentUsage, LastReset per dimension | **BUILD** | `TokenMax/TokenResetPeriod` + `RequestMax/RequestResetPeriod` additive config fields; both dimensions enforced in `Evaluate` via LIVE request_log aggregates (no in-memory `CurrentUsage`/`LastReset` — those are SQL-derived, D1/D3). PARTIAL→PARTIAL-upgraded (dual-dimension HAVE; the persisted-counter `CurrentUsage`/`LastReset` schema is VAR-by-design = SQL-live; CAS bump ESC). |
| PAR-BF-GOV-038 | `BeforeSave` on RateLimit validates reset-duration format + requires reset duration when max limit set | **BUILD** (inline, NOT GORM hook) | `governance.ValidateRateLimit(...)` pure fn: errors if `Max>0 && ResetDuration==""`, and if `ResetDuration` is not one of the accepted forms (D5). GORM BeforeSave mechanism ESC; behavior built inline. MISSING→HAVE (validation behavior). |
| PAR-BF-GOV-015 | Budget calendar-aligned reset (`IsCalendarAligned` derived from owner; reset path reads stamped value) | **BUILD** (reuse SHIPPED windowStart) | bf-gov-1 ALREADY built daily/weekly/monthly `windowStart` (`quota.go:192-206`). bf-gov-3 reuses it for the rate-limit dimensions' calendar window (D2). The owner-derived `IsCalendarAligned` PROPAGATION via AfterFind is ESC (GORM); g0router stamps the period directly. MISSING→HAVE (calendar window for the built dimensions). |
| PAR-BF-GOV-019 | Rate-limit calendar-aligned reset via `GetCalendarPeriodStart` | **BUILD** (reuse SHIPPED windowStart) | The rate-limit lazy reset uses the SAME `windowStart(period)` the budget uses (D2/D3). MISSING→HAVE. |
| PAR-BF-GOV-039 | `CheckBudget` treats expired (rolling-window-elapsed) budgets as reset by skipping the check | **BUILD** | Lazy reset is INHERENT for SQL-live dimensions: the `windowStart(period)` lower bound makes a rolled-over window aggregate only current-period rows (matrix quirk #8) — no counter, no `lastReset` (D2/D3). Applied to both rate-limit dimensions. MISSING→HAVE. |
| PAR-BF-GOV-021 | `UsageUpdate` struct w/ streaming-aware fields: IsStreaming, IsFinalChunk, HasUsageData | **VAR (verify, do not build)** | g0router's `SaveUsage` (`requestlog.go:36`) writes ONE `request_log` row per COMPLETED request with final token usage — this IS the matrix finalization rule (tokens counted once with usage data; one request on completion, never per-chunk). No separate `UsageUpdate` type is needed; the SQL-live aggregates honor it inherently. bf-gov-3 adds a test asserting a streamed request yields exactly one row with final usage (or cites existing coverage). MISSING→VAR (streaming-aware finalized write is the existing SaveUsage). |
| PAR-BF-GOV-022 | `UpdateUsage` order: global → user → per-user-scoped → VK → per-VK-scoped | **PARTIAL** (VK→Team slice only) | The VK→Team accrual IS the persisted `request_log` row consumed live by the dual-dimension aggregates (D1/D3). global/user/per-user-scoped tiers ESC (§3, same slice boundary as bf-gov-1). MISSING→PARTIAL (VK→Team slice HAVE via the persisted rows; no `Accrue` method). |
| PAR-BF-GOV-023 | Background reset worker: every 10s resets expired RLs + budgets, dumps to DB | **VAR/ESC (by-design)** | g0router governance is fully SQL-live (no in-memory accumulators); a 10s in-memory→DB sync worker is N/A by-design — the SAME architectural divergence as the ESC'd `sync.Map` in-memory store it would maintain. Restart-survival is automatic (every dimension reads persisted `request_log` live); calendar reset is inherent in the windowStart-bounded aggregate. MISSING→VAR/ESC (legitimate sound-parity; recorded §3/§7/open-questions). |
| PAR-BF-GOV-016 | Budget DB sync every 10s: `DumpBudgets` writes in-memory CurrentUsage to DB | **VAR/ESC (by-design)** | g0router has NO in-memory budget accumulator (budget is the LIVE `SumCostBy*` SQL aggregate, bf-gov-1 D8 — there is nothing to dump). Same VAR rationale as 023. MISSING→VAR/ESC. |
| PAR-BF-GOV-044 | `PerformStartupResets` checks ALL VKs (active+inactive) for expired RLs on startup | **VAR/ESC (by-design)** | No in-memory counter exists to reset on startup — every dimension reads `request_log` live, so a fresh process is already correct (no stale in-memory state to repair). Same VAR rationale as 023. MISSING→VAR/ESC. |
| PAR-BF-GOV-045 | `UsageTracker.Cleanup` flushes in-memory deltas to DB before shutdown | **VAR/ESC (by-design)** | No in-memory deltas to flush — usage is persisted by `SaveUsage` on each request. Same VAR rationale as 023. MISSING→VAR/ESC. |
| PAR-BF-GOV-018 (SQL-accrual variant ONLY) | RL atomic CAS increment via `BumpRateLimitUsage`, auto-resets expired, retries on CAS failure | **BUILD (SQL-accrual variant)** | The MAP (§246/§253) directs the **SQL-accrual variant**, NOT the CAS-on-in-memory bump. g0router "accrues" by the persisted `request_log` row written by `SaveUsage`; the limit check is the LIVE aggregate over those rows (D1/D3). Lost-increment safety is the DB write, not `CompareAndSwap`. MISSING→PARTIAL (SQL-accrual correctness HAVE; CAS-spin mechanism ESC). |
| PAR-BF-GOV-035 | `Decision` enum: Allow/VirtualKeyNotFound/VirtualKeyBlocked/RateLimited/BudgetExceeded/TokenLimited/RequestLimited/ModelBlocked/ProviderBlocked/MCPToolBlocked | **BUILD** | `governance.Decision` typed enum (the buildable subset; ModelBlocked/ProviderBlocked/MCPToolBlocked are produced by bf-gov-2/mcp tiers — declared but only the gov-3-reachable values are EMITTED, D8). MISSING→HAVE (enum + the emitted subset). |
| PAR-BF-GOV-036 | `EvaluationResult` carries Decision, Reason, VirtualKey, RateLimitInfo, BudgetInfo, UsageInfo | **BUILD** | `governance.EvaluationResult{Decision,Reason,Status}` returned by the NEW `Evaluate` method; mapped to `{data,error}` (D8, Go-Port note #6). MISSING→PARTIAL (Decision+Reason+Status HAVE; the embedded RateLimitInfo/BudgetInfo/UsageInfo sub-structs are the minimal g0router subset — full Bifrost sub-structs ESC-REF-ABSENT). |

**Honest scoping note:** every BUILD row is closed only for the **VK→Team slice**
(the same boundary bf-gov-1 set), and every dimension is **SQL-live** over
`request_log` — the same architecture bf-gov-1's budget tier already proved sound
(D8). The 10s sync worker + startup-reset + graceful-flush rows (016/023/044/045) are
**VAR/ESC by-design**: g0router has no in-memory governance accumulator to sync, so
the worker is the maintenance mechanism for exactly the `sync.Map` in-memory store
the program already ESCs — declining it is legitimate sound-parity (like 9router's
deploy-relay escalations), not a gap. The streaming `UsageUpdate` row (021) is
**VAR** (the existing `SaveUsage` IS the finalized write). The CAS-spin mechanism,
the `sync.Map` store, GORM `AfterFind` calendar propagation, the cluster
remote-baseline rate check, and global/user/per-user-scoped usage tiers are ESC (§3).
Where the matrix's full behavior exceeds the built slice, the row flips to **PARTIAL**
or **VAR** (not HAVE) and the residual is recorded in `open-questions.md` (§7). No row
is closed by inventing un-evidenced Bifrost internals (ESC-REF-ABSENT).

### 0.3 Preconditions already satisfied (evidence — read the live tree)

- **bf-gov-1 SHIPPED the calendar window + clock — REUSE, do NOT re-add**
  (`internal/governance/quota.go`): `windowStart(period)` returns the daily/weekly/
  monthly period start (`:192-206`); injectable `clock func() time.Time` (`:58`);
  `NewQuotaEngine(spend, clock)` (`:65`). bf-gov-3 calendar-aligned rate-limit reset
  uses THIS `windowStart` (D2). **Re-adding calendar logic is FORBIDDEN (§2).**
- **bf-gov-1 SHIPPED the 2-level hierarchy + the denial tuple**
  (`quota.go:81-97`): `Allow(vk,*model)` runs `checkBudget`→`checkRPM`→
  `checkTeamBudget`→`checkTeamRPM`, fail-closed, returning `(ok,status,reason)`.
  bf-gov-3 ADDS the token + request dimensions into this chain via `Evaluate`
  (D3/D8) and re-expresses `Allow` as a wrapper. **Allow's signature is PRESERVED.**
- **bf-gov-1 SHIPPED `SumCostByTeam`/`SumCostByAPIKey` + the SpendReader iface**
  (`quota.go:10-13`, `requestlog.go:249-284`): the LIVE budget aggregates. bf-gov-3
  adds the two sibling dimension aggregates `SumTokensByAPIKey` (D1) and
  `SumRequestsByAPIKey` (D3) to this interface — both additive, both SQL-live,
  identical in class to the shipped methods.
- **bf-gov-1 SHIPPED `rpmHits` + the per-minute window** (`quota.go:60-61,73-77,
  114-140`): the request-per-minute counter keyed by VK key / `team:<id>`. The NEW
  dual-dimension limits (per `resetDuration`, not per-minute) are SEPARATE SQL-live
  checks — the existing per-minute RPM is UNCHANGED (D3). (The shipped per-minute RPM
  is the one in-memory counter g0router keeps, and it is intentionally NOT persisted —
  it is a coarse burst guard, not a calendar-window accumulator.)
- **bf-gov-1 SHIPPED the hermetic harness** (`quota_test.go`): `fixedClock(t)`
  (`:45-47`) + the in-memory `fakeSpendReader` (`:9-43`, NO net/sleep/`time.Now`).
  bf-gov-3 EXTENDS this harness with token + request maps (D7) — NO `time.Ticker`, NO
  `time.Sleep`, NO worker (there is none).
- **request_log carries tokens + one row per request** (`migrate.go:114-115`):
  `prompt_tokens`/`completion_tokens INTEGER NOT NULL DEFAULT 0`; `SaveUsage`
  (`requestlog.go:36-124`) writes exactly ONE row per COMPLETED request with final
  usage. The token dimension sums the token columns (D1); the request dimension
  COUNTs the rows (D3) — NO new write path, NO new column, NO new table. `SaveUsage`
  being the one-row-per-completed-request write is the live evidence for the 021/022
  streaming-finalization VAR (§3).
- **The Allow consumer chain** (`internal/api/vk.go:AllowVK` →
  `routes_openai.go:vkQuotaAdapter.Allow:223-234` → `engine.Allow`): the gate calls
  `g.quota.Allow(vk,model)` (`vk.go:76-82`) and surfaces `(status,reason)` into the
  HTTP response. The Decision enum surfaces LIVE here: `vkQuotaAdapter.Allow` calls
  the NEW `engine.Evaluate(...)` and maps `EvaluationResult` → the existing tuple
  (D8) — the enum is NOT dead because its `Reason`/`Status`/`error.code` flow to the
  real `{data,error}` response (proof §5).
- **Server lifecycle is UNTOUCHED** (`server.go`): bf-gov-3 adds NO worker, so the
  `Server` wrapper, `New`/`NewWithShutdown`, and `Close()` are all UNCHANGED. (The
  w6-j `SetShutdownFunc` precedent is noted only to confirm that even IF a worker were
  wanted later it would be additive — but bf-gov-3 needs none.)

---

## 1. Decisions made (and why) — binding

### D1 — Both new dimensions are SQL-live aggregates over request_log (NOT in-memory CAS counters)

The matrix's `TableRateLimit` (matrix:132-142) tracks `TokenCurrentUsage`/
`RequestCurrentUsage` via in-memory CAS bumps (matrix:193, quirk #7). g0router has NO
in-memory CAS store (ESC, §3); bf-gov-1 already proved the sound g0router pattern is
the LIVE SQL aggregate over `request_log` (D8). Both new dimensions follow it:

**Decision — two additive Store methods, both mirroring `SumCostByAPIKey`
(`requestlog.go:249-262`, incl. the `sql.Null*` → 0 handling):**
```sql
-- token dimension (SumTokensByAPIKey → int64)
SELECT SUM(prompt_tokens + completion_tokens) FROM request_log
WHERE api_key = ? AND timestamp >= ?
-- request dimension (SumRequestsByAPIKey → int64)
SELECT COUNT(*) FROM request_log
WHERE api_key = ? AND timestamp >= ?
```
Both are added to the `SpendReader` interface (it already carries `SumCostByAPIKey`/
`SumCostByTeam`/`SumCostBy*`; two more methods are additive — no `NewQuotaEngine`/
`Allow` signature change; the only production implementer `*store.Store` gains both;
the test `fakeSpendReader` gains an in-memory token map + count map). The token-LIMIT
check denies when `SumTokensByAPIKey(vk.Key, windowStart(TokenResetPeriod))` exceeds
`TokenMax` (guard `TokenMax>0`); the request-LIMIT check denies when
`SumRequestsByAPIKey(vk.Key, windowStart(RequestResetPeriod))` ≥ `RequestMax` (guard
`RequestMax>0`). Both are **real, live enforcement paths** that deny on actual
persisted usage — identical in class to the shipped budget tier, with NO in-memory
counter, NO `lastReset`, NO `CurrentUsage` field, NO CAS, NO accrual method, and
therefore NO sync worker and NO new table (the whole apparatus the previous draft
needed collapses). Restart-survival is automatic (the COUNT/SUM reads persisted rows
that outlive the process); lazy calendar reset is inherent (D2).

**Optimization (note, do NOT require):** the three VK-scoped aggregates (cost SUM,
token SUM, request COUNT) MAY be combined into ONE query —
`SELECT SUM(cost), SUM(prompt_tokens+completion_tokens), COUNT(*) FROM request_log
WHERE api_key=? AND timestamp>=?` — fewer queries than even the shipped budget+team
path. The plan does not require it; the separate `SumTokensByAPIKey`/
`SumRequestsByAPIKey` methods keep the `SpendReader` interface clean and testable.

### D2 — Calendar-aligned reset REUSES the SHIPPED `windowStart` — do NOT duplicate

bf-gov-1 already built `windowStart(period)` (daily/weekly/monthly, `quota.go:192-
206`) and the injectable `clock`. The matrix wants rate-limit calendar alignment
(015/019) via `GetCalendarPeriodStart`.

**Decision:** the rate-limit dimensions' window start IS `windowStart(resetPeriod)`.
Accepted period tokens are the SHIPPED set `{"daily","weekly","monthly"}` PLUS the
rolling-duration tokens the matrix mentions (`"1h"`, `"1d"`, `"1M"` — matrix:122).
`windowStart` is EXTENDED additively to recognize the duration tokens (parse → a
rolling `now.Add(-d)` lower bound) while the existing calendar tokens are UNCHANGED
(behavior-preserving for bf-gov-1's budget callers — the existing 3 cases keep their
exact code path; only the `default` branch gains duration parsing). **No new
calendar function; no second clock.** If a period token is neither a known calendar
word nor a parseable duration, `ValidateRateLimit` (D5) rejects it at config time so
it never reaches `windowStart`.

### D3 — Dual-dimension rate limit layered ADDITIVELY over the shipped per-minute RPM

The shipped `checkRPM` is a per-MINUTE counter (`quota.go:114-140`). The matrix's
dual-dimension limiter is per-`resetDuration` (token + request), with lazy reset.

**Decision (precedence, deterministic, documented) — `Evaluate` runs, fail-closed at
the first failing level, in this order (EXTENDS the bf-gov-1 D3 order):**
1. VK budget (shipped `checkBudget`) → `BudgetExceeded`.
2. VK per-minute RPM (shipped `checkRPM`) → `RateLimited`.
3. **VK request-limit** (NEW dual-dimension, per `RequestResetPeriod`) →
   `RequestLimited`.
4. **VK token-limit** (NEW dual-dimension, per `TokenResetPeriod`, live
   `SumTokensByAPIKey`) → `TokenLimited`.
5. Team budget (shipped `checkTeamBudget`) → `BudgetExceeded`.
6. Team per-minute RPM (shipped `checkTeamRPM`) → `RateLimited`.

Both new dimensions (steps 3 + 4) are SQL-live (D1): the request-limit denies when
`SumRequestsByAPIKey(vk.Key, windowStart(RequestResetPeriod))` ≥ `RequestMax`; the
token-limit denies when `SumTokensByAPIKey(vk.Key, windowStart(TokenResetPeriod))` >
`TokenMax`. There is NO in-memory `usage`/`lastReset` counter for either — lazy reset
is inherent in the `windowStart`-bounded WHERE clause (a rolled-over window
aggregates only current-period rows; matrix quirk #8, PAR-BF-GOV-039), so neither
dimension needs a worker, a reset, or a persisted snapshot. The shipped per-minute
RPM (steps 2/6) is UNCHANGED — the dual-dimension limits are a SEPARATE, configurable
SQL-live layer (a VK may set per-minute RPM, or token/request limits, or both). All
four new denial reasons keep status **429** (the shipped convention, `quota.go:83-94`).

### D4 — All dimensions SQL-live; NO in-memory counter, NO `UsageUpdate`/`Accrue`, NO sync worker (VAR rationale for 016/021/022/023/044/045)

The previous draft introduced an in-memory request counter, an `UsageUpdate`+`Accrue`
type, a `vk_rate_limit_state` table + store, a `RateLimitSyncWorker`, and a
server-lifecycle wiring — an apparatus that exists ONLY to persist in-memory
governance state across restarts and reset windows. g0router has no such state by
design (the `sync.Map` in-memory governance store is already ESC), so that apparatus
is the maintenance mechanism for exactly the re-architecture the program defers.

**Decision (the simplification):** because budget (bf-gov-1 D8), tokens (D1), and
requests (D3) are ALL SQL-live aggregates over `request_log`, there is nothing in
memory to accrue, reset, dump, or rehydrate. Therefore bf-gov-3 builds NONE of:
- a `UsageUpdate` struct or an `Accrue` method (021/022) — **VAR**: g0router's
  `SaveUsage` (`requestlog.go:36-124`) is ALREADY the streaming-aware finalized write
  — it writes exactly ONE `request_log` row per COMPLETED request with the final
  token usage, which IS quirk #6's finalization rule (tokens counted once with usage
  data; one request counted on completion, never per-chunk). bf-gov-3 satisfies 021/
  022 by VERIFYING this: add a test asserting a streamed request produces exactly one
  `request_log` row with final usage (cite existing coverage if it already exists in
  `internal/api/usage_glue_test.go`/`requestlog`-level tests; e.g.
  `usage_glue_test.go:747-751` already asserts finalized token rows). Do NOT build an
  `Accrue` method with no live consumer.
- a `RateLimitStateStore`/`vk_rate_limit_state` table or a `RateLimitSyncWorker`
  (016/023/044/045) — **VAR/ESC by-design**: g0router governance has no in-memory
  accumulator, so a 10s in-memory→DB sync worker + startup-reset + graceful-flush are
  N/A. Restart-survival is STRICTLY BETTER under SQL-live (the COUNT/SUM reads
  persisted rows that always survive a restart — the worker's rehydrate/flush would
  be an inferior reimplementation of "just read the rows"); calendar reset is inherent
  in the `windowStart`-bounded aggregate (no counter to reset). This is legitimate
  sound-parity (the same divergence class as the ESC'd `sync.Map` store and 9router's
  deploy-relay escalations), recorded in §3/§7/open-questions — NOT a gap, NOT inert
  wiring.

**Net effect:** ~3 files deleted from the plan (`ratelimitstate.go`,
`ratelimitstate_test.go`, `worker_test.go`), no `vk_rate_limit_state` table, no
`server.go` change, and ALL worker-inertness / no-leftovers risk removed — every
surface bf-gov-3 builds is trivially non-inert because it is a live SQL read consumed
by `Evaluate`.

### D5 — Rate-limit validation is an inline pure function (NOT a GORM BeforeSave hook)

PAR-BF-GOV-038's mechanism is GORM `BeforeSave` (ESC — no GORM, §3). The behavior is
buildable inline (mirrors bf-gov-1 D2 `ValidateBudgetOwner`).

**Decision:** `governance.ValidateRateLimit(rl RateLimitConfig) error` — a pure
value-in/error-out fn (errors-as-values, `fmt.Errorf("rate_limit: %w")`) that errors
when: `TokenMax > 0 && TokenResetDuration == ""` (matrix quirk #5); `RequestMax > 0
&& RequestResetDuration == ""`; or a non-empty `*ResetDuration` is neither a known
calendar word (`daily`/`weekly`/`monthly`) nor a parseable rolling duration
(`1h`/`1d`/`1M` form). It is called inline from the VK admin create/update path that
assigns rate-limit config (mirror bf-gov-1 D2's wiring into
`internal/admin/virtualkeys.go`). **No-leftovers:** T-validaterl must prove ≥1 live
production caller (grep §5) or fold it into the VK-update path; else STOP+escalate.

### D6 — (removed) DB-sync worker — see D4 (VAR/ESC by-design, no worker built)

The DB-sync worker decision of the prior draft is REMOVED. Rationale folded into D4:
g0router governance is fully SQL-live, so there is no in-memory state to sync,
rehydrate, reset, or flush — the worker would maintain exactly the `sync.Map`
in-memory store the program ESCs. Rows 016/023/044/045 are VAR/ESC by-design (§3/§7).
No `RateLimitSyncWorker`, no `RateLimitStateStore`, no `vk_rate_limit_state` table, no
`server.go` lifecycle change.

### D7 — Hermetic tests, injected clock (reuse + extend the shipped harness)

ALL bf-gov-3 tests use `fixedClock` + the extended `fakeSpendReader` (now with a
token map + a request-count map) + temp/in-memory SQLite via the shipped `store.Open`
test pattern for the store-layer aggregate tests. **NO `time.Now`, NO `time.Sleep`,
NO `time.Ticker`, NO network, NO subprocess** — binding (Wave-7 hermetic lesson,
BIFROST-MAP §494). Window rollover is asserted by constructing a new `fixedClock(t2)`
engine or advancing a test-owned `now` var (the shipped `TestVKRateLimitRPM` pattern,
`quota_test.go:86-119`); SQL-live rollover is asserted by seeding `request_log` rows
with timestamps inside/outside `windowStart(period)` and checking the aggregate.

### D8 — `Decision` enum + `EvaluationResult` reconcile with the shipped Allow tuple WITHOUT a signature change

bf-gov-1 DEFERRED the enum to bf-gov-3 and kept `Allow(bool,int,string)`
(`quota.go:81`, open-questions "Decision/EvaluationResult enum … deferred to
bf-gov-3"). The enum must surface LIVE without breaking the shipped `Allow`.

**Decision (additive, signature-preserving):**
```go
type Decision int
const (
    DecisionAllow Decision = iota
    DecisionVirtualKeyNotFound
    DecisionVirtualKeyBlocked
    DecisionRateLimited
    DecisionBudgetExceeded
    DecisionTokenLimited
    DecisionRequestLimited
    DecisionModelBlocked     // emitted by bf-gov-2 list tier (declared, not emitted here)
    DecisionProviderBlocked  // declared, not emitted here
    DecisionMCPToolBlocked   // declared, not emitted here
)
type EvaluationResult struct {
    Decision Decision
    Reason   string // human-readable, snake_case-friendly
    Status   int    // HTTP status (429 for the gov-3 denials; 0 on allow)
}
func (e *QuotaEngine) Evaluate(vk *VirtualKeyInfo, model string) EvaluationResult
```
**Reconciliation:** the SHIPPED `Allow` is RE-EXPRESSED as a thin wrapper —
```go
func (e *QuotaEngine) Allow(vk *VirtualKeyInfo, model string) (ok bool, status int, reason string) {
    r := e.Evaluate(vk, model)
    return r.Decision == DecisionAllow, r.Status, r.Reason
}
```
so EVERY existing caller (`vkQuotaAdapter.Allow`, the shipped quota tests) keeps the
exact `(bool,int,string)` contract; the enum is purely additive. **Live surface
(no-leftovers):** `vkQuotaAdapter.Allow` (`routes_openai.go:223`) is updated to call
`engine.Evaluate` and map the `Reason`/`Status` into the gate's response — the enum's
`Reason`/`Status` flow into the real `{data,error}` HTTP path the gate already writes
(`vk.go:76-82`). Mapping `EvaluationResult` → `{data,error}` follows Go-Port note #6
(matrix:214): on denial the gate writes the existing error envelope with the
`Reason` string and `Status`; the `Decision` value is also stamped into the response
`error.code` (snake_case, e.g. `"token_limited"`) so the enum is observably surfaced,
NOT dead. T-decision proves a `TokenLimited`/`RequestLimited` denial reaches the gate
response with the matching `error.code` (proof §5) or STOP+escalate.

The full Bifrost `EvaluationResult` sub-structs (RateLimitInfo/BudgetInfo/UsageInfo
with per-dimension remaining/reset metadata) are NOT fully ported — their exact
shape is ESC-REF-ABSENT. bf-gov-3 carries the minimal `{Decision,Reason,Status}`;
the richer info sub-structs are recorded as a follow-up in open-questions (§7).

---

## 2. Target files

### IN-SCOPE — EXTEND (additive only)

| File | Change (additive ONLY) |
|---|---|
| `internal/store/requestlog.go` | ADD `SumTokensByAPIKey(key, sinceISO string) (int64, error)` — `SELECT SUM(prompt_tokens+completion_tokens) … WHERE api_key=? AND timestamp>=?` — and `SumRequestsByAPIKey(key, sinceISO string) (int64, error)` — `SELECT COUNT(*) … WHERE api_key=? AND timestamp>=?` — both `sql.Null*`→0, mirroring `SumCostByAPIKey` (`:249-262`). NOTHING else in this file. (Impl MAY fold the three VK-scoped aggregates into one query as an optimization, D1 — not required.) |
| `internal/store/requestlog_test.go` (EXTEND) | RED first: seed rows → `SumTokensByAPIKey(key,since)` sums prompt+completion only for that key+window; `SumRequestsByAPIKey(key,since)` counts only that key's rows in-window; 0 for unknown key / pre-window. Window-rollover assertion via inside/outside `windowStart` timestamps. |
| `internal/governance/quota.go` | **INTERNAL SERIAL slot (final holder).** ADD: `SumTokensByAPIKey` + `SumRequestsByAPIKey` to the `SpendReader` iface (`:10-13`); `Decision` enum + `EvaluationResult` (D8); `Evaluate` method + re-express `Allow` as its wrapper (D8, signature PRESERVED); dual-dimension fields on `VirtualKeyInfo` (`TokenMax int64,TokenResetPeriod string,RequestMax int64,RequestResetPeriod string` — additive struct fields) + `checkTokenLimit`/`checkRequestLimit` (both SQL-live, D1/D3, NO in-memory counter); `ValidateRateLimit` + `RateLimitConfig` (D5); EXTEND `windowStart` `default` branch to parse rolling durations (D2, existing 3 cases UNCHANGED). PRESERVE `NewQuotaEngine`/`Allow` signatures. **NO `UsageUpdate`/`Accrue`, NO worker, NO state store iface (D4/D6).** |
| `internal/governance/quota_test.go` (EXTEND) | RED first (D7 hermetic): dual-dimension token/request deny (`TokenLimited`/`RequestLimited`) via the fake's token/count maps; lazy reset on window rollover; `Evaluate`/`Allow`-wrapper equivalence over the shipped cases; `ValidateRateLimit` cases; `windowStart` duration parsing. ADD `SumTokensByAPIKey` + `SumRequestsByAPIKey` to `fakeSpendReader` (in-memory token + count maps). |
| `internal/admin/virtualkeys.go` | EXTEND the VK create/update validation path to call `governance.ValidateRateLimit(...)` on the assigned rate-limit config (mirror the bf-gov-1 D2 `ValidateBudgetOwner` wiring point) — LIVE consumer for D5. PRESERVE handler signatures; additive validation only. |
| `internal/admin/virtualkeys_test.go` (EXTEND) | RED first: a VK create with `TokenMax>0` and empty reset duration → 4xx via the validation path. |
| `internal/schemas/governance.go` | ADD dual-dimension rate-limit fields to the VK config schema (additive `json:"…,omitempty"`: token/request max + reset durations) so the store can persist + the resolver can thread them. Additive fields only. |
| `internal/store/virtualkeys.go` | Thread the new dual-dimension config fields through the `config_json` blob (they ride the existing blob, NOT new columns — they are not queried by key). PRESERVE all signatures. |
| `internal/api/vk.go` | ADD the dual-dimension fields to `api.VKInfo` (additive); they pass through to the quota checker. PRESERVE `NewVKGate`/`AllowVK` signatures. (The Decision `error.code` surfacing is in the gate's denial-write path — additive.) |
| `internal/server/routes_openai.go` | EXTEND `storeVKToAPI` to populate the dual-dimension fields; EXTEND `vkQuotaAdapter.Allow` to call `engine.Evaluate` and map `EvaluationResult` (Decision `error.code` + Reason + Status) into the gate result (D8). **NO new route registration.** Adapter-body edits only. |
| `internal/api/usage_glue_test.go` (CITE / extend if needed) | VERIFY the 021/022 streaming-finalization VAR: one `request_log` row per completed request with final usage (cite `usage_glue_test.go:747-751` if it already asserts this; else add a focused assertion). NO `Accrue` built. |

### FORBIDDEN (automatic REJECT if touched)

- **Re-adding calendar/`windowStart` logic** that bf-gov-1 already shipped — REUSE
  `quota.go:192-206`; only the `default` branch may gain duration parsing (D2).
- **Changing `NewQuotaEngine`/`Allow`/`NewVKGate`/`AllowVK` signatures** — all
  extensions are additive struct fields, new methods, or new sibling types.
- **Any in-memory governance counter / accrual / sync worker / state table** —
  `UsageUpdate`/`Accrue`, `RateLimitSyncWorker`, `RateLimitStateStore`,
  `vk_rate_limit_state` are all REMOVED from scope (D4/D6 VAR/ESC). All dimensions
  are SQL-live; building any of these is a REJECT.
- `internal/server/server.go` — UNTOUCHED. NO worker wiring, NO lifecycle change, NO
  `New`/`NewWithShutdown` edit (bf-gov-3 needs no worker → no server change).
- `internal/server/routes_admin.go` — bf-gov-3 registers NO admin routes.
- Any **new route registration** in `routes_openai.go` (only adapter bodies change).
- `internal/schemas/lists.go` / WhiteList / BlackList / BlacklistedModels — that is
  **bf-gov-2** (the `ModelBlocked` Decision value is DECLARED but NOT emitted here).
- Any Customer schema/column/table; `sync.Map` in-memory governance store; CAS-spin
  (`CompareAndSwap`) bumps; GORM `BeforeSave`/`AfterFind`; cluster remote-baseline
  rate check; `GovernanceStore` 40-method interface — all **ESC** (§3).
- Any UI file (`ui/**`) — bf-gov-3 is Go-only; the additive VK config fields surface
  to no asserted UI behavior (the VK admin page may ignore them).
- The display-only `teams.budget_used_usd` accumulator — NOT enforced against
  (bf-gov-1 D8 holds; budgets stay SQL-live).

---

## 3. Scope / Non-goals — explicit ESC list

**bf-gov-3 builds ONLY the VK→Team slice's dual-dimension RL (SQL-live) + inherent
lazy reset + Decision enum + inline validation.** The following are **ESC/VAR**
(recorded in `open-questions.md` at close):

| ESC/VAR item | Matrix row(s) | Why ESC/VAR |
|---|---|---|
| **10s in-memory→DB sync worker + startup-reset + graceful-flush** (`UsageTracker`, `DumpBudgets`, `PerformStartupResets`, `Cleanup`) | 016, 023, 044, 045 | **VAR/ESC by-design.** g0router governance is fully SQL-live (no in-memory accumulators); a 10s in-memory→DB sync worker + startup-reset + graceful-flush are N/A — the SAME architectural divergence as the ESC'd `sync.Map` in-memory store the worker would maintain. Restart-survival is automatic (every dimension reads persisted `request_log` live — strictly better than rehydrate/flush); calendar reset is inherent in the windowStart-bounded aggregate (no counter to reset). Legitimate sound-parity (like 9router's deploy-relay escalations), NOT a gap, NOT inert wiring. |
| **`UsageUpdate` struct + `Accrue` method** (streaming-aware in-memory accrual) | 021, 022 (non-VK tiers ESC) | **VAR (verify, don't build).** g0router's `SaveUsage` (`requestlog.go:36-124`) is ALREADY the streaming-aware finalized write — one `request_log` row per COMPLETED request with final usage = quirk #6's rule (tokens once with usage data; one request on completion). The SQL-live aggregates honor it inherently; building an `Accrue` type with no live consumer would be dead code. bf-gov-3 VERIFIES the existing write (cite/add a test), 022's VK→Team slice is PARTIAL via the persisted rows; global/user/scoped tiers ESC. |
| **CAS-spin atomic bumps** (`CompareAndSwap` retry loop on an in-memory counter) | 018 (CAS variant), 013, 041 | Presupposes the `sync.Map` atomic store (004/049). g0router "accrues" by the persisted `request_log` row and checks the LIVE SQL aggregate; the MAP (§246/§253) explicitly directs the **SQL-accrual variant** of 018, NOT the CAS bump. |
| **`sync.Map` lock-free in-memory governance store** + 40-method `GovernanceStore` iface | 004, 049, 050 | g0router uses `database/sql` reads (BIFROST-MAP §252, Go-Port note #2). The behaviors are covered by the SQL-live aggregates + the single worker-synced request counter. |
| **GORM `AfterFind` calendar-alignment owner-propagation** (VK/Team/Customer) | 005, 046, 047 | g0router has no GORM hooks; calendar alignment is stamped directly via `windowStart` (D2). The propagation MECHANISM is GORM-specific — n/a. |
| **Customer tier** (schema/budgets/rate-limits/AfterFind) | 008, 019(customer), 046, 047 | phase-18 design is VK+Team only (bf-gov-1 §3); a third tier is scope expansion. |
| **global / user-level / per-user-scoped / per-VK-scoped usage tiers** of `UpdateUsage` | 022 (non-VK tiers), 043 | bf-gov-3 builds the VK→Team accrual slice only (same boundary as bf-gov-1). |
| **Cluster remote-baseline rate check** (`CheckRateLimit` local+remote for multi-node) | 040 | Single-binary SQLite gateway; clustering is a product-category change (BIFROST-MAP §3). |
| **Ghost-node reconciliation / log-stamped governance IDs** | 024, 025 | Cluster-coupled; no model catalog / cluster in g0router. |
| **Full `EvaluationResult` sub-structs** (RateLimitInfo/BudgetInfo/UsageInfo per-dimension metadata) | 036 (sub-structs) | Exact shape is ESC-REF-ABSENT; bf-gov-3 carries `{Decision,Reason,Status}` only. |
| **`ModelBlocked`/`ProviderBlocked`/`MCPToolBlocked` Decision EMISSION** | 035 (those values) | Produced by bf-gov-2 (lists) / bf-mcp tiers; bf-gov-3 DECLARES the enum values but does not emit them (no-leftovers: a declared-but-unemitted enum constant is acceptable as a shared type the sibling plans fill — documented). |

**No-leftovers (binding, §3 CLI_ORCHESTRATOR):** because every dimension is a LIVE
SQL read consumed by `Evaluate`, the no-leftovers risk that drove the prior draft's
STOP-conditions is structurally removed — there is no in-memory counter, no `Accrue`,
no worker, and no state table to be inert. The two remaining STOP-conditions are: D5
(`ValidateRateLimit` must have a live production caller — the VK admin path — or be
folded into it; else STOP), and D8 (the Decision enum's `Reason`/`Status`/`error.code`
must reach the real `{data,error}` response; else STOP). The dual-dimension fields
(consumed by `checkToken`/`checkRequest` in `Evaluate`) and the two SQL aggregates
(consumed by those checks) are trivially live. If any surface cannot be made live
additively, it is marked ESC rather than shipped inert.

---

## 4. Task graph (TDD; `N. [step] -> verify: [check]`)

Cadence (AGENTS.md "TDD always"): no impl file/field lands before its covering
`_test.go` is committed RED. `go test ./... && go vet ./... && go build ./...` green
at EVERY commit. Internal serial: bf-gov-3 holds quota.go (final holder).

1. **[token + request aggregates, RED→GREEN]** Extend `requestlog_test.go`: seed rows
   → `SumTokensByAPIKey(key,since)` sums prompt+completion for that key+window;
   `SumRequestsByAPIKey(key,since)` counts only that key's in-window rows; 0
   otherwise; rollover via inside/outside `windowStart` timestamps. Add both methods
   to `requestlog.go` + both to the `SpendReader` iface + the `fakeSpendReader`
   token+count maps. -> verify: `go test ./internal/store/ -run 'SumTokens|SumRequests'`
   then `go test ./internal/governance/...` build green; `go vet ./... && go build
   ./...` exit 0. Commit RED then GREEN: `phase-1/bf-gov-3: failing token+request
   aggregate tests (TDD red)` / `phase-1/bf-gov-3: SumTokensByAPIKey + SumRequestsByAPIKey
   (additive, SQL-live)`.

2. **[Decision enum + Evaluate + Allow-wrapper, RED→GREEN]** Extend `quota_test.go`:
   `Evaluate` returns `DecisionAllow` on pass; `Allow` wrapper returns the identical
   `(bool,int,string)` for every shipped case (equivalence test over the existing
   budget/RPM/team cases). Add `Decision`/`EvaluationResult`/`Evaluate`; re-express
   `Allow` as the wrapper (D8). -> verify: `go test ./internal/governance/ -run
   'Evaluate|Allow'` green; ALL pre-existing quota tests still pass (wrapper
   equivalence); `go vet ./... && go build ./...` exit 0. Commit RED then GREEN:
   `phase-1/bf-gov-3: Decision enum + Evaluate sibling; Allow as wrapper`.

3. **[dual-dimension RL (SQL-live) + inherent lazy reset, RED→GREEN]** Extend
   `quota_test.go` (D7): VK request-limit deny → `RequestLimited` 429 (fake count map
   ≥ RequestMax); VK token-limit deny → `TokenLimited` 429 (fake token map > TokenMax);
   window rollover re-allows (advance `fixedClock` so the windowStart lower bound
   excludes prior rows — inherent reset); precedence (budget → RPM → request → token →
   team) holds. Add the dual-dimension fields on `VirtualKeyInfo`, the SQL-live
   `checkTokenLimit`/`checkRequestLimit` (NO in-memory counter), EXTEND `windowStart`
   default-branch duration parsing (existing 3 cases UNCHANGED), wire into `Evaluate`
   (D1/D3). -> verify: `go test ./internal/governance/ -run 'Token|Request|Window|Precedence'`
   green; `go vet ./... && go build ./...` exit 0. Commit: `phase-1/bf-gov-3:
   dual-dimension token+request rate limit (SQL-live, inherent calendar reset)`.

4. **[streaming finalization VERIFY (VAR, no build), GREEN]** Confirm `SaveUsage`
   writes one `request_log` row per completed request with final usage; cite or add a
   focused test (`usage_glue_test.go:747-751` or a `requestlog`-level assertion). NO
   `UsageUpdate`/`Accrue` is built (D4). -> verify: `go test ./internal/api/ -run
   Usage` (or the cited test) green; `! grep -rn 'type UsageUpdate\|func.*Accrue'
   internal/governance/` empty (no accrual built). Commit (folds into close or its
   own): `phase-1/bf-gov-3: verify SaveUsage one-row-per-completed-request finalization (021/022 VAR)`.

5. **[rate-limit validation, RED→GREEN]** Extend `quota_test.go` + `admin/
   virtualkeys_test.go`: `ValidateRateLimit` errors on `Max>0 && reset==""` and on
   an unparseable reset duration; passes otherwise; a VK create with `TokenMax>0` +
   empty reset → 4xx. Add `ValidateRateLimit` + `RateLimitConfig` (D5) + the live
   call in `admin/virtualkeys.go`. -> verify: `go test ./internal/governance/ -run
   ValidateRateLimit && go test ./internal/admin/ -run VirtualKey` green; ≥1 live
   caller (grep §5); `go vet ./... && go build ./...` exit 0. Commit: `phase-1/
   bf-gov-3: inline rate-limit reset-duration validation`.

6. **[schema + VK store threading, GREEN]** Add the dual-dimension config fields to
   `schemas/governance.go` (additive) + thread through the `virtualkeys.go`
   `config_json` blob (additive). -> verify: `go test ./internal/store/ -run
   VirtualKey` green (blob round-trips the new fields); `go vet ./... && go build
   ./...` exit 0. Commit: `phase-1/bf-gov-3: dual-dimension RL config on VK schema+store`.

7. **[adapter + Decision surfacing, RED→GREEN]** Extend `storeVKToAPI` + `api.VKInfo`
   to thread the dual-dimension fields; extend `vkQuotaAdapter.Allow` to call
   `engine.Evaluate` and surface `Decision` `error.code` + Reason + Status in the gate
   result (D8). Add a server/gate test proving a `TokenLimited`/`RequestLimited`
   denial reaches the response with the matching `error.code`. -> verify: `go test
   ./internal/server/... ./internal/api/... && go test ./... && go vet ./... && go
   build ./...` exit 0; NO new route (grep §5); NO `New*` signature change AND
   `server.go` UNCHANGED (grep §5). Commit: `phase-1/bf-gov-3: surface Decision enum
   through the VK gate (error.code + reason + status)`.

8. **[close]** Full validation (§6); flip matrix rows (§7); append `open-questions.md`
   (ESC/VAR §3 + D-deferred); update `docs/WORKFLOW.md`; RELEASE the quota.go internal
   serial (chain END — releases to nobody). -> verify: §6 all green; matrix + WORKFLOW
   + open-questions committed. Commit: `phase-1/bf-gov-3: close — dual-dim RL (SQL-live)
   + reset + Decision enum; matrix flip; serial end`.

---

## 5. Acceptance criteria (binary; file:line / grep where possible)

**Test gates** (each yes/no, exit 0):
- `go test ./... && go vet ./... && go build ./...` → exit 0.
- `go test ./internal/governance/ -run 'Evaluate|Allow|Token|Request|Window|ValidateRateLimit' -v` → all pass.
- `go test ./internal/store/ -run 'SumTokens|SumRequests|VirtualKey' -v` → all pass.
- `go test ./internal/admin/ -run VirtualKey -v` → rate-limit validation case passes.
- `go test ./internal/server/ -v` / `./internal/api/ -v` → Decision-surface (error.code) + streaming-finalization VAR pass.

**TDD-order proof** (each impl's covering test in an earlier-or-equal commit):
```bash
for pair in \
  "internal/store/requestlog_test.go:internal/store/requestlog.go" \
  "internal/governance/quota_test.go:internal/governance/quota.go" \
  "internal/admin/virtualkeys_test.go:internal/admin/virtualkeys.go" ; do
  tf=${pair%%:*}; im=${pair##*:}
  ct=$(git log --format=%ct -1 -- "$tf"); cf=$(git log --format=%ct -1 -- "$im")
  [ "$ct" -le "$cf" ] || echo "TDD VIOLATION: $im"   # prints nothing
done
```

**Grep proofs:**
```bash
# REUSE shipped windowStart/clock — NOT re-added (only default-branch extended)
grep -n "func (e \*QuotaEngine) windowStart" internal/governance/quota.go              # exactly ONE definition
! grep -cE 'case "daily"|case "weekly"|case "monthly"' internal/governance/quota.go | grep -vq '^3$' && echo "shipped 3 calendar cases intact"
# signatures PRESERVED + server.go UNTOUCHED
grep -n "func NewQuotaEngine(spend SpendReader, clock func() time.Time)" internal/governance/quota.go
grep -n "func (e \*QuotaEngine) Allow(vk \*VirtualKeyInfo, model string) (ok bool, status int, reason string)" internal/governance/quota.go
git diff <base>..HEAD -- internal/server/server.go | wc -l                              # = 0 (server.go untouched, no worker)
# Decision enum LIVE-surfaced (no-leftovers D8)
grep -n "DecisionTokenLimited\|DecisionRequestLimited\|EvaluationResult\|func (e \*QuotaEngine) Evaluate" internal/governance/quota.go
grep -n "Evaluate(" internal/server/routes_openai.go                                   # adapter calls Evaluate
grep -n "error.code\|token_limited\|request_limited" internal/server/routes_openai.go internal/api/vk.go  # Decision surfaced in response
# BOTH dimensions are LIVE SQL aggregates (not in-memory CAS, not a worker-synced counter)
grep -n "func (s \*Store) SumTokensByAPIKey\|func (s \*Store) SumRequestsByAPIKey" internal/store/requestlog.go
grep -n "SumTokensByAPIKey\|SumRequestsByAPIKey" internal/governance/quota.go          # in iface + checkToken/checkRequest
! grep -n "CompareAndSwap\|sync.Map" internal/governance/quota.go && echo "no CAS/sync.Map OK"
# NO worker, NO accrual, NO state table (the apparatus is gone — D4/D6 VAR/ESC)
! grep -rn "UsageUpdate\|func.*Accrue\|RateLimitSyncWorker\|RateLimitStateStore\|vk_rate_limit_state" internal/ --include='*.go' && echo "no worker/accrue/state-table OK"
! test -e internal/store/ratelimitstate.go && ! test -e internal/governance/worker_test.go && echo "no worker files OK"
! grep -n "time.NewTicker\|time.Ticker" internal/governance/*.go internal/server/server.go && echo "no ticker OK"
# validation present + live caller (no-leftovers D5)
grep -n "func ValidateRateLimit" internal/governance/quota.go
grep -rn "ValidateRateLimit(" internal/ --include='*.go' | grep -v _test.go            # ≥1 production caller
# streaming finalization is VERIFIED (VAR), not built
grep -rn "request_log row\|one row per\|final usage" internal/api/usage_glue_test.go internal/store/requestlog_test.go  # the 021/022 VAR test
# NO new route, NO admin route, NO new table, NO forbidden mechanisms
! grep -nE 'r\.(GET|POST|PUT|DELETE)\(' internal/server/routes_openai.go | grep -iE 'team|virtual|rate' && echo "no new route OK"
test ! -e internal/store/customers.go && echo "no Customer tier OK"
! grep -rn "BeforeSave\|AfterFind\|GovernanceStore" internal/governance/ && echo "no GORM/in-mem-store OK"
! grep -rn "func init(" internal/governance/quota.go && echo "no init() OK"
# budgets stay SQL-live (bf-gov-1 D8 invariant holds)
! grep -n "budget_used_usd" internal/governance/quota.go && echo "budget tier still SQL-live OK"
# NO migrate.go change at all (no new table — fewer files than the prior draft)
git diff <base>..HEAD -- internal/store/migrate.go | wc -l                             # = 0 (no vk_rate_limit_state)
# hermetic governance tests
! grep -nE 'time\.Now|time\.Sleep|time\.Ticker|http\.Get|net\.Dial' internal/governance/quota_test.go && echo "hermetic OK"
```

**Behavioral acceptance (binary):**
- A VK whose token-limit dimension's live `SumTokensByAPIKey` over the window exceeds
  `TokenMax` is DENIED 429 with `Decision=DecisionTokenLimited`, even when budget +
  RPM + request-limit all pass.
- A VK whose request-limit dimension's live `SumRequestsByAPIKey` (COUNT) over the
  window reaches `RequestMax` is DENIED 429 `DecisionRequestLimited`; after window
  rollover (advanced `fixedClock`, so `windowStart` excludes prior rows) it is
  re-allowed — the reset is INHERENT in the bounded aggregate, with no counter and no
  worker.
- `Evaluate` returns `DecisionAllow` on pass and the shipped `Allow` wrapper returns
  the IDENTICAL `(bool,int,string)` for every pre-existing test case.
- Streaming finalization (021/022) holds via the EXISTING write path: a completed
  (streamed or not) request produces exactly one `request_log` row with final usage —
  verified by a cited/added test, NOT by a built `Accrue` method.
- `ValidateRateLimit` errors when a positive max has an empty/unparseable reset
  duration; the VK admin path rejects such a config (4xx).
- A `TokenLimited`/`RequestLimited` denial reaches the gate's `{data,error}` response
  with `error.code` = the snake_case Decision name (the enum is observably surfaced).
- `server.go`, `migrate.go`, and the `New`/`NewWithShutdown` signatures are UNCHANGED;
  no `vk_rate_limit_state` table, no worker, no `UsageUpdate`/`Accrue` exists anywhere.

---

## 6. Validation commands

```bash
go test ./... && go vet ./... && go build ./...                 # exit 0 (binding)
go test ./internal/governance/ -run 'Evaluate|Allow|Token|Request|Window|ValidateRateLimit' -v
go test ./internal/store/ -run 'SumTokens|SumRequests|VirtualKey' -v
go test ./internal/admin/ -run VirtualKey -v
go test ./internal/server/ -v
go test ./internal/api/ -v
```
No UI build / Playwright needed — bf-gov-3 ships NO UI touch and NO mock correction
(the additive VK config fields surface to no asserted UI behavior). Hermetic only
(D7): no test may hit the network, sleep, or call real `time.Now` (there is no worker
and no `time.Ticker` anywhere in this plan).

---

## 7. Freeze rules + matrix-flip + WORKFLOW + no-leftovers

**Freeze rules (binding):**
- `internal/governance/quota.go` — bf-gov-3 is the **SECOND and FINAL** holder of the
  bf-gov-1 → bf-gov-3 internal serial (BIFROST-MAP §394). bf-gov-1 released it on its
  close; bf-gov-3 holds it now and RELEASES to NOBODY (chain end). Additive edits
  only; PRESERVE `NewQuotaEngine`/`Allow`; REUSE the shipped `windowStart`/`clock`.
- bf-gov-3 is **NOT** a `routes_admin.go` holder and **NOT** a `routes_openai.go`
  route-block holder (adapter bodies only; NO new route). It takes NO serial route
  slot. `internal/server/server.go` is UNTOUCHED (no worker → no lifecycle change).
- Migrations: NONE — bf-gov-3 adds NO table and NO column (every dimension reads the
  existing `request_log`; the dual-dimension config rides the existing VK
  `config_json` blob). `migrate.go` is unchanged.
- bf-gov-2 (`lists.go`) runs ∥; quota.go (gov-3) vs lists.go (gov-2) are disjoint
  (BIFROST-MAP §330). The `DecisionModelBlocked` value is the only shared touchpoint
  — declared in gov-3, emitted by gov-2.
- No reverse-engineering of the absent Bifrost ref (ESC-REF-ABSENT) — build to matrix
  + g0router conventions only. STOP-escalate on any undocumented Bifrost detail.

**Matrix-flip (at close, in `.planning/parity/matrix/bifrost-governance.md`):**
- PAR-BF-GOV-017 → **PARTIAL** (dual-dimension token+request HAVE, SQL-live; the
  persisted-counter `CurrentUsage`/`LastReset` schema is VAR=SQL-derived; CAS ESC).
- PAR-BF-GOV-038 → **HAVE** (inline `ValidateRateLimit`; GORM BeforeSave ESC).
- PAR-BF-GOV-015 → **HAVE** (calendar window via reused `windowStart`; AfterFind ESC).
- PAR-BF-GOV-019 → **HAVE** (rate-limit calendar reset via `windowStart`).
- PAR-BF-GOV-039 → **HAVE** (lazy reset INHERENT in the windowStart-bounded aggregate).
- PAR-BF-GOV-021 → **VAR** (g0router's `SaveUsage` IS the streaming-aware finalized
  write — one row per completed request with final usage; no `UsageUpdate` type).
- PAR-BF-GOV-022 → **PARTIAL** (VK→Team slice HAVE via the persisted `request_log`
  rows the SQL-live aggregates read; global/user/scoped tiers ESC).
- PAR-BF-GOV-023 → **VAR/ESC** (10s in-memory→DB sync worker N/A by-design — g0router
  has no in-memory governance accumulator; restart-survival is automatic under
  SQL-live; same divergence class as the ESC'd sync.Map store).
- PAR-BF-GOV-016 → **VAR/ESC** (budget DB-sync N/A by-design — budgets are SQL-live
  (bf-gov-1 D8), nothing to dump; same rationale as 023).
- PAR-BF-GOV-044 → **VAR/ESC** (startup reset N/A by-design — no in-memory counter to
  reset; a fresh process reading `request_log` live is already correct).
- PAR-BF-GOV-045 → **VAR/ESC** (graceful flush N/A by-design — no in-memory deltas;
  `SaveUsage` persists each request).
- PAR-BF-GOV-018 → **PARTIAL** (SQL-accrual variant HAVE — persisted-row + live
  aggregate; CAS-spin mechanism ESC).
- PAR-BF-GOV-035 → **HAVE** (`Decision` enum + emitted gov-3 subset; Model/Provider/
  MCP values declared-not-emitted, owned by gov-2/mcp).
- PAR-BF-GOV-036 → **PARTIAL** (`EvaluationResult{Decision,Reason,Status}` HAVE; full
  RateLimitInfo/BudgetInfo/UsageInfo sub-structs ESC-REF-ABSENT).

**`open-questions.md` (append at close):**
```
## bf-gov-3 — dual-dim RL (SQL-live) + reset + Decision enum — 2026-06-15
- [ ] 10s in-memory→DB sync worker + startup-reset + graceful-flush (gov 016/023/044/045) — VAR/ESC by-design. g0router governance is fully SQL-live (budget bf-gov-1 D8, tokens+requests bf-gov-3 D1/D3): there is no in-memory accumulator to sync, reset, dump, or rehydrate. A worker would be the maintenance mechanism for exactly the sync.Map in-memory store the program ESCs. Restart-survival is automatic (every dimension reads persisted request_log live — strictly better than rehydrate/flush); calendar reset is inherent in the windowStart-bounded aggregate. Why: legitimate sound-parity (like 9router's deploy-relay escalations), NOT a missing worker.
- [ ] UsageUpdate/Accrue streaming-accrual type (gov 021, VK slice of 022) — VAR. g0router's SaveUsage (requestlog.go:36-124) is the streaming-aware finalized write: one request_log row per completed request with final usage = matrix quirk #6 (tokens once with usage data; one request on completion, never per-chunk). The SQL-live aggregates honor it inherently; an Accrue method would be dead code. Why: verify the existing write, don't build a no-consumer type.
- [ ] EvaluationResult sub-structs (RateLimitInfo/BudgetInfo/UsageInfo per-dimension remaining/reset metadata, gov 036) — ESC-REF-ABSENT; bf-gov-3 carries {Decision,Reason,Status} only. Why: the rich sub-struct shapes are unverifiable without the frozen Bifrost ref.
- [ ] Decision values ModelBlocked/ProviderBlocked/MCPToolBlocked (gov 035) — DECLARED in the gov-3 enum but EMITTED by bf-gov-2 (lists) / bf-mcp tiers, not gov-3. Why: shared enum type; emission owned by the sibling plans.
- [ ] CAS-spin atomic bumps (gov 013/018-CAS/041) — ESC; g0router "accrues" by the persisted request_log row + checks the live SQL aggregate (the MAP-directed 018 SQL variant). Why: presupposes the sync.Map in-memory store (ESC).
- [ ] global/user/per-user-scoped/per-VK-scoped usage tiers (gov 022 non-VK, 043) — ESC; bf-gov-3 builds the VK→Team slice only. Why: same slice boundary as bf-gov-1.
- [ ] Cluster remote-baseline rate check + ghost-node reconciliation (gov 040, 024, 025) — ESC; single-binary SQLite gateway. Why: clustering is a product-category change.
- [ ] Dual-dimension SQL hot-path cost — token SUM + request COUNT are read per gate call (alongside the shipped budget SUM); D1 notes they MAY be folded into one combined query. If profiling shows a hot-path concern, that single-query optimization (or a cached snapshot) is an additive follow-up. Why: tracked as a perf knob, not a correctness gap (and still no worker — a cache would be process-local + best-effort).
```

**`docs/WORKFLOW.md` (update at close):** add a bf-gov-3 row — dual-dimension RL
(token + request, BOTH SQL-live over request_log) + calendar-aligned lazy reset
(reusing the shipped `windowStart`, inherent in the bounded aggregate) +
`Decision`/`EvaluationResult` enum surfaced through the gate `error.code` + inline
`ValidateRateLimit` shipped (Go-only, no routes, no UI, **no new table, no server
change**); rows 015/017(part)/018(SQL-part)/019/035/036(part)/038/039 BUILD/PARTIAL,
021/022(part)/016/023/044/045 VAR/ESC-by-design (fully SQL-live governance — no
in-memory accumulator → no sync worker) flipped per §7; ESC/VAR items recorded in
open-questions; quota.go internal serial CLOSED (chain end, released to nobody);
ESC-REF-ABSENT honored (built to matrix only).

**No-leftovers confirmation (binding):** because every dimension is a LIVE SQL read
consumed by `Evaluate`, the apparatus that carried no-leftovers risk in the prior
draft (in-memory counter, `Accrue`, sync worker, state table) is REMOVED — there is
nothing inert to ship. bf-gov-3 adds the dual-dimension fields (consumed by
`Evaluate`'s checkToken/checkRequest, D1/D3), `SumTokensByAPIKey` + `SumRequestsByAPIKey`
(consumed by those live checks, D1), `ValidateRateLimit` (consumed by the VK admin
path, D5 — STOP if no live caller), and the `Decision` enum + `Evaluate` (surfaced in
the real `{data,error}` gate response via `error.code`, D8 — STOP if dead). The
streaming finalization (021/022) is satisfied by VERIFYING the existing `SaveUsage`
write (a cited/added test), not by building a no-consumer `Accrue`. No dead column,
field, method, enum value (except the documented gov-2/mcp-owned Decision values),
table, or worker is introduced; each new surface has a grep-proven live consumer (§5)
or the plan STOPS and escalates.
```
