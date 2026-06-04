# g0router Wave 8.BG Evaluation

Evaluate completed wave `8.BG` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files, do not commit, and do not clean protected local files.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/PROVIDERS.md`
- `internal/providers/types.go`
- `internal/providers/openaicompat/provider.go`
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
- Alibaba is promoted only where runtime support exists: provider constant, default OpenAI-compatible DashScope config, startup registration, provider-qualified dynamic route, API/CLI provider visibility, and matrix/docs status.
- Zhipu is promoted only where runtime support exists: provider constant, default Z.AI config, startup registration, provider-qualified dynamic route, API/CLI provider visibility, and matrix/docs status.
- Zhipu uses the documented `/api/paas/v4/chat/completions` shape without appending an extra `/v1` segment.
- Alibaba and Zhipu do not claim static model catalog, embedded pricing, or quota fetcher support.
- Existing `auth_only` providers such as Cursor, Kimi, GitLab, Kiro, and Xiaomi are not accidentally promoted.
- Replicate remains `adapter_only` until its public semantics are proven.
- Tests use local fakes/servers and no external network or real provider credentials.
- Workflow status accurately records Wave 8.BG gate evidence and evaluator status.

Run:
- `go test ./internal/providers/openaicompat ./internal/provider ./internal/proxy ./internal/cli ./api/handlers -run 'TestConfiguredProvidersUseOpenAICompatibleEndpoints|TestZhipuDefaultProviderUsesDocumentedPaaSPathWithoutV1Prefix|TestDefaultConfigsAreRegistered|TestProviderMatrixMarksAuthOnlyProvidersExplicitly|TestProviderMatrixMarksDeploymentDefinedAdaptersAsDynamicPublicRoutes|TestPublicInferenceProvidersExcludeUnsupportedAndAuthOnlyEntries|TestPublicProvidersDoNotClaimQuotaSupport|TestOpenAICompatibleGatewayProvidersUseDynamicPublicRoutesWithoutFakeCatalogs|TestDispatchUsesProviderQualifiedDynamicRouteForDeploymentDefinedProviders|TestProvidersListShowsKnownProviders|TestProvidersListShowsSupportedInferenceProvidersOnly|TestProvidersListKnownProviders' -count=1`
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
