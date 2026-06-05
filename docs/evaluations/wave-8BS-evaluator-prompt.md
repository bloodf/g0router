# g0router Wave 8.BS Evaluation

Evaluate completed wave `8.BS` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PROVIDERS.md`
- `internal/provider/matrix.go`
- `internal/proxy/engine.go`
- `internal/cli/provider_runtime.go`
- `internal/providers/gitlabduo/gitlabduo.go`
- `internal/providers/anthropic/anthropic.go`
- Diff/commits for Wave 8.BS

Check:
- `gitlab-duo` is a supported public direct-dispatch provider, not auth-only.
- Normal server startup registers the GitLab Duo runtime provider.
- Provider-qualified dynamic routing accepts `gitlab-duo/<model>`.
- GitLab Duo dispatch exchanges the stored GitLab OAuth token for a direct-access token through `/api/v4/ai/third_party_agents/direct_access`.
- Direct-access requests include `DuoAgentPlatformNext`.
- GitLab-provided direct-access headers are forwarded to OpenAI and Anthropic proxy calls.
- OpenAI Duo aliases route through `/ai/v1/proxy/openai/v1/chat/completions`.
- Anthropic Duo aliases route through `/ai/v1/proxy/anthropic/v1/messages`.
- Direct-access tokens are cached without adding mutable package global state.
- Unsupported Duo aliases fail closed.
- `ListModels` exposes deterministic Duo aliases without claiming a static priced catalog or quota fetcher.
- Docs, provider matrix, CLI, and API provider surfaces agree.
- No unrelated provider was demoted or promoted.
- No secrets are logged, committed, or exposed through provider metadata.

Required gates:
- `go test ./internal/providers/anthropic ./internal/providers/gitlabduo ./internal/provider ./internal/cli ./api/handlers -run 'Test(NewForProviderWithHeadersAddsProviderHeaders|ChatCompletionExchangesDirectAccessAndRoutesOpenAIModel|ChatCompletionRoutesAnthropicModelWithDirectAccessHeaders|ChatCompletionCachesDirectAccessToken|ChatCompletionRejectsUnsupportedModel|ListModelsReturnsDuoAliasesDeterministically|GitLabDuoPublicDynamicProvider|ProviderMatrixMarksDeploymentDefinedAdaptersAsDynamicPublicRoutes|PublicInferenceProvidersExcludeUnsupportedAndAuthOnlyEntries|ProvidersListShowsKnownProviders|ProvidersListKnownProviders)' -count=1`
- `go test ./... -count=1`
- `go vet ./...`
- `go build ./cmd/g0router`
- `npm --prefix ui test -- --run`
- `npm --prefix ui run build`
- `npm --prefix ui run e2e`
- `make build`
- `git diff --check`

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

Whether `docs/WORKFLOW.md` is accurate.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
