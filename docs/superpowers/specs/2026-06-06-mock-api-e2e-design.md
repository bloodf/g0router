# Mock API Layer for Playwright E2E Tests

## Context

The g0router dashboard is a React 19 + Vite SPA that communicates with a Go fastHTTP backend. We already have a partial Playwright mock layer in `ui/e2e/mocks/` covering ~30 routes. The real backend exposes ~70 API routes. The goal is a 1:1 mock of every route the UI actually calls, so E2E tests can validate every screen, button, form, input, modal, and dialog without starting the Go server.

## Scope

### In Scope (Phase 1)

Mock every route the current UI screens exercise:

| Domain | Routes |
|--------|--------|
| Auth | `/api/auth/status`, `/api/auth/login`, `/api/auth/logout`, `/api/auth/setup`, `/api/auth/password`, `/api/auth/users` |
| Settings | `/api/settings` (GET/PUT) |
| Providers | `/api/providers`, `/api/providers/:id`, `/api/providers/:id/connections`, `/api/providers/:id/models`, `/api/providers/:id/suggested-models`, `/api/providers/test-batch` |
| Connections | `/api/connections`, `/api/connections/:id`, `/api/connections/:id/test`, `/api/connections/bulk-enable`, `/api/connections/bulk-disable` |
| Keys | `/api/keys`, `/api/keys/:id`, `/api/keys/:id/regenerate` |
| Virtual Keys | `/api/virtual-keys`, `/api/virtual-keys/:id` |
| Models | `/api/models`, `/api/models/disabled`, `/api/models/custom`, `/api/models/custom/:id` |
| Model Limits | `/api/model-limits`, `/api/model-limits/:id` |
| Combos | `/api/combos`, `/api/combos/:id` |
| Aliases | `/api/aliases`, `/api/aliases/:id` |
| Pricing | `/api/pricing`, `/api/pricing/:provider/:model` |
| Routing Rules | `/api/routing-rules`, `/api/routing-rules/:id` |
| Teams | `/api/teams`, `/api/teams/:id` |
| Tunnels | `/api/tunnels`, `/api/tunnels/:type` |
| Usage | `/api/usage`, `/api/usage/summary`, `/api/usage/chart` |
| Quota | `/api/quota` |
| Logs | `/api/logs` |
| Audit | `/api/audit` |
| Chat Sessions | `/api/chat-sessions`, `/api/chat-sessions/:id` |
| Guardrails | `/api/guardrails`, `/api/guardrails/test` |
| Prompt Templates | `/api/prompt-templates`, `/api/prompt-templates/:id`, `/api/prompt-templates/test` |
| Alert Channels | `/api/alert-channels`, `/api/alert-channels/:id`, `/api/alert-channels/:id/test` |
| Feature Flags | `/api/feature-flags`, `/api/feature-flags/:id` |
| MCP | `/api/mcp/clients`, `/api/mcp/clients/:id`, `/api/mcp/instances`, `/api/mcp/instances/:id`, `/api/mcp/instances/:id/accounts`, `/api/mcp/instances/:id/auth/start`, `/api/mcp/tools`, `/api/mcp/tools/:id/execute`, `/api/mcp/tool-groups`, `/api/mcp/tool-groups/:id` |
| Proxy Pools | `/api/proxy-pools`, `/api/proxy-pools/:id`, `/api/proxy-pools/batch`, `/api/proxy-pools/:id/test` |
| MITM | `/api/mitm/status`, `/api/mitm/toggle`, `/api/mitm/ca-cert`, `/api/mitm/tools/:id` |
| Diagnostics | `/api/version`, `/api/skills`, `/api/locale` |
| Streams | `/api/traffic/stream`, `/api/console-logs/stream`, `/api/console-logs` |
| Inference | `/v1/chat/completions` |

### Out of Scope (Phase 1)

Routes not exercised by the current dashboard UI. These may be added later:

- OAuth flows (`/api/oauth/*`, `/api/mcp/oauth/callback`, `/api/mcp/instances/:id/oauth/complete`)
- Settings extras (`/api/settings/proxy-test`, `/api/settings/backup`, `/api/settings/restore`)
- Update (`/api/update/check`, `/api/update/apply`)
- Semantic cache (`/api/cache/semantic`)
- WebSocket (`/api/ws`)
- Image/audio/embeddings endpoints (`/v1/images/generations`, `/v1/audio/*`, `/v1/embeddings`)
- Provider model test (`/api/providers/:id/models/:model/test`)

## Architecture

### Technology: Playwright `page.route()`

We keep the existing Playwright interception approach. No new dependencies. The mock runs in the Playwright Node process, not in the browser, which means:

- Direct access to Node APIs and the test process
- Easy to inspect/modify `MockStore` from test code
- No service-worker or CORS complexity
- Fast — no extra server to start

### Directory Structure

```
ui/e2e/mocks/
  fixture.ts            # Test fixture (worker-scoped store + page.route setup)
  store.ts              # MockStore class — in-memory DB for all domains
  seed/
    index.ts            # Re-export all seeders
    auth.ts             # seedUsers()
    providers.ts        # seedProviders()
    connections.ts      # seedConnections()
    ...                 # One seeder per domain
  handlers/
    index.ts            # setupMockApi(page, store) — compose all domains
    auth.ts             # registerAuthHandlers(page, store)
    providers.ts
    connections.ts
    keys.ts
    virtual-keys.ts
    models.ts
    combos.ts
    aliases.ts
    pricing.ts
    routing-rules.ts
    teams.ts
    tunnels.ts
    usage.ts
    quota.ts
    audit.ts
    logs.ts
    chat-sessions.ts
    settings.ts
    guardrails.ts
    prompts.ts
    alert-channels.ts
    model-limits.ts
    feature-flags.ts
    mcp.ts
    proxy-pools.ts
    mitm.ts
    skills.ts
    version.ts
    inference.ts
    streams.ts
    utils.ts            # json(), error(), routeMatch helpers
```

### MockStore

A single worker-scoped class holding in-memory state:

- `auth: AuthStatus`
- `users: User[]`
- `settings: Settings`
- `providers = new Map<string, Provider>()`
- `connections = new Map<string, Connection>()`
- `keys = new Map<string, ApiKey>()`
- `virtualKeys = new Map<string, VirtualKey>()`
- `models = new Map<string, Model>()`
- `disabledModels: string[]`
- `customModels: CustomModel[]`
- `modelLimits = new Map<string, ModelLimit>()`
- `combos = new Map<string, Combo>()`
- `aliases = new Map<string, Alias>()`
- `pricing = new Map<string, PricingOverride>()`
- `routingRules = new Map<string, RoutingRule>()`
- `teams = new Map<string, Team>()`
- `tunnels = new Map<string, Tunnel>()`
- `usageLogs: UsageLog[]`
- `quotas: Quota[]`
- `auditLogs: AuditLog[]`
- `chatSessions: ChatSession[]`
- `guardrails: GuardrailsConfig`
- `promptTemplates = new Map<string, PromptTemplate>()`
- `alertChannels = new Map<string, AlertChannel>()`
- `featureFlags = new Map<string, FeatureFlag>()`
- `mcpClients = new Map<string, MCPClient>()`
- `mcpInstances = new Map<string, MCPInstance>()`
- `mcpTools: MCPTool[]`
- `mcpToolGroups = new Map<string, MCPToolGroup>()`
- `proxyPools = new Map<string, ProxyPool>()`
- `mitmStatus: MITMStatus`
- `skills: Skill[]`
- `consoleLogs: ConsoleLogEntry[]`
- `nextId(): string` — auto-incrementing mock ID generator
- `reset()` — clear all state
- `seedAll()` — call all seeders

### Handler Behavior

Each domain handler follows these rules:

1. **Route matching** uses exact pathname + method checks. Path params extracted via `pathname.split("/")`.
2. **Request body** read via `request.postDataJSON()`.
3. **Validation** checks required fields and returns `400` with `{error: "..."}` if missing.
4. **Persistence** updates `MockStore` Maps/arrays in place.
5. **Response envelope** is always `{data: T}` for success, `{error: string}` for errors.
6. **Snake_case** keys in all JSON responses to match the Go backend contract.
7. **GET after mutation** returns the updated entity from store.
8. **Lists** return arrays; paginated lists return `{items: T[], total: number}`.

### Seeding Strategy

Each test worker gets a fresh seeded `MockStore`. Auth resets to `authenticated: false` before each test (existing behavior). Entity data persists across serial tests in the same worker.

Seed data must be realistic enough to render every dashboard screen:
- **20 providers** (mix of active/inactive)
- **20 models** across providers
- **6 connections** linked to providers
- **2 API keys**, **2 virtual keys**, **2 teams**
- **25 usage logs**, **5 audit logs**, **4 quotas**
- **2 chat sessions** with messages
- **2 combos**, **3 aliases**, **2 pricing overrides**, **2 routing rules**
- **2 prompt templates**, **2 alert channels**, **2 feature flags**
- **2 MCP clients**, **2 MCP instances**, **4 MCP tools**, **2 tool groups**
- **2 proxy pools**
- MITM status: disabled, with mock CA cert string

### Streaming Endpoints

- **Traffic SSE** (`/api/traffic/stream`): returns a single SSE event with a mock `TrafficEvent`, then closes. Content-Type `text/event-stream`.
- **Console SSE** (`/api/console-logs/stream`): returns a single SSE event with a mock `ConsoleLogEntry`, then closes.
- **Chat completions** (`/v1/chat/completions`): reads `messages` from body, returns 3 SSE chunks (role, content, stop) plus `[DONE]`. Content-Type `text/event-stream`.

### Error Scenarios

The mock should support intentional error injection for edge-case testing:

- `store.forceErrors.add("/api/keys")` — next request to that route returns 500
- `store.latencyMs = 200` — optional artificial delay for all routes
- These are optional enhancements; baseline is happy-path + basic validation.

## E2E Test Expansion

After the mock is complete, expand `comprehensive.spec.ts`:

1. **Unskip** routing-rules CRUD (form now has condition fields)
2. **Connections CRUD**: create connection on provider detail, test it, delete it
3. **Model-limits CRUD**: create, edit, delete
4. **Alert-channels CRUD**: create, test, delete
5. **Feature-flags**: toggle enabled/disabled
6. **Guardrails**: update config, run test prompt
7. **Prompt-templates**: create, test, delete
8. **MCP**: create instance, view tools, execute tool
9. **Proxy-pools**: create, test, delete
10. **MITM**: toggle status, download CA cert
11. **Chat**: send message, verify streamed mock response appears
12. **Chat sessions**: create session, rename, delete

## Trade-offs Considered

| Approach | Pros | Cons | Decision |
|----------|------|------|----------|
| **Playwright `page.route()`** (chosen) | No new deps; direct store access from tests; fast; already exists | Interception logic lives in Node, not browser | ✅ Keep and extend |
| MSW service worker | More realistic network stack; can use in Storybook | Adds dependency; SW registration complexity; harder to share state with tests | ❌ Not needed |
| Vite dev-server plugin | Clean separation; could reuse for local dev | Requires running separate mock server; more moving parts in CI | ❌ Overkill |

## Risks & Mitigations

| Risk | Mitigation |
|------|------------|
| Mock drifts from real backend contract | Keep `MockStore` types imported from `src/lib/types.ts`. Add a CI step that diffs API route list against mock handlers. |
| Seed data becomes stale | Seeders are co-located with mocks; update them when UI adds new fields. |
| Tests become flaky due to shared worker store | Serial test mode for mutating tests; `fixture.ts` already resets auth per test. |
| Large handler file becomes unmaintainable | Split into per-domain files (this design). |

## Success Criteria

1. `npm run e2e` passes without starting the Go backend.
2. Every dashboard route loads without 404s in the mock.
3. CRUD operations on all 15+ entity types persist and reflect in the UI.
4. Chat streaming returns mock responses that render in the UI.
5. `comprehensive.spec.ts` covers all major screens and interactions.
