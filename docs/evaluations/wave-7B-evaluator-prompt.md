# g0router Wave 7.B Evaluation Prompt

Evaluate completed wave `7.B` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

## Review Inputs

- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- Relevant phase/remediation docs in `docs/`
- Commit refs:
  - range `850aefc..HEAD`

## Check

- `g0router serve` constructs a real `InferenceEngine` in normal startup.
- Default startup registers implemented provider adapters instead of leaving `/v1/chat/completions` and `/v1/models` disconnected.
- A fresh store returns catalog-backed models and chat requests reach the engine, returning `no active connections` instead of `inference engine unavailable`.
- `api.ServerConfig.ModelSource` uses the runtime engine, not the old static placeholder source.
- MCP client and tool managers are wired into normal startup, and `/api/mcp/clients` no longer returns `mcp runtime unavailable` for valid runtime configuration.
- Request contexts are propagated through inference, streaming inference, model listing, provider model listing, quota fetches, MCP client registration, and MCP tool execution.
- Clean-checkout UI embed behavior from Wave 7.A remains intact.
- Existing `.DS_Store`, `.pi/`, and untracked `AGENTS.md` state was not cleaned up or committed.

## Known Deferred Work

- Model dispatch is still prefix-based and must be replaced in Wave 7.E.
- MCP runtime only initializes transport and caches an empty manifest; real JSON-RPC tool discovery/calls remain Wave 7.G work.
- OAuth persistence and token refresh remain Wave 7.C work.

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

Issues that must be fixed before Wave 7.C.

## Non-Blocking Findings

Risks or cleanup notes.

## Gate Results

Command results with exact failures if any.

## Workflow Status Review

Whether `docs/WORKFLOW.md` accurately reflects Wave 7.B and advances to Wave 7.C.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
```
