# g0router Wave 8.AH Evaluation

Evaluate completed wave `8.AH` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PLAN.md`
- `api/server_integration_test.go`
- Commit `1623081` and the workflow/docs commit that records Wave 8.AH.

Start read-only. Do not edit files.

Check:
- The real-server integration suite covers authenticated provider connection create, test, list, update, and delete through `/api/connections`.
- Tests use the real API server, authenticated control-plane middleware, and temp SQLite store.
- Tests prove `access_token`, `refresh_token`, `api_key`, `Authorization`, nested API keys, and nested token values are persisted when submitted but redacted from management API responses.
- Provider ID canonicalization through the real server is covered without production test shortcuts.
- No production handlers were weakened to satisfy tests.
- Wave 8.AH is accurately recorded in `docs/WORKFLOW.md`.
- `docs/PLAN.md` and `docs/ORCHESTRATION.md` align Stage 8 through Wave 8.AH.
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

Whether `docs/WORKFLOW.md`, `docs/PLAN.md`, and `docs/ORCHESTRATION.md` are accurate for Wave 8.AH.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
