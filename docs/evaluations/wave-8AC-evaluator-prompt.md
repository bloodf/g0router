# g0router Wave 8.AC Evaluation

Evaluate completed wave `8.AC` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files or commit.

Review:
- `docs/PLAN.md`
- `docs/ORCHESTRATION.md`
- `docs/WORKFLOW.md`
- `ui/src/App.tsx`
- `ui/src/App.test.tsx`
- `ui/src/pages/SettingsPage.tsx`
- `ui/src/pages/SettingsSecurityPage.tsx`
- `ui/src/pages/SettingsSecurityPage.test.tsx`
- `ui/e2e/dashboard.e2e.ts`

Diff/commit:
- Wave 8.AC commit after `93de90e phase-8/task-workflow: record dashboard mcp split commit`

Check:
- The dashboard has a dedicated `Settings/Security` route and navigation item.
- The new route reuses the real `/api/settings` contract through existing settings helpers, not production fixtures.
- The route exposes control-plane protection and request logging controls.
- The existing `Settings` route remains intact.
- Playwright E2E covers normal, empty, auth-expired, save success, and save failure states for the new route.
- `docs/PLAN.md` and `docs/ORCHESTRATION.md` are aligned to Stage 8 running through Wave `8.AC`.
- `docs/WORKFLOW.md` accurately records Wave 8.AC completion and evaluation-pending state.
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

Whether `docs/WORKFLOW.md` accurately reflects Wave 8.AC completion and evaluation-pending state.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
