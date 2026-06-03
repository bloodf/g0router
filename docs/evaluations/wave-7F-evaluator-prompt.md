# g0router Wave 7.F Evaluation Prompt

Evaluate completed wave `7.F` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

## Review Inputs

- `AGENTS.md`
- `CLAUDE.md`
- `README.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PROVIDERS.md`
- Relevant code:
  - `api/handlers/inference.go`
  - `api/handlers/inference_test.go`
  - `api/handlers/providers.go`
  - `api/handlers/providers_test.go`
  - `internal/cli/root_test.go`
  - `internal/provider/matrix.go`
  - `internal/provider/matrix_test.go`
  - `internal/providers/openai/openai.go`
  - `internal/providers/openai/openai_test.go`
  - `internal/providers/azure/azure.go`
  - `internal/providers/azure/azure_test.go`
  - `internal/providers/openaicompat/provider.go`
  - `internal/providers/openaicompat/provider_test.go`
- Commit refs:
  - `10b5039 phase-7/task-f1: stream upstream provider chunks`
  - `122bd98 phase-7/task-f2: sanitize public inference errors`
  - `c436e6a phase-7/task-f3: lock down bedrock status`
  - range `07fb4d0..HEAD`

## Check

- OpenAI streaming uses the upstream response body as a live stream and does not buffer the whole upstream body before emitting downstream chunks.
- Azure streaming uses the same live upstream streaming behavior.
- OpenAI-compatible streaming uses the same live upstream streaming behavior.
- Streaming tests prove chunk latency/order with local HTTP servers rather than mocks.
- Public inference errors are stable OpenAI-compatible error objects for provider missing, no active connections, quota exhaustion, and generic upstream failures.
- Generic upstream failures do not leak raw provider error text, API keys, bearer tokens, request URLs, or other credential material in sync or streaming responses.
- Provider-specific errors still use appropriate HTTP status codes where the implementation can classify them.
- Gemini, Vertex, and Bedrock do not falsely advertise implemented streaming if they still lack correct live upstream streaming support.
- Bedrock is not documented or exposed as a Converse implementation unless it actually uses the documented Converse API path and request/response semantics.
- Bedrock remains `adapter_only`, with `RegisteredAdapter=true`, but no public direct dispatch, no public inference, no streaming, no model catalog/ListModels, and no quota support.
- `GET /api/providers` exposes the matrix `Inference` capability, not just `PublicInference`, so adapter-only inference-capable providers do not look inert.
- `docs/PROVIDERS.md` does not contain stale Wave 7.F TODO wording or overclaim Bedrock, Gemini, Vertex, or OpenAI-compatible provider correctness.
- Existing `.DS_Store`, `.pi/`, and untracked `AGENTS.md` state was not cleaned up or committed.
- `docs/WORKFLOW.md` accurately marks Wave 7.F complete and advances to Wave 7.G only after all Wave 7.F tasks are done.

## Known Deferred Work

- Real MCP OAuth, stdio/Streamable HTTP/SSE JSON-RPC clients, startup rehydration, schema validation, and tool sync remain Wave 7.G work.
- Real dashboard API reads/mutations remain Wave 7.H work.
- Expanded usage/cost/log/quota behavior remains Wave 7.I work.
- Non-public adapter providers can remain adapter-only as long as the matrix and docs describe their status honestly.

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

Whether `docs/WORKFLOW.md` accurately reflects Wave 7.F and advances to Wave 7.G.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
```
