# Fix micro-plan v3 — w1-c round-2 findings (2026-06-10)

Author: Fable 5. Implementer: kimi-for-coding via run-worker.sh.
Authorizing artifact: `artifacts/w1-c-stream-processor-diff-scoped-gpt.txt`
(round 2). All four findings verified REAL by the planner against the frozen
ref; v2's Task 1 was implemented incompletely (only the middle early return
was converted to sequential mutation).

## Task 1 — AdjustMaxTokens: NO early returns at all (TDD)

Ref `maxTokensHelper.js:9` is `let maxTokens = body.max_tokens || DEFAULT_MAX_TOKENS;`
— a default feeding the SEQUENTIAL pipeline, not a return. The ref's own comment
(:19-20) names the case: default 64000 with `budget_tokens >= 64000` must still
bump to `budget + 1024`.

1. FIRST add to `TestAdjustMaxTokensToolsThenThinkingBudget` (or a sibling test):
   - missing `max_tokens` + `thinking.budget_tokens: 70000` → **71024**
   - `max_tokens: 0` + `budget_tokens: 70000` → **71024**
   - missing `max_tokens`, no tools, no thinking → **64000** (unchanged default)
   Run; see the first two fail.
2. Rewrite `AdjustMaxTokens` (`internal/translation/maxtokens.go:12-32`) as pure
   sequence: `value := toInt(raw)`; `if !ok || raw == nil || value <= 0 { value = defaultMaxTokens }`;
   tools floor; budget bump; `return value`. Zero `return` statements before the
   final one.

## Task 2 — make the handler-level Azure test non-vacuous (TDD)

`internal/api/chat_test.go` `TestChatHandlerPassthroughNormalization`: the fake
provider stream MUST emit at least one chunk that CONTAINS
`prompt_filter_results` (top level) and `choices[0].content_filter_results`.
Assert the emitted SSE output (a) contains the chunk's content and (b) does NOT
contain either Azure key. A test that asserts absence of fields never sent is
vacuous — prove presence-in, absence-out.

## Task 3 — processor-level Azure test (TDD)

`internal/translation/stream_test.go` `TestProcessPassthroughStripsAzureFields`:
feed `ProcessPassthroughStream` a chunk channel whose chunk carries both Azure
fields (use the schemas.StreamChunk shape; if the typed schema cannot carry
them, marshal-inject via the chunk's raw/extension fields — check how the
processor unmarshals to map[string]any and pick the seam that reaches it);
assert the written SSE lacks both keys and retains the content. Remove or
demote the direct `stripAzureFields` call to a secondary assertion.

## Task 4 — SSE scanner EOF contract

`internal/providers/utils/sse.go:43-52`: at EOF with a non-empty final
unterminated line, when `parseLine` returns not-ok the scanner currently
returns `("", nil)` — callers interpret that as a valid empty payload. Return
`("", io.EOF)` in the not-ok branch instead (the skipped line is the last
input; there is nothing more to scan). TDD: add a case to the scanner tests —
input `"data: ok\n\ndata: {bad json"` (no trailing newline) yields exactly one
payload then `io.EOF`; also input ending in a valid unterminated `data:` line
still yields that final payload with `nil` error (existing behavior, keep).

## Acceptance (binary)

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `grep -c 'return' internal/translation/maxtokens.go` section for AdjustMaxTokens shows a single return (the final one) — verify with `awk '/func AdjustMaxTokens/,/^}/' internal/translation/maxtokens.go | grep -c return` → 1.
- `grep -c 'prompt_filter_results' internal/api/chat_test.go` ≥ 2 (sent in fixture AND asserted absent in output).
- `grep -c 'prompt_filter_results' internal/translation/stream_test.go` ≥ 2.
- Files touched ONLY: maxtokens.go, maxtokens_test.go, chat_test.go (api), stream_test.go (translation), sse.go + sse_test.go (providers/utils). Do NOT run git commit.

## Out of scope

Wave-5 usage helpers. Handler production code (chat.go). Any other file.

---

## Deviation ratified (Fable 5, 2026-06-10)

Task 3's seam investigation concluded the typed `schemas.StreamChunk`/`StreamChoice`
drop Azure fields at JSON decode, making PAR-TRANS-049's strip behavior
unreachable and untestable. The implementer added two optional tagged fields
(`StreamChunk.PromptFilterResults`, `StreamChoice.ContentFilterResults`) in
`internal/schemas/chat.go` — a w1-a-owned file. RATIFIED: this is the minimal
change that makes the row's strip semantics real instead of
accidental-by-omission; documented in the impl report's Deviations section.
