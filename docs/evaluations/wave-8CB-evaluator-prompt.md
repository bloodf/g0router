# g0router Wave 8.CB Evaluation

Evaluate completed wave `8.CB` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files, do not commit, and do not clean local dirt.

Review:

- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PLAN.md`
- `docs/PROVIDERS.md`
- `docs/SCHEMA.md`
- `docs/phases/phase-08-usage-tracking-cost-logging.md`
- Implementation commit: `ee96357 phase-8/task-openrouter-quota: add real quota fetcher`

Check:

- OpenRouter is the only provider promoted to quota-capable by this wave.
- `internal/usage.OpenRouterQuotaFetcher` calls the current API-key quota endpoint with bearer auth, parses decimal credit values, and treats null upstream limits as unlimited without fabricating a finite remaining value.
- Default `g0router serve` startup wires the OpenRouter quota fetcher, while providers whose matrix `quota=false` still return `usage.ErrQuotaUnsupported`.
- Quota enforcement does not block dispatch when a provider quota response is explicitly unlimited.
- `/api/providers`, provider matrix tests, docs, API schema, and dashboard quota UI agree on the OpenRouter quota contract.
- The dashboard renders unlimited quota responses without showing fake `0 remaining` or `12.25 of 0` copy.
- The implementation remains surgical, has meaningful tests, does not commit secrets, and does not touch protected local dirt such as `.DS_Store`, `.pi/`, or untracked `AGENTS.md`.
- Gates pass:
  - `go test ./... -count=1`
  - `go vet ./...`
  - `go build ./cmd/g0router`
  - `npm --prefix ui test -- --run`
  - `npm --prefix ui run build`
  - `npm --prefix ui run e2e`
  - `make build`
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

Whether `docs/WORKFLOW.md` is accurate for Wave 8.CB.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
