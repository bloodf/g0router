# g0router Wave 8.BF Evaluation

Evaluate completed wave `8.BF` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files, do not commit, and do not clean protected local files.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `ui/playwright.config.ts`
- `ui/e2e/dashboard.e2e.ts`
- `ui/e2e/real-server.e2e.ts`
- `internal/cli/root.go`
- `api/server.go`

Check:
- The new Playwright test starts a real `g0router serve` process using a temp data dir and loopback port.
- The test mints a real API key through the CLI and uses it in the dashboard instead of mocking `/api/*`.
- The test loads the embedded dashboard from the real Go server, not Vite-only fixtures.
- The test exercises real authenticated dashboard reads and mutations through `/api/settings` and `/api/keys`.
- The test does not use external network, committed secrets, shared persistent state, or production credentials.
- The broad mocked dashboard E2E suite remains intact.
- Workflow status accurately records Wave 8.BE evaluator PASS and Wave 8.BF gate evidence.

Run:
- `npm --prefix ui run e2e -- real-server.e2e.ts`
- `npm --prefix ui test -- --run`
- `npm --prefix ui run e2e`
- `npm --prefix ui run build`
- `go test ./... -count=1`
- `go vet ./...`
- `go build ./cmd/g0router`
- `git diff --check`
- `git status --short`

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

Whether workflow/docs status is accurate.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
