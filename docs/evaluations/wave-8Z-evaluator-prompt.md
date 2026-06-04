# g0router Wave 8.Z Evaluation

Evaluate completed wave `8.Z` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files or commit.

Review:
- `docs/PLAN.md`
- `docs/ORCHESTRATION.md`
- `docs/WORKFLOW.md`
- `ui/src/pages/ModelsPage.test.tsx`

Diff/commit:
- Wave 8.Z commit after `6a06d2d phase-8/task-workflow: record dashboard api keys commit`

Check:
- The Wave 8.W evaluator finding is fixed: `ModelsPage.test.tsx` now covers loading, selected-provider empty, non-auth API error, auth-expired, and successful model table states.
- The Wave 8.X evaluator finding is fixed: `docs/PLAN.md` and `docs/ORCHESTRATION.md` no longer say Stage 8 ends at Waves `8.A-8.N` or has only `14` Stage 8 waves / `40` total waves.
- `docs/WORKFLOW.md` truthfully records the 8.W and 8.X evaluator failures and points to remediation in Wave 8.Z.
- `docs/WORKFLOW.md` accurately records Wave 8.Z completion and evaluation-pending state.
- No production behavior changed for this remediation wave.
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

Whether `docs/WORKFLOW.md` accurately reflects 8.W/8.X failures remediated by 8.Z and Wave 8.Z evaluation-pending state.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
