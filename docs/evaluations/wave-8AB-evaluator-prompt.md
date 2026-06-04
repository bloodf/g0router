# g0router Wave 8.AB Evaluation

Evaluate completed wave `8.AB` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files or commit.

Review:
- `docs/PLAN.md`
- `docs/ORCHESTRATION.md`
- `docs/WORKFLOW.md`
- `ui/src/App.tsx`
- `ui/src/App.test.tsx`
- `ui/src/pages/McpPage.tsx`
- `ui/src/pages/McpInstancesPage.tsx`
- `ui/src/pages/McpAccountsPage.tsx`
- `ui/src/pages/McpToolsPage.tsx`
- `ui/src/pages/McpSplitPages.test.tsx`
- `ui/e2e/dashboard.e2e.ts`

Diff/commit:
- Wave 8.AB commit after `9ec5041 phase-8/task-workflow: record dashboard connections auth commit`

Check:
- The dashboard has dedicated `MCP Instances`, `MCP Accounts`, and `MCP Tools` routes and navigation items.
- The existing combined `MCP` route remains intact.
- Split pages reuse the real MCP API contract through shared MCP page loading and do not introduce production fixtures.
- The Instances route shows create-instance controls and instance table without OAuth/tool execution controls.
- The Accounts route shows OAuth start/complete controls and account rows without the instance table or tool execution controls.
- The Tools route shows tool execution and discovered tools without instance table or OAuth controls.
- MCP credential redaction remains intact.
- Playwright E2E covers normal, empty, auth-expired, and mutation/action paths across the split routes on desktop and mobile.
- `docs/PLAN.md` and `docs/ORCHESTRATION.md` are aligned to Stage 8 running through Wave `8.AB`.
- `docs/WORKFLOW.md` accurately records Wave 8.AB completion and evaluation-pending state.
- Gates pass:
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

Whether `docs/WORKFLOW.md` accurately reflects Wave 8.AB completion and evaluation-pending state.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
