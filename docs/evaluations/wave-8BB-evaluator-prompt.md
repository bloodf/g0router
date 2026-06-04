# g0router Wave 8.BB Evaluation

Evaluate completed wave `8.BB` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

Review:

- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PROVIDERS.md`
- `internal/provider/oauth/cursor.go`
- `internal/provider/oauth/cursor_test.go`
- `internal/provider/matrix.go`
- OMP reference: `/Users/heitor/Developer/github.com/bloodf/oh-my-pi/packages/ai/src/utils/oauth/cursor.ts`
- Diff/commit for Wave 8.BB

Check:

- Cursor `Start` uses OMP-style `https://cursor.com/loginDeepControl` semantics: PKCE challenge, UUID, `mode=login`, and `redirectTarget=cli`.
- Cursor no longer advertises or depends on callback authorization-code exchange.
- Cursor `Poll` calls the configured poll endpoint with `uuid` and `verifier`, treats 404 as pending, and maps successful camelCase `accessToken`/`refreshToken` responses into `TokenResult`.
- Cursor refresh posts `{}` to `exchange_user_api_key` with `Authorization: Bearer <refresh token>`.
- Token expiry uses JWT `exp` when present with a refresh margin, and falls back safely when absent.
- Cursor remains `auth_only`; no unsupported Cursor inference adapter is advertised.
- Tests use local HTTP servers and fakes, not external network or mocks.
- No secrets are committed, logged, or embedded.
- No unrelated refactors, `init()` functions, or mutable globals were added.
- Workflow status and provider docs accurately describe the implemented scope and remaining Cursor inference gap.

Run gates:

- `go test ./internal/provider/oauth -count=1`
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

Whether `docs/WORKFLOW.md` is accurate.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
