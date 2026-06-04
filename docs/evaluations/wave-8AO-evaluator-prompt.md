# g0router Wave 8.AO Evaluation

Evaluate completed wave `8.AO` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/phases/phase-10-dashboard-ui.md`
- `docs/evaluations/wave-8AO-evaluator-prompt.md`
- `ui/src/api.ts`
- `ui/src/pages/ProvidersPage.tsx`
- `ui/src/pages/ProvidersPage.test.tsx`
- `ui/src/pages/ConnectionsAuthPage.test.tsx`
- `ui/e2e/dashboard.e2e.ts`
- Diff/commits:
  - `95a6f22 phase-8/task-ui: add provider oauth connect flow`
  - `d2a2972 Merge wave 8.AO dashboard provider OAuth connect flow`
  - `5b1586f phase-8/task-workflow: record provider oauth dashboard`

Check:
- OAuth controls render only for providers whose `auth_types` include `oauth`.
- Provider OAuth is separate from API-key connection creation and does not regress API-key create/test/delete behavior.
- Starting OAuth posts to `/api/oauth/{provider}/authorize` with `account_label`.
- Started OAuth state displays only public fields such as authorization URL/session/device/status, not access tokens, refresh tokens, API keys, or provider secrets.
- Completing OAuth posts to `/api/oauth/{provider}/exchange`, reloads redacted connection data, and keeps success/error feedback visible across refresh.
- Failure handling does not leak secrets.
- Connections/Auth exposes the same provider OAuth control plane.
- Mocked Playwright E2E proves start and exchange request bodies and visible success state on desktop and mobile.
- Changes are surgical and limited to the owned UI files plus workflow/evaluator documentation.
- Workflow status is accurate for Wave 8.AO.

Run gates:
- `npm --prefix ui test -- --run ProvidersPage ConnectionsAuthPage`
- `npm --prefix ui run e2e -- --grep "OAuth"`
- `npm --prefix ui test -- --run`
- `npm --prefix ui run build`

After running UI gates, do not commit generated artifacts. If `ui/test-results/` or tracked `ui/dist` rewrites appear, report them rather than editing.

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
