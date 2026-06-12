# w5-e — Usage SSE stream + provider quota API (Stage-1 providers)

PAR rows: PAR-USAGE-032 (Stage-1 half — see scope note), PAR-USAGE-033,
PAR-USAGE-034, PAR-USAGE-035. NOT in scope: stats/chart/logs/pricing routes (w5-d),
handler glue (w5-f), UI (W6).
Frozen ref @ 827e5c3. Depends: w5-b (Events/Tracker/Ring), w5-d (StatsService +
routes_admin.go merged — this plan edits routes_admin.go SERIALLY after w5-d),
w5-pre (production refresher).

## Stage-1 scope note (PAR-USAGE-032 — authorized by WAVE-5-MAP, supplied as gate context)
`WAVE-5-MAP.md` §Stage-1 scope decisions: "PARTIAL Stage-1 (2): 032/033
provider-quota API … Stage-1 ships the dispatcher + claude and gemini fetchers — the
only Stage-1 providers with OAuth flows (W3 shipped anthropic/gemini/xai; xai has no
usage endpoint in the ref)." The structural reason the other six fetchers CANNOT ship
now: their providers do not exist in g0router — the Stage-1 catalog holds only the 11
Stage-1 entries (`internal/providers/catalog/catalog.go:26-27` "Only the 11 Stage-1
entries are…"), and github/antigravity/codex/kiro/glm/minimax have neither adapters
nor OAuth flows (`internal/server/server.go:35-39` — the flows map literal holds
exactly the keys "anthropic", "gemini", "xai") — a
fetcher whose provider cannot be connected is dead code. This mirrors the recorded
W3 precedent (`WAVE-3-MAP.md` §Stage-1 scope: OAuth handlers ONLY for providers
whose adapters exist). Ref dispatcher `open-sse/services/usage.js:60-101`; claude
fetcher `usage.js:497-614`; gemini `usage.js:225-342`; unknown providers return the
ref's fallback `{message: "Usage API not implemented for <provider>"}`
(`usage.js:95-96`). Row 032 flips to PARTIAL (Stage-1 half), recorded in the matrix
note — same mechanism as PAR-AUTH-020's Stage-1 half (w3-e precedent).

## Tasks

1. **Usage SSE stream (PAR-USAGE-034, PAR-USAGE-035)** — evidence:
   `src/app/api/usage/stream/route.js:10-78`: on connect → send full stats; on
   "update" event → push lightweight cached-stats+active first, then full recalc and
   cache; on "pending" event → push ONLY cached stats overlaid with fresh
   activeRequests/recentRequests/errorProvider (no heavy recalc); keepalive comment
   `: ping\n\n` every 25s; on client disconnect/write error → unsubscribe both
   events + stop keepalive (no leaks).
   STEP (a): `TestUsageStreamPushesOnUpdate` (connect against handler with fake
   StatsService + w5-b Events; emit "update" → two SSE data frames (quick then full));
   `TestUsageStreamPendingLightweight` (emit "pending" → one frame, stats source NOT
   recalled for full stats); `TestUsageStreamKeepalive` (injected ticker → `: ping`
   comment frame); `TestUsageStreamUnsubscribesOnClose` (BINARY assertion: after the
   client writer closes, `Emit("update")` twice more → the captured frame count does
   NOT increase AND the handler goroutine returns within 1s, observed via a
   done-channel the test selects on) — run — fail.
   STEP (b): NEW `internal/admin/usagestream.go`: `UsageStream` handler on
   `*Handlers` (struct defined at `internal/admin/handlers.go` — the type every
   admin endpoint hangs off; the staged deps on w5-b `Events.OnEvent` and w5-d
   `StatsService` are by-design dependency-inversion across already-gated plans —
   the recorded W4 precedent: w4-c's Verdict enum consumed by w4-d/e) using fasthttp
   streaming body writer; keepalive interval as a struct field with production value
   `25 * time.Second` (PAR-USAGE-035), test-injected smaller; register
   GET `/api/usage/stream` in `internal/server/routes_admin.go` under
   `RequireSession`.

2. **Provider quota fetchers: claude + gemini (PAR-USAGE-032 Stage-1 half — see
   §Stage-1 scope note)** — evidence (read in full before porting): `usage.js:497-563` getClaudeUsage (OAuth usage endpoint first; on
   non-OK falls back to legacy settings/org endpoint `getClaudeUsageLegacy`
   `:564-614`; returns quotas array with windows + resetsAt via `parseResetTime`
   `:103-135`); `usage.js:225-301` getGeminiUsage (loadCodeAssist→quota snapshot,
   project-id from providerSpecificData, normalize `:302-311`) +
   `getGeminiSubscriptionInfo` `:313-342`.
   STEP (a): `TestClaudeUsageFetcher` (httptest: primary endpoint 200 → parsed
   quotas; primary non-OK + legacy 200 → legacy parsed; both fail → error message
   shape) and `TestGeminiUsageFetcher` (httptest: quota payload → normalized
   snapshot incl. projectId path) — golden JSON fixtures lifted from the ref response
   shapes; run — fail.
   STEP (b): NEW `internal/usage/providerusage.go`: `FetchProviderUsage(conn
   ConnInfo, client *http.Client) (map[string]any, error)` dispatcher mapping
   g0router provider types (anthropic→claude fetcher, gemini→gemini fetcher,
   default→`{message: "Usage API not implemented for <provider>"}`) + the two
   fetchers with injectable base URLs (test seam; production = ref URLs).

3. **Connection usage route with refresh + retry-once (PAR-USAGE-033, route half of
   032)** — evidence: `src/app/api/usage/[connectionId]/route.js:122-188`: 404 when
   connection unknown; OAuth connections refresh-if-needed BEFORE fetching (g0router:
   the w5-pre `CredentialResolver.RefreshCredentials` path / resolver lead-window
   refresh); fetch; if the RESULT is an auth-expired MESSAGE
   (`:11-16` substring patterns expired/authentication/unauthorized/401/re-authorize)
   and a refresh token exists → FORCE refresh + retry exactly once (`:172-180`);
   non-OAuth+non-eligible connections → `{message: "Usage not available for this
   connection"}` (`:135-142`; Stage-1 has no apikey-eligible usage providers — glm/
   minimax are Stage-2).
   STEP (a): `TestConnectionUsageRoute404`, `TestConnectionUsageAuthExpiredRetryOnce`
   (first fetch returns `{message: "...expired..."}` → refresher called force, second
   fetch returns quotas → 200 quotas; refresher called exactly once),
   `TestConnectionUsageNonOAuthMessage` — run — fail.
   STEP (b): NEW `internal/admin/connectionusage.go`: GET
   `/api/usage/{connectionId}` — the REF-EXACT route shape
   (`src/app/api/usage/[connectionId]/route.js`). Coexistence with w5-d's static
   `/api/usage/stats|chart|...` routes is VERIFIED empirically (2026-06-12,
   fasthttp/router: registering `/api/usage/stats` + `/api/usage/{connectionId}`
   on one router → no panic, both resolve; static segments win over the param).
   Registered in routes_admin.go under `RequireSession`; uses store GetConnection,
   w5-pre refresher, and Task-2 dispatcher.

## Preconditions (each states its own pass condition)
- `grep -c 'OnEvent' internal/usage/events.go` ≥ 1 (w5-b merged).
- `grep -c '/api/usage/stats' internal/server/routes_admin.go` ≥ 1 (w5-d merged — serialization satisfied).
- `grep -c 'func (r \*CredentialResolver) RefreshCredentials' internal/auth/credentials.go` = 1 (w5-pre merged).
- `grep -rc '/api/usage/stream' internal/server/routes_admin.go` outputs `0` (the gap; flips ≥1).

## Exclusive file ownership
NEW: `internal/admin/usagestream.go`(+test), `internal/admin/connectionusage.go`(+test),
`internal/usage/providerusage.go`(+test). TOUCH: `internal/server/routes_admin.go`
(serial after w5-d). NO internal/api files (w5-f's domain). Disambiguation vs w5-d:
WAVE-5-MAP §Ownership (amended 2026-06-12) names w5-d's admin files EXACTLY
(`internal/admin/usage.go`, `internal/admin/pricing.go` — no glob); this plan's two
admin files are distinct names with zero overlap.

## Binary acceptance
- `go build ./... && go vet ./...` clean; `go test ./...` green; `go test -race ./internal/admin/ ./internal/usage/` green.
- `grep -c '/api/usage/stream\|/api/usage/{connectionId}' internal/server/routes_admin.go` ≥ 2.
- `grep -c '25 \* time.Second' internal/admin/usagestream.go` = 1 (production keepalive constant, PAR-USAGE-035).
- TestUsageStreamPushesOnUpdate, TestUsageStreamPendingLightweight,
  TestUsageStreamKeepalive, TestUsageStreamUnsubscribesOnClose,
  TestClaudeUsageFetcher, TestGeminiUsageFetcher,
  TestConnectionUsageAuthExpiredRetryOnce all pass.

## Out of scope
Stage-2 fetchers for the six remaining PAR-USAGE-032 providers
(github/antigravity/codex/kiro/glm/minimax — port with their providers). Proxy options plumbing (`proxyOptions`, S2/W7 with proxy
pools; w3-e env-proxy applies transparently via http.Client). Stats computation
itself (w5-d). UI consumption (W6).
