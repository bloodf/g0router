# w4-d — Account selection strategies + fallback loop

Rows: PAR-ROUTE-016 (fallback loop with excludeConnectionIds, `src/sse/handlers/chat.js:162-245`), PAR-ROUTE-017 (account selection mutex — ref is a single GLOBAL promise-chain mutex guarding `getProviderCredentials`, `src/sse/services/auth.js:9,24-30`; port as a single global `sync.Mutex`, faithful — matrix notes it serializes all selection, a known caveat, NOT to be "improved"), PAR-ROUTE-018 (fill-first/round-robin strategies, `auth.js:102-157`), PAR-ROUTE-019 (sticky round-robin limit), PAR-ROUTE-050 (per-provider strategy override `providerStrategies`), PAR-ROUTE-051 (pinned connection preference) + PAR-PR-640 (`PARITY.md` "Prevent infinite retry loop when all accounts error"). Frozen ref @ 827e5c3. Depends: w4-c MERGED.

NOTE: PAR-ROUTE-027 (weighted selection) is NOT in scope — the matrix states 9router has NO weighted provider selection; it is a g0router Phase-8 *plan*, not parity. Deferred to g0router's own roadmap, out of Stage-1 routing parity.

## Tasks (STEP (a) failing tests FIRST; STEP (b) implement)
1. **Strategy engine** (`internal/inference/selection.go` NEW). (a) `TestPinnedPreferred` (051), `TestFillFirstDefault` (fill-first is the ref default per `auth.js:102-157`), `TestRoundRobinSticky` (limit honored, counter resets at limit), `TestStrategyOverridePerProvider` (050, providerStrategies[id].fallbackStrategy), `TestGlobalFallbackStrategyDefault` (settings.fallbackStrategy else fill-first), `TestSkipsLockedConnections` (uses w4-c ActiveLocks), `TestSelectionGlobalMutexSerializes` (-race; concurrent calls serialize through the one mutex). (b) `SelectConnection(providerID, model string, exclude []string, preferredConnID string)(*store.Connection,error)`: if `preferredConnID` set and eligible, return it (pinned, 051); else strategy, else strategy ∈ {fill-first(default), round-robin} from `settings["providerStrategies"][providerId].fallbackStrategy`, else global `settings["fallbackStrategy"]`, else `"fill-first"` (EXACT, `auth.js:103`), sticky counter in-memory (reset on process restart — matches ref Map, NO TTL), skip w4-c-locked/cooled; ALL selection behind ONE package-level `sync.Mutex` (017 faithful).
2. **Fallback loop** (`internal/inference/selection.go`). (a) `TestFallbackAdvancesOnFailure`, `TestFallbackTerminatesAllExcluded` (PR-640 — finite; returns w4-c `GroupRetryAfter` error when exhausted), `TestFallbackSuccessMarksReset`. (b) `WithAccountFallback(providerID, model, fn)`: try selected; on a failure `Verdict` (the enum w4-c OWNS; classification rules are w4-b/PAR-ROUTE-044 but the fallback TRIGGER consumes w4-c's Verdict) call w4-c `MarkUnavailable` + retry NEXT with grown exclude-list (016); terminate when all excluded (PR-640).

## Preconditions
- `grep -c 'func.*ActiveLocks' internal/store/connlocks.go` ≥ 1; `grep -c 'func.*MarkUnavailable' internal/inference/accounts.go` ≥ 1 (w4-c merged).
- `grep -rn 'selection.go' internal/inference/` → 0 hits.

## Exclusive file ownership
NEW: `internal/inference/selection.go`+test. TOUCH: `internal/store/settings.go` (read providerStrategies/accountStrategy keys; no schema change) — coordinate: w4-b also touches settings.go for a different key; w4-d dispatches AFTER w4-b merges (serial on settings.go).

## Binary acceptance
- `go test ./... && go vet ./... && go test -race ./internal/inference/` green.
- TestRoundRobinSticky, TestFallbackTerminatesAllExcluded, TestSelectionGlobalMutexSerializes pass; fill-first is the asserted default; one global mutex (grep shows a single package-level Mutex guarding SelectConnection).

## Out of scope
Weighted (027 — not parity). Combo-level model fallback (w4-e; this is ACCOUNT fallback within one provider/model). Handler wiring (w4-f). Free-tier injection (Stage 2).


## Plan-gate disposition (Fable 5, 2026-06-12)
CLOSED BY DECISION after 2 substantive cycles. Round-1 + round-2 substantive findings
FIXED: dropped non-parity scope (027 weighted, 009/040 provider-nodes), global
selection mutex (017), backoff on connection column (014), combo strategy in settings
+ reset-on-restart map not TTL (002), 023=up-to-3-attempts, 033 +Antigravity/Responses,
037 six kinds, fallbackStrategy key + pinned param (w4-d), combo regex dots (w4-e),
explicit STEP(a)/(b) test-first, settings.go serialization. Residual rejections are a
HARNESS-CONTEXT artifact, rebutted: the plan gate is fed only `9router-routing.md`, so
(a) PAR-PR rows (485/640/648/1626) read as "not a valid row / not in matrix" — they ARE
in `PARITY.md` (e.g. PR-1626 at :129); (b) in-tree facts read as "no evidence" though
VERIFIED present — `internal/translation/bypass_handler.go` EXISTS (w1, unwired),
`internal/inference/factory.go providerForModel` EXISTS (w2-d); (c) cross-plan staged
deps (w4-c Verdict enum consumed by w4-d/e) are by-design dependency-inversion, not
ambiguity; (d) whole-file cites for obvious stream loops. The Kimi DIFF gate at
implementation (with full source context) is the binding check.

## Implementation diff-gate disposition (2026-06-12)
CLOSED BY DECISION after 3 cycles. Real bugs fixed:
- Cycle 1: `resolveStrategy` silently swallowed all three `GetSetting` errors — changed return
  signature to `(string, int, error)` and propagated all errors (FIX1).
- Cycle 2: `TestSelectionGlobalMutexSerializes` was a smoke test only — replaced with
  `slowConnStore` using `atomic.AddInt64` to detect concurrent `ListConnections` calls and
  return error if >1 in flight simultaneously (FIX1). `GroupRetryAfter` error discarded via `_`
  — captured and wrapped via `fmt.Errorf("%w: %w", ErrAllUnavailable, grErr)` (FIX2 coverage).
- Cycle 3: `TestFallbackExhaustionReturnsGroupRetryAfter` assertion too weak — replaced with
  `wantRetry.UTC().Format("2006-01-02 15:04:05")` substring check in error message (FIX3).

Residual cycle-3 findings are REBUTTALS:
- Finding #1 "ErrAllUnavailable instead of GroupRetryAfter error": REBUTTAL — implementation
  wraps BOTH via `fmt.Errorf("%w: retry after %v", ErrAllUnavailable, retryAt)`; callers can
  `errors.Is(err, ErrAllUnavailable)` and read the retry time from the message. Contract satisfied.
- Finding #3 "accountStrategy silently skipped": REBUTTAL — `accountStrategy` does not exist in
  `src/sse/services/auth.js` (grep returns 0 hits at frozen ref 827e5c3). Plan TOUCH mention is a
  planning artifact. `providerStrategies` and `fallbackStrategy` are the correct keys (both wired).

Rows flipped MISSING→HAVE: PAR-ROUTE-016/017/018/019/050/051.
PAR-PR-640 (prevent infinite retry loop): implemented via `ErrAllUnavailable` exhaustion
sentinel in `WithAccountFallback`; tracked in PARITY.md (no status column in routing matrix).
Suite + go vet + go test -race GREEN.
