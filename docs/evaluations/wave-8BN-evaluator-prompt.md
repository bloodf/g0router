# g0router Wave 8.BN Evaluation

Evaluate completed wave `8.BN` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files, do not commit, and do not clean protected local files.

Review:

- CLAUDE.md
- docs/README.md
- docs/WORKFLOW.md
- docs/ORCHESTRATION.md
- docs/PROVIDERS.md
- docs/PLAN.md
- internal/providers/types.go
- internal/providers/openaicompat/registry.go
- internal/providers/openaicompat/provider_test.go
- internal/provider/matrix.go
- internal/provider/matrix_test.go
- internal/proxy/engine.go
- internal/proxy/engine_test.go
- internal/cli/provider_runtime.go
- internal/cli/providers_test.go
- internal/cli/root_test.go
- api/handlers/providers_test.go

Check:

- Kilo is represented as g0router provider ID `kilo`.
- Kilo uses the Kilo AI Gateway OpenAI-compatible base URL `https://api.kilo.ai/api/gateway`.
- `kilo/<model>` routes to upstream `<model>` through provider-qualified dynamic routing.
- Public provider API and CLI lists include `kilo`.
- Matrix and docs mark Kilo as supported dynamic inference with API-key auth, streaming, no static catalog, no model listing, and no quota.
- Kilo and Kiro remain distinct: `kilo` is supported dynamic inference and `kiro` remains auth-only.
- No unsupported/auth-only provider is accidentally promoted.
- Changes are surgical and stay within the wave scope.

Run gates:

- `go test ./internal/providers/openaicompat ./internal/provider ./internal/proxy ./internal/cli ./api/handlers -run 'TestKiloDefaultConfigUsesGatewayEndpoint|TestConfiguredProvidersUseOpenAICompatibleEndpoints|TestDefaultConfigsAreRegistered|TestProviderMatrixMarksDeploymentDefinedAdaptersAsDynamicPublicRoutes|TestPublicInferenceProvidersExcludeUnsupportedAndAuthOnlyEntries|TestPublicProvidersDoNotClaimQuotaSupport|TestDeploymentDefinedPublicProvidersExposeDynamicRouting|TestDispatchUsesProviderQualifiedDynamicRouteForDeploymentDefinedProviders|TestProvidersListShowsKnownProviders|TestProvidersListShowsSupportedProvidersOnly|TestProvidersListKnownProviders|TestProvidersLoginListsSupportedProvidersOnly|TestDefaultInferenceEngineRegistersKiloProvider' -count=1`
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
