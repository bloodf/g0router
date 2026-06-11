# w2-a — Provider config catalog + model catalogs (Stage-1 providers)

Rows: PAR-PROV-004 (groq), 005 (deepseek), 006 (mistral), 007 (cohere), 008 (together), 009 (fireworks), 010 (ollama/ollama-local), 014 (openrouter), 027 (xai, API-key path), 029 (perplexity) — config + model-catalog substrate only (the adapters that consume this are w2-b/w2-c; routing is w2-d). Scope per `matrix/9router-providers.md:216-218` §"Stage 1 Go-port ranking → Include now" (indexed in `WAVE-2-MAP.md`).

Frozen ref (@ 827e5c3), read whole before porting:
- `open-sse/config/providers.js:1-457` (entries: groq :269-271, deepseek :257-259, mistral :281-283, cohere :301-303, together :289-291, fireworks :293-295, openrouter :115-122, xai :273-280, perplexity :285-287, ollama :333-336, ollama-local :337-340; helpers `resolveOllamaLocalHost` :442-445, stainless OS/arch :4-21)
- `open-sse/config/providerModels.js` (the per-provider model arrays for those 10 keys — port each block verbatim)

## Preconditions (a "0 hits" grep exits 1 — that IS the pass)

- `test ! -d internal/providers/catalog` (the package directory does not yet exist) — exit 0 means absent, the pass condition
- `grep -n 'ProviderDeepSeek\|ProviderGroq\|ProviderOpenRouter' internal/schemas/provider.go` → present (constants exist :11-22; openrouter :21)
- `grep -rn 'providerModels\|ProviderCatalog\|ModelCatalog' internal/providers/` → 0 hits (no prior catalog)

## Exclusive file ownership

NEW: `internal/providers/catalog/catalog.go` + `catalog_test.go`, `internal/providers/catalog/models.go` + `models_test.go`.
TOUCH-ONLY: none (pure new package; no registry/router change here — that is w2-d).

## Tasks (STEP (a) write named failing tests; STEP (b) port)

1. **Provider config struct + entries** (`catalog.go`) — the Go port of the ref's `PROVIDERS` map (`providers.js:50-438`). `Providers` IS that map; `Lookup` is the idiomatic Go accessor for what the ref does as `PROVIDERS[provider]` (`base.js getBaseUrls/buildHeaders` read it); `ResolveOllamaHost` IS the ref's exported `resolveOllamaLocalHost` (`:442-445`). No invented abstraction — same data + accessors as the ref. Port the entries:
   - `type ProviderConfig struct { Name string; BaseURL string; Format string; Headers map[string]string; AuthHeader string; NoAuth bool }` — `Format` ∈ {"openai","ollama"} for Stage-1; `AuthHeader` default "" means bearer `Authorization` (all Stage-1 providers are bearer or no-auth; ollama/ollama-local NoAuth; none use x-api-key in this set). **xai (PAR-PROV-027):** the ref carries OAuth fields `clientId`/`tokenUrl`/`refreshUrl` (`providers.js:273-280`), but the Stage-1 ranking includes xai via its API-key (bearer) path only and OAuth is Stage-2 — so the Stage-1 `ProviderConfig` struct intentionally OMITS those fields (the struct gains them in Wave 3 when xai-OAuth lands). Carry only xai's BaseURL/Format/Headers; document the omitted OAuth fields in a code comment citing :273-280.
   - `var Providers = map[string]ProviderConfig{...}` with **11 entries** (the 10 providers, with `ollama` AND `ollama-local` as two separate keys), each BaseURL/Format/Headers byte-exact from the cited `providers.js` lines (e.g. openrouter carries `HTTP-Referer`/`X-Title` headers :118-121; perplexity/groq/etc. have no custom headers; ollama/ollama-local Format "ollama", NoAuth true).
   - `func Lookup(provider string) (ProviderConfig, bool)`.
   - `func ResolveOllamaHost(baseURLOverride string) string` — port `resolveOllamaLocalHost` (:442-445): trimmed override or `http://localhost:11434`, trailing slash stripped.
   Tests (`catalog_test.go`): `TestLookupKnownProviders` (all 11 keys present incl. both ollama and ollama-local, correct BaseURL+Format), `TestLookupUnknown` (ok=false), `TestOpenRouterHeaders` (HTTP-Referer + X-Title exact), `TestOllamaConfig` (Format "ollama", NoAuth true, both ollama and ollama-local), `TestResolveOllamaHost` (override trimmed; default; trailing slash stripped).

2. **Model catalogs** (`models.go`), port the `providerModels.js` block for each of the 10 keys VERBATIM:
   - `type ModelEntry struct { ID string; Name string; UpstreamModelID string; Type string; Params []string }` — `Type` stored VERBATIM from the ref array — "" when the entry has no `type` field, else the exact value ∈ {"llm","embedding","stt","image","tts"} (do NOT default "" to "llm" in the catalog; the ref applies `model.type||"llm"` only at read sites, which are Wave-4/5 — the catalog carries raw data). `UpstreamModelID` defaults to `ID` only when the ref omits it (the ref `upstreamModelId || id` is a data-normalization the catalog may bake in); `Params` carries the ref's per-model `params` array verbatim when present (e.g. stt/image/tts entries) to avoid catalog data loss (recorded, unused until Wave 4/5).
   - `var Models = map[string][]ModelEntry{...}` for groq, deepseek, mistral, cohere, together, fireworks, openrouter, xai, perplexity, ollama — each entry (incl. `Type` and `Params`) copied from `providerModels.js` (e.g. deepseek's 6 incl. `deepseek-v4-pro-max`/`-none` with `UpstreamModelID:"deepseek-v4-pro"`; groq's 4 llm + 3 stt with `type:"stt"`; xai's grok-2-image `type:"image"`; mistral-embed/together-embeddings `type:"embedding"`). openrouter: port the STATIC block `providerModels.js:302-320` (7 embedding + 3 tts + 4 image entries — NOT empty). ollama: port the STATIC block `providerModels.js:572-579` — exactly 6 entries: `gpt-oss:120b`, `kimi-k2.5`, `glm-5`, `minimax-m2.5`, `glm-4.7-flash`, `qwen3.5` (no others). cohere: `providerModels.js:508-512` (3 command models).
   - `func ModelsFor(provider string) []ModelEntry`; `func ResolveModel(provider, id string) (ModelEntry, bool)` returning the entry (so callers get `UpstreamModelID`).
   Tests (`models_test.go`): `TestModelsForDeepSeek` (6 entries; `-max`/`-none` UpstreamModelID == "deepseek-v4-pro"), `TestModelTypeVerbatim` (a no-type entry stores Type "" — NOT defaulted; an stt entry stores "stt"), `TestGroqSTTModels` (3 type "stt"), `TestResolveModelUpstream` (alias id → entry with correct UpstreamModelID), `TestOpenRouterCatalogTypes` (embedding+tts+image present with Params), `TestModelsForOllama` (static entries, non-empty), `TestModelsForUnknown` (empty slice).

## Binary acceptance criteria

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `grep -rn 'func init(\|panic(' internal/providers/catalog/` → 0 hits.
- `go test ./internal/providers/catalog/ -run 'TestLookupKnownProviders|TestModelsForDeepSeek|TestGroqSTTModels' -count=1` passes with >0 tests.
- `Lookup` returns all 11 keys (10 providers; ollama+ollama-local). `ModelsFor` is non-empty for ALL 10 providers (deepseek/groq/mistral/cohere/together/fireworks/xai/perplexity/openrouter/ollama — each has a static `providerModels.js` block).
- `TestOpenRouterCatalogTypes`: openrouter entries include `Type` "embedding", "tts", AND "image" with `Params` populated (no data loss vs `providerModels.js:302-320`).
- `grep -rnE 'fasthttp|net/http|ClientPool|errorConverter' internal/providers/catalog/` → 0 hits (pure config/data; execution is w2-b/c).

## Out of scope

Request execution / HTTP (w2-b generic adapter, w2-c ollama). Router wiring + /v1/models (w2-d). OAuth fields/handlers (Wave 3 — Stage-1 providers are API-key/no-auth; do NOT add clientId/tokenUrl). Any provider outside the 10-row "Include now" set. Capability routing by model `Type` (Wave 4/5; the field is recorded only).
