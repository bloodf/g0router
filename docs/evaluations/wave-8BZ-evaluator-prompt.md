# g0router Wave 8.BZ Evaluation

Evaluate completed wave `8.BZ` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PROVIDERS.md`
- `internal/providers/types.go`
- `internal/providers/bedrock/bedrock.go`
- `internal/providers/bedrock/bedrock_test.go`
- `internal/providers/replicate/replicate.go`
- `internal/providers/replicate/replicate_test.go`
- `internal/proxy/errors.go`
- `api/handlers/inference_test.go`
- Relevant commits for Wave 8.BZ

Check:
- Bedrock and Replicate remain explicitly non-streaming in provider matrix/docs unless native streaming is actually implemented.
- Bedrock and Replicate stream methods wrap a shared provider-level unsupported-streaming sentinel.
- `proxy.ClassifyDispatchError` maps that shared sentinel to `501`, message `streaming unsupported for provider`, type `invalid_request_error`, code `streaming_unsupported`.
- The API response does not leak provider-specific error strings or upstream detail.
- Existing Gemini/Vertex unsupported-streaming classification still works.
- No provider is promoted to public streaming capability by this wave.
- Workflow status is accurate.

Run gates:

```bash
go test ./internal/providers/bedrock ./internal/providers/replicate ./api/handlers -run 'TestUnsupportedMethodsReturnErrors|TestChatCompletionStreamUnsupported|TestStreamInferenceUnsupportedProviderUsesStableError' -count=1
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
