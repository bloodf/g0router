# Architect Plan

Canonical spec: [`docs/phases/phase-14-providers-testing.md`](../../phases/phase-14-providers-testing.md)

## Summary

- New `proxy_pools` table: id, name, protocol (`http`|`https`|`socks5`), host, port, username, `password_enc` (encrypted, NEVER plaintext — reuse oauthsessions.go pattern), is_active, last_check_at/status.
- New `disabled_models` and `custom_models` tables, each `UNIQUE(provider, model)`; custom_models adds `display_name`.
- `connections` gains `proxy_pool_id INTEGER` via `ensureColumn` (additive).
- `DELETE /api/models/disabled` takes `{provider, model}` in the request body (composite key, no `:id` path param); custom uses `DELETE /api/models/custom/:id`.
- fasthttp proxy caveat: provider adapters use fasthttp clients in `internal/providers/utils` — read actual client construction first; wire `http.Transport.Proxy`/SOCKS5 dialer or the fasthttp dialer equivalent; record the real approach in the phase `## Outcome`.
- Proxy-pool CRUD/test/batch responses never include the password; batch import parses `host:port` and `socks5://user:pass@host:port`, rejecting garbage with per-line errors.
- Model test endpoints return `{ok, latency_ms, error}` and never 500 on upstream failure; batch test streams SSE one event per connection plus a terminal `done` event.
- Disabled models filtered from `/v1/models`, `/api/models`, and routing candidate sets; custom models appended to provider model lists.
- Audit rows for every mutating endpoint (pool create/update/delete, proxy assignment, model disable/custom).
- Task list:
  - task-1: store — proxy_pools (encrypted password) + tests
  - task-2: store — disabled_models + custom_models + tests
  - task-3: handlers — proxy pools CRUD/test/batch + tests
  - task-4: proxy wiring into provider clients + tests
  - task-5: handlers — provider detail/connections/suggested + tests
  - task-6: handlers — model test single/batch SSE + tests
  - task-7: disabled/custom model filtering in listing + routing + tests
  - checkpoint: per-phase gate + WORKFLOW + Outcome
