# Micro-plan w6-c — Auth pages (`/login` + `/callback`)

```
wave: 6
plan: w6-c
status: READY (rev 1 — authored against merged w6-a + w6-b, live tree @ 8bdb85c)
runs: page wave 1, AFTER w6-b MERGE (consumes frozen ui/src/components/ui/*).
  Disjoint from w6-e/w6-g/w6-h/w6-i (different routes/components/specs).
branch: main (direct commits, per repo convention)
commit-prefix: phase-1/w6-c:
ref-source: 9router frozen @ 827e5c3 (login/page.js, callback/page.js)
base: <base> = git rev-parse HEAD recorded at P0 (expected 8bdb85c at authoring;
  if main advanced, record the actual SHA and substitute everywhere §5 says <base>)
freeze-exception: ONE sanctioned edit to frozen header.tsx (fill the empty
  LogoutSlot the MAP/​w6-b reserved "for w6-c") + the matching one-line flip of
  navigation.spec.ts's logout-slot assertion. Bounded in §3/§5. This is the
  single remaining sanctioned header.tsx exception; after w6-c it is SPENT.
```

---

## 1. Scope — PAR rows

### Rows this plan closes

| Row | Claim | Target state after w6-c |
|---|---|---|
| PAR-UI-002 | Route `/login` renders password + OIDC login form | HAVE |
| PAR-UI-003 | Route `/callback` handles OAuth callback via postMessage, BroadcastChannel, localStorage | HAVE |
| PAR-UI-065 | Auth: password login `POST /api/auth/login` + rate-limit retry countdown | HAVE (variant — see §1.2) |
| PAR-UI-066 | Auth: OIDC flow (PKCE/state/nonce) entry from `/login` + provider callback relay | HAVE (variant — see §1.3) |
| PAR-UI-067 | Auth: `GET /api/auth/status` on mount → drives authMode/OIDC visibility | HAVE (variant — see §1.2) |
| PAR-UI-068 | Auth: logout `POST /api/auth/logout` clears session, redirects to `/login` | HAVE |

6 rows across two NEW route files (`login.tsx` rewrite from stub, `callback.tsx`
new), two NEW page-component dirs (`ui/src/components/auth/**`), a new
`ui/src/lib/auth.ts` helper, extended `auth.spec.ts`, corrected
`mocks/handlers/auth.ts`, and ONE sanctioned header.tsx logout-slot fill.
This matches WAVE-6-MAP §Ownership for w6-c plus the logout-slot reservation
(MAP decision 9; w6-b §3 "LogoutSlot untouched … for w6-c").

### 1.1 Preconditions already satisfied by merged waves (evidence)

- `/login` stub exists, must be rewritten: `ui/src/routes/login.tsx:1-9` (renders
  `<h1>Login</h1>` only).
- Frozen primitives this plan CONSUMES (w6-b, never edited):
  `Button` `ui/src/components/ui/button.tsx:69` (`loading`, `variant`,
  `disabled`, `icon` props confirmed); `Input`
  `ui/src/components/ui/input.tsx:12` (`label`/`error`/`hint`/`id`/`type`);
  `Card`/`CardHeader`/`CardTitle`/`CardContent` `ui/src/components/ui/card.tsx:70`;
  `Spinner`/`Loading` `ui/src/components/ui/loading.tsx`.
- Frozen foundation this plan CONSUMES (w6-a, never edited): `apiFetch`
  `ui/src/lib/api.ts:19` + `ApiError` `ui/src/lib/api.ts:3`; `useUserStore`
  (`setUser`/`setToken`/`clear`) `ui/src/stores/user.ts:17`; toast surface via
  `useNotificationStore.push` `ui/src/stores/notification.ts:22` (sonner renders
  `[data-sonner-toast]` through `ui/src/components/layout/toaster.tsx:5`).
- Header logout reservation: `ui/src/components/layout/header.tsx:29-31`
  (`LogoutSlot` = empty `<span data-testid="logout-slot" />`), mounted at
  `header.tsx:77`. w6-b §3 explicitly left it empty "for w6-c".
- Material Symbols available for callback spinner glyphs:
  `ui/src/index.css:3` (`@import "material-symbols/outlined.css"`).
- Existing acceptance contract: `ui/e2e/auth.spec.ts:1-18` (2 tests, currently
  the login helper at `ui/e2e/helpers.ts:3` drives `#username`/`#password`/submit
  → `**/dashboard`). Existing mock: `ui/e2e/mocks/handlers/auth.ts:5-31`
  (status/login/logout already registered); seed
  `ui/e2e/mocks/seed/auth.ts:3` (admin/123456). Mock auto-registered at
  `ui/e2e/mocks/handlers/index.ts:37`.

### 1.2 Real Go contract (file:line evidence — w6-c is UI-ONLY, NO Go changes)

Routes (`internal/server/routes_admin.go`):
- `POST /api/auth/login` → `h.Login` (`routes_admin.go:34`)
- `GET /api/auth/status` → `h.Status` (registered; guard-allowlisted
  `internal/server/guard.go:25`)
- `POST /api/auth/logout` → `h.RequireSession(h.Logout)` (`routes_admin.go:40`)
- `GET /api/auth/oidc/start` → `h.OIDCStart` (`routes_admin.go:35`)
- `GET /api/auth/oidc/callback` → `h.OIDCCallback` (`routes_admin.go:36`)

Body / response shapes (snake_case `{data,error}` envelope):
- **Login** (`internal/admin/auth.go:40-124`): request body `{username, password}`
  (`auth.go:41-44`); both required else 400 (`auth.go:49-52`). Success → 200
  `{data:{token, user:{id, username}}}` (`auth.go:120-123`) + `g0_session`
  cookie. Invalid → 401 `{error:{message:"invalid username or password"}}`
  (`auth.go:94`). Lockout → 429 with
  `{error:{message, retry_after, reset_hint}}` + `Retry-After` header
  (`auth.go:126-140`). **Note**: error envelope is `{error:{message,...}}`;
  `retry_after`/`reset_hint` are siblings of `message` inside `error`.
- **Status** (`internal/admin/auth.go:166-180`): GET → 200
  `{data:{auth_mode}}` where `auth_mode ∈ {"password","oidc","both"}` (default
  `"password"`). **The real status payload exposes only `auth_mode`** — it does
  NOT return `oidc_configured`/`oidc_login_label`/`require_login`/`has_password`
  (the 9router camelCase fields at `login/page.js:40-48` do not exist here).
  See §1.4 for how the UI degrades.
- **Logout** (`internal/admin/auth.go:142-154`): POST (session required) → 200
  `{data:{logged_out:true}}`, clears `g0_session` + OIDC cookies.
- **OIDC** is **server-driven**: `/login` navigates the browser to
  `GET /api/auth/oidc/start` (302 to IdP), and the IdP returns to
  `GET /api/auth/oidc/callback` which sets the session cookie and **302-redirects
  to `/dashboard`** (`internal/admin/oidc.go:243`). PKCE/state/nonce are minted
  and validated entirely in Go (`oidc.go:110-141`, `146-204`). The UI's only OIDC
  responsibility is the login-button navigation; there is NO UI OIDC callback page.

### 1.3 `/callback` is the PROVIDER-OAuth popup relay, NOT the OIDC-login callback

Decision (binding, with evidence): the new `/callback` route ports
`9router/src/app/callback/page.js` — it relays `code`/`state`/`error` from a
provider-authorization popup back to the opener via **postMessage** (origin-
allowlisted), **BroadcastChannel**, and **localStorage**, then auto-closes
(`callback/page.js:44-83`). This is the surface later consumed by w6-e provider
OAuth modals (MAP w6-e "OAuth popup contract", depends on w6-c). It is distinct
from OIDC *login*, which never reaches a UI page (the Go handler 302s straight to
`/dashboard`, §1.2). PAR-UI-003 names exactly this postMessage/BroadcastChannel/
localStorage behavior — so the row maps to this route, not to OIDC login.

### 1.4 Variant notes (recorded HAVE rationale)

- **PAR-UI-067 / status-driven UI**: the real `GET /api/auth/status` returns only
  `auth_mode`. The login page therefore derives visibility from `auth_mode`:
  `password`→password form only; `oidc`→OIDC button primary (+ a password
  recovery affordance, since Go still permits password unless OIDC is configured,
  `auth.go:81-84`); `both`→both. The 9router `oidc_configured`/`oidc_login_label`
  fields are absent server-side; the UI uses a static label ("Sign in with OIDC")
  and shows the OIDC button whenever `auth_mode ∈ {oidc, both}`. Recorded as
  variant-HAVE: the parity behavior (status-on-mount gating which auth methods
  render) is delivered against the *real* g0router contract.
- **PAR-UI-065 / rate-limit countdown**: on a 429 the page reads `retry_after`
  from the error envelope, disables the submit button, and renders a 1-second
  `setInterval` countdown ("Wait {n}s") that re-enables at 0 — porting
  `login/page.js:20-24,158-162,170-178`. Variant: error fields live under
  `error.retry_after` (g0router envelope), not top-level `retryAfter` (9router).
- **PAR-UI-066 / OIDC**: variant per §1.3 — UI entry is the start-navigation; the
  `/callback` page covers the provider-OAuth relay half; OIDC-login token exchange
  is fully Go-side. No UI PKCE/JWKS code (correctly server-owned).
- **PAR-UI-002 login form lives in the app chrome**: `__root.tsx:19-39` wraps
  *every* route (incl. `/login`) in Sidebar+Header. This plan does NOT change root
  (frozen). The login form renders inside `<Outlet>`; auth.spec only asserts the
  form fields, the error toast, and the `/dashboard` redirect, all of which hold
  with the chrome present. Recorded as an accepted constraint, not a gap.

### NOT in scope (explicit)

- **No Go changes.** All of `internal/` is FORBIDDEN; the auth/OIDC handlers
  already exist from W3 (§1.2 evidence). If a real handler's body/response shape
  contradicts this plan's mock, that is an ESCALATION to the orchestrator
  (§4 T1 note), never an in-plan Go edit or mock fudge.
- **No edits to frozen w6-a/w6-b files** except the single sanctioned
  `header.tsx` logout-slot fill (§3) + its `navigation.spec.ts` assertion flip.
  Specifically NOT: `__root.tsx`, `sidebar.tsx`, `mobile-sidebar.tsx`,
  `toaster.tsx`, any `ui/src/components/ui/*`, any `ui/src/stores/*`,
  `ui/src/lib/api.ts`, `ui/src/lib/utils.ts`, `ui/src/providers/*`.
- **No new dependencies.** Every import resolves to already-installed packages
  (react, @tanstack/react-router, sonner, lucide-react) or w6-a/w6-b outputs.
- **No OIDC PKCE/JWKS/token-exchange code in the UI** — server-owned (§1.3).
- **No real OAuth-provider integration** in `/callback` — it is a pure relay;
  the providers that open the popup are w6-e's concern.
- **No changes to `mocks/handlers/index.ts` / `mocks/seed/index.ts`** — the auth
  handler is already registered (`index.ts:37`) and the user seed already
  exported (`seed/index.ts:1`). w6-c edits the auth handler *body* only.
- **No other e2e spec files** beyond `auth.spec.ts` (+ the single sanctioned
  `navigation.spec.ts` logout-slot line).
- **No `routeTree.gen.ts` hand-edits** — it regenerates from the new route
  files via the Vite plugin (MAP decision 6); the build performs the regen.

---

## 2. Precondition checks

Run all before any edit; abort and report to orchestrator on any failure.

```bash
# P0 — clean tree; record <base>
git status --porcelain     # must be empty (untracked tooling artifacts must be
                           # gitignored; worker uses explicit `git add <file>`,
                           # never `git add -A`, per w6-b runtime disposition)
git rev-parse HEAD         # record as <base> for §5 (expected 8bdb85c)

# P1 — w6-b primitives present and frozen (consumed by this plan)
grep -n "export interface ButtonProps" ui/src/components/ui/button.tsx   # loading/icon props
grep -n "loading" ui/src/components/ui/button.tsx
grep -n "label?:" ui/src/components/ui/input.tsx
grep -n "export { Card" ui/src/components/ui/card.tsx
ls ui/src/components/ui/*.tsx | grep -v test | wc -l    # = 16 (w6-b set intact)

# P2 — w6-a foundation present and frozen (consumed by this plan)
grep -n "export async function apiFetch" ui/src/lib/api.ts
grep -n "export class ApiError" ui/src/lib/api.ts
grep -n "useUserStore" ui/src/stores/user.ts
grep -n "push:" ui/src/stores/notification.ts
grep -n "data-sonner-toast\|Toaster" ui/src/components/layout/toaster.tsx

# P3 — header logout reservation is still empty (the sanctioned-exception target)
grep -n 'data-testid="logout-slot"' ui/src/components/layout/header.tsx   # the empty span
grep -n "LogoutSlot" ui/src/components/layout/header.tsx
# navigation.spec currently asserts the slot is empty — w6-c flips exactly this:
grep -n 'logout-slot' ui/e2e/navigation.spec.ts                          # lines 51-53

# P4 — login route is still the bare stub (safe to rewrite)
grep -n "<h1>Login</h1>" ui/src/routes/login.tsx
test ! -e ui/src/routes/callback.tsx && echo "callback absent (good)"
test ! -d ui/src/components/auth && echo "auth components dir absent (good)"
test ! -e ui/src/lib/auth.ts && echo "lib/auth absent (good)"

# P5 — existing auth mock + seed match the real Go contract (§1.2)
grep -n "/api/auth/login\|/api/auth/status\|/api/auth/logout" ui/e2e/mocks/handlers/auth.ts
grep -n "registerAuthHandlers" ui/e2e/mocks/handlers/index.ts            # already wired
grep -n "username: \"admin\"" ui/e2e/mocks/seed/auth.ts                  # admin/123456 seed

# P6 — e2e + unit harness green at base
cd ui && npx playwright test e2e/auth.spec.ts        # 2/2 PASS at base?  (see note)
cd ui && npx playwright test e2e/navigation.spec.ts  # 9/9 green at base
cd ui && npm run build                               # exit 0
go test ./... && go vet ./...                        # exit 0 (Go untouched-green)
```

**P6 note on auth.spec baseline**: auth.spec.ts already exists with 2 tests.
Determine its base state and record it:
- If both PASS at base (the stub `/login` plus existing mock somehow satisfy the
  helper — unlikely, the stub has no `#username`), the arc is "extend the spec
  RED with the new assertions, then implement". 
- If they FAIL at base (expected — stub renders only `<h1>`, no `#username`), the
  spec is already the red contract for the existing two tests, and T1 only ADDS
  the new RED assertions. Either way: **every new assertion is committed RED
  before the implementation that greens it** (strict TDD, §4). Record the
  observed base result in WORKFLOW.md at closeout.

---

## 3. Exclusive file ownership

After w6-c merges, all CREATE files below are owned by w6-c; later plans consume
the `/callback` relay contract but do not edit these files (MAP decision 7).

**CREATE — routes (`ui/src/routes/`):**

| File | Exports / contract |
|---|---|
| `login.tsx` (REWRITE from stub) | `Route = createFileRoute("/login")`; `LoginPage` component: on mount calls `getAuthStatus()` (`lib/auth.ts`) → sets `authMode`; renders inside a `Card`; password `<Input id="username">` + `<Input id="password" type="password">` (the e2e helper targets `#username`/`#password`, `helpers.ts:3-9`); submit `<Button type="submit" loading variant="primary">`; OIDC `<Button>` (navigates `window.location.href="/api/auth/oidc/start"`) shown when `authMode ∈ {oidc,both}`; on 401 → push error toast (notification store) "invalid username or password"; on 429 → read `error.retry_after`, start 1s countdown, disable submit, label "Wait {n}s"; on success → `setUser`/`setToken` from `{data:{token,user}}`, navigate to `/dashboard`. Visibility logic per §1.4. |
| `callback.tsx` (NEW) | `Route = createFileRoute("/callback")`; `CallbackPage`: reads `code`/`state`/`error`/`error_description` from `window.location.search`; relays via the three methods in `lib/auth.ts` `relayOAuthCallback(...)` (postMessage to origin-allowlist `[window.location.origin, "http://localhost:1455"]`, BroadcastChannel `"oauth_callback"`, localStorage `"oauth_callback"`); status state machine `processing → success → done` (auto `window.close()` after 1.5s) or `manual` when neither code nor error present; renders Material-Symbols spinner + status text (ports `callback/page.js:9-148`, adapted to TanStack — no Next `Suspense`/`useSearchParams`). |

**CREATE — page components (`ui/src/components/auth/`):**

| File | Exports / contract |
|---|---|
| `login-form.tsx` | `LoginForm` — the password+OIDC form body (consumes `Button`/`Input`/`Card`); props for `authMode`, `loading`, `error`, `retryAfter`, `onSubmit(username,password)`, `onOidc()`. Pure-ish presentational; keeps `login.tsx` route thin. |
| `logout-button.tsx` | `LogoutButton` — `<Button variant="ghost" size="icon">` (lucide `LogOut` icon, `aria-label="Log out"`, `data-testid="logout-button"`); onClick → `logout()` (`lib/auth.ts`) → `useUserStore.clear()` → navigate `/login`. This is the component mounted into the header's `LogoutSlot` (§ sanctioned exception). Lives in w6-c-owned dir so the header edit is a 2-line import+mount only. |

**CREATE — lib (`ui/src/lib/auth.ts`, NEW file — NOT w6-a's frozen `api.ts`):**

| Export | Contract |
|---|---|
| `getAuthStatus(): Promise<{auth_mode: "password"\|"oidc"\|"both"}>` | `apiFetch("/api/auth/status")`; on error defaults to `{auth_mode:"password"}` (graceful, mirrors `login/page.js:50-56`). |
| `loginWithPassword(username, password): Promise<{token, user}>` | `apiFetch("/api/auth/login", {method:"POST", body: JSON.stringify({username, password})})`; throws `ApiError` (carries `status` for 401/429 branching; the 429 `retry_after`/`reset_hint` are surfaced via a typed extension — read from the thrown `ApiError` or a thin re-fetch helper; the implementer extends `ApiError` consumption WITHOUT editing `api.ts`). |
| `logout(): Promise<void>` | `apiFetch("/api/auth/logout", {method:"POST"})`; ignores benign errors. |
| `startOidc(): void` | `window.location.href = "/api/auth/oidc/start"`. |
| `relayOAuthCallback(data): void` | postMessage (origin allowlist) + BroadcastChannel(`"oauth_callback"`) + localStorage(`"oauth_callback"`), per §1.3. |

Note on 429 fields: the frozen `apiFetch` throws `ApiError(message, status, code)`
and does not expose sibling envelope fields. To read `retry_after` WITHOUT editing
`api.ts`, `loginWithPassword` performs the POST and, on a non-OK response, parses
the `{error:{message,retry_after,reset_hint}}` envelope itself (a local fetch in
`lib/auth.ts`, not a change to `api.ts`). This keeps `api.ts` frozen and still
yields the countdown data. Document this choice inline.

**CREATE — tests (unit, vitest — for the pure helpers only):**

| File | Contents |
|---|---|
| `ui/src/lib/auth.test.ts` | ≥4 tests for the pure/branching logic reachable without a DOM: `getAuthStatus` returns parsed `auth_mode`; `getAuthStatus` defaults to `password` on fetch error; `relayOAuthCallback` posts to BroadcastChannel + writes localStorage (stub `BroadcastChannel`/`localStorage` as w6-a's `theme.test.ts` hand-stubs globals — no jsdom); `startOidc` sets `window.location.href`. Committed RED before `lib/auth.ts` exists (strict TDD). |

**MODIFY — extend existing e2e spec (the acceptance contract, owned jointly with
w6-a's authorship but extended by w6-c per MAP §Ownership "auth.spec.ts"):**

| File | Change |
|---|---|
| `ui/e2e/auth.spec.ts` | KEEP the 2 existing tests. ADD the new RED tests in §4 T1 (status-driven OIDC button, rate-limit countdown, logout flow, callback relay). |
| `ui/e2e/mocks/handlers/auth.ts` | CORRECTIONS ONLY to mirror the real Go contract (§1.2): (a) login success returns `{data:{token, user:{id, username}}}` not `{token}` (current `auth.ts:19` returns bare `{token}` — wrap to match `auth.go:120-123`); (b) status returns `{data:{auth_mode}}` (current `auth.ts:7` returns the whole `store.auth` object — narrow to `{auth_mode}` per `auth.go:177-179`), with a store-settable `auth_mode` so the OIDC test can drive `both`; (c) add a rate-limit branch returning 429 `{error:{message,retry_after,reset_hint}}` after N failed attempts (or a dedicated header/flag the spec toggles) to exercise the countdown. Mocks mirror reality (MAP decision 4); if reality differs, escalate (§4 T1). |

**SANCTIONED FREEZE EXCEPTION — header logout-slot fill (one commit, §4 T5):**

| File | Change (and ONLY this change) |
|---|---|
| `ui/src/components/layout/header.tsx` | `import { LogoutButton } from "@/components/auth/logout-button"` + `LogoutSlot` body: `<span data-testid="logout-slot" />` → `<span data-testid="logout-slot">{user ? <LogoutButton /> : null}</span>`. Nothing else. Diff bound §5: ≤6 added lines. (Logout only shown when a user is present, preserving the navigation.spec logged-out expectation — see below.) |
| `ui/e2e/navigation.spec.ts` | logout-slot assertion ONLY (lines 51-53): keep `toBeAttached()`; the `toHaveText("")` stays valid IF navigation.spec runs logged-out (no `user` in store → `LogoutButton` not rendered → slot empty). **VERIFY** navigation.spec's auth state at run time: if it runs logged-out, NO change is needed and the freeze exception touches header.tsx ONLY. If navigation.spec runs with a user mounted, flip the empty-text assertion to "attached + contains a `[data-testid=\"logout-button\"]`". Resolve with the observed run result; prefer the zero-change path. Diff bound §5. |

After this commit, `header.tsx` is FROZEN for good — this is the last sanctioned
exception WAVE-6-MAP reserved.

**FORBIDDEN:** everything else. Explicitly: all of `internal/` (Go auth exists,
§1.2); `__root.tsx`, `sidebar.tsx`, `mobile-sidebar.tsx`, `toaster.tsx`; all
`ui/src/components/ui/*` (w6-b frozen); all `ui/src/stores/*`,
`ui/src/lib/api.ts`, `ui/src/lib/utils.ts`, `ui/src/providers/*` (w6-a frozen);
`ui/package.json` + lockfile; `ui/vite.config.ts`; `ui/playwright.config.ts`;
`ui/components.json`; `ui/src/index.css`; `ui/e2e/mocks/handlers/index.ts` and
`mocks/seed/index.ts` (already wired — body of `auth.ts` handler only);
all other `ui/e2e/*.spec.ts` except auth.spec.ts and the single
navigation.spec.ts logout-slot line; `ui/src/routeTree.gen.ts` (generated only).

---

## 4. TDD tasks

Cadence (strict): **no route/component/lib file may exist in the tree before the
failing test that covers it is committed.** `cd ui && npm run build` green at
**every** commit (test files and red specs are never imported by production code,
so the bundle stays buildable — same rationale as w6-b §4). The plan's e2e spec
stays RED from T1 until the implementation tasks green it; that is the arc.

### T1 — STEP(a): extend `auth.spec.ts` + correct the auth mock (commit RED)

Add these tests to `ui/e2e/auth.spec.ts` (names are the acceptance contract, §5):

1. `login with valid credentials redirects to dashboard` — **already exists**
   (`auth.spec.ts:5-8` via `login()` helper). Keep; it greens at T3.
2. `invalid credentials show error toast` — **already exists**
   (`auth.spec.ts:10-17`). Keep; greens at T3.
3. `login page shows OIDC button when auth_mode is "both"` — set the mock
   `auth_mode="both"` (via the store-settable field added to the handler); goto
   `/login`; assert an OIDC sign-in button is visible AND `#password` is visible.
4. `login page hides password form when auth_mode is "oidc"` (variant per §1.4:
   shows OIDC button primary; password recovery affordance may still render) —
   assert OIDC button visible; assert the primary action is OIDC.
5. `rate-limit returns a retry countdown and disables submit` — drive the mock to
   return 429 `{error:{message,retry_after:30,reset_hint}}`; submit; assert a
   "Wait" / countdown text appears AND `button[type="submit"]` is `disabled`.
6. `logout from header clears session and returns to /login` — `login()` first;
   on `/dashboard` assert `[data-testid="logout-button"]` visible; click;
   assert URL → `/login` (and a subsequent protected nav stays at `/login`).
7. `callback relays code+state via BroadcastChannel` — open a listener page on
   the same origin subscribing to `BroadcastChannel("oauth_callback")`; navigate a
   second context to `/callback?code=abc&state=xyz`; assert the listener receives
   `{code:"abc", state:"xyz"}`. (postMessage path needs a real opener; assert the
   localStorage fallback too: `/callback?code=abc` writes
   `localStorage["oauth_callback"]` containing `code:"abc"`.)
8. `callback shows manual-copy state when no code/error present` — goto
   `/callback` (no query); assert the "Copy This URL" manual state renders.

STEP(b): `cd ui && npx playwright test e2e/auth.spec.ts` — **record the failure
output** (new tests red: no OIDC button, no countdown, no logout button, no
`/callback` route). Commit RED:
`phase-1/w6-c: failing auth e2e (OIDC/rate-limit/logout/callback) + mock corrections (TDD red)`.

**Mock-vs-reality gate**: while correcting `mocks/handlers/auth.ts`, re-read the
Go handlers (§1.2 file:lines). If any real shape contradicts the corrected mock
(e.g. login success envelope, status fields, 429 envelope), **STOP and ESCALATE**
to the orchestrator — w6-c does not adjudicate spec mocks against Go and makes NO
Go change (MAP decision 4 + decision 5 boundary).

### T2 — STEP(a): `ui/src/lib/auth.test.ts` (commit RED)

Write the ≥4 unit tests per §3 (status parse, status default-on-error,
relayOAuthCallback BroadcastChannel+localStorage, startOidc). Stub
`BroadcastChannel`/`localStorage`/`window.location`/`fetch` in-test (w6-a
`theme.test.ts` precedent — no jsdom). `cd ui && npx vitest run src/lib/auth.test.ts`
→ FAILS (module missing). **Record failure.** Commit RED:
`phase-1/w6-c: failing unit tests for lib/auth (TDD red)`.

### T3 — `lib/auth.ts` + `login.tsx` + `components/auth/login-form.tsx`

STEP(b): implement `lib/auth.ts` per §3 (helpers green the T2 units); implement
`login-form.tsx` and rewrite `login.tsx` per §3. Gates:
`npx vitest run src/lib/auth.test.ts` green; auth.spec tests 1-5 green
(login, error toast, OIDC visibility, countdown) — tests 6-8 (logout/callback)
STILL red. `npm run build` green. Commit:
`phase-1/w6-c: login page (password + OIDC + rate-limit countdown), lib/auth`.

### T4 — `callback.tsx`

STEP(b): implement `callback.tsx` per §3 (consumes `relayOAuthCallback`). The
Vite plugin regenerates `routeTree.gen.ts` on build (do NOT hand-edit it; if it
appears dirty after build, that is the expected generated output — commit it as
part of this task). Gates: auth.spec tests 7-8 (callback relay + manual state)
green; `npm run build` green. Commit:
`phase-1/w6-c: /callback OAuth relay route (postMessage + BroadcastChannel + localStorage)`.

### T5 — `components/auth/logout-button.tsx` + sanctioned header logout-slot fill

STEP(b): implement `logout-button.tsx` per §3. Then make the SINGLE sanctioned
`header.tsx` edit (import + mount into the empty `LogoutSlot`), guarded so logout
only renders when a `user` is present. Re-run `navigation.spec.ts`:
- If 9/9 still green (slot empty when logged-out), the freeze exception touches
  header.tsx ONLY — make NO navigation.spec change.
- If navigation.spec mounts a user and goes red on the empty-text assertion, flip
  ONLY that assertion (lines 51-53) to "attached + contains
  `[data-testid=\"logout-button\"]`", atomically in this same commit.
Gates: auth.spec test 6 (logout) green; `npx playwright test e2e/navigation.spec.ts`
9/9 green; `npm run build` green; §5 added-line checks pass. Commit:
`phase-1/w6-c: logout button wired into header LogoutSlot (sanctioned freeze exception)`.
**After this commit header.tsx is FROZEN for good.**

### T6 — full gates + closeout

```bash
cd ui && npm run build
cd ui && npx playwright test e2e/auth.spec.ts            # all green (8 tests)
cd ui && npx playwright test e2e/navigation.spec.ts      # 9/9 green
cd ui && npx playwright test                             # full suite: no spec
                                                         # green-at-base may be red
cd ui && npx vitest run src/                             # all green incl new auth.test
go test ./... && go vet ./...                            # untouched-green
```
Flip §1 matrix rows in `.planning/parity/matrix/9router-ui.md`: PAR-UI-002,
PAR-UI-003 → HAVE; PAR-UI-065, PAR-UI-066, PAR-UI-067 → HAVE (variant, cite
§1.4); PAR-UI-068 → HAVE. Update `docs/WORKFLOW.md` (record the P6 auth.spec base
observation and the T5 navigation.spec resolution). Final commit:
`phase-1/w6-c: close — auth pages; matrix flips`.

---

## 5. Binary acceptance criteria

All must hold; each is yes/no. `<base>` = the commit recorded at P0 (8bdb85c at
authoring). Diff gate is **w6-c commit-range-scoped** (§7) because page wave 1
plans commit to main concurrently.

**Test gates**
- `cd ui && npx playwright test e2e/auth.spec.ts` → exit 0, all tests pass
  (the 2 original + the 6 added = 8), 0 skipped.
- `cd ui && npx playwright test e2e/navigation.spec.ts` → exit 0, 9/9.
- `cd ui && npx vitest run src/lib/auth.test.ts` → exit 0, ≥4 passed.
- `cd ui && npx vitest run src/` → exit 0 (all prior unit suites still green).
- `cd ui && npm run build` → exit 0.
- `go test ./... && go vet ./...` → exit 0 (Go untouched).

**TDD-order proof** — each implementation file's covering test appears in an
earlier-or-equal commit:
```bash
# lib/auth.ts after lib/auth.test.ts
ct=$(git log --format=%ct --diff-filter=A -1 -- ui/src/lib/auth.test.ts)
cf=$(git log --format=%ct --diff-filter=A -1 -- ui/src/lib/auth.ts)
[ "$ct" -le "$cf" ] || echo "TDD VIOLATION: lib/auth.ts"          # prints nothing
# auth.spec RED-extension commit precedes login.tsx/callback.tsx implementations
sa=$(git log --format=%ct -1 --grep="failing auth e2e" )
li=$(git log --format=%ct --diff-filter=M -1 -- ui/src/routes/login.tsx)
[ "$sa" -le "$li" ] || echo "TDD VIOLATION: login.tsx before red spec"  # nothing
```

**Grep proofs**
```bash
test -f ui/src/routes/callback.tsx && echo OK                           # PAR-UI-003
grep -n '#username\|id="username"\|"username"' ui/src/routes/login.tsx ui/src/components/auth/login-form.tsx  # PAR-UI-002 form fields
grep -rn "/api/auth/login" ui/src/lib/auth.ts                           # PAR-UI-065
grep -rn "retry_after\|Wait\|setInterval" ui/src/routes/login.tsx ui/src/components/auth/login-form.tsx  # PAR-UI-065 countdown
grep -rn "/api/auth/oidc/start" ui/src/lib/auth.ts ui/src/routes/login.tsx ui/src/components/auth/login-form.tsx  # PAR-UI-066 OIDC entry
grep -rn "/api/auth/status\|auth_mode" ui/src/lib/auth.ts ui/src/routes/login.tsx  # PAR-UI-067
grep -rn "postMessage\|BroadcastChannel\|oauth_callback" ui/src/lib/auth.ts ui/src/routes/callback.tsx  # PAR-UI-003
grep -n "localhost:1455\|window.location.origin" ui/src/lib/auth.ts     # PAR-UI-003 origin allowlist
grep -rn "/api/auth/logout" ui/src/lib/auth.ts ui/src/components/auth/logout-button.tsx  # PAR-UI-068
grep -n 'data-testid="logout-button"' ui/src/components/auth/logout-button.tsx          # PAR-UI-068
grep -n "LogoutButton" ui/src/components/layout/header.tsx              # PAR-UI-068 wiring
```

**Negative / freeze proofs (w6-c commit-range — see §7)**
```bash
# Range = <first-w6-c>^..<last-w6-c> (excludes other page-wave-1 commits on main)
R="<first-w6-c>^..<last-w6-c>"
git diff $R --name-only -- internal/ | wc -l                            # = 0 (no Go)
git diff $R --name-only -- ui/package.json ui/package-lock.json ui/vite.config.ts ui/playwright.config.ts ui/components.json ui/src/index.css | wc -l   # = 0
git diff $R --name-only -- ui/src/components/ui/ | wc -l                # = 0 (w6-b frozen)
git diff $R --name-only -- ui/src/stores/ ui/src/providers/ ui/src/lib/api.ts ui/src/lib/utils.ts | wc -l   # = 0 (w6-a frozen)
git diff $R --name-only -- ui/src/routes/__root.tsx ui/src/components/layout/sidebar.tsx ui/src/components/layout/mobile-sidebar.tsx ui/src/components/layout/toaster.tsx | wc -l   # = 0
git diff $R --name-only -- 'ui/src/routes/' | grep -vE 'login\.tsx|callback\.tsx' | wc -l   # = 0 (only the two auth routes)
git diff $R --name-only -- ui/e2e/ | grep -vE 'auth\.spec\.ts|navigation\.spec\.ts' | wc -l # = 0 (mocks: only auth.ts handler — see next)
git diff $R --name-only -- ui/e2e/mocks/ | grep -vE 'handlers/auth\.ts' | wc -l            # = 0 (mock index/seed untouched)
git diff $R -- ui/src/components/layout/header.tsx | grep "^+" | wc -l  # ≤ 6 (import + slot fill; incl +++ header line)
git log --oneline $R -- ui/src/components/layout/header.tsx | wc -l     # = 1 (exactly ONE commit touches the frozen header)
git diff $R -- ui/e2e/navigation.spec.ts | grep "^+" | wc -l           # ≤ 5 (logout-slot line only, or 0 if no change needed)
grep -rn "internal/\|package.json" /dev/null; true                     # sanity placeholder
```

---

## 6. Out of scope (restated, binding)

No Go changes (auth/OIDC exist from W3, §1.2); no UI OIDC PKCE/JWKS/token-exchange
(server-owned, §1.3); no real provider OAuth in `/callback` (pure relay, w6-e
consumes it); no edits to any w6-a/w6-b frozen file except the single sanctioned
header logout-slot fill; no dependency additions; no playwright/vite/components.json
/index.css changes; no mocks index/seed edits (auth handler body only); no other
e2e specs beyond auth.spec.ts (+ the single navigation.spec.ts logout-slot line);
no `routeTree.gen.ts` hand-edit (regenerated by build). Mock-vs-Go contradiction →
escalate, never patch Go or fudge the mock (§4 T1).

## 7. Diff-gate scope

Page-wave-1 plans (w6-c/e/g/h/i) commit to main concurrently, so a broad
`<base>..HEAD` range sweeps in sibling commits. The diff gate MUST be scoped to
w6-c's own commits. The orchestrator isolates them with:
`git log --oneline main | grep "^[0-9a-f]* phase-1/w6-c:" | awk '{print $1}'`
and runs `git diff <first-w6-c>^..<last-w6-c> -- [file list]` (same commit-range
scoping as w6-b §7 / w5-f split gate).

`git diff <first-w6-c>^..<last-w6-c> --name-only` must be exactly a subset of:

```
ui/src/routes/login.tsx
ui/src/routes/callback.tsx
ui/src/components/auth/login-form.tsx
ui/src/components/auth/logout-button.tsx
ui/src/lib/auth.ts
ui/src/lib/auth.test.ts
ui/e2e/auth.spec.ts
ui/e2e/mocks/handlers/auth.ts
ui/src/routeTree.gen.ts                  (generated by build; route additions)
ui/src/components/layout/header.tsx      (sanctioned logout-slot fill; ONE commit)
ui/e2e/navigation.spec.ts                (logout-slot assertion only, IF needed)
.planning/parity/matrix/9router-ui.md
docs/WORKFLOW.md
```

Any file outside this list in the scoped diff is an automatic review REJECT.
The header.tsx + navigation.spec.ts edits must appear in exactly the ONE
sanctioned commit (the §5 `git log … header.tsx | wc -l` = 1 proof). After
merge, `header.tsx` is frozen for good and the last Wave-6 sanctioned exception
is SPENT; `/login`, `/callback`, `ui/src/components/auth/**`, and `ui/src/lib/auth.ts`
become consume-only for later plans (w6-e consumes the `/callback` relay).

## 8. Escalations / cross-track dependencies

- **None blocking at authoring.** w6-a + w6-b are merged (live tree @ 8bdb85c:
  16 primitives present, header logout-slot reserved-empty, auth mock+seed wired,
  Go auth/OIDC handlers in-tree per §1.2). w6-c is fully unblocked for page wave 1.
- **Conditional escalation (T1)**: if a corrected mock shape contradicts the real
  Go handler, escalate — no Go edit, no mock fudge.
- **Conditional resolution (T5)**: navigation.spec logout-slot — prefer the
  zero-change path (logout hidden when logged-out); only flip the assertion if the
  spec mounts a user. Recorded, not a blocker.
```
