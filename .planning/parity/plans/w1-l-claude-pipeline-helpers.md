# w1-l — claude pipeline helpers (cloaking, bypass, prepareClaudeRequest)

Split from w1-k 2026-06-10. These three share claude-cloaking / thinking-signature / appConstants dependencies, so they cohere as one plan.

Rows: PAR-TRANS-022 (claude cloaking — billing header + fake user id + `_ide` tool suffix + decoy tools; matrix row 022 corrected 2026-06-10 from `_cc` to `_ide` per ref `appConstants.js:75`), PAR-TRANS-054 (handleBypassRequest), PAR-TRANS-055 (prepareClaudeRequest).

Frozen ref (@ 827e5c3), read whole:
- `open-sse/utils/claudeCloaking.js:1-155` (row 022)
- `open-sse/config/appConstants.js:75-105` (`CLAUDE_TOOL_SUFFIX = "_ide"` at :75; `CC_DEFAULT_TOOLS` set at :79 — 26 entries)
- `open-sse/config/defaultThinkingSignature.js:2` (`DEFAULT_THINKING_CLAUDE_SIGNATURE` literal)
- `open-sse/utils/bypassHandler.js:1-195` (row 054, incl. `SKIP_PATTERNS`, `createStreamingResponse`/`createNonStreamingResponse`)
- `open-sse/services/provider.js:49` (`detectFormat(body)` body-shape detector used by bypass)
- `open-sse/translator/helpers/claudeHelper.js:8` (`hasValidContent`), `:23` (`fixToolUseOrdering`), `:81-215` (`prepareClaudeRequest`)

## Preconditions

- `grep -n 'deriveSessionId' internal/translation/cloud_code.go` → present (`cloud_code.go:54`, w1-f — reused by row 022/055 cloaking)
- `grep -rn 'cloakClaudeTools\|applyCloaking\|HandleBypassRequest\|PrepareClaudeRequest\|claudeToolSuffix' internal/translation/` → 0 hits (re-run before impl; hit → IMPL-BLOCKED)
- `grep -n 'func claudeToOpenAIRequest' internal/translation/` → present (w1-b, PAR-TRANS-057)

## Exclusive file ownership

NEW: `claude_appconstants.go`+`_test.go`, `claude_cloaking.go`+`_test.go`, `bypass_handler.go`+`_test.go`, `claude_prepare.go`+`_test.go`, `claude_thinking_signature.go`.
Non-overlap is filename-auditable against every other plan's "## Exclusive file ownership": w1-g `responses_*`; w1-h `*ollama*`/`*commandcode*`; w1-i `kiro_*`; w1-j `*cursor*`; w1-k `gemini_openai_request`/`strip_content_types`/`tool_deduper`/`reasoning_injector`. None matches `claude_*`/`bypass_*`.
TOUCH-ONLY: none (no registry change — these are pipeline helpers, not registered translators; the Wave-2/4 routing layer consumes them).

## Tasks (each: STEP (a) write named failing tests; STEP (b) port)

1. **Constants** (`claude_appconstants.go`, `claude_thinking_signature.go`): `claudeToolSuffix = "_ide"` (appConstants.js:75); `ccDefaultTools` set (appConstants.js:79-105, all 26 names — byte-exact, pin count in test); `ccDecoyTools` (claudeCloaking.js:95-116, 20 entries each `{name, description:"This tool is currently unavailable.", input_schema:{type:object,properties:{}}}`); `defaultThinkingClaudeSignature` constant (defaultThinkingSignature.js:2 verbatim — pin by `len` + sha256 in test).
   Tests: `TestCCDefaultToolsCount` (==26), `TestCCDecoyToolsShape` (20 entries, all unavailable desc), `TestClaudeThinkingSignaturePinned` (sha256 + length).

2. **Row 022: cloaking** (`claude_cloaking.go`), port `claudeCloaking.js:1-155`:
   - `generateBillingHeader(payload) (string, error)` (:9-14): sha256-hex first-5 of `json.Marshal(payload)`, 3-hex build via crypto/rand (wrap errors), exact format `x-anthropic-billing-header: cc_version=2.1.92.<build>; cc_entrypoint=sdk-cli; cch=<cch>;`.
   - `generateFakeUserID(sessionID string) (string, error)` (:18-23): 64-hex device (crypto/rand), uuid account, uuid session (or provided), exact JSON shape.
   - `cloakClaudeTools(body) (cloaked map[string]any, toolNameMap map[string]string)` (:34-92): suffix every client tool name with `_ide`, build map suffixed→original, append `ccDecoyTools`, rename `tool_use` names in message history, rewrite forced `tool_choice.name` only when it targets a renamed client tool. `decloakToolNames(body, toolNameMap)` (:83-92).
   - `applyCloaking(body, apiKey, sessionID) map[string]any` (:128-155): only when apiKey contains `sk-ant-oat`; inject billing header as system[0] (string/array/absent shapes, skip if already injected); inject `metadata.user_id` fake id when absent.
   Tests: `TestCloakToolsSuffixAndDecoys` (`_ide` suffix, map, decoys appended, tool_use renamed, forced tool_choice rewritten only for client tools), `TestDecloakToolNames`, `TestApplyCloakingOAuthOnly`, `TestApplyCloakingBillingHeaderShapes`, `TestApplyCloakingFakeUserIdPreserved`, `TestGenerateBillingHeaderFormat`.

3. **Row 054: bypass** (`bypass_handler.go`), port `bypassHandler.js:1-195`. Body-shape `detectFormat(body)` (provider.js:49) is needed — port it as an unexported `detectBypassSourceFormat(body)` in this file IF g0router lacks a body-shape detector (it has only `DetectFormatByEndpoint`); cite provider.js:49. `SKIP_PATTERNS` const (find definition in bypassHandler.js). `HandleBypassRequest(body map[string]any, model, userAgent string, ccFilterNaming bool) ([]map[string]any, bool)`: nil/false unless userAgent contains "claude-cli"; five patterns (assistant `{` title, "Warmup", single-user "count", SKIP_PATTERNS, isNewTopic naming); naming synthesizes `{isNewTopic:true, title:<first 3 words>}`; `createStreamingResponse`/`createNonStreamingResponse` build the source-format response (:128-195).
   Tests: `TestBypassNonClaudeCliNil`, `TestBypassTitleExtraction`, `TestBypassWarmup`, `TestBypassCount`, `TestBypassSkipPatterns`, `TestBypassNamingIsNewTopic`, `TestBypassNoMatchNil`.

4. **Row 055: prepareClaudeRequest** (`claude_prepare.go`), port `claudeHelper.js:8,23,81-215`: `hasValidContent` (:8), `fixToolUseOrdering` (:23), `PrepareClaudeRequest(body map[string]any, provider, apiKey, connectionId string) map[string]any`: minimax/minimax-cn output_config delete; system cache_control reset → last block ttl 1h; messages pass-1 (strip cache_control, filter empty keep final assistant), pass-1.5 fixToolUseOrdering, pass-2 reverse (last-assistant cache_control on last non-thinking block; for claude/anthropic-compatible: replace thinking signatures with `defaultThinkingClaudeSignature`, inject thinking block when enabled+tool_use+no-thinking); tools filter built-ins for non-claude, last-tool cache_control ttl 1h, drop empty tools+tool_choice; applyCloaking for claude/anthropic-compatible+apiKey using `deriveSessionId(connectionId)` (reuse `cloud_code.go:54`).
   Tests: `TestPrepareSystemCacheControl`, `TestPrepareFiltersEmptyKeepsFinalAssistant`, `TestPrepareToolUseOrdering`, `TestPrepareLastAssistantCacheControl`, `TestPrepareThinkingInjection`, `TestPrepareToolsBuiltinFilterNonClaude`, `TestPrepareMinimaxOutputConfigDropped`, `TestPrepareCloakingForOAuth`.

## Binary acceptance criteria

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `grep -rn 'func init(\|panic(' <all new files>` → 0 hits.
- `grep -c '"_ide"' internal/translation/claude_appconstants.go` ≥ 1.
- `grep -c 'sk-ant-oat' internal/translation/claude_cloaking.go` ≥ 1.
- `TestCCDefaultToolsCount` (==26), `TestCCDecoyToolsShape` (==20), `TestClaudeThinkingSignaturePinned` all pass.

## Parity-observability note (addresses "dead helper" gate concern)

These three are exported, directly-unit-tested functions — the SAME shape 9router ships them in: `handleBypassRequest` is a util called from the request handler (`bypassHandler.js` is imported by the chat handler), `prepareClaudeRequest` is a `claudeHelper.js` export called by the Claude-endpoint path, `applyCloaking`/`cloakClaudeTools` are called from `prepareClaudeRequest` and the executor. Wave 1's parity unit is "the function exists and behaves per ref, proven by unit tests"; wiring them into g0router's request handler is Wave-2/4 routing scope (the handlers that consume them do not exist until then). This mirrors the accepted w1-h decision for `OllamaBodyToOpenAI` (exported converter, consumer arrives with its handler wave).

## Out of scope

Registry wiring (none — pipeline helpers). Pipeline integration into routing/executors (Wave 2/4). gemini-client + small helpers (w1-k). MCP rows.
