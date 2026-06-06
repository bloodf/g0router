# Phase 13: Auth & Core Infrastructure

> Process, contracts, gates: see `docs/phases/STAGE-13-19-PROCESS.md`.
> Security review: **mandatory** at checkpoint (auth surface).

## Goal
Password-based dashboard authentication with server-side sessions, CSRF
protection, first-run setup, and minimal user management — without breaking
existing bearer/X-API-Key auth on `/api/*`.

UI-only items from the original plan (toast system, theme, breadcrumbs, Cmd+K
search) are **Lovable's job** (see `docs/lovable-prompt.md`) — no backend work.

## Features (6 backend)
1. First-run setup (create first admin user)
2. Password login + bcrypt hashing
3. Server-side sessions + HTTP-only cookie
4. CSRF protection (SameSite=Strict + Origin check)
5. Rate-limited login (per-IP, proxy-header aware)
6. Require-login toggle (runtime setting) + minimal user management

## Auth Design (READ CAREFULLY — foundation for all later phases)

### Coexistence rule
`/api/*` auth middleware accepts **either**:
- `Authorization: Bearer <key>` / `X-API-Key` (existing control-plane auth — unchanged), **or**
- valid `g0router_session` cookie (new).

Existing CLI, tests, and the current embedded UI keep working untouched.

### Exempt routes (no auth required)
`POST /api/auth/login`, `POST /api/auth/setup`, `GET /api/auth/status`.
`/healthz` and `/metrics` are outside `/api/*` and unchanged.

### Sessions
- Server-side: `dashboard_sessions` table (below). Sessions survive restart.
- Token: 32 random bytes, hex — stored hashed (SHA-256) in DB, raw value only
  in the cookie.
- Cookie: name `g0router_session`, `HttpOnly`, `SameSite=Strict`, `Path=/`,
  `Secure` when the request arrived over TLS.
- TTL: 7 days, sliding — `last_seen_at` refreshed at most once per minute.
- Logout deletes the DB row and clears the cookie.
- Expired sessions purged opportunistically on login.

### CSRF
- `SameSite=Strict` is the primary defense.
- Additionally: for mutating methods (POST/PUT/DELETE/PATCH) authenticated via
  **cookie**, the middleware requires an `Origin` (or `Referer`) header whose
  host matches the request `Host`; otherwise `403`. Bearer-authenticated
  requests skip this check.

### `require_login` semantics
- Runtime setting (default `false` — preserves current behavior).
- When `false`: session middleware passes everything through (no auth needed
  on `/api/*` beyond what exists today).
- When `true`: every non-exempt `/api/*` request needs bearer key OR session.
- `require_login` can only be set to `true` when at least one dashboard user
  exists (`409` otherwise — prevents lockout).

### Login rate limiting
- 5 failed attempts per 15 minutes per client IP → `429` with
  `retry_after_seconds`.
- Client IP = remote addr by default. `X-Forwarded-For` (first hop) is trusted
  **only** when new setting `trust_proxy_headers=true` (needed behind
  Cloudflare/Tailscale tunnels where remote addr is constant). Document the
  tradeoff in the setting description.
- In-memory limiter (map + mutex), no table needed.

### Note on existing "admin/123456" banner
`internal/cli/root.go` auto-creates a control-plane **API key** named `admin`
with raw value `123456` on first boot. This is NOT a dashboard user. Phase 13
does not change it. First dashboard user is created via `/api/auth/setup`.

## New Database Tables
```sql
CREATE TABLE IF NOT EXISTS dashboard_users (
    id INTEGER PRIMARY KEY,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,        -- bcrypt
    display_name TEXT,
    role TEXT NOT NULL DEFAULT 'user',  -- 'admin' | 'user'
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS dashboard_sessions (
    token_hash TEXT PRIMARY KEY,        -- sha256(raw token)
    user_id INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_seen_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL,
    user_agent TEXT,
    ip TEXT
);
```

## New API Endpoints
- `GET /api/auth/status` → `{require_login, has_users, authenticated, username, display_name, role}` (public)
- `POST /api/auth/setup` — `{username, password, display_name?}`; only when zero users exist (`409` after); creates `role='admin'`; logs the user in (sets cookie)
- `POST /api/auth/login` — `{username, password}`; bcrypt verify; sets cookie; `429` when rate limited
- `POST /api/auth/logout` — deletes session, clears cookie
- `PUT /api/auth/password` — `{current_password, new_password}`; verifies current; invalidates **other** sessions of the user
- `GET /api/auth/users` — list users (admin only)
- `POST /api/auth/users` — create user (admin only)
- `DELETE /api/auth/users/:id` — delete user + their sessions (admin only; cannot delete last admin)

Password rules: min 8 chars; reject empty/whitespace-only. All mutating auth
endpoints write `audit_log` rows (never log passwords or tokens).

## New Settings Keys
- `require_login` (bool, default false)
- `trust_proxy_headers` (bool, default false)

## Tasks (TDD each; commit per task)
1. `phase-13/task-1`: store — `dashboard_users.go` CRUD + bcrypt hash/verify (`internal/store/dashboard_users_test.go` first; add `golang.org/x/crypto`)
2. `phase-13/task-2`: store — `dashboard_sessions.go` create/get/touch/delete/purge (tests first)
3. `phase-13/task-3`: handlers — setup/login/logout/status (`api/handlers/auth.go`, tests first incl. rate limiter + 429 shape)
4. `phase-13/task-4`: middleware — session validation + coexistence + exempt routes + Origin check (`api/middleware.go`, tests first)
5. `phase-13/task-5`: handlers — password change + users CRUD + last-admin guard (tests first)
6. `phase-13/task-6`: settings — `require_login`, `trust_proxy_headers` + lockout guard (tests first)
7. `phase-13/checkpoint`: security pass + WORKFLOW update + Outcome section

## Test Requirements (minimum)
- Setup succeeds once; second call `409`
- Login correct password → 200 + cookie set; wrong → 401; 6th failure in window → 429 with `retry_after_seconds`
- Session cookie grants `/api/*` access when `require_login=true`; garbage/expired cookie → 401
- Bearer key still works on `/api/*` with `require_login=true`
- Exempt routes reachable without auth
- Mutating request with cookie + mismatched Origin → 403; with bearer → passes
- Logout invalidates session (subsequent request 401)
- Password change: wrong current → 403; success invalidates other sessions
- `require_login=true` with zero users → 409
- Cannot delete last admin
- Sessions survive store reopen (restart simulation)
- XFF honored only when `trust_proxy_headers=true`

## Files to Read First
`internal/store/apikeys.go`, `api/handlers/apikeys.go`, `api/middleware.go`,
`internal/store/settings.go`, `internal/store/audit.go`

## Commit Message (final)
`phase-13/auth-core: dashboard users, sessions, csrf, login rate limit`
