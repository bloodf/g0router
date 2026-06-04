# g0router Wave 8.AU Evaluation

Evaluate completed wave `8.AU` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PLAN.md`
- `docs/PROVIDERS.md`
- `internal/providers/gemini/gemini.go`
- `internal/providers/gemini/gemini_test.go`
- `internal/provider/matrix.go`
- `internal/provider/matrix_test.go`

Check:
- Gemini `ChatCompletionStream` no longer returns the unsupported-operation stub.
- Streaming uses the native Gemini `streamGenerateContent` endpoint with `alt=sse`.
- API-key streaming sends `key=` in the query and does not send a bearer token.
- OAuth streaming sends `Authorization: Bearer ...` and does not put the token in the query.
- Streaming tests use a local HTTP server, not mocks or external network.
- SSE parsing maps Gemini text chunks to OpenAI-compatible stream deltas.
- SSE parsing maps Gemini `functionCall` parts to OpenAI-compatible tool-call deltas.
- Finish reasons and usage metadata are preserved in stream chunks.
- Malformed upstream SSE produces a stream error chunk instead of panicking.
- Non-streaming Gemini request and response behavior remains unchanged.
- `internal/provider/matrix.go` and `docs/PROVIDERS.md` now mark Gemini streaming as supported.
- Vertex remains explicitly non-streaming until its own implementation exists.
- Workflow, plan, and orchestration wave counts are accurate through Wave 8.AU.
- Changes are surgical and limited to Gemini streaming parity plus matching metadata.

Run gates:
- `go test ./internal/providers/gemini -run 'TestChatCompletionStreamMapsGeminiSSEChunks|TestChatCompletionStreamWithOAuthUsesBearerAndAltSSE' -count=1`
- `go test ./internal/providers/gemini -count=1`
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
