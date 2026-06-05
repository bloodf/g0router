# g0router Wave 8.CF Evaluation

Evaluate completed wave `8.CF` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PLAN.md`
- `docs/phases/phase-05-oauth-flows-cli.md`
- Changed implementation/tests:
  - `api/handlers/oauth.go`
  - `api/handlers/oauth_test.go`
  - `api/handlers/providers_test.go`
  - `api/handlers/usage_test.go`
  - `internal/provider/matrix_test.go`

Scope:
- OAuth start errors must not leak token, verifier, authorization, or callback-code material.
- `/api/usage/quota/{provider}` raw JSON contract must be covered.
- `/api/providers/{provider}/models` dynamic provider behavior must be covered.
- Cursor OAuth phase docs must describe the OMP `loginDeepControl` polling behavior, not stale PKCE wording.
- Workflow, plan, and orchestration docs must agree that Stage 8 is tracked through Wave 8.CF.

Run gates:
- `go test ./api/handlers ./internal/provider -run 'TestUsageQuotaRawJSONContract|TestProvidersListModelsForDynamicProvider|TestOAuthStartDoesNotLeakFlowErrorSecrets|TestOAuthPhaseDocsDescribeCursorOMPFlow' -count=1`
- `make verify`
- `git diff --check`

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
