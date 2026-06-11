# Fix micro-plan — w1-k diff-gate findings (2026-06-11)

Author: Fable 5. Implementer: kimi. Dispatch after the w1-j fix merges
(serialize kimi). Authorizing artifact:
`artifacts/w1-k-gemini-client-claude-helpers-diff-scoped-gpt.txt`. Both real.

## Task 1 — tool-call id format + uniqueness (MAJOR, real)

`gemini_openai_request.go` builds functionCall ids with `UnixNano`; the plan
specified `call_<unixms>_<part-index>` (ref `gemini-to-openai.js:98`
`call_${Date.now()}_${rand}` — `Date.now()` is unix MILLIS, not nanos). The id
is opaque (parity = a unique `call_` id, not exact bytes), but align to the
plan's documented shape AND prove uniqueness:
- STEP (a): strengthen `TestGeminiOpenAIFunctionCall` — a content with TWO
  `functionCall` parts → two tool_calls whose ids both start `call_` AND are
  DISTINCT from each other (uniqueness within the same message, which a bare
  millis-only id would violate — the part-index disambiguates).
- STEP (b): change the id to `call_<unixMilli>_<partIndex>` (use the part loop
  index as the disambiguator; `time.Now().UnixMilli()`).

## Task 2 — registry wiring test asserts response translators UNCHANGED (MAJOR, real)

`registry_test.go` `TestRegistryWiresGeminiClientRequest` checks the response
translators are non-nil but not that they are still `geminiToOpenAIResponse`
(w1-k only ADDS the request side; wiring the wrong response fn must fail).
Strengthen: reflect-pointer-assert that `ResponseTranslatorFor(FormatGemini,
FormatOpenAI)` and `ResponseTranslatorFor(FormatGeminiCLI, FormatOpenAI)` are
both still `geminiToOpenAIResponse` (the w1-e function), AND that the request
translators are `geminiToOpenAIRequest`. This proves the change is additive.

## Acceptance (binary)

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `grep -c 'UnixNano' internal/translation/gemini_openai_request.go` → 0 (uses UnixMilli + index).
- `TestGeminiOpenAIFunctionCall` asserts two distinct `call_`-prefixed ids.
- `TestRegistryWiresGeminiClientRequest` reflect-asserts response = `geminiToOpenAIResponse`, request = `geminiToOpenAIRequest` for both pairs.
- Files touched ONLY: `gemini_openai_request.go`(+test), `registry_test.go`. Do NOT git commit.

## Out of scope

Any other w1-k behavior (gate-clean).
