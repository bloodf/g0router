# w1-k — gemini-format client request + independent translation helpers

Scope: gemini client-request translator (PAR-TRANS-066) plus three self-contained helpers (PAR-TRANS-006/053/051+052) that share no dependencies. The claude-cloaking/bypass/prepare helpers (PAR-TRANS-022/054/055) are a separate cohesive unit in **w1-l** (split recorded in WAVE-MAP.md "Wave 1 remainder slicing").

Rows: PAR-TRANS-066 (gemini→openai client request), PAR-TRANS-006 (stripContentTypes), PAR-TRANS-053 (dedupeTools), PAR-TRANS-051 + PAR-TRANS-052 (injectReasoningContent + deepseek-v4-pro alias). Registration parity basis: PAR-TRANS-001.

Frozen ref (@ 827e5c3), read whole:
- `open-sse/translator/request/gemini-to-openai.js:6-146` (row 066)
- `open-sse/translator/index.js:58-72` (`stripContentTypes`, row 006)
- `open-sse/utils/toolDeduper.js:1-49` (row 053)
- `open-sse/utils/reasoningContentInjector.js:1-79` (rows 051+052)

## Preconditions (a "0 hits" grep exits 1 — that IS the pass)

- `grep -n 'FormatGemini ' internal/translation/formats.go` → present (verified 2026-06-10: `formats.go:14` `FormatGemini Format = "gemini"`)
- `grep -n 'AdjustMaxTokens' internal/translation/maxtokens.go` → present (`maxtokens.go:12`, w1-c — consumed by row 066)
- `grep -n 'geminiToOpenAIResponse' internal/translation/registry.go` → present (w1-e — row 066 adds the REQUEST direction to the same `FormatGemini→FormatOpenAI` pair)
- `grep -rn 'geminiToOpenAIRequest\|StripContentTypes\|DedupeTools\|InjectReasoningContent' internal/translation/` → 0 hits (re-run before impl; any hit → IMPL-BLOCKED for that symbol)

## Exclusive file ownership

NEW: `gemini_openai_request.go`+`_test.go`, `strip_content_types.go`+`_test.go`, `tool_deduper.go`+`_test.go`, `reasoning_injector.go`+`_test.go`.
Non-overlap is filename-auditable against every other plan's "## Exclusive file ownership" section: w1-g lists `responses_*`/`openai_responses_*`; w1-h `*ollama*`/`*commandcode*`; w1-i `kiro_*`/`openai_kiro_*`; w1-j `*cursor*`; w1-l `claude_*`/`bypass_*`. None matches this plan's NEW filenames.
TOUCH-ONLY: `registry.go` (1 Register for row 066), `registry_test.go`.
Dispatch-order gate (shared registry.go): dispatch only after w1-h/i/j registry hunks merged — `grep -c 'FormatOllama\|FormatKiro\|FormatCursor' internal/translation/registry.go` ≥ 3, else IMPL-BLOCKED with no registry edit.

## Tasks (each: STEP (a) write named failing tests; STEP (b) port)

1. **Row 066: gemini→openai client request** (`gemini_openai_request.go`), port `gemini-to-openai.js:6-146`. Signature `(model, body, stream, credentials)` (credentials ignored — ref is 3-arg). generationConfig → `max_tokens` via `AdjustMaxTokens(map{"max_tokens":maxOutputTokens, "tools":body["tools"]})`, temperature, top_p (:14-26); systemInstruction → system message via `extractGeminiText` (string passthrough or parts text join, :136-142); contents → messages via `convertGeminiContent` (:72-133): text→`{type:text}`, inlineData→`{type:image_url, image_url:{url:data:<mime>;base64,<data>}}`, functionCall→assistant tool_calls (the ref id `call_${Date.now()}_${random}` at :98 is OPAQUE and non-deterministic — parity is "a unique id string is present", NOT exact bytes; Go uses `call_<unixms>_<part-index>`; tests assert id format+uniqueness, never an exact value), functionResponse→EARLY-RETURN `{role:tool, tool_call_id: id||name, content: JSON(response.result||response||{})}` (:107-113), assistant-with-tool_calls keeps text as content (single→string), single-text content collapse (:116-130).
   Tests: `TestGeminiOpenAIGenerationConfig`, `TestGeminiOpenAISystemInstruction`, `TestGeminiOpenAIContentTextAndImage`, `TestGeminiOpenAIFunctionCall`, `TestGeminiOpenAIFunctionResponse` (id then name fallback), `TestGeminiOpenAITools`, `TestGeminiOpenAISingleTextCollapse`.

2. **Row 006: StripContentTypes** (`strip_content_types.go`), port `index.js:58-72`: `StripContentTypes(body map[string]any, stripList []string)` mutating in place; image set {image_url,image} gated on "image" in list, audio set {audio_url,input_audio} on "audio"; filter content-array blocks, empty result → set content to `""`. No-op when list empty or message content not an array.
   Tests: `TestStripContentTypesImages`, `TestStripContentTypesAudio`, `TestStripContentTypesEmptyToString`, `TestStripContentTypesNoOp`.

3. **Row 053: DedupeTools** (`tool_deduper.go`), port `toolDeduper.js:1-49`: three DEDUP_RULES — Exa triggers (`mcp__exa__web_search_exa`/`web_fetch_exa`)→strip `WebSearch`/`WebFetch`/`mcp__workspace__web_fetch`; Tavily triggers→same strip; browsermcp regex `^mcp__browsermcp__`→strip `^mcp__Claude_in_Chrome__`. `getToolName` (name||function.name), `matches` (string-eq or regexp), `DedupeTools(tools []any) (out []any, stripped []string)`. Package-scope compiled regexps acceptable (read-only constants).
   Tests: `TestDedupeExaStripsBuiltins`, `TestDedupeBrowserRegex`, `TestDedupeNoTriggerNoChange`, `TestDedupeEmpty`.

4. **Rows 051+052: InjectReasoningContent + deepseek alias** (`reasoning_injector.go`), port `reasoningContentInjector.js:1-79`: PROVIDER_RULES (deepseek/minimax/minimax-cn → scope "all"), MODEL_RULES (kimi-* → "toolCalls", deepseek → "all"), placeholder " "; `shouldInject` (assistant + empty reasoning_content + scope gate); `applyDeepSeekV4ProAlias` (model `deepseek-v4-pro-max`/`-none` AND provider deepseek → model→`deepseek-v4-pro`, set `extra_body.thinking.type`, set/delete `reasoning_effort`); `InjectReasoningContent(provider, model string, body map[string]any) map[string]any` = alias then rule.
   Tests: `TestInjectProviderRuleDeepseek`, `TestInjectModelRuleKimiToolCalls`, `TestInjectSkipsNonEmptyReasoning`, `TestDeepSeekV4ProMaxAlias`, `TestDeepSeekV4ProNoneAliasEffortDeleted`, `TestInjectNoRuleNoChange`.

5. **Registration (row 066)** — parity basis PAR-TRANS-001. Add the request translator to the existing `FormatGemini→FormatOpenAI` and `FormatGeminiCLI→FormatOpenAI` pairs (response side already wired in w1-e): change those two `Register` calls to pass `geminiToOpenAIRequest` as the request arg (ref `gemini-to-openai.js:145-146`). Registering both pairs is verbatim ref parity (`gemini-to-openai.js:145` registers `GEMINI→OPENAI`, `:146` registers `GEMINI_CLI→OPENAI` — same translator fn). StripContentTypes/DedupeTools/InjectReasoningContent are exported helpers invoked by the ref's `translateRequest` pipeline (`index.js` `translateRequest` calls them); they are NOT registry translators, so they ship as exported functions with direct unit tests — integration into g0router's request path is Wave-2 routing scope (tracked there), exactly as 9router keeps them as pipeline utilities rather than registered translators.
   Tests: `TestRegistryWiresGeminiClientRequest` (`RequestTranslatorFor(FormatGemini, FormatOpenAI)` non-nil + reflect identity; same for GeminiCLI).

## Binary acceptance criteria

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `grep -rn 'func init(\|panic(' internal/translation/gemini_openai_request.go internal/translation/strip_content_types.go internal/translation/tool_deduper.go internal/translation/reasoning_injector.go` → 0 hits.
- `grep -c 'RequestTranslatorFor(FormatGemini, FormatOpenAI)' internal/translation/registry_test.go` ≥ 1.
- Every named test in Tasks 1-4 exists and passes (the specific behavioral tests, not just non-empty files) — verified by `go test ./internal/translation/ -run 'TestGeminiOpenAI|TestStripContentTypes|TestDedupe|TestInject|TestDeepSeek'` exiting 0 with >0 tests run.

## Out of scope

Claude cloaking / bypass / prepareClaudeRequest → **w1-l**. Pipeline wiring of the exported helpers into routing/executors (Wave 2/4). PAR-TRANS-057 (already HAVE).
