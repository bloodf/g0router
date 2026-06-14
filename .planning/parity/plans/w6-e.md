# Micro-plan w6-e — Providers + Connections + Models cluster (UI + provider-shaped Go read API)

```
wave: 6
plan: w6-e
status: READY (rev 1 — authored against merged w6-a + w6-b + w6-c, live tree @ cdfa5d2)
runs: page wave 1, AFTER w6-b MERGE (consumes frozen ui/src/components/ui/*) and
  AFTER w6-c MERGE (consumes the /callback OAuth popup relay + lib/auth.ts
  relayOAuthCallback contract). Disjoint from w6-c/w6-g/w6-h/w6-i (different
  routes/components/specs). Holds the routes_admin.go SERIAL SLOT (order
  w6-pre → w6-d → w6-e → w6-j, MAP §"Cross-cutting rules").
branch: main (direct commits, per repo convention)
commit-prefix: phase-1/w6-e:
ref-source: 9router frozen @ 827e5c3 —
  src/app/(dashboard)/dashboard/providers/{page.js,new/page.js,[id]/page.js,
  [id]/ConnectionRow.js,[id]/ModelRow.js,components/{ConnectionsCard,ModelsCard,
  ModelAvailabilityBadge}.js}; src/shared/components/{OAuthModal,
  EditConnectionModal,ManualConfigModal,CursorAuthModal,KiroAuthModal,
  IFlowCookieModal,GitLabAuthModal,AddCustomEmbeddingModal,NoAuthProxyCard,
  ProviderInfoCard}.js
base: <base> = git rev-parse HEAD recorded at P0 (expected cdfa5d2 at authoring;
  if main advanced, record the actual SHA and substitute everywhere §5 says <base>)
freeze-exception: NONE. The header.tsx / __root.tsx exceptions are SPENT (w6-b
  wiring, w6-c logout-slot). w6-e touches no frozen w6-a/w6-b/w6-c file.
go-serial-slot: this plan holds the SINGLE in-flight edit to
  internal/server/routes_admin.go (additive route registrations only). It must be
  the only unmerged routes_admin.go holder while live (W3/W4/W5 lesson; MAP
  decision 5).
```

---

## 1. Scope — PAR rows

### Rows this plan closes

| Row | Claim | Target state after w6-e |
|---|---|---|
| PAR-UI-007 | Providers page: OAuth / Free / API-Key / Compatible provider groups (cards) | HAVE (variant — flat route `/providers`, §1.5) |
| PAR-UI-008 | Provider "new" flow (add a connection for a provider) | HAVE (variant — in-page flow on `/providers`, NOT a nested route, §1.5) |
| PAR-UI-009 | Provider detail flow (connections + models for one provider) | HAVE (variant — in-page detail panel/drawer on `/providers`, §1.5) |
| PAR-UI-051 | OAuthModal (provider OAuth via popup → `/callback` relay) | HAVE (consumes w6-c relay, §1.4) |
| PAR-UI-052 | EditConnectionModal | HAVE |
| PAR-UI-053 | ManualConfigModal (paste manual OAuth/token config) | HAVE |
| PAR-UI-058 | CursorAuthModal | HAVE |
| PAR-UI-059 | KiroAuthModal | HAVE |
| PAR-UI-060 | IFlowCookieModal | HAVE |
| PAR-UI-062 | GitLabAuthModal | HAVE |
| PAR-UI-063 | AddCustomEmbeddingModal | HAVE |
| PAR-UI-064 | NoAuthProxyCard + ProviderInfoCard | HAVE |
| PAR-UI-087 | Provider-shaped read: list providers w/ status + connection_count | HAVE (Go, variant — §1.6 / §8 ESCALATION-1) |
| PAR-UI-088 | Provider-shaped read: provider detail + its connections | HAVE (Go, variant — §1.6) |
| PAR-UI-089 | Provider-shaped read: provider models + suggested-models | HAVE (Go, variant — §1.6) |
| PAR-UI-090 | Batch connection test (`POST /api/providers/test-batch`) | HAVE (Go — §1.6) |
| PAR-UI-130 (subset) | g0router route `/connections` page | HAVE |

16 PAR-UI rows + the PAR-UI-130 `/connections` subset. Matches WAVE-6-MAP w6-e
row (~line 131) and §Ownership w6-e (~line 169). The cluster also rewrites
`/models` (PAR-UI-130 has no standalone `/models` entry — `/models` is a 9router
parity page covered by the PAR-UI-007..009 provider/model surface and the
`models.spec.ts` contract; recorded here, no separate row).

### 1.1 Preconditions already satisfied by merged waves (evidence)

- Route STUBS exist, must be REWRITTEN (not created — so no new route file, so
  `routeTree.gen.ts` does NOT change for these three; MAP decision 6 / §1.7):
  `ui/src/routes/providers.tsx:1-9` (`<h1>Providers</h1>`),
  `ui/src/routes/connections.tsx:1-9` (`<h1>Connections</h1>`),
  `ui/src/routes/models.tsx:1-9` (`<h1>Models</h1>`).
- Frozen primitives this plan CONSUMES (w6-b, never edited): `Button`
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
  `ui/src/lib/api.ts:19` + `ApiError` `ui/src/lib/api.ts:3`; `useProviderStore`
  `ui/src/stores/provider.ts:17`; toast via `useNotificationStore.push`
  `ui/src/stores/notification.ts`; Material Symbols `ui/src/index.css:3`;
  provider PNGs present (`ls ui/public/providers/*.png | wc -l` → 36).
- Frozen w6-c relay this plan CONSUMES (never edited): `relayOAuthCallback`
  `ui/src/lib/auth.ts:128`; channel name `OAUTH_CHANNEL="oauth_callback"`
  `ui/src/lib/auth.ts:118`; origin allowlist `[window.location.origin,
  "http://localhost:1455"]` `ui/src/lib/auth.ts:121`; the `/callback` route
  `ui/src/routes/callback.tsx:5` that fires the relay. The opener-side contract
  is in §1.4.
- UI types this plan CONSUMES/EXTENDS: `Provider` `ui/src/lib/types.ts:196`
  (`id,name,display_name,description,auth_types[],capabilities[],
  connection_count,status`); `Connection` `ui/src/lib/types.ts:78`
  (`id,provider,name,auth_type,is_active,models[],priority,needs_reauth`);
  `Model` `ui/src/lib/types.ts:159`
  (`id,provider,name,input_cost,output_cost,context_window,is_disabled,
  is_custom`).
- Mock harness already present and registered (CONSUME-ONLY unless a Go conflict
  forces a correction, §1.6 / §8): handlers
  `ui/e2e/mocks/handlers/{providers,connections,models}.ts` registered at
  `ui/e2e/mocks/handlers/index.ts:6,40-42`; seeds
  `ui/e2e/mocks/seed/{providers,connections,models}.ts` exported at
  `ui/e2e/mocks/seed/index.ts:2,3,6`; envelope helpers
  `ui/e2e/mocks/handlers/utils.ts` (`json`→`{data}`, `error`→`{error:<string>}`).
- Existing acceptance specs (the contract — §1.3): `ui/e2e/providers.spec.ts:1-19`
  (2 tests), `ui/e2e/connections.spec.ts:1-13` (1 test), `ui/e2e/models.spec.ts:1-13`
  (1 test). Login helper `ui/e2e/helpers.ts:3` drives `#username`/`#password`.

### 1.2 Existing Go contract for the underlying tables (file:line evidence)

The connection/provider DB CRUD already exists and is FORBIDDEN to rewrite — the
new code only ADDS a provider-shaped read overlay (§1.6):

- `GET /api/providers` → `h.ListProviders` (`routes_admin.go:46`) →
  `providerDTO{id,name,type,base_url,enabled,created_at,updated_at}`
  (`internal/admin/providers.go:11-19,40-52`). **This is a CRUD over the
  `providers` table — its DTO has NO `display_name`/`auth_types`/`capabilities`/
  `connection_count`/`status` and NO per-provider connections/models
  sub-routes.** This is the divergence resolved in §1.6 / §8 ESCALATION-1.
- `POST/PUT/DELETE /api/providers[/{id}]` (`routes_admin.go:47-49`).
- `GET /api/connections` → `h.ListConnections` (`routes_admin.go:51`) →
  `connectionDTO{id,provider_id,name,kind,secret_set,access_token_set,
  refresh_token_set,expires_at,metadata,created_at,updated_at}`
  (`internal/admin/connections.go:11-24,53-65`). **Field names diverge from the
  UI `Connection` type (`provider` vs `provider_id`, `auth_type` vs `kind`,
  `is_active`/`models[]`/`priority`/`needs_reauth` absent).** §8 ESCALATION-2.
- `POST/PUT/DELETE /api/connections[/{id}]`, `POST /api/connections/{id}/refresh`
  (`routes_admin.go:52-55`).
- `GET/POST/DELETE /api/models/disabled` (`routes_admin.go:72-74`). **There is NO
  `GET /api/models` (full catalog), NO `/api/models/custom`, NO
  `/api/providers/{id}/connections|models|suggested-models`, NO
  `/api/providers/test-batch`, NO `/api/connections/bulk-enable|bulk-disable`
  on the Go side** — these are mock-only today.
- Store layer (`*store.Store`, concrete — NOT an interface): `ListProviders`/
  `GetProvider` (`internal/store/providers.go:42,65`), `ListConnections`/
  `GetConnection` (`internal/store/connections.go:54,78`), `ProviderRecord`
  (`providers.go:11`), `Connection` (`connections.go:13`). Envelope writers
  `writeData`/`writeError` (`internal/admin/respond.go:19,23`) — `{data,error:{message}}`.
- Admin test harness (`_test.go` precedent — real store, temp DB, no mocks):
  `internal/admin/admin_test.go:23-41` (`newTestEnv` → `t.TempDir()` +
  `store.Open` + `New(...)`).

### 1.3 The e2e specs are LOOSE smoke contracts (binding interpretation)

The three pre-existing specs are deliberately thin and are the FULL contract:

- `providers.spec.ts`: (a) `/providers` body contains text "Providers"; (b) at
  least one element matching `[class*='card-elev']` is visible. **`card-elev` is
  the binding marker** — it appears nowhere in `ui/src` today (`grep -rn
  "card-elev" ui/src/` → empty), so the providers page MUST render provider cards
  carrying a class that contains the substring `card-elev`.
- `connections.spec.ts`: `/connections` body contains text "Connections".
- `models.spec.ts`: `/models` body contains text "Models".

These minimal assertions are the GREEN bar. w6-e ADDS richer RED assertions
(§4 T1) covering the modals, groups, new/detail flows, and the provider-shaped
data — but the original four assertions above must also pass at closeout. No
existing assertion may be deleted or weakened.

### 1.4 OAuth popup contract — how w6-e modals consume w6-c's `/callback` (binding)

w6-c shipped the popup *sender*: a popup navigated to `/callback?code=…&state=…`
calls `relayOAuthCallback(payload)` (`ui/src/lib/auth.ts:128`), which delivers the
payload three ways. w6-e implements the *opener/listener* side, ENTIRELY inside
w6-e-owned files (a new `ui/src/lib/oauth-popup.ts` helper + the modals) — it
imports the channel/origin constants' BEHAVIOR but does NOT edit `lib/auth.ts`:

1. **Open the popup**: a provider OAuth modal computes the provider authorize URL
   (from `GET /api/oauth/{provider}/start`, `routes_admin.go:69` — server returns
   the IdP URL/redirect; the modal opens it with `window.open(url, "oauth",
   "width=…,height=…")`).
2. **Subscribe BEFORE opening** to all three relay channels, matching w6-c's
   sender exactly:
   - `BroadcastChannel("oauth_callback")` → `onmessage` receives the raw
     `OAuthCallbackPayload` (`{code?,state?,error?,error_description?}`)
     (sender: `auth.ts:142-143`). Channel name MUST be the literal
     `"oauth_callback"` (w6-c `OAUTH_CHANNEL`).
   - `window.addEventListener("message", …)` → accept ONLY when
     `event.origin === window.location.origin` (the opener is same-origin in the
     `vite preview` harness; the sender also posts to `http://localhost:1455`,
     which the listener ignores). The message shape is
     `{type:"oauth_callback", ...payload}` (sender: `auth.ts:129,134`); the
     listener matches on `event.data?.type === "oauth_callback"`.
   - `window.addEventListener("storage", …)` for key `"oauth_callback"`
     (sender: `auth.ts:150`) as the cross-tab fallback; parse `event.newValue`.
3. **On payload**: unsubscribe all three listeners, close the popup if still
   open, then on `code` → POST the finalize call
   (`POST /api/oauth/{provider}/callback` with `{code,state}`,
   `routes_admin.go:70`) which creates the connection; on `error` → toast
   `error_description||error` and leave the modal open.
4. **De-dup**: a single received `code` must finalize at most once (a `handled`
   flag) since up to three channels can deliver the same payload.

This is the surface the MAP calls the "OAuth popup contract" (w6-e depends on
w6-c). It is asserted live in `providers.spec.ts` T1 test 4 (§4) by driving a
`/callback` navigation in a popup context and observing the modal finalize.
**`lib/auth.ts`, `callback.tsx` are CONSUMED, never edited** (§3 FORBIDDEN).

### 1.5 Variant notes (recorded HAVE rationale)

- **PAR-UI-007 grouping**: 9router groups provider cards into OAuth / Free /
  API-Key / Compatible sections (`providers/page.js`). g0router groups by the
  `auth_types`/`status` fields on the `Provider` type (§1.1): OAuth =
  `auth_types` includes `oauth`; API-Key = includes `api_key`; Free/No-auth =
  includes `noauth`; Compatible = OpenAI-compatible custom (`type`/`base_url`
  driven). The page renders a `SegmentedControl` or section headers per group;
  each card carries the `card-elev*` class (§1.3). Flat route `/providers`
  (MAP decision 1). Recorded variant-HAVE.
- **PAR-UI-008 "new" + PAR-UI-009 "detail" are IN-PAGE flows, NOT nested routes**:
  9router uses `/providers/new` and `/providers/[id]`. Adding route files is
  restricted to ONE plan per concurrency wave and w6-e is NOT that plan (w6-i is,
  MAP §"Impl order"). So w6-e implements new/detail as in-page state on
  `/providers` (a detail drawer/panel + a "new connection" modal flow keyed by
  selected provider), NOT `ui/src/routes/providers.new.tsx` or
  `providers.$id.tsx`. This keeps `routeTree.gen.ts` unchanged (§1.7). Recorded
  variant-HAVE.
- **Data layer = plain `apiFetch` + React state, NOT TanStack Query**: although
  `@tanstack/react-query@^5.83.0` is installed (`ui/package.json:51`), NO
  `QueryClientProvider` is mounted (`grep QueryClient ui/src/routes/__root.tsx
  ui/src/main.tsx` → 0 matches), and `__root.tsx`/`main.tsx` are FROZEN (w6-a).
  Mounting a provider would require editing a frozen file. Therefore w6-e fetches
  via `apiFetch` in `useEffect` with local `useState` (same pattern as w6-c
  `login.tsx`). MAP decision 2 (TanStack Query) is satisfied wave-wide by a later
  follow-up that wires the provider through the orchestrator; w6-e records this as
  an accepted constraint, not a gap. (PAR-UI-081 is a w6-g row, not w6-e's.)
- **Pages render inside app chrome**: `__root.tsx` wraps every route in
  Sidebar+Header; the pages render in `<Outlet>`. Specs assert page content +
  cards, all of which hold with chrome present (w6-c precedent §1.4). Accepted
  constraint.

### 1.6 Go gap — provider-shaped read API + batch test (this plan's NEW Go, TDD)

The e2e mock (`mocks/handlers/providers.ts`) models a provider-CATALOG read API
that does NOT exist on the Go side (§1.2). w6-e adds it as a NEW, ADDITIVE,
read-mostly overlay that composes the existing `providers` + `connections` store
data — it does NOT alter the existing CRUD DTOs or tables. **Decision (§8
ESCALATION-1): the new endpoints live under a distinct path prefix
`/api/providers/...` sub-routes + `/api/providers/catalog` to avoid colliding
with the existing `GET /api/providers` CRUD list, OR re-shape is recorded as an
escalation — see §8 for the resolved path map.** All new Go is TDD (`_test.go`
committed RED first), snake_case `{data,error}` envelope (`respond.go`), layered
(handler → `*store.Store`), additive `ensureColumn` migrations ONLY if a column
is needed (none expected — read-only composition), no `init()`, errors-as-values.

New file `internal/admin/providers_catalog.go` (NEW — the existing
`providers.go` is FORBIDDEN to edit) provides:

| Handler | Route (resolved, §8) | Shape (snake_case, `{data}`) | PAR |
|---|---|---|---|
| `ListProviderCatalog` | `GET /api/providers/catalog` | `[{id,name,display_name,description,auth_types[],capabilities[],connection_count,status}]` — composed from known-provider metadata + live `connection_count` (count of connections whose `provider_id==id`) + `status` (`active` if ≥1 active connection else `inactive`) | PAR-UI-087 |
| `GetProviderCatalog` | `GET /api/providers/{id}/catalog` | one provider object (above) or 404 | PAR-UI-088 |
| `GetProviderConnections` | `GET /api/providers/{id}/connections` | `[Connection]` UI-shaped (`{id,provider,name,auth_type,is_active,models[],priority,needs_reauth}`) derived from store connections for that provider | PAR-UI-088 |
| `GetProviderModels` | `GET /api/providers/{id}/models` | `[Model]` for that provider (from catalog metadata; empty if none) | PAR-UI-089 |
| `GetProviderSuggestedModels` | `GET /api/providers/{id}/suggested-models` | `[{id,name}]` (top N of models) | PAR-UI-089 |
| `TestProvidersBatch` | `POST /api/providers/test-batch` | `{results:[{provider,ok,latency_ms}]}` — pings each provider's connections (live HTTP best-effort; in test, deterministic ok=has-active-connection) | PAR-UI-090 |

These compose existing store reads (`ListProviders`, `ListConnections`,
`GetProvider`) — NO new tables, NO secret exposure (connection DTO masks secrets
exactly as `connections.go:37-51`). The "known provider metadata"
(display_name/auth_types/capabilities) is a static in-Go catalog table
(mirrors the mock's `getKnownProvider`); if a provider id is absent from the
static table it falls back to `{display_name:id, auth_types:["api_key"],
capabilities:[]}`. Route registration is the SERIAL-SLOT additive edit to
`routes_admin.go` (§3).

### 1.7 `routeTree.gen.ts` is NOT touched

The three routes already exist as stubs (§1.1); rewriting their component bodies
does not change the route tree. No new route file is added (new/detail are
in-page, §1.5). Therefore `ui/src/routeTree.gen.ts` is UNCHANGED by w6-e
(MAP decision 6; w6-e is not the wave-1 new-route plan — w6-i is). If a build
incidentally reformats it, that is an ESCALATION (§8), not an in-plan edit.

### NOT in scope (explicit)

- **No new route FILES** — only the three existing stubs are rewritten; no
  `providers.new.tsx`/`providers.$id.tsx`/etc.; `routeTree.gen.ts` untouched
  (§1.7).
- **No edits to existing Go CRUD** — `internal/admin/providers.go`,
  `internal/admin/connections.go`, `internal/store/**` are FORBIDDEN; w6-e only
  ADDS `internal/admin/providers_catalog.go` (+ its `_test.go`) and ADDITIVE
  route lines in `routes_admin.go`.
- **No edits to any frozen w6-a/w6-b/w6-c file** — no `__root.tsx`, layout, ui/*,
  stores, `lib/api.ts`, `lib/auth.ts`, `lib/utils.ts`, providers/*, `callback.tsx`,
  `login.tsx`. No header exception remains (SPENT).
- **No TanStack Query wiring** (§1.5) — plain `apiFetch`.
- **No dependency additions** — every import resolves to installed packages or
  w6-a/b/c outputs.
- **No mocks index/seed edits** — `mocks/handlers/index.ts`, `mocks/seed/index.ts`
  already register/export the three domains; w6-e edits handler/seed BODIES ONLY
  if a Go contract conflict forces it (§8), never the index.
- **No other e2e specs** beyond `providers.spec.ts`/`connections.spec.ts`/
  `models.spec.ts` (+ the three matching mock handler bodies if corrected).
- **No SSE/streaming** — provider/connection/model reads are request/response.
- **No virtual-keys / endpoint / keys** (w6-f), no usage/pricing (w6-g).
- **No real outbound provider network calls in tests** — `test-batch` is
  deterministic under test (ok = provider has an active connection).

---

## 2. Precondition checks

Run all before any edit; abort and report to orchestrator on any failure.

```bash
# P0 — clean tree; record <base>
git status --porcelain     # must be empty (untracked tooling artifacts must be
                           # gitignored; worker uses explicit `git add <file>`,
                           # never `git add -A`, per w6-b runtime disposition)
git rev-parse HEAD         # record as <base> for §5 (expected cdfa5d2)

# P1 — w6-b primitives present and frozen (consumed)
ls ui/src/components/ui/*.tsx | grep -v test | wc -l    # = 16 (w6-b set intact)
grep -n "export interface ModalProps\|export function Modal" ui/src/components/ui/modal.tsx
grep -n "ProviderIcon" ui/src/components/ui/provider-icon.tsx
grep -n "SegmentedControl" ui/src/components/ui/segmented-control.tsx
grep -n "ConfirmModal" ui/src/components/ui/confirm-modal.tsx

# P2 — w6-a foundation present and frozen (consumed)
grep -n "export async function apiFetch" ui/src/lib/api.ts
grep -n "export class ApiError" ui/src/lib/api.ts
grep -n "useProviderStore" ui/src/stores/provider.ts
grep -n "push:" ui/src/stores/notification.ts
ls ui/public/providers/*.png | wc -l    # ≥ 1 (ProviderIcon PNG fallback path)

# P3 — w6-c OAuth relay present and frozen (consumed — the popup contract)
grep -n "export function relayOAuthCallback" ui/src/lib/auth.ts
grep -n 'OAUTH_CHANNEL = "oauth_callback"' ui/src/lib/auth.ts
grep -n "http://localhost:1455" ui/src/lib/auth.ts
test -f ui/src/routes/callback.tsx && echo "callback present (good)"

# P4 — the three route stubs are still bare (safe to rewrite); no new dirs yet
grep -n "<h1>Providers</h1>" ui/src/routes/providers.tsx
grep -n "<h1>Connections</h1>" ui/src/routes/connections.tsx
grep -n "<h1>Models</h1>" ui/src/routes/models.tsx
test ! -d ui/src/components/providers && echo "providers components dir absent (good)"
test ! -e ui/src/lib/oauth-popup.ts && echo "oauth-popup helper absent (good)"

# P5 — e2e mock harness present and registered (CONSUME-ONLY unless §8 forces it)
grep -n "registerProvidersHandlers\|registerConnectionsHandlers\|registerModelsHandlers" ui/e2e/mocks/handlers/index.ts
grep -n "seedProviders\|seedConnections\|seedModels" ui/e2e/mocks/seed/index.ts
grep -n "card-elev" ui/src/ -r ; echo "^ expect EMPTY: card-elev must be introduced by this plan"

# P6 — the Go gap is real: provider-shaped sub-routes + test-batch are ABSENT
grep -n "test-batch\|suggested-models\|ListProviderCatalog\|/catalog" internal/server/routes_admin.go internal/admin/*.go ; echo "^ expect EMPTY (the gap §1.6)"
test ! -e internal/admin/providers_catalog.go && echo "providers_catalog.go absent (good)"
grep -n "ListProviders\|ListConnections" internal/store/providers.go internal/store/connections.go  # store reads exist

# P7 — routes_admin.go serial slot is FREE (no other unmerged holder)
git log --oneline -5 -- internal/server/routes_admin.go   # confirm last touch is merged (w6-d/w6-pre done)
# Orchestrator MUST confirm no concurrent plan holds an unmerged routes_admin.go
# edit before w6-e begins T6 (MAP §Cross-cutting: order w6-pre→w6-d→w6-e→w6-j).

# P8 — harness green at base
cd ui && npx playwright test e2e/providers.spec.ts e2e/connections.spec.ts e2e/models.spec.ts
# Record base result: stubs render only <h1>, so the providers `card-elev` test
# FAILS at base (good — it is already part of the red contract); the three
# text-only assertions PASS at base. Record exact pass/fail in WORKFLOW.md.
cd ui && npm run build                               # exit 0
go test ./... && go vet ./...                        # exit 0 (Go untouched-green)
```

---

## 3. Exclusive file ownership

After w6-e merges, all CREATE files below are owned by w6-e; later plans consume,
never edit (MAP decision 7).

**CREATE — routes (REWRITE existing stubs; no new route files, §1.7):**

| File | Exports / contract |
|---|---|
| `ui/src/routes/providers.tsx` (REWRITE) | `Route=createFileRoute("/providers")`; `ProvidersPage`: on mount `apiFetch("/api/providers/catalog")` → group by §1.5; renders section per group; each provider as a `<ProviderCard>` (class includes `card-elev`); clicking a card opens the in-page DETAIL panel (PAR-UI-009); a "Add connection" action opens the NEW flow (PAR-UI-008) → the relevant auth modal. Header text contains "Providers". |
| `ui/src/routes/connections.tsx` (REWRITE) | `Route=createFileRoute("/connections")`; `ConnectionsPage`: `apiFetch("/api/connections")` → table/list of connections (name, provider via `ProviderIcon`, `auth_type`, `is_active` `Toggle`, `needs_reauth` `Badge`); per-row Edit (`EditConnectionModal`), Test (`POST /api/connections/{id}/test`), Delete (`ConfirmModal`); bulk enable/disable buttons. Header text contains "Connections". |
| `ui/src/routes/models.tsx` (REWRITE) | `Route=createFileRoute("/models")`; `ModelsPage`: `apiFetch("/api/models")` (catalog) → table (name, provider, costs, context window, `is_disabled` `Toggle` → `/api/models/disabled`); add-custom via `AddCustomEmbeddingModal`/custom model flow (`/api/models/custom`). Header text contains "Models". |

**CREATE — page/domain components (`ui/src/components/providers/`):**

| File | Exports / contract |
|---|---|
| `provider-card.tsx` | `ProviderCard` — consumes `Card`+`ProviderIcon`+`Badge`; root element className includes literal substring `card-elev` (e.g. `card-elev`/`card-elevated`, §1.3); shows display_name, description, connection_count, status badge, auth_types chips; click → detail/new actions via props. |
| `provider-detail-panel.tsx` | `ProviderDetailPanel` — in-page detail (PAR-UI-009): fetches `/api/providers/{id}/connections` + `/api/providers/{id}/models`; lists connection rows + model rows; "add connection" trigger. |
| `provider-info-card.tsx` | `ProviderInfoCard` (PAR-UI-064) — port of ref `ProviderInfoCard.js`. |
| `no-auth-proxy-card.tsx` | `NoAuthProxyCard` (PAR-UI-064) — port of ref `NoAuthProxyCard.js`. |
| `oauth-modal.tsx` | `OAuthModal` (PAR-UI-051) — consumes `Modal`; on open subscribes to the popup contract (§1.4) via `lib/oauth-popup.ts`; opens `window.open` to `GET /api/oauth/{provider}/start` URL; on relayed `code` finalizes `POST /api/oauth/{provider}/callback`. |
| `edit-connection-modal.tsx` | `EditConnectionModal` (PAR-UI-052) — `Modal`+`Input`/`Select`/`Toggle`; PUT `/api/connections/{id}`. |
| `manual-config-modal.tsx` | `ManualConfigModal` (PAR-UI-053) — paste manual token/config; POST `/api/connections`. |
| `cursor-auth-modal.tsx` | `CursorAuthModal` (PAR-UI-058) — port of ref. |
| `kiro-auth-modal.tsx` | `KiroAuthModal` (PAR-UI-059) — port of ref. |
| `iflow-cookie-modal.tsx` | `IFlowCookieModal` (PAR-UI-060) — port of ref. |
| `gitlab-auth-modal.tsx` | `GitLabAuthModal` (PAR-UI-062) — port of ref. |
| `add-custom-embedding-modal.tsx` | `AddCustomEmbeddingModal` (PAR-UI-063) — port of ref; POST `/api/models/custom`. |

**CREATE — lib (`ui/src/lib/oauth-popup.ts`, NEW — NOT w6-c's frozen `auth.ts`):**

| Export | Contract |
|---|---|
| `subscribeOAuthPopup(handler, opts?): () => void` | Subscribes to all three relay channels of the w6-c contract (§1.4): `BroadcastChannel("oauth_callback")`, `window message` filtered to `event.origin===window.location.origin` && `data.type==="oauth_callback"`, and `storage` event for key `"oauth_callback"`. Calls `handler(payload)` at most once (`handled` flag), then auto-unsubscribes; returns an explicit unsubscribe too. CONSUMES the w6-c channel/origin constants by value; does NOT import or edit `auth.ts`. |
| `openOAuthPopup(url): Window \| null` | `window.open(url, "oauth", "width=600,height=720")`. |

**CREATE — unit tests (vitest — pure/branching logic reachable without a DOM):**

| File | Contents |
|---|---|
| `ui/src/lib/oauth-popup.test.ts` | ≥4 tests: a BroadcastChannel message fires `handler` once with the payload; a same-origin window `message` of `{type:"oauth_callback",...}` fires `handler`; a cross-origin message is IGNORED; a `storage` event for key `oauth_callback` fires `handler`; double-delivery (BroadcastChannel + storage) fires `handler` only once. Stub `BroadcastChannel`/`window`/`localStorage` in-test (w6-a `theme.test.ts` precedent — no jsdom). Committed RED before `oauth-popup.ts` exists. |
| `ui/src/components/providers/provider-card.test.tsx` | ≥3 tests via `renderToString`: renders display_name + status badge; root className contains `card-elev`; renders `connection_count`. Committed RED before `provider-card.tsx`. |

**CREATE — Go (`internal/admin/providers_catalog.go` + `_test.go`, NEW):**

| File | Contents |
|---|---|
| `internal/admin/providers_catalog.go` | The six handlers + static known-provider metadata table + DTO structs, per §1.6. Uses `writeData`/`writeError` (`respond.go`), `h.store.ListProviders/ListConnections/GetProvider`. No `init()`; errors-as-values; secrets masked. |
| `internal/admin/providers_catalog_test.go` | Table-driven tests via `newTestEnv` (`admin_test.go:23`): catalog list returns display_name+connection_count+status; provider-not-found → 404; `/{id}/connections` returns UI-shaped connections with NO secret fields; `/{id}/models` + `/{id}/suggested-models`; `test-batch` returns one result per provider with `ok` reflecting active-connection presence. Committed RED before the impl file. |

**MODIFY — serial-slot route registration (additive only):**

| File | Change (and ONLY this change) |
|---|---|
| `internal/server/routes_admin.go` | ADD, in the providers block (after line 49) and a new sub-block: `r.GET("/api/providers/catalog", h.RequireSession(h.ListProviderCatalog))`, `r.GET("/api/providers/{id}/catalog", h.RequireSession(h.GetProviderCatalog))`, `r.GET("/api/providers/{id}/connections", h.RequireSession(h.GetProviderConnections))`, `r.GET("/api/providers/{id}/models", h.RequireSession(h.GetProviderModels))`, `r.GET("/api/providers/{id}/suggested-models", h.RequireSession(h.GetProviderSuggestedModels))`, `r.POST("/api/providers/test-batch", h.RequireSession(h.TestProvidersBatch))`. Route-order note: register the STATIC `/api/providers/catalog` and `/api/providers/test-batch` so they do not get shadowed by `/{id}` — verify against the fasthttp `router` precedence at impl time; if a conflict arises, that is an ESCALATION (§8 ESCALATION-3), not a silent path change. NOTHING else in the file changes. Diff bound §5: ≤ 10 added lines. SERIAL SLOT — only holder while live. |

**MODIFY — e2e (the acceptance contract; CONSUME mocks, correct BODY only if §8):**

| File | Change |
|---|---|
| `ui/e2e/providers.spec.ts` | KEEP the 2 existing tests. ADD the RED tests in §4 T1 (groups, new/detail flow, OAuth popup relay, modals open). |
| `ui/e2e/connections.spec.ts` | KEEP the 1 existing test. ADD RED tests (list rows, edit, test, delete, bulk). |
| `ui/e2e/models.spec.ts` | KEEP the 1 existing test. ADD RED tests (rows, disable toggle, add-custom). |
| `ui/e2e/mocks/handlers/{providers,connections,models}.ts` | CONSUME as-is. CORRECT a handler BODY only if T1/T6 finds the real Go shape contradicts it (§8) — never the index, never the seed export list. The mocks already serve `/api/providers/catalog`?? NO — they serve `/api/providers` (list). If the page calls `/catalog`, ADD a `/api/providers/catalog` route to the mock body mirroring the existing `providerList` (small body addition), OR point the page at the existing mock path — the page and mock must agree AND match the real Go path resolved in §8. Resolve at T1: pick ONE path and use it in page + mock + Go consistently. |

**FORBIDDEN:** everything else. Explicitly: all of `internal/admin/providers.go`,
`internal/admin/connections.go`, `internal/store/**` (existing CRUD frozen);
all `ui/src/components/ui/*` (w6-b); all `ui/src/stores/*`, `ui/src/lib/api.ts`,
`ui/src/lib/utils.ts`, `ui/src/providers/*`, `ui/src/lib/auth.ts`,
`ui/src/routes/callback.tsx`, `ui/src/routes/login.tsx`,
`ui/src/components/auth/**` (w6-a/w6-b/w6-c); `__root.tsx`, layout components;
`ui/src/main.tsx`; `ui/package.json` + lockfile; `ui/vite.config.ts`;
`ui/playwright.config.ts`; `ui/components.json`; `ui/src/index.css`;
`ui/src/routeTree.gen.ts` (generated; UNCHANGED §1.7); `ui/e2e/mocks/handlers/index.ts`
and `mocks/seed/index.ts` (already wired); all other `ui/e2e/*.spec.ts`; all
other `internal/server/*` routes; any other `internal/admin/*.go`.

---

## 4. TDD tasks

Cadence (strict): **no route/component/lib/Go file may exist (or be rewritten
beyond its stub) before the failing test that covers it is committed.** Both
tracks are strict-TDD: Go `_test.go` before Go impl; UI red specs/units before UI
impl. `cd ui && npm run build` green at EVERY commit (test files + red specs are
never imported by production code — w6-b/w6-c rationale). `go test ./... && go vet
./...` and `go build ./...` green at EVERY commit. The plan's e2e specs stay RED
from T1 until the implementation tasks green them; that is the arc.

### T1 — STEP(a): extend the three e2e specs (commit RED)

Add to `ui/e2e/providers.spec.ts` (names are the acceptance contract, §5):
1. `list loads` — EXISTS (body contains "Providers"). Keep; greens at T4.
2. `provider cards are visible` — EXISTS (`[class*='card-elev']` visible). Keep;
   greens at T4 (ProviderCard introduces `card-elev`).
3. `providers are grouped (OAuth / API-Key / Free / Compatible)` — assert group
   headings/segments render and at least one card sits under a group.
4. `OAuth modal finalizes via the /callback popup relay` — open a provider's
   OAuth modal; assert the modal is visible; simulate the relay by dispatching a
   `BroadcastChannel("oauth_callback")` message (or navigating a popup context to
   `/callback?code=abc&state=xyz`) and `page.route('**/api/oauth/*/callback', …)`
   fulfilling `{data:{...}}`; assert the finalize POST fires exactly once and the
   modal closes. (Live proof of the §1.4 contract.)
5. `provider detail panel shows connections + models` — click a card with
   `connection_count>0`; assert the detail panel lists ≥1 connection row.
6. `manual config / edit-connection modals open` — assert each opens with traffic
   lights `[data-testid="modal-traffic-lights"]`.

Add to `ui/e2e/connections.spec.ts`:
1. `page loads` — EXISTS. Keep.
2. `connection rows render with provider + auth type` — assert ≥1 row with a
   provider name and an `is_active` toggle.
3. `test a connection` — `page.route('**/api/connections/*/test', …)`
   `{data:{ok:true,latency_ms:42}}`; click Test; assert a success toast/indicator.
4. `delete a connection asks for confirmation` — click Delete → `ConfirmModal`
   visible → confirm → row gone.

Add to `ui/e2e/models.spec.ts`:
1. `models page loads` — EXISTS. Keep.
2. `model rows render with cost + context window` — assert ≥1 row showing a cost.
3. `disable a model toggles /api/models/disabled` — `page.route` capture the POST.

STEP(b): run all three specs — **record failure output** (no cards/`card-elev`,
no groups, no modals, no detail panel). Commit RED:
`phase-1/w6-e: failing providers/connections/models e2e + mock alignment (TDD red)`.

**Mock-vs-reality + path gate**: resolve the `/api/providers/catalog` vs
`/api/providers` path decision NOW (§3 mock note + §8 ESCALATION-1). Pick ONE
path and use it identically in page, mock body, and the Go route (§3 serial
slot). If the real Go shape (once T6 lands) contradicts the mock, STOP and
ESCALATE (§8) — never fudge the mock to hide a backend mismatch, never edit the
existing Go CRUD.

### T2 — STEP(a): Go `providers_catalog_test.go` (commit RED)

Write the table-driven tests per §3 against `newTestEnv` (real store, temp DB).
`go test ./internal/admin/ -run Catalog` → FAILS (handlers missing). Record
failure. Commit RED:
`phase-1/w6-e: failing provider-catalog Go tests (TDD red)`.

### T3 — STEP(b): Go `providers_catalog.go` + serial-slot route registration

Implement the six handlers per §1.6 (compose existing store reads; static
metadata table; secrets masked). Add the additive route lines to
`routes_admin.go` (§3; verify static-vs-`{id}` precedence — §8 ESCALATION-3 if
conflict). Gates: `go test ./... && go vet ./... && go build ./...` green
(catalog tests now green). Commit:
`phase-1/w6-e: provider-shaped read API (catalog/connections/models/suggested) + batch test`.

### T4 — STEP(b): `lib/oauth-popup.ts` + `providers.tsx` + provider components

STEP(a) first: ensure `oauth-popup.test.ts` + `provider-card.test.tsx` are
committed RED (write them here if not yet, run vitest red, commit:
`phase-1/w6-e: failing unit tests for oauth-popup + provider-card (TDD red)`).
STEP(b): implement `lib/oauth-popup.ts` (greens its units, §1.4 contract);
`provider-card.tsx` (introduces `card-elev`); `provider-detail-panel.tsx`,
`provider-info-card.tsx`, `no-auth-proxy-card.tsx`; rewrite `providers.tsx`.
Gates: vitest green; `providers.spec.ts` tests 1-3,5 green (cards, groups,
detail) — tests 4,6 (OAuth relay, modals) green once the modals land in T5.
`npm run build` green. Commit:
`phase-1/w6-e: providers page (grouped cards, detail panel), oauth-popup helper`.

### T5 — STEP(b): the auth/config modals + connections + models pages

Implement all modal components (`oauth-modal`, `edit-connection-modal`,
`manual-config-modal`, `cursor-auth-modal`, `kiro-auth-modal`,
`iflow-cookie-modal`, `gitlab-auth-modal`, `add-custom-embedding-modal`),
consuming `Modal`/`Input`/`Select`/`Button` and (for OAuth) `lib/oauth-popup.ts`.
Rewrite `connections.tsx` and `models.tsx`. Gates: ALL of `providers.spec.ts`,
`connections.spec.ts`, `models.spec.ts` green; `npm run build` green. Commit:
`phase-1/w6-e: auth/config modals + connections + models pages`.

### T6 — full gates + closeout

```bash
cd ui && npm run build
cd ui && npx playwright test e2e/providers.spec.ts e2e/connections.spec.ts e2e/models.spec.ts   # all green
cd ui && npx playwright test                             # full suite: no spec green-at-base may be red
cd ui && npx vitest run src/                             # all green incl new units
go test ./... && go vet ./... && go build ./...          # green
```
Flip §1 matrix rows in `.planning/parity/matrix/9router-ui.md`: PAR-UI-007/008/009
→ HAVE (variant, cite §1.5); 051/052/053/058/059/060/062/063/064 → HAVE;
087/088/089/090 → HAVE (Go, variant, cite §1.6/§8); PAR-UI-130 `/connections`
subset → HAVE. Update `docs/WORKFLOW.md` (record P8 base observation, the
resolved catalog path §8, and the serial-slot release). Final commit:
`phase-1/w6-e: close — providers/connections/models cluster; matrix flips`.
**On the close commit, RELEASE the routes_admin.go serial slot to w6-j.**

---

## 5. Binary acceptance criteria

All must hold; each is yes/no. `<base>` = the commit recorded at P0 (cdfa5d2 at
authoring). Diff gate is **w6-e commit-range-scoped** (§7).

**Test gates**
- `cd ui && npx playwright test e2e/providers.spec.ts` → exit 0, all tests pass
  (2 original + added), 0 skipped.
- `cd ui && npx playwright test e2e/connections.spec.ts` → exit 0, all pass.
- `cd ui && npx playwright test e2e/models.spec.ts` → exit 0, all pass.
- `cd ui && npx vitest run src/` → exit 0 (all prior + new units green).
- `cd ui && npm run build` → exit 0.
- `go test ./... && go vet ./... && go build ./...` → exit 0.
- `go test ./internal/admin/ -run Catalog -v` → exit 0, ≥6 catalog cases pass.

**TDD-order proof** — each impl file's covering test appears in an
earlier-or-equal commit:
```bash
# Go: providers_catalog.go after providers_catalog_test.go
ct=$(git log --format=%ct --diff-filter=A -1 -- internal/admin/providers_catalog_test.go)
cf=$(git log --format=%ct --diff-filter=A -1 -- internal/admin/providers_catalog.go)
[ "$ct" -le "$cf" ] || echo "TDD VIOLATION: providers_catalog.go"      # prints nothing
# UI lib: oauth-popup.ts after oauth-popup.test.ts
ct=$(git log --format=%ct --diff-filter=A -1 -- ui/src/lib/oauth-popup.test.ts)
cf=$(git log --format=%ct --diff-filter=A -1 -- ui/src/lib/oauth-popup.ts)
[ "$ct" -le "$cf" ] || echo "TDD VIOLATION: oauth-popup.ts"            # nothing
# provider-card.tsx after its test
ct=$(git log --format=%ct --diff-filter=A -1 -- ui/src/components/providers/provider-card.test.tsx)
cf=$(git log --format=%ct --diff-filter=A -1 -- ui/src/components/providers/provider-card.tsx)
[ "$ct" -le "$cf" ] || echo "TDD VIOLATION: provider-card.tsx"         # nothing
# e2e RED-extension commit precedes the page rewrites
sa=$(git log --format=%ct -1 --grep="failing providers/connections/models e2e")
pi=$(git log --format=%ct --diff-filter=M -1 -- ui/src/routes/providers.tsx)
[ "$sa" -le "$pi" ] || echo "TDD VIOLATION: providers.tsx before red spec"  # nothing
```

**Grep proofs**
```bash
grep -rn "card-elev" ui/src/components/providers/provider-card.tsx        # PAR-UI-007 marker
grep -rn "/api/providers/catalog\|/api/providers" ui/src/routes/providers.tsx  # PAR-UI-087 read
grep -rn "oauth_callback\|subscribeOAuthPopup\|BroadcastChannel" ui/src/lib/oauth-popup.ts  # PAR-UI-051 §1.4
grep -n "window.location.origin" ui/src/lib/oauth-popup.ts               # PAR-UI-051 origin filter
grep -rn "/api/oauth/.*start\|window.open" ui/src/components/providers/oauth-modal.tsx  # PAR-UI-051 popup
test -f ui/src/components/providers/edit-connection-modal.tsx && echo OK  # PAR-UI-052
test -f ui/src/components/providers/manual-config-modal.tsx && echo OK    # PAR-UI-053
test -f ui/src/components/providers/cursor-auth-modal.tsx && echo OK      # PAR-UI-058
test -f ui/src/components/providers/kiro-auth-modal.tsx && echo OK        # PAR-UI-059
test -f ui/src/components/providers/iflow-cookie-modal.tsx && echo OK     # PAR-UI-060
test -f ui/src/components/providers/gitlab-auth-modal.tsx && echo OK      # PAR-UI-062
test -f ui/src/components/providers/add-custom-embedding-modal.tsx && echo OK  # PAR-UI-063
test -f ui/src/components/providers/no-auth-proxy-card.tsx && test -f ui/src/components/providers/provider-info-card.tsx && echo OK  # PAR-UI-064
grep -rn "/api/connections" ui/src/routes/connections.tsx                # PAR-UI-130 /connections
grep -rn "/api/models" ui/src/routes/models.tsx                          # models page
# Go provider-shaped read + batch:
grep -n "ListProviderCatalog\|GetProviderConnections\|GetProviderModels\|GetProviderSuggestedModels\|TestProvidersBatch" internal/admin/providers_catalog.go  # PAR-UI-087/088/089/090
grep -n "display_name\|auth_types\|capabilities\|connection_count" internal/admin/providers_catalog.go  # PAR-UI-087 shape
grep -n "test-batch\|suggested-models\|/catalog" internal/server/routes_admin.go  # routes registered
grep -n "writeData\|writeError" internal/admin/providers_catalog.go      # snake_case {data,error} envelope
! grep -n "Secret\b\|AccessToken\b\|RefreshToken\b" internal/admin/providers_catalog.go && echo "no secret exposure OK"  # secrets masked
! grep -n "func init(" internal/admin/providers_catalog.go && echo "no init() OK"
```

**Negative / freeze proofs (w6-e commit-range — see §7)**
```bash
R="<first-w6-e>^..<last-w6-e>"
git diff $R --name-only -- internal/admin/providers.go internal/admin/connections.go internal/store/ | wc -l  # = 0 (existing CRUD frozen)
git diff $R --name-only -- internal/ | grep -vE 'internal/admin/providers_catalog(_test)?\.go|internal/server/routes_admin\.go' | wc -l  # = 0 (only the new file + serial slot)
git diff $R --name-only -- ui/src/components/ui/ | wc -l                # = 0 (w6-b frozen)
git diff $R --name-only -- ui/src/stores/ ui/src/providers/ ui/src/lib/api.ts ui/src/lib/utils.ts ui/src/lib/auth.ts | wc -l  # = 0 (w6-a/w6-c frozen)
git diff $R --name-only -- ui/src/routes/__root.tsx ui/src/routes/callback.tsx ui/src/routes/login.tsx ui/src/components/layout/ ui/src/components/auth/ ui/src/main.tsx | wc -l  # = 0
git diff $R --name-only -- ui/src/routeTree.gen.ts | wc -l             # = 0 (§1.7 unchanged)
git diff $R --name-only -- ui/package.json ui/package-lock.json ui/vite.config.ts ui/playwright.config.ts ui/components.json ui/src/index.css | wc -l  # = 0
git diff $R --name-only -- 'ui/src/routes/' | grep -vE 'providers\.tsx|connections\.tsx|models\.tsx' | wc -l  # = 0 (only the three stubs rewritten)
git diff $R --name-only -- ui/e2e/ | grep -vE 'providers\.spec\.ts|connections\.spec\.ts|models\.spec\.ts|mocks/handlers/(providers|connections|models)\.ts' | wc -l  # = 0 (no other spec/mock; index/seed untouched)
git diff $R --name-only -- ui/e2e/mocks/ | grep -vE 'handlers/(providers|connections|models)\.ts' | wc -l  # = 0 (mock index/seed untouched)
git diff $R -- internal/server/routes_admin.go | grep "^+" | wc -l     # ≤ 10 (additive route lines + +++ header)
git log --oneline $R -- internal/server/routes_admin.go | wc -l        # = 1 (exactly ONE commit touches the serial-slot file)
```

---

## 6. Out of scope (restated, binding)

No new route files / no `routeTree.gen.ts` change (§1.7); no edits to existing Go
CRUD (`providers.go`/`connections.go`/`store/**`) — only the additive
`providers_catalog.go` + serial-slot routes; no edits to any frozen
w6-a/w6-b/w6-c file (no header exception remains — SPENT); no TanStack Query
wiring (§1.5); no dependency additions; no mocks index/seed edits (handler bodies
only, if §8 forces); no other e2e specs; no SSE; no virtual-keys/endpoint/keys
(w6-f); no usage/pricing (w6-g); no real outbound provider network in tests.
Mock-vs-Go contradiction → escalate (§8), never patch existing Go or fudge a mock.

## 7. Diff-gate scope

Page-wave-1 plans (w6-c/e/g/h/i) commit to main concurrently, so a broad
`<base>..HEAD` range sweeps in sibling commits. The diff gate MUST be scoped to
w6-e's own commits. The orchestrator isolates them with:
`git log --oneline main | grep "^[0-9a-f]* phase-1/w6-e:" | awk '{print $1}'`
and runs `git diff <first-w6-e>^..<last-w6-e> -- [file list]` (same commit-range
scoping as w6-c §7 / w6-b §7 / w5-f split gate).

`git diff <first-w6-e>^..<last-w6-e> --name-only` must be exactly a subset of:

```
ui/src/routes/providers.tsx
ui/src/routes/connections.tsx
ui/src/routes/models.tsx
ui/src/components/providers/provider-card.tsx
ui/src/components/providers/provider-card.test.tsx
ui/src/components/providers/provider-detail-panel.tsx
ui/src/components/providers/provider-info-card.tsx
ui/src/components/providers/no-auth-proxy-card.tsx
ui/src/components/providers/oauth-modal.tsx
ui/src/components/providers/edit-connection-modal.tsx
ui/src/components/providers/manual-config-modal.tsx
ui/src/components/providers/cursor-auth-modal.tsx
ui/src/components/providers/kiro-auth-modal.tsx
ui/src/components/providers/iflow-cookie-modal.tsx
ui/src/components/providers/gitlab-auth-modal.tsx
ui/src/components/providers/add-custom-embedding-modal.tsx
ui/src/lib/oauth-popup.ts
ui/src/lib/oauth-popup.test.ts
ui/e2e/providers.spec.ts
ui/e2e/connections.spec.ts
ui/e2e/models.spec.ts
ui/e2e/mocks/handlers/providers.ts        (body only, IF §8 forces; else untouched)
ui/e2e/mocks/handlers/connections.ts      (body only, IF §8 forces; else untouched)
ui/e2e/mocks/handlers/models.ts           (body only, IF §8 forces; else untouched)
internal/admin/providers_catalog.go
internal/admin/providers_catalog_test.go
internal/server/routes_admin.go           (serial-slot additive route lines; ONE commit)
.planning/parity/matrix/9router-ui.md
docs/WORKFLOW.md
```

Any file outside this list in the scoped diff is an automatic review REJECT.
`internal/admin/providers.go`, `internal/admin/connections.go`,
`ui/src/routeTree.gen.ts`, and any frozen w6-a/b/c file are deliberately ABSENT —
touching them is an automatic REJECT. The `routes_admin.go` edit must appear in
exactly ONE commit (the §5 `git log … | wc -l` = 1 proof) and the serial slot is
released to w6-j on close. After merge, the three pages, `ui/src/components/providers/**`,
`ui/src/lib/oauth-popup.ts`, and `internal/admin/providers_catalog.go` become
consume-only for later plans (w6-f consumes the provider/connection read shapes).

## 8. Escalations / cross-track dependencies

- **ESCALATION-1 (RESOLVED at authoring — path decision, binding)**: the e2e mock
  models a provider-CATALOG read API (`display_name`/`auth_types`/`capabilities`/
  `connection_count`/`status` + `/{id}/connections|models|suggested-models` +
  `test-batch`) that the existing Go `GET /api/providers` (a plain `providers`
  table CRUD, §1.2) does NOT provide and whose shape it CANNOT take without
  breaking the CRUD consumers. **Decision**: add the catalog overlay under
  DISTINCT paths — `GET /api/providers/catalog` (list) and
  `GET /api/providers/{id}/catalog` (detail) — leaving the existing CRUD
  `GET /api/providers` / `GET /api/providers/{id}` untouched, plus the
  sub-routes `/{id}/connections|models|suggested-models` and the static
  `POST /api/providers/test-batch`. The page, the mock body, and the Go route MUST
  use the SAME chosen paths (resolved at T1). Rationale: additive, non-breaking,
  honors "mocks mirror reality" by making reality match the mock's *capabilities*
  on new paths rather than mutating the existing CRUD contract. Recorded as a
  variant for PAR-UI-087/088/089.
- **ESCALATION-2 (RESOLVED at authoring — connection shape)**: the existing Go
  `connectionDTO` (`provider_id`/`kind`/`secret_set`, §1.2) diverges from the UI
  `Connection` type (`provider`/`auth_type`/`is_active`/`models[]`/`priority`/
  `needs_reauth`, §1.1). **Decision**: the NEW `GetProviderConnections` handler
  emits the UI-shaped connection (mapping `provider_id→provider`, `kind→auth_type`,
  deriving `is_active` from connection state, `needs_reauth` from token expiry,
  `models[]`/`priority` from metadata where present), WITHOUT changing the
  existing `GET /api/connections` CRUD DTO. The `/connections` page consumes the
  existing `GET /api/connections` CRUD list (already shaped acceptably for the
  list view) OR the UI maps the CRUD DTO client-side; pick one at T5 and keep
  page+mock consistent. If the existing CRUD DTO cannot satisfy the
  `connections.spec.ts` assertions without a Go change, STOP and ESCALATE — do
  not edit the existing CRUD; raise a serial follow-up.
- **ESCALATION-3 (CONDITIONAL — fasthttp route precedence)**: registering static
  `/api/providers/catalog` and `/api/providers/test-batch` alongside
  `/api/providers/{id}/...` may collide depending on the `fasthttp/router`
  matcher's static-vs-param precedence. If registration order cannot
  disambiguate (e.g. `{id}` captures `catalog`), STOP and ESCALATE for a path
  rename (e.g. `/api/provider-catalog`) — never silently diverge page/mock/Go.
- **ESCALATION-4 (CONDITIONAL — mock-vs-Go)**: if at T6 the real Go shape
  contradicts the corrected mock body, STOP and ESCALATE; no existing-Go edit, no
  mock fudge (MAP decision 4).
- **Serial-slot dependency**: w6-e holds the routes_admin.go slot after w6-d
  (and w6-pre) merge and releases it to w6-j on close (MAP §Cross-cutting). The
  orchestrator MUST confirm the slot is free (P7) before T3.
- **No other blocking dependency**: w6-a + w6-b + w6-c are merged (live tree @
  cdfa5d2: 16 primitives present, `relayOAuthCallback`/`/callback` in-tree, Go
  provider/connection CRUD + store reads in-tree). w6-e is unblocked for page
  wave 1.
```
