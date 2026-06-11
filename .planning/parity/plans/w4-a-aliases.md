# w4-a — Model & provider aliases + prefix resolution

Rows: PAR-ROUTE-005 (model alias map resolution, `open-sse/services/model.js:182-208`), PAR-ROUTE-006 (~140 provider alias→ID map, `open-sse/services/model.js:1-143`), PAR-ROUTE-007 PARTIAL→complete (`provider/model` & `alias/model` prefix parsing, `model.js:155-176`), PAR-ROUTE-008 (model-name prefix inference when no alias, `model.js`), PAR-ROUTE-010 (circular alias validation — NOTE: ref has NO implementation; `docs/ARCHITECTURE.md:17` + 9router Phase-9 PLAN list it as a RISK; g0router implements it DEFENSIVELY at write time, not as a ref port) + PAR-PR-485 (`PARITY.md` "Use providerId for passthrough model alias lookup"). Frozen ref @ 827e5c3. Depends: w4-pre MERGED. Parallel-safe with w4-b/w4-c.

Integration point: `internal/inference/factory.go` `providerForModel` (w2-d moved routing here; the matrix's `router.go:35-54` cite predates w2-d — current source is factory.go). Go-port consideration (verbatim): "Implement alias resolution as a catalog lookup cache with cycle detection (DFS on alias graph at write time)."

## Tasks (STEP (a) failing tests FIRST, run, show fail; STEP (b) implement)
1. **Provider alias table** (`internal/providers/catalog/aliases.go` NEW). (a) `TestProviderAliasCount` (exact count from `model.js:1-143`), `TestProviderAliasSamples` (byte-exact entries per ref lines), `TestProviderAliasUnknown`. (b) port the map VERBATIM; `ResolveProviderAlias(alias)(id string,ok bool)`.
2. **Model alias store + resolution** (`internal/inference/alias.go` NEW + `internal/store/aliases.go` NEW, additive migrate). (a) `TestAliasChainResolution`, `TestAliasCycleRejectedOnWrite` (g0router-own defensive, PAR-ROUTE-010), `TestAliasMissingPassthrough`. (b) name→target persisted; `ResolveModelAlias` follows chains; DFS cycle detection at WRITE (reject; this is g0router-defensive per the row note, not a ref port).
3. **Prefix parsing + inference + PR-485** (`internal/inference/alias.go`, integrate into `factory.go providerForModel`). (a) `TestPrefixParsing` (`provider/model`, `alias/model`, bare — `model.js:155-176`), `TestNamePrefixInference` (008), `TestPassthroughLookupByProviderID` (PR-485), `TestFactoryCatalogPrecedenceUnchanged` (all w2-d factory/router tests stay green). (b) implement split + inference order from ref; passthrough lookup keyed by providerId.

## Preconditions
- `grep -rn 'ResolveProviderAlias\|aliases.go' internal/` → 0 hits.
- `grep -c 'func providerForModel' internal/inference/factory.go` ≥ 1.

## Exclusive file ownership
NEW: `internal/providers/catalog/aliases.go`+test, `internal/inference/alias.go`+test, `internal/store/aliases.go`+test. TOUCH: `internal/inference/factory.go`+`factory_test.go`, `internal/store/migrate.go`.

## Binary acceptance
- `go test ./... && go vet ./...` green; alias count test pins the ref map.
- TestAliasCycleRejectedOnWrite, TestPrefixParsing, TestPassthroughLookupByProviderID pass; pre-existing factory/router tests unchanged.

## Out of scope
PAR-ROUTE-009/040 provider-NODE routing (dynamic user-configured custom providers via `/api/provider-nodes` — g0router has NO provider-node domain; that subsystem is **Stage-2/Wave-6**, deferred). Combos (w4-e). Selection (w4-d). Disabled models (w4-c).
