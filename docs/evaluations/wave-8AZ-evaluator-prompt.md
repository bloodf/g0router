# g0router Wave 8.AZ Evaluation

Evaluate completed wave `8.AZ` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PLAN.md`
- `docs/PROVIDERS.md`
- `internal/proxy/engine.go`
- `internal/proxy/engine_test.go`
- `internal/provider/matrix.go`
- `internal/provider/matrix_test.go`
- `api/handlers/providers_test.go`
- `internal/cli/providers_test.go`
- `internal/cli/root_test.go`

Start read-only. Do not edit files.

Check:
- Provider-qualified dynamic routing works for deployment-defined adapters:
  - `azure/<deployment>`
  - `litellm/<model>`
  - `lm-studio/<loaded-model>`
  - `vllm/<served-model>`
- Exact catalog matches are resolved before dynamic provider prefixes, so catalog-owned slash models like `openai/gpt-4o-mini` still route through OpenRouter.
- The provider prefix is stripped before upstream dispatch, while the original request model remains unchanged for callers.
- Provider matrix/API/CLI mark Azure, LiteLLM, LM Studio, and vLLM as `supported` public inference providers.
- Replicate remains adapter-only until its public API semantics are proven; it must not be advertised through provider-qualified public routing in this wave.
- The implementation does not add fake static catalog/pricing entries or claim quota support for deployment-defined providers.
- Docs clearly explain provider-qualified dynamic routing and do not overclaim model catalog or quota support.
- Auth-only providers such as Cursor and GitHub Copilot remain excluded from public inference lists.
- The change is surgical and does not alter unrelated provider behavior.

Run gates:
- `go test ./internal/proxy -run 'TestDispatchUsesProviderQualifiedDynamicRouteForDeploymentDefinedProviders|TestDispatchPrefersExactCatalogBeforeProviderQualifiedDynamicRoute|TestDispatchStreamUsesProviderQualifiedDynamicRouteForDeploymentDefinedProviders|TestDispatchRejectsInvalidProviderQualifiedDynamicRoutes' -count=1`
- `go test ./internal/provider -run 'TestProviderMatrixMarksDeploymentDefinedAdaptersAsDynamicPublicRoutes|TestReplicateRemainsAdapterOnlyUntilPublicSemanticsAreProven|TestPublicInferenceProvidersExcludeUnsupportedAndAuthOnlyEntries|TestOpenAICompatibleGatewayProvidersUseDynamicPublicRoutesWithoutFakeCatalogs|TestPublicProvidersDoNotClaimQuotaSupport' -count=1`
- `go test ./api/handlers -run TestProvidersListKnownProviders -count=1`
- `go test ./internal/cli -run 'TestProvidersListShowsKnownProviders|TestProvidersListShowsSupportedInferenceProvidersOnly' -count=1`
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

Whether `docs/WORKFLOW.md`, `docs/PLAN.md`, `docs/ORCHESTRATION.md`, and `docs/PROVIDERS.md` are accurate for Wave 8.AZ.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
