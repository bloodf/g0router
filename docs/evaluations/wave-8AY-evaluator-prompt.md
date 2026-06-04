# g0router Wave 8.AY Evaluation

Evaluate completed wave `8.AY` in `/Users/heitor/Developer/github.com/bloodf/g0router`.

Start read-only. Do not edit files.

Review:

- `AGENTS.md`
- `CLAUDE.md`
- `docs/README.md`
- `docs/WORKFLOW.md`
- `docs/ORCHESTRATION.md`
- `docs/PLAN.md`
- `docs/PROVIDERS.md`
- `internal/modelcatalog/catalog.go`
- `internal/provider/matrix.go`
- `internal/proxy/engine.go`
- `internal/providers/bedrock/bedrock.go`
- Diff/commits for Wave 8.AY

Check:

- Bedrock is promoted from `adapter_only` to `supported` only for catalog-backed non-streaming Converse dispatch.
- `anthropic.claude-3-5-haiku-20241022-v1:0` is a real Bedrock Runtime Converse model ID, not a placeholder.
- The catalog route maps that Bedrock model to `providers.ProviderBedrock` without rewriting the upstream model ID.
- The embedded pricing is explicit and non-zero.
- Matrix, `/api/providers`, and `g0router providers list` all agree that Bedrock is public direct-dispatch capable.
- Bedrock does not overclaim unsupported capabilities: streaming remains false and quota remains false.
- Existing explicit alias/combo Bedrock dispatch remains valid.
- Docs accurately state Bedrock public catalog routing while preserving streaming/quota caveats.
- No unrelated providers were promoted or removed.
- No mocks, `init()` functions, mutable globals, or unrelated refactors were added.

Run gates:

```bash
go test ./internal/modelcatalog -run 'TestCatalogRouteForBedrockConverseModel|TestCatalogIncludesRepresentativeWave7IProviderCoverage|TestCatalogHostedModelsHaveExplicitNonZeroRates|TestCatalogOmitsProvidersWithoutDefensibleEmbeddedPricing' -count=1
go test ./internal/provider -run 'TestProviderMatrix.*Bedrock|TestPublicInferenceProvidersExcludeUnsupportedAndAuthOnlyEntries|TestPublicProvidersDoNotClaimQuotaSupport' -count=1
go test ./internal/proxy -run 'TestDispatchUsesCatalogForBedrockConverseModel|TestDispatchUsesBedrockAliasThroughAdapterOnlyInference|TestComboDispatchUsesBedrockAdapterOnlyStep' -count=1
go test ./api/handlers -run TestProvidersListKnownProviders -count=1
go test ./internal/cli -run 'TestProvidersListShowsKnownProviders|TestProvidersListShowsSupportedInferenceProvidersOnly' -count=1
go test ./... -count=1
go vet ./...
go build ./cmd/g0router
npm --prefix ui test -- --run
npm --prefix ui run build
npm --prefix ui run e2e
make build
```

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

Whether `docs/WORKFLOW.md` accurately reflects Wave 8.AY completion and evaluation-pending state.

## Suggested Fix Prompt

If FAIL, provide a surgical prompt for a fix worker with files to edit and required verification.
