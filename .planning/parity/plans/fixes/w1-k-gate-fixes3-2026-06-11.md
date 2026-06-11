# Fix micro-plan 3 — w1-k functionResponse full truthiness chain (2026-06-11)

Author: Fable 5. Implementer: kimi. Dispatch after w1-l fix merges.
Authorizing artifact: `artifacts/w1-k-gemini-client-claude-helpers-diff-scoped-gpt.txt` (round 3).

## Task 1 — complete the `result || response || {}` chain (BLOCKER, real)

fix2 handled the first `||` (falsey `result` → fall back to `response`). The ref
`gemini-to-openai.js:111` is a TWO-level chain: `response?.result || response || {}`.
The SECOND `||` is unhandled — when `response` itself is a falsey primitive
(`0`, `false`, `""`, or empty), it must fall back to `{}`, but the code marshals
the falsey `response` directly.
STEP (a) tests first, STEP (b) fix:
- Apply `jsTruthy` to the chosen value at BOTH levels: `contentVal = result if
  jsTruthy(result) else (response if jsTruthy(response) else map[string]any{})`.
  `jsTruthy`: non-nil AND not `0`/`0.0`/`false`/`""` AND (for maps/slices) non-empty.
  Reuse the existing `jsTruthy`/`jsString`-adjacent helper if present; else add it.
Tests (extend `TestGeminiOpenAIFunctionResponse`): `response: 0` (no result) →
content `{}`; `response: ""` → `{}`; `response: false` → `{}`; `response: {}`
(empty map) → `{}`; `response: {"x":1}` (no result) → `{"x":1}`; existing
falsey-result cases still pass.

## Acceptance (binary)

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `TestGeminiOpenAIFunctionResponse` covers falsey `response` → `{}` AND falsey `result` → response (both `||` levels).
- Files touched ONLY: `gemini_openai_request.go`(+test). Do NOT git commit.

## Out of scope

Everything else (gate-clean). This closes the row-066 functionResponse parity.
