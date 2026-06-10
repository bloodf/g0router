# 9router Translation Engine Parity Matrix

Reference: `/Users/heitor/Developer/github.com/bloodf/_refs/9router` @ `827e5c3`
Target: `/Users/heitor/Developer/github.com/bloodf/g0router`

## Row Table

| ID | Behavior | Evidence (file:line) | g0router status | Notes |
|---|---|---|---|---|
| PAR-TRANS-001 | Translator registry maps `from:to` pairs for requests and responses separately | `open-sse/translator/index.js:10-25` | MISSING | g0router has no central translator registry; each provider owns a converter |
| PAR-TRANS-002 | Supported formats: `openai`, `openai-responses`, `claude`, `gemini`, `gemini-cli`, `vertex`, `codex`, `antigravity`, `kiro`, `cursor`, `ollama`, `commandcode` | `open-sse/translator/formats.js:2-16` | MISSING | g0router only implements `openai`, `anthropic`, `gemini` shaped providers |
| PAR-TRANS-003 | Endpoint-based format detection: `/v1/responses` → `openai-responses`, `/v1/messages` → `claude` | `open-sse/translator/formats.js:22-35` | MISSING | g0router `internal/api/chat.go:23-28` parses only `/v1/chat/completions` |
| PAR-TRANS-004 | Request translation normalizes through OpenAI as intermediate (`source → openai → target`) | `open-sse/translator/index.js:92-110` | MISSING | g0router converts directly from OpenAI schema to provider schema in per-provider converters |
| PAR-TRANS-005 | Response translation normalizes through OpenAI as intermediate (`target → openai → source`) | `open-sse/translator/index.js:160-186` | MISSING | g0router emits OpenAI-shaped `StreamChunk` directly from provider converters |
| PAR-TRANS-006 | `stripContentTypes` removes image/audio parts from messages when provider model opts in via `strip[]` | `open-sse/translator/index.js:58-72` | MISSING | No content stripping stage exists in g0router |
| PAR-TRANS-007 | `normalizeThinkingConfig` removes thinking block if last message is not user | `open-sse/translator/index.js:83` | MISSING | No thinking normalization in g0router |
| PAR-TRANS-008 | `ensureToolCallIds` validates tool IDs against `^[a-zA-Z0-9_-]+$` and regenerates invalid IDs | `open-sse/translator/helpers/toolCallHelper.js:4-67` | MISSING | g0router `schemas.Message.ToolCalls` accepts any string ID with no validation |
| PAR-TRANS-009 | `fixMissingToolResponses` inserts empty `role:tool` messages after assistant tool_calls if missing | `open-sse/translator/helpers/toolCallHelper.js:116-147` | MISSING | g0router `internal/providers/anthropic/converter.go:133-164` does not insert placeholder tool results |
| PAR-TRANS-010 | `filterToOpenAIFormat` strips `thinking`, `redacted_thinking`, signature, cache_control; normalizes roles/tools/tool_choice back to OpenAI | `open-sse/translator/helpers/openaiHelper.js:9-129` | MISSING | g0router `Message` schema has no `thinking` or `reasoning_content` fields |
| PAR-TRANS-011 | Claude request translator uses `adjustMaxTokens`: missing/zero `max_tokens` → `DEFAULT_MAX_TOKENS=64000`; boosts to `DEFAULT_MIN_TOKENS=32000` when tools present and value below it; ensures `max_tokens > thinking.budget_tokens` by setting `budget_tokens + 1024` | `open-sse/translator/helpers/maxTokensHelper.js:8-26`, constants `open-sse/config/runtimeConfig.js:41-42` | MISSING | g0router `internal/providers/anthropic/converter.go:88-100` uses static `defaultMaxTokens=4096` without tool/thinking adjustments. *(Row corrected 2026-06-09: original text said "min 4096", conflating g0router's constant with 9router's — frozen-ref constants are 64000/32000; see GATE-RESOLUTION.md.)* |
| PAR-TRANS-012 | OpenAI → Claude maps `max_tokens` to Anthropic `max_tokens` | `open-sse/translator/request/openai-to-claude.js:16` | HAVE | g0router `internal/providers/anthropic/converter.go:94-100` maps `MaxTokens` to `MaxTokens` |
| PAR-TRANS-013 | OpenAI → Claude maps `temperature`, passes through unchanged | `open-sse/translator/request/openai-to-claude.js:21-23` | HAVE | g0router `internal/providers/anthropic/converter.go:101-103` passes `Temperature` through |
| PAR-TRANS-014 | OpenAI → Claude extracts `role:system` messages into top-level `system` array with `CLAUDE_SYSTEM_PROMPT` prefix and `cache_control` | `open-sse/translator/request/openai-to-claude.js:27-134` | MISSING | g0router `internal/providers/anthropic/converter.go:111-119` maps system to a single flat string with no prompt injection or cache_control |
| PAR-TRANS-015 | OpenAI → Claude merges consecutive same-role messages and splits tool_result into separate user message | `open-sse/translator/request/openai-to-claude.js:42-88` | MISSING | g0router `convertMessages` does not merge or split; tool results become `role:user` blocks inline |
| PAR-TRANS-016 | OpenAI → Claude adds `cache_control: {type:"ephemeral",ttl:"1h"}` to last system block (`openai-to-claude.js:130`) and last tool (`:164-166`); last cacheable block of last assistant message gets `cache_control: {type:"ephemeral"}` WITHOUT ttl (`:91-105`, verbatim `block.cache_control = { type: "ephemeral" };`) | `open-sse/translator/request/openai-to-claude.js:91-105,128-131,164-166`, `open-sse/translator/helpers/claudeHelper.js:94-103` | MISSING | g0router adds no cache_control. *(Row corrected 2026-06-09: original text claimed ttl:"1h" on the assistant block; frozen ref has ttl only on system/tools blocks; see GATE-RESOLUTION.md.)* |
| PAR-TRANS-017 | OpenAI → Claude converts `response_format.type:json_schema` into appended system prompt with schema | `open-sse/translator/request/openai-to-claude.js:108-121` | MISSING | g0router `ChatRequest.ResponseFormat` exists (`internal/schemas/chat.go:87-99`) but is never translated by Anthropic converter |
| PAR-TRANS-018 | OpenAI → Claude converts tools: passes non-`function` types through; rewrites `function` to `{name,description,input_schema}` with optional OAuth prefix | `open-sse/translator/request/openai-to-claude.js:137-165` | PARTIAL | g0router `convertTools` (`internal/providers/anthropic/converter.go:166-176`) converts to `{name,description,input_schema}` but lacks pass-through for non-function tools and has no OAuth prefix logic |
| PAR-TRANS-019 | OpenAI → Claude `tool_choice` mapping: `"required"` → `{type:"any"}`; `{type:"function",function:{name}}` → `{type:"tool",name}`; rejects unknown object types | `open-sse/translator/request/openai-to-claude.js:294-324` | PARTIAL | g0router `convertToolChoice` (`internal/providers/anthropic/converter.go:178-192`) maps `auto/none/function/required` but does not reject malformed types; defaults to `auto` |
| PAR-TRANS-020 | OpenAI → Claude maps `reasoning_effort` to `thinking.{type:"enabled",budget_tokens}` with lookup `{none:0,low:4096,medium:8192,high:16384,xhigh:32768}` | `open-sse/translator/request/openai-to-claude.js:182-199` | MISSING | g0router `ChatRequest` has no `reasoning_effort` field |
| PAR-TRANS-021 | OpenAI → Claude supports explicit `thinking` block pass-through with `budget_tokens` and `max_tokens` | `open-sse/translator/request/openai-to-claude.js:173-180` | MISSING | g0router `ChatRequest` has no `thinking` field |
| PAR-TRANS-022 | Claude request cloaking: OAuth tokens (`sk-ant-oat`) trigger billing header injection, fake user ID, and `_cc` tool suffix with decoy tools | `open-sse/utils/claudeCloaking.js:128-155`, `open-sse/utils/claudeCloaking.js:34-80` | MISSING | g0router has no cloaking layer |
| PAR-TRANS-023 | OpenAI → Gemini maps `max_tokens` → `generationConfig.maxOutputTokens`, `top_p`→`topP`, `top_k`→`topK` | `open-sse/translator/request/openai-to-gemini.js:47-59` | HAVE | g0router `internal/providers/gemini/converter.go:109-129` maps `MaxTokens`, `TopP`, `Stop` but omits `TopK` |
| PAR-TRANS-024 | OpenAI → Gemini places `role:system` into `systemInstruction` (only if more than one message) | `open-sse/translator/request/openai-to-gemini.js:92-97` | HAVE | g0router `internal/providers/gemini/converter.go:131-140` maps system to `SystemInstruction` |
| PAR-TRANS-025 | OpenAI → Gemini converts assistant `reasoning_content` to `thought:true` part plus `thoughtSignature` separator | `open-sse/translator/request/openai-to-gemini.js:105-115` | MISSING | g0router `Message` has no `reasoning_content`; Gemini `Part` lacks `thought`/`thoughtSignature` fields |
| PAR-TRANS-026 | OpenAI → Gemini co-locates tool calls and tool responses: emits tool responses immediately after assistant `functionCall` when responses exist | `open-sse/translator/request/openai-to-gemini.js:124-182` | MISSING | g0router `convertMessages` (`internal/providers/gemini/converter.go:154-186`) emits tool results as separate `role:user` contents without pairing |
| PAR-TRANS-027 | OpenAI → Gemini sanitizes function names: replace invalid chars with `_`, prefix with `_` if first char invalid, truncate to 64 | `open-sse/translator/request/openai-to-gemini.js:26-36` | MISSING | g0router `convertTools` (`internal/providers/gemini/converter.go:188-198`) passes names unchanged |
| PAR-TRANS-028 | OpenAI → Gemini cleans JSON schemas via `cleanJSONSchemaForAntigravity`: removes unsupported keywords, flattens `anyOf/oneOf/allOf`, converts `const`→`enum`, ensures `type:object`, adds placeholder properties to empty objects | `open-sse/translator/helpers/geminiHelper.js:298-371` | MISSING | g0router passes `Function.Parameters` map unchanged |
| PAR-TRANS-029 | Gemini schema sanitizer removes keywords: `minLength`, `maxLength`, `pattern`, `format`, `default`, `examples`, `$schema`, `$defs`, `additionalProperties`, `anyOf`, `oneOf`, `allOf`, `title`, `x-*`, UI styling keys, etc. | `open-sse/translator/helpers/geminiHelper.js:3-23` | MISSING | No schema normalization in g0router |
| PAR-TRANS-030 | OpenAI → Gemini (plain path) DROPS `tool_choice` entirely — `openaiToGeminiBase` never reads it and emits no `toolConfig`; the only `toolConfig` in the frozen ref is the Antigravity envelope's hardcoded `functionCallingConfig: {mode:"VALIDATED"}` (`openai-to-gemini.js:305-308,416-417`) | `open-sse/translator/request/openai-to-gemini.js:39-221` (no tool_choice reference), `:305-308,416-417` (Antigravity VALIDATED) | PARTIAL | g0router `convertToolChoice` (`internal/providers/gemini/converter.go:200-211`) invents an AUTO/NONE/ANY mapping not present in the ref translator. *(Row corrected 2026-06-09: original text claimed an auto/none/required→mode mapping that does not exist in the frozen ref; see GATE-RESOLUTION.md.)* |
| PAR-TRANS-031 | OpenAI Responses API ↔ OpenAI Chat Completions bidirectional translation exists | `open-sse/translator/request/openai-responses.js:1-318`, `open-sse/translator/response/openai-responses.js:1-590` | MISSING | g0router `Responses`/`ResponsesStream` stubs return 501 (`internal/providers/openai/stubs.go:17-23`) and no `/v1/responses` route exists |
| PAR-TRANS-032 | Responses → Chat: `instructions` becomes `role:system`; `input[]` items grouped into messages; `function_call`/`function_call_output`/`reasoning` item types handled | `open-sse/translator/request/openai-responses.js:24-139` | MISSING | No Responses handler in g0router |
| PAR-TRANS-033 | Responses → Chat clamps `call_id` to 64 characters | `open-sse/translator/request/openai-responses.js:11-13` | MISSING | Not implemented |
| PAR-TRANS-034 | Responses → Chat skips hosted/nameless tools and ensures `parameters.properties` exists | `open-sse/translator/request/openai-responses.js:156-196` | MISSING | Not implemented |
| PAR-TRANS-035 | Chat → Responses converts messages to `input[]` with `input_text`/`output_text`/`input_image`; tool calls to `function_call`; tool results to `function_call_output` | `open-sse/translator/request/openai-responses.js:201-313` | MISSING | Not implemented |
| PAR-TRANS-036 | Responses response translator emits event-based SSE (`response.created`, `response.in_progress`, `response.output_item.added`, `response.output_text.delta`, `response.reasoning_summary_text.delta`, `response.function_call_arguments.delta`, `response.completed`) | `open-sse/translator/response/openai-responses.js:12-113` | MISSING | g0router `SSEScanner` only understands `data:` lines and `[DONE]` |
| PAR-TRANS-037 | Responses response translator buffers reasoning into `rs_*` items with summary parts | `open-sse/translator/response/openai-responses.js:116-180` | MISSING | No reasoning buffering |
| PAR-TRANS-038 | Responses → Chat response translator handles `response.output_text.delta`, `response.function_call_arguments.delta`, `response.completed`, error events, and synthesizes finish_reason | `open-sse/translator/response/openai-responses.js:370-585` | MISSING | Not implemented |
| PAR-TRANS-039 | OpenAI → Antigravity wraps Gemini/Claude request in Cloud Code envelope with `project`, `requestId`, `requestType`, `userAgent`, `sessionId`, double system prompt injection | `open-sse/translator/request/openai-to-gemini.js:271-317`, `319-447` | MISSING | No Antigravity provider or envelope in g0router |
| PAR-TRANS-040 | Antigravity → OpenAI unwraps Cloud Code envelope, normalizes schema types to lowercase, strips `enumDescriptions`, maps `thought`/`thoughtSignature`/`functionCall`/`functionResponse` | `open-sse/translator/request/antigravity-to-openai.js:5-217` | MISSING | No Antigravity provider |
| PAR-TRANS-041 | Claude stream → OpenAI emits `message_start`, `content_block_start`, `content_block_delta`, `content_block_stop`, `message_delta`, `message_stop` handling; wraps thinking in `<think>`/`</think>` and maps `thinking_delta` to `reasoning_content` | `open-sse/translator/response/claude-to-openai.js:20-191` | MISSING | g0router `ConvertStreamEventToChunk` (`internal/providers/anthropic/converter.go:248-284`) maps `text_delta` to `content` and `message_delta.stop_reason` to finish_reason; ignores thinking blocks and emits no reasoning_content |
| PAR-TRANS-042 | Claude stream maps `stop_reason` (`end_turn→stop`, `max_tokens→length`, `tool_use→tool_calls`) and includes cache token breakdown | `open-sse/translator/response/claude-to-openai.js:193-202`, `106-128` | PARTIAL | g0router maps stop reasons (`internal/providers/anthropic/converter.go:232-245`) but does not surface cache tokens |
| PAR-TRANS-043 | Gemini stream → OpenAI initializes state on first chunk, maps `thoughtSignature`/`thought:true` to `reasoning_content`, regular text to `content`, `functionCall` to `tool_calls`, includes `thoughtsTokenCount` in usage | `open-sse/translator/response/gemini-to-openai.js:5-245` | MISSING | g0router `ConvertStreamChunk` (`internal/providers/gemini/converter.go:321-350`) maps text only; no `thought`/`thoughtSignature` handling; generates synthetic `call_<name>` IDs |
| PAR-TRANS-044 | OpenAI → Claude response translator buffers tool arguments and sanitizes at finish; strips `proxy_` prefix; handles `reasoning_content` as `thinking` block | `open-sse/translator/response/openai-to-claude.js:64-253` | MISSING | g0router only produces OpenAI-shaped output |
| PAR-TRANS-045 | OpenAI → Antigravity response accumulates tool call argument deltas silently and emits complete `functionCall` parts only on `finish_reason` | `open-sse/translator/response/openai-to-antigravity.js:39-71` | MISSING | Not implemented |
| PAR-TRANS-046 | Central stream processor `createSSEStream` supports `TRANSLATE` and `PASSTHROUGH` modes, tracks TTFT, accumulates content/thinking lengths, estimates usage on finish, injects `[DONE]` | `open-sse/utils/stream.js:17-433` | MISSING | g0router `internal/api/chat.go:41-59` writes raw chunks with `[DONE]`; no accumulation or usage estimation |
| PAR-TRANS-047 | SSE parser handles standard `data:` lines, `[DONE]`, NDJSON for Ollama, event-prefixed lines for Claude and Responses, and warns on parse failures | `open-sse/utils/streamHelpers.js:4-34` | PARTIAL | g0router `SSEScanner.Scan` (`internal/providers/utils/sse.go:22-49`) handles `data:` lines and `[DONE]`; ignores `event:` and NDJSON |
| PAR-TRANS-048 | Stream helper `hasValuableContent` filters empty OpenAI/Claude chunks; `fixInvalidId` replaces short/generic IDs; `formatSSE` emits event framing per format | `open-sse/utils/streamHelpers.js:36-122` | MISSING | g0router emits every parsed chunk unconditionally with `data: <json>\n\n` |
| PAR-TRANS-049 | Passthrough mode strips Azure-specific `prompt_filter_results` and `content_filter_results`; injects missing `object`/`created` fields | `open-sse/utils/stream.js:109-127` | MISSING | g0router does not normalize passthrough chunks |
| PAR-TRANS-050 | Stream flush synthesizes `response.failed` for OpenAI Responses streams that never reached a terminal event | `open-sse/utils/stream.js:203-207`, `397-404` | MISSING | Not implemented |
| PAR-TRANS-051 | `injectReasoningContent` injects placeholder `reasoning_content:" "` for DeepSeek/MiniMax/Kimi models when assistant message lacks it | `open-sse/utils/reasoningContentInjector.js:1-79` | MISSING | g0router has no reasoning injection |
| PAR-TRANS-052 | DeepSeek v4 Pro alias expansion: maps `deepseek-v4-pro-max/none` to base model with `extra_body.thinking.type` and `reasoning_effort` adjustments | `open-sse/utils/reasoningContentInjector.js:20-71` | MISSING | Not implemented |
| PAR-TRANS-053 | `dedupeTools` strips built-in web tools when MCP equivalents are present (Exa/Tavily/Browser triggers) | `open-sse/utils/toolDeduper.js:6-47` | MISSING | No tool deduplication in g0router |
| PAR-TRANS-054 | `handleBypassRequest` short-circuits Claude CLI warmup/count/title/skip-pattern requests with fake streaming/non-streaming responses | `open-sse/utils/bypassHandler.js:11-92` | MISSING | Not implemented |
| PAR-TRANS-055 | `prepareClaudeRequest` filters built-in tools for non-Anthropic providers, fixes tool_use/tool_result ordering, removes empty messages, replaces thinking signatures | `open-sse/translator/helpers/claudeHelper.js:81-215` | MISSING | g0router performs none of these cleanups |
| PAR-TRANS-056 | `/v1/messages` route serves Claude-format clients through the shared chat pipeline (endpoint detection → request translation → routing); streaming responses are translated openai→claude per chunk; non-streaming responses return OpenAI-shaped JSON untranslated (`translateNonStreamingResponse` only converts provider→OpenAI: `if (targetFormat === sourceFormat \|\| targetFormat === FORMATS.OPENAI) return responseBody;`) | `src/app/api/v1/messages/route.js:28-34`, `open-sse/handlers/chatCore.js:34,263-277`, `open-sse/handlers/chatCore/nonStreamingHandler.js:15-16` | MISSING | g0router has no `/v1/messages` route (`internal/api/chat.go:23-28` parses only `/v1/chat/completions`) |
| PAR-TRANS-057 | Claude→OpenAI request translator: system string/array → single system message; content blocks (text/image/tool_use/tool_result) → OpenAI messages; local missing-tool-response insertion with `[No response received]`; tools → function schema; tool_choice mapping (`auto/any/tool`) | `open-sse/translator/request/claude-to-openai.js:6-232` (registered line 231) | MISSING | g0router has no claude-source request translation; only provider-direction converters exist |

*Amendment (Fable, 2026-06-09): rows 056-057 added during w1-b planning — the original matrix covered the openai→claude request direction (012-022) and claude→openai response direction (041/042) but omitted the `/v1/messages` route and the claude→openai request direction that make `/v1/messages` functional. Evidence verified against frozen ref source. Recorded in GATE-RESOLUTION.md.*

## Data Models

### 9router Core Translation Types (reference)

`FORMAT` identifiers (`open-sse/translator/formats.js`):
- `openai`, `openai-responses`, `openai-response`, `claude`, `gemini`, `gemini-cli`, `vertex`, `codex`, `antigravity`, `kiro`, `cursor`, `ollama`, `commandcode`

`TranslatorState` (`open-sse/translator/index.js:204-246`):
- `messageId`, `model`, `textBlockStarted`, `thinkingBlockStarted`, `inThinkingBlock`, `currentBlockIndex`, `toolCalls: Map`, `finishReason`, `finishReasonSent`, `usage`, `contentBlockIndex`
- OpenAI-responses extends with: `seq`, `responseId`, `created`, `started`, `msgTextBuf`, `msgItemAdded`, `msgContentAdded`, `msgItemDone`, `reasoningId`, `reasoningIndex`, `reasoningBuf`, `reasoningPartAdded`, `reasoningDone`, `inThinking`, `funcArgsBuf`, `funcNames`, `funcCallIds`, `funcArgsDone`, `funcItemDone`, `completedSent`

OpenAI Chat request fields translated (`open-sse/translator/request/openai-to-claude.js:11-207`, `openai-to-gemini.js:39-221`):
- `model`, `messages`, `max_tokens`, `temperature`, `top_p`, `top_k`, `tools`, `tool_choice`, `thinking`, `reasoning_effort`, `response_format`

Claude request shape (`open-sse/translator/request/openai-to-claude.js:14-207`):
- `model`, `max_tokens`, `stream`, `temperature`, `messages: [{role,content:[]}]`, `system: [{type,text,cache_control}]`, `tools: [{name,description,input_schema,cache_control}]`, `tool_choice`, `thinking`

Gemini request shape (`open-sse/translator/request/openai-to-gemini.js:40-221`):
- `model`, `contents: [{role,parts:[]}]`, `systemInstruction: {role,parts:[]}`, `generationConfig: {temperature,topP,topK,maxOutputTokens}`, `safetySettings`, `tools: [{functionDeclarations:[]}]`, `toolConfig`
- Part variants: `{text}`, `{thought:true,text}`, `{thoughtSignature,text}`, `{functionCall:{id,name,args}}`, `{functionResponse:{id,name,response}}`, `{inlineData:{mime_type,data}}`, `{fileData:{fileUri,mimeType}}`

Responses API request shape (`open-sse/translator/request/openai-responses.js:18-187`):
- `model`, `input: string|array`, `instructions`, `include`, `prompt_cache_key`, `store`, `reasoning`, `tools`, `temperature`, `max_tokens`, `top_p`
- Input item types: `message`, `function_call` (`call_id`, `name`, `arguments`), `function_call_output` (`call_id`, `output`), `reasoning`

### g0router Core Types

`schemas.ChatRequest` (`internal/schemas/chat.go:4-21`):
- `Model`, `Messages`, `Temperature`, `MaxTokens`, `TopP`, `N`, `Stream`, `Stop`, `PresencePenalty`, `FrequencyPenalty`, `LogitBias`, `User`, `Tools`, `ToolChoice`, `ResponseFormat`, `Seed`
- No `thinking`, `reasoning_effort`, `max_completion_tokens`, `modalities`, `prediction`

`schemas.Message` (`internal/schemas/chat.go:34-40`):
- `Role`, `Content`, `Name`, `ToolCalls`, `ToolCallID`
- No `reasoning_content`, `thinking`, `refusal`, `audio`

`schemas.StreamChunk`/`StreamChoice` (`internal/schemas/chat.go:137-153`):
- `ID`, `Object`, `Created`, `Model`, `Choices []StreamChoice`, `Usage`
- `StreamChoice` has `Index`, `Delta Message`, `FinishReason`, `Logprobs`
- `Message` delta has no `reasoning_content` field

Anthropic `MessagesRequest` (`internal/providers/anthropic/converter.go:10-22`):
- `Model`, `MaxTokens`, `System string`, `Messages`, `Tools`, `ToolChoice`, `Temperature`, `TopP`, `TopK`, `StopSequences`, `Stream`
- No `thinking`, cache_control, or multi-block system array

Gemini `GenerateContentRequest` (`internal/providers/gemini/converter.go:10-17`):
- `Contents`, `SystemInstruction`, `GenerationConfig`, `Tools`, `ToolConfig`
- `Part` only has `Text`, `FunctionCall`; no `thought`/`thoughtSignature`/`functionResponse`/`inlineData`/`fileData`

## Edge Cases and Quirks

- Claude tool ID validation rejects IDs outside `^[a-zA-Z0-9_-]+$` and regenerates them deterministically from message index + tool name (`toolCallHelper.js:4-10`). Missing tool IDs generate `call_msg{i}_tc{j}_{name}`.
- Claude `tool_choice` translator rejects native pass-through of malformed types; only `auto/any/tool/none` allowed, and OpenAI `function` name shape is explicitly converted before pass-through (`openai-to-claude.js:298-324`).
- Claude message translator forces tool_result into its own user message immediately after tool_use, and flushes assistant message on tool_use boundaries (`openai-to-claude.js:42-88`).
- Gemini function name sanitizer prefixes with underscore if first character is not a letter or underscore, then truncates to 64 chars (`openai-to-gemini.js:26-36`).
- Gemini schema cleaner mutates the schema in place, flattens `anyOf`/`oneOf` by selecting the highest-scored non-null branch, converts `const` to single-value `enum`, forces `type:string` when `enum` exists, and adds a `reason` placeholder to empty object schemas (`geminiHelper.js:298-371`).
- OpenAI Responses API translator handles Cursor CLI sending a Responses-shaped body to `/v1/chat/completions` by detecting `Array.isArray(body.input)` (`formats.js:30-32`).
- Responses translator normalizes empty `input` array and empty string input to a placeholder user message with `...` to avoid provider rejection (`responsesApiHelper.js:9-21`).
- Responses request clamps `call_id` to 64 chars because upstream enforces that limit (`openai-responses.js:11-13`).
- Responses request filters out hosted tools without a `name` field to avoid nameless `functionDeclarations` being rejected by Gemini (`openai-responses.js:152-176`).
- Responses response translator buffers reasoning text and only emits structured `response.output_item.added` / `response.reasoning_summary_part.added` / `response.reasoning_summary_text.delta` events (`openai-responses.js:60-64`, `116-180`).
- Responses same-format passthrough tracks terminal events and synthesizes `response.failed` if the stream ends without one (`stream.js:195-216`, `397-404`).
- Stream processor in passthrough mode strips Azure `prompt_filter_results` / `content_filter_results` and injects missing `object`/`created` fields for Letta compatibility (`stream.js:109-127`).
- Stream processor always emits `data: [DONE]\n\n` at flush even in passthrough mode because OpenClaw hangs without it (`stream.js:339-345`).
- `injectReasoningContent` applies a model-level rule for `kimi-` only to tool-call scope and provider-level rules for DeepSeek/MiniMax to all assistant messages (`reasoningContentInjector.js:7-38`).
- Antigravity Cloud Code envelope performs double system prompt injection: once normal and once wrapped in `[ignore]...[/ignore]` (`openai-to-gemini.js:294-297`, `424-425`).
- Claude cloaking only triggers for OAuth tokens containing `sk-ant-oat` (`claudeCloaking.js:129`).

## Go-port Considerations

- Introduce a format registry and per-direction converter interface instead of per-provider monolithic converters.
- Add `reasoning_content` and `thinking` fields to `schemas.Message` and `StreamChunk.Delta` before any provider-specific work.
- Implement `/v1/responses` route and bidirectional Responses↔Chat translators; the schema stubs already exist but handlers and converters are missing.
- Port schema sanitizers as pure functions over `map[string]any`; avoid mutating user input by cloning before clean.
- SSE layer needs `event:` prefix parsing and format-specific serialization; current `SSEScanner` discards non-`data:` lines.
- Tool-call ID validation and placeholder insertion should run as request preprocessing before provider routing.
- Cache-control and provider-specific cloaking are transport-layer concerns; keep them out of core schema converters.
