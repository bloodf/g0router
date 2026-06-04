# g0router Wave 8.Y Evaluation

Evaluate completed wave `8.Y` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files or commit.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `ui/src/App.tsx`
- `ui/src/App.test.tsx`
- `ui/src/pages/APIKeysPage.tsx`
- `ui/src/pages/APIKeysPage.test.tsx`
- `ui/src/pages/EndpointPage.tsx`
- `ui/e2e/dashboard.e2e.ts`

Diff/commit:
- Wave 8.Y commit after `ded7d73 phase-8/task-workflow: record stage status commit`

Check:
- The dashboard has a dedicated `API Keys` route and navigation item.
- The API Keys page uses the real `/api/keys` API contract through the existing API key control plane, not static fixtures or production-only mock data.
- Endpoint-copy controls remain on `Endpoint Setup` and are not shown on the dedicated API Keys page.
- Unit coverage verifies API key rendering without endpoint-copy controls.
- Playwright E2E covers normal, empty, and auth-expired API Keys states on desktop and mobile.
- Credential safety remains intact: full provider credentials and stored API key secrets are not displayed; only the transient raw key from creation may be shown.
- `docs/WORKFLOW.md` accurately records Wave 8.Y completion and evaluation-pending state.
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

Whether `docs/WORKFLOW.md` accurately reflects Wave 8.Y completion and evaluation-pending state.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
