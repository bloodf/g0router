# Fix micro-plan 4 — w1-k jsTruthy must match JS object/array truthiness (2026-06-11)

Author: Fable 5. Implementer: kimi. Dispatch after w1-l fix2 merges.
Authorizing artifact: `artifacts/w1-k-gemini-client-claude-helpers-diff-scoped-gpt.txt` (round 4).

## Root cause — PLANNER ERROR in fix3's jsTruthy spec

fix3 told the worker `jsTruthy` should treat empty maps/slices as FALSEY. That
is WRONG: in JavaScript, objects and arrays are ALWAYS truthy, even when empty —
only `null`/`undefined`/`0`/`NaN`/`""`/`false` are falsey. So for
`response?.result || response || {}` (gemini-to-openai.js:111) with
`result = {}` (empty object), JS picks `result` (truthy) and serializes `{}`.
The current Go treats `{}` as falsey, wrongly falling back to `response`.

## Task 1 — correct jsTruthy to JS semantics (BLOCKER, real)

Redefine the truthiness helper used in the functionResponse fallback:
- `nil` → false
- `bool` → the value
- `float64`/numeric → `v != 0`
- `string` → `v != ""`
- `map[string]any` → true (any non-nil map, INCLUDING empty `{}`)
- `[]any` → true (any non-nil slice, INCLUDING empty `[]`)
- anything else non-nil → true
This is the standard JS `Boolean(x)` for these JSON types. The
`result || response || {}` chain then works: empty-object `result` is used (→ `{}`),
falsey-scalar `result` falls back to `response`, falsey-scalar/absent `response`
falls back to `{}`.

## Task 2 — tests for empty-object truthiness (TDD)

Extend `TestGeminiOpenAIFunctionResponse`:
- `response:{result:{}}` → content `{}` (empty result object is truthy, used directly).
- `response:{result:[]}` → content `[]` (empty array truthy).
- `response:{result:0}` → content is the full response (`{result:0}`)? NO — re-derive:
  `result=0` falsey → fall to `response={result:0}` (a non-empty map, truthy) →
  serialize `{"result":0}`. Assert that.
- `response:{}` (no result) → `{}` (empty-map response is truthy → used → `{}`).
- Keep prior falsey-scalar-result cases consistent with the corrected helper.

## Acceptance (binary)

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `TestGeminiOpenAIFunctionResponse` asserts `result:{}` → `{}` (empty object truthy) and `result:0` → `{"result":0}` (falsey scalar falls back to the response map).
- Files touched ONLY: `gemini_openai_request.go`(+test). Do NOT git commit.

## Out of scope

Everything else (gate-clean). This corrects the planner's fix3 jsTruthy spec and closes row-066 functionResponse parity.
