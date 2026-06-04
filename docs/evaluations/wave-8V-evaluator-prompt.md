# g0router Wave 8.V Evaluation

Evaluate completed wave `8.V` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files or commit.

Review:
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/phases/phase-09-mcp-gateway.md`
- `internal/proxy/engine.go`
- `internal/proxy/engine_test.go`
- `internal/mcp/agent.go`
- `internal/mcp/agent_test.go`
- `internal/cli/root.go`
- `internal/cli/root_test.go`
- `api/server.go`

Diff/commit:
- Wave 8.V commit after `3a0ed83 phase-8/task-workflow: record rtk caveman evaluator pass`

Check:
- Non-streaming proxy dispatch uses the existing `mcp.Agent` loop when configured with a registered MCP `ToolManager`.
- Provider tool calls are executed through `mcp.ToolManager.Call`, and tool results are appended back into the next provider request.
- Ordinary caller-provided tools that are not registered MCP tools are not forced through the MCP agent loop.
- `g0router serve` default server config registers the same MCP tool manager on the inference engine that the API uses for MCP management.
- Streaming dispatch is not falsely claimed to execute MCP tool loops.
- Existing RTK/caveman preprocessing and MCP tool injection still work at the API wrapper layer.
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

Whether `docs/WORKFLOW.md` accurately reflects Wave 8.V completion and evaluation-pending state.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
