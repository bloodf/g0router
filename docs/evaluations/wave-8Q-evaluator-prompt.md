# g0router Wave 8.Q Evaluation

Evaluate completed wave `8.Q` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files or commit.

Review:
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/CONFIG.md`
- `docs/PROVIDERS.md`
- `.env.example`
- `internal/cli/provider_runtime.go`
- `internal/provider/matrix.go`
- `internal/providers/vertex/vertex.go`
- `internal/modelcatalog/catalog.go`
- Vertex/provider/API/CLI tests touched by commit `1891a0c`

Diff/commit:
- `1891a0c phase-8/task-providers: promote vertex direct routing`

Check:
- Vertex is public-routable only because there is both a registered native adapter and catalog-backed Gemini model routes.
- Vertex runtime configuration reads `VERTEX_PROJECT_ID` and `VERTEX_LOCATION`, and missing configuration fails before any upstream network call with a clear non-secret error.
- `docs/CONFIG.md`, `.env.example`, `docs/PROVIDERS.md`, provider matrix, `/api/providers`, and `g0router providers list` agree on Vertex support and configuration requirements.
- Vertex does not claim streaming or quota support until those paths are implemented.
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

Whether `docs/WORKFLOW.md` accurately reflects Wave 8.Q completion and evaluation-pending state.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
