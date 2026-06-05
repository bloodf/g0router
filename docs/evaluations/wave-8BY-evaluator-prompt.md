# g0router Wave 8.BY Evaluation

Evaluate completed wave `8.BY` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/SCHEMA.md`
- `api/handlers/inference.go`
- `api/handlers/inference_test.go`
- Relevant commits for Wave 8.BY

Check:
- `/v1/messages` with `stream:true` no longer returns the old hard-coded 501 when the request is compatible with the existing OpenAI-style chat request shape.
- The handler dispatches through `InferenceEngine.DispatchStream` with the request context.
- The stream emits Anthropic Messages SSE events, including `message_start`, `content_block_start`, `content_block_delta`, `content_block_stop`, `message_delta`, and `message_stop`.
- The Messages stream does not append OpenAI's `[DONE]` sentinel.
- Stop reason mapping is compatible with Anthropic Messages streaming (`stop` to `end_turn`, `length` to `max_tokens`, `tool_calls` to `tool_use`).
- Existing explicit rejection for native Anthropic tool definitions, tool choices, `tool_use`, and `tool_result` input blocks still happens before dispatch.
- Existing `/v1/chat/completions`, `/v1/responses`, and non-streaming `/v1/messages` behavior still works.
- Stream errors remain sanitized and do not leak provider credentials or upstream detail.
- Workflow status is accurate.

Run gates:

```bash
go test ./api/handlers -run 'TestMessagesStreamingTranslatesChatStream|TestMessagesResponsePreservesToolUseBlocks|TestResponsesStreamingTranslatesChatStream|TestStreamInferenceWritesSanitizedStreamError' -count=1
go test ./... -count=1
go vet ./...
go build ./cmd/g0router
npm --prefix ui test -- --run
npm --prefix ui run build
npm --prefix ui run e2e
make build
git diff --check
```

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
