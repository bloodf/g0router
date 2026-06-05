# g0router Wave 8.BU Evaluation

Evaluate completed wave `8.BU` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

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
- `internal/providers/replicate/replicate.go`
- `internal/providers/replicate/replicate_test.go`
- Diff/commits for Wave 8.BU

Check:
- `replicate` is a supported public direct-dispatch provider, not auth-only.
- Normal server startup registers the native Replicate provider.
- Provider-qualified dynamic routing accepts `replicate/<model>` and strips only the provider prefix.
- Replicate dispatch creates predictions at `/v1/predictions` using the stored API key.
- Prediction creation uses the provider-qualified model suffix as the Replicate model ID.
- Prediction polling follows `urls.get` or `/v1/predictions/{id}` until a terminal status.
- Succeeded prediction output maps string and string-array output into one assistant message.
- Failed/canceled predictions return errors.
- Pending predictions time out without hanging tests.
- Streaming and model listing are explicitly unsupported.
- Docs, provider matrix, CLI, and API provider surfaces agree.
- No OpenAI-compatible Replicate wrapper was reintroduced.
- No unrelated provider was demoted or promoted.
- No secrets are logged, committed, or exposed through provider metadata.

Required gates:
- `go test ./internal/providers/replicate ./internal/proxy ./internal/provider ./api/handlers ./internal/cli -run 'Test(ChatCompletionCreatesAndPollsPrediction|ChatCompletionMapsStringPredictionOutput|ChatCompletionReportsFailedPrediction|ChatCompletionTimesOutPendingPrediction|ChatCompletionStreamUnsupported|ListModelsUnsupported|DispatchUsesProviderQualifiedDynamicRouteForDeploymentDefinedProviders|ReplicatePromotesToPredictionBackedInferenceProvider|ProvidersListShowsKnownProviders|ProvidersTestRequiresActiveConnectionForCredentialProvider|ProvidersListKnownProviders)' -count=1`
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
