# g0router Wave 8.BO Evaluation

Evaluate completed wave `8.BO` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files, do not commit, and do not clean protected local files.

Review:

- CLAUDE.md
- docs/README.md
- docs/WORKFLOW.md
- docs/ORCHESTRATION.md
- docs/PROVIDERS.md
- docs/PLAN.md
- internal/provider/matrix.go
- internal/provider/matrix_test.go
- internal/cli/auth.go
- internal/cli/auth_test.go
- api/handlers/providers.go
- api/handlers/providers_test.go

Check:

- Kagi and Tavily are represented as g0router provider IDs `kagi` and `tavily`.
- Both are `auth_only` with API-key auth only.
- CLI `auth list` includes both providers.
- CLI `login <provider> --key --api-key ...` persists redacted API-key connections for both providers.
- `/api/providers` reports both providers as auth-only and does not mark registered adapter, inference, public inference, direct dispatch, streaming, model catalog, model listing, or quota support.
- Docs clearly state this is credential capture for future search tooling, not a runtime search or inference implementation.
- Public inference provider lists do not include Kagi or Tavily.
- No unsupported/auth-only provider is accidentally promoted to inference.
- Changes are surgical and stay within the wave scope.

Run gates:

- `go test ./internal/provider ./internal/cli ./api/handlers -run 'TestProviderMatrixMarksSearchCredentialsAuthOnly|TestAuthListShowsSupportedProviders|TestLoginCommandPersistsSearchProviderAPIKeyConnection|TestProvidersListKnownProviders' -count=1`
- `go test ./... -count=1`
- `go vet ./...`
- `go build ./cmd/g0router`
- `npm --prefix ui test -- --run`
- `npm --prefix ui run build`
- `npm --prefix ui run e2e`
- `make build`
- `git diff --check`
- `git status --short`

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

Whether `docs/WORKFLOW.md`, `docs/PLAN.md`, `docs/ORCHESTRATION.md`, and `docs/PROVIDERS.md` are accurate.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
