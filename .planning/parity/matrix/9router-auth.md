# AUTH parity matrix: 9router → g0router

Reference: `/Users/heitor/Developer/github.com/bloodf/_refs/9router` (SHA `827e5c3`)

## Behavior rows

| ID | Behavior | Evidence (file:line) | g0router status | Notes |
|---|---|---|---|---|
| PAR-AUTH-001 | Dashboard password login (bcrypt verify) | `src/app/api/auth/login/route.js:45-51` | HAVE | g0router uses PBKDF2-SHA256 at 210k iterations (`internal/auth/password.go:14-36`). 9router uses `bcryptjs.compare`. |
| PAR-AUTH-002 | Default password fallback when no hash stored | `src/app/api/auth/login/route.js:49-50` | HAVE | 9router falls back to `INITIAL_PASSWORD` env or `"123456"`. g0router seeds one admin on first run via `auth.SeedAdmin` (`internal/auth/session.go:32-48`). |
| PAR-AUTH-003 | JWT dashboard session (HS256, 24h TTL) | `src/lib/auth/dashboardSession.js:28-33` | MISSING | 9router uses Jose `SignJWT` with `HS256`. g0router uses opaque random hex tokens stored in SQLite (`internal/auth/session.go:63-68`) with 7-day TTL. |
| PAR-AUTH-004 | Cookie flags: httpOnly, secure, sameSite=lax | `src/lib/auth/dashboardSession.js:58-63` | HAVE | Both set `httpOnly` and `SameSite=Lax`. 9router toggles `secure` via `x-forwarded-proto` or `AUTH_COOKIE_SECURE` env (`dashboardSession.js:21-26`). g0router hardcodes no secure flag (`internal/admin/auth.go:61-67`). |
| PAR-AUTH-005 | OIDC dashboard login with PKCE | `src/lib/auth/oidc.js:74-78`, `src/app/api/auth/oidc/start/route.js:24-46`, `src/app/api/auth/oidc/callback/route.js:20-86` | HAVE | 9router supports full OIDC flow for dashboard auth. g0router has OAuth only for provider connections (Anthropic PKCE) with no OIDC dashboard integration. |
| PAR-AUTH-006 | Login rate limiter (progressive lockout) | `src/lib/auth/loginLimiter.js:3-5` | HAVE | 9router locks after 5 fails with steps 30s, 2m, 10m, 30m. g0router has no rate limiting. |
| PAR-AUTH-007 | Centralized dashboard guard middleware with path lists | `src/dashboardGuard.js:22-65`, `src/dashboardGuard.js:165-241` | HAVE | 9router uses a single `proxy()` function with `PUBLIC_API_PATHS`, `PUBLIC_PREFIXES`, `ALWAYS_PROTECTED`, `PROTECTED_API_PATHS`, `LOCAL_ONLY_PATHS`. g0router wraps each admin route individually with `RequireSession` (`internal/server/routes_admin.go:31-50`). |
| PAR-AUTH-008 | Public LLM API access from loopback without key | `src/dashboardGuard.js:35`, `src/dashboardGuard.js:102-104`, `src/dashboardGuard.js:118-122` | HAVE | 9router allows `/v1` from `localhost` / `127.0.0.1` without API key. g0router `/v1` routes are public with no origin check (`internal/server/routes_openai.go:15-18`). |
| PAR-AUTH-009 | API key validation for remote LLM API access | `src/dashboardGuard.js:106-116` | HAVE | 9router checks `Bearer` or `x-api-key` against SQLite (`src/lib/db/repos/apiKeysRepo.js:70-75`). g0router inference router does not validate API keys (`internal/inference/router.go:33-54`). |
| PAR-AUTH-010 | API key format with machineId and CRC8 | `src/shared/utils/apiKey.js:34-38` | HAVE | 9router generates `sk-{machineId}-{keyId}-{crc8}` using HMAC-SHA256. g0router has no API key system. |
| PAR-AUTH-011 | Local-only route gate (loopback host + origin check) | `src/dashboardGuard.js:69-81`, `src/dashboardGuard.js:91-100`, `src/dashboardGuard.js:124-129`, `src/dashboardGuard.js:169-173` | HAVE | 9router requires loopback `Host` and `Origin` for routes like `/api/cli-tools/*`, `/api/mcp/*`, `/api/tunnel/*`. g0router has no local-only restrictions. |
| PAR-AUTH-012 | CLI token auth (machineId-based) | `src/dashboardGuard.js:6-19`, `src/dashboardGuard.js:177` | HAVE | 9router accepts `x-9r-cli-token` derived from `getConsistentMachineId`. g0router has no CLI token mechanism. |
| PAR-AUTH-013 | Tunnel dashboard access toggle | `src/dashboardGuard.js:197-214`, `src/app/api/settings/require-login/route.js` | HAVE | 9router redirects tunnel hosts to `/login` when `tunnelDashboardAccess` is false. g0router has no tunnel awareness. |
| PAR-AUTH-014 | Password reset to default via CLI | `cli/src/cli/menus/settings.js:177-204` | HAVE | 9router CLI deletes the password hash from `db.json`. g0router has no password reset endpoint or CLI tool. |
| PAR-AUTH-015 | Auth mode switch (password / oidc / both) | `src/app/api/auth/login/route.js:40-41`, `src/app/api/auth/status/route.js:13` | HAVE | 9router stores `authMode` in settings. g0router supports only password dashboard auth. |
| PAR-AUTH-016 | Secret storage at rest (encryption) | `internal/store/crypto.go:12-31`, `internal/store/connections.go:111-124`, `internal/store/oauthsessions.go:20-24` | HAVE | g0router encrypts connection secrets and OAuth verifiers with AES-256-GCM (`*_enc` columns). 9router stores provider secrets in plaintext JSON blobs (`providerConnections.data`). |
| PAR-AUTH-017 | Request log header sanitization | `src/lib/db/repos/requestDetailsRepo.js:46-54` | MISSING | 9router strips `authorization`, `x-api-key`, `cookie`, `token` from logged headers. g0router has no request logging system. |
| PAR-AUTH-018 | Debug log disabled in production | `open-sse/utils/debugLog.js:3` | MISSING | 9router checks `NODE_ENV !== "production"`. g0router has no debug logging utility. |
| PAR-AUTH-019 | OAuth credential manager for provider connections | `open-sse/services/oauthCredentialManager.js` (referenced), `src/lib/oauth/services/*.js` | HAVE | g0router implements Anthropic OAuth with PKCE and refresh (`internal/auth/oauth.go:34-42`, `internal/admin/oauth.go:34-87`). 9router supports ~15 provider OAuth flows. |
| PAR-AUTH-020 | SSRF protections (outbound proxy, MITM DNS bypass) | `open-sse/utils/proxyFetch.js:314-334`, `src/lib/network/outboundProxy.js` | PARTIAL | 9router resolves real IPs via Google DNS for MITM-bypass hosts and supports env proxies. g0router has no outbound proxy or SSRF mitigation. |
| PAR-AUTH-021 | Logout clears OIDC cookies | `src/app/api/auth/logout/route.js:8-10` | HAVE | 9router deletes `oidc_state`, `oidc_nonce`, `oidc_code_verifier`. g0router deletes only the session cookie (`internal/admin/auth.go:84`). |
| PAR-AUTH-022 | OIDC cookie TTL (10 min maxAge) | `src/app/api/auth/oidc/start/route.js:42` | HAVE | 9router sets 10-minute cookie expiry. g0router stores OAuth state in DB with 10-minute expiry (`internal/auth/oauth.go:19`, `internal/store/oauthsessions.go:56`) but uses no cookies. |
| PAR-AUTH-023 | Progressive lockout auto-reset (1h window) | `src/lib/auth/loginLimiter.js:5` | HAVE | 9router resets failed attempts after 1h of inactivity. g0router has no lockout logic. |
| PAR-AUTH-024 | CORS middleware | `internal/server/middleware.go:31-47` | HAVE | g0router sets `Access-Control-Allow-Origin` to the request `Origin` or `*`, allows credentials. 9router uses Next.js built-in behavior. |
| PAR-AUTH-025 | Connection DTO masks secrets | `internal/admin/connections.go:12-24`, `internal/admin/connections.go:37-51` | HAVE | g0router returns `SecretSet` booleans. 9router returns full `data` JSON blobs to the client (secrets visible). |
| PAR-AUTH-026 | Client IP extraction for rate limiting | `src/lib/auth/loginLimiter.js:48-51` | HAVE | 9router reads `x-forwarded-for`, `x-real-ip`. g0router does not extract client IP. |
| PAR-AUTH-027 | Tunnel login block | `src/app/api/auth/login/route.js:11-16`, `src/app/api/auth/login/route.js:33-35` | HAVE | 9router rejects login via tunnel host when `tunnelDashboardAccess` is disabled. g0router has no tunnel rules. |
| PAR-AUTH-028 | OIDC client secret probe endpoint | `src/lib/auth/oidc.js:144-210` | HAVE | 9router probes the token endpoint with an invalid code to validate `client_secret`. g0router has no equivalent. |
| PAR-AUTH-029 | API key table with `machineId` | `src/lib/db/schema.js:74-84` | HAVE | 9router stores `machineId` per key. g0router has no API key table. |
| PAR-AUTH-030 | Session storage (opaque token in DB) | `internal/store/sessions.go:11-62` | HAVE | g0router stores tokens in SQLite with `expires_at` and enforces expiry on read (`GetSession` checks `ExpiresAt`). 9router uses stateless JWT. |

## Data models

### 9router (SQLite)

**`settings`** (single-row JSON blob)
- `id INTEGER PRIMARY KEY CHECK (id = 1)`
- `data TEXT NOT NULL` — contains `password` (bcrypt hash), `authMode`, `requireLogin`, `tunnelDashboardAccess`, `oidcIssuerUrl`, `oidcClientId`, `oidcClientSecret`, `oidcScopes`, `oidcLoginLabel`

**`apiKeys`**
- `id TEXT PRIMARY KEY`
- `key TEXT UNIQUE NOT NULL`
- `name TEXT`
- `machineId TEXT`
- `isActive INTEGER DEFAULT 1`
- `createdAt TEXT NOT NULL`

**`providerConnections`**
- `id TEXT PRIMARY KEY`
- `provider TEXT NOT NULL`
- `authType TEXT NOT NULL`
- `name TEXT`, `email TEXT`, `priority INTEGER`, `isActive INTEGER DEFAULT 1`
- `data TEXT NOT NULL` — JSON blob with secrets in plaintext

**`requestDetails`**
- `id TEXT PRIMARY KEY`
- `timestamp TEXT NOT NULL`
- `provider TEXT`, `model TEXT`, `connectionId TEXT`, `status TEXT`
- `data TEXT NOT NULL` — JSON blob; headers sanitized before insert

### g0router (SQLite)

**`users`**
- `id TEXT PRIMARY KEY`
- `username TEXT NOT NULL UNIQUE`
- `password_hash TEXT NOT NULL`
- `created_at INTEGER NOT NULL`
- `updated_at INTEGER NOT NULL`

**`sessions`**
- `token TEXT PRIMARY KEY`
- `user_id TEXT NOT NULL`
- `expires_at INTEGER NOT NULL`
- `created_at INTEGER NOT NULL`

**`connections`**
- `id TEXT PRIMARY KEY`
- `provider_id TEXT NOT NULL`
- `name TEXT NOT NULL`
- `kind TEXT NOT NULL`
- `secret_enc TEXT NOT NULL DEFAULT ''`
- `access_token_enc TEXT NOT NULL DEFAULT ''`
- `refresh_token_enc TEXT NOT NULL DEFAULT ''`
- `expires_at INTEGER NOT NULL DEFAULT 0`
- `metadata TEXT NOT NULL DEFAULT ''`
- `created_at INTEGER NOT NULL`
- `updated_at INTEGER NOT NULL`

**`oauth_sessions`**
- `state TEXT PRIMARY KEY`
- `provider TEXT NOT NULL`
- `verifier_enc TEXT NOT NULL DEFAULT ''`
- `expires_at INTEGER NOT NULL`
- `created_at INTEGER NOT NULL`

**`settings`**
- `key TEXT PRIMARY KEY`
- `value TEXT NOT NULL`
- `updated_at INTEGER NOT NULL`

**`providers`**
- `id TEXT PRIMARY KEY`
- `name TEXT NOT NULL`
- `type TEXT NOT NULL`
- `base_url TEXT NOT NULL DEFAULT ''`
- `enabled INTEGER NOT NULL DEFAULT 1`
- `created_at INTEGER NOT NULL`
- `updated_at INTEGER NOT NULL`

## Edge cases and quirks

### 9router

- **JWT secret generation**: `loadJwtSecret` reads `JWT_SECRET` env first, else reads `DATA_DIR/jwt-secret`, else generates 32 random bytes and writes with `mode 0o600` (`src/lib/auth/dashboardSession.js:7-17`).
- **Password reset race**: CLI reset mutates `db.json` directly while the server may hold the file handle, risking corruption (`cli/src/cli/menus/settings.js:194-199`).
- **Local-only origin check**: `isLocalRequest` rejects when `Origin` header points to a non-loopback host, even if `Host` is loopback (`src/dashboardGuard.js:91-100`). This blocks CSRF from external sites.
- **API key CRC weakness**: `generateApiKeyWithMachine` produces an 8-char hex CRC from HMAC-SHA256 truncated to 8 chars. Collision probability is high by design (`src/shared/utils/apiKey.js:20-26`).
- **Login lockout memory-only**: `attempts` Map resets on process restart (`src/lib/auth/loginLimiter.js:7`).
- **OIDC nonce/state cookies**: 10-minute `maxAge` but no `expires` attribute; relies on browser session plus `maxAge` (`src/app/api/auth/oidc/start/route.js:42`).
- **Request logger masking disabled**: `maskSensitiveHeaders` currently returns headers unchanged; old masking code is commented out (`open-sse/utils/requestLogger.js:72-91`).
- **Debug logs leak in dev**: `dbg()` prints everything when `NODE_ENV !== "production"` with no filtering (`open-sse/utils/debugLog.js:3-12`).

### g0router

- **Session expiry check in DB read**: `GetSession` returns `ErrNotFound` when `expires_at <= now` (`internal/store/sessions.go:42-44`). No background cleaner runs automatically.
- **Empty secret preservation on update**: `UpdateConnection` skips encryption when `req.Secret == ""`, preserving the stored value (`internal/admin/connections.go:133-134`).
- **Password hash constant-time compare**: `VerifyPassword` uses `subtle.ConstantTimeCompare` (`internal/auth/password.go:61`).
- **Cipher key length guard**: `NewCipher` rejects keys not equal to 32 bytes (`internal/store/crypto.go:18-21`).
- **Secret file permissions**: `LoadOrCreateSecret` writes with `0o600` and creates data dir with `0o700` (`internal/store/secret.go:15-16`, `36`).
- **OAuth state TTL mismatch**: DB expiry is 10 minutes (`internal/auth/oauth.go:19`), but no cleanup job deletes expired rows automatically.

## Go-port considerations

- Replace bcrypt with `golang.org/x/crypto/bcrypt` or keep PBKDF2. PBKDF2 is already present and OWASP-recommended.
- Add centralized middleware guard instead of per-route `RequireSession` wrappers. Use path prefix lists matching 9router logic.
- Implement API key table with CRC validation or replace with simpler random tokens and HMAC.
- Add login rate limiter; use `golang.org/x/time/rate` or in-memory map with TTL.
- Add `x-forwarded-proto` detection for secure cookie flag.
- Add tunnel host detection if tunnel feature is planned.
