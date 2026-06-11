# w3-b — Centralized dashboard guard middleware, local-only gate, tunnel toggles

Rows: PAR-AUTH-007 (centralized guard with path lists, `src/dashboardGuard.js:22-65,165-241`), PAR-AUTH-011 (local-only route gate: loopback host + origin, `src/dashboardGuard.js:69-81,83-100`), PAR-AUTH-013 (tunnel dashboard access toggle, `src/dashboardGuard.js:197-214`, `src/app/api/settings/require-login/route.js`), PAR-AUTH-027 (tunnel login block, `src/app/api/auth/login/route.js:11-16,33-35`)(PAR-AUTH-008/009 — the /v1 public-LLM-API gating — are ENTIRELY w3-d's: gating + validators land together there so remote /v1 is never broken in an interim state; in THIS plan the guard passes the LLM prefixes through unchanged). (PAR-PR-1711 is NOT in this plan: its 30d TTL is rejected by decision 2 — "Keep g0router opaque SQLite session tokens (7-day TTL). No JWT." — and its parser-unification half is incidental engineering, not parity scope; recorded in WAVE-3-MAP.) Scope per `WAVE-3-MAP.md` track 1, plan 2. Frozen ref @ 827e5c3. Depends on w3-a MERGED (it owns `internal/admin/auth.go` first; this plan touches it after).

In-repo integration: `internal/server/server.go:16-40` (`New` builds `*fasthttp.Server`; guard wraps the root handler), `internal/admin/auth.go:100` (`RequireSession` — superseded for /api/* by the central guard; kept for direct handler use), `internal/auth/session.go:75` (`Validate` — the opaque-token check replacing the ref's `verifyDashboardAuthToken`, decision 2), `internal/store/settings.go:9` (settings keys: requireLogin, tunnelUrl, tailscaleUrl, tunnelDashboardAccess).

## Ref behavior to port (`dashboardGuard.js` — read whole file)

Evaluation order of `proxy(request)` (:165-241), ported as a fasthttp middleware
`internal/server/guard.go` evaluated BEFORE route dispatch:

1. **LOCAL_ONLY_PATHS** (:69-81): prefix match → require local request, else 403
   `{"error":"Local only: CLI token required"}`. Stage-1 list = the ref entries whose
   routes EXIST in g0router today: `/api/mcp/` ONLY (the mcp package exists; tunnel
   routes are Wave 7, cursor/kiro auto-import are Stage-2, cowork is excluded by
   decision 4). Each future feature ADDS its own entries with its plan — the list is
   a package-level var with a comment citing :69-81 for the full ref set. Acceptance
   test pins today's list exactly (no nonexistent-route behavior change).
   `isLocalRequest` (:91-100): Host header hostname ∈ {localhost, 127.0.0.1, ::1}
   (strip port + IPv6 brackets, :85-89) AND, when an Origin header is present, its
   hostname must also be loopback; malformed Origin → NOT local. (PAR-AUTH-011)
2. **ALWAYS_PROTECTED** (:38-45): prefix match → require valid session (opaque
   `Sessions.Validate`, decision 2) OR valid CLI token; else 401. Stage-1 list = ref
   entries with existing g0router routes: `/api/settings/database` does not exist
   yet → Stage-1 list is EMPTY with the ref set in a comment (:38-45); future plans
   add entries with their routes. The evaluation step itself ships now (tested with
   a synthetic entry in tests). CLI-token
   validation is PAR-AUTH-012 (w3-d): the guard struct carries
   `CLITokenValidator func(*fasthttp.RequestCtx) bool` — nil means "no CLI tokens
   exist" → deny (today's truth; w3-d injects the real validator). This mirrors the
   ref's own structure: the guard CALLS `hasValidCliToken` defined with the apiKey
   utilities (`src/dashboardGuard.js:6-19` imports), which are w3-d's files.
3. **Public LLM API** (:35, :102-104): prefixes /v1, /v1beta, /api/v1, /api/v1beta
   (exact-or-prefix match, port `isPublicLlmApi` :102-104) → PASS THROUGH unchanged
   in this plan (current open behavior preserved — no regression while validators
   don't exist). The loopback/API-key gating of `dashboardGuard.js:118-122`
   (PAR-AUTH-008/009) is added by w3-d TOGETHER with its validators, via the
   `APIKeyValidator` field this plan declares (unused/nil here).
4. **/api/* deny-by-default** (:188-194): PUBLIC_API_PATHS allow-list (:22-32 — keep verbatim: every entry is a no-auth
   endpoint by definition, safe to allow-list before its route exists since an
   unrouted path 404s AFTER the guard) bypasses; everything else requires session or CLI token; else 401.
5. **Dashboard routes** (:196-235): settings requireLogin (default true) +
   tunnelDashboardAccess (default false per `settings.tunnelDashboardAccess === true`
   truthiness); when tunnel access disabled AND Host matches tunnelUrl/tailscaleUrl
   hostname → redirect /login (PAR-AUTH-013). requireLogin false → allow. Else
   opaque-token cookie check via `Sessions.Validate` → invalid/missing → redirect
   /login. g0router serves the SPA from embedded FS — "dashboard routes" = non-/api,
   non-LLM paths.
6. **Root redirect** (:237-239): `/` → `/dashboard`.

**Tunnel login block** (PAR-AUTH-027, `login/route.js:11-16,33-35`): in the Login
handler (w3-a's file, now owned here for this addition): `isTunnelRequest` (Host
hostname equals tunnelUrl/tailscaleUrl hostname) AND `tunnelDashboardAccess != true`
→ 403 `{"error":"Dashboard access via tunnel is disabled"}` before password checks.

## Preconditions (a "0 hits" grep exits 1 = pass)

- `grep -rn 'guard.go' internal/server/` → 0 hits (new file)
- `grep -c 'func (s \*Sessions) Validate' internal/auth/session.go` ≥ 1 (reuse)
- `grep -c 'authMode' internal/admin/auth.go` ≥ 1 (w3-a merged — its Login changes present)
- `grep -rn 'CLITokenValidator\|APIKeyValidator' internal/` → 0 hits (new)

## Exclusive file ownership

NEW: `internal/server/guard.go` + `guard_test.go`.
TOUCH: `internal/server/server.go` (wrap root handler with the guard; :16-40),
`internal/server/server_test.go` / `integration_test.go` (wiring tests),
`internal/admin/auth.go` + `auth_test.go` (tunnel login block ONLY — w3-a merged
first; this is the declared serialized touch).
NOT touched: `internal/auth/*` (Validate reused as-is), API-key/CLI-token validator
implementations (w3-d injects via the two fields), OIDC paths (w3-c adds its routes
to PUBLIC_API_PATHS' existing `/api/auth/oidc` prefix — already in the ported list).

## Tasks (each: STEP (a) named failing tests first; STEP (b) implement)

1. **Guard core**. STEP (a) FIRST write `guard_test.go` with: `TestGuardLocalOnlyPaths` (loopback+no-origin allow; remote 403; loopback host + remote origin 403; malformed origin 403), `TestGuardAlwaysProtected` (no session 401; valid session allow; nil CLITokenValidator denies), `TestGuardPublicLlmApiPassthrough` (loopback AND remote /v1 both pass through to the LLM handler — today's behavior preserved; the field is inert until w3-d), `TestGuardApiDenyByDefault` (unlisted /api/x 401; each PUBLIC_API_PATHS entry allowed), `TestGuardDashboardRedirects` (no token → /login redirect; requireLogin=false allows; tunnel host + access-disabled → /login), `TestGuardRootRedirect` — run, all fail (no guard.go). STEP (b) implement `guard.go`: `type Guard struct { Sessions *auth.Sessions; Settings settingsReader; CLITokenValidator, APIKeyValidator func(*fasthttp.RequestCtx) bool }`; `Wrap(next fasthttp.RequestHandler) fasthttp.RequestHandler` implementing order 1-6; path lists as package-level `var` slices per the Stage-1 existing-routes rule above (PUBLIC verbatim :22-32; LOCAL_ONLY ["/api/mcp/"]; ALWAYS_PROTECTED [] — ref sets in comments); pure helpers `isLoopbackHostname`/`isLocalRequest`/`isPublicLlmApi` exactly (:85-104).

2. **Wiring**. Tests FIRST: `TestServerGuardWired` (remote /api/settings without
   session → 401 through the real server handler; /v1 chat reaches its handler
   unchanged) — run, fails. THEN wrap the root handler in `server.go`; all existing
   integration tests must stay green.

3. **Tunnel login block** (`admin/auth.go`): the PAR-AUTH-027 check before password verification; settings-driven.
   Tests FIRST: `TestLoginBlockedViaTunnelHost` (Host == tunnelUrl hostname + tunnelDashboardAccess unset/false → 403 with error "Dashboard access via tunnel is disabled", NO session cookie set; tunnelDashboardAccess=true + correct password → 200 AND Set-Cookie session present), `TestLoginNormalHostUnaffected` (non-tunnel Host + correct password → 200 regardless of toggle).

4. **Cookie contract test** (no refactor): the guard MUST read the same session
   cookie the Login handler sets (it consumes `Sessions.Validate` with the token from
   the existing cookie name). Tests FIRST: `TestSessionCookieRoundTrip` (behavioral:
   POST login → capture Set-Cookie name+token → a guarded /api request with exactly
   that cookie passes; a renamed cookie → 401). This is the guard-correctness test
   for PAR-AUTH-007's session check — no parser refactor involved.

## Binary acceptance criteria

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `grep -c 'verifyDashboardAuthToken\|jwt\|JWT' internal/server/guard.go` → 0 (decision 2: opaque only).
- `grep -rn 'func init(\|panic(' internal/server/guard.go` → 0 hits.
- All six Task-1 test groups pass; pre-existing server/integration tests pass unchanged.
- PUBLIC_API_PATHS matches `dashboardGuard.js:22-32` verbatim; LOCAL_ONLY_PATHS == ["/api/mcp/"]; ALWAYS_PROTECTED == [] with the ref sets cited in comments (Stage-1 existing-routes rule above); `TestGuardListContents` pins all three exactly.

## Out of scope

API-key + CLI-token validator IMPLEMENTATIONS (the validation halves of PAR-AUTH-008/009 + 010/012/029 — w3-d injects into the two nil-deny fields; this plan ships only the guard-side routing/loopback/deny semantics declared PARTIAL in the Rows header). OIDC flow (w3-c). Tunnel feature itself (Wave 7 — only settings-driven host checks here). JWT (decision 2). The login-limiter (w3-a, merged).
