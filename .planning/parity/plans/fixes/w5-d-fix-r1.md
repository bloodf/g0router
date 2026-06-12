# w5-d fix micro-plan — diff-gate round 1 (Fable 5, 2026-06-12)

Source: `artifacts/w5-d-usage-read-apis-diff-scoped-gpt.txt` (cycle 1, REJECT).
ALL FOUR findings verified REAL against the live tree. DISPATCH ORDER CONSTRAINT:
this fix runs AFTER w5-f merges (Finding 2's shared-instance wiring needs w5-f's
server-side Tracker/Recorder construction to exist; w5-f owns server.go).

## Finding 1 (BLOCKER) — "stats.go live aggregation increments only request counts;
prompt_tokens/completion_tokens/cost stay zero for today/24h."
REAL (verified: the live loop passes `nil` counter maps into addOrUpdate*, so only
requests increment). FIX: accumulate `r.PromptTokens`, `r.CompletionTokens`,
`r.Cost` into totals AND all five breakdowns on the live path, mirroring
`usageRepo.js:546-613` (per-row tokens parsed + added to each dimension). Test
FIRST: extend `TestUsageStatsLivePath` with golden token/cost assertions on totals,
byProvider, byModel, byAccount, byApiKey, byEndpoint (currently asserts shape only —
the gap that let this pass); run failing → fix.

## Finding 2 (BLOCKER) — "BuildUsageServices creates fresh Tracker/Ring instead of
consuming the existing w5-b tracker/ring; pending/active/recent stats disconnected
from real traffic."
REAL (verified: NewTracker/NewRing constructed inside BuildUsageServices). FIX:
change `BuildUsageServices(st, deps)` to ACCEPT the shared `*usage.Events`,
`*usage.Tracker`, `*usage.Ring` (constructed ONCE in `internal/server` — w5-f's
merged glue constructs them for the API handlers; reuse those exact instances and
pass them to both RegisterOpenAIRoutes and the admin bootstrap). After the change,
`grep -c 'usage.NewTracker' internal/admin/` → 0 (construction lives only in
internal/server). Test FIRST: `TestStatsSeesSharedTracker` — construct services with
an injected tracker, Start a pending request on it, Stats() shows it in
activeRequests; run failing (currently impossible to inject) → fix.

## Finding 3 (MAJOR) — "PATCH /api/pricing returns merged default+user pricing; plan
requires updated USER pricing."
REAL (verified: handler returns h.resolver.Merged()). Ref: `updatePricing` returns
`getUserPricing()` (`pricingRepo.js:77`) and the route returns that
(`src/app/api/pricing/route.js:75-82`). FIX: add/use a `Resolver.UserPricing()`
passthrough (or store read) and return THAT (snake_cased) from PATCH; DELETE has the
same contract (`resetPricing` returns user pricing, `route.js:104-115`) — align it
in the same pass. Extend `TestPricingPatchValidation` to assert the response body
contains ONLY user overrides (a canonical default model NOT in the body).

## Finding 4 (MAJOR) — "aggregateDaily unchecked map type assertions panic on
malformed usage_daily JSON."
REAL (verified: `day["byProvider"].(map[string]any)` etc.). FIX: use checked
assertions (`v, ok := ...`); missing/malformed sections are SKIPPED (ref parity —
JS `day.byProvider || {}` treats absent as empty, `usageRepo.js:432-440`); never
panic, never fail the whole stats call for one bad day row. Test FIRST:
`TestStatsDailyMalformedRow` — usage_daily row whose data is `{"byProvider": 42}` or
missing sections → Stats returns successfully, bad sections contribute nothing; run
(panics) → fix.

## Ownership
`internal/usage/stats.go`(+test), `internal/admin/usage.go`(+test),
`internal/admin/pricing.go`(+test), `internal/usage/pricing.go` (only if
UserPricing passthrough is added there), `internal/server/routes_admin.go` +
`internal/server/server.go` (shared-instance plumbing — ONLY the
construction/injection lines; coordinate with w5-f's merged wiring, do not restructure it).

## Binary acceptance
- `go build ./... && go vet ./... && go test ./...` green; `go test -race ./internal/usage/ ./internal/admin/ ./internal/server/` green.
- `grep -rc 'usage.NewTracker' internal/admin/` → all `:0` (single construction point in internal/server).
- TestStatsSeesSharedTracker, TestStatsDailyMalformedRow pass; extended
  TestUsageStatsLivePath token/cost golden assertions pass; PATCH/DELETE pricing
  tests assert user-only payload.
