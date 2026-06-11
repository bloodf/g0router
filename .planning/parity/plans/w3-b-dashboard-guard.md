# w3-b — Centralized dashboard guard middleware, local-only gate, tunnel toggles

Rows: PAR-AUTH-007 (centralized guard with path lists, `src/dashboardGuard.js:22-65,165-241`), PAR-AUTH-011 (local-only route gate: loopback host + origin, `src/dashboardGuard.js:69-81,83-100`), PAR-AUTH-013 (tunnel dashboard access toggle, `src/dashboardGuard.js:197-214`, `src/app/api/settings/require-login/route.js`), PAR-AUTH-027 (tunnel login block, `src/app/api/auth/login/route.js:11-16,33-35`) + PAR-PR-1711 (`PARITY.md:250` "Persistent dashboard session cookie — 30d TTL, unified parser") ADAPTED under decision 2 ("Keep g0router opaque SQLite session tokens (7-day TTL). No JWT.", `matrix/9router-auth.md` decisions table): unified cookie parsing yes; 30d TTL no — 7-day stands. Scope per `WAVE-3-MAP.md` track 1, plan 2. Frozen ref @ 827e5c3. Depends on w3-a MERGED (it owns `internal/admin/auth.go` first; this plan touches it after).

In-repo integration: `internal/server/server.go:16-40` (`New` builds `*fasthttp.Server`; guard wraps the root handler), `internal/admin/auth.go:100` (`RequireSession` — superseded for /api/* by the central guard; kept for direct handler use), `internal/auth/session.go:75` (`Validate` — the opaque-token check replacing the ref's `verifyDashboardAuthToken`, decision 2), `internal/store/settings.go:9` (settings keys: requireLogin, tunnelUrl, tailscaleUrl, tunnelDashboardAccess).

## Ref behavior to port (`dashboardGuard.js` — read whole file)

Evaluation order of `proxy(request)` (:165-241), ported as a fasthttp middleware
`internal/server/guard.go` evaluated BEFORE route dispatch:

1. **LOCAL_ONLY_PATHS** (:69-81): prefix match → require local request, else 403
   `{"error":"Local only: CLI token required"}`. Port the list verbatim (:69-81);
   entries for routes that do not exist yet in g0router stay in the list (harmless —
   deny-by-default protects them regardless).
   `isLocalRequest` (:91-100): Host header hostname ∈ {localhost, 127.0.0.1, ::1}
   (strip port + IPv6 brackets, :85-89) AND, when an Origin header is present, its
   hostname must also be loopback; malformed Origin → NOT local. (PAR-AUTH-011)
2. **ALWAYS_PROTECTED** (:38-45): prefix match → require valid session (opaque
   `Sessions.Validate`, decision 2) OR valid CLI token; else 401. CLI-token
   validation is PAR-AUTH-012 (w3-d): the guard struct carries
   `CLITokenValidator func(*fasthttp.RequestCtx) bool` — nil means "no CLI tokens
   exist" → deny (today's truth; w3-d injects the real validator). This mirrors the
   ref's own structure: the guard CALLS `hasValidCliToken` defined with the apiKey
   utilities (`src/dashboardGuard.js:6-19` imports), which are w3-d's files.
3. **Public LLM API** (:35, :102-104, :118-122): prefixes /v1, /v1beta, /api/v1,
   /api/v1beta (exact-or-prefix match :102-104) → allow when loopback
   (PAR-AUTH-008's loopback clause) else require API key — `APIKeyValidator` field,
   nil → 401 `{"error":"API key required for remote API access"}` (PAR-AUTH-009 is
   w3-d's; nil-deny is today's truth and fail-closed).
4. **/api/* deny-by-default** (:188-194): PUBLIC_API_PATHS allow-list (:22-32,
   verbatim) bypasses; everything else requires session or CLI token; else 401.
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

**Unified cookie parser** (PR-1711 adapted): one helper `sessionTokenFromRequest`
(cookie name used by the existing admin handlers) used by guard + handlers — no
duplicate parsing. TTL stays 7-day (decision 2).

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

1. **Guard core** (`guard.go`): `type Guard struct { Sessions *auth.Sessions; Settings settingsReader; CLITokenValidator, APIKeyValidator func(*fasthttp.RequestCtx) bool }`; `func (g *Guard) Wrap(next fasthttp.RequestHandler) fasthttp.RequestHandler` implementing order 1-6 above; path lists as package-level `var` slices (verbatim from :22-81). Pure helpers `isLoopbackHostname`, `isLocalRequest`, `isPublicLlmApi` ported exactly (:85-104).
   Tests (`guard_test.go`): `TestGuardLocalOnlyPaths` (loopback+no-origin allow; remote 403; loopback host + remote origin 403; malformed origin 403), `TestGuardAlwaysProtected` (no session 401; valid session allow; nil CLITokenValidator denies), `TestGuardPublicLlmApiLoopback` (loopback allow; remote + nil APIKeyValidator 401), `TestGuardApiDenyByDefault` (unlisted /api/x 401; each PUBLIC_API_PATHS entry allowed), `TestGuardDashboardRedirects` (no token → /login redirect; requireLogin=false allows; tunnel host + access-disabled → /login), `TestGuardRootRedirect`.

2. **Wiring** (`server.go`): wrap the root handler; all existing routes keep working (existing integration tests must stay green — public paths cover /v1 loopback test traffic).
   Tests: `TestServerGuardWired` (remote /api/settings without session → 401 through the real server handler).

3. **Tunnel login block** (`admin/auth.go`): the PAR-AUTH-027 check before password verification; settings-driven.
   Tests: `TestLoginBlockedViaTunnelHost` (Host == tunnelUrl hostname + access disabled → 403; enabled → proceeds), `TestLoginNormalHostUnaffected`.

4. **Unified cookie parser**: extract `sessionTokenFromRequest` used by guard + admin handlers (PR-1711's parser unification; TTL untouched).
   Tests: `TestSessionTokenFromRequestSingleSource` (guard and handler read the same cookie name).

## Binary acceptance criteria

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `grep -c 'verifyDashboardAuthToken\|jwt\|JWT' internal/server/guard.go` → 0 (decision 2: opaque only).
- `grep -rn 'func init(\|panic(' internal/server/guard.go` → 0 hits.
- All six Task-1 test groups pass; pre-existing server/integration tests pass unchanged.
- The PUBLIC_API_PATHS, ALWAYS_PROTECTED, LOCAL_ONLY_PATHS slices match the ref lists verbatim (diff gate checks against `dashboardGuard.js:22-81`).

## Out of scope

API-key + CLI-token validator IMPLEMENTATIONS (PAR-AUTH-008 remote-key path/009/010/012/029 — w3-d injects into the two nil-deny fields). OIDC flow (w3-c). Tunnel feature itself (Wave 7 — only settings-driven host checks here). JWT (decision 2). The login-limiter (w3-a, merged).
