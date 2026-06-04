# g0router Wave 8.M Evaluation

Evaluate completed wave `8.M` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/CONFIG.md`
- `internal/providers/openaicompat/live_minimax_test.go`
- Commit: `f83addd`

Start read-only. Do not edit files.

Check:
- MiniMax live provider smoke tests are skipped by default unless `G0ROUTER_LIVE_TESTS=1` is set.
- MiniMax tokens are read only from `G0ROUTER_E2E_MINIMAX_API_KEY`.
- The test never embeds, logs, or commits real provider tokens.
- Optional base URL override uses `G0ROUTER_E2E_MINIMAX_BASE_URL`.
- Normal release gates do not require external network or live provider availability.
- `docs/CONFIG.md` documents the opt-in controls clearly.
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

Whether `docs/WORKFLOW.md` is accurate.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
