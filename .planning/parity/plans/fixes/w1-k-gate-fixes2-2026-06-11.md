# Fix micro-plan 2 — w1-k residual diff-gate finding (2026-06-11)

Author: Fable 5. Implementer: kimi. Dispatch after w1-j fix2 merges.
Authorizing artifact: `artifacts/w1-k-gemini-client-claude-helpers-diff-scoped-gpt.txt` (re-gate).

## Task 1 — functionResponse content uses JS-truthiness fallback (BLOCKER, real)

`gemini_openai_request.go` (functionResponse branch, ~:203) selects
`response.result` whenever it is non-nil. The ref
(`gemini-to-openai.js:111`) is `response?.result || response || {}` — JS `||`
truthiness, so a FALSEY `result` (`0`, `false`, `""`, `null`/absent) falls back
to the full `response` object, and a falsey/absent `response` falls back to `{}`.
The Go `result != nil` check keeps falsey-but-non-nil values, diverging.
STEP (a) tests first, STEP (b) fix:
- Use a JS-truthiness predicate (reuse/port the existing `jsTruthy` if one exists
  from the w1-e `jsString` work; else add a small local helper): `result` is used
  only when truthy (non-nil AND not `0`/`0.0`/`false`/`""`); otherwise fall back
  to the `response` map when it is truthy (non-empty); otherwise `{}`.
- The content is then `tryParseJSONValue`/JSON-encoded per the existing path.
Tests (`gemini_openai_request_test.go`, extend `TestGeminiOpenAIFunctionResponse`):
`result: 0` → content is the full response object (not `0`); `result: ""` → full
response; `result: false` → full response; `result: {"x":1}` → that result;
absent `result` → full response; absent/empty `response` → `{}`.

## Acceptance (binary)

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `TestGeminiOpenAIFunctionResponse` covers falsey-result fallback (0/false/""/absent) → response, and empty response → `{}` (covered by go test).
- Files touched ONLY: `gemini_openai_request.go`(+test). Do NOT git commit.

## Out of scope

Any other w1-k behavior (gate-clean after w1-k fix 1).
