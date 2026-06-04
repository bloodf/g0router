# g0router Wave 8.AJ Evaluation

Evaluate completed wave `8.AJ` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PLAN.md`
- `internal/mcp/oauth.go`
- `internal/mcp/oauth_test.go`
- `api/server_integration_test.go`
- Commit `3b8aa81` and the workflow/docs commit that records Wave 8.AJ.

Start read-only. Do not edit files.

Check:
- MCP OAuth completion prefers an instance-selected account label over a token endpoint `account_label`.
- Token-derived fallback labels still work when no selected instance account label exists.
- The authenticated server integration test expects the selected instance label when the token endpoint returns a conflicting label.
- Token email, resource URI, redaction, PKCE verifier, and runtime reapply assertions remain intact.
- Changes are surgical and limited to the OAuth label precedence behavior and matching test contract.
- No credentials are serialized in MCP instance/account API responses.
- Wave 8.AJ is accurately recorded in `docs/WORKFLOW.md`.
- `docs/PLAN.md` and `docs/ORCHESTRATION.md` align Stage 8 through Wave 8.AJ.
- Gates pass:
  - `go test ./internal/mcp -count=1`
  - `go test ./api -run TestIntegrationMCPInstanceOAuthRoundTripThroughAuthenticatedServer -count=1`
  - `go test ./... -count=1`
  - `go vet ./...`
  - `go build ./cmd/g0router`
  - `npm --prefix ui test -- --run`
  - `npm --prefix ui run build`
  - `npm --prefix ui run e2e`
  - `make build`

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

Whether `docs/WORKFLOW.md`, `docs/PLAN.md`, and `docs/ORCHESTRATION.md` are accurate for Wave 8.AJ.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
