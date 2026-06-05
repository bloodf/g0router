# g0router Wave 8.CI Evaluation

Evaluate completed wave `8.CI` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

Review:

- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/PLAN.md`
- `docs/ORCHESTRATION.md`
- `docs/WORKFLOW.md`
- `ui/src/pages/McpSplitPages.test.tsx`
- Diff/commits: `ff51b3e`, workflow record commit `25b5e48`, E2E fix commit `6d02935`, and the workflow record commit that updates the evaluator result

Check:

- `docs/WORKFLOW.md` reports `project_status: COMPLETE`, `current_wave: "COMPLETE"`, and `last_completed_wave: "8.CI"`.
- `docs/README.md`, `docs/PLAN.md`, and `docs/ORCHESTRATION.md` all describe completion through Wave 8.CI.
- The MCP accounts split-page tests wait for loaded OAuth controls instead of racing on the page-level heading.
- The dashboard E2E MCP instance create-flow test waits for the mocked create request before asserting the refreshed MCP instances table.
- The Wave 8.CI workflow record accurately describes the focused test and full `make verify` gate.
- No unrelated files were changed except protected local dirt that must remain ignored by the evaluation.
- Gates pass:
  - `npm --prefix ui test -- --run src/pages/McpSplitPages.test.tsx`
  - `make verify`
  - `git diff --check`

Return:

## Verdict

PASS or FAIL

## Blocking Findings

Issues that must be fixed before marking Wave 8.CI complete.

## Non-Blocking Findings

Risks or cleanup notes.

## Gate Results

Command results with exact failures if any.

## Workflow Status Review

Whether `docs/WORKFLOW.md` is accurate.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
