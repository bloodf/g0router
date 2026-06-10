# Fix micro-plan — w1-g diff-gate findings (2026-06-10)

Author: Fable 5. Implementer: kimi-for-coding. **Dispatch AFTER w1-h merges**
(both touch translation files; w1-g fix touches only `responses_*`/`openai_responses_*`
so no file overlap with w1-h's ollama/commandcode files — but serialize to be safe).
Authorizing artifact: `artifacts/w1-g-responses-api-diff-scoped-gpt.txt`.

## Task 1 — remove dead code (BLOCKER, real)

`internal/translation/openai_responses_response.go:481` — `parseIntAny` is
unused. Delete the function. Verify no caller: `grep -rn parseIntAny internal/`
returns 0 after removal.

## Task 2 — assert the third tool-result coercion case (MAJOR, real test gap)

`internal/translation/openai_responses_request_test.go`
`TestOpenAIResponsesToolResultCoercion`: the third message (c3, `content` is a
JSON object `{result:42}`) is never asserted — the test re-checks `input[1]`
instead of `input[2]`. Fix: assert `input[2]["output"]` equals the
JSON-stringified object `{"result":42}` (ref `request/openai-responses.js:130`
`JSON.stringify(item.output)` for non-string tool output via the chat→responses
path; here it is the chat→responses `tool` message coercion at `:273-284`).
Confirm the production code already JSON-stringifies (it should — only the test
is wrong); if production is also wrong, fix it to match the ref and note it.

## Rebuttal — Finding 2 (hosted-tool conversion) is a FALSE POSITIVE

`responses_openai_request.go:202` converts any named tool, dropping only
nameless ones. This is faithful to the frozen ref `request/openai-responses.js:156-176`:
the ref's only filter is `if (!name || name.trim()==="") return null` — it does
NOT additionally gate on `type==="function"`. Hosted tools "carry no explicit
name field" (ref comment :152-155), so they are nameless and dropped by the name
check. The Go port applies `normalizeToolParameters` exactly as the ref. No
change; rebut in the gate prompt with these ref lines.

## Acceptance (binary)

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `grep -rn parseIntAny internal/translation/` → 0 hits.
- `TestOpenAIResponsesToolResultCoercion` asserts `input[2]` output ==
  `{"result":42}` (covered by go test).
- Files touched ONLY: `openai_responses_response.go`,
  `openai_responses_request_test.go`. Do NOT run git commit.

## Out of scope

Any other w1-g file. The hosted-tool behavior (rebutted, ref-faithful).
