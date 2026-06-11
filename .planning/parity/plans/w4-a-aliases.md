# w4-a — Model aliases & resolution

Rows: PAR-ROUTE-005 (alias map resolution, `src/sse/services/model.js:22-25`, `open-sse/services/model.js:182-208`), PAR-ROUTE-006 (~140 provider alias→ID map, `open-sse/services/model.js:1-143`), PAR-ROUTE-007 PARTIAL→complete (`provider/model` prefix parsing, `model.js:155-176`; g0router has only prefix defaults in `internal/inference/factory.go`), PAR-ROUTE-008 (name-prefix inference when no alias matches), PAR-ROUTE-009 + PAR-ROUTE-040 (provider-node prefixes `openai-compatible-*`/`anthropic-compatible-*`, ref `executors/base.js:30-43`), PAR-ROUTE-010 (circular alias validation) + PAR-PR-485 (`PARITY.md` "Use providerId for passthrough model alias lookup"). Frozen ref @ 827e5c3. Depends: w4-pre MERGED.

Go-port consideration (matrix, verbatim): "Implement alias resolution as a catalog lookup cache with cycle detection (DFS on alias graph at write time)."

## Tasks (STEP (a) named failing tests FIRST; STEP (b) implement)
1. Provider alias table (`internal/providers/catalog/aliases.go` NEW): port the alias→provider-ID map from `model.js:1-143` VERBATIM (count and list pinned by test). `ResolveProviderAlias(alias) (id string, ok bool)`. Tests: `TestProviderAliasCount` (exact count from ref), `TestProviderAliasSamples` (cc→claude etc. byte-exact per ref lines), `TestProviderAliasUnknown`.
2. Model alias store + resolution (`internal/inference/alias.go` NEW + store `internal/store/aliases.go` NEW, additive migrate): aliases map (name→target model) persisted; `ResolveModelAlias` follows chains; CYCLE DETECTION at WRITE time (DFS; reject circular — PAR-ROUTE-010, ref `model.js:182-208` guard). Tests: `TestAliasChainResolution`, `TestAliasCycleRejectedOnWrite`, `TestAliasMissingPassthrough`.
3. Prefix parsing + inference (`internal/inference/alias.go`): `provider/model` and `alias/model` split (`model.js:155-176`); when no alias matches, infer provider from model-name prefix (PAR-ROUTE-008, port the inference order); provider-node prefixes `openai-compatible-{id}`/`anthropic-compatible-{id}` resolve to generic adapter configs with the node's baseUrl (PAR-ROUTE-009/040; consumes catalog). PR-485: passthrough lookup keyed by providerId not display name. Integrate into `providerForModel` (`internal/inference/factory.go` — this plan OWNS factory.go changes). Tests: `TestPrefixParsing` (provider/, alias/, bare), `TestNamePrefixInference`, `TestProviderNodePrefixes`, `TestPassthroughLookupByProviderID` (PR-485), `TestFactoryPrecedenceUnchangedForCatalogModels` (w2-d tests stay green).

## Preconditions
- `grep -rn 'ResolveProviderAlias\|aliases.go' internal/` → 0 hits (new).
- `grep -c 'func providerForModel' internal/inference/factory.go` ≥ 1 (integration point).

## Exclusive file ownership
NEW: `internal/providers/catalog/aliases.go`+test, `internal/inference/alias.go`+test, `internal/store/aliases.go`+test. TOUCH: `internal/inference/factory.go`+`factory_test.go`, `internal/store/migrate.go`. NOT: router.go, api/, errorclass/retry (w4-b), accounts (w4-c).

## Binary acceptance
- `go test ./... && go vet ./...` green; alias count test pins the ref map exactly.
- `TestAliasCycleRejectedOnWrite`, `TestPrefixParsing`, `TestProviderNodePrefixes` pass; all pre-existing factory/router tests pass unchanged.

## Out of scope
Combos (w4-e). Selection (w4-d). Custom models/sub-config exposure (Wave 6, deferred). Disabled-model exclusion (w4-c).
