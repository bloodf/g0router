# g0router Wave 8.BK Evaluation

Evaluate completed wave `8.BK` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files, do not commit, and do not clean protected local files.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/PROVIDERS.md`
- `internal/providers/types.go`
- `internal/providers/openaicompat/registry.go`
- `internal/providers/openaicompat/provider_test.go`
- `internal/provider/matrix.go`
- `internal/provider/matrix_test.go`
- `internal/proxy/engine.go`
- `internal/proxy/engine_test.go`
- `internal/cli/provider_runtime.go`
- `internal/cli/providers_test.go`
- `internal/cli/root_test.go`
- `api/handlers/providers_test.go`

Check:
- Kimi is promoted only where runtime support exists: provider constant, OpenAI-compatible registry config, startup registration, provider-qualified dynamic route, API/CLI provider visibility, and matrix/docs status.
- The adapter base URL is `https://api.moonshot.ai/v1` and uses normal `/v1/chat/completions` and `/v1/models` OpenAI-compatible paths.
- Dynamic routing strips `kimi/` before upstream dispatch, so `kimi/kimi-k2.6` reaches the provider as `kimi-k2.6`.
- Kimi remains non-catalog and non-quota; no fake static pricing or quota support is claimed.
- Existing auth-only providers such as Cursor, GitLab, Kiro, and Xiaomi are not accidentally promoted.
- Replicate remains `adapter_only` until its public semantics are proven.
- Tests use local fakes/servers and no external network or real provider credentials.
- Workflow status accurately records Wave 8.BK gate evidence and evaluator status.

Run:
- `go test ./internal/provider ./internal/providers/openaicompat ./internal/proxy ./internal/cli ./api/handlers -run 'TestProviderMatrixMarksAuthOnlyProvidersExplicitly|TestProviderMatrixMarksDeploymentDefinedAdaptersAsDynamicPublicRoutes|TestPublicInferenceProvidersExcludeUnsupportedAndAuthOnlyEntries|TestPublicProvidersDoNotClaimQuotaSupport|TestOpenAICompatibleGatewayProvidersUseDynamicPublicRoutesWithoutFakeCatalogs|TestConfiguredProvidersUseOpenAICompatibleEndpoints|TestDefaultConfigsAreRegistered|TestDispatchUsesProviderQualifiedDynamicRouteForDeploymentDefinedProviders|TestProvidersListKnownProviders|TestProvidersListShowsKnownProviders|TestProvidersListShowsSupportedInferenceProvidersOnly' -count=1`
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

Whether workflow/docs status is accurate.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
