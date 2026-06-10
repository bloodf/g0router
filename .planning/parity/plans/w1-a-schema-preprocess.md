# w1-a — translation foundation: schema fields + request preprocessing (rev 3)

Rows: PAR-TRANS-007, PAR-TRANS-008, PAR-TRANS-009, PAR-TRANS-020 (schema field only), PAR-TRANS-021 (schema field only), PAR-TRANS-023 (schema field only), PAR-PR-1054, PAR-PR-664 (schema field only).
(PAR-TRANS-006 `stripContentTypes` dropped: its stripList input is per-model provider config landing in Wave 2; a helper with no production caller is dead code — row stays MISSING until Wave 2. PAR-TRANS-011 `adjustMaxTokens` moved to w1-c: its only consumer is the Claude request translator built there.)
PAR-PR row evidence (PARITY.md §3, "Mapped to 9router-translation.md", lines 163 and 175):
- `| #1054 | PAR-PR-1054 | Support OpenAI max_completion_tokens parameter |`
- `| #664 | PAR-PR-664 | Translate max_tokens to max_completion_tokens for openai-compatible providers |`
Wave 1 plan 1 of ~10. Coordination evidence: `WAVE-MAP.md:35` — "Plans are written wave-by-wave (not all upfront)" — and `WAVE-MAP.md:24-30` (plan factory protocol: gate → dispatch → merge per plan). w1-a is the only Wave 1 plan authored at dispatch time and merges before any other Wave 1 plan is written, the same solo-window rule Wave 0 used for w0-a (`w0-a-rand-marshal-errors.md` "File ownership (time-boxed exclusivity)").
Worker: Kimi. Reviewer: gpt-5.5 diff gate.

## Why these rows together
The 9router request pipeline runs preprocessing steps on every request before any format translation (`_refs/9router/open-sse/translator/index.js:75-90`): `normalizeThinkingConfig` → `ensureToolCallIds` → `fixMissingToolResponses` (plus config-driven `stripContentTypes`, deferred — see Rows note). They are pure functions over the OpenAI-shaped body and require `thinking`/`reasoning_effort` request fields that g0router's schema lacks (matrix "Data Models": `schemas.ChatRequest` has no `thinking`, `reasoning_effort`, `max_completion_tokens`). Schema fields and the helpers that read them land together so nothing is dead on arrival; the chat handler wires them as production callers.

## File ownership (exclusive while in flight)
- `internal/schemas/chat.go` (field additions only; no removals)
- `internal/translation/` — NEW package: `preprocess.go`, `preprocess_test.go`
- `internal/api/chat.go`, `internal/api/chat_test.go` (preprocessing call only)

## Tasks (TDD order — each test fails before its fix)

1. **Schema fields** (`internal/schemas/chat.go`). Test `TestChatRequestNewFieldsRoundTrip` in `internal/translation/preprocess_test.go`: marshal/unmarshal a ChatRequest carrying every new field and assert values survive. Add to `ChatRequest`: `TopK *int json:"top_k,omitempty"` (PAR-TRANS-023: 9router maps `top_k`→`topK`, `_refs/9router/open-sse/translator/request/openai-to-gemini.js:47-59`; the request field must exist before w1-d can map it), `MaxCompletionTokens *int json:"max_completion_tokens,omitempty"` (PAR-PR-1054/PAR-PR-664 quoted above), `ReasoningEffort string json:"reasoning_effort,omitempty"` (PAR-TRANS-020; values none/low/medium/high/xhigh per `_refs/9router/open-sse/translator/request/openai-to-claude.js:182-199`), `Thinking *ThinkingConfig json:"thinking,omitempty"` (PAR-TRANS-021) with `type ThinkingConfig struct { Type string json:"type"; BudgetTokens int json:"budget_tokens,omitempty" }`. No `Message` changes (reasoning_content is w1-e scope).
2. **PAR-TRANS-008 `EnsureToolCallIDs`** (`internal/translation/preprocess.go`). Tests: (a) assistant tool_call with ID failing `^[a-zA-Z0-9_-]+$` is sanitized (strip invalid chars) or, when empty after sanitize, regenerated as `call_msg{i}_tc{j}_{name}` (name keeps only `[a-zA-Z0-9_-]`); (b) valid IDs untouched; (c) `role:tool` messages with invalid `tool_call_id` fixed the same way; (d) missing `Type` set to `"function"`. Source contract is sanitize-then-regenerate, verbatim from `_refs/9router/open-sse/translator/helpers/toolCallHelper.js:29-32`:
   ```js
   if (!tc.id || !TOOL_ID_PATTERN.test(tc.id)) {
     const sanitized = sanitizeToolId(tc.id);
     tc.id = sanitized || generateToolCallId(i, j, tc.function?.name);
   }
   ```
   The matrix edge-case note ("regenerated deterministically") summarizes only the fallback branch; the source above is authoritative. (The JS "arguments to string" step is skipped: `Function.Arguments` is already `string` in g0router's schema.)
3. **PAR-TRANS-009 `FixMissingToolResponses`** (`preprocess.go`). Tests: (a) assistant message with N tool_calls followed by a non-tool message gets N inserted `{Role:"tool", ToolCallID:id, Content:""}` messages directly after it; (b) when the next message answers the IDs, nothing inserted; (c) assistant tool_calls as final message: 9router only inserts when a next message exists and lacks results (`toolCallHelper.js:127-138` — `if (nextMsg && !hasToolResults(...))`), so trailing tool_calls get nothing; test pins that. Source: `toolCallHelper.js:116-147`.
4. **PAR-TRANS-007 `NormalizeThinkingConfig`** (`preprocess.go`). Tests: last message not `role:user` → `Thinking` and `ReasoningEffort` cleared; last message user → untouched. The row text mentions only the thinking block; the cited source clears both fields, verbatim from `_refs/9router/open-sse/services/provider.js:350-356`:
   ```js
   export function normalizeThinkingConfig(body) {
     if (!isLastMessageFromUser(body)) {
       delete body.reasoning_effort;
       delete body.thinking;
     }
     return body;
   }
   ```
5. **`PreprocessChatRequest` + production wiring**. Add `translation.PreprocessChatRequest(req *schemas.ChatRequest)` running, in 9router's order (index.js:80-90): `NormalizeThinkingConfig` → `EnsureToolCallIDs` → `FixMissingToolResponses`. Test `TestPreprocessChatRequestOrder`: input has an assistant tool_call with an invalid ID and no following tool response; after preprocessing, the inserted `role:tool` message carries the *sanitized* ID — passes only if ID-fixing runs before response insertion.
6. **Behavioral wiring test seam** (`internal/api/chat.go`, `chat_test.go`). Introduce `type modelResolver interface { ResolveForModel(*schemas.ChatRequest) (schemas.Provider, schemas.Key, error) }` in the api package; `ChatHandler.router` field becomes `modelResolver` (`*inference.Router` already satisfies it — `internal/inference/router.go:58-60`; `NewChatHandler` keeps its `*inference.Router` parameter type, so no caller changes). This is the repo's interfaces-and-fakes convention — AGENTS.md `## Conventions` bullet 5, verbatim: "No mocks — use interfaces and fakes; test real behavior." Precedent: w0-a's approved `randRead`/`jsonMarshal` seams (`w0-a-rand-marshal-errors.md` "Error-injection seams") established that minimal test seams accompany the behavior they verify. `Handle` calls `translation.PreprocessChatRequest(&req)` after unmarshal, before resolve. Test `TestChatHandlerPreprocessesRequest`: a fake resolver captures the request it receives and returns an error (handler then 400s); post a body whose assistant tool_call has an invalid ID and no tool response; assert the captured request has the sanitized ID and the inserted `role:tool` message — production wiring observed behaviorally, not textually.

## Acceptance (binary)
- `go test ./internal/translation/ ./internal/api/ ./internal/schemas/...` green; `go test ./...` green; `go vet ./...` clean.
- `TestChatHandlerPreprocessesRequest` passes (behavioral proof the handler preprocesses before resolving) and fails if the `PreprocessChatRequest` call is removed from `Handle`.
- These exact tests exist and pass: `TestChatRequestNewFieldsRoundTrip`; `TestEnsureToolCallIDsSanitizesInvalid`, `TestEnsureToolCallIDsRegeneratesEmpty`, `TestEnsureToolCallIDsKeepsValid`, `TestEnsureToolCallIDsFixesToolMessages`, `TestEnsureToolCallIDsSetsType`; `TestFixMissingToolResponsesInserts`, `TestFixMissingToolResponsesSkipsAnswered`, `TestFixMissingToolResponsesIgnoresTrailing`; `TestNormalizeThinkingConfigClears`, `TestNormalizeThinkingConfigKeepsForUser`; `TestPreprocessChatRequestOrder`; `TestChatHandlerPreprocessesRequest`.
- No existing `schemas.ChatRequest`/`Message` field removed or renamed (`git diff` on `internal/schemas/chat.go` shows additions only).

## Out of scope
Format registry and translateRequest pipeline (w1-b). Claude/Gemini translator changes (w1-c/w1-d). `filterToOpenAIFormat` (PAR-TRANS-010, response direction — w1-e). reasoning_effort→thinking mapping (w1-c). stripList provider config (Wave 2). `StreamChoice.Delta` reasoning_content emission (w1-e).
