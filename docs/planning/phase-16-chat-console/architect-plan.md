# Architect Plan — Phase 16 Chat & Console

Canonical spec: [`docs/phases/phase-16-chat-console.md`](../../phases/phase-16-chat-console.md)

## Summary

- New `internal/console/` domain package: 1000-entry ring buffer of `{ts, level, msg, attrs}`, wrapping `slog.Handler` (tee to ring + parent handler, levels DEBUG/INFO/WARN/ERROR mapped with color hints), and a subscriber broker — all mirroring `internal/traffic/` (read it first); no fasthttp imports.
- Broker mirrors traffic broker: replay-on-connect then live fan-out, nil-broker guard, clean shutdown closing subscriber streams — both have known coverage tests to mirror.
- Wire the console slog tee into logger construction at CLI/server startup via constructor injection; no global state.
- `chat_sessions` table (additive migration in `internal/store/sqlite.go`): `id, title, model, provider, messages_json, created_at, updated_at`.
- Store repo owns persistence + validation: `messages_json` ≤2MB; images base64 data URLs only, ≤4 per message, ≤5MB each, mime image/png|jpeg|webp|gif; reject otherwise → handler returns 400.
- Endpoints: `GET /api/chat-sessions` (no message bodies), `GET /api/chat-sessions/:id` (full transcript), `POST`/`PUT`/`DELETE`; `GET /api/console-logs/stream` (SSE), `DELETE /api/console-logs` (audited clear).
- Snake_case + `{data,error}` envelope on all JSON; SSE exempt. `updated_at` bumps on PUT.
- Task list: task-1 console domain; task-2 store repo+validation; task-3 chat handlers; task-4 console SSE/clear handlers + startup wiring; checkpoint.

## Layer notes (DDD-lite)

- `internal/console/` owns ring/broker/handler business logic; pure Go, no fasthttp; defines its own narrow repo-style interfaces where needed.
- `internal/store/` holds `chat_sessions` repository + validation helpers; persistence only, one file per aggregate.
- `api/handlers/` is transport-only: parse, validate boundary, envelope, status codes, audit on mutations; delegates to console domain and store. Dependency direction strictly inward.
