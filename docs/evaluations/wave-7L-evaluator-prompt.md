# g0router Wave 7.L Evaluation Prompt

Evaluate completed Wave `7.L` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

## Review Inputs

- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/evaluations/wave-7K-evaluator-prompt.md`
- Wave 7.L commit:
  - `phase-7/task-l1: finalize mcp delete cleanup`

## Checks

- MCP instance DELETE first deletes the store row.
- If store deletion fails, runtime close is not attempted and the handler returns the store failure.
- If store deletion succeeds and runtime close fails, the handler returns `204 No Content` because there is no remaining store row to retry from.
- Successful delete still attempts runtime close and removes the store row.
- The behavior is covered by focused tests in `api/handlers/mcp_test.go`.
- `docs/WORKFLOW.md` records Wave 7.L as a final post-7.K cleanup with evaluation pending.
- Protected local dirt remains uncommitted: `.DS_Store`, `docs/.DS_Store`, `.pi/`, and untracked `AGENTS.md`.

## Required Gates

Run:

```bash
go test ./... -count=1
go vet ./...
go build ./cmd/g0router
npm --prefix ui test -- --run
npm --prefix ui run build
make build
```

## Return Format

## Verdict

PASS or FAIL

## Blocking Findings

Issues that must be fixed before declaring remediation complete.

## Non-Blocking Findings

Risks or cleanup notes.

## Gate Results

Command results with exact failures if any.

## Workflow Status Review

Whether `docs/WORKFLOW.md` accurately reflects Wave 7.L completion and evaluation-pending state.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
