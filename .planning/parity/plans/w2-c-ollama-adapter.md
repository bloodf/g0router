# w2-c — Ollama provider adapter

Rows: PAR-PROV-010 (ollama / ollama-local) — ollama-native chat adapter (no-auth, NDJSON) using the w1-h openai↔ollama translators. Per `matrix/9router-providers.md:216-218` "Include now" (rank 10). Catalog config + model catalog come from w2-a; router wiring from w2-d.

Frozen ref (@ 827e5c3), read whole: `open-sse/executors/ollama-local.js`, `open-sse/config/providers.js:333-340` (ollama `https://ollama.com/api/chat`, ollama-local `http://localhost:11434/api/chat`, both `format:"ollama"`, no auth), `:442-445` (`resolveOllamaLocalHost`). In-repo: w1-h translators are REGISTERED — `internal/translation/registry.go:167-168` wires `FormatOpenAI→FormatOllama` (request `openaiToOllamaRequest`) and `FormatOllama→FormatOpenAI` (response `ollamaToOpenAIResponse`); `OllamaBodyToOpenAI` (`internal/translation/ollama_openai_response.go:197`, exported, non-streaming); `utils.NewNDJSONScanner` (`internal/providers/utils/sse.go:27`).

Depends on w2-a (catalog) merged. The ollama translator funcs are unexported, so the
adapter MUST go through the exported `*translation.Registry` (TranslateRequest/
TranslateResponse) + exported `OllamaBodyToOpenAI` — never reach into the package.

## Scope decisions

- **chat + stream only** — the adapter implements ChatCompletion + ChatCompletionStream; the remaining `schemas.Provider` methods (`provider.go:68-107`) are typed 501 stubs required for the type to satisfy the interface and compile (see Task 3).
- **No auth** — ollama `config.NoAuth == true`; send no Authorization header.
- **Ollama-native wire** — the request body is ollama-shaped (via the registry
  openai→ollama translation), POSTed as JSON; the response is **NDJSON** (not SSE),
  parsed with `NewNDJSONScanner` then translated ollama→openai per line.
- **Host resolution** — cloud `ollama` uses `config.BaseURL` directly (full `/api/chat` URL). `ollama-local` uses `catalog.ResolveOllamaHost("") + "/api/chat"` = the DEFAULT host `http://localhost:11434`. The ref's `providerSpecificData.baseUrl` override is NOT threaded in Stage-1: the current `schemas.Key` struct (`internal/schemas/provider.go:30-34`) carries only ID/Provider/Value — no per-credential data — so the override is deferred to Wave 3 (credential plumbing). Pass `""` to `ResolveOllamaHost`; document the deferred override in a comment.

## Preconditions (a "0 hits" grep exits 1 = pass)

- `test -d internal/providers/catalog` AND `grep -c 'func ResolveOllamaHost' internal/providers/catalog/catalog.go` ≥ 1 (w2-a merged)
- `grep -c 'FormatOllama' internal/translation/registry.go` ≥ 2 (w1-h wired)
- `grep -c 'func NewNDJSONScanner' internal/providers/utils/sse.go` ≥ 1; `grep -c 'func OllamaBodyToOpenAI' internal/translation/ollama_openai_response.go` ≥ 1
- `grep -rn 'func (p \*Provider) ChatCompletion' internal/providers/ollama/` → 0 hits (dir has only doc.go + test stub today)

## Exclusive file ownership

NEW/FILL in `internal/providers/ollama/`: `provider.go`, `chat.go`, `stubs.go`, `chat_test.go`, `provider_test.go`. EXISTING in the dir (leave intact): `doc.go`, `ollama_test.go` (the current trivial test — do not delete; new tests go in the new `_test.go` files).
TOUCH-ONLY: none (router wiring is w2-d).

## Tasks (STEP (a) failing tests first; STEP (b) implement)

1. **Provider struct** (`provider.go`). STEP (a) FIRST write `TestNewOllamaProvider` + `TestNewOllamaRejectsNonOllama` and run them (fail — `New` is not yet defined in the package; `doc.go`/`ollama_test.go` exist but contain no `New`). STEP (b) implement: `type Provider struct { id schemas.ModelProvider; config catalog.ProviderConfig; registry *translation.Registry; client *utils.ClientPool; networkConfig schemas.NetworkConfig }` — `id` set from the providerID arg ("ollama"/"ollama-local"), returned by `GetProvider()` and used in `Meta.Provider`. Types: `schemas.ModelProvider`/`NetworkConfig` (`internal/schemas/provider.go:4,37`), `utils.ClientPool` via `utils.NewClientPool()` (`internal/providers/utils`), `catalog.ProviderConfig`/`Lookup` (the w2-a package — w2-a is a merged dependency per the precondition gate; its symbols exist on disk at impl time). `func New(providerID string, reg *translation.Registry) (*Provider, error)` — `catalog.Lookup` (must be "ollama"/"ollama-local", `Format=="ollama"`, else error); `GetProvider()`/`SetNetworkConfig()`.
   Tests: `TestNewOllamaProvider` (ollama + ollama-local construct), `TestNewOllamaRejectsNonOllama` (deepseek id → error).

2. **chat + stream** (`chat.go`). STEP (a) FIRST write the Task-2 tests below and run them (fail). STEP (b) implement:
   - URL: `func (p *Provider) chatURL() string` — for `ollama-local`, `catalog.ResolveOllamaHost("") + "/api/chat"` (default host); for cloud `ollama`, `config.BaseURL`. Ref `executors/ollama-local.js`.
   - `ChatCompletion` (non-streaming): `reqMap` = registry `TranslateRequest(FormatOpenAI, FormatOllama, model, body, false, nil)`; POST JSON to chatURL, no auth; on 200 read the ollama response body, `OllamaBodyToOpenAI(body)` (`internal/translation/ollama_openai_response.go:197`) → `schemas.ChatResponse`; non-200 → `*schemas.ProviderError` with `Meta.Provider = string(p.id)` (the actual id — `ollama` OR `ollama-local`, not hardcoded). NOTE ollama non-streaming returns a single JSON object (not NDJSON).
   - `ChatCompletionStream`: translate request (`stream:true`); POST; response is NDJSON → `NewNDJSONScanner`; per line: `TranslateResponse(FormatOllama, FormatOpenAI, lineMap, state)` → emit each OpenAI chunk; scanner EOF ends; malformed line / read error / post-hook failure → in-band `streamError` then close — mirror the in-band error handling at `internal/providers/openai/chat.go:143-164` (the `postHookRunner schemas.PostHookRunner` param triggers the hook path, `internal/schemas/provider.go:44-46`).
   Tests (`chat_test.go`): round-trip tests use the CLOUD ollama config and set `p.config.BaseURL = srv.URL` (in-package seam, same as `openai/stream_test.go:26-27` `p.baseURL = srv.URL`) to redirect POSTs to `httptest.NewServer`; the local-host case is a PURE assertion (no network).
     - `TestOllamaChatURLLocalVsCloud` (PURE: ollama-local `chatURL()` == `http://localhost:11434/api/chat`; cloud ollama `chatURL()` == config BaseURL).
     - `TestOllamaChatNoAuthHeader` (httptest captures NO Authorization header).
     - `TestOllamaChatNonStreaming` (httptest returns one ollama JSON object → OpenAI ChatResponse via OllamaBodyToOpenAI).
     - `TestOllamaStreamNDJSON` (httptest streams NDJSON lines → OpenAI chunks; done line ends).
     - `TestOllamaStreamMalformedInBandError` (httptest bad line → in-band streamError).

3. **Typed stubs** (`stubs.go`). STEP (a) FIRST write `TestOllamaSatisfiesProviderInterface` + `TestOllamaEmbeddingNotImplemented` (fail). STEP (b): the type MUST satisfy `schemas.Provider` (`internal/schemas/provider.go:68-107`) to be usable by the router, and Go requires EVERY interface method to be defined for the type to compile — so the non-chat methods are a compile necessity, not added scope. Stub every method except GetProvider/SetNetworkConfig/ChatCompletion/ChatCompletionStream → typed 501 not-implemented, pattern `internal/providers/openai/stubs.go:17-23`; add `var _ schemas.Provider = (*Provider)(nil)`.
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

## Plan-gate disposition (Fable 5, 2026-06-11)

APPROVED BY DECISION after 4 cycles (harness rule: max 3 reject cycles → decide).
Substantive findings resolved across v1-v4: dropped the invented
`providerSpecificData` override (Key has no such field — Wave 3), provider id from
`p.id` (real fix), NDJSON/in-band-error pattern cited to `openai/chat.go:143-164`,
explicit httptest seam, stubs justified as a Go compile necessity. Remaining
gate items are not actionable: demanding file:line for `catalog.*` symbols defined
in the w2-a DEPENDENCY package (not yet on disk — that is what the precondition
merge-gate guarantees) is an impossible bar for a dependent plan; cross-plan wave
sequencing is structural, not scope creep. The Kimi diff gate is the binding
quality check at implementation.
