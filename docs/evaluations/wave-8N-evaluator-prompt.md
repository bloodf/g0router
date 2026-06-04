# g0router Wave 8.N Evaluation

Evaluate completed wave `8.N` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Review:
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PROVIDERS.md`
- `README.md`
- Commits: `09d68ac`, `f83ca6d`, `9d98320`, `d13892d`, `743e581`, `a005601`, `8ce739f`, `f8c3910`, `e674de4`, `f98638b`, `e34491d`, `b2f6fe2`

Start read-only. Do not edit files.

Check:
- Provider dashboard connection create/test/delete uses real authenticated API contracts and never displays full credentials.
- MCP dashboard OAuth completion, tool execution, and deletion use real API contracts and handle failure states.
- Provider matrix does not overclaim quota; providers with no fetcher report `quota=false`.
- OpenAI-compatible base URL normalization avoids duplicate `/v1` paths.
- OAuth exchange failures and streaming errors are sanitized and do not leak upstream token material.
- Docker Compose no longer requires an unused `JWT_SECRET`; `API_KEY_SECRET` remains required.
- Anthropic, OpenAI, Azure, and OpenAI-compatible streaming surface malformed SSE/upstream error events as sanitized stream errors instead of silent `[DONE]`.
- `g0router providers test` is alias-aware, matrix-aware, and connection-aware.
- `/api/providers/:id/models` canonicalizes provider aliases and rejects auth-only/non-inference providers explicitly.
- Workflow, provider docs, README, and orchestration docs accurately reflect Stage 8 and do not claim release readiness before external evaluation.
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

Whether `docs/WORKFLOW.md` accurately reflects Wave 8.N completion and evaluation-pending state.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
