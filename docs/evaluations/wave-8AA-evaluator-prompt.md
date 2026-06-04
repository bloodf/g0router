# g0router Wave 8.AA Evaluation

Evaluate completed wave `8.AA` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files or commit.

Review:
- `docs/PLAN.md`
- `docs/ORCHESTRATION.md`
- `docs/WORKFLOW.md`
- `ui/src/App.tsx`
- `ui/src/App.test.tsx`
- `ui/src/pages/ConnectionsAuthPage.tsx`
- `ui/src/pages/ConnectionsAuthPage.test.tsx`
- `ui/src/pages/ProvidersPage.tsx`
- `ui/e2e/dashboard.e2e.ts`

Diff/commit:
- Wave 8.AA commit after `c938513 phase-8/task-workflow: record evaluator remediation commit`

Check:
- The dashboard has a dedicated `Connections/Auth` route and navigation item.
- The route uses the real `/api/providers` and `/api/connections` API contracts through shared connection-control code, not production fixtures.
- The dedicated Connections/Auth page shows provider account rows and connection actions without rendering full provider credentials or provider contract table details.
- The Providers page still shows provider contract details and connection rows.
- Playwright E2E covers normal navigation and connection create/test/delete through the dedicated Connections/Auth route on desktop and mobile.
- `docs/PLAN.md` and `docs/ORCHESTRATION.md` are aligned to Stage 8 running through Wave `8.AA`.
- `docs/WORKFLOW.md` accurately records Wave 8.AA completion and evaluation-pending state.
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

Whether `docs/WORKFLOW.md` accurately reflects Wave 8.AA completion and evaluation-pending state.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
