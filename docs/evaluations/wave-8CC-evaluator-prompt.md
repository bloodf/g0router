# g0router Wave 8.CC Evaluation

Evaluate completed wave `8.CC` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files, do not commit, and do not clean local dirt.

Review:

- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PLAN.md`
- `docs/PROVIDERS.md`
- `internal/provider/matrix.go`
- `internal/provider/matrix_test.go`
- Implementation commit: `fd80d7e phase-8/task-provider-docs: expose quota column`

Check:

- `docs/PROVIDERS.md` has an explicit `Quota` column in the provider matrix table.
- The docs quota column matches `internal/provider/matrix.go` for every provider.
- OpenRouter is the only provider documented with `Quota` = `yes`.
- The docs do not imply unsupported provider quota fetchers exist.
- Regression coverage fails without the docs quota column and passes after the update.
- The implementation remains docs/test-only and does not alter runtime provider behavior.
- The workflow, plan, orchestration docs, and evaluator prompt agree on Wave 8.CC.
- Gates pass:
  - `go test ./internal/provider -run TestProviderDocsExposeQuotaColumnMatchingMatrix -count=1`
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

Whether `docs/WORKFLOW.md` is accurate for Wave 8.CC.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
