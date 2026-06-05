# g0router Wave 8.CG Evaluation

Evaluate completed wave `8.CG` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

Review:
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/PLAN.md`
- `docs/ORCHESTRATION.md`
- `docs/phases/phase-00-project-bootstrap.md`
- Final audit thread findings recorded in `docs/WORKFLOW.md`

Scope:
- `docs/WORKFLOW.md` must mark the project complete, with no live current wave pretending work remains open.
- `docs/README.md` must not say Stage 8 remains active or direct agents to pick a nonexistent pending task.
- `docs/phases/phase-00-project-bootstrap.md` must not say `g0router serve` is not implemented.
- Stage 8 range references in `docs/PLAN.md` and `docs/ORCHESTRATION.md` must include Wave 8.CG.
- Historical evaluator prompts may contain examples of TODO/PENDING wording; do not treat those archived prompts as live workflow tasks.

Run gates:
- `rg -n 'project_status: COMPLETE|current_wave: "COMPLETE"|last_completed_wave: "8.CG"' docs/WORKFLOW.md`
- `rg -n 'Stage 8 completion hardening remains active|Pick up the next `PENDING` task|not yet implemented' docs/README.md docs/phases/phase-00-project-bootstrap.md && false || true`
- `rg -n '8\.L.*8\.CG|through Wave 8\.CG|through 8\.CG' docs/PLAN.md docs/ORCHESTRATION.md`
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
