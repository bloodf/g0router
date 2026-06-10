# w1-i — kiro translator pair (+ kiro constants/helpers)

Rows: requires matrix rows for kiro translator behavior — **added 2026-06-10 as PAR-TRANS-062 (request) and PAR-TRANS-063 (response)** following the w1-h precedent (granular rows for the WAVE-MAP Wave-1 "12 wire formats" mandate; cross-ref PAR-PROV-022). Registration parity basis: PAR-TRANS-001.

Frozen ref (@ 827e5c3), read whole before porting:
- `open-sse/translator/request/openai-to-kiro.js` (1-584)
- `open-sse/translator/response/kiro-to-openai.js` (1-196)
- `open-sse/config/kiroConstants.js` (1-263)
- NOT ported: `openai-to-kiro.old.js` (dead file, superseded).

## Preconditions (a "0 hits" grep exits 1 — that IS the pass)

- `grep -n 'FormatKiro' internal/translation/formats.go` → present
- `grep -rn 'buildKiroPayload\|kiroToOpenAIResponse\|resolveKiroModel' internal/translation/` → 0 hits

## Exclusive file ownership

NEW: `internal/translation/kiro_constants.go` + `_test.go`, `openai_kiro_request.go` + `_test.go`, `kiro_openai_response.go` + `_test.go`.
TOUCH-ONLY: `registry.go` (2 Register calls + StreamState fields below), `registry_test.go` (wiring tests).

## Tasks (TDD: named failing tests FIRST per task)

1. **Constants + helpers** (`kiro_constants.go`). STEP (a) — write ALL the task's named tests below and run them (they must fail to compile/pass). STEP (b) — port `kiroConstants.js:1-263`:
   - Constants: `kiroAgenticSuffix "-agentic"`, `kiroThinkingSuffix "-thinking"`, `kiroThinkingBudgetDefault 16000`, `kiroAgenticSystemPrompt` (byte-exact multi-line string `:23-71`, `.trim()`-ed as in ref).
   - `isAgenticModel`/`stripAgenticSuffix` (:140-153), `isThinkingModel`/`stripThinkingSuffix` (:167-180), `ResolveKiroModel` (exported — the Wave-2 kiro executor in `internal/providers/kiro` is a cross-package consumer) (:198-211 — agentic stripped FIRST, then thinking; returns upstream+flags exactly per the four doc examples :186-193).
   - `isThinkingEnabled(body map[string]any, headers map[string]string, model string) bool` (:84-130): anthropic-beta header contains "interleaved-thinking" (case-insensitive key lookup, :224-236); `thinking.type=="enabled"` with budget non-finite-or-positive (:100-106); `reasoning_effort` or `reasoning.effort` in {low,medium,high,auto}, "none" false (:108-115); `<thinking_mode>enabled|interleaved</thinking_mode>` tag in system/user message text or body.system string (:238-262); model name contains "thinking" or "-reason" (:122-127).
   - `buildThinkingSystemPrefix(budget int) string` (:219-222): clamp [1,32000], default 16000, exact format `<thinking_mode>enabled</thinking_mode>\n<max_thinking_length>%d</max_thinking_length>`.
   Tests: `TestResolveKiroModel` (all four suffix combos, exported symbol), `TestIsThinkingEnabled` (each trigger + "none" negative + header), `TestBuildThinkingSystemPrefix` (default/clamp low/clamp high), `TestKiroAgenticPromptByteExact` — `sha256(kiroAgenticSystemPrompt) == "df38d752b7913306e1d8885a32134e9ce214cb5b7303c979852a23e5e6080f6a"` and `len == 1864` (hash of the ref's `.trim()`-ed template literal, derivation command in the comment: `python3 -c "import hashlib,re; s=open('<ref>/open-sse/config/kiroConstants.js').read(); import re; m=re.search(r'KIRO_AGENTIC_SYSTEM_PROMPT = \x60\\n(.*?)\x60.trim', s, re.S); print(hashlib.sha256(m.group(1).strip().encode()).hexdigest())"`).

2. **Row 062: openai→kiro request** (`openai_kiro_request.go`). STEP (a) — write the task's named tests below; see them fail. STEP (b) — port `openai-to-kiro.js:16-581`:
   - Text renderers `toolCallToText` (:16-24, marshal fallback "{}"), `toolResultToText` (:27-32).
   - `flattenToolInteractions` (:52-97) — ONLY when client sent no tools; tool msgs → user text lines, assistant tool_use/tool_calls → text lines, user tool_result blocks → text blocks.
   - `convertMessages` (:171-492): system/tool→user role normalization (:268-270); role-change flush with pending user content join "\n\n" default "continue" (:194), assistant join default "..." (:252); images — data-URI → `{format, source:{bytes}}` (format = media-type subtype, :291-295), http(s) URL → text `[Image: url]` (:296-299), claude base64 source blocks (:300-307); tool_result blocks → `{toolUseId, status:"success", content:[{text}]}` (:312-325, 329-335); assistant toolUses attach to LAST flushed assistant msg with `safeJSONParse(arguments, {})` and uuid fallback ids (:354-387, currentRole reset to null :386); tools injected into first user message then moved: pop last userInputMessage as currentMessage (:397-402), capture first-history tools BEFORE cleanup (:405), cleanup deletes history tools/empty contexts and sets modelId (:408-419), consecutive-user merge combining content "\n\n" + toolResults/tools contexts (:424-450), synthesize empty currentMessage when none (:455-462), `reconcileOrphanedToolResults` ONLY on tools-present path (:114-155, 474-476 — orphans folded into user text, context deleted when empty), tools re-injected into currentMessage (:481-489). Tool spec mapping: name/description (default `Tool: <name>`), schema default `{type:object,properties:{},required:[]}` else `required` ensured (:221-242).
   - `buildKiroPayload` (:511-581): maxTokens hardcoded 32000 (:514); `ResolveKiroModel` + `isThinkingEnabled(body, nil, model)` (:518-519 — headers are nil at this call site); profileArn from `credentials.providerSpecificData.profileArn` (:523); prefix order thinking-tag → `[Context: Current time is <RFC3339>]` → agentic prompt, joined "\n\n" then prepended (:532-540); envelope `conversationState{chatTriggerType:"MANUAL", conversationId:uuid, currentMessage{userInputMessage{content, modelId:upstream, origin:"AI_EDITOR", images?, userInputMessageContext?}}, history}` (:542-561); profileArn top-level when non-empty (:563-565); `inferenceConfig{maxTokens, temperature?, topP?}` (:567-572); the ref's non-enumerable `_kiroUpstreamModel` tag (:575-578) is NOT emitted: non-enumerable in JS means it never serializes, so adding any key would change wire bytes vs the ref. The Go executor (Wave 2) re-derives the upstream id via the exported `ResolveKiroModel(model)` — document with a comment citing :575-578.
   Tests: `TestKiroFlattenWhenNoTools`, `TestKiroToolSpecInjectionAndMove` (first-user inject → cleanup → currentMessage carry), `TestKiroOrphanToolResultSalvage` (dangling toolUseId folded to text, kept ids preserved), `TestKiroConsecutiveUserMerge` (content + toolResults merged), `TestKiroImagesDataURIAndHTTP`, `TestKiroAssistantToolUses` (uuid fallback, malformed args → {}), `TestKiroPayloadEnvelope` (MANUAL/AI_EDITOR/32000/upstream modelId; asserts NO `_kiroUpstreamModel` key — serialized payload byte-parity with the ref), `TestKiroThinkingPrefixOrder` (tag before context line before agentic prompt), `TestKiroEmptyCurrentMessageSynthesized`.

3. **Row 063: kiro→openai response** (`kiro_openai_response.go`). STEP (a) — write the task's named tests below; see them fail. STEP (b) — port `kiro-to-openai.js:12-192` as ResponseTranslator over parsed maps:
   - Pass through chunks already shaped `object=="chat.completion.chunk"` with choices (:17-19). The raw-SSE-string parsing branch (:23-53) is NOT ported — g0router's scanner/executor delivers parsed maps carrying `_eventType` or wrapped event keys; comment citing :23-53.
   - State init `chatcmpl-<unix-ms>` + created + chunkIndex (:56-60); event resolution `data._eventType || data.event` plus wrapped-key detection (:62-65 dual paths).
   - `assistantResponseEvent` → content delta, role injected on first chunk, empty → nil (:65-86); `reasoningContentEvent` → `reasoning_content` delta with string/text/content fallbacks (:93-117); `toolUseEvent` → single tool_calls delta `{index:0, id:toolUseId||call_<ms>, function:{name, arguments:JSON-stringified input}}` (:120-151); `messageStopEvent`/`done` → finish chunk `stop` + state usage when present (:154-175); `usageEvent` → buffer usage from inputTokens/outputTokens, emit nil (:178-188); unknown events → nil (:191).
   - StreamState additions: `KiroID string`, `KiroCreated int64`, `KiroModel string`, `KiroChunkIndex int`, `KiroFinishReason string`, `KiroUsage map[string]any`.
   Tests: `TestKiroOpenAIPassthroughChunk`, `TestKiroOpenAIContentAndRoleInjection` (both `_eventType` and wrapped-key forms), `TestKiroOpenAIReasoningDelta`, `TestKiroOpenAIToolUseEvent`, `TestKiroOpenAIUsageThenStop` (usage buffered, attached on stop chunk), `TestKiroOpenAIUnknownEventNil`.

4. **Registration**. STEP (a) — write `TestNewRegistryWiresKiroPair`; see it fail. STEP (b) — parity basis PAR-TRANS-001. Two `Register` calls in `NewRegistry`:
   - `Register(FormatOpenAI, FormatKiro, buildKiroPayload, nil)` — ref `openai-to-kiro.js:583`
   - `Register(FormatKiro, FormatOpenAI, nil, kiroToOpenAIResponse)` — ref `kiro-to-openai.js:195`
   Tests: `TestNewRegistryWiresKiroPair` (presence + reflect identity).

## Binary acceptance criteria

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `grep -rn 'func init(' internal/translation/kiro_*.go internal/translation/openai_kiro_*.go` → 0 hits; same for `panic(`.
- `TestKiroPayloadEnvelope` asserts the payload map has NO `_kiroUpstreamModel` key (covered by go test; the source comment citing :575-578 may name the identifier — no grep gate on the source text).
- `TestKiroAgenticPromptByteExact` passes (sha256 + length pin; covered by go test).
- `TestKiroOrphanToolResultSalvage` proves orphaned content survives as text (covered by go test).

## Out of scope

Kiro executor/auth/event-stream framing (Wave 2/3 — PAR-PROV-022). The raw-SSE parsing branch (:23-53). cursor (w1-j). `openai-to-kiro.old.js`.
