# g0router Wave 7.A Evaluation Prompt

Evaluate completed wave `7.A` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

## Review Inputs

- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/CONFIG.md`
- Commit refs:
  - `4b77c5b` (`phase-7/task-a1: protect management api`)
  - `20d97dd` (`phase-7/task-a2: validate serve config`)
  - range `5317d2b..HEAD`

## Check

- `/api/*` management endpoints require API-key auth when `REQUIRE_API_KEY=true`.
- OAuth callback endpoints remain public for provider redirects.
- CORS no longer emits wildcard `Access-Control-Allow-Origin`; local origins are allowed.
- `GET`, `POST`, and `PUT` connection responses never serialize `AccessToken`, `RefreshToken`, or `APIKey`.
- Stored connection credentials are not mutated by response redaction.
- `g0router serve` uses validated config loading for `PORT`, `BIND_ADDRESS`, booleans, and `API_KEY_SECRET`.
- Default bind address is localhost, and Docker host binding is localhost-only unless deliberately changed.
- Existing `.DS_Store`, `.pi/`, and untracked `AGENTS.md` state was not cleaned up or committed.

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

Issues that must be fixed before Wave 7.B.

## Non-Blocking Findings

Risks or cleanup notes.

## Gate Results

Command results with exact failures if any.

## Workflow Status Review

Whether `docs/WORKFLOW.md` accurately reflects Wave 7.A.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
```
