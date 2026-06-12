# w5-e fix micro-plan — diff-gate round 1 (Fable 5, 2026-06-12)

Source: `artifacts/w5-e-sse-provider-quota-diff-scoped-gpt.txt` (cycle 1, REJECT).
All four findings verified REAL. Per-finding:

## Finding 1 (BLOCKER) — "provider quota API emits non-snake_case keys
(remainingPercentage, resetAt)."
REAL with a recorded tension: the REF itself emits camelCase quota keys
(`open-sse/services/usage.js:191` resetAt etc.) — but `AGENTS.md:26` ("All API
responses use snake_case JSON with a `{data, error}` envelope") is the binding
g0router decision and overrides ref styling on OUR api surface (the W6 dashboard
consumes g0router's contract, not 9router's). FIX: snake_case every key the quota
fetchers emit (`remaining_percentage`, `reset_at`, and audit the full payload for
other camelCase keys). Extend the fetcher tests' golden assertions to the snake_case
keys.

## Finding 2 (MAJOR) — "StreamSnapshot hardcodes active_requests to an empty slice."
REAL (verified `providerusage.go:62`). FIX: assemble activeRequests from the
tracker's byAccount state + connection-name cache, porting `getActiveRequests`
(`usageRepo.js:198-241`): per (connectionId, modelKey) count>0 → {model, provider
(parsed from "model (provider)" key), account name with `Account <id[:8]>...`
fallback, count}. Test FIRST: `TestStreamSnapshotActiveRequests` — Start two pending
requests on the tracker → snapshot active_requests has the entries with correct
shapes; run failing → fix.

## Finding 3 (MAJOR) — "TestConnectionUsageRoute404 does not register both
/api/usage/stats and /api/usage/{connectionId}; coexistence proof skipped."
REAL (the w5-e plan's cycle-3 disposition PROMISED "the implementer re-proves it in
TestConnectionUsageRoute404's router setup, which registers both shapes"). FIX:
register BOTH routes on the test router; assert /api/usage/stats still resolves to
the static handler AND an unknown connection id under /api/usage/{connectionId}
returns the 404 envelope.

## Finding 4 (MAJOR) — "fetchGeminiSubscriptionInfo returns raw errors without
wrapping."
REAL (AGENTS errors-as-values convention). FIX: wrap with `fmt.Errorf("gemini
subscription info: %w", err)` (and audit sibling fetcher error returns in the same
file for the same gap).

## Ownership
`internal/usage/providerusage.go`(+test), `internal/admin/connectionusage_test.go`,
`internal/admin/usagestream_test.go` (only if snapshot-shape assertions live there).
DISJOINT from the concurrent w5-f job (internal/api, internal/translation,
internal/server/{server,routes_openai}.go) — do not touch its files; if `git commit`
hits index.lock, wait 10s and retry up to 5 times. Test failures in internal/api are
the other job's TDD state — judge acceptance on owned packages.

## Binary acceptance
- `go build ./internal/admin/... ./internal/usage/... && go vet` same — green; `go test ./internal/admin/... ./internal/usage/...` green; `go test -race ./internal/admin/ ./internal/usage/` green.
- `grep -c 'remainingPercentage\|resetAt' internal/usage/providerusage.go` → 0 (emitted keys; parsing REF-side camelCase INPUT fields like resetTime remains correct).
- TestStreamSnapshotActiveRequests passes; coexistence-registering TestConnectionUsageRoute404 passes.
