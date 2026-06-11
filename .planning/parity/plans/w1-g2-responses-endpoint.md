# w1-g2 — /v1/responses endpoint + response.failed flush

Rows: PAR-TRANS-031 (route clause — the translator-existence clause closed in w1-g; this delivers the `/v1/responses` HTTP surface its gap note named), PAR-TRANS-050 (translate-mode `response.failed` flush — the site with a live caller via this endpoint; passthrough-mode is PAR-TRANS-050b, Wave 2), PAR-TRANS-002 (openai-responses format now reachable end-to-end). Depends on w1-g translators (MERGED `6640b33`+`ca8274e`).

Frozen ref (@ 827e5c3):
- `open-sse/translator/formats.js:23-24` (`/v1/responses` → `openai-responses`)
- `open-sse/utils/stream.js:202` (translate-mode `response.failed` flush) + `:397` (passthrough-mode flush)
- In-repo precedent (read whole): `internal/api/messages.go:1-110` (the `/v1/messages` handler — exact template), `internal/server/routes_openai.go:1-20`.

## Preconditions (a "0 hits" grep exits 1 — that IS the pass)

- `grep -n '"/v1/responses"' internal/translation/formats.go` → present (`formats.go:29` `DetectFormatByEndpoint` already maps it)
- `grep -n 'FormatOpenAIResponses' internal/translation/registry.go` → present (w1-g, both directions wired)
- `grep -n 'POST("/v1/responses"' internal/server/routes_openai.go` → 0 hits (route not yet added)
- `grep -rn 'ResponsesHandler\|NewResponsesHandler' internal/api/` → 0 hits

## Exclusive file ownership

NEW: `internal/api/responses.go` + `internal/api/responses_test.go`, `internal/translation/responses_stream_helpers.go` + `_test.go`.
TOUCH-ONLY: `internal/server/routes_openai.go` (add `responses := api.NewResponsesHandler(router_)` + `r.POST("/v1/responses", responses.Handle)`), `internal/translation/stream.go` (new `ResponsesTerminalSeen` field + flush block in `ProcessTranslateStream`). Coordination: the remaining in-flight Wave-1 plans are translator-file-scoped per their own "## Exclusive file ownership" — w1-j `*cursor*`, w1-l `claude_*`/`bypass_*`, w1-k `gemini_openai_request`; none names stream.go. w1-h/w1-i are already MERGED. So no concurrent stream.go writer.
Non-overlap: `responses.go`/`responses_test.go` match no other plan's ownership.

## Tasks (STEP (a) write named failing tests; STEP (b) implement)

1. **ResponsesHandler** (`internal/api/responses.go`), mirror `messages.go:14-110` structure exactly (read 3 neighbors: messages.go, chat.go, embeddings.go):
   - `ResponsesHandler{router modelResolver, registry *translation.Registry}`, `NewResponsesHandler(router *inference.Router)`.
   - `Handle(ctx)`: unmarshal body; `model` extraction; **streaming is forced true** — the ref request translator ALWAYS sets `stream:true` (`openai-responses.js:203,208`), so 9router's `/v1/responses` is streaming-only; this handler ignores any inbound `stream:false` and always streams (document with the ref citation). `TranslateRequest(FormatOpenAIResponses, FormatOpenAI, model, body, stream, nil)`; marshal→`schemas.ChatRequest`→`PreprocessChatRequest`; `ResolveForModel`; streaming path uses `ProcessTranslateStream(ctx, ch, registry, FormatOpenAI, FormatOpenAIResponses, state)` (note the to-format is FormatOpenAIResponses so the SSE event framing from w1-g applies). No non-streaming branch: the ref endpoint is streaming-only (`stream:true` forced at `openai-responses.js:203,208`), so there is no Responses-shaped non-streaming body to synthesize — this removes the messages.go non-streaming ambiguity entirely.
   Tests (`responses_test.go`, follow `messages_test.go` style — no mocks, fake provider via existing seams): `TestResponsesEndpointTranslatesRequest` (input[] body → provider sees chat messages), `TestResponsesEndpointStreamsEvents` (SSE frames carry `event:` names from the w1-g response translator), `TestResponsesEndpointRejectsInvalidBody`, `TestResponsesEndpointForcesStreaming` (inbound `stream:false` still streams).

2. **Row 050: translate-mode response.failed flush** (`internal/translation/stream.go` + new `responses_stream_helpers.go`), port `stream.js:203-207` + shared helpers `responsesStreamHelpers.js:18-52`. STEP (a) tests first, STEP (b):
   - **Shared helpers** (`responses_stream_helpers.go`): `formatIncompleteResponsesStreamFailure() []byte` (port `responsesStreamHelpers.js:33-52` — `response.failed` SSE frame: `resp_<unixms>` id, `status:"failed"`, `error{type:"stream_error", code:"stream_disconnected", message:"stream closed before response.completed"}`, framed via `FormatSSE(FormatOpenAIResponses, ...)`); `isResponsesTerminalEvent(chunk map[string]any) bool` (port `:18-23` — event type in {response.completed, response.failed} OR `chunk.response.status` in {completed, failed}).
   - **Translate-mode flush** (`ProcessTranslateStream`): add `ResponsesTerminalSeen bool` to `StreamState`, set when a translated chunk satisfies `isResponsesTerminalEvent`; after drain+flush, when `to == FormatOpenAIResponses` and not seen, write `formatIncompleteResponsesStreamFailure()` before `[DONE]` (ref `stream.js:203-207`). This is the ONLY flush site in scope — it has a live caller (the `/v1/responses` streaming path from Task 1). The passthrough-mode site (`stream.js:397-404`) is PAR-TRANS-050b, deferred to Wave 2 with the responses-passthrough provider path that would call it; NO `ProcessResponsesPassthroughStream` is added here (it would be an uncalled abstraction).
   Tests (`stream_test.go` / `responses_stream_helpers_test.go`): `TestFormatIncompleteResponsesStreamFailure` (exact frame: event name, status, error code), `TestIsResponsesTerminalEvent` (completed/failed by type and by status; non-terminal → false), `TestProcessTranslateStreamSynthesizesResponseFailed` (responses target, no terminal → failure before `[DONE]`), `TestProcessTranslateStreamNoSynthesisWhenCompleted` (terminal seen → none).

3. **Route** (`routes_openai.go`). STEP (a) FIRST: write/extend a reachability test asserting POST `/v1/responses` routes to the handler (in `responses_test.go` via the server's router, mirroring how `messages` route tests work); see it fail. STEP (b): add `responses := api.NewResponsesHandler(router_)` + `r.POST("/v1/responses", responses.Handle)`.

## Binary acceptance criteria

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `grep -c 'POST("/v1/responses"' internal/server/routes_openai.go` → exactly 1.
- `grep -rn 'func init(\|panic(' internal/api/responses.go` → 0 hits.
- All four row-050 tests pass (helper, terminal-detect, translate-mode ×2) — `stream.js:203-207` covered. (Passthrough `:397-404` is PAR-TRANS-050b, Wave 2.)
- The four endpoint tests pass (incl. `TestResponsesEndpointForcesStreaming`).

## Out of scope

PAR-TRANS-050b passthrough-mode flush (`stream.js:397-404`) — Wave 2, with the responses-passthrough provider path that calls it. Non-streaming Responses output (the ref endpoint is streaming-only). Codex executor/auth. Usage estimation (Wave 5). Any change to the w1-g translators (closed).
