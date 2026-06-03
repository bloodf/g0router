# g0router Wave 7.F Remediation Evaluation Prompt

Evaluate the Wave `7.F` evaluator remediation in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

## Review Inputs

- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PROVIDERS.md`
- `docs/evaluations/wave-7F-evaluator-prompt.md`
- Relevant code:
  - `api/handlers/inference.go`
  - `api/handlers/inference_test.go`
  - `internal/proxy/errors.go`
  - `internal/proxy/engine.go`
  - `internal/proxy/engine_test.go`
  - `internal/proxy/combo.go`
  - `internal/proxy/combo_test.go`
  - `internal/provider/matrix.go`
- Commit refs:
  - `3844d78 phase-7/task-f4: add wave 7f evaluator prompt`
  - remediation commit after `3844d78`

## Prior Blocking Findings To Re-check

- Classified upstream provider errors must preserve appropriate public HTTP status codes while keeping sanitized OpenAI-compatible error bodies.
- Provider auth errors should map to a safe 401 `invalid_request_error` response without leaking provider text, URLs, bearer tokens, or API keys.
- Provider rate-limit errors should map to a safe 429 `rate_limit_error` response without leaking provider text, URLs, bearer tokens, or API keys.
- Generic upstream failures must remain sanitized 502 `server_error` responses.
- Providers with matrix `Inference=false`, especially `bedrock`, must not be reachable through stored model aliases.
- Providers with matrix `Inference=false`, especially `bedrock`, must not be invoked through combo step routing.
- Adapter-only providers with matrix `Inference=true` may remain reachable through explicit alias/combo routes if tests still cover that behavior.
- Workflow status should not claim Wave 7.G is active until this remediation is evaluated and accepted.

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

Issues that must be fixed before Wave 7.G implementation advances.

## Non-Blocking Findings

Risks or cleanup notes.

## Gate Results

Command results with exact failures if any.

## Workflow Status Review

Whether `docs/WORKFLOW.md` accurately reflects Wave 7.F remediation status.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
```
