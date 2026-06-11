# w2-d — Provider registry/factory + catalog-driven router + /v1/models

Rows: completes reachability for PAR-PROV-004 (groq), 005 (deepseek), 006 (mistral), 007 (cohere), 008 (together), 009 (fireworks), 010 (ollama), 014 (openrouter), 027 (xai), 029 (perplexity) — these flip HAVE after w2-d (the adapters from w2-b/w2-c become routable + visible in `/v1/models`). w2-a/b/c build the catalog+adapters; w2-d wires routing so a request reaches them.

Frozen ref (@ 827e5c3): `open-sse/services/provider.js` (model→provider resolution + getExecutor dispatch), `open-sse/executors/index.js:46-52` (`getExecutor` factory). In-repo to REPLACE: `internal/inference/router.go:1-63` (the Phase-5 prefix stub — 3 hardcoded providers), `internal/api/models.go:21-40` (single-provider ListModels).

Depends on w2-a (catalog), w2-b (generic adapter), w2-c (ollama adapter) ALL merged.

## Scope decisions (read first)

- **Single-provider catalog resolution ONLY.** `Resolve(model)` returns the ONE
  provider whose catalog lists that model. Combo chains / fallback / rate-limit
  rotation / model aliases are **Wave 4** — explicitly out of scope.
- **Keys unchanged.** Key sourcing stays as today (management layer supplies
  `Key.Value`; empty → provider auth error). w2-d does NOT add a key store (existing/
  later). It only routes to the correct provider + passes the Key through.
- **Existing providers preserved.** openai/anthropic/gemini (the 3 current HAVE
  providers) keep their dedicated packages; w2-d's factory routes their models to them.
  The 9 openai-format Stage-1 providers route to `generic.New(id)`; ollama to
  `ollama.New("ollama"|"ollama-local", registry)`. The superseded empty per-dir packages
  (deepseek/groq/mistral/cohere/together/fireworks) are removed in this plan (their models
  now route to the generic adapter).

## Preconditions (a "0 hits" grep exits 1 = pass)

- `test -d internal/providers/catalog` AND `grep -c 'func ModelsFor' internal/providers/catalog/models.go` ≥ 1 (w2-a)
- `grep -c 'package generic' internal/providers/generic/provider.go` ≥ 1 (w2-b merged)
- `grep -c 'func New' internal/providers/ollama/provider.go` ≥ 1 (w2-c merged)
- `grep -c 'func (r \*Router) Resolve' internal/inference/router.go` ≥ 1 (replacing it)

## Exclusive file ownership

NEW: `internal/inference/factory.go` + `factory_test.go`.
TOUCH-ONLY: `internal/inference/router.go` (rewrite Resolve/ResolveForModel + Router struct + NewRouter), `internal/inference/router_test.go`, `internal/api/models.go` (aggregate), `internal/api/models_test.go`.
DELETE: `internal/providers/{deepseek,groq,mistral,cohere,together,fireworks}/` (empty doc.go+test stubs superseded by the generic adapter — confirm each contains only doc.go + a trivial test before removal; if any has real code, IMPL-BLOCKED).

## Tasks (STEP (a) failing tests first; STEP (b) implement)

1. **Model→provider index + factory** (`factory.go`):
   - `func providerForModel(model string) (providerID string, ok bool)` — search `catalog.Models` across all Stage-1 providers for an entry whose `ID == model`; also keep the existing prefix routing for openai/anthropic/gemini (claude-*/gemini-* and default openai) so current behavior is preserved. Catalog match wins over prefix default.
   - `func buildProvider(providerID string, reg *translation.Registry) (schemas.Provider, error)` — the factory (Go analog of `getExecutor`): openai→`openai.NewProvider()`, anthropic→`anthropic.NewProvider()`, gemini→`gemini.NewProvider()`, ollama/ollama-local→`ollama.New(id, reg)`, any other catalog openai-format id→`generic.New(id)`; unknown→error.
   Tests (`factory_test.go`): `TestProviderForModelCatalog` (`deepseek-chat`→"deepseek", `grok-4`→"xai", `sonar`→"perplexity"), `TestProviderForModelPrefix` (`claude-…`→"anthropic", `gemini-…`→"gemini", unknown→openai default), `TestBuildProviderGeneric` (deepseek id → *generic.Provider), `TestBuildProviderOllama` (ollama → *ollama.Provider), `TestBuildProviderExisting` (openai/anthropic/gemini → their types).

2. **Catalog-driven Router** (`router.go`, REPLACE the prefix stub):
   - `Router` holds a `*translation.Registry` (shared, for ollama) + cached provider instances (lazy via `buildProvider`). `NewRouter(reg *translation.Registry)` — UPDATE all callers (`internal/server/*`, `internal/api/*` construct the router) to pass the shared registry; if a caller has no registry, it constructs one (`translation.NewRegistry()`).
   - `Resolve(model) (schemas.Provider, schemas.Key, error)`: `providerForModel(model)` → `buildProvider` (cached) → return with `schemas.Key{Provider: providerID}` (Value supplied by management layer as today). `ResolveForModel(req)` delegates.
   Tests (`router_test.go`): keep/extend existing — `TestResolveDeepSeekRoutesToGeneric`, `TestResolveOllamaRoutesToOllama`, `TestResolveClaudePrefixUnchanged`, `TestResolveUnknownDefaultsOpenAI`.

3. **/v1/models aggregation** (`models.go`):
   - `List` aggregates `catalog.ModelsFor(...)` across ALL wired Stage-1 providers + the existing openai/anthropic/gemini catalogs into one `schemas.ListModelsResponse` (`object:"list"`, each `ModelEntry{ID, Object:"model", OwnedBy:providerID}`) — instead of calling one provider's `ListModels`. Deterministic order (sort by ID) for stable output.
   Tests (`models_test.go`): `TestListModelsAggregatesCatalog` (deepseek + groq + xai + ollama models all present, OwnedBy correct), `TestListModelsDeterministicOrder`.

## Binary acceptance criteria

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `grep -c 'func providerForModel' internal/inference/factory.go` ≥ 1; `grep -c 'func buildProvider' internal/inference/factory.go` ≥ 1.
- `test ! -d internal/providers/deepseek && test ! -d internal/providers/groq` (superseded dirs removed) — exit 0.
- `TestResolveDeepSeekRoutesToGeneric`, `TestResolveOllamaRoutesToOllama`, `TestResolveClaudePrefixUnchanged`, `TestListModelsAggregatesCatalog` pass.
- A chat request for `deepseek-chat` routes to a `*generic.Provider` configured with the deepseek catalog (covered by go test).

## Out of scope

Combo/fallback/rate-limit/alias routing (Wave 4). Key store / virtual keys (existing/later). OAuth (Wave 3). Per-model capability routing by `Type` (Wave 4/5). Any Stage-2+ provider. Adapter internals (w2-b/c).
