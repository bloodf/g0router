# Micro-plan w6-j — Settings/profile + version cluster (UI + version Go; FINAL serial-slot holder)

```
wave: 6
plan: w6-j
status: READY (rev 1 — authored against merged w6-a + w6-b + w6-d + w6-e + w6-f,
  live tree @ e0fe9b9)
runs: page wave 2, AFTER w6-b MERGE (consumes frozen ui/src/components/ui/*),
  AFTER w6-a MERGE (consumes apiFetch/ApiError, the FROZEN themeStore/useTheme,
  the FROZEN settingsStore incl. setUpdateInfo, the FROZEN sidebar update-badge,
  the e2e mock fixture), AFTER w6-d MERGE (consumes the FROZEN i18n useI18n/
  setLocale + LOCALES), and alongside the REAL settings/OIDC Go (w3/earlier).
  Disjoint from w6-f/w6-k/w6-l/w6-m (different routes/components/specs). TAKES the
  routes_admin.go SERIAL SLOT as the FINAL holder, §1.8.
branch: main (direct commits, per repo convention)
commit-prefix: phase-1/w6-j:
ref-source: 9router frozen @ 827e5c3 —
  src/store/settingsStore.js (settings shape + update-checker concept),
  src/shared/components/ChangelogModal.js (GitHub CHANGELOG.md → marked render),
  src/shared/components/DonateModal.js (GitHub donate JSON → render),
  src/app/api/version/route.js (GET version + npm latest compare),
  src/app/api/version/shutdown/route.js (POST shutdown → process.exit after delay),
  src/app/dashboard/settings/* (the settings panels concept — note 9router's
  general settings page is composed inline; only settings/pricing/page.js survives
  as a file, pricing is w6-g's; w6-j ports the theme/lang/OIDC/password/DB panels).
base: <base> = git rev-parse HEAD recorded at P0 (expected e0fe9b9 at authoring;
  if main advanced, record the actual SHA and substitute everywhere §5 says <base>)
freeze-exception: NONE. The header.tsx / __root.tsx / main.tsx exceptions are
  SPENT (w6-a/b/c). w6-j touches NO frozen file — NOT sidebar.tsx, NOT header.tsx,
  NOT the stores' definitions (it CONSUMES settingsStore.setUpdateInfo via its
  existing public action, §1.6), NOT the i18n/theme hooks.
go-serial-slot: this plan holds the SINGLE in-flight edit to
  internal/server/routes_admin.go (additive version route registrations only). The
  slot is FREE at authoring — w6-f TOOK it for provider-nodes and RELEASED it to
  w6-j on close (open-questions.md w6-f line 33: "RELEASES it to w6-j on close";
  routes_admin.go:60-62 shows the merged w6-f provider-nodes block). w6-j is the
  MAP serial chain's FINAL holder (w6-pre→w6-d→w6-e→w6-j, MAP §Cross-cutting
  line 213). w6-j TAKES the slot, lands ONE additive routes_admin.go commit (T3),
  and RELEASES it to NOBODY — the wave-6 serial chain CLOSES on w6-j's close (§1.8).
new-route: NO. The settings route exists as a stub (§1.1); rewrite-only;
  routeTree.gen.ts is UNCHANGED (§1.7). w6-l is wave-2's new-route plan, not w6-j.
```

---

## 1. Scope — PAR rows

### Rows this plan closes

| Row | Claim | Target state after w6-j |
|---|---|---|
| PAR-UI-021 | Sidebar update-check badge driven by a version-check data source | HAVE (variant — the badge itself is FROZEN w6-a; w6-j supplies the DATA SOURCE: a version-check hook that sets `settingsStore.setUpdateInfo`, §1.6) |
| PAR-UI-055 | DonateModal (donation info modal) | HAVE (mounted from the settings page — a w6-j-owned surface; NO frozen-file edit, §1.7b) |
| PAR-UI-056 | ChangelogModal (changelog viewer) | HAVE (mounted from the settings page; consumes the version-check + a changelog source, §1.7b) |
| PAR-UI-097 | API `GET /api/settings` (read settings) | HAVE (REAL Go `settings.go` already exists, §1.2; consume) |
| PAR-UI-098 | API `PUT /api/settings` (update settings) | HAVE (REAL Go `settings.go`, §1.2; consume) |
| PAR-UI-099 | Settings: OIDC config panel | HAVE (variant — OIDC config persists into the flat settings map via `PUT /api/settings`; test via the REAL `POST /api/auth/oidc/test`, §1.4) |
| PAR-UI-100 | Settings: password change | HAVE (variant — NO Go password endpoint today; mock-only + serial Go follow-up, §1.4/§8 ESC-2) |
| PAR-UI-101 | Settings: DB info panel | HAVE (variant — NO Go DB-info endpoint today; mock-only + serial Go follow-up, §1.4/§8 ESC-3) |
| PAR-UI-102 | API `GET /api/version` (version + update info) | HAVE (Go — NEW `internal/admin/version.go`, §1.5) |
| PAR-UI-103 | API `POST /api/version/shutdown` (graceful shutdown) | HAVE (Go — NEW, testable-without-killing-process via an injectable shutdown hook, §1.5b) |

10 PAR-UI rows. Matches WAVE-6-MAP w6-j row (~line 136) and §Ownership w6-j
(~lines 191-194). Two rows have REAL existing Go (097/098 settings — consume); two
are NEW Go (102 version, 103 shutdown — `internal/admin/version.go`); 099 consumes
the REAL settings + the REAL OIDC-test endpoint; 100/101 are mock-only variants with
serial Go follow-ups; 021/055/056 are UI/data-source.

### 1.1 Preconditions already satisfied by merged waves (evidence)

- **Route STUB exists, must be REWRITTEN** (not created — so no new route file, so
  `routeTree.gen.ts` does NOT change; MAP decision 6 / §1.7). It renders only an
  `<h1>`: `ui/src/routes/settings.tsx:1-9` (`createFileRoute("/settings")`,
  `function SettingsPage(){ return <h1>Settings</h1>; }`).
  `ui/src/routeTree.gen.ts` ALREADY registers `/settings` (verify P4) — w6-j adds
  NO route file, so the tree does not change. The sidebar already links to
  `/settings` (`sidebar.tsx:56`, FROZEN — consume).
- Frozen primitives this plan CONSUMES (w6-b, never edited; 16 present): `Button`
  `ui/src/components/ui/button.tsx`; `Input` `ui/src/components/ui/input.tsx`;
  `Select` `ui/src/components/ui/select.tsx`; `Card`/`CardHeader`/`CardTitle`/
  `CardContent` `ui/src/components/ui/card.tsx`; `Modal`
  `ui/src/components/ui/modal.tsx` (controlled `open`/`onClose`, traffic lights,
  Escape, overlay, scroll-lock); `ConfirmModal`
  `ui/src/components/ui/confirm-modal.tsx`; `Toggle` `ui/src/components/ui/toggle.tsx`
  (the `require_login` toggle the settings spec drives, §1.3); `SegmentedControl`
  `ui/src/components/ui/segmented-control.tsx`; `Badge`
  `ui/src/components/ui/badge.tsx`; `Loading`/`Spinner`/`Skeleton`
  `ui/src/components/ui/{loading,skeleton}.tsx`; `Tooltip`
  `ui/src/components/ui/tooltip.tsx`.
- Frozen foundation this plan CONSUMES (w6-a, never edited): `apiFetch`
  `ui/src/lib/api.ts:19` + `ApiError` `ui/src/lib/api.ts:3`; toast via
  `useNotificationStore.push` `ui/src/stores/notification.ts`; Material Symbols
  `ui/src/index.css:3`.
- **Theme consumption (w6-a, FROZEN — CONSUME ONLY, §1.4):** `useTheme`
  `ui/src/hooks/use-theme.ts:14` returns `{theme, setTheme}` (backed by
  `useThemeStore`, persist key "theme"). The settings theme panel calls
  `setTheme(value)`. w6-j does NOT edit `use-theme.ts` or `stores/theme.ts`.
- **i18n consumption (w6-d, FROZEN — CONSUME ONLY, §1.4):** `useI18n`
  `ui/src/providers/i18n.tsx:19` returns `{currentLocale, locales: LOCALES,
  setLocale}` (`setLocale: (code:string)=>Promise<void>`, `i18n.tsx:10,45`). The
  settings language panel calls `setLocale(code)`. w6-j does NOT edit `i18n.tsx`,
  `i18n/*`, or `stores`.
- **settingsStore update-checker action (w6-a, FROZEN definition — CONSUME the
  public action, §1.6):** `ui/src/stores/settings.ts` exposes
  `setUpdateInfo(updateAvailable: boolean, latestVersion: string)`
  (`settings.ts:9,19-20`) and the `settings`/`setSettings` fields. The FROZEN
  sidebar reads `updateAvailable`/`latestVersion` and renders the
  `data-testid="update-badge"` block when both are truthy
  (`sidebar.tsx:72-73,101-110`). **w6-j NEVER edits `stores/settings.ts` or
  `sidebar.tsx`** — it CALLS `setUpdateInfo(...)` from a w6-j-owned hook (§1.6).
  Calling a store's existing public action is consumption, not a frozen edit.
- UI types this plan CONSUMES: settings are a flat `Record<string, unknown>`
  (`settings.ts:5` `settings: Record<string, unknown>`) mirroring the Go flat
  key→value map (§1.2). No new shared type is required (version DTO is local).
- **e2e mock harness already present + registered (CONSUME; correct bodies only on
  Go conflict, §1.4):** handler `ui/e2e/mocks/handlers/settings.ts`
  (`registerSettingsHandlers`: `/api/settings` GET → `store.settings`, PUT merges)
  registered at `ui/e2e/mocks/handlers/index.ts:4,40`; handler
  `ui/e2e/mocks/handlers/version.ts` (`registerVersionHandlers`: `/api/version` GET
  → `{version:"0.9.0-mock",build_date:"2024-01-01"}`, `/healthz` GET → `{status}`)
  registered at `index.ts:5,41`. **There is NO `version` seed** (the version mock is
  self-contained, returns a literal). **Neither mock serves `/api/version/shutdown`,
  password change, or DB info today** — w6-j adds those mock bodies (§1.4) inside the
  ALREADY-REGISTERED `settings.ts`/`version.ts` handlers (NO new handler file, NO
  `index.ts` edit — the registrations exist).
- Existing acceptance spec (the contract — §1.3): `ui/e2e/settings.spec.ts:1-37`
  (2 tests: (1) `/settings` body contains "Settings" + a non-hidden form control
  visible; (2) toggles a `require_login` switch — `label:has-text("Require login")
  + button, button[role="switch"]` — and clicks a Save button, then expects body to
  contain `/saved|success|salvo/i`; the test is conditional/`if visible`, so it must
  not regress). **There is NO `ui/e2e/version.spec.ts`** (`test ! -e` → true). w6-j
  EXTENDS `settings.spec.ts` (it is the version/changelog/donate carrier too — there
  is no separate version spec) with RED assertions; it does NOT create a new spec.
  Login helper `ui/e2e/helpers.ts:3` drives `#username`/`#password`,
  `username="admin" password="123456"`.

### 1.2 Real Go contract (file:line evidence)

Settings backend ALREADY EXISTS. The ONLY new Go is `internal/admin/version.go`
(version + shutdown, §1.5).

Routes (`internal/server/routes_admin.go`):
- `GET /api/settings` → `h.GetSettings` (`routes_admin.go:43`) — **PAR-UI-097 (real)**
- `PUT /api/settings` → `h.PutSettings` (`routes_admin.go:44`) — **PAR-UI-098 (real)**
- `GET /api/auth/oidc/start` → `h.OIDCStart` (`routes_admin.go:35`)
- `GET /api/auth/oidc/callback` → `h.OIDCCallback` (`routes_admin.go:36`)
- `POST /api/auth/oidc/test` → `h.OIDCTest` (`routes_admin.go:37`) — **the OIDC
  config TEST surface PAR-UI-099 consumes (NOT public; no session wrapper, but
  guard.go allows it)**
- `GET /api/health` → `healthHandler()` (`server.go:35`; public)

Body / response shapes (snake_case `{data,error:{message}}` envelope,
`internal/admin/respond.go:19-27` — `writeData`/`writeError`):
- **GetSettings** (`internal/admin/settings.go:10-17`): → 200
  `{data:<flat map[string]string>}` from `store.GetSettings()`
  (`internal/store/settings.go:11-32` reads the `settings` table key→value).
- **PutSettings** (`settings.go:20-37`): body is a flat `map[string]string` JSON
  object; merges via `store.SetSettings` (upsert, `settings.go:51-75`); re-reads;
  → 200 `{data:<flat map>}`. Bad JSON → 400. **This is the ONLY settings write —
  there is no per-key endpoint; OIDC/theme/lang/require_login are all keys in this
  flat map.**
- **OIDCTest** (`internal/admin/oidc.go:249-…`): body
  `{token_endpoint?,issuer_url?,client_id,client_secret,redirect_uri,scopes?}`
  (`oidc.go:250-256`); requires `client_id`+`redirect_uri` (400 otherwise);
  probes the OIDC token endpoint (live, but bounded); → `{data:<probe result>}`.
  **Under e2e the page does NOT call this (no live IdP); the OIDC panel test
  surface is mock-only/optional in e2e — the panel SAVES via `PUT /api/settings`.**
- **GetSettings keys (binding):** the settings flat map holds operator prefs.
  9router/g0router precedent keys observed: `theme`, `log_level`, `require_login`
  (the spec's toggle, `admin_test.go:264` puts `{"theme","log_level"}`). The OIDC
  panel writes `oidc_*` keys (e.g. `oidc_issuer_url`, `oidc_client_id`,
  `oidc_redirect_uri`, `oidc_scopes` — NEVER the secret in plaintext if the store
  has an `*_enc` pattern; verify at T4 whether OIDC secret has a dedicated
  encrypted column, else store under settings only if already the precedent —
  §8 ESC-4). **Do NOT invent new Go; the panel reads/writes existing settings
  keys via the flat map.**

**Gaps that have NO Go and ship as NEW version.go OR variant-mock (§1.5/§8):**
- `GET /api/version`, `POST /api/version/shutdown` (`grep -nE '/api/version'
  internal/server/routes_admin.go` → EMPTY; `internal/admin/version.go` ABSENT;
  guard.go:38-39 lists `/api/version/shutdown`+`/api/version/update` as FUTURE
  comment-only entries) — **NEW Go in `internal/admin/version.go` (PAR-UI-102/103),
  §1.5.**
- Password change: `grep -nE '/api/auth/password|/api/password|change-password'
  internal/server/routes_admin.go` → EMPTY; no `ChangePassword` handler
  (`auth.go` only has login/me/logout). **NO Go; mock-only.** PAR-UI-100 variant.
  §8 ESC-2.
- DB info: `grep -rnE '/api/settings/database|/api/database|DatabaseInfo'
  internal/` → EMPTY (only the guard.go:38 future comment). **NO Go; mock-only.**
  PAR-UI-101 variant. §8 ESC-3.

### 1.3 The settings page surface (binding interpretation)

The settings page is the operator's preferences panel. The EXISTING
`settings.spec.ts` is the binding contract; the page MUST keep both its tests green:
1. **General/theme panel**: theme `SegmentedControl` (light/dark/system) → calls
   `useTheme().setTheme` (FROZEN, §1.4); a `require_login` `Toggle`
   (`label:has-text("Require login") + button[role="switch"]`, the spec marker
   `settings.spec.ts:23`) bound to the `require_login` settings key; a `Save`
   button (`button:has-text("Save")`, marker `settings.spec.ts:29`) that `PUT
   /api/settings` and shows a success toast/text matching `/saved|success|salvo/i`
   (`settings.spec.ts:32`).
2. **Language panel**: locale `Select` of `LOCALES` → calls `useI18n().setLocale`
   (FROZEN, §1.4).
3. **OIDC config panel** (PAR-UI-099): inputs `issuer_url`/`client_id`/
   `client_secret`/`redirect_uri`/`scopes`; SAVE writes the `oidc_*` keys via `PUT
   /api/settings`; an optional "Test" button hits `POST /api/auth/oidc/test` (real
   Go; under e2e it is not exercised against a live IdP — the panel renders + saves,
   §1.2/§8 ESC-4).
4. **Password-change panel** (PAR-UI-100): current/new/confirm inputs; submit POSTs
   the mock password endpoint (NO Go, §1.4/§8 ESC-2). Variant-HAVE against mock.
5. **DB info panel** (PAR-UI-101): displays DB path/size/table counts from the mock
   DB-info endpoint (NO Go, §1.4/§8 ESC-3). Variant-HAVE against mock.
6. **About / version block**: shows the current version from `GET /api/version`
   (real new Go, §1.5); "View changelog" opens `<ChangelogModal>`; "Donate" opens
   `<DonateModal>` (both mounted here, the w6-j-owned surface, §1.7b). The version
   block also drives the update-checker hook (§1.6).

The page acceptance is the EXTENDED `settings.spec.ts` (§1.8): body contains
"Settings"; a form control visible (existing); the require_login-toggle+save flow
(existing, kept green); plus RED additions for the version/changelog/donate panels.

### 1.4 Mock paths/shapes (binding interpretation — CONSUME; correct BODY only on Go conflict)

| Surface | Mock route (file, owner) | Mock shape | Real Go (§1.2) | Resolution |
|---|---|---|---|---|
| Settings read/write | `/api/settings` GET/PUT (`handlers/settings.ts`, ALREADY registered) | GET→`store.settings`; PUT merges into `store.settings` | `GET/PUT /api/settings` real (`settings.go`), flat `map[string]string` | **AGREE.** CONSUME the `settings.ts` handler. Ensure `store.settings` seed has the keys the page reads (`theme`,`require_login`,`oidc_*` if displayed); if the seed lacks them the page tolerates absent keys (renders defaults). Correct the BODY only if the page needs a key the mock omits — w6-j-owned body (the handler is consumed ONLY by `settings.spec.ts`). |
| Version | `/api/version` GET (`handlers/version.ts`, ALREADY registered) | `{version:"0.9.0-mock",build_date:"2024-01-01"}` | NEW Go `version.go` (§1.5) → `{version,build_date,update_available?,latest_version?}` | **CORRECT the `version.ts` BODY** to mirror the NEW Go DTO: add `update_available`+`latest_version` so the update-checker hook (§1.6) can set the badge deterministically in e2e (e.g. `update_available:true,latest_version:"v9.9.9"`). w6-j-owned body. |
| Version shutdown | **NONE today** (`grep 'version/shutdown' ui/e2e/mocks/handlers/version.ts` → EMPTY) | n/a | NEW Go `version.go` (§1.5b) | **ADD a route in the EXISTING `version.ts` handler**: `POST /api/version/shutdown` → `{ok:true}` (deterministic; the mock NEVER simulates a real shutdown). w6-j-owned body. NO new handler file, NO `index.ts` edit. |
| Password change | **NONE today** | n/a | NONE | **ADD a route in the EXISTING `settings.ts` handler** (or `version.ts`): `POST /api/auth/password` (or `/api/settings/password`) → `{ok:true}` on a well-formed body, `400` on mismatch. PAR-UI-100 variant; serial Go follow-up (§8 ESC-2). |
| DB info | **NONE today** | n/a | NONE | **ADD a route in the EXISTING `settings.ts` handler**: `GET /api/settings/database` → `{path,size_bytes,tables:[...]}` (deterministic). PAR-UI-101 variant; serial Go follow-up (§8 ESC-3). |
| Changelog source | n/a (fetched from a static/markdown source, NOT an admin API) | markdown | n/a | The ChangelogModal fetches a changelog source; under e2e this MUST be intercepted deterministically. Resolve at T4: either bundle a static changelog string (no network) OR mock a `/api/version/changelog` route in `version.ts` (preferred — keeps it within the registered handler). Choose the route-mock to avoid an outbound network in tests. §1.7b. |
| Donate source | n/a | json | n/a | DonateModal fetches donate info; under e2e MUST be deterministic. Resolve at T4: bundle a static donate config OR mock a `/api/version/donate` route in `version.ts`. Prefer the route-mock. §1.7b. |

**Binding rule (MAP decision 4):** where mock and real Go disagree the real Go
wins and the mock body is corrected in-plan. For w6-j: CONSUME `settings.ts`
(agrees); CORRECT `version.ts` to mirror the NEW Go DTO (add `update_available`/
`latest_version`) and ADD the shutdown/changelog/donate routes to it; ADD the
password + DB-info mock routes to the registered `settings.ts` (or `version.ts`)
handler. **NO new mock-handler FILE and NO `ui/e2e/mocks/handlers/index.ts` edit —
both `settings.ts` and `version.ts` are ALREADY registered** (`index.ts:4-5,40-41`).
This is the key structural difference from w6-f/w6-i (which created new handlers):
w6-j only edits the BODIES of two already-registered handlers. If a body correction
would break a non-w6-j spec, STOP and ESCALATE (§8 ESC-5) — but `settings.ts`/
`version.ts` are consumed ONLY by `settings.spec.ts` (no other plan owns settings/
version). The seed `store.ts`/`seed/index.ts` are NOT touched (the version mock is
seedless; settings keys are tolerated-absent or added in the handler body literal).

### 1.5 Version Go contract (this plan's NEW Go, TDD)

The settings about-block + the update-checker hook reference `/api/version`.
9router (`api/version/route.js`) returns the current version and compares against
the npm "latest" to set update info; `api/version/shutdown/route.js` POSTs to exit
the process after a short delay. w6-j adds these as NEW, ADDITIVE Go in
`internal/admin/version.go`.

**Version exposure (binding).** The binary's version lives in package-level vars
`version`/`buildDate` in `cmd/g0router/main.go:20-21` — NOT reachable from
`internal/admin`. The handler MUST receive the version via injection (mirroring the
existing `SetUsageServices` pattern, `handlers.go:43-47`): add a
`SetVersionInfo(version, buildDate string)` setter on `Handlers` (sets unexported
fields `version`/`buildDate`); `version.go`'s `GetVersion` reads them. In tests the
env sets them via the setter (or they default to "" / a test literal). **Do NOT add
a new constructor param** (keep `New(...)` signature frozen — many callers; use a
setter like `SetUsageServices`). Wiring from `main.go` is a SINGLE call
`h.SetVersionInfo(version, buildDate)` after `New(...)` — but `main.go` constructs
the server via `server.NewWithShutdown`, which builds the `admin.Handlers`
internally (`server.go:28+`). **Resolve at T3 (§1.5b/§8 ESC-1):** either (a)
`NewWithShutdown` accepts version/builddate and forwards to `SetVersionInfo` +
`SetShutdownFunc`, OR (b) a post-construction setter exposed on `*server.Server`.
Prefer the minimal additive path; if it requires editing `server.go`/`main.go`
beyond a couple of additive lines, that is recorded as the §8 ESC-1 wiring decision
(it is allowed — `server.go`/`main.go` are NOT frozen w6-a UI files; but keep the
edit additive and justify in §7).

New file `internal/admin/version.go` (NEW) provides:

| Handler | Route (resolved) | Shape (snake_case, `{data}`) | PAR |
|---|---|---|---|
| `GetVersion` | `GET /api/version` | `{version,build_date,update_available,latest_version}` — `version`/`build_date` from the injected fields; `update_available`/`latest_version` are best-effort (in test: deterministic — derived from an injectable latest-version source/func, default `update_available:false,latest_version:""` so it NEVER makes a live network call in tests) | PAR-UI-102 |
| `Shutdown` | `POST /api/version/shutdown` | triggers the injected shutdown hook (§1.5b); returns `{ok:true}` (or `{success:true,message}` to mirror 9router) BEFORE the hook fires; 501/`{ok:false}` if no hook wired | PAR-UI-103 |

Route registration is the SERIAL-SLOT additive edit to `routes_admin.go` (§1.8/§3),
registered with `h.RequireSession(...)` (consistent with the settings routes;
verify guard.go does not need a public exception — guard.go:27 already lists
`/api/version` as a known path). New Go follows AGENTS.md: snake_case `{data,error}`
(`respond.go`), layered (handler→injected deps), no `init()`, errors-as-values, no
secret exposure, TDD (`version_test.go` committed RED first).

### 1.5b Shutdown — testable WITHOUT killing the test process (binding, RESOLVED)

**The problem.** `POST /api/version/shutdown` must, in production, trigger a graceful
process shutdown (the real wiring closes `srv` — `cmd/g0router/main.go:144`
`srv.Close()` / `server.go` `Shutdown()`). But the handler test runs IN the test
process; calling `os.Exit` or closing the real server would kill the test runner.

**The mechanism (binding).** The `Handlers` struct gets an injectable, nil-able
shutdown hook field: `shutdownFunc func()` (unexported), set via a public setter
`SetShutdownFunc(fn func())` (mirroring `SetUsageServices`/the proposed
`SetVersionInfo`, §1.5). The `Shutdown` handler:
1. If `h.shutdownFunc == nil` → respond `501`/`{ok:false,message:"shutdown not
   wired"}` (do NOT exit). (Tests that do not stub it assert this path.)
2. If set → respond `{ok:true}` FIRST (write the response), THEN invoke the hook
   asynchronously (`go h.shutdownFunc()` or via a short `time.AfterFunc` — mirrors
   9router's `setTimeout(...exit, 500)`), so the response is flushed before the
   process tears down.
3. The hook NEVER calls `os.Exit` directly inside the handler synchronously.

**Test design (binding).** `version_test.go` stubs the hook:
`env.handlers.SetShutdownFunc(func(){ called.Store(true) })` and asserts (a) the
response is `{ok:true}`, (b) after a bounded wait `called` is true, (c) the stub is
invoked exactly once, (d) `os.Exit` is NEVER reached (the stub records instead of
exiting — proof = the test process survives + a `! grep 'os.Exit' version.go`
freeze proof in §5). A SECOND test with NO hook set asserts the `501`/no-exit path.
**Real wiring (§1.5/§8 ESC-1):** `main.go`/`server.go` sets
`SetShutdownFunc(func(){ go func(){ time.Sleep(500*ms); srv.Close() /* or signal
the shutdown channel */ }() })` — the real graceful path. This wiring is the only
place a real shutdown is triggered, and it is NOT exercised by the unit test.

**Binding rule:** the handler is pure-testable (hook injected, async, response-first,
nil-safe); `os.Exit`/`srv.Close` appear ONLY in the production wiring, NEVER in
`version.go`'s handler body or its test. §5 proves `! grep -nE 'os.Exit|syscall'
internal/admin/version.go`.

### 1.6 Update-checker data source (PAR-UI-021/056) — NO frozen-file edit (binding, RESOLVED)

**The surface.** The FROZEN sidebar (`sidebar.tsx:72-73,101-110`) renders the
`data-testid="update-badge"` block when `useSettingsStore` has
`updateAvailable && latestVersion`. The store (`stores/settings.ts:9,19-20`,
FROZEN definition) exposes the public action
`setUpdateInfo(updateAvailable, latestVersion)`. **w6-j must NOT edit the sidebar
or the store definition** — it provides the DATA SOURCE that calls the existing
public action.

**Decision (binding).** w6-j creates a w6-j-OWNED hook
`ui/src/hooks/use-version-check.ts` (NEW file — `ui/src/hooks/` is not a frozen w6-a
set; only `hooks/use-theme.ts` is w6-a's, which w6-j does NOT touch). The hook:
1. `apiFetch("/api/version")` on mount → `{version,build_date,update_available,
   latest_version}` (the NEW Go / corrected mock, §1.4/§1.5).
2. If `update_available && latest_version` → calls
   `useSettingsStore.getState().setUpdateInfo(true, latest_version)` (the existing
   FROZEN public action — consumption, not an edit).
3. Returns `{version, updateAvailable, latestVersion}` for the settings about-block
   to display.

The hook is INVOKED from the w6-j-owned **settings page** (so mounting it requires
no frozen-file edit — the settings page is a w6-j CREATE). Visiting `/settings`
triggers the check, which sets the store, which lights the FROZEN sidebar badge.
This satisfies PAR-UI-021 (badge data source) without editing `sidebar.tsx` or
`stores/settings.ts`. **Calling `setUpdateInfo` is the sanctioned bridge** — the
store was explicitly designed (by w6-a) with this settable action for exactly this
consumer. Recorded in §8 + `open-questions.md`.

**Why not mount the check globally?** Mounting it in `__root.tsx`/sidebar would be a
frozen-file edit (FORBIDDEN — exceptions SPENT). Driving it from the settings page is
the parity behavior (9router's settings/about surface is where the version check
lives) and requires zero frozen edits. If a later orchestrator decision wants the
check on every page load, that is a serial follow-up wiring the hook into the
(frozen) root — NOT a w6-j blocker (§8 ESC-6).

### 1.7 `routeTree.gen.ts` is NOT touched

The settings route already exists as a stub (§1.1); rewriting its component body
does not change the route tree, and no new route file is added. Therefore
`ui/src/routeTree.gen.ts` is UNCHANGED by w6-j (MAP decision 6; w6-l is wave-2's
new-route plan, not w6-j). If a build incidentally reformats it, that is an
ESCALATION (§8), not an in-plan edit.

### 1.7b ChangelogModal + DonateModal mounting (PAR-UI-055/056) — NO frozen-file edit (binding, RESOLVED)

**The problem.** 9router triggers Changelog/Donate from a header/sidebar surface.
In g0router the header.tsx and sidebar.tsx are FROZEN (w6-a) and the header
exception is SPENT. Mounting either modal's trigger from a frozen file is FORBIDDEN.

**Evidence.** `grep -nE 'Donate|Changelog' ui/src/components/layout/header.tsx` →
EMPTY and `... sidebar.tsx` → EMPTY: neither frozen surface has a donate/changelog
trigger today, and w6-a left donate unwired (per the w6-b plan §NOT-in-scope note
"donate is w6-j"). So there is NO frozen trigger to consume.

**Decision (binding).** Both `<ChangelogModal>` and `<DonateModal>` are CREATED by
w6-j (`ui/src/components/settings/`) and MOUNTED from the w6-j-owned **settings
page** about-block (two `Button`s — "View changelog" / "Donate" — toggling local
`useState` open flags). This is a w6-j-owned surface, so NO frozen-file edit is
needed. Both modals consume the frozen `Modal` primitive. Their content sources are
intercepted deterministically under e2e (§1.4 changelog/donate rows — prefer a
`/api/version/changelog` + `/api/version/donate` mock route in the registered
`version.ts` handler over an outbound fetch). **No header/sidebar edit, no SPENT
exception reused.** Recorded in §8 + `open-questions.md`.

### 1.8 `settings.spec.ts` EXTENDED (no new spec); serial-slot handling (FINAL holder)

**No new spec.** `settings.spec.ts` already exists (§1.1) and is the carrier for
settings + version + changelog + donate (there is no separate `version.spec.ts`,
and the MAP w6-j evidence column names only `e2e/settings.spec.ts`). w6-j EXTENDS it
with RED assertions and KEEPS its two existing tests green. No new e2e file.

**Serial slot (binding — FINAL holder).** The MAP serial order is
w6-pre→w6-d→w6-e→w6-j (`MAP §Cross-cutting line 213`); **w6-j is the FINAL holder**.
w6-f TOOK the slot for provider-nodes and RELEASED it to w6-j on close
(`open-questions.md` w6-f line 33; routes_admin.go:60-62 shows the merged
provider-nodes block, so w6-f is merged). **Resolution:** the orchestrator MUST
confirm at P7 that NO other plan holds an unmerged routes_admin.go edit (w6-f is
merged; no other wave-2 plan adds Go routes — w6-k/l/m are UI-only per the MAP).
w6-j TAKES the slot, lands its SINGLE additive routes_admin.go commit (T3 — the two
version routes), and **RELEASES it to NOBODY**: the wave-6 serial chain CLOSES on
w6-j's close. State this explicitly in WORKFLOW.md at closeout. **Only ONE unmerged
routes_admin.go holder at a time** (MAP decision 5) — w6-j is the last.

### 1.9 NO new mock-handler file, NO mock-index edit (the structural difference)

Unlike w6-f/w6-i (which created new handler files + one sanctioned `index.ts`
append), **w6-j creates NO new mock-handler file and does NOT edit
`ui/e2e/mocks/handlers/index.ts`** — both `settings.ts` and `version.ts` are
ALREADY registered (`index.ts:4-5,40-41`). w6-j only edits the BODIES of those two
already-registered handlers (add the `update_available`/`latest_version` fields +
the shutdown/password/db-info/changelog/donate routes, §1.4). The seed `index.ts`,
`store.ts`, and `fixture.ts` are NOT touched (the version mock is seedless; settings
keys are added in the handler-body literal or tolerated-absent). If a body
correction would break a non-w6-j spec, STOP and ESCALATE (§8 ESC-5).

### NOT in scope (explicit)

- **No new route FILES** — only the existing `settings.tsx` stub is rewritten;
  `routeTree.gen.ts` untouched (§1.7).
- **No edits to existing Go** — `internal/admin/settings.go`, `internal/admin/
  oidc.go`, `internal/admin/auth.go`, `internal/store/settings.go`,
  `internal/store/**`, `internal/schemas/**` are FORBIDDEN; w6-j only ADDS
  `internal/admin/version.go` (+ its `_test.go`), the `SetVersionInfo`/
  `SetShutdownFunc` setters on `Handlers` (additive methods — `handlers.go` gains
  two unexported fields + two setter methods; that is the ONE additive edit to an
  existing Go file, justified in §7/§8 ESC-1), ADDITIVE version route lines in
  `routes_admin.go`, and the minimal additive version/shutdown wiring in
  `server.go`/`main.go` (§1.5/§8 ESC-1). NO schema change.
- **No edits to any frozen w6-a/w6-b/w6-d/w6-e/w6-f file** — NOT `sidebar.tsx`,
  NOT `header.tsx`, NOT `__root.tsx`, NOT `main.tsx`, NOT layout, NOT
  `ui/src/components/ui/*`, NOT `ui/src/stores/*` (incl. `settings.ts` — only its
  public action is CALLED, §1.6), NOT `ui/src/hooks/use-theme.ts`, NOT
  `ui/src/providers/i18n.tsx`, NOT `ui/src/i18n/*`, NOT `lib/api.ts`/`auth.ts`/
  `utils.ts`, NOT any prior-wave route/component. No header/sidebar exception
  remains (SPENT).
- **No TanStack Query wiring** — plain `apiFetch`.
- **No dependency additions** — every import resolves to installed packages or
  w6-a/b/d/e/f outputs. (If ChangelogModal wants markdown rendering, render plain
  text / sanitize manually — do NOT add `marked` unless already installed; verify
  at T4, else render the changelog as preformatted text. §8 ESC-7.)
- **No new mock-handler file, no `handlers/index.ts` edit, no seed/store/fixture
  edit** (§1.9) — only `settings.ts` + `version.ts` BODIES.
- **No real outbound network in tests** — version latest-check is injectable/default-
  off; changelog/donate sources are mock-route-intercepted; shutdown hook is stubbed.
- **No usage/pricing (w6-g), no providers/connections/models (w6-e), no
  endpoint/keys/virtual-keys (w6-f), no combos/routing (w6-h), no chat/console/
  translator (w6-i), no governance (w6-k), no mcp/skills (w6-l), no platform (w6-m).**

---

## 2. Precondition checks

Run all before any edit; abort and report to orchestrator on any failure.

```bash
# P0 — clean tree; record <base>
git status --porcelain     # must be empty (untracked tooling artifacts must be
                           # gitignored; worker uses explicit `git add <file>`,
                           # never `git add -A`; ui/dist/** is gitignored — never
                           # stage it)
git rev-parse HEAD         # record as <base> for §5 (expected e0fe9b9)

# P1 — w6-b primitives present and frozen (consumed)
ls ui/src/components/ui/*.tsx | grep -v test | wc -l    # = 16 (w6-b set intact)
grep -n "export function Modal\|export interface ModalProps" ui/src/components/ui/modal.tsx
grep -n "Toggle" ui/src/components/ui/toggle.tsx
grep -n "SegmentedControl" ui/src/components/ui/segmented-control.tsx
grep -n "ConfirmModal" ui/src/components/ui/confirm-modal.tsx

# P2 — w6-a foundation present and frozen (consumed)
grep -n "export async function apiFetch" ui/src/lib/api.ts
grep -n "export class ApiError" ui/src/lib/api.ts
grep -n "push:" ui/src/stores/notification.ts
grep -n "setUpdateInfo" ui/src/stores/settings.ts        # the public action w6-j CALLS (§1.6)
grep -n "update-badge\|updateAvailable" ui/src/components/layout/sidebar.tsx  # FROZEN badge consumer (do NOT edit)
grep -n "export function useTheme" ui/src/hooks/use-theme.ts  # FROZEN (consume setTheme)

# P3 — w6-d i18n present and frozen (consumed)
grep -n "export const useI18n\|setLocale" ui/src/providers/i18n.tsx
grep -n "LOCALES" ui/src/i18n/locales.ts

# P4 — the settings route stub is still bare (safe to rewrite); routeTree has it; no new dirs
grep -n "<h1>Settings</h1>" ui/src/routes/settings.tsx
grep -nE "'/settings'|SettingsRoute" ui/src/routeTree.gen.ts ; echo "^ expect present (no new route file; tree UNCHANGED §1.7)"
test ! -d ui/src/components/settings && echo "settings components dir absent (good)"
test ! -e ui/src/hooks/use-version-check.ts && echo "version-check hook absent (good — CREATE §1.6)"

# P5 — e2e mock harness present + registered (CONSUME settings; correct version body §1.4; NO index edit §1.9)
grep -n "registerSettingsHandlers\|registerVersionHandlers" ui/e2e/mocks/handlers/index.ts  # ALREADY registered (4-5,40-41)
grep -n "/api/settings" ui/e2e/mocks/handlers/settings.ts
grep -n "/api/version" ui/e2e/mocks/handlers/version.ts
grep -rn "version/shutdown\|settings/database\|auth/password" ui/e2e/mocks/handlers/ ; echo "^ expect EMPTY (new mock routes added to settings.ts/version.ts bodies §1.4)"
test ! -e ui/e2e/version.spec.ts && echo "no version spec (good — settings.spec.ts is the carrier §1.8)"

# P6 — Go reality: settings/OIDC real; version/shutdown/password/db-info ABSENT
grep -nE '/api/settings' internal/server/routes_admin.go            # GET+PUT present (43-44)
grep -nE '/api/auth/oidc/(start|callback|test)' internal/server/routes_admin.go  # OIDC present (35-37)
grep -nE '/api/version|/api/version/shutdown' internal/server/routes_admin.go ; echo "^ expect EMPTY (the gaps §1.5)"
test ! -e internal/admin/version.go && echo "version.go absent (good — NEW §1.5)"
grep -rnE '/api/auth/password|/api/settings/database' internal/ ; echo "^ expect EMPTY/comment-only (password+db-info no Go §1.4)"
grep -n "func New(\|SetUsageServices" internal/admin/handlers.go   # the setter pattern to mirror for SetVersionInfo/SetShutdownFunc
grep -nE 'version\s*=|buildDate\s*=' cmd/g0router/main.go          # version vars (§1.5 injection source)
grep -nE 'srv.Close|Shutdown' cmd/g0router/main.go internal/server/server.go  # the real shutdown path (§1.5b wiring)

# P7 — routes_admin.go serial slot is FREE (FINAL holder); take it (§1.8)
git log --oneline -3 -- internal/server/routes_admin.go   # last touch = w6-f provider-nodes (merged)
grep -n "/api/provider-nodes" internal/server/routes_admin.go  # confirm w6-f merged (slot released to w6-j)
# Orchestrator MUST confirm no concurrent wave-2 plan holds an unmerged
# routes_admin.go edit before w6-j begins T3. w6-j is the FINAL holder; releases to NOBODY.

# P8 — harness green at base
cd ui && npx playwright test e2e/settings.spec.ts
# Record base result: stub renders only <h1>Settings</h1>. Test 1 ("settings form
# loads") asserts body "Settings" (present via <h1>/sidebar chrome) + a non-hidden
# form control visible — likely FAILS on the bare stub (no form control beyond
# chrome) — RECORD exact pass/fail. Test 2 (require_login toggle) is conditional
# (`if visible`) — likely PASSES vacuously on the stub. Record in WORKFLOW.md.
cd ui && npm run build                               # exit 0
cd ui && npx vitest run src/                         # exit 0 (existing units green)
go test ./... && go vet ./...                        # exit 0 (Go untouched-green)
```

---

## 3. Exclusive file ownership

After w6-j merges, all CREATE files below are owned by w6-j; later plans consume,
never edit (MAP decision 7).

**CREATE — route (REWRITE existing stub; no new route file, §1.7):**

| File | Exports / contract |
|---|---|
| `ui/src/routes/settings.tsx` (REWRITE) | `Route=createFileRoute("/settings")`; `SettingsPage`: composes the panels (§1.3) — general/theme (`SegmentedControl`+`require_login` `Toggle`+`Save`→`PUT /api/settings`+success toast), language (`Select`→`useI18n().setLocale`), OIDC (`<OidcConfigPanel>`), password (`<PasswordPanel>`), DB info (`<DbInfoPanel>`), about/version (calls `useVersionCheck()`, shows version, "View changelog"→`<ChangelogModal>`, "Donate"→`<DonateModal>`). Body contains "Settings". Keeps `settings.spec.ts` test-1 + test-2 green. |

**CREATE — page/domain components (`ui/src/components/settings/`):**

| File | Exports / contract |
|---|---|
| `general-settings-panel.tsx` | `GeneralSettingsPanel` (PAR-UI-097/098) — consumes `Card`+`SegmentedControl`+`Toggle`+`Button`; theme via FROZEN `useTheme().setTheme`; `require_login` toggle bound to the `require_login` settings key; reads `apiFetch("/api/settings")`; Save → `apiFetch("/api/settings",{method:"PUT",body:{...keys}})` + success toast (`/saved|success|salvo/i`). The `require_login`-toggle+Save flow that `settings.spec.ts:23-32` drives. |
| `language-settings-panel.tsx` | `LanguageSettingsPanel` — consumes `Card`+`Select`; lists FROZEN `useI18n().locales` (LOCALES); change → `useI18n().setLocale(code)`. |
| `oidc-config-panel.tsx` | `OidcConfigPanel` (PAR-UI-099) — consumes `Card`+`Input`+`Button`; fields `issuer_url`/`client_id`/`client_secret`/`redirect_uri`/`scopes`; Save → `PUT /api/settings` writing `oidc_*` keys; optional "Test" → `apiFetch("/api/auth/oidc/test",{method:"POST",body})` (real Go; not exercised vs live IdP in e2e, §1.2/§8 ESC-4). Secret never echoed back into a value attribute. |
| `password-panel.tsx` | `PasswordPanel` (PAR-UI-100) — consumes `Card`+`Input`+`Button`; current/new/confirm; submit → `apiFetch("/api/auth/password",{method:"POST",body})` (mock-only, §1.4/§8 ESC-2); client-side confirm-match validation. |
| `db-info-panel.tsx` | `DbInfoPanel` (PAR-UI-101) — consumes `Card`; `apiFetch("/api/settings/database")` (mock-only, §1.4/§8 ESC-3) → display path/size/tables. |
| `changelog-modal.tsx` | `ChangelogModal` (PAR-UI-056) — consumes frozen `Modal`; on open fetches the changelog source (mock route `/api/version/changelog`, §1.4/§1.7b — NO outbound network in tests) → renders as preformatted/sanitized text (NO `marked` dep unless already installed, §8 ESC-7). Mounted from settings page (§1.7b). |
| `donate-modal.tsx` | `DonateModal` (PAR-UI-055) — consumes frozen `Modal`; on open fetches the donate source (mock route `/api/version/donate`, §1.4/§1.7b) → renders donation info. Mounted from settings page (§1.7b). |

**CREATE — hook (`ui/src/hooks/`, NEW file — not the frozen `use-theme.ts`):**

| File | Exports / contract |
|---|---|
| `use-version-check.ts` | `useVersionCheck()` (PAR-UI-021/102) — `apiFetch("/api/version")` on mount; if `update_available && latest_version` → `useSettingsStore.getState().setUpdateInfo(true, latest_version)` (FROZEN public action, §1.6 — consumption, not an edit); returns `{version, buildDate, updateAvailable, latestVersion, loading}`. Invoked from the settings page → lights the FROZEN sidebar badge. |

**CREATE — unit tests (vitest — logic reachable without a DOM):**

| File | Contents |
|---|---|
| `ui/src/hooks/use-version-check.test.ts` | ≥3 tests via stubbed `apiFetch` + a stubbed/real `useSettingsStore`: (1) fetches `/api/version` and returns the version; (2) when `update_available:true` it calls `setUpdateInfo(true, latest_version)` (assert the store action invoked / state updated); (3) when `update_available:false` it does NOT set the badge. Committed RED before `use-version-check.ts`. |
| `ui/src/components/settings/general-settings-panel.test.tsx` | ≥2 tests via `renderToString`/stubbed `apiFetch`: (1) renders the `require_login` toggle + Save reflecting seeded settings; (2) Save PUTs `/api/settings` with the toggled key. Committed RED before `general-settings-panel.tsx`. |

(`oidc-config-panel.tsx`, `password-panel.tsx`, `db-info-panel.tsx`,
`changelog-modal.tsx`, `donate-modal.tsx`, `language-settings-panel.tsx` are
DOM/modal-heavy; their coverage is the e2e assertions — same disposition as
w6-e/w6-f/w6-g modal components.)

**CREATE — Go (`internal/admin/version.go` + `_test.go`, NEW):**

| File | Contents |
|---|---|
| `internal/admin/version.go` | `GetVersion`/`Shutdown` + a local version DTO, per §1.5/§1.5b. Reads injected `version`/`buildDate`; `Shutdown` triggers the injectable nil-safe `shutdownFunc` async (response-first), `501` if unwired; NEVER `os.Exit`/`srv.Close` inside the handler. Uses `writeData`/`writeError` (`respond.go`). No `init()`; errors-as-values; no secret exposure. |
| `internal/admin/version_test.go` | Table-driven tests via `newTestEnv` (`admin_test.go:24`) + the `call` helper (`admin_test.go:27`): GetVersion returns the injected version/build_date (set via `SetVersionInfo`) and a deterministic `update_available:false` by default; Shutdown with a stubbed `SetShutdownFunc` returns `{ok:true}` AND invokes the stub exactly once (bounded wait) WITHOUT exiting; Shutdown with NO hook returns `501`/`{ok:false}` and does NOT invoke anything. Committed RED before the impl file. |

**MODIFY — additive setters on the existing Handlers (justified, §1.5/§8 ESC-1):**

| File | Change (and ONLY this change) |
|---|---|
| `internal/admin/handlers.go` | ADD two unexported fields (`version string`, `buildDate string`, `shutdownFunc func()`) to the `Handlers` struct and two additive setter methods `SetVersionInfo(version, buildDate string)` + `SetShutdownFunc(fn func())` (mirroring `SetUsageServices`, `handlers.go:43-47`). NO change to `New(...)` signature, NO change to existing fields/methods. Diff bound §5: small additive block. |

**MODIFY — minimal version/shutdown wiring (NOT frozen UI; additive, §1.5/§8 ESC-1):**

| File | Change |
|---|---|
| `internal/server/server.go` | ADD a minimal path to forward version/buildDate + the real shutdown hook into the admin `Handlers` (either via an extended `NewWithShutdown` param OR a setter on `*Server`). Additive only; justify in §7. |
| `cmd/g0router/main.go` | ADD the single call that injects `version`/`buildDate` (vars at `main.go:20-21`) and wires the real shutdown hook (`go func(){ time.Sleep(500ms); srv.Close() }()`) into the Handlers. Additive only. |

**MODIFY — serial-slot route registration (additive only):**

| File | Change (and ONLY this change) |
|---|---|
| `internal/server/routes_admin.go` | ADD (near the settings block or end): `r.GET("/api/version", h.RequireSession(h.GetVersion))`, `r.POST("/api/version/shutdown", h.RequireSession(h.Shutdown))`. NOTHING else changes. Diff bound §5: ≤ 4 added lines. SERIAL SLOT — only holder while live; FINAL holder, RELEASE to NOBODY (§1.8). |

**MODIFY — e2e (the acceptance contract; correct version body + add mock routes §1.4):**

| File | Change |
|---|---|
| `ui/e2e/settings.spec.ts` | KEEP the 2 existing tests (form-loads "Settings" + require_login-toggle+Save). ADD RED: theme/language panels render; OIDC panel inputs render; password panel renders; DB-info panel shows DB data (from the mock); the about/version block shows a version (from `/api/version`); "View changelog" opens the ChangelogModal; "Donate" opens the DonateModal; visiting `/settings` makes the sidebar `data-testid="update-badge"` appear (mock `update_available:true`, drives `setUpdateInfo` via the hook, §1.6). |
| `ui/e2e/mocks/handlers/version.ts` (BODY) | CORRECT to add `update_available`+`latest_version` to the GET `/api/version` body (e.g. `update_available:true,latest_version:"v9.9.9"` for the badge e2e); ADD routes `POST /api/version/shutdown`→`{ok:true}`, `GET /api/version/changelog`→a small markdown/text, `GET /api/version/donate`→a small JSON. ALREADY registered (`index.ts:5`) — body only, NO index edit. w6-j-owned. |
| `ui/e2e/mocks/handlers/settings.ts` (BODY) | KEEP the GET/PUT `/api/settings` behavior; ensure the GET seed/body carries the keys the page reads (`theme`,`require_login`,`oidc_*` optional); ADD routes `POST /api/auth/password`→`{ok:true}`/`400` on mismatch, `GET /api/settings/database`→`{path,size_bytes,tables}`. ALREADY registered (`index.ts:4`) — body only, NO index edit. w6-j-owned. |

**FORBIDDEN:** everything else. Explicitly: all of `internal/admin/settings.go`,
`internal/admin/oidc.go`, `internal/admin/auth.go`, `internal/store/**`,
`internal/schemas/**` (the version setters live in `handlers.go` + the new
`version.go` ONLY); all `ui/src/components/ui/*` (w6-b); all `ui/src/stores/*`
(incl. `settings.ts` — only its public `setUpdateInfo` action is CALLED, §1.6);
`ui/src/hooks/use-theme.ts` (w6-a — w6-j adds the SEPARATE `use-version-check.ts`);
`ui/src/providers/i18n.tsx`, `ui/src/i18n/*` (w6-d); `ui/src/lib/api.ts`,
`utils.ts`, `auth.ts`, `oauth-popup.ts`; `ui/src/components/layout/*` (sidebar,
header, __root, mobile-sidebar, toaster — FROZEN, §1.6/§1.7b); `ui/src/main.tsx`;
all prior-wave routes/components; `ui/package.json` + lockfile; `ui/vite.config.ts`;
`ui/playwright.config.ts`; `ui/components.json`; `ui/src/index.css`;
`ui/src/routeTree.gen.ts` (UNCHANGED §1.7); `ui/e2e/mocks/handlers/index.ts`
(ALREADY registers settings+version — NO edit, §1.9); `mocks/seed/index.ts`;
`mocks/store.ts`; `mocks/fixture.ts`; all other `ui/e2e/*.spec.ts`; all other mock
handlers; all other `internal/server/*` routes; any other `internal/admin/*.go`.

---

## 4. TDD tasks

Cadence (strict): **no route/component/hook/Go file may exist (or be rewritten
beyond its stub) before the failing test that covers it is committed.** Both tracks
are strict-TDD: Go `_test.go` before Go impl; UI red specs/units before UI impl.
`cd ui && npm run build` green at EVERY commit (test files + red specs are never
imported by production code). `go test ./... && go vet ./... && go build ./...`
green at EVERY commit. The `settings.spec.ts` RED additions stay RED from T1 until
impl greens them; the two pre-existing tests must NEVER regress.

### T1 — STEP(a): extend settings.spec + correct version/settings mock bodies (commit RED)

Add RED assertions to `ui/e2e/settings.spec.ts` (§3) — KEEP the 2 existing tests.
CORRECT `ui/e2e/mocks/handlers/version.ts` BODY (add `update_available`/
`latest_version`; add `POST /api/version/shutdown`, `GET /api/version/changelog`,
`GET /api/version/donate`). ADD the password + DB-info routes to the registered
`ui/e2e/mocks/handlers/settings.ts` BODY (§1.4). NO `index.ts`/seed/store/fixture
edit (§1.9). CONSUME all other handlers unchanged.

STEP(b): run the spec — **record failure output** (no panels, no version block, no
modals, no badge). Verify the 2 pre-existing tests still pass (no regression).
Commit RED: `phase-1/w6-j: failing settings e2e (panels/version/changelog/donate/badge) + version/settings mock-body corrections (TDD red)`.

**Mock-vs-reality gate**: while correcting `version.ts`/`settings.ts` bodies,
re-read the Go (§1.2 — settings real; version is the NEW DTO §1.5). If correcting a
body breaks a non-w6-j spec, STOP and ESCALATE (§8 ESC-5) — no existing-Go edit, no
mock fudge, no foundation-mock/index edit.

### T2 — STEP(a): Go `version_test.go` (commit RED)

Write the table-driven tests per §3 against `newTestEnv` + `call`
(`admin_test.go:24,27`). Stub `SetShutdownFunc` to record-not-exit; assert the
response-first + async-invoke + nil-safe-501 behavior (§1.5b). Decide the
`SetVersionInfo`/`SetShutdownFunc` setter shape here (mirror `SetUsageServices`).
`go test ./internal/admin/ -run Version` → FAILS (handlers + setters missing).
Record failure. Commit RED: `phase-1/w6-j: failing version/shutdown Go tests (TDD red)`.

### T3 — STEP(b): Go `version.go` + setters + wiring + serial-slot routes

Implement `GetVersion`/`Shutdown` per §1.5/§1.5b; add the `version`/`buildDate`/
`shutdownFunc` fields + `SetVersionInfo`/`SetShutdownFunc` setters to `handlers.go`
(additive, §3). Wire version/buildDate + the real shutdown hook from `server.go`/
`main.go` (additive, §1.5/§8 ESC-1). Add the two additive route lines to
`routes_admin.go` (§3). **Take the serial slot first (§1.8 — orchestrator confirms
it is free; w6-j is the FINAL holder).** Gates: `go test ./... && go vet ./... &&
go build ./...` green (version tests now green; `! grep os.Exit version.go` proof
§5). Commit: `phase-1/w6-j: version + shutdown admin API (testable shutdown hook) + serial-slot routes`.

### T4 — STEP(b): version-check hook + general/language/about + changelog/donate + settings page (part 1)

STEP(a) first: ensure `use-version-check.test.ts` + `general-settings-panel.test.tsx`
are committed RED (write here if not; run `cd ui && npx vitest run
src/hooks/ src/components/settings/` red; commit:
`phase-1/w6-j: failing unit tests for use-version-check + general-settings-panel (TDD red)`).
STEP(b): implement `use-version-check.ts`, `general-settings-panel.tsx`,
`language-settings-panel.tsx`, `changelog-modal.tsx`, `donate-modal.tsx`; begin the
`settings.tsx` rewrite mounting general/language/about + the two modals + the hook.
Decide the changelog markdown rendering (NO `marked` dep unless installed — §8
ESC-7) and the changelog/donate mock-route sources (§1.4/§1.7b) here. Gates: vitest
green; the badge + version-block + changelog/donate parts of `settings.spec.ts`
green; the 2 pre-existing settings tests STILL green; `npm run build` green; `go
test ./...` green. Commit: `phase-1/w6-j: version-check hook, general/language/about panels, changelog + donate modals, settings page`.

### T5 — STEP(b): OIDC + password + DB-info panels (settings page part 2)

Implement `oidc-config-panel.tsx`, `password-panel.tsx`, `db-info-panel.tsx`;
finish the `settings.tsx` rewrite mounting them. Gates: ALL of `settings.spec.ts`
green (incl. the 2 pre-existing); `cd ui && npx vitest run src/` green; `npm run
build` green; `go test ./... && go vet ./...` green. Commit:
`phase-1/w6-j: OIDC config + password + DB-info settings panels`.

### T6 — full gates + closeout

```bash
cd ui && npm run build
cd ui && npx playwright test e2e/settings.spec.ts        # all green (2 existing + RED additions)
cd ui && npx playwright test                             # full suite: no spec green-at-base may be red
cd ui && npx vitest run src/                             # all green incl new units
go test ./... && go vet ./... && go build ./...          # green
go test ./internal/admin/ -run Version -v                # ≥3 version/shutdown cases pass
! grep -nE 'os\.Exit|syscall' internal/admin/version.go && echo "shutdown handler does not exit OK"
```
Flip §1 matrix rows in `.planning/parity/matrix/9router-ui.md`: PAR-UI-021 → HAVE
(variant, cite §1.6); PAR-UI-055 → HAVE (cite §1.7b); PAR-UI-056 → HAVE (cite
§1.7b); PAR-UI-097/098 → HAVE (real Go, consume); PAR-UI-099 → HAVE (variant, cite
§1.4); PAR-UI-100/101 → HAVE (variant mock-only, cite §8 ESC-2/3); PAR-UI-102/103 →
HAVE (Go, cite §1.5/§1.5b). Update `docs/WORKFLOW.md` (record P8 base spec
observations, the version/shutdown Go design + the wiring decision §1.5/§8 ESC-1,
the update-checker + changelog/donate mounting decisions §1.6/§1.7b, the serial-slot
take-from-w6-f-released/**release-to-NOBODY — wave-6 serial chain CLOSED** §1.8, and
the §8 mock-only follow-ups). Append §8 open items to
`.planning/parity/plans/open-questions.md`. Final commit:
`phase-1/w6-j: close — settings/version cluster; version Go; serial chain closed; matrix flips`.
**On the close commit, the routes_admin.go serial slot is RELEASED TO NOBODY — the
wave-6 serial chain is CLOSED (§1.8).**

---

## 5. Binary acceptance criteria

All must hold; each is yes/no. `<base>` = the commit recorded at P0 (e0fe9b9 at
authoring). Diff gate is **w6-j commit-range-scoped** (§7).

**Test gates**
- `cd ui && npx playwright test e2e/settings.spec.ts` → exit 0, all pass, 0 skipped
  (the 2 pre-existing + the RED additions).
- `cd ui && npx vitest run src/` → exit 0 (all prior + new units green).
- `cd ui && npm run build` → exit 0.
- `go test ./... && go vet ./... && go build ./...` → exit 0.
- `go test ./internal/admin/ -run Version -v` → exit 0, ≥3 version/shutdown cases pass.

**TDD-order proof** — each impl file's covering test appears in an
earlier-or-equal commit:
```bash
# Go: version.go after version_test.go
ct=$(git log --format=%ct --diff-filter=A -1 -- internal/admin/version_test.go)
cf=$(git log --format=%ct --diff-filter=A -1 -- internal/admin/version.go)
[ "$ct" -le "$cf" ] || echo "TDD VIOLATION: version.go"                # prints nothing
# UI units after their tests
ct=$(git log --format=%ct --diff-filter=A -1 -- ui/src/hooks/use-version-check.test.ts)
cf=$(git log --format=%ct --diff-filter=A -1 -- ui/src/hooks/use-version-check.ts)
[ "$ct" -le "$cf" ] || echo "TDD VIOLATION: use-version-check.ts"      # nothing
ct=$(git log --format=%ct --diff-filter=A -1 -- ui/src/components/settings/general-settings-panel.test.tsx)
cf=$(git log --format=%ct --diff-filter=A -1 -- ui/src/components/settings/general-settings-panel.tsx)
[ "$ct" -le "$cf" ] || echo "TDD VIOLATION: general-settings-panel.tsx"  # nothing
# e2e RED-extension commit precedes the settings page rewrite
sa=$(git log --format=%ct -1 --grep="failing settings e2e")
si=$(git log --format=%ct --diff-filter=M -1 -- ui/src/routes/settings.tsx)
[ "$sa" -le "$si" ] || echo "TDD VIOLATION: settings.tsx before red spec"  # nothing
```

**Grep proofs**
```bash
grep -rn "/api/settings" ui/src/components/settings/general-settings-panel.tsx     # PAR-UI-097/098
grep -rn "require_login" ui/src/components/settings/general-settings-panel.tsx      # spec marker §1.3
grep -rn "useTheme\|setTheme" ui/src/components/settings/general-settings-panel.tsx # FROZEN theme consume §1.4
grep -rn "useI18n\|setLocale" ui/src/components/settings/language-settings-panel.tsx # FROZEN i18n consume §1.4
grep -rn "/api/auth/oidc/test\|/api/settings" ui/src/components/settings/oidc-config-panel.tsx  # PAR-UI-099
grep -rn "/api/auth/password" ui/src/components/settings/password-panel.tsx          # PAR-UI-100 (mock)
grep -rn "/api/settings/database" ui/src/components/settings/db-info-panel.tsx       # PAR-UI-101 (mock)
grep -rn "/api/version" ui/src/hooks/use-version-check.ts                            # PAR-UI-021/102
grep -rn "setUpdateInfo" ui/src/hooks/use-version-check.ts                           # §1.6 badge data source (CALLS frozen action)
grep -rn "Modal" ui/src/components/settings/changelog-modal.tsx ui/src/components/settings/donate-modal.tsx  # PAR-UI-055/056 frozen Modal
! grep -rn "sidebar.tsx\|stores/settings" ui/src/components/settings/ ui/src/hooks/use-version-check.ts && echo "no frozen sidebar/store-definition edit OK"  # §1.6/§1.7b
# Go version/shutdown:
grep -n "GetVersion\|func (h \*Handlers) Shutdown" internal/admin/version.go         # PAR-UI-102/103
grep -n "writeData\|writeError" internal/admin/version.go                            # snake_case {data,error}
grep -n "shutdownFunc\|SetShutdownFunc\|SetVersionInfo" internal/admin/handlers.go internal/admin/version.go  # injectable hook §1.5b
grep -nE '/api/version|/api/version/shutdown' internal/server/routes_admin.go        # routes registered
! grep -nE 'os\.Exit|syscall' internal/admin/version.go && echo "shutdown not exiting in handler OK"  # §1.5b testable-without-killing
! grep -n "func init(" internal/admin/version.go && echo "no init() OK"
```

**Negative / freeze proofs (w6-j commit-range — see §7)**
```bash
R="<first-w6-j>^..<last-w6-j>"
git diff $R --name-only -- internal/admin/settings.go internal/admin/oidc.go internal/admin/auth.go internal/store/ internal/schemas/ | wc -l  # = 0 (existing settings/OIDC Go + store frozen)
git diff $R --name-only -- internal/ | grep -vE 'internal/admin/version(_test)?\.go|internal/admin/handlers\.go|internal/server/routes_admin\.go|internal/server/server\.go' | wc -l  # = 0 (only new file + additive setters + slot + wiring)
git diff $R -- internal/admin/handlers.go | grep "^+" | grep -vE 'version |buildDate |shutdownFunc |func .*SetVersionInfo|func .*SetShutdownFunc|^\+\+\+|^\+\s*$|^\+\s*h\.|^\+}' | wc -l  # additive setters/fields only (manual review the diff is small + additive)
git diff $R --name-only -- ui/src/components/ui/ | wc -l                # = 0 (w6-b frozen)
git diff $R --name-only -- ui/src/stores/ ui/src/providers/ ui/src/i18n/ ui/src/hooks/use-theme.ts ui/src/lib/api.ts ui/src/lib/utils.ts ui/src/lib/auth.ts | wc -l   # = 0 (w6-a/d frozen; settings store NOT edited §1.6)
git diff $R --name-only -- ui/src/components/layout/ ui/src/routes/__root.tsx ui/src/main.tsx | wc -l   # = 0 (sidebar/header/root frozen §1.6/§1.7b)
git diff $R --name-only -- ui/src/routeTree.gen.ts | wc -l             # = 0 (§1.7 unchanged)
git diff $R --name-only -- ui/package.json ui/package-lock.json ui/vite.config.ts ui/playwright.config.ts ui/components.json ui/src/index.css | wc -l  # = 0
git diff $R --name-only -- 'ui/src/routes/' | grep -vE 'settings\.tsx' | wc -l  # = 0 (only the settings stub rewritten)
git diff $R --name-only -- ui/e2e/ | grep -vE 'settings\.spec\.ts|mocks/handlers/(settings|version)\.ts' | wc -l  # = 0 (no other spec; only two registered handler bodies)
git diff $R --name-only -- ui/e2e/mocks/handlers/index.ts ui/e2e/mocks/seed/ ui/e2e/mocks/store.ts ui/e2e/mocks/fixture.ts | wc -l  # = 0 (NO index/seed/store/fixture edit §1.9)
git diff $R -- internal/server/routes_admin.go | grep "^+" | wc -l     # ≤ 4 (additive route lines + +++ header)
git log --oneline $R -- internal/server/routes_admin.go | wc -l        # = 1 (exactly ONE commit touches the serial-slot file)
```

---

## 6. Out of scope (restated, binding)

No new route files / no `routeTree.gen.ts` change (§1.7); no edits to existing
settings/OIDC/auth Go or the store (FROZEN — only the additive `version.go` + the
additive `handlers.go` setters + the additive `server.go`/`main.go` version/shutdown
wiring + serial-slot routes, §1.5/§8 ESC-1); no schema change; no edits to any
frozen w6-a/b/d/e/f file — NOT `sidebar.tsx`/`header.tsx`/`__root.tsx` (the badge is
lit by CALLING the frozen `setUpdateInfo` action from a w6-j hook, §1.6; the modals
mount from the w6-j settings page, §1.7b), NOT `stores/settings.ts` definition, NOT
`use-theme.ts`/`i18n.tsx` (consume only); no header/sidebar exception (SPENT); no
TanStack Query wiring; no dependency additions (no `marked` unless installed, §8
ESC-7); NO new mock-handler file and NO `handlers/index.ts`/seed/store/fixture edit
(settings+version already registered — body-only, §1.9); no other e2e specs (no
separate `version.spec.ts` — `settings.spec.ts` is the carrier, §1.8); no
usage/pricing/providers/keys/combos/chat/governance/mcp/platform pages; no real
outbound network in tests (version latest-check off by default; changelog/donate
mock-route-intercepted; shutdown hook stubbed, NEVER `os.Exit` in the handler,
§1.5b). Mock-vs-Go contradiction → escalate (§8), never patch existing Go or fudge a
mock. The shutdown handler is testable WITHOUT killing the test process (§1.5b).

## 7. Diff-gate scope

Page-wave-2 plans (w6-j/k/l/m) may commit to main concurrently, so a broad
`<base>..HEAD` range sweeps in sibling commits. The diff gate MUST be scoped to
w6-j's own commits. The orchestrator isolates them with:
`git log --oneline main | grep "^[0-9a-f]* phase-1/w6-j:" | awk '{print $1}'`
and runs `git diff <first-w6-j>^..<last-w6-j> -- [file list]` (same commit-range
scoping as w6-f §7 / w6-e §7 / w6-i §7).

`git diff <first-w6-j>^..<last-w6-j> --name-only` must be exactly a subset of:

```
ui/src/routes/settings.tsx
ui/src/components/settings/general-settings-panel.tsx
ui/src/components/settings/general-settings-panel.test.tsx
ui/src/components/settings/language-settings-panel.tsx
ui/src/components/settings/oidc-config-panel.tsx
ui/src/components/settings/password-panel.tsx
ui/src/components/settings/db-info-panel.tsx
ui/src/components/settings/changelog-modal.tsx
ui/src/components/settings/donate-modal.tsx
ui/src/hooks/use-version-check.ts
ui/src/hooks/use-version-check.test.ts
ui/e2e/settings.spec.ts
ui/e2e/mocks/handlers/version.ts          (body only — version DTO + shutdown/changelog/donate routes §1.4)
ui/e2e/mocks/handlers/settings.ts         (body only — password + db-info routes §1.4)
internal/admin/version.go
internal/admin/version_test.go
internal/admin/handlers.go                (additive setters + fields ONLY §1.5/§8 ESC-1)
internal/server/routes_admin.go           (serial-slot additive route lines; ONE commit; FINAL holder)
internal/server/server.go                 (additive version/shutdown wiring §1.5/§8 ESC-1)
cmd/g0router/main.go                       (additive version/shutdown injection §1.5/§8 ESC-1)
.planning/parity/matrix/9router-ui.md
docs/WORKFLOW.md
```

Any file outside this list in the scoped diff is an automatic review REJECT.
`internal/admin/settings.go`, `internal/admin/oidc.go`, `internal/admin/auth.go`,
`internal/store/**`, `internal/schemas/**`, `ui/src/routeTree.gen.ts`,
`ui/src/components/layout/**` (sidebar/header/root), `ui/src/stores/**`,
`ui/src/hooks/use-theme.ts`, `ui/src/providers/i18n.tsx`, `ui/src/i18n/**`,
`ui/e2e/mocks/handlers/index.ts`, `ui/e2e/mocks/{seed,store,fixture}.*`, and any
frozen w6-a/b/d/e/f file are deliberately ABSENT — touching them is an automatic
REJECT. The `routes_admin.go` edit must appear in exactly ONE commit (the §5
`git log … | wc -l` = 1 proof) and the serial slot is RELEASED TO NOBODY on close —
**the wave-6 serial chain CLOSES on w6-j** (§1.8). The `handlers.go`/`server.go`/
`main.go` edits MUST be additive (the version/shutdown wiring §1.5/§8 ESC-1); a
non-additive change there is an ESCALATION, not an in-plan edit. After merge, the
settings page, `ui/src/components/settings/**`, `ui/src/hooks/use-version-check.ts`,
`internal/admin/version.go`, and the corrected version/settings mock bodies become
consume-only for later plans.

## 8. Escalations / cross-track dependencies

- **ESC-1 (RESOLVED at authoring — version/shutdown wiring path, binding)**: the
  binary's `version`/`buildDate` (`main.go:20-21`) and the real shutdown
  (`srv.Close()`, `main.go:144`) are NOT reachable from `internal/admin`. **Decision**:
  add additive `SetVersionInfo`/`SetShutdownFunc` setters on `Handlers` (mirror
  `SetUsageServices`, `handlers.go:43-47`) and a minimal additive forward in
  `server.go`/`main.go`. `New(...)` signature is UNCHANGED. The handler is pure
  (injected deps, nil-safe, response-first async). If the minimal wiring would force
  a non-additive change to `server.go`/`main.go` (beyond a couple of forwarding
  lines), STOP and ESCALATE for the orchestrator to choose the wiring shape — never
  break the frozen `New(...)` signature or edit existing settings/OIDC Go.
- **ESC-2 (RESOLVED — password change mock-only)**: NO Go password-change endpoint
  (`auth.go` has login/me/logout only; no `/api/auth/password` route). **Decision**:
  PAR-UI-100 ships variant-HAVE against a mock route added to the registered
  `settings.ts` handler body. Serial Go follow-up: add a real `POST /api/auth/password`
  (verify current-password, hash, persist). Recorded in `open-questions.md`. —
  Live password change has no real backend until the follow-up.
- **ESC-3 (RESOLVED — DB info mock-only)**: NO Go DB-info endpoint
  (`/api/settings/database` is a guard.go:38 future comment only). **Decision**:
  PAR-UI-101 ships variant-HAVE against a mock route added to the registered
  `settings.ts` handler body. Serial Go follow-up: add a real `GET /api/settings/
  database` (path/size/table counts from the store). Recorded in `open-questions.md`.
- **ESC-4 (RESOLVED — OIDC config persistence + e2e)**: the OIDC panel persists
  `oidc_*` keys via the REAL `PUT /api/settings` (flat map) and TESTS via the REAL
  `POST /api/auth/oidc/test`. **Decision**: the panel SAVES via settings; the "Test"
  button is NOT exercised against a live IdP under e2e (no real IdP) — e2e asserts
  the panel renders + saves only. If the OIDC client secret needs encrypted-at-rest
  storage (the `*_enc` precedent) rather than the flat settings map, that is a serial
  Go follow-up (an `oidc_secret_enc` column) — w6-j stores via the existing settings
  surface and flags the secret-at-rest question. Recorded in `open-questions.md`.
- **ESC-5 (CONDITIONAL — shared mock body)**: if correcting the `version.ts`/
  `settings.ts` bodies breaks a non-w6-j spec, STOP and ESCALATE for orchestrator
  serialization — no fudge, no `index.ts`/seed/store/fixture edit. (Low risk:
  settings/version handlers are consumed only by `settings.spec.ts`.)
- **ESC-6 (CONDITIONAL — global update-check mount)**: w6-j drives the version-check
  from the settings page (no frozen edit, §1.6). If a later decision wants the check
  on every page load (mounting the hook in the frozen root/sidebar), that is an
  orchestrator serial follow-up — NOT a w6-j blocker. The store's `setUpdateInfo`
  action and the sidebar badge are already in place (w6-a).
- **ESC-7 (CONDITIONAL — changelog markdown dep)**: 9router's ChangelogModal uses
  `marked`. **Decision**: w6-j does NOT add `marked` (no dep additions); render the
  changelog as preformatted/sanitized text. If a sanctioned markdown dep is later
  added, upgrade the render — serial follow-up, not a w6-j blocker. Verify at T4
  whether `marked`/a markdown renderer is ALREADY installed; if so it may be
  consumed.
- **Serial-slot dependency (§1.8 — FINAL holder)**: w6-j TAKES the routes_admin.go
  slot (free after w6-f's release to w6-j; `open-questions.md` w6-f line 33,
  routes_admin.go:60-62 merged) and RELEASES it to NOBODY — the wave-6 serial chain
  CLOSES on w6-j. The orchestrator MUST confirm the slot is free at P7 before T3
  (only one unmerged holder, MAP decision 5; w6-j is the last).
- **No other blocking dependency**: w6-a/b/d/e/f merged + the REAL settings/OIDC Go
  in-tree (live tree @ e0fe9b9: 16 primitives, apiFetch/stores/fixture, FROZEN
  settingsStore.setUpdateInfo + sidebar badge, FROZEN useTheme + i18n,
  `GET/PUT /api/settings`, OIDC start/callback/test, the merged w6-f provider-nodes
  block confirming the slot release). w6-j is unblocked for page wave 2.
```
