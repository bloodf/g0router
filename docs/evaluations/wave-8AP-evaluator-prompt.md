# g0router Wave 8.AP Evaluation

Evaluate completed wave `8.AP` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PLAN.md`
- `docs/phases/phase-10-dashboard-ui.md`
- Implementation commit: `7e0830b phase-8/task-ui: reconcile quotas dashboard label`
- Merge commit: `15da585 Merge wave 8.AP dashboard quotas label`
- Related selector stabilization commit that must remain preserved: `541adc8 phase-8/task-ui: stabilize provider e2e selectors`

Check:
- User-facing dashboard navigation, route heading, and page panel copy use `Quotas`, not singular `Quota`.
- The stable page id remains `quota` and the backend API path remains `/api/usage/quota/{provider}`.
- The provider combobox E2E selectors still use exact matching so `Provider` does not collide with `OAuth provider`.
- TDD evidence exists in tests for navigation, page panel copy, and mocked Playwright navigation.
- Changes are surgical and limited to the owned UI files plus workflow/evaluator docs.
- No generated `ui/test-results/` or tracked `ui/dist` rewrites remain.
- Workflow status for Wave 8.AP is accurate.

Gates to run:
- `npm --prefix ui test -- --run App QuotaPage`
- `npm --prefix ui run e2e -- --grep "Quotas"`
- `npm --prefix ui run e2e`
- `npm --prefix ui test -- --run`
- `npm --prefix ui run build`

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
