# w0-d recon — UI scaffolding (rev 3)

Comparison artifact for the Kimi diff gate. Generated BEFORE any code change.
The two lists below are the ground truth: types.ts exports must equal recon-1
exactly, and route files must exist for every recon-2 path (no extras).

## Recon-1 — `lib/types` symbols imported by e2e mocks and src

Source command (multi-line safe — joins continuation lines, extracts every
`import type { ... } from "...lib/types"` symbol across all files in
`ui/e2e/` and `ui/src/`):

```python
import re, os
symbols = set()
for root, _, files in os.walk('ui/e2e'):
    for f in files:
        if f.endswith(('.ts', '.tsx')):
            with open(os.path.join(root, f)) as fh:
                for m in re.finditer(r'import type\s*\{([^}]+)\}\s*from\s*[\"\\'][^\"\\']*lib/types', fh.read()):
                    for sym in m.group(1).split(','):
                        sym = sym.strip()
                        if sym: symbols.add(sym)
# (same loop for ui/src/)
for s in sorted(symbols): print(s)
```

AUD-075 cited 36 (count of `import type { ... } from "lib/types"` *lines*,
including the `} from` closer of multi-line imports). True unique-symbol
count is **32** — the multi-line import in `ui/e2e/mocks/store.ts` is a
single statement spread across 33 lines, so an awk-based single-line parser
missed all 32 symbols inside it. The three types absent from any single-line
import block are `AuthStatus`, `Settings`, and `TrafficEvent`.

32 symbols:

```
AlertChannel
Alias
ApiKey
AuditLog
AuthStatus
ChatSession
Combo
Connection
ConsoleLogEntry
FeatureFlag
Guardrails
McpClient
McpInstance
McpTool
McpToolGroup
MitmTool
Model
ModelLimit
PricingOverride
PromptTemplate
Provider
ProxyPool
Quota
RoutingRule
Settings
Skill
Team
TrafficEvent
Tunnel
UsageLog
User
VirtualKey
```

## Recon-2 — routes e2e specs visit

Source command: `grep -rhoE 'page\.goto\("[^"]+"' ui/e2e | sed -E 's/page\.goto\("//; s/"$//' | sort -u`

Excludes `page.goto(route.path)` (variable navigation in
`ui/e2e/helpers.ts:745`) because that pattern resolves to the per-spec route
list at runtime, not a literal path. `helpers.ts` itself only hard-codes
`/login`.

31 paths:

```
/alerts
/aliases
/audit
/chat
/combos
/connections
/console
/dashboard
/endpoint
/feature-flags
/guardrails
/keys
/login
/logs
/mcp
/mcp/tools
/mitm
/model-limits
/models
/pricing
/prompts
/providers
/proxy-pools
/quota
/routing-rules
/settings
/teams
/traffic
/tunnels
/usage
/virtual-keys
```

## Recon-3 — backend JSON-tag conventions

Read for type-shape reference only (AUD-075 scope: types must mirror snake_case
`{data, error}` envelope from the admin API).

- `internal/admin/connections.go`: `id, provider_id, name, kind, secret_set,
  access_token_set, refresh_token_set, expires_at, metadata, created_at,
  updated_at` — all snake_case
- `internal/admin/providers.go`: `id, name, type, base_url, enabled,
  created_at, updated_at` — all snake_case
- `internal/admin/handlers.go`: uses `{data, error}` envelope pattern

Implication for types.ts: every field name is `snake_case`; names are NOT
camelCased in the wire format. The TypeScript types must reflect snake_case
field names so the eventual API wrapper (Wave 6) can map them without a
renaming pass.

## Field shapes (from `ui/e2e/mocks/seed/*.ts`)

Observed field shapes for all 32 types, extracted from the seed files that
import each type. These are the minimum field set the e2e mocks exercise;
the eventual API client (Wave 6) may add more.

```
AlertChannel     { id: number, name: string, channel_type: string, config: Record<string, unknown>, events: string[], is_active: boolean, created_at: string }
Alias            { id: string, alias: string, provider: string, model: string }
ApiKey           { id: string, name: string, prefix: string, full_key?: string, scopes: string[], rpm_limit?: number, tpm_limit?: number, daily_spend_cap?: number, is_active: boolean, created_at: string }
AuditLog         { id: string, timestamp: string, actor: string, action: string, target: string, details?: string }
AuthStatus       { require_login: boolean, has_users: boolean, authenticated: boolean, username: string, display_name: string, role: string }
ChatSession      { id: string, title: string, model: string, provider: string, messages: Array<{ role: string, content: string }>, created_at: string, updated_at: string }
Combo            { id: string, name: string, strategy: string, steps: Array<{ provider: string, model: string }>, is_active: boolean }
Connection       { id: string, provider: string, name: string, auth_type: string, is_active: boolean, models: string[], priority: number, needs_reauth: boolean }
ConsoleLogEntry  { timestamp: string, level: string, message: string }
FeatureFlag      { id: number, key: string, enabled: boolean, description: string, created_at: string }
Guardrails       { guardrails_enabled: boolean, guardrails_blocklist: string[], pii_redaction_enabled: boolean, pii_redaction_types: string[] }
McpClient        { ID: string, Name: string, Transport: string, Command?: string, Args?: string[], Env?: Record<string, string>, URL?: string, IsActive: boolean, HealthStatus: string, CreatedAt: string }
McpInstance      { ID: string, Name: string, Transport: string, Command?: string, Args?: string[], IsActive: boolean, HealthStatus: string, CreatedAt: string }
McpTool          { type: string, function: { name: string, description: string, parameters: Record<string, unknown> } }
McpToolGroup     { id: number, name: string, tool_ids: string[], is_active: boolean, created_at: string }
MitmTool         { id: string, name: string, enabled: boolean, dns_override: string, status: "active" | "inactive" }
Model            { id: string, provider: string, name: string, input_cost: number, output_cost: number, context_window: number, is_disabled: boolean, is_custom: boolean }
ModelLimit       { id: number, model: string, max_tokens: number, max_rpm: number, allowed_key_ids: string[], created_at: string }
PricingOverride  { id: string, provider: string, model: string, input_cost: number, output_cost: number }
PromptTemplate   { id: number, name: string, system_prompt: string, models: string[], is_active: boolean, created_at: string }
Provider         { id: string, name: string, display_name: string, description: string, auth_types: string[], capabilities: string[], connection_count: number, status: string }
ProxyPool        { id: string, name: string, protocol: string, host: string, port: number, username: string, is_active: boolean, last_check_at: string, last_check_status: string }
Quota            { connection_id: string, provider: string, connection_name: string, account_label?: string, plan: string, used: number, limit: number, unit: string, reset_at: string, is_active: boolean }
RoutingRule      { id: string, name: string, priority: number, cond_field: string, cond_operator: string, cond_value: string, target_provider: string, is_active: boolean, created_at: string }
Settings         { require_api_key: boolean, require_login: boolean, rtk_enabled: boolean, caveman_enabled: boolean, caveman_level: string, enable_request_logs: boolean, log_retention_days: number, cache_enabled: boolean, cache_ttl_seconds: number, proxy_url: string, notify_webhook_url: string, notify_on_reauth: boolean, allowed_sources: string[], tunnel_dashboard_access: boolean, theme: string, language: string }
Skill            { name: string, category: string, description: string, url: string }
Team             { id: string, name: string, budget_usd: number, budget_used_usd: number, budget_period: string, rate_limit_rpm: number }
TrafficEvent     {}  (no field references observed in seed; opaque for now)
Tunnel           { type: string, is_enabled: boolean, url: string, status: string }
UsageLog         { id: string, timestamp: string, provider: string, model: string, api_key_id: string, api_key_name: string, status: string, status_code: number, prompt_tokens: number, completion_tokens: number, total_tokens: number, cost_usd: number, latency_ms: number, rtk_enabled: boolean, caveman_enabled: boolean }
User             { id: string, username: string, display_name: string, role: string, password?: string }
VirtualKey       { id: string, name: string, prefix: string, budget_usd: number, budget_used_usd: number, budget_period: string, rate_limit_rpm: number, is_active: boolean }
```

## Out of scope for this plan

- API client wrapper (Wave 6) — types are declared as opaque `interface`s,
  no runtime fetch code, no `fetch` / `axios` / `react-query` calls.
- Auth/login wiring — `/login` is a route shell only.
- i18n, styling, page content, data fetching.
- New e2e specs or test infra.
- Go code or backend DTO changes.
