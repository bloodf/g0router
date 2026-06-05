# g0router Wave 8.BV Evaluation

Evaluate completed wave `8.BV` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `internal/cli/root_test.go`
- Diff/commits for Wave 8.BV

Check:
- Wave 8.BU evaluator's non-blocking startup-registration test note is resolved.
- `TestDefaultInferenceEngineRegistersReplicateProvider` exists and verifies `newDefaultInferenceEngine` registers `providers.ProviderReplicate`.
- The wave changes only the CLI test and workflow/evaluator docs.
- No provider runtime behavior changed.

Required gates:
- `go test ./internal/cli -run TestDefaultInferenceEngineRegistersReplicateProvider -count=1`
- `go test ./... -count=1`
- `go vet ./...`
- `go build ./cmd/g0router`
- `npm --prefix ui test -- --run`
- `npm --prefix ui run build`
- `npm --prefix ui run e2e`
- `make build`
- `git diff --check`

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
