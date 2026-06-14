# Micro-plan w7-gov-1 — Governance backends A: teams + user-management auth + audit-log (Go)

```
wave: 7
plan: w7-gov-1
status: READY (rev 1 — authored against merged Waves 0–6, live tree @ <base>;
  WAVE-7-MAP w7-gov-1 row ~line 171; serial chain §219-224; reconciliation §245;
  freeze rules §267)
runs: governance+routing track. Disjoint domain/store/admin files from w7-gov-2,
  w7-gov-3 (run ∥). TAKES the internal/server/routes_admin.go SERIAL SLOT after
  w7-route releases it (chain: w7-platnodes → w7-route → **w7-gov-1** → w7-gov-2 →
  w7-gov-3 → w7-mcp-3 → w7-plat-1 → w7-plat-2 → w7-plat-3 → w7-misc; MAP §219-224).
branch: main (direct commits, per repo convention)
commit-prefix: phase-1/w7-gov-1:
ref-source: 9router frozen @ 827e5c3 — governance teams/audit/user-management
  surfaces; the BINDING contract for W7 is the W6 e2e mock (decision 1: real Go
  wins, mock corrected to mirror it). Mock sources:
    ui/e2e/mocks/handlers/{teams,audit,auth}.ts + seed/{teams,audit,auth}.ts.
base: <base> = git rev-parse HEAD recorded at P0. Substitute the actual SHA
  everywhere §5 says <base>.
go-serial-slot: this plan holds the SINGLE in-flight edit to
  internal/server/routes_admin.go while live (W3/W4/W5/W6 lesson; MAP decision 3).
  Slot must be FREE at P-check (w7-route merged + slot released) before T-routes.
new-route: NO UI route files. All three UI pages (/teams, /audit) + the w6-k Users
  panel ALREADY SHIPPED in w6-k against mocks; this plan builds the REAL Go so the
  pages flip mock→true-HAVE and corrects the mock bodies to mirror the Go DTOs.
```

---

## 1. Scope — PAR rows + the three domains

### Rows this plan closes

| Row / item | Claim | Target state after w7-gov-1 |
|---|---|---|
| open-questions w6-k **ESC-1a** (teams backend absent) | real `/api/teams` CRUD | true-HAVE (Go — NEW `internal/store/teams.go` + `internal/governance/teams.go` + `internal/admin/teams.go`, §1.4) |
| open-questions w6-k **ESC-1b** (audit backend absent) | real `GET /api/audit` paginated read + store | true-HAVE (Go — NEW `internal/store/auditlog.go` + `internal/admin/audit.go` + a `WriteAudit` helper, §1.5) |
| open-questions w6-k **ESCALATION-2** / **PAR-UI-132** (user-management absent) | real `/api/auth/{setup,password,users[/{id}]}` | true-HAVE (Go — NEW `internal/admin/usermgmt.go` over existing `internal/store/users.go`, password hashing reuses `internal/auth/password.go`, §1.6) |

Matrix flips at closeout (§4 T-close): in `.planning/parity/matrix/9router-ui.md`,
PAR-UI-132 → HAVE (real Go). The teams/audit rows live in the open-questions ledger
as w6-k ESC-1a/1b serial Go follow-ups; mark them RESOLVED in `open-questions.md`
and (if a matrix row exists for them) flip mock→true-HAVE with a cite to this plan.

### 1.1 Preconditions already satisfied by merged waves (evidence)

- **W6-k UI is SHIPPED and FROZEN (consume-only, MAP decision 8 / §267).** The
  `/teams` page (incl. the Users panel `ui/src/components/governance/users-panel.tsx`)
  and `/audit` page render against the registered mocks. The binding acceptance
  contracts are the existing specs (must stay green at closeout):
  - `ui/e2e/teams.spec.ts` — 6 tests: page loads ("Teams"); `[data-testid="team-row"]`
    ≥2 from seed (Engineering, Data Science); team-form modal traffic lights;
    create via `#team-name` + `[data-testid="team-save"]`; delete via confirm modal
    ("Delete team"); **Users panel** lists seeded `admin` user + an
    `input[aria-label="New password"]` change-password control.
  - `ui/e2e/audit.spec.ts` — 3 tests: page loads ("Audit"); `[data-testid="audit-row"]`
    ≥5 from seed (renders "create_key" + "admin"); `[data-testid="audit-limit"]`
    pagination/limit control.
- **Real Go store layer for users ALREADY EXISTS** (`internal/store/users.go`,
  the big de-risk): `CreateUser(username,passwordHash)` (`users.go:20`),
  `GetUserByUsername` (`users.go:44`), `GetUserByID` (`users.go:50`), `CountUsers`
  (`users.go:68`), `FirstUser` (`users.go:77`), `UpdateUserPassword(id,hash)`
  (`users.go:83`), `SetUserPasswordHash(username,hash)` (`users.go:102`). The
  `users` table is `{id,username,password_hash,created_at,updated_at}`
  (`migrate.go:19-25`). **No user LIST or user DELETE store method exists yet** —
  this plan ADDS `ListUsers()` + `DeleteUser(id)` to `users.go` (additive methods,
  §1.6 / §3).
- **Password hashing EXISTS and is the canonical surface** (`internal/auth/password.go`):
  `HashPassword(pw) (string,error)` (PBKDF2-SHA256, format
  `pbkdf2-sha256$<iter>$<salt>$<hash>`, `password.go:22`) +
  `VerifyPassword(hash,pw) bool` (`password.go:40`). **REUSE these; do NOT
  reimplement.** First-user seeding precedent: `auth.Sessions.SeedAdmin(user,pw)`
  (`internal/auth/session.go:34`) — creates a user ONLY when `CountUsers()==0`,
  hashing via `HashPassword`. Login's empty-hash / `INITIAL_PASSWORD` default path
  is `session.go:53-66` (relevant to setup semantics, §8 ESC-USERMGMT).
- **Envelope + handler patterns** (`internal/admin/respond.go`): `writeData(ctx,
  status, data)` / `writeError(ctx, status, message)` → `{data,error:{message}}`
  snake_case (`respond.go:19,23`). `pathID(ctx.UserValue("id"))` extracts `{id}`
  (`handlers.go:71`). CRUD template = `internal/admin/virtualkeys.go`
  (List/Create/Get/Update/Delete with DTO + request structs + validate fn +
  ErrNotFound→404, `virtualkeys.go:86-216`).
- **Store CRUD template** (`internal/store/virtualkeys.go`): `newID()` for IDs
  (`virtualkeys.go:56`), `time.Now().Unix()` timestamps, `boolToInt` for SQLite
  bools, `scanX` helper, `ErrNotFound` on `sql.ErrNoRows`, JSON-blob config column
  (`config_json`) for nested data (`virtualkeys.go:23-51,84-92`).
- **Domain template** (`internal/governance/quota.go`): a domain package with a
  constructor (`NewQuotaEngine`), interface-typed deps (`SpendReader`), no
  `init()`, errors-as-values, no global state. Governance pkg already exists
  (`internal/governance/{doc.go,quota.go,quota_test.go}`).
- **Migrations are additive-only** (`internal/store/migrate.go`): new tables via the
  `tables []struct{name,create}` slice with `CREATE TABLE IF NOT EXISTS`
  (`migrate.go:15-140`); new columns via `ensureColumn(db,table,column,decl)`
  (`migrate.go:180-188,261`). Secret-at-rest precedent = the `*_enc` reversible
  columns (`connections.secret_enc`/`access_token_enc`/`refresh_token_enc`,
  `migrate.go:51-53`) written/read via `s.cipher.Encrypt/Decrypt`
  (`oauthsessions.go:21,58`).
- **Admin test harness** (`internal/admin/admin_test.go:24` `newTestEnv`): real
  `store.Open(tempDB, secret)` + `auth.NewSessions` + `SeedAdmin("admin","123456")`
  + `New(st,sessions,nil)`. NO mocks. `call(t, handler, method, uri, body,
  userValues, headers)` (`admin_test.go:72`) drives a handler and decodes the
  envelope; `dataField[T]` / `errMessage` extract fields; `loginToken` mints a
  session (`admin_test.go:125`). This is the authoritative proof surface.
- **Handlers injection** (`internal/admin/handlers.go`): the `Handlers` struct
  holds `store`/`sessions`/`flows`/… and exposes additive setters
  (`SetUsageServices`, `SetVersionInfo`, `SetShutdownFunc`). New domains compose
  `h.store` directly (like virtualkeys/apikeys) — NO new global state, NO
  `New(...)` signature change (MAP decision 9).

### 1.2 The mock contracts these flips must mirror (binding — decision 1)

**Decision 1 (MAP §36, §245):** real Go wins; the W6 mock body + seed are corrected
IN THIS PLAN to mirror the real Go `{data,error}` snake_case DTO. The page is FROZEN
(decision 8); where the real DTO and the mock disagree, **prefer matching the mock's
existing field names in the Go DTO** (the mock fields were modeled to match 9router);
only ESCALATE if impossible. Below is the exact mock-vs-Go reconciliation per domain.

**Teams** (`ui/e2e/mocks/handlers/teams.ts` + `seed/teams.ts`):
- Routes: `GET /api/teams` (list), `POST /api/teams` (create),
  `GET|PUT|DELETE /api/teams/{id}` (`teams.ts:6,17`).
- Mock list/seed shape = the UI `Team` type (`ui/src/lib/types.ts:270-278`):
  `{id,name,budget_usd,budget_used_usd,budget_period,rate_limit_rpm}` — **this is
  the canonical Go DTO** (matches the brief).
- **Mock POST divergence** (`teams.ts:11`): on create the mock spreads extras
  `{budget_used_usd:0, keys_count:0, members:0, ...body}`. `keys_count`/`members`
  are NOT in the UI `Team` type and NOT consumed by the page (the spec only checks
  the team name renders, `teams.spec.ts:35-43`). **Reconciliation:** the Go create
  DTO returns the 6 canonical fields with `budget_used_usd` defaulted to 0; the
  corrected mock DROPS `keys_count`/`members` from the POST body (mock mirrors Go).
- Mock DELETE returns `json(route, {})` → Go returns `{data:{message:"..."}}`
  (matches `DeleteVirtualKey` shape); the page ignores the body on delete.

**Audit** (`ui/e2e/mocks/handlers/audit.ts` + `seed/audit.ts`):
- Routes: `GET /api/audit?limit=N` → `{data:{items:[...],total:N}}` (`audit.ts:6-12`);
  the mock also has a `POST /api/audit` (`audit.ts:13-19`) but the page never POSTs
  audit (audit entries are written server-side). **Reconciliation:** Go serves the
  READ `GET /api/audit?limit=` → `{data:{items,total}}`; the corrected mock keeps
  the GET and **removes the unused POST** (mock mirrors the Go read-only public
  surface; audit writes are internal-only, §1.5).
- Mock/seed entry shape = UI `AuditLog` (`types.ts:42-49`):
  `{id,timestamp,actor,action,target,details?}` — the canonical Go DTO. `timestamp`
  is ISO-8601 (`new Date(...).toISOString()`, `seed/audit.ts`). `details` is
  optional.

**User-management** (`ui/e2e/mocks/handlers/auth.ts` + `seed/auth.ts`):
- Routes the page/Users-panel consumes (`auth.ts:83-130`):
  `POST /api/auth/setup` (first-user onboarding); `PUT /api/auth/password`
  (`{current_password,new_password}`); `GET /api/auth/users` (list, password
  stripped); `POST /api/auth/users` (`{username,display_name?,role?,password}` →
  409 on dup); `DELETE /api/auth/users/{id}`.
- Mock/seed `User` shape (`types.ts:309-315`, `seed/auth.ts`):
  `{id,username,display_name,role,password?}`. **DIVERGENCE:** the Go `store.User`
  is `{id,username,password_hash,created_at,updated_at}` — it has **NO `display_name`
  / `role`** (`users.go:11-17`). The Users-panel spec only asserts the seeded
  `admin` username renders + a "New password" input (`teams.spec.ts:60-71`) — it
  does NOT assert `display_name`/`role` values. **Reconciliation decision:** see
  §8 ESC-USERMGMT — RECOMMENDED default = ADD additive `ensureColumn`
  `users.display_name` + `users.role` (additive-only, MAP decision 2) so the Go DTO
  carries the mock's fields; the corrected `auth.ts` users routes mirror the Go
  `{data}` envelope and strip `password`/`password_hash`. (Note: `auth.ts` was
  w6-c-owned and consumed-unchanged by w6-k; w7-gov-1 now OWNS the users-route
  corrections in `auth.ts` since it builds their backend — §3.)

### 1.3 Architecture (binding — layered DDD, decision 4)

Three domains, each layered transport → domain → repository:

```
teams:   admin/teams.go      → governance/teams.go     → store/teams.go (NEW table teams)
audit:   admin/audit.go      → governance/audit.go     → store/auditlog.go (NEW table audit_log)
                                (+ WriteAudit helper on the audit domain service)
usermgmt: admin/usermgmt.go  → (reuses auth.HashPassword/VerifyPassword)
                              → store/users.go (EXTEND: +ListUsers +DeleteUser; +cols)
```

- Teams is a CRUD domain (mirror `virtualkeys.go`); a thin `governance.TeamService`
  wraps `*store.Store` for testability and to satisfy the arch test (transport must
  not skip the domain layer for new backends — decision 4). If the quota arch test
  permits a direct store CRUD for teams (as virtualkeys/apikeys do — they call
  `h.store` directly with NO domain wrapper), the domain wrapper for teams is
  OPTIONAL; **decide at T-teams against the arch test** (default: follow the
  virtualkeys precedent = handler→store directly, NO new domain file for teams,
  since teams is pure CRUD like virtualkeys; add `governance/teams.go` ONLY if the
  arch test forbids handler→store for a brand-new domain — §8 ESC-ARCH).
- Audit gets a domain service `governance.AuditService` because it exposes a
  `WriteAudit(entry)` helper that other admin mutations will call (the write
  integration, §1.5) — a domain seam is warranted here even if teams skips it.
- User-management reuses `auth.HashPassword`/`VerifyPassword` + the existing
  `store/users.go` methods; the transport handler is `admin/usermgmt.go`. Some
  endpoints (setup/password) touch session state — the handler composes
  `h.store` + `h.sessions` (already on `Handlers`).

### 1.4 Teams Go contract (NEW, TDD)

Table `teams` (additive, `migrate.go` tables slice):
```sql
CREATE TABLE IF NOT EXISTS teams (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  budget_usd REAL NOT NULL DEFAULT 0,
  budget_used_usd REAL NOT NULL DEFAULT 0,
  budget_period TEXT NOT NULL DEFAULT 'monthly',
  rate_limit_rpm INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
)
```

`internal/store/teams.go` (NEW): `Team` struct + `CreateTeam`/`ListTeams`/
`GetTeamByID`/`UpdateTeam`/`DeleteTeam` (mirror `virtualkeys.go`: `newID()`,
unix timestamps, `ErrNotFound`, `scanTeam`).

`internal/admin/teams.go` (NEW):

| Handler | Route | Shape (snake_case, `{data}`) | Notes |
|---|---|---|---|
| `ListTeams` | `GET /api/teams` | `{teams:[teamDTO]}` OR a bare array (match the mock `GET` which returns `Array.from(...values())` → a bare JSON array under `{data}`). **Pick bare array under `{data}`** to mirror `teams.ts:8`; confirm the page reads `data` as an array | `teamDTO{id,name,budget_usd,budget_used_usd,budget_period,rate_limit_rpm}` |
| `CreateTeam` | `POST /api/teams` | body `{name,budget_usd?,budget_period?,rate_limit_rpm?}`; `budget_used_usd` defaults 0; returns `{data:teamDTO}`. 400 on empty name | DROP mock's `keys_count`/`members` (§1.2) |
| `GetTeam` | `GET /api/teams/{id}` | `{data:teamDTO}` or 404 | |
| `UpdateTeam` | `PUT /api/teams/{id}` | body = create body; returns updated `{data:teamDTO}` or 404 | |
| `DeleteTeam` | `DELETE /api/teams/{id}` | `{data:{message:"Team deleted successfully"}}` or 404 | mock returns `{}`; page ignores body |

### 1.5 Audit Go contract (NEW, TDD) + the write-integration decision

Table `audit_log` (additive):
```sql
CREATE TABLE IF NOT EXISTS audit_log (
  id TEXT PRIMARY KEY,
  timestamp TEXT NOT NULL,          -- ISO-8601 (RFC3339), mirrors mock
  actor TEXT NOT NULL,
  action TEXT NOT NULL,
  target TEXT NOT NULL DEFAULT '',
  details TEXT NOT NULL DEFAULT ''
)
```
Index `idx_audit_log_timestamp ON audit_log(timestamp DESC)` for the ordered read.

`internal/store/auditlog.go` (NEW): `AuditEntry{ID,Timestamp,Actor,Action,Target,Details}`
+ `InsertAuditEntry(e)` + `ListAuditEntries(limit int) ([]AuditEntry,error)` (ORDER
BY timestamp DESC, LIMIT) + `CountAuditEntries() (int,error)`.

`internal/governance/audit.go` (NEW): `AuditService` wrapping `*store.Store` with:
- `List(limit int) (items []AuditEntry, total int, err error)`.
- **`WriteAudit(actor, action, target, details string) error`** — the internal
  write hook. Generates id + RFC3339 timestamp, inserts. **Secret-safety: `details`
  MUST NOT contain secrets** — callers pass human-readable summaries only
  (e.g. "Created team Engineering"), never raw passwords/tokens (§5 grep proof).

`internal/admin/audit.go` (NEW):

| Handler | Route | Shape | Notes |
|---|---|---|---|
| `GetAudit` | `GET /api/audit?limit=N` | `{data:{items:[auditDTO],total:N}}` | `limit` default 100 (mirror `audit.ts:10`), clamp to a sane max (e.g. 1000); `auditDTO{id,timestamp,actor,action,target,details?}` |

**Write-integration scope (DECISION — §8 ESC-AUDIT-WRITE).** RECOMMENDED default
for w7-gov-1: ship the **read endpoint + store + the `WriteAudit` helper**, and
wire writes for the mutations THIS PLAN introduces (team create/update/delete, user
create/delete, password change, setup) — calling `auditSvc.WriteAudit(actor, ...)`
inside each w7-gov-1 handler AFTER the successful mutation, best-effort (a
WriteAudit error is logged, never fails the parent request). Do NOT retrofit audit
writes into pre-existing handlers owned by other files (apikeys/virtualkeys/
providers/connections) in this plan — that is a forbidden cross-file edit; a follow-up
plan wires those (recorded in `open-questions.md`). The `actor` is the authenticated
user from `ctx.UserValue(userKey).(*store.User).Username` (`auth.go:20,157-163`).

### 1.6 User-management Go contract (NEW, TDD)

Reuses `internal/store/users.go` (existing methods) + ADDS two store methods +
(per §8 ESC-USERMGMT default) two additive columns. Password hashing reuses
`internal/auth/password.go`. NEW transport file `internal/admin/usermgmt.go`
(prefer a NEW file + MINIMAL `auth.go` touch — the brief's stated preference; the
only `auth.go` edit, if any, is none — all new handlers live in usermgmt.go).

Store additions (`internal/store/users.go`, additive methods only):
- `ListUsers() ([]*User, error)` — `SELECT ... ORDER BY created_at ASC`.
- `DeleteUser(id string) error` — `DELETE FROM users WHERE id=?`, `ErrNotFound`
  on 0 rows. **Guard: refuse to delete the last user** (return a sentinel/400 at
  the handler) so the dashboard can't lock itself out.
- Additive columns (per ESC-USERMGMT default): `ensureColumn("users",
  "display_name", "TEXT NOT NULL DEFAULT ''")` + `ensureColumn("users", "role",
  "TEXT NOT NULL DEFAULT 'user'")`. Extend `User` struct + `CreateUser` signature
  carefully: to avoid churning the existing `CreateUser(username,passwordHash)`
  callers (`SeedAdmin`, tests), ADD a separate `CreateUserFull(username,
  passwordHash, displayName, role)` OR widen via an options struct — **default:
  add `CreateUserFull` + keep `CreateUser` as a thin wrapper** (displayName=username,
  role="admin" for the seed path) so `SeedAdmin` is unchanged. Decide at T-usermgmt.

`internal/admin/usermgmt.go` (NEW):

| Handler | Route | Body / response | Notes |
|---|---|---|---|
| `AuthSetup` | `POST /api/auth/setup` | body `{username,password,display_name?}`; **first-user onboarding ONLY** — succeeds iff `CountUsers()==0` (mirrors `SeedAdmin` semantics); creates the admin user (hashed), then logs them in (issues a session token + sets the `g0_session` cookie, mirroring `Login`, `auth.go:110-123`). 409/403 if users already exist. Returns `{data:{token,user:userDTO}}` | §8 ESC-USERMGMT defines this; mock auto-authenticates (`auth.ts:88-92`) |
| `ChangePassword` | `PUT /api/auth/password` | body `{current_password,new_password}`; resolves the CURRENT authenticated user (`ctx.UserValue(userKey)`), `VerifyPassword(user.PasswordHash, current)` → 400 "Current password is incorrect" on mismatch (also handle the empty-hash/`INITIAL_PASSWORD` default-password case, `session.go:53-66`), then `HashPassword(new)` + `UpdateUserPassword(id,hash)`. Returns `{data:{}}`. NEVER echoes either password | mock `auth.ts:97-107` |
| `ListUsers` | `GET /api/auth/users` | `{data:[userDTO]}` — password_hash STRIPPED | mock `auth.ts:110` |
| `CreateUser` | `POST /api/auth/users` | body `{username,display_name?,role?,password}`; 409 if username exists; hash pw; create; returns `{data:userDTO}` (no hash) | mock `auth.ts:111-117` |
| `DeleteUser` | `DELETE /api/auth/users/{id}` | `{data:{}}` or 404; refuse last-user delete (400) | mock `auth.ts:120-129` |

`userDTO{id,username,display_name,role}` (NO password/hash, NEVER). `setup` +
`change-password` are session-touching; `users` list/create/delete are admin CRUD
behind `RequireSession`.

**Route protection nuance:** `POST /api/auth/setup` must be reachable WITHOUT a
session (it bootstraps the first user) — register it PUBLIC, like
`POST /api/auth/login` (`routes_admin.go:34`), but it self-guards on
`CountUsers()==0`. All other usermgmt routes are `RequireSession`-wrapped.

### 1.7 routes_admin.go registration (serial-slot additive, §3)

Add (additive appends; static-before-`{id}` precedence already honored by the file,
see `routes_admin.go:56-61`):
```go
// Public first-user onboarding (self-guards on CountUsers()==0).
r.POST("/api/auth/setup", h.AuthSetup)
// Protected user-management.
r.PUT("/api/auth/password", h.RequireSession(h.ChangePassword))
r.GET("/api/auth/users", h.RequireSession(h.ListUsers))
r.POST("/api/auth/users", h.RequireSession(h.CreateUser))
r.DELETE("/api/auth/users/{id}", h.RequireSession(h.DeleteUser))
// Teams CRUD (static collection before {id}).
r.GET("/api/teams", h.RequireSession(h.ListTeams))
r.POST("/api/teams", h.RequireSession(h.CreateTeam))
r.GET("/api/teams/{id}", h.RequireSession(h.GetTeam))
r.PUT("/api/teams/{id}", h.RequireSession(h.UpdateTeam))
r.DELETE("/api/teams/{id}", h.RequireSession(h.DeleteTeam))
// Audit read.
r.GET("/api/audit", h.RequireSession(h.GetAudit))
```
Route-precedence note: `/api/auth/users` (static) vs `/api/auth/users/{id}` — the
file already registers static-before-param elsewhere; verify the
`fasthttp/router` precedence at impl. A genuine collision is §8 ESC-ROUTE, not a
silent path change. Diff bound §5: the route block is ONE commit, additive only.

### NOT in scope (explicit)

- **No UI page/component/route/store edits** — `/teams`, `/audit`, the Users panel,
  and all w6-k components are FROZEN consume-only (decision 8). The ONLY UI-tree
  touches are the mock-body + seed corrections (§1.2 / §3).
- **No edits to other governance domains** — feature-flags (w7-gov-2), prompts
  (w7-gov-2), guardrails/alerts (w7-gov-3) are disjoint; do not touch their mocks/
  seeds/Go.
- **No edits to pre-existing admin handlers' bodies** — apikeys.go, virtualkeys.go,
  providers*.go, connections.go, combos.go, disabledmodels.go, auth.go (login/
  logout/me/status), version.go, usage/pricing handlers are FORBIDDEN. (Audit
  writes for THOSE mutations are a tracked follow-up, §1.5.)
- **No JWT** — sessions stay opaque DB tokens (the existing `auth.Sessions`).
- **No destructive DDL / column renames** — additive `ensureTable`/`ensureColumn`
  ONLY (decision 2).
- **No new global state / no `New(...)` signature change** (decision 9) — handlers
  compose `h.store`/`h.sessions`; the audit service is constructed where needed
  (a thin `auditService(h)` accessor over `h.store`, or an additive `h.audit`
  field set in `New` with NO signature change — decide at T-audit).
- **No secret exposure** — passwords hashed, never echoed; audit `details` carry
  no secrets (§5 grep proofs).

---

## 2. Precondition checks

Run all before any edit; abort and report to orchestrator on any failure.

```bash
# P0 — clean tree; record <base>
git status --porcelain     # must be empty (explicit `git add <file>`, never -A;
                           # ui/dist/** gitignored — never stage it)
git rev-parse HEAD         # record as <base> for §5

# P1 — the three gaps are REAL (no Go for any domain)
grep -nE '/api/teams|/api/audit' internal/server/routes_admin.go ; echo "^ expect EMPTY"
grep -nE '/api/auth/(setup|password|users)' internal/server/routes_admin.go ; echo "^ expect EMPTY"
test ! -e internal/store/teams.go && test ! -e internal/store/auditlog.go && echo "store gap OK"
test ! -e internal/admin/teams.go && test ! -e internal/admin/audit.go && test ! -e internal/admin/usermgmt.go && echo "admin gap OK"
test ! -e internal/governance/audit.go && echo "audit domain gap OK"

# P2 — reused surfaces present (the de-risk)
grep -n "func (s \*Store) CreateUser\|GetUserByUsername\|GetUserByID\|CountUsers\|FirstUser\|UpdateUserPassword" internal/store/users.go
grep -n "func HashPassword\|func VerifyPassword" internal/auth/password.go
grep -n "func (s \*Sessions) SeedAdmin" internal/auth/session.go
grep -n "func writeData\|func writeError" internal/admin/respond.go
grep -n "func newTestEnv\|func call\|func dataField\|func loginToken" internal/admin/admin_test.go

# P3 — migrate pattern + secret-at-rest precedent
grep -n "CREATE TABLE IF NOT EXISTS\|func ensureColumn\|cipher" internal/store/migrate.go | head
grep -n "s.cipher.Encrypt\|s.cipher.Decrypt" internal/store/oauthsessions.go

# P4 — the W6-k UI + specs are present (consume-only) and the mocks to correct
test -f ui/e2e/teams.spec.ts && test -f ui/e2e/audit.spec.ts && echo "specs present"
test -f ui/e2e/mocks/handlers/teams.ts && test -f ui/e2e/mocks/handlers/audit.ts && test -f ui/e2e/mocks/handlers/auth.ts && echo "mocks present"
test -f ui/e2e/mocks/seed/teams.ts && test -f ui/e2e/mocks/seed/audit.ts && test -f ui/e2e/mocks/seed/auth.ts && echo "seeds present"
grep -n "keys_count\|members" ui/e2e/mocks/handlers/teams.ts ; echo "^ the POST extras to drop (§1.2)"
grep -n "POST" ui/e2e/mocks/handlers/audit.ts ; echo "^ the unused audit POST to remove (§1.2)"

# P5 — routes_admin.go serial slot is FREE (w7-route merged + released)
git log --oneline -5 -- internal/server/routes_admin.go   # last touch = w7-route (merged)
# Orchestrator MUST confirm no concurrent W7 plan holds an unmerged routes_admin.go
# edit before w7-gov-1 begins T-routes (chain: w7-platnodes→w7-route→**w7-gov-1**).
# w7-gov-1 TAKES the slot, then RELEASES it to w7-gov-2 on close.

# P6 — green at base
go test ./... && go vet ./... && go build ./...     # exit 0 (Go untouched-green)
cd ui && npx playwright test e2e/teams.spec.ts e2e/audit.spec.ts
# Record base: these PASS at base against the W6 mocks. They must STAY green after
# the mock-body corrections. Record exact pass/fail in WORKFLOW.md.
cd ui && npm run build                               # exit 0
```

---

## 3. Exclusive file ownership

After w7-gov-1 merges, all CREATE files are owned by w7-gov-1; later plans consume,
never edit (MAP decision 7).

**CREATE — store (NEW):**

| File | Contract |
|---|---|
| `internal/store/teams.go` | `Team` struct + `CreateTeam`/`ListTeams`/`GetTeamByID`/`UpdateTeam`/`DeleteTeam` + `scanTeam`; mirrors `virtualkeys.go`. `newID()`, unix ts, `ErrNotFound`. |
| `internal/store/teams_test.go` | Table-driven store tests via a temp `store.Open` (mirror existing store tests): create→get→list→update→delete→404. RED first. |
| `internal/store/auditlog.go` | `AuditEntry` struct + `InsertAuditEntry`/`ListAuditEntries(limit)`/`CountAuditEntries`. RFC3339 timestamps; ORDER BY timestamp DESC LIMIT. |
| `internal/store/auditlog_test.go` | insert N → list(limit) returns ≤limit newest-first; count == N. RED first. |

**EXTEND — store (additive methods + columns only):**

| File | Change (additive ONLY) |
|---|---|
| `internal/store/users.go` | ADD `ListUsers()`, `DeleteUser(id)` (+ last-user guard at handler), `CreateUserFull(username,hash,displayName,role)` (keep `CreateUser` as a wrapper so `SeedAdmin` is unchanged); extend `User` struct with `DisplayName`/`Role`; update `scanUser`/SELECTs to include the new columns. The existing `CreateUser`/`GetUserBy*`/`CountUsers`/`UpdateUserPassword` signatures are PRESERVED. |
| `internal/store/users_test.go` (CREATE if absent, else EXTEND additively) | RED first: list users; delete user→404 on missing; CreateUserFull persists display_name/role; password_hash never returned by a DTO-shaped read. |
| `internal/store/migrate.go` | ADD the `teams` + `audit_log` tables to the `tables` slice; ADD the `idx_audit_log_timestamp` index; ADD `ensureColumn("users","display_name",...)` + `ensureColumn("users","role",...)` to the additive-column loop. ADDITIVE ONLY. |
| `internal/store/migrate_test.go` (if present — EXTEND; else rely on store tests) | assert the new tables/columns exist post-migrate (additive). |

**CREATE — domain (NEW):**

| File | Contract |
|---|---|
| `internal/governance/audit.go` | `AuditService` over `*store.Store`: `List(limit)`, `WriteAudit(actor,action,target,details)`. Constructor `NewAuditService(st)`. No `init()`; errors-as-values. |
| `internal/governance/audit_test.go` | WriteAudit→List round-trips; details with a secret-looking string is the CALLER's responsibility (test asserts WriteAudit stores verbatim — the secret-safety contract is at the call sites, §5). RED first. |
| `internal/governance/teams.go` (CONDITIONAL — only if arch test forbids handler→store, §8 ESC-ARCH) | `TeamService` thin CRUD wrapper. Default: NOT created (follow virtualkeys handler→store precedent). |

**CREATE — transport (NEW):**

| File | Contract |
|---|---|
| `internal/admin/teams.go` | `ListTeams`/`CreateTeam`/`GetTeam`/`UpdateTeam`/`DeleteTeam` + `teamDTO` + request/validate; `writeData`/`writeError`; after each mutation calls `WriteAudit` (best-effort, §1.5). |
| `internal/admin/teams_test.go` | via `newTestEnv`: create→list(≥1)→get→update→delete→404; create empty-name→400; **assert an audit entry is written on create** (`GetAudit` shows it). RED first. |
| `internal/admin/audit.go` | `GetAudit` (limit parse + clamp) → `{data:{items,total}}`. |
| `internal/admin/audit_test.go` | seed entries via WriteAudit → `GetAudit?limit=2` returns 2 items + correct total; newest-first. RED first. |
| `internal/admin/usermgmt.go` | `AuthSetup`/`ChangePassword`/`ListUsers`/`CreateUser`/`DeleteUser` + `userDTO`; reuses `auth.HashPassword`/`VerifyPassword`; password never echoed. |
| `internal/admin/usermgmt_test.go` | via `newTestEnv` (seeds an `admin` user): setup-when-users-exist→409/403; change-password wrong-current→400, correct→200 + new password logs in; list strips hash (grep response for `password`/hash → absent); create dup→409; delete→404 missing; **delete-last-user→400**; **no response leaks a password/hash**. RED first. |

**MODIFY — serial-slot route registration (additive only):**

| File | Change |
|---|---|
| `internal/server/routes_admin.go` | ADD the 11 route lines (§1.7). NOTHING else changes. ONE commit. SERIAL SLOT — only holder while live; RELEASE to w7-gov-2 on close. |

**MODIFY — handlers wiring (additive only, IF an `h.audit` field is chosen):**

| File | Change |
|---|---|
| `internal/admin/handlers.go` | OPTIONAL additive: add an `audit *governance.AuditService` field constructed in `New` (NO signature change — `New` already takes `st`; build `governance.NewAuditService(st)` inside). If a free function/accessor is cleaner, skip this edit. Decide at T-audit. |

**MODIFY — e2e mock corrections (mirror real Go, decision 1):**

| File | Change |
|---|---|
| `ui/e2e/mocks/handlers/teams.ts` (BODY) | POST: DROP `keys_count`/`members` extras; default `budget_used_usd:0`; mirror the 6-field `teamDTO`. GET/PUT/DELETE already mirror. |
| `ui/e2e/mocks/handlers/audit.ts` (BODY) | KEEP `GET` → `{items,total}`; REMOVE the unused `POST /api/audit` branch (audit writes are internal, §1.5). |
| `ui/e2e/mocks/handlers/auth.ts` (BODY — users routes ONLY) | Correct `setup`/`password`/`users` route bodies to mirror the Go `{data}` envelope + `userDTO{id,username,display_name,role}` (strip password). DO NOT touch the `login`/`logout`/`status` branches (w6-c contract, consumed). |
| `ui/e2e/mocks/seed/teams.ts` (BODY) | Already the 6-field shape — verify; correct only if a field name diverges. |
| `ui/e2e/mocks/seed/audit.ts` (BODY) | Already `{id,timestamp,actor,action,target,details?}` — verify; no change expected. |
| `ui/e2e/mocks/seed/auth.ts` (BODY) | Already `{id,username,display_name,role,password}` — verify; no change expected. |

**FORBIDDEN:** everything else. Explicitly: all pre-existing `internal/admin/*.go`
except the NEW teams/audit/usermgmt files + the OPTIONAL handlers.go additive field;
`internal/admin/auth.go` login/logout/me/status (FROZEN — no edit); all other
`internal/store/*.go` except teams/auditlog (NEW) + users/migrate (additive);
`internal/auth/*` (REUSE password.go/session.go, no edit); all UI `ui/src/**`
(FROZEN, decision 8); all other mocks/seeds/specs; `ui/package.json` + lockfile;
`ui/vite.config.ts`; `ui/playwright.config.ts`. Touching any of these is an
automatic REJECT.

---

## 4. TDD tasks

Cadence (strict, decision per AGENTS.md "TDD always"): **no Go impl file may exist
before its `_test.go` is committed RED.** `go test ./... && go vet ./... && go build
./...` green at EVERY commit (a RED test commit is allowed to fail ONLY the
new package's targeted run; commit the test in the same commit that makes the
package compile-but-fail, or guard with a skipped scaffold — prefer table tests
that fail on assertion, not compile). The two e2e specs stay green throughout
(real Go is additive; mock corrections mirror it). Three domains are independent —
order is teams → audit → usermgmt, then the single serial-slot routes commit, then
mock corrections + closeout.

### T-teams — STEP(a) RED store+admin tests, STEP(b) impl
STEP(a): write `internal/store/teams_test.go` + `internal/admin/teams_test.go`
(table-driven, `newTestEnv`); add the `teams` table to `migrate.go` (so tests
compile + the table exists). `go test ./internal/store/ -run Team` and
`go test ./internal/admin/ -run Teams` → FAIL (impl missing). Commit RED:
`phase-1/w7-gov-1: failing teams store+admin tests (TDD red)`.
STEP(b): implement `internal/store/teams.go` + `internal/admin/teams.go`. Gates:
`go test ./... && go vet ./... && go build ./...` green. Commit:
`phase-1/w7-gov-1: teams store + admin CRUD`.

### T-audit — STEP(a) RED, STEP(b) impl
STEP(a): write `internal/store/auditlog_test.go` + `internal/governance/audit_test.go`
+ `internal/admin/audit_test.go`; add the `audit_log` table + index to `migrate.go`.
Run targeted tests → FAIL. Commit RED:
`phase-1/w7-gov-1: failing audit store+domain+admin tests (TDD red)`.
STEP(b): implement `internal/store/auditlog.go`, `internal/governance/audit.go`,
`internal/admin/audit.go` (+ optional `h.audit` wiring in handlers.go). Wire the
teams handlers' `WriteAudit` calls (the teams_test audit-on-create assertion greens
here or in T-teams STEP(b) — sequence so the assertion's covering code lands with
the audit service; if teams_test asserts audit, make audit land first OR mark that
sub-assertion in T-audit). Gates green. Commit:
`phase-1/w7-gov-1: audit-log store + domain (WriteAudit) + read endpoint`.

### T-usermgmt — STEP(a) RED, STEP(b) impl
STEP(a): write `internal/store/users_test.go` additions + `internal/admin/usermgmt_test.go`;
add `ensureColumn` users.display_name/role + `ListUsers`/`DeleteUser`/`CreateUserFull`
signatures (stubs returning nil/err so tests compile-fail or assert-fail). Run
targeted → FAIL. Commit RED:
`phase-1/w7-gov-1: failing user-management store+admin tests (TDD red)`.
STEP(b): implement the store additions + `internal/admin/usermgmt.go`. Reuse
`auth.HashPassword`/`VerifyPassword`. Gates green. Commit:
`phase-1/w7-gov-1: user-management auth (setup/password/users) over user store`.

### T-routes — serial-slot route registration
TAKE the serial slot (orchestrator confirms FREE at P5). Add the 11 route lines to
`routes_admin.go` (§1.7). Gates: `go test ./... && go vet ./... && go build ./...`
green. Commit (ONE commit touches the serial file):
`phase-1/w7-gov-1: register teams/audit/user-management admin routes (serial slot)`.

### T-mocks — mock-body corrections (mirror real Go, decision 1)
Correct `teams.ts` (drop POST extras), `audit.ts` (remove POST), `auth.ts` users
routes (mirror `{data}`+`userDTO`); verify the three seeds. Gates:
`cd ui && npm run build` green; `npx playwright test e2e/teams.spec.ts
e2e/audit.spec.ts` green (still). If a correction reds a non-w7-gov-1 spec, STOP +
ESCALATE (§8 ESC-MOCK). Commit:
`phase-1/w7-gov-1: correct teams/audit/auth mocks to mirror real Go DTOs`.

### T-close — full gates + closeout
```bash
go test ./... && go vet ./... && go build ./...
go test ./internal/admin/ -run 'Teams|Audit|User' -v
go test ./internal/store/ -run 'Team|Audit|User' -v
go test ./internal/governance/ -run 'Audit' -v
cd ui && npm run build
cd ui && npx playwright test e2e/teams.spec.ts e2e/audit.spec.ts      # green
cd ui && npx playwright test                                          # full suite green (no regressions)
cd ui && npx vitest run src/                                          # unaffected, green
```
Flip `.planning/parity/matrix/9router-ui.md`: PAR-UI-132 → HAVE (real Go, cite
§1.6). Mark `open-questions.md` w6-k ESC-1a/1b/2 RESOLVED with a cite to this plan;
append any new open items (§8). Update `docs/WORKFLOW.md` (P6 base observation, the
ESC-USERMGMT setup + ESC-AUDIT-WRITE decisions, the serial-slot take-from-w7-route /
release-to-w7-gov-2, the mock corrections). Final commit:
`phase-1/w7-gov-1: close — teams/audit/user-management Go; matrix flip; mock mirror`.
**On the close commit, RELEASE the routes_admin.go serial slot to w7-gov-2.**

---

## 5. Binary acceptance criteria

All must hold; each is yes/no. `<base>` = the commit recorded at P0. Diff gate is
**w7-gov-1 commit-range-scoped** (§7).

**Test gates**
- `go test ./... && go vet ./... && go build ./...` → exit 0.
- `go test ./internal/admin/ -run 'Teams|Audit|User' -v` → exit 0, all pass
  (teams CRUD ≥6 cases; audit read ≥2; usermgmt ≥6 incl delete-last-user + no-leak).
- `go test ./internal/store/ -run 'Team|Audit|User' -v` → exit 0.
- `go test ./internal/governance/ -run 'Audit' -v` → exit 0.
- `cd ui && npx playwright test e2e/teams.spec.ts e2e/audit.spec.ts` → exit 0,
  all pass (6 teams + 3 audit), 0 skipped.
- `cd ui && npx playwright test` → exit 0, no spec green-at-base goes red.
- `cd ui && npm run build` → exit 0. `cd ui && npx vitest run src/` → exit 0.

**TDD-order proof** — each impl file's covering test appears in an earlier-or-equal
commit:
```bash
for pair in \
  "internal/store/teams_test.go:internal/store/teams.go" \
  "internal/store/auditlog_test.go:internal/store/auditlog.go" \
  "internal/governance/audit_test.go:internal/governance/audit.go" \
  "internal/admin/teams_test.go:internal/admin/teams.go" \
  "internal/admin/audit_test.go:internal/admin/audit.go" \
  "internal/admin/usermgmt_test.go:internal/admin/usermgmt.go" ; do
  tf=${pair%%:*}; im=${pair##*:}
  ct=$(git log --format=%ct --diff-filter=A -1 -- "$tf")
  cf=$(git log --format=%ct --diff-filter=A -1 -- "$im")
  [ "$ct" -le "$cf" ] || echo "TDD VIOLATION: $im"     # prints nothing
done
```

**Grep proofs (per domain)**
```bash
# teams
grep -n "func (h \*Handlers) ListTeams\|CreateTeam\|GetTeam\|UpdateTeam\|DeleteTeam" internal/admin/teams.go
grep -n "budget_usd\|budget_used_usd\|budget_period\|rate_limit_rpm" internal/admin/teams.go   # canonical 6-field DTO
grep -n "func (s \*Store) CreateTeam\|ListTeams\|GetTeamByID\|UpdateTeam\|DeleteTeam" internal/store/teams.go
grep -n "writeData\|writeError" internal/admin/teams.go                       # {data,error} envelope
! grep -n "keys_count\|members" internal/admin/teams.go && echo "no mock-only extras OK"
# audit
grep -n "func (h \*Handlers) GetAudit" internal/admin/audit.go
grep -n "items\|total" internal/admin/audit.go                                # {items,total} read shape
grep -n "func.*WriteAudit" internal/governance/audit.go
grep -n "func (s \*Store) InsertAuditEntry\|ListAuditEntries\|CountAuditEntries" internal/store/auditlog.go
# usermgmt
grep -n "func (h \*Handlers) AuthSetup\|ChangePassword\|ListUsers\|CreateUser\|DeleteUser" internal/admin/usermgmt.go
grep -n "auth.HashPassword\|auth.VerifyPassword" internal/admin/usermgmt.go   # reuses canonical hashing
grep -n "func (s \*Store) ListUsers\|DeleteUser\|CreateUserFull" internal/store/users.go
# routes
grep -nE '/api/teams|/api/audit|/api/auth/(setup|password|users)' internal/server/routes_admin.go
# no init(); no global state
! grep -rn "func init(" internal/admin/teams.go internal/admin/audit.go internal/admin/usermgmt.go internal/store/teams.go internal/store/auditlog.go internal/governance/audit.go && echo "no init() OK"
```

**No-secret-exposure proofs (binding)**
```bash
# password/hash never appear in any usermgmt DTO/response field
! grep -nE 'password|PasswordHash|password_hash' internal/admin/usermgmt.go | grep -iE 'json:"' && echo "no password json field OK"
# the userDTO struct has no password/hash field
grep -nA8 'type userDTO struct' internal/admin/usermgmt.go ; echo "^ must NOT contain password/hash"
# additive migrations only (no DROP/RENAME COLUMN introduced by this plan)
git diff <base>..HEAD -- internal/store/migrate.go | grep -E '^\+' | grep -iE 'DROP COLUMN|RENAME COLUMN|DROP TABLE' | wc -l   # = 0
# secret-at-rest: if any team/audit field were a secret it would be *_enc — none expected
grep -n "_enc" internal/store/teams.go internal/store/auditlog.go ; echo "^ expect EMPTY (no secret fields in teams/audit)"
```
Plus a runtime no-leak assertion in `usermgmt_test.go`: marshal every users-route
response and assert it contains neither the cleartext password nor the
`pbkdf2-sha256$` hash prefix.

**Negative / freeze proofs (w7-gov-1 commit-range — §7)**
```bash
R="<first-w7-gov-1>^..<last-w7-gov-1>"
# Only the sanctioned Go files changed:
git diff $R --name-only -- internal/ | grep -vE \
 'internal/store/(teams|auditlog|users|migrate)(_test)?\.go|internal/governance/(audit|teams)(_test)?\.go|internal/admin/(teams|audit|usermgmt)(_test)?\.go|internal/admin/handlers\.go|internal/server/routes_admin\.go' \
 | wc -l                                                                  # = 0
# Frozen admin handlers untouched:
git diff $R --name-only -- internal/admin/auth.go internal/admin/apikeys.go internal/admin/virtualkeys.go internal/admin/providers.go internal/admin/providers_catalog.go internal/admin/connections.go internal/admin/combos.go internal/admin/disabledmodels.go | wc -l   # = 0
# Frozen auth pkg untouched (reused, not edited):
git diff $R --name-only -- internal/auth/password.go internal/auth/session.go | wc -l   # = 0
# UI is frozen except the sanctioned mock/seed bodies:
git diff $R --name-only -- ui/ | grep -vE \
 'ui/e2e/mocks/handlers/(teams|audit|auth)\.ts|ui/e2e/mocks/seed/(teams|audit|auth)\.ts' | wc -l   # = 0
git diff $R --name-only -- ui/src/ | wc -l                               # = 0 (src frozen)
# routes_admin.go = exactly ONE commit, additive:
git log --oneline $R -- internal/server/routes_admin.go | wc -l          # = 1
git diff $R -- internal/server/routes_admin.go | grep -E '^-' | grep -v '^---' | wc -l   # = 0 (no deletions)
```

---

## 6. Out of scope (restated, binding)

No UI src edits (decision 8 — pages/components/routes/stores frozen); only the
sanctioned teams/audit/auth mock-body + seed corrections. No edits to pre-existing
admin handlers (auth.go login/logout/me/status, apikeys, virtualkeys, providers*,
connections, combos, disabledmodels, version, usage, pricing). No audit-write
retrofit into pre-existing mutations (tracked follow-up, §1.5). No JWT (opaque DB
tokens stand). No destructive DDL — additive `ensureTable`/`ensureColumn` only. No
`New(...)` signature change / no new global state. No other governance domains
(w7-gov-2/3). No secret exposure (passwords hashed/never echoed; audit details
carry no secrets). Mock-vs-Go contradiction → escalate (§8), never fudge a mock or
edit a frozen handler.

## 7. Diff-gate scope

W7 governance plans (gov-1/2/3) commit to main concurrently, so a broad
`<base>..HEAD` range sweeps in sibling commits. The diff gate MUST be scoped to
w7-gov-1's own commits. Isolate them:
`git log --oneline main | grep "^[0-9a-f]* phase-1/w7-gov-1:" | awk '{print $1}'`
then `git diff <first-w7-gov-1>^..<last-w7-gov-1> -- [file list]`.

`git diff <first>^..<last> --name-only` must be exactly a subset of:
```
internal/store/teams.go
internal/store/teams_test.go
internal/store/auditlog.go
internal/store/auditlog_test.go
internal/store/users.go                (additive methods + struct/columns)
internal/store/users_test.go
internal/store/migrate.go              (additive tables/index/columns; ONE commit per domain ok)
internal/governance/audit.go
internal/governance/audit_test.go
internal/governance/teams.go           (CONDITIONAL — only if §8 ESC-ARCH fires)
internal/governance/teams_test.go      (CONDITIONAL)
internal/admin/teams.go
internal/admin/teams_test.go
internal/admin/audit.go
internal/admin/audit_test.go
internal/admin/usermgmt.go
internal/admin/usermgmt_test.go
internal/admin/handlers.go             (OPTIONAL additive h.audit field; no New() sig change)
internal/server/routes_admin.go        (serial-slot additive routes; ONE commit)
ui/e2e/mocks/handlers/teams.ts         (body only — drop POST extras)
ui/e2e/mocks/handlers/audit.ts         (body only — remove unused POST)
ui/e2e/mocks/handlers/auth.ts          (body only — users routes mirror Go; login/logout/status untouched)
ui/e2e/mocks/seed/teams.ts             (verify; correct only on divergence)
ui/e2e/mocks/seed/audit.ts             (verify)
ui/e2e/mocks/seed/auth.ts              (verify)
.planning/parity/matrix/9router-ui.md
.planning/parity/plans/open-questions.md
docs/WORKFLOW.md
```
Any file outside this list in the scoped diff is an automatic review REJECT.
`internal/admin/auth.go`, the pre-existing admin handlers, `internal/auth/*`, and
all `ui/src/**` are deliberately ABSENT — touching them is an automatic REJECT. The
`routes_admin.go` edit must appear in exactly ONE commit (§5) and the serial slot
is released to w7-gov-2 on close.

## 8. Escalations / decisions (explicit — recommended defaults, do not fabricate)

- **ESC-USERMGMT (RESOLVED at authoring — `/api/auth/setup` semantics + User shape,
  binding default).** Two coupled questions: (a) what does `POST /api/auth/setup`
  do, and (b) the `store.User` lacks `display_name`/`role` that the mock/UI `User`
  carries. **Decision (a):** `setup` is **first-user onboarding** — succeeds ONLY
  when `CountUsers()==0` (the `SeedAdmin` invariant, `session.go:34-49`), creates
  the admin user (hashed via `HashPassword`), and auto-authenticates (issues a
  session token + sets the `g0_session` cookie, mirroring `Login` `auth.go:110-123`
  and the mock `auth.ts:88-92`); returns 409/403 when users already exist. It is
  distinct from `login` (login authenticates an EXISTING user; setup BOOTSTRAPS the
  first one). It is registered PUBLIC (no `RequireSession`) but self-guards.
  **Decision (b):** ADD additive `ensureColumn` `users.display_name` (default `''`)
  + `users.role` (default `'user'`) so the Go DTO carries the mock's fields; keep
  the existing `CreateUser(username,hash)` callers (SeedAdmin/tests) intact by
  adding `CreateUserFull`. If the operator prefers NOT to add columns (keep `User`
  minimal and DERIVE `display_name=username`, `role="admin"` in the DTO), that is a
  zero-migration alternative — but the Users panel would not persist role/display
  edits. **RECOMMENDED: add the two columns** (additive, cheap, future-proofs the
  panel). Flag for orchestrator confirmation; the plan proceeds on the recommended
  default.
- **ESC-AUDIT-WRITE (RESOLVED at authoring — write-integration scope, binding
  default).** The brief asks whether w7-gov-1 wires audit WRITES now or just ships
  read+store+helper. **Decision:** ship the **store + read endpoint + `WriteAudit`
  helper** AND wire writes for the mutations THIS PLAN owns (team create/update/
  delete; user create/delete; password change; setup) — best-effort, post-success,
  never failing the parent request, `actor` from the authenticated user. Do NOT
  retrofit `WriteAudit` into pre-existing handlers (apikeys/virtualkeys/providers/
  connections) — that is a forbidden cross-file edit; a follow-up plan wires those
  (recorded in `open-questions.md`). The mock seed shows key-lifecycle actions
  (`create_key`/`copy_key`/…) which belong to those other handlers — they are NOT
  produced by w7-gov-1's own mutations; the audit READ still serves them from any
  source, and the seed stays as illustrative data. RECOMMENDED as stated; flag for
  confirmation.
- **ESC-ARCH (CONDITIONAL — arch test on the teams layer).** The phase-12B arch
  test enforces transport→domain→repository. The existing `virtualkeys.go`/
  `apikeys.go` handlers call `h.store` DIRECTLY (no domain wrapper). If the arch
  test ALLOWS that for teams (CRUD-only), skip `governance/teams.go`. If it FORBIDS
  a new transport→store edge, ADD the thin `governance/teams.go` `TeamService`
  wrapper. Decide at T-teams by running the arch test; do NOT pre-build the wrapper.
- **ESC-ROUTE (CONDITIONAL — fasthttp/router precedence).** `/api/auth/users`
  (static) vs `/api/auth/users/{id}` and `/api/teams` vs `/api/teams/{id}` follow
  the file's existing static-before-param ordering (`routes_admin.go:56-61`
  precedent). If the matcher mis-disambiguates, STOP and ESCALATE for a path
  arrangement — never silently diverge page/mock/Go.
- **ESC-MOCK (CONDITIONAL — shared mock ripple).** `auth.ts` is shared (its
  login/logout/status branches back many specs). w7-gov-1 edits ONLY its
  setup/password/users branches. If a correction reds a non-w7-gov-1 spec, or a
  seed correction ripples, STOP and ESCALATE for orchestrator serialization — no
  fudge, no frozen-branch edit.
- **Serial-slot dependency (§1.7 / P5).** w7-gov-1 TAKES the routes_admin.go slot
  after w7-route releases it (chain MAP §219-224) and RELEASES it to w7-gov-2 on
  close. Orchestrator confirms exactly one unmerged holder (decision 3) before
  T-routes.
- **No other blocking dependency.** All reused surfaces (store/users.go,
  auth/password.go, respond.go, newTestEnv, migrate additive pattern) are in-tree
  at <base>. w7-gov-1 is unblocked once the serial slot is free.
```
