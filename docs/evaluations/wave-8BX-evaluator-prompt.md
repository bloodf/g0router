# g0router Wave 8.BX Evaluation

Evaluate completed wave `8.BX` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/SCHEMA.md`
- `api/handlers/inference.go`
- `api/handlers/inference_test.go`
- Diff/commits for Wave 8.BX

Check:
- `/v1/responses` no longer rejects `stream:true` before dispatch when the request translates to OpenAI chat.
- Responses streaming uses the existing request context and `DispatchStream`.
- The SSE response emits Responses-style events for output text delta, output text done, completion, and `[DONE]`.
- Stream errors are sanitized and do not leak upstream provider details.
- Existing `/v1/chat/completions`, non-streaming `/v1/responses`, and explicit unsupported native Responses input rejection still work.
- Docs/workflow accurately describe that `/v1/responses` streaming is implemented only through chat stream translation; `/v1/messages` streaming remains a separate gap.

Required gates:
- `go test ./api/handlers -run 'TestResponsesStreamingTranslatesChatStream|Test(StreamInference|Responses|Inference)' -count=1`
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

Whether `docs/WORKFLOW.md` is accurate.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
