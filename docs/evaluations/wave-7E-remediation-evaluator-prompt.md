# g0router Wave 7.E Remediation Evaluation Prompt

Evaluate the Wave `7.E` evaluator remediation in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

## Review Inputs

- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PROVIDERS.md`
- `docs/evaluations/wave-7E-evaluator-prompt.md`
- Diff/range: `fe3293b..HEAD` on branch `codex/wave-7e-eval-fix`
- Relevant code:
  - `api/handlers/inference.go`
  - `api/handlers/inference_test.go`
  - `api/server.go`
  - `api/server_test.go`
  - `internal/providers/types.go`
  - `internal/proxy/engine.go`
  - `internal/proxy/engine_test.go`
  - `internal/translate/responses.go`
  - `internal/translate/responses_test.go`

## Prior Blocking Findings To Re-check

- `/v1/messages` must not silently accept Anthropic-native tool shapes that are not converted. Native `tools`, native `tool_choice`, request `tool_use`, and request `tool_result` should fail explicitly before dispatch unless full conversion exists.
- `/v1/messages` responses must preserve provider `ToolCalls` as Anthropic `tool_use` content blocks and map the OpenAI-style `tool_calls` finish reason to Anthropic `tool_use`.
- `/v1/responses` must not silently skip unsupported input item/content types such as `function_call_output` or `input_image`.
- `/v1/responses` responses must not omit provider tool calls; they should appear as Responses `function_call` output items or fail explicitly.
- `/v1/messages` and `/v1/responses` successful non-stream dispatches must honor request logging when enabled.
- Request logs should use dispatch/provider metadata when available, including aliases and non-prefix provider routes, instead of relying only on `gpt-*` / `claude-*` model prefixes.
- `docs/PROVIDERS.md` and `docs/WORKFLOW.md` should accurately reflect the remediation.

## Gates

Run:

```bash
go test ./... -count=1
go vet ./...
go build ./cmd/g0router
npm --prefix ui test -- --run
npm --prefix ui run build
make build
```

## Return

```markdown
## Verdict

PASS or FAIL

## Blocking Findings

Issues that must be fixed before Wave 7.F.

## Non-Blocking Findings

Risks or cleanup notes.

## Gate Results

Command results with exact failures if any.

## Workflow Status Review

Whether `docs/WORKFLOW.md` accurately reflects Wave 7.E remediation and is ready to advance to Wave 7.F.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
```
