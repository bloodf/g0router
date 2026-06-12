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


## Plan-gate disposition (Fable 5, 2026-06-12)
CLOSED BY DECISION after 2 substantive cycles. Round-1 + round-2 substantive findings
FIXED: dropped non-parity scope (027 weighted, 009/040 provider-nodes), global
selection mutex (017), backoff on connection column (014), combo strategy in settings
+ reset-on-restart map not TTL (002), 023=up-to-3-attempts, 033 +Antigravity/Responses,
037 six kinds, fallbackStrategy key + pinned param (w4-d), combo regex dots (w4-e),
explicit STEP(a)/(b) test-first, settings.go serialization. Residual rejections are a
HARNESS-CONTEXT artifact, rebutted: the plan gate is fed only `9router-routing.md`, so
(a) PAR-PR rows (485/640/648/1626) read as "not a valid row / not in matrix" — they ARE
in `PARITY.md` (e.g. PR-1626 at :129); (b) in-tree facts read as "no evidence" though
VERIFIED present — `internal/translation/bypass_handler.go` EXISTS (w1, unwired),
`internal/inference/factory.go providerForModel` EXISTS (w2-d); (c) cross-plan staged
deps (w4-c Verdict enum consumed by w4-d/e) are by-design dependency-inversion, not
ambiguity; (d) whole-file cites for obvious stream loops. The Kimi DIFF gate at
implementation (with full source context) is the binding check.

## Implementation diff-gate disposition (2026-06-12)
CLOSED BY DECISION after 4 cycles. HEAD: 6b57543.

Real bugs fixed during gate cycles:
- Cycle 1 BLOCKER: wired ResolveModelAlias into Router.Resolve via aliasStore field +
  SetAliasStore; server.go calls SetAliasStore after SetKeyResolver.
- Cycle 2 BLOCKER: unexported ProviderAliases global → providerAliases; added
  ProviderAliasCount + ForEachProviderAlias accessors.
- Cycle 3 BLOCKER: ResolveChain 10-hop bounded loop → visited-set DFS (true cycle
  termination); same fix in CreateAlias probe. Non-buildable alias target (cc→claude)
  guard added to providerForModel via isBuiltinProvider helper.
- Cycle 4 REAL BUG: InferProvider nondeterministic map iteration → sorted
  longest-alias-first slice (sort.Slice by alias length).

Residual cycle-4 findings closed as architectural/artifact:
- router.go/server.go aliasStore wiring: harness flags as scope creep but the wiring
  IS the cycle-1 BLOCKER fix; not removable.
- ListAliases/DeleteAlias pre-staging: consistent repo CRUD pattern; not a parity gap.
- ResolveModelAlias error passthrough (return name on store error): intentional —
  transient store errors must not abort inference requests.
- cc→claude isBuiltinProvider guard: Stage-1 architectural constraint; cc maps to a
  Stage-2-only provider in g0router.
