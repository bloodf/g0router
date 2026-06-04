# g0router Wave 8.AN Evaluation

Evaluate completed wave `8.AN` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files, do not commit, and do not clean protected local files.

Review:
- CLAUDE.md
- AGENTS.md if present
- docs/README.md
- docs/WORKFLOW.md
- docs/ORCHESTRATION.md
- docs/phases/phase-09-mcp-gateway.md
- docs/phases/phase-12-advanced-mcp-gateway.md
- Commit refs: `1323d51`, merge commit `325b248`
- ui/src/pages/McpPage.tsx
- ui/src/pages/McpSplitPages.test.tsx
- ui/e2e/dashboard.e2e.ts

Check:
- The dashboard MCP OAuth start form can submit when `Authorization URL` is blank and `Resource URI` is present.
- The request body preserves `authorization_url: ""`, includes the selected `resource_uri`, and keeps the redirect URI.
- The dashboard validates that at least one of Authorization URL or Resource URI is present before POSTing.
- Browser `required` attributes do not block the Resource URI discovery path.
- Existing MCP OAuth start/complete flows, instance creation, tool execution, and credential redaction still work.
- Unit coverage proves both Resource URI discovery and the empty-field validation path.
- Playwright coverage proves the mocked dashboard Resource URI discovery flow on desktop and mobile.
- Changes are surgical and limited to the owned UI files plus workflow/evaluator documentation.
- Workflow status is accurate for Wave 8.AN.

Run gates:
- `npm --prefix ui test -- --run McpSplitPages --reporter=dot`
- `npm --prefix ui test -- --run`
- `npm --prefix ui run build`
- `npm --prefix ui run e2e`
- `go test ./api -run 'TestPublicRoutesBypassAuth|TestInferenceLoggingRecordsFailedRequestWhenEnabled' -count=100`
- `go test ./api -count=20 -shuffle=on`
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
