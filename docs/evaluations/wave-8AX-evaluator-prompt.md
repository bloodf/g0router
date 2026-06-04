# g0router Wave 8.AX Evaluation

Evaluate completed wave `8.AX` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files, do not commit, and do not clean protected local files.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PLAN.md`
- `docs/PROVIDERS.md`
- `internal/providers/bedrock/bedrock.go`
- `internal/providers/bedrock/types.go`
- `internal/providers/bedrock/bedrock_test.go`
- `internal/provider/matrix.go`
- `internal/provider/matrix_test.go`
- `api/handlers/providers_test.go`
- `internal/proxy/engine_test.go`
- `internal/proxy/combo_test.go`

Important commit:
- Implementation commit: `{commit}`

Check:
- Bedrock non-streaming chat completions use `POST /model/{modelId}/converse`, not the old `/invoke` Anthropic-native payload.
- The Converse request maps text messages to `messages[].content[].text` and common inference settings to `inferenceConfig`.
- The Converse response maps `output.message.content[].text`, `stopReason`, and `usage` into OpenAI-compatible chat response fields.
- SigV4 signing and session token propagation still work.
- Bedrock remains `adapter_only`: explicit aliases and combo steps may use it, but public direct dispatch and catalog routing remain disabled.
- Bedrock still does not claim streaming or quota support.
- Tests use local HTTP servers/fakes, not external network or mocks.
- No secrets are logged or serialized.
- No `init()` functions, mutable globals, speculative abstractions, or unrelated refactors were added.

Run:
- `go test ./internal/providers/bedrock -run 'TestChatCompletionSignsConverseRequest|TestChatCompletionParsesBedrockResponse' -count=1`
- `go test ./internal/providers/bedrock -count=1`
- `go test ./internal/provider -count=1`
- `go test ./api/handlers -run TestProvidersListKnownProviders -count=1`
- `go test ./internal/proxy -run 'TestDispatchUsesBedrockAliasThroughAdapterOnlyInference|TestComboDispatchUsesBedrockAdapterOnlyStep' -count=1`
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
