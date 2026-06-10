# w1-h — ollama + commandcode translator pairs

Rows: **PAR-TRANS-058, PAR-TRANS-059, PAR-TRANS-060, PAR-TRANS-061** (added 2026-06-10 — granular rows for the WAVE-MAP Wave-1 "12 wire formats" mandate, cross-ref PAR-PROV-010/040); PAR-TRANS-002 ollama + commandcode registration clauses close alongside.

Frozen ref (@ 827e5c3), read whole before porting:
- `open-sse/translator/request/openai-to-ollama.js` (1-193)
- `open-sse/translator/response/ollama-to-openai.js` (1-153)
- `open-sse/translator/request/openai-to-commandcode.js` (1-171)
- `open-sse/translator/response/commandcode-to-openai.js` (1-198)

NOT read/ported here: `utils/ollamaTransform.js` — its sole consumer is the
ollama-format client endpoint (`src/app/api/v1/api/chat/route.js`), which is
Wave-4 routing scope; porting it now would create orphan utility code.

## Preconditions (a "0 hits" expectation means grep exits 1 — that IS the pass)

- `grep -n 'FormatOllama\|FormatCommandCode' internal/translation/formats.go` → both present
- `grep -rn 'openaiToOllamaRequest\|ollamaToOpenAIResponse\|openaiToCommandCodeRequest\|commandcodeToOpenAIResponse' internal/translation/` → 0 hits
- `grep -n 'NewNDJSONScanner' internal/providers/utils/sse.go` → present (w1-c)

## Exclusive file ownership

NEW: `internal/translation/openai_ollama_request.go` + `_test.go`, `ollama_openai_response.go` + `_test.go`, `openai_commandcode_request.go` + `_test.go`, `commandcode_openai_response.go` + `_test.go`.
TOUCH-ONLY: `registry.go` (registrations + StreamState fields below), `registry_test.go` (wiring tests).

## Tasks (TDD: named failing tests FIRST per task)

1. **Row 058: openai→ollama request** (`openai_ollama_request.go`), port `openai-to-ollama.js:18-189`. Signature `(model string, body map[string]any, stream bool, credentials map[string]any)` — credentials ignored (ref is 3-arg). Result `{model, messages, stream}`; `options.temperature/num_predict/top_p` only when present (:26-41); tools/tool_choice passthrough (:44-51). `normalizeMessages` (:62-142): first pass builds tool_call_id→tool_name map from assistant tool_calls; tool messages → `{role:tool, tool_name, content}` with fallback `msg.name` else `"unknown_tool"`, skipped when content empty (:82-95); assistant+tool_calls → ollama tool_calls `{type:function, function:{index, name, arguments:PARSED-OBJECT}}` — string arguments go through `JSON.parse` with NO catch in the ref (:108), so malformed arguments must return a wrapped error (Go equivalent of the JS throw); normal messages: content flattened to string via text-block join "\n" (:148-163), data-URI images extracted as RAW base64 (strip prefix) into `images[]` (:171-189), empty non-assistant messages skipped (:127).
   Tests: `TestOpenAIOllamaOptionsMapping`, `TestOpenAIOllamaToolResultNaming` (map hit, msg.name fallback, unknown_tool), `TestOpenAIOllamaAssistantToolCallsParsedArgs` (object args; malformed string args → error), `TestOpenAIOllamaContentFlattenAndImages` (text join; data-URI → raw base64; http URL ignored), `TestOpenAIOllamaSkipsEmptyNonAssistant`.

2. **Row 059: ollama→openai response** (`ollama_openai_response.go`), port `ollama-to-openai.js:15-121` as a ResponseTranslator. State init on first chunk: `chatcmpl-<unix-ms>` id, created, model from chunk else state (:19-25). `done:true` → final chunk with finish_reason `tool_calls` when `done_reason=="tool_calls"` OR prior tool_calls seen, else `stop`; usage from `prompt_eval_count`/`eval_count` (:30-51). Content chunks: content/thinking(→`reasoning_content`)/tool_calls deltas, empty → nil (:54-93); ollama tool_calls → OpenAI shape with `id` fallback `call_<i>_<unix-ms>`, arguments JSON-stringified when not string (:109-121). Also port `ollamaBodyToOpenAI` (:126-149) as `OllamaBodyToOpenAI` — part of row 059: the ref consumes it at `handlers/chatCore/nonStreamingHandler.js:126` for non-streaming ollama responses; the g0router consumer arrives with that handler's wave, the converter is the row's translation surface.
   StreamState additions: `OllamaID string`, `OllamaCreated int64`, `OllamaModel string`, `OllamaHadToolCalls bool`, `OllamaContent string`, `OllamaThinking string` (accumulators :65-70).
   Tests: `TestOllamaOpenAIInitAndContentDelta`, `TestOllamaOpenAIThinkingDelta`, `TestOllamaOpenAIToolCallsAndDoneReason` (hadToolCalls path + done_reason path), `TestOllamaOpenAIUsageOnDone`, `TestOllamaBodyToOpenAI` (content+thinking+tool_calls+empty-content default).

3. **Row 060: openai→commandcode request** (`openai_commandcode_request.go`), port `openai-to-commandcode.js:16-168`: `flattenText` (:16-28), `toContentBlocks` (:30-51 — images become `{type:text, text:"[image omitted]"}`), `safeParseJson` (:53-57 — malformed → `{}`), `convertMessages` (:59-110 — system texts join "\n\n" into top-level STRING param; tool → tool-result block `{toolCallId, toolName, output:{type:text, value}}`; assistant → text + tool-call blocks `{toolCallId, toolName, input}`; default branch → user with content blocks), `convertTools` (:112-132 — both OpenAI and Claude tool shapes → `{name, description, input_schema}`). Envelope (:134-168): `threadId` = random UUID (`uuid.NewRandom`, wrapped error — no panic), `memory:""`, `config` with `workingDir` = `os.Getwd()` (error → ""), `date` = `time.Now().UTC().Format("2006-01-02")`, `environment` = `runtime.GOOS`, empty structure/git fields verbatim; `params` with defaults `max_tokens` 64000 (after `max_tokens` else `max_output_tokens`), `temperature` 0.3, `stream: stream != false` semantics → Go: the bool as passed; `system` only when non-empty; `top_p` when non-nil.
   Tests: `TestCommandCodeSystemExtraction` ("\n\n" join, string param), `TestCommandCodeToolResultBlocks`, `TestCommandCodeAssistantToolCallBlocks` (safeParseJson malformed → {}), `TestCommandCodeToolsBothShapes`, `TestCommandCodeEnvelopeDefaults` (64000/0.3/threadId non-empty/GOOS), `TestCommandCodeImageOmitted`.

4. **Row 061: commandcode→openai response** (`commandcode_openai_response.go`), port `commandcode-to-openai.js:21-195`: pass through chunks already shaped `object=="chat.completion.chunk"` (:62-64); state init `chatcmpl-<unix-ms>` (:21-34); event switch — `text-delta` (first chunk injects `role:assistant`, :87-95), `reasoning-delta` → `reasoning_content` (:96-106), `tool-input-start` (per-id index map, :107-127), `tool-input-delta` (unknown id → skip, :128-140), `tool-call` consolidated only when id unseen (:141-160), `finish-step` records `mapFinishReason` + usage (:161-165), `finish` emits final chunk with usage from `totalUsage ?? state.usage` mapping inputTokens/outputTokens/totalTokens (:166-179), `error` emits error-content chunk + stop chunk (:180-187), all other event types ignored (:188-191). `mapFinishReason` (:46-56) incl. `tool-calls`/`tool_use` → `tool_calls`, `content-filter` → `content_filter`, default passthrough-or-stop. NOTE: the raw-string/`data:`-framing tolerance (:66-79) is NOT ported — g0router's NDJSON scanner (w1-c) delivers parsed maps; document with a comment citing :66-79.
   StreamState additions: `CommandCodeID string`, `CommandCodeCreated int64`, `CommandCodeModel string`, `CommandCodeChunkIndex int`, `CommandCodeToolIndex int`, `CommandCodeToolIndexByID map[string]int`, `CommandCodeFinishReason string`, `CommandCodeUsage map[string]any` (init in NewStreamState).
   Tests: `TestCommandCodeOpenAIPassthroughChunk`, `TestCommandCodeOpenAITextAndRoleInjection`, `TestCommandCodeOpenAIReasoningDelta`, `TestCommandCodeOpenAIToolInputLifecycle` (start/delta/unknown-id skip), `TestCommandCodeOpenAIConsolidatedToolCallDedup`, `TestCommandCodeOpenAIFinishUsage` (finish-step then finish; totalUsage precedence), `TestCommandCodeOpenAIErrorTwoChunks`, `TestCommandCodeOpenAIFinishReasonMapping`.

5. **Registration (PAR-TRANS-002 clauses)**: `Register(FormatOpenAI, FormatOllama, openaiToOllamaRequest, openaiToOllamaLinesResponse?)` — NO: ref registers `(OPENAI, OLLAMA, request, null)` and `(OLLAMA, OPENAI, null, ollamaToOpenAI)`; the lines-transform is a util, not a registered translator (consumed by the ollama endpoint later). CommandCode: `(OPENAI, COMMANDCODE, request, null)` (:170) and `(COMMANDCODE, OPENAI, null, response)` (:197). Four `Register` calls total.
   Tests: `TestNewRegistryWiresOllamaPair`, `TestNewRegistryWiresCommandCodePair` (presence + reflect identity, registry_test.go pattern).

## Binary acceptance criteria

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `grep -rn 'func init(' <new files>` → 0 hits; `grep -rn 'panic(' <new files>` → 0 hits.
- `grep -c 'unknown_tool' internal/translation/openai_ollama_request.go` ≥ 1.
- `grep -c '\[image omitted\]' internal/translation/openai_commandcode_request.go` ≥ 1.
- `TestCommandCodeOpenAIConsolidatedToolCallDedup` proves no duplicate tool_call when `tool-call` follows `tool-input-*` for the same id (covered by go test).

## Out of scope

Ollama/CommandCode HTTP endpoints + provider adapters (Wave 2/4). The TransformStream/Response shell of ollamaTransform.js. kiro/cursor (w1-i/j). gemini-format client requests (w1-k). `openai-to-kiro.old.js` (dead).
