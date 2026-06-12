# w5-f fix micro-plan — diff-gate round 2, both halves (Fable 5, 2026-06-12)

Sources: `artifacts/w5-f-plan-A-diff-scoped-gpt.txt` + `artifacts/w5-f-plan-B-diff-scoped-gpt.txt` (cycle 2).

## REBUTTED — no change
- A-BLOCKER "AddBufferToUsage must recompute total": SECOND occurrence of a finding
  rebutted in fix-r1 — ref `usageTracking.js:46-51` INCREMENTS an existing
  total_tokens by BUFFER_TOKENS; recompute only when absent. Port is ref-verbatim.
- B-BLOCKER "cmd/main.go scope creep": FALSE POSITIVE — `fixes/w5-f-fix-r1.md`
  §Fix 1 EXPLICITLY granted `cmd/g0router/main.go` ("OWNERSHIP GRANT") to close the
  cycle-1 BLOCKER the gate itself raised (shutdown flush unreachable from
  production). The gate was not shown the fix plan.
- B-MAJOR "server.go admin construction changed": FALSE POSITIVE — range pollution:
  `NewAdminHandlers(st, usageDeps)` was committed by `2530523` (phase-1/w5-d fix-r1,
  shared-instance plumbing), gated under w5-d. It appears here only because the
  w5-f range spans the interleaved w5-d fix commit (blame-provable).

## REAL → FIX

### Fix 1 (A-MAJOR) — passthrough valid-usage accumulation untested
Only `TestPassthroughStreamEstimatesOnFinish` exists. ADD
`TestPassthroughSummaryUsage`: ProcessPassthroughStream over chunks carrying REAL
provider usage → summary.Usage = extracted (not estimated), no `estimated` flag.

### Fix 2 (A-MAJOR) — isArrayish dead stub
`usage_tracking.go:355-359` always returns false with a "future changes" comment.
DELETE the function and its call site(s); dead defensive code violates repo
conventions.

### Fix 3 (B-MAJOR) — glue drops Claude-format token keys
`usage_glue.go:249-253` reads only `prompt_tokens`/`completion_tokens` from
summary.Usage; Claude-shaped usage (input_tokens/output_tokens) loses counts. FIX:
read via synonym fallback exactly like the w5-a normalizer (prompt_tokens ||
input_tokens; completion_tokens || output_tokens — `usageRepo.js:121-122`); also
persist the raw key set into entry.Tokens unchanged. Test FIRST:
`TestRecordStreamClaudeUsageKeys` — summary.Usage with input_tokens/output_tokens →
recorded entry has non-zero PromptTokens/CompletionTokens; run failing → fix.

### Fix 4 (B-MAJOR) — silently discarded Record/Save errors
Six `_ = g.recorder.Record(...)` / `_ = g.detail.Save(...)` sites. The ref
fire-and-forgets but LOGS failures (`usageRepo.js:284-285` console.error "Failed to
save usage stats"). FIX: keep non-blocking semantics (a failed usage write must not
fail the client request — ref parity) but LOG the error (match the api package's
existing `log.Printf` usage, e.g. chat.go stream-error logging). Add one test
asserting a failing fake recorder does NOT fail the request (status still 200).

## Ownership
`internal/translation/usage_tracking.go`(+test), `internal/translation/stream_test.go`,
`internal/api/usage_glue.go`(+test). NOTHING else. A concurrent job owns
internal/usage/stats.go + internal/store/requestlog.go — never touch; ABSOLUTE
PROHIBITION on git checkout/restore/stash of any unowned path; index.lock retry
5×10s; package-scoped verification if the tree is mid-edit.

## Binary acceptance
- `go build ./internal/translation/... ./internal/api/...` + vet green; `go test ./internal/translation/ ./internal/api/` green; `go test -race` same packages green.
- `grep -c 'isArrayish' internal/translation/usage_tracking.go` → 0.
- `grep -c 'input_tokens' internal/api/usage_glue.go` ≥ 1.
- TestPassthroughSummaryUsage, TestRecordStreamClaudeUsageKeys pass.
