# Fix micro-plan 2 — w1-j residual diff-gate findings (2026-06-11)

Author: Fable 5. Implementer: kimi. Dispatch after w1-k fix merges.
Authorizing artifact: `artifacts/w1-j-cursor-pair-diff-scoped-gpt.txt` (re-gate).

## Task 1 — raw meta-map fallback test case (BLOCKER, real)

`openai_cursor_request_test.go` `TestCursorToolRoleNamePrecedence` covers
msg.name precedence, the "tool" default, and the normalized-id fallback, but NOT
the plain raw meta-map fallback. Add a sub-case: an assistant tool_call
`{id:"tc1", function:{name:"from_meta"}}` and a `tool` message
`{tool_call_id:"tc1"}` with NO `name` → the emitted `<tool_name>` must be
`from_meta` (resolved from the meta map by raw id, not the "tool" default).
Test-only; the production lookup already does this (ref `openai-to-cursor.js:93-94`).

## Rebuttal — MINOR (passthrough redundant branches) is ref-faithful

`cursor_openai_response.go` keeps separate `chat.completion.chunk` and
`chat.completion` branches plus a fallback. This mirrors the frozen ref verbatim
(`cursor-to-openai.js:16-27`), which has the identical three branches with
explanatory comments (the executor already emits OpenAI shape; the translator is
a documented passthrough). Collapsing to a single return would diverge from the
ref's intentional structure. NO change — keep ref-faithful; note in the gate prompt.

## Acceptance (binary)

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `TestCursorToolRoleNamePrecedence` includes the raw-meta (no msg.name) sub-case asserting `from_meta` (covered by go test).
- Files touched ONLY: `openai_cursor_request_test.go`. Do NOT git commit.

## Out of scope

Passthrough branch structure (rebutted — ref-faithful). Any production change.
