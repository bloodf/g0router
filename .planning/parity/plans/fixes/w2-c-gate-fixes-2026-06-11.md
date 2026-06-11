# Fix micro-plan — w2-c diff-gate findings (2026-06-11)

Author: Fable 5. Implementer: kimi. Authorizing artifact:
`artifacts/w2-c-ollama-adapter-diff-scoped-gpt.txt`. One real; one accepted (plan amendment).

## Task 1 — TestOllamaStreamMalformedInBandError must test malformed NDJSON (BLOCKER, real)

The test was implemented as a post-hook-failure test, not malformed-NDJSON. Fix it
to actually feed a malformed NDJSON line: an httptest server emits a valid ollama
NDJSON line then a broken line (e.g. `{not json`); assert the stream emits the valid
chunk then an in-band `streamError` and closes. (If a separate post-hook test is
desired, add it as `TestOllamaStreamPostHookError` — but the malformed-named test
must test malformed input.)

## Plan amendment — stubs_test.go is an accepted ownership addition (MAJOR #2)

The diff added `internal/providers/ollama/stubs_test.go` (Task-3 stub tests),
which was not in the original ownership list (`chat_test.go`/`provider_test.go`).
This is a sensible split (stub tests belong with stubs.go) and is hereby ADDED to
w2-c's owned files. No action needed beyond this amendment; keep stubs_test.go.

## Acceptance (binary)

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `TestOllamaStreamMalformedInBandError` feeds a malformed NDJSON line and asserts an in-band streamError (covered by go test; grep the test body for a broken-JSON literal).
- Files touched ONLY: `internal/providers/ollama/chat_test.go` (+ optionally stubs_test.go). Do NOT git commit.

## Out of scope

Any production change (the adapter is correct; this is test-fidelity only).
