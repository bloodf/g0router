# g0router Wave 8.BH Evaluation

Evaluate completed wave `8.BH` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files, do not commit, and do not clean protected local files.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/PROVIDERS.md`
- `internal/provider/oauth/qianfan.go`
- `internal/provider/oauth/qianfan_test.go`
- `internal/providers/types.go`
- `internal/providers/openaicompat/registry.go`
- `internal/providers/openaicompat/provider_test.go`
- `internal/provider/matrix.go`
- `internal/provider/matrix_test.go`
- `internal/proxy/engine.go`
- `internal/proxy/engine_test.go`
- `internal/cli/auth.go`
- `internal/cli/provider_runtime.go`
- `internal/cli/providers_test.go`
- `internal/cli/root_test.go`
- `api/handlers/providers_test.go`

Check:
- Qianfan is promoted only where runtime support exists: API-key credential capture, provider constant, default OpenAI-compatible Qianfan config, startup registration, provider-qualified dynamic route, API/CLI provider visibility, and matrix/docs status.
- Qianfan uses API-key Bearer auth through the shared OpenAI-compatible provider and targets `https://api.baiduqianfan.ai/v1`.
- Qianfan dynamic routing strips `qianfan/` from public model IDs before upstream dispatch.
- Qianfan does not claim static model catalog, embedded pricing, or quota fetcher support.
- Existing `auth_only` providers such as Cursor, Kimi, GitLab, Kiro, and Xiaomi are not accidentally promoted.
- Replicate remains `adapter_only` until its public semantics are proven.
- Tests use local fakes/servers and no external network or real provider credentials.
- Workflow status accurately records Wave 8.BH gate evidence and evaluator status.

Run:
- `go test ./internal/provider/oauth ./internal/providers/openaicompat ./internal/provider ./internal/proxy ./internal/cli ./api/handlers -run 'TestQianfanFlow|TestConfiguredProvidersUseOpenAICompatibleEndpoints|TestDefaultConfigsAreRegistered|TestProviderMatrixMarksDeploymentDefinedAdaptersAsDynamicPublicRoutes|TestPublicInferenceProvidersExcludeUnsupportedAndAuthOnlyEntries|TestPublicProvidersDoNotClaimQuotaSupport|TestOpenAICompatibleGatewayProvidersUseDynamicPublicRoutesWithoutFakeCatalogs|TestDispatchUsesProviderQualifiedDynamicRouteForDeploymentDefinedProviders|TestProvidersListShowsKnownProviders|TestProvidersListShowsSupportedInferenceProvidersOnly|TestProvidersListKnownProviders' -count=1`
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
