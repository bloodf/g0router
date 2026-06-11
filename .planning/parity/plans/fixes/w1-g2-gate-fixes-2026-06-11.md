# Fix micro-plan — w1-g2 diff-gate finding (2026-06-11)

Author: Fable 5. Implementer: kimi. Authorizing artifact:
`artifacts/w1-g2-responses-endpoint-diff-scoped-gpt.txt`. One real test gap.

## Task 1 — test terminal detection by `type` field (BLOCKER, real, test-only)

`responses_stream_helpers.go isResponsesTerminalEvent` correctly checks the
`event` field, the `type` field, AND `response.status` (verified — impl is
complete and ref-faithful to responsesStreamHelpers.js:18-23). But
`TestIsResponsesTerminalEvent` only exercises the `event`-field path. Add cases:
- `{"type":"response.completed"}` → true
- `{"type":"response.failed"}` → true
- `{"response":{"status":"completed"}}` → true; `{"response":{"status":"failed"}}` → true
- a non-terminal `{"type":"response.output_text.delta"}` → false
Keep the existing `event`-field cases. No production change (impl already correct).

## Acceptance (binary)

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `TestIsResponsesTerminalEvent` covers the `type`-field and `response.status` paths (covered by go test).
- Files touched ONLY: `responses_stream_helpers_test.go`. Do NOT git commit.

## Out of scope

Everything else (gate-clean — handler, flush, route all pass).
