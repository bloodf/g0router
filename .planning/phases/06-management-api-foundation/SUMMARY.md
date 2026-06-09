# Phase 6: Management API Foundation — Summary

**Status:** Complete ✅
**Completed:** 2026-06-09
**Phase:** 06 — Management API Foundation

---

## What Was Built

The admin API foundation: SQLite persistence with encrypted secrets, session auth, settings/provider/connection CRUD, and a PKCE OAuth flow (Anthropic) — all behind a session-protected `/api/*` surface with `{data, error}` snake_case envelopes.

### Commits

| Commit | Subject |
|---|---|
| `cbad6f80` | phase-06/task-1: SQLite store with WAL, additive migrations, AES-GCM cipher, and repositories (users, sessions, settings, providers, connections, oauth sessions) |
| `e1a4a869` | phase-06/task-2: auth package with PBKDF2 password hashing, session manager, and PKCE OAuth flow (Anthropic) |
| `31d0f43c` | phase-06/task-3: admin handlers — login/logout/me, settings, provider CRUD, connection CRUD with masked secrets, OAuth start/callback/refresh |
| `35618972` | phase-06/task-4: wire admin routes into fasthttp server; main.go opens store, seeds default admin, serves management API |
| `7c5d4a82` | phase-06/task-5: end-to-end management API integration test (login, settings, provider/connection CRUD, encrypted secrets, OAuth flow, logout) |

### Deliverables

1. **internal/store/** — `Open` (WAL + foreign keys, single-conn serialization), additive-only `ensureColumn` migrations, `LoadOrCreateSecret` (32-byte key in `<datadir>/secret.key`, 0600), AES-256-GCM `Cipher`, and repositories: users, sessions, settings, providers, connections (`secret_enc`/`access_token_enc`/`refresh_token_enc`), oauth_sessions (`verifier_enc`). Driver: `modernc.org/sqlite` (pure Go, single-binary friendly).
2. **internal/auth/** — PBKDF2-SHA256 password hashing (210k iterations, salted, constant-time compare), `Sessions` manager (login/validate/logout/seed), `OAuthFlow` with authorization-code + PKCE S256, single-use signed state persisted server-side with 10-minute expiry, token exchange + refresh.
3. **internal/admin/** — fasthttp handlers with `{data, error}` envelope: `Login`/`Logout`/`Me`, `GetSettings`/`PutSettings`, provider CRUD, connection CRUD (secrets masked in responses via `*_set` booleans; empty fields on update preserve stored secrets), `OAuthStart`/`OAuthCallback`/`RefreshConnection`, and `RequireSession` middleware (Bearer header or `g0_session` cookie).
4. **internal/server/routes_admin.go** — admin route registration; everything except `POST /api/auth/login` is session-protected.
5. **cmd/g0router/main.go** — opens the store under `~/.g0router` (override: `G0ROUTER_DATA`), seeds default admin (`admin`/`123456`) on first run only, serves the management API alongside the OpenAI surface.

### Routes

| Method | Path | Auth |
|---|---|---|
| POST | /api/auth/login | public |
| POST | /api/auth/logout | session |
| GET | /api/auth/me | session |
| GET/PUT | /api/settings | session |
| GET/POST | /api/providers | session |
| PUT/DELETE | /api/providers/{id} | session |
| GET/POST | /api/connections | session |
| PUT/DELETE | /api/connections/{id} | session |
| POST | /api/connections/{id}/refresh | session |
| GET | /api/oauth/{provider}/start | session |
| POST | /api/oauth/{provider}/callback | session |

### Quality Gates

- `go test ./...` ✅ PASS (30 packages)
- `go vet ./...` ✅ PASS
- `go build ./...` ✅ PASS
- Manual smoke: real binary — login via curl, provider + connection created, unauthenticated request 401 ✅

### Verification Criteria (from PLAN.md)

1. Login endpoint issues a valid session token ✅ (`TestLoginSuccessAndEnvelope`, integration test)
2. Admin routes reject unauthenticated requests ✅ (`TestAdminRoutesRegistered`, `TestRequireSession`)
3. Provider CRUD persists to SQLite ✅ (`TestProviderCRUD`, `TestProviderCRUDHandlers`)
4. Connection CRUD encrypts secrets ✅ (`TestConnectionCRUDEncryptsSecrets` — raw `*_enc` columns checked)
5. OAuth callback stores tokens and refresh token ✅ (`TestOAuthStartCallbackRefresh`, integration test)

## Deviations

- **Route paths:** PLAN.md listed `/api/login`, `/api/logout`, `/api/me`; implemented as `/api/auth/login`, `/api/auth/logout`, `/api/auth/me` per the phase execution brief. Dashboard (Phase 7) should target the `/api/auth/*` paths.
- **No env-based default password:** PLAN.md mentioned "environment-based default password", but AGENTS.md forbids secrets via env vars. Seeded `admin`/`123456` on first run only (matches the v1 precedent in WORKFLOW.md Wave UI-2). Password change endpoint deferred to Phase 7.
- **Encryption key:** PLAN risk table referenced `API_KEY_SECRET` env. Instead the key is auto-generated and persisted at `<datadir>/secret.key` (0600) — no env var, no startup failure mode.
- **Handlers live in `internal/admin/`** (per PLAN scope), not `internal/api/` — the OpenAI-compatible surface stays in `internal/api/`.

## Left for Phase 7+

- Dashboard UI consuming these endpoints (Phase 7).
- Password change endpoint + suggested-models endpoint for providers (PLAN mentioned "suggested models"; no backing data exists until the catalog work).
- Wiring connection secrets into the inference router key store (Phase 8).
- Session garbage collection scheduling (store has `DeleteExpiredSessions`; nothing calls it periodically yet).

## Self-Check

- [x] All tasks executed (TDD: RED → GREEN per task)
- [x] Each task committed individually
- [x] Tests pass
- [x] Build passes
- [x] No regressions

---

*End of summary*
