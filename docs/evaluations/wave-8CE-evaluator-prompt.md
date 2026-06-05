# g0router Wave 8.CE Evaluation

Evaluate completed wave `8.CE` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Review:
- AGENTS.md
- CLAUDE.md
- README.md
- docs/README.md
- docs/PLAN.md
- docs/WORKFLOW.md
- docs/ORCHESTRATION.md
- docs/SCHEMA.md
- docs/evaluations/wave-8CE-evaluator-prompt.md
- Commit for Wave 8.CE: `{commit_ref}`

Start read-only. Do not edit files.

Check:
- Dashboard exposes usable update controls for the documented PUT endpoints:
  - `PUT /api/connections/:id`
  - `PUT /api/aliases/:alias`
  - `PUT /api/combos/:id`
  - `PUT /api/pricing/:provider/:model`
- Connection update requests do not serialize access tokens, refresh tokens, API keys, or secret-shaped provider metadata.
- Pricing, Logs, and Diagnostics page tests cover loading, empty, error, and auth-expired states where those states exist.
- Mocked dashboard E2E covers connection activate/deactivate, alias update, combo update, pricing update, and auth-expired states for Pricing, Usage, Logs, Quotas, and Diagnostics.
- UI controls remain backed by real dashboard API helpers, not production-only test fixtures.
- No unrelated files, generated artifacts, or protected local files were changed.
- `docs/WORKFLOW.md`, `docs/PLAN.md`, and `docs/ORCHESTRATION.md` accurately describe Wave 8.CE.

Required gates:

```bash
npm --prefix ui test -- --run src/api.test.ts src/pages/AliasesPage.test.tsx src/pages/CombosPage.test.tsx src/pages/PricingPage.test.tsx src/pages/ProvidersPage.test.tsx src/pages/LogsPage.test.tsx src/pages/DiagnosticsPage.test.tsx
npm --prefix ui run e2e
make verify
git diff --check
```

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
