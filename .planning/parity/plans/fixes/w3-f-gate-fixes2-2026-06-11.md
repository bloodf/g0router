# Fix — w3-f diff-gate round 2 (2026-06-11): resolver race + no-connection test

Author: Fable 5. Implementer: kimi. 2 real items; env-override + OAuth-test-attribution rebutted.

## Task 1 — Router.SetKeyResolver race (MAJOR, real)
`internal/inference/router.go`: `SetKeyResolver` writes `r.keyResolver` while `Resolve`
reads it on the request path — unsynchronized. The Router already has `r.mu sync.RWMutex`
(added in w2-d for the provider cache). Guard `keyResolver` with it: SetKeyResolver
takes `r.mu.Lock()`; the Resolve read takes `r.mu.RLock()` (or snapshot under the
existing provider-cache lock). Add `TestSetKeyResolverConcurrent` (-race: concurrent
SetKeyResolver + Resolve, no race, no panic).

## Task 2 — TestResolveKeyNoConnection tests the right path (MAJOR, real)
`internal/auth/credentials_test.go`: the test must exercise "known provider, NO stored
connection" → ResolveKey returns a clean not-found error. Currently it does not set up
that scenario. Fix the test to use a store with no connection row for the provider and
assert the specific error.

## Rebuttals (no change)
- GeminiOAuth env override: existing AnthropicOAuth pattern (`oauth.go:37-42`) +
  authorized by `fixes/w3-f-secret-literal-fix.md` (GitHub push protection forbids the
  raw literal). The no-env default IS the byte-exact provider.js value, asserted by
  `TestGeminiOAuthConfig` (now byte-exact via split-parts, verified in-tree).
- Gemini/xAI start/callback tests "missing": TestOAuthStartGemini/Xai/Redirect* exist
  in `internal/admin/admin_test.go` (co-located, committed under the w3-a commit); they
  pass. Attribution artifact, not absence.

## Acceptance (binary)
- `go test ./... && go vet ./... && go test -race ./internal/auth/ ./internal/inference/` green.
- `TestSetKeyResolverConcurrent` passes under -race; `grep -c 'keyResolver' internal/inference/router.go` shows lock-guarded access.
- `TestResolveKeyNoConnection` sets up a no-connection provider and asserts the not-found error.
- Files ONLY: internal/inference/router.go, router_test.go, internal/auth/credentials_test.go. Do NOT git commit.
