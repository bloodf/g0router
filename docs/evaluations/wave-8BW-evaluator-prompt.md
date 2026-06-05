# g0router Wave 8.BW Evaluation

Evaluate completed wave `8.BW` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PROVIDERS.md`
- `docs/CONFIG.md`
- `docs/SCHEMA.md`
- `internal/search/search.go`
- `internal/search/search_test.go`
- `internal/cli/root.go`
- `internal/cli/root_test.go`
- `internal/provider/matrix.go`
- Diff/commits for Wave 8.BW

Check:
- Kagi and Tavily remain `auth_only` providers and are not promoted into public inference dispatch, `/v1` routing, catalog, model listing, streaming, pricing, quota, or `g0router providers list`.
- Active stored API-key connections register built-in MCP tools `kagi__search` and `tavily__search` during normal server startup.
- No active API-key connection means no built-in search MCP tool is exposed.
- Kagi search uses local-testable HTTP behavior, Bearer authorization, a JSON search request, recency conversion to a date filter, and normalized source/related/answer output.
- Tavily search uses local-testable HTTP behavior, Bearer authorization, OMP-style request fields, no `topic`, and normalized answer/source output.
- Upstream error handling does not leak stored API keys or raw upstream error bodies.
- Docs accurately describe built-in MCP search tools without implying a new `/api/search` route or inference provider support.

Required gates:
- `go test ./internal/search ./internal/cli ./internal/provider ./api/handlers -run 'Test(KagiSearchTool|TavilySearchTool|SearchToolRequiresActiveAPIKey|SearchToolErrorsAreSanitized|BuiltInSearchTools|DefaultServerConfigRegistersBuiltInSearchTools|ProviderMatrixKeepsSearchProvidersAuthOnly|ProviderMatrixMarksSearchCredentialsAuthOnly)' -count=1`
- `go test ./... -count=1`
- `go vet ./...`
- `go build ./cmd/g0router`
- `npm --prefix ui test -- --run`
- `npm --prefix ui run build`
- `npm --prefix ui run e2e`
- `make build`
- `git diff --check`

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

Whether `docs/WORKFLOW.md` is accurate.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
