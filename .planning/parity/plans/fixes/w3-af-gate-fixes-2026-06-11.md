# Fix — w3-a + w3-f diff-gate findings (2026-06-11)

Author: Fable 5. Implementer: kimi. Real code items only; artifacts rebutted in re-gate notes.

## Rebuttals (no code change)
- **authMode vs auth_mode**: kimi's `auth_mode` is CORRECT — g0router settings keys are
  snake_case (verified: default_model, log_level, oauth_redirect_uri, oidc_client_id,
  oidc_client_secret, oidc_issuer_url, theme). The plan's `authMode` was a
  ref-camelCase mis-transcription; w3-a plan amended. No change.
- **OAuth admin tests "missing" (w3-f) / "out of scope" (w3-a)**: TestOAuthStartGemini/
  Xai/RedirectURIFromRequestOrigin/SettingsOverride EXIST and pass in admin_test.go;
  they are w3-f's, co-located with w3-a's login tests in one file. The w3-f diff
  attributed them to w3-a's commit (both staged the shared file). Re-gate w3-f with
  admin_test.go included. No change.
- **Gemini env overrides "out of scope"**: env override follows the EXISTING
  AnthropicOAuth pattern (oauth.go:37-42 `G0ROUTER_ANTHROPIC_CLIENT_ID`) and is
  authorized by fixes/w3-f-secret-literal-fix.md. Determinism is proven by a no-env
  default test (added below). No removal.

## Task 1 — credentials.go: stop swallowing JSON errors (MAJOR, real; errors-are-values)
`internal/auth/credentials.go:73,86,173,177` use `_ = json.Unmarshal(...)`. Replace:
parse provider-specific-data ONLY when `conn.Metadata`/`current.Metadata` is non-empty;
on a non-empty-but-invalid payload return `fmt.Errorf("parse provider metadata: %w", err)`
(propagated from ResolveKey/merge). Empty metadata → empty psd, no error. Add
`TestResolveKeyInvalidMetadataErrors` (non-empty bad JSON → wrapped error) and
`TestResolveKeyEmptyMetadataOK`.

## Task 2 — strengthen TestGeminiOAuthConfig within push-protection limits (MAJOR, real)
`internal/auth/oauth_test.go`: assert the DEFAULT (no env) GeminiOAuth() ClientID and
ClientSecret EXACTLY by comparing against the same split-parts expression used in
oauth.go (no scanner-matching literal), PLUS exact length + prefix + suffix. Same for
XaiOAuth (its xai values are not GOCSPX/google-id patterns — assert byte-exact directly
per providers.js:277-279). Add `TestGeminiOAuthNoEnvDeterministic` (unset env → the
fixed default).

## Task 3 — w3-a cleanups (MAJOR+MINOR, real)
- `internal/auth/limiter_test.go`: remove the dead/unused `advance` helper (no callers).
- `cmd/g0router/main_test.go`: assert the reset subcommand prints "Password reset to default."

## Acceptance (binary)
- `go test ./... && go vet ./...` green; `go test -race ./internal/auth/` green.
- `git grep -n '_ = json.Unmarshal' internal/auth/credentials.go` → 0.
- `grep -c 'func advance' internal/auth/limiter_test.go` → 0.
- `TestResolveKeyInvalidMetadataErrors`, `TestGeminiOAuthNoEnvDeterministic`, reset-output assertion pass.
- Files touched ONLY: internal/auth/credentials.go, internal/auth/credentials_test.go, internal/auth/oauth_test.go, internal/auth/limiter_test.go, cmd/g0router/main_test.go. Do NOT git commit.
