# g0router Wave 7.G Evaluation Prompt

Evaluate completed Wave `7.G` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

## Review Inputs

- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/phases/phase-09-mcp-gateway.md`
- Relevant code:
  - `api/handlers/mcp.go`
  - `api/handlers/mcp_test.go`
  - `api/server.go`
  - `internal/cli/mcp_auth.go`
  - `internal/cli/mcp_auth_test.go`
  - `internal/cli/mcp_runtime.go`
  - `internal/cli/mcp_runtime_test.go`
  - `internal/cli/mcp_http_runtime_test.go`
  - `internal/cli/root.go`
  - `internal/cli/root_test.go`
  - `internal/mcp/jsonrpc.go`
  - `internal/mcp/stdio.go`
  - `internal/mcp/stdio_test.go`
  - `internal/mcp/http.go`
  - `internal/mcp/httpclient.go`
  - `internal/mcp/httpclient_test.go`
  - `internal/mcp/oauth.go`
  - `internal/mcp/oauth_test.go`
  - `internal/mcp/toolmanager.go`
  - `internal/mcp/toolmanager_test.go`
  - `internal/mcp/agent.go`
  - `internal/mcp/agent_test.go`
  - `internal/mcp/healthmonitor.go`
  - `internal/store/mcpoauth.go`
  - `internal/store/mcpoauth_test.go`
  - `internal/store/mcpinstances.go`
  - `internal/store/mcpinstances_test.go`
- Commit refs:
  - `d8131a1 phase-7/task-g1: implement stdio mcp json-rpc client`
  - `a95807a phase-7/task-g2: implement mcp oauth token exchange`
  - `e37fe26 phase-7/task-g2: record mcp oauth completion`
  - `f5e149e phase-7/task-g3: implement http mcp json-rpc clients`
  - `7a4b4a1 phase-7/task-g4: start tool manager remediation`
  - `8be8705 phase-7/task-g4: implement mcp tool validation`
  - `a66d229 phase-7/task-g4: record tool manager completion`
  - final `phase-7/task-g5: implement mcp startup rehydration` commit after `a66d229`

## Required Checks

- Stdio MCP client is a real JSON-RPC client over process stdin/stdout, not a placeholder process wrapper.
- Streamable HTTP and SSE clients perform initialize, initialized notification, `tools/list`, and `tools/call` against real JSON-RPC endpoints.
- MCP OAuth stores state/verifier metadata, exchanges with the stored PKCE verifier, persists account tokens, and avoids exposing raw token material through API or CLI responses.
- OAuth refresh behavior uses persisted endpoint metadata and does not invent/fabricate fresh tokens except where explicitly documented as legacy fallback.
- Tool manager state is concurrency-safe, including registration, lookup, list, client registration, and call paths.
- Tool argument validation enforces registered JSON schemas before dispatching to clients.
- Request-context tool filtering limits both visible tools and callable tools for a request, including API handler behavior.
- Startup rehydrates active MCP instances from the store, uses unredacted runtime launch config, registers tools into the live runtime, persists manifests, and records health.
- Management API responses still redact credentials while runtime startup still receives the real secrets needed to connect.
- No `init()` functions, mutable package globals, mocks, speculative abstractions, or unrelated refactors were introduced.
- Workflow status accurately reflects Wave 7.G completion and does not advance to Wave 7.H before this evaluation is accepted.

## Gates

Run:

```bash
go test ./... -count=1
go test -race ./internal/mcp -run TestToolManagerConcurrentRegistrationListAndCalls -count=1
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

Issues that must be fixed before Wave 7.H implementation advances.

## Non-Blocking Findings

Risks or cleanup notes.

## Gate Results

Command results with exact failures if any.

## Workflow Status Review

Whether `docs/WORKFLOW.md` accurately reflects Wave 7.G status.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
```
