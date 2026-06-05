# g0router Wave 8.CD Evaluation

Evaluate completed wave `8.CD` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Review:
- AGENTS.md
- CLAUDE.md
- README.md
- Makefile
- docs/README.md
- docs/PLAN.md
- docs/WORKFLOW.md
- docs/ORCHESTRATION.md
- docs/evaluations/wave-8CD-evaluator-prompt.md
- Commit for Wave 8.CD: `{commit_ref}`

Start read-only. Do not edit files.

Check:
- `make verify` bootstraps UI dependencies before raw UI test/build/E2E commands.
- A clean checkout can run the documented release gate without first running ad hoc `npm ci`.
- `make build` still installs UI development dependencies and builds the embedded UI plus Go binary.
- README and workflow docs point users/evaluators at the bootstrapped verification target for clean-checkout validation.
- No historical gate records were rewritten as if they had originally used the new target.
- No unrelated files, generated artifacts, or protected local files were changed.

Required gates:

```bash
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
