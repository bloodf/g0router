# Micro-plan w6-l — MCP + skills cluster (UI-only, ZERO new Go; THE wave-2 new-route plan)

```
wave: 6
plan: w6-l
status: READY (rev 1 — authored against merged w6-a + w6-b + page waves, live
  tree @ b23bead)
runs: page wave 2, AFTER w6-b MERGE (consumes frozen ui/src/components/ui/*) and
  AFTER w6-a MERGE (consumes apiFetch/ApiError, stores, the e2e mock harness +
  the ALREADY-REGISTERED mcp + skills handlers/seeds). Disjoint from every other
  wave-6 plan (mcp/mcp-tools/skills routes, a unique ui/src/components/{mcp,skills}/
  dir, the mcp spec + a new skills spec). Holds NO Go serial slot (zero new Go).
branch: main (direct commits, per repo convention)
commit-prefix: phase-1/w6-l:
ref-source: 9router frozen @ 827e5c3 —
  src/app/(dashboard)/dashboard/skills/page.js (skills directory: categorized
  SkillRow list + copy-to-clipboard, sourced from a SKILLS constant + repo URL),
  src/shared/components/McpMarketplaceModal.js (MCP server marketplace: browse +
  per-server tool-selection + add — ref hits /api/cli-tools/cowork-mcp-* which
  g0router LACKS, so the install flow is REMAPPED to the in-tree mcp mock, §1.6).
  There is NO 9router mcp/ or mcp-tools/ PAGE — those two pages are g0router-EXTRA
  (port the list/table/modal CRUD conventions from the SHIPPED g0router pages:
  w6-h `ui/src/routes/routing-rules.tsx` — apiFetch+useState list/Toggle/edit-modal/
  ConfirmModal-delete — exactly the pattern this plan mirrors, §1.1).
base: <base> = git rev-parse HEAD recorded at P0 (expected b23bead at authoring;
  if main advanced, record the actual SHA and substitute everywhere §5 says <base>)
freeze-exception: NONE. The header.tsx / __root.tsx / main.tsx exceptions are
  SPENT (w6-a/w6-b/w6-c). w6-l touches no frozen w6-a/w6-b/page-wave file.
go-serial-slot: NONE. w6-l adds ZERO Go. The routes_admin.go serial chain CLOSED
  on w6-j (MAP §Cross-cutting; chain was w6-pre→w6-d→w6-e→w6-j). w6-l does NOT take
  it and NEVER touches internal/server/routes_admin.go. The MAP w6-l row's "MCP
  gateway backend in-tree" claim is FALSE — VERIFIED in §1.2: `internal/mcp/` is a
  Phase-1 PLACEHOLDER package (doc.go + a no-op compile test), there are NO
  `/api/mcp/*` admin routes, NO `internal/admin/mcp.go`/`skills.go`. Every surface
  ships variant-HAVE against the registered MOCK with the Go backend as a serial
  follow-up (the w6-g quota / w6-h aliases / w6-k governance precedent).
new-route: YES — w6-l is THE wave-2 new-route plan (the wave-2 analogue of w6-i in
  wave 1). It ADDS one new route FILE (`ui/src/routes/skills.tsx`, §1.7); the
  TanStack Router Vite plugin REGENERATES `ui/src/routeTree.gen.ts` on build to
  register `/skills`. That generated change IS in w6-l's allowed diff (§3 / §7),
  unlike sibling page-wave-2 plans (w6-f/j/k/m) where routeTree.gen.ts must be
  UNCHANGED. Same handling as w6-i's `translator.tsx` (w6-i §1.7) and w6-c's
  `callback.tsx`. NEVER hand-edited. The two MCP routes (`mcp.tsx`, `mcp.tools.tsx`)
  are EXISTING stubs (already registered, §1.1) — rewriting them does NOT change the
  tree; ONLY the skills addition does.
```

---

## 1. Scope — PAR rows

### Rows this plan closes

| Row | Claim | Target state after w6-l |
|---|---|---|
| PAR-UI-020 | Route `/skills` skills directory page (NEW route file) | HAVE (variant — NEW route file; consumes the registered `/api/skills` MOCK; NO Go `/api/skills` — §1.2 / §1.5 / §8 ESCALATION-1c) |
| PAR-UI-054 | McpMarketplaceModal — MCP server marketplace browse + add/install flow | HAVE (variant — ref's cli-tools registry endpoints absent; REMAPPED to the in-tree mcp mock: browse `/api/mcp/clients`, install via `POST /api/mcp/instances`; NO Go — §1.6 / §8 ESCALATION-1a) |
| PAR-UI-130 (subset: `/mcp`) | Route `/mcp` MCP clients/instances management page | HAVE (variant — REWRITE stub; list/CRUD against the `/api/mcp/{clients,instances}` MOCK; NO Go — §1.2 / §1.4 / §8 ESCALATION-1a) |
| PAR-UI-130 (subset: `/mcp/tools`) | Route `/mcp/tools` MCP tools + tool-groups management (nested under `/mcp`) | HAVE (variant — REWRITE stub; list/execute/groups against the `/api/mcp/{tools,tool-groups}` MOCK; NO Go — §1.2 / §1.4 / §1.8 / §8 ESCALATION-1b) |

4 row-entries: PAR-UI-020 (skills) + PAR-UI-054 (McpMarketplaceModal) + the
`/mcp` and `/mcp/tools` slices of PAR-UI-130. Matches WAVE-6-MAP w6-l row (~line 138:
scope "mcp page, mcp/tools page, McpMarketplaceModal, skills page (NEW route file)";
rows "PAR-UI-020, PAR-UI-054 + PAR-UI-130 subset (`/mcp`, `/mcp/tools`)") and
§Ownership w6-l (~line 199-201: `ui/src/routes/{mcp,mcp.tools,skills(NEW)}.tsx`,
`ui/src/components/mcp/**`, `ui/e2e/mcp.spec.ts` + new `ui/e2e/skills.spec.ts` +
mocks). ALL backing services have NO Go (`internal/mcp/` is a placeholder, §1.2);
every surface ships variant-HAVE against the registered mocks with the Go backends
as serial follow-ups (§8).

> **Matrix-row note (binding).** PAR-UI-130 is a SHARED row owned across wave 6
> (PAR-UI-130 is partly w6-e `/connections`, w6-f `/virtual-keys`+`/endpoint`,
> w6-h `/routing-rules`+`/model-limits`, w6-k six governance routes). w6-l does NOT
> flip PAR-UI-130 wholesale — it ANNOTATES it with the `/mcp` + `/mcp/tools` routes
> it closes (§4 T5). If the row is already partial from sibling plans, w6-l APPENDS
> its subset note rather than overwriting (cross-plan-safe, w6-h/w6-k precedent).
> PAR-UI-020 + PAR-UI-054 are w6-l-exclusive (no sibling owns them).

### 1.1 Preconditions already satisfied by merged waves (evidence)

- **Route STUBS that must be REWRITTEN** (not created — already registered, so
  rewriting them does NOT change the route tree): both render only an `<h1>`:
  `ui/src/routes/mcp.tsx:1-9` (`createFileRoute("/mcp")`, `<h1>MCP</h1>`),
  `ui/src/routes/mcp.tools.tsx:1-9` (`createFileRoute("/mcp/tools")`,
  `<h1>MCP Tools</h1>`). The flat-file name `mcp.tools.tsx` registers the NESTED
  route `/mcp/tools` UNDER `/mcp` (TanStack flat-route dotted naming, §1.8) —
  confirmed in `routeTree.gen.ts` (`McpToolsRoute` imported from `./routes/mcp.tools`
  `routeTree.gen.ts:44`, `getParentRoute: () => McpRoute` `routeTree.gen.ts:209`,
  `McpRouteChildren { McpToolsRoute }` `routeTree.gen.ts:697-705`). **BOTH are
  already in the route tree** (`'/mcp'`/`McpRoute` `routeTree.gen.ts:126-128,575-579`;
  `'/mcp/tools'`/`McpToolsRoute` `routeTree.gen.ts:206-209,687-692`).
- **NEW route file to CREATE** (§1.7): `ui/src/routes/skills.tsx` does NOT exist
  (`test ! -e ui/src/routes/skills.tsx` → true at authoring). Adding it is what
  triggers the `routeTree.gen.ts` regeneration. `routeTree.gen.ts` has NO `/skills`
  entry today (`grep -n 'skills\|SkillsRoute' ui/src/routeTree.gen.ts` → EMPTY).
- Frozen primitives this plan CONSUMES (w6-b, never edited; 16 present): `Button`
  `ui/src/components/ui/button.tsx`; `Input` `ui/src/components/ui/input.tsx`;
  `Select` `ui/src/components/ui/select.tsx`; `Card`/`CardHeader`/`CardTitle`/
  `CardContent` (`ui/src/components/ui/card.tsx:70`); `Modal`
  (`ui/src/components/ui/modal.tsx:73`, controlled `open`/`onClose`, traffic lights,
  Escape, overlay, scroll-lock — the McpMarketplaceModal + form-modal base);
  `ConfirmModal` (`ui/src/components/ui/confirm-modal.tsx:46`); `Badge`
  `ui/src/components/ui/badge.tsx` (transport / health-status / category badges);
  `Toggle` `ui/src/components/ui/toggle.tsx` (is_active toggles);
  `SegmentedControl` `ui/src/components/ui/segmented-control.tsx` (marketplace
  filter all/oauth/authless); `Loading`/`Spinner`/`Skeleton`/`CardSkeleton`
  `ui/src/components/ui/{loading,skeleton}.tsx`; `Tooltip`
  `ui/src/components/ui/tooltip.tsx`. (`Pagination`/`ProviderIcon` available but
  likely unused by this cluster.)
- Frozen foundation this plan CONSUMES (w6-a, never edited): `apiFetch`
  (`ui/src/lib/api.ts:19`) + `ApiError` (`ui/src/lib/api.ts:3`); toast via
  `useNotificationStore.push` (`ui/src/stores/notification.ts:11,22`); Material
  Symbols (`ui/src/index.css:3`, `@import "material-symbols/outlined.css"` — the
  skills copy-button icon, §1.3).
- Shipped-page pattern this plan MIRRORS (consume-as-template, never edited): the
  w6-h routing-rules page `ui/src/routes/routing-rules.tsx:1-20` —
  `import { apiFetch }`, `import { Button, Badge, Toggle, ConfirmModal,
  CardSkeleton }`, `import { RoutingRuleModal }`, `createFileRoute("/routing-rules")`,
  `apiFetch<T[]>` in a load + `useState` list/loading/editing/creating/deleting,
  `CardSkeleton` while loading, empty-state copy, list rows with
  `data-testid="…-row"`, a `Toggle` for active-state, Edit→modal, Delete→
  `ConfirmModal`, `onSaved={load}`. The `/mcp` + `/mcp/tools` pages follow this
  shape exactly (no TanStack Query; plain apiFetch+useState).
- UI types this plan CONSUMES (`ui/src/lib/types.ts`):
  `McpClient` (`types.ts:110-121`,
  `{ID,Name,Transport,Command?,Args?,Env?,URL?,IsActive,HealthStatus,CreatedAt}` —
  note PascalCase keys, the real MCP DTO shape, §1.4);
  `McpInstance` (`types.ts:123-132`,
  `{ID,Name,Transport,Command?,Args?,IsActive,HealthStatus,CreatedAt}`);
  `McpTool` (`types.ts:134-141`,
  `{type, function:{name,description,parameters}}` — OpenAI function-tool shape);
  `McpToolGroup` (`types.ts:143-149`,
  `{id:number,name,tool_ids:string[],is_active,created_at}` — snake_case);
  `Skill` (`types.ts:263-268`, `{name,category,description,url}`).
- **No new dependency needed (VERIFIED):** every w6-l surface is list/table/modal/
  toggle/copy-button — built from frozen primitives only. NO charting, NO DnD, NO
  editor, NO SSE, NO Monaco. (If any surface unexpectedly needed a new dep it would
  be an ESCALATION, §8; it is not.)
- **e2e mock harness present + REGISTERED (CONSUME-ONLY — the key w6-l simplifier,
  §1.4 / §1.5):** the mcp + skills handlers AND seeds are BOTH already in-tree AND
  already wired — UNLIKE w6-i's translator (which had to add a registration):
  - `ui/e2e/mocks/handlers/mcp.ts` exists (3.8K, serving `/api/mcp/clients`,
    `/api/mcp/clients/{id}`, `/api/mcp/instances` GET+POST, `/api/mcp/instances/{id}`
    GET+DELETE, `…/{id}/accounts`, `…/{id}/auth/start`, `/api/mcp/tools`,
    `/api/mcp/tools/{name}/execute`, `/api/mcp/tool-groups` GET+POST,
    `/api/mcp/tool-groups/{id}` GET+PUT+DELETE — `mcp.ts:6-92`).
  - `ui/e2e/mocks/handlers/skills.ts` exists (`registerSkillsHandlers` serving
    `GET /api/skills` → `store.skills`, `skills.ts:5-10`).
  - BOTH registered in `ui/e2e/mocks/handlers/index.ts`:
    `import { registerSkillsHandlers } from "./skills"` (`index.ts:6`) called at
    `index.ts:41`; `import { registerMcpHandlers } from "./mcp"` (`index.ts:30`)
    called at `index.ts:65`.
  - Seeds present + exported: `seed/mcp.ts` (`seedMcpClients`/`seedMcpInstances`/
    `seedMcpTools`/`seedMcpToolGroups`) at `seed/index.ts:21`; `seed/skills.ts`
    (`seedSkills` → 2 skills, `skills.ts:3-8`) at `seed/index.ts:24`.
  - Store fields seeded: `mcpClients`/`mcpInstances`/`mcpTools`/`mcpToolGroups`
    (`store.ts:110-113`, seeded `store.ts:209-212`); `skills`
    (`store.ts:117`, seeded `store.ts:217`).
  - **CONSEQUENCE (binding):** w6-l needs ZERO `handlers/index.ts` edit, ZERO
    `seed/index.ts` edit, ZERO `store.ts` edit — every mock route + seed it needs is
    already registered. This is the ONE place w6-l is SIMPLER than w6-i (which had to
    add a translator registration). The MAP's "mock handler+seed exist" claim for
    skills is VERIFIED TRUE **and already wired** (§1.5). The sanctioned-index-edit
    exception that w6-i used is therefore NOT INVOKED here.
- Existing acceptance spec (the contract):
  `ui/e2e/mcp.spec.ts:9-17` (TWO tests: `/mcp` body contains "MCP";
  `/mcp/tools` body contains "Tools"). **NO separate `ui/e2e/mcp-tools.spec.ts`
  exists** (the mcp-tools test lives inside `mcp.spec.ts:14-17`) and **NO
  `ui/e2e/skills.spec.ts` exists** (`test ! -e` → true) — w6-l CREATES
  `skills.spec.ts` as the skills acceptance contract, committed RED first (§1.8 /
  §3 / T1), the w6-i `translator.spec.ts` precedent. Login helper
  `ui/e2e/helpers.ts:3` drives `#username`/`#password` (admin/123456).

### 1.2 Go contract for MCP + skills (file:line evidence — w6-l adds ZERO Go)

**VERIFICATION RESULT (the prompt's required per-surface check).** The MAP w6-l row
asserts "MCP gateway backend in-tree". This was VERIFIED FALSE — the in-tree
`internal/mcp/` package is a Phase-1 PLACEHOLDER with no implementation, and there
are NO admin MCP/skills routes:

- `internal/mcp/` contains ONLY `doc.go` (a package comment: "Package mcp implements
  the Model Context Protocol gateway … real MCP gateway tests arrive in Phase 12+")
  and `mcp_test.go` (`TestPackageCompiles` — a Phase-1 no-op placeholder,
  `mcp_test.go:5-9`). **There is NO MCP gateway implementation, NO handler, NO
  store.**
- `internal/server/routes_admin.go` registers NO MCP/skills routes
  (`grep -nE 'mcp|Mcp|/api/mcp|skills|Skill' internal/server/routes_admin.go` →
  EMPTY).
- `internal/admin/` has NO `mcp.go`/`skills.go`
  (`ls internal/admin/ | grep -iE 'mcp|skill'` → EMPTY).
- `grep -rniE '"/api/mcp|"/api/skills|McpHandler|SkillHandler|/api/mcp/clients|
  /api/mcp/tools|/api/mcp/instances|/api/mcp/tool-groups' internal/ cmd/` (excluding
  `_test`) → the ONLY match is `internal/server/guard.go:46`, which lists `/api/mcp/`
  in `LOCAL_ONLY_PATHS` — a localhost-restriction guard PRECONFIGURED for a FUTURE
  endpoint (the comment at `guard.go:44-45` says "Stage-1 list contains only routes
  that exist today"; the MCP guard entry is forward-looking — there is no live route
  behind it).

**Per-surface backend verification table:**

| Surface | Real Go endpoint? | Evidence | Disposition |
|---|---|---|---|
| MCP clients/instances | **ABSENT** | no `/api/mcp/clients|instances` in routes_admin.go; `internal/mcp/` placeholder; grep ZERO live route | variant-HAVE vs the `/api/mcp/{clients,instances}` MOCK (`handlers/mcp.ts`); serial Go follow-up (§8 ESC-1a) |
| MCP tools/tool-groups | **ABSENT** | no `/api/mcp/tools|tool-groups`; grep ZERO | variant-HAVE vs the `/api/mcp/{tools,tool-groups}` MOCK; serial Go follow-up (§8 ESC-1b) |
| Skills | **ABSENT** | no `/api/skills` route; no `internal/admin/skills.go`; grep ZERO | variant-HAVE vs the `/api/skills` MOCK (`handlers/skills.ts`); serial Go follow-up (§8 ESC-1c) |
| MCP marketplace (PAR-UI-054) | **ABSENT** | 9router ref hits `/api/cli-tools/cowork-mcp-registry` + `…cowork-mcp-tools` (`McpMarketplaceModal.js:6-7`) — NEITHER exists in g0router Go NOR in the g0router mock (`grep -rniE 'cowork-mcp' e2e/mocks/` → EMPTY) | variant-HAVE — install flow REMAPPED to the in-tree mcp mock (browse `/api/mcp/clients`, install `POST /api/mcp/instances`, §1.6); serial Go follow-up (§8 ESC-1a) |

**Binding consequence:** w6-l adds ZERO Go. ALL four surfaces ship as **variant-HAVE
against the registered MOCK contract** (the mocks are the binding capability contract
for THIS plan); the runtime Go gaps are recorded as serial follow-ups in §8 and
appended to `.planning/parity/plans/open-questions.md`. This mirrors w6-g quota,
w6-h aliases/routing-rules/model-limits, and the whole w6-k governance cluster (all
variant-HAVE, Go deferred). **The MAP "MCP gateway backend in-tree" assumption is
recorded as INCORRECT** in WORKFLOW.md + open-questions.

Envelope: the mcp/skills mock handlers use the `json`/`error` helpers
(`ui/e2e/mocks/handlers/utils.ts:3,11`) which mirror the real admin
`writeData`/`writeError` (`internal/admin/respond.go`) → snake_case `{data}` /
`{error:{message}}`. apiFetch (`ui/src/lib/api.ts:19`) unwraps `{data}`. **Caveat
(binding, §1.4):** the mcp mocks return the seeded objects DIRECTLY through `json(...)`
— i.e. `GET /api/mcp/clients` → `{data: McpClient[]}` with PascalCase keys
(`ID`/`Name`/`HealthStatus`/`IsActive`), and `/api/mcp/tool-groups` uses snake_case
(`tool_ids`/`is_active`/`created_at`). The pages MUST read the exact key-casing the
mock emits (§1.4); this is the binding contract.

### 1.3 The skills directory page — copy-to-clipboard + categories (binding decision)

**The reference.** 9router's skills page (`skills/page.js`) renders a categorized
directory of skills, each row showing the skill name/description and a
**copy-to-clipboard** button for the skill's raw URL (using `useCopyToClipboard` +
Material Symbols `content_copy`/`check` icons), sourced from a client-side `SKILLS`
constant + a `SKILLS_REPO_URL`.

**The g0router constraints (evidence):**
- The g0router skill source is the registered `/api/skills` MOCK
  (`handlers/skills.ts:6-10`) backed by `seedSkills()` (`seed/skills.ts:3-8`, 2
  skills: `filesystem`/`github`, each `{name,category,description,url}`,
  `types.ts:263-268`) — NOT a client-side constant. The page fetches via `apiFetch`.
- NO `useCopyToClipboard` hook in g0router (w6-a did not ship one); the copy button
  uses the native `navigator.clipboard.writeText` directly (no new hook file needed —
  a small inline handler, or a tiny w6-l-owned helper if extracted, §1.6 point 4).
  Material Symbols are available (`index.css:3`).

**Decision (binding):**
1. **`/skills` is a NEW route FILE** `ui/src/routes/skills.tsx` (§1.7) — this is the
   route addition that regenerates `routeTree.gen.ts`. PAR-UI-020 variant-HAVE.
2. **The page `apiFetch<Skill[]>("/api/skills")`** and renders the skills grouped by
   `category` (e.g. "Endpoint Skills") as rows, each with name, description, and the
   skill `url` + a copy-to-clipboard button (`navigator.clipboard.writeText(url)`,
   with a transient "Copied!" state). Body contains "Skills".
3. **Copy logic is the pure/unit-tested seam (§1.6 point 4):** any deterministic
   grouping/format logic (e.g. `groupSkillsByCategory(skills)`) is extracted into a
   PURE helper `ui/src/lib/skills-format.ts`, unit-tested without a DOM (the w6-h
   `combo-order.ts` / w6-i `translator-format.ts` precedent). The clipboard wiring is
   covered by the e2e; the grouping logic by the unit.
4. **No Go.** `/api/skills` has no backend (§1.2); PAR-UI-020 ships variant-HAVE vs
   the registered `/api/skills` mock (§8 ESC-1c).

### 1.4 Mock paths/shapes (binding interpretation — CONSUME unchanged; no Go to mirror)

The mcp + skills handlers (`ui/e2e/mocks/handlers/{mcp,skills}.ts`) are ALREADY
REGISTERED (§1.1) and model the MCP/skills capability contract; w6-l CONSUMES them
unchanged (no real Go to mirror, §1.2):

| Surface | Mock routes (file:line) | Shape (seed) | Resolution |
|---|---|---|---|
| MCP clients | `GET /api/mcp/clients` (`mcp.ts:6-9`); `GET /api/mcp/clients/{id}` (`mcp.ts:10-17`) | `McpClient[]` PascalCase `{ID,Name,Transport,Command?,Args?,Env?,URL?,IsActive,HealthStatus,CreatedAt}` (`seed/mcp.ts:3-26`, 2 rows: Filesystem/stdio, GitHub/sse) | CONSUME unchanged. `/mcp` page reads `/api/mcp/clients` (PascalCase keys, §1.2 caveat). Read/list (the mock has GET-only for clients — no POST/PUT/DELETE). Variant-HAVE; Go deferred (§8 ESC-1a). |
| MCP instances | `GET/POST /api/mcp/instances` (`mcp.ts:18-28`); `GET/DELETE /api/mcp/instances/{id}` (`mcp.ts:29-41`); `…/{id}/accounts` GET (`mcp.ts:42-45`); `…/{id}/auth/start` POST → `{url}` (`mcp.ts:46-49`) | `McpInstance[]` PascalCase (`seed/mcp.ts:28-41`, 1 row) + on POST the mock fabricates `{ID:nextId, CreatedAt, UpdatedAt, IsActive:true, ...body}` (`mcp.ts:22-24`) | CONSUME unchanged. `/mcp` page lists instances; **POST is the install/create action** (also the marketplace install target, §1.6); DELETE removes an instance via `ConfirmModal`; the `auth/start` POST returns an OAuth url (instance OAuth flow). Variant-HAVE; Go deferred (§8 ESC-1a). |
| MCP tools | `GET /api/mcp/tools` → `store.mcpTools` (`mcp.ts:50-53`); `POST /api/mcp/tools/{name}/execute` → `{result}` (`mcp.ts:54-60`) | `McpTool[]` `{type,function:{name,description,parameters}}` (`seed/mcp.ts:43-62`, 2 tools: read_file/write_file) | CONSUME unchanged. `/mcp/tools` page lists tools; a per-tool "Execute" action POSTs `…/{name}/execute` and renders the `{result}` string. Variant-HAVE; Go deferred (§8 ESC-1b). |
| MCP tool-groups | `GET/POST /api/mcp/tool-groups` (`mcp.ts:61-71`); `GET/PUT/DELETE /api/mcp/tool-groups/{id}` (`mcp.ts:72-92`) | `McpToolGroup[]` snake_case `{id,name,tool_ids,is_active,created_at}` (`seed/mcp.ts:64-67`, 1 group: File Operations) | CONSUME unchanged. `/mcp/tools` page lists tool-groups; New/Edit via `<McpToolGroupModal>` (POST/PUT), `is_active` `Toggle`, delete via `ConfirmModal`. Variant-HAVE; Go deferred (§8 ESC-1b). |
| Skills | `GET /api/skills` → `store.skills` (`skills.ts:6-9`) | `Skill[]` `{name,category,description,url}` (`seed/skills.ts:3-8`, 2 skills) | CONSUME unchanged. `/skills` page reads `/api/skills`, groups by category, copy-to-clipboard per skill (§1.3). Variant-HAVE; Go deferred (§8 ESC-1c). |

**Binding rule (MAP decision 4):** where mock and real Go disagree, real Go wins and
the mock is corrected in the SAME plan. BUT w6-l adds ZERO Go and EVERY surface has
no runtime Go to mirror; therefore the mocks ARE the binding capability contract for
this plan, CONSUMED UNCHANGED, and the runtime gaps are escalated (§8). **w6-l makes
NO mock-side edit whatsoever** — no handler body, no seed, no `index.ts`, no
`seed/index.ts`, no `store.ts`, no `fixture.ts` (every route + seed it needs is
already registered, §1.1). If a within-mock inconsistency breaks THIS cluster's
specs, ESCALATE (§8 ESC-3) rather than editing a shared/foundation mock — never add
Go, never fudge a mock. (Distinct from w6-i, which legitimately ADDED a translator
handler+registration because none existed; w6-l's mocks already exist and are wired.)

### 1.5 Skills mock is REGISTERED (no sanctioned-index-edit exception needed)

w6-i's §1.9 carved a ONE-TIME `handlers/index.ts` registration exception because the
translator surface had NO mock at all. **w6-l does NOT need that exception:** both the
mcp handler (`index.ts:30,65`) and the skills handler (`index.ts:6,41`) are ALREADY
imported + called, and both seeds are exported (`seed/index.ts:21,24`) + applied
(`store.ts:209-217`). Therefore `handlers/index.ts`, `seed/index.ts`, and `store.ts`
are FROZEN/untouched in w6-l (they appear in §3 FORBIDDEN and the §7 negative
proofs assert ZERO diff on them). The MAP's "mock handler+seed exist" note is
verified TRUE and already wired — w6-l is purely a CONSUMER of the mock layer.

### 1.6 McpMarketplaceModal — install flow REMAPPED to the in-tree mcp mock (binding)

**The reference.** `McpMarketplaceModal.js` (9router) browses a registry of MCP
servers (`fetch("/api/cli-tools/cowork-mcp-registry")` → `{servers:[…]}`,
`McpMarketplaceModal.js:6,23-28`), supports search + a filter
(all/oauth/authless, `McpMarketplaceModal.js:36-47`), expands a server to fetch its
tools (`POST /api/cli-tools/cowork-mcp-tools {url}`, `McpMarketplaceModal.js:54-60`)
with per-tool checkboxes, and calls an `onAdd` callback to install the selected
server+tools.

**The g0router constraints (evidence):**
- NEITHER `/api/cli-tools/cowork-mcp-registry` NOR `/api/cli-tools/cowork-mcp-tools`
  exists in g0router Go (§1.2) OR in the g0router mock (`grep -rniE 'cowork-mcp'
  ui/e2e/mocks/` → EMPTY). Adding a NEW cli-tools registry mock would be a new mock
  surface — but the in-tree mcp mock ALREADY models server browse + install.
- The in-tree mcp mock serves a server list via `GET /api/mcp/clients` (the
  catalog of configured MCP servers, `mcp.ts:6-9`) and an install/create via
  `POST /api/mcp/instances` (which fabricates a new active instance, `mcp.ts:21-26`).

**Decision (binding):**
1. **The marketplace browses the in-tree mcp contract, NOT the ref's cli-tools
   endpoints.** `McpMarketplaceModal` (g0router) opens, `apiFetch<McpClient[]>
   ("/api/mcp/clients")` for the browsable server list, renders search + a
   `SegmentedControl` filter (all / by transport, mapping the ref's
   all/oauth/authless intent onto the available `Transport` field), and an "Add" /
   "Install" button per server.
2. **The install action is `POST /api/mcp/instances`** with the chosen server's
   config (name/transport/command/args/url derived from the selected `McpClient`).
   The mock returns the fabricated active instance (`mcp.ts:21-26`); the modal calls
   `onAdded`/`onClose` and the `/mcp` page reloads its instance list. This is the
   **install contract** (PAR-UI-054 variant-HAVE). NO new cli-tools mock is added.
3. **The modal is mounted from the `/mcp` page** (a "Browse marketplace" / "Add MCP
   server" button opens it) — the natural host for server install. It is a
   w6-l-owned component (`ui/src/components/mcp/mcp-marketplace-modal.tsx`) built on
   the frozen `Modal` primitive (`modal.tsx:73`).
4. **Pure seam (the authoritative install-contract proof):** extract the
   server→instance install payload mapping into a PURE helper
   `ui/src/lib/mcp-install.ts` (e.g. `toInstancePayload(client): InstanceCreate`),
   unit-tested without a DOM. The modal's browse/select wiring is covered by the e2e
   (mcp.spec); the install-payload logic by the unit. (The marketplace browse/install
   is DOM-heavy; its render+open is e2e-proven, the payload mapping is the unit.)

### 1.7 NEW route file + `routeTree.gen.ts` handling (the wave-2 exception)

w6-l is THE only page-wave-2 plan that ADDS a route file
(`ui/src/routes/skills.tsx`) — the wave-2 analogue of w6-i (wave 1). The TanStack
Router Vite plugin REGENERATES `ui/src/routeTree.gen.ts` on `npm run build`/dev to
register `/skills`. Therefore:

- `ui/src/routeTree.gen.ts` **WILL change** in w6-l's diff (a `SkillsRoute`
  registration + the `/skills` path entries) — this is EXPECTED generated output,
  scoped into the route-adding commit, EXACTLY like w6-i handled `translator.tsx`
  (w6-i §1.7) and w6-c handled `callback.tsx`.
- `routeTree.gen.ts` is **NEVER hand-edited** — it is regenerated by the build/dev
  plugin. If the worker cannot regenerate it cleanly (plugin not running, stale tree),
  that is an ESCALATION (§8 ESCALATION-2), not a manual edit.
- The two REWRITTEN MCP stubs (`mcp.tsx`, `mcp.tools.tsx`) do NOT add routes
  (already registered, §1.1) and do not themselves change the tree. The `mcp/tools`
  nested route already exists via the `mcp.tools.tsx` flat-file name (§1.8). Only the
  skills addition changes the tree.
- `routeTree.gen.ts` IS in w6-l's allowed diff (§3 / §7) — the SOLE difference from
  sibling page-wave-2 plans (w6-f/j/k/m), where it must be UNCHANGED.

### 1.8 `/mcp/tools` nested-route naming + the skills spec must be CREATED (RED first)

- **`/mcp/tools` is a NESTED route under `/mcp`** via TanStack flat-route DOTTED file
  naming: the file is `ui/src/routes/mcp.tools.tsx` (NOT `ui/src/routes/mcp/tools.tsx`
  — that nested-dir form does NOT exist, `test ! -e ui/src/routes/mcp/tools.tsx` →
  true). The dotted name `mcp.tools` makes `McpToolsRoute` a child of `McpRoute`
  (`routeTree.gen.ts:44,209,697-705`). w6-l REWRITES the EXISTING `mcp.tools.tsx`
  stub in place; this does NOT change the route tree (it is already registered,
  §1.1). The `/mcp` page renders its own content (NOT an `<Outlet>` for tools unless
  the design nests them; the existing tree has `mcp/tools` as a child route reached
  by navigating to `/mcp/tools`, which the spec does via `page.goto("/mcp/tools")`).
- **`skills.spec.ts` must be CREATED (committed RED first):** there is NO
  `ui/e2e/skills.spec.ts` today (§1.1). w6-l CREATES it as the skills acceptance
  contract and commits it RED in T1 (before any skills impl). It is NEW-spec
  ownership by w6-l (the w6-i `translator.spec.ts` precedent). It asserts: `/skills`
  body contains "Skills"; skill rows render (≥2 from `seedSkills` — `filesystem`/
  `github` visible, grouped by `category`); a copy-to-clipboard control is present
  per skill. The `mcp.spec.ts` ALREADY exists (2 tests) and is EXTENDED with RED
  assertions (§3), not created. There is NO separate `mcp-tools.spec.ts` — the
  `/mcp/tools` assertions live in `mcp.spec.ts`.

### Variant notes (recorded HAVE rationale)

- **PAR-UI-020 skills**: variant — NEW route file; consumes the registered
  `/api/skills` mock; copy-to-clipboard via native `navigator.clipboard`; NO Go.
  §1.3/§1.7/§8 ESC-1c.
- **PAR-UI-054 McpMarketplaceModal**: variant — the ref's cli-tools registry
  endpoints are absent; the browse/install flow is REMAPPED to the in-tree mcp mock
  (browse `/api/mcp/clients`, install `POST /api/mcp/instances`); NO Go. §1.6/§8
  ESC-1a.
- **PAR-UI-130 `/mcp` + `/mcp/tools`**: variant — REWRITE existing stubs;
  list/CRUD/execute against the `/api/mcp/{clients,instances,tools,tool-groups}`
  mocks; NO Go (`internal/mcp/` is a placeholder). §1.2/§1.4/§8 ESC-1a/1b.
- **MCP DTO casing**: the mcp mock emits PascalCase keys for clients/instances
  (`ID`/`Name`/`HealthStatus`/`IsActive`) and snake_case for tool-groups
  (`tool_ids`/`is_active`) — the pages read the exact casing the mock emits (§1.2
  caveat / §1.4). Accepted constraint, not a gap.
- **Pages render inside app chrome**: `__root.tsx` wraps every route in
  Sidebar+Header; pages render in `<Outlet>`; specs assert page content with chrome
  present (w6-c/w6-e/w6-i/w6-k precedent). Accepted constraint, not a gap.
- **Data layer = plain `apiFetch` + React state, NOT TanStack Query**: `QueryClient`
  is NOT mounted (`__root.tsx`/`main.tsx` FROZEN, w6-a; verified
  `grep -rn QueryClientProvider ui/src/routes/__root.tsx ui/src/main.tsx` → EMPTY).
  PAR-UI-081 already HAVE from w6-a (`open-questions.md:6`); w6-l consumes `apiFetch`
  and does NOT mount a provider, NOT edit any frozen file. Accepted constraint.

### NOT in scope (explicit)

- **No Go changes.** ALL of `internal/` is FORBIDDEN. The MCP/skills backends are
  ABSENT (`internal/mcp/` is a Phase-1 placeholder, §1.2) — those are ESCALATIONS
  (§8), NEVER an in-plan Go edit (the MAP assigns no Go to w6-l; the serial chain
  closed on w6-j). No new `internal/admin/{mcp,skills}.go`, no `internal/mcp/*`
  implementation, no `routes_admin.go` MCP routes, no `guard.go` edit.
- **No dependency additions** — every surface uses frozen primitives only (§1.1). NO
  `package.json`/lockfile edit.
- **No edits to any frozen w6-a/w6-b/page-wave file** — no `__root.tsx`, `main.tsx`,
  layout components, `ui/src/components/ui/*`, `ui/src/stores/*`, `ui/src/lib/api.ts`,
  `ui/src/lib/utils.ts`, `ui/src/lib/auth.ts`, `ui/src/providers/*`,
  `ui/src/routes/{login,callback}.tsx`, nor any sibling page-plan's routes/components.
  No header exception remains (SPENT).
- **No `QueryClientProvider` mount** — plain `apiFetch`; PAR-UI-081 already HAVE
  (w6-a).
- **No mock-layer edits at all** (§1.4/§1.5) — `ui/e2e/mocks/handlers/index.ts`,
  `ui/e2e/mocks/seed/index.ts`, `ui/e2e/mocks/store.ts`, `ui/e2e/mocks/fixture.ts`,
  and the mcp/skills handler BODIES + seed files are CONSUMED unchanged (already
  registered). NO new mock handler, NO seed correction. (Distinct from w6-i.)
- **No new cli-tools registry mock** — the marketplace install is remapped to the
  existing mcp mock (§1.6), not a new `/api/cli-tools/cowork-mcp-*` surface.
- **No new spec files beyond `skills.spec.ts`** — only `mcp.spec.ts` (extended) +
  the NEW `skills.spec.ts`. NO `mcp-tools.spec.ts` (the `/mcp/tools` assertions live
  in `mcp.spec.ts`, §1.8).
- **No SSE/streaming/charts/DnD/editor** — all surfaces are request/response CRUD +
  copy-to-clipboard + one execute POST.
- **No providers/connections/models (w6-e), no keys/VK/endpoint (w6-f), no usage/
  quota/pricing (w6-g), no combos/routing (w6-h), no chat/console/translator (w6-i),
  no settings/version (w6-j), no governance pages (w6-k), no platform pages (w6-m).**
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
git rev-parse HEAD         # record as <base> for §5 (expected b23bead)

# P1 — w6-b primitives present and frozen (consumed)
ls ui/src/components/ui/*.tsx | grep -v test | wc -l    # = 16 (w6-b set intact)
grep -n "export { Modal }" ui/src/components/ui/modal.tsx          # :73
grep -n "ConfirmModal" ui/src/components/ui/confirm-modal.tsx      # :46
grep -n "SegmentedControl" ui/src/components/ui/segmented-control.tsx
grep -n "export { Card" ui/src/components/ui/card.tsx              # :70

# P2 — w6-a foundation present and frozen (consumed; PAR-UI-081 already HAVE)
grep -n "export async function apiFetch" ui/src/lib/api.ts         # :19
grep -n "export class ApiError" ui/src/lib/api.ts                  # :3
grep -n "push:" ui/src/stores/notification.ts                     # :11
grep -n 'material-symbols' ui/src/index.css                        # :3 (copy-button icon)
grep -rn "QueryClientProvider\|QueryClient" ui/src/routes/__root.tsx ui/src/main.tsx ; echo "^ expect EMPTY (not mounted; do NOT mount)"

# P3 — the shipped-page template is present (consume-as-template, never edit)
grep -n "apiFetch\|ConfirmModal\|createFileRoute" ui/src/routes/routing-rules.tsx

# P4 — mcp/mcp-tools stubs bare (safe to rewrite); skills route + dirs ABSENT
grep -n "<h1>MCP</h1>"       ui/src/routes/mcp.tsx
grep -n "<h1>MCP Tools</h1>" ui/src/routes/mcp.tools.tsx
grep -n 'createFileRoute("/mcp/tools")' ui/src/routes/mcp.tools.tsx ; echo "^ dotted flat-route → /mcp/tools nested under /mcp (§1.8)"
test ! -e ui/src/routes/mcp/tools.tsx && echo "no nested-dir mcp/tools.tsx (flat dotted name is canonical — good)"
test ! -e ui/src/routes/skills.tsx && echo "skills route absent (good — NEW file §1.7)"
test ! -d ui/src/components/mcp    && echo "mcp components dir absent (good)"
test ! -d ui/src/components/skills && echo "skills components dir absent (good)"
test ! -e ui/src/lib/skills-format.ts && echo "skills-format helper absent (good)"
test ! -e ui/src/lib/mcp-install.ts && echo "mcp-install helper absent (good)"
test ! -e ui/e2e/skills.spec.ts && echo "skills spec absent (good — CREATE RED §1.8)"
grep -n "skills\|SkillsRoute" ui/src/routeTree.gen.ts ; echo "^ expect EMPTY (no /skills yet; build regenerates it §1.7)"
grep -n "McpRoute\|McpToolsRoute\|'/mcp'\|'/mcp/tools'" ui/src/routeTree.gen.ts | head ; echo "^ /mcp + /mcp/tools ALREADY registered (stubs, §1.1)"

# P5 — e2e mock harness present + ALREADY REGISTERED (CONSUME unchanged; §1.4/§1.5)
grep -n "registerMcpHandlers\|registerSkillsHandlers" ui/e2e/mocks/handlers/index.ts   # :6/:30 imports, :41/:65 calls
grep -n "seedMcp\|seedSkills" ui/e2e/mocks/seed/index.ts                                # :21/:24
grep -n "mcpClients\|mcpInstances\|mcpTools\|mcpToolGroups\|skills" ui/e2e/mocks/store.ts | head   # :110-117 fields, :209-217 seeded
grep -n "/api/mcp/clients\|/api/mcp/instances\|/api/mcp/tools\|/api/mcp/tool-groups" ui/e2e/mocks/handlers/mcp.ts   # the consumed contract
grep -n "/api/skills" ui/e2e/mocks/handlers/skills.ts                                   # :6
grep -rn "cowork-mcp\|cli-tools/cowork" ui/e2e/mocks/ ; echo "^ expect EMPTY (ref's cli-tools registry NOT in g0router; marketplace remaps to mcp mock §1.6)"

# P6 — Go reality: MCP/skills backends ABSENT (internal/mcp is a placeholder; §1.2/§8)
grep -nE 'mcp|Mcp|/api/mcp|skills|Skill' internal/server/routes_admin.go ; echo "^ expect EMPTY (no Go MCP/skills routes)"
ls internal/admin/ | grep -iE 'mcp|skill' ; echo "^ expect EMPTY (no admin mcp/skills handler)"
cat internal/mcp/doc.go ; echo "^ Phase-1 PLACEHOLDER package (no implementation)"
grep -n "TestPackageCompiles" internal/mcp/mcp_test.go ; echo "^ Phase-1 no-op test (real tests Phase 12+)"
grep -rniE '"/api/mcp|"/api/skills|McpHandler|SkillHandler' internal/ cmd/ | grep -v _test ; echo "^ ONLY guard.go:46 /api/mcp/ LOCAL_ONLY (forward-looking guard, no live route)"

# P7 — routes_admin.go serial slot: w6-l does NOT take it (zero Go; chain closed on w6-j)
echo "w6-l adds ZERO Go → does NOT touch internal/server/routes_admin.go; the serial chain (w6-pre→w6-d→w6-e→w6-j) CLOSED on w6-j."

# P8 — harness green at base
cd ui && npx playwright test e2e/mcp.spec.ts
# Record base result: stubs render only <h1> ("MCP" / "MCP Tools"), so the two
# text-contains smoke assertions ("MCP" on /mcp; "Tools" on /mcp/tools) PASS at base
# (the <h1> + sidebar chrome carry the names). The RED arc is the ADDED list/modal/
# marketplace assertions in §4 T1 + the NEW skills.spec.ts (no skills page yet).
# Record exact pass/fail per spec in WORKFLOW.md.
cd ui && npm run build                               # exit 0
cd ui && npx vitest run src/                         # exit 0 (existing units green)
go test ./... && go vet ./...                        # exit 0 (Go untouched-green)
```

---

## 3. Exclusive file ownership

After w6-l merges, all CREATE files below are owned by w6-l; later plans consume,
never edit (MAP decision 7).

**CREATE — routes:**

| File | Exports / contract |
|---|---|
| `ui/src/routes/mcp.tsx` (REWRITE) | `Route=createFileRoute("/mcp")`; `McpPage`: `apiFetch<McpClient[]>("/api/mcp/clients")` + `apiFetch<McpInstance[]>("/api/mcp/instances")` → list rows (Name, Transport `Badge`, HealthStatus `Badge`, IsActive); install via `<McpMarketplaceModal>` (browse `/api/mcp/clients`, install `POST /api/mcp/instances`, §1.6); delete an instance via `ConfirmModal` (DELETE `/api/mcp/instances/{id}`); per-instance OAuth via `POST …/{id}/auth/start`. Reads PascalCase keys (§1.2/§1.4). Body contains "MCP". g0router-EXTRA (no 9router page ref). |
| `ui/src/routes/mcp.tools.tsx` (REWRITE; dotted flat-route → `/mcp/tools`, §1.8) | `Route=createFileRoute("/mcp/tools")`; `McpToolsPage`: `apiFetch<McpTool[]>("/api/mcp/tools")` → tool rows (function.name, description) each with an "Execute" action POSTing `…/{name}/execute` and rendering `{result}`; PLUS `apiFetch<McpToolGroup[]>("/api/mcp/tool-groups")` → group list with New/Edit via `<McpToolGroupModal>` (POST/PUT), `is_active` `Toggle`, delete via `ConfirmModal`. Body contains "Tools". g0router-EXTRA. |
| `ui/src/routes/skills.tsx` (**NEW FILE**, §1.7) | `Route=createFileRoute("/skills")`; `SkillsPage`: `apiFetch<Skill[]>("/api/skills")` → skills grouped by `category`, each row name/description/url + a copy-to-clipboard button (`navigator.clipboard.writeText(url)`, transient "Copied!"). Body contains "Skills". Adding this file regenerates `routeTree.gen.ts`. Ports `skills/page.js` adapted to apiFetch + native clipboard. |

**CREATE — domain components (`ui/src/components/mcp/`):**

| File | Exports / contract |
|---|---|
| `mcp-marketplace-modal.tsx` | `McpMarketplaceModal` (PAR-UI-054, §1.6) — `Modal`+`Input`(search)+`SegmentedControl`(filter); `apiFetch<McpClient[]>("/api/mcp/clients")` browse list; per-server "Install" → `POST /api/mcp/instances` (payload via `mcp-install` helper); `onAdded`/`onClose`. Built on the frozen `Modal` (`modal.tsx:73`). |
| `mcp-client-card.tsx` | `McpClientCard` (PAR-UI-130 `/mcp`) — renders one `McpClient`/`McpInstance` row (Name, Transport `Badge`, HealthStatus `Badge`, actions). |
| `mcp-tool-group-modal.tsx` | `McpToolGroupModal` (PAR-UI-130 `/mcp/tools`) — `Modal`+`Input`; fields name, tool_ids (select from the tools list), is_active; POST `/api/mcp/tool-groups` (new) / PUT `…/{id}` (edit); `onSaved` reload. |

**CREATE — lib (pure helpers, §1.3/§1.6):**

| Export | Contract |
|---|---|
| `ui/src/lib/skills-format.ts` — `groupSkillsByCategory(skills: Skill[]): Record<string, Skill[]>` (and/or a `skillCopyValue(skill)` accessor) | Pure, deterministic, no DOM. The AUTHORITATIVE skills-grouping proof (§1.3 point 3). |
| `ui/src/lib/mcp-install.ts` — `toInstancePayload(client: McpClient): InstanceCreate` | Pure mapping from a browsed `McpClient` to the `POST /api/mcp/instances` body (§1.6 point 4). The AUTHORITATIVE install-contract proof. |

**CREATE — unit tests (vitest — pure logic reachable without a live backend):**

| File | Contents |
|---|---|
| `ui/src/lib/skills-format.test.ts` | ≥3 tests: `groupSkillsByCategory` deterministic grouping for sample `Skill[]`; empty input → empty grouping; idempotence/immutability. Committed RED before `skills-format.ts`. |
| `ui/src/lib/mcp-install.test.ts` | ≥3 tests: `toInstancePayload` maps Name/Transport/Command/Args/URL from a `McpClient`; handles stdio vs sse transports; omits absent optional fields. Committed RED before `mcp-install.ts`. |

(`mcp-marketplace-modal.tsx`, `mcp-client-card.tsx`, `mcp-tool-group-modal.tsx`, and
the page-level wiring are DOM-heavy; their coverage is the e2e assertions, not units
— same disposition as w6-e/w6-h/w6-k modal components. If a tester wants extra
coverage, a stubbed-apiFetch component unit is OPTIONAL, not required.)

**CREATE — NEW e2e spec (§1.8):**

| File | Contents |
|---|---|
| `ui/e2e/skills.spec.ts` (**NEW**) | RED first (T1): `/skills` body contains "Skills"; skill rows render (≥2 from `seedSkills` — `filesystem`/`github` visible, grouped by `category`); a copy-to-clipboard control is present per skill. |

**MODIFY — existing e2e spec (the acceptance contract):**

| File | Change |
|---|---|
| `ui/e2e/mcp.spec.ts` | KEEP the 2 existing tests (`/mcp` body "MCP"; `/mcp/tools` body "Tools"). ADD RED: `/mcp` lists MCP clients/instances from seed (≥1 row, Name + Transport `Badge`); a "Browse marketplace"/"Add MCP server" button opens `McpMarketplaceModal` (`[data-testid="modal-traffic-lights"]`) and an install fires `POST /api/mcp/instances`. ADD RED: `/mcp/tools` lists tools (≥2: read_file/write_file) with an Execute action; tool-groups list (≥1: File Operations) with an `is_active` `Toggle`; New/Edit via `McpToolGroupModal`. (No `mcp-tools.spec.ts` — the `/mcp/tools` assertions live here, §1.8.) |
| `ui/e2e/mocks/handlers/{mcp,skills}.ts`, `ui/e2e/mocks/seed/{mcp,skills}.ts` | CONSUME as-is (§1.4/§1.5). NO edit. If a within-mock inconsistency breaks this cluster's specs, ESCALATE (§8 ESC-3) — never edit a shared/foundation mock, never add Go. |

**GENERATED (expected, §1.7):**

| File | Change |
|---|---|
| `ui/src/routeTree.gen.ts` | REGENERATED by the TanStack Router Vite plugin to register `/skills`. NEVER hand-edited; appears in w6-l's diff as expected route-addition output (w6-i/w6-c precedent). The `/mcp` + `/mcp/tools` entries are ALREADY present (stubs) and are NOT changed by the rewrites. |

**FORBIDDEN:** everything else. Explicitly: ALL of `internal/` (`internal/mcp/`
placeholder + absent admin MCP/skills routes — ESCALATIONS §8, never an in-plan Go
edit; w6-l holds NO serial slot; NO `guard.go` edit); ALL of
`ui/src/components/ui/*` (w6-b frozen); `ui/src/stores/*`, `ui/src/lib/api.ts`,
`ui/src/lib/utils.ts`, `ui/src/lib/auth.ts`, `ui/src/providers/*` (w6-a frozen);
`ui/src/routes/__root.tsx`, `ui/src/main.tsx`, `ui/src/components/layout/*`,
`ui/src/routes/{login,callback}.tsx`, `ui/src/components/auth/*`; ALL sibling
page-plan routes/components (`ui/src/routes/{providers,connections,models,keys,
virtual-keys,endpoint,dashboard,usage,logs,traffic,quota,pricing,combos,
routing-rules,model-limits,aliases,chat,console,translator,settings,teams,audit,
feature-flags,guardrails,prompts,alerts,mitm,proxy-pools,tunnels}.tsx` and their
component dirs); `ui/package.json` + lockfile; `ui/vite.config.ts`;
`ui/playwright.config.ts`; `ui/components.json`; `ui/src/index.css`;
`ui/e2e/mocks/fixture.ts`, `ui/e2e/mocks/store.ts`,
`ui/e2e/mocks/handlers/index.ts`, `ui/e2e/mocks/seed/index.ts` (foundation/wiring —
mcp/skills ALREADY registered, §1.5); the mcp/skills handler BODIES + seed files
(CONSUME unchanged — NO edit, §1.4); all other `ui/e2e/*.spec.ts`; NO new mock
files (no cli-tools registry mock — §1.6). `ui/dist/**` is gitignored — never stage
it. `ui/src/routeTree.gen.ts` is the ONE generated exception (route addition, §1.7)
and IS allowed.

---

## 4. TDD tasks

Cadence (strict): **no route/component/lib file may exist (or be rewritten beyond
its stub) before the failing test that covers it is committed.** `cd ui && npm run
build` green at EVERY commit (test files + red specs are never imported by production
code — w6-b/w6-c/w6-e/w6-i/w6-k rationale; the skills route addition lands WITH its
impl so the build can regenerate `routeTree.gen.ts` against a real component, §1.7).
`go test ./... && go vet ./...` stays untouched-green at EVERY commit (ZERO new Go).
The mcp spec (extended) + the NEW skills spec stay RED until the implementation tasks
green them; that is the arc.

### T1 — STEP(a): extend mcp spec + CREATE skills spec (commit RED)

Add the RED tests in §3 to `ui/e2e/mcp.spec.ts` (mcp clients/instances list +
marketplace open/install; mcp-tools list/execute/tool-groups, §1.8). CREATE
`ui/e2e/skills.spec.ts` RED (§1.8). CONSUME `handlers/{mcp,skills}.ts` +
`seed/{mcp,skills}.ts` unchanged (§1.4/§1.5) — NO mock edit, NO registration edit
(already wired).

STEP(b): run `e2e/mcp.spec.ts e2e/skills.spec.ts` — **record failure output** (no
mcp client/instance rows; no marketplace modal; no tools/tool-groups; no skills
page/rows). Commit RED:
`phase-1/w6-l: failing mcp e2e extensions + new skills spec (TDD red)`.

**Mock-vs-reality gate**: re-confirm §1.2 (MCP/skills backends ABSENT —
`internal/mcp/` placeholder; ESCALATIONS §8, NOT in-plan Go). CONSUME the registered
mocks as the capability contract; w6-l makes NO mock-side edit. If a within-mock
inconsistency breaks a spec, ESCALATE (§8 ESC-3) — never edit a shared/foundation
mock, never add Go, never edit `index.ts`/`seed/index.ts`/`store.ts`/`fixture.ts`.

### T2 — STEP(a): unit tests for skills-format + mcp-install (commit RED)

Write `ui/src/lib/skills-format.test.ts` (§3, §1.3 — the grouping proof) and
`ui/src/lib/mcp-install.test.ts` (§3, §1.6 — the install-payload proof). Pure logic;
no DOM (the w6-h `combo-order.ts` / w6-i `translator-format.ts` precedent). Run
`cd ui && npx vitest run src/lib/skills-format.test.ts src/lib/mcp-install.test.ts`
→ FAILS (modules missing). Record failure. Commit RED:
`phase-1/w6-l: failing unit tests for skills-format + mcp-install (TDD red)`.

### T3 — STEP(b): mcp page + mcp-tools page + their components/helpers

Implement `ui/src/lib/mcp-install.ts` (greens its unit), `mcp-client-card.tsx`,
`mcp-marketplace-modal.tsx`, `mcp-tool-group-modal.tsx`; rewrite `mcp.tsx`,
`mcp.tools.tsx`. Gates: `npx vitest run src/lib/mcp-install.test.ts` green;
`mcp.spec.ts` green (clients/instances list, marketplace open+install, tools execute,
tool-groups CRUD); `npm run build` green; `go test ./... && go vet ./...` untouched
green. Commit:
`phase-1/w6-l: mcp page (clients/instances + marketplace) + mcp-tools page (tools/tool-groups)`.

### T4 — STEP(b): skills page + helper + NEW route + routeTree regen

Implement `ui/src/lib/skills-format.ts` (greens its unit); CREATE
`ui/src/routes/skills.tsx` (NEW route, §1.7). Run `npm run build` to REGENERATE
`ui/src/routeTree.gen.ts` (NEVER hand-edit, §1.7); verify `/skills` is registered.
Gates: `npx vitest run src/lib/skills-format.test.ts` green; `skills.spec.ts` green
(rows render grouped by category, copy control present); BOTH specs green;
`npx vitest run src/` green; `npm run build` green (routeTree regenerated cleanly);
`go test ./... && go vet ./...` untouched green. Commit (the route-adding commit that
carries the routeTree change):
`phase-1/w6-l: skills page (NEW route) + skills-format helper + routeTree regen`.

### T5 — full gates + closeout

```bash
cd ui && npm run build                                    # regenerates routeTree.gen.ts
cd ui && npx playwright test e2e/mcp.spec.ts e2e/skills.spec.ts   # all green
cd ui && npx playwright test                              # full suite: no spec green-at-base may be red
cd ui && npx vitest run src/                              # all green incl new units
go test ./... && go vet ./...                             # untouched-green (ZERO new Go)
grep -n "SkillsRoute\|/skills" ui/src/routeTree.gen.ts    # /skills now registered (§1.7)
```
Flip/annotate §1 matrix rows in `.planning/parity/matrix/9router-ui.md`:
PAR-UI-020 → HAVE (variant, cite §1.3/§1.7/§8 ESC-1c); PAR-UI-054 → HAVE (variant,
cite §1.6/§8 ESC-1a); PAR-UI-130 → APPEND `/mcp,/mcp/tools` HAVE (variant, cite
§1.2/§1.4/§8 ESC-1a/1b — do NOT overwrite sibling-owned partials, §1 note). Update
`docs/WORKFLOW.md` (record the P8 base spec observations, the new-route/routeTree
regen, and the **MAP "MCP gateway backend in-tree" assumption recorded as INCORRECT**
with the MCP + skills serial Go follow-ups §8). Append the §8 open items to
`.planning/parity/plans/open-questions.md`. Final commit:
`phase-1/w6-l: close — mcp/mcp-tools/skills cluster; new skills route; matrix flips`.
**w6-l holds NO serial slot — nothing to release.**

---

## 5. Binary acceptance criteria

All must hold; each is yes/no. `<base>` = the commit recorded at P0 (b23bead at
authoring). Diff gate is **w6-l commit-range-scoped** (§7) because page-wave plans
commit to main concurrently.

**Test gates**
- `cd ui && npx playwright test e2e/mcp.spec.ts` → exit 0, all tests pass (2 original
  + added: mcp list, marketplace, tools/tool-groups), 0 skipped.
- `cd ui && npx playwright test e2e/skills.spec.ts` → exit 0, all pass (NEW spec).
- `cd ui && npx vitest run src/lib/skills-format.test.ts` → exit 0, ≥3 cases pass.
- `cd ui && npx vitest run src/lib/mcp-install.test.ts` → exit 0, ≥3 cases pass.
- `cd ui && npx vitest run src/` → exit 0 (all prior + new units green).
- `cd ui && npm run build` → exit 0 (routeTree.gen.ts regenerated with `/skills`).
- `go test ./... && go vet ./...` → exit 0 (Go untouched — ZERO new Go).

**TDD-order proof** — each impl file's covering test appears in an earlier-or-equal
commit:
```bash
ct=$(git log --format=%ct --diff-filter=A -1 -- ui/src/lib/skills-format.test.ts)
cf=$(git log --format=%ct --diff-filter=A -1 -- ui/src/lib/skills-format.ts)
[ "$ct" -le "$cf" ] || echo "TDD VIOLATION: skills-format.ts"           # prints nothing
ct=$(git log --format=%ct --diff-filter=A -1 -- ui/src/lib/mcp-install.test.ts)
cf=$(git log --format=%ct --diff-filter=A -1 -- ui/src/lib/mcp-install.ts)
[ "$ct" -le "$cf" ] || echo "TDD VIOLATION: mcp-install.ts"             # nothing
# the NEW skills.spec.ts RED commit precedes the skills route
sa=$(git log --format=%ct -1 --grep="new skills spec")
sk=$(git log --format=%ct --diff-filter=A -1 -- ui/src/routes/skills.tsx)
[ "$sa" -le "$sk" ] || echo "TDD VIOLATION: skills.tsx before red spec" # nothing
# the mcp e2e RED-extension precedes the mcp/mcp-tools rewrites
ma=$(git log --format=%ct --diff-filter=M -1 -- ui/src/routes/mcp.tsx)
[ "$sa" -le "$ma" ] || echo "TDD VIOLATION: mcp.tsx before red spec"    # nothing
```

**Grep proofs**
```bash
grep -rn "/api/mcp/clients\|/api/mcp/instances" ui/src/routes/mcp.tsx ui/src/components/mcp/   # PAR-UI-130 /mcp + PAR-UI-054 install
grep -rn "/api/mcp/instances" ui/src/components/mcp/mcp-marketplace-modal.tsx                  # PAR-UI-054 install action (§1.6)
grep -rn "/api/mcp/tools\|/api/mcp/tool-groups" ui/src/routes/mcp.tools.tsx ui/src/components/mcp/   # PAR-UI-130 /mcp/tools
grep -rn '/api/mcp/tools/.*execute\|execute' ui/src/routes/mcp.tools.tsx                       # tools execute action
grep -rn "/api/skills" ui/src/routes/skills.tsx                                                # PAR-UI-020
grep -rn "navigator.clipboard\|writeText" ui/src/routes/skills.tsx                             # §1.3 copy-to-clipboard
grep -rn "groupSkillsByCategory\|export " ui/src/lib/skills-format.ts                          # §1.3 pure helper
grep -rn "toInstancePayload\|export " ui/src/lib/mcp-install.ts                                # §1.6 pure helper
grep -rn "McpMarketplaceModal" ui/src/routes/mcp.tsx                                           # PAR-UI-054 modal mounted on /mcp (§1.6)
test -f ui/src/routes/skills.tsx && echo OK                                                    # PAR-UI-020 new route
grep -n 'SkillsRoute\|/skills' ui/src/routeTree.gen.ts                                         # §1.7 route registered (generated)
grep -n 'McpToolsRoute\|/mcp/tools' ui/src/routeTree.gen.ts                                    # §1.8 /mcp/tools still registered (unchanged)
# No new cli-tools registry mock added:
! grep -rniE 'cowork-mcp|cli-tools/cowork' ui/e2e/mocks/ ui/src/components/mcp/ && echo "no cli-tools registry mock (remapped to mcp mock §1.6) OK"
# NO QueryClientProvider introduced:
! grep -rn 'QueryClientProvider' ui/src/components/mcp ui/src/routes/mcp.tsx ui/src/routes/mcp.tools.tsx ui/src/routes/skills.tsx && echo "no QueryClientProvider added OK"
# No editor dep / no new dep import:
! grep -rniE 'monaco|codemirror' ui/src/routes/skills.tsx ui/src/routes/mcp.tsx ui/src/routes/mcp.tools.tsx ui/src/components/mcp/ && echo "no editor dep OK"
```

**Negative / freeze proofs (w6-l commit-range — see §7)**
```bash
R="<first-w6-l>^..<last-w6-l>"
git diff $R --name-only -- internal/ | wc -l                            # = 0 (ZERO new Go; routes_admin.go/internal/mcp/guard.go untouched, no serial slot)
git diff $R --name-only -- ui/package.json ui/package-lock.json ui/vite.config.ts ui/playwright.config.ts ui/components.json ui/src/index.css | wc -l   # = 0 (no deps/config)
git diff $R --name-only -- ui/src/components/ui/ | wc -l                # = 0 (w6-b frozen)
git diff $R --name-only -- ui/src/stores/ ui/src/providers/ ui/src/lib/api.ts ui/src/lib/utils.ts ui/src/lib/auth.ts | wc -l   # = 0 (w6-a frozen)
git diff $R --name-only -- ui/src/routes/__root.tsx ui/src/main.tsx ui/src/components/layout/ ui/src/routes/login.tsx ui/src/routes/callback.tsx ui/src/components/auth/ | wc -l   # = 0
git diff $R --name-only -- ui/e2e/mocks/fixture.ts ui/e2e/mocks/store.ts ui/e2e/mocks/handlers/index.ts ui/e2e/mocks/seed/index.ts | wc -l   # = 0 (foundation/wiring untouched — mcp/skills already registered §1.5)
git diff $R --name-only -- ui/e2e/mocks/handlers/ | wc -l               # = 0 (NO mcp/skills handler body edited; mocks consumed as-is §1.4)
git diff $R --name-only -- ui/e2e/mocks/seed/ | wc -l                   # = 0 (mcp/skills seeds consumed unchanged)
git diff $R --name-only -- 'ui/src/routes/' | grep -vE 'mcp\.tsx|mcp\.tools\.tsx|skills\.tsx' | wc -l   # = 0 (only the two mcp stubs rewritten + the one new skills route)
git diff $R --name-only -- ui/e2e/ | grep -vE 'mcp\.spec\.ts|skills\.spec\.ts' | wc -l   # = 0 (only the mcp spec + new skills spec; no mock edit)
# routeTree.gen.ts IS allowed here (the wave-2 exception, §1.7) — it WILL appear:
git diff $R --name-only -- ui/src/routeTree.gen.ts | wc -l             # = 1 (regenerated /skills addition — EXPECTED, unlike sibling wave-2 plans)
```

---

## 6. Out of scope (restated, binding)

ZERO Go changes — the MCP/skills backends are ABSENT (`internal/mcp/` is a Phase-1
placeholder, §1.2; the MAP "MCP gateway backend in-tree" claim is FALSE and recorded
as such); ALL four surfaces ship variant-HAVE against the registered/consumed MOCK
contract with the runtime Go gaps deferred to serial follow-ups (§8), NEVER an
in-plan Go edit (MAP assigns no Go to w6-l; the serial chain closed on w6-j) and w6-l
holds NO serial slot; no `QueryClientProvider` mount (PAR-UI-081 already HAVE from
w6-a); no dependency additions (frozen primitives only); no edits to any frozen
w6-a/w6-b/page-wave file (no header exception remains — SPENT); NO mock-layer edits
at all — `index.ts`/`seed/index.ts`/`store.ts`/`fixture.ts` and the mcp/skills
handler bodies + seeds are CONSUMED unchanged (already registered, §1.4/§1.5; the
sanctioned-index-edit exception that w6-i used is NOT invoked here); no new cli-tools
registry mock (the marketplace install is remapped to the existing mcp mock, §1.6);
no new spec files beyond `skills.spec.ts`; no SSE/charts/DnD/editor. **The new skills
route file DOES regenerate `ui/src/routeTree.gen.ts`, which IS in w6-l's allowed diff
(§1.7/§7) — the SOLE difference from sibling page-wave-2 plans (w6-f/j/k/m).**
Mock-vs-Go divergence, an absent backend, a shared/foundation-mock edit that would
ripple to a non-w6-l spec, or a routeTree regen failure → escalate (§8), never patch
Go, never fudge a mock, never hand-edit routeTree.gen.ts.

## 7. Diff-gate scope

Page-wave plans (w6-f/j/k/l/m) commit to main concurrently, so a broad
`<base>..HEAD` range sweeps in sibling commits. The diff gate MUST be scoped to
w6-l's own commits. The orchestrator isolates them with:
`git log --oneline main | grep "^[0-9a-f]* phase-1/w6-l:" | awk '{print $1}'`
and runs `git diff <first-w6-l>^..<last-w6-l> -- [file list]` (same commit-range
scoping as w6-c §7 / w6-e §7 / w6-i §7 / w6-k §7).

`git diff <first-w6-l>^..<last-w6-l> --name-only` must be exactly a subset of:

```
ui/src/routes/mcp.tsx
ui/src/routes/mcp.tools.tsx
ui/src/routes/skills.tsx                       (NEW route file, §1.7)
ui/src/routeTree.gen.ts                         (generated by build; route addition — the wave-2 exception §1.7)
ui/src/components/mcp/mcp-marketplace-modal.tsx
ui/src/components/mcp/mcp-client-card.tsx
ui/src/components/mcp/mcp-tool-group-modal.tsx
ui/src/lib/skills-format.ts
ui/src/lib/skills-format.test.ts
ui/src/lib/mcp-install.ts
ui/src/lib/mcp-install.test.ts
ui/e2e/mcp.spec.ts
ui/e2e/skills.spec.ts                           (NEW spec, §1.8)
.planning/parity/matrix/9router-ui.md
.planning/parity/plans/open-questions.md
docs/WORKFLOW.md
```

Any file outside this list in the scoped diff is an automatic review REJECT.
`internal/**` (incl. `routes_admin.go`/`internal/mcp/*`/`guard.go` — w6-l holds NO
serial slot and adds no MCP/skills Go), `ui/package.json`,
`ui/e2e/mocks/fixture.ts`, `ui/e2e/mocks/store.ts`,
`ui/e2e/mocks/handlers/index.ts`, `ui/e2e/mocks/seed/index.ts`, the mcp/skills
handler bodies + seeds (CONSUME unchanged), any sibling page-plan file, and any
frozen w6-a/b file are deliberately ABSENT — touching them is an automatic REJECT.
`ui/src/routeTree.gen.ts` is the ONE exception (route addition, §1.7) and IS allowed.
`ui/dist/**` is gitignored and must never appear. After merge, the three pages,
`ui/src/components/mcp/**`, `ui/src/lib/{skills-format,mcp-install}.ts`, and the new
skills spec become consume-only for later plans.

## 8. Escalations / cross-track dependencies

- **No blocking dependency at authoring.** w6-a + w6-b + the page waves are merged
  (live tree @ b23bead: 16 primitives present, `apiFetch`/stores/mock harness + the
  mcp + skills handlers/seeds ALREADY REGISTERED per §1.1/§1.5, the routing-rules.tsx
  page template shipped). w6-l holds NO Go serial slot (ZERO new Go) and no frozen
  exception. Fully unblocked for page wave 2.
- **ESCALATION-1 (RESOLVED at authoring — MCP/skills backends ABSENT, contradicting
  the MAP "MCP gateway backend in-tree" claim, §1.2):** `internal/mcp/` is a Phase-1
  PLACEHOLDER (doc.go + a no-op `TestPackageCompiles`); NO `/api/mcp/*` or
  `/api/skills` admin routes; NO `internal/admin/{mcp,skills}.go`. The ONLY in-tree
  MCP reference is `guard.go:46` (a forward-looking LOCAL_ONLY guard, no live route).
  - **1a (MCP clients/instances + marketplace, PAR-UI-130 `/mcp` + PAR-UI-054):**
    no Go `/api/mcp/clients|instances`. Ship variant-HAVE vs the registered mcp MOCK
    (browse `/api/mcp/clients`, install `POST /api/mcp/instances`, §1.6). Serial Go
    follow-up: implement the `internal/mcp/` gateway + admin `GET /api/mcp/clients`,
    `GET/POST/DELETE /api/mcp/instances`, instance OAuth `…/auth/start`. NOT in w6-l.
  - **1b (MCP tools/tool-groups, PAR-UI-130 `/mcp/tools`):** no Go
    `/api/mcp/tools|tool-groups`. Variant-HAVE vs the MOCK. Serial Go follow-up:
    `GET /api/mcp/tools`, `POST …/{name}/execute`, tool-groups CRUD over the MCP
    gateway. NOT in w6-l.
  - **1c (skills, PAR-UI-020):** no Go `/api/skills` + no `internal/admin/skills.go`.
    Variant-HAVE vs the registered `/api/skills` MOCK. Serial Go follow-up: a real
    `GET /api/skills` endpoint (over a skills catalog/store). NOT in w6-l.
- **ESCALATION-2 (CONDITIONAL — routeTree regen, §1.7):** if `npm run build`/dev does
  not regenerate `ui/src/routeTree.gen.ts` to register `/skills` (plugin not running,
  stale tree), STOP and ESCALATE; resolve by regeneration, NEVER by a manual edit of
  the generated file. (Same as w6-i ESCALATION-6.)
- **ESCALATION-3 (CONDITIONAL — shared/foundation mock pressure):** if any surface
  genuinely needs an `index.ts`/`seed/index.ts`/`store.ts`/`fixture.ts`/mcp/skills
  handler-body/seed edit (it must not — every route + seed is already registered,
  §1.5, and the mocks are consumed unchanged, §1.4), OR if a within-mock
  inconsistency breaks THIS cluster's specs in a way that would ripple to a non-w6-l
  spec, STOP and ESCALATE (orchestrator serializes the shared change) — do not fudge
  the mock, do not add Go, do not edit any foundation mock.
- **PAR-UI-054 marketplace contract (RESOLVED — remap, §1.6):** the 9router ref's
  `/api/cli-tools/cowork-mcp-registry` + `…cowork-mcp-tools` endpoints exist in
  NEITHER g0router Go NOR the g0router mock. **Decision:** the marketplace browses
  `/api/mcp/clients` and installs via `POST /api/mcp/instances` (the in-tree mcp
  mock) — NO new cli-tools registry mock is added. Serial follow-up (orchestrator,
  NOT w6-l): if a true marketplace registry is later wanted, add a real
  `/api/cli-tools/cowork-mcp-registry`-equivalent Go endpoint + mock.
- **PAR-UI-081 dependency (RESOLVED):** already HAVE from w6-a (apiFetch = queryFn
  adapter, `open-questions.md:6`). w6-l consumes `apiFetch`; no `QueryClientProvider`
  mount; the MAP-decision-2 TanStack-Query provider wiring (if ever wanted) is an
  orchestrator serial follow-up, NOT w6-l.
- **MAP-assumption follow-up (record, non-blocking):** the WAVE-6-MAP w6-l row states
  "MCP gateway backend in-tree". §1.2 VERIFIED this FALSE (`internal/mcp/` is a
  placeholder; no MCP/skills admin routes). Record in WORKFLOW.md + open-questions so
  the orchestrator updates the MAP and schedules the three serial Go follow-ups
  (ESC-1a..1c).
```
