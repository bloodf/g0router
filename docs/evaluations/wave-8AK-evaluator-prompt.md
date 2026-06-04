# g0router Wave 8.AK Evaluation

Evaluate completed wave `8.AK` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

Review:
- CLAUDE.md
- AGENTS.md if present
- docs/README.md
- docs/WORKFLOW.md
- docs/ORCHESTRATION.md
- docs/phases/phase-12-advanced-mcp-gateway.md
- Commit refs: `e57c6d6`, merge commit `e751673`

Check:
- API and CLI MCP OAuth start flows accept optional `client_id` and `client_secret`.
- `client_id` is included in the returned authorization URL only when supplied.
- `client_id` and `client_secret` are persisted in the short-lived OAuth flow and consumed during callback completion.
- Token exchange posts `client_id` and `client_secret` only when supplied.
- No API response, CLI output, logs, or test failure messages expose `client_secret`.
- SQLite flow schema/select/insert/scan remain consistent.
- Changes are surgical and limited to the documented owned files plus this workflow/evaluator documentation.
- Workflow status is accurate for Wave 8.AK.

Run gates:
- `go test ./internal/mcp -run 'TestOAuthStartIncludesClientID|TestOAuthEnginePostsClientCredentialsWhenFlowProvidesThem' -count=1`
- `go test ./internal/store -run TestMCPOAuthFlow -count=1`
- `go test ./api/handlers -run TestMCPOAuthStart -count=1`
- `go test ./internal/cli -run TestMCPOAuthStartCommand -count=1`
- `go test ./... -count=1`
- `go vet ./...`
- `go build ./cmd/g0router`

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
