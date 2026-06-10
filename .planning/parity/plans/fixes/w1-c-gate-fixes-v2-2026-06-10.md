# Fix micro-plan v2 — w1-c diff-gate findings (2026-06-10, supersedes v1)

Author: Fable 5. Implementer: kimi-for-coding via run-worker.sh.
Authorizing artifact: `artifacts/w1-c-stream-processor-diff-scoped-gpt.txt` (first
properly-scoped w1-c review). v1's "no code changes" triage covered only the two
HANDOFF carryover items; this scoped run surfaced one real behavioral blocker.

## Triage

| # | Finding | Verdict | Action |
|---|---------|---------|--------|
| 1 | BLOCKER `AdjustMaxTokens` early-returns 32000 with tools, skipping the thinking-budget bump | **REAL** — ref `maxTokensHelper.js:8-26` applies BOTH adjustments sequentially: tools floor THEN `budget+1024` when `maxTokens <= budget`. tools + max_tokens:1000 + budget:60000 must yield 61024 | Task 1 |
| 2 | MAJOR w1-b framing tests "added/changed in this diff" | **FALSE POSITIVE — stale gate base**: `TestMessagesHandlerStreamingFraming`/`...AbortsOnErrorChunk` were ADDED by w1-b commit `14a23db` ("PAR-TRANS-056 part 2"), which sits INSIDE the stale `32108f04..` range. Base corrected to `14a23db` in diff-scopes.json | Task 4 (scope fix only) |
| 3 | MAJOR `TestWriteSSEStreamAbortsOnMarshalError` deleted in `0df5dd9` | **REAL** — coverage was removed, not relocated | Task 2 |
| 4 | MAJOR `TestChatStreamPassthroughNormalization` does not exercise `ChatHandler` nor assert Azure stripping | **REAL** vs plan task 6 | Task 3 |
| 5 | MAJOR Azure-strip tested on helper, not `ProcessPassthroughStream` | **REAL** vs plan task 5 test list | Task 3 |

## Task 1 — AdjustMaxTokens sequential adjustments (TDD)

1. FIRST add to `internal/translation/maxtokens_test.go`:
   `TestAdjustMaxTokensToolsThenThinkingBudget` — body `{max_tokens:1000, tools:[1 tool], thinking:{budget_tokens:60000}}` → **61024**; body `{max_tokens:1000, tools:[...], thinking:{budget_tokens:10000}}` → **32000** (floor applied, 32000 > 10000 so no bump); body `{max_tokens:70000, tools:[...], thinking:{budget_tokens:70000}}` → **71024** (`<=` is inclusive, ref :21). Run; see the first case fail.
2. Restructure `AdjustMaxTokens` (`internal/translation/maxtokens.go:12-32`) from early returns to sequential mutation, mirroring ref order exactly: default → tools floor → budget bump → return. No signature change.

## Task 2 — restore marshal-abort coverage (TDD)

`internal/api/chat_test.go`: re-add `TestWriteSSEStreamAbortsOnMarshalError`
asserting the streaming writer aborts (returns/propagates error, stops writing)
when a chunk cannot be marshaled. Recover the deleted test body as the starting
point: `git show 32108f04:internal/api/chat_test.go`. Adapt only to the current
error-returning signature introduced in `0df5dd9` — keep the original abort
semantics asserted.

## Task 3 — handler/processor-level passthrough tests (TDD)

1. `internal/api/chat_test.go`: extend `TestChatStreamPassthroughNormalization`
   (or add `TestChatHandlerPassthroughNormalization`) to drive **ChatHandler**
   end-to-end with a fake provider stream (follow the package's existing fake
   seams; no mocks) whose chunks include `prompt_filter_results` and
   `choices[0].content_filter_results`; assert emitted SSE has both stripped and
   required fields injected.
2. `internal/translation/stream_test.go`: repoint
   `TestProcessPassthroughStripsAzureFields` to call `ProcessPassthroughStream`
   over a chunk channel and assert the written SSE lacks Azure fields (keep the
   helper-level assertions as a sub-check if desired; processor-level is the
   requirement).

## Task 4 — gate scope correction (orchestrator, not worker)

`diff-scopes.json` `w1-c-stream-processor.base`: `32108f04` → `14a23db` with note
"base = w1-b part 2 (/v1/messages streaming); framing tests belong to w1-b".

## Acceptance (binary)

- `go test ./...`, `go vet ./...` green.
- The three new/changed tests fail before their fixes and pass after (report must show the failing run for Task 1).
- `AdjustMaxTokens` has no early return between the tools floor and the budget bump.
- Do NOT touch any file other than: maxtokens.go, maxtokens_test.go, chat_test.go (api), stream_test.go (translation). Do NOT run git commit.

## Out of scope

estimateUsage/addBufferToUsage/filterUsageForFormat (Wave 5). Handler production
code changes (chat.go/messages.go) — if Task 3 reveals a production defect,
write IMPL-BLOCKED with the evidence instead of fixing it.
