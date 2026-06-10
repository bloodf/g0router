# w1-k — gemini-format client request translator

Restructured 2026-06-10 (w1-k r2 gate): this plan now contains ONLY the gemini client-request translator, a true registered translator with a Wave-1 consumer (the registry pipeline). The request-pipeline preprocessing helpers (stripContentTypes / dedupeTools / injectReasoningContent — PAR-TRANS-006/051/052/053) have no Wave-1 consumer — 9router calls them from the routing layer's `translateRequest` preprocessing, not from any registered translator — so they ship in **Wave 4 (w4-pre)** together with that integration, where the rows become observable behavior rather than dead helpers. See WAVE-MAP.md "Wave 1 remainder slicing".

Rows: PAR-TRANS-066 (gemini→openai client request, registered for both `gemini` and `gemini-cli` source formats). Registration parity basis: PAR-TRANS-001.

Frozen ref (@ 827e5c3), read whole: `open-sse/translator/request/gemini-to-openai.js:6-146` (translator) and `:145-146` (the two `register` calls).

## Preconditions (a "0 hits" grep exits 1 — that IS the pass)

- `grep -n 'FormatGemini ' internal/translation/formats.go` → present (verified 2026-06-10: `formats.go:14` `FormatGemini Format = "gemini"`)
- `grep -n 'FormatGeminiCLI' internal/translation/formats.go` → present (`formats.go:15`)
- `grep -n 'AdjustMaxTokens' internal/translation/maxtokens.go` → present (`maxtokens.go:12`, w1-c — consumed by row 066)
- `grep -n 'geminiToOpenAIResponse' internal/translation/registry.go` → present (w1-e — row 066 adds the REQUEST direction to the same `FormatGemini→FormatOpenAI`/`FormatGeminiCLI→FormatOpenAI` pairs)
- `grep -rn 'geminiToOpenAIRequest' internal/translation/` → 0 hits (re-run before impl; a hit → IMPL-BLOCKED)

## Exclusive file ownership

NEW: `internal/translation/gemini_openai_request.go` + `_test.go`.
TOUCH-ONLY: `registry.go` (change two existing `Register` calls to add the request fn), `registry_test.go` (wiring tests).
Non-overlap is filename-auditable against every other plan's "## Exclusive file ownership": w1-g `responses_*`/`openai_responses_*`; w1-h `*ollama*`/`*commandcode*`; w1-i `kiro_*`/`openai_kiro_*`; w1-j `*cursor*`; w1-l `claude_*`/`bypass_*`. None matches `gemini_openai_request`.
Dispatch-order gate (shared registry.go; same enforceable pattern the gate approved for w1-j): dispatch only after the w1-h/i/j registry hunks are merged — worker precondition `grep -c 'FormatOllama\|FormatKiro\|FormatCursor' internal/translation/registry.go` ≥ 3; if < 3 → IMPL-BLOCKED with no registry edit.

## Tasks (STEP (a) write named failing tests; STEP (b) port)

1. **Row 066: gemini→openai client request** (`gemini_openai_request.go`), port `gemini-to-openai.js:6-146`. Signature `(model, body, stream, credentials)` (credentials ignored — ref is 3-arg). generationConfig → `max_tokens` via `AdjustMaxTokens(map{"max_tokens":maxOutputTokens, "tools":body["tools"]})`, temperature, top_p (:14-26); systemInstruction → system message via `extractGeminiText` (string passthrough or parts text join, :136-142); contents → messages via `convertGeminiContent` (:72-133): text→`{type:text}`, inlineData→`{type:image_url, image_url:{url:data:<mime>;base64,<data>}}`, functionCall→assistant tool_calls (the ref id `call_${Date.now()}_${random}` at :98 is OPAQUE/non-deterministic — parity is "a unique id string is present", NOT exact bytes; Go uses `call_<unixms>_<part-index>`; tests assert id format+uniqueness, never an exact value), functionResponse→EARLY-RETURN `{role:tool, tool_call_id: id||name, content: JSON(response.result||response||{})}` (:107-113), assistant-with-tool_calls keeps text as content (single→string), single-text content collapse (:116-130).
   Tests: `TestGeminiOpenAIGenerationConfig`, `TestGeminiOpenAISystemInstruction`, `TestGeminiOpenAIContentTextAndImage`, `TestGeminiOpenAIFunctionCall`, `TestGeminiOpenAIFunctionResponse` (id then name fallback), `TestGeminiOpenAITools`, `TestGeminiOpenAISingleTextCollapse`.

2. **Registration (row 066)** — parity basis PAR-TRANS-001. The response side of both pairs is already wired (w1-e). Change the existing `Register(FormatGemini, FormatOpenAI, ...)` and `Register(FormatGeminiCLI, FormatOpenAI, ...)` calls to pass `geminiToOpenAIRequest` as the request arg, mirroring the ref's two `register` calls (`gemini-to-openai.js:145` GEMINI→OPENAI, `:146` GEMINI_CLI→OPENAI — same translator fn). No other registry change.
   Tests: `TestRegistryWiresGeminiClientRequest` — `RequestTranslatorFor(FormatGemini, FormatOpenAI)` and `RequestTranslatorFor(FormatGeminiCLI, FormatOpenAI)` both non-nil + reflect identity against `geminiToOpenAIRequest`; assert the existing response translators on those pairs are unchanged.

## Binary acceptance criteria

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `grep -rn 'func init(\|panic(' internal/translation/gemini_openai_request.go` → 0 hits.
- `grep -c 'RequestTranslatorFor(FormatGemini, FormatOpenAI)' internal/translation/registry_test.go` ≥ 1.
- All seven Task-1 tests + `TestRegistryWiresGeminiClientRequest` exist and pass (covered by go test).

## Out of scope

stripContentTypes / dedupeTools / injectReasoningContent (→ Wave 4 w4-pre, with routing integration). Claude cloaking / bypass / prepare (→ w1-l). PAR-TRANS-057 (already HAVE). Any pipeline/routing wiring.
