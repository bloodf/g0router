# g0router Wave 8.AW Evaluation

Evaluate completed wave `8.AW` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files, do not commit, and do not clean protected local files.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PLAN.md`
- `docs/PROVIDERS.md`
- `internal/providers/bedrock/bedrock.go`
- `internal/providers/bedrock/bedrock_test.go`
- `internal/provider/matrix.go`
- `internal/provider/matrix_test.go`
- `api/handlers/providers_test.go`

Important commit:
- Implementation commit: `d088f8f phase-8/task-provider: implement bedrock model listing`

Check:
- Bedrock `ListModels` no longer returns the unsupported stub.
- `ListModels` signs `GET /foundation-models` with the existing SigV4 path and propagates session tokens.
- Tests use a local HTTP server, not mocks or external network.
- Parsed Bedrock foundation model summaries return `providers.Model` values with provider `bedrock`.
- Provider matrix and `/api/providers` expose Bedrock `ListModels=true` while keeping public inference, direct dispatch, inference, streaming, catalog routing, and quota false.
- Docs do not overclaim Bedrock Converse, streaming, quota, model catalog routing, or public direct dispatch.
- No secrets are logged or serialized.
- No `init()` functions, mutable globals, speculative abstractions, or unrelated refactors were added.

Run:
- `go test ./internal/providers/bedrock -run TestListModelsSignsAndParsesFoundationModels -count=1`
- `go test ./internal/providers/bedrock -count=1`
- `go test ./internal/provider -count=1`
- `go test ./api/handlers -run 'TestProvidersMatrixExposesCapabilityStatus|TestProvidersListModelsForProvider' -count=1`
- `go test ./... -count=1`
- `go vet ./...`
- `go build ./cmd/g0router`
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
