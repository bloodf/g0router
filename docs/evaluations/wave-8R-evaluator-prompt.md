# g0router Wave 8.R Evaluation

Evaluate completed wave `8.R` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files or commit.

Review:
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/PROVIDERS.md`
- `docs/CONFIG.md`
- `internal/modelcatalog/catalog.go`
- `internal/modelcatalog/pricing_test.go`
- `internal/proxy/engine.go`
- `internal/proxy/engine_test.go`
- `api/server.go`
- `api/server_test.go`

Diff/commit:
- Wave 8.R commit after `75a7222 phase-8/task-workflow: record vertex evaluator pass`

Check:
- Unqualified `gemini-2.5-flash` still resolves to the Gemini provider.
- Provider-qualified `vertex/gemini-2.5-flash` resolves to the Vertex provider.
- Vertex dispatch rewrites the upstream provider request model to `gemini-2.5-flash` without mutating the original request.
- `/v1` request logging preserves the public `vertex/gemini-*` model when a Vertex response reports upstream `gemini-*`, so catalog cost lookup works.
- `docs/CONFIG.md`, `docs/PROVIDERS.md`, and `docs/WORKFLOW.md` no longer imply that unqualified Gemini model IDs directly route to Vertex.
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

Whether `docs/WORKFLOW.md` accurately reflects Wave 8.R completion and evaluation-pending state.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
