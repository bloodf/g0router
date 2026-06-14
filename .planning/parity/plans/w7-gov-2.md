# Micro-plan w7-gov-2 — Governance backends B: feature-flags + prompt-templates (Go)

```
wave: 7
plan: w7-gov-2
status: READY (rev 1 — authored against merged Waves 0–6 + w7-gov-1 (shipped,
  gate-green), live tree @ <base>; WAVE-7-MAP w7-gov-2 row ~line 172; serial chain
  §219-224; reconciliation decision 1 §36/§245; freeze rules §267)
runs: governance+routing track. Disjoint domain/store/admin files from w7-gov-1
  (merged) and w7-gov-3 (run ∥). TAKES the internal/server/routes_admin.go SERIAL
  SLOT after w7-gov-1 RELEASES it (chain: w7-platnodes → w7-route → w7-gov-1 →
  **w7-gov-2** → w7-gov-3 → w7-mcp-3 → w7-plat-1 → w7-plat-2 → w7-plat-3 → w7-misc;
  MAP §219-224).
branch: main (direct commits, per repo convention)
commit-prefix: phase-1/w7-gov-2:
ref-source: 9router frozen @ 827e5c3 — governance feature-flags + prompt-templates
  surfaces; the BINDING contract for W7 is the W6 e2e mock (decision 1: real Go wins,
  mock corrected to mirror it). Mock sources:
    ui/e2e/mocks/handlers/{feature-flags,prompts}.ts + seed/{feature-flags,prompts}.ts.
base: <base> = git rev-parse HEAD recorded at P0. Substitute the actual SHA
  everywhere §5 says <base>.
go-serial-slot: this plan holds the SINGLE in-flight edit to
  internal/server/routes_admin.go while live (W3/W4/W5/W6/w7-gov-1 lesson; MAP
  decision 3). Slot must be FREE at P-check (w7-gov-1 merged + slot released) before
  T-routes. RELEASE to w7-gov-3 on close.
new-route: NO UI route files. Both UI pages (/feature-flags, /prompts) ALREADY
  SHIPPED in w6-k against mocks; this plan builds the REAL Go so the pages flip
  mock→true-HAVE and corrects the mock bodies to mirror the Go DTOs.
```

---

## 1. Scope — PAR rows + the two domains

### Rows this plan closes

| Row / item | Claim | Target state after w7-gov-2 |
|---|---|---|
| open-questions w6-k **ESC-1c** (feature-flags backend absent) | real `GET /api/feature-flags` (list) + `PUT /api/feature-flags/{id}` (toggle `enabled`); NO POST/DELETE | true-HAVE (Go — NEW `internal/store/featureflags.go` + `internal/admin/featureflags.go`, §1.4) |
| open-questions w6-k **ESC-1e** (prompt-templates backend absent) | real `GET/POST /api/prompt-templates` + `GET/PUT/DELETE /api/prompt-templates/{id}` + `POST /api/prompt-templates/test` | true-HAVE (Go — NEW `internal/store/prompttemplates.go` + `internal/admin/prompttemplates.go`, §1.5) |

Matrix flips at closeout (§4 T-close): in `.planning/parity/matrix/9router-ui.md`, the
feature-flags + prompts governance rows (PAR-UI-130/131 governance cluster, mirroring
how w7-gov-1 flipped teams/audit) → mock→true-HAVE with a cite to this plan. Mark
`open-questions.md` w6-k ESC-1c/1e RESOLVED.

### 1.1 Preconditions already satisfied by merged waves (evidence)

- **W6-k UI is SHIPPED and FROZEN (consume-only, MAP decision 8 / §267).** The
  `/feature-flags` page (`ui/src/routes/feature-flags.tsx`) and `/prompts` page
  (`ui/src/routes/prompts.tsx` + `ui/src/components/governance/prompt-form-modal.tsx`)
  render against the registered mocks. The binding acceptance contracts are the
  existing specs (must stay green at closeout):
  - `ui/e2e/feature-flags.spec.ts` — 3 tests: page loads ("Feature Flags",
    `feature-flags.spec.ts:11`); `[data-testid="feature-flag-row"]` ≥3 from seed +
    renders "mcp_gateway" + "Enable MCP gateway" (`:18-20`); toggling a row's
    `button[role="switch"]` fires a `PUT` matching `/\/api\/feature-flags\/\d+$/`
    (**NUMERIC id in the URL** — `:29`). NO create/delete in the spec.
  - `ui/e2e/prompts.spec.ts` — 4 tests: page loads ("Prompts", `:11`);
    `[data-testid="prompt-row"]` ≥2 from seed + renders "Code Review" + "gpt-4o"
    (`:18-20`); create via `[data-testid="prompt-new"]` modal (traffic-lights) +
    `#prompt-name` + `[data-testid="prompt-save"]` fires `POST /api/prompt-templates`
    (`:30-34`); delete via `[data-testid="prompt-delete"]` + confirm dialog
    ("Delete prompt") decrements the row count (`:42-46`).
- **w7-gov-1 audit-write seam is IN-TREE and reusable (the de-risk).** w7-gov-1
  added `Handlers.audit *governance.AuditService` constructed in `New`
  (`internal/admin/handlers.go:20,50`), the `auditService()` accessor (`:55-56`),
  and the best-effort write helper `func (h *Handlers) recordAudit(ctx, action,
  target, details string)` (`internal/admin/audit.go:64-72` — resolves the actor
  from `ctx.UserValue(userKey).(*store.User)`, logs + swallows write errors).
  **REUSE `h.recordAudit` on this plan's mutations** (flag toggle, prompt
  create/update/delete) — best-effort, post-success, NEVER fails the parent
  (mirrors w7-gov-1 ESC-AUDIT-WRITE). NO new audit code; NO edit to audit.go.
- **CRUD templates EXIST.** Store CRUD template = `internal/store/teams.go`
  (`Team` struct + `CreateTeam`/`ListTeams`/`GetTeamByID`/`UpdateTeam`/`DeleteTeam`
  + `scanTeam`, `teams.go:11-60`; `time.Now().Unix()` timestamps, `ErrNotFound`,
  unix-int ts). Simple-table template = `internal/store/auditlog.go`. Transport CRUD
  template = `internal/admin/teams.go` (DTO + request structs + validate +
  `writeData`/`writeError` + `pathID`). The brief's `internal/store/kv.go`
  flag-store option is NOT used (a dedicated `feature_flags` table is cleaner and
  mirrors the mock shape; §8 ESC-FF-STORE).
- **Envelope + handler patterns** (`internal/admin/respond.go`): `writeData(ctx,
  status, data)` / `writeError(ctx, status, message)` → `{data, error:{message}}`
  snake_case (`respond.go:9-12,19-23`). `pathID(ctx.UserValue("id"))` extracts
  `{id}` (`handlers.go:84`).
- **Migrations are additive-only** (`internal/store/migrate.go`): new tables via the
  `tables []struct{name,create}` slice with `CREATE TABLE IF NOT EXISTS` (the
  w7-gov-1 `teams`/`audit_log` additions are at `migrate.go:140,150`; index at
  `:174`); the `ensureColumn` additive-column loop is at `:209`. ADDITIVE ONLY
  (decision 2).
- **Admin test harness** (`internal/admin/admin_test.go` `newTestEnv` + `call` +
  `dataField[T]` + `errMessage` + `loginToken`): real `store.Open(tempDB, secret)` +
  `auth.NewSessions` + `SeedAdmin` + `New(...)`. NO mocks. This is the authoritative
  proof surface (mirrors w7-gov-1 §1.1).
- **Handlers injection** (`internal/admin/handlers.go`): the `Handlers` struct holds
  `store`/`sessions`/`audit`/… ; new domains compose `h.store` directly (like
  teams/virtualkeys) — NO new global state, NO `New(...)` signature change (MAP
  decision 9).

### 1.2 The mock contracts these flips must mirror (binding — decision 1)

**Decision 1 (MAP §36, §245):** real Go wins; the W6 mock body + seed are corrected
IN THIS PLAN to mirror the real Go `{data,error}` snake_case DTO. The page is FROZEN
(decision 8); where the real DTO and the mock disagree, **prefer matching the mock's
existing field names in the Go DTO** (modeled to match 9router); only ESCALATE if
impossible.

**Envelope reconciliation (both domains).** The mock `json(route, data)` util wraps
every payload as `{data}` (`ui/e2e/mocks/handlers/utils.ts:3-8`) and the page reads
the list as a bare array via `apiFetch<FeatureFlag[]>("/api/feature-flags")`
(`feature-flags.tsx:25`) / `apiFetch<PromptTemplate[]>("/api/prompt-templates")`
(`prompts.tsx:32`). So the Go list endpoint returns `{data:[...]}` (a bare JSON array
under `data`) — **exactly the teams precedent.** The mock `error(route, message)` util
returns `{error: message}` (a STRING, `utils.ts:11-16`) which DIVERGES from the real
Go `{error:{message}}` (`respond.go:11-12`); the page surfaces errors generically, so
this is a tolerated mock divergence — leave the mock `error()` helper alone (it is
shared across all handlers; editing it is out of scope) and only correct the per-route
SUCCESS bodies (same disposition as w7-gov-1, which did not touch `error()`).

**Feature-flags** (`ui/e2e/mocks/handlers/feature-flags.ts` + `seed/feature-flags.ts`):
- Routes the page consumes: `GET /api/feature-flags` (list, `feature-flags.ts:6-8`)
  and `PUT /api/feature-flags/{id}` (toggle, `:17-23`). The mock ALSO has a
  `GET /api/feature-flags/{id}` single-read branch (`:13-15`) the page never calls.
  **There is NO POST and NO DELETE** (toggle-only surface) — the Go MUST match: build
  ONLY `GET` (list) + `PUT/{id}` (toggle). (Optional: serve `GET /{id}` for parity
  since the mock has it; §8 ESC-FF-GETBYID — RECOMMENDED: include it, cheap, no spec
  rides on it.)
- Mock/seed shape = the UI `FeatureFlag` type (`ui/src/lib/types.ts:95-101`):
  **`{id:number, key, enabled, description, created_at}`** — this is the canonical
  Go DTO. `created_at` is ISO-8601 in the seed (`new Date(...).toISOString()`,
  `seed/feature-flags.ts:5`). **`id` is NUMERIC** (`id:1,2,3` in the seed; the spec
  PUT regex requires `\d+`, `feature-flags.spec.ts:29`) — see §8 ESC-IDTYPE.
- Toggle PUT body = `{enabled}` (`feature-flags.tsx:48`); returns the updated flag.
  The mock spreads `{...existing, ...body}` (`feature-flags.ts:21`) so a PUT can
  carry any field, but the page sends only `{enabled}`. **Reconciliation:** the Go
  `PUT` accepts `{enabled bool}` (and optionally `description`), updates, returns the
  full flag DTO; mock already mirrors. No mock body change needed beyond the §1.2
  envelope note (the mock GET/PUT already return the 5-field shape).

**Prompt-templates** (`ui/e2e/mocks/handlers/prompts.ts` + `seed/prompts.ts`):
- Routes: `GET /api/prompt-templates` (list, `prompts.ts:6-8`), `POST` (create,
  `:9-14`), `GET|PUT|DELETE /api/prompt-templates/{id}` (`:17-36`), `POST
  /api/prompt-templates/test` (`:38-41`).
- **ROUTE-PRECEDENCE divergence (the one real wrinkle).** The mock registers the
  `/{id}` regex `/\/api\/prompt-templates\/[^/]+$/` (`:17`) BEFORE the static
  `/api/prompt-templates/test` (`:38`); in Playwright the `/test` registration is
  later so it can still win for that exact path, but in `fasthttp/router` STATIC must
  be registered BEFORE the `{id}` param OR the param swallows `test`. **Reconciliation:
  register `POST /api/prompt-templates/test` BEFORE `…/{id}` in routes_admin.go**
  (static-before-param, the file's existing convention, `routes_admin.go:52-57`
  teams precedent). §8 ESC-ROUTE if the matcher still mis-disambiguates.
- Mock/seed shape = the UI `PromptTemplate` type (`ui/src/lib/types.ts:187-194`):
  **`{id:number, name, system_prompt, models:[]string, is_active, created_at}`** —
  the canonical Go DTO. `id` is NUMERIC (`id:1,2`, `seed/prompts.ts:5-6`; §8
  ESC-IDTYPE). The mock POST also injects `updated_at` (`prompts.ts:11`) which the UI
  `PromptTemplate` type does NOT carry and no spec asserts — **Reconciliation:** the
  Go DTO surfaces the 6 canonical fields; whether to ALSO surface `updated_at` is §8
  ESC-PROMPT-UPDATEDAT (RECOMMENDED: store `updated_at` in the table for hygiene but
  OMIT it from the DTO to match the UI type; the corrected mock POST/PUT drop the
  `updated_at` field to mirror Go).
- Create body = `{name, system_prompt, models:[]string, is_active}`
  (`prompt-form-modal.tsx:45-51`). Returns the created DTO.
- **`/test` endpoint** (`POST /api/prompt-templates/test`): the mock returns
  `{rendered: "Mock rendered prompt"}` (`prompts.ts:39`). **It is NOT consumed by the
  live UI** (`grep -rn "prompt-templates/test" ui/src/` → ZERO) and **no spec
  exercises it.** It must still be built for backend parity + to keep the mock route
  honest. Response shape is escalation-defaulted: §8 ESC-PROMPT-TEST.

### 1.3 Architecture (binding — layered DDD, decision 4 + the w7-gov-1 ESC-ARCH finding)

w7-gov-1 RESOLVED ESC-ARCH: **no phase-12B arch test exists in-tree** enforcing
transport→domain→repository (grep found none; teams/virtualkeys/apikeys handlers call
`h.store` DIRECTLY with no enforcing test). Follow that precedent:

```
feature-flags:    admin/featureflags.go     → store/featureflags.go (NEW table feature_flags)
                  (handler→store directly; NO governance/featureflags.go — trivial 2-route CRUD)
prompt-templates: admin/prompttemplates.go  → store/prompttemplates.go (NEW table prompt_templates)
                  (handler→store directly by default; a thin governance/prompttemplates.go
                   ONLY if the /test render logic warrants a domain seam — §8 ESC-PROMPT-DOMAIN)
```

- Feature-flags is a trivial GET+PUT surface (no create/delete) — pure handler→store,
  mirroring teams. NO domain file. The MAP's listed `internal/governance/featureflags.go`
  is therefore NOT created by default (deviation documented here; §8 ESC-FF-DOMAIN).
- Prompt-templates is standard CRUD. The `/test` endpoint applies a template to a
  sample — if it stays a trivial echo/substitution (per ESC-PROMPT-TEST default) it
  needs NO domain layer; if it grows real rendering logic a thin
  `governance/prompttemplates.go` `PromptService.Render(template, sample)` is the seam.
  **Default: NO domain file** (handler→store + an inline render helper), matching the
  ESC-ARCH precedent. Re-decide at T-prompts ONLY if `/test` logic is non-trivial.

### 1.4 Feature-flags Go contract (NEW, TDD)

Table `feature_flags` (additive, `migrate.go` tables slice). **INTEGER autoincrement
PK** to mirror the mock's numeric ids (§8 ESC-IDTYPE):
```sql
CREATE TABLE IF NOT EXISTS feature_flags (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  key TEXT NOT NULL UNIQUE,
  enabled INTEGER NOT NULL DEFAULT 0,   -- SQLite bool 0/1
  description TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL              -- ISO-8601 (RFC3339), mirrors mock seed
)
```

`internal/store/featureflags.go` (NEW): `FeatureFlag{ID int64, Key string, Enabled
bool, Description string, CreatedAt string}` + methods:
- `ListFeatureFlags() ([]*FeatureFlag, error)` — `SELECT … ORDER BY id ASC`.
- `GetFeatureFlagByID(id int64) (*FeatureFlag, error)` — `ErrNotFound` on `sql.ErrNoRows`.
- `SetFeatureFlagEnabled(id int64, enabled bool) (*FeatureFlag, error)` — UPDATE then
  re-read; `ErrNotFound` if 0 rows. (And/or a general `UpdateFeatureFlag` if
  `description` is mutable — keep minimal: toggle-only per the mock surface.)
- `scanFeatureFlag` helper; `boolToInt` for the `enabled` column (mirror the
  virtualkeys/teams SQLite-bool precedent).
- **Seeding note:** the live store starts EMPTY (no feature_flags seeded server-side);
  the e2e seed is mock-only. The page renders whatever the API returns. The spec runs
  against the MOCK (which seeds 3 flags), so the real Go empty-list behavior does not
  break the spec. Do NOT seed flags in Go (no requirement; §8 ESC-FF-SEED notes the
  option if a default flag set is later wanted).

`internal/admin/featureflags.go` (NEW):

| Handler | Route | Shape (snake_case, `{data}`) | Notes |
|---|---|---|---|
| `ListFeatureFlags` | `GET /api/feature-flags` | `{data:[flagDTO]}` (bare array under data, mirrors `feature-flags.ts:7`) | `flagDTO{id,key,enabled,description,created_at}` |
| `GetFeatureFlag` (OPTIONAL — §8 ESC-FF-GETBYID) | `GET /api/feature-flags/{id}` | `{data:flagDTO}` or 404 | mock has it (`:13-15`); page never calls it |
| `ToggleFeatureFlag` | `PUT /api/feature-flags/{id}` | body `{enabled:bool}`; returns `{data:flagDTO}`; 404 if missing; 400 on bad id/body | parse `{id}` as int64 (numeric); `recordAudit(ctx,"feature_flag.toggle",key,fmt.Sprintf("set %s enabled=%v",key,enabled))` best-effort |

**NO `POST` and NO `DELETE`** — the surface is GET + PUT-toggle only (match the mock,
brief-mandated). Registering POST/DELETE is a scope violation.

### 1.5 Prompt-templates Go contract (NEW, TDD)

Table `prompt_templates` (additive). **INTEGER autoincrement PK** (§8 ESC-IDTYPE):
```sql
CREATE TABLE IF NOT EXISTS prompt_templates (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  system_prompt TEXT NOT NULL DEFAULT '',
  models_json TEXT NOT NULL DEFAULT '[]',   -- JSON-encoded []string (mirror virtualkeys config_json blob precedent)
  is_active INTEGER NOT NULL DEFAULT 1,      -- SQLite bool
  created_at TEXT NOT NULL,                  -- ISO-8601 (RFC3339)
  updated_at TEXT NOT NULL                   -- stored for hygiene; OMITTED from DTO (§8 ESC-PROMPT-UPDATEDAT)
)
```

`internal/store/prompttemplates.go` (NEW): `PromptTemplate{ID int64, Name string,
SystemPrompt string, Models []string, IsActive bool, CreatedAt string, UpdatedAt
string}` + methods:
- `CreatePromptTemplate(in *PromptTemplate) (*PromptTemplate, error)` — JSON-encode
  `Models` into `models_json`; RFC3339 `created_at`/`updated_at`.
- `ListPromptTemplates() ([]*PromptTemplate, error)` — `ORDER BY id ASC`; JSON-decode
  `models_json`.
- `GetPromptTemplateByID(id int64) (*PromptTemplate, error)` — `ErrNotFound`.
- `UpdatePromptTemplate(id int64, in *PromptTemplate) (*PromptTemplate, error)` —
  update fields; bump `updated_at`; `ErrNotFound`.
- `DeletePromptTemplate(id int64) error` — `ErrNotFound` on 0 rows.
- `scanPromptTemplate` helper; `boolToInt`/`intToBool` for `is_active`;
  `encoding/json` Marshal/Unmarshal for `models`.

`internal/admin/prompttemplates.go` (NEW):

| Handler | Route | Body / response | Notes |
|---|---|---|---|
| `ListPromptTemplates` | `GET /api/prompt-templates` | `{data:[promptDTO]}` (bare array under data) | `promptDTO{id,name,system_prompt,models,is_active,created_at}` (NO `updated_at`, §8 ESC-PROMPT-UPDATEDAT) |
| `CreatePromptTemplate` | `POST /api/prompt-templates` | body `{name,system_prompt?,models?,is_active?}`; 400 on empty name; `is_active` defaults true; returns `{data:promptDTO}` | `recordAudit(ctx,"prompt_template.create",name,…)` |
| `TestPromptTemplate` | `POST /api/prompt-templates/test` | §8 ESC-PROMPT-TEST: default body `{prompt_id?, system_prompt?, sample?}` → returns `{data:{rendered:string}}` mirroring the mock `{rendered}` (`prompts.ts:39`). Default impl: echo/substitute the resolved system_prompt + sample into a rendered string. NOT UI-consumed; no spec. **Register BEFORE `…/{id}`** (§1.2 route-precedence) | no audit (read-only test) |
| `GetPromptTemplate` | `GET /api/prompt-templates/{id}` | `{data:promptDTO}` or 404 | numeric id |
| `UpdatePromptTemplate` | `PUT /api/prompt-templates/{id}` | body = create body; returns updated `{data:promptDTO}` or 404 | `recordAudit(ctx,"prompt_template.update",name,…)` |
| `DeletePromptTemplate` | `DELETE /api/prompt-templates/{id}` | `{data:{message:"Prompt template deleted successfully"}}` or 404 | mock returns `{}` (`prompts.ts:34`); page decrements on success, ignores body. `recordAudit(ctx,"prompt_template.delete",id,…)` |

### 1.6 routes_admin.go registration (serial-slot additive, §3)

Append AFTER the w7-gov-1 audit block (`routes_admin.go:60`), static-before-`{id}`:
```go
// Feature flags (GET list + PUT toggle only — no create/delete).
r.GET("/api/feature-flags", h.RequireSession(h.ListFeatureFlags))
r.GET("/api/feature-flags/{id}", h.RequireSession(h.GetFeatureFlag))   // OPTIONAL, §8 ESC-FF-GETBYID
r.PUT("/api/feature-flags/{id}", h.RequireSession(h.ToggleFeatureFlag))
// Prompt templates CRUD (+ test). STATIC /test registered BEFORE {id} (§1.2).
r.GET("/api/prompt-templates", h.RequireSession(h.ListPromptTemplates))
r.POST("/api/prompt-templates", h.RequireSession(h.CreatePromptTemplate))
r.POST("/api/prompt-templates/test", h.RequireSession(h.TestPromptTemplate))
r.GET("/api/prompt-templates/{id}", h.RequireSession(h.GetPromptTemplate))
r.PUT("/api/prompt-templates/{id}", h.RequireSession(h.UpdatePromptTemplate))
r.DELETE("/api/prompt-templates/{id}", h.RequireSession(h.DeletePromptTemplate))
```
Route-precedence note: `/api/prompt-templates/test` (static) vs `…/{id}` — register
static FIRST. A genuine `fasthttp/router` collision is §8 ESC-ROUTE, not a silent path
change. Diff bound §5: the route block is ONE commit, additive only.

### NOT in scope (explicit)

- **No UI page/component/route/store edits** — `/feature-flags`, `/prompts`, the
  prompt-form-modal, and all w6-k components are FROZEN consume-only (decision 8). The
  ONLY UI-tree touches are the mock-body + seed corrections (§1.2 / §3).
- **No POST/DELETE for feature-flags** — toggle-only surface (brief-mandated; mock has
  neither).
- **No edits to other governance domains** — teams/audit/user-management (w7-gov-1,
  merged), guardrails/alerts (w7-gov-3) are disjoint; do not touch their Go/mocks/seeds.
- **No edits to pre-existing admin handlers' bodies** — teams.go, audit.go,
  usermgmt.go, apikeys.go, virtualkeys.go, providers*.go, connections.go, combos.go,
  disabledmodels.go, auth.go, version.go, usage/pricing handlers are FORBIDDEN.
- **No edit to `internal/admin/audit.go` / `internal/governance/audit.go`** — REUSE
  `h.recordAudit` only (read-only consumption of the w7-gov-1 seam).
- **No edit to the shared mock `error()` / `json()` utils** (`utils.ts`) or the mock
  index / seed-index / `store.ts` / `fixture.ts` (§1.2 envelope note).
- **No destructive DDL / column renames** — additive `ensureTable`/`ensureColumn` ONLY
  (decision 2).
- **No new global state / no `New(...)` signature change** (decision 9) — handlers
  compose `h.store` + reuse `h.recordAudit`.
- **No secret exposure** — neither domain carries secrets; audit `details` carry no
  secrets (human-readable summaries only, §5 grep proof).

---

## 2. Precondition checks

Run all before any edit; abort and report to orchestrator on any failure.

```bash
# P0 — clean tree; record <base>
git status --porcelain     # must be empty (explicit `git add <file>`, never -A;
                           # ui/dist/** gitignored — never stage it)
git rev-parse HEAD         # record as <base> for §5

# P1 — the two gaps are REAL (no Go for either domain)
grep -nE '/api/feature-flags|/api/prompt-templates' internal/server/routes_admin.go ; echo "^ expect EMPTY"
test ! -e internal/store/featureflags.go && test ! -e internal/store/prompttemplates.go && echo "store gap OK"
test ! -e internal/admin/featureflags.go && test ! -e internal/admin/prompttemplates.go && echo "admin gap OK"

# P2 — w7-gov-1 seams present (the de-risk) + reused surfaces
grep -n "func (h \*Handlers) recordAudit" internal/admin/audit.go
grep -n "audit *\*governance.AuditService\|auditService()" internal/admin/handlers.go
grep -n "func writeData\|func writeError" internal/admin/respond.go
grep -n "func pathID" internal/admin/handlers.go
grep -n "func newTestEnv\|func call\|func dataField\|func loginToken" internal/admin/admin_test.go
grep -n "func (s \*Store) CreateTeam\|ListTeams\|GetTeamByID\|UpdateTeam\|DeleteTeam" internal/store/teams.go   # CRUD template

# P3 — migrate pattern (additive tables slice + index + ensureColumn)
grep -n '"teams"\|"audit_log"\|CREATE TABLE IF NOT EXISTS\|func ensureColumn\|CREATE INDEX IF NOT EXISTS' internal/store/migrate.go | head

# P4 — the W6-k UI + specs are present (consume-only) and the mocks to correct
test -f ui/e2e/feature-flags.spec.ts && test -f ui/e2e/prompts.spec.ts && echo "specs present"
test -f ui/e2e/mocks/handlers/feature-flags.ts && test -f ui/e2e/mocks/handlers/prompts.ts && echo "mocks present"
test -f ui/e2e/mocks/seed/feature-flags.ts && test -f ui/e2e/mocks/seed/prompts.ts && echo "seeds present"
grep -n "id:" ui/e2e/mocks/seed/feature-flags.ts ui/e2e/mocks/seed/prompts.ts ; echo "^ NUMERIC ids (ESC-IDTYPE)"
grep -n "updated_at" ui/e2e/mocks/handlers/prompts.ts ; echo "^ mock POST injects updated_at to drop (§1.2)"
grep -n "/api/prompt-templates/test" ui/e2e/mocks/handlers/prompts.ts ; echo "^ /test route (route-precedence §1.2)"

# P5 — routes_admin.go serial slot is FREE (w7-gov-1 merged + released)
git log --oneline -5 -- internal/server/routes_admin.go   # last touch = w7-gov-1 (merged)
# Orchestrator MUST confirm no concurrent W7 plan holds an unmerged routes_admin.go
# edit before w7-gov-2 begins T-routes (chain: …→w7-gov-1→**w7-gov-2**→w7-gov-3).
# w7-gov-2 TAKES the slot, then RELEASES it to w7-gov-3 on close.

# P6 — green at base
go test ./... && go vet ./... && go build ./...     # exit 0 (Go untouched-green)
# e2e ISOLATED — kill stale chromium/vite-preview first (e2e-hygiene rule); never
# run concurrently with another playwright invocation.
pkill -f 'chromium|vite preview' 2>/dev/null ; true
cd ui && npx playwright test e2e/feature-flags.spec.ts e2e/prompts.spec.ts
# Record base: these PASS at base against the W6 mocks. They must STAY green after
# the mock-body corrections. Record exact pass/fail in WORKFLOW.md.
cd ui && npm run build                               # exit 0
```

---

## 3. Exclusive file ownership

After w7-gov-2 merges, all CREATE files are owned by w7-gov-2; later plans consume,
never edit (MAP decision 7).

**CREATE — store (NEW):**

| File | Contract |
|---|---|
| `internal/store/featureflags.go` | `FeatureFlag` struct (INT64 id) + `ListFeatureFlags`/`GetFeatureFlagByID`/`SetFeatureFlagEnabled` + `scanFeatureFlag`; `boolToInt`. `ErrNotFound`. |
| `internal/store/featureflags_test.go` | Table-driven via temp `store.Open`: list(empty→0); after a direct insert, list returns it; get→404 missing; toggle persists `enabled`; toggle missing→404. RED first. |
| `internal/store/prompttemplates.go` | `PromptTemplate` struct (INT64 id, `[]string` models) + `CreatePromptTemplate`/`ListPromptTemplates`/`GetPromptTemplateByID`/`UpdatePromptTemplate`/`DeletePromptTemplate` + `scanPromptTemplate`; JSON models blob; `boolToInt`. |
| `internal/store/prompttemplates_test.go` | create→get→list→update→delete→404; models round-trip via JSON; is_active default true. RED first. |

**EXTEND — store (additive table registration only):**

| File | Change (additive ONLY) |
|---|---|
| `internal/store/migrate.go` | ADD the `feature_flags` + `prompt_templates` tables to the `tables` slice (after the w7-gov-1 `audit_log` entry). ADDITIVE ONLY — no DROP/RENAME. (Optional index on `feature_flags(key)` if a uniqueness lookup is wanted; the table already declares `key UNIQUE`.) |
| `internal/store/migrate_test.go` (if present — EXTEND additively; else rely on store tests) | assert the two new tables exist post-migrate. |

**CREATE — transport (NEW):**

| File | Contract |
|---|---|
| `internal/admin/featureflags.go` | `ListFeatureFlags`/`GetFeatureFlag`(optional)/`ToggleFeatureFlag` + `flagDTO` + toggle request/validate; `writeData`/`writeError`; numeric `{id}` parse; `h.recordAudit` on toggle (best-effort). |
| `internal/admin/featureflags_test.go` | via `newTestEnv`: list (after seeding a flag through the store) ≥1; toggle flips `enabled` + `GetAudit` shows an audit entry; toggle missing id→404; bad id→400. **No POST/DELETE handler exists** (assert absence is implicit). RED first. |
| `internal/admin/prompttemplates.go` | `ListPromptTemplates`/`CreatePromptTemplate`/`TestPromptTemplate`/`GetPromptTemplate`/`UpdatePromptTemplate`/`DeletePromptTemplate` + `promptDTO` (no `updated_at`) + request/validate; `h.recordAudit` on create/update/delete. |
| `internal/admin/prompttemplates_test.go` | via `newTestEnv`: create→list(≥1)→get→update→delete→404; create empty-name→400; models round-trip; `/test` returns `{rendered}` (default shape); **assert an audit entry on create** (`GetAudit`). RED first. |

**MODIFY — serial-slot route registration (additive only):**

| File | Change |
|---|---|
| `internal/server/routes_admin.go` | ADD the 8–9 route lines (§1.6). NOTHING else changes. ONE commit. SERIAL SLOT — only holder while live; RELEASE to w7-gov-3 on close. |

**MODIFY — e2e mock corrections (mirror real Go, decision 1):**

| File | Change |
|---|---|
| `ui/e2e/mocks/handlers/feature-flags.ts` (BODY) | Verify the GET/PUT branches return the 5-field flag shape under `{data}` (they already do via `json()`). Likely NO change beyond confirming no extra fields leak. Keep GET/{id} + GET + PUT; no POST/DELETE added. |
| `ui/e2e/mocks/handlers/prompts.ts` (BODY) | POST/PUT: DROP the injected `updated_at` field (mirror the Go DTO which omits it, §1.2); keep GET/POST/PUT/DELETE/test. Confirm `/test` returns `{rendered}` (matches Go default). DELETE returns `{}` (page ignores body; Go returns `{message}` — page-tolerated). |
| `ui/e2e/mocks/seed/feature-flags.ts` (BODY) | Already the 5-field `{id:number,key,enabled,description,created_at}` shape — verify; correct only if a field name diverges (none expected). |
| `ui/e2e/mocks/seed/prompts.ts` (BODY) | Already `{id:number,name,system_prompt,models,is_active,created_at}` — verify; no change expected. |

**FORBIDDEN:** everything else. Explicitly: all pre-existing `internal/admin/*.go`
(including teams.go, audit.go, usermgmt.go, handlers.go — NO edit; reuse `h.recordAudit`
read-only); all other `internal/store/*.go` except featureflags/prompttemplates (NEW) +
migrate (additive table registration only); `internal/governance/*` (no new file by
default — §8 ESC-FF-DOMAIN / ESC-PROMPT-DOMAIN; do NOT touch `governance/audit.go`);
all UI `ui/src/**` (FROZEN, decision 8); the shared mock `utils.ts`/index/seed-index/
`store.ts`/`fixture.ts`; all other mocks/seeds/specs; `ui/package.json` + lockfile;
`ui/vite.config.ts`; `ui/playwright.config.ts`. Touching any of these is an automatic
REJECT.

---

## 4. TDD tasks

Cadence (strict, AGENTS.md "TDD always"): **no Go impl file may exist before its
`_test.go` is committed RED.** `go test ./... && go vet ./... && go build ./...` green
at EVERY commit (RED test commits fail only the new package's targeted run; prefer
table tests that fail on assertion, not compile). The two e2e specs stay green
throughout (real Go is additive; mock corrections mirror it). The two domains are
independent — order is feature-flags → prompt-templates, then the single serial-slot
routes commit, then mock corrections + closeout.

### T-ff — STEP(a) RED store+admin tests, STEP(b) impl
STEP(a): write `internal/store/featureflags_test.go` + `internal/admin/featureflags_test.go`
(table-driven, `newTestEnv`); add the `feature_flags` table to `migrate.go` (so tests
compile + the table exists). `go test ./internal/store/ -run FeatureFlag` and
`go test ./internal/admin/ -run FeatureFlag` → FAIL. Commit RED:
`phase-1/w7-gov-2: failing feature-flags store+admin tests (TDD red)`.
STEP(b): implement `internal/store/featureflags.go` + `internal/admin/featureflags.go`
(GET list + GET/{id} optional + PUT toggle; reuse `h.recordAudit`). Gates:
`go test ./... && go vet ./... && go build ./...` green. Commit:
`phase-1/w7-gov-2: feature-flags store + admin (list + toggle)`.

### T-prompts — STEP(a) RED, STEP(b) impl
STEP(a): write `internal/store/prompttemplates_test.go` + `internal/admin/prompttemplates_test.go`;
add the `prompt_templates` table to `migrate.go`. Run targeted tests → FAIL. Commit RED:
`phase-1/w7-gov-2: failing prompt-templates store+admin tests (TDD red)`.
STEP(b): implement `internal/store/prompttemplates.go` + `internal/admin/prompttemplates.go`
(CRUD + `/test`; JSON models blob; reuse `h.recordAudit`). Gates green. Commit:
`phase-1/w7-gov-2: prompt-templates store + admin CRUD + test endpoint`.

### T-routes — serial-slot route registration
TAKE the serial slot (orchestrator confirms FREE at P5). Add the 8–9 route lines to
`routes_admin.go` (§1.6; static `/test` BEFORE `…/{id}`). Gates:
`go test ./... && go vet ./... && go build ./...` green. Commit (ONE commit touches the
serial file):
`phase-1/w7-gov-2: register feature-flags + prompt-templates admin routes (serial slot)`.

### T-mocks — mock-body corrections (mirror real Go, decision 1)
Correct `prompts.ts` (drop `updated_at` from POST/PUT); verify `feature-flags.ts` +
both seeds (no change expected). Gates: `cd ui && npm run build` green; isolated
`npx playwright test e2e/feature-flags.spec.ts e2e/prompts.spec.ts` green (still). If a
correction reds a non-w7-gov-2 spec, STOP + ESCALATE (§8 ESC-MOCK). Commit:
`phase-1/w7-gov-2: correct feature-flags/prompts mocks to mirror real Go DTOs`.

### T-close — full gates + closeout
```bash
go test ./... && go vet ./... && go build ./...
go test ./internal/admin/ -run 'FeatureFlag|Prompt' -v
go test ./internal/store/ -run 'FeatureFlag|Prompt' -v
cd ui && npm run build
pkill -f 'chromium|vite preview' 2>/dev/null ; true
cd ui && npx playwright test e2e/feature-flags.spec.ts e2e/prompts.spec.ts   # green
pkill -f 'chromium|vite preview' 2>/dev/null ; true
cd ui && npx playwright test                                                 # full suite green (no regressions)
cd ui && npx vitest run src/                                                 # unaffected, green
```
Flip `.planning/parity/matrix/9router-ui.md`: the feature-flags + prompts governance
rows → mock→true-HAVE (real Go, cite §1.4/§1.5). Mark `open-questions.md` w6-k
ESC-1c/1e RESOLVED with a cite to this plan; append any new open items (§8). Update
`docs/WORKFLOW.md` (P6 base observation, the ESC-IDTYPE int-PK decision, the
ESC-PROMPT-TEST `/test` shape decision, the serial-slot take-from-w7-gov-1 /
release-to-w7-gov-3, the mock corrections). Final commit:
`phase-1/w7-gov-2: close — feature-flags/prompts Go; matrix flip; mock mirror`.
**On the close commit, RELEASE the routes_admin.go serial slot to w7-gov-3.**

---

## 5. Binary acceptance criteria

All must hold; each is yes/no. `<base>` = the commit recorded at P0. Diff gate is
**w7-gov-2 commit-range-scoped** (§7).

**Test gates**
- `go test ./... && go vet ./... && go build ./...` → exit 0.
- `go test ./internal/admin/ -run 'FeatureFlag|Prompt' -v` → exit 0, all pass
  (feature-flags: list + toggle + 404 + bad-id ≥4 cases incl audit-on-toggle;
  prompts CRUD ≥6 cases incl audit-on-create + `/test`).
- `go test ./internal/store/ -run 'FeatureFlag|Prompt' -v` → exit 0.
- `cd ui && npx playwright test e2e/feature-flags.spec.ts e2e/prompts.spec.ts` →
  exit 0, all pass (3 feature-flags + 4 prompts), 0 skipped. (Run ISOLATED; kill stale
  chromium/vite-preview first.)
- `cd ui && npx playwright test` → exit 0, no spec green-at-base goes red.
- `cd ui && npm run build` → exit 0. `cd ui && npx vitest run src/` → exit 0.

**TDD-order proof** — each impl file's covering test appears in an earlier-or-equal
commit:
```bash
for pair in \
  "internal/store/featureflags_test.go:internal/store/featureflags.go" \
  "internal/store/prompttemplates_test.go:internal/store/prompttemplates.go" \
  "internal/admin/featureflags_test.go:internal/admin/featureflags.go" \
  "internal/admin/prompttemplates_test.go:internal/admin/prompttemplates.go" ; do
  tf=${pair%%:*}; im=${pair##*:}
  ct=$(git log --format=%ct --diff-filter=A -1 -- "$tf")
  cf=$(git log --format=%ct --diff-filter=A -1 -- "$im")
  [ "$ct" -le "$cf" ] || echo "TDD VIOLATION: $im"     # prints nothing
done
```

**Grep proofs (per domain)**
```bash
# feature-flags
grep -n "func (h \*Handlers) ListFeatureFlags\|ToggleFeatureFlag" internal/admin/featureflags.go
grep -n "key\|enabled\|description\|created_at" internal/admin/featureflags.go        # 5-field DTO
grep -n "func (s \*Store) ListFeatureFlags\|GetFeatureFlagByID\|SetFeatureFlagEnabled" internal/store/featureflags.go
grep -n "writeData\|writeError" internal/admin/featureflags.go                         # {data,error}
grep -n "h.recordAudit" internal/admin/featureflags.go                                 # audit on toggle
! grep -nE "func \(h \*Handlers\) (Create|Delete)FeatureFlag" internal/admin/featureflags.go && echo "no POST/DELETE OK (toggle-only)"
# prompt-templates
grep -n "func (h \*Handlers) ListPromptTemplates\|CreatePromptTemplate\|TestPromptTemplate\|GetPromptTemplate\|UpdatePromptTemplate\|DeletePromptTemplate" internal/admin/prompttemplates.go
grep -n "name\|system_prompt\|models\|is_active\|created_at" internal/admin/prompttemplates.go
grep -n "rendered" internal/admin/prompttemplates.go                                   # /test response
grep -n "func (s \*Store) CreatePromptTemplate\|ListPromptTemplates\|GetPromptTemplateByID\|UpdatePromptTemplate\|DeletePromptTemplate" internal/store/prompttemplates.go
grep -n "h.recordAudit" internal/admin/prompttemplates.go                              # audit on mutations
! grep -n '"updated_at"' internal/admin/prompttemplates.go && echo "promptDTO omits updated_at OK (§1.2)"
# routes (static /test before {id})
grep -nE '/api/feature-flags|/api/prompt-templates' internal/server/routes_admin.go
# numeric id handling (int parse, not pathID-string-only)
grep -nE "ParseInt|strconv" internal/admin/featureflags.go internal/admin/prompttemplates.go ; echo "^ numeric id parse (ESC-IDTYPE)"
# no init(); no new global state
! grep -rn "func init(" internal/admin/featureflags.go internal/admin/prompttemplates.go internal/store/featureflags.go internal/store/prompttemplates.go && echo "no init() OK"
```

**No-secret-exposure proofs (binding)**
```bash
# neither domain carries secret fields → no *_enc columns expected
grep -n "_enc" internal/store/featureflags.go internal/store/prompttemplates.go ; echo "^ expect EMPTY"
# additive migrations only (no DROP/RENAME introduced by this plan)
git diff <base>..HEAD -- internal/store/migrate.go | grep -E '^\+' | grep -iE 'DROP COLUMN|RENAME COLUMN|DROP TABLE' | wc -l   # = 0
# audit details carry no secrets (human-readable summaries only — visual + the
# recordAudit contract at audit.go:60-72; this plan only passes flag keys / template names)
grep -n "h.recordAudit" internal/admin/featureflags.go internal/admin/prompttemplates.go ; echo "^ details = key/name summaries, never raw payloads"
```

**Negative / freeze proofs (w7-gov-2 commit-range — §7)**
```bash
R="<first-w7-gov-2>^..<last-w7-gov-2>"
# Only the sanctioned Go files changed:
git diff $R --name-only -- internal/ | grep -vE \
 'internal/store/(featureflags|prompttemplates|migrate)(_test)?\.go|internal/admin/(featureflags|prompttemplates)(_test)?\.go|internal/server/routes_admin\.go' \
 | wc -l                                                                  # = 0
# Frozen w7-gov-1 + pre-existing admin handlers untouched:
git diff $R --name-only -- internal/admin/teams.go internal/admin/audit.go internal/admin/usermgmt.go internal/admin/handlers.go internal/admin/auth.go internal/admin/apikeys.go internal/admin/virtualkeys.go internal/governance/audit.go | wc -l   # = 0
# Other store files untouched (except migrate additive):
git diff $R --name-only -- internal/store/ | grep -vE 'internal/store/(featureflags|prompttemplates|migrate)(_test)?\.go' | wc -l   # = 0
# UI is frozen except the sanctioned mock/seed bodies:
git diff $R --name-only -- ui/ | grep -vE \
 'ui/e2e/mocks/handlers/(feature-flags|prompts)\.ts|ui/e2e/mocks/seed/(feature-flags|prompts)\.ts' | wc -l   # = 0
git diff $R --name-only -- ui/src/ | wc -l                               # = 0 (src frozen)
git diff $R --name-only -- ui/e2e/mocks/handlers/utils.ts ui/e2e/mocks/store.ts ui/e2e/mocks/fixture.ts | wc -l   # = 0 (shared infra frozen)
# routes_admin.go = exactly ONE commit, additive:
git log --oneline $R -- internal/server/routes_admin.go | wc -l          # = 1
git diff $R -- internal/server/routes_admin.go | grep -E '^-' | grep -v '^---' | wc -l   # = 0 (no deletions)
```

---

## 6. Out of scope (restated, binding)

No UI src edits (decision 8 — pages/components/routes/stores frozen); only the
sanctioned feature-flags/prompts mock-body + seed corrections (and NOT the shared
`utils.ts`/`store.ts`/`fixture.ts`). No POST/DELETE for feature-flags (toggle-only). No
edits to w7-gov-1 files (teams/audit/usermgmt/handlers/governance-audit) — `h.recordAudit`
is consumed read-only. No edits to pre-existing admin handlers. No new governance domain
file by default (ESC-FF-DOMAIN / ESC-PROMPT-DOMAIN). No JWT. No destructive DDL —
additive `ensureTable`/`ensureColumn` only. No `New(...)` signature change / no new
global state. No other governance domains (w7-gov-1 merged / w7-gov-3 ∥). No secret
exposure. Mock-vs-Go contradiction → escalate (§8), never fudge a mock or edit a frozen
handler.

## 7. Diff-gate scope

W7 governance plans (gov-1 merged; gov-2/3 concurrent) commit to main, so a broad
`<base>..HEAD` range sweeps in sibling commits. The diff gate MUST be scoped to
w7-gov-2's own commits. Isolate them:
`git log --oneline main | grep "^[0-9a-f]* phase-1/w7-gov-2:" | awk '{print $1}'`
then `git diff <first-w7-gov-2>^..<last-w7-gov-2> -- [file list]`.

`git diff <first>^..<last> --name-only` must be exactly a subset of:
```
internal/store/featureflags.go
internal/store/featureflags_test.go
internal/store/prompttemplates.go
internal/store/prompttemplates_test.go
internal/store/migrate.go               (additive table registration; additive ONLY)
internal/store/migrate_test.go          (CONDITIONAL — only if it already exists)
internal/admin/featureflags.go
internal/admin/featureflags_test.go
internal/admin/prompttemplates.go
internal/admin/prompttemplates_test.go
internal/server/routes_admin.go         (serial-slot additive routes; ONE commit)
internal/governance/featureflags.go     (CONDITIONAL — only if §8 ESC-FF-DOMAIN fires)
internal/governance/prompttemplates.go  (CONDITIONAL — only if §8 ESC-PROMPT-DOMAIN fires)
ui/e2e/mocks/handlers/feature-flags.ts  (body only — verify; minimal/no change)
ui/e2e/mocks/handlers/prompts.ts        (body only — drop updated_at from POST/PUT)
ui/e2e/mocks/seed/feature-flags.ts      (verify; correct only on divergence)
ui/e2e/mocks/seed/prompts.ts            (verify)
.planning/parity/matrix/9router-ui.md
.planning/parity/plans/open-questions.md
docs/WORKFLOW.md
```
Any file outside this list in the scoped diff is an automatic review REJECT.
`internal/admin/{teams,audit,usermgmt,handlers,auth}.go`, `internal/governance/audit.go`,
all other pre-existing admin/store handlers, and all `ui/src/**` + the shared mock infra
are deliberately ABSENT — touching them is an automatic REJECT. The `routes_admin.go`
edit must appear in exactly ONE commit (§5) and the serial slot is released to w7-gov-3
on close.

## 8. Escalations / decisions (explicit — recommended defaults, do not fabricate)

- **ESC-IDTYPE (RESOLVED at authoring — INT vs TEXT primary key, binding default).**
  w7-gov-1's teams used a TEXT PK via `newID()` (`store/teams.go:25`). But BOTH this
  plan's domains diverge: the mocks + UI types use NUMERIC ids
  (`FeatureFlag.id:number` `types.ts:96`; `PromptTemplate.id:number` `types.ts:188`;
  seeds `id:1,2,3`), and the feature-flags toggle spec hard-asserts a numeric URL
  (`/\/api\/feature-flags\/\d+$/`, `feature-flags.spec.ts:29`). **Decision: use
  `INTEGER PRIMARY KEY AUTOINCREMENT`** for both `feature_flags` and `prompt_templates`
  (id type int64 in Go; handlers parse `{id}` via `strconv.ParseInt`, not the
  string-only `pathID` path). This is a deliberate, documented divergence from the
  teams TEXT-PK precedent, driven by the binding mock/spec contract. RECOMMENDED as
  stated; flag for orchestrator confirmation; the plan proceeds on this default.
- **ESC-PROMPT-TEST (RESOLVED at authoring — `/test` response shape, binding default).**
  `POST /api/prompt-templates/test` is NOT consumed by the live UI (`grep -rn
  "prompt-templates/test" ui/src/` → ZERO) and no spec exercises it; the mock returns
  `{rendered: "Mock rendered prompt"}` (`prompts.ts:39`). **Decision:** the Go endpoint
  returns `{data:{rendered: string}}` mirroring the mock key. Default request body:
  `{prompt_id?:int, system_prompt?:string, sample?:string}` — if `prompt_id` is given,
  load the stored template's `system_prompt`; otherwise use the inline `system_prompt`;
  `rendered` = the resolved system_prompt with the `sample` appended/substituted (a
  deterministic, dependency-free "apply template to a sample" — NO live LLM call,
  matching the mock's static behavior). If the operator wants richer rendering (e.g.
  variable interpolation), that is a follow-up. RECOMMENDED as stated; flag for
  confirmation.
- **ESC-PROMPT-UPDATEDAT (RESOLVED at authoring — DTO field set).** The mock POST
  injects `updated_at` (`prompts.ts:11`) but the UI `PromptTemplate` type omits it
  (`types.ts:187-194`) and no spec asserts it. **Decision:** STORE `updated_at` in the
  table (hygiene/ordering) but OMIT it from `promptDTO` to match the UI type; the
  corrected mock POST/PUT DROP the `updated_at` field (mock mirrors Go). RECOMMENDED;
  low-risk.
- **ESC-FF-GETBYID (RESOLVED at authoring — optional single-read route).** The mock
  has `GET /api/feature-flags/{id}` (`feature-flags.ts:13-15`) the page never calls.
  **Decision: include the Go `GET /{id}` for parity** (cheap, mirrors the mock; no spec
  rides on it). If the orchestrator prefers a strictly minimal toggle-only surface,
  omit it — the spec passes either way. RECOMMENDED: include.
- **ESC-FF-STORE (RESOLVED at authoring — dedicated table vs `kv.go`).** The MAP listed
  `internal/store/kv.go` as a "flag-store option." **Decision: a dedicated
  `feature_flags` table** (NEW `store/featureflags.go`) — it cleanly mirrors the mock's
  5-field row shape (`{id,key,enabled,description,created_at}`), gives numeric ids
  (ESC-IDTYPE), and matches the teams/auditlog table precedent. NOT the kv table.
- **ESC-FF-DOMAIN / ESC-PROMPT-DOMAIN (RESOLVED at authoring — no governance file by
  default).** Per the w7-gov-1 ESC-ARCH finding (NO in-tree arch test enforces
  transport→domain→repository; teams/virtualkeys call `h.store` directly), this plan
  builds handler→store DIRECTLY for both domains and creates NEITHER
  `internal/governance/featureflags.go` NOR `internal/governance/prompttemplates.go`
  (a deliberate deviation from the MAP's listed new files). ADD a thin domain file
  ONLY if (a) the `/test` render logic becomes non-trivial (prompt domain) or (b) an
  arch test is introduced before T-routes. Decide at T-prompts/T-routes; do NOT
  pre-build the wrappers.
- **ESC-ROUTE (CONDITIONAL — fasthttp/router precedence).** `/api/prompt-templates/test`
  (static) vs `…/{id}` and `/api/feature-flags` vs `…/{id}` follow the file's existing
  static-before-param ordering (`routes_admin.go:52-57` teams precedent). Register
  `/test` BEFORE `…/{id}`. If the matcher mis-disambiguates (the `{id}` param swallows
  `test`, returning a "prompt template not found" 404 for the test path), STOP and
  ESCALATE for a path arrangement (e.g. `/api/prompt-templates-test`) — never silently
  diverge page/mock/Go. (NOTE: `/test` is not UI-consumed, so a mis-disambiguation
  would NOT red a spec — verify with an explicit Go handler test that the `/test`
  route resolves to `TestPromptTemplate`, not `GetPromptTemplate`.)
- **ESC-MOCK (CONDITIONAL — shared mock ripple).** `feature-flags.ts`/`prompts.ts` are
  consumed only by the w7-gov-2 specs (no other spec references them — verify with
  `grep -rn "feature-flags\|prompt-templates" ui/e2e/*.spec.ts`). The shared `utils.ts`
  `json()`/`error()` helpers are NOT edited. If a body correction reds a non-w7-gov-2
  spec, STOP and ESCALATE for orchestrator serialization — no fudge, no frozen-branch
  edit.
- **ESC-AUDIT-REUSE (RESOLVED at authoring — reuse w7-gov-1 seam).** This plan's
  mutations (flag toggle; prompt create/update/delete) call `h.recordAudit(ctx,
  action, target, details)` (`audit.go:64-72`) best-effort, post-success, mirroring
  w7-gov-1's ESC-AUDIT-WRITE disposition. NO edit to audit.go / governance/audit.go /
  handlers.go (the `audit` field + accessor already exist). `details` are
  human-readable summaries (flag key, template name) — NEVER raw payloads.
- **Serial-slot dependency (§1.6 / P5).** w7-gov-2 TAKES the routes_admin.go slot after
  w7-gov-1 releases it (chain MAP §219-224) and RELEASES it to w7-gov-3 on close.
  Orchestrator confirms exactly one unmerged holder (decision 3) before T-routes.
- **No other blocking dependency.** All reused surfaces (store/teams.go CRUD template,
  respond.go, pathID, newTestEnv, migrate additive pattern, the w7-gov-1
  `h.recordAudit` seam) are in-tree at <base>. w7-gov-2 is unblocked once the serial
  slot is free.
```
