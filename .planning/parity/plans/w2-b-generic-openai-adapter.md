# w2-b — Generic OpenAI-compatible provider adapter

Rows: PAR-PROV-004 (groq), 005 (deepseek), 006 (mistral), 007 (cohere), 008 (together), 009 (fireworks), 014 (openrouter), 027 (xai — API-key/bearer path; the ranking `matrix/9router-providers.md:228` rank 7 states "has OAuth but also API-key path (grok models)", so Stage-1 serves xai via bearer and defers its OAuth to Wave 3), 029 (perplexity) — the 9 pure-OpenAI-format Stage-1 providers, served by ONE config-driven adapter (Go-port consideration #2: "DefaultExecutor covers ~80% via OpenAI-compatible passthrough; a generic Go executor would collapse many adapters into one struct"). ollama (010) is w2-c; router wiring is w2-d.

Frozen ref (@ 827e5c3), read whole: `open-sse/executors/base.js` (buildUrl/buildHeaders), `open-sse/executors/default.js` (chat+stream execute; the `refresh*` methods are Wave-3 token refresh — NOT ported here). In-repo pattern to mirror: `internal/providers/openai/chat.go:1-169` (ClientPool, SSEScanner, ErrorConverter, in-band stream errors AUD-045/046/047), `internal/providers/openai/stubs.go` (typed not-implemented pattern).

Depends on w2-a (catalog) — MERGED before dispatch (precondition gate below).

## Scope decisions (read first)

- **chat + stream only.** Per Go-port consideration #1 ("most 9router providers only
  implement chat + streaming"), the generic adapter implements `ChatCompletion` +
  `ChatCompletionStream`. Embeddings and ALL other Provider-interface capabilities are
  **typed not-implemented stubs** (decision 9); embedding-serving is a later increment
  (consideration #6). The catalog may list embedding/image/tts models (w2-a) that this
  adapter does not yet serve — those calls return a typed not-implemented error; that is
  the documented Stage-1 state.
- **Full-URL endpoints.** The catalog `BaseURL` is the COMPLETE chat endpoint
  (`providers.js` baseUrls are full, e.g. `https://api.deepseek.com/chat/completions`) —
  POST directly to `config.BaseURL`; do NOT append `/v1/chat/completions` (unlike the
  hardcoded `openai/chat.go:26`).
- **PAR-PR-664 is NOT in scope.** Verified in ref: `max_tokens`→`max_completion_tokens`
  lives ONLY in `executors/github.js:140`, `qoder.js`, `codex.js` (all Stage-2-deferred,
  gated on newer OpenAI models) — it is NOT a DefaultExecutor behavior. Applying it
  blanket would break deepseek/groq/etc. which accept `max_tokens`. Do NOT add it.

## Preconditions (a "0 hits" grep exits 1 = pass)

- `test -d internal/providers/catalog` (w2-a merged) AND `grep -c 'func Lookup' internal/providers/catalog/catalog.go` ≥ 1
- `grep -rn 'package generic' internal/providers/generic/` → 0 hits (new package)
- `grep -n 'ClientPool\|SetAuthHeader\|NewSSEScanner' internal/providers/utils/*.go` → present (reuse, do not reimplement)

## Exclusive file ownership

NEW: `internal/providers/generic/provider.go`, `chat.go`, `stubs.go`, `provider_test.go`, `chat_test.go`.
TOUCH-ONLY: none (router wiring of these 9 ids → GenericProvider is w2-d; the per-dir empty packages deepseek/groq/mistral/cohere/together/fireworks remain untouched and are superseded — w2-d removes/bypasses them).

## Tasks (STEP (a) failing tests first; STEP (b) implement)

1. **GenericProvider struct + construction** (`provider.go`):
   - `type Provider struct { id schemas.ModelProvider; config catalog.ProviderConfig; client *utils.ClientPool; networkConfig schemas.NetworkConfig; errorConverter *openai.ErrorConverter }` — `errorConverter` from the exported `openai.NewErrorConverter()` (`internal/providers/openai/errors.go:11-19`; provider-agnostic `Convert`).
   - `func New(providerID string) (*Provider, error)` — `catalog.Lookup(providerID)`; error if unknown or `config.Format != "openai"` (this adapter is openai-format only; ollama is w2-c). Reuse `utils.NewClientPool()`.
   - `GetProvider()`/`SetNetworkConfig()` like `openai/provider.go:27-35`.
   Tests: `TestNewGenericKnownProvider` (deepseek/groq/… construct, GetProvider==id), `TestNewGenericUnknown` (error), `TestNewGenericRejectsNonOpenAIFormat` (ollama id → error).

2. **chat + stream** (`chat.go`), mirror `openai/chat.go` but config-driven:
   - URL helper: `func (p *Provider) chatURL() string { return p.config.BaseURL }` (full URL, no path append — the catalog BaseURL is the complete endpoint).
   - `ChatCompletion`: POST `p.chatURL()`; headers = Content-Type + `config.Headers` + (unless `config.NoAuth`) bearer `Authorization: Bearer <key.Value>`; marshal request; status/error via `p.errorConverter.Convert(status, body, schemas.ErrorMeta{Provider:string(p.id), ModelRequested:request.Model, RequestType:"chat", StatusCode:status, RawBody:body})`; decode `schemas.ChatResponse`.
   - `ChatCompletionStream`: same URL/headers; `Stream=true`; SSE via `utils.NewSSEScanner`; `[DONE]` ends; malformed chunk → in-band `streamError` (AUD-045); read error → in-band (AUD-046); post-hook failure → in-band (AUD-047): the trigger is the `postHookRunner schemas.PostHookRunner` parameter (same signature as the interface) — `postHookRunner.Run(ctx, &chunk)` returning a non-nil error emits an in-band `streamError` then closes, identical to `openai/chat.go:158-164`.
   Tests (`chat_test.go`; round-trip tests use `httptest.NewServer` and set `p.config.BaseURL = srv.URL` — same in-package seam as `openai/stream_test.go:26-27` `p.baseURL = srv.URL`; no mocks):
     - `TestGenericChatURL` (PURE, no network: a deepseek-config provider's `chatURL()` == `https://api.deepseek.com/chat/completions`, with NO `/v1/chat/completions` appended).
     - `TestGenericChatBearerAuth` (httptest captures `Authorization: Bearer <key>`).
     - `TestGenericChatCustomHeaders` (openrouter config → httptest sees HTTP-Referer + X-Title).
     - `TestGenericChatErrorStatus` (httptest 500 → ProviderError with provider id).
     - `TestGenericStreamParsesSSE` (httptest streams SSE → chunks emitted, [DONE] ends).
     - `TestGenericStreamMalformedChunkInBandError` (httptest emits bad chunk → in-band streamError, AUD-045).

3. **Typed not-implemented stubs** (`stubs.go`) — decision 9 ("full Bifrost-size interface with typed not-implemented stubs"). The authoritative method set is the `schemas.Provider` interface (`internal/schemas/provider.go:68-107`); stub EVERY method not implemented in Tasks 1-2 (i.e. all except GetProvider/SetNetworkConfig/ChatCompletion/ChatCompletionStream): TextCompletion(+Stream), Responses(+Stream), Embedding, ImageGeneration(+Stream)/ImageEdit/ImageVariation, Speech(+Stream), Transcription(+Stream), File{Upload,List,Retrieve,Delete,Content}, Batch{Create,List,Retrieve,Cancel}, ListModels, CountTokens. Pattern: `internal/providers/openai/stubs.go`. Each returns a typed `*schemas.ProviderError{Type:"not_implemented", StatusCode:501, Message:"<method> not implemented for generic openai-compatible provider"}` (or nil+stub for channel returns). `Provider` must satisfy `schemas.Provider` (compile-time `var _ schemas.Provider = (*Provider)(nil)`).
   Tests: `TestGenericSatisfiesProviderInterface` (compile-time assertion present), `TestGenericEmbeddingNotImplemented` (501 typed error).

## Binary acceptance criteria

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `var _ schemas.Provider = (*Provider)(nil)` compiles (interface satisfied).
- `grep -rn 'func init(\|panic(' internal/providers/generic/` → 0 hits.
- `grep -c 'max_completion_tokens' internal/providers/generic/` → 0 (PAR-PR-664 excluded).
- `grep -c '/v1/chat/completions' internal/providers/generic/chat.go` → 0 (uses full BaseURL).
- `TestGenericChatURL`, `TestGenericChatCustomHeaders`, `TestGenericStreamMalformedChunkInBandError` pass.

## Out of scope

ollama (w2-c). Router/registry wiring + removing superseded per-dir packages (w2-d). Embeddings/images/speech/etc. real impls (later increments — stubs only here). OAuth/token refresh (Wave 3 — adapter uses `key.Value` as given). `max_completion_tokens` (Stage-2 github/qoder/codex). Static model `Type`/`Params` capability routing (Wave 4/5).
