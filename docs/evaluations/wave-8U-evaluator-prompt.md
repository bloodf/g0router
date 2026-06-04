# g0router Wave 8.U Evaluation

Evaluate completed wave `8.U` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files or commit.

Review:
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/phases/phase-09-mcp-gateway.md`
- `api/server.go`
- `api/server_test.go`
- `api/handlers/inference.go`
- `api/handlers/mcp.go`
- `internal/mcp/toolmanager.go`
- `internal/providers/types.go`
- `internal/logging/requestlog.go`

Diff/commit:
- Wave 8.U commit after `c985c16 phase-8/task-workflow: record rtk caveman commit`

Check:
- `/v1/chat/completions`, `/v1/messages`, and `/v1/responses` attach registered MCP tools before dispatch when the caller supplies no tools.
- Caller-supplied tools are not overwritten by registered MCP tools.
- Tool injection uses the existing concurrency-safe `mcp.ToolManager`.
- RTK/caveman preprocessing still runs before dispatch.
- Request logging records route-accurate source formats: `openai`, `anthropic`, and `responses`.
- Target format continues to reflect the selected provider.
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

Whether `docs/WORKFLOW.md` accurately reflects Wave 8.U completion and evaluation-pending state.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
