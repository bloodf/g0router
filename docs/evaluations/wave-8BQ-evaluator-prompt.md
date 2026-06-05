# g0router Wave 8.BQ Evaluation

Evaluate completed wave `8.BQ` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

Review:
- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PROVIDERS.md`
- `internal/provider/ids.go`
- `internal/provider/oauth/types.go`
- `internal/provider/oauth/gitlab.go`
- `internal/provider/matrix.go`
- `internal/cli/auth.go`
- `api/handlers/oauth.go`
- Diff/commits for Wave 8.BQ

Check:
- `gitlab` canonicalizes to `gitlab-duo` in runtime provider IDs and OAuth flow IDs.
- `ProviderAliases("gitlab")` and `ProviderAliases("gitlab-duo")` include both legacy and canonical IDs.
- GitLab OAuth emits provider `gitlab-duo`, uses OMP-style GitLab Duo defaults, scope `api`, and callback `http://localhost:8080/callback`.
- CLI `auth list` advertises `gitlab-duo`, and `auth login gitlab` still resolves to the GitLab Duo flow.
- `/api/oauth/gitlab/*` resolves to the `gitlab-duo` flow and persists a `gitlab-duo` connection without leaking tokens.
- `internal/provider/matrix.go` and `docs/PROVIDERS.md` mark `gitlab-duo` as `auth_only`.
- No GitLab Duo inference adapter or public provider routing was advertised before implementing the direct-access runtime path.
- Existing `cursor`, `kiro`, search, and adapter-only providers were not accidentally promoted.

Required gates:
- `go test ./api/handlers ./internal/provider ./internal/provider/oauth ./internal/cli -run 'Test(OAuthExchangeAcceptsGitLabAliasAndStoresGitLabDuoConnection|CanonicalProviderIDNormalizesRuntimeAliases|ProviderAliasesIncludeLegacyIDs|CanonicalFlowProviderIDNormalizesAuthAliases|CanonicalProviderIDKeepsVertexRuntimeProvider|GitLabFlowStartBuildsPKCEAuthURL|GitLabFlowExchangePostsAuthorizationCode|GitLabFlowPollUnsupported|ConnectionFromOAuthTokenNormalizesGitLabToGitLabDuo|ProviderMatrixCoversRemediationParityTiers|ProviderMatrixMarksOAuthOnlyProvidersExplicitly|PublicInferenceProvidersExcludeUnsupportedAndAuthOnlyEntries|AuthListShowsSupportedProviders|OAuthFlowAcceptsCanonicalProviderAliases)' -count=1`
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
