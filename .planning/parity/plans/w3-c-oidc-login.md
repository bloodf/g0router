# w3-c — OIDC dashboard login with PKCE, logout cookie clearing, secret probe

Rows: PAR-AUTH-005 (OIDC dashboard login with PKCE — `src/lib/auth/oidc.js:74-78` `createPkcePair`, `src/app/api/auth/oidc/start/route.js:24-46`), PAR-AUTH-021 (logout clears OIDC cookies — `src/app/api/auth/logout/route.js:8-10`: deletes `oidc_state`, `oidc_nonce`, `oidc_code_verifier` + the dashboard auth cookie), PAR-AUTH-022 (OIDC cookie TTL 10 min — `oidc/start/route.js:42` `maxAge: 10 * 60`), PAR-AUTH-028 (client-secret probe endpoint — `src/lib/auth/oidc.js:144-210` `probeOidcClientSecret`). Scope per `WAVE-3-MAP.md` track 1, plan 3. Frozen ref @ 827e5c3. Depends on w3-a MERGED (authMode + `oidcConfigured` helper exist) and w3-b MERGED (guard live; `/api/auth/oidc` is in the ported PUBLIC_API_PATHS, `dashboardGuard.js:29`).

In-repo integration: `internal/auth/oauth.go:176-188` (`randomURLSafe` + `pkceChallenge` — the PKCE primitives ALREADY EXIST, reuse; do not re-implement), `internal/auth/session.go:51-72` (session issue on successful OIDC login — same opaque token as password login, decision 2), `internal/admin/auth.go` (Logout handler — extend to clear OIDC cookies), `internal/store/settings.go` (OIDC settings keys defined by w3-a's `oidcConfigured`).

## Ref behavior to port (read `src/lib/auth/oidc.js` + the three route files whole)

- **Start** (`oidc/start/route.js:24-46`): create PKCE pair (verifier = 32 random
  bytes base64url; challenge = SHA256(verifier) base64url — `oidc.js:74-78`; matches
  the existing `pkceChallenge`, `oauth.go:184-188`), random state + nonce; redirect
  URI = `<public origin>/api/auth/oidc/callback`; build the authorization URL from
  the discovery document's `authorization_endpoint` with client_id, scopes, state,
  nonce, code_challenge(S256); set THREE cookies `oidc_state`, `oidc_nonce`,
  `oidc_code_verifier` — httpOnly, sameSite=lax, secure per request proto, path=/,
  `maxAge: 10*60` = 600s (PAR-AUTH-022). Discovery: fetch
  `<issuer>/.well-known/openid-configuration` (the ref's `discovery` object — port a
  minimal fetch+parse of authorization_endpoint + token_endpoint).
- **Callback**: verify returned `state` equals the `oidc_state` cookie; exchange code
  at token_endpoint with code_verifier (PKCE) + client_secret when configured;
  validate ID-token nonce claim equals the `oidc_nonce` cookie; on success issue the
  NORMAL opaque dashboard session (`Sessions` — decision 2; there is no separate OIDC
  session kind) and delete the three oidc_* cookies; on any mismatch → 401, no session.
- **Logout** (`logout/route.js:8-10`): existing Logout additionally deletes
  `oidc_state`, `oidc_nonce`, `oidc_code_verifier` (PAR-AUTH-021).
- **Probe** (`oidc.js:144-210` + route `src/app/api/auth/oidc/test/route.js`):
  POST `/api/auth/oidc/test` — given tokenEndpoint/clientId/clientSecret/redirectUri,
  sends a deliberately invalid code (`__oidc_test_invalid_code__`, `:148-153` body
  shape) and classifies the IdP error to report whether the secret is accepted; no
  secret → `{tested:false, valid:null, message:"No client secret was provided, so
  secret validation was skipped."}` (`:144-147` verbatim). PUBLIC route in the ref:
  the path is under the `/api/auth/oidc` PUBLIC_API_PATHS prefix
  (`dashboardGuard.js:29`) — port as public for parity (it acts only on
  caller-provided values; no stored secrets read).

## Preconditions (a "0 hits" grep exits 1 = pass)

- `grep -c 'func pkceChallenge' internal/auth/oauth.go` ≥ 1 (reuse)
- `grep -c 'oidcConfigured' internal/admin/auth.go` ≥ 1 (w3-a merged)
- `grep -c 'func (g \*Guard) Wrap' internal/server/guard.go` ≥ 1 (w3-b merged)
- `grep -rn 'oidc_state\|OIDCStart' internal/` → 0 hits (new)

## Exclusive file ownership

NEW: `internal/auth/oidc.go` + `oidc_test.go` (discovery, PKCE pair via existing
primitives, auth-URL builder, code exchange, nonce/state checks, probe logic);
`internal/admin/oidc.go` + `oidc_test.go` (start/callback/probe handlers + cookies).
TOUCH: `internal/admin/auth.go` + `auth_test.go` (Logout: delete the 3 oidc cookies),
`internal/server/routes_admin.go:27+` (`RegisterAdminRoutes`) ONLY to register the 3 routes (start/callback/test) (w3-b's guard
file untouched — `/api/auth/oidc` prefix is already public-listed).
NOT touched: `internal/auth/oauth.go` (primitives reused), guard.go, limiter, w3-d files.

## Tasks (each: STEP (a) named failing tests FIRST, run, show fail; STEP (b) implement)

1. **OIDC core** (`internal/auth/oidc.go`). Tests FIRST (`oidc_test.go`, httptest IdP):
   `TestDiscoveryFetch` (well-known JSON → endpoints), `TestAuthURLContainsPKCEAndState`
   (code_challenge=S256(verifier), state, nonce, scopes, client_id, redirect_uri),
   `TestExchangeSendsVerifierAndSecret` (form fields incl. code_verifier; client_secret
   only when set), `TestNonceMismatchRejected`, `TestStateMismatchRejected`,
   `TestProbeNoSecretSkips` (`tested:false, valid:null`, verbatim message),
   `TestProbeInvalidCodeClassification` (IdP `invalid_client` → valid:false; `invalid_grant`
   → valid:true — secret accepted, code rejected; per `oidc.js:160-210` classification —
   read and port the exact mapping).
2. **Handlers** (`internal/admin/oidc.go`). Tests FIRST: `TestOIDCStartSetsThreeCookies`
   (httpOnly, SameSite=Lax, Max-Age=600 exactly — PAR-AUTH-022; secure flag follows
   request proto), `TestOIDCCallbackIssuesOpaqueSession` (valid state+nonce → Set-Cookie
   session via Sessions; oidc_* cookies deleted), `TestOIDCCallbackBadState401`,
   `TestProbeEndpointPublic` (reachable without session — parity with the ref's public `/api/auth/oidc` prefix, `dashboardGuard.js:29`).
3. **Logout extension** (`admin/auth.go`). Tests FIRST: `TestLogoutClearsOIDCCookies`
   (response carries deletion Set-Cookie for all 3 names + session cleared).

## Binary acceptance criteria

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `grep -c 'maxAge\|Max-Age\|600' internal/admin/oidc.go` ≥ 1 AND `TestOIDCStartSetsThreeCookies` asserts Max-Age=600 exactly.
- `grep -rn 'func init(\|panic(' internal/auth/oidc.go internal/admin/oidc.go` → 0 hits.
- Decision-2 criterion (behavioral, not grep): `TestOIDCCallbackIssuesOpaqueSession` proves the callback issues the session via `auth.Sessions` (opaque store token); NO new session-token mechanism is introduced (ID-token PARSING for the nonce claim is required and permitted — it is not a session token).
- `TestNonceMismatchRejected`, `TestStateMismatchRejected`, `TestProbeNoSecretSkips`, `TestLogoutClearsOIDCCookies` pass.

## Out of scope

JWT sessions (decision 2). OIDC for the LLM API (dashboard only, per the rows). UI
login page (Wave 6). authMode switch logic (w3-a, merged). Guard changes (w3-b,
merged — route registration only). Provider OAuth (w3-f).

## Diff-gate disposition (2026-06-11)
CLOSED BY DECISION after 3 cycles. Real findings all FIXED & verified: (BLOCKER,
security) public probe `OIDCTest` (oidc.go:249) reads ONLY request-body values — NO
settings/stored-secret access (verified; stored reads at OIDCStart/OIDCCallback are
legit config); caller `token_endpoint` accepted; snake_case JSON; routed
TestProbeEndpointPublic; HttpOnly now asserted via raw Set-Cookie attribute. Residual
"/api/keys scope creep" is a cumulative-diff artifact: routes_admin.go is shared with
w3-d, so base..HEAD shows w3-d's /api/keys registration; w3-c registered only the 3
OIDC routes. Suite + admin tests green. PAR-AUTH-005/021/022/028 satisfied.
