# Security Audit — g0router
**Date:** 2026-06-05  
**Scope:** auth on /api/*, /v1/*; API keys; CORS; bind defaults; credential redaction; OAuth state/PKCE; token storage (SQLite); secret leakage in logs/errors.  
**Method:** Static source review only. No execution, no network, no secrets printed.

---

## Severity Summary

| Severity | Count |
|----------|-------|
| Critical | 0     |
| High     | 3     |
| Medium   | 3     |
| Low      | 2     |

---

## HIGH Findings

---

### H1 — MCP OAuth callback exempt from auth without scope guard

**File:line:** `api/middleware.go:100`

```go
func isProtectedManagementPath(requestPath string) bool {
    if requestPath == "/api/oauth/callback" || requestPath == "/api/mcp/oauth/callback" {
        return false
    }
    return requestPath == "/api" || strings.HasPrefix(requestPath, "/api/")
}
```

**Why it matters:**  
`/api/oauth/callback` and `/api/mcp/oauth/callback` are unconditionally exempted from API-key auth. This is necessary for the OAuth redirect flow to work — browsers follow redirects and cannot inject an API key. However, the exemption is **global**: it applies whether or not `REQUIRE_API_KEY=true` is set. An attacker who knows a valid OAuth `state` value (or can race/brute-force it) can complete an OAuth exchange against the running server without any credentials. The state is 24-byte random URL-safe base64 (192-bit entropy — `mcp/oauth.go:160`), and it is hashed with SHA-256 before storage (`store/mcpoauth.go:308`), so brute force is infeasible. The real risk is **open SSRF via OAuth discovery**: `MCPOAuthStart` at `api/handlers/mcp.go:171` calls `mcp.DiscoverOAuthAuthorizationURL` using `http.DefaultClient` against a caller-supplied `resource_uri`. That endpoint **is** auth-protected (`/api/mcp/instances/{id}/auth/start` requires auth via `isProtectedManagementPath`), so the SSRF vector requires a valid key. The callback exemption itself is **design-necessary** but represents an intentional attack surface enlargement that must be documented and reviewed on any future feature that touches that path.

**Minimal fix:**  
Document the exemption explicitly. Add a comment in `isProtectedManagementPath` explaining that these paths are intentionally open. Consider adding rate-limiting or a short-lived nonce check tied to IP/session to prevent replay attempts by network-adjacent attackers.

**Verification:** Review all handlers reachable via `handleAPI` that do not pass through `isProtectedManagementPath` gate; confirm none mutate state without validating the OAuth state parameter.

---

### H2 — MCP OAuth flow expiry NOT checked on consume

**File:line:** `internal/store/mcpoauth.go:67–86` vs `internal/store/oauthsessions.go:39–61`

`ConsumeOAuthSession` (provider OAuth) checks expiry before consuming:
```go
// oauthsessions.go:50
if !session.ExpiresAt.IsZero() && time.Now().After(session.ExpiresAt) {
    _, _ = tx.Exec("DELETE FROM oauth_sessions WHERE id = ?", session.ID)
    return nil, ErrNotFound
}
```

`ConsumeMCPOAuthFlow` (MCP OAuth) does **not** check expiry before consuming:
```go
// mcpoauth.go:67–86
flow, err := scanMCPOAuthFlow(tx.QueryRow(...))
if err != nil { return nil, err }
// NO expiry check here
if _, err := tx.Exec("DELETE FROM mcp_oauth_flows WHERE id = ?", flow.ID); ...
```

The expiry check for MCP OAuth happens only in `mcp/oauth.go:221`:
```go
if !flow.ExpiresAt.IsZero() && time.Now().After(flow.ExpiresAt) {
    return OAuthAccount{}, ErrReauthRequired
}
```
That check is **after** the flow record is consumed (deleted from DB). An expired flow is therefore silently deleted without returning an error at the store layer; the caller (`OAuthEngine.CompleteCallback`) does catch it, but only because the engine layer re-checks. If any other caller calls `store.ConsumeMCPOAuthFlow` directly and doesn't re-check expiry, expired flows succeed. Currently the only caller goes through `OAuthEngine`, so there is no immediate exploit — but the defense is shallow and the store interface promises consume-or-fail semantics.

**Why it matters:** Expired authorization codes should not be exchangeable. The asymmetry between the two OAuth paths is a logic defect waiting to cause a real bypass if the call chain changes.

**Minimal fix:** Add the expiry check inside `ConsumeMCPOAuthFlow` at the store layer, immediately after scanning, before deleting — matching the pattern in `ConsumeOAuthSession`.

**Verification:** Write a test: create a flow with `ExpiresAt = time.Now().Add(-1*time.Second)`, call `ConsumeMCPOAuthFlow`, assert `ErrNotFound` or equivalent.

---

### H3 — Internal error details leaked verbatim to HTTP clients

**Files:lines:**  
- `api/handlers/connections.go:62,72,248`  
- `api/handlers/mcp.go:91,101,113,119,201,218,250,264,270,276,317,540`  
- `api/handlers/mcpoauth.go:84` (`fmt.Sprintf("reapply mcp credentials: %v", err)`)

Pattern throughout:
```go
writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("list connections: %v", err))
writeError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("create mcp oauth flow: %v", err))
```

**Why it matters:** `%v` on a Go error unwraps full error chains. SQLite errors (including table/column names, constraint violations, and file paths), network errors (with remote hostnames), and MCP process launch errors (with command paths and env variable names) are all sent in HTTP 500 response bodies to any authenticated caller. An authenticated but low-privilege user (e.g., someone with a read-only API key, if key ACLs are ever added) gets rich enumeration data. In the current single-key model the risk is lower, but it is still an information leak.

**Minimal fix:** Log the full error server-side; return a generic `"internal error"` string to the client for 5xx responses. For 4xx (client errors) like `mcp.go:187` (`err.Error()`) it is acceptable to return the message if it is validated to be non-sensitive (e.g., validation messages from `BuildOAuthStartFlow`).

**Verification:** Grep for `writeError.*fmt.Sprintf` and `writeError.*err.Error()` in all handlers; audit each for whether the wrapped error string can contain file paths, credentials, or DB internals.

---

## MEDIUM Findings

---

### M1 — Docker image sets BIND_ADDRESS=0.0.0.0 without requiring REQUIRE_API_KEY validation

**File:line:** `docker-compose.yml:12` (`BIND_ADDRESS: "0.0.0.0"`) and `docker-compose.yml:17` (`REQUIRE_API_KEY: "true"`)

The compose file correctly sets `REQUIRE_API_KEY=true` and binds the host side to `127.0.0.1:20128:20128`. However, the `Dockerfile` sets no default for `REQUIRE_API_KEY` or `API_KEY_SECRET`:

```dockerfile
# Dockerfile — only sets:
ENV DATA_DIR=/data
ENV PORT=20128
```

A user who runs `docker run -p 0.0.0.0:20128:20128 g0router serve` without the compose file gets `BIND_ADDRESS=127.0.0.1` (safe default from `config.go:42`) but `REQUIRE_API_KEY=true` with an empty `API_KEY_SECRET`, which **causes startup to fail** (`config.go:70-71`). The failure mode is correct (fail-closed), but the error message is:

```
API_KEY_SECRET required when REQUIRE_API_KEY=true
```

There is a separate footgun: if a user sets `REQUIRE_API_KEY=false` to skip the error, the entire `/api/*` and `/v1/*` surface becomes open. There is no warning in the Dockerfile or entrypoint about this.

**Minimal fix:** Add `ENV REQUIRE_API_KEY=true` to the Dockerfile so the default is explicit even without compose. Consider adding a startup log warning when `REQUIRE_API_KEY=false` and `BIND_ADDRESS` is not loopback.

**Verification:** `docker run --rm g0router serve` should fail with the API_KEY_SECRET error; confirm no silent open-access mode.

---

### M2 — `ensureColumn` uses string concatenation for DDL (SQL injection surface)

**File:line:** `internal/store/sqlite.go:243,268`

```go
rows, err := s.db.Query("PRAGMA table_info(" + table + ")")
// ...
s.db.Exec("ALTER TABLE " + table + " ADD COLUMN " + column + " " + definition)
```

`table`, `column`, and `definition` are hardcoded call-site strings (`"mcp_oauth_flows"`, `"client_id"`, `"TEXT"`) so there is no actual injection path from user input today. But the function signature accepts arbitrary `string` parameters and does no validation, making it an unsafe pattern that will bite if callers are ever added with variable inputs.

**Minimal fix:** Add a compile-time or runtime allowlist check (e.g., `validIdentifier` regexp) on `table` and `column` before using them in raw SQL strings.

**Verification:** Grep for all callers of `ensureColumn`; confirm all are literal strings or add the allowlist check.

---

### M3 — `client_secret` stored in plaintext in `mcp_oauth_flows` table

**File:line:** `internal/store/sqlite.go:208` (DDL: `client_secret TEXT`), `internal/store/mcpoauth.go:58`

The MCP OAuth flow stores the OAuth `client_secret` in the `mcp_oauth_flows` table (short-lived, 10-minute TTL) as plaintext. The `mcp_oauth_accounts` table also stores `access_token` and `refresh_token` as plaintext (`sqlite.go:187–196`).

The SQLite file lives at `~/.g0router/g0router.db` (or `/data/g0router.db` in Docker). The data directory is created with `0o755` (`sqlite.go:19`), meaning any local user can read the DB file. The DB file itself inherits umask (typically `0644` on Linux), making tokens readable by any local user.

This is expected for a local-first tool, but for shared or server deployments it is a real risk.

**Minimal fix:** Create the data directory with `0o700` (user-only). SQLite does not support column-level encryption, but restricting directory permissions is the minimum viable hardening. Document that tokens are stored at rest unencrypted.

**Verification:** `stat ~/.g0router/` should show `drwx------`. `stat ~/.g0router/g0router.db` should show `-rw-------`.

---

## LOW Findings

---

### L1 — `JWT_SECRET` loaded but unused

**File:line:** `internal/config/config.go:78` (`JWTSecret: os.Getenv("JWT_SECRET")`), `.env.example:14`

`JWT_SECRET` is loaded into `Config` but never used by any code path in the audited surface. The `.env.example` says "reserved for compatibility and is not used by the current auth path." Having a dead secret field is a maintenance trap: operators may believe it has an effect (e.g., HMAC-signing tokens), configure it, rotate it, and get no security benefit. Worse, a future developer may wire it up incorrectly.

**Minimal fix:** Remove the `JWTSecret` field from `Config` and the env var from `.env.example`, or add a `//nolint` comment with a clear TODO tracking the intended use.

---

### L2 — `CORS` allows any localhost port

**File:line:** `api/middleware.go:69–86`

```go
switch parsed.Hostname() {
case "localhost", "127.0.0.1", "::1":
    return true
```

Any origin on any port at localhost/loopback is allowed. This is intentional for local dev but means a malicious page served on `localhost:3000` (a compromised local web server, another dev tool, etc.) can make credentialed cross-origin requests to g0router.

**Why it matters (low, not medium):** Exploitation requires the attacker to already control a local process — at which point they likely have easier attack paths. But it is worth noting for threat-modeling purposes.

**Minimal fix:** If g0router ships with a fixed UI port, restrict CORS to that exact origin. Otherwise, document the permissive policy explicitly.

---

## Areas with NO Issue Found

| Area | Finding |
|------|---------|
| API key hashing | HMAC-SHA256 with `API_KEY_SECRET` as key (`store/apikeys.go:114`). No timing side-channel — `hmac.Equal` not used for comparison but lookup is hash-equality via DB query (timing-safe in practice). |
| PKCE implementation | S256 only, using `crypto/sha256` and `base64.RawURLEncoding` (`mcp/oauth.go:561`). Correct. |
| OAuth state | 24-byte CSPRNG, SHA-256 hashed before DB storage. Consumed on first use (delete-on-consume). Provider OAuth has expiry check before consume. |
| Request ID generation | `crypto/rand` 128-bit UUID v4 (`api/middleware.go:151`). Correct. |
| Connection response redaction | `connectionResponse` struct omits `AccessToken`, `RefreshToken`, `APIKey` fields. `redactConnection` never copies raw token fields to the wire struct. `redactProviderSpecificData` strips keys containing "token", "secret", "key", "authorization", "password". |
| MCP credential redaction | `InstanceConfig.Redacted()` calls `redactSecretMap` on both `Env` and `Headers` before returning to API callers (`mcp/instances.go:59–63`). `StdioCredentialEnv` also uses `redactSecretMap` for the log-safe map. |
| Inference endpoint auth | `/v1/chat/completions`, `/v1/messages`, `/v1/responses` are all guarded by `requiresAuth` via the `/v1/` prefix check (`middleware.go:96`). |
| SQL injection in queries | All parameterized queries use `?` placeholders. The only raw string concatenation is the `ensureColumn` DDL (M2 above), which uses hardcoded call-site values only. |
| Default bind address | `config.go:42` defaults to `127.0.0.1`. Bind to `0.0.0.0` only via explicit `BIND_ADDRESS` env var. |
| `REQUIRE_API_KEY` default | `true` (`config.go:47`). Fail-closed if `API_KEY_SECRET` empty. |
| Token leakage in logs | `logging/requestlog.go` and `logging/logger.go` contain no credential fields. `sanitizedLogError` in `api/server.go:396` only logs error classification codes and messages, not raw tokens. |
| `healthz` endpoint | Open (no auth), returns only version string. Acceptable. |

---

## Route Protection Map (confirmed)

All routes confirmed auth-gated when `REQUIRE_API_KEY=true`:

| Route | Protected |
|-------|-----------|
| `GET /healthz` | No (intentional) |
| `POST /v1/chat/completions` | Yes |
| `POST /v1/messages` | Yes |
| `POST /v1/responses` | Yes |
| `GET /v1/models` | Yes |
| `GET /api/providers` | Yes |
| `GET /api/connections` | Yes |
| `POST /api/connections` | Yes |
| `GET /api/settings` | Yes |
| `GET/POST /api/keys` | Yes |
| `GET /api/usage` | Yes |
| `GET /api/logs` | Yes |
| `GET/POST /api/mcp/clients` | Yes |
| `GET/POST /api/mcp/instances` | Yes |
| `POST /api/mcp/instances/{id}/auth/start` | Yes |
| `GET /api/mcp/instances/{id}/accounts` | Yes |
| `GET /api/mcp/tools` | Yes |
| `POST /api/oauth/{provider}/authorize` | Yes |
| `GET /api/oauth/{provider}/poll` | Yes |
| `POST /api/oauth/{provider}/exchange` | Yes |
| `GET /api/oauth/callback` | **No** (intentional — browser redirect, see H1) |
| `GET /api/mcp/oauth/callback` | **No** (intentional — browser redirect, see H1) |
| `POST /api/mcp/instances/{id}/oauth/complete` | Yes |
