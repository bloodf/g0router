# Micro-plan w6-f — Endpoint + keys + virtual-keys cluster (UI + provider-nodes Go)

```
wave: 6
plan: w6-f
status: READY (rev 1 — authored against merged w6-a + w6-b + w6-c + w6-e + w5-g,
  live tree @ bfa9436)
runs: page wave 2, AFTER w6-b MERGE (consumes frozen ui/src/components/ui/*),
  AFTER w6-a MERGE (consumes apiFetch/ApiError, stores, the e2e mock fixture),
  AFTER w6-e MERGE (consumes the SHIPPED providers_catalog.go model endpoints +
  the w6-e-owned models.ts/catalog.ts mock surface), and built on w5-g's REAL
  virtual-key CRUD (provider_configs[].key_ids, §1.2). Disjoint from
  w6-j/w6-k/w6-l/w6-m (different routes/components/specs). TAKES the
  routes_admin.go SERIAL SLOT (provider-nodes routes), §1.8.
branch: main (direct commits, per repo convention)
commit-prefix: phase-1/w6-f:
ref-source: 9router frozen @ 827e5c3 —
  src/app/(dashboard)/dashboard/endpoint/{page.js,EndpointPageClient.js}
  (base-url display + machineId + API-keys CRUD against /api/keys),
  src/shared/components/ModelSelectModal.js (hierarchical model picker: combos +
  provider-nodes + custom + disabled), src/app/(dashboard)/dashboard/providers/page.js
  (the provider-nodes create/validate flow, ports to ModelSelectModal/endpoint here).
  9router has NO virtual-keys page — /virtual-keys is a g0router-EXTRA route
  (PAR-UI-130) backed by w5-g Go + the e2e virtual-keys mock (§1.5).
base: <base> = git rev-parse HEAD recorded at P0 (expected bfa9436 at authoring;
  if main advanced, record the actual SHA and substitute everywhere §5 says <base>)
freeze-exception: NONE. The header.tsx / __root.tsx / main.tsx exceptions are
  SPENT (w6-a/w6-b/w6-c). w6-f touches no frozen w6-a/w6-b/w6-c/w6-e file.
go-serial-slot: this plan holds the SINGLE in-flight edit to
  internal/server/routes_admin.go (additive provider-nodes route registrations
  only). The slot must be FREE when w6-f starts T3 — w6-e RELEASED it to w6-j on
  its close (docs/WORKFLOW.md:34); w6-f (page wave 2, runs ∥ w6-j) TAKES it,
  then RELEASES it to w6-j on close (§1.8). Only holder while live (W3/W4/W5
  lesson; MAP decision 5).
new-route: NO. All three routes exist as stubs (§1.1); rewrite-only;
  routeTree.gen.ts is UNCHANGED (§1.7). w6-l is wave-2's new-route plan, not w6-f.
```

---

## 1. Scope — PAR rows

### Rows this plan closes

| Row | Claim | Target state after w6-f |
|---|---|---|
| PAR-UI-006 | Route `/endpoint` API endpoint config + API-key management | HAVE (variant — flat route `/endpoint`; base-url + keys panel, §1.3/§1.5) |
| PAR-UI-049 | ModelSelectModal hierarchical model picker (combos + custom + disabled) | HAVE (consumes real `/api/combos` + `/api/models` + `/api/models/disabled`; mock `/api/models/custom`, §1.4/§1.6) |
| PAR-UI-109 | API `GET /api/provider-nodes` (custom compatible nodes) | HAVE (Go — NEW `internal/admin/nodes.go`, §1.6) |
| PAR-UI-110 | API `POST /api/provider-nodes` (create node) | HAVE (Go — §1.6) |
| PAR-UI-111 | API `POST /api/provider-nodes/validate` (validate endpoint) | HAVE (Go — §1.6) |
| PAR-UI-115 | API `POST /api/keys` (create API key) | HAVE (REAL Go `apikeys.go` already exists, §1.2; DTO divergence §8 ESC-2) |
| PAR-UI-117 | API `GET /api/models/custom` (custom models) | HAVE (variant — mock-only; no Go; §1.4/§8 ESC-3) |
| PAR-UI-118 | API `GET /api/models/disabled` (disabled models) | HAVE (REAL Go `disabledmodels.go`, §1.2) |
| PAR-UI-119 | API `POST /api/models/test` (test model inference) | HAVE (variant — NEW mock body; no Go; §1.4/§8 ESC-3) |
| PAR-UI-120 | API `GET /api/models/availability` (model availability) | HAVE (variant — NEW mock body; no Go; §1.4/§8 ESC-3) |
| PAR-UI-130 (subset) | g0router routes `/virtual-keys` + `/endpoint` | HAVE (`/virtual-keys` CRUD + KeyIDs editor on real w5-g VK API, §1.5; `/endpoint` §1.3) |

10 PAR-UI rows + the PAR-UI-130 `/virtual-keys`+`/endpoint` subset. Matches
WAVE-6-MAP w6-f row (~line 132) and §Ownership w6-f (~line 174-176). Three of the
six model/node API rows have REAL Go (115 keys, 118 disabled, 109/110/111 the new
nodes.go); two (117 custom, 119 test, 120 availability) are mock-only and ship
variant-HAVE with serial Go follow-ups (§8).

### 1.1 Preconditions already satisfied by merged waves (evidence)

- **Route STUBS exist, must be REWRITTEN** (not created — so no new route file, so
  `routeTree.gen.ts` does NOT change; MAP decision 6 / §1.7). All three render
  only an `<h1>`:
  `ui/src/routes/endpoint.tsx:1-9` (`createFileRoute("/endpoint")`, `<h1>Endpoint</h1>`),
  `ui/src/routes/keys.tsx:1-9` (`createFileRoute("/keys")`, `<h1>Keys</h1>`),
  `ui/src/routes/virtual-keys.tsx:1-9` (`createFileRoute("/virtual-keys")`,
  `<h1>Virtual Keys</h1>`).
  `ui/src/routeTree.gen.ts` ALREADY registers `/endpoint`, `/keys`, `/virtual-keys`
  (verify P4) — w6-f adds NO route file, so the tree does not change.
- Frozen primitives this plan CONSUMES (w6-b, never edited; 16 present): `Button`
  `ui/src/components/ui/button.tsx`; `Input` `ui/src/components/ui/input.tsx`;
  `Select` `ui/src/components/ui/select.tsx`; `Card`/`CardHeader`/`CardTitle`/
  `CardContent` `ui/src/components/ui/card.tsx`; `Modal`
  `ui/src/components/ui/modal.tsx` (controlled `open`/`onClose`, traffic lights,
  Escape, overlay, scroll-lock); `ConfirmModal`
  `ui/src/components/ui/confirm-modal.tsx`; `Badge` `ui/src/components/ui/badge.tsx`;
  `Toggle` `ui/src/components/ui/toggle.tsx`; `SegmentedControl`
  `ui/src/components/ui/segmented-control.tsx`; `ProviderIcon`
  `ui/src/components/ui/provider-icon.tsx`; `Loading`/`Spinner`/`Skeleton`
  `ui/src/components/ui/{loading,skeleton}.tsx`; `Tooltip`
  `ui/src/components/ui/tooltip.tsx`; `Pagination`
  `ui/src/components/ui/pagination.tsx`.
- Frozen foundation this plan CONSUMES (w6-a, never edited): `apiFetch`
  `ui/src/lib/api.ts:19` + `ApiError` `ui/src/lib/api.ts:3`; toast via
  `useNotificationStore.push` `ui/src/stores/notification.ts`; Material Symbols
  `ui/src/index.css:3`.
- UI types this plan CONSUMES/EXTENDS (`ui/src/lib/types.ts`): `ApiKey`
  (`types.ts:29`, `{id,name,prefix,full_key?,scopes[],rpm_limit?,tpm_limit?,
  daily_spend_cap?,is_active,created_at}`); `VirtualKey` (`types.ts:317`,
  `{id,name,prefix,budget_usd,budget_used_usd,budget_period,rate_limit_rpm,
  is_active}`); `Model` (`types.ts:159`, `{id,provider,name,input_cost,
  output_cost,context_window,is_disabled,is_custom}`). **Both `ApiKey` and
  `VirtualKey` UI types DIVERGE from the real Go DTOs** (§1.2 / §8 ESC-2): the Go
  `apiKeyDTO` is `{id,key,name,machine_id,is_active,created_at}` and the Go
  `virtualKeyDTO` is `{id,key,name,provider_configs[],budget?,rate_limit_rpm?,
  is_active,created_at,updated_at}`. The KeyIDs editor consumes the Go VK shape
  (`provider_configs[].key_ids`), NOT the thin UI `VirtualKey` (§1.6).
- Frozen w6-e outputs this plan CONSUMES (never edited): the SHIPPED
  provider-shaped read overlay `internal/admin/providers_catalog.go` —
  `GET /api/providers/catalog` (`routes_admin.go:53`),
  `GET /api/providers/{id}/models` (`routes_admin.go:57`) →
  `catalogModelDTO{id,provider,name,input_cost,output_cost,context_window,
  is_disabled,is_custom}` (`providers_catalog.go:43-50`),
  `GET /api/providers/{id}/suggested-models` (`routes_admin.go:58`). The full-model
  catalog read is `GET /api/models` (`routes_admin.go`, served by the e2e
  `models.ts` mock + `catalog.ts`). These are the "available models/connections"
  surface the KeyIDs editor + ModelSelectModal consume (§1.6).
- e2e mock harness already present + registered (CONSUME-ONLY; bodies corrected
  only on Go conflict, §1.4 / §8): handlers `ui/e2e/mocks/handlers/keys.ts`
  (`/api/keys` GET/POST + `/api/keys/{id}` GET/PUT/DELETE + `/{id}/regenerate`),
  `ui/e2e/mocks/handlers/virtual-keys.ts` (`/api/virtual-keys` GET/POST +
  `/api/virtual-keys/{id}` GET/PUT/DELETE), registered at
  `ui/e2e/mocks/handlers/index.ts:11-12,45-46`; seeds `seedKeys`
  (`seed/keys.ts:3`), `seedVirtualKeys` (`seed/virtual-keys.ts:3`) exported at
  `seed/index.ts:4-5`. The ModelSelectModal data handlers are w6-e/foundation
  owned (CONSUME-ONLY, §1.4): `models.ts` (`/api/models`, `/api/models/disabled`,
  `/api/models/custom`, `ui/e2e/mocks/handlers/models.ts:7,16,35`),
  `combos.ts` (`/api/combos`, `combos.ts:6`), `catalog.ts`
  (`ui/e2e/mocks/catalog.ts` → `getAllCatalogModels`).
- Existing acceptance specs (the contract — §1.3 thin-smoke interpretation):
  `ui/e2e/keys.spec.ts:10-13` (1 test: `/keys` body contains "API Keys"),
  `ui/e2e/virtual-keys.spec.ts:11-13` (1 test: `/virtual-keys` body contains
  "Virtual Keys"). **NO `ui/e2e/endpoint.spec.ts` exists** (`test ! -e` → true at
  authoring) — w6-f CREATES it RED first (§1.8). Login helper `ui/e2e/helpers.ts:3`
  drives `#username`/`#password`, `username="admin" password="123456"`.

### 1.2 Real Go contract (file:line evidence)

Keys + virtual-keys backends ALREADY EXIST (w5-g + earlier). w6-f adds NO Go for
them. The ONLY new Go is `internal/admin/nodes.go` (provider-nodes, §1.6).

Routes (`internal/server/routes_admin.go`):
- `GET /api/keys` → `h.ListAPIKeys` (`routes_admin.go:66`)
- `POST /api/keys` → `h.CreateAPIKey` (`routes_admin.go:67`) — **PAR-UI-115 (real)**
- `GET /api/keys/{id}` → `h.GetAPIKey` (`routes_admin.go:68`)
- `PUT /api/keys/{id}` → `h.UpdateAPIKey` (`routes_admin.go:69`)
- `DELETE /api/keys/{id}` → `h.DeleteAPIKey` (`routes_admin.go:70`)
- `GET /api/virtual-keys` → `h.ListVirtualKeys` (`routes_admin.go:72`)
- `POST /api/virtual-keys` → `h.CreateVirtualKey` (`routes_admin.go:73`)
- `GET /api/virtual-keys/{id}` → `h.GetVirtualKey` (`routes_admin.go:74`)
- `PUT /api/virtual-keys/{id}` → `h.UpdateVirtualKey` (`routes_admin.go:75`)
- `DELETE /api/virtual-keys/{id}` → `h.DeleteVirtualKey` (`routes_admin.go:76`)
- `GET/POST/DELETE /api/models/disabled` (`routes_admin.go:81-83`) — **PAR-UI-118 (real)**
- `GET/POST/PUT/DELETE /api/combos[/{name}]` (`routes_admin.go:85-88`) — for ModelSelectModal
- Provider-shaped model reads (w6-e, SHIPPED): `routes_admin.go:53-58` (§1.1).

Body / response shapes (snake_case `{data,error:{message}}` envelope,
`internal/admin/respond.go:9-25` — `writeData`/`writeError`):
- **apiKeyDTO** (`internal/admin/apikeys.go:11-17`):
  `{id,key,name,machine_id,is_active,created_at}`. **DIVERGES from UI `ApiKey`**
  (`prefix`/`full_key`/`scopes`/`rpm_limit`/`tpm_limit`/`daily_spend_cap` absent;
  Go has `key`/`machine_id`). `CreateAPIKey` body is `{name}` only
  (`apikeys.go:48`); `UpdateAPIKey` body is `{is_active}` (`apikeys.go:100`). §8 ESC-2.
- **virtualKeyDTO** (`internal/admin/virtualkeys.go:13-22`):
  `{id,key,name,provider_configs[],budget?,rate_limit_rpm?,is_active,created_at,
  updated_at}` where `provider_configs[]` is `schemas.ProviderConfig`
  (`internal/schemas/governance.go:13-19`):
  `{provider,allowed_models[],key_ids[],weight?}`, and `budget` is
  `schemas.Budget` (`governance.go:21-25`): `{limit,period,used}`. **DIVERGES from
  UI `VirtualKey`** (`prefix`/`budget_usd`/`budget_used_usd`/`budget_period`/
  `rate_limit_rpm` flat vs Go's nested `budget{limit,period,used}` +
  `provider_configs[]`). `CreateVirtualKey`/`UpdateVirtualKey` require
  `provider_configs` non-empty with `key_ids` non-empty per config
  (`virtualkeys.go:55-66`). **`provider_configs[].key_ids` IS the KeyIDs pinning
  field** (§1.6). §8 ESC-2.
- **GetDisabledModels** (`internal/admin/disabledmodels.go:9-28`):
  `?provider_alias=` optional → `{data:<ids>}`. `PostDisabledModels` body
  `{provider_alias,ids[]}` (`disabledmodels.go:32-35`). `DeleteDisabledModels`
  `?provider_alias=&id=` (`disabledmodels.go:55`).
- **catalogModelDTO** (w6-e, `providers_catalog.go:43-50`):
  `{id,provider,name,input_cost,output_cost,context_window,is_disabled,is_custom}`.

**Gaps that have NO Go and ship as the NEW nodes.go OR variant-mock (§1.6/§8):**
- `GET/POST /api/provider-nodes`, `POST /api/provider-nodes/validate`
  (`grep -nE '/api/provider-nodes' internal/server/routes_admin.go` → EMPTY) —
  **NEW Go in `internal/admin/nodes.go` (PAR-UI-109/110/111), §1.6.**
- `GET/POST /api/models/custom` (`grep '/api/models/custom' internal/` → EMPTY) —
  **NO Go; mock-only** (`models.ts:35`, w6-e owned). PAR-UI-117 variant. §8 ESC-3.
- `POST /api/models/test` (`grep '/api/models/test' internal/` → EMPTY) —
  **NO Go, NO mock today; NEW mock body in w6-f.** PAR-UI-119 variant. §8 ESC-3.
- `GET /api/models/availability` (`grep '/api/models/availability' internal/` →
  EMPTY) — **NO Go, NO mock today; NEW mock body in w6-f.** PAR-UI-120 variant.
  §8 ESC-3.

### 1.3 The endpoint page surface (binding interpretation)

9router's endpoint page (`endpoint/EndpointPageClient.js`, 60KB) is the operator's
"how to call the gateway" panel: it shows the base URL + a machine-derived ID, and
manages API keys via `/api/keys` GET/POST/PUT/DELETE (`EndpointPageClient.js:364,
720,745,763`). g0router ports the essential surface:
1. **Base-URL panel**: display the gateway base URL (`window.location.origin` +
   `/v1`, the OpenAI-compatible prefix) with a copy action; a sample curl/snippet
   block. (Static, no API call.)
2. **API-keys panel**: list `apiFetch("/api/keys")`; create (`POST /api/keys`,
   body `{name}` per the real Go `apikeys.go:48`); toggle active
   (`PUT /api/keys/{id}` `{is_active}`); delete (`DELETE /api/keys/{id}` +
   `ConfirmModal`). The created key's full `key` is shown once (the Go DTO returns
   `key`, `apikeys.go:13`). This is the SAME `/api/keys` surface the `/keys` page
   uses — the endpoint page embeds a compact keys widget (a shared
   `<ApiKeysPanel>` component, §3), the `/keys` route renders the full panel.
3. The endpoint page renders the gateway base URL + the keys widget; its acceptance
   is the NEW `endpoint.spec.ts` (§1.8) — body contains "Endpoint", base-url block
   visible, keys widget renders ≥1 seeded key.

The `/keys` page (PAR-UI-006/115 keys half) renders the full `<ApiKeysPanel>`. Its
header text MUST contain "API Keys" (the `keys.spec.ts:12` assertion — the stub
renders only `<h1>Keys</h1>`, so VERIFY at P7 whether the existing test passes on
chrome alone; the page MUST render an "API Keys" heading regardless).

### 1.4 Mock paths/shapes (binding interpretation — CONSUME; correct BODY only on Go conflict)

| Surface | Mock route (file, owner) | Mock shape | Real Go (§1.2) | Resolution |
|---|---|---|---|---|
| API keys | `/api/keys` GET/POST + `/{id}` GET/PUT/DELETE + `/{id}/regenerate` (`handlers/keys.ts`, w6-f-owned body) | `{id,name,prefix,full_key,scopes[],rpm_limit?,...,is_active,created_at}` (`seed/keys.ts`) | `apikeyDTO{id,key,name,machine_id,is_active,created_at}`; create body `{name}` | **DIVERGES** (§8 ESC-2). The page calls the REAL `/api/keys` path. CORRECT the mock BODY to mirror the real Go DTO field names (`key` not `full_key`, drop `scopes`/`rpm_limit` OR keep them as page-tolerated extras) AND the create body `{name}`. The page must tolerate the real shape. **NO `/{id}/regenerate` on the Go side** → drop that mock route OR keep page from calling it (the real API has no regenerate; reissue = delete+create). Resolve at T1. |
| Virtual keys | `/api/virtual-keys` GET/POST + `/{id}` GET/PUT/DELETE (`handlers/virtual-keys.ts`, w6-f-owned body) | `{id,name,prefix,budget_usd,budget_used_usd,budget_period,rate_limit_rpm,is_active}` (`seed/virtual-keys.ts`) | `virtualKeyDTO{id,key,name,provider_configs[],budget{limit,period,used}?,rate_limit_rpm?,is_active,created_at,updated_at}` | **DIVERGES** (§8 ESC-2). The page calls the REAL `/api/virtual-keys`. CORRECT the mock BODY + seed to mirror the real Go DTO: nested `budget{limit,period,used}`, `provider_configs[{provider,allowed_models[],key_ids[],weight?}]`. Create/update require `provider_configs` with non-empty `key_ids` (`virtualkeys.go:55-66`). The KeyIDs editor (§1.6) writes `provider_configs`. Resolve at T1. |
| Combos (ModelSelectModal) | `/api/combos` (`handlers/combos.ts`, w6-e/foundation owned) | combo list | `GET /api/combos` (`routes_admin.go:85`) | CONSUME UNCHANGED (real Go agrees; w6-h owns combos page, w6-f only READS for the modal). |
| Models catalog (ModelSelectModal) | `/api/models` + `/api/models/disabled` (`handlers/models.ts`, w6-e owned) | catalog + disabled set | `GET /api/models` (mock+catalog) + `GET/POST/DELETE /api/models/disabled` (real, `routes_admin.go:81-83`) | CONSUME UNCHANGED (w6-e owns models.ts; w6-f READS only). `/api/models/disabled` is REAL Go. |
| Custom models (ModelSelectModal) | `/api/models/custom` GET/POST + `/{id}` DELETE (`handlers/models.ts:35`, w6-e owned) | `{id,...,is_custom:true}` | NONE | **No Go.** CONSUME the w6-e mock UNCHANGED; PAR-UI-117 variant-HAVE; serial Go follow-up (§8 ESC-3). |
| Provider-nodes | **NONE today** (`grep 'provider-nodes' ui/e2e/mocks/handlers/` → EMPTY) | n/a | NEW Go `nodes.go` (§1.6) | **NEW mock body** `handlers/nodes.ts` mirroring the new Go (§1.6): `GET /api/provider-nodes` → `{nodes:[...]}`, `POST /api/provider-nodes` `{name,prefix,apiType,baseUrl,type}` → `{node}`, `POST /api/provider-nodes/validate` `{baseUrl,apiKey,type,modelId?}` → `{valid,error?}`. Registered in `handlers/index.ts` (one sanctioned append, §1.9). |
| Model test | **NONE today** | n/a | NONE | **NEW mock body** in `handlers/nodes.ts` (or a small `models-extra.ts`): `POST /api/models/test` → `{ok,latency_ms?}` (deterministic). PAR-UI-119 variant; serial Go follow-up (§8 ESC-3). |
| Model availability | **NONE today** | n/a | NONE | **NEW mock body**: `GET /api/models/availability` → `{available:[...]}` or per-model map. PAR-UI-120 variant; serial Go follow-up (§8 ESC-3). |

**Binding rule (MAP decision 4):** where mock and real Go disagree the real Go
wins and the mock body is corrected in-plan (mocks mirror reality). For w6-f:
correct the `keys.ts` + `virtual-keys.ts` handler BODIES + their seeds to the real
Go DTOs (§8 ESC-2); CONSUME the w6-e-owned `models.ts`/`combos.ts`/`catalog.ts`
UNCHANGED; CREATE a new `nodes.ts` mock (the provider-nodes + model-test +
availability surfaces) and register it (§1.9). If correcting `keys.ts`/
`virtual-keys.ts` would break a non-w6-f spec, STOP and ESCALATE (§8 ESC-5) — but
those two handlers are consumed ONLY by w6-f's specs (no other plan owns keys/VK).

### 1.5 Variant notes (recorded HAVE rationale)

- **PAR-UI-006/115 endpoint+keys**: variant — flat `/endpoint` + `/keys` routes;
  base-url panel + `<ApiKeysPanel>` against the REAL `/api/keys` Go CRUD; the Go
  `apiKeyDTO` shape (`key`/`machine_id`) is honored, the UI `ApiKey` type's extra
  fields (`scopes`/`rpm_limit`) are display-optional (§1.3/§8 ESC-2). Recorded
  variant-HAVE.
- **`/virtual-keys` (PAR-UI-130 subset)**: variant — 9router has NO virtual-keys
  page; this is a g0router-EXTRA route backed by the REAL w5-g VK CRUD. The page
  lists VKs (name, budget used/limit, RPM, active toggle), creates/edits via a
  `<VirtualKeyFormModal>` that includes the **KeyIDs pinning editor** (§1.6).
  Recorded variant-HAVE.
- **KeyIDs pinning editor (the w6-pre integration, RESOLVED §1.6)**: w6-pre's
  standalone `/api/catalog` did NOT land (`internal/admin/catalog.go` ABSENT,
  `internal/inference/catalog.go` ABSENT, no `/api/catalog` route). The
  "available models/connections" the KeyIDs editor needs is satisfied by the
  SHIPPED w6-e endpoints (`GET /api/providers/{id}/models`, `GET /api/models`) +
  the REAL VK `provider_configs[].key_ids` write surface (w5-g). The editor does
  NOT consume an absent `/api/catalog`. §1.6 / §8 ESC-1.
- **PAR-UI-049 ModelSelectModal**: consumes real `/api/combos` + `/api/models` +
  `/api/models/disabled` and the mock `/api/models/custom`; a hierarchical picker
  (combos group / per-provider models / custom / disabled-hidden). Recorded HAVE.
- **PAR-UI-117/119/120**: variant — custom-models read, model-test, model-
  availability have NO Go; ship against mock bodies; serial Go follow-ups (§8
  ESC-3). Recorded variant-HAVE.
- **Pages render inside app chrome**: `__root.tsx` wraps every route in
  Sidebar+Header; pages render in `<Outlet>`; specs assert page content with
  chrome present (w6-c/w6-e/w6-g/w6-i precedent). Accepted constraint, not a gap.
- **Data layer = plain `apiFetch` + React state, NOT TanStack Query**:
  `QueryClientProvider` is NOT mounted (`__root.tsx`/`main.tsx` FROZEN, w6-a;
  PAR-UI-081 already HAVE from w6-a). w6-f uses `apiFetch` in `useEffect` +
  `useState`. Accepted constraint.

### 1.6 KeyIDs pinning editor + the w6-pre catalog decision (binding, RESOLVED)

**The surface.** The VK editor lets an operator pin specific connection KeyIDs to a
virtual key per provider. The REAL write target is the w5-g VK CRUD: a VK carries
`provider_configs[]` (`schemas.ProviderConfig`, `governance.go:13-19`), each
`{provider, allowed_models[], key_ids[], weight?}`. The `key_ids` array is the
pinning field (`virtualkeys.go:65-66` requires it non-empty per config). PAR-ROUTE-030
(KeyIDs half) is the routing-side consumption of this same field.

**The "w6-pre catalog API" decision (RESOLVED — binding).** The MAP (w6-f row,
~line 132) says the KeyIDs editor consumes "w6-pre's catalog API." **w6-pre has NOT
landed** (verified at authoring: `internal/admin/catalog.go` ABSENT,
`internal/inference/catalog.go` ABSENT, `grep '/api/catalog' internal/server/
routes_admin.go` → EMPTY, no `KeyIDs`/`catalog` refs in `internal/inference/
selection.go`). w6-f does NOT block on it and does NOT consume an absent endpoint.
The available models/connections the editor needs are sourced from SHIPPED surfaces:
1. **Connections/KeyIDs to pin**: the connection IDs are the pinnable keys. Source
   the connection list per provider from `GET /api/providers/{id}/connections`
   (w6-e SHIPPED, `routes_admin.go:56` → `catalogConnectionDTO{id,provider,...}`) —
   each connection `id` is a candidate KeyID. (If the operator pins raw key IDs
   rather than connection IDs, the editor accepts free-text/multiselect that writes
   into `key_ids[]`; the dropdown is populated from the connections list as the
   discoverable source.)
2. **Allowed models per provider**: `GET /api/providers/{id}/models` (w6-e SHIPPED,
   `routes_admin.go:57` → `catalogModelDTO[]`) or the full `/api/models` catalog,
   filtered by provider → populates `allowed_models[]`.
3. **Write**: the editor serializes the pinned selections into
   `provider_configs:[{provider,allowed_models,key_ids,weight?}]` and POSTs/PUTs to
   the REAL `/api/virtual-keys[/{id}]` (w5-g).

**Decision (binding):** the KeyIDs editor consumes w6-e's SHIPPED provider model/
connection endpoints + the e2e `/api/providers/...` mocks (w6-e owned, CONSUME) and
writes the real VK `provider_configs[].key_ids`. It does NOT depend on the absent
w6-pre `/api/catalog`. If a later orchestrator decision lands w6-pre's `/api/catalog`
and prefers it as the source, that is a serial follow-up swap (page reads a
different path) — NOT a w6-f blocker. Recorded in §8 ESC-1 + `open-questions.md`.

### 1.6b Provider-nodes Go contract (this plan's NEW Go, TDD)

The ModelSelectModal + endpoint flows reference `/api/provider-nodes` (custom
OpenAI-compatible endpoints). 9router (`providers/page.js:148,876,909`):
- `GET /api/provider-nodes` → `{nodes:[...]}` (`page.js:148,15`).
- `POST /api/provider-nodes` body `{name,prefix,apiType,baseUrl,type:"openai-compatible"}`
  → `{node}` (`page.js:876-889`).
- `POST /api/provider-nodes/validate` body `{baseUrl,apiKey,type,modelId?}` →
  `{valid,error?}` (`page.js:909-925`).

w6-f adds these as NEW, ADDITIVE Go in `internal/admin/nodes.go` (the existing
`providers.go` CRUD is FORBIDDEN to edit). A "provider node" is an
OpenAI-compatible custom provider — it COMPOSES the existing `providers` table
(`ProviderRecord{id,name,type,base_url,enabled,...}`, `internal/store/providers.go:
11-19`) filtered to `type == "openai-compatible"`. **No new table** — list reads
`store.ListProviders()` filtered by type; create calls `store.CreateProvider` with
`type="openai-compatible"`. **If a node needs fields the `providers` table lacks
(prefix/apiType), use ADDITIVE `ensureColumn` migrations ONLY** (pattern
`internal/store/migrate.go:259-261`; precedent the `*_enc` columns) — but prefer
mapping `prefix`→`name`/`id` and `apiType` into a metadata column only if strictly
needed; default is no schema change (decide at T2 against the test shape). New Go
follows AGENTS.md: snake_case `{data,error}` (`respond.go`), layered
(handler→`*store.Store`), no `init()`, errors-as-values, no secret exposure (the
validate `apiKey` is used transiently and NEVER persisted/echoed), TDD
(`nodes_test.go` committed RED first).

New file `internal/admin/nodes.go` (NEW) provides:

| Handler | Route (resolved) | Shape (snake_case, `{data}`) | PAR |
|---|---|---|---|
| `ListProviderNodes` | `GET /api/provider-nodes` | `{nodes:[{id,name,base_url,type,enabled}]}` — `store.ListProviders()` filtered to `type=="openai-compatible"`, mapped to node DTO | PAR-UI-109 |
| `CreateProviderNode` | `POST /api/provider-nodes` | body `{name,prefix?,api_type?,base_url}` (accept camelCase `baseUrl`/`apiType` from the 9router client OR snake_case — normalize at decode); creates a `providers` row `type="openai-compatible"`; returns `{node:{id,name,base_url,type,enabled}}`. 400 on missing name/base_url | PAR-UI-110 |
| `ValidateProviderNode` | `POST /api/provider-nodes/validate` | body `{base_url,api_key?,type,model_id?}`; performs a best-effort reachability check (in test: deterministic — `valid=true` if `base_url` is a well-formed http(s) URL, else `{valid:false,error:"invalid url"}`); NEVER persists `api_key`; returns `{valid,error?}` | PAR-UI-111 |

Route registration is the SERIAL-SLOT additive edit to `routes_admin.go` (§1.8/§3).
Register STATIC `/api/provider-nodes/validate` BEFORE the bare `/api/provider-nodes`
collision is irrelevant (different verbs/paths; validate is a distinct suffix) —
but verify fasthttp/router precedence at impl; a conflict is ESC-4 (§8).

### 1.7 `routeTree.gen.ts` is NOT touched

All three routes already exist as stubs (§1.1); rewriting their component bodies
does not change the route tree, and no new route file is added. Therefore
`ui/src/routeTree.gen.ts` is UNCHANGED by w6-f (MAP decision 6; w6-l is wave-2's
new-route plan, not w6-f). If a build incidentally reformats it, that is an
ESCALATION (§8), not an in-plan edit.

### 1.8 `endpoint.spec.ts` must be CREATED (committed RED first); serial-slot handling

**New spec.** There is NO `ui/e2e/endpoint.spec.ts` today (§1.1). The endpoint page
has no existing acceptance contract. Extending `keys.spec.ts` would conflate two
routes, so w6-f CREATES `ui/e2e/endpoint.spec.ts` as the endpoint acceptance
contract and commits it RED in T1 (precedent: w6-i created `translator.spec.ts`).
It is NEW-spec ownership by w6-f (§3 CREATE + §7 allowed diff). `keys.spec.ts` and
`virtual-keys.spec.ts` ALREADY exist and are EXTENDED with RED assertions, not
created.

**Serial slot (binding).** The MAP serial order is w6-pre→w6-d→w6-e→w6-j
(`MAP §Cross-cutting`). w6-e RELEASED the routes_admin.go slot to w6-j on its close
(`docs/WORKFLOW.md:34` — "serial slot released to w6-j"). w6-f (page wave 2, runs
∥ w6-j/w6-k/w6-l/w6-m) ALSO needs the slot (provider-nodes routes). **Resolution:**
the orchestrator MUST confirm at P7 that NO plan currently holds an unmerged
routes_admin.go edit (w6-e is merged; w6-j must not have started its slot edit yet).
w6-f TAKES the slot, lands its single additive routes_admin.go commit (T3), and
RELEASES it to w6-j on close. Since w6-f and w6-j both want the slot in wave 2, the
orchestrator serializes them — run w6-f's T3 (the slot edit) before w6-j begins its
routes_admin.go edit, OR run w6-j's slot edit first and have w6-f rebase. State the
chosen order in WORKFLOW.md at closeout. **Only ONE unmerged routes_admin.go holder
at a time** (MAP decision 5).

### 1.9 Sanctioned mock-index edit (the ONE exception)

Page plans normally must NOT edit `ui/e2e/mocks/handlers/index.ts`. w6-f's
provider-nodes surface has NO existing mock (§1.4), so w6-f CREATES
`handlers/nodes.ts` (provider-nodes + model-test + availability) AND adds its
registration to `handlers/index.ts`. This is the ONE sanctioned `index.ts` edit,
bounded to ADDING the nodes registration (an import + a
`registerNodesHandlers(page, store)` call) — it does NOT modify/reorder/remove any
existing registration. If adding it would collide with a sibling plan's pending
index change, STOP and ESCALATE (§8 ESC-5). The seed `index.ts` and `store.ts` are
NOT touched unless the corrected `keys.ts`/`virtual-keys.ts` seeds require a store
field change — if a store field is genuinely required, that is an ESCALATION, not an
in-plan store edit (the keys/VK store fields `keys`/`virtualKeys` already exist).

### NOT in scope (explicit)

- **No new route FILES** — only the three existing stubs are rewritten;
  `routeTree.gen.ts` untouched (§1.7).
- **No edits to existing Go** — `internal/admin/apikeys.go`,
  `internal/admin/virtualkeys.go`, `internal/admin/providers.go`,
  `internal/admin/providers_catalog.go`, `internal/admin/disabledmodels.go`,
  `internal/admin/combos.go`, `internal/store/**`, `internal/schemas/**` are
  FORBIDDEN; w6-f only ADDS `internal/admin/nodes.go` (+ its `_test.go`) and
  ADDITIVE provider-nodes route lines in `routes_admin.go`. ADDITIVE `ensureColumn`
  ONLY if the node shape strictly requires it (§1.6b) — default no schema change.
- **No edits to any frozen w6-a/w6-b/w6-c/w6-e file** — no `__root.tsx`, layout,
  `ui/src/components/ui/*`, stores, `lib/api.ts`, `lib/auth.ts`, `lib/utils.ts`,
  `providers/*`, `callback.tsx`, `login.tsx`, `ui/src/routes/{providers,
  connections,models}.tsx`, `ui/src/components/providers/**`,
  `ui/src/lib/oauth-popup.ts`. No header exception remains (SPENT).
- **No TanStack Query wiring** (§1.5) — plain `apiFetch`.
- **No dependency additions** — every import resolves to installed packages or
  w6-a/b/c/e outputs.
- **No edits to w6-e/foundation-owned mocks** — `models.ts`, `combos.ts`,
  `catalog.ts`, `mocks/seed/index.ts`, `mocks/store.ts`, `mocks/fixture.ts`,
  `mocks/handlers/providers.ts`/`connections.ts` are CONSUMED unchanged. The ONLY
  `handlers/index.ts` edit is the nodes registration (§1.9). w6-f corrects ONLY
  the `keys.ts` + `virtual-keys.ts` handler BODIES + `seed/keys.ts` +
  `seed/virtual-keys.ts` (w6-f-owned, §8 ESC-2).
- **No other e2e specs** beyond `keys.spec.ts` + `virtual-keys.spec.ts` (extended)
  and the NEW `endpoint.spec.ts`. NOT `providers`/`models`/`combos` specs (w6-e/
  w6-h own those).
- **No real outbound provider network in tests** — `validate` is deterministic
  under test (URL well-formedness), `models/test` returns a fixed `{ok}`.
- **No combos/routing/aliases pages** (w6-h), no providers/connections/models
  pages (w6-e), no settings (w6-j), no usage/pricing (w6-g).

---

## 2. Precondition checks

Run all before any edit; abort and report to orchestrator on any failure.

```bash
# P0 — clean tree; record <base>
git status --porcelain     # must be empty (untracked tooling artifacts must be
                           # gitignored; worker uses explicit `git add <file>`,
                           # never `git add -A`; ui/dist/** is gitignored — never
                           # stage it)
git rev-parse HEAD         # record as <base> for §5 (expected bfa9436)

# P1 — w6-b primitives present and frozen (consumed)
ls ui/src/components/ui/*.tsx | grep -v test | wc -l    # = 16 (w6-b set intact)
grep -n "export function Modal\|export interface ModalProps" ui/src/components/ui/modal.tsx
grep -n "ConfirmModal" ui/src/components/ui/confirm-modal.tsx
grep -n "SegmentedControl" ui/src/components/ui/segmented-control.tsx
grep -n "Pagination" ui/src/components/ui/pagination.tsx

# P2 — w6-a foundation present and frozen (consumed; PAR-UI-081 already HAVE)
grep -n "export async function apiFetch" ui/src/lib/api.ts
grep -n "export class ApiError" ui/src/lib/api.ts
grep -n "push:" ui/src/stores/notification.ts
grep -rn "QueryClientProvider\|QueryClient" ui/src/routes/__root.tsx ui/src/main.tsx ; echo "^ expect EMPTY (not mounted; do NOT mount)"

# P3 — w6-e provider model/connection read endpoints SHIPPED (consumed by KeyIDs editor/ModelSelectModal)
grep -n "ListProviderCatalog\|GetProviderModels\|GetProviderConnections\|GetProviderSuggestedModels" internal/server/routes_admin.go  # routes 53-58
grep -n "catalogModelDTO\|catalogConnectionDTO" internal/admin/providers_catalog.go
test -f ui/e2e/mocks/handlers/models.ts && test -f ui/e2e/mocks/handlers/combos.ts && echo "models+combos mocks present (consume)"

# P4 — the three route stubs are still bare (safe to rewrite); routeTree already has them; no new dirs
grep -n "<h1>Endpoint</h1>"     ui/src/routes/endpoint.tsx
grep -n "<h1>Keys</h1>"         ui/src/routes/keys.tsx
grep -n "<h1>Virtual Keys</h1>" ui/src/routes/virtual-keys.tsx
grep -nE "'/endpoint'|'/keys'|'/virtual-keys'|EndpointRoute|KeysRoute|VirtualKeysRoute" ui/src/routeTree.gen.ts ; echo "^ expect ALL THREE present (no new route file; tree UNCHANGED §1.7)"
test ! -d ui/src/components/keys && echo "keys components dir absent (good)"
test ! -e ui/e2e/endpoint.spec.ts && echo "endpoint spec absent (good — CREATE RED §1.8)"

# P5 — e2e mock harness present + registered (CONSUME; correct keys/VK bodies §1.4; NEW nodes mock §1.9)
grep -n "registerKeysHandlers\|registerVirtualKeysHandlers" ui/e2e/mocks/handlers/index.ts
grep -n "seedKeys\|seedVirtualKeys" ui/e2e/mocks/seed/index.ts
grep -rn "provider-nodes\|registerNodes" ui/e2e/mocks/handlers/ ; echo "^ expect EMPTY (no nodes mock yet; CREATE handlers/nodes.ts + register §1.9)"
grep -rn "models/test\|models/availability" ui/e2e/mocks/handlers/ ; echo "^ expect EMPTY (new mock bodies §1.4)"
grep -n "/api/models/custom" ui/e2e/mocks/handlers/models.ts ; echo "^ custom-models mock present (w6-e owned; consume)"

# P6 — Go reality: keys/VK/disabled real; provider-nodes/models-test/availability ABSENT
grep -nE '/api/(keys|virtual-keys)' internal/server/routes_admin.go        # full CRUD present (66-76)
grep -n "key_ids\|ProviderConfigs" internal/admin/virtualkeys.go internal/schemas/governance.go  # KeyIDs field present
grep -nE '/api/provider-nodes|/api/models/test|/api/models/availability' internal/server/routes_admin.go ; echo "^ expect EMPTY (the gaps §1.6/§1.6b)"
test ! -e internal/admin/nodes.go && echo "nodes.go absent (good — NEW §1.6b)"
grep -rn '/api/catalog\|inference/catalog' internal/ ; echo "^ expect EMPTY (w6-pre NOT landed; KeyIDs editor uses w6-e endpoints §1.6)"

# P7 — routes_admin.go serial slot is FREE (no other unmerged holder); take it (§1.8)
git log --oneline -3 -- internal/server/routes_admin.go   # last touch = w6-e (merged); slot released to w6-j
grep -n "serial slot released to w6-j" docs/WORKFLOW.md     # confirm w6-e released it
# Orchestrator MUST confirm no concurrent wave-2 plan (esp. w6-j) holds an unmerged
# routes_admin.go edit before w6-f begins T3. w6-f takes the slot, releases to w6-j on close.

# P8 — harness green at base
cd ui && npx playwright test e2e/keys.spec.ts e2e/virtual-keys.spec.ts
# Record base result: stubs render only <h1>. keys.spec asserts body "API Keys"
# (stub renders <h1>Keys</h1>; PASS only if sidebar chrome carries "API Keys" —
# RECORD exact pass/fail). virtual-keys.spec "Virtual Keys" likely PASSES on the
# stub <h1>. Record in WORKFLOW.md.
cd ui && npm run build                               # exit 0
cd ui && npx vitest run src/                         # exit 0 (existing units green)
go test ./... && go vet ./...                        # exit 0 (Go untouched-green)
```

---

## 3. Exclusive file ownership

After w6-f merges, all CREATE files below are owned by w6-f; later plans consume,
never edit (MAP decision 7).

**CREATE — routes (REWRITE existing stubs; no new route files, §1.7):**

| File | Exports / contract |
|---|---|
| `ui/src/routes/endpoint.tsx` (REWRITE) | `Route=createFileRoute("/endpoint")`; `EndpointPage`: base-url panel (origin + `/v1`, copy action, sample snippet) + embedded `<ApiKeysPanel>` (compact). Reads `apiFetch("/api/keys")`. Body contains "Endpoint". Ports `EndpointPageClient.js`. |
| `ui/src/routes/keys.tsx` (REWRITE) | `Route=createFileRoute("/keys")`; `KeysPage`: full `<ApiKeysPanel>` — list/create/toggle/delete against `/api/keys` (real Go shape, §1.3). Header text contains "API Keys" (the `keys.spec.ts` marker). |
| `ui/src/routes/virtual-keys.tsx` (REWRITE) | `Route=createFileRoute("/virtual-keys")`; `VirtualKeysPage`: list VKs (name, budget used/limit, RPM, active `Toggle`) from `apiFetch("/api/virtual-keys")`; create/edit via `<VirtualKeyFormModal>` (incl. the KeyIDs pinning editor, §1.6); delete via `ConfirmModal`. Body contains "Virtual Keys". |

**CREATE — page/domain components (`ui/src/components/keys/`):**

| File | Exports / contract |
|---|---|
| `api-keys-panel.tsx` | `ApiKeysPanel` (PAR-UI-006/115) — consumes `Card`+`Button`+`Input`+`ConfirmModal`; list `/api/keys`; create (`POST {name}`); toggle (`PUT {is_active}`); delete; show created `key` once (real Go `apiKeyDTO`, §1.3). Props for compact (endpoint) vs full (keys page) mode. |
| `virtual-key-form-modal.tsx` | `VirtualKeyFormModal` (PAR-UI-130 VK) — consumes `Modal`+`Input`+`Select`+`Toggle`; fields name, budget (`{limit,period}`), `rate_limit_rpm`, is_active; embeds `<KeyIdsEditor>`; serializes `provider_configs[{provider,allowed_models,key_ids,weight?}]`; POST/PUT `/api/virtual-keys[/{id}]` (real w5-g shape, §1.6). |
| `key-ids-editor.tsx` | `KeyIdsEditor` (the w6-pre integration, §1.6) — per provider-config row: provider `Select`; `allowed_models` multiselect from `apiFetch("/api/providers/{id}/models")` or `/api/models`; `key_ids` multiselect/free-text sourced from `apiFetch("/api/providers/{id}/connections")` (w6-e SHIPPED). Emits `ProviderConfig[]`. |
| `model-select-modal.tsx` | `ModelSelectModal` (PAR-UI-049) — consumes `Modal`+`Input`+`SegmentedControl`; hierarchical picker: combos group (`/api/combos`), per-provider models (`/api/models`), custom (`/api/models/custom`), disabled hidden (`/api/models/disabled`). Optional model-test (`POST /api/models/test`) + availability (`GET /api/models/availability`) badges. Ports `ModelSelectModal.js`. |
| `provider-node-modal.tsx` | `ProviderNodeModal` (PAR-UI-109/110/111 UI) — consumes `Modal`+`Input`+`Button`; create custom OpenAI-compatible node (`POST /api/provider-nodes` `{name,prefix,apiType,baseUrl,type}`); validate (`POST /api/provider-nodes/validate` `{baseUrl,apiKey,modelId?}`) → show `{valid,error?}`. Lists nodes from `GET /api/provider-nodes`. |

**CREATE — unit tests (vitest — logic reachable without a DOM):**

| File | Contents |
|---|---|
| `ui/src/components/keys/key-ids-editor.test.tsx` | ≥3 tests via `renderToString`/stubbed `apiFetch`: renders a provider-config row; selecting models populates `allowed_models`; the emitted value serializes to `provider_configs[].key_ids` (the real VK shape, §1.6). Committed RED before `key-ids-editor.tsx`. |
| `ui/src/components/keys/api-keys-panel.test.tsx` | ≥2 tests via `renderToString`/stubbed `apiFetch`: renders seeded keys (real DTO `{key,name,is_active}`); a create call POSTs `{name}` and shows the returned `key`. Committed RED before `api-keys-panel.tsx`. |

(`model-select-modal.tsx`, `provider-node-modal.tsx`, `virtual-key-form-modal.tsx`
are DOM-heavy; their coverage is the e2e assertions — same disposition as w6-e/w6-g
modal components.)

**CREATE — Go (`internal/admin/nodes.go` + `_test.go`, NEW):**

| File | Contents |
|---|---|
| `internal/admin/nodes.go` | `ListProviderNodes`/`CreateProviderNode`/`ValidateProviderNode` + node DTO structs, per §1.6b. Composes `h.store.ListProviders`/`CreateProvider` filtered to `type=="openai-compatible"`. Uses `writeData`/`writeError` (`respond.go`). No `init()`; errors-as-values; `api_key` NEVER persisted/echoed. |
| `internal/admin/nodes_test.go` | Table-driven tests via `newTestEnv` (`admin_test.go:24`): list returns only openai-compatible providers as nodes; create persists a node and returns `{node}`; create with missing name/base_url → 400; validate returns `{valid:true}` for a well-formed URL and `{valid:false,error}` for a bad one; validate NEVER persists the api_key (assert no provider row created by validate). Committed RED before the impl file. |

**MODIFY — serial-slot route registration (additive only):**

| File | Change (and ONLY this change) |
|---|---|
| `internal/server/routes_admin.go` | ADD (in/after the providers block): `r.GET("/api/provider-nodes", h.RequireSession(h.ListProviderNodes))`, `r.POST("/api/provider-nodes", h.RequireSession(h.CreateProviderNode))`, `r.POST("/api/provider-nodes/validate", h.RequireSession(h.ValidateProviderNode))`. Route-order note: `/api/provider-nodes/validate` is a distinct suffix from the bare collection — verify fasthttp/router precedence at impl; a conflict is ESC-4 (§8), not a silent path change. NOTHING else changes. Diff bound §5: ≤ 6 added lines. SERIAL SLOT — only holder while live; RELEASE to w6-j on close (§1.8). |

**MODIFY — e2e (the acceptance contract; correct keys/VK bodies+seeds §1.4/§8 ESC-2):**

| File | Change |
|---|---|
| `ui/e2e/keys.spec.ts` | KEEP the 1 existing test ("API Keys"). ADD RED: keys list renders ≥1 row; create-key flow POSTs `{name}` and shows the returned `key`; delete asks `ConfirmModal`. |
| `ui/e2e/virtual-keys.spec.ts` | KEEP the 1 existing test ("Virtual Keys"). ADD RED: VK list renders ≥1 row (budget used/limit, RPM); opening the form modal shows the KeyIDs editor; saving POSTs `provider_configs[].key_ids`. |
| `ui/e2e/endpoint.spec.ts` (**NEW**, §1.8) | RED first: `/endpoint` body contains "Endpoint"; base-url block visible (origin/`/v1`); embedded keys widget renders ≥1 seeded key; (optional) provider-node modal opens. |
| `ui/e2e/mocks/handlers/keys.ts` (BODY) | CORRECT to mirror real Go `apiKeyDTO` (`key`/`machine_id`, create body `{name}`); drop or no-op `/{id}/regenerate` if the page does not call it (no Go regenerate, §1.4). w6-f-owned body. |
| `ui/e2e/mocks/seed/keys.ts` (BODY) | CORRECT seed entries to the real DTO shape (`key`/`machine_id`/`is_active`/`created_at`); extra UI fields optional. |
| `ui/e2e/mocks/handlers/virtual-keys.ts` (BODY) | CORRECT to real `virtualKeyDTO` (nested `budget{limit,period,used}`, `provider_configs[]`); create/update require `provider_configs` w/ non-empty `key_ids` (`virtualkeys.go:55-66`). w6-f-owned body. |
| `ui/e2e/mocks/seed/virtual-keys.ts` (BODY) | CORRECT seed to the real DTO (nested budget + `provider_configs[{provider,allowed_models,key_ids,weight?}]`). |
| `ui/e2e/mocks/handlers/nodes.ts` (**NEW**, §1.9) | `registerNodesHandlers(page, store)`: `GET /api/provider-nodes`→`{nodes:[...]}`, `POST /api/provider-nodes`→`{node}`, `POST /api/provider-nodes/validate`→`{valid,error?}`, `POST /api/models/test`→`{ok,latency_ms}`, `GET /api/models/availability`→`{available:[...]}`. Self-contained. |
| `ui/e2e/mocks/handlers/index.ts` (ADD ONE registration, §1.9) | Add `import { registerNodesHandlers } from "./nodes";` + one `registerNodesHandlers(page, store);` call. NO modify/reorder/remove of existing registrations. |

**FORBIDDEN:** everything else. Explicitly: all of `internal/admin/apikeys.go`,
`internal/admin/virtualkeys.go`, `internal/admin/providers.go`,
`internal/admin/providers_catalog.go`, `internal/admin/disabledmodels.go`,
`internal/admin/combos.go`, `internal/store/**` (default — additive `ensureColumn`
ONLY if §1.6b strictly requires it, else FORBIDDEN), `internal/schemas/**`; all
`ui/src/components/ui/*` (w6-b); all `ui/src/stores/*`, `ui/src/lib/api.ts`,
`ui/src/lib/utils.ts`, `ui/src/providers/*`, `ui/src/lib/auth.ts`,
`ui/src/lib/oauth-popup.ts`, `ui/src/routes/{callback,login,providers,connections,
models}.tsx`, `ui/src/components/{auth,providers}/**` (w6-a/b/c/e); `__root.tsx`,
layout, `ui/src/main.tsx`; `ui/package.json` + lockfile; `ui/vite.config.ts`;
`ui/playwright.config.ts`; `ui/components.json`; `ui/src/index.css`;
`ui/src/routeTree.gen.ts` (UNCHANGED §1.7); `ui/e2e/mocks/handlers/{models,combos,
catalog,providers,connections}.ts` (w6-e/foundation owned — CONSUME); `mocks/seed/
index.ts`; `mocks/store.ts`; `mocks/fixture.ts`; all other `ui/e2e/*.spec.ts`; all
other `internal/server/*` routes; any other `internal/admin/*.go`.

---

## 4. TDD tasks

Cadence (strict): **no route/component/Go file may exist (or be rewritten beyond
its stub) before the failing test that covers it is committed.** Both tracks are
strict-TDD: Go `_test.go` before Go impl; UI red specs/units before UI impl; new
specs committed RED first. `cd ui && npm run build` green at EVERY commit (test
files + red specs are never imported by production code). `go test ./... && go vet
./...` and `go build ./...` green at EVERY commit. The three e2e specs (keys
extended, virtual-keys extended, endpoint NEW) stay RED from T1 until impl greens
them.

### T1 — STEP(a): extend keys/VK specs + CREATE endpoint spec + NEW nodes mock + correct keys/VK bodies (commit RED)

Add RED assertions to `keys.spec.ts` + `virtual-keys.spec.ts` (§3). CREATE
`ui/e2e/endpoint.spec.ts` RED (§1.8). CREATE `ui/e2e/mocks/handlers/nodes.ts` +
register in `handlers/index.ts` (§1.9). CORRECT the `keys.ts`/`virtual-keys.ts`
handler BODIES + `seed/keys.ts`/`seed/virtual-keys.ts` to the real Go DTOs (§1.4/§8
ESC-2). CONSUME `models.ts`/`combos.ts`/`catalog.ts`/`providers.ts`/`connections.ts`
unchanged.

STEP(b): run all three specs — **record failure output** (no keys list/create, no
VK form/KeyIDs editor, no endpoint base-url/keys widget). Commit RED:
`phase-1/w6-f: failing keys/virtual-keys e2e + new endpoint spec + nodes mock + keys/VK body corrections (TDD red)`.

**Mock-vs-reality gate**: while correcting `keys.ts`/`virtual-keys.ts` bodies,
re-read the Go DTOs (§1.2 file:lines). If a real shape contradicts a corrected body,
OR a correction breaks a non-w6-f spec, STOP and ESCALATE (§8) — no existing-Go
edit, no mock fudge, no foundation-mock edit. The KeyIDs editor consumes w6-e's
SHIPPED `/api/providers/{id}/models|connections` — confirm those mocks exist
(`models.ts`/`providers.ts`/`connections.ts`); if the editor needs a path the
mocks lack, ESCALATE (do NOT edit w6-e mocks).

### T2 — STEP(a): Go `nodes_test.go` (commit RED)

Write the table-driven tests per §3 against `newTestEnv` (real store, temp DB).
Decide the no-schema-change-vs-`ensureColumn` question here against the test shape
(§1.6b — default no schema change). `go test ./internal/admin/ -run Nodes` →
FAILS (handlers missing). Record failure. Commit RED:
`phase-1/w6-f: failing provider-nodes Go tests (TDD red)`.

### T3 — STEP(b): Go `nodes.go` + serial-slot route registration

Implement the three handlers per §1.6b (compose `store.ListProviders`/
`CreateProvider` filtered to openai-compatible; deterministic validate; api_key
never persisted). Add the additive route lines to `routes_admin.go` (§3; verify
precedence — ESC-4 if conflict). **Take the serial slot first (§1.8 — orchestrator
confirms it is free).** Gates: `go test ./... && go vet ./... && go build ./...`
green (nodes tests now green). Commit:
`phase-1/w6-f: provider-nodes admin API (list/create/validate) + serial-slot routes`.

### T4 — STEP(b): keys/endpoint pages + ApiKeysPanel + provider-node modal

STEP(a) first: ensure `api-keys-panel.test.tsx` is committed RED (write here if
not, run vitest red, commit:
`phase-1/w6-f: failing unit tests for api-keys-panel (TDD red)`). STEP(b): implement
`api-keys-panel.tsx`, `provider-node-modal.tsx`; rewrite `endpoint.tsx` + `keys.tsx`.
Gates: vitest green; `keys.spec.ts` + `endpoint.spec.ts` green; `npm run build`
green; `go test ./... && go vet ./...` green. Commit:
`phase-1/w6-f: endpoint + keys pages, ApiKeysPanel, provider-node modal`.

### T5 — STEP(b): virtual-keys page + KeyIDs editor + ModelSelectModal

STEP(a) first: ensure `key-ids-editor.test.tsx` is committed RED (write here if
not, run vitest red, commit:
`phase-1/w6-f: failing unit tests for key-ids-editor (TDD red)`). STEP(b): implement
`key-ids-editor.tsx`, `virtual-key-form-modal.tsx`, `model-select-modal.tsx`;
rewrite `virtual-keys.tsx`. Gates: vitest green; ALL of `keys.spec.ts`/
`virtual-keys.spec.ts`/`endpoint.spec.ts` green; `npm run build` green. Commit:
`phase-1/w6-f: virtual-keys page, KeyIDs editor, ModelSelectModal`.

### T6 — full gates + closeout

```bash
cd ui && npm run build
cd ui && npx playwright test e2e/keys.spec.ts e2e/virtual-keys.spec.ts e2e/endpoint.spec.ts   # all green
cd ui && npx playwright test                             # full suite: no spec green-at-base may be red
cd ui && npx vitest run src/                             # all green incl new units
go test ./... && go vet ./... && go build ./...          # green
go test ./internal/admin/ -run Nodes -v                  # ≥5 node cases pass
```
Flip §1 matrix rows in `.planning/parity/matrix/9router-ui.md`: PAR-UI-006 → HAVE
(variant, cite §1.3/§1.5); PAR-UI-049 → HAVE (cite §1.4); PAR-UI-109/110/111 → HAVE
(Go, cite §1.6b); PAR-UI-115 → HAVE (real Go, cite §8 ESC-2); PAR-UI-117/119/120 →
HAVE (variant, cite §1.4/§8 ESC-3); PAR-UI-118 → HAVE (real Go); PAR-UI-130
`/virtual-keys`+`/endpoint` subset → HAVE. Update `docs/WORKFLOW.md` (record P8 base
spec observations, the keys/VK body corrections §8 ESC-2, the KeyIDs/w6-pre decision
§1.6, the serial-slot take-from-w6-e-released/release-to-w6-j order §1.8, the new
endpoint.spec + nodes mock, and the §8 mock-only follow-ups). Append §8 open items
to `.planning/parity/plans/open-questions.md`. Final commit:
`phase-1/w6-f: close — endpoint/keys/virtual-keys cluster; provider-nodes Go; matrix flips`.
**On the close commit, RELEASE the routes_admin.go serial slot to w6-j (§1.8).**

---

## 5. Binary acceptance criteria

All must hold; each is yes/no. `<base>` = the commit recorded at P0 (bfa9436 at
authoring). Diff gate is **w6-f commit-range-scoped** (§7).

**Test gates**
- `cd ui && npx playwright test e2e/keys.spec.ts` → exit 0, all pass, 0 skipped.
- `cd ui && npx playwright test e2e/virtual-keys.spec.ts` → exit 0, all pass.
- `cd ui && npx playwright test e2e/endpoint.spec.ts` → exit 0, all pass (NEW spec).
- `cd ui && npx vitest run src/` → exit 0 (all prior + new units green).
- `cd ui && npm run build` → exit 0.
- `go test ./... && go vet ./... && go build ./...` → exit 0.
- `go test ./internal/admin/ -run Nodes -v` → exit 0, ≥5 node cases pass.

**TDD-order proof** — each impl file's covering test appears in an
earlier-or-equal commit:
```bash
# Go: nodes.go after nodes_test.go
ct=$(git log --format=%ct --diff-filter=A -1 -- internal/admin/nodes_test.go)
cf=$(git log --format=%ct --diff-filter=A -1 -- internal/admin/nodes.go)
[ "$ct" -le "$cf" ] || echo "TDD VIOLATION: nodes.go"                  # prints nothing
# UI units after their tests
ct=$(git log --format=%ct --diff-filter=A -1 -- ui/src/components/keys/key-ids-editor.test.tsx)
cf=$(git log --format=%ct --diff-filter=A -1 -- ui/src/components/keys/key-ids-editor.tsx)
[ "$ct" -le "$cf" ] || echo "TDD VIOLATION: key-ids-editor.tsx"        # nothing
ct=$(git log --format=%ct --diff-filter=A -1 -- ui/src/components/keys/api-keys-panel.test.tsx)
cf=$(git log --format=%ct --diff-filter=A -1 -- ui/src/components/keys/api-keys-panel.tsx)
[ "$ct" -le "$cf" ] || echo "TDD VIOLATION: api-keys-panel.tsx"        # nothing
# e2e RED commit precedes the page rewrites
sa=$(git log --format=%ct -1 --grep="failing keys/virtual-keys e2e")
pi=$(git log --format=%ct --diff-filter=M -1 -- ui/src/routes/virtual-keys.tsx)
[ "$sa" -le "$pi" ] || echo "TDD VIOLATION: virtual-keys.tsx before red spec"  # nothing
```

**Grep proofs**
```bash
grep -rn "/api/keys" ui/src/components/keys/api-keys-panel.tsx              # PAR-UI-006/115
grep -rn "/api/virtual-keys" ui/src/routes/virtual-keys.tsx                 # PAR-UI-130 VK
grep -rn "provider_configs\|key_ids" ui/src/components/keys/key-ids-editor.tsx ui/src/components/keys/virtual-key-form-modal.tsx  # §1.6 KeyIDs editor
grep -rn "/api/providers/.*/\(models\|connections\)\|/api/models" ui/src/components/keys/key-ids-editor.tsx  # §1.6 source endpoints
grep -rn "/api/combos\|/api/models" ui/src/components/keys/model-select-modal.tsx  # PAR-UI-049
grep -rn "/api/models/disabled" ui/src/components/keys/model-select-modal.tsx       # PAR-UI-118
grep -rn "/api/models/custom" ui/src/components/keys/model-select-modal.tsx         # PAR-UI-117
grep -rn "/api/provider-nodes" ui/src/components/keys/provider-node-modal.tsx       # PAR-UI-109/110/111 UI
grep -rn "/v1\|location.origin" ui/src/routes/endpoint.tsx                  # PAR-UI-006 base-url
# Go provider-nodes:
grep -n "ListProviderNodes\|CreateProviderNode\|ValidateProviderNode" internal/admin/nodes.go  # PAR-UI-109/110/111
grep -n "openai-compatible" internal/admin/nodes.go                          # node = compatible provider §1.6b
grep -n "writeData\|writeError" internal/admin/nodes.go                      # snake_case {data,error}
grep -nE '/api/provider-nodes' internal/server/routes_admin.go               # routes registered
! grep -nE "api_?[Kk]ey.*=.*store|store.*api_?[Kk]ey" internal/admin/nodes.go && echo "validate api_key not persisted OK"  # §1.6b no secret persistence
! grep -n "func init(" internal/admin/nodes.go && echo "no init() OK"
```

**Negative / freeze proofs (w6-f commit-range — see §7)**
```bash
R="<first-w6-f>^..<last-w6-f>"
git diff $R --name-only -- internal/admin/apikeys.go internal/admin/virtualkeys.go internal/admin/providers.go internal/admin/providers_catalog.go internal/admin/disabledmodels.go internal/admin/combos.go internal/schemas/ | wc -l  # = 0 (existing Go frozen)
git diff $R --name-only -- internal/store/ | wc -l                      # = 0 (default; nonzero ONLY if §1.6b ensureColumn used — then ≤1 additive file, justify)
git diff $R --name-only -- internal/ | grep -vE 'internal/admin/nodes(_test)?\.go|internal/server/routes_admin\.go' | wc -l  # = 0 (only the new file + serial slot)
git diff $R --name-only -- ui/src/components/ui/ | wc -l                # = 0 (w6-b frozen)
git diff $R --name-only -- ui/src/stores/ ui/src/providers/ ui/src/lib/api.ts ui/src/lib/utils.ts ui/src/lib/auth.ts ui/src/lib/oauth-popup.ts | wc -l  # = 0 (w6-a/c/e frozen)
git diff $R --name-only -- ui/src/routes/__root.tsx ui/src/routes/callback.tsx ui/src/routes/login.tsx ui/src/routes/providers.tsx ui/src/routes/connections.tsx ui/src/routes/models.tsx ui/src/components/layout/ ui/src/components/auth/ ui/src/components/providers/ ui/src/main.tsx | wc -l  # = 0
git diff $R --name-only -- ui/src/routeTree.gen.ts | wc -l             # = 0 (§1.7 unchanged)
git diff $R --name-only -- ui/package.json ui/package-lock.json ui/vite.config.ts ui/playwright.config.ts ui/components.json ui/src/index.css | wc -l  # = 0
git diff $R --name-only -- 'ui/src/routes/' | grep -vE 'endpoint\.tsx|keys\.tsx|virtual-keys\.tsx' | wc -l  # = 0 (only the three stubs rewritten)
git diff $R --name-only -- ui/e2e/ | grep -vE 'keys\.spec\.ts|virtual-keys\.spec\.ts|endpoint\.spec\.ts|mocks/handlers/(keys|virtual-keys|nodes|index)\.ts|mocks/seed/(keys|virtual-keys)\.ts' | wc -l  # = 0
git diff $R --name-only -- ui/e2e/mocks/handlers/ | grep -vE '(keys|virtual-keys|nodes|index)\.ts' | wc -l  # = 0 (models/combos/catalog/providers/connections untouched)
git diff $R --name-only -- ui/e2e/mocks/ | grep -E 'fixture\.ts|store\.ts|seed/index\.ts|catalog\.ts' | wc -l  # = 0 (foundation untouched)
git diff $R -- internal/server/routes_admin.go | grep "^+" | wc -l     # ≤ 6 (additive route lines + +++ header)
git log --oneline $R -- internal/server/routes_admin.go | wc -l        # = 1 (exactly ONE commit touches the serial-slot file)
```

---

## 6. Out of scope (restated, binding)

No new route files / no `routeTree.gen.ts` change (§1.7); no edits to existing Go
(apikeys/virtualkeys/providers/providers_catalog/disabledmodels/combos/store/schemas
frozen — additive `ensureColumn` ONLY if §1.6b strictly requires, else none) —
only the additive `nodes.go` + serial-slot routes; no edits to any frozen
w6-a/b/c/e file (no header exception remains — SPENT); no TanStack Query wiring
(§1.5); no dependency additions; no edits to w6-e/foundation mocks
(models/combos/catalog/providers/connections, seed/index, store, fixture) — only
`keys.ts`/`virtual-keys.ts` bodies + seeds corrected + NEW `nodes.ts` + one index
append (§1.9); no other e2e specs; no combos/routing/aliases (w6-h); no providers/
connections/models pages (w6-e); no settings (w6-j); no usage/pricing (w6-g); no
real outbound provider network in tests. Mock-vs-Go contradiction → escalate (§8),
never patch existing Go or fudge a mock. The KeyIDs editor does NOT consume the
absent w6-pre `/api/catalog` (§1.6 / §8 ESC-1).

## 7. Diff-gate scope

Page-wave-2 plans (w6-f/j/k/l/m) may commit to main concurrently, so a broad
`<base>..HEAD` range sweeps in sibling commits. The diff gate MUST be scoped to
w6-f's own commits. The orchestrator isolates them with:
`git log --oneline main | grep "^[0-9a-f]* phase-1/w6-f:" | awk '{print $1}'`
and runs `git diff <first-w6-f>^..<last-w6-f> -- [file list]` (same commit-range
scoping as w6-e §7 / w6-i §7).

`git diff <first-w6-f>^..<last-w6-f> --name-only` must be exactly a subset of:

```
ui/src/routes/endpoint.tsx
ui/src/routes/keys.tsx
ui/src/routes/virtual-keys.tsx
ui/src/components/keys/api-keys-panel.tsx
ui/src/components/keys/api-keys-panel.test.tsx
ui/src/components/keys/virtual-key-form-modal.tsx
ui/src/components/keys/key-ids-editor.tsx
ui/src/components/keys/key-ids-editor.test.tsx
ui/src/components/keys/model-select-modal.tsx
ui/src/components/keys/provider-node-modal.tsx
ui/e2e/keys.spec.ts
ui/e2e/virtual-keys.spec.ts
ui/e2e/endpoint.spec.ts                    (NEW spec, §1.8)
ui/e2e/mocks/handlers/keys.ts             (body only — real Go DTO correction §8 ESC-2)
ui/e2e/mocks/handlers/virtual-keys.ts     (body only — real Go DTO correction §8 ESC-2)
ui/e2e/mocks/handlers/nodes.ts            (NEW mock, §1.9)
ui/e2e/mocks/handlers/index.ts            (ADD ONE nodes registration only, §1.9)
ui/e2e/mocks/seed/keys.ts                 (body only — real DTO correction)
ui/e2e/mocks/seed/virtual-keys.ts         (body only — real DTO correction)
internal/admin/nodes.go
internal/admin/nodes_test.go
internal/server/routes_admin.go           (serial-slot additive route lines; ONE commit)
.planning/parity/matrix/9router-ui.md
docs/WORKFLOW.md
[internal/store/<file> + migrate ONLY IF §1.6b ensureColumn proves strictly needed — justify]
```

Any file outside this list in the scoped diff is an automatic review REJECT.
`internal/admin/apikeys.go`, `internal/admin/virtualkeys.go`,
`internal/admin/providers*.go`, `internal/schemas/**`, `ui/src/routeTree.gen.ts`,
`ui/e2e/mocks/handlers/{models,combos,catalog,providers,connections}.ts`,
`ui/e2e/mocks/{fixture,store}.ts`, `ui/e2e/mocks/seed/index.ts`, and any frozen
w6-a/b/c/e file are deliberately ABSENT — touching them is an automatic REJECT. The
`routes_admin.go` edit must appear in exactly ONE commit (the §5 `git log … | wc -l`
= 1 proof) and the serial slot is released to w6-j on close (§1.8). After merge,
the three pages, `ui/src/components/keys/**`, `internal/admin/nodes.go`, and the
corrected keys/VK mock bodies become consume-only for later plans.

## 8. Escalations / cross-track dependencies

- **ESC-1 (RESOLVED at authoring — w6-pre catalog / KeyIDs source, binding)**: the
  MAP says the KeyIDs editor consumes "w6-pre's catalog API," but **w6-pre has NOT
  landed** (`internal/admin/catalog.go`/`internal/inference/catalog.go` ABSENT, no
  `/api/catalog` route, no KeyIDs threading in `selection.go`). **Decision**: the
  editor sources available models from w6-e's SHIPPED `GET /api/providers/{id}/models`
  + `GET /api/models`, and pinnable KeyIDs from `GET /api/providers/{id}/connections`
  (w6-e SHIPPED), and WRITES the real w5-g VK `provider_configs[].key_ids`. w6-f does
  NOT block on w6-pre. If w6-pre's `/api/catalog` later lands and is preferred, a
  serial follow-up swaps the page's source path — not a w6-f blocker.
- **ESC-2 (RESOLVED at authoring — keys/VK DTO divergence, binding)**: the e2e
  `keys.ts`/`virtual-keys.ts` mock + seed shapes diverge from the REAL Go DTOs
  (apiKey: mock `full_key`/`scopes`/`rpm_limit` vs Go `key`/`machine_id`; VK: mock
  flat `budget_usd`/`budget_period` vs Go nested `budget{limit,period,used}` +
  `provider_configs[]`). **Decision**: the pages call the REAL paths; CORRECT the
  `keys.ts`/`virtual-keys.ts` BODIES + their SEEDS to mirror the Go DTOs (mocks
  mirror reality, MAP decision 4); the pages tolerate the real shapes. NO edit to
  the existing Go DTOs. The UI `ApiKey`/`VirtualKey` types' extra fields are
  display-optional. If the corrected body cannot satisfy the spec without a Go
  change, STOP and ESCALATE — do not edit the existing Go.
- **ESC-3 (RESOLVED — model-test/availability/custom mock-only)**: `GET/POST
  /api/models/custom` (PAR-UI-117), `POST /api/models/test` (PAR-UI-119), `GET
  /api/models/availability` (PAR-UI-120) have NO Go. **Decision**: ship variant-HAVE
  against mock bodies (custom = w6-e's `models.ts`, CONSUMED; test+availability =
  NEW `nodes.ts` bodies). Serial Go follow-ups: add real `/api/models/custom` CRUD,
  `/api/models/test` (live inference probe), `/api/models/availability`. Recorded in
  `open-questions.md`.
- **ESC-4 (CONDITIONAL — fasthttp route precedence)**: registering
  `/api/provider-nodes/validate` alongside `/api/provider-nodes` should not collide
  (distinct suffix + verb), but if the `fasthttp/router` matcher disambiguates
  poorly, STOP and ESCALATE for a path rename — never silently diverge page/mock/Go.
- **ESC-5 (CONDITIONAL — shared mock/index)**: if correcting `keys.ts`/
  `virtual-keys.ts` bodies breaks a non-w6-f spec, or the `index.ts` nodes
  registration collides with a sibling plan's pending append, STOP and ESCALATE for
  orchestrator serialization — no fudge, no foundation-mock edit.
- **ESC-6 (CONDITIONAL — store schema)**: if the provider-node shape strictly
  requires a field the `providers` table lacks (prefix/api_type), use ADDITIVE
  `ensureColumn` ONLY (§1.6b); if that ripples into the providers CRUD or a frozen
  store contract, STOP and ESCALATE — do not edit existing store contracts.
- **Serial-slot dependency (§1.8)**: w6-f TAKES the routes_admin.go slot (free after
  w6-e's release to w6-j; `WORKFLOW.md:34`) and RELEASES it to w6-j on close. The
  orchestrator MUST serialize w6-f's T3 slot edit against w6-j's slot edit (only one
  unmerged holder, MAP decision 5) and confirm the slot is free at P7 before T3.
- **No other blocking dependency**: w6-a/b/c/e merged + w5-g VK CRUD in-tree (live
  tree @ bfa9436: 16 primitives, apiFetch/stores/fixture, providers_catalog model/
  connection endpoints, keys/VK CRUD, `/api/models/disabled`, `/api/combos`). w6-f
  is unblocked for page wave 2 (does NOT require w6-pre, ESC-1).
```
