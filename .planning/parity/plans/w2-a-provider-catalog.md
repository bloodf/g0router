# w2-a — Provider config catalog + model catalogs (Stage-1 providers)

Rows: PAR-PROV-004 (groq), 005 (deepseek), 006 (mistral), 007 (cohere), 008 (together), 009 (fireworks), 010 (ollama/ollama-local), 014 (openrouter), 027 (xai, API-key path), 029 (perplexity) — config + model-catalog substrate only (the adapters that consume this are w2-b/w2-c; routing is w2-d). Scope per `WAVE-2-MAP.md` "Include now" ranking.

Frozen ref (@ 827e5c3), read whole before porting:
- `open-sse/config/providers.js:1-457` (entries: groq :269-271, deepseek :257-259, mistral :281-283, cohere :301-303, together :289-291, fireworks :293-295, openrouter :115-122, xai :273-280, perplexity :285-287, ollama :333-336, ollama-local :337-340; helpers `resolveOllamaLocalHost` :442-445, stainless OS/arch :4-21)
- `open-sse/config/providerModels.js` (the per-provider model arrays for those 10 keys — port each block verbatim)

## Preconditions (a "0 hits" grep exits 1 — that IS the pass)

- `grep -rn 'package catalog' internal/providers/catalog/` → 0 hits (new package)
- `grep -n 'ProviderDeepSeek\|ProviderGroq\|ProviderOpenRouter' internal/schemas/provider.go` → present (constants exist :11-22; openrouter :21)
- `grep -rn 'providerModels\|ProviderCatalog\|ModelCatalog' internal/providers/` → 0 hits (no prior catalog)

## Exclusive file ownership

NEW: `internal/providers/catalog/catalog.go` + `catalog_test.go`, `internal/providers/catalog/models.go` + `models_test.go`.
TOUCH-ONLY: none (pure new package; no registry/router change here — that is w2-d).

## Tasks (STEP (a) write named failing tests; STEP (b) port)

1. **Provider config struct + entries** (`catalog.go`), port the 10 entries from `providers.js`:
   - `type ProviderConfig struct { Name string; BaseURL string; Format string; Headers map[string]string; AuthHeader string; NoAuth bool }` — `Format` ∈ {"openai","ollama"} for Stage-1; `AuthHeader` default "" means bearer `Authorization` (the 10 Stage-1 providers are all bearer or no-auth — ollama is NoAuth; none use x-api-key in this set).
   - `var Providers = map[string]ProviderConfig{...}` with the 10 entries, each BaseURL/Format/Headers byte-exact from the cited `providers.js` lines (e.g. openrouter carries `HTTP-Referer`/`X-Title` headers :118-121; perplexity/groq/etc. have no custom headers; ollama/ollama-local Format "ollama", NoAuth true).
   - `func Lookup(provider string) (ProviderConfig, bool)`.
   - `func ResolveOllamaHost(baseURLOverride string) string` — port `resolveOllamaLocalHost` (:442-445): trimmed override or `http://localhost:11434`, trailing slash stripped.
   Tests (`catalog_test.go`): `TestLookupKnownProviders` (all 10 present, correct BaseURL+Format), `TestLookupUnknown` (ok=false), `TestOpenRouterHeaders` (HTTP-Referer + X-Title exact), `TestOllamaConfig` (Format "ollama", NoAuth true, both ollama and ollama-local), `TestResolveOllamaHost` (override trimmed; default; trailing slash stripped).

2. **Model catalogs** (`models.go`), port the `providerModels.js` block for each of the 10 keys VERBATIM:
   - `type ModelEntry struct { ID string; Name string; UpstreamModelID string; Type string }` — `Type` ∈ {"", "llm", "embedding", "stt", "image"} ("" treated as "llm" per ref `(model.type||"llm")`); `UpstreamModelID` defaults to `ID` when the ref omits it.
   - `var Models = map[string][]ModelEntry{...}` for groq, deepseek, mistral, cohere, together, fireworks, openrouter, xai, perplexity, ollama — each entry copied from `providerModels.js` (e.g. deepseek's 6 incl. `deepseek-v4-pro-max`/`-none` with `UpstreamModelID:"deepseek-v4-pro"`; groq's 4 llm + 3 stt with `type:"stt"`; xai's grok-2-image `type:"image"`; mistral-embed/together-embeddings `type:"embedding"`). openrouter: port its providerModels.js block (if dynamic/empty in ref, an empty slice with a comment citing the ref). ollama: port from `ollamaModels.js` if a static block exists, else empty slice with ref comment (ollama models are user-pulled).
   - `func ModelsFor(provider string) []ModelEntry`; `func ResolveModel(provider, id string) (ModelEntry, bool)` returning the entry (so callers get `UpstreamModelID`).
   Tests (`models_test.go`): `TestModelsForDeepSeek` (6 entries; `-max`/`-none` UpstreamModelID == "deepseek-v4-pro"), `TestModelTypeDefaulting` (a no-type entry resolves Type "llm" via a `EffectiveType()` helper or stored "llm"), `TestGroqSTTModels` (3 type "stt"), `TestResolveModelUpstream` (alias id → entry with correct UpstreamModelID), `TestModelsForUnknown` (empty slice).

## Binary acceptance criteria

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `grep -rn 'func init(\|panic(' internal/providers/catalog/` → 0 hits.
- `go test ./internal/providers/catalog/ -run 'TestLookupKnownProviders|TestModelsForDeepSeek|TestGroqSTTModels' -count=1` passes with >0 tests.
- `Lookup` returns all 10 Stage-1 providers; `ModelsFor` non-empty for the 9 with static catalogs (deepseek/groq/mistral/cohere/together/fireworks/xai/perplexity + openrouter-or-documented-empty).
- No HTTP/execution code in this package (pure config/data — adapters are w2-b/c).

## Out of scope

Request execution / HTTP (w2-b generic adapter, w2-c ollama). Router wiring + /v1/models (w2-d). OAuth fields/handlers (Wave 3 — Stage-1 providers are API-key/no-auth; do NOT add clientId/tokenUrl). Any provider outside the 10-row "Include now" set. Capability routing by model `Type` (Wave 4/5; the field is recorded only).
