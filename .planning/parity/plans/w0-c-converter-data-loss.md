# w0-c — provider converter silent data loss (rev 2)

Rows: AUD-031, AUD-035, AUD-038, AUD-039, AUD-040, AUD-041, AUD-042, AUD-049. Runs after w0-a merges; parallel with w0-b (disjoint files).
Worker: M3. Reviewer: Kimi diff gate.

## File ownership (exclusive while in flight)
- `internal/providers/anthropic/converter.go`, `internal/providers/anthropic/converter_test.go`
- `internal/providers/gemini/converter.go`, `internal/providers/gemini/converter_test.go`
- `internal/providers/gemini/chat.go`, `internal/providers/gemini/chat_test.go`

## Tasks (TDD order)

1. **AUD-031 / AUD-035** multiple system messages: table-driven test per converter — request with two `role:system` messages yields a system prompt containing both, joined `"\n\n"`, original order. Behavior evidence: 9router merges all system messages into the system field rather than keeping the last (PAR-TRANS-014, `matrix/9router-translation.md` — evidence `_refs/9router/open-sse/translator/request/openai-to-claude.js:27-134`). Fix both converters to accumulate.
2. **AUD-038** Gemini `ToolCallID`: test — tool-role message with `ToolCallID: "call_abc"` produces a functionResponse part with `id: "call_abc"`. Mechanism evidence: Gemini API functionResponse carries `id`; 9router does exactly this (`_refs/9router/open-sse/translator/request/openai-to-gemini.js:171-177` — `functionResponse: { id: fid, name: ..., response: ... }`). Fix `convertMessages`.
3. **AUD-039 / AUD-040** empty IDs: tests — `ConvertChatResponse` and `ConvertStreamChunk` return `ID` matching `^chatcmpl-[A-Za-z0-9]+$` when upstream provides none (row remediation: "Generate response/chunk IDs in Gemini instead of leaving empty"). Generate per call; no cross-chunk requirements.
4. **AUD-041** colliding tool-call IDs: test — same function invoked twice in one response yields two distinct `ToolCall.ID` values. Fix: append a per-response counter or random suffix to the current `"call_" + name` scheme.
5. **AUD-042** malformed tool args: test — invalid JSON in function-call arguments returns an error from the converter (row remediation: "Return error on malformed JSON tool arguments"). Fix: check `json.Unmarshal` error and propagate.
6. **AUD-049** model sanitization: test — chat request with model `"gemini/gemini-1.5-pro"` builds an upstream URI containing `gemini-1.5-pro`, not the prefixed form. Fix: call existing `sanitizeModelName` in `gemini/chat.go` (embeddings path already does — mirror it).

## Acceptance (binary — one check per row)
- AUD-031: anthropic two-system test passes. AUD-035: gemini two-system test passes.
- AUD-038: functionResponse `id` round-trip test passes.
- AUD-039/040: ID-regex tests pass for response and chunk.
- AUD-041: distinct-IDs test passes.
- AUD-042: malformed-args error test passes.
- AUD-049: URI sanitization test passes.
- `go test ./...` green; `go vet ./...` clean.

## Out of scope
Unmapped request fields (w0-e). Stream-lifecycle ID stability. Anthropic/OpenAI `chat.go` (w0-a, w0-e). `internal/api`. Any new converter capability beyond the 8 rows.
