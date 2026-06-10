# w1-g — OpenAI Responses API translator pair + /v1/responses endpoint

Rows: PAR-TRANS-031, PAR-TRANS-032, PAR-TRANS-033, PAR-TRANS-034, PAR-TRANS-035 (request directions); PAR-TRANS-036, PAR-TRANS-037, PAR-TRANS-038 (response directions).

Frozen ref (@ 827e5c3): `open-sse/translator/request/openai-responses.js` (1-318), `open-sse/translator/response/openai-responses.js` (1-590), `open-sse/translator/helpers/responsesApiHelper.js:1-22` (`normalizeResponsesInput` only — `convertResponsesApiFormat` at :24+ is a request-path duplicate the translators do not call; NOT ported).

These translators register `openai-responses:openai` and `openai:openai-responses` (both directions each) in `NewRegistry`. `FormatOpenAIResponses` ("openai-responses") already exists in `formats.go:11`. The endpoint task mirrors w1-b's `/v1/messages` precedent: PAR-TRANS-031's gap note says "no `/v1/responses` route exists"; this plan closes the row with route + handler over the registry pipeline.

## Preconditions (grep checks — run before任 task; if any fails, IMPL-BLOCKED)

- `grep -n 'FormatOpenAIResponses' internal/translation/formats.go` → hits (constant exists)
- `grep -rn 'openaiResponsesToOpenAIRequest\|openaiToOpenAIResponsesRequest' internal/translation/` → 0 hits (no prior implementation)
- `grep -n 'POST("/v1/responses"' internal/server/routes_openai.go` → 0 hits
- `grep -n 'func NewRegistry' internal/translation/registry.go` → hit

## Exclusive file ownership

NEW (this plan only):
- `internal/translation/responses_helpers.go`, `responses_helpers_test.go`
- `internal/translation/responses_openai_request.go`, `responses_openai_request_test.go`
- `internal/translation/openai_responses_request.go`, `openai_responses_request_test.go`
- `internal/translation/openai_responses_response.go`, `openai_responses_response_test.go`
- `internal/translation/responses_openai_response.go`, `responses_openai_response_test.go`
- `internal/api/responses.go`, `internal/api/responses_test.go`

TOUCH-ONLY (specified hunks, nothing else):
- `internal/translation/registry.go` — NewRegistry registrations; StreamState gains Responses-prefixed fields (task 4/5)
- `internal/translation/registry_test.go` — wiring tests
- `internal/translation/sse.go` + `sse_test.go` — FormatSSE event framing branch for FormatOpenAIResponses (task 4)
- `internal/server/routes_openai.go` — one `r.POST("/v1/responses", ...)` line (task 6)

## Tasks (TDD: named failing tests FIRST in every task)

1. **Helpers** (`responses_helpers.go`; rows 033/034 substrate). Tests first: `TestNormalizeResponsesInput` (string → single user message item with `input_text`; empty/whitespace string → text "..." placeholder per `responsesApiHelper.js:10-13`; empty array → same placeholder item per `:15-18` (#389); non-string/non-array → nil), `TestClampCallID` (>64 chars truncated to 64 per `request/openai-responses.js:11-13` (#393); ≤64 unchanged; non-string passthrough), `TestNormalizeToolParameters` (nil → `{type:object, properties:{}}`; object without properties → properties added on a COPY; others unchanged — `request/openai-responses.js:192-196`). Port: `normalizeResponsesInput(input any) []any`, `clampCallID(id any) any`, `normalizeToolParameters(params map[string]any) map[string]any`.

2. **Rows 031/032/033/034: responses→chat request** (`responses_openai_request.go`), port `request/openai-responses.js:18-187` exactly:
   - no `input` key → return body unchanged (:19); `normalizeResponsesInput` nil → return body (:34-35).
   - `instructions` → leading `{role:system, content}` message (:25-27).
   - item loop (:50-139): itemType fallback `item.type || (item.role ? "message" : null)` (:53, Droid CLI); `message` items flush pending assistant/tool-results then convert content `input_text`/`output_text`→text, `input_image`→`image_url` with `detail` default "auto", url from `image_url` or `file_id` (:69-80); buffered reasoning attaches as `reasoning_content` on the next assistant message (:82-86) or assistant tool_calls msg (:97-100); `function_call` items append to a `{role:assistant, content:null, tool_calls:[]}` accumulator, SKIPPING empty/missing names (:102-103, #444), `id: item.call_id` unclamped here (:104-111); `function_call_output` flushes accumulators then appends `{role:tool, tool_call_id, content}` with non-string output JSON-stringified (:113-131); `reasoning` items buffer text via summary[].text join "\n" else content[].text join "\n" (:37-48, 133-138).
   - trailing flush (:141-149); tools conversion: keep `tool.function` as-is, convert `{type:function, name, description, parameters, strict}` → chat shape with `normalizeToolParameters`, DROP hosted/nameless tools (:156-176); delete `input`, `instructions`, `include`, `prompt_cache_key`, `store`, `reasoning` (:178-184).
   Tests: `TestResponsesOpenAIInstructions`, `TestResponsesOpenAIStringInput`, `TestResponsesOpenAIEmptyInputPlaceholder`, `TestResponsesOpenAIItemGrouping` (message + function_call + function_call_output ordering incl. assistant flush), `TestResponsesOpenAIRoleOnlyItems` (no type, has role), `TestResponsesOpenAIReasoningBuffering` (reasoning → next assistant `reasoning_content`), `TestResponsesOpenAINamelessFunctionCallSkipped`, `TestResponsesOpenAIHostedToolsDropped`, `TestResponsesOpenAIFieldCleanup` (six keys deleted), `TestResponsesOpenAIImageInput`.

3. **Row 035: chat→responses request** (`openai_responses_request.go`), port `:201-314`: body already has `input` → `{...body, model, stream:true}` (:203); else build `{model, input:[], stream:true, store:false}`; first system message → `instructions`, others skipped (:216-224); user/assistant string or array content → `input_text` (user) / `output_text` (assistant) items, `image_url`→`input_image` with url-string flattening + detail (:237-240), unknown block types serialized as text (:242-244); content-empty assistant (tool_calls only) pushes no message block (:248-257); assistant `tool_calls` → `function_call` items with `clampCallID` (:261-270); `tool` messages → `function_call_output` with string coercion (array join via text/JSON, else JSON) and `clampCallID` (:273-284); no system → `instructions:""` (:288-290); tools chat→responses flat shape with `normalizeToolParameters` (:293-306); passthrough temperature/max_tokens/top_p only when present (:309-311).
   Tests: `TestOpenAIResponsesPassthroughWhenInput`, `TestOpenAIResponsesSystemToInstructions`, `TestOpenAIResponsesNoSystemEmptyInstructions`, `TestOpenAIResponsesContentMapping` (text/image/unknown), `TestOpenAIResponsesToolCallsClamped` (65-char id → 64), `TestOpenAIResponsesToolResultCoercion`, `TestOpenAIResponsesToolsFlatShape`, `TestOpenAIResponsesParamPassthrough`.

4. **Rows 036/037: chat→responses response** (`openai_responses_response.go`) + SSE framing, port `response/openai-responses.js:12-356`:
   - Each emitted event is `map[string]any{"event": <name>, "data": <payload>}`; every payload gets `sequence_number` from a monotonically incremented `state.ResponsesSeq` (:20-25).
   - First chunk: `response.created` + `response.in_progress` with `resp_<chunk.id>` id (:32-58). `reasoning_content` deltas open a `rs_<respId>_<idx>` reasoning item (`response.output_item.added` + `response.reasoning_summary_part.added`, :116-136) and stream `response.reasoning_summary_text.delta` while buffering (:138-148). `<think>`/`</think>` markers in content split into reasoning open/delta/close (:70-94). Text content opens message item + content part once per index then `response.output_text.delta`, buffering full text (:182-217). tool_calls close the message item first (:97-102), then `response.output_item.added` (`fc_<call_id>` item) once per index + `response.function_call_arguments.delta`, buffering args (:255-292). finish_reason closes all open message items, reasoning, tool calls (done events with buffered totals, :104-110, 150-180, 219-253, 294-321) and emits `response.completed` once (:323-338). nil chunk (flush) → same close-all sequence when not completed (:340-356).
   - StreamState additions (registry.go): `ResponsesSeq int`, `ResponsesStarted bool`, `ResponsesID string`, `ResponsesCreated int64`, `ResponsesReasoningID string`, `ResponsesReasoningIndex int`, `ResponsesReasoningBuf string`, `ResponsesReasoningDone bool`, `ResponsesInThinking bool`, `ResponsesMsgItemAdded/ContentAdded/ItemDone map[int]bool`, `ResponsesMsgTextBuf map[int]string`, `ResponsesFuncCallIDs/FuncNames/FuncArgsBuf map[int]string`, `ResponsesFuncItemDone/FuncArgsDone map[int]bool` — maps initialized in `NewStreamState` (no nil-map writes).
   - `FormatSSE` (sse.go): for `FormatOpenAIResponses`, an event map `{event, data}` frames as `event: <name>\ndata: <json-of-data>\n\n` (event name preservation; PAR-TRANS-036). Non-event maps keep the default `data:` frame.
   Tests: `TestOpenAIResponsesResponseLifecycle` (created/in_progress once, sequence numbers strictly increasing), `TestOpenAIResponsesResponseTextDeltas` (item added once, deltas, done-on-finish with full text), `TestOpenAIResponsesResponseReasoning` (rs_ item, summary part, deltas, done set), `TestOpenAIResponsesResponseThinkMarkers` (PAR-PR-1084 does NOT apply here — the ref splits literal markers in THIS translator, :70-84; port as-is), `TestOpenAIResponsesResponseToolCalls`, `TestOpenAIResponsesResponseFlush` (nil chunk closes everything + completed), `TestFormatSSEResponsesEventFraming`.

5. **Row 038: responses→chat response** (`responses_openai_response.go`), port `:360-586`:
   - nil chunk → final chunk with `computeFinishReason` (`tool_calls` if any tool call seen else `stop`, :360-364) + usage when buffered; suppressed when already sent or not started (:371-397).
   - event resolution `chunk.type || chunk.event`, data `chunk.data || chunk` (:400-401). First event initializes `chatcmpl-` id/created (:404-410). `response.output_text.delta` → content delta (:413-428); `output_text.done` ignored (:431-433); `response.output_item.added` with `function_call`/`custom_tool_call` item → tool_calls start chunk, `call_id` fallback `call_<now-ms>` (:436-461); `response.function_call_arguments.delta` + `response.custom_tool_call_input.delta` → arguments delta (:464-484); `output_item.done` for those types increments `toolCallIndex`, emits nil (:487-490); `response.completed`/`response.done` extracts usage (`input_tokens`/`prompt_tokens` aliases, cached via `input_tokens_details.cached_tokens` or `cache_read_input_tokens` → `prompt_tokens_details.cached_tokens` when >0) and emits the final finish chunk once (:493-543); `error`/`response.failed` dedup-guarded → `[Error] <message>` content chunk with finish stop (:546-569); `response.reasoning_summary_text.delta` → `reasoning_content` delta (:572-582); all other events → nil (:585).
   - StreamState additions: `ResponsesChatID string`, `ResponsesToolCallIndex int`, `ResponsesCurrentToolCallID string` (+ reuse `FinishReasonSent`, `Usage`).
   Tests: `TestResponsesOpenAIResponseTextDelta`, `TestResponsesOpenAIResponseToolCallLifecycle` (added → args deltas → done → completed yields finish_reason tool_calls), `TestResponsesOpenAIResponseCompletedUsage` (aliases + cached_tokens), `TestResponsesOpenAIResponseErrorEvent` (error then response.failed → single chunk), `TestResponsesOpenAIResponseReasoningDelta`, `TestResponsesOpenAIResponseFlushFinish` (nil → finish chunk once).

6. **Registration + endpoint (closes PAR-TRANS-031)**. `NewRegistry`: `Register(FormatOpenAIResponses, FormatOpenAI, responsesToOpenAIRequest, responsesToOpenAIResponse)` and `Register(FormatOpenAI, FormatOpenAIResponses, openaiToResponsesRequest, openaiToResponsesResponse)` (ref registers each direction across the two files, :316-318 / :588-590). `internal/api/responses.go`: `POST /v1/responses` handler mirroring `internal/api/messages.go` structure (read 3 neighbors first: messages.go, chat.go, embeddings.go): translate request `FormatOpenAIResponses→FormatOpenAI` via registry, dispatch through the same provider path chat uses, translate streaming response `FormatOpenAI→FormatOpenAIResponses` via `ProcessTranslateStream`; non-streaming returns the chat-shaped response converted through the response translator's event collection. Route added in `routes_openai.go`.
   Tests: `TestNewRegistryWiresResponsesPair` (all four translators non-nil; response aliases identity-checked via reflect like `TestResponseAliasesUseGeminiTranslator`), `internal/api/responses_test.go`: `TestResponsesEndpointTranslatesRequest`, `TestResponsesEndpointStreamsEvents` (SSE frames carry `event:` names), `TestResponsesEndpointRejectsInvalidBody` — same harness style as `messages_test.go` (no mocks; fake provider via the package's existing test seams).

## Binary acceptance criteria

- `go test ./...` and `go vet ./...` green.
- `grep -c 'sequence_number' internal/translation/openai_responses_response.go` ≥ 1; every emitted event carries it.
- `grep -n 'POST("/v1/responses"' internal/server/routes_openai.go` → exactly 1 hit.
- `grep -rn 'Date.now\|time.Now().UnixNano' internal/translation/responses_*.go internal/translation/openai_responses_*.go` — time usage only via `time.Now()` patterns already used in the package (UnixMilli/Unix), no `init()`, no panics, errors wrapped.
- All four directions resolvable from `NewRegistry()`.
- Call-id clamp boundary test passes with exactly 64 chars retained.

## Out of scope

`convertResponsesApiFormat` (`responsesApiHelper.js:24+`, dead duplicate). Codex executor/auth (`FormatCodex` flows — Wave 2/3). Usage estimation/persistence (Wave 5). RTK/compression. Cursor/Kiro/Ollama formats (w1-h..j). Passthrough-mode Responses event preservation in `ProcessPassthroughStream` beyond what w1-c shipped. Any provider adapter changes (Wave 2).
