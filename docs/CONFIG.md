# Configuration Reference

All configuration via environment variables. Runtime overrides via the `settings` SQLite table or `PUT /api/settings`.

## Environment Variables

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `PORT` | int | No | `20128` | HTTP listen port. Range: 1–65535. |
| `DATA_DIR` | path | No | `~/.g0router` | Directory for SQLite database and any persistent data. Created automatically if missing. `~` is expanded to `$HOME`. |
| `JWT_SECRET` | string | For dashboard auth | — | HMAC secret for signing JWT session tokens. Min 32 chars recommended. Generate: `openssl rand -hex 32`. |
| `API_KEY_SECRET` | string | When `REQUIRE_API_KEY=true` | — | HMAC secret for hashing gateway API keys. Same generation method as JWT_SECRET. **Different secret from JWT.** |
| `REQUIRE_API_KEY` | bool | No | `true` | When true, all `/v1/*` inference endpoints require a valid API key via `Authorization: Bearer <key>` or `X-API-Key` header. Management API (`/api/*`) is always accessible (dashboard auth separate). |
| `ENABLE_REQUEST_LOGS` | bool | No | `false` | Store request/response metadata in `request_log` table. Increases disk usage. Does NOT store request/response bodies — only metadata (tokens, cost, latency, model, etc.). |
| `RTK_ENABLED` | bool | No | `true` | Enable Response Token Kompression. Autodetects tool output format and applies compression filters. See [Phase 7](phases/phase-07-rtk-caveman.md). |
| `CAVEMAN_ENABLED` | bool | No | `false` | Inject caveman-mode system prompt to compress LLM output. See [Phase 7](phases/phase-07-rtk-caveman.md). |
| `CAVEMAN_LEVEL` | string | No | `full` | Caveman compression level. Values: `lite` (gentle), `full` (standard), `ultra` (maximum compression). |
| `HTTPS_PROXY` | string | No | — | HTTP proxy URL for all upstream provider requests. Example: `http://proxy.corp:8080`. |

## Boolean Parsing

All boolean env vars accept (case-insensitive):
- **True**: `true`, `1`, `yes`
- **False**: `false`, `0`, `no`, `""` (empty)
- **Any other value**: config load error

## Validation Rules

| Rule | Condition | Error |
|------|-----------|-------|
| Port range | `PORT` < 1 or > 65535 | `"port must be 1-65535"` |
| API key secret required | `REQUIRE_API_KEY=true` and `API_KEY_SECRET` empty | `"API_KEY_SECRET required when REQUIRE_API_KEY=true"` |
| Caveman level | `CAVEMAN_LEVEL` not in `{lite, full, ultra}` | `"caveman level must be lite, full, or ultra"` |
| Data dir writable | `DATA_DIR` path not writable | `"data dir not writable: <path>"` |

## Precedence

```
1. Environment variable (startup)           ← highest priority
2. SQLite settings table (runtime)          ← changed via API/dashboard
3. Compiled defaults                        ← lowest priority
```

Env vars set the **initial** values. The dashboard/API can override `require_api_key`, `rtk_enabled`, `caveman_enabled`, `caveman_level`, and `enable_request_logs` at runtime via the `settings` table. These runtime overrides persist across restarts (stored in SQLite).

To force an env var value and prevent runtime override, don't expose that setting in the dashboard.

## Config Struct (Go)

```go
type Config struct {
    Port              int    // Default: 20128
    DataDir           string // Default: ~/.g0router (expanded)
    JWTSecret         string // From JWT_SECRET env
    APIKeySecret      string // From API_KEY_SECRET env
    RequireAPIKey     bool   // Default: true
    EnableRequestLogs bool   // Default: false
    RTKEnabled        bool   // Default: true
    CavemanEnabled    bool   // Default: false
    CavemanLevel      string // Default: "full"
}

func Load() (*Config, error)  // Reads env, applies defaults, validates
```

## .env.example

```bash
# g0router configuration
# Copy to .env and edit. Or set as real environment variables.

# ─── Server ───
PORT=20128
DATA_DIR=~/.g0router

# ─── Security (REQUIRED in production) ───
# Generate each with: openssl rand -hex 32
JWT_SECRET=
API_KEY_SECRET=

# ─── Access Control ───
REQUIRE_API_KEY=true

# ─── Logging ───
ENABLE_REQUEST_LOGS=false

# ─── RTK (Response Token Kompression) ───
RTK_ENABLED=true

# ─── Caveman Mode ───
CAVEMAN_ENABLED=false
CAVEMAN_LEVEL=full   # lite | full | ultra

# ─── Network ───
# HTTPS_PROXY=http://proxy:8080
```

## SQLite Settings Table

```sql
CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
```

Pre-seeded keys on first run:

| Key | Default Value | Type |
|-----|---------------|------|
| `require_api_key` | `"true"` | bool (as string) |
| `rtk_enabled` | `"true"` | bool |
| `caveman_enabled` | `"false"` | bool |
| `caveman_level` | `"full"` | string |
| `enable_request_logs` | `"false"` | bool |
| `proxy_url` | `""` | string |
| `data_dir` | from env | string |
