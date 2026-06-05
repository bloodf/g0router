# g0router Wave 8.BP Evaluation

Evaluate completed wave `8.BP` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files, do not commit, and do not clean protected local files.

Review:

- CLAUDE.md
- docs/README.md
- docs/WORKFLOW.md
- docs/ORCHESTRATION.md
- docs/PROVIDERS.md
- docs/PLAN.md
- internal/providers/types.go
- internal/providers/ollamacloud/ollamacloud.go
- internal/providers/ollamacloud/ollamacloud_test.go
- internal/provider/matrix.go
- internal/provider/matrix_test.go
- internal/proxy/engine.go
- internal/proxy/engine_test.go
- internal/cli/provider_runtime.go
- internal/cli/providers_test.go
- internal/cli/root_test.go
- api/handlers/providers.go
- api/handlers/providers_test.go

Check:

- `ollama-cloud` is a distinct g0router provider ID and does not overwrite local `ollama`.
- Ollama Cloud uses native Ollama Cloud endpoints: `/api/chat` for chat and `/api/tags` for model listing.
- Requests send bearer API-key auth and strip the `ollama-cloud/` provider-qualified prefix before upstream dispatch.
- The provider matrix marks Ollama Cloud as public direct-dispatch supported with API-key auth, streaming, and model listing, but no static catalog, pricing, or quota support.
- CLI and `/api/providers` expose Ollama Cloud as a public inference provider.
- Normal startup registers the Ollama Cloud provider.
- Docs accurately describe native Ollama Cloud behavior and do not claim OpenAI-compatible semantics, static catalog, pricing, or quota support.
- No auth-only/search provider is accidentally promoted.

Run gates:

- `go test ./internal/providers/ollamacloud ./internal/provider ./internal/cli ./api/handlers ./internal/proxy -run 'Test(ChatCompletionUsesNativeOllamaCloudChat|ListModelsUsesNativeTagsEndpoint|NewDefaultUsesOllamaCloudProvider|OllamaCloudPublicNativeProvider|ProviderMatrixMarksDeploymentDefinedAdaptersAsDynamicPublicRoutes|PublicInferenceProvidersExcludeUnsupportedAndAuthOnlyEntries|PublicProvidersDoNotClaimQuotaSupport|ProvidersListShowsKnownProviders|DefaultInferenceEngineRegistersOllamaCloudProvider|ProvidersListKnownProviders|DispatchUsesProviderQualifiedDynamicRouteForDeploymentDefinedProviders)' -count=1`
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
