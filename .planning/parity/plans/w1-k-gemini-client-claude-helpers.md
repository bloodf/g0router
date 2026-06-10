# w1-k — gemini-format client requests + claude pipeline helpers

Rows: PAR-TRANS-006 (stripContentTypes), PAR-TRANS-022 (claude cloaking), PAR-TRANS-051 (injectReasoningContent placeholder), PAR-TRANS-052 (deepseek-v4-pro alias), PAR-TRANS-053 (dedupeTools), PAR-TRANS-054 (handleBypassRequest), PAR-TRANS-055 (prepareClaudeRequest), PAR-TRANS-066 (gemini→openai client request). PAR-TRANS-057 (claude→openai request) is already HAVE (w1-b) — this plan VERIFIES it via a precondition grep, no re-implementation.

Frozen ref (@ 827e5c3), read whole before porting each:
- `open-sse/translator/request/gemini-to-openai.js:6-146` (row 066)
- `open-sse/translator/index.js:58-72` (`stripContentTypes`, row 006)
- `open-sse/utils/toolDeduper.js:1-49` (row 053)
- `open-sse/utils/reasoningContentInjector.js:1-79` (rows 051 + 052)
- `open-sse/utils/claudeCloaking.js:1-155` (row 022)
- `open-sse/utils/bypassHandler.js:11-92` + support `:128-195` (row 054)
- `open-sse/translator/helpers/claudeHelper.js:81-215` + support `hasValidContent:8`, `fixToolUseOrdering:23` (row 055)

## Preconditions (a "0 hits" grep exits 1 — that IS the pass)

- `grep -n 'FormatGemini\b' internal/translation/formats.go` → present (verified 2026-06-10: `formats.go:14`)
- `grep -rn 'geminiToOpenAIRequest\|stripContentTypes\|dedupeTools\|injectReasoningContent\|cloakClaudeTools\|handleBypassRequest\|prepareClaudeRequest' internal/translation/` → 0 hits (re-run before impl; any hit → IMPL-BLOCKED for that symbol)
- PAR-TRANS-057 already merged: `grep -n 'func claudeToOpenAIRequest' internal/translation/` → present (w1-b); if absent, IMPL-BLOCKED
- `grep -n 'AdjustMaxTokens' internal/translation/maxtokens.go` → present (w1-c, consumed by row 066)

## Exclusive file ownership

NEW: `gemini_openai_request.go`+`_test.go`, `strip_content_types.go`+`_test.go`, `tool_deduper.go`+`_test.go`, `reasoning_injector.go`+`_test.go`, `claude_cloaking.go`+`_test.go`, `bypass_handler.go`+`_test.go`, `claude_prepare.go`+`_test.go` (all `internal/translation/`).
TOUCH-ONLY: `registry.go` (1 Register for row 066 + StreamState if needed), `registry_test.go`.
Ownership non-overlap (filename-auditable): no other Wave-1 plan claims `*gemini_openai_request*`/`strip_*`/`tool_deduper*`/`reasoning_injector*`/`claude_cloaking*`/`bypass_*`/`claude_prepare*`. Dispatch-order gate: dispatch after w1-h, w1-i, w1-j registry hunks are merged (`grep -c 'FormatOllama\|FormatKiro\|FormatCursor' internal/translation/registry.go` ≥ 3) → else IMPL-BLOCKED, no registry edit.

## Tasks (each: STEP (a) write named failing tests; STEP (b) port)

1. **Row 066: gemini→openai client request** (`gemini_openai_request.go`), port `gemini-to-openai.js:6-146`. Signature `(model, body, stream, credentials)` (credentials ignored). generationConfig → `max_tokens` via `AdjustMaxTokens({max_tokens:maxOutputTokens, tools})`, temperature, top_p (:14-26); systemInstruction → system message via `extractGeminiText` (parts text join, :136-142); contents → messages via `convertGeminiContent` (:72-133): text parts, inlineData → `image_url` data-URI, functionCall → assistant tool_calls with `call_<ms>_<rand6>` id (vary by index in Go — no Math.random; use a deterministic counter+state or uuid), functionResponse → `{role:tool, tool_call_id: id||name, content: JSON(result||response||{})}` (early return), assistant tool_calls path keeps text as content when present, single-text collapse rule (:116-130); functionDeclarations → tools (:50-66).
   Tests: `TestGeminiOpenAIGenerationConfig`, `TestGeminiOpenAISystemInstruction`, `TestGeminiOpenAIContentTextAndImage`, `TestGeminiOpenAIFunctionCall` (assistant tool_calls), `TestGeminiOpenAIFunctionResponse` (tool role, id/name fallback), `TestGeminiOpenAITools`, `TestGeminiOpenAISingleTextCollapse`.

2. **Row 006: stripContentTypes** (`strip_content_types.go`), port `index.js:58-72`: `StripContentTypes(body map[string]any, stripList []string)` mutating in place; image set {image_url,image} gated on "image" in list, audio set {audio_url,input_audio} on "audio"; filter content blocks, empty array → `""`. No-op when list empty or no array messages.
   Tests: `TestStripContentTypesImages`, `TestStripContentTypesAudio`, `TestStripContentTypesEmptyToString`, `TestStripContentTypesNoOp`.

3. **Row 053: dedupeTools** (`tool_deduper.go`), port `toolDeduper.js:1-49`: the three DEDUP_RULES (Exa/Tavily string triggers→strip WebSearch/WebFetch/mcp__workspace__web_fetch; browsermcp regex trigger→strip Claude_in_Chrome regex). `getToolName` (name or function.name), `matches` (string eq or regexp), `DedupeTools(tools []any) (out []any, stripped []string)`. Regexes compiled once at package scope are acceptable here (read-only constants, not mutable state) — OR compile locally; document choice.
   Tests: `TestDedupeExaStripsBuiltins`, `TestDedupeBrowserRegex`, `TestDedupeNoTriggerNoChange`, `TestDedupeEmpty`.

4. **Rows 051+052: reasoning injection + deepseek alias** (`reasoning_injector.go`), port `reasoningContentInjector.js:1-79`: PROVIDER_RULES (deepseek/minimax/minimax-cn scope "all"), MODEL_RULES (kimi-* → toolCalls, deepseek → all), placeholder " "; `shouldInject` (assistant, empty reasoning_content, scope gate); `applyDeepSeekV4ProAlias` (model in {deepseek-v4-pro-max/none} AND provider deepseek → rewrite model to base, set extra_body.thinking.type, reasoning_effort set or deleted); `InjectReasoningContent(provider, model string, body map[string]any) map[string]any` applies alias then rule.
   Tests: `TestInjectProviderRule` (deepseek all), `TestInjectModelRuleKimiToolCalls`, `TestInjectSkipsNonEmptyReasoning`, `TestDeepSeekV4ProMaxAlias`, `TestDeepSeekV4ProNoneAlias` (reasoning_effort deleted), `TestInjectNoRuleNoChange`.

5. **Row 022: claude cloaking** (`claude_cloaking.go`), port `claudeCloaking.js:1-155` EXCEPT the decoy-tool/cloakClaudeTools surface tied to `CC_DEFAULT_TOOLS`/`CLAUDE_TOOL_SUFFIX` from appConstants — port `applyCloaking` (:128-155, the row-022-cited billing-header + fake-user-id path) and its helpers `generateBillingHeader` (:9-14, sha256 first-5 of payload JSON + 3-hex build), `generateFakeUserID` (:18-23, 64-hex device + 2 uuids). `cloakClaudeTools`/`decloakToolNames` (:34-92) need `CC_DECOY_TOOLS` + suffix const — port those too IF `appConstants` values are available; if `CLAUDE_TOOL_SUFFIX`/`CC_DEFAULT_TOOLS` are not yet in g0router, port only `applyCloaking` (row 022's cited :128-155) and mark cloakClaudeTools IMPL-BLOCKED-pending-appConstants in the report. Cloaking only applies when apiKey contains `sk-ant-oat` (:129). Randomness: use crypto/rand, wrap errors, no panic.
   Tests: `TestApplyCloakingOAuthOnly` (non-oat → unchanged), `TestApplyCloakingBillingHeaderFirstSystemBlock` (string/array/absent system shapes), `TestApplyCloakingFakeUserId` (metadata.user_id injected, existing preserved), `TestGenerateBillingHeaderFormat` (prefix + cch length 5).

6. **Row 054: bypass handler** (`bypass_handler.go`), port `bypassHandler.js:11-92` + `createStreamingResponse`/`createNonStreamingResponse` (:128-195) + the `SKIP_PATTERNS` constant. `HandleBypassRequest(body map[string]any, model, userAgent string, ccFilterNaming bool) ([]map[string]any, bool)` — returns (response-or-nil, handled): nil unless userAgent contains "claude-cli"; five bypass patterns (title `{`, Warmup, count, skip-patterns, isNewTopic naming); naming bypass synthesizes `{isNewTopic, title}` from first 3 words. Uses `detectFormat(body)` — reuse the existing `DetectFormatByEndpoint`/body-hint detector (cite formats.go) or the body-shape detector; document which.
   Tests: `TestBypassNonClaudeCli` (nil), `TestBypassTitleExtraction`, `TestBypassWarmup`, `TestBypassCount`, `TestBypassSkipPatterns`, `TestBypassNamingIsNewTopic` (title from 3 words), `TestBypassNoMatch` (nil).

7. **Row 055: prepareClaudeRequest** (`claude_prepare.go`), port `claudeHelper.js:81-215` + `hasValidContent` (:8) + `fixToolUseOrdering` (:23): `PrepareClaudeRequest(body map[string]any, provider, apiKey, connectionId string) map[string]any`. output_config delete for minimax/minimax-cn; system cache_control reset to last block ttl 1h; messages pass 1 (strip cache_control, filter empty keep final assistant), pass 1.5 fixToolUseOrdering, pass 2 reverse (last-assistant cache_control on last non-thinking block; thinking signature replacement + thinking-block injection when enabled+tool_use+no-thinking for claude/anthropic-compatible providers); tools filter built-ins for non-claude, last-tool cache_control ttl 1h, drop empty tools+tool_choice; applyCloaking for claude/anthropic-compatible + apiKey (deriveSessionId from connectionId — reuse the w1-f `deriveSessionId` in cloud_code.go, cite it). DEFAULT_THINKING_CLAUDE_SIGNATURE from the ref signature config (port as constant, cite source).
   Tests: `TestPrepareSystemCacheControl`, `TestPrepareFiltersEmptyKeepsFinalAssistant`, `TestPrepareToolUseOrdering`, `TestPrepareLastAssistantCacheControl`, `TestPrepareThinkingInjection` (claude provider, tool_use, no thinking → injected), `TestPrepareToolsBuiltinFilterNonClaude`, `TestPrepareMinimaxOutputConfigDropped`, `TestPrepareCloakingAppliedForOAuth`.

8. **Registration (row 066 only)** — parity basis PAR-TRANS-001. `Register(FormatGemini, FormatOpenAI, geminiToOpenAIRequest, geminiToOpenAIResponse)` — the response translator is already wired (w1-e); this adds the REQUEST translator to the same pair (ref `gemini-to-openai.js:145`) plus `Register(FormatGeminiCLI, FormatOpenAI, geminiToOpenAIRequest, <existing resp>)` (ref `:146`). The other helpers (stripContentTypes, dedupeTools, injectReasoningContent, cloaking, bypass, prepareClaudeRequest) are pipeline utilities invoked by Wave-2/4 routing, NOT registry translators — they ship as exported functions consumed later; document each as "exported helper, consumer arrives in Wave 2/4".
   Tests: `TestRegistryWiresGeminiClientRequest` (RequestTranslatorFor(FormatGemini, FormatOpenAI) non-nil + identity).

## Binary acceptance criteria

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `grep -rn 'func init(\|panic(' <all new files>` → 0 hits.
- `grep -c 'sk-ant-oat' internal/translation/claude_cloaking.go` ≥ 1.
- `grep -c 'RequestTranslatorFor(FormatGemini, FormatOpenAI)' internal/translation/registry_test.go` ≥ 1.
- Each ported helper has ≥1 direct test (filenames above).

## Out of scope

Pipeline wiring of the helpers into routing/executors (Wave 2/4). `cloakClaudeTools` decoy surface if appConstants values are absent (report IMPL-BLOCKED-pending for that sub-item only). MCP rows. PAR-TRANS-057 (already HAVE, verified not re-implemented).
