# w1-f — gemini-cli / antigravity / vertex envelope translators

Rows: **PAR-TRANS-039**, **PAR-TRANS-040**, **PAR-TRANS-045**; **PAR-TRANS-002** partial — this plan registers only `gemini-cli`, `vertex`, `antigravity` (remaining PAR-TRANS-002 formats: `openai-responses`→w1-g, `codex`/`kiro`/`cursor`/`ollama`/`commandcode`→w1-h..j); **PAR-TRANS-043** response alias registrations (`gemini-to-openai.js:241-245`, deferred from w1-e).
NOT in scope: PAR-TRANS-022 cloaking (w1-h); PAR-TRANS-031..038 Responses API (w1-g); plain `openai:gemini` (w1-e HAVE); provider adapter consumption (Wave 2).

Precondition check — worker runs BEFORE any edit and STOPs with IMPL-BLOCKED if any fails:
1. `grep -n "func openaiToGeminiRequest" internal/translation/openai_gemini_request.go`
2. `grep -n "func geminiToOpenAIResponse" internal/translation/gemini_openai_response.go`
3. `grep -n "func openaiToClaudeRequest" internal/translation/openai_claude_request.go`

## Why credentials must thread through the registry (task 0 evidence)
Frozen ref passes `credentials` into envelope translators; g0router does not yet:
- `_refs/9router/open-sse/translator/index.js:75,107` — `translateRequest(..., credentials = null, ...)` calls `fromOpenAI(model, result, stream, credentials)`.
- `_refs/9router/open-sse/translator/request/openai-to-gemini.js:271-280` — `wrapInCloudCodeEnvelope` reads `credentials?.projectId` and `deriveSessionId(credentials?.email || credentials?.connectionId)` for antigravity `sessionId` (row **039** envelope field).
- Same file `:456-463,468-469` — `openaiToAntigravityRequest` and `openai:gemini-cli` registration pass `credentials` into wrap helpers.
- `_refs/9router/open-sse/translator/request/openai-to-vertex.js:37-38` — vertex request translator accepts `credentials` (unused but same 4-arg contract).
- g0router gap: `internal/translation/registry.go:7` — `RequestTranslator func(model, body, stream)` missing 4th param.

Task 0 is limited to matching this ref contract; no new credential semantics beyond what row **039** envelope construction reads.

## Production-wiring scope ruling
Wave 1 delivers registered, unit-tested translators; provider dispatch consumes them in Wave 2 (`WAVE-MAP.md:14`). `internal/api/messages.go` change is one call-site pass-through of `nil` credentials until Wave 2.

## File ownership
**Exclusive NEW (all logic lives here):**
- `internal/translation/cloud_code.go`, `cloud_code_test.go` — envelope wrap + id/session helpers required by row **039** (`openai-to-gemini.js:271-447`)
- `internal/translation/openai_gemini_cli_request.go`, `openai_gemini_cli_request_test.go`
- `internal/translation/openai_antigravity_request.go`, `openai_antigravity_request_test.go`
- `internal/translation/openai_claude_antigravity.go`, `openai_claude_antigravity_test.go`
- `internal/translation/antigravity_openai_request.go`, `antigravity_openai_request_test.go` — row **040**
- `internal/translation/openai_vertex_request.go`, `openai_vertex_request_test.go`
- `internal/translation/openai_antigravity_response.go`, `openai_antigravity_response_test.go` — row **045**

**Shared touch-only** — prerequisite translators merged (verify at plan start):
- `internal/translation/openai_claude_request.go:16` — `openaiToClaudeRequest` (w1-d, commit `e63da1d9`)
- `internal/translation/gemini_openai_response.go:13` — `geminiToOpenAIResponse` (w1-e, commit `478ccff8`)
- `internal/translation/openai_gemini_request.go:14` — `openaiToGeminiRequest` (w1-e, commit `e5b2791a`)
- `internal/translation/registry.go:79-82` — `openai:claude`, `claude:openai`, `openai:gemini`, `gemini:openai` already registered

Touch-only in this plan (no behavioral edits beyond compile + new registrations):
- `internal/translation/registry.go:7,116`, `registry_test.go`
- `internal/translation/claude_request.go:10` — `claudeToOpenAIRequest` 4th param ignored
- `internal/translation/openai_claude_request.go:16` — `openaiToClaudeRequest` 4th param ignored
- `internal/translation/openai_gemini_request.go:14` — `openaiToGeminiRequest` 4th param ignored
- `internal/api/messages.go` — pass `nil` credentials

## Tasks (TDD order)

0. **Registry credentials threading** (rows **039**, **002** partial, ref API). **TDD first:** write `TestTranslateRequestForwardsCredentials` in `registry_test.go` (registers fake translator, calls `TranslateRequest(..., creds)`, asserts `credentials["connectionId"]` received) — fails to compile because signature lacks 4th param. **Then** extend signature to match ref:
   - `_refs/9router/open-sse/translator/index.js:75,97,107`
   - `_refs/9router/open-sse/translator/request/openai-to-gemini.js:468-469`
   - `_refs/9router/open-sse/translator/request/openai-to-vertex.js:37-42`
   ```go
   type RequestTranslator func(model string, body map[string]any, stream bool, credentials map[string]any) (map[string]any, error)
   ```
   Update `TranslateRequest` at `registry.go:116` to accept and forward `credentials`. Add ignored 4th param only at the three call sites listed in File ownership. `messages.go` passes `nil`.

1. **Row 039 envelope helpers** (`cloud_code.go`). TDD first:
   - `TestWrapCloudCodeEnvelopeGeminiCLIHasSafetySettings`
   - `TestWrapCloudCodeEnvelopeAntigravityHasValidatedToolConfig`
   - `TestWrapCloudCodeEnvelopeUsesCredentialsProjectId`
   - `TestWrapCloudCodeEnvelopeDerivesSessionFromConnectionId`
   - `TestWrapCloudCodeEnvelopeForClaudeToolBlocks`
   Then port helpers per table:
   | Helper | Ref lines | Envelope field consumed |
   |--------|-----------|-------------------------|
   | `generateProjectId()` | `_refs/9router/open-sse/translator/helpers/geminiHelper.js:104-120` via `openai-to-gemini.js:272` | `project` when `credentials.projectId` absent |
   | `generateRequestId()` | `geminiHelper.js:104-120` via `openai-to-gemini.js:278` | `requestId` (gemini-cli) |
   | `generateSessionId()` | `geminiHelper.js:104-120` via `openai-to-gemini.js:280` | `request.sessionId` (gemini-cli) |
   | `deriveSessionId(key)` | `_refs/9router/open-sse/utils/sessionManager.js:44-64` via `openai-to-gemini.js:280` | `request.sessionId` (antigravity) |
   | `wrapInCloudCodeEnvelope` | `:271-317` | full gemini envelope incl. `toolConfig.functionCallingConfig.mode:"VALIDATED"` when tools (`:305-310`) |
   | `wrapInCloudCodeEnvelopeForClaude` | `:319-447` | claude-model antigravity envelope |

2. **gemini-cli request** (`openai_gemini_cli_request.go`). TDD first: `TestNewRegistryWiresOpenAIGeminiCLIRequest` fails (nil translator), then `TestOpenAIGeminiCLIThinkingConfigFromReasoningEffort` fails, then `TestOpenAIGeminiCLIEnvelopeShape` fails. Then port `openaiToGeminiCLIRequest` (`openai-to-gemini.js:229-268`) and register `openai:gemini-cli` (`:468`).

3. **Row 039: openai→antigravity request** (`openai_antigravity_request.go`, `openai_claude_antigravity.go`). TDD first (each fails before implementation):
   - `TestNewRegistryWiresOpenAIAntigravityRequest`
   - `TestOpenAIAntigravityClaudeModelUsesClaudePath` — model `"claude-3-5-sonnet"` → envelope uses claude contents shape (not gemini `contents`)
   - `TestOpenAIAntigravityGeminiModelUsesGeminiCLIEnvelope` — model `"gemini-2.0-flash"` → envelope has `userAgent:"antigravity"`, `requestType:"agent"`
   - `TestOpenAIClaudeAntigravityStripsClaudeCodeSystem` — system block `"You are Claude Code..."` removed from `request.systemInstruction`
   - `TestOpenAIClaudeAntigravityToolPrefixNoOp` — with verbatim prefix `""` (`openai-to-claude.js:8`), tool `"my_tool"` unchanged after strip loop
   Then port `openaiToAntigravityRequest` (`openai-to-gemini.js:456-464`) + `openaiToClaudeRequestForAntigravity` (`openai-to-claude.js:346-394`).

4. **Row 040: antigravity→openai request** (`antigravity_openai_request.go`). TDD first:
   - `TestNewRegistryWiresAntigravityOpenAIRequest`
   - `TestAntigravityOpenAIUnwrapsEnvelope` — `{request:{contents:[...]}}` → top-level `messages`
   - `TestAntigravityOpenAIThinkingConfigToReasoningEffort` — budget 1024→`low`, 8192→`medium`, 32768→`high`
   - `TestAntigravityOpenAIConvertContentToolResults` — one `functionResponse` → separate `role:tool` message each
   - `TestAntigravityOpenAINormalizeSchemaTypes` — `OBJECT`→`object`, strips `enumDescriptions`
   Then port `antigravity-to-openai.js:5-217`.

5. **Vertex request direction** (`openai_vertex_request.go`). **Row PAR-TRANS-002** registration gap for `openai:vertex` — frozen ref is the whole file `_refs/9router/open-sse/translator/request/openai-to-vertex.js` (registration `:42`). Post-process behaviors are verbatim ref lines `:6-34`:
   - `:9-10` — replace synthetic `thoughtSignature` with `DEFAULT_THINKING_VERTEX_SIGNATURE`
   - `:23-29` — strip `id` from `functionCall` and `functionResponse` (Vertex rejects these)
   TDD first:
   - `TestNewRegistryWiresOpenAIVertexRequest`
   - `TestOpenAIVertexReplacesThoughtSignature`
   - `TestOpenAIVertexStripsFunctionIDs`
   Then port `openaiToVertexRequest` (`:37-39`) = `openaiToGeminiRequest` + `postProcessForVertex`.

6. **Row 045: openai→antigravity response** (`openai_antigravity_response.go`). TDD first:
   - `TestNewRegistryWiresOpenAIAntigravityResponse`
   - `TestOpenAIAntigravityResponseAccumulatesToolCalls`
   - `TestOpenAIAntigravityResponseFinishReasonMapping`
   - `TestOpenAIAntigravityResponseUsageMetadata`
   Then port `openai-to-antigravity.js:8-118`.

7. **PAR-TRANS-043 response alias registrations** (`registry.go`, `registry_test.go`). w1-e registered only `gemini:openai`; ref registers the same `geminiToOpenAIResponse` for gemini-cli/vertex/antigravity at `gemini-to-openai.js:241-245` (boundary documented in w1-e plan row-043 clarification). TDD first:
   - `TestRegistryGeminiCLIResponseUsesGeminiOpenAI` — `ResponseTranslatorFor(FormatGeminiCLI, FormatOpenAI) != nil`
   - `TestRegistryVertexResponseUsesGeminiOpenAI`
   - `TestRegistryAntigravityResponseUsesGeminiOpenAI`
   Then register three alias lines matching ref `:242-244`.

## Acceptance (binary)
- `go test ./...` green; `go vet ./...` clean.
- Each named test in tasks 0-7 exists and passes (29 tests total).
- `TestWrapCloudCodeEnvelopeAntigravityHasValidatedToolConfig` passes.
- `grep '"You are Claude Code"' internal/translation/openai_claude_antigravity.go | wc -l` prints `0`.
- `git diff <merge-base> -- internal/api/ internal/providers/` only `messages.go` nil-credentials call.

## Out of scope
Responses API (w1-g). Ollama/Kiro/Cursor (w1-h..j). Provider adapters. MITM/tunnel (Wave 7).
