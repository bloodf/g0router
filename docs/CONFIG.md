# Configuration Reference

All configuration via environment variables. Runtime overrides via the `settings` SQLite table or `PUT /api/settings`.

## Environment Variables

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| `PORT` | int | No | `20128` | HTTP listen port. Range: 1–65535. |
| `BIND_ADDRESS` | IP address | No | `127.0.0.1` | HTTP listen address. Use `0.0.0.0` or `::` only when public exposure is intentional and protected by Docker host binding, firewall, or reverse proxy controls. |
| `DATA_DIR` | path | No | `~/.g0router` | Directory for SQLite database and any persistent data. Created automatically if missing. `~` is expanded to `$HOME`. |
| `JWT_SECRET` | string | No | — | Reserved bootstrap setting loaded by config but not used by the current control-plane auth path. |
| `API_KEY_SECRET` | string | When `REQUIRE_API_KEY=true` | — | HMAC secret for hashing gateway API keys. Generate with `openssl rand -hex 32`. |
| `REQUIRE_API_KEY` | bool | No | `true` | When true, all `/v1/*` inference endpoints and `/api/*` management endpoints require a valid API key via `Authorization: Bearer <key>` or `X-API-Key` header. OAuth callback endpoints remain public so provider redirects can complete. |
| `ENABLE_REQUEST_LOGS` | bool | No | `false` | Store request/response metadata in `request_log` table. Increases disk usage. Does NOT store request/response bodies — only metadata (tokens, cost, latency, model, etc.). |
| `RTK_ENABLED` | bool | No | `true` | Enable Response Token Kompression. Autodetects tool output format and applies compression filters. See [Phase 7](phases/phase-07-rtk-caveman.md). |
| `CAVEMAN_ENABLED` | bool | No | `false` | Inject caveman-mode system prompt to compress LLM output. See [Phase 7](phases/phase-07-rtk-caveman.md). |
| `CAVEMAN_LEVEL` | string | No | `full` | Caveman compression level. Values: `lite` (gentle), `full` (standard), `ultra` (maximum compression). |
| `HTTPS_PROXY` | string | No | — | HTTP proxy URL for all upstream provider requests. Example: `http://proxy.corp:8080`. |
| `VERTEX_PROJECT_ID` | string | For Vertex dispatch | — | Google Cloud project ID used by the Vertex adapter for provider-qualified catalog models such as `vertex/gemini-2.5-flash`. |
| `VERTEX_LOCATION` | string | No | `us-central1` | Google Cloud location used by the Vertex adapter. |

## Optional Live Provider Smoke Tests

Normal test gates use local fake upstreams and do not require network provider
credentials. Live smoke tests are opt-in only:

| Variable | Required | Description |
|----------|----------|-------------|
| `G0ROUTER_LIVE_TESTS=1` | Yes | Enables live provider smoke tests. Tests are skipped unless this exact value is set. |
| `G0ROUTER_E2E_MINIMAX_API_KEY` | For MiniMax live tests | MiniMax API key read from the environment. Never commit or echo this value. |
| `G0ROUTER_E2E_MINIMAX_BASE_URL` | No | Override MiniMax OpenAI-compatible base URL for live smoke tests. |

## MCP Instance Configuration

MCP servers are runtime-managed records, not global environment variables. Configure them with the dashboard, the management API, or CLI commands:

```bash
# Remote streamable HTTP MCP server
g0router mcp add atlassian-a \
  --server-key atlassian \
  --launch-type http \
  --transport streamable-http \
  --url https://mcp.atlassian.com/mcp \
  --account-label account-a

# Local command MCP server
g0router mcp add local-files \
  --server-key filesystem \
  --launch-type command \
  --transport stdio \
  --command mcp-files \
  --arg --stdio

# npx MCP server. The generated launch spec is argv-based and does not use a shell.
g0router mcp add expo \
  --server-key expo \
  --launch-type npx \
  --transport stdio \
  --command @expo/mcp \
  --arg --stdio

# Docker MCP server. Env values are passed to the subprocess and redacted from list output.
g0router mcp add docker-search \
  --server-key docker-search \
  --launch-type docker \
  --transport stdio \
  --command mcp/search:latest \
  --env TOKEN=secret
```

For HTTP MCP OAuth:

```bash
g0router mcp auth start atlassian-a \
  --authorization-url https://auth.example/authorize \
  --resource https://mcp.atlassian.com \
  --redirect-url http://localhost:20128/api/mcp/oauth/callback

g0router mcp auth complete atlassian-a "http://localhost:20128/api/mcp/oauth/callback?code=...&state=..."
```

Multiple accounts for the same MCP server are modeled as multiple instances with the same `server_key` and different `name`/`account_label` values.

## Boolean Parsing

All boolean env vars accept (case-insensitive):
- **True**: `true`, `1`, `yes`
- **False**: `false`, `0`, `no`, `""` (empty)
- **Any other value**: config load error

## Validation Rules

| Rule | Condition | Error |
|------|-----------|-------|
| Port range | `PORT` < 1 or > 65535 | `"port must be 1-65535"` |
| Bind address | `BIND_ADDRESS` is not an IP address, or includes a port | `"BIND_ADDRESS must be an IP address"` |
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
    BindAddress       string // Default: 127.0.0.1
    JWTSecret         string // Reserved; loaded from JWT_SECRET env
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
BIND_ADDRESS=127.0.0.1
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

# ─── Vertex AI (optional) ───
# VERTEX_PROJECT_ID=
# VERTEX_LOCATION=us-central1
```

## Docker Binding

The checked-in `docker-compose.yml` publishes `127.0.0.1:20128:20128` on the host and sets `BIND_ADDRESS=0.0.0.0` inside the container. This keeps the default compose deployment local to the host while making the container listener explicit. To publish the control plane publicly, change the host port binding deliberately only with firewall or reverse-proxy protection in place; keep `REQUIRE_API_KEY=true` with `API_KEY_SECRET` set for inference and management routes.

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
