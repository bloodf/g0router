# Micro-plan w6-h — Combos + routing cluster (UI-only, ZERO new Go)

```
wave: 6
plan: w6-h
status: READY (rev 1 — authored against merged w6-a + w6-b, live tree @ e2ef375)
runs: page wave 1, AFTER w6-b MERGE (consumes frozen ui/src/components/ui/*) and
  AFTER w6-a MERGE (consumes apiFetch/ApiError, stores, the e2e mock harness).
  Disjoint from w6-c/w6-e/w6-g/w6-i (different routes/components/specs). Holds NO
  Go serial slot (zero new Go — see §1.2 / go-serial-slot below).
branch: main (direct commits, per repo convention)
commit-prefix: phase-1/w6-h:
ref-source: 9router frozen @ 827e5c3 —
  src/app/(dashboard)/dashboard/combos/page.js (combos page WITH @dnd-kit reorder
  of the member-model list: DndContext+SortableContext+arrayMove, page.js:4-7,
  411,520-543), src/shared/components/ComboFormModal.js (combo create/edit form),
  plus the routing/aliases/model-limits surfaces (9router exposes these via combos
  + alias repos — see §1.4 ref note). g0router ports each adapted to React 19 +
  TanStack Router + Tailwind v4 + frozen w6-a/w6-b primitives.
base: <base> = git rev-parse HEAD recorded at P0 (expected e2ef375 at authoring;
  if main advanced, record the actual SHA and substitute everywhere §5 says <base>)
freeze-exception: NONE. The header.tsx / __root.tsx / main.tsx exceptions are
  SPENT (w6-b wiring, w6-c logout-slot). w6-h touches no frozen w6-a/w6-b file.
go-serial-slot: NONE. w6-h adds ZERO Go and therefore does NOT take the
  routes_admin.go serial slot (which passes w6-pre → w6-d → w6-e → w6-j, MAP
  §Cross-cutting). routes_admin.go is NOT touched. See §1.2 (combos backend
  exists; routing-rules/model-limits/aliases admin endpoints are ABSENT — that is
  an ESCALATION, §8, NOT new Go in this plan, because the MAP assigns NO Go to
  w6-h).
```

---

## 1. Scope — PAR rows

### Rows this plan closes

| Row | Claim | Target state after w6-h |
|---|---|---|
| PAR-UI-010 | Route `/combos` combos management page (with member reordering) | HAVE (variant — flat route; @dnd-kit reorder of combo member list, §1.3/§1.5) |
| PAR-UI-050 | ComboFormModal (create/edit a combo; DnD member list) | HAVE (§1.3) |
| PAR-UI-091 | Combo API: list combos | HAVE (variant — real Go `GET /api/combos`; DTO divergence §1.2 / §8 ESCALATION-1) |
| PAR-UI-092 | Combo API: create combo | HAVE (variant — real Go `POST /api/combos`; §1.2 / §8 ESCALATION-1) |
| PAR-UI-093 | Combo API: update combo | HAVE (variant — real Go `PUT /api/combos/{name}` keyed by NAME not id; §1.2 / §8 ESCALATION-1) |
| PAR-UI-094 | Combo API: delete combo | HAVE (variant — real Go `DELETE /api/combos/{name}`; §1.2 / §8 ESCALATION-1) |
| PAR-UI-116 | Aliases page (model alias management) | HAVE (variant — mock-contract HAVE; NO admin Go `/api/aliases` endpoint — §1.2 / §8 ESCALATION-2) |
| PAR-PR-339 | g0router combo list UI (W5 carry-forward) | HAVE (the combos page list view, §1.3) |
| PAR-UI-130 (subset) | g0router routes `/routing-rules`, `/model-limits` | HAVE (variant — mock-contract HAVE; NO Go backend for either — §1.2 / §8 ESCALATION-3) |

8 PAR-UI rows + PAR-PR-339 + the PAR-UI-130 `/routing-rules`+`/model-limits`
subset. Matches WAVE-6-MAP w6-h row (~line 134) and §Ownership w6-h (~line 183-186).
The aliases page closes PAR-UI-116; the `/aliases` route is a g0router page covered
by the `aliases.spec.ts` contract (recorded with PAR-UI-116, no separate PAR-UI-130
entry — `/aliases` is not in the PAR-UI-130 list at MAP line 86, it is the 9router
model-alias surface).

### 1.1 Preconditions already satisfied by merged waves (evidence)

- Route STUBS exist, must be REWRITTEN (not created — so no new route file, so
  `routeTree.gen.ts` does NOT change; MAP decision 6 / §1.7). All four render only
  an `<h1>`:
  `ui/src/routes/combos.tsx:1-9` (`<h1>Combos</h1>`),
  `ui/src/routes/routing-rules.tsx:1-9` (`<h1>Routing Rules</h1>`),
  `ui/src/routes/model-limits.tsx:1-9` (`<h1>Model Limits</h1>`),
  `ui/src/routes/aliases.tsx:1-9` (`<h1>Aliases</h1>`).
- Frozen primitives this plan CONSUMES (w6-b, never edited; 16 present): `Button`
  `ui/src/components/ui/button.tsx`; `Input` `ui/src/components/ui/input.tsx`;
  `Select` `ui/src/components/ui/select.tsx`; `Card`/`CardHeader`/`CardTitle`/
  `CardContent` `ui/src/components/ui/card.tsx`; `Modal`
  `ui/src/components/ui/modal.tsx` (controlled `open`/`onClose`, traffic lights,
  Escape, overlay, scroll-lock); `ConfirmModal`
  `ui/src/components/ui/confirm-modal.tsx`; `Badge`
  `ui/src/components/ui/badge.tsx`; `Toggle` `ui/src/components/ui/toggle.tsx`;
  `SegmentedControl` `ui/src/components/ui/segmented-control.tsx`; `ProviderIcon`
  `ui/src/components/ui/provider-icon.tsx`; `Loading`/`Spinner`/`Skeleton`/
  `CardSkeleton` `ui/src/components/ui/{loading,skeleton}.tsx`; `Tooltip`
  `ui/src/components/ui/tooltip.tsx`; `Pagination`
  `ui/src/components/ui/pagination.tsx`.
- Frozen foundation this plan CONSUMES (w6-a, never edited): `apiFetch`
  `ui/src/lib/api.ts:19` + `ApiError` `ui/src/lib/api.ts:3`; toast via
  `useNotificationStore.push` `ui/src/stores/notification.ts`; Material Symbols
  `ui/src/index.css:3`.
- **@dnd-kit dependency present (no additions allowed — VERIFIED):** all four
  packages the combos reorder needs are installed in `ui/package.json`:
  `@dnd-kit/core@^6.3.1` (`package.json:18`), `@dnd-kit/modifiers@^9.0.0`
  (`package.json:19`), `@dnd-kit/sortable@^10.0.0` (`package.json:20`),
  `@dnd-kit/utilities@^3.2.2` (`package.json:21`). The reference combos page uses
  exactly this set — `@dnd-kit/core` (`DndContext`, `closestCenter`,
  `KeyboardSensor`, `PointerSensor`, `useSensor`, `useSensors`), `@dnd-kit/sortable`
  (`arrayMove`, `SortableContext`, `sortableKeyboardCoordinates`, `useSortable`,
  `verticalListSortingStrategy`), `@dnd-kit/utilities` (`CSS`), `@dnd-kit/modifiers`
  (`restrictToVerticalAxis`, `restrictToParentElement`) — ref
  `combos/page.js:4-7`. No `package.json` edit, no new dep. (Absence would be an
  ESCALATION, §8; it is not.)
- UI types this plan CONSUMES (`ui/src/lib/types.ts`): `Combo` (`types.ts:70`,
  `{id,name,strategy,steps:Array<{provider,model}>,is_active}`); `RoutingRule`
  (`types.ts:232`, `{id,name,priority,cond_field,cond_operator,cond_value,
  target_provider,is_active,created_at}`); `ModelLimit` (`types.ts:170`,
  `{id:number,model,max_tokens,max_rpm,allowed_key_ids[],created_at}`); `Alias`
  (`types.ts:22`, `{id,alias,provider,model}`).
- **e2e mock harness present + registered (CONSUME-ONLY, §1.4 / §8):** handlers
  `ui/e2e/mocks/handlers/{combos,routing-rules,model-limits,aliases}.ts` registered
  at `ui/e2e/mocks/handlers/index.ts:14,15,17,26` (and called at
  `index.ts:47,48,50,59`); seeds `ui/e2e/mocks/seed/{combos,routing-rules,
  model-limits,aliases}.ts` exported at `ui/e2e/mocks/seed/index.ts:8,9,11,19`.
  Mock paths/shapes enumerated in §1.4.
- Existing acceptance specs (the contract — §1.3 thin-smoke interpretation):
  `ui/e2e/combos.spec.ts:9-12` (1 test: `/combos` body contains "Combos"),
  `ui/e2e/routing-rules.spec.ts:9-12` (1 test: `/routing-rules` body contains
  "Routing"), `ui/e2e/model-limits.spec.ts:9-12` (1 test: `/model-limits` body
  contains "Model Limits"), `ui/e2e/aliases.spec.ts:9-12` (1 test: `/aliases` body
  contains "Aliases"). Login helper `ui/e2e/helpers.ts:3` drives `#username`/
  `#password`.

### 1.2 Go contract for the four domains (file:line evidence — w6-h adds ZERO Go)

The MAP assigns NO Go to w6-h (line 134: "combo API endpoints (PAR-UI-091..094)
already covered by w4-e backend — no new Go needed"; §Ownership line 186: "No
Go"). The worker VERIFIES each needed endpoint and, where one is genuinely absent,
ESCALATES (§8) rather than adding Go (which the MAP forbids for this plan).

**Combos — backend EXISTS, but DTO + key diverge (ESCALATION-1):**
- `GET /api/combos` → `h.ListCombos` (`routes_admin.go:85`)
- `POST /api/combos` → `h.CreateCombo` (`routes_admin.go:86`)
- `PUT /api/combos/{name}` → `h.UpdateCombo` (`routes_admin.go:87`) — **keyed by
  NAME, not id**
- `DELETE /api/combos/{name}` → `h.DeleteCombo` (`routes_admin.go:88`)
- Response shape `comboResponse` (`internal/admin/combos.go:15-21`):
  `{name, models:[]string}` — a flat NAME + string-array-of-models. **This
  DIVERGES from the mock/UI `Combo` type** (`{id, name, strategy,
  steps:[{provider,model}], is_active}`, §1.1 / mock `seed/combos.ts:5-6`): real Go
  has NO `id` (key is `name`), NO `strategy`, NO `is_active`, and `models` is
  `["llama-3-70b", …]` not `[{provider,model}]`. `CreateCombo` body is
  `{name, models:[]string}` (`combos.go:40-43`), name validated against
  `^[a-zA-Z0-9_.-]+$` (`combos.go:13,48`); `UpdateCombo`/`DeleteCombo` route on
  `{name}` (`combos.go:65-67,89-91`). This is the divergence resolved in §1.4 / §8
  ESCALATION-1. PAR-UI-091..094 ship as variant-HAVE against the corrected mock; if
  closing the runtime gap requires shaping the real Go to the UI `Combo` type, that
  is NEW Go → forbidden here → serial follow-up (§8).

**Aliases — store layer EXISTS, but NO admin HTTP endpoint (ESCALATION-2):**
- `store.ListAliases()` (`internal/store/aliases.go:64-65`) + `ModelAlias` record
  exist and are consumed by the OpenAI route adapter
  (`internal/server/routes_openai.go:205-206` `aliasModelsAdapter.ListAliasNames`).
- **There is NO `/api/aliases` admin route** (`grep '/api/aliases'
  internal/server/routes_admin.go` → empty) and NO `internal/admin/aliases.go`
  handler. The admin CRUD surface the page needs is mock-only. PAR-UI-116 ships as
  variant-HAVE against the `/api/aliases` MOCK; the admin Go endpoint is a serial
  follow-up (§8 ESCALATION-2), NOT in w6-h.

**Routing-rules — NO backend at all (ESCALATION-3a):**
- `grep -rE 'routing-rules|RoutingRule' internal/` → ZERO matches (no store, no
  admin, no route). `/api/routing-rules` is mock-only. PAR-UI-130 `/routing-rules`
  subset ships as variant-HAVE against the mock; the full Go backend is a serial
  follow-up (§8 ESCALATION-3a), NOT in w6-h.

**Model-limits — NO backend at all (ESCALATION-3b):**
- `grep -rE 'model-limits|ModelLimit' internal/` → ZERO matches. `/api/model-limits`
  is mock-only. PAR-UI-130 `/model-limits` subset ships as variant-HAVE against the
  mock; the full Go backend is a serial follow-up (§8 ESCALATION-3b), NOT in w6-h.

Envelope: all admin responses use `writeData`/`writeError`
(`internal/admin/respond.go`) → snake_case `{data,error:{message}}`. The mock
`json`/`error` helpers (`ui/e2e/mocks/handlers/utils.ts`) mirror this
(`{data}` / `{error}`).

**Binding consequence:** w6-h adds ZERO Go. Three of the four domains
(routing-rules, model-limits, aliases-admin) have NO runtime Go backend, and the
fourth (combos) has a divergent DTO. All four pages ship as **variant-HAVE against
the corrected mock contract** (mocks mirror the *capabilities*); the runtime Go gaps
are recorded as serial follow-ups in §8 and `open-questions.md`. This mirrors the
w6-g quota disposition (PAR-UI-012 variant-HAVE against `/api/quota` mock with the
runtime Go deferred).

### 1.3 The combos DnD reorder — testing strategy (binding decision, the §1.x call)

**The surface.** The combos page lets the operator build/edit a combo whose
MEMBER LIST (the ordered `steps`/`models` a combo falls back through) is reorderable
by drag-and-drop. The reference implements this with `@dnd-kit` —
`DndContext`+`SortableContext` wrapping sortable rows, `onDragEnd` calling
`arrayMove(prev, oldIndex, newIndex)` (ref `combos/page.js:411,520-543`). The new
order is persisted on save via the combo update call.

**The problem.** Drag-and-drop reorder is (a) effectively untestable via the
`renderToString` unit harness (no DOM, no pointer events, w6-a `theme.test.ts`
precedent is string-render only) and (b) finicky/flaky under Playwright (pointer
drag against `@dnd-kit`'s sensor activation constraints and the `vite preview`
chrome is timing-sensitive).

**Decision (binding) — pure helper is the authoritative proof:**
1. **Extract the reorder logic into a PURE helper** owned by w6-h:
   `ui/src/lib/combo-order.ts` exporting `moveStep(steps, from, to)` (a thin,
   deterministic wrapper over the array-move algorithm — it MAY call
   `@dnd-kit/sortable`'s `arrayMove`, or implement the splice directly; either way
   it is a pure `(T[], number, number) => T[]` function with no DOM). The combos
   page's `onDragEnd` delegates to `moveStep`. **This pure helper is the
   AUTHORITATIVE reorder proof.**
2. **Unit-test `moveStep` deterministically** (`combo-order.test.ts`, vitest, no
   DOM): move down, move up, no-op (from===to), out-of-range index, immutability
   (input array not mutated), order preserved for untouched elements. Committed RED
   before `combo-order.ts`.
3. **e2e asserts the list RENDERS in order** (the deterministic part): after
   opening the ComboFormModal for a seeded combo, assert the member rows appear in
   the seed order (drag-handle present per row, `[data-testid="combo-step-row"]` in
   document order matching the seed). This is harness-stable.
4. **e2e attempts a reorder ONLY via the keyboard sensor / programmatic path if
   trivially green**: `@dnd-kit` ships a `KeyboardSensor` (ref uses it,
   `combos/page.js:4`) — the e2e MAY focus the first drag handle, press
   `Space`+`ArrowDown`+`Space`, and assert the rows reordered. If keyboard DnD is
   flaky/infeasible under the harness, the e2e INSTEAD asserts the **persisted-order
   POST/PUT fires with the reordered members** (intercept the combo update call via
   `page.route` and assert the request body's member order), which proves the
   reorder wiring end-to-end without a live pointer drag. **Pick whichever is green;
   the unit test (point 2) remains the authoritative correctness proof regardless.**
5. **Escalation (§8 ESCALATION-4):** if BOTH the keyboard-DnD e2e AND the
   persisted-order-POST e2e are infeasible under `vite preview` + `@dnd-kit` (e.g.
   sensors never activate headless), STOP and ESCALATE — DROP the live-DnD e2e
   assertion, keep the render-in-order e2e (point 3) + the pure-helper unit
   (point 2) as the binding proof, and record the disposition in WORKFLOW.md. The
   plan does NOT block on live DnD e2e; the pure reorder-helper unit test is the
   floor.

PAR-PR-339 (combo list UI) is the combos page's list view (point 3's render
assertion covers it). PAR-UI-050 (ComboFormModal) is proven by point 3 (modal opens,
member rows render, save fires the update with member order).

### 1.4 Mock paths/shapes (binding interpretation — CONSUME; correct BODY only on Go conflict)

The four mock handlers (`ui/e2e/mocks/handlers/`) model a uniform REST-by-id CRUD:

| Domain | Mock routes (file) | Mock list shape (seed) | Real Go (§1.2) | Resolution |
|---|---|---|---|---|
| Combos | `GET/POST /api/combos`; `GET/PUT/DELETE /api/combos/{id}` (`combos.ts:6-37`) | `{id,name,strategy,steps:[{provider,model}],is_active}` (`seed/combos.ts:5-6`) | `GET/POST /api/combos`; `PUT/DELETE /api/combos/{name}`; DTO `{name,models:[]string}` (§1.2) | **DTO + key divergence.** Page consumes the mock contract for the page surface (id, strategy, steps, is_active). The real Go does NOT serve this shape and keys by `name`. **Decision (§8 ESCALATION-1, RESOLVED):** ship PAR-UI-091..094 as variant-HAVE against the mock (CONSUMED as-is for list/create/update/delete by-id); the runtime gap (real Go shape ≠ mock) is a serial Go follow-up. CONSUME the mock body UNCHANGED unless T6 finds a within-mock inconsistency. NO existing-Go edit, NO new Go. |
| Routing rules | `GET/POST /api/routing-rules`; `GET/PUT/DELETE /api/routing-rules/{id}` (`routing-rules.ts:6-37`) | `{id,name,priority,cond_field,cond_operator,cond_value,target_provider,is_active,created_at}` (`seed/routing-rules.ts:5-6`) | NONE (§1.2) | **No Go.** Page consumes the `/api/routing-rules` MOCK; variant-HAVE; Go backend deferred (§8 ESCALATION-3a). CONSUME the mock body unchanged. |
| Model limits | `GET/POST /api/model-limits`; `GET/PUT/DELETE /api/model-limits/{id}` (`model-limits.ts:6-37`) | `{id:number,model,max_tokens,max_rpm,allowed_key_ids[],created_at}` (`seed/model-limits.ts:5-6`) | NONE (§1.2) | **No Go.** Page consumes the `/api/model-limits` MOCK; variant-HAVE; Go backend deferred (§8 ESCALATION-3b). CONSUME the mock body unchanged. |
| Aliases | `GET/POST /api/aliases`; `GET/PUT/DELETE /api/aliases/{id}` (`aliases.ts:6-37`) | `{id,alias,provider,model}` (`seed/aliases.ts:5-7`) | store-only, NO admin endpoint (§1.2) | **No admin Go.** Page consumes the `/api/aliases` MOCK; variant-HAVE; admin Go endpoint deferred (§8 ESCALATION-2). CONSUME the mock body unchanged. |

**Binding rule (MAP decision 4):** where mock and real Go disagree, the real Go
wins and the mock body is corrected in the SAME plan. BUT w6-h adds ZERO Go and
three of four domains have no runtime Go to mirror; therefore the mocks are the
binding capability contract for THIS plan, CONSUMED unchanged, and the runtime gaps
are escalated (§8). w6-h corrects a handler BODY ONLY if T1/T6 finds a within-mock
inconsistency that breaks this cluster's specs — NEVER the `handlers/index.ts`
registration, NEVER the seed files, NEVER the seed `index.ts`. If correcting a body
would break a non-w6-h spec, STOP and ESCALATE (§8 ESCALATION-5).

### 1.5 Variant notes (recorded HAVE rationale)

- **PAR-UI-010 combos / PAR-UI-050 ComboFormModal**: flat route `/combos` (MAP
  decision 1); ComboFormModal is an in-page modal (not a nested route), DnD member
  reorder per §1.3. Recorded variant-HAVE.
- **PAR-UI-091..094 combo API**: variant — the real Go combos endpoints exist but
  with a divergent DTO/key (§1.2/§1.4); page consumes the mock contract; runtime Go
  shaping deferred. Recorded variant-HAVE.
- **PAR-UI-116 aliases / PAR-UI-130 `/routing-rules` / `/model-limits`**: variant —
  mock-contract HAVE; no runtime Go backend (§1.2); Go deferred to serial
  follow-ups (§8). Recorded variant-HAVE.
- **Pages render inside app chrome**: `__root.tsx` wraps every route in
  Sidebar+Header; pages render in `<Outlet>`; specs assert page content with chrome
  present (w6-c/w6-e/w6-g precedent). Accepted constraint, not a gap.
- **Data layer = plain `apiFetch` + React state, NOT TanStack Query**: `QueryClient`
  is NOT mounted (`__root.tsx`/`main.tsx` FROZEN, w6-a); w6-h fetches via `apiFetch`
  in `useEffect` with local `useState` (w6-e/w6-g precedent). PAR-UI-081 is already
  HAVE from w6-a (open-questions.md:6); w6-h consumes `apiFetch` and does NOT mount
  a provider, NOT edit any frozen file. Accepted constraint.

### 1.6 `routeTree.gen.ts` is NOT touched

All four routes already exist as stubs (§1.1); rewriting their component bodies does
not change the route tree, and no new route file is added (ComboFormModal etc. are
in-page modals/components, not routes). Therefore `ui/src/routeTree.gen.ts` is
UNCHANGED by w6-h (MAP decision 6; w6-h is NOT the wave-1 new-route plan — w6-i is).
If a build incidentally reformats it, that is an ESCALATION (§8), not an in-plan edit.

### NOT in scope (explicit)

- **No Go changes.** ALL of `internal/` is FORBIDDEN. The combos backend exists
  (divergent DTO, §1.2); routing-rules/model-limits/aliases-admin backends are
  ABSENT — those are ESCALATIONS (§8), NEVER an in-plan Go edit (the MAP assigns no
  Go to w6-h). No new `internal/admin/{routing,model_limits,aliases}.go`.
- **No new route FILES** — only the four existing stubs are rewritten;
  `routeTree.gen.ts` untouched (§1.6). ComboFormModal/forms are in-page components.
- **No dependency additions** — all four `@dnd-kit/*` packages already installed
  (§1.1); every import resolves to installed packages or w6-a/w6-b outputs. NO
  `package.json` / lockfile edit.
- **No edits to any frozen w6-a/w6-b file** — no `__root.tsx`, `main.tsx`, layout
  components, `ui/src/components/ui/*`, `ui/src/stores/*`, `ui/src/lib/api.ts`,
  `ui/src/lib/utils.ts`, `ui/src/lib/auth.ts`, `ui/src/providers/*`,
  `ui/src/routes/{login,callback}.tsx`. No header exception remains (SPENT).
- **No `QueryClientProvider` mount** (§1.5) — plain `apiFetch`; PAR-UI-081 already
  HAVE (w6-a).
- **No mocks index/seed/fixture edits** — `mocks/handlers/index.ts`,
  `mocks/seed/index.ts`, `mocks/seed/*`, `mocks/fixture.ts`, `mocks/store.ts` are
  untouched; w6-h corrects a handler BODY only if a within-mock inconsistency forces
  it (§1.4 / §8), never the index/seed/fixture.
- **No other e2e specs** beyond `{combos,routing-rules,model-limits,aliases}.spec.ts`
  (+ matching mock handler bodies only if §8 forces).
- **No SSE/streaming** — all four surfaces are request/response CRUD.
- **No providers/connections/models (w6-e), no usage/quota/pricing (w6-g), no
  virtual-keys/endpoint (w6-f), no settings (w6-j).**
- **No real outbound network** — all reads are mock-intercepted.

---

## 2. Precondition checks

Run all before any edit; abort and report to orchestrator on any failure.

```bash
# P0 — clean tree; record <base>
git status --porcelain     # must be empty (untracked tooling artifacts must be
                           # gitignored; worker uses explicit `git add <file>`,
                           # never `git add -A`, per w6-b runtime disposition;
                           # NOTE ui/dist/** build artifacts are gitignored — do
                           # not stage them)
git rev-parse HEAD         # record as <base> for §5 (expected e2ef375)

# P1 — w6-b primitives present and frozen (consumed)
ls ui/src/components/ui/*.tsx | grep -v test | wc -l    # = 16 (w6-b set intact)
grep -n "export function Modal\|export interface ModalProps" ui/src/components/ui/modal.tsx
grep -n "ConfirmModal" ui/src/components/ui/confirm-modal.tsx
grep -n "SegmentedControl" ui/src/components/ui/segmented-control.tsx
grep -n "ProviderIcon" ui/src/components/ui/provider-icon.tsx

# P2 — w6-a foundation present and frozen (consumed; PAR-UI-081 already HAVE)
grep -n "export async function apiFetch" ui/src/lib/api.ts
grep -n "export class ApiError" ui/src/lib/api.ts
grep -n "push:" ui/src/stores/notification.ts
grep -rn "QueryClientProvider\|QueryClient" ui/src/routes/__root.tsx ui/src/main.tsx ; echo "^ expect EMPTY (not mounted; do NOT mount)"

# P3 — @dnd-kit present (NO additions allowed; absence ⇒ ESCALATION §8)
grep -n '"@dnd-kit/core"' ui/package.json        # ^6.x   (line 18)
grep -n '"@dnd-kit/sortable"' ui/package.json     # ^10.x  (line 20)
grep -n '"@dnd-kit/modifiers"' ui/package.json    # ^9.x   (line 19)
grep -n '"@dnd-kit/utilities"' ui/package.json    # ^3.x   (line 21)

# P4 — the four route stubs are still bare (safe to rewrite); no new dirs yet
grep -n "<h1>Combos</h1>"       ui/src/routes/combos.tsx
grep -n "<h1>Routing Rules</h1>" ui/src/routes/routing-rules.tsx
grep -n "<h1>Model Limits</h1>" ui/src/routes/model-limits.tsx
grep -n "<h1>Aliases</h1>"      ui/src/routes/aliases.tsx
test ! -d ui/src/components/combos && echo "combos components dir absent (good)"
test ! -d ui/src/components/routing && echo "routing components dir absent (good)"
test ! -e ui/src/lib/combo-order.ts && echo "combo-order helper absent (good)"

# P5 — e2e mock harness present + registered (CONSUME; correct bodies only §1.4)
grep -n "registerCombosHandlers\|registerRoutingRulesHandlers\|registerModelLimitsHandlers\|registerAliasesHandlers" ui/e2e/mocks/handlers/index.ts
grep -n "seedCombos\|seedRoutingRules\|seedModelLimits\|seedAliases" ui/e2e/mocks/seed/index.ts

# P6 — Go reality: combos exists (divergent DTO); the other three ABSENT (§1.2/§8)
grep -n "/api/combos" internal/server/routes_admin.go            # 4 lines (85-88), {name} key
grep -n "comboResponse\|func (h \*Handlers) ListCombos" internal/admin/combos.go  # {name,models[]}
grep -rnE '/api/routing-rules|RoutingRule' internal/ ; echo "^ expect EMPTY (no Go — ESCALATION-3a)"
grep -rnE '/api/model-limits|ModelLimit' internal/ ; echo "^ expect EMPTY (no Go — ESCALATION-3b)"
grep -n '"/api/aliases"' internal/server/routes_admin.go ; echo "^ expect EMPTY (no admin Go — ESCALATION-2; store.ListAliases exists in internal/store/aliases.go)"

# P7 — routes_admin.go serial slot: w6-h does NOT take it (zero Go)
echo "w6-h adds ZERO Go → does NOT touch internal/server/routes_admin.go and does NOT hold the serial slot (which passes w6-pre→w6-d→w6-e→w6-j)."

# P8 — harness green at base
cd ui && npx playwright test e2e/combos.spec.ts e2e/routing-rules.spec.ts e2e/model-limits.spec.ts e2e/aliases.spec.ts
# Record base result: stubs render only <h1> (which already contains the asserted
# text), so the four text-contains assertions PASS at base (the <h1> + sidebar
# chrome carry "Combos"/"Routing"/"Model Limits"/"Aliases"). The RED arc is the
# ADDED assertions in §4 T1 (lists/forms/DnD), which FAIL at base. Record exact
# pass/fail per spec in WORKFLOW.md.
cd ui && npm run build                               # exit 0
cd ui && npx vitest run src/                         # exit 0 (existing units green)
go test ./... && go vet ./...                        # exit 0 (Go untouched-green)
```

---

## 3. Exclusive file ownership

After w6-h merges, all CREATE files below are owned by w6-h; later plans consume,
never edit (MAP decision 7).

**CREATE — routes (REWRITE existing stubs; no new route files, §1.6):**

| File | Exports / contract |
|---|---|
| `ui/src/routes/combos.tsx` (REWRITE) | `Route=createFileRoute("/combos")`; `CombosPage`: on mount `apiFetch("/api/combos")` → combo list (PAR-PR-339) of cards/rows (name, strategy `Badge`, member count, `is_active` `Toggle`); "New combo" + per-combo Edit open `<ComboFormModal>`; Delete via `ConfirmModal` → `DELETE /api/combos/{id}`. Header text contains "Combos". |
| `ui/src/routes/routing-rules.tsx` (REWRITE) | `Route=createFileRoute("/routing-rules")`; `RoutingRulesPage`: `apiFetch("/api/routing-rules")` → table (name, priority, `cond_field`/`cond_operator`/`cond_value`, `target_provider` via `ProviderIcon`, `is_active` `Toggle`); New/Edit via `<RoutingRuleModal>` (POST/PUT); Delete via `ConfirmModal`. Header text contains "Routing". |
| `ui/src/routes/model-limits.tsx` (REWRITE) | `Route=createFileRoute("/model-limits")`; `ModelLimitsPage`: `apiFetch("/api/model-limits")` → table (model, max_tokens, max_rpm, allowed_key_ids); New/Edit via `<ModelLimitModal>`; Delete via `ConfirmModal`. Header text contains "Model Limits". |
| `ui/src/routes/aliases.tsx` (REWRITE) | `Route=createFileRoute("/aliases")`; `AliasesPage`: `apiFetch("/api/aliases")` → table (alias, provider via `ProviderIcon`, model); New/Edit via `<AliasModal>`; Delete via `ConfirmModal`. Header text contains "Aliases". |

**CREATE — combos components (`ui/src/components/combos/`):**

| File | Exports / contract |
|---|---|
| `combo-form-modal.tsx` | `ComboFormModal` (PAR-UI-050) — consumes `Modal`+`Input`+`Select`+`Button`; name field, strategy `Select`, and the DnD-reorderable MEMBER LIST (steps) using `@dnd-kit/core` `DndContext`+`@dnd-kit/sortable` `SortableContext`/`useSortable`/`verticalListSortingStrategy` + `@dnd-kit/modifiers` `restrictToVerticalAxis`/`restrictToParentElement` + `@dnd-kit/utilities` `CSS`; `onDragEnd` delegates to `moveStep` from `lib/combo-order.ts` (§1.3); each member row carries `[data-testid="combo-step-row"]` + a drag handle. Save → POST `/api/combos` (new) or PUT `/api/combos/{id}` (edit) with members in current order. Ports ref `ComboFormModal.js` + the reorder block from `combos/page.js:411,520-543`. |
| `combo-list.tsx` | `ComboList` (PAR-PR-339) — the combos list view (cards/rows) consumed by `combos.tsx`. |

**CREATE — routing components (`ui/src/components/routing/`):**

| File | Exports / contract |
|---|---|
| `routing-rule-modal.tsx` | `RoutingRuleModal` — `Modal`+`Input`/`Select`; fields name, priority, cond_field, cond_operator, cond_value, target_provider, is_active; POST/PUT `/api/routing-rules`. |
| `model-limit-modal.tsx` | `ModelLimitModal` — `Modal`+`Input`; fields model, max_tokens, max_rpm, allowed_key_ids; POST/PUT `/api/model-limits`. |
| `alias-modal.tsx` | `AliasModal` — `Modal`+`Input`/`Select`; fields alias, provider, model; POST/PUT `/api/aliases`. |

(Modals are DOM-heavy; their coverage is the e2e open/save assertions, not units —
same disposition as w6-e/w6-g modal components.)

**CREATE — lib (`ui/src/lib/combo-order.ts`, NEW — the pure reorder helper, §1.3):**

| Export | Contract |
|---|---|
| `moveStep<T>(steps: T[], from: number, to: number): T[]` | Pure, deterministic array-move (MAY wrap `@dnd-kit/sortable`'s `arrayMove` or splice directly). Returns a NEW array (input not mutated); `from===to` or out-of-range indices return an equivalent array unchanged; untouched elements keep relative order. No DOM. The AUTHORITATIVE reorder proof (§1.3). |

**CREATE — unit tests (vitest — pure logic reachable without a DOM):**

| File | Contents |
|---|---|
| `ui/src/lib/combo-order.test.ts` | ≥5 tests: move-down reorders correctly; move-up reorders correctly; `from===to` is a no-op; out-of-range index leaves order intact; immutability (original array unchanged). Committed RED before `combo-order.ts`. |

**MODIFY — e2e specs (the acceptance contract; CONSUME mocks, correct BODY only if §1.4/§8):**

| File | Change |
|---|---|
| `ui/e2e/combos.spec.ts` | KEEP the 1 existing test (body "Combos"). ADD RED: combo list rows render from seed (≥2 rows, names visible); open ComboFormModal (traffic lights `[data-testid="modal-traffic-lights"]`); member rows render in seed order (`[data-testid="combo-step-row"]`); a reorder is proven per §1.3 (keyboard-DnD reorder OR persisted-order PUT body intercept — whichever is green); delete asks `ConfirmModal`. |
| `ui/e2e/routing-rules.spec.ts` | KEEP the 1 existing test. ADD RED: rule rows render (name + priority + target_provider); open RoutingRuleModal; save fires POST; delete via ConfirmModal. |
| `ui/e2e/model-limits.spec.ts` | KEEP the 1 existing test. ADD RED: limit rows render (model + max_tokens + max_rpm); open ModelLimitModal; save fires POST. |
| `ui/e2e/aliases.spec.ts` | KEEP the 1 existing test. ADD RED: alias rows render (alias + provider + model); open AliasModal; save fires POST; delete via ConfirmModal. |
| `ui/e2e/mocks/handlers/{combos,routing-rules,model-limits,aliases}.ts` | CONSUME as-is (§1.4). CORRECT a handler BODY only if T1/T6 finds a within-mock inconsistency that breaks this cluster's specs — never the index, never the seed, never the seed export list, never `fixture.ts`/`store.ts`. |

**FORBIDDEN:** everything else. Explicitly: ALL of `internal/` (combos backend
exists with divergent DTO; routing-rules/model-limits/aliases-admin backends absent
— ESCALATIONS §8, never an in-plan Go edit); ALL of `ui/src/components/ui/*` (w6-b
frozen); `ui/src/stores/*`, `ui/src/lib/api.ts`, `ui/src/lib/utils.ts`,
`ui/src/lib/auth.ts`, `ui/src/providers/*` (w6-a frozen);
`ui/src/routes/__root.tsx`, `ui/src/main.tsx`, `ui/src/components/layout/*`,
`ui/src/routes/{login,callback}.tsx`, `ui/src/components/auth/*`;
`ui/package.json` + lockfile; `ui/vite.config.ts`; `ui/playwright.config.ts`;
`ui/components.json`; `ui/src/index.css`; `ui/src/routeTree.gen.ts` (generated;
UNCHANGED §1.6); `ui/e2e/mocks/fixture.ts`, `ui/e2e/mocks/store.ts`,
`ui/e2e/mocks/handlers/index.ts`, `ui/e2e/mocks/seed/index.ts`,
`ui/e2e/mocks/seed/*` (foundation/wiring untouched); all other `ui/e2e/*.spec.ts`;
all other `ui/e2e/mocks/handlers/*` except the four named bodies above (only if §8
forces). `ui/dist/**` build artifacts are gitignored — never stage them.

---

## 4. TDD tasks

Cadence (strict): **no route/component/lib file may exist (or be rewritten beyond
its stub) before the failing test that covers it is committed.** `cd ui && npm run
build` green at EVERY commit (test files + red specs are never imported by
production code — w6-b/w6-c/w6-e/w6-g rationale). `go test ./... && go vet ./...`
stays untouched-green at EVERY commit (ZERO new Go). The four e2e specs stay RED
(on the ADDED assertions) from T1 until the implementation tasks green them; that is
the arc.

### T1 — STEP(a): extend the four e2e specs (commit RED)

Add the RED tests in §3 to `combos/routing-rules/model-limits/aliases.spec.ts`
(names are the acceptance contract, §5). CONSUME the four mock handlers/seeds
unchanged (§1.4). Resolve the §1.3 DnD-e2e approach NOW (keyboard-DnD vs
persisted-order PUT intercept) — write whichever is green; if both prove infeasible
at impl time, fall back to the render-in-order assertion + the pure-helper unit and
record ESCALATION-4 in WORKFLOW.md.

STEP(b): run all four specs — **record failure output** (no list rows, no modals,
no member rows/reorder). Commit RED:
`phase-1/w6-h: failing combos/routing-rules/model-limits/aliases e2e (TDD red)`.

**Mock-vs-reality gate**: re-confirm the §1.2 Go reality (combos DTO divergence;
routing-rules/model-limits/aliases backends ABSENT). These are ESCALATIONS (§8),
NOT in-plan Go. CONSUME the mocks as the capability contract; if a within-mock
inconsistency breaks a spec, correct ONLY the handler body; if correcting it would
break a non-w6-h spec, STOP and ESCALATE (§8 ESCALATION-5). NEVER add Go, NEVER
edit index/seed/fixture.

### T2 — STEP(a): unit test for the pure reorder helper (commit RED)

Write `ui/src/lib/combo-order.test.ts` per §3 (the authoritative DnD-reorder proof,
§1.3). Run `cd ui && npx vitest run src/lib/combo-order.test.ts` → FAILS (module
missing). Record failure. Commit RED:
`phase-1/w6-h: failing combo-order reorder helper unit test (TDD red)`.

### T3 — STEP(b): combo-order helper + combos page + ComboFormModal + ComboList

Implement `lib/combo-order.ts` (greens its unit). Implement `combo-list.tsx`,
`combo-form-modal.tsx` (DnD member list delegating to `moveStep`, §1.3); rewrite
`combos.tsx`. Gates: `npx vitest run src/lib/combo-order.test.ts` green;
`combos.spec.ts` green (list, modal, member render-in-order, the chosen reorder
proof, delete); `npm run build` green; `go test ./... && go vet ./...` untouched
green. Commit:
`phase-1/w6-h: combos page (list + ComboFormModal with @dnd-kit member reorder) + combo-order helper`.

### T4 — STEP(b): routing-rules + model-limits + aliases pages + modals

Implement `routing-rule-modal.tsx`, `model-limit-modal.tsx`, `alias-modal.tsx`;
rewrite `routing-rules.tsx`, `model-limits.tsx`, `aliases.tsx`. Gates:
`routing-rules.spec.ts`, `model-limits.spec.ts`, `aliases.spec.ts` green; all four
specs green; `npx vitest run src/` green; `npm run build` green;
`go test ./... && go vet ./...` untouched green. Commit:
`phase-1/w6-h: routing-rules + model-limits + aliases pages and modals`.

### T5 — full gates + closeout

```bash
cd ui && npm run build
cd ui && npx playwright test e2e/combos.spec.ts e2e/routing-rules.spec.ts e2e/model-limits.spec.ts e2e/aliases.spec.ts   # all green
cd ui && npx playwright test                             # full suite: no spec green-at-base may be red
cd ui && npx vitest run src/                             # all green incl new combo-order unit
go test ./... && go vet ./...                            # untouched-green (ZERO new Go)
```
Flip §1 matrix rows in `.planning/parity/matrix/9router-ui.md`: PAR-UI-010 → HAVE
(variant); PAR-UI-050 → HAVE; PAR-UI-091/092/093/094 → HAVE (variant, cite §1.2/§8
ESCALATION-1); PAR-UI-116 → HAVE (variant, cite §8 ESCALATION-2); PAR-UI-130
`/routing-rules`+`/model-limits` subset → HAVE (variant, cite §8 ESCALATION-3);
PAR-PR-339 → HAVE (cite §1.3). Update `docs/WORKFLOW.md` (record P8 base spec
observations, the §1.3 DnD-e2e disposition chosen, and the three runtime-Go
follow-ups §8). Append the §8 open items to `.planning/parity/plans/open-questions.md`.
Final commit:
`phase-1/w6-h: close — combos/routing/model-limits/aliases cluster; matrix flips`.
**w6-h holds NO serial slot — nothing to release.**

---

## 5. Binary acceptance criteria

All must hold; each is yes/no. `<base>` = the commit recorded at P0 (e2ef375 at
authoring). Diff gate is **w6-h commit-range-scoped** (§7) because page wave 1
plans commit to main concurrently.

**Test gates**
- `cd ui && npx playwright test e2e/combos.spec.ts` → exit 0, all tests pass
  (1 original + added), 0 skipped.
- `cd ui && npx playwright test e2e/routing-rules.spec.ts` → exit 0, all pass.
- `cd ui && npx playwright test e2e/model-limits.spec.ts` → exit 0, all pass.
- `cd ui && npx playwright test e2e/aliases.spec.ts` → exit 0, all pass.
- `cd ui && npx vitest run src/lib/combo-order.test.ts` → exit 0, ≥5 cases pass.
- `cd ui && npx vitest run src/` → exit 0 (all prior + new units green).
- `cd ui && npm run build` → exit 0.
- `go test ./... && go vet ./...` → exit 0 (Go untouched — ZERO new Go).

**TDD-order proof** — each impl file's covering test appears in an
earlier-or-equal commit:
```bash
# combo-order.ts after combo-order.test.ts
ct=$(git log --format=%ct --diff-filter=A -1 -- ui/src/lib/combo-order.test.ts)
cf=$(git log --format=%ct --diff-filter=A -1 -- ui/src/lib/combo-order.ts)
[ "$ct" -le "$cf" ] || echo "TDD VIOLATION: combo-order.ts"            # prints nothing
# e2e RED-extension commit precedes the combos page rewrite
sa=$(git log --format=%ct -1 --grep="failing combos/routing-rules/model-limits/aliases e2e")
ci=$(git log --format=%ct --diff-filter=M -1 -- ui/src/routes/combos.tsx)
[ "$sa" -le "$ci" ] || echo "TDD VIOLATION: combos.tsx before red spec"  # nothing
# e2e RED-extension precedes the routing/model-limits/aliases rewrites
ra=$(git log --format=%ct --diff-filter=M -1 -- ui/src/routes/routing-rules.tsx)
[ "$sa" -le "$ra" ] || echo "TDD VIOLATION: routing-rules.tsx before red spec"  # nothing
```

**Grep proofs**
```bash
grep -rn "/api/combos" ui/src/routes/combos.tsx ui/src/components/combos/combo-form-modal.tsx  # PAR-UI-091..094
grep -rn "@dnd-kit/core\|@dnd-kit/sortable\|DndContext\|SortableContext\|useSortable" ui/src/components/combos/combo-form-modal.tsx  # PAR-UI-050 DnD
grep -rn "moveStep" ui/src/components/combos/combo-form-modal.tsx ui/src/lib/combo-order.ts  # §1.3 reorder delegation
grep -rn "combo-step-row" ui/src/components/combos/combo-form-modal.tsx  # member-row test marker
test -f ui/src/components/combos/combo-list.tsx && echo OK              # PAR-PR-339
grep -rn "/api/routing-rules" ui/src/routes/routing-rules.tsx ui/src/components/routing/routing-rule-modal.tsx  # PAR-UI-130 routing-rules
grep -rn "/api/model-limits" ui/src/routes/model-limits.tsx ui/src/components/routing/model-limit-modal.tsx     # PAR-UI-130 model-limits
grep -rn "/api/aliases" ui/src/routes/aliases.tsx ui/src/components/routing/alias-modal.tsx                     # PAR-UI-116
grep -rn "export function moveStep\|export const moveStep" ui/src/lib/combo-order.ts  # §1.3 pure helper
# No QueryClientProvider introduced anywhere by w6-h:
! grep -rn "QueryClientProvider" ui/src/components/combos ui/src/components/routing ui/src/routes/combos.tsx ui/src/routes/routing-rules.tsx ui/src/routes/model-limits.tsx ui/src/routes/aliases.tsx && echo "no QueryClientProvider added OK"
# No new dep added:
grep -c '"@dnd-kit' ui/package.json    # = 4 (unchanged; verify package.json untouched via §7 diff)
```

**Negative / freeze proofs (w6-h commit-range — see §7)**
```bash
R="<first-w6-h>^..<last-w6-h>"
git diff $R --name-only -- internal/ | wc -l                            # = 0 (ZERO new Go; routes_admin.go untouched, no serial slot)
git diff $R --name-only -- ui/package.json ui/package-lock.json ui/vite.config.ts ui/playwright.config.ts ui/components.json ui/src/index.css | wc -l   # = 0 (no deps/config)
git diff $R --name-only -- ui/src/components/ui/ | wc -l                # = 0 (w6-b frozen)
git diff $R --name-only -- ui/src/stores/ ui/src/providers/ ui/src/lib/api.ts ui/src/lib/utils.ts ui/src/lib/auth.ts | wc -l   # = 0 (w6-a frozen)
git diff $R --name-only -- ui/src/routes/__root.tsx ui/src/main.tsx ui/src/components/layout/ ui/src/routes/login.tsx ui/src/routes/callback.tsx ui/src/components/auth/ | wc -l   # = 0
git diff $R --name-only -- ui/src/routeTree.gen.ts | wc -l             # = 0 (§1.6 unchanged)
git diff $R --name-only -- ui/e2e/mocks/fixture.ts ui/e2e/mocks/store.ts ui/e2e/mocks/handlers/index.ts ui/e2e/mocks/seed/index.ts | wc -l   # = 0 (foundation/wiring untouched)
git diff $R --name-only -- ui/e2e/mocks/seed/ | wc -l                   # = 0 (seed files untouched)
git diff $R --name-only -- 'ui/src/routes/' | grep -vE 'combos\.tsx|routing-rules\.tsx|model-limits\.tsx|aliases\.tsx' | wc -l   # = 0 (only the four stubs rewritten)
git diff $R --name-only -- ui/e2e/ | grep -vE 'combos\.spec\.ts|routing-rules\.spec\.ts|model-limits\.spec\.ts|aliases\.spec\.ts|mocks/handlers/(combos|routing-rules|model-limits|aliases)\.ts' | wc -l   # = 0 (no other spec; index/seed/fixture untouched)
git diff $R --name-only -- ui/e2e/mocks/handlers/ | grep -vE '(combos|routing-rules|model-limits|aliases)\.ts' | wc -l   # = 0 (only the four handler bodies, if §8 forces)
```

---

## 6. Out of scope (restated, binding)

ZERO Go changes — combos backend exists with a divergent DTO (§1.2 ESCALATION-1),
and routing-rules/model-limits/aliases-admin backends are ABSENT (§1.2
ESCALATION-2/-3); ALL four pages ship variant-HAVE against the corrected/consumed
MOCK contract with the runtime Go gaps deferred to serial follow-ups (§8), NEVER an
in-plan Go edit (MAP assigns no Go to w6-h) and w6-h holds NO serial slot; no
`QueryClientProvider` mount (§1.5; PAR-UI-081 already HAVE from w6-a); no new route
files / no `routeTree.gen.ts` change (§1.6); no dependency additions (all four
`@dnd-kit/*` already installed); no edits to any frozen w6-a/w6-b file (no header
exception remains — SPENT); no mocks `index.ts`/seed/`fixture.ts`/`store.ts` edits
(handler bodies only, if §8 forces); no other e2e specs; no SSE. Mock-vs-Go
divergence, an absent backend, a shared-handler-body correction that breaks a
non-w6-h spec, or an infeasible live-DnD e2e → escalate (§8), never patch Go, never
fudge a mock, never block on live DnD (the pure reorder-helper unit is the floor).

## 7. Diff-gate scope

Page-wave-1 plans (w6-c/e/g/h/i) commit to main concurrently, so a broad
`<base>..HEAD` range sweeps in sibling commits. The diff gate MUST be scoped to
w6-h's own commits. The orchestrator isolates them with:
`git log --oneline main | grep "^[0-9a-f]* phase-1/w6-h:" | awk '{print $1}'`
and runs `git diff <first-w6-h>^..<last-w6-h> -- [file list]` (same commit-range
scoping as w6-c §7 / w6-e §7 / w6-g §7 / w6-b §7).

`git diff <first-w6-h>^..<last-w6-h> --name-only` must be exactly a subset of:

```
ui/src/routes/combos.tsx
ui/src/routes/routing-rules.tsx
ui/src/routes/model-limits.tsx
ui/src/routes/aliases.tsx
ui/src/components/combos/combo-list.tsx
ui/src/components/combos/combo-form-modal.tsx
ui/src/components/routing/routing-rule-modal.tsx
ui/src/components/routing/model-limit-modal.tsx
ui/src/components/routing/alias-modal.tsx
ui/src/lib/combo-order.ts
ui/src/lib/combo-order.test.ts
ui/e2e/combos.spec.ts
ui/e2e/routing-rules.spec.ts
ui/e2e/model-limits.spec.ts
ui/e2e/aliases.spec.ts
ui/e2e/mocks/handlers/combos.ts          (body only, IF §8 forces; else untouched)
ui/e2e/mocks/handlers/routing-rules.ts   (body only, IF §8 forces; else untouched)
ui/e2e/mocks/handlers/model-limits.ts    (body only, IF §8 forces; else untouched)
ui/e2e/mocks/handlers/aliases.ts         (body only, IF §8 forces; else untouched)
.planning/parity/matrix/9router-ui.md
.planning/parity/plans/open-questions.md
docs/WORKFLOW.md
```

Any file outside this list in the scoped diff is an automatic review REJECT.
`internal/**` (incl. `internal/server/routes_admin.go` — w6-h holds NO serial slot),
`ui/package.json`, `ui/src/routeTree.gen.ts`, `ui/e2e/mocks/fixture.ts`,
`ui/e2e/mocks/store.ts`, `ui/e2e/mocks/handlers/index.ts`,
`ui/e2e/mocks/seed/**`, and any frozen w6-a/b file are deliberately ABSENT —
touching them is an automatic REJECT. `ui/dist/**` is gitignored and must never
appear. After merge, the four pages, `ui/src/components/{combos,routing}/**`, and
`ui/src/lib/combo-order.ts` become consume-only for later plans.

## 8. Escalations / cross-track dependencies

- **No blocking dependency at authoring.** w6-a + w6-b are merged (live tree @
  e2ef375: 16 primitives present, `apiFetch`/stores/mock harness in-tree, all four
  `@dnd-kit/*` packages installed per §1.1). w6-h holds NO Go serial slot (ZERO new
  Go) and no frozen exception. Fully unblocked for page wave 1.
- **ESCALATION-1 (RESOLVED at authoring — combos DTO/key divergence):** the real Go
  combos endpoints (`routes_admin.go:85-88`) serve `comboResponse`
  `{name,models:[]string}` keyed by `{name}` (`internal/admin/combos.go:15-21,
  65-67,89-91`), which DIVERGES from the mock/UI `Combo` type
  `{id,name,strategy,steps:[{provider,model}],is_active}` (§1.4). **Decision:** ship
  PAR-UI-091..094 as variant-HAVE against the `/api/combos` MOCK (consumed as-is);
  do NOT edit the existing Go (forbidden) and do NOT add Go (MAP assigns none to
  w6-h). Raise a serial Go follow-up to reconcile the combos DTO/key with the UI
  `Combo` type (add `id`/`strategy`/`is_active`/structured `steps`, or adapt the
  page to the `{name,models[]}` shape) — orchestrator decision, NOT in w6-h.
- **ESCALATION-2 (RESOLVED at authoring — no aliases admin endpoint):**
  `store.ListAliases()` exists (`internal/store/aliases.go:64-65`) and is consumed
  by the OpenAI adapter (`internal/server/routes_openai.go:205-206`), but there is
  NO `/api/aliases` ADMIN route and no `internal/admin/aliases.go` (§1.2).
  **Decision:** ship PAR-UI-116 as variant-HAVE against the `/api/aliases` MOCK
  (consumed unchanged); raise a serial Go follow-up to add the admin
  `GET/POST/PUT/DELETE /api/aliases` handler over the existing alias store —
  orchestrator decision, NOT in w6-h.
- **ESCALATION-3 (RESOLVED at authoring — no routing-rules / model-limits backend):**
  - 3a **routing-rules**: `grep -rE 'routing-rules|RoutingRule' internal/` → ZERO
    matches; no store/admin/route. Ship PAR-UI-130 `/routing-rules` subset as
    variant-HAVE against the `/api/routing-rules` MOCK (consumed unchanged); raise a
    serial Go follow-up to add the routing-rules store + admin CRUD — NOT in w6-h.
  - 3b **model-limits**: `grep -rE 'model-limits|ModelLimit' internal/` → ZERO
    matches. Ship PAR-UI-130 `/model-limits` subset as variant-HAVE against the
    `/api/model-limits` MOCK (consumed unchanged); raise a serial Go follow-up to
    add the model-limits store + admin CRUD — NOT in w6-h.
- **ESCALATION-4 (CONDITIONAL — live-DnD e2e infeasible, §1.3):** if BOTH the
  keyboard-DnD reorder e2e AND the persisted-order-PUT-body-intercept e2e prove
  infeasible/flaky under `vite preview` + `@dnd-kit` (sensors never activate
  headless), STOP and ESCALATE: DROP the live-DnD e2e assertion, keep the
  render-in-order e2e + the `combo-order.test.ts` pure-helper unit as the binding
  reorder proof, and record the disposition in WORKFLOW.md. The plan does NOT block
  on live DnD; the pure reorder-helper unit is the authoritative floor.
- **ESCALATION-5 (CONDITIONAL — shared mock handler body):** if correcting a
  `combos.ts`/`routing-rules.ts`/`model-limits.ts`/`aliases.ts` handler body to fix
  a within-mock inconsistency breaks a NON-w6-h spec that consumes the same handler,
  STOP and ESCALATE (orchestrator serializes the shared-handler change) — do not
  fudge the mock, do not edit `index.ts`/seed/`fixture.ts`.
- **`routeTree.gen.ts` (CONDITIONAL):** if a build reformats it, that is an
  ESCALATION (§1.6), not an in-plan edit; resolve by regeneration, never manual.
```
