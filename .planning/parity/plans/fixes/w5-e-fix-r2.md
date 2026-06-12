# w5-e fix micro-plan — diff-gate round 2 (Fable 5, 2026-06-12)

Source: `artifacts/w5-e-sse-provider-quota-diff-scoped-gpt.txt` (cycle 2, REJECT).

## Finding 1 (BLOCKER) — "Initial OAuth refresh resolves credentials by provider ID,
not the requested connection ID; can fetch usage with a different connection's token."
REAL (verified: `connectionusage.go:68` `resolver.ResolveKey(provider.ID)` —
ResolveKey performs provider-scoped connection selection, ignoring the requested
{connectionId}). FIX: drop the ResolveKey call entirely. Use the REQUESTED
connection's own credentials: if the connection is OAuth and within the refresh lead
window (or expired) → call the w5-pre production
`resolver.RefreshCredentials(conn.ID)` (connection-scoped, persists rotation) and
use the returned token; otherwise use `conn.AccessToken` as-is. This is the ref's
own shape: `refreshAndUpdateCredentials(connection)` operates on THE connection
(`src/app/api/usage/[connectionId]/route.js:23-117`). Test FIRST:
`TestConnectionUsageUsesRequestedConnection` — store with TWO connections for the
same provider (different tokens); request {connectionId} of the second → the fetch
saw the SECOND connection's token (fake fetcher captures it); run failing (current
code may resolve the first) → fix.

## Finding 2 (MAJOR) — "retry-once not asserted via refresher call count."
REAL (cheap): extend `TestConnectionUsageAuthExpiredRetryOnce` with a counting fake
refresher asserting EXACTLY one forced-refresh call (and zero on the success path).

## Finding 3 (MAJOR) — "Gemini project lookup uses conn.Metadata, plan requires
provider-specific-data path."
FALSE POSITIVE — `store.Connection.Metadata` IS g0router's providerSpecificData
storage: w3-f's credential plumbing persists providerSpecificData into the
`metadata` column (`internal/store/connections.go:22` Metadata field; w3-f plan
"credentials plumbing (providerSpecificData → …)"). Reading projectId from
conn.Metadata is the specified source. NO CHANGE.

## Finding 4 (MINOR) — "authExpiredPatterns is mutable package global state."
REBUTTED — it is an immutable lookup table never written after initialization,
identical in kind to the package-level data tables the repo already ships
(`internal/usage/pricingdata.go` ModelPricing/PatternPricing,
`internal/providers/catalog/aliases.go` ~140-entry alias map,
`internal/inference/errorclass.go` rule tables). The no-global-state convention
targets mutable state; constant tables are the established pattern. NO CHANGE.

## Ownership
`internal/admin/connectionusage.go`(+test) ONLY. DISJOINT from the concurrent w5-d
fix job (stats.go, admin/usage.go, admin/pricing.go, routes_admin.go, server.go) —
index.lock retry up to 5×10s if needed.

## Binary acceptance
- `go build ./... && go vet ./...` green; `go test ./internal/admin/...` green; `go test -race ./internal/admin/` green.
- `grep -c 'ResolveKey' internal/admin/connectionusage.go` → 0.
- `grep -c 'RefreshCredentials' internal/admin/connectionusage.go` ≥ 1.
- TestConnectionUsageUsesRequestedConnection passes; counting-refresher assertion passes.
