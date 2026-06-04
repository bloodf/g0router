# g0router Wave 8.T Evaluation

Evaluate completed wave `8.T` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files or commit.

Review:
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ARCHITECTURE.md`
- `docs/phases/phase-07-rtk-caveman.md`
- `api/server.go`
- `api/server_test.go`
- `api/handlers/inference.go`
- `internal/rtk/rtk.go`
- `internal/rtk/caveman.go`
- `internal/store/settings.go`
- `internal/logging/requestlog.go`

Diff/commit:
- Wave 8.T commit after `ac08662 phase-8/task-workflow: record vertex oauth commit`

Check:
- `/v1/chat/completions`, `/v1/messages`, and `/v1/responses` requests are preprocessed before reaching the inference engine.
- When settings enable RTK, tool output content is compressed before dispatch and the caller request is not mutated.
- When settings enable caveman, the configured caveman prompt is injected before dispatch and the caller request is not mutated.
- Request logging records source format, target format, RTK enabled, and caveman enabled metadata.
- Streaming dispatch uses the same preprocessing wrapper.
- No provider token, API key, or leaked MiniMax credential appears in source, docs, tests, logs, or command output.
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

Whether `docs/WORKFLOW.md` accurately reflects Wave 8.T completion and evaluation-pending state.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
