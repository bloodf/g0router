# g0router Wave 8.AF Evaluation

Evaluate completed wave `8.AF` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PLAN.md`
- `api/server_integration_test.go`
- Commits: `32c2131` and the workflow/docs commit that records Wave 8.AF.

Start read-only. Do not edit files.

Check:
- The real-server integration suite covers authenticated `/v1/messages` and `/v1/responses`.
- The tests start the real API server with temp storage and use a local fake upstream server, not mocks or external network.
- Assertions verify public response shapes and usage mapping for both routes.
- No production-only handlers, fixtures, or shortcuts were added to make the tests pass.
- Wave 8.AF is accurately recorded in `docs/WORKFLOW.md`.
- `docs/PLAN.md` and `docs/ORCHESTRATION.md` align Stage 8 through Wave 8.AF.
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

Whether `docs/WORKFLOW.md`, `docs/PLAN.md`, and `docs/ORCHESTRATION.md` are accurate for Wave 8.AF.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
