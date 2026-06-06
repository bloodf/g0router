# Phase 18: Bifrost Features (Governance)

> Process, contracts, gates, architecture: see `docs/phases/STAGE-13-19-PROCESS.md`.
> Security review: **mandatory** at checkpoint (budget enforcement, backup export).
> Largest phase — 4 sub-stages (18A-18D), each with its own checkpoint-lite
> (per-phase gate run, WORKFLOW note). Do NOT attempt as one pass.

## Goal
Governance layer: virtual keys with budgets, teams, routing rules, guardrails,
model limits, prompt templates, alerts, feature flags, backup/restore.

## Architecture
- `internal/governance/` — virtual keys, budgets, hierarchical limits.
  Defines its own repository interface; enforcement is pure domain logic.
- `internal/guardrails/` — blocklist + PII redaction. Pure functions over
  request text, config injected.
- `internal/alerts/` — channel dispatch (webhook/discord/telegram), event
  types, retry policy. No store imports — config passed in.
- Handlers stay thin CRUD; middleware calls into governance.

## Deferred (decided now, revisit post-stage)
- **Adaptive routing (heuristic classifier)** — g0router already ships an
  `auto` combo strategy with a 5-category task classifier. Duplicate. Skip.
- **OTel distributed tracing** — optional infra, no UI dependency. Skip.
- **MCP tool groups** → moved here from "maybe": kept, small (18C).
- **Custom pricing UI enhancements** — UI-only, Lovable's job.

## Sub-Stage 18A: Virtual Keys, Teams, Governance Middleware

### Tables
```sql
CREATE TABLE IF NOT EXISTS teams (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    budget_usd REAL,
    budget_period TEXT DEFAULT 'monthly',   -- 'daily' | 'weekly' | 'monthly'
    budget_used_usd REAL DEFAULT 0,
    budget_reset_at DATETIME,
    rate_limit_rpm INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS virtual_keys (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    key_prefix TEXT NOT NULL,
    key_hash TEXT NOT NULL,                 -- sha256, same scheme as api_keys
    budget_usd REAL,
    budget_period TEXT DEFAULT 'monthly',
    budget_used_usd REAL DEFAULT 0,
    budget_reset_at DATETIME,
    rate_limit_rpm INTEGER,
    rate_limit_tpm INTEGER,
    team_id INTEGER,
    is_active INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Endpoints
- `GET/POST /api/virtual-keys`, `PUT/DELETE /api/virtual-keys/:id` (raw key returned once on create)
- `GET/POST /api/teams`, `PUT/DELETE /api/teams/:id`

### Enforcement (in `/v1/*` auth middleware)
- Virtual key checked BEFORE regular API keys (prefix-distinguished: `gvk-`).
- Pre-request: active? budget remaining? RPM/TPM within limit? Team budget/RPM
  remaining (hierarchical: key limit AND team limit must both pass)?
  Reject `429` (limits) / `403` (budget exhausted, inactive).
- Post-request: accumulate `budget_used_usd` on key AND team from computed
  request cost; attribute usage in request_log (new `virtual_key_id` column
  via ensureColumn).
- Budget reset: lazy — on check, if `now > budget_reset_at`, zero used + roll
  period forward.

### Tests (minimum)
- Raw vkey works on `/v1/*`; regular keys unaffected
- Budget exhaustion → 403; resets after period rollover
- Key RPM passes but team RPM exceeded → 429 (hierarchical)
- Usage accumulates on key + team after request; survives restart
- Create returns raw key once; list returns prefix only

## Sub-Stage 18B: Routing Rules, Model Limits

### Tables
```sql
CREATE TABLE IF NOT EXISTS routing_rules (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    priority INTEGER NOT NULL DEFAULT 0,
    cond_field TEXT NOT NULL,      -- 'model' | 'provider' | 'header'
    cond_operator TEXT NOT NULL,   -- 'equals' | 'contains' | 'starts_with'
    cond_value TEXT NOT NULL,
    target_provider TEXT NOT NULL,
    target_model TEXT,
    is_active INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS model_limits (
    id INTEGER PRIMARY KEY,
    model TEXT NOT NULL,
    max_tokens INTEGER,
    max_rpm INTEGER,
    allowed_key_ids TEXT,          -- JSON array, empty = all
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(model)
);
```

### Endpoints
- `GET/POST /api/routing-rules`, `PUT/DELETE /api/routing-rules/:id`
- `GET/POST /api/model-limits`, `PUT/DELETE /api/model-limits/:id`

### Enforcement
- Routing rules evaluated in priority order BEFORE alias/combo resolution in
  `internal/proxy` routing; first match rewrites target. Read alias TTL-cache
  pattern; rules cached the same way, invalidated on write.
- Model limits checked at dispatch: token cap clamps/rejects `max_tokens`,
  RPM via in-memory limiter, key allowlist → 403.

### Tests (minimum)
- Priority order respected; inactive rule skipped; no-match falls through to existing routing
- Cache invalidates on rule write
- max_tokens above limit → 400 (or clamp — pick one, document); disallowed key → 403

## Sub-Stage 18C: Guardrails, PII, Prompts, MCP Tool Groups

### Tables
```sql
CREATE TABLE IF NOT EXISTS prompt_templates (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    system_prompt TEXT,
    user_prompt_template TEXT,
    variables_json TEXT,
    is_active INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS mcp_tool_groups (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    tool_names_json TEXT NOT NULL DEFAULT '[]',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```
Guardrails config: settings keys (`guardrails_enabled`, `guardrails_blocklist_json`,
`pii_redaction_enabled`, `pii_types_json`) — no table.

### Endpoints
- `GET/PUT /api/guardrails` — config; `POST /api/guardrails/test` — `{prompt}` → `{blocked, redacted_prompt, matches}`
- `GET/POST /api/prompt-templates`, `PUT/DELETE /api/prompt-templates/:id`
- `GET/POST /api/mcp/tool-groups`, `PUT/DELETE /api/mcp/tool-groups/:id`

### Enforcement `[flag: guardrails]` `[flag: pii_redaction]`
- In dispatch pipeline BEFORE RTK/Caveman: blocklist match → 400 with reason;
  PII redaction rewrites matched spans (`email`, `phone`, `ssn`, `credit_card`,
  `ip_address` regex set) with `[REDACTED:<type>]`.
- Tool groups filter MCP tool injection set when a group is selected on a key/combo.

### Tests (minimum)
- Blocklist blocks (case-insensitive); flag off → pass-through
- Each PII type redacts; non-PII text untouched; streaming + non-streaming paths
- Template variable extraction from `{{var}}` syntax
- Tool group filters injected tool list

## Sub-Stage 18D: Alerts, Feature Flags, Backup/Restore

### Tables
```sql
CREATE TABLE IF NOT EXISTS alert_channels (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    channel_type TEXT NOT NULL,    -- 'webhook' | 'discord' | 'telegram'
    config_enc TEXT NOT NULL,      -- encrypted (URLs/tokens)
    events_json TEXT NOT NULL DEFAULT '[]',  -- ['quota_depleted','connection_stale','rate_limit','budget_exhausted']
    is_active INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS feature_flags (
    id INTEGER PRIMARY KEY,
    key TEXT NOT NULL UNIQUE,
    enabled INTEGER DEFAULT 0,
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```
Flags seeded in migration: `semantic_cache`, `guardrails`, `pii_redaction`,
`websocket_chat`, `mitm_proxy` (all 0).

### Endpoints
- `GET/POST /api/alert-channels`, `PUT/DELETE /api/alert-channels/:id`, `POST /api/alert-channels/:id/test`
- `GET /api/feature-flags`, `PUT /api/feature-flags/:id` — toggle only (no user-created flags; no POST/DELETE)
- `POST /api/settings/backup` — JSON export
- `POST /api/settings/restore` — JSON import

### Backup/Restore Security Rules
- Export **excludes** all secret material: api key hashes exported (needed),
  but OAuth tokens, proxy passwords, tunnel configs, alert configs exported
  ONLY as `null` placeholders + a `redacted_fields` manifest. Restore keeps
  existing secrets when placeholder is null.
- Endpoint: admin-session or bearer auth; both audited; restore validates
  schema version and rejects unknown shape with 400.

### Tests (minimum)
- Alert dispatch to fake webhook on event; inactive channel skipped; test endpoint sends
- Flag toggle persists; unknown flag → 404
- Backup output contains no token/password values (assert scan)
- Restore round-trip preserves data; null placeholders keep existing secrets; bad payload → 400

## Tasks
1. `phase-18/task-1..3` (18A): teams+vkeys store / governance domain / middleware wiring
2. `phase-18/task-4..5` (18B): routing rules + model limits (store/domain/proxy wiring)
3. `phase-18/task-6..8` (18C): guardrails domain / prompts / tool groups
4. `phase-18/task-9..11` (18D): alerts / flags / backup-restore
5. `phase-18/checkpoint` (incl. security pass)

## Commit Message (final)
`phase-18/bifrost-features: governance, routing rules, guardrails, alerts`
