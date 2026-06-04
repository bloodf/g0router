# g0router Wave 8.AL Evaluation

Evaluate completed wave `8.AL` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

Review:
- CLAUDE.md
- AGENTS.md if present
- docs/README.md
- docs/WORKFLOW.md
- docs/ORCHESTRATION.md
- docs/phases/phase-10-dashboard-ui.md
- ui/src/pages/McpPage.tsx
- ui/src/pages/McpSplitPages.test.tsx
- ui/e2e/dashboard.e2e.ts
- Commit refs: `c1c5b3a`, merge commit `42762e9`, follow-up test commit `566ab53`

Check:
- The MCP Instances dashboard form exposes Args JSON, Headers JSON, Env JSON, and Working directory fields.
- Args JSON is parsed as a JSON array; Headers JSON and Env JSON are parsed as JSON objects.
- Empty advanced fields are omitted from create requests.
- Invalid JSON shows a form error and does not POST.
- Create requests include parsed `args`, `headers`, `env`, and `cwd` when supplied.
- Full header/env secret values are not rendered in instance rows after create.
- Playwright mocked API assertions prove the POST body contains the advanced fields.
- The split MCP page tests wait for async data and do not assert against the loading state.
- Changes are surgical and limited to the documented owned UI files plus workflow/evaluator documentation.
- Workflow status is accurate for Wave 8.AL.

Run gates:
- `npm --prefix ui test -- --run McpSplitPages`
- `npm --prefix ui run e2e -- --grep "MCP"`
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
