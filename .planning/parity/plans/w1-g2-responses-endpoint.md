# w1-g2 — /v1/responses endpoint + response.failed flush

Rows: PAR-TRANS-031 (route clause — the translator-existence clause closed in w1-g; this delivers the `/v1/responses` HTTP surface its gap note named), PAR-TRANS-050 (stream flush synthesizes `response.failed` for Responses streams that never reached a terminal event), PAR-TRANS-002 (openai-responses format now reachable end-to-end). Depends on w1-g translators (MERGED `6640b33`+`ca8274e`).

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
TOUCH-ONLY: `internal/server/routes_openai.go` (add `responses := api.NewResponsesHandler(router_)` + `r.POST("/v1/responses", responses.Handle)`), `internal/translation/stream.go` (PAR-TRANS-050 flush in `ProcessTranslateStream` — the only translation-pkg change; coordinate: no other in-flight plan touches stream.go's flush path).
Non-overlap: `responses.go`/`responses_test.go` match no other plan's ownership.

## Tasks (STEP (a) write named failing tests; STEP (b) implement)

1. **ResponsesHandler** (`internal/api/responses.go`), mirror `messages.go:14-110` structure exactly (read 3 neighbors: messages.go, chat.go, embeddings.go):
   - `ResponsesHandler{router modelResolver, registry *translation.Registry}`, `NewResponsesHandler(router *inference.Router)`.
   - `Handle(ctx)`: unmarshal body; `model`/`stream` extraction (ref `openai-responses.js:203` `return {...body, model, stream:true}` and `:208` `stream:true` — Responses requests default streaming; honor explicit `stream:false` from the inbound body); `TranslateRequest(FormatOpenAIResponses, FormatOpenAI, model, body, stream, nil)`; marshal→`schemas.ChatRequest`→`PreprocessChatRequest`; `ResolveForModel`; streaming path uses `ProcessTranslateStream(ctx, ch, registry, FormatOpenAI, FormatOpenAIResponses, state)` (note the to-format is FormatOpenAIResponses so the SSE event framing from w1-g applies); non-streaming returns the OpenAI-shaped response unchanged (same decision as messages.go:100-110 — document with the same comment).
   Tests (`responses_test.go`, follow `messages_test.go` style — no mocks, fake provider via existing seams): `TestResponsesEndpointTranslatesRequest` (input[] body → provider sees chat messages), `TestResponsesEndpointStreamsEvents` (SSE frames carry `event:` names from the w1-g response translator), `TestResponsesEndpointRejectsInvalidBody`, `TestResponsesEndpointNonStreaming`.

2. **Row 050: response.failed flush** (`internal/translation/stream.go` + new `responses_stream_helpers.go`), port BOTH ref flush sites (`stream.js:203-207` translate-mode AND `:397-404` passthrough-mode) plus their shared helpers from `responsesStreamHelpers.js:18-52`. STEP (a) tests first, STEP (b):
   - **Shared helpers** (`responses_stream_helpers.go`): `formatIncompleteResponsesStreamFailure() []byte` (port `responsesStreamHelpers.js:33-52` — `response.failed` SSE frame with `resp_<unixms>` id, `status:"failed"`, `error{type:"stream_error", code:"stream_disconnected", message:"stream closed before response.completed"}`, framed via `FormatSSE(FormatOpenAIResponses, ...)`); `isResponsesTerminalEvent(chunk map[string]any) bool` (port `:18-23` — event type in {response.completed, response.failed} OR `chunk.response.status` in {completed, failed}).
   - **Translate-mode** (`ProcessTranslateStream`): track `ResponsesTerminalSeen bool` on `StreamState`, set when a translated chunk satisfies `isResponsesTerminalEvent`; after drain+flush, when `to == FormatOpenAIResponses` and not seen, write `formatIncompleteResponsesStreamFailure()` before `[DONE]` (ref `stream.js:203-207`).
   - **Passthrough-mode**: add `ProcessResponsesPassthroughStream(w io.Writer, ch <-chan *schemas.StreamChunk) (StreamSummary, error)` — a responses-format passthrough variant (the existing `ProcessPassthroughStream(w, ch)` at stream.go:93 and its only caller chat.go:32 stay UNCHANGED). It mirrors passthrough normalization but tracks terminal-seen and, on flush with none seen, synthesizes the same failure frame before `[DONE]` (ref `stream.js:397-404`, gated there on `source==target==OPENAI_RESPONSES`). Its routing consumer arrives with the Wave-2 responses-passthrough provider path; Wave-1 ships+unit-tests the function.
   Tests (`stream_test.go` / `responses_stream_helpers_test.go`): `TestFormatIncompleteResponsesStreamFailure` (exact frame: event name, status, error code), `TestIsResponsesTerminalEvent` (completed/failed by type and by status; non-terminal → false), `TestProcessTranslateStreamSynthesizesResponseFailed` (responses target, no terminal → failure before `[DONE]`), `TestProcessTranslateStreamNoSynthesisWhenCompleted` (terminal seen → none), `TestProcessResponsesPassthroughSynthesizesFailure`, `TestProcessResponsesPassthroughNoSynthesisWhenTerminal`.

3. **Route** (`routes_openai.go`). STEP (a) FIRST: write/extend a reachability test asserting POST `/v1/responses` routes to the handler (in `responses_test.go` via the server's router, mirroring how `messages` route tests work); see it fail. STEP (b): add `responses := api.NewResponsesHandler(router_)` + `r.POST("/v1/responses", responses.Handle)`.

## Binary acceptance criteria

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `grep -c 'POST("/v1/responses"' internal/server/routes_openai.go` → exactly 1.
- `grep -rn 'func init(\|panic(' internal/api/responses.go` → 0 hits.
- All six row-050 tests pass (helper, terminal-detect, translate-mode ×2, passthrough-mode ×2) — both `stream.js:203-207` and `:397-404` behaviors covered.
- The four endpoint tests pass.

## Out of scope

Wiring `ProcessResponsesPassthroughStream` into a live route (its provider path is Wave-2; the function + tests ship here per row 050). Codex executor/auth. Usage estimation (Wave 5). Any change to the w1-g translators (closed).
