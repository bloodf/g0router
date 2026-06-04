# g0router Wave 8.BA Evaluation

Evaluate completed wave `8.BA` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PLAN.md`
- `docs/PROVIDERS.md`
- `internal/providers/openaicompat/provider.go`
- `internal/providers/openaicompat/registry.go`
- `internal/providers/openaicompat/provider_test.go`
- `internal/proxy/engine.go`
- `internal/proxy/engine_test.go`
- `internal/provider/matrix.go`
- `internal/provider/matrix_test.go`
- `api/handlers/providers_test.go`
- `internal/cli/provider_runtime.go`
- `internal/cli/providers_test.go`
- `internal/cli/root_test.go`

Start read-only. Do not edit files.

Check:
- GitHub Copilot is registered through the OpenAI-compatible adapter with base URL `https://api.githubcopilot.com`.
- Copilot requests include the OMP-compatible `User-Agent: opencode/1.3.15` header without dropping bearer authorization.
- `github-copilot/<model>` routes through provider-qualified dynamic dispatch and strips the provider prefix before upstream dispatch.
- Copilot is public-supported in provider matrix/API/CLI, but without fake static catalog or quota support.
- Cursor and other auth-only providers remain explicitly not advertised as public inference providers.
- The change does not alter unrelated provider routing, exact catalog precedence, or deployment-defined dynamic routing.
- Docs accurately describe Copilot status and do not claim unsupported Cursor or remaining OMP providers are complete.
- No provider token, API key, or leaked MiniMax credential appears in source, docs, tests, logs, or command output.

Run gates:
- `go test ./internal/providers/openaicompat -run 'TestConfiguredProvidersUseOpenAICompatibleEndpoints|TestGitHubCopilotDefaultProviderSendsOMPHeaders|TestDefaultConfigsAreRegistered' -count=1`
- `go test ./internal/proxy -run 'TestDispatchUsesProviderQualifiedDynamicRouteForDeploymentDefinedProviders|TestDispatchStreamUsesProviderQualifiedDynamicRouteForDeploymentDefinedProviders|TestDispatchPrefersExactCatalogBeforeProviderQualifiedDynamicRoute|TestDispatchRejectsInvalidProviderQualifiedDynamicRoutes' -count=1`
- `go test ./internal/provider -run 'TestProviderMatrixMarksAuthOnlyProvidersExplicitly|TestProviderMatrixMarksDeploymentDefinedAdaptersAsDynamicPublicRoutes|TestPublicInferenceProvidersExcludeUnsupportedAndAuthOnlyEntries|TestPublicProvidersDoNotClaimQuotaSupport|TestOpenAICompatibleGatewayProvidersUseDynamicPublicRoutesWithoutFakeCatalogs|TestProviderMatrixSupportedEntriesHaveUsableSurface' -count=1`
- `go test ./api/handlers -run TestProvidersListKnownProviders -count=1`
- `go test ./internal/cli -run 'TestProvidersListShowsKnownProviders|TestProvidersListShowsSupportedInferenceProvidersOnly|TestProvidersTestReportsAuthOnlyProvider' -count=1`
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

Whether `docs/WORKFLOW.md`, `docs/PLAN.md`, `docs/ORCHESTRATION.md`, and `docs/PROVIDERS.md` are accurate for Wave 8.BA.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
