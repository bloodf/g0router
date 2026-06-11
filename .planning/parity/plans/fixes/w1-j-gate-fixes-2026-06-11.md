# Fix micro-plan — w1-j diff-gate findings (2026-06-11)

Author: Fable 5. Implementer: kimi. Dispatch after w1-l merges (no file overlap
— w1-j fix touches only cursor files — but serialize kimi). Authorizing
artifact: `artifacts/w1-j-cursor-pair-diff-scoped-gpt.txt`.

## Rebuttal — BLOCKER #1 (tool-role normalized-id fallback) is a FALSE POSITIVE

The finding claims `openai_cursor_request.go:163` tool-role name lookup "omits
normalized-id fallback." It does not need one: the ref's tool-role branch
(`openai-to-cursor.js:90-98`) is also a plain `toolCallMetaMap.get(toolCallId)`
with NO inline normalize — both ref and Go rely on the META MAP already holding
BOTH the raw and the newline-normalized keys. The Go `rememberToolMeta`
(`openai_cursor_request.go:101-114`) stores both forms
(`toolCallMetaMap[toolCallID]` AND `toolCallMetaMap[normalized]`), byte-faithful
to ref `rememberToolMeta` (`:55-63`). So a newline-suffixed `tool_call_id`
resolves via the normalized key already present in the map. The inline
normalize the finding references exists ONLY in the user-array tool_result
branch (ref `:115-117`), which the Go code also mirrors. No code change.

## Task 1 — cover the normalized-id fallback in the test (BLOCKER #2, real)

`openai_cursor_request_test.go` `TestCursorToolRoleNamePrecedence` asserts
msg.name precedence and meta-map fallback but not the normalized-id case the
plan named. STEP (a) extend it, STEP (b) confirm it passes (no code change
expected — the dual-key map already handles it; if it fails, the map population
diverges from ref and THAT is the fix):
- Sub-case: assistant tool_call with `id: "tcX\nextra"` (newline suffix) and
  `function.name: "from_meta_norm"`; then a `tool` message with
  `tool_call_id: "tcX"` (the normalized form). Assert the emitted
  `<tool_name>` is `from_meta_norm` (resolved via the normalized key), NOT the
  `"tool"` default.
- Keep the existing msg.name-precedence and raw-meta sub-cases.

## Acceptance (binary)

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `TestCursorToolRoleNamePrecedence` includes a newline-suffixed-id sub-case
  asserting normalized-key resolution (covered by go test).
- Files touched ONLY: `openai_cursor_request_test.go` (test-only; no production
  change unless the sub-case reveals a real map-population divergence). Do NOT
  git commit.

## Out of scope

Any production change to the cursor translator (rebutted — ref-faithful). The
response passthrough (gate-clean).
