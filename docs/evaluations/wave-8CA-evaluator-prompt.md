# g0router Wave 8.CA Evaluation

Evaluate completed wave `8.CA` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/SCHEMA.md`
- `docs/PROVIDERS.md`
- `docs/phases/phase-08-usage-tracking-cost-logging.md`
- `internal/cli/root.go`
- `internal/cli/root_test.go`
- `internal/provider/matrix.go`
- `internal/usage/quota.go`
- Relevant commits for Wave 8.CA

Check:
- Quota docs do not imply every provider has a real provider quota fetcher.
- `/api/usage/quota/:provider` is documented as capability-gated and unsupported for providers without real fetchers.
- Default startup quota fetchers for public inference providers whose matrix `quota=false` return `usage.ErrQuotaUnsupported`.
- Auth-only providers are not incorrectly required to register dispatch quota fetchers.
- Provider matrix/API docs still do not overclaim quota support.
- No provider is promoted to quota-capable by this wave.
- Workflow status is accurate.

Run gates:

```bash
go test ./internal/cli -run TestDefaultQuotaFetchersReturnUnsupportedForQuotaFalseProviders -count=1
go test ./... -count=1
go vet ./...
go build ./cmd/g0router
npm --prefix ui test -- --run
npm --prefix ui run build
npm --prefix ui run e2e
make build
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
