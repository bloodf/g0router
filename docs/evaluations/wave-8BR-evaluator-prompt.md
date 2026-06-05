# g0router Wave 8.BR Evaluation

Evaluate completed wave `8.BR` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PROVIDERS.md`
- `internal/provider/matrix.go`
- `internal/cli/provider_runtime.go`
- `internal/cli/auth.go`
- `api/handlers/providers.go`
- Diff/commits for Wave 8.BR

Check:
- Replicate is API-key `auth_only`, not `adapter_only` or `supported`.
- Replicate does not claim registered adapter, inference, streaming, list-models, static catalog, quota, direct dispatch, or public inference.
- Normal server startup no longer registers an unproven Replicate OpenAI-compatible adapter.
- `g0router auth list` still exposes Replicate for API-key credential capture.
- `g0router providers list` does not expose Replicate as a public inference provider.
- `g0router providers test replicate` reports `replicate is auth_only`.
- The deleted `internal/providers/replicate` wrapper is not still referenced by startup code.
- Docs clearly say Replicate needs a future prediction-backed runtime before inference support.
- No unrelated provider was demoted or promoted.

Required gates:
- `go test ./internal/provider ./api/handlers ./internal/cli -run 'Test(ReplicateRemainsAuthOnlyUntilPredictionRuntimeIsImplemented|ProvidersListKnownProviders|AuthListShowsSupportedProviders|LoginCommandPersistsSearchProviderAPIKeyConnection|ProvidersTestReportsAuthOnlyProvider)' -count=1`
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

Whether `docs/WORKFLOW.md` is accurate.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
