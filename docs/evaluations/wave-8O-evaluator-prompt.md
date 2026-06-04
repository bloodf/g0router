# g0router Wave 8.O Evaluation

Evaluate completed wave `8.O` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files or commit.

Review:
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PROVIDERS.md`
- `internal/providers/types.go`
- `internal/providers/openaicompat/registry.go`
- `internal/cli/provider_runtime.go`
- `internal/provider/matrix.go`
- `internal/modelcatalog/catalog.go`
- Provider/API/CLI/model catalog tests touched by commit `d14b736`

Diff/commit:
- `d14b736 phase-8/task-providers: add gateway adapter coverage`

Check:
- Vercel AI Gateway is a real public OpenAI-compatible provider with default base URL `https://ai-gateway.vercel.sh/v1`, a catalog-backed direct-dispatch model, CLI/API visibility, and tests.
- LiteLLM, vLLM, and LM Studio are registered OpenAI-compatible adapters with real default base URLs, but are not advertised as public direct-dispatch providers because their model IDs are instance-defined.
- No unsupported provider is made to look usable without an adapter path, auth type, and tests.
- Provider matrix, provider docs, CLI list, and API provider responses agree.
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

Whether `docs/WORKFLOW.md` accurately reflects Wave 8.O completion and evaluation-pending state.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
