# g0router Wave 8.BI Evaluation

Evaluate completed wave `8.BI` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files, do not commit, and do not clean protected local files.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/PROVIDERS.md`
- `internal/providers/types.go`
- `internal/providers/types_test.go`
- `internal/providers/cloudflare/cloudflare.go`
- `internal/providers/cloudflare/cloudflare_test.go`
- `internal/provider/matrix.go`
- `internal/provider/matrix_test.go`
- `internal/proxy/engine.go`
- `internal/proxy/engine_test.go`
- `internal/cli/provider_runtime.go`
- `internal/cli/providers_test.go`
- `internal/cli/root_test.go`
- `api/handlers/providers_test.go`

Check:
- Cloudflare AI Gateway is promoted only where runtime support exists: provider constant, native adapter, startup registration, provider-qualified dynamic route, API/CLI provider visibility, and matrix/docs status.
- The adapter uses the documented Cloudflare REST OpenAI-compatible shape: `/client/v4/accounts/{account_id}/ai/v1/chat/completions`.
- Dispatch propagates stored connection `account_id` into `providers.Key.AccountID`; the adapter rejects missing account id instead of fabricating one.
- Cloudflare dynamic routing strips `cloudflare-ai-gateway/` before upstream dispatch.
- Cloudflare does not claim static model catalog, embedded pricing, model listing, or quota fetcher support.
- Existing `auth_only` providers such as Cursor, Kimi, GitLab, Kiro, and Xiaomi are not accidentally promoted.
- Replicate remains `adapter_only` until its public semantics are proven.
- Tests use local fakes/servers and no external network or real provider credentials.
- Workflow status accurately records Wave 8.BI gate evidence and evaluator status.

Run:
- `go test ./internal/providers ./internal/providers/cloudflare ./internal/provider ./internal/proxy ./internal/cli ./api/handlers -run 'TestKeyCarriesProviderAccountID|TestChatCompletionUsesAccountScopedCloudflareOpenAIEndpoint|TestChatCompletionRequiresAccountID|TestProviderMatrixMarksDeploymentDefinedAdaptersAsDynamicPublicRoutes|TestPublicInferenceProvidersExcludeUnsupportedAndAuthOnlyEntries|TestPublicProvidersDoNotClaimQuotaSupport|TestOpenAICompatibleGatewayProvidersUseDynamicPublicRoutesWithoutFakeCatalogs|TestDispatchUsesProviderQualifiedDynamicRouteForDeploymentDefinedProviders|TestProvidersListShowsKnownProviders|TestProvidersListShowsSupportedInferenceProvidersOnly|TestProvidersListKnownProviders' -count=1`
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
