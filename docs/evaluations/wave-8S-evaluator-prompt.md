# g0router Wave 8.S Evaluation

Evaluate completed wave `8.S` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files or commit.

Review:
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/PROVIDERS.md`
- `internal/provider/oauth/types.go`
- `internal/provider/oauth/types_test.go`
- `internal/provider/credentials.go`
- `internal/cli/auth.go`
- `internal/cli/auth_test.go`
- `api/handlers/oauth.go`
- `api/handlers/oauth_test.go`

Diff/commit:
- Wave 8.S commit after `e729177 phase-8/task-workflow: record vertex route commit`

Check:
- `oauth.CanonicalFlowProviderID("vertex")` returns `gemini`.
- `oauth.CanonicalProviderID("vertex")` returns `vertex`, not `gemini`.
- CLI `auth login vertex --device` uses the Gemini OAuth flow but persists an active `vertex` connection with `oauth_provider=gemini`.
- HTTP `/api/oauth/vertex/authorize`, `/api/oauth/vertex/exchange`, and callback sessions use the Gemini OAuth flow but persist runtime provider `vertex`.
- Codex/OpenAI and GitHub/GitHub Copilot alias behavior still works.
- Token material is never serialized in auth responses or command output.
- Docs/provider matrix truthfully describe Vertex auth as Gemini-flow-backed runtime provider `vertex`.
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

Whether `docs/WORKFLOW.md` accurately reflects Wave 8.S completion and evaluation-pending state.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
