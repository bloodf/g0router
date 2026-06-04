# g0router Wave 8.AS Evaluation

Evaluate completed wave `8.AS` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PLAN.md`
- `docs/phases/phase-12-advanced-mcp-gateway.md`

Check:
- Phase 12 no longer has the stale `Data Model Plan` heading.
- Phase 12 no longer says the implementation "should add or migrate" the MCP instance storage.
- The replacement wording is still honest: it may describe an implementation contract, but it must not claim exact DDL where the phase doc is only a conceptual sketch.
- Stage 8 wave counts and current wave metadata are accurate for Wave 8.AS.
- No unchecked task boxes were introduced in docs.
- No generated artifacts or unrelated files are changed.

Gates to run:
- `rg -n 'Data Model Plan|Phase 12 should add or migrate' docs/phases/phase-12-advanced-mcp-gateway.md`
- `rg -n -- '- \[ \]' docs/phases docs/WORKFLOW.md docs/PLAN.md docs/ORCHESTRATION.md docs/README.md`
- `git diff --check`

Expected gate behavior: the two `rg` commands should return no matches and exit 1. Treat matches as failures.

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
