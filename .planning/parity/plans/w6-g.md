# Micro-plan w6-g — Usage + logs + quota + pricing cluster (UI-only, ZERO new Go)

```
wave: 6
plan: w6-g
status: READY (rev 1 — authored against merged w6-a + w6-b, live tree @ 9feca41)
runs: page wave 1, AFTER w6-b MERGE (consumes frozen ui/src/components/ui/*) and
  AFTER w6-a MERGE (consumes apiFetch/ApiError, stores, the e2e mock fixture +
  MockEventSource, the W5 usage/pricing Go APIs). Disjoint from w6-c/w6-e/w6-h/
  w6-i (different routes/components/specs). Holds NO Go serial slot (zero new Go).
branch: main (direct commits, per repo convention)
commit-prefix: phase-1/w6-g:
ref-source: 9router frozen @ 827e5c3 —
  src/app/(dashboard)/dashboard/{page.js,usage/page.js,quota/page.js},
  src/app/dashboard/settings/pricing/page.js,
  src/app/(dashboard)/dashboard/usage/components/{ProviderTopology.js,
  RequestDetailsTab.js,ProviderLimits.js},
  src/shared/components/{UsageStats.js,RequestLogger.js,PricingModal.js}
base: <base> = git rev-parse HEAD recorded at P0 (expected 9feca41 at authoring;
  if main advanced, record the actual SHA and substitute everywhere §5 says <base>)
freeze-exception: NONE. The header.tsx / __root.tsx / main.tsx exceptions are
  SPENT (w6-b wiring, w6-c logout-slot). w6-g touches no frozen w6-a/w6-b file.
go-serial-slot: NONE. w6-g adds zero Go (consumes W5 usage/pricing handlers,
  verified in §1.2). routes_admin.go is NOT touched.
```

---

## 1. Scope — PAR rows

### Rows this plan closes

| Row | Claim | Target state after w6-g |
|---|---|---|
| PAR-UI-005 | Route `/dashboard` overview (9router aliases to `/dashboard/endpoint`) | HAVE (variant — `/dashboard` is itself the overview page, §1.5; MAP scope decision) |
| PAR-UI-011 | Route `/usage` with overview/logs/details tabs + period selector | HAVE (variant — g0router splits the 9router single page; §1.5) |
| PAR-UI-012 | Route `/quota` shows provider limits | HAVE (variant — see quota data-source §1.4 / §8 ESCALATION-1) |
| PAR-UI-025 | Route `/pricing` pricing management | HAVE |
| PAR-UI-047 | UsageStats: period selector, overview cards, provider topology, usage table, SSE updates | HAVE (variant — SSE strategy §1.3) |
| PAR-UI-048 | RequestLogger auto-refreshing table (3s poll, refresh toggle) | HAVE |
| PAR-UI-057 | PricingModal pricing config | HAVE |
| PAR-UI-081 | Data fetching layer (apiFetch ≈ TanStack Query queryFn adapter) | **ALREADY HAVE (variant) — closed by w6-a; w6-g CONSUMES, does NOT re-do. See §1.6** |
| PAR-UI-082 | Real-time SSE `EventSource` for usage stats at `/api/usage/stream` | HAVE (variant — §1.3) |
| PAR-UI-095 | API endpoint `GET /api/usage/stats?period=` (usage statistics) | HAVE (variant — g0router has the real Go endpoint; mock path divergence §1.4) |
| PAR-UI-096 | API endpoint for request logs | HAVE (variant — g0router `GET /api/usage/request-logs`; §1.4) |
| PAR-USAGE-036 | Dashboard UsageStats fetches `/api/usage/stats?period=` + subscribes `/api/usage/stream` | HAVE (§1.2/§1.3) |
| PAR-USAGE-037 | Dashboard RequestLogger polls request-logs every 3s with auto-refresh toggle | HAVE (§1.2) |

13 rows (11 PAR-UI + 2 PAR-USAGE). PAR-UI-081 is a **consume-only** entry (already
HAVE from w6-a, §1.6) — no work, recorded for traceability. The other 12 rows are
delivered by this plan. Matches WAVE-6-MAP w6-g row (~line 133) and §Ownership w6-g
(~line 177-182).

### 1.1 Preconditions already satisfied by merged waves (evidence)

- Route STUBS exist, must be REWRITTEN (not created — so no new route file, so
  `routeTree.gen.ts` does NOT change; MAP decision 6 / §1.7). All six render only
  an `<h1>`:
  `ui/src/routes/dashboard.tsx:1-9` (`<h1>Dashboard</h1>`),
  `ui/src/routes/usage.tsx:1-9` (`<h1>Usage</h1>`),
  `ui/src/routes/logs.tsx:1-9` (`<h1>Logs</h1>`),
  `ui/src/routes/traffic.tsx:1-9` (`<h1>Traffic</h1>`),
  `ui/src/routes/quota.tsx:1-9` (`<h1>Quota</h1>`),
  `ui/src/routes/pricing.tsx:1-9` (`<h1>Pricing</h1>`).
- Frozen primitives this plan CONSUMES (w6-b, never edited): `Card`/`CardHeader`/
  `CardTitle`/`CardContent` `ui/src/components/ui/card.tsx`; `Button`
  `ui/src/components/ui/button.tsx`; `Input` `ui/src/components/ui/input.tsx`;
  `Select` `ui/src/components/ui/select.tsx`; `Modal`
  `ui/src/components/ui/modal.tsx` (controlled `open`/`onClose`, traffic lights);
  `Badge` `ui/src/components/ui/badge.tsx`; `Toggle` `ui/src/components/ui/toggle.tsx`;
  `SegmentedControl` `ui/src/components/ui/segmented-control.tsx` (period/tab
  selectors); `ProviderIcon` `ui/src/components/ui/provider-icon.tsx`;
  `Loading`/`Spinner`/`Skeleton`/`CardSkeleton`
  `ui/src/components/ui/{loading,skeleton}.tsx`; `Pagination`
  `ui/src/components/ui/pagination.tsx`; `Tooltip` `ui/src/components/ui/tooltip.tsx`.
- Frozen foundation this plan CONSUMES (w6-a, never edited): `apiFetch`
  `ui/src/lib/api.ts:19` + `ApiError` `ui/src/lib/api.ts:3` (the PAR-UI-081 queryFn
  adapter, §1.6); toast via `useNotificationStore.push`
  `ui/src/stores/notification.ts`; Material Symbols `ui/src/index.css:3`.
- UI types this plan CONSUMES: `UsageLog` `ui/src/lib/types.ts:291`
  (`id,timestamp,provider,model,api_key_id,api_key_name,status,status_code,
  prompt_tokens,completion_tokens,total_tokens,cost_usd,latency_ms,rtk_enabled,
  caveman_enabled`); `Quota` `ui/src/lib/types.ts:219`
  (`connection_id,provider,connection_name,account_label?,plan,used,limit,unit,
  reset_at,is_active`); `PricingOverride` `ui/src/lib/types.ts:179`
  (`id,provider,model,input_cost,output_cost`).
- **Dependencies present (no additions allowed — VERIFIED):**
  `@xyflow/react@^12.11.0` `ui/package.json:54` (provider topology, PAR-UI-047);
  `recharts@^2.15.4` `ui/package.json:76` (charts). Both already installed —
  NO `package.json` edit, NO new dep. (If either were absent it would be an
  ESCALATION, §8; it is not.)
- **e2e mock harness present + registered (CONSUME-ONLY, §1.4 / §8):** handlers
  `ui/e2e/mocks/handlers/{usage,quota,pricing,streams,logs}.ts` registered at
  `ui/e2e/mocks/handlers/index.ts:49,52,53,55,65`; seeds
  `seed/usage.ts` (`seedUsageLogs`/`seedQuota`), `seed/pricing.ts` exported at
  `ui/e2e/mocks/seed/index.ts:10,13`. Mock store fields `usageLogs`/`quotas`/
  `pricing` `ui/e2e/mocks/store.ts:92,97,98`.
- **MockEventSource present in the e2e fixture (the SSE testing surface, §1.3):**
  `ui/e2e/mocks/fixture.ts:33-112` injects (via `page.addInitScript`) a
  `MockEventSource` class that replaces `window.EventSource`. It currently streams
  ONLY `/api/traffic/stream` (`fixture.ts:60-77`) and `/api/console-logs/stream`
  (`fixture.ts:78-97`). It does NOT yet stream `/api/usage/stream` — resolved in
  §1.3 (the usage stream is exercised via the component's graceful no-op + REST
  fallback; the fixture is NOT edited by w6-g).
- Existing acceptance specs (the contract — §1.3 thin-smoke interpretation):
  `ui/e2e/dashboard.spec.ts:1-22` (2 tests: body visible + a `[class*='grid']`
  visible), `ui/e2e/usage.spec.ts:1-18` (2 tests: `/usage` body contains "Usage";
  `/logs` body contains "Logs"), `ui/e2e/traffic.spec.ts:1-13` (1 test: body
  contains "Traffic"), `ui/e2e/quota.spec.ts:1-13` (1 test: body contains
  "Quota"), `ui/e2e/pricing.spec.ts:1-13` (1 test: body contains "Pricing").
  Login helper `ui/e2e/helpers.ts:3` drives `#username`/`#password`.

### 1.2 Real Go contract (file:line evidence — w6-g is UI-ONLY, ZERO Go changes)

ALL endpoints this cluster needs already exist from W5 (w5-d read APIs, w5-e
SSE/quota). Verified present — w6-g adds NO Go:

Routes (`internal/server/routes_admin.go`):
- `GET /api/usage/stats` → `h.GetUsageStats` (`routes_admin.go:90`)
- `GET /api/usage/chart` → `h.GetUsageChart` (`routes_admin.go:91`)
- `GET /api/usage/request-logs` → `h.GetUsageRequestLogs` (`routes_admin.go:92`)
- `GET /api/usage/logs` → `h.GetUsageRequestLogs` (`routes_admin.go:93`, alias)
- `GET /api/usage/request-details` → `h.GetRequestDetails` (`routes_admin.go:94`)
- `GET /api/usage/stream` → `(&admin.UsageStreamHandler{…}).UsageStream`
  (`routes_admin.go:95`) — **the SSE endpoint, §1.3**
- `GET /api/usage/{connectionId}` → `(&admin.ConnectionUsageHandler{…}).GetConnectionUsage`
  (`routes_admin.go:96`) — **per-connection quota/usage; the real quota source, §1.4**
- `GET /api/pricing` → `h.GetPricing` (`routes_admin.go:98`)
- `PATCH /api/pricing` → `h.PatchPricing` (`routes_admin.go:99`)
- `DELETE /api/pricing` → `h.DeletePricing` (`routes_admin.go:100`)

Body / response shapes (snake_case `{data,error}` envelope, `respond.go`):
- **GetUsageStats** (`internal/admin/usage.go:101-117`): `?period=` ∈
  `{today,24h,7d,30d,60d,all}` (`usage.go:202-213`), default `all`; bad period →
  400. Success → 200 `{data:<stats map>}` (`h.stats.Stats(period)`).
- **GetUsageChart** (`usage.go:120-136`): `?period=` ∈ `{today,24h,7d,30d,60d}`
  (no `all`), default `7d`. Success → 200 `{data:<buckets>}`.
- **GetUsageRequestLogs** (`usage.go:139-142`): → 200 `{data:<recent logs (200)>}`.
- **GetRequestDetails** (`usage.go:145-172`): paginated; → 200
  `{data:{data:<rows>, pagination:<…>}}`.
- **UsageStream** (`internal/admin/usagestream.go:33-114`): `Content-Type:
  text/event-stream`, `Cache-Control: no-cache`, `Connection: keep-alive`
  (`usagestream.go:34-36`). Emits an initial full frame `data: {json}\n\n`
  (`sendFull`, `usagestream.go:124-137,171-194`), then on tracker events emits
  quick/full frames, plus a comment keepalive `: ping\n\n` every 25s
  (`productionKeepalive`, `usagestream.go:15,158-169`). Frame payload is the stats
  map (the snapshot overlays `activeRequests`/`recentRequests`/`pending`/
  `errorProvider` onto the cached full stats — matches the 9router client's merge,
  §1.3 / ref `UsageStats.js:256-278`).
- **GetPricing** (`internal/admin/pricing.go:20-27`): → 200 `{data:<nested
  provider→model→{input,output,cached,reasoning,cache_creation}>}`
  (`pricingToSnakeCase`, `pricing.go:79-95`).
- **PatchPricing** (`pricing.go:30-53`): body
  `{provider:{model:{field:value}}}`, fields ∈
  `{input,output,cached,reasoning,cache_creation}` (`pricing.go:11-17`),
  non-negative; bad → 400. → 200 `{data:<user pricing>}`.
- **DeletePricing** (`pricing.go:56-77`): `?provider=&model=` (empty provider →
  reset all). → 200 `{data:<user pricing>}`.
- **NO `GET /api/quota` endpoint exists on the Go side** (`grep '"/api/quota"'
  internal/` → empty). The real per-connection quota is `GET
  /api/usage/{connectionId}` → `GetConnectionUsage` (`connectionusage.go:60-…`).
  This is the divergence resolved in §1.4 / §8 ESCALATION-1.

### 1.3 SSE under `vite preview` — the highest-risk surface (binding decision)

**The problem.** The e2e suite runs against `vite preview` with NO backend; every
API is intercepted. `EventSource` (server-push) cannot be intercepted by
`page.route` the way `fetch` is — `page.route` cannot keep an HTTP response open
and push frames as a browser `EventSource` expects. So a raw `new
EventSource("/api/usage/stream")` in the component would, under the harness,
either hang or fail to deliver messages.

**The mechanism that already exists (evidence).** The e2e fixture
(`ui/e2e/mocks/fixture.ts:33-112`) already solves this for OTHER streams by
injecting a `MockEventSource` class over `window.EventSource` via
`page.addInitScript`. It dispatches synthetic `message` events on a timer for
`/api/traffic/stream` (`fixture.ts:60-77`) and `/api/console-logs/stream`
(`fixture.ts:78-97`). For any OTHER url (including `/api/usage/stream`) the mock
constructs, fires `open`, and then `startStreaming()` simply does nothing
(`fixture.ts:59-98` — no branch matches), and `close()` is a clean no-op.

**The reference client degrades gracefully (evidence).** 9router's `UsageStats`
(`UsageStats.js:255-278`) attaches `es.onmessage` (merges only the real-time
fields onto REST-fetched stats) and `es.onerror = () => setLoading(false)` — the
stream is PURELY additive. The page's data comes from the REST
`GET /api/usage/stats?period=` fetch (`UsageStats.js:242`); the SSE only overlays
live `activeRequests`/`recentRequests`/`pending`. If the stream never delivers a
message (the harness case), the page still renders fully from REST.

**Decision (binding) — three points, no fixture edit, no escalation needed:**
1. **`UsageStats` (w6-g) ports the additive-SSE pattern exactly**: render all
   overview cards / topology / table from the REST `apiFetch("/api/usage/stats?
   period=…")` result; open `new EventSource("/api/usage/stream")` ONLY to overlay
   live fields; on `onmessage` merge `activeRequests`/`recentRequests`/`pending`/
   `errorProvider`; on `onerror` call a no-op / clear any loading flag. The page
   is fully functional with ZERO stream messages. This is the parity behavior and
   the harness-safe behavior in one.
2. **Under the harness, `MockEventSource` opens cleanly and never errors** for
   `/api/usage/stream` (it just never pushes a message — `startStreaming` no-ops).
   So the e2e asserts the **REST-driven** content (cards, table, topology, period
   switch) — NOT a streamed update. The SSE-presence assertion is a UNIT test
   (§3) that stubs `EventSource` and verifies the component (a) constructs
   `EventSource("/api/usage/stream")`, (b) merges an injected `message` payload's
   `activeRequests` into rendered state, (c) survives an `error` event without
   throwing. This proves PAR-UI-082 deterministically without a live stream.
3. **OPTIONAL e2e live-overlay proof (only if trivially green)**: the e2e MAY
   additionally extend `MockEventSource` behavior at the TEST level by
   constructing a real `EventSource` is unnecessary — the unit test (point 2)
   is the authoritative SSE proof. The e2e stays REST-deterministic. **w6-g does
   NOT edit `ui/e2e/mocks/fixture.ts`** (foundation file, not in w6-g ownership,
   §3 FORBIDDEN); the existing no-op branch is sufficient.

**Escalation trigger (only if the decision fails):** if, at T-impl, a real
`EventSource("/api/usage/stream")` under `vite preview` + `MockEventSource`
throws or blocks render (it must not — the mock fires `open` then idles), STOP and
ESCALATE (§8 ESCALATION-2) for a sanctioned one-line `fixture.ts` addition of a
`/api/usage/stream` no-message branch. Expected outcome: NOT needed (the default
no-op branch already handles unknown urls). Record the observed behavior at P8.

### 1.4 Quota + path divergences — mock vs real Go (binding interpretation)

The e2e mocks model paths that diverge from the real W5 Go (§1.2). Resolve at T1,
keeping page + mock + Go consistent and "mocks mirror reality":

| Surface | e2e mock path | Real Go path (§1.2) | Resolution |
|---|---|---|---|
| Usage stats | `/api/usage/summary` (`handlers/usage.ts:6`) | `/api/usage/stats?period=` (`routes_admin.go:90`) | **Page calls the REAL `/api/usage/stats`**; correct mock BODY to serve `/api/usage/stats` (mirrors reality). §8 ESCALATION-1a. |
| Usage chart | `/api/usage/chart` (`handlers/usage.ts:17`) | `/api/usage/chart` (`routes_admin.go:91`) | Paths AGREE. Consume as-is (correct body shape if needed). |
| Request logs | `/api/logs` (`handlers/logs.ts:6`) + `/api/usage` (`handlers/usage.ts:30`) | `/api/usage/request-logs` (`routes_admin.go:92`) | **Page calls `/api/usage/request-logs`**; correct mock BODY to serve it. §8 ESCALATION-1b. |
| Usage SSE | `/api/traffic/stream` (`handlers/streams.ts:5` + `fixture.ts:60`) | `/api/usage/stream` (`routes_admin.go:95`) | **Page calls the REAL `/api/usage/stream`** (PAR-UI-082); under harness it idles (§1.3). Traffic page MAY consume `/api/traffic/stream` for its live feed (that IS the mock+fixture surface). §1.3. |
| Quota | `/api/quota` (`handlers/quota.ts:6`) | NO `/api/quota`; real is per-connection `GET /api/usage/{connectionId}` (`routes_admin.go:96`) | **§8 ESCALATION-1c (RESOLVED):** quota page consumes the existing `/api/quota` MOCK path AND the real Go must answer it — but there is NO Go `/api/quota` and w6-g adds ZERO Go. Therefore quota is a **variant**: the page renders provider limits from the data the mock supplies on `/api/quota` (the `Quota[]` seed, `seed/usage.ts:31-38`), and at runtime the real source is the per-connection `/api/usage/{connectionId}` aggregation. Since closing the runtime gap requires NEW Go (forbidden here), record PAR-UI-012 as variant-HAVE against the mock contract and raise the Go aggregation as a serial follow-up (§8). The mock `/api/quota` body is CONSUMED unchanged. |
| Pricing | `/api/pricing` GET/POST + `/api/pricing/{id}` PUT/DELETE (`handlers/pricing.ts`) | `/api/pricing` GET/**PATCH**/DELETE (`routes_admin.go:98-100`) | **Page calls real `/api/pricing` GET + PATCH + DELETE** (matches ref `PricingModal.js:20,54-55,79`). The mock body currently models a REST collection (POST/PUT-by-id) that the real Go does NOT have. Correct mock BODY to serve GET + PATCH (nested provider→model→fields) + DELETE(`?provider=&model=`), mirroring `pricing.go`. §8 ESCALATION-1d. |

**Binding rule (MAP decision 4):** where mock and real Go disagree, the **real Go
wins** and the mock body is corrected in this plan (mocks mirror reality). w6-g
corrects ONLY handler BODIES (`usage.ts`, `logs.ts`, `pricing.ts`, and CONSUMES
`quota.ts`/`streams.ts` unchanged) — NEVER the `handlers/index.ts` registration,
NEVER the seed export list, NEVER `fixture.ts`. If correcting a body to match Go
would break another page's spec that consumes the same handler, STOP and ESCALATE
(§8 ESCALATION-3) — w6-g owns `usage/quota/pricing/logs/streams` handler bodies
for THIS cluster's specs only.

### 1.5 Variant notes (recorded HAVE rationale)

- **PAR-UI-005 `/dashboard` overview**: 9router aliases `/dashboard` →
  `/dashboard/endpoint` (its overview renders the endpoint client,
  ref `dashboard/page.js`). g0router's MAP scope decision records `/dashboard` as
  itself the overview page (the redirect `/`→`/dashboard` is w6-a's). w6-g's
  `/dashboard` renders a metrics overview (UsageStats summary cards) — the
  `dashboard.spec.ts` `[class*='grid']` assertion holds. Recorded variant-HAVE.
- **PAR-UI-011 usage tabs**: 9router has ONE `/dashboard/usage` page with
  overview/logs/details tabs (`usage/page.js:30-72`). g0router ALSO splits some
  surfaces into `/logs` and `/traffic` (matrix note line 22). w6-g implements
  `/usage` with the overview/logs/details tab structure (`SegmentedControl` +
  period `SegmentedControl`, ports `usage/page.js:44-72`) AND the standalone
  `/logs` (RequestLogger) and `/traffic` (live SSE feed) routes. Recorded
  variant-HAVE.
- **PAR-UI-012 quota**: variant per §1.4 — provider-limits view rendered from the
  `/api/quota` mock contract; the runtime per-connection aggregation Go is a
  serial follow-up (no new Go in w6-g). 9router's quota page renders
  `ProviderLimits` (ref `quota/page.js`).
- **PAR-UI-047 SSE**: variant per §1.3 — additive `EventSource("/api/usage/stream")`
  overlay; page fully functional from REST; SSE proven by unit test.
- **Pages render inside app chrome**: `__root.tsx` wraps every route in
  Sidebar+Header; pages render in `<Outlet>`; specs assert page content with chrome
  present (w6-c/w6-e precedent). Accepted constraint, not a gap.

### 1.6 PAR-UI-081 is ALREADY HAVE — w6-g consumes it (RESOLVED, with evidence)

The prompt asked to resolve whether PAR-UI-081 (data layer) needs work here. It
does NOT. Evidence:

- **Matrix current state**: `.planning/parity/matrix/9router-ui.md:92` —
  `PAR-UI-081 | … | ui/src/lib/api.ts | HAVE (variant) | apiFetch unwraps Go
  {data,error} envelope; serves as TanStack Query queryFn adapter`. The row is
  already flipped HAVE.
- **w6-a closure record**: `docs/WORKFLOW.md:6629` —
  "PAR-UI-081 MISSING→HAVE (variant: apiFetch = TanStack Query queryFn adapter)";
  and `WORKFLOW.md:6645` lists "PAR-UI-081 HAVE (variant)" among w6-a's MERGED
  flips.
- **Tree state**: `apiFetch` exists at `ui/src/lib/api.ts:19`; it is already the
  fetch+envelope-unwrap layer every page uses (`auth.ts`, `i18n.tsx` consume it).
  `QueryClientProvider` is NOT mounted (`grep QueryClient ui/src/routes/__root.tsx
  ui/src/main.tsx` → 0 matches) and `__root.tsx`/`main.tsx` are FROZEN (w6-a).
- **w6-e open-questions note** (`open-questions.md:6`) flagged that TanStack Query
  is installed-but-unwired and recorded PAR-UI-081 as "not assumed satisfied by
  w6-e" — i.e. it was satisfied EARLIER, by w6-a, not by w6-e.

**Resolution (binding):** PAR-UI-081 was closed by w6-a as a variant (apiFetch is
the data-fetch layer / TanStack Query `queryFn` adapter). w6-g **consumes
`apiFetch`** for every REST read and **does NOT** mount a `QueryClientProvider`,
**does NOT** edit `__root.tsx`/`main.tsx`, and **does NOT** re-flip the row. There
is NO sanctioned frozen-file exception for w6-g (the header exception is SPENT,
§freeze-exception). Any need to wire `QueryClientProvider` is an orchestrator
serial follow-up (MAP decision 2), NOT in-plan. PAR-UI-081 is listed in §1 scope as
consume-only for traceability and is asserted by the existing
`api.test.ts` (already green) — w6-g adds nothing to it.

### 1.7 `routeTree.gen.ts` is NOT touched

All six routes already exist as stubs (§1.1); rewriting their component bodies does
not change the route tree, and no new route file is added (no nested `/usage/$tab`
routes — tabs are in-page state, §1.5). Therefore `ui/src/routeTree.gen.ts` is
UNCHANGED by w6-g (MAP decision 6; w6-g is not the wave-1 new-route plan — w6-i
is). If a build incidentally reformats it, that is an ESCALATION (§8), not an
in-plan edit.

### NOT in scope (explicit)

- **No Go changes.** ALL of `internal/` is FORBIDDEN; the W5 usage/pricing/SSE
  handlers already exist (§1.2 evidence). If a real handler's shape contradicts a
  corrected mock body, that is an ESCALATION (§8), never an in-plan Go edit or mock
  fudge. Specifically NO new `/api/quota` Go endpoint (§1.4 quota is variant).
- **No new route FILES** — only the six existing stubs are rewritten;
  `routeTree.gen.ts` untouched (§1.7). No nested tab routes (in-page state).
- **No dependency additions** — `@xyflow/react` + `recharts` already installed
  (§1.1); every import resolves to installed packages or w6-a/w6-b outputs. NO
  `package.json` / lockfile edit.
- **No edits to any frozen w6-a/w6-b file** — no `__root.tsx`, `main.tsx`, layout
  components, `ui/src/components/ui/*`, `ui/src/stores/*`, `ui/src/lib/api.ts`,
  `ui/src/lib/utils.ts`, `ui/src/providers/*`, `ui/src/lib/auth.ts`,
  `ui/src/routes/{login,callback}.tsx`. No header exception remains (SPENT).
- **No `QueryClientProvider` mount** (§1.6) — plain `apiFetch`; PAR-UI-081 already
  HAVE.
- **No edits to `ui/e2e/mocks/fixture.ts`** (foundation/MockEventSource, §1.3) nor
  to `mocks/handlers/index.ts` / `mocks/seed/index.ts` (already wired) — w6-g
  corrects `usage.ts`/`logs.ts`/`pricing.ts` handler BODIES only; consumes
  `quota.ts`/`streams.ts` unchanged.
- **No other e2e specs** beyond the five
  `{dashboard,usage,traffic,quota,pricing}.spec.ts` (+ the matching corrected mock
  bodies). NOT `console.spec.ts` (w6-i owns `/api/console-logs/stream`).
- **No console/chat/translator** (w6-i), no providers/connections/models (w6-e),
  no virtual-keys/endpoint (w6-f), no combos/routing (w6-h), no settings (w6-j).
- **No real outbound network** — all reads are mock-intercepted; SSE idles (§1.3).

---

## 2. Precondition checks

Run all before any edit; abort and report to orchestrator on any failure.

```bash
# P0 — clean tree; record <base>
git status --porcelain     # must be empty (untracked tooling artifacts must be
                           # gitignored; worker uses explicit `git add <file>`,
                           # never `git add -A`, per w6-b runtime disposition)
git rev-parse HEAD         # record as <base> for §5 (expected 9feca41)

# P1 — w6-b primitives present and frozen (consumed)
ls ui/src/components/ui/*.tsx | grep -v test | wc -l    # = 16 (w6-b set intact)
grep -n "export { Card" ui/src/components/ui/card.tsx
grep -n "SegmentedControl" ui/src/components/ui/segmented-control.tsx
grep -n "export function Modal\|export interface ModalProps" ui/src/components/ui/modal.tsx
grep -n "CardSkeleton\|Skeleton" ui/src/components/ui/skeleton.tsx

# P2 — w6-a foundation present and frozen (consumed; PAR-UI-081 already HAVE §1.6)
grep -n "export async function apiFetch" ui/src/lib/api.ts
grep -n "export class ApiError" ui/src/lib/api.ts
grep -n "push:" ui/src/stores/notification.ts
grep -rn "QueryClientProvider\|QueryClient" ui/src/routes/__root.tsx ui/src/main.tsx ; echo "^ expect EMPTY (not mounted; do NOT mount — §1.6)"

# P3 — dependencies present (NO additions allowed; absence ⇒ ESCALATION §8)
grep -n '"@xyflow/react"' ui/package.json    # ^12.x present
grep -n '"recharts"'      ui/package.json    # ^2.x present

# P4 — the six route stubs are still bare (safe to rewrite); no new dirs yet
grep -n "<h1>Dashboard</h1>" ui/src/routes/dashboard.tsx
grep -n "<h1>Usage</h1>"     ui/src/routes/usage.tsx
grep -n "<h1>Logs</h1>"      ui/src/routes/logs.tsx
grep -n "<h1>Traffic</h1>"   ui/src/routes/traffic.tsx
grep -n "<h1>Quota</h1>"     ui/src/routes/quota.tsx
grep -n "<h1>Pricing</h1>"   ui/src/routes/pricing.tsx
test ! -d ui/src/components/usage && echo "usage components dir absent (good)"

# P5 — e2e mock harness present + registered (CONSUME; correct bodies only §1.4)
grep -n "registerUsageHandlers\|registerQuotaHandlers\|registerPricingHandlers\|registerLogsHandlers\|registerStreamsHandlers" ui/e2e/mocks/handlers/index.ts
grep -n "seedUsageLogs\|seedQuota\|seedPricing" ui/e2e/mocks/seed/index.ts
grep -n "MockEventSource\|/api/traffic/stream\|/api/usage/stream" ui/e2e/mocks/fixture.ts ; echo "^ MockEventSource handles traffic+console only; /api/usage/stream idles (§1.3)"

# P6 — the real W5 Go endpoints exist (ZERO new Go; §1.2)
grep -n "/api/usage/stats\|/api/usage/chart\|/api/usage/request-logs\|/api/usage/stream\|/api/pricing" internal/server/routes_admin.go
grep -n "func (h \*Handlers) GetUsageStats\|func (h \*Handlers) GetPricing" internal/admin/usage.go internal/admin/pricing.go
grep -rn '"/api/quota"' internal/ ; echo "^ expect EMPTY (no Go /api/quota — quota is variant §1.4)"

# P7 — e2e + unit harness green at base
cd ui && npx playwright test e2e/dashboard.spec.ts e2e/usage.spec.ts e2e/traffic.spec.ts e2e/quota.spec.ts e2e/pricing.spec.ts
# Record base result: stubs render only <h1>. The text-contains assertions
# (Usage/Logs/Traffic/Quota/Pricing) PASS at base (the route names appear in the
# sidebar chrome / the <h1>); dashboard `[class*='grid']` may PASS (chrome grid)
# or FAIL — record exact pass/fail per spec in WORKFLOW.md at P8/closeout.
cd ui && npm run build                               # exit 0
cd ui && npx vitest run src/                         # exit 0 (existing units green)
go test ./... && go vet ./...                        # exit 0 (Go untouched-green)
```

**P8 note (SSE harness probe):** before/while implementing UsageStats, manually
verify a `new EventSource("/api/usage/stream")` under `vite preview` +
`MockEventSource` fires `open` and idles WITHOUT throwing (§1.3 expectation). If it
throws/blocks, escalate (§8 ESCALATION-2). Record the observed behavior.

---

## 3. Exclusive file ownership

After w6-g merges, all CREATE files below are owned by w6-g; later plans consume,
never edit (MAP decision 7).

**CREATE — routes (REWRITE existing stubs; no new route files, §1.7):**

| File | Exports / contract |
|---|---|
| `ui/src/routes/dashboard.tsx` (REWRITE) | `Route=createFileRoute("/dashboard")`; `DashboardPage`: overview — renders `<UsageStats period="today">` summary (overview cards in a `[class*='grid']` layout, satisfying `dashboard.spec.ts`) + a compact `<RequestLogger>` preview. Reads `apiFetch("/api/usage/stats?period=today")`. |
| `ui/src/routes/usage.tsx` (REWRITE) | `Route=createFileRoute("/usage")`; `UsagePage`: `SegmentedControl` tabs `overview`/`logs`/`details` (tab in component state, §1.5) + period `SegmentedControl` (`today/24h/7d/30d/60d`) shown on overview; overview→`<UsageStats>`, logs→`<RequestLogger>`, details→`<RequestDetailsTab>`. Body contains "Usage". Ports `usage/page.js:44-72`. |
| `ui/src/routes/logs.tsx` (REWRITE) | `Route=createFileRoute("/logs")`; `LogsPage`: renders `<RequestLogger>` standalone (3s poll, refresh toggle). Body contains "Logs". |
| `ui/src/routes/traffic.tsx` (REWRITE) | `Route=createFileRoute("/traffic")`; `TrafficPage`: live traffic feed via `EventSource("/api/traffic/stream")` (the mock+fixture surface, §1.4) — table/list of streamed events. Body contains "Traffic". |
| `ui/src/routes/quota.tsx` (REWRITE) | `Route=createFileRoute("/quota")`; `QuotaPage`: provider-limits view via `apiFetch("/api/quota")` (variant, §1.4) → `<ProviderLimits>` cards (used/limit progress, plan, reset_at). Body contains "Quota". |
| `ui/src/routes/pricing.tsx` (REWRITE) | `Route=createFileRoute("/pricing")`; `PricingPage`: pricing management — table of provider→model rates from `apiFetch("/api/pricing")`; edit via `<PricingModal>` (PATCH); reset via DELETE. Body contains "Pricing". |

**CREATE — domain components (`ui/src/components/usage/`):**

| File | Exports / contract |
|---|---|
| `usage-stats.tsx` | `UsageStats` (PAR-UI-047/082, PAR-USAGE-036) — props `period`, `setPeriod?`, `hidePeriodSelector?`. REST: `apiFetch("/api/usage/stats?period=…")` on mount + period change → overview cards (requests, tokens, cost, latency) in a `grid`. SSE (§1.3): `new EventSource("/api/usage/stream")`, `onmessage` merges `activeRequests`/`recentRequests`/`pending`/`errorProvider` onto stats, `onerror` clears loading (no throw); `close()` on unmount. Renders `<ProviderTopology>` + usage table. Ports `UsageStats.js:192-505` adapted to React 19 + apiFetch + frozen primitives. |
| `request-logger.tsx` | `RequestLogger` (PAR-UI-048, PAR-USAGE-037) — `apiFetch("/api/usage/request-logs")` on mount; `setInterval(fetch, 3000)` when auto-refresh `Toggle` on; table of recent logs (timestamp, provider via `ProviderIcon`, model, status `Badge`, tokens, cost, latency). Refresh toggle + manual refresh `Button`. Ports `RequestLogger.js:12-34`. |
| `request-details-tab.tsx` | `RequestDetailsTab` — paginated details via `apiFetch("/api/usage/request-details?page=&pageSize=")` (`{data:{data,pagination}}`, §1.2) + `<Pagination>`; filter inputs (provider/model/status). Ports `RequestDetailsTab.js`. |
| `provider-topology.tsx` | `ProviderTopology` (PAR-UI-047 topology) — `@xyflow/react` `ReactFlow` graph of provider→model nodes from stats; imports `@xyflow/react/dist/style.css`. Ports `ProviderTopology.js:5-308`. |
| `provider-limits.tsx` | `ProviderLimits` (PAR-UI-012) — quota cards from `Quota[]`: used/limit progress bar, plan `Badge`, unit, reset_at; `unlimited` when `limit===0`. |
| `usage-charts.tsx` | `UsageCharts` — `recharts` line/area charts from `apiFetch("/api/usage/chart?period=…")` (`{buckets,tokens_input,tokens_output,costs,requests}`, §1.2). |
| `pricing-modal.tsx` | `PricingModal` (PAR-UI-057) — consumes `Modal`+`Input`; fields `input,output,cached,reasoning,cache_creation` (`pricing.go:11-17`); save → `apiFetch("/api/pricing",{method:"PATCH",body:{provider:{model:{field:val}}}})`; reset → DELETE `?provider=&model=`. Ports `PricingModal.js:20-94`. |

**CREATE — unit tests (vitest — logic reachable without a live backend/stream):**

| File | Contents |
|---|---|
| `ui/src/components/usage/usage-stats.test.tsx` | ≥4 tests (the authoritative SSE proof, §1.3): stub `EventSource` + `fetch`/`apiFetch`; (1) renders overview cards from a REST stats payload; (2) constructs `EventSource("/api/usage/stream")` on mount; (3) an injected `message` event with `{activeRequests:N,recentRequests:[…]}` merges into rendered state; (4) an `error` event does NOT throw and clears loading; (5) `close()` called on unmount. Committed RED before `usage-stats.tsx`. |
| `ui/src/components/usage/request-logger.test.tsx` | ≥3 tests: renders rows from a REST logs payload; toggling auto-refresh starts/stops the 3s interval (fake timers); manual refresh re-fetches. Committed RED before `request-logger.tsx`. |
| `ui/src/components/usage/provider-limits.test.tsx` | ≥2 tests via `renderToString`: renders used/limit + plan badge; `limit===0` renders "unlimited". Committed RED before `provider-limits.tsx`. |

(Topology/charts are `@xyflow/react`/`recharts`-DOM-heavy; their coverage is the
e2e `[class*='grid']`/visibility assertions, not a unit — same disposition as
w6-e's modal components.)

**MODIFY — e2e specs (the acceptance contract) + mock body corrections (§1.4):**

| File | Change |
|---|---|
| `ui/e2e/dashboard.spec.ts` | KEEP the 2 existing tests. ADD RED: overview cards render (a metric value visible); a `[class*='grid']` of cards present. |
| `ui/e2e/usage.spec.ts` | KEEP the 2 existing (`/usage`→"Usage", `/logs`→"Logs"). ADD RED: usage tabs (`overview/logs/details`) switchable; period selector switches and re-fetches; overview shows cards; logs tab shows a request-log table. |
| `ui/e2e/traffic.spec.ts` | KEEP the 1 existing. ADD RED: a live traffic row appears (driven by `MockEventSource` `/api/traffic/stream`, `fixture.ts:60-77`). |
| `ui/e2e/quota.spec.ts` | KEEP the 1 existing. ADD RED: ≥1 provider-limit card with a used/limit indicator (from `/api/quota` seed). |
| `ui/e2e/pricing.spec.ts` | KEEP the 1 existing. ADD RED: pricing table rows render; opening `PricingModal` shows the rate fields; saving fires a PATCH. |
| `ui/e2e/mocks/handlers/usage.ts` | CORRECT BODY (§1.4): serve `/api/usage/stats?period=` (was `/api/usage/summary`) and keep `/api/usage/chart`; the page reads stats/chart from the real paths. Mirror the real `{data:<stats>}` shape. NO index/seed edit. |
| `ui/e2e/mocks/handlers/logs.ts` | CORRECT BODY (§1.4): serve `/api/usage/request-logs` (was/in addition to `/api/logs`) returning recent `UsageLog[]`. NO index/seed edit. |
| `ui/e2e/mocks/handlers/pricing.ts` | CORRECT BODY (§1.4): serve `/api/pricing` GET (nested provider→model→fields) + PATCH + DELETE(`?provider=&model=`), matching `pricing.go`; drop the REST POST/PUT-by-id model. NO index/seed edit. |
| `ui/e2e/mocks/handlers/quota.ts` | CONSUME UNCHANGED (`/api/quota` → `store.quotas`, §1.4 variant). |
| `ui/e2e/mocks/handlers/streams.ts` | CONSUME UNCHANGED (`/api/traffic/stream` for the traffic page). |

**FORBIDDEN:** everything else. Explicitly: ALL of `internal/` (W5 Go exists,
§1.2; quota is a variant with NO new Go); `ui/src/components/ui/*` (w6-b frozen);
`ui/src/stores/*`, `ui/src/lib/api.ts`, `ui/src/lib/utils.ts`,
`ui/src/lib/auth.ts`, `ui/src/providers/*` (w6-a frozen);
`ui/src/routes/__root.tsx`, `ui/src/main.tsx`, `ui/src/components/layout/*`,
`ui/src/routes/{login,callback}.tsx`, `ui/src/components/auth/*`;
`ui/package.json` + lockfile; `ui/vite.config.ts`; `ui/playwright.config.ts`;
`ui/components.json`; `ui/src/index.css`; `ui/src/routeTree.gen.ts` (generated;
UNCHANGED §1.7); **`ui/e2e/mocks/fixture.ts`** (foundation/MockEventSource, §1.3);
`ui/e2e/mocks/handlers/index.ts` and `mocks/seed/index.ts` (already wired);
`ui/e2e/mocks/store.ts`; all other `ui/e2e/*.spec.ts`; all other
`ui/e2e/mocks/handlers/*` except the five named bodies above.

---

## 4. TDD tasks

Cadence (strict): **no route/component file may exist (or be rewritten beyond its
stub) before the failing test that covers it is committed.** `cd ui && npm run
build` green at EVERY commit (test files + red specs are never imported by
production code — w6-b/w6-c/w6-e rationale). `go test ./... && go vet ./...` stays
untouched-green at EVERY commit (zero new Go). The five e2e specs stay RED from T1
until the implementation tasks green them; that is the arc.

### T1 — STEP(a): extend the five e2e specs + correct mock bodies (commit RED)

Add the RED tests in §3 to `dashboard/usage/traffic/quota/pricing.spec.ts`
(names are the acceptance contract, §5). Correct the `usage.ts`/`logs.ts`/
`pricing.ts` mock BODIES to the real Go paths/shapes (§1.4); consume
`quota.ts`/`streams.ts` unchanged.

STEP(b): run all five specs — **record failure output** (no cards, no tabs, no
limits, no pricing table/modal). Commit RED:
`phase-1/w6-g: failing usage/dashboard/traffic/quota/pricing e2e + mock-path corrections (TDD red)`.

**Mock-vs-reality gate**: while correcting each mock body, re-read the Go handlers
(§1.2 file:lines). If a real shape contradicts a corrected body, OR correcting a
shared handler body breaks a non-w6-g spec, STOP and ESCALATE (§8) — no Go edit,
no mock fudge, no `index.ts`/`seed`/`fixture.ts` edit.

### T2 — STEP(a): unit tests for usage-stats / request-logger / provider-limits (commit RED)

Write the unit tests per §3 (the SSE proof lives here — stub `EventSource`,
§1.3 point 2). Stub `EventSource`/`fetch`/timers in-test (w6-a `theme.test.ts`
precedent — no jsdom needed beyond the existing vitest config). Run
`cd ui && npx vitest run src/components/usage/` → FAILS (modules missing). Record
failure. Commit RED:
`phase-1/w6-g: failing unit tests for usage-stats (SSE) + request-logger + provider-limits (TDD red)`.

### T3 — STEP(b): UsageStats + RequestLogger + charts/topology + dashboard + usage

Implement `usage-stats.tsx` (greens its units incl. the SSE assertions),
`request-logger.tsx`, `request-details-tab.tsx`, `provider-topology.tsx`,
`usage-charts.tsx`; rewrite `dashboard.tsx` + `usage.tsx`. Gates:
`npx vitest run src/components/usage/` green; `dashboard.spec.ts` green;
`usage.spec.ts` green; `logs.spec.ts` content covered by usage logs tab + the
standalone `/logs` route (rewritten here too if not in T4 — keep `/logs` in this
task). `npm run build` green. Commit:
`phase-1/w6-g: usage stats (REST + additive SSE), request logger, charts/topology, dashboard + usage pages`.

### T4 — STEP(b): traffic + quota + pricing pages + PricingModal + ProviderLimits

Implement `provider-limits.tsx` (greens its unit), `pricing-modal.tsx`; rewrite
`traffic.tsx` (live `/api/traffic/stream` feed), `quota.tsx` (provider limits),
`pricing.tsx` (table + modal). Gates: `traffic.spec.ts`, `quota.spec.ts`,
`pricing.spec.ts` green; all five specs green; `npx vitest run src/` green;
`npm run build` green. Commit:
`phase-1/w6-g: traffic + quota + pricing pages, PricingModal, ProviderLimits`.

### T5 — full gates + closeout

```bash
cd ui && npm run build
cd ui && npx playwright test e2e/dashboard.spec.ts e2e/usage.spec.ts e2e/traffic.spec.ts e2e/quota.spec.ts e2e/pricing.spec.ts   # all green
cd ui && npx playwright test                             # full suite: no spec green-at-base may be red
cd ui && npx vitest run src/                             # all green incl new usage units
go test ./... && go vet ./...                            # untouched-green (zero new Go)
```
Flip §1 matrix rows in `.planning/parity/matrix/9router-ui.md`: PAR-UI-005 → HAVE
(variant); PAR-UI-011 → HAVE (variant); PAR-UI-012 → HAVE (variant, cite §1.4);
PAR-UI-025 → HAVE; PAR-UI-047 → HAVE (variant, cite §1.3); PAR-UI-048 → HAVE;
PAR-UI-057 → HAVE; PAR-UI-082 → HAVE (variant, cite §1.3); PAR-UI-095/096 → HAVE
(variant, cite §1.4). In `.planning/parity/matrix/9router-usage.md`:
PAR-USAGE-036/037 → HAVE. **Do NOT touch PAR-UI-081** (already HAVE from w6-a,
§1.6). Update `docs/WORKFLOW.md` (record P7 base spec observations, the P8 SSE
harness probe result, the §1.4 path resolutions, and the quota Go-follow-up note).
Final commit: `phase-1/w6-g: close — usage/logs/quota/pricing cluster; matrix flips`.

---

## 5. Binary acceptance criteria

All must hold; each is yes/no. `<base>` = the commit recorded at P0 (9feca41 at
authoring). Diff gate is **w6-g commit-range-scoped** (§7) because page wave 1
plans commit to main concurrently.

**Test gates**
- `cd ui && npx playwright test e2e/dashboard.spec.ts` → exit 0, all pass, 0 skipped.
- `cd ui && npx playwright test e2e/usage.spec.ts` → exit 0, all pass.
- `cd ui && npx playwright test e2e/traffic.spec.ts` → exit 0, all pass.
- `cd ui && npx playwright test e2e/quota.spec.ts` → exit 0, all pass.
- `cd ui && npx playwright test e2e/pricing.spec.ts` → exit 0, all pass.
- `cd ui && npx vitest run src/components/usage/` → exit 0, ≥9 passed (4+3+2).
- `cd ui && npx vitest run src/` → exit 0 (all prior unit suites still green).
- `cd ui && npm run build` → exit 0.
- `go test ./... && go vet ./...` → exit 0 (Go untouched — ZERO new Go).

**TDD-order proof** — each impl file's covering test appears in an
earlier-or-equal commit:
```bash
ct=$(git log --format=%ct --diff-filter=A -1 -- ui/src/components/usage/usage-stats.test.tsx)
cf=$(git log --format=%ct --diff-filter=A -1 -- ui/src/components/usage/usage-stats.tsx)
[ "$ct" -le "$cf" ] || echo "TDD VIOLATION: usage-stats.tsx"            # prints nothing
ct=$(git log --format=%ct --diff-filter=A -1 -- ui/src/components/usage/request-logger.test.tsx)
cf=$(git log --format=%ct --diff-filter=A -1 -- ui/src/components/usage/request-logger.tsx)
[ "$ct" -le "$cf" ] || echo "TDD VIOLATION: request-logger.tsx"         # nothing
ct=$(git log --format=%ct --diff-filter=A -1 -- ui/src/components/usage/provider-limits.test.tsx)
cf=$(git log --format=%ct --diff-filter=A -1 -- ui/src/components/usage/provider-limits.tsx)
[ "$ct" -le "$cf" ] || echo "TDD VIOLATION: provider-limits.tsx"        # nothing
# e2e RED-extension commit precedes the page rewrites
sa=$(git log --format=%ct -1 --grep="failing usage/dashboard/traffic/quota/pricing e2e")
ui=$(git log --format=%ct --diff-filter=M -1 -- ui/src/routes/usage.tsx)
[ "$sa" -le "$ui" ] || echo "TDD VIOLATION: usage.tsx before red spec"  # nothing
```

**Grep proofs**
```bash
grep -rn '/api/usage/stats' ui/src/components/usage/usage-stats.tsx        # PAR-UI-095/USAGE-036
grep -rn 'EventSource(.*\/api\/usage\/stream' ui/src/components/usage/usage-stats.tsx  # PAR-UI-082/USAGE-036 SSE
grep -rn 'activeRequests\|recentRequests\|onerror' ui/src/components/usage/usage-stats.tsx  # PAR-UI-082 additive merge + graceful degrade
grep -rn '@xyflow/react' ui/src/components/usage/provider-topology.tsx     # PAR-UI-047 topology
grep -rn 'recharts' ui/src/components/usage/usage-charts.tsx               # charts
grep -rn '/api/usage/request-logs' ui/src/components/usage/request-logger.tsx  # PAR-UI-096/USAGE-037
grep -rn '3000\|setInterval' ui/src/components/usage/request-logger.tsx    # PAR-UI-048/USAGE-037 3s poll
grep -rn '/api/quota' ui/src/routes/quota.tsx ui/src/components/usage/provider-limits.tsx  # PAR-UI-012
grep -rn '/api/pricing' ui/src/routes/pricing.tsx ui/src/components/usage/pricing-modal.tsx  # PAR-UI-025/057
grep -rn 'PATCH' ui/src/components/usage/pricing-modal.tsx                  # PAR-UI-057 real Go verb
grep -rn 'input\|output\|cached\|reasoning\|cache_creation' ui/src/components/usage/pricing-modal.tsx  # PAR-UI-057 fields
grep -rn 'overview\|logs\|details' ui/src/routes/usage.tsx                  # PAR-UI-011 tabs
grep -rn '/api/traffic/stream' ui/src/routes/traffic.tsx                    # traffic live feed
test -f ui/src/routes/dashboard.tsx && echo OK                             # PAR-UI-005
# PAR-UI-081 NOT re-done: no QueryClientProvider introduced anywhere by w6-g
! grep -rn 'QueryClientProvider' ui/src/ && echo "no QueryClientProvider added OK"  # §1.6
```

**Negative / freeze proofs (w6-g commit-range — see §7)**
```bash
R="<first-w6-g>^..<last-w6-g>"
git diff $R --name-only -- internal/ | wc -l                            # = 0 (ZERO new Go)
git diff $R --name-only -- ui/package.json ui/package-lock.json ui/vite.config.ts ui/playwright.config.ts ui/components.json ui/src/index.css | wc -l   # = 0 (no deps/config)
git diff $R --name-only -- ui/src/components/ui/ | wc -l                # = 0 (w6-b frozen)
git diff $R --name-only -- ui/src/stores/ ui/src/providers/ ui/src/lib/api.ts ui/src/lib/utils.ts ui/src/lib/auth.ts | wc -l   # = 0 (w6-a frozen)
git diff $R --name-only -- ui/src/routes/__root.tsx ui/src/main.tsx ui/src/components/layout/ ui/src/routes/login.tsx ui/src/routes/callback.tsx ui/src/components/auth/ | wc -l   # = 0
git diff $R --name-only -- ui/src/routeTree.gen.ts | wc -l             # = 0 (§1.7 unchanged)
git diff $R --name-only -- ui/e2e/mocks/fixture.ts ui/e2e/mocks/handlers/index.ts ui/e2e/mocks/seed/index.ts ui/e2e/mocks/store.ts | wc -l   # = 0 (foundation/wiring untouched)
git diff $R --name-only -- 'ui/src/routes/' | grep -vE 'dashboard\.tsx|usage\.tsx|logs\.tsx|traffic\.tsx|quota\.tsx|pricing\.tsx' | wc -l   # = 0 (only the six stubs rewritten)
git diff $R --name-only -- ui/e2e/ | grep -vE 'dashboard\.spec\.ts|usage\.spec\.ts|traffic\.spec\.ts|quota\.spec\.ts|pricing\.spec\.ts|mocks/handlers/(usage|logs|pricing)\.ts' | wc -l   # = 0 (no other spec; quota/streams handlers + index/seed untouched)
git diff $R --name-only -- ui/e2e/mocks/handlers/ | grep -vE '(usage|logs|pricing)\.ts' | wc -l   # = 0 (only three handler bodies corrected)
```

---

## 6. Out of scope (restated, binding)

ZERO Go changes (W5 usage/pricing/SSE exist, §1.2; quota is a variant against the
mock with the runtime Go aggregation deferred to a serial follow-up §8); no
`QueryClientProvider` mount / no PAR-UI-081 re-do (already HAVE from w6-a, §1.6);
no new route files / no `routeTree.gen.ts` change (§1.7); no dependency additions
(`@xyflow/react`+`recharts` already installed); no edits to any frozen w6-a/w6-b
file (no header exception remains — SPENT); no `fixture.ts`/MockEventSource edit
(§1.3 — the usage stream idles harmlessly); no mocks `index.ts`/seed/`store.ts`
edits (handler bodies only); no other e2e specs (NOT console — w6-i). Mock-vs-Go
contradiction, or a shared-handler-body correction that breaks a non-w6-g spec, or
an SSE-under-preview throw → escalate (§8), never patch Go or fudge a mock.

## 7. Diff-gate scope

Page-wave-1 plans (w6-c/e/g/h/i) commit to main concurrently, so a broad
`<base>..HEAD` range sweeps in sibling commits. The diff gate MUST be scoped to
w6-g's own commits. The orchestrator isolates them with:
`git log --oneline main | grep "^[0-9a-f]* phase-1/w6-g:" | awk '{print $1}'`
and runs `git diff <first-w6-g>^..<last-w6-g> -- [file list]` (same commit-range
scoping as w6-c §7 / w6-e §7 / w6-b §7).

`git diff <first-w6-g>^..<last-w6-g> --name-only` must be exactly a subset of:

```
ui/src/routes/dashboard.tsx
ui/src/routes/usage.tsx
ui/src/routes/logs.tsx
ui/src/routes/traffic.tsx
ui/src/routes/quota.tsx
ui/src/routes/pricing.tsx
ui/src/components/usage/usage-stats.tsx
ui/src/components/usage/usage-stats.test.tsx
ui/src/components/usage/request-logger.tsx
ui/src/components/usage/request-logger.test.tsx
ui/src/components/usage/request-details-tab.tsx
ui/src/components/usage/provider-topology.tsx
ui/src/components/usage/provider-limits.tsx
ui/src/components/usage/provider-limits.test.tsx
ui/src/components/usage/usage-charts.tsx
ui/src/components/usage/pricing-modal.tsx
ui/e2e/dashboard.spec.ts
ui/e2e/usage.spec.ts
ui/e2e/traffic.spec.ts
ui/e2e/quota.spec.ts
ui/e2e/pricing.spec.ts
ui/e2e/mocks/handlers/usage.ts          (body only — real-path correction §1.4)
ui/e2e/mocks/handlers/logs.ts           (body only — real-path correction §1.4)
ui/e2e/mocks/handlers/pricing.ts        (body only — real-path correction §1.4)
.planning/parity/matrix/9router-ui.md
.planning/parity/matrix/9router-usage.md
docs/WORKFLOW.md
```

Any file outside this list in the scoped diff is an automatic review REJECT.
`internal/**`, `ui/package.json`, `ui/src/routeTree.gen.ts`, `ui/e2e/mocks/fixture.ts`,
`ui/e2e/mocks/handlers/{index,quota,streams}.ts`, `ui/e2e/mocks/seed/**`, and any
frozen w6-a/b file are deliberately ABSENT — touching them is an automatic REJECT.
After merge, the six pages, `ui/src/components/usage/**`, and the three corrected
mock bodies become consume-only for later plans.

## 8. Escalations / cross-track dependencies

- **None blocking at authoring.** w6-a + w6-b are merged (live tree @ 9feca41:
  16 primitives present, `apiFetch`/stores/`fixture.ts`+MockEventSource in-tree,
  the W5 usage/pricing/SSE Go handlers in-tree per §1.2, `@xyflow/react`+`recharts`
  installed per §1.1). w6-g holds NO Go serial slot (zero new Go) and no frozen
  exception. Fully unblocked for page wave 1.
- **ESCALATION-1 (RESOLVED at authoring — mock path/shape divergences, §1.4):**
  - 1a usage stats `/api/usage/summary`(mock) → `/api/usage/stats`(real): page uses
    real path; correct mock body. Bounded, no Go.
  - 1b request logs `/api/logs`+`/api/usage`(mock) → `/api/usage/request-logs`(real):
    page uses real path; correct mock body. Bounded, no Go.
  - 1c **quota** `/api/quota`(mock) — NO Go equivalent (real is per-connection
    `/api/usage/{connectionId}`). Closing the runtime gap needs NEW Go (forbidden
    here). **Resolution:** PAR-UI-012 ships as variant-HAVE against the `/api/quota`
    mock (consumed unchanged); raise a serial Go follow-up to add an aggregating
    `GET /api/quota` (or wire the page to per-connection reads) — orchestrator
    decision, NOT in w6-g.
  - 1d pricing REST POST/PUT-by-id(mock) → GET/PATCH/DELETE(real, `pricing.go`):
    page + corrected mock use the real verbs/shape. Bounded, no Go.
- **ESCALATION-2 (CONDITIONAL — SSE under `vite preview`, §1.3):** if a real
  `EventSource("/api/usage/stream")` under the harness throws or blocks render
  (it must not — `MockEventSource` fires `open` then idles for unknown urls,
  `fixture.ts:59-98`), STOP and ESCALATE for a sanctioned one-line `fixture.ts`
  addition of a `/api/usage/stream` idle branch. **Expected: NOT needed.** Record
  the P8 probe result either way.
- **ESCALATION-3 (CONDITIONAL — shared mock handler body):** if correcting a
  `usage.ts`/`logs.ts`/`pricing.ts` body to match Go breaks a NON-w6-g spec that
  consumes the same handler, STOP and ESCALATE (the orchestrator serializes the
  shared-handler change) — do not fudge the mock, do not edit `index.ts`.
- **ESCALATION-4 (CONDITIONAL — mock-vs-Go):** if at T5 the real Go shape
  contradicts a corrected mock body, STOP and ESCALATE; no Go edit, no mock fudge
  (MAP decision 4).
- **PAR-UI-081 dependency (RESOLVED, §1.6):** already HAVE from w6-a (apiFetch =
  queryFn adapter). w6-g consumes `apiFetch`; the MAP-decision-2 TanStack-Query
  provider wiring (if ever wanted) is an orchestrator serial follow-up, NOT w6-g.
```
