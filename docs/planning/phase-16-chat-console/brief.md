# Brief

**Problem:** g0router has no persistence for chat transcripts and no live console log stream. Operators cannot save/replay chat sessions (with base64 image attachments) nor watch structured logs in real time.

**Success criteria:**
- SQLite-backed `chat_sessions` CRUD; list excludes message bodies, GET `/api/chat-sessions/:id` returns full transcript.
- `internal/console/` domain owns 1000-entry ring buffer + `slog.Handler` tee + subscriber broker (mirrors `internal/traffic`, incl. nil-broker guard); SSE replays then streams live; clear endpoint audited.
- Validation enforced: `messages_json` ≤2MB; images base64-only, ≤4/message, ≤5MB each, mime png/jpeg/webp/gif; reject with 400.
- Per-phase gate green, coverage ≥95.0%.

**Non-goals:**
- No UI work (history panel, auto-scroll are phases 20-21).
- No multipart image endpoint (client-side base64 only).
- No chat inference (transcripts only; inference stays on `/v1/chat/completions`).

**Constraints:** snake_case JSON, `{data,error}` envelope, mutating endpoints audited; DDD-lite (handlers transport-only, console domain owns logic, store persistence-only, no fasthttp in domain); no global state; direct push to main.

**Verification:** `go test ./... -count=1` + `go vet` + `go build` + `go test -race` green with coverage ≥95.0%, plus manual curl smoke of new endpoints.

**QA criteria:**
```yaml
qa_skip: null
scenarios:
  - method: api
    name: chat sessions CRUD round-trip; list omits messages; updated_at changes on PUT
  - method: api
    name: message limits — oversized messages_json / image / wrong mime rejected with 400
  - method: manual_smoke
    name: console SSE replays ring buffer then streams live; clear empties ring
```

**Linked artifacts:** architect-plan: ./architect-plan.md; orchestration: ./orchestration.jsonl
