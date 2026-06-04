# g0router Wave 8.BL Evaluation

Evaluate completed wave `8.BL` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files, do not commit, and do not clean protected local files.

Important commit:
- Wave 8.BL implementation commit after orchestration merge

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/PROVIDERS.md`
- `docs/PLAN.md`
- `docs/ORCHESTRATION.md`
- `internal/providers/types.go`
- `internal/providers/anthropic/anthropic.go`
- `internal/providers/xiaomi/xiaomi.go`
- `internal/providers/xiaomi/xiaomi_test.go`
- `internal/provider/matrix.go`
- `internal/provider/matrix_test.go`
- `internal/proxy/engine.go`
- `internal/proxy/engine_test.go`
- `internal/cli/provider_runtime.go`
- `internal/cli/providers_test.go`
- `internal/cli/root_test.go`

Check:
- Xiaomi is promoted only where runtime support exists: provider constant, default startup registration, provider-qualified dynamic route, CLI visibility, matrix/docs status, and focused tests.
- Xiaomi dispatch uses the existing Anthropic-compatible request/stream translation under runtime provider `xiaomi`, not provider `anthropic`.
- Standard Xiaomi keys use the standard Anthropic-compatible base URL under `https://api.xiaomimimo.com/anthropic`.
- `tp-` token-plan keys use the OMP token-plan endpoint under `https://token-plan-ams.xiaomimimo.com/anthropic`.
- Dynamic routing strips `xiaomi/` before upstream dispatch, so `xiaomi/claude-sonnet-4` reaches the provider as `claude-sonnet-4`.
- Xiaomi remains non-catalog, non-listing, and non-quota; no fake static pricing, fake model catalog, or quota support is claimed.
- Cursor, GitLab, and Kiro remain `auth_only`; Replicate remains `adapter_only`; unsupported tool/search providers remain unsupported.
- Tests use local fakes/servers and no external network or real provider credentials.
- Workflow status accurately records Wave 8.BL gate evidence and evaluator status.

Run:
- `go test ./internal/providers/xiaomi ./internal/provider ./internal/proxy ./internal/cli -run 'TestProviderRoutesStandardKeysToXiaomiAnthropicEndpoint|TestProviderRoutesTokenPlanKeysToTokenPlanEndpoint|TestProviderMatrixMarksDeploymentDefinedAdaptersAsDynamicPublicRoutes|TestPublicInferenceProvidersExcludeUnsupportedAndAuthOnlyEntries|TestPublicProvidersDoNotClaimQuotaSupport|TestDispatchUsesProviderQualifiedDynamicRouteForDeploymentDefinedProviders|TestProvidersListShowsKnownProviders|TestProvidersListShowsSupportedInferenceProvidersOnly' -count=1`
- `go test ./... -count=1`
- `go vet ./...`
- `go build ./cmd/g0router`
- `npm --prefix ui test -- --run`
- `npm --prefix ui run build`
- `npm --prefix ui run e2e`
- `make build`
- `git diff --check`
- `git status --short`

Return exactly:

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
