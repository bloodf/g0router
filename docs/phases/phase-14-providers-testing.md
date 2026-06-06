# Phase 14: Providers & Testing

> Process, contracts, gates: see `docs/phases/STAGE-13-19-PROCESS.md`.

## Goal
Provider detail APIs, model testing (single + batch), proxy pools with
encrypted credentials, and disabled/custom model management.

## Features (9)
1. Provider detail API
2. Individual model testing
3. Batch provider testing (SSE progress)
4. Proxy pools CRUD (credentials encrypted at rest)
5. Proxy pool assignment to connections
6. Suggested models fetch
7. Disabled models toggle
8. Custom model addition
9. Provider health inline

## New Database Tables
```sql
CREATE TABLE IF NOT EXISTS proxy_pools (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    protocol TEXT NOT NULL,           -- 'http' | 'https' | 'socks5'
    host TEXT NOT NULL,
    port INTEGER NOT NULL,
    username TEXT,
    password_enc TEXT,                -- encrypted (oauthsessions.go pattern), NEVER plaintext
    is_active INTEGER DEFAULT 1,
    last_check_at DATETIME,
    last_check_status TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS disabled_models (
    id INTEGER PRIMARY KEY,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(provider, model)
);

CREATE TABLE IF NOT EXISTS custom_models (
    id INTEGER PRIMARY KEY,
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    display_name TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(provider, model)
);
```
`connections` gains `proxy_pool_id INTEGER` via `ensureColumn`.

## New API Endpoints
- `GET /api/providers/:id` — full provider details (matrix info + connection counts + health)
- `GET /api/providers/:id/connections` — provider connections
- `POST /api/providers/:id/models/:model/test` — live test request through model; returns `{ok, latency_ms, error}`
- `POST /api/providers/test-batch` — SSE stream of per-connection results `{provider, connection_id, ok, latency_ms, error}`
- `GET /api/providers/:id/suggested-models` — fetch model list from provider API
- `GET/POST /api/proxy-pools`, `PUT/DELETE /api/proxy-pools/:id` — CRUD (responses never include password)
- `POST /api/proxy-pools/:id/test` — connectivity check via the proxy
- `POST /api/proxy-pools/batch` — batch import (`{lines: ["host:port", "socks5://user:pass@host:port", ...]}`)
- `PUT /api/connections/:id/proxy` — `{proxy_pool_id|null}` assign/clear
- `GET /api/models/disabled` — list
- `POST /api/models/disabled` — `{provider, model}` disable
- `DELETE /api/models/disabled` — `{provider, model}` in body (composite key, no `:id`)
- `POST /api/models/custom` — `{provider, model, display_name?}` add
- `DELETE /api/models/custom/:id`

## Proxy Engine Integration
- Wire `http.Transport.Proxy` / SOCKS5 dialer from the connection's assigned
  proxy pool into outbound provider HTTP clients.
- **Caveat**: provider adapters use fasthttp clients in places — read the
  actual client construction in `internal/providers/utils` first; if fasthttp,
  use its proxy dialer equivalents. Note actual approach in `## Outcome`.
- Disabled models filtered out of `/v1/models`, `/api/models`, and routing
  candidate sets. Custom models appended to provider model lists.

## Tasks
1. `phase-14/task-1`: store — proxy_pools (encrypted password) + tests
2. `phase-14/task-2`: store — disabled_models + custom_models + tests
3. `phase-14/task-3`: handlers — proxy pools CRUD/test/batch + tests
4. `phase-14/task-4`: proxy wiring into provider clients + tests
5. `phase-14/task-5`: handlers — provider detail/connections/suggested + tests
6. `phase-14/task-6`: handlers — model test single/batch SSE + tests
7. `phase-14/task-7`: disabled/custom model filtering in model listing + routing + tests
8. `phase-14/checkpoint`

## Test Requirements (minimum)
- Proxy password round-trips encrypted; list/get responses omit it
- Batch import parses `host:port` and URL formats; rejects garbage lines with per-line errors
- Outbound request uses assigned proxy (fake upstream + fake proxy listener)
- Disabled model absent from `/v1/models` and rejected by routing with clear error
- Custom model appears in listings
- Model test returns `{ok:false, error}` on upstream failure — never 500
- Batch test SSE emits one event per connection + terminal `done` event
- Audit rows for pool create/update/delete and proxy assignment

## Commit Message (final)
`phase-14/providers-testing: detail api, model tests, proxy pools, model mgmt`
