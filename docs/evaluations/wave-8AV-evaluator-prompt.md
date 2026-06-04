# g0router Wave 8.AV Evaluation

Evaluate completed wave `8.AV` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PLAN.md`
- `docs/PROVIDERS.md`
- `internal/providers/vertex/vertex.go`
- `internal/providers/vertex/vertex_test.go`
- `internal/provider/matrix.go`
- `internal/provider/matrix_test.go`

Check:
- Vertex `ChatCompletionStream` no longer returns the unsupported-operation stub when project/location config is present.
- Streaming validates missing project/location before attempting an upstream request.
- Streaming uses the native Vertex `streamGenerateContent` endpoint with `alt=sse`.
- Streaming sends `Authorization: Bearer ...` and never puts credentials in the query.
- Streaming tests use a local HTTP server, not mocks or external network.
- SSE parsing maps Vertex text chunks to OpenAI-compatible stream deltas.
- Finish reasons and usage metadata are preserved in stream chunks.
- Malformed upstream SSE produces a stream error chunk instead of panicking.
- Non-streaming Vertex request and response behavior remains unchanged.
- `internal/provider/matrix.go` and `docs/PROVIDERS.md` now mark Vertex streaming as supported.
- Workflow, plan, and orchestration wave counts are accurate through Wave 8.AV.
- Changes are surgical and limited to Vertex streaming parity plus matching metadata.

Run gates:
- `go test ./internal/providers/vertex -run 'TestChatCompletionStreamMapsVertexSSEChunks|TestChatCompletionStreamMalformedSSEEmitsErrorChunk' -count=1`
- `go test ./internal/providers/vertex -count=1`
- `go test ./internal/provider -count=1`
- `go test ./... -count=1`
- `go vet ./...`
- `go build ./cmd/g0router`
- `git diff --check`
- `git status --short`

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

Whether `docs/WORKFLOW.md`, `docs/PLAN.md`, `docs/ORCHESTRATION.md`, and `docs/PROVIDERS.md` are accurate.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
