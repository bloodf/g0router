# w2-c — Ollama provider adapter

Rows: PAR-PROV-010 (ollama / ollama-local) — ollama-native chat adapter (no-auth, NDJSON) using the w1-h openai↔ollama translators. Per `matrix/9router-providers.md:216-218` "Include now" (rank 10). Catalog config + model catalog come from w2-a; router wiring from w2-d.

Frozen ref (@ 827e5c3), read whole: `open-sse/executors/ollama-local.js`, `open-sse/config/providers.js:333-340` (ollama `https://ollama.com/api/chat`, ollama-local `http://localhost:11434/api/chat`, both `format:"ollama"`, no auth), `:442-445` (`resolveOllamaLocalHost`). In-repo: w1-h translators are REGISTERED — `internal/translation/registry.go:167-168` wires `FormatOpenAI→FormatOllama` (request `openaiToOllamaRequest`) and `FormatOllama→FormatOpenAI` (response `ollamaToOpenAIResponse`); `OllamaBodyToOpenAI` (`internal/translation/ollama_openai_response.go:197`, exported, non-streaming); `utils.NewNDJSONScanner` (`internal/providers/utils/sse.go:27`).

Depends on w2-a (catalog) merged. The ollama translator funcs are unexported, so the
adapter MUST go through the exported `*translation.Registry` (TranslateRequest/
TranslateResponse) + exported `OllamaBodyToOpenAI` — never reach into the package.

## Scope decisions

- **chat + stream only** (consideration #1); embeddings/other capabilities = typed
  not-implemented stubs (decision 9), matching w2-b.
- **No auth** — ollama `config.NoAuth == true`; send no Authorization header.
- **Ollama-native wire** — the request body is ollama-shaped (via the registry
  openai→ollama translation), POSTed as JSON; the response is **NDJSON** (not SSE),
  parsed with `NewNDJSONScanner` then translated ollama→openai per line.
- **Host resolution** — POST URL is `catalog.ResolveOllamaHost(override)` joined with
  the ollama chat path (`/api/chat`); the cloud `ollama` provider uses the catalog
  BaseURL directly. `ollama-local` resolves the local host (default
  `http://localhost:11434`), honoring a `providerSpecificData.baseUrl` override (Stage-1:
  the override source is the key/credentials struct; thread it from `key`).

## Preconditions (a "0 hits" grep exits 1 = pass)

- `test -d internal/providers/catalog` AND `grep -c 'func ResolveOllamaHost' internal/providers/catalog/catalog.go` ≥ 1 (w2-a merged)
- `grep -c 'FormatOllama' internal/translation/registry.go` ≥ 2 (w1-h wired)
- `grep -c 'func NewNDJSONScanner' internal/providers/utils/sse.go` ≥ 1; `grep -c 'func OllamaBodyToOpenAI' internal/translation/ollama_openai_response.go` ≥ 1
- `grep -rn 'func (p \*Provider) ChatCompletion' internal/providers/ollama/` → 0 hits (dir has only doc.go + test stub today)

## Exclusive file ownership

NEW/FILL: `internal/providers/ollama/provider.go`, `chat.go`, `stubs.go`, `chat_test.go`, `provider_test.go` (the dir exists with `doc.go`+`ollama_test.go`; add these, leave `doc.go`).
TOUCH-ONLY: none (router wiring is w2-d).

## Tasks (STEP (a) failing tests first; STEP (b) implement)

1. **Provider struct** (`provider.go`): `type Provider struct { config catalog.ProviderConfig; registry *translation.Registry; client *utils.ClientPool; networkConfig schemas.NetworkConfig }`; `func New(providerID string, reg *translation.Registry) (*Provider, error)` — `catalog.Lookup` (must be "ollama"/"ollama-local", `Format=="ollama"`, else error); `GetProvider()`/`SetNetworkConfig()`.
   Tests: `TestNewOllamaProvider` (ollama + ollama-local construct), `TestNewOllamaRejectsNonOllama` (deepseek id → error).

2. **chat + stream** (`chat.go`):
   - URL: `func (p *Provider) chatURL(key schemas.Key) string` — for `ollama-local`, `catalog.ResolveOllamaHost(<override from key>) + "/api/chat"`; for cloud `ollama`, `config.BaseURL` (already the full `/api/chat` URL). Cite `ollama-local.js`.
   - `ChatCompletion` (non-streaming): `reqMap` = registry `TranslateRequest(FormatOpenAI, FormatOllama, model, body, false, nil)`; POST JSON to chatURL, no auth; on 200 read the ollama response body, `OllamaBodyToOpenAI(body)` → `schemas.ChatResponse`; non-200 → ProviderError (provider id "ollama"). NOTE ollama non-streaming returns a single JSON object (not NDJSON).
   - `ChatCompletionStream`: translate request (`stream:true`); POST; response is NDJSON → `NewNDJSONScanner`; per line: `TranslateResponse(FormatOllama, FormatOpenAI, lineMap, state)` → emit each OpenAI chunk; scanner EOF ends; malformed line → in-band `streamError` (AUD-045 parity with openai/chat.go); post-hook failure → in-band (AUD-047).
   Tests (`chat_test.go`, fake upstream): `TestOllamaChatURLLocalVsCloud` (local resolves localhost:11434/api/chat; cloud uses BaseURL), `TestOllamaChatNoAuthHeader` (no Authorization sent), `TestOllamaChatNonStreaming` (single ollama JSON → OpenAI ChatResponse via OllamaBodyToOpenAI), `TestOllamaStreamNDJSON` (NDJSON lines → OpenAI chunks; done line ends), `TestOllamaStreamMalformedInBandError`.

3. **Typed stubs** (`stubs.go`): every other `schemas.Provider` method → typed 501 not-implemented (mirror w2-b/`openai/stubs.go`); `var _ schemas.Provider = (*Provider)(nil)`.
   Tests: `TestOllamaSatisfiesProviderInterface`, `TestOllamaEmbeddingNotImplemented`.

## Binary acceptance criteria

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `var _ schemas.Provider = (*Provider)(nil)` compiles.
- `grep -rn 'func init(\|panic(' internal/providers/ollama/*.go` → 0 hits (excluding `_test.go`).
- `grep -c 'Authorization' internal/providers/ollama/chat.go` → 0 (no-auth).
- `grep -c 'NewNDJSONScanner' internal/providers/ollama/chat.go` ≥ 1 (NDJSON, not SSE).
- `TestOllamaChatURLLocalVsCloud`, `TestOllamaStreamNDJSON`, `TestOllamaChatNoAuthHeader` pass.

## Out of scope

Router/registry wiring + /v1/models (w2-d). Generic openai providers (w2-b). Embeddings/other capabilities (stubs only). OAuth (n/a — ollama is no-auth). Pulling/listing ollama-local installed models dynamically (Stage-2; w2-a's static catalog is used).
