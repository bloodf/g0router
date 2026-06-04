# g0router Wave 8.BE Evaluation

Evaluate completed wave `8.BE` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files, do not commit, and do not clean protected local files.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/SCHEMA.md`
- `api/server_integration_test.go`
- `api/server.go`

Check:
- Real-server integration coverage exercises authenticated server middleware, not direct handlers, for usage, logs, quota, and provider OAuth routes.
- Usage/log tests seed real SQLite request logs and verify `/api/usage`, `/api/logs`, and `/api/usage/summary` through `NewServer`.
- Quota tests verify `/api/usage/quota/{provider}` uses active stored provider credentials with a fake quota fetcher.
- Provider OAuth tests verify `/api/oauth/{provider}/authorize`, `/api/oauth/{provider}/poll`, `/api/oauth/callback`, and `/api/oauth/{provider}/exchange` through stored session state, with redacted API responses.
- No external network, mocks, secret leakage, `init()` functions, mutable globals, or unrelated refactors were added.
- Workflow status accurately records Wave 8.BD evaluator PASS and Wave 8.BE gate evidence.

Run:
- `go test ./api -run TestIntegrationUsageQuotaLogsAndProviderOAuthThroughAuthenticatedServer -count=1`
- `go test ./api -run TestIntegration -count=1`
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
