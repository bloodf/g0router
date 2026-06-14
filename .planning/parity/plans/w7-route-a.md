# Micro-plan w7-route-a — Routing admin backends: aliases + routing-rules + model-limits + combos-DTO reconciliation + /api/quota (Go)

```
wave: 7
plan: w7-route-a  (split half A of the original w7-route; see §0 SPLIT NOTE)
status: READY (rev 1 — authored against merged Waves 0–6 + the shipped W7
  governance/platform/platnodes plans, live tree @ <base>; WAVE-7-MAP w7-route row
  ~line 174; serial chain §219-224; e2e reconciliation §245; freeze rules §267)
runs: governance+routing track. TAKES the internal/server/routes_admin.go SERIAL
  SLOT (chain: w7-platnodes → **w7-route-a** → w7-gov-1 → w7-gov-2 → w7-gov-3 →
  w7-mcp-3 → w7-plat-1 → w7-plat-2 → w7-plat-3 → w7-misc; MAP §219-224).
  Disjoint store/admin files from the sibling plans (run ∥). Depends on
  w7-platnodes (merged — provider-node prefix routing prerequisite).
branch: main (direct commits, per repo convention)
commit-prefix: phase-1/w7-route-a:
ref-source: 9router frozen @ 827e5c3 — aliases/routing-rules/model-limits/combos/
  quota admin surfaces; the BINDING contract for W7 is the W6 e2e mock (decision 1:
  real Go wins, mock corrected to mirror it). Mock sources:
    ui/e2e/mocks/handlers/{aliases,routing-rules,model-limits,combos,quota}.ts
    + ui/e2e/mocks/seed/{aliases,routing-rules,model-limits,combos}.ts
    + ui/e2e/mocks/seed/usage.ts (seedQuota).
base: <base> = git rev-parse HEAD recorded at P0. Substitute the actual SHA
  everywhere §5 says <base>.
go-serial-slot: this plan holds the SINGLE in-flight edit to
  internal/server/routes_admin.go while live (W3/W4/W5/W6 lesson; MAP decision 3).
  As the SECOND chain holder (after w7-platnodes) the slot is free at P-check;
  RELEASE to w7-gov-1 on close.
new-route: NO UI route files. All five UI pages (/aliases, /routing-rules,
  /model-limits, /combos, /quota) ALREADY SHIPPED in w6-h / w6-g against mocks; this
  plan builds the REAL Go so the pages flip mock→true-HAVE and corrects the mock
  bodies to mirror the Go DTOs.
```

---

## 0. SPLIT NOTE (binding — why w7-route became w7-route-a + w7-route-b)

The original w7-route row (WAVE-7-MAP ~line 174) bundled (1) routing **admin CRUD +
quota** and (2) the **dynamic routing engine** (selection.go / factory.go /
ModelsHandler edits). These two halves are **file-disjoint** and have **different
serial dependencies**:

- **w7-route-a (THIS PLAN)** = admin CRUD + quota. Pure transport+store work,
  hermetic via `newTestEnv`, mirrors w7-gov-1 exactly. Takes the **routes_admin.go
  serial slot** and releases it to w7-gov-1 — keeping the routes_admin serial chain
  moving.
- **w7-route-b** (`.planning/parity/plans/w7-route-b.md`) = dynamic routing engine
  (PAR-ROUTE-027/035/039/053/055/056/059/060). Inference-path work touching
  `internal/inference/selection.go`, `factory.go`, `internal/server/routes_openai.go`
  (`ModelsHandler`), and providers. It needs **NO routes_admin slot** (its surfaces
  are `/v1/models` + inference-internal, not admin routes) but DOES need the
  **selection.go micro-serial** vs w7-plat-1 (MAP §234-238 / §267).

The two plans touch **zero shared Go files** (verified §3 / §7 of each), so they run
**in parallel** — w7-route-a queues for the routes_admin slot; w7-route-b coordinates
selection.go with w7-plat-1. Splitting unblocks the routes_admin chain immediately
instead of stalling it behind the larger engine work. Per the brief's explicit
authorization to split when warranted, this split is warranted.

---

## 1. Scope — open-questions items + the five admin surfaces

### Rows / items this plan closes

| Row / item | Claim | Target state after w7-route-a |
|---|---|---|
| open-questions w6-h **ESC-2** (aliases admin absent) | real `/api/aliases` admin CRUD | true-HAVE (Go — NEW `internal/admin/aliases.go` over existing `store` alias methods, §1.4) |
| open-questions w6-h **ESC-3a** (routing-rules backend absent) | real `/api/routing-rules` store + CRUD | true-HAVE (Go — NEW `internal/store/routingrules.go` + `internal/admin/routingrules.go`, §1.5) |
| open-questions w6-h **ESC-3b** (model-limits backend absent) | real `/api/model-limits` store + CRUD | true-HAVE (Go — NEW `internal/store/modellimits.go` + `internal/admin/modellimits.go`, §1.6) |
| open-questions w6-h **ESC-1** + WAVE-7-MAP combos-reconciliation note (~line 81) | combos DTO reconciled to the frozen UI `Combo` shape | true-HAVE via a NEW admin combos surface (§1.7 / §8 ESC-COMBOS) — the existing `{name,models[]}` combo engine is UNTOUCHED |
| open-questions w6-g **ESC-1c** / PAR-UI-012 (no `/api/quota` Go source) | real aggregating `GET /api/quota` | true-HAVE (Go — NEW `internal/admin/quota.go` aggregating per-connection usage, §1.8) |

Matrix flips at closeout (§4 T-close): in `.planning/parity/matrix/9router-ui.md`,
PAR-UI-012 (quota) → HAVE (real Go); PAR-UI-116 (aliases), PAR-UI-091..094 (combos),
the PAR-UI-130 aliases/routing-rules/model-limits subsets → mock→true-HAVE. Mark
`open-questions.md` w6-h ESC-1/2/3a/3b + w6-g ESC-1c RESOLVED with a cite to this
plan; append any new open items (§8).

### 1.1 Preconditions already satisfied by merged waves (evidence)

- **W6-h / w6-g UIs are SHIPPED and FROZEN (consume-only, MAP decision 8 / §267).**
  The binding acceptance contracts are the existing specs (must stay green at
  closeout). Verified assertions (read directly from the spec files):
  - `ui/e2e/aliases.spec.ts` — 4 tests: page loads ("Aliases"); `[data-testid="alias-row"]`
    ==3 from seed (renders "gpt4" + "gpt-4o"); modal `#alias-name` + `[data-testid="alias-save"]`
    fires a **POST `/api/aliases`**; delete via confirm modal ("Delete").
  - `ui/e2e/routing-rules.spec.ts` — 4 tests: page loads ("Routing");
    `[data-testid="routing-rule-row"]` ==2 from seed (renders "Route GPT-4 to
    OpenAI"); modal `#routing-rule-name` + `[data-testid="routing-rule-save"]`
    fires a **POST `/api/routing-rules`**; delete confirm.
  - `ui/e2e/model-limits.spec.ts` — 3 tests: page loads ("Model Limits");
    `[data-testid="model-limit-row"]` ==2 from seed (renders "gpt-4o" + "128000");
    modal `#model-limit-model` + `[data-testid="model-limit-save"]` fires a **POST
    `/api/model-limits`**.
  - `ui/e2e/combos.spec.ts` — 5 tests: page loads ("Combos"); `[data-testid="combo-row"]`
    ==2 (renders "Fast + Cheap" + "Best Quality"); edit opens
    `[data-testid="combo-step-row"]` ==2 in seed order ("llama-3-70b" then
    "gpt-4o-mini"); **save fires a PUT to `/api/combos/{id}` whose body is
    `{steps:[{model}]}` with `["llama-3-70b","gpt-4o-mini"]`**; delete confirm.
  - `ui/e2e/quota.spec.ts` — 2 tests: page loads ("Quota");
    `[data-testid="quota-card"]` + `[data-testid="quota-progress"]` visible.
- **The frozen UI types are the canonical DTO targets** (`ui/src/lib/types.ts`):
  - `Alias` = `{id, alias, provider, model}` (`types.ts:22`). **NOT** the Go store's
    `{name, target}` (`internal/store/aliases.go:11-15`) — major divergence, §1.4 /
    §8 ESC-ALIAS-SHAPE.
  - `RoutingRule` = `{id, name, priority, cond_field, cond_operator, cond_value,
    target_provider, is_active, created_at}` (`types.ts:232`).
  - `ModelLimit` = `{id:number, model, max_tokens, max_rpm, allowed_key_ids:string[],
    created_at}` (`types.ts:170`) — note `id` is a **number** (§8 ESC-IDTYPE,
    same divergence w7-gov-2 hit for feature-flags/prompt-templates).
  - `Combo` = `{id, name, strategy, steps:Array<{provider,model}>, is_active}`
    (`types.ts:70`) — irreconcilable with the Go combo engine's `{name,models[]}`
    (§1.7 / §8 ESC-COMBOS).
  - `Quota` = `{connection_id, provider, connection_name, account_label?, plan,
    used, limit, unit, reset_at, is_active}` (`types.ts:219`).
- **The alias STORE already exists** (`internal/store/aliases.go`): `ModelAlias{Name,
  Target,CreatedAt}` + `CreateAlias(name,target)` (cycle-checked, INSERT OR REPLACE,
  `aliases.go:19`), `ListAliases()` (`aliases.go:65`), `DeleteAlias(name)`
  (`aliases.go:87`), `ResolveChain(name)` (`aliases.go:104`). **Consumed by the live
  gateway** (`routes_openai.go:205-206`) — DO NOT break these signatures (§8
  ESC-ALIAS-SHAPE explains the id/provider/model vs name/target reconciliation).
- **Envelope + handler patterns** (`internal/admin/respond.go`): `writeData(ctx,
  status, data)` / `writeError(ctx, status, message)` → `{data,error:{message}}`
  snake_case. `pathID(ctx.UserValue("id"))` extracts string `{id}`
  (`handlers.go:158`). CRUD template = `internal/admin/virtualkeys.go`
  (List/Create/Get/Update/Delete + DTO + request structs + validate + ErrNotFound→404).
- **Store CRUD template** (`internal/store/virtualkeys.go`): `newID()` for TEXT ids,
  `time.Now().Unix()` timestamps, `boolToInt` for SQLite bools, `scanX` helper,
  `ErrNotFound` on `sql.ErrNoRows`, JSON-blob column for nested data (the
  `allowed_key_ids []string` model-limits field → a `key_ids_json` blob, §1.6).
- **The audit seam shipped in w7-gov-1** — `internal/admin/audit.go:64`
  `func (h *Handlers) recordAudit(ctx, action, target, details string)` (resolves the
  actor from `ctx.UserValue(userKey).(*store.User)`, best-effort, logs on failure).
  REUSE `h.recordAudit` on every mutation (NO audit-write retrofit into other files).
- **The per-connection usage source EXISTS** — `GET /api/usage/{connectionId}`
  (`internal/admin/connectionusage.go:61` `GetConnectionUsage`), backed by
  `usage.FetchProviderUsage(providerType, conn, client, baseURL...)`
  (`internal/usage/providerusage.go:132`) for oauth connections; non-oauth returns
  "Usage not available". `store.ListConnections()` (`connections.go:55`) +
  `store.GetConnection(id)` (`connections.go:79`) + `store.GetProvider(id)` enumerate
  connections + provider types — the aggregation inputs for `/api/quota` (§1.8).
- **Migrations are additive-only** (`internal/store/migrate.go`): new tables via the
  `tables []struct{name,create}` slice with `CREATE TABLE IF NOT EXISTS`
  (`migrate.go:15-280`); the indexed-table slice at `migrate.go:294-301`; new columns
  via `ensureColumn(db,table,column,decl)` (`migrate.go:406`).
- **Admin test harness** (`internal/admin/admin_test.go`): `newTestEnv(t)`
  (`admin_test.go:24`) = real `store.Open(tempDB, secret)` + `auth.NewSessions` +
  `SeedAdmin("admin","123456")` (`admin_test.go:38`) + `New(st,sessions,nil)`. NO
  mocks. `call(t, h, method, uri, body, userValues, headers)` (`admin_test.go:72`)
  drives a handler + decodes the envelope; `dataField[T]` (`admin_test.go:97`) /
  `loginToken` (`admin_test.go:125`) helpers. Authoritative proof surface.
- **Handlers injection** (`internal/admin/handlers.go:39`): `New(st, sessions, flows)`
  — handlers compose `h.store` directly (like virtualkeys); the w7-gov-1 `h.audit`
  field is already present. NO new global state, NO `New(...)` signature change
  (MAP decision 9).

### 1.2 The mock contracts these flips must mirror (binding — decision 1)

**Decision 1 (MAP §36, §245):** real Go wins; the W6 mock body + seed are corrected
IN THIS PLAN to mirror the real Go `{data,error}` snake_case DTO. The page is FROZEN
(decision 8); prefer matching the mock's existing field names in the Go DTO (they were
modeled to match the frozen UI types); only ESCALATE when impossible (combos, §8).
Per-domain reconciliation (mock bodies read directly):

**Aliases** (`handlers/aliases.ts` + `seed/aliases.ts`):
- Routes: `GET /api/aliases` (bare array under `{data}`), `POST /api/aliases`
  (`{id:nextId(), ...body}`), `GET|PUT|DELETE /api/aliases/{id}` (regex
  `\/api\/aliases\/[^/]+$`). DELETE → `{}`.
- Mock/seed shape = UI `Alias` `{id, alias, provider, model}` (seed ids
  `alias-1..3`). **Divergence:** Go store keys aliases by `name` with a single
  `target` string; the UI carries `{alias, provider, model}`. Reconciliation in §1.4
  / §8 ESC-ALIAS-SHAPE (RECOMMENDED: a NEW typed `aliases` admin table mirroring the
  UI shape; the existing `model_aliases` store stays the gateway's resolution source).

**Routing-rules** (`handlers/routing-rules.ts` + `seed/routing-rules.ts`):
- Routes: `GET /api/routing-rules` (bare array), `POST` (`{id:nextId(),
  is_active:true, created_at:ISO, ...body}`), `GET|PUT|DELETE /api/routing-rules/{id}`.
  DELETE → `{}`.
- Mock/seed shape = UI `RoutingRule` (8 fields + id; seed ids `rule-1..2`). Canonical
  Go DTO. `id` is a string in the seed (`rule-1`) → TEXT-PK `newID()` (§1.5).

**Model-limits** (`handlers/model-limits.ts` + `seed/model-limits.ts`):
- Routes: `GET /api/model-limits` (bare array, mock lazily seeds), `POST`
  (`{id:Date.now(), created_at:ISO, ...body}`), `GET|PUT|DELETE
  /api/model-limits/{id}`. DELETE → `{}`.
- Mock/seed shape = UI `ModelLimit` `{id:number, model, max_tokens, max_rpm,
  allowed_key_ids:string[], created_at}` (seed ids `1`,`2`). **`id` is numeric** →
  INTEGER-PK (§8 ESC-IDTYPE — follow the w7-gov-2 precedent: `strconv.ParseInt` for
  `{id}`, NOT the string-only `pathID`). `allowed_key_ids` → a JSON blob column.

**Combos** (`handlers/combos.ts` + `seed/combos.ts`):
- Routes: `GET /api/combos` (bare array), `POST` (`{id:nextId(), is_active:true,
  ...body}`), `GET|PUT|DELETE /api/combos/{id}` (regex `\/api\/combos\/[^/]+$`).
- Mock/seed shape = UI `Combo` `{id, name, strategy, steps:[{provider,model}],
  is_active}` (seed ids `combo-1..2`). **The spec PUTs `/api/combos/{id}` with body
  `{steps:[{model}]}`** (`combos.spec.ts:53-66`). The REAL Go combos engine is
  `{name, models[]}` keyed by `name` (`internal/admin/combos.go:15-21`,
  `internal/store/combos.go` table `combos(name PRIMARY KEY, models_json)`,
  `routes_admin.go:192-195` `PUT /api/combos/{name}`). **IRRECONCILABLE by additive
  enrichment** — see §1.7 / §8 ESC-COMBOS.

**Quota** (`handlers/quota.ts` + `seed/usage.ts` `seedQuota`):
- Route: `GET /api/quota` → bare array under `{data}` of `Quota` objects
  (`handlers/quota.ts:7` returns `store.quotas`). Seed = 4 entries `{connection_id,
  provider, connection_name, account_label?, plan, used, limit, unit, reset_at,
  is_active}` (`seed/usage.ts:31-39`). Canonical Go DTO; built by aggregation (§1.8).

### 1.3 Architecture (binding — layered DDD, decision 4)

Five surfaces, layered transport → (domain) → repository:

```
aliases:        admin/aliases.go      → store/aliases.go (REUSE) + NEW aliases admin table (§8 ESC-ALIAS-SHAPE)
routing-rules:  admin/routingrules.go → store/routingrules.go (NEW table routing_rules)
model-limits:   admin/modellimits.go  → store/modellimits.go  (NEW table model_limits)
combos-admin:   admin/combos_admin.go → store/combosadmin.go  (NEW table combos_admin — §1.7, separate from the engine's combos table)
quota:          admin/quota.go        → aggregates store.ListConnections + the existing per-connection usage fetcher (no store table)
```

**Arch-test note (ESC-ARCH).** AGENTS.md cites a phase-12B arch test
(transport→domain→repository). No standalone `arch_test.go` was found in-tree at
authoring; the established pattern is handler→store DIRECT for pure CRUD
(virtualkeys/apikeys/combos all call `h.store` with no domain wrapper). **Default:
follow that precedent** — handler→store directly for the four CRUD domains; add a
`internal/governance/*` (or `internal/routing/*`) domain wrapper ONLY if the arch
test (run at T-each) forbids the new transport→store edge. Quota has no store; it is
pure transport aggregation over existing store + usage seams. Decide per-domain at
its task by running `go test ./...` (which would surface any arch test); do NOT
pre-build domain wrappers.

### 1.4 Aliases Go contract (NEW admin table, TDD) — ESC-ALIAS-SHAPE

The frozen UI `Alias` is `{id, alias, provider, model}` (id-keyed CRUD via
`/api/aliases/{id}`). The existing `store.ModelAlias` is `{name, target}` keyed by
`name` and is consumed by the live gateway resolver (`routes_openai.go:205`) — it
CANNOT be reshaped without breaking resolution. **Decision (ESC-ALIAS-SHAPE,
RECOMMENDED default):** add a NEW typed admin table `aliases` mirroring the UI shape;
the existing `model_aliases` table + `ListAliases`/`CreateAlias`/`DeleteAlias`/
`ResolveChain` stay the gateway's resolution source UNCHANGED. The admin handler
optionally mirror-writes the resolution alias (`alias` → `provider/model` target) into
`model_aliases` so admin edits feed the live resolver — but the mirror-write is a
best-effort additive convenience (the binding contract is the id-keyed admin DTO the
frozen page reads). If the operator prefers the admin endpoint to drive the EXISTING
`model_aliases` directly (deriving `{id:=name, alias:=name, provider/model:=split(target)}`),
that is the zero-new-table alternative — but it loses the distinct `provider`+`model`
fields the UI form sets and forces a synthetic id. RECOMMENDED: the NEW typed table
(clean id-keyed CRUD matching the frozen page). Flag for orchestrator confirmation.

Table `aliases` (additive, `migrate.go` tables slice):
```sql
CREATE TABLE IF NOT EXISTS aliases (
  id TEXT PRIMARY KEY,
  alias TEXT NOT NULL,
  provider TEXT NOT NULL DEFAULT '',
  model TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
)
```

`internal/store/aliasesadmin.go` (NEW — distinct file from the gateway `aliases.go`):
`AliasRecord{ID,Alias,Provider,Model,CreatedAt,UpdatedAt}` + `CreateAliasRecord`/
`ListAliasRecords`/`GetAliasRecordByID`/`UpdateAliasRecord`/`DeleteAliasRecord` +
`scanAliasRecord` (mirror `virtualkeys.go`: `newID()`, unix ts, `ErrNotFound`).
(NEW FILE name avoids editing the gateway `internal/store/aliases.go` — keeps the
resolution path frozen.)

`internal/admin/aliases.go` (NEW):

| Handler | Route | Shape (snake_case, `{data}`) | Notes |
|---|---|---|---|
| `ListAliases` | `GET /api/aliases` | bare array under `{data}` of `aliasDTO` | `aliasDTO{id,alias,provider,model}` |
| `CreateAlias` | `POST /api/aliases` | body `{alias,provider?,model?}`; returns `{data:aliasDTO}`; 400 on empty `alias` | mirror-write to `model_aliases` best-effort (ESC-ALIAS-SHAPE) |
| `GetAlias` | `GET /api/aliases/{id}` | `{data:aliasDTO}` or 404 | |
| `UpdateAlias` | `PUT /api/aliases/{id}` | body = create body; updated `{data:aliasDTO}` or 404 | |
| `DeleteAlias` | `DELETE /api/aliases/{id}` | `{data:{message:"Alias deleted successfully"}}` or 404 | mock returns `{}`; page ignores body |

(Handler type name `ListAliases`/`DeleteAlias` collides with no existing method —
the gateway alias logic lives in `store`, not `admin`; verify at T-aliases there is
no name clash in package `admin`; rename to `ListAliasAdmin` etc. if a collision
exists — §8 ESC-NAME.)

### 1.5 Routing-rules Go contract (NEW, TDD)

Table `routing_rules` (additive):
```sql
CREATE TABLE IF NOT EXISTS routing_rules (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  priority INTEGER NOT NULL DEFAULT 0,
  cond_field TEXT NOT NULL DEFAULT '',
  cond_operator TEXT NOT NULL DEFAULT '',
  cond_value TEXT NOT NULL DEFAULT '',
  target_provider TEXT NOT NULL DEFAULT '',
  is_active INTEGER NOT NULL DEFAULT 1,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
)
```
`internal/store/routingrules.go` (NEW): `RoutingRule` struct + `CreateRoutingRule`/
`ListRoutingRules` (ORDER BY priority ASC, created_at ASC) /`GetRoutingRuleByID`/
`UpdateRoutingRule`/`DeleteRoutingRule` + `scanRoutingRule` (`newID()`, unix ts,
`boolToInt`, `ErrNotFound`).

`internal/admin/routingrules.go` (NEW): `ListRoutingRules`/`CreateRoutingRule`/
`GetRoutingRule`/`UpdateRoutingRule`/`DeleteRoutingRule` + `routingRuleDTO{id,name,
priority,cond_field,cond_operator,cond_value,target_provider,is_active,created_at}`
(`created_at` rendered as RFC3339 ISO to mirror the mock seed's ISO string — store
the int64 unix ts, format on read; §8 ESC-CREATED-AT). 400 on empty `name`. `{data}`
envelope; `h.recordAudit` on mutations. **This plan does NOT wire routing_rules into
live inference** — it is admin CRUD only (the live rule-application engine is a
tracked follow-up, recorded in open-questions; the frozen page only lists/creates/
deletes rules).

### 1.6 Model-limits Go contract (NEW, TDD) — INTEGER PK (ESC-IDTYPE)

Table `model_limits` (additive; INTEGER PK per the numeric UI `id`, §8 ESC-IDTYPE —
the same divergence w7-gov-2 resolved for feature_flags/prompt_templates):
```sql
CREATE TABLE IF NOT EXISTS model_limits (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  model TEXT NOT NULL,
  max_tokens INTEGER NOT NULL DEFAULT 0,
  max_rpm INTEGER NOT NULL DEFAULT 0,
  key_ids_json TEXT NOT NULL DEFAULT '[]',   -- allowed_key_ids []string as JSON blob
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
)
```
`internal/store/modellimits.go` (NEW): `ModelLimit{ID int64, Model, MaxTokens,
MaxRPM, AllowedKeyIDs []string, CreatedAt, UpdatedAt}` + `CreateModelLimit` (returns
the autoincrement id), `ListModelLimits`, `GetModelLimitByID(int64)`,
`UpdateModelLimit`, `DeleteModelLimit(int64)` + `scanModelLimit`
(JSON-marshal/unmarshal `key_ids_json`; `ErrNotFound`).

`internal/admin/modellimits.go` (NEW): handlers parse `{id}` via `strconv.ParseInt`
(NOT `pathID`, which is string-only — ESC-IDTYPE). `modelLimitDTO{id:int64, model,
max_tokens, max_rpm, allowed_key_ids:[]string, created_at:ISO}`. 400 on empty
`model`. `{data}`; `h.recordAudit` on mutations.

### 1.7 Combos DTO reconciliation (ESC-COMBOS — the binding decision)

**The conflict (evidence).** The frozen `/combos` page + spec require:
- id-keyed routes `GET|PUT|DELETE /api/combos/{id}` (`combos.spec.ts` PUTs
  `/api/combos/combo-1`),
- a `{id, name, strategy, steps:[{provider,model}], is_active}` shape,
- a PUT body of `{steps:[{provider,model}]}` (`combos.spec.ts:60-66` asserts
  `body.steps.map(s=>s.model)`).

The REAL Go combos engine is `{name, models:[]string}` keyed by `name`
(`internal/admin/combos.go:15-21`, store table `combos(name PRIMARY KEY,
models_json)`), with routes `PUT /api/combos/{name}` (`routes_admin.go:194`). It is
consumed by the live `/v1/models` combo-lister (`routes_openai.go` `SetComboLister`)
and `store.UpdateCombo(name, models)`. **The two shapes are irreconcilable by
additive enrichment**: the engine keys by name and stores a flat `[]string`; the page
keys by id and stores `[{provider,model}]` steps with `strategy`/`is_active`. Enriching
the existing read DTO with id/strategy/steps would still leave the PUT-by-id +
steps-body contract unservable by `UpdateCombo(name, []string)`, and changing the
engine's key/shape would break `/v1/models` + the existing combo store/tests
(FORBIDDEN — frozen handler bodies).

**Decision (ESC-COMBOS, RECOMMENDED default): a NEW, SEPARATE admin combos surface.**
Build a NEW `combos_admin` table + handlers that serve the frozen page's id-keyed
`{id,name,strategy,steps[],is_active}` contract, leaving the existing `{name,models[]}`
combo engine + its `/api/combos` routes + the `/v1/models` combo-lister **completely
untouched**. Because the frozen mock + page already register `/api/combos` as the
combos surface (not a new path), and the EXISTING Go `/api/combos` routes serve the
engine shape, this is a genuine path collision → **escalate to the orchestrator for
the path decision** (§8 ESC-COMBOS, two sub-options):
  - **Option A (RECOMMENDED): the admin combos surface OWNS `/api/combos[/{id}]`**
    serving the UI shape; the engine's `{name,models[]}` CRUD moves to an internal/
    differently-pathed surface OR the engine keeps its store + `/v1/models` lister
    but its admin `/api/combos` HTTP routes are RETIRED in favor of the new admin
    surface. This requires touching the EXISTING `routes_admin.go:192-195` combos
    routes (replacing the 4 engine routes with the new admin handlers) — a serial-slot
    edit within scope, BUT it changes existing route registrations (not purely
    additive) → orchestrator must confirm the engine's `/api/combos` HTTP surface has
    no other consumer (the live gateway uses the store + lister, NOT the `/api/combos`
    HTTP routes — verify at T-combos: `grep -rn '/api/combos' internal/ ui/src` shows
    only the admin page consumes the HTTP routes).
  - **Option B (zero-collision fallback): the new admin surface gets a DISTINCT path**
    (e.g. `/api/combos-admin` or `/api/combo-presets`) and the frozen mock + page
    would need a path change — but the page is FROZEN, so Option B requires a frozen-
    file edit → NOT viable without a separate frozen-file serial follow-up.

**Net recommendation:** Option A — retire the engine's `/api/combos` *HTTP routes*
(after verifying no non-page consumer), register the new admin combos handlers on
`/api/combos[/{id}]`, keep the engine's store table + `/v1/models` combo-lister
intact (the lister reads the store directly, not via HTTP). The new admin table
persists `{id,name,strategy,steps_json,is_active}`; a best-effort mirror-write keeps
the engine's `combos(name,models_json)` table populated (deriving `models` =
`steps.map(s=>provider/model or model)`) so `/v1/models` still lists combos. **This
is the single highest-risk decision in the plan — escalate with this recommended
default; do NOT fabricate a silent reshape.**

Table `combos_admin` (additive):
```sql
CREATE TABLE IF NOT EXISTS combos_admin (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  strategy TEXT NOT NULL DEFAULT 'fallback',
  steps_json TEXT NOT NULL DEFAULT '[]',     -- [{provider,model}] as JSON blob
  is_active INTEGER NOT NULL DEFAULT 1,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
)
```
`internal/store/combosadmin.go` (NEW): `ComboAdmin{ID,Name,Strategy,Steps
[]ComboStep,IsActive,...}` where `ComboStep{Provider,Model string}` +
Create/List/Get/Update/Delete + `scanComboAdmin`.
`internal/admin/combos_admin.go` (NEW): id-keyed CRUD serving `comboAdminDTO{id,name,
strategy,steps:[{provider,model}],is_active}`; PUT accepts `{steps:[{provider,model}],
name?,strategy?,is_active?}`; `{data}`; `h.recordAudit`; best-effort engine
mirror-write (ESC-COMBOS).

### 1.8 /api/quota aggregation Go contract (NEW, TDD) — no store table

`internal/admin/quota.go` (NEW): `GetQuota` handler for `GET /api/quota`. Aggregates
per-connection usage into the UI `Quota` shape. Algorithm (deterministic, hermetic):
1. `store.ListConnections()` → for each connection, `store.GetProvider(conn.ProviderID)`
   for the provider type + connection name.
2. For oauth connections, resolve usage via an **injectable fetcher seam** (default
   `usage.FetchProviderUsage`, mirroring `ConnectionUsageHandler.Fetcher` —
   `connectionusage.go:53`) so unit tests inject a deterministic fake (NO network).
   Map the provider usage `map[string]any` → `{used, limit, unit, plan, account_label,
   reset_at}` via a small per-provider/normalizing mapper (best-effort; missing fields
   default to `0`/`""`). Non-oauth connections contribute a card with `used:0,limit:0`
   or are skipped (mirror the mock which lists active connections — VERIFY which the
   page renders; default: include all connections, §8 ESC-QUOTA-SCOPE).
3. Build `[]quotaDTO{connection_id, provider, connection_name, account_label?, plan,
   used, limit, unit, reset_at, is_active}`; return bare array under `{data}`.

`quotaDTO` mirrors the UI `Quota` (`types.ts:219`). The handler takes the injectable
fetcher via a struct field (like `ConnectionUsageHandler`) so the admin test drives
it with a fake returning fixed usage maps — fully hermetic. **Determinism (binding —
AGENTS.md "No mocks; use interfaces and fakes"):** NO live provider calls in tests;
the fetcher seam is the test injection point.

### 1.9 routes_admin.go registration (serial-slot additive, §3)

Add (additive appends; static-collection before `{id}` precedence honored by the
file). For ESC-COMBOS Option A, the existing combos block (`routes_admin.go:192-195`)
is REPLACED — see §1.7 (orchestrator-confirmed); otherwise these are pure additive
appends:
```go
// Aliases admin CRUD.
r.GET("/api/aliases", h.RequireSession(h.ListAliases))
r.POST("/api/aliases", h.RequireSession(h.CreateAlias))
r.GET("/api/aliases/{id}", h.RequireSession(h.GetAlias))
r.PUT("/api/aliases/{id}", h.RequireSession(h.UpdateAlias))
r.DELETE("/api/aliases/{id}", h.RequireSession(h.DeleteAlias))
// Routing-rules CRUD.
r.GET("/api/routing-rules", h.RequireSession(h.ListRoutingRules))
r.POST("/api/routing-rules", h.RequireSession(h.CreateRoutingRule))
r.GET("/api/routing-rules/{id}", h.RequireSession(h.GetRoutingRule))
r.PUT("/api/routing-rules/{id}", h.RequireSession(h.UpdateRoutingRule))
r.DELETE("/api/routing-rules/{id}", h.RequireSession(h.DeleteRoutingRule))
// Model-limits CRUD.
r.GET("/api/model-limits", h.RequireSession(h.ListModelLimits))
r.POST("/api/model-limits", h.RequireSession(h.CreateModelLimit))
r.GET("/api/model-limits/{id}", h.RequireSession(h.GetModelLimit))
r.PUT("/api/model-limits/{id}", h.RequireSession(h.UpdateModelLimit))
r.DELETE("/api/model-limits/{id}", h.RequireSession(h.DeleteModelLimit))
// Combos admin (ESC-COMBOS Option A — REPLACES the engine's /api/combos routes;
// orchestrator-confirmed). The new admin handlers serve the id-keyed UI shape.
r.GET("/api/combos", h.RequireSession(h.ListCombosAdmin))
r.POST("/api/combos", h.RequireSession(h.CreateComboAdmin))
r.GET("/api/combos/{id}", h.RequireSession(h.GetComboAdmin))
r.PUT("/api/combos/{id}", h.RequireSession(h.UpdateComboAdmin))
r.DELETE("/api/combos/{id}", h.RequireSession(h.DeleteComboAdmin))
// Quota aggregation.
r.GET("/api/quota", h.RequireSession((&admin.QuotaHandler{Handlers: h}).GetQuota))
```
Route-precedence note: static collection before `{id}`; the combos `{name}` →
`{id}` rename is part of ESC-COMBOS Option A. A genuine `fasthttp/router`
precedence collision is §8 ESC-ROUTE, not a silent path change. Diff bound §5: the
route block is ONE commit.

### NOT in scope (explicit)

- **No UI page/component/route/store edits** — all five pages + their components are
  FROZEN consume-only (decision 8). The ONLY UI-tree touches are the mock-body + seed
  corrections (§1.2 / §3).
- **No dynamic-routing-engine edits** — selection.go / factory.go / routes_openai.go
  / providers are w7-route-b's exclusive territory (§0). w7-route-a touches NONE of
  them.
- **No edits to the EXISTING combos engine** — `internal/admin/combos.go`,
  `internal/store/combos.go`, the `/v1/models` combo-lister are FROZEN. ESC-COMBOS
  Option A only RE-REGISTERS the `/api/combos` HTTP routes to the new admin handlers
  (a routes_admin.go serial-slot edit) and mirror-writes the engine store; it does
  NOT edit the engine's handler/store bodies.
- **No edits to the gateway alias resolver** — `internal/store/aliases.go` +
  `routes_openai.go:205` are FROZEN; the admin alias surface uses a NEW table (§1.4).
- **No live routing-rule application** — routing_rules is admin CRUD only; the
  inference engine wiring is a tracked follow-up (§8 / open-questions).
- **No edits to pre-existing admin handlers' bodies** other than the ESC-COMBOS
  route re-registration in routes_admin.go.
- **No destructive DDL / column renames** — additive `ensureTable`/`ensureColumn`
  ONLY (decision 2).
- **No new global state / no `New(...)` signature change** (decision 9).
- **No secret exposure** — quota/usage responses carry no tokens (the per-connection
  fetcher already strips credentials; §5 grep proof). `allowed_key_ids` are key IDs,
  not secrets.

---

## 2. Precondition checks

Run all before any edit; abort and report to orchestrator on any failure.

```bash
# P0 — clean tree; record <base>
git status --porcelain     # must be empty (explicit `git add <file>`, never -A;
                           # ui/dist/** gitignored — never stage it)
git rev-parse HEAD         # record as <base> for §5

# P1 — the gaps are REAL
grep -nE '/api/aliases|/api/routing-rules|/api/model-limits|/api/quota' internal/server/routes_admin.go ; echo "^ expect EMPTY"
test ! -e internal/store/routingrules.go && test ! -e internal/store/modellimits.go && echo "store gap OK"
test ! -e internal/admin/aliases.go && test ! -e internal/admin/routingrules.go && test ! -e internal/admin/modellimits.go && test ! -e internal/admin/quota.go && echo "admin gap OK"
# combos engine present (frozen):
grep -n 'func (h \*Handlers) ListCombos' internal/admin/combos.go
grep -nE '/api/combos' internal/server/routes_admin.go ; echo "^ engine routes — ESC-COMBOS will re-register {name}->{id}"
# Verify the engine /api/combos HTTP routes have no non-page consumer (ESC-COMBOS):
grep -rnE '/api/combos' internal/ ui/src 2>/dev/null ; echo "^ expect only the admin page consumes the HTTP routes; the lister reads the store"

# P2 — reused surfaces present (the de-risk)
grep -nF 'func (s *Store) ListAliases' internal/store/aliases.go
grep -nF 'func (s *Store) ListConnections' 'func (s *Store) GetConnection' internal/store/connections.go
grep -nF 'func FetchProviderUsage' internal/usage/providerusage.go
grep -nF 'Fetcher func(' internal/admin/connectionusage.go     # the injectable fetcher precedent
grep -nF 'func writeData' 'func writeError' internal/admin/respond.go
grep -nF 'func (h *Handlers) recordAudit' internal/admin/audit.go
grep -nF 'func newTestEnv' 'func call' internal/admin/admin_test.go
grep -nF 'func pathID' internal/admin/handlers.go

# P3 — migrate pattern
grep -nF 'CREATE TABLE IF NOT EXISTS' internal/store/migrate.go | head
grep -nF 'func ensureColumn' internal/store/migrate.go

# P4 — the W6 UIs + specs present (consume-only) and the mocks to correct
for s in aliases routing-rules model-limits combos quota ; do test -f ui/e2e/$s.spec.ts && echo "$s spec present" ; done
for m in aliases routing-rules model-limits combos quota ; do test -f ui/e2e/mocks/handlers/$m.ts && echo "$m handler present" ; done
for d in aliases routing-rules model-limits combos ; do test -f ui/e2e/mocks/seed/$d.ts && echo "$d seed present" ; done
grep -nF 'seedQuota' ui/e2e/mocks/seed/usage.ts ; echo "^ quota seed lives in usage.ts"

# P5 — routes_admin.go serial slot FREE (w7-platnodes merged + released)
git log --oneline -5 -- internal/server/routes_admin.go   # last touch = w7-platnodes (merged)
# Orchestrator MUST confirm no concurrent W7 plan holds an unmerged routes_admin.go
# edit before w7-route-a begins T-routes (chain: w7-platnodes→**w7-route-a**→w7-gov-1).

# P6 — green at base
go test ./... && go vet ./... && go build ./...     # exit 0 (Go untouched-green)
cd ui && npm run build                               # exit 0 (BEFORE playwright — e2e-hygiene)
cd ui && npx playwright test e2e/aliases.spec.ts e2e/routing-rules.spec.ts e2e/model-limits.spec.ts e2e/combos.spec.ts e2e/quota.spec.ts
# Record base: these PASS at base against the W6 mocks. They must STAY green after
# the mock-body corrections. Record exact pass/fail in WORKFLOW.md.
```

---

## 3. Exclusive file ownership

After w7-route-a merges, all CREATE files are owned by w7-route-a; later plans
consume, never edit (MAP decision 7).

**CREATE — store (NEW):**

| File | Contract |
|---|---|
| `internal/store/aliasesadmin.go` | `AliasRecord` + Create/List/Get/Update/Delete + `scanAliasRecord`; `newID()`, unix ts, `ErrNotFound`. Distinct from the frozen gateway `aliases.go`. |
| `internal/store/aliasesadmin_test.go` | table-driven via temp `store.Open`: create→get→list→update→delete→404. RED first. |
| `internal/store/routingrules.go` | `RoutingRule` + Create/List(priority order)/Get/Update/Delete + `scanRoutingRule`; `boolToInt`. |
| `internal/store/routingrules_test.go` | create→list ordered→get→update→delete→404. RED first. |
| `internal/store/modellimits.go` | `ModelLimit` (INTEGER PK; `AllowedKeyIDs []string` ↔ `key_ids_json`) + Create(returns id)/List/Get(int64)/Update/Delete + `scanModelLimit`. |
| `internal/store/modellimits_test.go` | create→get-by-int-id→list→update→delete→404; key_ids round-trip JSON. RED first. |
| `internal/store/combosadmin.go` | `ComboAdmin` + `ComboStep` (`steps_json` blob) + Create/List/Get/Update/Delete + `scanComboAdmin`. SEPARATE from the engine `combos.go`. |
| `internal/store/combosadmin_test.go` | create→get→list→update(steps reorder persists)→delete→404. RED first. |

**EXTEND — store (additive tables only):**

| File | Change (additive ONLY) |
|---|---|
| `internal/store/migrate.go` | ADD the `aliases`, `routing_rules`, `model_limits`, `combos_admin` tables to the `tables` slice. ADDITIVE ONLY. (No edit to the existing `combos`/`model_aliases` tables.) |

**CREATE — transport (NEW):**

| File | Contract |
|---|---|
| `internal/admin/aliases.go` | `ListAliases`/`CreateAlias`/`GetAlias`/`UpdateAlias`/`DeleteAlias` + `aliasDTO` + validate; `writeData`/`writeError`; `h.recordAudit`; best-effort `model_aliases` mirror (ESC-ALIAS-SHAPE). |
| `internal/admin/aliases_test.go` | via `newTestEnv`: create→list(3 after seed-equiv)→get→update→delete→404; empty alias→400; audit-on-create. RED first. |
| `internal/admin/routingrules.go` | CRUD + `routingRuleDTO` (created_at as ISO) + validate; `h.recordAudit`. |
| `internal/admin/routingrules_test.go` | CRUD ≥5 cases; empty name→400; audit-on-create. RED first. |
| `internal/admin/modellimits.go` | CRUD; `{id}` via `strconv.ParseInt` (ESC-IDTYPE); `modelLimitDTO`; `h.recordAudit`. |
| `internal/admin/modellimits_test.go` | CRUD ≥5; numeric-id get/update/delete; empty model→400; audit. RED first. |
| `internal/admin/combos_admin.go` | id-keyed CRUD serving `comboAdminDTO{id,name,strategy,steps[],is_active}`; PUT body `{steps,...}`; best-effort engine mirror-write; `h.recordAudit`. |
| `internal/admin/combos_admin_test.go` | create→list→get→PUT(steps order persists, matches the spec's `{steps:[{model}]}` body)→delete→404; audit. RED first. |
| `internal/admin/quota.go` | `QuotaHandler{Handlers, Fetcher}` + `GetQuota`; injectable fetcher seam; `quotaDTO`; aggregates `ListConnections`. NO secret in output. |
| `internal/admin/quota_test.go` | via `newTestEnv` + a FAKE fetcher: seed N connections → `GET /api/quota` returns N cards with correct used/limit mapping; non-oauth handling; **no token/secret in any response field**. RED first. |

**MODIFY — serial-slot route registration (additive + ESC-COMBOS re-register):**

| File | Change |
|---|---|
| `internal/server/routes_admin.go` | ADD the aliases/routing-rules/model-limits/quota route lines; RE-REGISTER `/api/combos[/{id}]` to the new admin combos handlers (ESC-COMBOS Option A, orchestrator-confirmed — replaces the 4 engine `/api/combos` route lines). ONE commit. SERIAL SLOT — only holder while live; RELEASE to w7-gov-1 on close. |

**MODIFY — e2e mock corrections (mirror real Go, decision 1):**

| File | Change |
|---|---|
| `ui/e2e/mocks/handlers/aliases.ts` (BODY) | Mirror the Go `{data}` envelope + `aliasDTO`; DELETE → `{data:{message}}` (page ignores body). Verify field names match seed. |
| `ui/e2e/mocks/handlers/routing-rules.ts` (BODY) | Mirror Go `{data}` + `routingRuleDTO` (created_at ISO). |
| `ui/e2e/mocks/handlers/model-limits.ts` (BODY) | Mirror Go `{data}` + numeric id + `allowed_key_ids`; keep the lazy-seed convenience. |
| `ui/e2e/mocks/handlers/combos.ts` (BODY) | Mirror the new admin combos `{data}` + `comboAdminDTO` (id-keyed, steps). |
| `ui/e2e/mocks/handlers/quota.ts` (BODY) | Already returns `store.quotas` under `{data}` — verify the envelope matches Go bare-array-under-`{data}`. |
| `ui/e2e/mocks/seed/{aliases,routing-rules,model-limits,combos}.ts` (BODY) | Verify field names mirror the Go DTOs; correct only on divergence. `seed/usage.ts` `seedQuota` — verify; no change expected. |

**FORBIDDEN:** everything else. Explicitly: all pre-existing `internal/admin/*.go`
except the NEW aliases/routingrules/modellimits/combos_admin/quota files; the EXISTING
`internal/admin/combos.go` + `internal/store/combos.go` + `internal/store/aliases.go`
(FROZEN — no body edit; ESC-COMBOS only re-registers routes); all
`internal/inference/*` + `internal/server/routes_openai.go` + `internal/providers/*`
(w7-route-b territory); all other `internal/store/*.go` except the four NEW files +
migrate (additive); all UI `ui/src/**` (FROZEN, decision 8); all other mocks/seeds/
specs; `ui/package.json` + lockfile; `ui/vite.config.ts`; `ui/playwright.config.ts`;
`ui/dist/**` (gitignored — NEVER stage, NEVER revert `ui/dist/index.html`). Touching
any of these is an automatic REJECT.

---

## 4. TDD tasks

Cadence (strict, AGENTS.md "TDD always"): **no Go impl file may exist before its
`_test.go` is committed RED.** `go test ./... && go vet ./... && go build ./...` green
at EVERY commit. The five e2e specs stay green throughout (real Go is additive; mock
corrections mirror it). The four CRUD domains are independent — order: aliases →
routing-rules → model-limits → combos-admin → quota, then the single serial-slot
routes commit, then mock corrections + closeout.

### T-aliases — STEP(a) RED store+admin, STEP(b) impl
STEP(a): write `internal/store/aliasesadmin_test.go` + `internal/admin/aliases_test.go`;
add the `aliases` table to `migrate.go`. Targeted runs FAIL. Commit RED:
`phase-1/w7-route-a: failing aliases admin store+admin tests (TDD red)`.
STEP(b): implement `internal/store/aliasesadmin.go` + `internal/admin/aliases.go`
(+ best-effort `model_aliases` mirror). Gates green. Commit:
`phase-1/w7-route-a: aliases admin store + CRUD`.

### T-routingrules — STEP(a) RED, STEP(b) impl
STEP(a): tests + `routing_rules` table → FAIL. Commit RED:
`phase-1/w7-route-a: failing routing-rules store+admin tests (TDD red)`.
STEP(b): impl. Gates green. Commit:
`phase-1/w7-route-a: routing-rules store + admin CRUD`.

### T-modellimits — STEP(a) RED, STEP(b) impl (INTEGER PK)
STEP(a): tests (numeric-id cases) + `model_limits` table → FAIL. Commit RED:
`phase-1/w7-route-a: failing model-limits store+admin tests (TDD red)`.
STEP(b): impl (`strconv.ParseInt` ids). Gates green. Commit:
`phase-1/w7-route-a: model-limits store + admin CRUD (INTEGER PK)`.

### T-combos-admin — STEP(a) RED, STEP(b) impl (ESC-COMBOS)
PRE: orchestrator confirms ESC-COMBOS Option A (re-register `/api/combos` to the
admin surface; engine store + `/v1/models` lister untouched). If unconfirmed, STOP
and ESCALATE (do not guess).
STEP(a): `combosadmin_test.go` + `combos_admin_test.go` (PUT-steps-order assertion
mirroring `combos.spec.ts`) + `combos_admin` table → FAIL. Commit RED:
`phase-1/w7-route-a: failing combos-admin store+admin tests (TDD red)`.
STEP(b): impl + best-effort engine mirror-write. Gates green. Commit:
`phase-1/w7-route-a: combos admin store + id-keyed CRUD (ESC-COMBOS)`.

### T-quota — STEP(a) RED, STEP(b) impl (hermetic fetcher seam)
STEP(a): `quota_test.go` via `newTestEnv` + a fake fetcher (no network) → FAIL.
Commit RED: `phase-1/w7-route-a: failing /api/quota aggregation tests (TDD red)`.
STEP(b): implement `internal/admin/quota.go` (injectable fetcher). Gates green.
Commit: `phase-1/w7-route-a: /api/quota aggregation over per-connection usage`.

### T-routes — serial-slot route registration (+ ESC-COMBOS re-register)
TAKE the serial slot (orchestrator confirms FREE at P5). Add the aliases/routing-
rules/model-limits/quota routes + re-register `/api/combos[/{id}]` (ESC-COMBOS).
Gates green. Commit (ONE commit touches the serial file):
`phase-1/w7-route-a: register routing-admin + quota routes (serial slot)`.

### T-mocks — mock-body corrections (mirror real Go, decision 1)
Correct the five handler bodies + verify seeds. Gates: `cd ui && npm run build`
green (BEFORE playwright); the five specs green (still) in ONE plain playwright
invocation. If a correction reds a non-w7-route-a spec, STOP + ESCALATE (§8
ESC-MOCK). Commit:
`phase-1/w7-route-a: correct aliases/routing-rules/model-limits/combos/quota mocks to mirror real Go`.

### T-close — full gates + closeout
```bash
go test ./... && go vet ./... && go build ./...
go test ./internal/admin/ -run 'Alias|Routing|ModelLimit|Combo|Quota' -v
go test ./internal/store/ -run 'Alias|Routing|ModelLimit|Combo' -v
cd ui && npm run build                                       # BEFORE playwright
cd ui && npx playwright test e2e/aliases.spec.ts e2e/routing-rules.spec.ts e2e/model-limits.spec.ts e2e/combos.spec.ts e2e/quota.spec.ts   # green (ISOLATED)
cd ui && npx playwright test                                 # full suite green (no regressions)
```
Flip `.planning/parity/matrix/9router-ui.md`: PAR-UI-012 (quota) → HAVE; aliases/
combos/routing-rules/model-limits subsets mock→true-HAVE (cite §1.4-1.8). Mark
`open-questions.md` w6-h ESC-1/2/3a/3b + w6-g ESC-1c RESOLVED with a cite; append the
new open items (§8 — live routing-rule application follow-up; combos engine HTTP-route
retirement note). Update `docs/WORKFLOW.md` (P6 base observation; the ESC-COMBOS /
ESC-ALIAS-SHAPE / ESC-IDTYPE / ESC-QUOTA-SCOPE decisions; the serial-slot take-from-
w7-platnodes / release-to-w7-gov-1; the mock corrections). Final commit:
`phase-1/w7-route-a: close — routing-admin + quota Go; matrix flip; mock mirror`.
**On the close commit, RELEASE the routes_admin.go serial slot to w7-gov-1.**

---

## 5. Binary acceptance criteria

All must hold; each is yes/no. `<base>` = the commit recorded at P0. Diff gate is
**w7-route-a commit-range-scoped** (§7).

**Test gates**
- `go test ./... && go vet ./... && go build ./...` → exit 0.
- `go test ./internal/admin/ -run 'Alias|Routing|ModelLimit|Combo|Quota' -v` → exit 0,
  all pass (aliases CRUD ≥5; routing-rules ≥5; model-limits ≥5 incl numeric-id;
  combos-admin ≥5 incl PUT-steps-order; quota ≥3 incl no-secret-leak).
- `go test ./internal/store/ -run 'Alias|Routing|ModelLimit|Combo' -v` → exit 0.
- `cd ui && npm run build` → exit 0 (BEFORE playwright).
- `cd ui && npx playwright test e2e/aliases.spec.ts e2e/routing-rules.spec.ts e2e/model-limits.spec.ts e2e/combos.spec.ts e2e/quota.spec.ts` → exit 0, all pass
  (4+4+3+5+2), 0 skipped (ISOLATED; no concurrent playwright; `ui/dist/index.html`
  NEVER reverted).
- `cd ui && npx playwright test` → exit 0, no spec green-at-base goes red.

**TDD-order proof** — each impl file's covering test appears in an earlier-or-equal
commit:
```bash
for pair in \
  "internal/store/aliasesadmin_test.go:internal/store/aliasesadmin.go" \
  "internal/store/routingrules_test.go:internal/store/routingrules.go" \
  "internal/store/modellimits_test.go:internal/store/modellimits.go" \
  "internal/store/combosadmin_test.go:internal/store/combosadmin.go" \
  "internal/admin/aliases_test.go:internal/admin/aliases.go" \
  "internal/admin/routingrules_test.go:internal/admin/routingrules.go" \
  "internal/admin/modellimits_test.go:internal/admin/modellimits.go" \
  "internal/admin/combos_admin_test.go:internal/admin/combos_admin.go" \
  "internal/admin/quota_test.go:internal/admin/quota.go" ; do
  tf=${pair%%:*}; im=${pair##*:}
  ct=$(git log --format=%ct --diff-filter=A -1 -- "$tf")
  cf=$(git log --format=%ct --diff-filter=A -1 -- "$im")
  [ "$ct" -le "$cf" ] || echo "TDD VIOLATION: $im"     # prints nothing
done
```

**Grep proofs**
```bash
grep -nE "func \(h \*Handlers\) (List|Create|Get|Update|Delete)Alias" internal/admin/aliases.go
grep -nE "func \(h \*Handlers\) (List|Create|Get|Update|Delete)RoutingRule" internal/admin/routingrules.go
grep -nE "func \(h \*Handlers\) (List|Create|Get|Update|Delete)ModelLimit" internal/admin/modellimits.go
grep -nF "strconv.ParseInt" internal/admin/modellimits.go            # numeric id (ESC-IDTYPE)
grep -nE "func \(h \*Handlers\) (List|Create|Get|Update|Delete)ComboAdmin" internal/admin/combos_admin.go
grep -nF "GetQuota" internal/admin/quota.go
grep -nF "writeData" "writeError" "recordAudit" internal/admin/aliases.go
grep -nE '/api/aliases|/api/routing-rules|/api/model-limits|/api/quota|/api/combos' internal/server/routes_admin.go
# combos engine store/handler UNTOUCHED:
git diff <base>..HEAD -- internal/admin/combos.go internal/store/combos.go internal/store/aliases.go | wc -l   # = 0
# no init():
! grep -rn "func init(" internal/admin/aliases.go internal/admin/routingrules.go internal/admin/modellimits.go internal/admin/combos_admin.go internal/admin/quota.go internal/store/aliasesadmin.go internal/store/routingrules.go internal/store/modellimits.go internal/store/combosadmin.go && echo "no init() OK"
```

**No-secret-exposure proofs (binding)**
```bash
# quota/usage responses carry no tokens. The quotaDTO has no secret field:
grep -nA12 'type quotaDTO struct' internal/admin/quota.go ; echo "^ must NOT contain access_token/secret/password"
! grep -niE 'access_token|refresh_token|secret|password' internal/admin/quota.go | grep -iE 'json:"' && echo "no secret json field in quota OK"
# additive migrations only:
git diff <base>..HEAD -- internal/store/migrate.go | grep -E '^\+' | grep -iE 'DROP COLUMN|RENAME COLUMN|DROP TABLE' | wc -l   # = 0
```
Plus a runtime no-leak assertion in `quota_test.go`: marshal every `/api/quota`
response and assert it contains no `access_token`/`refresh_token`/secret substring
from the fake connections.

**Negative / freeze proofs (w7-route-a commit-range — §7)**
```bash
R="<first-w7-route-a>^..<last-w7-route-a>"
# Only the sanctioned Go files changed:
git diff $R --name-only -- internal/ | grep -vE \
 'internal/store/(aliasesadmin|routingrules|modellimits|combosadmin|migrate)(_test)?\.go|internal/admin/(aliases|routingrules|modellimits|combos_admin|quota)(_test)?\.go|internal/server/routes_admin\.go' \
 | wc -l                                                                  # = 0
# Frozen combos engine + gateway alias resolver untouched:
git diff $R --name-only -- internal/admin/combos.go internal/store/combos.go internal/store/aliases.go | wc -l   # = 0
# Dynamic-routing engine (w7-route-b territory) untouched:
git diff $R --name-only -- internal/inference/ internal/server/routes_openai.go internal/providers/ | wc -l   # = 0
# Other frozen admin handlers untouched:
git diff $R --name-only -- internal/admin/auth.go internal/admin/apikeys.go internal/admin/virtualkeys.go internal/admin/providers.go internal/admin/connections.go internal/admin/connectionusage.go | wc -l   # = 0
# UI frozen except sanctioned mock/seed bodies:
git diff $R --name-only -- ui/ | grep -vE \
 'ui/e2e/mocks/handlers/(aliases|routing-rules|model-limits|combos|quota)\.ts|ui/e2e/mocks/seed/(aliases|routing-rules|model-limits|combos|usage)\.ts' | wc -l   # = 0
git diff $R --name-only -- ui/src/ | wc -l                               # = 0 (src frozen)
git diff $R --name-only -- ui/dist/ | wc -l                              # = 0 (dist never staged)
# routes_admin.go = exactly ONE commit:
git log --oneline $R -- internal/server/routes_admin.go | wc -l          # = 1
```

---

## 6. Out of scope (restated, binding)

No UI src edits (decision 8 — pages/components/routes/stores frozen); only the
sanctioned five mock-body + seed corrections. No dynamic-routing-engine edits
(selection.go/factory.go/routes_openai.go/providers = w7-route-b). No edits to the
existing combos engine bodies or the gateway alias resolver (ESC-COMBOS only
re-registers `/api/combos` HTTP routes; ESC-ALIAS-SHAPE uses a new table). No live
routing-rule application (admin CRUD only; follow-up tracked). No destructive DDL —
additive `ensureTable`/`ensureColumn` only. No `New(...)` signature change / no new
global state. No secret exposure (quota carries no tokens). Mock-vs-Go contradiction →
escalate (§8), never fudge a mock or edit a frozen page/handler. NEVER revert
`ui/dist/index.html`; NEVER run concurrent playwright; `npm run build` before e2e.

## 7. Diff-gate scope

W7 plans commit to main concurrently, so a broad `<base>..HEAD` range sweeps in
sibling commits. The diff gate MUST be scoped to w7-route-a's own commits:
`git log --oneline main | grep "^[0-9a-f]* phase-1/w7-route-a:" | awk '{print $1}'`
then `git diff <first-w7-route-a>^..<last-w7-route-a> -- [file list]`.

`git diff <first>^..<last> --name-only` must be exactly a subset of:
```
internal/store/aliasesadmin.go (+_test)
internal/store/routingrules.go (+_test)
internal/store/modellimits.go (+_test)
internal/store/combosadmin.go (+_test)
internal/store/migrate.go                (additive tables; ONE concern)
internal/admin/aliases.go (+_test)
internal/admin/routingrules.go (+_test)
internal/admin/modellimits.go (+_test)
internal/admin/combos_admin.go (+_test)
internal/admin/quota.go (+_test)
internal/server/routes_admin.go          (serial-slot routes + ESC-COMBOS re-register; ONE commit)
ui/e2e/mocks/handlers/{aliases,routing-rules,model-limits,combos,quota}.ts   (body only)
ui/e2e/mocks/seed/{aliases,routing-rules,model-limits,combos,usage}.ts        (verify; correct on divergence)
.planning/parity/matrix/9router-ui.md
.planning/parity/plans/open-questions.md
docs/WORKFLOW.md
```
Any file outside this list is an automatic review REJECT. The existing combos engine,
the gateway alias resolver, all of `internal/inference/*` + `routes_openai.go` +
`internal/providers/*` (w7-route-b), and all `ui/src/**` are deliberately ABSENT —
touching them is an automatic REJECT. `routes_admin.go` = exactly ONE commit; serial
slot released to w7-gov-1 on close.

## 8. Escalations / decisions (explicit — recommended defaults, do not fabricate)

- **ESC-COMBOS (RESOLVED at authoring — combos DTO reconciliation, the highest-risk
  decision; awaiting orchestrator path confirmation).** Real Go combo engine is
  `{name,models[]}` keyed by `name` (`combos.go:15-21`, `routes_admin.go:194`);
  frozen page needs id-keyed `{id,name,strategy,steps[{provider,model}],is_active}`
  with a PUT-by-id `{steps}` body (`combos.spec.ts:53-66`) — IRRECONCILABLE by
  additive read-DTO enrichment. **Decision: a NEW separate `combos_admin` table +
  handlers serving the UI shape (§1.7), Option A** — the admin surface OWNS
  `/api/combos[/{id}]` (re-registering the engine's HTTP routes after verifying no
  non-page consumer; the engine's store + `/v1/models` lister stay intact, fed by a
  best-effort mirror-write). Option B (distinct path) needs a frozen-page edit →
  not viable without a separate serial follow-up. RECOMMENDED: Option A. **The
  orchestrator MUST confirm the engine's `/api/combos` HTTP routes have no other
  consumer (P1 grep) before T-routes.** Do NOT silently reshape the engine.
- **ESC-ALIAS-SHAPE (RESOLVED at authoring — alias admin shape, binding default).**
  UI `Alias{id,alias,provider,model}` (id-keyed) vs the frozen gateway
  `ModelAlias{name,target}`. **Decision: a NEW typed `aliases` admin table** mirroring
  the UI shape; the gateway `model_aliases` resolver stays frozen; admin edits
  best-effort mirror-write a resolution alias. Zero-new-table alternative (drive
  `model_aliases` directly, derive id/provider/model) loses the distinct
  provider+model fields. RECOMMENDED: new table. Flag.
- **ESC-IDTYPE (RESOLVED at authoring — model_limits INTEGER PK, binding default).**
  UI `ModelLimit.id` is a **number** (`types.ts:170`; seed ids `1`,`2`); the mock
  POST uses `Date.now()`. **Decision: `model_limits` uses `INTEGER PRIMARY KEY
  AUTOINCREMENT`; handlers parse `{id}` via `strconv.ParseInt` (NOT string `pathID`)**
  — the exact divergence + remedy w7-gov-2 applied to feature_flags/prompt_templates.
  routing_rules + aliases keep TEXT-PK `newID()` (their seed ids are strings:
  `rule-1`, `alias-1`). RECOMMENDED as stated; flag.
- **ESC-QUOTA-SCOPE (RESOLVED at authoring — which connections /api/quota lists,
  binding default).** The mock seeds 4 quota cards (active connections). **Decision:**
  `/api/quota` aggregates ALL connections via `ListConnections`, mapping per-connection
  usage through the injectable fetcher; oauth connections get real used/limit, non-oauth
  contribute `used:0,limit:0` (or are omitted — VERIFY which the page renders at
  T-mocks; default: include all, harmless extra cards). The per-provider usage→
  `{used,limit,unit,plan}` normalization is best-effort (missing fields default).
  RECOMMENDED; flag if the page asserts exactly-4 cards (it does not — the spec only
  asserts `.first()` visible).
- **ESC-CREATED-AT (CONDITIONAL — created_at representation).** The mock seeds
  `created_at` as an ISO string; the store keeps int64 unix. **Decision:** store
  int64, render RFC3339 ISO in the DTO so the mock mirror is a string. Verify the
  page does not date-math the value (it only renders it). Flag if a spec asserts a
  specific format.
- **ESC-NAME (CONDITIONAL — handler/method name collisions in package admin).**
  `ListAliases`/`DeleteAlias` etc. must not clash with any existing `admin` package
  symbol (the gateway alias logic is in `store`, so a clash is unlikely). If a clash
  exists, suffix the admin handlers (`ListAliasAdmin`, …). Decide at T-aliases.
- **ESC-ARCH (CONDITIONAL — arch test on the new CRUD domains).** No standalone
  `arch_test.go` found at authoring; the precedent is handler→store direct for CRUD.
  Default: handler→store directly. Add a domain wrapper ONLY if `go test ./...`
  surfaces an arch-test failure at a task. Do NOT pre-build wrappers.
- **ESC-ROUTE (CONDITIONAL — fasthttp/router precedence).** Static collection vs
  `{id}`, and the combos `{name}`→`{id}` rename (ESC-COMBOS). If the matcher
  mis-disambiguates or panics, STOP and ESCALATE for a path arrangement — never
  silently diverge page/mock/Go.
- **ESC-MOCK (CONDITIONAL — mock ripple).** The five handlers are w6-h/w6-g-owned and
  domain-specific (not shared). `seed/usage.ts` (quota lives there) also backs the
  usage specs — a `seedQuota` change could ripple; default: do NOT change `seedQuota`
  (only verify). If a correction reds a non-w7-route-a spec, STOP and ESCALATE.
- **Live routing-rule application (NEW follow-up).** w7-route-a ships routing_rules
  admin CRUD only; applying rules to live inference traffic is a separate, larger
  concern (record in `open-questions.md`).
- **Serial-slot dependency (§1.9 / P5).** w7-route-a TAKES the routes_admin.go slot
  after w7-platnodes releases it (chain MAP §219-224) and RELEASES it to w7-gov-1 on
  close. Orchestrator confirms exactly one unmerged holder (decision 3) before
  T-routes. w7-route-a depends on w7-platnodes (merged).
```
