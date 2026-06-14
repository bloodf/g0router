# Micro-plan w7-mcp-3 — MCP admin transport + tools + skills + agent loop (Go)

```
wave: 7
plan: w7-mcp-3
status: READY (rev 1 — authored against the SHIPPED w7-mcp-1 foundation
  (store + launcher + bridge + filter + allowlist + defaults) AND the SHIPPED
  w7-mcp-2 client engine (probe + registry + OAuth Engine + SSE/message transport +
  health + discovery) — ALL LIVE in-tree @ internal/mcp/{runner,bridge,launcher,
  filter,allowlist,defaults,process,probe,registry,oauth,sse,healthmonitor,
  discovery}.go + internal/store/{mcpinstances,mcpoauth}.go. PRIMARY template =
  the SHIPPED w7-gov-1 (admin-CRUD + serial-slot routes_admin + e2e-mock-
  reconciliation): same admin pattern (newTestEnv proof surface, writeData/
  writeError {data,error} envelope, h.recordAudit best-effort, additive
  routes_admin block in ONE commit). w7-gov-3 `alerts.go` is the freshest sibling
  for the recordAudit + injectable-Sender seam. live tree @ <base>;
  WAVE-7-MAP w7-mcp-3 row ~line 182; MCP track split §207-208; serial chain
  §219-224 (mcp-3 is the ONLY MCP route holder); reconciliation §245; freeze
  rules §267.)
runs: MCP track. EXTENDS the greenfield internal/mcp package (agent.go) + ADDS the
  internal/admin MCP handlers — disjoint from every other domain/store/admin file;
  runs ∥ governance + providers + platform tracks. INTERNALLY SERIAL:
  w7-mcp-1 (SHIPPED — foundation) ──▶ w7-mcp-2 (SHIPPED — client engine) ──▶
  w7-mcp-3 (THIS — admin transport + tools + skills + agent loop). DEPENDS on
  w7-mcp-1's NewLauncher/SetRunner + the Bridge/SessionSink seam + the *MCP* store
  methods + DefaultPlugins, AND w7-mcp-2's Probe/Registry/Engine/sseClient/discovery/
  health. THIS plan TAKES the routes_admin.go SERIAL SLOT (MAP §182/§208/§221) —
  the ONLY MCP plan that does so.
branch: main (direct commits, per repo convention)
commit-prefix: phase-1/w7-mcp-3:
ref-source: 9router frozen @ 827e5c3 — the MCP admin + tool-execute + agent surfaces;
  the BINDING contract for W7 is the W6-l e2e mock (decision 1: real Go wins, mock
  corrected to mirror it — BUT the w6-l UI page is FROZEN, so where the page already
  consumes a fixed key-casing the Go DTO matches that casing; §1.2). Mock sources:
    ui/e2e/mocks/handlers/{mcp,skills}.ts + seed/{mcp,skills}.ts;
    specs ui/e2e/mcp.spec.ts + ui/e2e/skills.spec.ts.
  9router ref rows (from .planning/parity/matrix/9router-mcp.md): PAR-MCP-018/019/045/
    046/047/048/049/050 (Cowork-settings managed-server / toolPolicy generation —
    cowork-settings/route.js + coworkPlugins.js); PAR-MCP-022 (custom plugin URL-vs-
    command mode — buildCustomEntries); PAR-MCP-040 (agent loop — no agent.go today);
    PAR-MCP-041 (MCP UI pages — already shipped variant-HAVE in w6-l); PAR-MCP-060
    (antigravity hardcoded-unavailable tool ride-along).
base: <base> = git rev-parse HEAD recorded at P0. Substitute the actual SHA
  everywhere §5 says <base>. (At authoring, HEAD = 0907979; recompute at P0.)
go-serial-slot: this plan holds the SINGLE in-flight edit to
  internal/server/routes_admin.go while live (MAP decision 3; W3/W4/W5/W6 lesson).
  Slot must be FREE at P5 (chain: …→w7-gov-3→**w7-mcp-3**→w7-plat-1; MAP §221)
  before T-routes. mcp-3 TAKES the slot, RELEASES it to w7-plat-1 on close.
new-route: NO UI route files. All four UI surfaces (/mcp clients+instances, /mcp/tools
  tools+tool-groups, the McpMarketplaceModal, the NEW /skills route) ALREADY SHIPPED
  in w6-l against mocks; this plan builds the REAL Go so the pages flip
  variant-HAVE → true-HAVE and corrects the mock bodies ONLY IF the real Go diverges
  (default: match the consumed casing → minimal/no mock change; §1.2).
```

---

## 1. Scope — PAR rows + the five surfaces

### Rows this plan closes

| Row / item | Claim (from `9router-mcp.md` / open-questions) | Target state after w7-mcp-3 |
|---|---|---|
| open-questions w6-l **ESC-1a** (mcp clients/instances + marketplace backend absent) | real `/api/mcp/clients` + `/api/mcp/instances` CRUD + OAuth `…/auth/start` | true-HAVE (Go — NEW `internal/admin/mcp.go` over the SHIPPED `store.*MCPClient*`/`*MCPInstance*` + the w7-mcp-2 `Engine.Start`, §1.4/§1.5) |
| open-questions w6-l **ESC-1b** (mcp tools/tool-groups backend absent) | real `GET /api/mcp/tools` + `POST …/{name}/execute` + `/api/mcp/tool-groups` CRUD | true-HAVE (Go — `internal/admin/mcp.go` tools list via the discovery/probe cache + execute via the agent/bridge, tool-groups over a NEW additive store table, §1.6/§1.7) |
| open-questions w6-l **ESC-1c** (skills backend absent) | real `GET /api/skills` catalog | true-HAVE (Go — NEW `internal/admin/skills.go` over a skills catalog, §1.8) |
| PAR-MCP-022 | Custom plugins support BOTH url (HTTP/SSE) and command (stdio) modes (`cowork-settings/route.js:179` `buildCustomEntries`) | HAVE (the `CreateInstance` handler branches on `url` vs `command`; command path is allowlist-gated via the SHIPPED `Launcher.StartStdio`, url path via `StartHTTP`/`StartSSE`; §1.5) |
| PAR-MCP-040 | No MCP agent loop (`internal/mcp/` no agent files) | HAVE (NEW `internal/mcp/agent.go`: a bounded multi-turn tool-execution loop primitive; the tool-execute path + the loop are the parity bar, §1.9) |
| PAR-MCP-041 | No MCP UI pages | HAVE (UI shipped w6-l; this plan supplies the real Go so it flips variant-HAVE → true-HAVE) |
| PAR-MCP-018 (subset) | Managed-server list shape `{name,url,transport,oauth?,toolPolicy?}` (`cowork-settings/route.js:262`) | HAVE-subset (the managed-server DTO assembly from instances + the per-tool `toolPolicy` map is ported as a PURE builder `buildManagedServers`/`buildToolPolicy`; §1.6 / ESC-TOOLPOLICY-SCOPE) |
| PAR-MCP-019 (subset) | toolPolicy emits bare + `{name}-{tool}` prefixed names as `"allow"` (`…:171`) | HAVE-subset (PURE `buildToolPolicy(serverName, toolNames)` emits both forms; §1.6) |
| PAR-MCP-045 | `buildManagedMcpServers` strips `{name}-` prefixes idempotently (`coworkPlugins.js:48`) | HAVE (PURE `stripServerPrefix(name, tool)` while-loop strips repeated prefixes; §1.6) |
| PAR-MCP-046 | Settings GET strips `{name}-` prefixes when returning tool names (`…:288`) | HAVE (the same PURE `stripServerPrefix` applied on the tools read; §1.6) |
| PAR-MCP-047 | Settings GET prefers default `toolNames` over stored toolPolicy keys (`…:297`) | HAVE (PURE precedence: `def.ToolNames` when present, else the derived bare set; §1.6) |
| PAR-MCP-048 | Settings POST sanitizes custom plugin names to `[^a-zA-Z0-9_-]`, truncates to 64 (`…:339`) | HAVE (PURE `sanitizePluginName(s) string` — regex-strip + 64-char cap; applied on `CreateClient`/`CreateInstance`; §1.5) |
| PAR-MCP-049 | Settings POST filters custom plugins by allowlist before registration (`…:337`) | HAVE (the command path reuses the SHIPPED `isAllowedCommand` via `Launcher.StartStdio`'s pre-spawn gate — the registration rejects before any spawn; §1.5) |
| PAR-MCP-050 | Settings DELETE resets config + skip-approvals + legacy cleanup (`…:396`) | HAVE-subset (the `DeleteInstance` handler removes the instance + best-effort stops the launcher bridge; the Cowork 3p-config/skip-approvals/1p-legacy file rewrite is NOT applicable to g0router — ESC-COWORK-CONFIG; §1.5) |
| PAR-MCP-060 | Antigravity hardcodes `mcp_sequential-thinking_sequentialthinking` as unavailable (`antigravity.js:433`) | HAVE-ride-along (a single PURE constant `unavailableAntigravityTool` + its inclusion in the tools list as `{unavailable:true}`; the antigravity EXECUTOR itself is w7-prov-special — this is the tool-definition ride-along only; §1.6 / ESC-ANTIGRAVITY) |

Matrix flips at closeout (§4 T-close): in `.planning/parity/matrix/9router-mcp.md`,
PAR-MCP-022,040,041,045,046,047,048,049 → HAVE (real Go; the live SSE stream + the
real agent LLM round-trip are integration-only behind injected seams — §1.10).
PAR-MCP-018,019,050,060 → HAVE-SUBSET with the footnote scoped to what g0router serves
(no Cowork 3p-config file writer; antigravity executor is w7-prov-special). Mark
`open-questions.md` w6-l **ESC-1a/1b/1c** RESOLVED with a cite to this plan; flip the
four w6-l surfaces variant-HAVE → true-HAVE. Append new open items (§8).

### 1.1 Preconditions already satisfied by the SHIPPED w7-mcp-1/2 + w7-gov-1 (evidence — cite file:line)

- **The launcher transport seam is LIVE (consume — do NOT edit).**
  `internal/mcp/launcher.go:31 NewLauncher(st)`, `:41 SetRunner`,
  `:53 StartStdio(name,command,args,env) (*Bridge,error)` (allowlist-gated BEFORE
  spawn — PAR-MCP-049), `:90 StartHTTP(name,url)`, `:96 StartSSE(name,url)`,
  `:111 IsRunning(name)`, `:119 Bridge(name) (*Bridge,bool)`, `:127 Stop(name)`.
  **The `CreateInstance` handler branches url-vs-command (PAR-MCP-022) onto exactly
  these** — `command` → `StartStdio`, `url`+`sse` → `StartSSE`, `url`+`http` →
  `StartHTTP`. The launcher is constructed/injected via a setter on `Handlers`
  (§1.3), NOT by editing launcher.go.
- **The bridge tool-call seam is LIVE.** `internal/mcp/bridge.go:122 Send(frame)`
  (write a JSON-RPC frame to the child stdin), `:127 IsRunning`,
  `:12 SessionSink`. The agent loop + tool-execute path drive a `Bridge` via `Send`
  + an in-memory `SessionSink` to capture the result frame (§1.7/§1.9).
- **The client engine is LIVE (consume).** `internal/mcp/probe.go:47 NewProbe`,
  `:56 Run(ctx,url) ProbeResult{Tools []ProbeTool, RequiresAuth, Error}`;
  `internal/mcp/registry.go:50 NewRegistry`, `:60 List(ctx,force) ([]RegistryServer,
  error)`; `internal/mcp/oauth.go:32 NewEngine(st,client)`,
  `:49 Start(ctx,serverURL,instanceID,redirectURI) (*StartResult{AuthURL,State},
  error)`, `:88 Complete(...)`, `:128 Refresh(...)`; `internal/mcp/discovery.go:16
  newToolsCache`, `:38 buildCompactManifest(tools)`. **The tools handler reads the
  discovery cache / probe; the OAuth `…/auth/start` handler calls `Engine.Start`.**
- **The MCP store is LIVE + complete (consume; tool-groups table is the ONLY
  additive store change).** `internal/store/mcpinstances.go:41 CreateMCPClient`,
  `:63 GetMCPClient`, `:69 ListMCPClients`, `:93 UpsertMCPClient`,
  `:113 DeleteMCPClient`, `:122 CreateMCPInstance`, `:157 GetMCPInstance`,
  `:164 ListMCPInstances`, `:188 UpdateMCPInstance`, `:210 SetMCPInstanceStatus`,
  `:222 DeleteMCPInstance`; `internal/store/mcpoauth.go:99
  GetMCPOAuthAccountByInstance`, `:106 ListMCPOAuthAccounts`. The store structs are
  Go PascalCase-field records (`MCPClient{ID,Name,Type,Config,CreatedAt,UpdatedAt}`
  @ `mcpinstances.go:14`; `MCPInstance{ID,ClientID,Name,Transport,URL,Command,Args,
  Env,Status,CreatedAt,UpdatedAt}` @ `:25`) — the DTO casing decision is §1.2.
- **The admin pattern is SHIPPED + frozen-stable (the gov-1/gov-3 template).**
  - `internal/admin/respond.go:19 writeData(ctx,status,data)` / `:23
    writeError(ctx,status,message)` → `{data,error:{message}}`. **Note: the JSON
    KEY-CASING of the `data` payload is determined by the DTO struct's json tags,
    NOT by the envelope** — so a PascalCase-tagged DTO emits PascalCase keys under
    `{data}` (the §1.2 casing exception is implemented purely with struct tags).
  - `internal/admin/handlers.go:115 pathID(v any) (string,bool)` extracts `{id}`.
  - `internal/admin/handlers.go:16 Handlers struct` already holds `audit
    *governance.AuditService` (`:23`); `:64 auditService()` accessor;
    `internal/admin/audit.go:64 recordAudit(ctx, action, target, details)` —
    best-effort, never fails the parent (the gov-1 seam — REUSE for MCP mutations).
  - **Setter-injection precedent (the EXACT seam for the launcher/engine):**
    `handlers.go:94 SetProxyProber`, `:102 SetTunnelRunner`, `:110 SetMitmProxy` —
    additive setters called by the server bootstrap AFTER `New`, NO `New(...)`
    signature change. **This plan adds `SetMCPLauncher`/`SetMCPEngine` the same
    way (§1.3).**
  - CRUD template = `internal/admin/teams.go` (List/Create/Get/Update/Delete with
    DTO + request structs + validate + `ErrNotFound`→404 + `recordAudit`). Test
    surface = `internal/admin/admin_test.go:24 newTestEnv`, `:72 call`,
    `:97 dataField[T]`, `:125 loginToken`. The freshest injectable-seam sibling is
    `internal/admin/alerts.go` (recordAudit on every mutation + a fake `Sender`).
- **The guard entry is LIVE (consume — do NOT edit).** `internal/server/guard.go:45
  LOCAL_ONLY_PATHS` contains `"/api/mcp/"` (`:46`); the loop at `:80` enforces
  local-only. **The new `/api/mcp/*` routes COEXIST with this guard automatically —
  do NOT edit guard.go.** (`/api/skills` is NOT under `/api/mcp/` — it is a normal
  `RequireSession` route, not local-only, matching the mock which serves it like any
  other authenticated read.)
- **Migrations are additive-only.** `internal/store/migrate.go` `tables` slice with
  `CREATE TABLE IF NOT EXISTS`; the four `mcp_*` tables shipped in mcp-1. The ONLY
  additive table here is `mcp_tool_groups` (§1.7). No column changes.

### 1.2 The mock contracts these flips must mirror (binding — decision 1 + the FROZEN-page exception)

**Decision 1 (MAP §36, §245):** real Go wins; the W6 mock body + seed are corrected
to mirror the real Go `{data,error}` DTO. **BUT the w6-l UI page is FROZEN
(decision 8); the page already consumes a fixed key-casing, so the Go DTO MUST emit
the casing the frozen page reads.** The mock fields were modeled to match what the
page consumes — therefore **prefer matching the mock's existing key-casing in the Go
DTO; correct the mock ONLY if the real Go genuinely diverges.** Below is the exact
mock-vs-Go reconciliation per surface, with the casing decision called out.

**THE CASING DECISION (binding — evidence-grounded, a deliberate exception recorded
in §8 ESC-CASING):**

- **MCP clients + instances → PascalCase DTOs** (`ID`, `Name`, `Transport`,
  `Command`, `Args`, `Env`, `IsActive`, `HealthStatus`, `CreatedAt`, `UpdatedAt`).
  Evidence: `ui/e2e/mocks/handlers/mcp.ts:7` returns `Array.from(store.mcpClients
  .values())` and the seed `ui/e2e/mocks/seed/mcp.ts:5-24` keys are PascalCase
  (`ID`/`Name`/`Transport`/`Command`/`Args`/`Env`/`IsActive`/`HealthStatus`/
  `CreatedAt`); the install POST `mcp.ts:23` spreads `{ID, CreatedAt, UpdatedAt,
  IsActive, ...body}` PascalCase. The frozen `/mcp` page + `McpMarketplaceModal`
  read these PascalCase keys (the spec asserts the transport badge "stdio" renders
  on `mcp-instance-row` — `mcp.spec.ts:26` — which the page reads from the
  PascalCase `Transport`). **Therefore the Go `clientDTO`/`instanceDTO` carry
  `json:"ID"`,`json:"Name"`,`json:"Transport"`,`json:"Command"`,`json:"Args"`,
  `json:"Env"`,`json:"IsActive"`,`json:"HealthStatus"`,`json:"CreatedAt"`,
  `json:"UpdatedAt"` tags — PascalCase, a deliberate exception to the snake_case
  convention because the consumed contract is PascalCase and the page is frozen.**
  The Go `store.MCPClient`/`MCPInstance` already use PascalCase Go fields (which
  default to PascalCase JSON keys when un-tagged) — but the DTO declares EXPLICIT
  PascalCase json tags so it is unambiguous and survives a future `gofmt`/linter
  that might add snake-case tags. Two store fields are NOT in the mock seed
  (`Type`, `ClientID`, `Status`) and two mock fields are NOT in the store
  (`IsActive`, `HealthStatus`): the DTO maps `Status` → both `IsActive`
  (`Status=="running"`) AND `HealthStatus` (`running→"healthy"`, `error→"unhealthy"`,
  else→`"unknown"`) — a PURE `instanceHealth(status)` derivation (§1.5). `URL` is
  emitted for http/sse instances (omitempty), matching the mock seed's GitHub entry.
- **MCP tools → snake_case / OpenAI-tool shape** (`{type:"function", function:
  {name, description, parameters}}`). Evidence: `seed/mcp.ts:43-61` + `mcp.ts:51`
  returns `store.mcpTools` in that exact shape; the frozen `/mcp/tools` page reads
  `function.name`/`function.description` (spec asserts "read_file"/"write_file"
  render — `mcp.spec.ts:50-51`). **Go `toolDTO` mirrors the OpenAI-tool shape with
  snake_case-irrelevant nested keys (`type`,`function:{name,description,
  parameters}`).**
- **MCP tools execute → `{result:string}`** (`mcp.ts:57` returns `{result: "Mock
  execution result for <name>"}`; spec asserts body contains "Mock execution
  result" — `mcp.spec.ts:59`). **Go `ExecuteTool` returns `{data:{result:<string>}}`.**
- **MCP tool-groups → snake_case** (`{id, name, tool_ids, is_active, created_at,
  updated_at}`). Evidence: `seed/mcp.ts:66` is `{id:1, name, tool_ids:[...],
  is_active:true, created_at}`; `mcp.ts:66` POST spreads `{id:Date.now(),
  created_at, updated_at, ...body}` snake_case; the frozen page reads "File
  Operations" + a toggle (`mcp.spec.ts:69-72`). **Go `toolGroupDTO` carries
  snake_case json tags (`json:"id"`,`json:"name"`,`json:"tool_ids"`,
  `json:"is_active"`,`json:"created_at"`,`json:"updated_at"`). `id` is a NUMERIC id
  (`Date.now()` / `1` in the mock) — the store table uses INTEGER PK numeric ids
  (mirror the w7-gov-2 `feature_flags` / w7-gov-3 `alert_channels` ESC-IDTYPE
  precedent), §1.7.**
- **Skills → snake_case-irrelevant flat shape** (`{name, category, description,
  url}`). Evidence: `seed/skills.ts:5-6` + `skills.ts:7` returns `store.skills` as
  `[{name,category,description,url}]`; the frozen `/skills` page groups by
  `category` + exposes a copy control, asserts exactly 2 rows + "filesystem"/
  "github"/"Endpoint Skills" (`skills.spec.ts:24-28`). **Go `skillDTO` mirrors
  `{name,category,description,url}` (all lowercase keys).**

**Per-surface reconciliation (mock body changes — DEFAULT is NO change because the
casing matches):**

| Mock file | Page-consumed shape | Go DTO | Mock correction needed? |
|---|---|---|---|
| `mcp.ts:6-9` GET `/api/mcp/clients` (+`/{id}`) | bare array of PascalCase clients under `{data}` | `[]clientDTO` PascalCase | NO (Go matches) — the mock returns a bare array; Go returns `{data:[clientDTO]}` (the page already unwraps `data`; gov-1 ListTeams used the same bare-array-under-data shape — `w7-gov-1 §1.4`) |
| `mcp.ts:18-41` GET/POST `/api/mcp/instances` (+`GET/DELETE /{id}`) | PascalCase instances; POST returns the created instance | `[]instanceDTO` / `instanceDTO` PascalCase | NO (Go matches) — POST install body is `{ClientID?, Name, Transport, URL?, Command?, Args?, Env?}` or the marketplace's `toInstancePayload`; Go reads both Pascal + lower keys defensively (ESC-CASING note) |
| `mcp.ts:42-45` GET `…/{id}/accounts` | array (mock returns `[]`) | `[]accountDTO` (tokens STRIPPED) | NO — Go returns `{data:[accountDTO]}` with NO token fields; mock `[]` is compatible |
| `mcp.ts:46-49` POST `…/{id}/auth/start` | `{url:string}` | `{data:{url:<AuthURL>}}` | NO (Go matches `{url}`) — maps `Engine.Start().AuthURL` → `url` |
| `mcp.ts:50-53` GET `/api/mcp/tools` | OpenAI-tool array | `[]toolDTO` | NO |
| `mcp.ts:54-60` POST `…/tools/{name}/execute` | `{result:string}` | `{data:{result}}` | NO |
| `mcp.ts:61-92` GET/POST `/api/mcp/tool-groups` (+`GET/PUT/DELETE /{id}`) | snake_case groups w/ numeric id | `[]toolGroupDTO` / `toolGroupDTO` snake_case | NO (Go matches) |
| `skills.ts:6-9` GET `/api/skills` | flat `{name,category,description,url}` array | `[]skillDTO` | NO |

**Binding:** if at impl ANY real Go field genuinely diverges from a consumed key
(e.g. the page reads a key the Go cannot supply), STOP and resolve per §8 ESC-CASING /
ESC-MOCK — never silently rename a frozen page's key, never fudge the Go to match a
key the page does not read. The DEFAULT expectation is ZERO mock-body change (the
casing was chosen to match) — the plan ships mock corrections ONLY on a proven
divergence, and any correction is body-only (never the mock index / store.ts /
seed-index / fixture.ts — those are FORBIDDEN, §3).

### 1.3 Engine injection (binding — mirror SetProxyProber/SetTunnelRunner/SetMitmProxy; NO New() change)

The MCP handlers need the SHIPPED `*mcp.Launcher` (for instance start/stop +
tool-execute via the bridge) and the SHIPPED `*mcp.Engine` (for OAuth `…/auth/start`)
and a `*mcp.Probe`/discovery cache (for the tools list). These are injected via
ADDITIVE setters on `Handlers`, EXACTLY as the shipped `SetProxyProber`
(`handlers.go:94`) / `SetTunnelRunner` (`:102`) / `SetMitmProxy` (`:110`) — the
server bootstrap calls them after `New`, with NO `New(...)` signature change and NO
new global state:

```go
// internal/admin/handlers.go (ADDITIVE — mirror the three existing Set* setters)
// new fields on the Handlers struct:
//   mcpLauncher *mcp.Launcher
//   mcpEngine   *mcp.Engine
//   mcpProbe    *mcp.Probe        // (+ a discovery toolsCache or *mcp.Registry as needed)
func (h *Handlers) SetMCPLauncher(l *mcp.Launcher) { h.mcpLauncher = l }
func (h *Handlers) SetMCPEngine(e *mcp.Engine)     { h.mcpEngine = e }
```
The fields are nil-able: when unset (e.g. a minimal test env that doesn't exercise
launch), the handlers degrade gracefully (a list still works from the store; an
execute/auth-start that needs an unset engine returns a typed 501/503, mirroring the
`SetShutdownFunc` nil-able precedent `handlers.go:88`). **The handler unit tests
inject a FAKE launcher/engine via these setters** — by `SetRunner(fakeRunner)` on a
real `NewLauncher(st)` (the SHIPPED w7-mcp-1 test seam) and a real `NewEngine(st,
&http.Client{Transport:fakeTransport})` (the SHIPPED w7-mcp-2 seam). NO real
subprocess, NO real network in any unit test (§1.10). **DECIDE at T-wire whether the
probe/discovery cache is a fourth field or folded into the launcher — default: a
`mcpProbe *mcp.Probe` field + a handler-owned `toolsCache` (or just call `Probe.Run`
behind the injected `*http.Client`); §8 ESC-PROBE-FIELD.**

### 1.4 MCP clients Go contract (handler over the SHIPPED store)

`internal/admin/mcp.go` (NEW). Clients are READ-only here (the store CRUD shipped in
mcp-1; the marketplace browses clients to install instances — the page does not
create clients directly, only instances):

| Handler | Route | Shape (`{data}`, PascalCase per §1.2) | Notes |
|---|---|---|---|
| `ListClients` | `GET /api/mcp/clients` | `{data:[clientDTO]}` (bare array under data, mirror `mcp.ts:7`) | `clientDTO{ID,Name,Transport,Command,Args,Env,URL,IsActive,HealthStatus,CreatedAt,UpdatedAt}` — assembled from `store.MCPClient` + the default-plugins / registry as the marketplace source (DECIDE source at T-clients: store clients ∪ `DefaultPlugins()` mapped to clientDTO so the marketplace shows Filesystem+GitHub-style entries the spec expects — §8 ESC-CLIENT-SRC; default: union of store + DefaultPlugins) |
| `GetClient` | `GET /api/mcp/clients/{id}` | `{data:clientDTO}` or 404 | `store.GetMCPClient(id)` → `ErrNotFound`→404 |

### 1.5 MCP instances Go contract (handler over store + launcher; URL-vs-command — PAR-MCP-022)

| Handler | Route | Shape | Notes |
|---|---|---|---|
| `ListInstances` | `GET /api/mcp/instances` | `{data:[instanceDTO]}` PascalCase | `store.ListMCPInstances()`; map each `MCPInstance` → `instanceDTO` deriving `IsActive`/`HealthStatus` from `Status` via PURE `instanceHealth(status)` (§1.2) |
| `CreateInstance` | `POST /api/mcp/instances` | body `{Name, Transport, ClientID?, URL?, Command?, Args?, Env?}` (read Pascal + lower defensively); returns `{data:instanceDTO}`; 400 on missing name/mode | **PAR-MCP-022 branch:** `Command!=""` → `Launcher.StartStdio(name,command,args,env)` (allowlist-gated pre-spawn — PAR-MCP-049; a non-allowlisted command → 400 BEFORE any spawn); `URL!=""` + `Transport=="sse"` → `StartSSE`; `URL!=""` + `Transport=="http"` → `StartHTTP`. Persist via `store.CreateMCPInstance`. Sanitize `Name` via PURE `sanitizePluginName` (PAR-MCP-048, regex `[^a-zA-Z0-9_-]` + 64-cap). `recordAudit("mcp_instance.create", name, ...)`. **In a unit test the launcher uses `SetRunner(fakeRunner)` — no real spawn.** |
| `GetInstance` | `GET /api/mcp/instances/{id}` | `{data:instanceDTO}` or 404 | |
| `DeleteInstance` | `DELETE /api/mcp/instances/{id}` | `{data:{}}` or 404 | `store.DeleteMCPInstance(id)`; best-effort `Launcher.Stop(name)` (PAR-MCP-050 subset — stop the bridge; the Cowork 3p-config/skip-approvals reset is N/A, ESC-COWORK-CONFIG). `recordAudit("mcp_instance.delete", ...)` |
| `ListInstanceAccounts` | `GET /api/mcp/instances/{id}/accounts` | `{data:[accountDTO]}` — tokens STRIPPED | `store.GetMCPOAuthAccountByInstance(id)` (or `ListMCPOAuthAccounts` filtered); `accountDTO{ID,InstanceID,ServerURL,Status,ExpiresAt,Scope}` — **NEVER `AccessToken`/`RefreshToken`** (the masked-read discipline; §5 no-leak proof) |
| `StartInstanceAuth` | `POST /api/mcp/instances/{id}/auth/start` | `{data:{url:<AuthURL>}}` | resolve the instance's `URL` as the server URL; `mcpEngine.Start(ctx, serverURL, instanceID, redirectURI)` → `{url:result.AuthURL}`. 503 if `mcpEngine` unset. `redirectURI` derived from request/config (DECIDE at T-auth — default: a configured admin callback path; §8 ESC-OAUTH-REDIRECT). NEVER echo the state/verifier. |

`instanceDTO{ID,ClientID,Name,Transport,URL,Command,Args,Env,IsActive,HealthStatus,
CreatedAt,UpdatedAt}` — PascalCase json tags (§1.2). `Status` is NOT echoed as a raw
field name the page doesn't read; `IsActive`/`HealthStatus` are the derived
page-consumed fields.

### 1.6 MCP tools Go contract (list via probe/discovery; execute via agent/bridge) + the toolPolicy builders

| Handler | Route | Shape | Notes |
|---|---|---|---|
| `ListTools` | `GET /api/mcp/tools` | `{data:[toolDTO]}` OpenAI-tool shape | Aggregate tools across instances: for each instance, read the discovery cache (or run `Probe.Run` behind the injected client) → `[]ProbeTool`; map each to `toolDTO{type:"function", function:{name,description,parameters}}`. Apply PURE `stripServerPrefix` on names (PAR-MCP-046) + prefer `def.ToolNames` (PAR-MCP-047). Include the antigravity ride-along tool `{unavailable:true}` (PAR-MCP-060, ESC-ANTIGRAVITY). The spec only requires read_file/write_file render + an execute action — the seed shape governs (§1.2). |
| `ExecuteTool` | `POST /api/mcp/tools/{name}/execute` | body `{arguments?:object}`; returns `{data:{result:<string>}}` | **The tool-execute path (the parity bar):** resolve the instance owning `{name}`; build a JSON-RPC `tools/call` frame; drive it through the SHIPPED `Bridge.Send` + an in-memory `SessionSink` that captures the result frame (or, for an HTTP/SSE instance, the w7-mcp-2 `sseClient.postMessage`); apply the SHIPPED `smartFilterText` to the result text; return `{result:<filtered text>}`. 503 if `mcpLauncher` unset; 404 if the tool/instance is unknown. **In a unit test the bridge is driven by the FAKE process (canned result frame) via `SetRunner` — no real spawn.** `recordAudit("mcp_tool.execute", name, ...)` best-effort. |

**The toolPolicy / managed-server PURE builders (PAR-MCP-018/019/045/046/047 —
unit-tested; in `internal/mcp/agent.go` or a small `internal/mcp/toolpolicy.go` —
DECIDE file at T-agent, default: `toolpolicy.go` alongside agent.go):**
```go
// stripServerPrefix idempotently removes repeated "<server>-" prefixes from a tool
// name (PAR-MCP-045/046; mirrors coworkPlugins.js:48 while-loop). PURE.
func stripServerPrefix(server, tool string) string
// buildToolPolicy emits, for each tool, BOTH the bare name and "<server>-<tool>" as
// "allow" (PAR-MCP-019; cowork-settings:171). PURE — returns map[string]string.
func buildToolPolicy(server string, toolNames []string) map[string]string
// buildManagedServers assembles the {name,url,transport,oauth,toolPolicy} list from
// instances (PAR-MCP-018; cowork-settings:262). PURE over instance + tool inputs.
func buildManagedServers(instances []managedServerInput) []ManagedServer
// sanitizePluginName strips [^a-zA-Z0-9_-] and truncates to 64 (PAR-MCP-048;
// cowork-settings:339). PURE.
func sanitizePluginName(s string) string
```
**ESC-TOOLPOLICY-SCOPE (§8): g0router has no Cowork 3p-config writer**, so PAR-MCP-018/
019/050 are ported as the PURE builders (the parity-meaningful logic) + applied where
g0router has a surface (the tools list prefix-strip, the create-name sanitize), NOT a
Claude-Desktop config-file rewrite. The builders are unit-tested directly; whether a
managed-server READ endpoint is exposed is an ESCALATION (default: ship the builders +
unit-test them; expose only if a frozen page consumes them — the w6-l specs do NOT).

### 1.7 MCP tool-groups Go contract (NEW additive store table + admin CRUD)

Table `mcp_tool_groups` (additive, `migrate.go` tables slice; INTEGER PK numeric id
per ESC-IDTYPE — mirror w7-gov-2 `feature_flags` / w7-gov-3 `alert_channels`):
```sql
CREATE TABLE IF NOT EXISTS mcp_tool_groups (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  name        TEXT NOT NULL,
  tool_ids    TEXT NOT NULL DEFAULT '[]',   -- JSON array of tool name strings
  is_active   INTEGER NOT NULL DEFAULT 1,
  created_at  TEXT NOT NULL,                 -- ISO-8601 (mirror mock created_at)
  updated_at  TEXT NOT NULL DEFAULT ''
)
```
`internal/store/mcptoolgroups.go` (NEW): `MCPToolGroup{ID int64, Name string,
ToolIDs []string, IsActive bool, CreatedAt, UpdatedAt string}` +
`CreateMCPToolGroup`/`ListMCPToolGroups`/`GetMCPToolGroup(id)`/`UpdateMCPToolGroup`/
`DeleteMCPToolGroup(id)` (JSON-marshal `ToolIDs` into `tool_ids`; ISO-8601
timestamps to mirror the mock; `boolToInt` for `is_active`; `ErrNotFound`).

| Handler | Route | Shape (snake_case per §1.2) | Notes |
|---|---|---|---|
| `ListToolGroups` | `GET /api/mcp/tool-groups` | `{data:[toolGroupDTO]}` | bare array under data |
| `CreateToolGroup` | `POST /api/mcp/tool-groups` | body `{name, tool_ids?, is_active?}`; returns `{data:toolGroupDTO}` | numeric id; `created_at`/`updated_at` ISO-8601; `recordAudit` |
| `GetToolGroup` | `GET /api/mcp/tool-groups/{id}` | `{data:toolGroupDTO}` or 404 | `pathID` → parse int |
| `UpdateToolGroup` | `PUT /api/mcp/tool-groups/{id}` | body = create body (+ `is_active` toggle); returns `{data:toolGroupDTO}` or 404 | the spec's "toggle" path; `recordAudit` |
| `DeleteToolGroup` | `DELETE /api/mcp/tool-groups/{id}` | `{data:{}}` or 404 | `recordAudit` |

`toolGroupDTO{id,name,tool_ids,is_active,created_at,updated_at}` — snake_case tags.

### 1.8 Skills Go contract (NEW catalog + admin read)

`internal/admin/skills.go` (NEW). The mock serves a flat catalog grouped by category
on the page. DECIDE the source at T-skills (§8 ESC-SKILLS-SRC):
- **Default (recommended): a static Go catalog** mirroring the seed (filesystem +
  github + the MCP default plugins mapped to skill entries) — a PURE
  `skillsCatalog() []skillDTO` constant assembly. No new store table (skills are a
  read-only catalog, like the providers catalog). The spec asserts exactly 2 seeded
  skills render grouped by "Endpoint Skills" — the Go catalog supplies ≥2 entries in
  ≥1 category; the corrected `seed/skills.ts` (only if Go diverges) stays the 2-entry
  shape.
- Alternative (escalate): a `skills` store table if operator-managed skills are
  wanted — NOT indicated by the frozen page (read-only, no create/delete control).

| Handler | Route | Shape | Notes |
|---|---|---|---|
| `ListSkills` | `GET /api/skills` | `{data:[skillDTO]}` flat array | `skillDTO{name,category,description,url}`; NOT under `/api/mcp/` so NOT local-only (a normal `RequireSession` route — §1.1 guard note) |

### 1.9 The agent loop (NEW — `internal/mcp/agent.go`) — PAR-MCP-040 (scope it; the parity bar = tool-execute path + loop primitive)

A server-side bounded multi-turn tool-execution loop: given a model turn that requests
tool calls, the agent executes each tool (via the launcher/bridge — the SAME path
`ExecuteTool` uses), feeds the results back, and repeats up to a bounded number of
turns. **SCOPE (binding — ESC-AGENT-DEPTH §8): mirror the ref's primitive, NOT a full
agent runtime.** The parity bar is (a) the tool-execute path is real, and (b) a loop
primitive exists that calls tools and feeds results back with a hard turn cap. The
LLM round-trip itself is behind an INJECTED seam (an interface `ToolCaller` /
`modelTurn func(...)`), so the loop is unit-tested with a fake model that returns a
canned tool-call then a canned final answer — NO real LLM, NO real network.

```go
// internal/mcp/agent.go
// ToolExecutor runs one tool call and returns its (filtered) result. The real impl
// drives a Bridge.Send / sseClient.postMessage; the test impl returns canned results.
type ToolExecutor interface {
    Execute(ctx context.Context, name string, args map[string]any) (string, error)
}
// Agent runs a bounded multi-turn loop: it asks the model for the next step, executes
// any requested tool via the ToolExecutor, appends the result, and repeats until the
// model returns a final answer OR maxTurns is hit (the bound — no runaway loop).
type Agent struct {
    exec     ToolExecutor
    maxTurns int            // hard cap (default e.g. 8) — PAR-MCP-040 bound
}
func NewAgent(exec ToolExecutor, maxTurns int) *Agent
// Run drives the loop given an initial request + an injected model step function.
// modelStep returns either a tool call to execute or a final answer.
func (a *Agent) Run(ctx context.Context, req AgentRequest, modelStep ModelStep) (AgentResult, error)
```
- The PURE/fake-tested core: the loop terminates on a final answer; it executes a
  requested tool then re-invokes modelStep with the appended result; it STOPS at
  `maxTurns` (the bound — assert no more than N tool executions); a tool error is
  surfaced (not swallowed into an infinite retry).
- The REAL `ToolExecutor` is the bridge/sse-backed executor `ExecuteTool` shares
  (drives `Bridge.Send` + a capturing `SessionSink` + `smartFilterText`); it is
  exercised by the `ExecuteTool` handler test via the FAKE process — NO real spawn.
- **Default `maxTurns`, the modelStep contract shape, and whether the agent is wired
  to a live `/api/mcp/.../agent` route are ESCALATIONS (ESC-AGENT-DEPTH §8). Default:
  ship the loop primitive + the shared real `ToolExecutor` + unit tests; expose NO
  new agent HTTP route in this plan (the w6-l page has no agent surface; the parity
  bar is the primitive + the tool-execute path). If the ref clearly demands a live
  agent endpoint, ESCALATE with the ref before adding a route.**

### 1.10 What is UNIT-TESTED vs INTEGRATION-ONLY (binding — the hermeticity guarantee)

**UNIT-TESTED (deterministic, hermetic — `go test ./...` with NO real subprocess / NO
real network / NO port bind / NO real-process sleep):**
- The admin handlers via `newTestEnv` + the injected FAKE launcher/engine
  (`SetMCPLauncher`/`SetMCPEngine` with a real `NewLauncher(st)` + `SetRunner(fake)`
  and a real `NewEngine(st, &http.Client{Transport:fakeTransport})`): clients list/
  get; instances list/create(stdio via fake runner — allowlist reject-before-spawn,
  http, sse)/get/delete; instance accounts (tokens STRIPPED); auth/start →
  `{url}` via the fake-transport engine; tools list; **tool execute via the FAKE
  process canned result frame**; tool-groups CRUD; skills list.
- The PURE builders: `stripServerPrefix` (idempotent repeated-prefix strip),
  `buildToolPolicy` (bare + prefixed allow), `sanitizePluginName` (regex + 64-cap),
  `instanceHealth` (status→IsActive/HealthStatus), `buildManagedServers`.
- The agent loop via a FAKE `ToolExecutor` + a fake `modelStep`: terminates on final
  answer; executes-then-feeds-back; STOPS at maxTurns; surfaces a tool error.
- The tool-groups store: CRUD round-trip + `ErrNotFound`.
- **No-secret-leak:** every accounts/auth-start/instance response marshaled in a test
  asserts it contains NEITHER `AccessToken`/`RefreshToken` cleartext NOR the
  `state`/`verifier`.

**INTEGRATION-ONLY (NOT unit-tested — thin, isolated, escalation-recorded):** the real
`os/exec` stdio spawn (the SHIPPED `osProcessRunner` — never invoked by a unit test,
fake injected); the live SSE stream + the real `sseClient` network dial (SHIPPED
w7-mcp-2 integration-only); a real LLM model round-trip for the agent (behind the
injected `modelStep`/`ToolCaller`). These are excluded from `go test ./...`
determinism (§5 grep proof "no-real-spawn/network-in-test").

### NOT in scope (explicit)

- **No UI page/component/route/store edits** — `/mcp`, `/mcp/tools`, the
  `McpMarketplaceModal`, the `/skills` route, and all w6-l components are FROZEN
  consume-only (decision 8). The ONLY UI-tree touches are the mock-body + seed
  corrections IF (and only if) the real Go diverges (§1.2 default = NO change).
- **No edits to the SHIPPED w7-mcp-1/2 files** — `internal/mcp/{runner,bridge,
  launcher,filter,allowlist,defaults,process,probe,registry,oauth,sse,healthmonitor,
  discovery}.go` are CONSUMED, not edited (MAP decision 7). mcp-3 adds NEW files
  (`agent.go`, optional `toolpolicy.go`) + the admin handlers ONLY.
- **No edits to `internal/server/guard.go`** — the `/api/mcp/` LOCAL_ONLY entry
  (`guard.go:46`) is consumed as-is; the new routes coexist (§1.1).
- **No edits to `internal/schemas/mcp.go`** — consume the types.
- **No edits to pre-existing admin handlers' bodies** — auth/teams/audit/usermgmt/
  featureflags/prompttemplates/guardrails/alerts/apikeys/virtualkeys/providers*/
  connections/combos/disabledmodels/version/usage/pricing/proxypools/tunnels/mitm are
  FORBIDDEN. The ONLY `handlers.go` edit is the ADDITIVE `SetMCPLauncher`/
  `SetMCPEngine` setters + the new nil-able fields (no `New(...)` change — §1.3).
- **No edits to pre-existing store files** except the additive `mcp_tool_groups`
  table in `migrate.go` + the NEW `mcptoolgroups.go`.
- **No new global state / no `New(...)` signature change** (decision 9).
- **No destructive DDL** — additive `ensureTable` only.
- **No secret exposure** — OAuth tokens + state/verifier `*_enc` at rest (already by
  mcp-1/2), NEVER echoed in any DTO (§5 no-leak proofs).
- **No real subprocess / network / port bind / LLM call in any unit test.**
- **No mock index / seed-index / store.ts / fixture.ts edit** — body-only mock/seed
  corrections, and only on a proven divergence (§3 FORBIDDEN).

---

## 2. Precondition checks

Run all before any edit; abort and report to orchestrator on any failure.

```bash
# P0 — clean tree; record <base>
git status --porcelain     # expect empty EXCEPT a possibly-dirty ui/dist/index.html
                           # (gitignored build artifact — NEVER stage it, NEVER revert it).
                           # If anything ELSE is dirty, STOP. Use explicit `git add <file>`, never -A.
git rev-parse HEAD         # record as <base> for §5

# P1 — the w7-mcp-1 + w7-mcp-2 engine is SHIPPED + present (this plan consumes it)
test -e internal/mcp/launcher.go && test -e internal/mcp/bridge.go && test -e internal/mcp/probe.go && test -e internal/mcp/oauth.go && echo "mcp-1/2 engine OK"
grep -nE 'func NewLauncher|func .*StartStdio|func .*StartHTTP|func .*StartSSE|func .*\) Stop|func .*\) Bridge|func .*SetRunner' internal/mcp/launcher.go
grep -nE 'func .*\) Send|type SessionSink' internal/mcp/bridge.go
grep -nE 'func NewProbe|func .*\) Run|type ProbeResult|type ProbeTool' internal/mcp/probe.go
grep -nE 'func NewEngine|func .*\) Start\(|type StartResult' internal/mcp/oauth.go
grep -nE 'func .*CreateMCPInstance|func .*ListMCPInstances|func .*DeleteMCPInstance|func .*ListMCPClients|func .*GetMCPOAuthAccountByInstance' internal/store/mcpinstances.go internal/store/mcpoauth.go
grep -nE 'func DefaultPlugins' internal/mcp/defaults.go

# P2 — the gap is REAL (no mcp/skills admin handlers, no agent, no tool-groups store/route)
test ! -e internal/admin/mcp.go && test ! -e internal/admin/skills.go && echo "admin mcp/skills gap OK"
test ! -e internal/mcp/agent.go && echo "agent gap OK"
test ! -e internal/store/mcptoolgroups.go && echo "tool-groups store gap OK"
grep -nE '/api/mcp|/api/skills' internal/server/routes_admin.go ; echo "^ expect EMPTY (mcp-3 adds routes)"
grep -nE 'mcp_tool_groups' internal/store/migrate.go ; echo "^ expect EMPTY"

# P3 — the admin pattern + injection seam + recordAudit are present (the gov-1 template)
grep -nE 'func writeData|func writeError' internal/admin/respond.go
grep -nE 'func pathID|func New\(|SetProxyProber|SetTunnelRunner|SetMitmProxy|audit ' internal/admin/handlers.go
grep -nE 'func .*recordAudit' internal/admin/audit.go
grep -nE 'func newTestEnv|func call|func dataField|func loginToken' internal/admin/admin_test.go

# P4 — guard.go /api/mcp/ LOCAL_ONLY entry present (consume, DO NOT edit)
grep -nE '/api/mcp/|LOCAL_ONLY_PATHS' internal/server/guard.go

# P5 — the W6-l UI + specs are present (consume-only) + the mocks to (maybe) correct
test -f ui/e2e/mcp.spec.ts && test -f ui/e2e/skills.spec.ts && echo "specs present"
test -f ui/e2e/mocks/handlers/mcp.ts && test -f ui/e2e/mocks/handlers/skills.ts && echo "handlers present"
test -f ui/e2e/mocks/seed/mcp.ts && test -f ui/e2e/mocks/seed/skills.ts && echo "seeds present"
grep -nE 'ID:|Name:|Transport:|IsActive:|HealthStatus:' ui/e2e/mocks/seed/mcp.ts ; echo "^ PascalCase clients/instances (§1.2 casing)"
grep -nE 'tool_ids|is_active|created_at' ui/e2e/mocks/seed/mcp.ts ; echo "^ snake_case tool-groups (§1.2 casing)"

# P6 — routes_admin.go serial slot is FREE (chain …→w7-gov-3→**w7-mcp-3**→w7-plat-1)
git log --oneline -5 -- internal/server/routes_admin.go   # last touch = w7-gov-3 (merged + released)
# Orchestrator MUST confirm no concurrent W7 plan holds an unmerged routes_admin.go
# edit before w7-mcp-3 begins T-routes. w7-mcp-3 TAKES the slot, RELEASES to w7-plat-1.

# P7 — green at base (HERMETIC) + the two specs green at base
go test ./... && go vet ./... && go build ./...     # exit 0 (no net/process)
cd ui && npm run build                               # exit 0 (SEPARATE step)
cd ui && npx playwright test e2e/mcp.spec.ts e2e/skills.spec.ts   # green at base vs mocks
# Record exact pass/fail in WORKFLOW.md. They must STAY green after the Go lands
# (and after any mock-body correction).
```

---

## 3. Exclusive file ownership

After w7-mcp-3 merges, all CREATE files are owned by w7-mcp-3; later plans consume,
never edit (MAP decision 7).

**CREATE — store (NEW):**

| File | Contract |
|---|---|
| `internal/store/mcptoolgroups.go` | `MCPToolGroup` struct + `CreateMCPToolGroup`/`ListMCPToolGroups`/`GetMCPToolGroup`/`UpdateMCPToolGroup`/`DeleteMCPToolGroup`; INTEGER PK numeric id; `tool_ids` JSON; ISO-8601 timestamps; `boolToInt`; `ErrNotFound`. Mirrors w7-gov-2 `featureflags.go` (numeric-id store). |
| `internal/store/mcptoolgroups_test.go` | Temp `store.Open`: create→get→list→update(toggle is_active)→delete→404. RED first. |

**EXTEND — store (additive table only):**

| File | Change (additive ONLY) |
|---|---|
| `internal/store/migrate.go` | ADD `mcp_tool_groups` to the `tables` slice (INTEGER PK). ADDITIVE ONLY — no DROP/RENAME, no column change to existing tables. |

**CREATE — domain (NEW files in package `internal/mcp`):**

| File | Contract |
|---|---|
| `internal/mcp/agent.go` | `ToolExecutor` interface + `Agent{exec,maxTurns}` + `NewAgent` + `Run` (bounded multi-turn loop with injected `modelStep`); + the real bridge/sse-backed `ToolExecutor` impl the handler shares. No `init()`; errors-as-values. |
| `internal/mcp/agent_test.go` | Via a FAKE `ToolExecutor` + fake `modelStep`: terminate-on-final; execute-then-feed-back; STOP-at-maxTurns; tool-error surfaced. RED first. NO real LLM/process. |
| `internal/mcp/toolpolicy.go` (DECIDE — may fold into agent.go) | PURE `stripServerPrefix`/`buildToolPolicy`/`buildManagedServers`/`sanitizePluginName` (PAR-MCP-018/019/045/046/047/048). |
| `internal/mcp/toolpolicy_test.go` | idempotent prefix-strip; bare+prefixed allow; sanitize regex+64-cap; managed-server assembly. RED first. PURE. |

**CREATE — transport (NEW):**

| File | Contract |
|---|---|
| `internal/admin/mcp.go` | `ListClients`/`GetClient`/`ListInstances`/`CreateInstance`(url-vs-command, allowlist-gated, sanitize-name)/`GetInstance`/`DeleteInstance`/`ListInstanceAccounts`(tokens stripped)/`StartInstanceAuth` + `ListTools`/`ExecuteTool` + `ListToolGroups`/`CreateToolGroup`/`GetToolGroup`/`UpdateToolGroup`/`DeleteToolGroup`; `clientDTO`/`instanceDTO` (PascalCase tags), `toolDTO` (OpenAI shape), `toolGroupDTO` (snake_case tags), `accountDTO` (no tokens); PURE `instanceHealth`; `writeData`/`writeError`; `recordAudit` on mutations. |
| `internal/admin/mcp_test.go` | via `newTestEnv` + injected fake launcher/engine (`SetMCPLauncher`/`SetMCPEngine`): clients list/get; instances list/create-stdio(allowlist reject-before-spawn)/create-http/create-sse/get/delete; accounts(tokens stripped); auth/start→`{url}`; tools list; **execute via fake process canned frame**; tool-groups CRUD; PascalCase keys present on instance JSON; **no-token-leak** assertion. RED first. NO real spawn/network. |
| `internal/admin/skills.go` | `ListSkills` → `{data:[skillDTO]}`; `skillDTO{name,category,description,url}`; PURE `skillsCatalog()` source. |
| `internal/admin/skills_test.go` | via `newTestEnv`: list ≥2 skills incl ≥1 category; shape matches the mock. RED first. |

**MODIFY — handlers wiring (additive only — the injection seam, §1.3):**

| File | Change |
|---|---|
| `internal/admin/handlers.go` | ADDITIVE: add nil-able fields `mcpLauncher *mcp.Launcher`, `mcpEngine *mcp.Engine` (+ optional `mcpProbe *mcp.Probe`) to the `Handlers` struct + `SetMCPLauncher`/`SetMCPEngine` (+ optional setter) — mirror `SetProxyProber`/`SetTunnelRunner`/`SetMitmProxy`. NO `New(...)` signature change. The server bootstrap (`internal/server/*`) wires the real launcher/engine after `New` — IF that bootstrap wiring requires a one-line call, it is an ADDITIVE append in the bootstrap file (DECIDE the exact file at T-wire; default: the same place `SetTunnelRunner`/`SetMitmProxy` are called — §8 ESC-BOOTSTRAP). |

**MODIFY — serial-slot route registration (additive only):**

| File | Change |
|---|---|
| `internal/server/routes_admin.go` | ADD the `/api/mcp/*` + `/api/skills` route lines (§1.7-style block). NOTHING else changes. ONE commit. SERIAL SLOT — only holder while live; RELEASE to w7-plat-1 on close. Static-before-`{id}` precedence (the file's existing ordering, `routes_admin.go:48-50` users precedent). |

**MODIFY — e2e mock corrections (mirror real Go, decision 1 — ONLY on proven divergence):**

| File | Change |
|---|---|
| `ui/e2e/mocks/handlers/mcp.ts` (BODY) | DEFAULT: NO change (PascalCase clients/instances + snake_case tool-groups already match the Go DTOs, §1.2). Correct a body ONLY if a real Go field genuinely diverges from a page-consumed key (then mirror Go; never rename a frozen-page key). |
| `ui/e2e/mocks/handlers/skills.ts` (BODY) | DEFAULT: NO change (`{name,category,description,url}` matches). |
| `ui/e2e/mocks/seed/mcp.ts` (BODY) | DEFAULT: verify only; correct a seed field ONLY on divergence. |
| `ui/e2e/mocks/seed/skills.ts` (BODY) | DEFAULT: verify only. |

**FORBIDDEN:** everything else. Explicitly: ALL pre-existing `internal/admin/*.go`
except the NEW mcp/skills files + the ADDITIVE `handlers.go` setters/fields; ALL
SHIPPED `internal/mcp/*.go` (CONSUME — the only NEW mcp files are `agent.go` +
optional `toolpolicy.go`); `internal/server/guard.go` (the `/api/mcp/` LOCAL_ONLY
entry — consume, DO NOT edit); `internal/schemas/mcp.go` (consume); ALL other
`internal/store/*.go` except the NEW `mcptoolgroups.go` + the additive `migrate.go`
table; ALL `internal/inference/*` (the antigravity EXECUTOR is w7-prov-special — only
the tool-definition ride-along constant lives here, §1.6); ALL `ui/src/**` (FROZEN,
decision 8); the mock **index.ts / store.ts / seed-index / fixture.ts** (FORBIDDEN —
body-only corrections only); all OTHER mocks/seeds/specs; `ui/package.json` +
lockfile; `ui/vite.config.ts`; `ui/playwright.config.ts`. Touching any of these is an
automatic REJECT.

---

## 4. TDD tasks

Cadence (strict, AGENTS.md "TDD always"): **no Go impl file may exist before its
`_test.go` is committed RED.** `go test ./... && go vet ./... && go build ./...` green
at EVERY commit (a RED commit may fail ONLY the new package's targeted run). FULLY
HERMETIC (no real subprocess, no real network, no port bind, no LLM, no real-process
sleep). The two e2e specs stay green throughout (real Go is additive; the casing was
chosen to match — default no mock change). Order: tool-groups store → pure builders +
agent loop → admin mcp handlers (over store + fake launcher/engine) → skills → the
single serial-slot routes commit → mock verify/correct → closeout.

### T-toolgroups — STEP(a) RED store, STEP(b) impl
STEP(a): write `internal/store/mcptoolgroups_test.go`; ADD `mcp_tool_groups` to
`migrate.go`. `go test ./internal/store/ -run MCPToolGroup` → FAIL. Commit RED:
`phase-1/w7-mcp-3: failing mcp tool-groups store tests (TDD red)`.
STEP(b): implement `internal/store/mcptoolgroups.go`. Gates green. Commit:
`phase-1/w7-mcp-3: mcp tool-groups store (additive numeric-id table)`.

### T-agent — STEP(a) RED, STEP(b) impl (agent loop + pure builders)
STEP(a): write `internal/mcp/agent_test.go` (loop: final/feed-back/maxTurns-cap/
tool-error via a FAKE ToolExecutor + fake modelStep) + `internal/mcp/toolpolicy_test.go`
(stripServerPrefix idempotent; buildToolPolicy bare+prefixed; sanitizePluginName
regex+cap; buildManagedServers). `go test ./internal/mcp/ -run 'Agent|ToolPolicy'` →
FAIL. Commit RED: `phase-1/w7-mcp-3: failing agent-loop + tool-policy tests (TDD red)`.
STEP(b): implement `internal/mcp/agent.go` (bounded loop + the shared real
`ToolExecutor` over Bridge.Send/sseClient + `smartFilterText`) + `toolpolicy.go`.
Gates green (fake executor only). Commit:
`phase-1/w7-mcp-3: mcp agent loop (bounded multi-turn) + tool-policy builders`.

### T-mcp — STEP(a) RED admin tests, STEP(b) impl (clients/instances/tools/tool-groups)
STEP(a): write `internal/admin/mcp_test.go` (via `newTestEnv` + the injected fake
launcher/engine through `SetMCPLauncher`/`SetMCPEngine`); add the `SetMCPLauncher`/
`SetMCPEngine` + nil-able fields to `handlers.go` so tests compile.
`go test ./internal/admin/ -run Mcp` → FAIL. Commit RED:
`phase-1/w7-mcp-3: failing mcp admin handler tests (TDD red)`.
STEP(b): implement `internal/admin/mcp.go` (all handlers + DTOs + `instanceHealth` +
`recordAudit`). Gates green (fake launcher/engine only). Commit:
`phase-1/w7-mcp-3: mcp admin transport (clients/instances/tools/tool-groups + oauth start)`.

### T-skills — STEP(a) RED, STEP(b) impl
STEP(a): write `internal/admin/skills_test.go` (list ≥2 grouped by category). →
FAIL. Commit RED: `phase-1/w7-mcp-3: failing skills handler tests (TDD red)`.
STEP(b): implement `internal/admin/skills.go` (`ListSkills` + `skillsCatalog`). Gates
green. Commit: `phase-1/w7-mcp-3: skills catalog admin endpoint`.

### T-routes — serial-slot route registration
TAKE the serial slot (orchestrator confirms FREE at P6). Add the `/api/mcp/*` +
`/api/skills` route lines to `routes_admin.go` + wire the bootstrap
`SetMCPLauncher`/`SetMCPEngine` (the additive bootstrap call, §3). Gates:
`go test ./... && go vet ./... && go build ./...` green. Commit (ONE commit touches
the serial file): `phase-1/w7-mcp-3: register mcp + skills admin routes (serial slot)`.

### T-mocks — mock verify / correct (mirror real Go, decision 1)
Diff the real Go DTOs against `mcp.ts`/`skills.ts` + seeds. DEFAULT: NO change (casing
matches per §1.2). Correct a body ONLY on a proven divergence (mirror Go, never rename
a frozen-page key). Gates (ISOLATED — separate steps, NO chained pkill, NEVER revert
ui/dist/index.html):
```bash
cd ui && npm run build                                            # SEPARATE step
cd ui && npx playwright test e2e/mcp.spec.ts e2e/skills.spec.ts   # PLAIN — no pkill chain
```
If a correction reds a non-w7-mcp-3 spec, STOP + ESCALATE (§8 ESC-MOCK). Commit (only
if a change was needed): `phase-1/w7-mcp-3: correct mcp/skills mocks to mirror real Go DTOs`.

### T-close — full gates + closeout
```bash
go test ./internal/admin/ -run 'Mcp|Skill' && go test ./internal/mcp/... -run Agent   # the binary-acceptance subset
go test ./... && go vet ./... && go build ./...                                        # HERMETIC — no net/process
go test ./internal/store/ -run 'MCPToolGroup' -v
cd ui && npm run build                                                                 # SEPARATE step
cd ui && npx playwright test e2e/mcp.spec.ts e2e/skills.spec.ts                        # green (PLAIN)
cd ui && npx playwright test                                                           # full suite green (no regressions)
```
Flip `.planning/parity/matrix/9router-mcp.md`: PAR-MCP-022,040,041,045,046,047,048,049
→ HAVE (real Go; live SSE stream + real agent LLM round-trip integration-only behind
injected seams — §1.10); PAR-MCP-018,019,050,060 → HAVE-SUBSET (footnote: no Cowork
3p-config writer; antigravity executor is w7-prov-special). Mark `open-questions.md`
w6-l ESC-1a/1b/1c RESOLVED with a cite to this plan; flip the four w6-l surfaces
variant-HAVE → true-HAVE. Append the §8 open items (ESC-CASING deliberate-exception
note, ESC-AGENT-DEPTH outcome, ESC-TOOLPOLICY-SCOPE, ESC-ANTIGRAVITY, ESC-COWORK-CONFIG,
ESC-CLIENT-SRC, ESC-SKILLS-SRC, ESC-OAUTH-REDIRECT, ESC-PROBE-FIELD, ESC-BOOTSTRAP, the
integration-only spawn/network/LLM note). Update `docs/WORKFLOW.md` (P0 base, the ESC
decisions, the serial-slot take-from-w7-gov-3 / release-to-w7-plat-1, the casing
exception, any mock corrections). Final commit:
`phase-1/w7-mcp-3: close — MCP admin transport + tools + skills + agent loop; matrix flip`.
**On the close commit, RELEASE the routes_admin.go serial slot to w7-plat-1.**

---

## 5. Binary acceptance criteria

All must hold; each is yes/no. `<base>` = the commit recorded at P0. Diff gate is
**w7-mcp-3 commit-range-scoped** (§7).

**Test gates (HERMETIC — no real subprocess, no real network, no port bind, no LLM,
no real-process sleep)**
- `go test ./... && go vet ./... && go build ./...` → exit 0.
- `go test ./internal/admin/ -run 'Mcp|Skill' ./internal/mcp/... -run Agent -v` →
  exit 0, all pass (clients list/get; instances list/create-stdio-reject-before-spawn/
  create-http/create-sse/get/delete; accounts tokens-stripped; auth/start→`{url}`;
  tools list; tool execute via fake process; tool-groups CRUD; skills list; agent
  loop final/feed-back/maxTurns-cap/tool-error; pure builders).
- `go test ./internal/store/ -run 'MCPToolGroup' -v` → exit 0.
- `cd ui && npm run build` (SEPARATE step) → exit 0; then PLAIN
  `cd ui && npx playwright test e2e/mcp.spec.ts e2e/skills.spec.ts` → exit 0, all pass
  (6 mcp + 3 skills), 0 skipped. NO chained pkill in the test shell line.
- `cd ui && npx playwright test` → exit 0, no spec green-at-base goes red. NEVER
  revert `ui/dist/index.html`.

**TDD-order proof** — each impl file's covering test appears in an earlier-or-equal
commit:
```bash
for pair in \
  "internal/store/mcptoolgroups_test.go:internal/store/mcptoolgroups.go" \
  "internal/mcp/agent_test.go:internal/mcp/agent.go" \
  "internal/mcp/toolpolicy_test.go:internal/mcp/toolpolicy.go" \
  "internal/admin/mcp_test.go:internal/admin/mcp.go" \
  "internal/admin/skills_test.go:internal/admin/skills.go" ; do
  tf=${pair%%:*}; im=${pair##*:}
  test -e "$im" || continue   # toolpolicy.go optional (may fold into agent.go)
  ct=$(git log --format=%ct --diff-filter=A -1 -- "$tf")
  cf=$(git log --format=%ct --diff-filter=A -1 -- "$im")
  [ "$ct" -le "$cf" ] || echo "TDD VIOLATION: $im"     # prints nothing
done
```

**Grep proofs (per surface)**
```bash
# clients/instances/tools/tool-groups handlers
grep -nE 'func \(h \*Handlers\) (ListClients|GetClient|ListInstances|CreateInstance|GetInstance|DeleteInstance|ListInstanceAccounts|StartInstanceAuth)' internal/admin/mcp.go
grep -nE 'func \(h \*Handlers\) (ListTools|ExecuteTool|ListToolGroups|CreateToolGroup|GetToolGroup|UpdateToolGroup|DeleteToolGroup)' internal/admin/mcp.go
grep -nE 'json:"ID"|json:"Name"|json:"Transport"|json:"IsActive"|json:"HealthStatus"' internal/admin/mcp.go   # PascalCase clients/instances DTO (§1.2)
grep -nE 'json:"tool_ids"|json:"is_active"|json:"created_at"' internal/admin/mcp.go                            # snake_case tool-groups DTO (§1.2)
grep -nE 'writeData|writeError|recordAudit' internal/admin/mcp.go                                              # envelope + audit
grep -nE 'StartStdio|StartHTTP|StartSSE' internal/admin/mcp.go                                                 # url-vs-command branch (PAR-MCP-022)
# skills
grep -nE 'func \(h \*Handlers\) ListSkills' internal/admin/skills.go
grep -nE 'json:"name"|json:"category"|json:"description"|json:"url"' internal/admin/skills.go
# agent + builders
grep -nE 'type Agent|func NewAgent|func .*\) Run|maxTurns' internal/mcp/agent.go
grep -nE 'func stripServerPrefix|func buildToolPolicy|func sanitizePluginName' internal/mcp/agent.go internal/mcp/toolpolicy.go
# store
grep -nE 'func \(s \*Store\) (CreateMCPToolGroup|ListMCPToolGroups|GetMCPToolGroup|UpdateMCPToolGroup|DeleteMCPToolGroup)' internal/store/mcptoolgroups.go
# injection seam (additive — mirrors the three Set* siblings)
grep -nE 'func \(h \*Handlers\) SetMCPLauncher|func \(h \*Handlers\) SetMCPEngine' internal/admin/handlers.go
grep -nE 'func New\(' internal/admin/handlers.go | head -1   # signature UNCHANGED (st, sessions, flows)
# routes
grep -nE '/api/mcp/clients|/api/mcp/instances|/api/mcp/tools|/api/mcp/tool-groups|/api/skills' internal/server/routes_admin.go
# guard NOT edited
# no init(); no global state
! grep -rn 'func init(' internal/admin/mcp.go internal/admin/skills.go internal/mcp/agent.go internal/store/mcptoolgroups.go && echo "no init() OK"
```

**No-secret-exposure proofs (binding)**
```bash
# OAuth tokens never appear in any mcp DTO/response field
! grep -nE 'AccessToken|RefreshToken|access_token|refresh_token|Verifier|verifier' internal/admin/mcp.go | grep -iE 'json:"' && echo "no token json field OK"
# the accountDTO struct has no token field
grep -nA10 'type accountDTO struct' internal/admin/mcp.go ; echo "^ must NOT contain Access/Refresh token"
# additive migrations only (no DROP/RENAME introduced by this plan)
git diff <base>..HEAD -- internal/store/migrate.go | grep -E '^\+' | grep -iE 'DROP COLUMN|RENAME COLUMN|DROP TABLE' | wc -l   # = 0
# no real spawn/network/LLM in unit tests (the hermeticity guarantee)
! grep -rnE 'osProcessRunner|os/exec|http\.Get|http\.Post|net\.Dial' internal/admin/mcp_test.go internal/admin/skills_test.go internal/mcp/agent_test.go && echo "no-real-spawn/network-in-test OK"
```
Plus a runtime no-leak assertion in `mcp_test.go`: marshal every
accounts/auth-start/instance response and assert it contains neither an OAuth token
nor the OAuth `state`/`verifier`.

**Negative / freeze proofs (w7-mcp-3 commit-range — §7)**
```bash
R="<first-w7-mcp-3>^..<last-w7-mcp-3>"
# Only the sanctioned Go files changed:
git diff $R --name-only -- internal/ | grep -vE \
 'internal/store/(mcptoolgroups|migrate)(_test)?\.go|internal/mcp/(agent|toolpolicy)(_test)?\.go|internal/admin/(mcp|skills)(_test)?\.go|internal/admin/handlers\.go|internal/server/routes_admin\.go' \
 | wc -l                                                                  # = 0
#   (if the bootstrap wiring needs a one-line append, its file joins this allowlist — §8 ESC-BOOTSTRAP)
# SHIPPED mcp engine files untouched (consumed, not edited):
git diff $R --name-only -- internal/mcp/runner.go internal/mcp/bridge.go internal/mcp/launcher.go internal/mcp/filter.go internal/mcp/allowlist.go internal/mcp/defaults.go internal/mcp/process.go internal/mcp/probe.go internal/mcp/registry.go internal/mcp/oauth.go internal/mcp/sse.go internal/mcp/healthmonitor.go internal/mcp/discovery.go | wc -l   # = 0
# guard + schemas untouched:
git diff $R --name-only -- internal/server/guard.go internal/schemas/mcp.go | wc -l   # = 0
# Frozen admin handlers untouched (handlers.go additive-only is allowed above):
git diff $R --name-only -- internal/admin/auth.go internal/admin/teams.go internal/admin/audit.go internal/admin/usermgmt.go internal/admin/alerts.go internal/admin/apikeys.go internal/admin/virtualkeys.go internal/admin/providers.go internal/admin/connections.go internal/admin/combos.go | wc -l   # = 0
# UI src frozen; only sanctioned mock/seed bodies (if any) touched:
git diff $R --name-only -- ui/src/ | wc -l                               # = 0 (src frozen)
git diff $R --name-only -- ui/ | grep -vE \
 'ui/e2e/mocks/handlers/(mcp|skills)\.ts|ui/e2e/mocks/seed/(mcp|skills)\.ts' | wc -l   # = 0
# mock index/store/seed-index/fixture untouched:
git diff $R --name-only -- ui/e2e/mocks/handlers/index.ts ui/e2e/mocks/store.ts ui/e2e/mocks/seed/index.ts ui/e2e/mocks/fixture.ts | wc -l   # = 0
# routes_admin.go = exactly ONE commit, additive:
git log --oneline $R -- internal/server/routes_admin.go | wc -l          # = 1
git diff $R -- internal/server/routes_admin.go | grep -E '^-' | grep -v '^---' | wc -l   # = 0 (no deletions)
# handlers.go additive only (no New() signature change, no deletions):
git diff $R -- internal/admin/handlers.go | grep -E '^-' | grep -v '^---' | wc -l        # = 0
```

---

## 6. Out of scope (restated, binding)

No UI src edits (decision 8 — pages/components/routes/stores frozen); only the
sanctioned mcp/skills mock-body + seed corrections IF the real Go diverges (default:
none — casing matches). No edits to the SHIPPED w7-mcp-1/2 engine files (consume). No
edit to `guard.go` (the `/api/mcp/` LOCAL_ONLY entry coexists). No edit to
`internal/schemas/mcp.go`. No edits to pre-existing admin handlers' bodies (the ONLY
`handlers.go` change is the additive `SetMCPLauncher`/`SetMCPEngine` setters + nil-able
fields — no `New(...)` change). No edits to pre-existing store files except the
additive `mcp_tool_groups` table + the NEW `mcptoolgroups.go`. No antigravity executor
(w7-prov-special — only the tool-definition ride-along constant). No JWT. No
destructive DDL — additive `ensureTable` only. No new global state. No secret exposure
(OAuth tokens + state/verifier `*_enc`, never echoed). No real subprocess / network /
LLM in any unit test. No mock index/store.ts/seed-index/fixture.ts edit. Mock-vs-Go or
casing contradiction → escalate (§8), never rename a frozen page's key, never fudge a
mock or edit a frozen handler.

## 7. Diff-gate scope

W7 plans commit to main concurrently, so a broad `<base>..HEAD` range sweeps in
sibling commits. The diff gate MUST be scoped to w7-mcp-3's own commits. Isolate them:
`git log --oneline main | grep "^[0-9a-f]* phase-1/w7-mcp-3:" | awk '{print $1}'`
then `git diff <first-w7-mcp-3>^..<last-w7-mcp-3> -- [file list]`.

`git diff <first>^..<last> --name-only` must be exactly a subset of:
```
internal/store/mcptoolgroups.go
internal/store/mcptoolgroups_test.go
internal/store/migrate.go              (additive mcp_tool_groups table; ONE commit)
internal/mcp/agent.go
internal/mcp/agent_test.go
internal/mcp/toolpolicy.go             (CONDITIONAL — may fold into agent.go)
internal/mcp/toolpolicy_test.go        (CONDITIONAL)
internal/admin/mcp.go
internal/admin/mcp_test.go
internal/admin/skills.go
internal/admin/skills_test.go
internal/admin/handlers.go             (additive SetMCPLauncher/SetMCPEngine + fields; no New() sig change)
internal/server/routes_admin.go        (serial-slot additive routes; ONE commit)
internal/server/<bootstrap>.go         (CONDITIONAL — one-line Set* wiring append; §8 ESC-BOOTSTRAP)
ui/e2e/mocks/handlers/mcp.ts           (body only — IF Go diverges; default no change)
ui/e2e/mocks/handlers/skills.ts        (body only — IF Go diverges)
ui/e2e/mocks/seed/mcp.ts               (verify; correct only on divergence)
ui/e2e/mocks/seed/skills.ts            (verify)
.planning/parity/matrix/9router-mcp.md
.planning/parity/plans/open-questions.md
docs/WORKFLOW.md
```
Any file outside this list in the scoped diff is an automatic review REJECT. The
SHIPPED `internal/mcp/*` engine files, `internal/server/guard.go`,
`internal/schemas/mcp.go`, all pre-existing admin handlers, and all `ui/src/**` are
deliberately ABSENT — touching them is an automatic REJECT. The `routes_admin.go` edit
must appear in exactly ONE commit (§5) and the serial slot is released to w7-plat-1 on
close.

## 8. Escalations / decisions (explicit — recommended defaults, do not fabricate)

- **ESC-CASING (RESOLVED at authoring — the PascalCase exception, binding default).**
  The w6-l page is FROZEN and consumes PascalCase keys for clients/instances
  (`seed/mcp.ts:5-24` PascalCase `ID/Name/Transport/Command/Args/Env/IsActive/
  HealthStatus/CreatedAt`; `mcp.ts:23` install POST spreads PascalCase) and
  snake_case for tool-groups (`seed/mcp.ts:66` `id/name/tool_ids/is_active/
  created_at`) + skills + tools. **Decision: the Go `clientDTO`/`instanceDTO` emit
  EXPLICIT PascalCase json tags (a deliberate, recorded exception to the project
  snake_case convention) because the consumed contract is PascalCase and the page is
  frozen; `toolGroupDTO`/`skillDTO`/`toolDTO` are snake_case/OpenAI-shape per their
  consumed contracts.** This makes the four w6-l surfaces flip variant-HAVE →
  true-HAVE with ZERO mock-body change (the default). The exception is justified by
  "real Go wins, but the frozen page's consumed casing governs" (decision 1 + 8). If
  the operator prefers global snake_case for clients/instances, that REQUIRES editing
  the frozen page (decision-8 violation) — NOT in scope; escalate before doing so.
- **ESC-AGENT-DEPTH (RESOLVED at authoring — agent-loop scope, binding default).** The
  brief asks "how deep" the agent loop goes. **Decision: ship the loop PRIMITIVE — a
  bounded multi-turn `Agent.Run` with an injected `modelStep`/`ToolExecutor` (hard
  `maxTurns` cap, default 8) + the shared real bridge/sse-backed `ToolExecutor` that
  the `ExecuteTool` handler also uses — and NO new agent HTTP route** (the w6-l page
  has no agent surface; the parity bar is the tool-execute path + the loop primitive,
  per PAR-MCP-040 = "no agent.go today"). The LLM round-trip is behind the injected
  seam (unit-tested with a fake model). If the ref clearly demands a live
  `/api/mcp/.../agent` endpoint or a specific turn cap, ESCALATE with the ref
  (antigravity/cowork executor) before adding a route — never fabricate the agent
  protocol. RECOMMENDED as stated; flag for confirmation.
- **ESC-TOOLPOLICY-SCOPE (RESOLVED at authoring — PAR-MCP-018/019/050 scope, binding
  default).** g0router has NO Cowork/Claude-Desktop 3p-config file writer, so the
  config-file rewrite half of these rows is N/A. **Decision: port the PURE,
  parity-meaningful logic (`buildToolPolicy`/`buildManagedServers`/`stripServerPrefix`/
  `sanitizePluginName`) as unit-tested functions, apply them where g0router has a
  surface (tools-list prefix-strip PAR-MCP-046, create-name sanitize PAR-MCP-048,
  allowlist filter PAR-MCP-049 via the launcher), and mark 018/019/050 HAVE-SUBSET**
  (the builders exist + are tested; no config-file writer). Expose a managed-server
  READ endpoint ONLY if a frozen page consumes it (it does not). RECOMMENDED; flag.
- **ESC-ANTIGRAVITY (RESOLVED at authoring — PAR-MCP-060 ride-along, binding default).**
  The antigravity EXECUTOR is w7-prov-special. **Decision: this plan ports ONLY the
  tool-definition ride-along — a PURE constant `unavailableAntigravityTool`
  (`mcp_sequential-thinking_sequentialthinking`, `{description:"This tool is currently
  unavailable.", unavailable:true}`) included in the tools list** (PAR-MCP-060
  HAVE-subset). The executor wiring is w7-prov-special; do NOT add an antigravity
  executor here. If the tools-list ride-along is unwanted, drop it (zero page impact —
  the spec only checks read_file/write_file). RECOMMENDED; flag.
- **ESC-COWORK-CONFIG (RESOLVED — PAR-MCP-050 reset semantics).** 9router's DELETE
  resets the Cowork 3p config + skip-approvals + 1p-legacy files. g0router has no such
  files. **Decision: `DeleteInstance` removes the instance + best-effort stops the
  launcher bridge (`Launcher.Stop`); the file-reset half is N/A and recorded as
  HAVE-subset.** No fabrication of a Cowork config surface.
- **ESC-CLIENT-SRC (CONDITIONAL — the marketplace clients source).** The
  `McpMarketplaceModal` browses `GET /api/mcp/clients` expecting Filesystem+GitHub-
  style entries (`seed/mcp.ts:3-25` + the spec asserts "GitHub" renders). The store
  may have zero clients at runtime. **Default: `ListClients` returns the union of
  `store.ListMCPClients()` ∪ `DefaultPlugins()` mapped to `clientDTO`** so the
  marketplace always shows the default catalog (Exa/Tavily/browsermcp + any
  registered) — the spec's "GitHub" assertion is satisfied by the SEED in the e2e mock
  (the mock serves its own seed; the Go default catalog satisfies the REAL runtime).
  Decide the exact mapping at T-clients; if a registry-backed catalog is preferred,
  use `Registry.List` — ESCALATE only if the default-plugins union can't satisfy the
  spec. Note: the e2e spec runs against the MOCK (seeded GitHub), so the Go source
  choice does NOT affect spec green; it affects real-runtime behavior only.
- **ESC-SKILLS-SRC (RESOLVED — static catalog default).** The frozen `/skills` page is
  read-only (no create/delete control; spec asserts 2 grouped rows + copy). **Decision:
  ship a static Go `skillsCatalog()` (no store table)** mirroring the seed shape. A
  `skills` store table is the alternative only if operator-managed skills are later
  wanted (NOT indicated). RECOMMENDED; flag.
- **ESC-OAUTH-REDIRECT (CONDITIONAL — `auth/start` redirect URI).** `Engine.Start`
  needs a `redirectURI`. **Default: a configured admin callback path (the same
  derivation the existing `internal/admin/oauth.go` provider flow uses, or a
  request-derived base + a fixed `/api/mcp/auth/callback` path).** Decide at T-auth by
  reading how the shipped provider-OAuth `auth/start` derives its redirect; mirror it.
  If no precedent exists, ESCALATE — never hardcode a wrong callback.
- **ESC-PROBE-FIELD (CONDITIONAL — tools-list source field).** Whether the handler
  holds a `mcpProbe *mcp.Probe` + a `toolsCache`, or reads instance-cached tools, is a
  wiring detail. **Default: a `mcpProbe *mcp.Probe` field (injected via a setter) + a
  handler-owned discovery cache**; the unit test injects `NewProbe(&http.Client{
  Transport:fakeTransport})`. Decide at T-wire; keep it nil-able (degraded list from
  store when unset).
- **ESC-BOOTSTRAP (CONDITIONAL — where the real launcher/engine is wired).** The
  server bootstrap must call `SetMCPLauncher`/`SetMCPEngine` with the real
  `NewLauncher(st)`/`NewEngine(st,nil)` after `New`, the SAME place
  `SetTunnelRunner`/`SetMitmProxy` are called. **Default: an additive one-line append
  in that bootstrap file** (it joins the diff-gate allowlist, §7). Decide the exact
  file at T-wire by grepping where `SetMitmProxy` is invoked; mirror it. If the
  bootstrap wiring is non-trivial, ESCALATE.
- **Serial-slot dependency (§1.7 / P6).** w7-mcp-3 TAKES the routes_admin.go slot
  after w7-gov-3 releases it (chain MAP §221) and RELEASES it to w7-plat-1 on close.
  Orchestrator confirms exactly one unmerged holder (decision 3) before T-routes.
- **No other blocking dependency.** All consumed surfaces (the w7-mcp-1/2 engine, the
  MCP store, the gov-1 admin/recordAudit/setter-injection pattern, the additive
  migrate pattern) are SHIPPED in-tree at <base>. w7-mcp-3 is unblocked once the
  serial slot is free.
```
