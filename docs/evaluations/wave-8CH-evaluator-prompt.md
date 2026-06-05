# g0router Wave 8.CH Evaluation

Evaluate completed wave `8.CH` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

Review:
- `docs/WORKFLOW.md`
- `docs/PLAN.md`
- `docs/ORCHESTRATION.md`
- `docs/README.md`
- `docs/phases/*.md`

Scope:
- The live workflow and planning docs must not direct agents to a current wave, pending task, or active Stage 8 work when `project_status` is complete.
- Historical phase or wave names that contain "remaining" are acceptable only when they describe old completed wave contents.
- Archived evaluator prompts are out of scope for this wave.
- Stage 8 range references in `docs/PLAN.md` and `docs/ORCHESTRATION.md` must include Wave 8.CH.

Run gates:
- `rg -n 'project_status: COMPLETE|current_wave: "COMPLETE"|last_completed_wave: "8.CH"' docs/WORKFLOW.md`
- `rg -n 'remains active|next PENDING|PENDING task|current wave|find IN_PROGRESS task|not yet implemented' docs/README.md docs/WORKFLOW.md docs/PLAN.md docs/ORCHESTRATION.md docs/phases --glob '*.md' | rg -v 'gate:|fresh completion audit' && false || true`
- `rg -n '8\.L.*8\.CH|through Wave 8\.CH|through 8\.CH' docs/PLAN.md docs/ORCHESTRATION.md`
- `git diff --check`

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
