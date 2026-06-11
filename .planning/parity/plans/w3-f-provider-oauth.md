# w3-f — Provider OAuth: anthropic complete + gemini + xai, credential refresh, key resolution

## Authorizing artifacts (verbatim quotes — verifiable in-repo)

- PAR-AUTH-019 (`matrix/9router-auth.md`): "OAuth credential manager for provider
  connections … g0router implements Anthropic OAuth with PKCE and refresh
  (`internal/auth/oauth.go:34-42`, `internal/admin/oauth.go:34-87`). 9router supports
  ~15 provider OAuth flows." — the row's gap IS manager+flows; key resolution into
  adapters is what "for provider connections" means operationally.
- PAR-PR-1249 (`PARITY.md:232`): "| #1249 | PAR-PR-1249 | Fix OAuth redirect URI
  handling for remote deployments |".
- w2-c deferral (`plans/w2-c-ollama-adapter.md` §Scope decisions, committed): "The
  ref's `providerSpecificData.baseUrl` override is NOT threaded in Stage-1 … the
  override is deferred to Wave 3 (credential plumbing)."
- w2-b deferral (`plans/w2-b-generic-openai-adapter.md` §Out of scope, committed):
  "OAuth/token refresh (Wave 3 — adapter uses `key.Value` as given)."
- Stage-1 handler set (`plans/WAVE-3-MAP.md` §Stage-1 scope decision, committed):
  anthropic + gemini + xai only.
- Ownership non-overlap: w3-f is the ONLY Wave-3 plan touching
  `internal/inference/*` or `internal/providers/*` (WAVE-3-MAP §Tracks: dashboard
  track owns admin/auth/server files; w3-e owns logging/proxy files). No in-flight
  plan shares these files.

Rows: PAR-AUTH-019 (OAuth credential manager for provider connections, PARTIAL — `open-sse/services/oauthCredentialManager.js`, `src/lib/oauth/services/*.js`; in-repo `internal/auth/oauth.go:34-42`, `internal/admin/oauth.go:34-87`) + PAR-PR-1249 (OAuth redirect URI handling for remote deployments) + the two recorded Wave-2 deferrals: ollama `providerSpecificData.baseUrl` override (w2-c plan §Host resolution) and the generic-adapter token-refresh hook (w2-b plan §refresh* = Wave-3). Scope per `WAVE-3-MAP.md` provider track. Frozen ref @ 827e5c3.

**Stage-1 provider set for handlers** (WAVE-3-MAP scope decision): anthropic (adapter
HAVE; complete the PARTIAL flow), gemini (adapter HAVE; OAuth fields
`providers.js:58-62` — clientId + clientSecret, Google OAuth), xai (generic adapter
HAVE; `providers.js:273-280` — clientId, tokenUrl, refreshUrl; service
`src/lib/oauth/services/xai.js:97-180`, endpoint discovery :52-80). Decision 1:
monolithic per-provider config constructors (like the existing `AnthropicOAuth()`),
NO generic handler abstraction — the shared piece is credential REFRESH orchestration,
which the ref itself centralizes in `oauthCredentialManager.js` (so a Go
`credentials.go` port is ref-faithful, not an invented abstraction).

## Ref behavior to port

- **shouldRefresh** (`oauthCredentialManager.js:43-56`): refresh when
  `expiresAt - now < refreshLead(provider)` (lead from `tokenRefresh.js getRefreshLeadMs`
  — read it; port the leads for anthropic/gemini/xai only). Codex staleness clause
  (:51-53) is Stage-2 — do NOT port.
- **Refresh lock** (`:129-142` `withCredentialRefreshLock`): one in-flight refresh per
  provider connection; concurrent callers await the same result. Go: per-connection
  `singleflight` or mutex map — must pass `-race`.
- **mergeRefreshedCredentials** (`:66-127`): refreshed token fields overwrite, but a
  missing/empty new refreshToken PRESERVES the old one (the preserve-on-empty rule is
  the ref's own logic at `oauthCredentialManager.js:66-127` — no PR row involved); `providerSpecificData` shallow-merged
  (`mergeProviderSpecificData` :58-64).
- **Per-provider configs**: `GeminiOAuth()` — Google endpoints, clientId+clientSecret
  from `providers.js:58-62`, refresh via `default.js:248-258 refreshGoogle` (form-encoded
  refresh_token grant). `XaiOAuth()` — clientId `providers.js:277`, token/refresh URL
  :278-279, authorize via `xai.js:103-130` (scope spaces %20), refresh `xai.js:160-180`.
  Both as constructors beside `AnthropicOAuth()` (`oauth.go:37`), reusing `OAuthFlow`
  (Start/Exchange/Refresh already generic, `oauth.go:68-174`). xai endpoint discovery
  (`xai.js:52-80`) is NOT ported — static config (the discovered values are the static
  ones; record as a comment).
- **PR-1249 redirect URI**: the redirect URI must derive from the request's
  origin/host when the dashboard is accessed remotely instead of hardcoding localhost
  — port the PR's rule into the Start handler (`internal/admin/oauth.go`): explicit
  override settings key, else request origin.
- **Key resolution into adapters**: new `internal/auth/credentials.go`
  `CredentialResolver` — `ResolveKey(providerID) (schemas.Key, map[string]string, error)`:
  find the provider's Connection (`connections.go:54-84`), decrypt, `shouldRefresh` →
  refresh-with-lock via the provider's `OAuthFlow.Refresh` + merge + persist
  (`UpdateConnection`), return `Key{Provider, Value: accessToken-or-apiKey}` +
  providerSpecificData. Router integration: add an OPTIONAL resolver to
  `internal/inference/router.go` (`SetKeyResolver`; nil → today's empty-Key behavior,
  preserving all w2-d tests). Ollama: `chatURL()` consumes the resolved
  `providerSpecificData["baseUrl"]` override via `catalog.ResolveOllamaHost(override)`
  (the w2-c deferral quoted above — chatURL gains the override parameter).

## Preconditions (a "0 hits" grep exits 1 = pass)

- `grep -c 'func AnthropicOAuth' internal/auth/oauth.go` ≥ 1; `grep -c 'func (f \*OAuthFlow) Refresh' internal/auth/oauth.go` ≥ 1 (reuse)
- `grep -rn 'GeminiOAuth\|XaiOAuth\|CredentialResolver' internal/` → 0 hits (new)
- `grep -c 'func (s \*Store) UpdateConnection' internal/store/connections.go` ≥ 1
- `grep -c 'func ResolveOllamaHost' internal/providers/catalog/catalog.go` ≥ 1

## Exclusive file ownership

NEW: `internal/auth/credentials.go` + `credentials_test.go`.
TOUCH: `internal/auth/oauth.go` (+`oauth_test.go` if present, else tests in
`auth_test.go`): add `GeminiOAuth()`/`XaiOAuth()` + refresh-lead table;
`internal/admin/oauth.go` (+ its test): PR-1249 redirect-URI rule + start/callback
for the two new providers; `internal/inference/router.go` + `router_test.go`
(SetKeyResolver only); `internal/providers/ollama/chat.go` + `chat_test.go`
(override param threading only).
NOT touched: generic adapter chat path (it receives a fresher Key transparently);
dashboard guard/login (track 1); any Stage-2 provider.

## Tasks (each: STEP (a) named failing tests first; STEP (b) implement)

1. **Provider configs + leads** (`oauth.go`): `GeminiOAuth()`, `XaiOAuth()` constructors
   (exact IDs/URLs from the cited providers.js lines; gemini carries clientSecret —
   OAuthFlow gains optional ClientSecret field included in token requests when set);
   `refreshLead(provider) time.Duration` table (values from `tokenRefresh.js`).
   Tests: `TestGeminiOAuthConfig`, `TestXaiOAuthConfig` (exact endpoint/ID values),
   `TestRefreshLeadTable`.

2. **CredentialResolver** (`credentials.go`): shouldRefresh (expiry-lead rule),
   single-flight refresh lock, merge rule (empty new RT preserves old; psd shallow-merge),
   persist via UpdateConnection, ResolveKey returning Key+psd.
   Tests (fake store + fake token endpoint via httptest): `TestShouldRefreshLeadWindow`,
   `TestRefreshSingleFlight` (N concurrent ResolveKey → exactly 1 token request; -race),
   `TestMergePreservesRefreshTokenWhenEmpty`, `TestMergeProviderSpecificData`,
   `TestResolveKeyNoConnection` (clean error), `TestResolveKeyPersistsRefreshed`.

3. **Start/callback + PR-1249** (`admin/oauth.go`): routes for gemini/xai start+callback
   (mirror the anthropic handlers `oauth.go:34-87`); redirect URI = settings override
   else request origin (PR-1249). Tests: `TestOAuthStartGemini`, `TestOAuthStartXai`,
   `TestRedirectURIFromRequestOrigin`, `TestRedirectURISettingsOverride`.

4. **Router + ollama wiring**: `SetKeyResolver` on Router (Resolve consults it when
   non-nil; w2-d tests stay green with nil); ollama chatURL override threading.
   Tests: `TestRouterUsesKeyResolver`, `TestRouterNilResolverUnchanged`,
   `TestOllamaHostOverrideFromCredentials`.

## Binary acceptance criteria

- `go test ./...` exits 0; `go vet ./...` exits 0; `go test -race ./internal/auth/ -run 'TestRefreshSingleFlight' -count=1` exits 0.
- `grep -c 'func GeminiOAuth\|func XaiOAuth' internal/auth/oauth.go` = 2.
- `grep -rn 'codex\|Codex' internal/auth/credentials.go` → 0 hits (Stage-2 clause excluded).
- `grep -rn 'func init(\|panic(' internal/auth/credentials.go` → 0 hits.
- `TestRefreshSingleFlight`, `TestMergePreservesRefreshTokenWhenEmpty`, `TestRouterNilResolverUnchanged`, `TestOllamaHostOverrideFromCredentials` pass.
- All pre-existing w2-d router tests pass unchanged.

## Out of scope

The ~11 Stage-2 provider OAuth handlers + PR-717/641/1388/1458/1004/665. Codex
staleness rule. xai endpoint discovery (static config). Device-code flows (none of
the 3 Stage-1 providers use one). Dashboard OIDC (w3-c). UI connect buttons (Wave 6).
