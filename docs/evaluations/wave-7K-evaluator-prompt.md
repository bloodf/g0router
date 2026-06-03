# g0router Wave 7.K Evaluation Prompt

Evaluate completed Wave `7.K` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

## Review Inputs

- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/DEPLOYMENT.md`
- `README.md`
- Previous evaluator prompt and findings:
  - `docs/evaluations/wave-7J-evaluator-prompt.md`
- Wave 7.K commit:
  - `phase-7/task-k1: fix release hygiene findings`

## Checks

- Frontend build output:
  - `ui/vite.config.ts` uses deterministic output file names.
  - `npm --prefix ui run build` does not generate fresh hashed asset names.
  - Running the UI build twice does not churn tracked `ui/dist` output.
  - Embedded UI tests still pass with the committed `ui/dist` output.
- Auth documentation:
  - `README.md` and `docs/DEPLOYMENT.md` clearly state that `API_KEY_SECRET` validates gateway/dashboard control-plane API keys.
  - Docs do not imply `JWT_SECRET` is the dashboard control-plane API key secret.
  - Docker first-key bootstrap still uses the same `API_KEY_SECRET` as the running container.
- MCP delete ordering:
  - MCP instance DELETE removes the store row before closing the live runtime.
  - If store deletion fails, runtime is not closed and live tools are not orphaned behind a stale row.
  - Existing successful delete behavior still closes runtime and removes tools.
- Workflow:
  - `docs/WORKFLOW.md` records Wave 7.K as a post-7.J hygiene remediation with evaluation pending.
  - Protected local dirt remains uncommitted: `.DS_Store`, `docs/.DS_Store`, `.pi/`, and untracked `AGENTS.md`.

## Required Gates

Run:

```bash
go test ./... -count=1
go vet ./...
go build ./cmd/g0router
npm --prefix ui test -- --run
npm --prefix ui run build
make build
JWT_SECRET=test-jwt API_KEY_SECRET=test-api docker compose config
docker build -t g0router:wave-7k-eval .
```

Also run a repeat-build stability check:

```bash
npm --prefix ui run build
git status --short ui/dist ui/vite.config.ts
```

The repeat-build status must not show new tracked `ui/dist` churn after the committed deterministic output is present.

## Return Format

## Verdict

PASS or FAIL

## Blocking Findings

Issues that must be fixed before declaring remediation complete.

## Non-Blocking Findings

Risks or cleanup notes.

## Gate Results

Command results with exact failures if any.

## Workflow Status Review

Whether `docs/WORKFLOW.md` accurately reflects Wave 7.K completion and evaluation-pending state.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
