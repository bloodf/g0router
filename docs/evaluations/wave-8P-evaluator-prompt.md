# g0router Wave 8.P Evaluation

Evaluate completed wave `8.P` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files or commit.

Review:
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PROVIDERS.md`
- `internal/provider/matrix.go`
- `internal/modelcatalog/catalog.go`
- `api/handlers/providers_test.go`
- `internal/cli/providers_test.go`
- `internal/cli/root_test.go`
- `internal/provider/matrix_test.go`
- `internal/modelcatalog/pricing_test.go`

Diff/commit:
- `d079d50 phase-8/task-providers: promote nvidia direct routing`

Check:
- NVIDIA is public-routable only because there is both a registered OpenAI-compatible adapter and a catalog-backed model route.
- `docs/PROVIDERS.md`, provider matrix, `/api/providers`, and `g0router providers list` agree that `nvidia` is supported public direct dispatch.
- NVIDIA does not claim quota support without a real quota fetcher.
- The catalog model is not a placeholder-only provider advertisement; it is a real NVIDIA NIM/API catalog model.
- No unrelated providers were promoted or downgraded by this wave.
- No provider token, API key, or leaked MiniMax credential appears in source, docs, tests, logs, or command output.
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

Whether `docs/WORKFLOW.md` accurately reflects Wave 8.P completion and evaluation-pending state.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
