# g0router Wave 8.AD Evaluation

Evaluate completed wave `8.AD` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files or commit.

Review:
- `docs/PLAN.md`
- `docs/ORCHESTRATION.md`
- `docs/WORKFLOW.md`
- `ui/src/App.tsx`
- `ui/src/App.test.tsx`
- `ui/e2e/dashboard.e2e.ts`
- `ui/dist/assets/index.js`

Diff/commit:
- Wave 8.AD implementation commit `e806fe8 phase-8/task-ui: align dashboard route names`
- Wave 8.AD workflow commit after implementation

Check:
- Dashboard navigation exposes `Endpoint Setup` for endpoint-copy/API-key setup.
- Dashboard navigation exposes `Combos/Routing` for combo route management.
- Existing page IDs/components remain wired to the same API-backed pages; this wave must not add fixtures or production test-only handlers.
- `App` unit coverage asserts the documented labels.
- Playwright E2E navigates by the documented labels across normal, mutation, empty, and auth-expired flows.
- The embedded UI asset is rebuilt consistently with the source change.
- `docs/PLAN.md` and `docs/ORCHESTRATION.md` are aligned to Stage 8 running through Wave `8.AD`.
- `docs/WORKFLOW.md` accurately records Wave 8.AD completion and evaluation-pending state.
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

Whether `docs/WORKFLOW.md` accurately reflects Wave 8.AD completion and evaluation-pending state.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
