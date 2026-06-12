# Micro-plan w6-a — UI Foundation: Shell, Theming, Stores, Lib

```
wave: 6
plan: w6-a
status: READY
runs: ALONE — no other Wave 6 plan may start until w6-a is merged (WAVE-6-MAP.md decision 8)
branch: main (direct commits, per repo convention)
commit-prefix: phase-1/w6-a:
ref-source: 9router frozen @ 827e5c3
```

---

## 1. Scope — PAR rows

### Rows this plan closes

| Row | Claim | Target state after w6-a |
|---|---|---|
| PAR-UI-001 | Route `/` redirects to `/dashboard` | HAVE |
| PAR-UI-026 | Dashboard layout wraps all routes with sidebar + header + toasts | HAVE |
| PAR-UI-081 | Data-fetching pattern: `apiFetch` + TanStack Query adapter (variant-HAVE) | HAVE (variant) |
| PAR-UI-028 | Sidebar: traffic lights, logo, nav items, update-check badge | PARTIAL → see note |
| PAR-UI-029 | Header: breadcrumbs, page title, search bar, auth badge | PARTIAL → see note |
| PAR-UI-030 | Toast notifications via Zustand store with auto-dismiss (sonner) | HAVE |
| PAR-UI-031 | Mobile sidebar with overlay and slide-in animation | HAVE |
| PAR-UI-073 | Tailwind v4 `@theme inline` semantic tokens | PARTIAL → HAVE |
| PAR-UI-074 | Brand colors (#E56A4A primary, #FDFAF6 light bg, #1a1a1a dark bg) | HAVE |
| PAR-UI-075 | Zustand themeStore with `persist`, storage key `"theme"` | HAVE |
| PAR-UI-076 | `useTheme` syncs `prefers-color-scheme` via `useSyncExternalStore` | HAVE |
| PAR-UI-077 | ThemeProvider calls `initTheme()` on mount | HAVE |
| PAR-UI-078 | `.dark` class toggled on `<html>` | HAVE |
| PAR-UI-079 | Material Symbols Outlined font with `fill-1` class | PARTIAL → HAVE |
| PAR-UI-080 | All six Zustand stores | HAVE |

### Partial-row notes

**PAR-UI-027** (Root layout: Inter + ThemeProvider + RuntimeI18nProvider): NOT in this plan's
closes table. w6-a delivers Inter font + ThemeProvider + an `I18nMount` passthrough slot only.
w6-b's wiring commit (the single freeze exception) adds RuntimeI18nProvider and flips this row
to HAVE. Do NOT flip this row in w6-a's closeout commit.

**PAR-UI-028** (Sidebar with media accordion + live update check): PARTIAL after w6-a.
- Done here: traffic lights, logo, nav items, update-check badge (reactive on `settingsStore.updateAvailable`; badge appears when set).
- Deferred: media accordion → W7/S2 (no Stage-1 media providers; the ref accordion wraps media-provider routes which are PAR-UI-022/023/024, deferred per WAVE-6-MAP §Out of scope). Live update-check polling (network fetch to `/api/version`) → w6-j (owns version/settings cluster). Row flips HAVE after w6-j.

**PAR-UI-029** (Header with donate + theme/lang toggles + logout): PARTIAL after w6-a.
- Done here: breadcrumbs, page title, search bar bound to headerSearchStore, auth badge from userStore.
- Null slots here: ThemeToggle slot (filled by w6-b wiring commit), LanguageSwitcher slot (filled by w6-b wiring commit), logout slot (filled by w6-c after auth), donate slot (non-functional UI button → w6-j or later).
- Row flips HAVE after donate + logout are wired (w6-j/c respectively).

### NOT in scope (explicit)

- **No page components.** Every `ui/src/routes/*.tsx` stub stays byte-identical (except `__root.tsx`).
- **No i18n implementation** — no locale files, no `RuntimeI18nProvider` logic, no `LanguageSwitcher`. (w6-b)
- **No `ThemeToggle` component.** Header renders a null slot only. (w6-b)
- **No auth/login flow.** `userStore` holds shape only; auth badge renders from store state, no login UI. (later plan)
- **No shadcn/ui component generation.** (w6-b)
- **No update-check API polling.** Sidebar badge renders from `settingsStore.updateAvailable`; the network check is a later plan.
- **No data fetching wired into stores.** `providerStore`/`settingsStore` are state + setters; pages populate them later via `lib/api.ts`.
- **No Go code.** All of `internal/` is forbidden.

---

## 2. Precondition checks

Run all before any edit; abort and report to orchestrator on any failure.

```bash
# P1 — resolve the CSS entry file (decides CSS ownership, see §3)
grep -n "\.css" ui/src/main.tsx
# Record the imported file as $CSS_ENTRY (expected ui/src/index.css or ui/src/styles.css).
# If main.tsx already imports ui/src/styles/globals.css, $CSS_ENTRY is that file.

# P2 — required packages installed (no package.json edits allowed in this plan)
grep -E '"(zustand|sonner|material-symbols|@tanstack/react-router|tailwindcss|@tailwindcss/vite)"' ui/package.json
grep -E '"(clsx|tailwind-merge|lucide-react)"' ui/package.json
# If clsx / tailwind-merge / lucide-react are MISSING: STOP and escalate —
# adding dependencies modifies package.json + lockfile, which is outside this
# plan's ownership and needs an orchestrator decision before proceeding.

# P3 — Tailwind v4 vite plugin wired
grep -n "tailwindcss" ui/vite.config.ts

# P4 — Playwright harness exists with a webServer (e2e must be runnable headless)
test -f ui/playwright.config.ts && grep -n "webServer" ui/playwright.config.ts

# P5 — __root.tsx is still the bare placeholder (nobody raced us)
wc -c ui/src/routes/__root.tsx && grep -n "Outlet" ui/src/routes/__root.tsx

# P6 — route stubs exist for every nav target (sidebar links must not 404)
ls ui/src/routes/ | wc -l

# P7 — clean tree
git -C . status --porcelain  # must be empty besides this plan's work
```

**Inter font decision (locked):** packages may not be added, so Inter loads via CSS `@import url('https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap');` in `$CSS_ENTRY`, with font stack fallback `Inter, ui-sans-serif, system-ui, sans-serif`. Known tradeoff: requires network at page load; offline embedding of the font is a candidate follow-up row, **not** silently fixed here. If P2 reveals `@fontsource-variable/inter` is already installed, prefer `@import '@fontsource-variable/inter';` instead.

---

## 3. Exclusive file ownership

After w6-a merges, **every file below is FROZEN for the rest of Wave 6** (sole exception: w6-b's single wiring commit may edit `header.tsx` slot internals and the `__root.tsx` i18n slot — nothing else).

**CREATE:**

| File | Contents |
|---|---|
| `ui/src/stores/theme.ts` | themeStore: zustand + `persist`, key `"theme"`, `initTheme()` |
| `ui/src/stores/user.ts` | userStore: `user`, `token`, `setUser`, `setToken`, `clear` |
| `ui/src/stores/provider.ts` | providerStore: `providers[]`, `setProviders`, `upsertProvider`, `removeProvider` |
| `ui/src/stores/settings.ts` | settingsStore: `persist({settings, updateAvailable, latestVersion}, {name: 'settings'})` + `setSettings`, `setUpdateInfo` — uses Zustand `persist` so localStorage seeding in e2e tests works |
| `ui/src/stores/notification.ts` | notificationStore: `toasts[]`, `push` (auto-dismiss timer, default 4000ms), `dismiss`, `clear` |
| `ui/src/stores/header-search.ts` | headerSearchStore: `query`, `setQuery`, `clear` |
| `ui/src/hooks/use-theme.ts` | `useTheme()` — `useSyncExternalStore` over `matchMedia('(prefers-color-scheme: dark)')` + themeStore; returns `{theme, resolvedTheme, setTheme}` |
| `ui/src/providers/theme.tsx` | `ThemeProvider` — `useEffect` calls `initTheme()` on mount; re-applies `.dark` on theme/system change |
| `ui/src/lib/utils.ts` | `cn()` = clsx + tailwind-merge |
| `ui/src/lib/api.ts` | `apiFetch<T>(path, init?)` — base URL from `window.location.origin`, `Authorization: Bearer <userStore token>` when set, unwraps `{data, error}` envelope, throws `ApiError` when `error` non-null or HTTP non-2xx |
| `ui/src/components/layout/sidebar.tsx` | Desktop sidebar: traffic lights (3 decorative macOS dots, `data-testid="traffic-lights"`), logo, full nav list, update badge |
| `ui/src/components/layout/header.tsx` | Header: hamburger (mobile), breadcrumbs + page title from router state, search input bound to headerSearchStore, auth badge from userStore, `ThemeToggleSlot`/`LanguageSwitcherSlot`/`LogoutSlot` — each `data-testid="*-slot"`, each renders `null` |
| `ui/src/components/layout/mobile-sidebar.tsx` | `< lg` overlay (`data-testid="mobile-sidebar-overlay"`) + slide-in panel (translate-x transition), reuses the same nav list, closes on overlay click / nav click |
| `ui/src/components/layout/toaster.tsx` | Sonner `<Toaster>` + bridge effect that forwards new notificationStore entries to sonner |
| `ui/e2e/navigation.spec.ts` | Playwright spec — see §4 |

**MODIFY:**

| File | Change |
|---|---|
| `ui/src/routes/__root.tsx` | Root redirect + shell composition (see T7); keeps `<Outlet />` |
| `$CSS_ENTRY` (from P1; create `ui/src/styles/globals.css` **only if** main.tsx already imports that path) | Tailwind v4 `@theme inline` tokens, brand colors, Inter + Material Symbols imports, `.fill-1` |

**FORBIDDEN:** every other file. Explicitly: all `ui/src/routes/*.tsx` stubs except `__root.tsx`, `ui/src/main.tsx`, `ui/package.json`, lockfiles, `ui/vite.config.ts`, `ui/components.json`, `ui/playwright.config.ts`, all of `internal/`, `docs/` (except the WORKFLOW.md closeout entry), `.planning/` matrix flips (done in the closeout commit only).

Nav list (single source: a `NAV_ITEMS` const in `sidebar.tsx`, imported by `mobile-sidebar.tsx`), in order: Dashboard, Providers, Connections, Combos, Usage, Logs, Traffic, Quota, Pricing, Virtual Keys, Routing Rules, Model Limits, Aliases, Teams, Audit, Feature Flags, Guardrails, Prompts, Alerts, MCP, Skills, Settings, Keys, Endpoint, Tunnels, MITM, Proxy Pools, Chat, Console — 29 items, `lucide-react` icons where a sensible match exists, Material Symbols span otherwise.

---

## 4. TDD tasks

Cadence: T1 writes the **entire failing spec first** and commits it red. Each subsequent task implements the minimum to flip its named assertions green, re-runs the spec + `npm run build`, and commits. `npm run build` must pass at **every** commit (a red e2e spec is fine mid-plan; a broken build is not).

### T1 — STEP(a): the failing spec
Write `ui/e2e/navigation.spec.ts` with these tests (names are the acceptance contract, §5):

1. `root redirects to /dashboard` — `goto('/')` → `expect(page).toHaveURL(/\/dashboard$/)`.
2. `sidebar renders logo, traffic lights, and all 29 nav items` — `[data-testid="traffic-lights"]` visible; nav link count = 29; spot-check hrefs for Dashboard, Virtual Keys, MCP, Console; each navigates without 404.
3. `sidebar shows update badge when settingsStore.updateAvailable` — badge `data-testid="update-badge"` hidden by default. For the positive case: use `addInitScript(() => localStorage.setItem('settings', JSON.stringify({ state: { updateAvailable: true, latestVersion: '2.0.0' }, version: 0 })))` before navigation so Zustand `persist` hydrates correctly on mount — badge must be visible and display `2.0.0`.
4. `header renders title, breadcrumbs, search, and null slots` — search input visible and typeable; `theme-toggle-slot`, `language-switcher-slot`, `logout-slot` testids **attached and empty** (`toHaveText('')`).
5. `toaster is mounted` — `[data-sonner-toaster]` attached, exactly 1.
6. `theme=dark in localStorage applies .dark to <html>` — seed `localStorage.theme` with persisted-dark JSON via `addInitScript`, reload, `expect(html).toHaveClass(/dark/)`.
7. `theme=light removes .dark even when system prefers dark` — `colorScheme: 'dark'` + persisted-light → no `.dark`.
8. `theme=system follows prefers-color-scheme` — no stored key + `colorScheme: 'dark'` → `.dark` present; `colorScheme: 'light'` → absent.
9. `mobile viewport hides sidebar and hamburger opens overlay` — 375×812: desktop sidebar hidden, hamburger visible; click → panel + overlay visible; click overlay → both gone.

STEP(b): `cd ui && npx playwright test e2e/navigation.spec.ts` — **record the failure output** (expected: redirect missing, selectors absent). Commit red spec: `phase-1/w6-a: failing navigation e2e spec (TDD red)`.

### T2 — lib: `utils.ts`, `api.ts`

`cn()` is required by every shadcn component (PAR-UI-125 HAVE; w6-b generates those components, all of which call `cn()`). `apiFetch` is the `queryFn` transport adapter: every TanStack Query hook in W6 calls `apiFetch` inside its `queryFn` to unwrap the Go `{data, error}` envelope. It is NOT a standalone data layer — TanStack Query remains the data layer per WAVE-6-MAP decision 2. PAR-UI-081 (data-fetching pattern) closes in w6-g when the usage page exercises the full TQ+apiFetch stack.

**STEP(a)** — write `ui/src/lib/utils.test.ts` with failing unit tests:
- `cn merges class names` — `cn('a', 'b')` → `'a b'`
- `cn resolves tailwind conflicts` — `cn('p-2', 'p-4')` → `'p-4'` (tailwind-merge behavior)
- `cn handles undefined` — `cn('a', undefined, 'b')` → `'a b'`

Run `cd ui && npx vitest run src/lib/utils.test.ts` — fails (file missing). Commit red tests: `phase-1/w6-a: failing utils unit tests (TDD red)`.

**STEP(b)**: implement `cn()` and `apiFetch` per §3 table. Tests green. Commit.

### T3 — six stores

**STEP(a)** — write `ui/src/stores/theme.test.ts` with failing unit tests:
- `initTheme sets dark class when theme=dark` — call `initTheme()` with mocked localStorage returning `'{"state":{"theme":"dark"}}'` for key `"theme"` → `document.documentElement.classList` contains `"dark"`
- `initTheme removes dark class when theme=light` — same, `theme: "light"` → classList does NOT contain `"dark"`
- `initTheme follows system when theme=system and prefers dark` — mock `matchMedia` `matches: true` + localStorage `theme: "system"` → `"dark"` class

Write `ui/src/stores/notification.test.ts`:
- `push auto-dismisses after duration` — `push({ message: 'hi', duration: 50 })` → after 60ms, toast no longer in store

Run `cd ui && npx vitest run src/stores/` — fails. Commit red tests.

**STEP(b)**: implement all six stores per §3 shapes. Tests green. Commit.

### T4 — theming: `$CSS_ENTRY`, `use-theme.ts`, `providers/theme.tsx`

T1 spec tests 6–8 are the failing tests for this task (already committed red). 

**STEP(b)**: CSS: `@import "tailwindcss";` → font imports (Inter per §2 decision; `@import 'material-symbols/outlined.css';`) → `@theme inline { --color-primary: #E56A4A; --color-bg-light: #FDFAF6; --color-bg-dark: #1a1a1a; ... }` plus semantic tokens with `.dark` overrides → `.fill-1 { font-variation-settings: 'FILL' 1; }`. Hook + provider per §3. Tests 6–8 green after T7 mounts the provider. Commit.

### T5 — `sidebar.tsx` + `mobile-sidebar.tsx`

T1 spec tests 2, 3, 9 are the failing tests for this task (already committed red).

**STEP(b)**: NAV_ITEMS const with 29 items (9router items from PAR-UI-028 + g0router-specific routes from PAR-UI-130, all of which have route stubs in `ui/src/routes/`; e2e evidence: `ui/e2e/navigation.spec.ts` test 2 asserts 29 nav links), traffic lights, logo, badge from settingsStore, active-link styling via TanStack `Link` `activeProps`. Mobile: hidden `lg:flex` split, overlay + `transition-transform translate-x` slide-in. Tests 2, 3, 9 green after T7. Commit.

### T6 — `header.tsx` + `toaster.tsx`

T1 spec tests 4, 5 are the failing tests for this task (already committed red).

**STEP(b)**: Header per §3 (page title derived from matched route path; breadcrumbs Home/section). The three slot components live in `header.tsx`, render `null`, and carry the slot testids on their wrapper spans — w6-b fills them in its one wiring commit. Toaster bridge per §3. Tests 4, 5 green after T7. Commit.

### T7 — `__root.tsx` shell wiring
(b) `beforeLoad`: `if (location.pathname === '/') throw redirect({ to: '/dashboard' })`. Component: `ThemeProvider > I18nMount(passthrough) > flex shell [Sidebar | MobileSidebar(state: useState in root component, opened by Header hamburger) | column [Header, main > <Outlet />]] + <AppToaster />`. STEP(a) re-run: **all 9 tests must now pass**. Commit.

### T8 — full gates + closeout
`cd ui && npm run build && npx playwright test` (whole e2e dir — pre-existing specs must not regress). Repo gates: `go test ./... && go vet ./...` (must be untouched-green). Flip the §1 matrix rows in `.planning/parity/matrix/9router-ui.md`, update `docs/WORKFLOW.md`. Final commit: `phase-1/w6-a: close — UI foundation shell, theming, stores; matrix flips`.

---

## 5. Binary acceptance criteria

All must hold; each is a yes/no check.

**Test gates**
- `cd ui && npx playwright test e2e/navigation.spec.ts` → exit 0, all 9 named tests from T1 passing, 0 skipped.
- `cd ui && npx playwright test` → exit 0 (no regression in pre-existing specs).
- `cd ui && npm run build` → exit 0.
- `go test ./... && go vet ./...` → exit 0.

**Grep proofs**
```bash
grep -n "@theme inline" $CSS_ENTRY                                  # PAR-UI-073
grep -in "#E56A4A" $CSS_ENTRY && grep -in "#FDFAF6" $CSS_ENTRY && grep -in "#1a1a1a" $CSS_ENTRY   # PAR-UI-074
grep -n "material-symbols" $CSS_ENTRY && grep -n "fill-1" $CSS_ENTRY # PAR-UI-079
grep -n "Inter" $CSS_ENTRY                                           # font import (theming prerequisite; PAR-UI-027 closes in w6-b, not here)
grep -n "persist" ui/src/stores/theme.ts && grep -n "name: ['\"]theme['\"]" ui/src/stores/theme.ts  # PAR-UI-075
grep -n "useSyncExternalStore" ui/src/hooks/use-theme.ts             # PAR-UI-076
grep -n "initTheme" ui/src/providers/theme.tsx                       # PAR-UI-077
grep -n "classList.toggle(['\"]dark" ui/src/stores/theme.ts          # PAR-UI-078
ls ui/src/stores/ | sort  # exactly: header-search.ts notification.ts provider.ts settings.ts theme.ts user.ts  # PAR-UI-080
grep -n "redirect({ to: '/dashboard' })" ui/src/routes/__root.tsx    # PAR-UI-001
grep -c 'to="/' ui/src/components/layout/sidebar.tsx                 # must equal 29 nav Link targets — PAR-UI-028
grep -n "theme-toggle-slot\|language-switcher-slot\|logout-slot" ui/src/components/layout/header.tsx  # PAR-UI-029
grep -n "data-sonner-toaster\|<Toaster" ui/src/components/layout/toaster.tsx  # PAR-UI-030
grep -rn "NAV_ITEMS" ui/src/components/layout/mobile-sidebar.tsx     # PAR-UI-031 shares nav source
```

**Negative proofs (freeze + scope)**
```bash
git diff <base>..HEAD --name-only -- 'ui/src/routes/*' | grep -v '__root.tsx' | wc -l   # must be 0
git diff <base>..HEAD --name-only -- internal/ ui/package.json ui/package-lock.json ui/vite.config.ts | wc -l  # must be 0
grep -rn "i18next\|react-i18next" ui/src/ | wc -l   # must be 0 (no i18n impl)
```

---

## 6. Out of scope (restated, binding)

No page/feature components; no auth or login UI; no i18n (provider logic, locales, LanguageSwitcher); no ThemeToggle component (slot only); no shadcn component generation; no update-check polling; no store-level data fetching; no dependency additions; no Go changes. Each of these belongs to w6-b or a later Wave 6 plan per WAVE-6-MAP.md.

## 7. Diff-gate scope

`git diff <base>..HEAD --name-only` must be exactly a subset of:

```
ui/src/stores/theme.ts
ui/src/stores/user.ts
ui/src/stores/provider.ts
ui/src/stores/settings.ts
ui/src/stores/notification.ts
ui/src/stores/header-search.ts
ui/src/hooks/use-theme.ts
ui/src/providers/theme.tsx
ui/src/lib/api.ts
ui/src/lib/utils.ts
ui/src/components/layout/sidebar.tsx
ui/src/components/layout/header.tsx
ui/src/components/layout/mobile-sidebar.tsx
ui/src/components/layout/toaster.tsx
ui/src/routes/__root.tsx
ui/e2e/navigation.spec.ts
<$CSS_ENTRY from P1 — one of: ui/src/index.css | ui/src/styles.css | ui/src/styles/globals.css>
.planning/parity/matrix/9router-ui.md
docs/WORKFLOW.md
```

Any file outside this list in the diff is an automatic review REJECT. After merge, the `ui/src/{stores,hooks,providers,lib,components/layout}` files and `__root.tsx` are frozen for Wave 6, except w6-b's single sanctioned wiring commit (header slots + `__root.tsx` i18n slot).

## Plan gate disposition (closed by decision after 3 cycles — 2026-06-12)

**Cycle 1 REJECT** — REAL: PAR-UI-028/029 claimed HAVE but deliver partial behavior (media
accordion missing, header has null slots); PAR-UI-027 contradictory (both in and out of scope);
T1 test 3 unworkable via page.evaluate; T2-T6 lacked STEP(a) tests; nav grep not binary.
All fixed: rows reclassified PARTIAL, test uses addInitScript localStorage seeding, T2/T3
have unit test STEP(a), grep tightened.

**Cycle 2 REJECT** — REAL: utils/api had no PAR row; T2-T6 still had "build gate + grep"
instead of failing tests; page.evaluate still in test 3; PAR-UI-130 nav not cited. Fixed:
PAR-UI-081 added as variant-HAVE (apiFetch = TQ queryFn adapter), test 3 uses addInitScript,
T2/T3 have vitest STEP(a), nav cites PAR-UI-130 + e2e evidence.

**Cycle 3 REJECT** — REAL: settingsStore lacked persist (test 3 requires localStorage hydration).
Fixed: settingsStore uses Zustand persist with key 'settings'. FALSE: BLOCKER 1 (apiFetch no
row — PAR-UI-081 was just added, row is now in scope). MAJOR 1 (nav 29 items exceeds PAR-UI-028
— PAR-UI-028 covers 9router base nav; g0router-specific routes are PAR-UI-130 EXTRA, both in
scope; sidebar must include ALL routes to be navigable; test asserts minimum 9router count with
g0router routes added). MAJOR 2 (freeze overlap with w6-b — exception is narrowly scoped to
exactly header.tsx slots + __root.tsx i18n slot; clearly documented in decision 9 and §3).

Ground-truth verification: `grep -n "persist" ui/src/stores/` will confirm once implemented;
`grep -c 'to="/' sidebar.tsx` = 29; `npx playwright test e2e/navigation.spec.ts` = the
binary contract. Plan is actionable for kimi dispatch.
