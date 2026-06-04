# g0router Wave 8.AE Evaluation

Evaluate completed wave `8.AE` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files or commit.

Review:
- `docs/PLAN.md`
- `docs/ORCHESTRATION.md`
- `docs/WORKFLOW.md`
- `internal/mcp/oauth.go`
- `internal/mcp/oauth_test.go`

Diff/commit:
- Wave 8.AE implementation commit `60a0e41 phase-8/task-mcp: discover oauth token metadata`
- Wave 8.AE workflow commit after implementation

Check:
- MCP OAuth callback completion can discover `token_endpoint` from `/.well-known/oauth-authorization-server` when the authorization URL is not an `/authorize` path.
- Existing `/authorize` to `/token` compatibility remains covered.
- Discovery uses the existing no-redirect OAuth client behavior.
- Metadata and token exchanges use request context.
- Token responses still require an access token before persisting an account.
- Missing metadata or missing token endpoint fails closed with `errOAuthTokenEndpointUnavailable`.
- Tests use local `httptest` servers, not external network or mocks.
- `docs/PLAN.md` and `docs/ORCHESTRATION.md` are aligned to Stage 8 running through Wave `8.AE`.
- `docs/WORKFLOW.md` accurately records Wave 8.AD evaluator PASS, Wave 8.AE completion, and Wave 8.AE evaluation-pending state.
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

Whether `docs/WORKFLOW.md` accurately reflects Wave 8.AD evaluator PASS, Wave 8.AE completion, and Wave 8.AE evaluation-pending state.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
