# Phase 16: Chat & Console

> Process, contracts, gates, architecture: see `docs/phases/STAGE-13-19-PROCESS.md`.

## Goal
SQLite-backed chat sessions (with base64 image attachments) and live console
log streaming via a domain-owned ring buffer.

## Architecture
- `internal/console/` — new domain package: ring buffer, slog.Handler,
  subscriber fan-out. No fasthttp imports. Mirrors existing
  `internal/traffic/` broker pattern — read it first.
- Chat sessions: store repository + thin handlers (pure CRUD, no domain
  package needed).

## Features (5 backend)
1. SQLite-backed chat sessions CRUD
2. Image attachments (base64 data URLs inside `messages_json` — client-side
   encoding, **no multipart endpoint**)
3. SSE live console log streaming
4. Log clearing endpoint
5. Console log levels (DEBUG/INFO/WARN/ERROR) with color hints

UI items (history panel, auto-scroll) are Lovable's job.

## New Database Tables
```sql
CREATE TABLE IF NOT EXISTS chat_sessions (
    id INTEGER PRIMARY KEY,
    title TEXT,
    model TEXT,
    provider TEXT,
    messages_json TEXT,               -- [{role, content, images?: [dataURL]}]
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## New API Endpoints
- `GET /api/chat-sessions` — list (id, title, model, provider, updated_at — NOT messages)
- `GET /api/chat-sessions/:id` — full session incl. messages
- `POST /api/chat-sessions` — create `{title?, model, provider}`
- `PUT /api/chat-sessions/:id` — update `{title?, messages?}`
- `DELETE /api/chat-sessions/:id`
- `GET /api/console-logs/stream` — SSE; replays ring buffer then streams live
- `DELETE /api/console-logs` — clear ring buffer (audited)

## Validation / Limits
- `messages_json` ≤ 2 MB per session; images ≤ 5 MB each, ≤ 4 per message;
  data URL mime must be image/png|jpeg|webp|gif. Reject with 400 otherwise.
- Chat inference itself goes through existing `/v1/chat/completions` — this
  phase stores transcripts only.

## Console Log Implementation
- Ring buffer: 1000 entries, `{ts, level, msg, attrs}`.
- Custom `slog.Handler` wrapping the existing handler (tee) — wire into logger
  construction in CLI/server startup, no global state.
- SSE: replay buffer on connect, then live via subscriber channel; follow
  `internal/traffic` broker + `handleTrafficStream` patterns (incl. nil-broker
  guard and shutdown path — both have known coverage tests to mirror).

## Tasks
1. `phase-16/task-1`: `internal/console/` — ring buffer + slog handler + broker + tests
2. `phase-16/task-2`: store — chat_sessions repository + validation + tests
3. `phase-16/task-3`: handlers — chat sessions CRUD + tests
4. `phase-16/task-4`: handlers — console SSE stream + clear + wiring into startup + tests
5. `phase-16/checkpoint`

## Test Requirements (minimum)
- Ring buffer wraps correctly at capacity; clear empties it
- slog handler tee: entry reaches both ring and parent handler; levels mapped
- SSE replays buffered entries then live entries; client disconnect releases subscriber; shutdown closes stream cleanly; nil-broker guard
- Session CRUD round-trip; list excludes message bodies
- Oversized messages_json / image / wrong mime → 400
- updated_at changes on PUT

## Commit Message (final)
`phase-16/chat-console: chat sessions, console ring buffer, sse logs`
