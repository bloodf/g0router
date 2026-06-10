# w1-k — gemini-format client request + independent translation helpers

Scope split 2026-06-10 (per w1-k r1 gate "eight unrelated helpers" finding): this plan keeps the gemini client-request translator and the three SELF-CONTAINED helpers with no claude-cloaking/appConstants dependencies. The three claude-pipeline helpers (cloaking, bypass, prepareClaudeRequest) move to **w1-l**.

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

NEW (none claimed by any other plan — auditable by filename, none match `*gemini_openai_request*`/`strip_content_types*`/`tool_deduper*`/`reasoning_injector*`): `gemini_openai_request.go`+`_test.go`, `strip_content_types.go`+`_test.go`, `tool_deduper.go`+`_test.go`, `reasoning_injector.go`+`_test.go`.
TOUCH-ONLY: `registry.go` (1 Register for row 066), `registry_test.go`.
Dispatch-order gate (shared registry.go): dispatch only after w1-h/i/j registry hunks merged — `grep -c 'FormatOllama\|FormatKiro\|FormatCursor' internal/translation/registry.go` ≥ 3, else IMPL-BLOCKED with no registry edit.

## Tasks (each: STEP (a) write named failing tests; STEP (b) port)

1. **Row 066: gemini→openai client request** (`gemini_openai_request.go`), port `gemini-to-openai.js:6-146`. Signature `(model, body, stream, credentials)` (credentials ignored — ref is 3-arg). generationConfig → `max_tokens` via `AdjustMaxTokens(map{"max_tokens":maxOutputTokens, "tools":body["tools"]})`, temperature, top_p (:14-26); systemInstruction → system message via `extractGeminiText` (string passthrough or parts text join, :136-142); contents → messages via `convertGeminiContent` (:72-133): text→`{type:text}`, inlineData→`{type:image_url, image_url:{url:data:<mime>;base64,<data>}}`, functionCall→assistant tool_calls (id `call_<unixms>_<idx>` — deterministic per-part index, NO Math.random), functionResponse→EARLY-RETURN `{role:tool, tool_call_id: id||name, content: JSON(response.result||response||{})}` (:107-113), assistant-with-tool_calls keeps text as content (single→string), single-text content collapse (:116-130).
   Tests: `TestGeminiOpenAIGenerationConfig`, `TestGeminiOpenAISystemInstruction`, `TestGeminiOpenAIContentTextAndImage`, `TestGeminiOpenAIFunctionCall`, `TestGeminiOpenAIFunctionResponse` (id then name fallback), `TestGeminiOpenAITools`, `TestGeminiOpenAISingleTextCollapse`.

2. **Row 006: StripContentTypes** (`strip_content_types.go`), port `index.js:58-72`: `StripContentTypes(body map[string]any, stripList []string)` mutating in place; image set {image_url,image} gated on "image" in list, audio set {audio_url,input_audio} on "audio"; filter content-array blocks, empty result → set content to `""`. No-op when list empty or message content not an array.
   Tests: `TestStripContentTypesImages`, `TestStripContentTypesAudio`, `TestStripContentTypesEmptyToString`, `TestStripContentTypesNoOp`.

3. **Row 053: DedupeTools** (`tool_deduper.go`), port `toolDeduper.js:1-49`: three DEDUP_RULES — Exa triggers (`mcp__exa__web_search_exa`/`web_fetch_exa`)→strip `WebSearch`/`WebFetch`/`mcp__workspace__web_fetch`; Tavily triggers→same strip; browsermcp regex `^mcp__browsermcp__`→strip `^mcp__Claude_in_Chrome__`. `getToolName` (name||function.name), `matches` (string-eq or regexp), `DedupeTools(tools []any) (out []any, stripped []string)`. Package-scope compiled regexps acceptable (read-only constants).
   Tests: `TestDedupeExaStripsBuiltins`, `TestDedupeBrowserRegex`, `TestDedupeNoTriggerNoChange`, `TestDedupeEmpty`.

4. **Rows 051+052: InjectReasoningContent + deepseek alias** (`reasoning_injector.go`), port `reasoningContentInjector.js:1-79`: PROVIDER_RULES (deepseek/minimax/minimax-cn → scope "all"), MODEL_RULES (kimi-* → "toolCalls", deepseek → "all"), placeholder " "; `shouldInject` (assistant + empty reasoning_content + scope gate); `applyDeepSeekV4ProAlias` (model `deepseek-v4-pro-max`/`-none` AND provider deepseek → model→`deepseek-v4-pro`, set `extra_body.thinking.type`, set/delete `reasoning_effort`); `InjectReasoningContent(provider, model string, body map[string]any) map[string]any` = alias then rule.
   Tests: `TestInjectProviderRuleDeepseek`, `TestInjectModelRuleKimiToolCalls`, `TestInjectSkipsNonEmptyReasoning`, `TestDeepSeekV4ProMaxAlias`, `TestDeepSeekV4ProNoneAliasEffortDeleted`, `TestInjectNoRuleNoChange`.

5. **Registration (row 066)** — parity basis PAR-TRANS-001. Add the request translator to the existing `FormatGemini→FormatOpenAI` and `FormatGeminiCLI→FormatOpenAI` pairs (response side already wired in w1-e): change those two `Register` calls to pass `geminiToOpenAIRequest` as the request arg (ref `gemini-to-openai.js:145-146`). StripContentTypes/DedupeTools/InjectReasoningContent are exported pipeline helpers (consumers arrive Wave 2/4 — `index.js:74+ translateRequest`, `chatCore.js`), NOT registry translators.
   Tests: `TestRegistryWiresGeminiClientRequest` (`RequestTranslatorFor(FormatGemini, FormatOpenAI)` non-nil + reflect identity; same for GeminiCLI).

## Binary acceptance criteria

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `grep -rn 'func init(\|panic(' internal/translation/gemini_openai_request.go internal/translation/strip_content_types.go internal/translation/tool_deduper.go internal/translation/reasoning_injector.go` → 0 hits.
- `grep -c 'RequestTranslatorFor(FormatGemini, FormatOpenAI)' internal/translation/registry_test.go` ≥ 1.
- Each of the 4 ported units has ≥1 direct test file (above).

## Out of scope

Claude cloaking / bypass / prepareClaudeRequest → **w1-l**. Pipeline wiring of the exported helpers into routing/executors (Wave 2/4). PAR-TRANS-057 (already HAVE).
