# g0router Wave 8.W Evaluation

Evaluate completed wave `8.W` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files or commit.

Review:
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/phases/phase-10-dashboard-ui.md`
- `ui/src/App.tsx`
- `ui/src/App.test.tsx`
- `ui/src/api.ts`
- `ui/src/pages/ModelsPage.tsx`
- `ui/src/pages/ModelsPage.test.tsx`
- `ui/e2e/dashboard.e2e.ts`

Diff/commit:
- Wave 8.W commit after `7d6909c phase-8/task-workflow: record mcp agent dispatch commit`

Check:
- The dashboard exposes a distinct Models page in desktop and mobile navigation.
- The Models page uses real API client helpers for `/api/providers` and `/api/providers/{provider}/models`; it must not contain static provider/model fixtures.
- Provider switching fetches the selected provider model endpoint.
- Loading, empty, error/auth-expired, and successful table states are covered.
- Playwright mocked API coverage includes the Models page in normal, empty, and auth-expired dashboard journeys.
- The page does not display credentials or provider secrets.
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

Whether `docs/WORKFLOW.md` accurately reflects Wave 8.W completion and evaluation-pending state.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
