# g0router Wave 8.BD Evaluation

Evaluate completed wave `8.BD` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files, do not commit, and do not clean protected local files.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PROVIDERS.md`
- `api/handlers/oauth.go`
- `api/handlers/oauth_test.go`

Check:
- `OAuthPoll` does not serialize raw provider poll errors into API responses.
- Regression coverage proves access tokens, refresh tokens, callback codes, and PKCE verifier material from poll errors are not rendered.
- Successful OAuth poll behavior from Wave 8.BC still works: stored verifier restoration, pending sessions not consumed, completed polls persisted, and account labels preserved.
- No unrelated files or refactors were added.
- No mocks, `init()` functions, mutable globals, or secret leakage were added.
- Workflow status accurately records the Wave 8.BC evaluator failure and Wave 8.BD remediation.

Run:
- `go test ./api/handlers -run 'TestOAuthPoll|TestOAuthHandlers' -count=1`
- `go test ./internal/store -run TestOAuthSessionCanBeReadBeforeSingleUseConsume -count=1`
- `go test ./api/handlers -run 'TestOAuthPollUsesStoredVerifierAndAccountLabel|TestOAuthPollUsesSessionFromQuery|TestOAuthPollAcceptsGitHubAlias' -count=1`
- `go test ./... -count=1`
- `go vet ./...`
- `go build ./cmd/g0router`
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

Whether workflow/docs status is accurate.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
