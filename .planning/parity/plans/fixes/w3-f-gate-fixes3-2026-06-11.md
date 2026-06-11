# Fix — w3-f diff-gate round 3 (2026-06-11): real-resolver integration test

Author: Fable 5. Implementer: kimi. ONE real item; admin_test attribution closed by decision.

## Task — Router + real CredentialResolver integration test (MAJOR, real & valuable)
The resolver path is currently proven only with a fake resolver. Add an integration
test wiring the REAL `auth.CredentialResolver` (backed by a real `store.Store` with a
seeded provider connection) into the `Router` via `SetKeyResolver`, then assert
`Resolve(model)` returns a `schemas.Key` whose `Value` is the connection's
access token (and `ProviderSpecificData` carries any metadata). Cover the refresh path
too if a near-expiry connection triggers it (httptest token endpoint). New file
`internal/auth/credentials_integration_test.go` OR extend `credentials_test.go`
(whichever keeps the store+router wiring clean) — must NOT touch internal/admin.

## Closed by decision (no code change)
- admin_test.go OAuth tests (TestOAuthStartGemini/Xai/RedirectURIFromRequestOrigin/
  SettingsOverride): they EXIST and pass; they were committed into the shared
  internal/admin/admin_test.go under the w3-a commit (2d92a43), which predates w3-f's
  diff base — so no w3-f diff slice can show them. Verified passing once the admin
  package is stable (after w3-c merges). This is a commit-attribution artifact of two
  plans sharing one test file, not absent coverage.
- INITIAL_PASSWORD test in admin_test.go is w3-a's (env-override handler case), not w3-f.

## Acceptance (binary)
- `go test ./... && go vet ./... && go test -race ./internal/auth/ ./internal/inference/` green.
- A test constructs a real CredentialResolver + store-backed connection + Router and
  asserts Resolve returns the token-bearing Key.
- Files ONLY: internal/auth/credentials_test.go (or new credentials_integration_test.go), internal/inference/router_test.go if needed. Do NOT touch internal/admin. Do NOT git commit.
