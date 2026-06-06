# Architect Plan: Auth & Core Infrastructure

Canonical spec: [`docs/phases/phase-13-auth-core-infrastructure.md`](../../phases/phase-13-auth-core-infrastructure.md)

## Summary
- New table `dashboard_users` (bcrypt `password_hash`, unique `username`, `role` admin|user) — distinct from control-plane API keys; first user created via `/api/auth/setup` as `admin`.
- New table `dashboard_sessions` keyed by `token_hash` (SHA-256 of a 32-byte random hex token; raw value only in the cookie), with `expires_at`, sliding 7-day TTL, `last_seen_at` refreshed at most once/minute, opportunistic purge on login.
- Session cookie `g0router_session`: `HttpOnly`, `SameSite=Strict`, `Path=/`, `Secure` only over TLS. Sessions persist across restart (server-side, DB-backed).
- CSRF: `SameSite=Strict` is primary; mutating cookie-authed requests (POST/PUT/DELETE/PATCH) additionally require an `Origin`/`Referer` host matching `Host` → else `403`. Bearer-authed requests skip the check.
- Coexistence: `/api/*` middleware accepts existing `Authorization: Bearer`/`X-API-Key` OR a valid session cookie. Exempt routes: `POST /api/auth/login`, `POST /api/auth/setup`, `GET /api/auth/status`. `/healthz`, `/metrics` unchanged.
- `require_login` runtime setting (default false → pass-through preserves today's behavior; true → every non-exempt `/api/*` needs bearer or session). Settable true only when ≥1 dashboard user exists, else `409` (lockout guard).
- Login rate limit: in-memory map+mutex, 5 fails / 15 min / client IP → `429` with `retry_after_seconds`. Client IP = remote addr unless new `trust_proxy_headers=true`, which trusts `X-Forwarded-For` first hop (documented tunnel tradeoff).
- Endpoints: `GET /api/auth/status` (public), `POST /api/auth/setup`, `POST /api/auth/login`, `POST /api/auth/logout`, `PUT /api/auth/password` (invalidates other sessions), `GET/POST /api/auth/users`, `DELETE /api/auth/users/:id` (admin-only; cannot delete last admin). Password rules: min 8 chars, reject empty/whitespace.
- Tasks: (1) store `dashboard_users.go` CRUD + bcrypt; (2) store `dashboard_sessions.go` create/get/touch/delete/purge; (3) handlers setup/login/logout/status + rate limiter; (4) middleware session validation + coexistence + exempt + Origin check; (5) handlers password change + users CRUD + last-admin guard; (6) settings `require_login`/`trust_proxy_headers` + lockout guard; (7) checkpoint security pass + WORKFLOW + Outcome.

## Security notes
- Session tokens are SHA-256-hashed at rest (raw only in cookie); passwords bcrypt-hashed. Never log passwords or tokens — every mutating auth endpoint writes an `audit_log` row (actor/action/target/details only).
- Rate limiting on login prevents brute force; `429` carries `retry_after_seconds`. `X-Forwarded-For` is trusted only when `trust_proxy_headers=true` to avoid spoofed-IP limiter bypass behind direct connections.
- Lockout guard: `require_login=true` rejected with `409` unless a dashboard user exists; `DELETE /api/auth/users/:id` cannot remove the last admin.
- CSRF: `SameSite=Strict` + Origin/Referer host match on cookie-authed mutations; bearer path is exempt by design to keep CLI/control-plane clients working.
- Security review is mandatory for this phase (process doc §7): verify input validation, authn/authz on every new route, secrets at rest, secrets-in-logs, and documented privilege requirements before checkpoint.
