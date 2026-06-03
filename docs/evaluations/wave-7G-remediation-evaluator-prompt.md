# g0router Wave 7.G Remediation Evaluation Prompt

Evaluate the Wave `7.G` evaluator remediation in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

## Review Inputs

- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/phases/phase-09-mcp-gateway.md`
- `docs/evaluations/wave-7G-evaluator-prompt.md`
- Relevant code:
  - `api/handlers/mcp.go`
  - `api/handlers/mcp_test.go`
  - `internal/cli/root.go`
  - `internal/cli/root_test.go`
  - `internal/cli/mcp_runtime.go`
  - `internal/mcp/oauth.go`
  - `internal/mcp/oauth_test.go`
  - `internal/mcp/advanced_integration_test.go`
  - `internal/store/mcpoauth.go`
  - `internal/store/mcpinstances.go`
- Commit refs:
  - `cb93c1b phase-7/task-g5: implement mcp startup rehydration`
  - remediation commit after `cb93c1b`

## Prior Blocking Findings To Re-check

- `/api/mcp/clients` list/create responses must redact environment secrets such as `TOKEN`, `SECRET`, `KEY`, `AUTHORIZATION`, and `PASSWORD`.
- Runtime startup must select persisted MCP OAuth accounts for active instances, refresh them when expired or near expiry using stored `token_endpoint`, persist refreshed tokens, and inject `Authorization: Bearer ...` into the runtime launch headers.
- MCP OAuth callback completion must not fabricate `mcp_<code>` access tokens when a token endpoint is unavailable or returns redirect-class responses.
- Existing redirect and pasted callback flows must still work through real token endpoint exchange.
- Management API responses must not leak raw MCP OAuth access or refresh tokens.
- Wave 7.G workflow status should include the remediation task and should not advance to Wave 7.H until this remediation passes.

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

Whether `docs/WORKFLOW.md` accurately reflects Wave 7.G remediation status.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
```
