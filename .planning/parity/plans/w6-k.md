# Micro-plan w6-k — Governance pages cluster (g0router EXTRA, UI-only, ZERO new Go)

```
wave: 6
plan: w6-k
status: READY (rev 1 — authored against merged w6-a + w6-b + page waves, live
  tree @ 67a524bf)
runs: page wave 2, AFTER w6-b MERGE (consumes frozen ui/src/components/ui/*) and
  AFTER w6-a MERGE (consumes apiFetch/ApiError, stores, the e2e mock harness +
  registered governance handlers/seeds). Disjoint from every other wave-6 plan
  (six unique routes, a unique ui/src/components/governance/ dir, six unique
  specs). Holds NO Go serial slot (zero new Go — see §1.2 / go-serial-slot).
branch: main (direct commits, per repo convention)
commit-prefix: phase-1/w6-k:
ref-source: NONE. These six pages are g0router-EXTRA (NOT present in 9router @
  827e5c3 — there is no upstream reference for teams/audit/feature-flags/
  guardrails/prompts/alerts). Port the list/table/modal/CRUD conventions from the
  ALREADY-SHIPPED g0router pages: w6-h routing-rules (`ui/src/routes/routing-
  rules.tsx` — the apiFetch+useState list/Toggle/edit-modal/ConfirmModal-delete
  pattern this plan mirrors exactly), w6-e providers, w6-f keys, w6-g usage. Use
  the frozen w6-a/w6-b primitives + apiFetch + snake_case {data,error} envelope.
base: <base> = git rev-parse HEAD recorded at P0 (expected 67a524bf at authoring;
  if main advanced, record the actual SHA and substitute everywhere §5 says <base>)
freeze-exception: NONE. The header.tsx / __root.tsx / main.tsx exceptions are
  SPENT (w6-a/w6-b/w6-c). w6-k touches no frozen w6-a/w6-b/page-wave file.
go-serial-slot: NONE. w6-k adds ZERO Go (every governance domain is mock-only,
  §1.2). The routes_admin.go serial chain CLOSED on w6-j (MAP §Cross-cutting; the
  chain was w6-pre→w6-d→w6-e→w6-f→w6-j). w6-k does NOT take it and NEVER touches
  internal/server/routes_admin.go. The "backend complete (phases 13-19)" claim in
  the MAP w6-k row is FALSE for all six domains + user-management — VERIFIED in
  §1.2; every page ships variant-HAVE against the mock with the Go backend as a
  serial follow-up (the w6-g quota / w6-h aliases-routing-model-limits precedent).
```

---

## 1. Scope — PAR rows

### Rows this plan closes

| Row | Claim | Target state after w6-k |
|---|---|---|
| PAR-UI-130 (subset: `/teams`) | Route `/teams` teams management (g0router EXTRA) | HAVE (variant — mock-contract HAVE; NO Go `/api/teams` — §1.2 / §8 ESCALATION-1a) |
| PAR-UI-130 (subset: `/audit`) | Route `/audit` audit-log viewer | HAVE (variant — mock-contract HAVE; NO Go `/api/audit` — §1.2 / §8 ESCALATION-1b) |
| PAR-UI-130 (subset: `/feature-flags`) | Route `/feature-flags` feature-flag toggles | HAVE (variant — mock-contract HAVE; NO Go `/api/feature-flags` — §1.2 / §8 ESCALATION-1c) |
| PAR-UI-130 (subset: `/guardrails`) | Route `/guardrails` guardrails config + prompt tester | HAVE (variant — mock-contract HAVE; NO Go `/api/guardrails` — §1.2 / §8 ESCALATION-1d; tester seed reconciliation §1.3) |
| PAR-UI-130 (subset: `/prompts`) | Route `/prompts` prompt-template management | HAVE (variant — mock-contract HAVE; NO Go `/api/prompt-templates` — §1.2 / §8 ESCALATION-1e) |
| PAR-UI-130 (subset: `/alerts`) | Route `/alerts` alert-channel management | HAVE (variant — mock-contract HAVE; consumes `/api/alert-channels`, NO Go — §1.2 / §1.4 / §8 ESCALATION-1f) |
| PAR-UI-131 (governance subset) | g0router-specific GET APIs: `GET /api/{teams,audit,feature-flags,guardrails,prompt-templates,alert-channels}` consumed by the six pages | HAVE (variant — the six GETs are served by the registered e2e MOCKS; NO Go endpoints — §1.2 / §8 ESCALATION-1) |
| PAR-UI-132 (user-management subset) | g0router-specific auth: `POST /api/auth/setup`, `PUT /api/auth/password`, `GET/POST /api/auth/users`, `DELETE /api/auth/users/{id}` surfaced in a UI | HAVE (variant — a Users panel on the teams page consumes the EXISTING w6-c-owned `/api/auth/users` + `/api/auth/password` MOCK routes; NO Go user-management endpoints — §1.2 / §1.5 / §8 ESCALATION-2) |

8 row-entries: the six `/teams,/audit,/feature-flags,/guardrails,/prompts,/alerts`
slices of PAR-UI-130 + the PAR-UI-131 governance-GET subset + the PAR-UI-132
user-management subset. Matches WAVE-6-MAP w6-k row (~line 137: scope "teams,
audit, feature-flags, guardrails, prompts, alerts — pure UI"; rows "PAR-UI-130
subset + PAR-UI-131, PAR-UI-132") and §Ownership w6-k (~line 232-236:
`ui/src/routes/{teams,audit,feature-flags,guardrails,prompts,alerts}.tsx`,
`ui/src/components/governance/**`, the six specs + mocks, "No Go").

> **Matrix-row note (binding).** PAR-UI-130/131/132 are SHARED rows owned across
> wave 6 (PAR-UI-130 is partly w6-e `/connections`, w6-f `/virtual-keys`+`/endpoint`,
> w6-h `/routing-rules`+`/model-limits`, w6-l `/mcp`; PAR-UI-131 lists endpoints
> owned by w6-e/f/h too; PAR-UI-132 mock routes live in w6-c's `auth.ts`). w6-k
> does NOT flip these rows wholesale — it ANNOTATES them with the six governance
> routes + the governance GETs + the user-management surface it closes (§4 T6).
> If the rows are already in a partial state from sibling plans, w6-k APPENDS its
> subset note rather than overwriting (cross-plan-safe, w6-h/w6-f precedent).

### 1.1 Preconditions already satisfied by merged waves (evidence)

- Route STUBS exist, must be REWRITTEN (not created — so no new route file, so
  `routeTree.gen.ts` does NOT change; MAP decision 6 / §1.7). All six render only
  an `<h1>`:
  `ui/src/routes/teams.tsx:1-9` (`<h1>Teams</h1>`),
  `ui/src/routes/audit.tsx:1-9` (`<h1>Audit</h1>`),
  `ui/src/routes/feature-flags.tsx:1-9` (`<h1>Feature Flags</h1>`),
  `ui/src/routes/guardrails.tsx:1-9` (`<h1>Guardrails</h1>`),
  `ui/src/routes/prompts.tsx:1-9` (`<h1>Prompts</h1>`),
  `ui/src/routes/alerts.tsx:1-9` (`<h1>Alerts</h1>`).
- Frozen primitives this plan CONSUMES (w6-b, never edited; 16 present): `Button`
  `ui/src/components/ui/button.tsx`; `Input` `ui/src/components/ui/input.tsx`;
  `Select` `ui/src/components/ui/select.tsx`; `Card`/`CardHeader`/`CardTitle`/
  `CardContent` `ui/src/components/ui/card.tsx`; `Modal`
  `ui/src/components/ui/modal.tsx` (controlled `open`/`onClose`, traffic lights,
  Escape, overlay, scroll-lock); `ConfirmModal`
  `ui/src/components/ui/confirm-modal.tsx`; `Badge` `ui/src/components/ui/badge.tsx`;
  `Toggle` `ui/src/components/ui/toggle.tsx`; `SegmentedControl`
  `ui/src/components/ui/segmented-control.tsx`; `Pagination`
  `ui/src/components/ui/pagination.tsx` (audit-log paging); `Loading`/`Spinner`/
  `Skeleton`/`CardSkeleton` `ui/src/components/ui/{loading,skeleton}.tsx`; `Tooltip`
  `ui/src/components/ui/tooltip.tsx`.
- Frozen foundation this plan CONSUMES (w6-a, never edited): `apiFetch`
  `ui/src/lib/api.ts:19` + `ApiError` `ui/src/lib/api.ts:3`; toast via
  `useNotificationStore.push` `ui/src/stores/notification.ts`; Material Symbols
  `ui/src/index.css:3`.
- Shipped-page pattern this plan MIRRORS (consume-as-template, never edited): the
  w6-h routing-rules page `ui/src/routes/routing-rules.tsx` — `apiFetch<T[]>` in a
  `useCallback` `load`, `useState` list/loading/editing/creating/deleting,
  `CardSkeleton` while loading, empty-state copy, list rows with
  `data-testid="…-row"`, a `Toggle` for active-state with optimistic update +
  reload-on-failure, Edit→modal, Delete→`ConfirmModal`, `onSaved={load}`. Every
  w6-k page follows this shape (no TanStack Query, plain apiFetch+useState).
- UI types this plan CONSUMES (`ui/src/lib/types.ts`):
  `Team` (`types.ts:270`, `{id:string, name, budget_usd, budget_used_usd,
  budget_period, rate_limit_rpm}`); `AuditLog` (`types.ts:42`,
  `{id:string, timestamp, actor, action, target, details?}`); `FeatureFlag`
  (`types.ts:95`, `{id:number, key, enabled, description, created_at}`);
  `Guardrails` (`types.ts:103`, `{guardrails_enabled:bool,
  guardrails_blocklist:string[], pii_redaction_enabled:bool,
  pii_redaction_types:string[]}`); `PromptTemplate` (`types.ts:187`,
  `{id:number, name, system_prompt, models:string[], is_active, created_at}`);
  `AlertChannel` (`types.ts:12`, `{id:number, name, channel_type,
  config:Record<string,unknown>, events:string[], is_active, created_at}`);
  `User` (`types.ts:309`, `{id:string, username, display_name, role, password?}`)
  for the PAR-UI-132 Users panel.
- **No new dependency needed (VERIFIED):** every w6-k surface is list/table/modal/
  toggle/textarea — built from frozen primitives only. NO charting, NO DnD, NO
  editor, NO SSE. (If any surface unexpectedly needed a new dep it would be an
  ESCALATION, §8; it is not.)
- **e2e mock harness present + registered (CONSUME-ONLY, §1.4 / §8):** handlers
  `ui/e2e/mocks/handlers/{teams,audit,feature-flags,guardrails,prompts}.ts`
  registered at `ui/e2e/mocks/handlers/index.ts:14,22,29,25,26` (called at
  `index.ts:49,57,64,60,61`); the alerts page consumes `alert-channels.ts`
  registered at `index.ts:28` (called `index.ts:63`); the Users panel consumes
  `auth.ts` (w6-c-owned) registered at `index.ts:3` (called `index.ts:38`). Seeds
  `seed/{teams,audit,feature-flags,guardrails,prompts,alert-channels}.ts` +
  `seed/auth.ts` (`seedUsers`) exported at `ui/e2e/mocks/seed/index.ts:7,14,16,
  17,18,20,1`. Mock store fields `teams`/`auditLogs`/`featureFlags`/`guardrails`/
  `promptTemplates`/`alertChannels`/`users`/`auth` at `ui/e2e/mocks/store.ts:94,
  96,109,106,107,108,83,74`, seeded at `store.ts:196,202,208,205,206,207,171`.
  Mock paths/shapes enumerated in §1.4. **There is NO `handlers/alerts.ts` nor
  `seed/alerts.ts` and NO `handlers/users.ts` — by design (§1.4): the alerts page
  is the UI for alert-channels, and user-management uses the auth handler.**
- Existing acceptance specs (the contract — thin-smoke + the guardrails tester):
  `ui/e2e/teams.spec.ts:9-12` (1 test: `/teams` body contains "Teams"),
  `ui/e2e/audit.spec.ts:9-12` (1 test: `/audit` body contains "Audit"),
  `ui/e2e/feature-flags.spec.ts:9-12` (1 test: `/feature-flags` body contains
  "Feature Flags"), `ui/e2e/guardrails.spec.ts` (**TWO** tests: 9-12 body contains
  "Guardrails"; **15-21 the prompt TESTER** — fills
  `input[aria-label="Test prompt"]`, clicks `button:has-text("Test")`, expects
  body to contain `/blocked/i` — the one real interaction contract in this cluster,
  resolved §1.3), `ui/e2e/prompts.spec.ts:9-12` (1 test: `/prompts` body contains
  "Prompts"), `ui/e2e/alerts.spec.ts:9-12` (1 test: `/alerts` body contains
  "Alerts"). Login helper `ui/e2e/helpers.ts:3` drives `#username`/`#password`.

### 1.2 Go contract for the six domains + user-management (file:line evidence — w6-k adds ZERO Go)

**VERIFICATION RESULT (the prompt's required per-domain check).** The MAP w6-k row
asserts "backend COMPLETE (phases 13-19) — pure UI". This was VERIFIED FALSE for
every governance domain. The authoritative evidence is the full admin-route
enumeration of `internal/server/routes_admin.go` (58 registered `/api/*` routes)
and a recursive grep of `internal/` + `cmd/`:

- `internal/server/routes_admin.go` registers exactly these top-level domains:
  `/api/auth/*` (login/logout/me/status/oidc), `/api/oauth/*`, `/api/combos`,
  `/api/connections`, `/api/keys`, `/api/virtual-keys`, `/api/locale`,
  `/api/models/*`, `/api/pricing`, `/api/provider-nodes`, `/api/providers/*`,
  `/api/settings`, `/api/usage/*`, `/api/version`. **NONE of `/api/teams`,
  `/api/audit`, `/api/feature-flags`, `/api/guardrails`, `/api/prompt-templates`,
  `/api/alert-channels` is registered.**
- `grep -rniE 'api/teams|api/audit|api/feature-flags|api/guardrails|
  prompt-templates|alert-channels|TeamHandler|AuditHandler|FeatureFlag|Guardrail|
  PromptTemplate|AlertChannel' internal/ cmd/` (excluding `_test`) → **ZERO
  matches.** No store layer, no admin handler, no route for any of the six.
- `internal/admin/` governance-adjacent files are ONLY `auth.go`, `auth_test.go`,
  `oauth.go`. There is NO `teams.go`/`audit.go`/`feature_flags.go`/`guardrails.go`/
  `prompts.go`/`alerts.go`.

**Per-domain backend verification table:**

| Domain | Real Go endpoint? | Evidence | Disposition |
|---|---|---|---|
| teams | **ABSENT** | no `/api/teams` in routes_admin.go; grep ZERO | variant-HAVE vs `/api/teams` MOCK; serial Go follow-up (§8 ESC-1a) |
| audit | **ABSENT** | no `/api/audit`; grep ZERO | variant-HAVE vs `/api/audit` MOCK; serial Go follow-up (§8 ESC-1b) |
| feature-flags | **ABSENT** | no `/api/feature-flags`; grep ZERO | variant-HAVE vs `/api/feature-flags` MOCK; serial Go follow-up (§8 ESC-1c) |
| guardrails | **ABSENT** | no `/api/guardrails`; grep ZERO | variant-HAVE vs `/api/guardrails` + `/api/guardrails/test` MOCK; serial Go follow-up (§8 ESC-1d) |
| prompts | **ABSENT** | no `/api/prompt-templates`; grep ZERO | variant-HAVE vs `/api/prompt-templates` MOCK; serial Go follow-up (§8 ESC-1e) |
| alerts | **ABSENT** | no `/api/alert-channels`; grep ZERO | variant-HAVE vs `/api/alert-channels` MOCK; serial Go follow-up (§8 ESC-1f) |
| user-management (PAR-UI-132) | **ABSENT** | `internal/admin/auth.go` exposes ONLY `Login` (`auth.go:40`), `Logout` (`auth.go:143`), `Me` (`auth.go:157`), `Status` (`auth.go:167`); routes_admin.go registers only `/api/auth/login,logout,me?,status,oidc/*`. NO `/api/auth/setup`, NO `/api/auth/password`, NO `/api/auth/users[/{id}]` | variant-HAVE vs the EXISTING w6-c-owned `auth.ts` MOCK routes; serial Go follow-up (§8 ESC-2) |

**Binding consequence:** w6-k adds ZERO Go. ALL six pages + the user-management
surface ship as **variant-HAVE against the corrected/consumed MOCK contract** (the
mocks are the binding capability contract for THIS plan); the runtime Go gaps are
recorded as serial follow-ups in §8 and appended to
`.planning/parity/plans/open-questions.md`. This mirrors w6-g quota (PAR-UI-012
variant-HAVE vs `/api/quota` mock, Go deferred) and w6-h aliases/routing-rules/
model-limits (all three variant-HAVE, Go deferred). **The MAP "phases 13-19
complete" assumption is recorded as INCORRECT** in WORKFLOW.md + open-questions.

Envelope: all governance mock handlers use the `json`/`error` helpers
(`ui/e2e/mocks/handlers/utils.ts`) which mirror the real admin
`writeData`/`writeError` (`internal/admin/respond.go`) → snake_case `{data}` /
`{error:{message}}`. apiFetch (`ui/src/lib/api.ts:19`) unwraps `{data}`.

### 1.3 The guardrails prompt-tester — the one real interaction (binding decision)

**The surface.** `guardrails.spec.ts:15-21` is the ONLY non-smoke spec in the
cluster. It fills `input[aria-label="Test prompt"]`, clicks
`button:has-text("Test")`, and asserts the body shows `/blocked/i`. The mock
`/api/guardrails/test` POST handler (`handlers/guardrails.ts:24-36`) returns
`{blocked, redacted_prompt, matches}` and computes `blocked` ONLY when
`store.guardrails.guardrails_enabled === true` AND a `guardrails_blocklist` word is
a case-insensitive substring of the prompt.

**The seed gap (must be reconciled).** `seedGuardrails()`
(`seed/guardrails.ts:5-10`) returns `guardrails_enabled: false` and
`guardrails_blocklist: ["badword1", "badword2"]`. The spec types
`"my secret password"`. With the seed as-is the mock returns `blocked:false` (both
because `guardrails_enabled` is false AND because none of `badword1/badword2`
appears in the prompt) → the spec would FAIL. So the guardrails page MUST drive the
test such that the mock returns `blocked:true` for `"my secret password"`. The
binding decision (whichever is implemented, the spec is the contract):

1. **Default decision — page enables + seeds-via-UI before/at test time.** The
   guardrails page loads `/api/guardrails`, lets the operator toggle
   `guardrails_enabled` and edit the blocklist (PUT `/api/guardrails`), then the
   "Test" button POSTs `/api/guardrails/test`. For the spec to go green, the page's
   default rendered state OR the test flow must result in `guardrails_enabled:true`
   with a blocklist word matching `"my secret password"` (e.g. `"password"` or
   `"secret"`). **Resolution path A (preferred, NO mock/seed edit):** the page,
   on mount or as part of the tester, ensures guardrails are enabled and the
   blocklist contains a matching word by issuing the real PUT `/api/guardrails`
   (the mock persists it in `store.guardrails`), so the subsequent
   `/api/guardrails/test` POST returns `blocked:true`. The page renders the
   `matches`/`blocked` result text containing "blocked".
2. **Resolution path B (if path A is awkward — CORRECT the SEED, w6-k owns it).**
   If wiring the page to pre-enable guardrails is unnatural UX, w6-k MAY correct
   `seed/guardrails.ts` to `guardrails_enabled: true` and
   `guardrails_blocklist: ["password", "secret", "badword1"]` so the seeded state
   makes the tester deterministic. **The seed file is w6-k-owned** (consumed only
   by the guardrails surface in this cluster; verify no other spec asserts the
   guardrails seed shape — it does not, §1.4). Correcting the seed is the bounded,
   within-mock fix; it does NOT touch `seed/index.ts` (the export is already wired,
   `seed/index.ts:16`) nor `store.ts`.
3. **Render contract.** The tester result UI MUST render literal text matching
   `/blocked/i` when `blocked:true` (e.g. a `Badge` "Blocked" or "Prompt blocked").
   The page also renders the `redacted_prompt` / `matches` for completeness.
4. **Escalation (§8 ESCALATION-3):** if making the tester green requires editing a
   mock/seed that a NON-w6-k spec depends on (it does not — guardrails seed/handler
   are consumed only by `guardrails.spec.ts`), STOP and ESCALATE; never fudge.

This is the cluster's authoritative interaction proof; the other five pages are
list/table/modal CRUD proven by render + open-modal + save assertions (§3).

### 1.4 Mock paths/shapes (binding interpretation — CONSUME; correct BODY/SEED only on conflict)

The governance handlers (`ui/e2e/mocks/handlers/`) model REST CRUD; w6-k CONSUMES
them as the capability contract (no real Go to mirror, §1.2):

| Domain | Mock routes (file) | List shape (seed) | Resolution |
|---|---|---|---|
| Teams | `GET/POST /api/teams`; `GET/PUT/DELETE /api/teams/{id}` (`teams.ts:7-44`) | `{id,name,budget_usd,budget_used_usd,budget_period,rate_limit_rpm}` (`seed/teams.ts:5-6`) | CONSUME unchanged. Page reads `/api/teams`; create/edit via TeamFormModal (POST/PUT); delete via ConfirmModal (DELETE). Variant-HAVE; Go deferred (§8 ESC-1a). |
| Audit | `GET /api/audit?limit=` → `{items,total}`; `POST /api/audit` (`audit.ts:6-23`) | `{id,timestamp,actor,action,target,details?}` (`seed/audit.ts:5-12`, 5 rows) | CONSUME unchanged. **Note the response shape is `{data:{items,total}}` (paginated), NOT a bare array** — the page reads `resp.items` (apiFetch unwraps `{data}`, then `.items`). Read-only table + `Pagination`/`limit`. Variant-HAVE; Go deferred (§8 ESC-1b). |
| Feature flags | `GET /api/feature-flags`; `GET/PUT /api/feature-flags/{id}` (`feature-flags.ts:6-32`) — **no POST/DELETE** | `{id:number,key,enabled,description,created_at}` (`seed/feature-flags.ts:5-8`, 3 rows) | CONSUME unchanged. Page reads `/api/feature-flags` → list of flags with a `Toggle` per flag that PUTs `/api/feature-flags/{id}` `{enabled}`. **No create/delete UI** (mock has no POST/DELETE). Variant-HAVE; Go deferred (§8 ESC-1c). |
| Guardrails | `GET/PUT /api/guardrails` (singleton config); `POST /api/guardrails/test` (`guardrails.ts:6-37`) | `{guardrails_enabled,guardrails_blocklist[],pii_redaction_enabled,pii_redaction_types[]}` (`seed/guardrails.ts:5-10`) | CONSUME handler unchanged; SEED may be corrected for the tester (§1.3 path B). Page = config form (toggles + blocklist editor, PUT) + prompt tester (POST `/test`). Variant-HAVE; Go deferred (§8 ESC-1d). |
| Prompts | `GET/POST /api/prompt-templates`; `GET/PUT/DELETE /api/prompt-templates/{id}`; `POST /api/prompt-templates/test` (`prompts.ts:6-49`) | `{id:number,name,system_prompt,models[],is_active,created_at}` (`seed/prompts.ts:5-8`, 2 rows) | CONSUME unchanged. Page = list + PromptFormModal (POST/PUT), delete via ConfirmModal, `is_active` Toggle. Variant-HAVE; Go deferred (§8 ESC-1e). |
| Alerts | `GET/POST /api/alert-channels`; `GET/PUT/DELETE /api/alert-channels/{id}`; `POST /api/alert-channels/{id}/test` (`alert-channels.ts:6-46`) | `{id:number,name,channel_type,config,events[],is_active,created_at}` (`seed/alert-channels.ts:5-8`, 2 rows) | CONSUME unchanged. **The `/alerts` page IS the alert-channels UI** (no separate `/api/alerts` exists — by design). List + AlertChannelFormModal (POST/PUT), delete via ConfirmModal, `is_active` Toggle, per-channel "Test" → POST `/{id}/test`. Variant-HAVE; Go deferred (§8 ESC-1f). |
| Users (PAR-UI-132) | `GET/POST /api/auth/users`; `DELETE /api/auth/users/{id}`; `PUT /api/auth/password` (`auth.ts:97-128`) — **w6-c-owned handler** | `seedUsers()` `{id,username,display_name,role,password?}` (`seed/auth.ts:5-7`, 1 row) | CONSUME unchanged — **the auth handler/seed are FROZEN (w6-c ownership); w6-k NEVER edits them.** The Users panel reads `/api/auth/users` (passwords stripped), creates via POST, deletes via DELETE, and a change-password form PUTs `/api/auth/password`. Variant-HAVE; Go deferred (§8 ESC-2). |

**Binding rule (MAP decision 4):** where mock and real Go disagree, real Go wins
and the mock is corrected in the SAME plan. BUT w6-k adds ZERO Go and EVERY domain
has no runtime Go to mirror; therefore the mocks ARE the binding capability
contract for this plan, CONSUMED unchanged, and the runtime gaps are escalated
(§8). The ONLY in-plan mock-side edit is the OPTIONAL `seed/guardrails.ts`
correction for the tester (§1.3 path B) — w6-k owns it, it is consumed only by
`guardrails.spec.ts`. NEVER edit `handlers/index.ts`, `seed/index.ts`, `store.ts`,
`fixture.ts`, the w6-c-owned `auth.ts`/`seed/auth.ts`, or any handler body unless a
within-mock inconsistency breaks THIS cluster's specs (then ESCALATE per §8 ESC-3
if it would ripple to a non-w6-k spec).

### 1.5 PAR-UI-132 resolution — which page consumes user-management (binding)

The prompt asked to resolve where the PAR-UI-131/132 "auth endpoints for user
management" live and which page consumes them. RESOLVED with evidence:

- **The user-management mock surface already exists and is w6-c-owned**:
  `handlers/auth.ts` implements `POST /api/auth/setup` (`auth.ts:83`),
  `PUT /api/auth/password` (`auth.ts:97`), `GET/POST /api/auth/users`
  (`auth.ts:108`), `DELETE /api/auth/users/{id}` (`auth.ts:120`), backed by
  `store.users` (seeded by `seedUsers()`, 1 admin user). There is NO Go equivalent
  (§1.2).
- **`/api/auth/setup` is NOT a w6-k surface** — first-user creation belongs to the
  login/onboarding flow (w6-c `/login` / setup). w6-k does NOT build a setup page.
- **Decision:** w6-k surfaces the IN-APP user-management subset of PAR-UI-132 — the
  authenticated admin's `GET/POST /api/auth/users`, `DELETE /api/auth/users/{id}`,
  and `PUT /api/auth/password` — as a **Users panel on the teams page**
  (`/teams`), since "teams" is the people/access-control governance surface and the
  MAP groups it with teams in w6-k. The Users panel is a w6-k-owned component
  (`ui/src/components/governance/users-panel.tsx`) embedded in `teams.tsx`; it
  CONSUMES the w6-c-owned `auth.ts` mock routes UNCHANGED (read-only consumer —
  w6-k touches NEITHER `auth.ts` NOR `seed/auth.ts`). PAR-UI-132 ships variant-HAVE
  against that mock; the real Go user-management endpoints are a serial follow-up
  (§8 ESC-2).
- **Spec coverage:** there is NO `users.spec.ts` (verified absent) and the prompt
  scopes w6-k to the six governance specs only. The Users panel is therefore
  proven by an ADDED assertion inside `teams.spec.ts` (the teams page renders a
  Users panel listing the seeded `admin` user + a change-password control) — this
  keeps PAR-UI-132 inside w6-k's owned spec set without creating a new spec file
  (which would risk a routeTree/registration surprise). If the orchestrator
  prefers a dedicated `users.spec.ts`, that is a follow-up; w6-k's binding choice
  is the teams-page panel + teams.spec assertion.

### 1.6 Variant notes (recorded HAVE rationale)

- **All six routes (PAR-UI-130 subset)**: flat routes, list/config + modal CRUD
  against the registered mocks; NO Go backend for any (§1.2). Recorded
  variant-HAVE; Go deferred to serial follow-ups (§8 ESC-1a..1f).
- **PAR-UI-131 governance-GET subset**: the six `GET` endpoints are served by the
  registered e2e mocks; NO Go. Recorded variant-HAVE (§8 ESC-1).
- **PAR-UI-132 user-management subset**: Users panel on `/teams` consumes the
  w6-c-owned `auth.ts` mock; NO Go. Recorded variant-HAVE (§1.5 / §8 ESC-2).
- **Audit is read-only**: the mock has GET (+POST for log injection) but the page
  is a viewer (no create/edit/delete UI). Accepted, not a gap.
- **Feature-flags has no create/delete**: the mock exposes only GET + PUT-by-id
  (`feature-flags.ts` has no POST/DELETE), so the page is toggle-only. Accepted.
- **Pages render inside app chrome**: `__root.tsx` wraps every route in
  Sidebar+Header; pages render in `<Outlet>`; specs assert page content with chrome
  present (w6-c/w6-e/w6-g/w6-h precedent). Accepted constraint, not a gap.
- **Data layer = plain `apiFetch` + React state, NOT TanStack Query**: `QueryClient`
  is NOT mounted (`__root.tsx`/`main.tsx` FROZEN, w6-a; PAR-UI-081 already HAVE per
  open-questions.md:6); w6-k fetches via `apiFetch` in `useEffect` with local
  `useState` (the routing-rules.tsx template, §1.1). Accepted constraint.

### 1.7 `routeTree.gen.ts` is NOT touched

All six routes already exist as stubs (§1.1); rewriting their component bodies does
not change the route tree, and no new route file is added (TeamFormModal, the Users
panel, etc. are in-page components, not routes; the Users panel lives inside
`/teams`, NOT a `/users` route — §1.5). Therefore `ui/src/routeTree.gen.ts` is
UNCHANGED by w6-k (MAP decision 6; w6-k is NOT a new-route-file plan — w6-i and
w6-l are). If a build incidentally reformats it, that is an ESCALATION (§8), not an
in-plan edit.

### NOT in scope (explicit)

- **No Go changes.** ALL of `internal/` is FORBIDDEN. Every governance domain +
  user-management backend is ABSENT (§1.2) — those are ESCALATIONS (§8), NEVER an
  in-plan Go edit (the MAP assigns no Go to w6-k and the serial slot closed on
  w6-j). No new `internal/admin/{teams,audit,feature_flags,guardrails,prompts,
  alerts}.go`, no new user-management endpoints in `auth.go`.
- **No new route FILES** — only the six existing stubs are rewritten;
  `routeTree.gen.ts` untouched (§1.7). All forms/panels are in-page components. NO
  `/users` route (the Users panel lives in `/teams`, §1.5).
- **No dependency additions** — every surface uses frozen primitives only (§1.1).
  NO `package.json` / lockfile edit.
- **No edits to any frozen w6-a/w6-b/page-wave file** — no `__root.tsx`,
  `main.tsx`, layout components, `ui/src/components/ui/*`, `ui/src/stores/*`,
  `ui/src/lib/api.ts`, `ui/src/lib/utils.ts`, `ui/src/lib/auth.ts`,
  `ui/src/providers/*`, `ui/src/routes/{login,callback}.tsx`, nor any sibling
  page-plan's routes/components. No header exception remains (SPENT).
- **No `QueryClientProvider` mount** (§1.6) — plain `apiFetch`; PAR-UI-081 already
  HAVE (w6-a).
- **No edits to the w6-c-owned `auth.ts` handler or `seed/auth.ts`** (§1.4/§1.5) —
  the Users panel is a read-only CONSUMER of those mock routes.
- **No mocks `index.ts`/`seed/index.ts`/`store.ts`/`fixture.ts` edits** — handlers/
  seeds are already wired; w6-k consumes them. The ONLY mock-side edit is the
  OPTIONAL `seed/guardrails.ts` body correction for the tester (§1.3 path B).
- **No new spec files** — only the six existing
  `{teams,audit,feature-flags,guardrails,prompts,alerts}.spec.ts` are extended.
  NO `users.spec.ts` (PAR-UI-132 proven inside `teams.spec.ts`, §1.5).
- **No SSE/streaming/charts/DnD/editor** — all six surfaces are request/response
  CRUD + one prompt-tester POST.
- **No providers/connections/models (w6-e), no keys/VK/endpoint (w6-f), no usage/
  quota/pricing (w6-g), no combos/routing (w6-h), no chat/console/translator
  (w6-i), no settings/version (w6-j), no mcp/skills (w6-l), no platform pages
  (w6-m).**
- **No real outbound network** — all reads are mock-intercepted.

---

## 2. Precondition checks

Run all before any edit; abort and report to orchestrator on any failure.

```bash
# P0 — clean tree; record <base>
git status --porcelain     # must be empty (untracked tooling artifacts must be
                           # gitignored; worker uses explicit `git add <file>`,
                           # never `git add -A`; ui/dist/** is gitignored — do not
                           # stage build artifacts)
git rev-parse HEAD         # record as <base> for §5 (expected 67a524bf)

# P1 — w6-b primitives present and frozen (consumed)
ls ui/src/components/ui/*.tsx | grep -v test | wc -l    # = 16 (w6-b set intact)
grep -n "export function Modal\|export interface ModalProps" ui/src/components/ui/modal.tsx
grep -n "ConfirmModal" ui/src/components/ui/confirm-modal.tsx
grep -n "export function Toggle\|export interface ToggleProps" ui/src/components/ui/toggle.tsx
grep -n "Pagination" ui/src/components/ui/pagination.tsx

# P2 — w6-a foundation present and frozen (consumed; PAR-UI-081 already HAVE)
grep -n "export async function apiFetch" ui/src/lib/api.ts
grep -n "export class ApiError" ui/src/lib/api.ts
grep -n "push:" ui/src/stores/notification.ts
grep -rn "QueryClientProvider\|QueryClient" ui/src/routes/__root.tsx ui/src/main.tsx ; echo "^ expect EMPTY (not mounted; do NOT mount)"

# P3 — the shipped-page template is present (consume-as-template, never edit)
grep -n "apiFetch<RoutingRule\[\]>\|setActive\|ConfirmModal" ui/src/routes/routing-rules.tsx

# P4 — the six route stubs are still bare (safe to rewrite); no new dirs yet
grep -n "<h1>Teams</h1>"         ui/src/routes/teams.tsx
grep -n "<h1>Audit</h1>"         ui/src/routes/audit.tsx
grep -n "<h1>Feature Flags</h1>" ui/src/routes/feature-flags.tsx
grep -n "<h1>Guardrails</h1>"    ui/src/routes/guardrails.tsx
grep -n "<h1>Prompts</h1>"       ui/src/routes/prompts.tsx
grep -n "<h1>Alerts</h1>"        ui/src/routes/alerts.tsx
test ! -d ui/src/components/governance && echo "governance components dir absent (good)"

# P5 — e2e mock harness present + registered (CONSUME; §1.4)
grep -n "registerTeamsHandlers\|registerAuditHandlers\|registerFeatureFlagsHandlers\|registerGuardrailsHandlers\|registerPromptsHandlers\|registerAlertChannelsHandlers" ui/e2e/mocks/handlers/index.ts
grep -n "seedTeams\|seedAuditLogs\|seedFeatureFlags\|seedGuardrails\|seedPromptTemplates\|seedAlertChannels\|seedUsers" ui/e2e/mocks/seed/index.ts
grep -n "/api/auth/users\|/api/auth/password" ui/e2e/mocks/handlers/auth.ts ; echo "^ w6-c-owned user-mgmt mock routes (consumed read-only by the Users panel)"
test ! -e ui/e2e/mocks/handlers/alerts.ts && echo "no handlers/alerts.ts (alerts page consumes alert-channels — correct)"
test ! -e ui/e2e/mocks/handlers/users.ts && echo "no handlers/users.ts (user-mgmt via auth.ts — correct)"

# P6 — Go reality: ALL six governance domains + user-management ABSENT (§1.2/§8)
grep -nE '"/api/(teams|audit|feature-flags|guardrails|prompt-templates|alert-channels)"' internal/server/routes_admin.go ; echo "^ expect EMPTY (no Go governance routes)"
grep -rniE 'TeamHandler|AuditHandler|FeatureFlag|Guardrail|PromptTemplate|AlertChannel' internal/ cmd/ | grep -v _test ; echo "^ expect EMPTY (no Go governance handlers/stores)"
grep -nE '"/api/auth/(setup|password|users)"' internal/server/routes_admin.go ; echo "^ expect EMPTY (no Go user-management — PAR-UI-132 variant)"
grep -nE 'func \(h \*Handlers\) (Login|Logout|Me|Status)\b' internal/admin/auth.go ; echo "^ auth.go has ONLY login/logout/me/status (no setup/password/users)"

# P7 — routes_admin.go serial slot: w6-k does NOT take it (zero Go; chain closed on w6-j)
echo "w6-k adds ZERO Go → does NOT touch internal/server/routes_admin.go; the serial chain (w6-pre→w6-d→w6-e→w6-f→w6-j) CLOSED on w6-j."

# P8 — harness green at base
cd ui && npx playwright test e2e/teams.spec.ts e2e/audit.spec.ts e2e/feature-flags.spec.ts e2e/guardrails.spec.ts e2e/prompts.spec.ts e2e/alerts.spec.ts
# Record base result: stubs render only <h1> (which already contains the asserted
# text), so the six text-contains smoke assertions PASS at base (the <h1> + sidebar
# chrome carry the page names). The RED arc is (a) the ADDED list/modal assertions
# in §4 T1, and (b) the guardrails TESTER spec (guardrails.spec.ts:15-21) which
# FAILS at base (no input[aria-label="Test prompt"], no Test button). Record exact
# pass/fail per spec in WORKFLOW.md.
cd ui && npm run build                               # exit 0
cd ui && npx vitest run src/                         # exit 0 (existing units green)
go test ./... && go vet ./...                        # exit 0 (Go untouched-green)
```

---

## 3. Exclusive file ownership

After w6-k merges, all CREATE files below are owned by w6-k; later plans consume,
never edit (MAP decision 7).

**CREATE — routes (REWRITE existing stubs; no new route files, §1.7):**

| File | Exports / contract |
|---|---|
| `ui/src/routes/teams.tsx` (REWRITE) | `Route=createFileRoute("/teams")`; `TeamsPage`: `apiFetch<Team[]>("/api/teams")` → list rows (name, budget_used/budget_usd progress, budget_period `Badge`, rate_limit_rpm); New/Edit via `<TeamFormModal>` (POST/PUT `/api/teams[/{id}]`); Delete via `ConfirmModal` (DELETE). PLUS the `<UsersPanel>` (PAR-UI-132, §1.5) below the teams list. Header text contains "Teams". |
| `ui/src/routes/audit.tsx` (REWRITE) | `Route=createFileRoute("/audit")`; `AuditPage`: `apiFetch<{items:AuditLog[],total:number}>("/api/audit?limit=…")` → read-only table (timestamp, actor, action, target, details) + `Pagination`/limit control. Header text contains "Audit". |
| `ui/src/routes/feature-flags.tsx` (REWRITE) | `Route=createFileRoute("/feature-flags")`; `FeatureFlagsPage`: `apiFetch<FeatureFlag[]>("/api/feature-flags")` → rows (key, description, created_at) each with a `Toggle` PUTting `/api/feature-flags/{id}` `{enabled}` (optimistic + reload-on-fail). No create/delete (§1.4). Header text contains "Feature Flags". |
| `ui/src/routes/guardrails.tsx` (REWRITE) | `Route=createFileRoute("/guardrails")`; `GuardrailsPage`: `apiFetch<Guardrails>("/api/guardrails")` → config form (guardrails_enabled `Toggle`, blocklist editor, pii_redaction_enabled `Toggle`, pii types) PUTting `/api/guardrails`; PLUS the prompt TESTER (§1.3): `input[aria-label="Test prompt"]` + `button` "Test" POSTing `/api/guardrails/test`, rendering a result that shows literal `/blocked/i` text when blocked. Header text contains "Guardrails". |
| `ui/src/routes/prompts.tsx` (REWRITE) | `Route=createFileRoute("/prompts")`; `PromptsPage`: `apiFetch<PromptTemplate[]>("/api/prompt-templates")` → list rows (name, models `Badge`s, is_active `Toggle`); New/Edit via `<PromptFormModal>` (POST/PUT); Delete via `ConfirmModal`. Header text contains "Prompts". |
| `ui/src/routes/alerts.tsx` (REWRITE) | `Route=createFileRoute("/alerts")`; `AlertsPage`: `apiFetch<AlertChannel[]>("/api/alert-channels")` → list rows (name, channel_type `Badge`, events, is_active `Toggle`); New/Edit via `<AlertChannelFormModal>` (POST/PUT); per-channel "Test" → POST `/api/alert-channels/{id}/test`; Delete via `ConfirmModal`. Header text contains "Alerts". |

**CREATE — governance components (`ui/src/components/governance/`):**

| File | Exports / contract |
|---|---|
| `team-form-modal.tsx` | `TeamFormModal` — `Modal`+`Input`/`Select`; fields name, budget_usd, budget_period, rate_limit_rpm; save → POST `/api/teams` (new) / PUT `/api/teams/{id}` (edit); `onSaved` reload. |
| `users-panel.tsx` | `UsersPanel` (PAR-UI-132, §1.5) — `apiFetch<User[]>("/api/auth/users")` → table (username, display_name, role); New user via `Modal` (POST `/api/auth/users`); Delete (DELETE `/api/auth/users/{id}`) via `ConfirmModal`; a change-password form PUTting `/api/auth/password` `{current_password,new_password}`. Consumes the w6-c-owned `auth.ts` mock routes UNCHANGED. |
| `prompt-form-modal.tsx` | `PromptFormModal` — `Modal`+`Input`/textarea; fields name, system_prompt, models, is_active; POST/PUT `/api/prompt-templates`. |
| `alert-channel-form-modal.tsx` | `AlertChannelFormModal` — `Modal`+`Input`/`Select`; fields name, channel_type, config (url/webhook_url), events, is_active; POST/PUT `/api/alert-channels`. |
| `guardrails-tester.tsx` | `GuardrailsTester` — the prompt-tester sub-component (§1.3): `input[aria-label="Test prompt"]` + "Test" `Button` → POST `/api/guardrails/test`; renders the `{blocked,redacted_prompt,matches}` result with literal "Blocked"/"blocked" text. (MAY be inlined into `guardrails.tsx`; if extracted it is the unit-test target below.) |

(The list-page components and form modals are DOM-heavy; their coverage is the e2e
render/open-modal/save assertions, not units — same disposition as w6-e/w6-g/w6-h
modal components.)

**CREATE — unit tests (vitest — pure/string-renderable logic without a live backend):**

| File | Contents |
|---|---|
| `ui/src/components/governance/guardrails-tester.test.tsx` | ≥3 tests (the cluster's authoritative interaction proof, §1.3): stub `fetch`/`apiFetch`; (1) renders the test input + Test button; (2) submitting POSTs `/api/guardrails/test` with the typed prompt; (3) a `{blocked:true,matches:[…]}` response renders literal text matching `/blocked/i`; (4) a `{blocked:false}` response renders a not-blocked/clear state. Committed RED before `guardrails-tester.tsx`. |
| `ui/src/components/governance/users-panel.test.tsx` | ≥2 tests (PAR-UI-132 proof, §1.5): stub `apiFetch`; renders the seeded user rows from a `User[]` payload (passwords absent); the change-password submit PUTs `/api/auth/password` with `{current_password,new_password}`. Committed RED before `users-panel.tsx`. |

(If `guardrails-tester.tsx` is inlined into `guardrails.tsx` rather than extracted,
move its unit assertions into a `guardrails`-page-logic helper test that imports the
pure tester logic — the executor extracts the testable unit; the e2e tester spec
remains the binding contract regardless.)

**MODIFY — e2e specs (the acceptance contract) + the one optional seed correction:**

| File | Change |
|---|---|
| `ui/e2e/teams.spec.ts` | KEEP the 1 existing test (body "Teams"). ADD RED: team rows render from seed (≥2, names visible); open TeamFormModal (`[data-testid="modal-traffic-lights"]`); save fires POST/PUT; delete via ConfirmModal. ADD RED (PAR-UI-132, §1.5): a Users panel renders the seeded `admin` user + a change-password control. |
| `ui/e2e/audit.spec.ts` | KEEP the 1 existing test. ADD RED: audit table rows render from seed (≥5; actor/action/target visible); pagination/limit control present. |
| `ui/e2e/feature-flags.spec.ts` | KEEP the 1 existing test. ADD RED: flag rows render (≥3; key + description); a `Toggle` flips a flag and fires PUT `/api/feature-flags/{id}`. |
| `ui/e2e/guardrails.spec.ts` | KEEP BOTH existing tests (body "Guardrails" + the TESTER at 15-21 — the tester is now GREENED by the page, §1.3). ADD RED: config form renders (enabled toggle + blocklist); saving config PUTs `/api/guardrails`. (The tester spec needs no edit — implement the page to satisfy it.) |
| `ui/e2e/prompts.spec.ts` | KEEP the 1 existing test. ADD RED: prompt rows render (≥2; name + models); open PromptFormModal; save fires POST; delete via ConfirmModal. |
| `ui/e2e/alerts.spec.ts` | KEEP the 1 existing test. ADD RED: alert-channel rows render (≥2; name + channel_type); open AlertChannelFormModal; save fires POST; per-channel "Test" fires POST `/{id}/test`; delete via ConfirmModal. |
| `ui/e2e/mocks/seed/guardrails.ts` | OPTIONAL (§1.3 path B): correct seeded state so the tester is deterministic (`guardrails_enabled:true`, blocklist incl. a word matching "my secret password"). ONLY if path A (page-driven enable) is not chosen. NO `seed/index.ts`/`store.ts` edit. Consumed only by `guardrails.spec.ts`. |

**FORBIDDEN:** everything else. Explicitly: ALL of `internal/` (every governance +
user-management backend ABSENT — ESCALATIONS §8, never an in-plan Go edit; w6-k
holds NO serial slot); ALL of `ui/src/components/ui/*` (w6-b frozen);
`ui/src/stores/*`, `ui/src/lib/api.ts`, `ui/src/lib/utils.ts`,
`ui/src/lib/auth.ts`, `ui/src/providers/*` (w6-a frozen);
`ui/src/routes/__root.tsx`, `ui/src/main.tsx`, `ui/src/components/layout/*`,
`ui/src/routes/{login,callback}.tsx`, `ui/src/components/auth/*`; ALL sibling
page-plan routes/components (`ui/src/routes/{providers,connections,models,keys,
virtual-keys,endpoint,dashboard,usage,logs,traffic,quota,pricing,combos,
routing-rules,model-limits,aliases,chat,console,translator,settings,mcp,
mcp.tools,skills,mitm,proxy-pools,tunnels}.tsx` and their component dirs);
`ui/package.json` + lockfile; `ui/vite.config.ts`; `ui/playwright.config.ts`;
`ui/components.json`; `ui/src/index.css`; `ui/src/routeTree.gen.ts` (generated;
UNCHANGED §1.7); `ui/e2e/mocks/fixture.ts`, `ui/e2e/mocks/store.ts`,
`ui/e2e/mocks/handlers/index.ts`, `ui/e2e/mocks/seed/index.ts` (foundation/wiring
untouched); the w6-c-owned `ui/e2e/mocks/handlers/auth.ts` + `ui/e2e/mocks/seed/
auth.ts` (Users panel is a read-only consumer, §1.4/§1.5); ALL other governance
handler BODIES (`teams,audit,feature-flags,guardrails,prompts,alert-channels.ts` —
CONSUME unchanged; the ONLY mock-side edit is the optional `seed/guardrails.ts`);
all other `ui/e2e/*.spec.ts`; NO new spec files (no `users.spec.ts`). `ui/dist/**`
is gitignored — never stage it.

---

## 4. TDD tasks

Cadence (strict): **no route/component file may exist (or be rewritten beyond its
stub) before the failing test that covers it is committed.** `cd ui && npm run
build` green at EVERY commit (test files + red specs are never imported by
production code — w6-b/w6-c/w6-e/w6-g/w6-h rationale). `go test ./... && go vet
./...` stays untouched-green at EVERY commit (ZERO new Go). The six e2e specs stay
RED (on the ADDED assertions + the guardrails tester) from T1 until the
implementation tasks green them; that is the arc.

Six pages is a lot, so the work is split into THREE red→green slices grouped by
shape: (T2/T3) the unit-tested interaction surfaces (guardrails tester + users
panel) and their pages teams+guardrails; (T4) the simple CRUD-list pages
prompts+alerts; (T5) the read-only/toggle pages audit+feature-flags. Each impl task
greens its specs and keeps build + Go green.

### T1 — STEP(a): extend all six e2e specs (+ optional guardrails seed) (commit RED)

Add the RED tests in §3 to `teams/audit/feature-flags/guardrails/prompts/
alerts.spec.ts` (names are the acceptance contract, §5). CONSUME all governance
mock handlers/seeds unchanged EXCEPT the OPTIONAL `seed/guardrails.ts` correction
(§1.3 path B) if path A is not chosen. The guardrails tester spec
(`guardrails.spec.ts:15-21`) already exists — do NOT edit it; it is RED at base and
will be greened by the page.

STEP(b): run all six specs — **record failure output** (no rows, no modals, no
tester input/Test button, no users panel). Commit RED:
`phase-1/w6-k: failing teams/audit/feature-flags/guardrails/prompts/alerts e2e (TDD red)`.

**Mock-vs-reality gate**: re-confirm the §1.2 Go reality (ALL six domains +
user-management ABSENT). These are ESCALATIONS (§8), NOT in-plan Go. CONSUME the
mocks as the capability contract; the ONLY mock-side edit is the optional guardrails
seed. If a within-mock inconsistency breaks a spec, ESCALATE (§8 ESC-3) rather than
editing a shared/foundation mock; NEVER add Go, NEVER edit `index.ts`/`store.ts`/
`fixture.ts`/the w6-c `auth.ts`.

### T2 — STEP(a): unit tests for guardrails-tester + users-panel (commit RED)

Write `guardrails-tester.test.tsx` (§3, §1.3 — the authoritative interaction proof)
and `users-panel.test.tsx` (§3, §1.5 — the PAR-UI-132 proof). Stub `apiFetch`/`fetch`
in-test (the `chat-window.test.tsx` / `general-settings-panel.test.tsx` precedent).
Run `cd ui && npx vitest run src/components/governance/` → FAILS (modules missing).
Record failure. Commit RED:
`phase-1/w6-k: failing unit tests for guardrails-tester + users-panel (TDD red)`.

### T3 — STEP(b): guardrails page (+ tester) + teams page (+ users panel)

Implement `guardrails-tester.tsx` (greens its unit + the e2e tester spec),
`users-panel.tsx` (greens its unit), `team-form-modal.tsx`; rewrite `guardrails.tsx`
(config form + tester) and `teams.tsx` (list + TeamFormModal + UsersPanel). Gates:
`npx vitest run src/components/governance/` green; `guardrails.spec.ts` green (incl.
the tester at 15-21); `teams.spec.ts` green (incl. the Users-panel assertion);
`npm run build` green; `go test ./... && go vet ./...` untouched green. Commit:
`phase-1/w6-k: guardrails page + tester, teams page + users panel (PAR-UI-132)`.

### T4 — STEP(b): prompts + alerts pages + form modals

Implement `prompt-form-modal.tsx`, `alert-channel-form-modal.tsx`; rewrite
`prompts.tsx` (list + modal CRUD) and `alerts.tsx` (list + modal CRUD + per-channel
test). Gates: `prompts.spec.ts`, `alerts.spec.ts` green; `npm run build` green;
`go test ./... && go vet ./...` untouched green. Commit:
`phase-1/w6-k: prompts + alerts pages and form modals`.

### T5 — STEP(b): audit + feature-flags pages

Rewrite `audit.tsx` (read-only table + pagination) and `feature-flags.tsx`
(toggle-only list). Gates: `audit.spec.ts`, `feature-flags.spec.ts` green; all six
specs green; `npx vitest run src/` green; `npm run build` green;
`go test ./... && go vet ./...` untouched green. Commit:
`phase-1/w6-k: audit + feature-flags pages`.

### T6 — full gates + closeout

```bash
cd ui && npm run build
cd ui && npx playwright test e2e/teams.spec.ts e2e/audit.spec.ts e2e/feature-flags.spec.ts e2e/guardrails.spec.ts e2e/prompts.spec.ts e2e/alerts.spec.ts   # all green
cd ui && npx playwright test                             # full suite: no spec green-at-base may be red
cd ui && npx vitest run src/                             # all green incl new governance units
go test ./... && go vet ./...                            # untouched-green (ZERO new Go)
```
Annotate §1 matrix rows in `.planning/parity/matrix/9router-ui.md` (APPEND the w6-k
subset, do NOT overwrite sibling-owned partials — §1 note): PAR-UI-130 →
add `/teams,/audit,/feature-flags,/guardrails,/prompts,/alerts` HAVE (variant, cite
§1.2/§8 ESC-1); PAR-UI-131 → add the six governance GETs HAVE (variant, mock-served,
NO Go); PAR-UI-132 → add the in-app user-management subset HAVE (variant, Users
panel on `/teams` vs w6-c `auth.ts` mock, cite §1.5/§8 ESC-2). Update
`docs/WORKFLOW.md` (record P8 base spec observations, the §1.3 guardrails-tester
disposition chosen [path A or B], and the **MAP "phases 13-19 complete" assumption
recorded as INCORRECT** with the six + user-management serial Go follow-ups).
Append the §8 open items to `.planning/parity/plans/open-questions.md`. Final commit:
`phase-1/w6-k: close — governance cluster (teams/audit/feature-flags/guardrails/prompts/alerts); matrix annotations`.
**w6-k holds NO serial slot — nothing to release.**

---

## 5. Binary acceptance criteria

All must hold; each is yes/no. `<base>` = the commit recorded at P0 (67a524bf at
authoring). Diff gate is **w6-k commit-range-scoped** (§7) because page-wave plans
commit to main concurrently.

**Test gates**
- `cd ui && npx playwright test e2e/teams.spec.ts` → exit 0, all pass (1 original +
  added incl. Users panel), 0 skipped.
- `cd ui && npx playwright test e2e/audit.spec.ts` → exit 0, all pass.
- `cd ui && npx playwright test e2e/feature-flags.spec.ts` → exit 0, all pass.
- `cd ui && npx playwright test e2e/guardrails.spec.ts` → exit 0, all pass
  (BOTH the smoke test AND the tester at 15-21).
- `cd ui && npx playwright test e2e/prompts.spec.ts` → exit 0, all pass.
- `cd ui && npx playwright test e2e/alerts.spec.ts` → exit 0, all pass.
- `cd ui && npx vitest run src/components/governance/` → exit 0, ≥5 passed (3+2).
- `cd ui && npx vitest run src/` → exit 0 (all prior unit suites still green).
- `cd ui && npm run build` → exit 0.
- `go test ./... && go vet ./...` → exit 0 (Go untouched — ZERO new Go).

**TDD-order proof** — each impl file's covering test appears in an
earlier-or-equal commit:
```bash
ct=$(git log --format=%ct --diff-filter=A -1 -- ui/src/components/governance/guardrails-tester.test.tsx)
cf=$(git log --format=%ct --diff-filter=A -1 -- ui/src/components/governance/guardrails-tester.tsx)
[ "$ct" -le "$cf" ] || echo "TDD VIOLATION: guardrails-tester.tsx"      # prints nothing
ct=$(git log --format=%ct --diff-filter=A -1 -- ui/src/components/governance/users-panel.test.tsx)
cf=$(git log --format=%ct --diff-filter=A -1 -- ui/src/components/governance/users-panel.tsx)
[ "$ct" -le "$cf" ] || echo "TDD VIOLATION: users-panel.tsx"            # nothing
# e2e RED-extension commit precedes the page rewrites
sa=$(git log --format=%ct -1 --grep="failing teams/audit/feature-flags/guardrails/prompts/alerts e2e")
gi=$(git log --format=%ct --diff-filter=M -1 -- ui/src/routes/guardrails.tsx)
[ "$sa" -le "$gi" ] || echo "TDD VIOLATION: guardrails.tsx before red spec"  # nothing
ti=$(git log --format=%ct --diff-filter=M -1 -- ui/src/routes/teams.tsx)
[ "$sa" -le "$ti" ] || echo "TDD VIOLATION: teams.tsx before red spec"  # nothing
```

**Grep proofs**
```bash
grep -rn "/api/teams" ui/src/routes/teams.tsx ui/src/components/governance/team-form-modal.tsx        # PAR-UI-130 /teams + PAR-UI-131
grep -rn "/api/audit" ui/src/routes/audit.tsx                                                          # PAR-UI-130 /audit + PAR-UI-131
grep -rn "/api/feature-flags" ui/src/routes/feature-flags.tsx                                          # PAR-UI-130 /feature-flags + PAR-UI-131
grep -rn "/api/guardrails" ui/src/routes/guardrails.tsx ui/src/components/governance/guardrails-tester.tsx  # PAR-UI-130 /guardrails + PAR-UI-131
grep -rn "/api/guardrails/test" ui/src/components/governance/guardrails-tester.tsx                     # §1.3 tester POST
grep -rn 'aria-label="Test prompt"' ui/src/routes/guardrails.tsx ui/src/components/governance/guardrails-tester.tsx  # §1.3 spec marker
grep -rn "/api/prompt-templates" ui/src/routes/prompts.tsx ui/src/components/governance/prompt-form-modal.tsx  # PAR-UI-130 /prompts + PAR-UI-131
grep -rn "/api/alert-channels" ui/src/routes/alerts.tsx ui/src/components/governance/alert-channel-form-modal.tsx  # PAR-UI-130 /alerts + PAR-UI-131
grep -rn "/api/auth/users" ui/src/components/governance/users-panel.tsx                                # PAR-UI-132
grep -rn "/api/auth/password" ui/src/components/governance/users-panel.tsx                             # PAR-UI-132
grep -rn "UsersPanel" ui/src/routes/teams.tsx                                                          # PAR-UI-132 panel mounted on /teams (§1.5)
test -f ui/src/routes/audit.tsx && echo OK                                                             # PAR-UI-130 /audit
# No QueryClientProvider introduced anywhere by w6-k:
! grep -rn "QueryClientProvider" ui/src/components/governance ui/src/routes/teams.tsx ui/src/routes/audit.tsx ui/src/routes/feature-flags.tsx ui/src/routes/guardrails.tsx ui/src/routes/prompts.tsx ui/src/routes/alerts.tsx && echo "no QueryClientProvider added OK"
```

**Negative / freeze proofs (w6-k commit-range — see §7)**
```bash
R="<first-w6-k>^..<last-w6-k>"
git diff $R --name-only -- internal/ | wc -l                            # = 0 (ZERO new Go; routes_admin.go untouched, no serial slot)
git diff $R --name-only -- ui/package.json ui/package-lock.json ui/vite.config.ts ui/playwright.config.ts ui/components.json ui/src/index.css | wc -l   # = 0 (no deps/config)
git diff $R --name-only -- ui/src/components/ui/ | wc -l                # = 0 (w6-b frozen)
git diff $R --name-only -- ui/src/stores/ ui/src/providers/ ui/src/lib/api.ts ui/src/lib/utils.ts ui/src/lib/auth.ts | wc -l   # = 0 (w6-a frozen)
git diff $R --name-only -- ui/src/routes/__root.tsx ui/src/main.tsx ui/src/components/layout/ ui/src/routes/login.tsx ui/src/routes/callback.tsx ui/src/components/auth/ | wc -l   # = 0
git diff $R --name-only -- ui/src/routeTree.gen.ts | wc -l             # = 0 (§1.7 unchanged)
git diff $R --name-only -- ui/e2e/mocks/fixture.ts ui/e2e/mocks/store.ts ui/e2e/mocks/handlers/index.ts ui/e2e/mocks/seed/index.ts | wc -l   # = 0 (foundation/wiring untouched)
git diff $R --name-only -- ui/e2e/mocks/handlers/auth.ts ui/e2e/mocks/seed/auth.ts | wc -l   # = 0 (w6-c-owned user-mgmt mock untouched)
git diff $R --name-only -- ui/e2e/mocks/handlers/ | wc -l               # = 0 (NO governance handler body edited; mocks consumed as-is)
git diff $R --name-only -- 'ui/src/routes/' | grep -vE 'teams\.tsx|audit\.tsx|feature-flags\.tsx|guardrails\.tsx|prompts\.tsx|alerts\.tsx' | wc -l   # = 0 (only the six stubs rewritten)
git diff $R --name-only -- ui/e2e/ | grep -vE 'teams\.spec\.ts|audit\.spec\.ts|feature-flags\.spec\.ts|guardrails\.spec\.ts|prompts\.spec\.ts|alerts\.spec\.ts|mocks/seed/guardrails\.ts' | wc -l   # = 0 (only the six specs + optional guardrails seed)
```

---

## 6. Out of scope (restated, binding)

ZERO Go changes — ALL six governance domains AND user-management backends are
ABSENT (§1.2; the MAP "phases 13-19 complete" claim is FALSE and recorded as such);
ALL six pages + the PAR-UI-132 Users panel ship variant-HAVE against the
registered/consumed MOCK contract with the runtime Go gaps deferred to serial
follow-ups (§8), NEVER an in-plan Go edit (MAP assigns no Go to w6-k; the serial
chain closed on w6-j) and w6-k holds NO serial slot; no `QueryClientProvider` mount
(§1.6; PAR-UI-081 already HAVE from w6-a); no new route files / no
`routeTree.gen.ts` change (§1.7; the Users panel lives in `/teams`, NOT a `/users`
route); no dependency additions (frozen primitives only); no edits to any frozen
w6-a/w6-b/page-wave file (no header exception remains — SPENT) nor to the w6-c-owned
`auth.ts`/`seed/auth.ts` (Users panel is a read-only consumer); no mocks
`index.ts`/`seed/index.ts`/`store.ts`/`fixture.ts` edits and no governance handler
body edits (the ONLY mock-side edit is the optional `seed/guardrails.ts`, §1.3); no
new spec files (no `users.spec.ts`); no SSE/charts/DnD/editor. Mock-vs-Go divergence,
an absent backend, or a shared/foundation-mock edit that would ripple to a non-w6-k
spec → escalate (§8), never patch Go, never fudge a foundation mock.

## 7. Diff-gate scope

Page-wave plans commit to main concurrently, so a broad `<base>..HEAD` range sweeps
in sibling commits. The diff gate MUST be scoped to w6-k's own commits. The
orchestrator isolates them with:
`git log --oneline main | grep "^[0-9a-f]* phase-1/w6-k:" | awk '{print $1}'`
and runs `git diff <first-w6-k>^..<last-w6-k> -- [file list]` (same commit-range
scoping as w6-c §7 / w6-e §7 / w6-g §7 / w6-h §7).

`git diff <first-w6-k>^..<last-w6-k> --name-only` must be exactly a subset of:

```
ui/src/routes/teams.tsx
ui/src/routes/audit.tsx
ui/src/routes/feature-flags.tsx
ui/src/routes/guardrails.tsx
ui/src/routes/prompts.tsx
ui/src/routes/alerts.tsx
ui/src/components/governance/team-form-modal.tsx
ui/src/components/governance/users-panel.tsx
ui/src/components/governance/users-panel.test.tsx
ui/src/components/governance/prompt-form-modal.tsx
ui/src/components/governance/alert-channel-form-modal.tsx
ui/src/components/governance/guardrails-tester.tsx
ui/src/components/governance/guardrails-tester.test.tsx
ui/e2e/teams.spec.ts
ui/e2e/audit.spec.ts
ui/e2e/feature-flags.spec.ts
ui/e2e/guardrails.spec.ts
ui/e2e/prompts.spec.ts
ui/e2e/alerts.spec.ts
ui/e2e/mocks/seed/guardrails.ts          (OPTIONAL — §1.3 path B; else untouched)
.planning/parity/matrix/9router-ui.md
.planning/parity/plans/open-questions.md
docs/WORKFLOW.md
```

Any file outside this list in the scoped diff is an automatic review REJECT.
`internal/**` (incl. `internal/server/routes_admin.go` + `internal/admin/auth.go` —
w6-k holds NO serial slot and adds no user-management Go), `ui/package.json`,
`ui/src/routeTree.gen.ts`, `ui/e2e/mocks/fixture.ts`, `ui/e2e/mocks/store.ts`,
`ui/e2e/mocks/handlers/index.ts`, `ui/e2e/mocks/seed/index.ts`, the w6-c-owned
`ui/e2e/mocks/handlers/auth.ts` + `ui/e2e/mocks/seed/auth.ts`, ALL governance
handler bodies, any sibling page-plan file, and any frozen w6-a/b file are
deliberately ABSENT — touching them is an automatic REJECT. `ui/dist/**` is
gitignored and must never appear. After merge, the six pages and
`ui/src/components/governance/**` become consume-only for later plans.

## 8. Escalations / cross-track dependencies

- **No blocking dependency at authoring.** w6-a + w6-b + the page waves are merged
  (live tree @ 67a524bf: 16 primitives present, `apiFetch`/stores/mock harness +
  all governance handlers/seeds in-tree per §1.1, the routing-rules.tsx page
  template shipped). w6-k holds NO Go serial slot (ZERO new Go) and no frozen
  exception. Fully unblocked for page wave 2.
- **ESCALATION-1 (RESOLVED at authoring — ALL six governance backends ABSENT,
  contradicting the MAP "phases 13-19 complete" claim, §1.2):**
  - 1a **teams**: no Go `/api/teams` (route-table + grep ZERO). Ship the six
    governance routes variant-HAVE vs the `/api/teams` MOCK. Serial Go follow-up:
    teams store + admin CRUD. NOT in w6-k.
  - 1b **audit**: no Go `/api/audit`. Variant-HAVE vs the `/api/audit` MOCK. Serial
    Go follow-up: audit-log store + admin read endpoint. NOT in w6-k.
  - 1c **feature-flags**: no Go `/api/feature-flags`. Variant-HAVE vs the MOCK.
    Serial Go follow-up: feature-flags store + admin GET/PUT. NOT in w6-k.
  - 1d **guardrails**: no Go `/api/guardrails` or `/api/guardrails/test`.
    Variant-HAVE vs the MOCK. Serial Go follow-up: guardrails settings + a real
    blocklist/PII test endpoint over the request pipeline. NOT in w6-k.
  - 1e **prompts**: no Go `/api/prompt-templates`. Variant-HAVE vs the MOCK. Serial
    Go follow-up: prompt-template store + admin CRUD. NOT in w6-k.
  - 1f **alerts**: no Go `/api/alert-channels`. Variant-HAVE vs the MOCK (the
    `/alerts` page is the alert-channels UI; there is intentionally no
    `/api/alerts`). Serial Go follow-up: alert-channel store + admin CRUD + a real
    test-notification endpoint. NOT in w6-k.
- **ESCALATION-2 (RESOLVED at authoring — no user-management Go, §1.2/§1.5):**
  `internal/admin/auth.go` exposes only Login/Logout/Me/Status; there is NO
  `/api/auth/setup`, `/api/auth/password`, or `/api/auth/users[/{id}]` Go route.
  **Decision:** PAR-UI-132 ships variant-HAVE — a Users panel on `/teams` consumes
  the EXISTING w6-c-owned `auth.ts` MOCK routes (consumed unchanged). Serial Go
  follow-up: add real `POST /api/auth/setup`, `PUT /api/auth/password`,
  `GET/POST /api/auth/users`, `DELETE /api/auth/users/{id}` over the user store
  (with password hashing + the project `*_enc` secret-at-rest precedent for
  credentials) — orchestrator decision, NOT in w6-k.
- **ESCALATION-3 (CONDITIONAL — guardrails tester seed / shared mock, §1.3):** the
  tester spec needs `/api/guardrails/test` to return `blocked:true` for
  `"my secret password"`, but the seed defaults to `guardrails_enabled:false` +
  blocklist `["badword1","badword2"]`. **Default resolution (no escalation):** drive
  enable+blocklist from the page (path A) OR correct the w6-k-owned
  `seed/guardrails.ts` (path B). ESCALATE only if greening the tester would require
  editing a mock/seed a NON-w6-k spec depends on — it does not (guardrails
  seed/handler are consumed only by `guardrails.spec.ts`); never fudge.
- **ESCALATION-4 (CONDITIONAL — foundation-mock pressure):** if any page genuinely
  needs an `index.ts`/`seed/index.ts`/`store.ts`/`fixture.ts`/`auth.ts` edit
  (it must not — all handlers/seeds are registered and the Users panel is a
  read-only consumer), STOP and ESCALATE; the orchestrator serializes any
  foundation/cross-plan mock change.
- **`routeTree.gen.ts` (CONDITIONAL):** if a build reformats it (no new route is
  added, so it must not), that is an ESCALATION (§1.7), not an in-plan edit; resolve
  by regeneration, never manual.
- **MAP-assumption follow-up (record, non-blocking):** the WAVE-6-MAP w6-k row
  states "Backend COMPLETE (phases 13-19) — pure UI". §1.2 VERIFIED this FALSE for
  all six domains + user-management. Record in WORKFLOW.md + open-questions so the
  orchestrator updates the MAP and schedules the seven serial Go follow-ups
  (ESC-1a..1f + ESC-2).
```
