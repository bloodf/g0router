# w5-f fix micro-plan — diff-gate round 1, both halves (Fable 5, 2026-06-12)

Sources: `artifacts/w5-f-plan-A-diff-scoped-gpt.txt` (translation half) +
`artifacts/w5-f-plan-B-diff-scoped-gpt.txt` (api/server half), cycle 1.

## REBUTTED — no change (recorded with ref lines)
- A-BLOCKER "Ollama extraction is scope creep": FALSE POSITIVE — the ref's
  extractUsage CONTAINS the Ollama done=true branch inside the cited evidence range
  (`usageTracking.js:226-233` "Ollama NDJSON format"); the plan cited `:172-238`.
- A-MAJOR "AddBufferToUsage must recompute total": FALSE POSITIVE — ref `:46-51`
  INCREMENTS an existing total (`result.total_tokens += BUFFER_TOKENS`); recompute
  happens only when total is absent. Port is ref-verbatim.
- A-MAJOR "Responses normalize vs filter mismatch" + A-MAJOR "Gemini estimates lose
  token fields": FALSE POSITIVES — the mismatch is the REF'S OWN behavior:
  extractUsage normalizes Responses usage into prompt_tokens/completion_tokens
  (`:189-196`), formatUsage emits OpenAI shape for everything non-Claude (comment
  `:283` "works for openai, gemini, responses, etc."), and filterUsageForFormat
  keeps only format-NATIVE keys (`:71-88`) — so estimated finish-chunks for
  gemini/responses clients carry only the `estimated` flag in 9router too. Parity
  ports quirks (recorded program rule).
- B-BLOCKER "usage_adapters.go outside ownership": FALSE POSITIVE-BY-ORGANIZATION —
  the plan's server ownership is WIRING; a same-package sibling file holding exactly
  that wiring is file organization, not scope (precedent: w5-pre's comboDispatcher
  adapter, same package). The map's ownership note binds the concurrent phase; no
  concurrent plan owns internal/server.

## REAL → FIX

### Fix 1 (B-BLOCKER) — shutdown flush unreachable from production
`cmd/g0router/main.go:124` calls `server.New(...)` + ListenAndServe; the
`NewWithShutdown`/`Server.Close()` flush path has zero production callers — the
exact dead-wiring pattern this wave keeps catching. FIX: main.go uses
`server.NewWithShutdown`; on SIGINT/SIGTERM call `srv.Close()` (graceful shutdown +
DetailWriter flush — the ref's `_shutdownHandler`, `requestDetailsRepo.js:183-200`);
keep `New()` as a thin wrapper for tests. OWNERSHIP GRANT: `cmd/g0router/main.go`
(one-screen change). Structural check: `grep -c 'NewWithShutdown'
cmd/g0router/main.go` ≥ 1.

### Fix 2 (B-MAJORs) — nominal smoke tests
`TestSmokeMessagesRequestPersistsRequestLog` / `TestSmokeEmbeddingsRequestPersists
RequestLog` assert only non-404. FIX: drive each request against a fake provider to
completion and assert `SELECT COUNT(*) FROM request_log` = 1 with endpoint
attribution ("/v1/messages", "/v1/embeddings") and non-empty provider/model — same
shape as the chat smoke.

### Fix 3 (B-MAJOR) — TestChatCapturesRequestDetail gaps
FIX: extend to assert (a) the ERROR path also captures a detail (provider error →
capture with status != success, per `streamingHandler.js:87`/`chatCore.js:196`) and
(b) captured request headers are sanitized (Authorization absent — SanitizeHeaders
applied before persistence).

### Fix 4 (B-MAJOR) — record-glue copy-paste across 4 handlers
FIX: extract the duplicated recordError/recordNonStream/recordStream blocks into
shared unexported helpers in `internal/api/usage_glue.go` parameterized by endpoint
path; handlers call the helpers. No behavior change; existing tests stay green.

## Ownership
`cmd/g0router/main.go`, `internal/server/usage_smoke_test.go`,
`internal/api/usage_glue.go`(+`usage_glue_test.go`),
`internal/api/{chat,messages,responses,embeddings}.go` (helper-call replacement
only).

## Binary acceptance
- `go build ./... && go vet ./... && go test ./...` green; `go test -race ./internal/api/ ./internal/server/` green.
- `grep -c 'NewWithShutdown' cmd/g0router/main.go` ≥ 1.
- Strengthened smoke tests assert request_log rows + endpoint attribution; chat
  detail test asserts error-path + sanitized headers.
