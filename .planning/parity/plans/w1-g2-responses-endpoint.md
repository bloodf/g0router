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

NEW: `internal/api/responses.go` + `internal/api/responses_test.go`.
TOUCH-ONLY: `internal/server/routes_openai.go` (add `responses := api.NewResponsesHandler(router_)` + `r.POST("/v1/responses", responses.Handle)`), `internal/translation/stream.go` (PAR-TRANS-050 flush in `ProcessTranslateStream` — the only translation-pkg change; coordinate: no other in-flight plan touches stream.go's flush path).
Non-overlap: `responses.go`/`responses_test.go` match no other plan's ownership.

## Tasks (STEP (a) write named failing tests; STEP (b) implement)

1. **ResponsesHandler** (`internal/api/responses.go`), mirror `messages.go:14-110` structure exactly (read 3 neighbors: messages.go, chat.go, embeddings.go):
   - `ResponsesHandler{router modelResolver, registry *translation.Registry}`, `NewResponsesHandler(router *inference.Router)`.
   - `Handle(ctx)`: unmarshal body; `model`/`stream` extraction (Responses streams default true per ref `openaiToOpenAIResponsesRequest` sets `stream:true`, but honor explicit `stream:false`); `TranslateRequest(FormatOpenAIResponses, FormatOpenAI, model, body, stream, nil)`; marshal→`schemas.ChatRequest`→`PreprocessChatRequest`; `ResolveForModel`; streaming path uses `ProcessTranslateStream(ctx, ch, registry, FormatOpenAI, FormatOpenAIResponses, state)` (note the to-format is FormatOpenAIResponses so the SSE event framing from w1-g applies); non-streaming returns the OpenAI-shaped response unchanged (same decision as messages.go:100-110 — document with the same comment).
   Tests (`responses_test.go`, follow `messages_test.go` style — no mocks, fake provider via existing seams): `TestResponsesEndpointTranslatesRequest` (input[] body → provider sees chat messages), `TestResponsesEndpointStreamsEvents` (SSE frames carry `event:` names from the w1-g response translator), `TestResponsesEndpointRejectsInvalidBody`, `TestResponsesEndpointNonStreaming`.

2. **Row 050: response.failed flush** (`internal/translation/stream.go`), port `stream.js:202`: in `ProcessTranslateStream`, after the channel drains and the flush translator runs, when the target format is `FormatOpenAIResponses` AND no terminal event (`response.completed`/`response.failed`) was emitted during the stream, synthesize a `response.failed` event before `[DONE]`. Track "terminal seen" via a flag on `StreamState` (`ResponsesTerminalSeen bool`, set when a `response.completed`/`response.failed` chunk passes through). The synthesized event mirrors the ref shape at `stream.js:202` (response object with `status:"failed"`, an `error`); frame via `FormatSSE(FormatOpenAIResponses, ...)`.
   Tests (`stream_test.go`): `TestProcessTranslateStreamSynthesizesResponseFailed` (responses target, stream ends with no terminal → a `response.failed` event is written before `[DONE]`), `TestProcessTranslateStreamNoSynthesisWhenCompleted` (terminal already emitted → no synthetic event).

3. **Route** (`routes_openai.go`): add the handler construction + `r.POST("/v1/responses", responses.Handle)` line.
   Test: extend the server route test (if present) or assert in `responses_test.go` that the handler is reachable.

## Binary acceptance criteria

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `grep -c 'POST("/v1/responses"' internal/server/routes_openai.go` → exactly 1.
- `grep -rn 'func init(\|panic(' internal/api/responses.go` → 0 hits.
- `TestProcessTranslateStreamSynthesizesResponseFailed` and `...NoSynthesisWhenCompleted` both pass.
- The four endpoint tests pass.

## Out of scope

Passthrough-mode `response.failed` (`stream.js:397` — `ProcessPassthroughStream`, deferred until a Responses passthrough path exists). Codex executor/auth. Usage estimation (Wave 5). Any change to the w1-g translators (closed).
