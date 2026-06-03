# g0router Wave 7.I Remediation Evaluation

Evaluate Wave 7.I quota remediation in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit, stage, commit, clean files, remove worktrees, archive threads, or change branches. Existing local dirt `.DS_Store`, `docs/.DS_Store`, `.pi/`, and untracked `AGENTS.md` must be ignored and not cleaned.

## Source Of Truth

Original Wave 7.I evaluator prompt:

- `docs/evaluations/wave-7I-evaluator-prompt.md`

Failed evaluator finding:

- Explicit quota exhaustion was treated as fallback input instead of a hard stop.

Remediation commits:

- `94f3d09 phase-7/task-i5: fix quota exhaustion hard stop`
- `3ca480d merge wave 7i quota remediation`

## Check

- Explicit `ErrQuotaExhausted` from quota fetchers returns an error wrapping `ErrQuotaExhausted`.
- Zero or negative quota `Remaining` returns an error wrapping `ErrQuotaExhausted`.
- Explicit quota exhaustion does not invoke the selected provider.
- Explicit quota exhaustion does not try another connection/account.
- Explicit quota exhaustion does not advance to the next combo step.
- Explicit quota exhaustion does not open a fallback combo stream.
- Quota exhaustion does not record fallback/backoff as if it were an upstream provider failure.
- `usage.ErrQuotaUnsupported` and transient quota fetcher errors still fail open.
- Existing alias, prefix, combo, and streaming quota tests still prove quota fetchers receive the selected provider key/connection.
- Workflow status accurately records the failed evaluator remediation.

## Gates

Run:

```bash
go test ./internal/proxy -count=1
go test ./... -count=1
go vet ./...
go build ./cmd/g0router
npm --prefix ui test -- --run
npm --prefix ui run build
make build
```

If a gate command modifies tracked files, report it. Do not stage those changes.

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
