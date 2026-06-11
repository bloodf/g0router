# Fix — w3-d diff-gate findings (2026-06-11). One SECURITY BLOCKER + 4 real.

Author: Fable 5. Implementer: kimi. Dispatch AFTER w3-c fix merges (shared internal/auth package).

## Task 1 — SECURITY: CLI token must NOT grant remote /v1 access (BLOCKER)
Ref guard order (`dashboardGuard.js:165-194`): LOCAL_ONLY → ALWAYS_PROTECTED (cli OR
session) → `isPublicLlmApi` → `canAccessPublicLlmApi` (loopback OR api-key — NO cli) →
/api deny-by-default (cli OR session). `internal/server/guard.go`: the public-LLM-API
step must accept ONLY loopback-or-APIKeyValidator; REMOVE CLI-token acceptance there.
CLI token remains valid for ALWAYS_PROTECTED + /api deny-by-default paths only.
Tests: fix `TestGuardCLIToken` to assert CLI token authorizes an /api ALWAYS_PROTECTED
(or deny-by-default) route, NOT /v1; add `TestGuardV1CLITokenRejectedRemote`
(remote /v1 + valid CLI token, no api-key → 401).

## Task 2 — ParseAPIKey enforces exact shapes (MAJOR)
`internal/auth/apikey.go` `ParseAPIKey`: new = `sk-{16hex}-{6 [a-z0-9]}-{8hex}`
(regex-validate each segment); legacy = `sk-{8 [a-z0-9]}` exactly. Reject anything
else (malformed `sk-*` must NOT parse). Tests for malformed inputs.

## Task 3 — store CreateAPIKey(name) generates the key (MAJOR)
`internal/store/apikeys.go`: `CreateAPIKey(name string)` GENERATES the formatted key
via `auth.GenerateAPIKey(MachineID(""))` (do not accept caller-supplied raw keys for
creation); returns the created row incl. the one-time full key. Update store tests to
use generated keys, not invalid literals.

## Task 4 — TestMigrationAdditive exercises re-run (MAJOR)
`internal/store/apikeys_test.go`: open a store, run migrations, insert a key, then
re-open/re-run migrations on the SAME DB and assert no error + the api_keys table and
row survive (additive `ensureColumn`/CREATE IF NOT EXISTS proven idempotent).

## Acceptance (binary)
- `go test ./... && go vet ./... && go test -race ./internal/server/` green.
- `TestGuardV1CLITokenRejectedRemote` passes; `TestGuardCLIToken` no longer asserts /v1 bypass.
- `grep -n 'cli\|CLI' internal/server/guard.go` shows CLI check is NOT in the public-LLM-API branch.
- ParseAPIKey rejects malformed sk-* (tests); CreateAPIKey(name) generates the key.
- TestMigrationAdditive re-runs migrations on an existing DB.
- Files ONLY: internal/server/guard.go, guard_test.go, internal/auth/apikey.go, apikey_test.go, internal/store/apikeys.go, apikeys_test.go. Do NOT git commit.
