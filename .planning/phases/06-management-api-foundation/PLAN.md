# Phase 6: Management API Foundation

**Phase:** 06  
**Goal:** Build the admin API foundation: auth, settings, providers, and connections.  
**Requirements:** MGMT-01..04  
**Estimated duration:** 4–5 days  
**Wave:** 2 — Core Providers + Admin

---

## Why

The dashboard cannot function without management APIs. This phase establishes auth, persistence, and the core CRUD endpoints.

---

## Scope

### In scope
- `internal/store/` — SQLite setup, migrations, query helpers.
- `internal/auth/` — session management, password hashing, API key auth, OAuth start/callback/refresh.
- `internal/admin/auth.go` — `/api/login`, `/api/logout`, `/api/me`.
- `internal/admin/settings.go` — `/api/settings` GET/PUT.
- `internal/admin/providers.go` — `/api/providers` CRUD + suggested models.
- `internal/admin/connections.go` — `/api/connections` CRUD.
- `internal/server/routes_admin.go` — admin route registration.
- Encrypted secret storage (`*_enc` columns).
- Initial user seeding with environment-based default password.

### Out of scope
- Virtual keys (Phase 8).
- API key management for endpoint consumers (Phase 8).
- Dashboard UI (Phase 7).

---

## Verification

### Tests
1. Login endpoint issues a valid session/token.
2. Admin routes reject unauthenticated requests.
3. Provider CRUD persists to SQLite.
4. Connection CRUD encrypts secrets.
5. OAuth callback stores tokens and refresh token.

### Manual verification
1. `curl` login and use token to create a provider.
2. Create a connection with an API key, verify it decrypts correctly.

---

## Tasks

1. Set up SQLite with migrations.
2. Implement auth packages.
3. Implement admin handlers.
4. Wire admin routes.
5. Write integration tests.
6. Verify gates.

---

## Risks

| Risk | Mitigation |
|------|------------|
| Migration order conflicts | Keep migrations additive-only; never modify existing tables in place. |
| Encryption key missing | Fail fast on startup if `API_KEY_SECRET` / encryption key is not set. |
| OAuth state parameter issues | Use signed state cookies with expiration. |
