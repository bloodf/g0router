# g0router Wave 8.AT Release-Lock Evaluation

Evaluate completed wave `8.AT` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files except for the explicit generated-artifact cleanup command listed below after build gates.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PLAN.md`
- `docs/evaluations/wave-8AT-evaluator-prompt.md`

Check:
- The release-lock wave records the full required gate chain.
- Stage 8 wave counts, total wave counts, current wave, gate results, and files owned are accurate for Wave 8.AT.
- Generated artifacts from UI build, Playwright, and `make build` are not left dirty.
- The leaked MiniMax token from chat is not present in tracked or untracked repo files.
- The unmerged branch audit is honest: stale conflicting branch `codex/wave-8an-dashboard-mcp-oauth-resource-discovery` may remain unmerged, but should not be counted as unfinished valid implementation.
- Only protected local dirt may remain: `.DS_Store`, `docs/.DS_Store`, `.pi/`, and untracked `AGENTS.md`.
- No unchecked task boxes or pending current-wave evaluator entries remain except this Wave 8.AT evaluation before it is recorded.

Gates to run:
- `go test ./... -count=1`
- `go vet ./...`
- `go build ./cmd/g0router`
- `npm --prefix ui test -- --run`
- `npm --prefix ui run build`
- `npm --prefix ui run e2e`
- `make build`
- `rm -rf ui/test-results && git restore ui/dist/assets/index.css ui/dist/assets/index.js ui/dist/index.html 2>/dev/null || true && rm -f g0router`
- `rg -n "sk-cp-|SKsM8|G0ROUTER_E2E_MINIMAX_API_KEY=.*sk|API_KEY_SECRET=.*sk" . --glob '!ui/node_modules/**' --glob '!ui/dist/**' --glob '!ui/test-results/**' --glob '!g0router' --glob '!.git/**' --glob '!docs/evaluations/wave-8AT-evaluator-prompt.md'`
- `git branch --no-merged main --format='%(refname:short) %(objectname:short)'`
- `git diff --check`
- `git status --short`

Expected behavior:
- All build/test commands should exit 0.
- The secret scan should return no matches and exit 1. Treat matches as failures. The evaluator prompt is excluded because it contains the scan expression itself, not a credential.
- `git status --short` should only show protected local dirt listed above after artifact cleanup.

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
