# Database Schema + API Contracts

## SQLite Schema

### connections
```sql
CREATE TABLE connections (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    provider TEXT NOT NULL,
    name TEXT NOT NULL DEFAULT '',
    auth_type TEXT NOT NULL CHECK (auth_type IN ('oauth', 'api_key', 'noauth')),
    access_token TEXT,
    refresh_token TEXT,
    expires_at INTEGER,           -- Unix timestamp
    api_key TEXT,
    is_active INTEGER NOT NULL DEFAULT 1,
    provider_specific_data TEXT,  -- JSON blob
    account_id TEXT,
    email TEXT,
    unavailable_until INTEGER,    -- Unix timestamp, cooldown expiry
    backoff_level INTEGER NOT NULL DEFAULT 0,
    model_locks TEXT,             -- JSON: {"model_name": unix_timestamp}
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_connections_provider ON connections(provider);
CREATE INDEX idx_connections_active ON connections(provider, is_active);
```

### settings
```sql
CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
-- Keys: password_hash, jwt_secret, require_api_key, rtk_enabled, caveman_enabled,
--        caveman_level, enable_request_logs, proxy_url, data_dir
```

### api_keys
```sql
CREATE TABLE api_keys (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    name TEXT NOT NULL UNIQUE,
    key_hash TEXT NOT NULL,        -- HMAC-SHA256
    prefix TEXT NOT NULL,          -- First 8 chars for display
    is_active INTEGER NOT NULL DEFAULT 1,
    last_used_at TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
```

### combos
```sql
CREATE TABLE combos (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    name TEXT NOT NULL UNIQUE,     -- e.g. "my-chain"
    steps TEXT NOT NULL,           -- JSON: [{"provider":"anthropic","model":"claude-sonnet-4-20250514"},...]
    is_active INTEGER NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
```

### model_aliases
```sql
CREATE TABLE model_aliases (
    alias TEXT PRIMARY KEY,
    provider TEXT NOT NULL,
    model TEXT NOT NULL
);
```

### pricing_overrides
```sql
CREATE TABLE pricing_overrides (
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    input_cost_per_token REAL,
    output_cost_per_token REAL,
    PRIMARY KEY (provider, model)
);
```

### request_log
```sql
CREATE TABLE request_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    request_id TEXT NOT NULL,
    timestamp TEXT NOT NULL DEFAULT (datetime('now')),
    provider TEXT NOT NULL,
    model TEXT NOT NULL,
    connection_id TEXT,
    auth_type TEXT NOT NULL,
    input_tokens INTEGER,
    output_tokens INTEGER,
    cache_read_tokens INTEGER,
    cache_write_tokens INTEGER,
    total_tokens INTEGER,
    cost_usd REAL,
    latency_ms INTEGER,
    status_code INTEGER,
    error TEXT,
    source_format TEXT,
    target_format TEXT,
    rtk_enabled INTEGER,
    rtk_bytes_saved INTEGER,
    caveman_enabled INTEGER,
    combo_name TEXT,
    api_key_id TEXT,
    client_tool TEXT,             -- detected client (claude-code, codex, cursor, etc.)
    FOREIGN KEY (connection_id) REFERENCES connections(id),
    FOREIGN KEY (api_key_id) REFERENCES api_keys(id)
);
CREATE INDEX idx_request_log_timestamp ON request_log(timestamp);
CREATE INDEX idx_request_log_provider ON request_log(provider);
CREATE INDEX idx_request_log_model ON request_log(model);
CREATE INDEX idx_request_log_auth ON request_log(auth_type);
```

### mcp_clients
```sql
CREATE TABLE mcp_clients (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    name TEXT NOT NULL UNIQUE,
    transport TEXT NOT NULL CHECK (transport IN ('stdio', 'sse', 'streamable-http')),
    command TEXT,                  -- For stdio transport
    args TEXT,                    -- JSON array for stdio
    url TEXT,                     -- For SSE/HTTP transport
    env TEXT,                     -- JSON object of env vars
    is_active INTEGER NOT NULL DEFAULT 1,
    health_status TEXT DEFAULT 'unknown',
    last_health_check TEXT,
    tool_manifest TEXT,           -- JSON: cached tool schemas
    manifest_updated_at TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);
```

### mcp_instances
```sql
CREATE TABLE mcp_instances (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    name TEXT NOT NULL UNIQUE,
    server_key TEXT NOT NULL,      -- server definition key, reusable across accounts
    launch_type TEXT NOT NULL CHECK (launch_type IN ('command', 'npx', 'docker', 'http')),
    transport TEXT NOT NULL CHECK (transport IN ('stdio', 'sse', 'streamable-http')),
    command TEXT,                  -- command, npm package, or docker image depending on launch_type
    args TEXT,                     -- JSON array
    url TEXT,                      -- remote HTTP/SSE endpoint for launch_type=http
    headers TEXT,                  -- JSON object; secret-like keys are redacted from list responses
    env TEXT,                      -- JSON object; secret-like keys are redacted from list responses
    cwd TEXT,
    account_label TEXT,            -- selected account label for this instance
    is_active INTEGER NOT NULL DEFAULT 1,
    health_status TEXT DEFAULT 'unknown',
    last_health_check TEXT,
    tool_manifest TEXT,            -- JSON cached tool schemas per instance
    manifest_updated_at TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX idx_mcp_instances_server_key ON mcp_instances(server_key);
```

### mcp_oauth_accounts
```sql
CREATE TABLE mcp_oauth_accounts (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    instance_id TEXT NOT NULL,
    account_label TEXT NOT NULL DEFAULT 'default',
    subject TEXT,
    email TEXT,
    issuer TEXT,
    resource_uri TEXT,
    scopes TEXT,                   -- JSON array
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    expires_at TEXT,
    auth_metadata TEXT,            -- JSON object
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(instance_id, account_label),
    FOREIGN KEY (instance_id) REFERENCES mcp_instances(id) ON DELETE CASCADE
);
CREATE INDEX idx_mcp_oauth_accounts_instance ON mcp_oauth_accounts(instance_id);
```

### mcp_oauth_flows
```sql
CREATE TABLE mcp_oauth_flows (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    instance_id TEXT NOT NULL,
    state_hash TEXT NOT NULL,
    code_verifier_secret TEXT NOT NULL,
    redirect_uri TEXT,
    authorization_url TEXT,
    resource_uri TEXT,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE(instance_id, state_hash),
    FOREIGN KEY (instance_id) REFERENCES mcp_instances(id) ON DELETE CASCADE
);
CREATE INDEX idx_mcp_oauth_flows_instance ON mcp_oauth_flows(instance_id);
```

---

## API Contracts

### Inference
```
POST /v1/chat/completions     — OpenAI chat format
POST /v1/messages             — Anthropic messages format
POST /v1/responses            — OpenAI Responses API format
GET  /v1/models               — List available models
```

### Management
```
GET    /api/providers                — List provider matrix entries with public_status/capability fields
GET    /api/providers/:id/models     — List models for provider
POST   /api/connections              — Create connection (OAuth or API key)
GET    /api/connections              — List connections
PUT    /api/connections/:id          — Update connection
DELETE /api/connections/:id          — Delete connection
POST   /api/connections/:id/test     — Test connection

POST   /api/oauth/:provider/authorize  — Start OAuth flow
GET    /api/oauth/:provider/poll       — Poll device-code flow
GET    /api/oauth/callback             — OAuth callback handler

GET    /api/combos                   — List combos
POST   /api/combos                   — Create combo
PUT    /api/combos/:id               — Update combo
DELETE /api/combos/:id               — Delete combo

GET    /api/settings                 — Get settings
PUT    /api/settings                 — Update settings

GET    /api/keys                     — List API keys
POST   /api/keys                     — Create API key
DELETE /api/keys/:id                 — Delete API key

GET    /api/usage                    — Usage log (filtered, paginated)
GET    /api/usage/summary            — Aggregated usage summary
GET    /api/usage/quota/:provider    — Provider quota/limits

GET    /api/mcp/clients              — List MCP clients
POST   /api/mcp/clients              — Add MCP client
DELETE /api/mcp/clients/:id          — Remove MCP client
GET    /api/mcp/instances            — List MCP instances with redacted env/headers
POST   /api/mcp/instances            — Add MCP instance
DELETE /api/mcp/instances/:id        — Remove MCP instance and cascade OAuth accounts/flows
POST   /api/mcp/instances/:id/auth/start     — Create MCP OAuth flow
POST   /api/mcp/instances/:id/oauth/complete — Complete pasted callback URL
GET    /api/mcp/instances/:id/accounts       — List redacted OAuth accounts
GET    /api/mcp/oauth/callback       — Browser OAuth callback completion
GET    /api/mcp/tools                — List discovered tools (compact); optional instance_id/account_label filters
POST   /api/mcp/tools/:name/execute  — Execute tool

GET    /api/logs                     — Query request/response logs

GET    /healthz                      — Health check
```

### CLI Commands
```
g0router serve [--port PORT] [--data-dir DIR]
g0router login <provider> [--device] [--key]
g0router logout <provider>
g0router keys add <name>
g0router keys list
g0router keys rm <name>
g0router providers list              # Public direct-dispatch providers only
g0router providers test <provider>
g0router mcp add <name> --server-key KEY --launch-type http --transport streamable-http --url URL [--account-label LABEL]
g0router mcp list
g0router mcp accounts <instance>
g0router mcp tools <instance>
g0router mcp auth start <instance> --authorization-url URL --resource URI --redirect-url URL
g0router mcp auth complete <instance> <callback-url>
g0router mcp remove <instance>
g0router status
g0router version
g0router install [--user]            # Install as systemd service
g0router uninstall                   # Remove systemd service (keeps data)
g0router healthcheck                 # Exit 0 if server is healthy (for Docker HEALTHCHECK)
```
