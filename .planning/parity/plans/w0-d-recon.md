# w0-d recon — UI scaffolding (rev 3)

Comparison artifact for the Kimi diff gate. Generated BEFORE any code change.
The two lists below are the ground truth: types.ts exports must equal recon-1
exactly, and route files must exist for every recon-2 path (no extras).

## Recon-1 — `lib/types` symbols imported by e2e mocks and src

Source command: `grep -rh "lib/types" ui/e2e ui/src | sort -u`
Parsed with: `grep -oE '\{[^}]+\}' | tr ',' '\n' | strip braces/whitespace | sort -u`

AUD-075 cited 36 (count of `import type { ... } from "lib/types"` *lines*).
True unique-symbol count is 29 — the per-line dedup of multi-symbol imports.

```
AlertChannel
Alias
ApiKey
AuditLog
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
Skill
Team
Tunnel
UsageLog
User
VirtualKey
```

## Recon-2 — routes e2e specs visit

Source command: `grep -rhoE 'page\.goto\("[^"]+"' ui/e2e | sed -E 's/page\.goto\("//; s/"$//' | sort -u`

Excludes `page.goto(route.path)` (variable navigation in `ui/e2e/helpers.ts:745`)
because that pattern resolves to the per-spec route list at runtime, not a
literal path. `helpers.ts` itself only hard-codes `/login`.

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
- `internal/admin/providers.go`: `id, name, type, base_url, enabled, created_at,
  updated_at` — all snake_case
- `internal/admin/handlers.go`: uses `{data, error}` envelope pattern

Implication for types.ts: every field name is `snake_case`; names are NOT
camelCased in the wire format. The TypeScript types must reflect snake_case
field names so the eventual API wrapper (Wave 6) can map them without a
renaming pass.

## Out of scope for this plan

- API client wrapper (Wave 6) — types are declared as opaque `interface`s,
  no runtime fetch code, no `fetch` / `axios` / `react-query` calls.
- Auth/login wiring — `/login` is a route shell only.
- i18n, styling, page content, data fetching.
- New e2e specs or test infra.
- Go code or backend DTO changes.
