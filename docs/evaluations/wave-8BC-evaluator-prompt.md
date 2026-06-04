# g0router Wave 8.BC Evaluation

Evaluate completed wave `8.BC` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

Review:

- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PROVIDERS.md`
- `api/handlers/oauth.go`
- `api/handlers/oauth_test.go`
- `internal/store/oauthsessions.go`
- `internal/store/oauthsessions_test.go`
- `ui/src/api.ts`
- `ui/src/api.test.ts`
- `ui/src/pages/ProvidersPage.tsx`
- `ui/src/pages/ProvidersPage.test.tsx`
- `ui/e2e/dashboard.e2e.ts`
- Diff/commit for Wave 8.BC

Check:

- `/api/oauth/{provider}/poll` restores the stored PKCE verifier for sessions created by `/api/oauth/{provider}/authorize`.
- Pending polls do not consume the stored OAuth session.
- Complete polls consume the stored session, persist a redacted connection, and preserve the requested account label.
- Provider aliases such as `github` to `github-copilot` still work.
- The dashboard exposes a polling completion path for device-style provider OAuth sessions without removing callback exchange support.
- UI and API responses never render or serialize access tokens, refresh tokens, API keys, callback codes, or PKCE verifiers.
- Tests use fakes/local behavior and mocked Playwright API responses only in the test harness.
- No unrelated refactors, `init()` functions, mutable globals, or production test-only handlers were added.
- Workflow status and provider docs accurately describe the implemented scope and remaining Cursor inference gap.

Run gates:

- `go test ./internal/store -run TestOAuthSessionCanBeReadBeforeSingleUseConsume -count=1`
- `go test ./api/handlers -run 'TestOAuthPollUsesStoredVerifierAndAccountLabel|TestOAuthPollUsesSessionFromQuery|TestOAuthPollAcceptsGitHubAlias' -count=1`
- `npm --prefix ui test -- --run src/api.test.ts src/pages/ProvidersPage.test.tsx`
- `npm --prefix ui run e2e -- --grep 'provider OAuth'`
- `go test ./... -count=1`
- `go vet ./...`
- `go build ./cmd/g0router`
- `npm --prefix ui test -- --run`
- `npm --prefix ui run build`
- `npm --prefix ui run e2e`
- `make build`
- `git diff --check`
- `git status --short`

Return:

## Verdict

PASS or FAIL

## Blocking Findings

Issues that must be fixed before advancing.

## Non-Blocking Findings

Risks or cleanup notes.

## Gate Results

Command results with exact failures if any.

## Workflow Status Review

Whether `docs/WORKFLOW.md` is accurate.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
