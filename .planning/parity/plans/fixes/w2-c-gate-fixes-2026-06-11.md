# Fix micro-plan — w2-c diff-gate findings (2026-06-11)

Author: Fable 5. Implementer: kimi. Authorizing artifact:
`artifacts/w2-c-ollama-adapter-diff-scoped-gpt.txt`. One real; one accepted (plan amendment).

## Task 1 — test malformed NDJSON is SKIPPED (CORRECTED — was a planner error)

PLANNER CORRECTION: the original spec assumed malformed NDJSON surfaces an in-band
`streamError`. It does NOT — and that is CORRECT, ref-faithful behavior. The NDJSON
scanner `utils.SSEScanner.parseLine` (`internal/providers/utils/sse.go:71-78`, w1-c)
SKIPS any line failing `json.Valid` (returns `("", false)`), so a malformed line
never reaches the translator/Unmarshal — exactly the SSE warn+skip design
(PAR-TRANS-047). The kimi worker correctly IMPL-BLOCKED rather than hack production.
Fix the TEST to assert the real behavior: rename `TestOllamaStreamMalformedInBandError`
→ `TestOllamaStreamSkipsMalformedNDJSON`. An httptest server emits a valid ollama
NDJSON line, then a malformed line (`{not json`), then the `done` line; assert the
valid chunk IS emitted, the malformed line is SILENTLY SKIPPED (no error chunk), and
the stream completes normally. NO production change (the scanner+adapter are correct).

## Plan amendment — stubs_test.go is an accepted ownership addition (MAJOR #2)

The diff added `internal/providers/ollama/stubs_test.go` (Task-3 stub tests),
which was not in the original ownership list (`chat_test.go`/`provider_test.go`).
This is a sensible split (stub tests belong with stubs.go) and is hereby ADDED to
w2-c's owned files. No action needed beyond this amendment; keep stubs_test.go.

## Acceptance (binary)

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `TestOllamaStreamSkipsMalformedNDJSON` feeds a malformed NDJSON line and asserts it is skipped (valid chunk emitted, NO error chunk, stream completes) — matching the scanner's `json.Valid` skip (sse.go:71-78).
- Files touched ONLY: `internal/providers/ollama/chat_test.go` (+ optionally stubs_test.go). Do NOT git commit.

## Out of scope

Any production change (the adapter is correct; this is test-fidelity only).
