# g0router Wave 7.E Evaluation Prompt

Evaluate completed wave `7.E` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

## Review Inputs

- `AGENTS.md`
- `CLAUDE.md`
- `README.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PROVIDERS.md`
- `docs/SCHEMA.md`
- Phase docs:
  - `docs/phases/phase-02-http-server-proxy-engine.md`
  - `docs/phases/phase-03-multi-provider-support.md`
  - `docs/phases/phase-04-persistence-provider-registry.md`
  - `docs/phases/phase-06-account-fallback-combos.md`
  - `docs/phases/phase-08-usage-tracking-cost-logging.md`
  - `docs/phases/phase-09-mcp-gateway.md`
- Relevant code:
  - `api/server.go`
  - `api/server_test.go`
  - `api/handlers/inference.go`
  - `api/handlers/inference_test.go`
  - `internal/proxy/engine.go`
  - `internal/proxy/engine_test.go`
  - `internal/proxy/combo.go`
  - `internal/proxy/combo_test.go`
  - `internal/modelcatalog/`
  - `internal/provider/fallback.go`
  - `internal/provider/matrix.go`
  - `internal/providers/anthropic/`
  - `internal/providers/gemini/`
  - `internal/translate/`
  - `internal/usage/`
- Commit refs:
  - `330ea06 phase-7/task-e1: resolve model aliases`
  - `21d4a24 phase-7/task-e2: honor request log settings`
  - `b660b2c phase-7/task-e3: add documented v1 routes`
  - `2a0380d phase-7/task-e4: preserve provider tool calls`
  - `6174fca phase-7/task-e5: harden combo dispatch`
  - range `b1ff7f0..HEAD`

## Check

- Model routing is catalog/alias-driven before falling back to legacy `gpt-*` and `claude-*` prefixes.
- Alias dispatch rewrites the upstream request model without mutating the caller request.
- `/v1/chat/completions`, `/v1/messages`, and `/v1/responses` routes are available and dispatch through the real inference engine.
- `/v1/messages` and `/v1/responses` either preserve required fields or fail explicitly instead of silently dropping unsupported request shape.
- Request logging honors `EnableRequestLogs`, records successful non-stream requests only when enabled, and includes meaningful usage/cost fields.
- Provider, model, and cost inference are not misleading for aliases, catalog models, or combo routes.
- OpenAI-style tool definitions, assistant tool calls, tool responses, and upstream tool-call responses are preserved across Anthropic and Gemini adapters.
- Anthropic tool handling includes `tools`, `tool_choice`, `tool_use`, `tool_result`, non-empty `tool_use_id`, default input schema, coalesced consecutive tool results, and streamed tool-use deltas.
- Gemini tool handling includes function declarations, function calls, function responses with function name plus call id, and response function-call mapping back to `providers.ToolCall`.
- Combo dispatch supports `combo/*` through `Engine.Dispatch` and `Engine.DispatchStream`.
- Combo steps resolve model aliases and catalog-owned models instead of blindly trusting stored provider IDs.
- Connection selection uses round-robin/fallback state, skips model-locked or globally unavailable connections, records fallback-worthy failures, and clears recovered backoff on success.
- Retry behavior is narrow: fallback-worthy errors may try the next account; non-fallback request/auth/unsupported errors should not poison connection backoff.
- Quota gates use the selected provider key, block explicit exhausted quota before provider invocation, treat missing/unsupported fetchers as no gate, and return 429 through API handlers.
- `g0router serve` startup registers the same quota fetchers with both the engine and management API.
- Workflow status accurately marks all Wave 7.E tasks and the evaluator prompt complete.
- Existing `.DS_Store`, `.pi/`, and untracked `AGENTS.md` state was not cleaned up or committed.

## Known Deferred Work

- Live provider streaming correctness for all providers remains Wave 7.F work.
- RTK/Caveman request-path integration remains incomplete unless verified otherwise.
- Real MCP runtime/tool adaptation remains Wave 7.G work.
- Dashboard API wiring remains Wave 7.H work.
- Request logging for failed and streaming requests remains Wave 7.I work unless implemented later.
- Quota enforcement depends on real provider quota fetchers; the current default unsupported fetchers are intentionally fail-open.

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

Whether `docs/WORKFLOW.md` accurately reflects Wave 7.E and is ready to advance to Wave 7.F.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
```
