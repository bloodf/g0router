# g0router Wave 8.AG Evaluation

Evaluate completed wave `8.AG` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PLAN.md`
- `internal/mcp/oauth.go`
- `internal/mcp/oauth_test.go`
- `api/handlers/mcp.go`
- `api/handlers/mcp_test.go`
- `internal/cli/mcp_auth.go`
- `internal/cli/mcp_auth_test.go`
- Commit `168feb5` and the workflow/docs commit that records Wave 8.AG.

Start read-only. Do not edit files.

Check:
- MCP OAuth protected-resource discovery reads `WWW-Authenticate` `resource_metadata`, protected resource metadata `authorization_servers`, and authorization-server metadata `authorization_endpoint`.
- Discovery uses request contexts and local HTTP behavior in tests; no mocks, external network, mutable globals, or `init()` functions were added.
- Existing explicit `authorization_url` behavior is preserved for API and CLI start flows.
- API and CLI MCP OAuth start flows can omit `authorization_url` when `resource_uri` supports metadata discovery.
- Tests prove PKCE/state flow storage and generated authorization URLs for API and CLI.
- Missing or invalid metadata fails closed without inventing endpoints or leaking credentials.
- Wave 8.AG is accurately recorded in `docs/WORKFLOW.md`.
- `docs/PLAN.md` and `docs/ORCHESTRATION.md` align Stage 8 through Wave 8.AG.
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

Whether `docs/WORKFLOW.md`, `docs/PLAN.md`, and `docs/ORCHESTRATION.md` are accurate for Wave 8.AG.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
