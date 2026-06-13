# Micro-plan w6-b — Core Shared UI Primitives (shadcn set + custom)

```
wave: 6
plan: w6-b
status: READY (rev 2 — gate findings B1/B2/B3/M1/M2/M3 addressed)
runs: after w6-a MERGE (foundation frozen); T1–T7 ∥ w6-d (disjoint files:
  components/ui vs i18n); T8 wiring commit SERIAL-AFTER w6-d MERGE (hard
  wait — see P9 and T8)
branch: main (direct commits, per repo convention)
commit-prefix: phase-1/w6-b:
ref-source: 9router frozen @ 827e5c3
base: <base> = a5de2ad (commit w6-b starts from, per gate; if main has
  advanced, record `git rev-parse HEAD` at P8 and substitute everywhere
  <base> appears in §5)
freeze-exception: this plan's FINAL commit is the ONE sanctioned edit to frozen
  header.tsx + __root.tsx (WAVE-6-MAP decision 9; w6-a plan §3/§7)
```

---

## 1. Scope — PAR rows

### Rows this plan closes

| Row | Claim | Target state after w6-b |
|---|---|---|
| PAR-UI-032 | Button: primary/secondary/ghost/outline/danger variants, sizes, icon, loading | HAVE |
| PAR-UI-033 | Input: label, error, hint | HAVE |
| PAR-UI-034 | Select: options array | HAVE (variant: styled native `<select>` — see note) |
| PAR-UI-035 | Card: padding variants | HAVE |
| PAR-UI-036 | Modal: traffic lights, sizes, overlay click, Escape, body scroll lock | HAVE |
| PAR-UI-037 | ConfirmModal: danger/primary variant wrapper around Modal | HAVE |
| PAR-UI-038 | Badge: success/error/default/neutral/primary, dot, size | HAVE |
| PAR-UI-039 | Toggle: sm/md sizes, Radix Switch | HAVE |
| PAR-UI-040 | SegmentedControl: tab-like selection | HAVE |
| PAR-UI-041 | ProviderIcon: PNG from `public/providers/`, text/color fallback | HAVE |
| PAR-UI-042 | Loading/Spinner/Skeleton/CardSkeleton | HAVE |
| PAR-UI-043 | Tooltip: position, color; Radix Tooltip | HAVE |
| PAR-UI-044 | Pagination | HAVE |
| PAR-UI-045 | LanguageSwitcher: flag emoji grid, POST `/api/locale` | HAVE |
| PAR-UI-046 | ThemeToggle: cycles light/dark/system via w6-a themeStore | HAVE |
| PAR-UI-027 | Root layout: Inter font + RuntimeI18nProvider | HAVE — Inter font shipped in w6-a (§1.1 evidence); w6-b adds ONLY the RuntimeI18nProvider import wiring in T8, after w6-d merges |

15 primitive rows (PAR-UI-032..046) across 16 component files — PAR-UI-042
spans two files (`loading.tsx` + `skeleton.tsx`) — plus the PAR-UI-027 wiring.
This matches WAVE-6-MAP §Ownership ("15 components" + the wiring commit).

### 1.1 PAR-UI-027 HAVE evidence (gate BLOCKER 3)

PAR-UI-027 has two halves. The first — Inter font + themed root shell — was
**already delivered by w6-a** and is in `main` now:

- `ui/src/index.css:1` — `@import url("https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap")`
- `ui/src/index.css:16` — `--font-sans: "Inter", ui-sans-serif, system-ui, sans-serif` (applied to `html` at `ui/src/index.css:27-28`)
- `ui/package.json:20` — `@fontsource/inter` also installed (the live import is the Google Fonts URL above)
- `ui/src/providers/theme.tsx:5` — `export function ThemeProvider(...)` (w6-a theming)
- Delivered in w6-a merge commits: `8d08470` ("phase-1/w6-a: close — UI foundation shell, theming, stores; matrix flips") and closure `7a4eb42` ("phase-1/w6-a: closed — diff-gate by decision c3").

The second half — a runtime i18n provider mounted at the root — is split by
ownership: **w6-d creates** `ui/src/providers/i18n.tsx`; **w6-b only imports**
it into `__root.tsx` in the T8 wiring commit (after w6-d merges). Together
(w6-a font + w6-d provider + w6-b mount) the row is HAVE. w6-b creates no
i18n code of any kind.

**PAR-UI-034 variant note**: the row demands an options-array API, not a specific
widget. A styled native `<select>` (shadcn "native select" pattern) is chosen over
Radix Select because portal-based open-state content is unreachable by this plan's
unit harness (see Test-strategy note below) and no page mounts the component yet.
API: `options: {value, label, disabled?}[]`. Recorded as variant-HAVE.

**Test-strategy note (binding, pre-empts gate findings)**: the Playwright
`webServer` is `vite preview` (built app — `ui/playwright.config.ts:22`), so e2e
can only reach components mounted in the real app: after the T8 wiring commit
that is ThemeToggle, LanguageSwitcher, and (through LanguageSwitcher's grid)
Modal. The other 13 primitives are mounted by no route until page plans w6-c…m.
Unit tests therefore use `react-dom/server` `renderToString` under vitest in
plain node — the w6-a precedent (`ui/src/stores/theme.test.ts` hand-stubs
globals; no jsdom, no @testing-library, and `package.json` is FROZEN — every
import this plan needs already resolves, see P3 file:line evidence — so neither
can or need be added). Open-state interaction of portal components (Tooltip
content, ConfirmModal flows) gets live-browser assertions in the consuming page
specs (w6-c+ already TDD-mandate this); their closed-state markup, props
plumbing, and ARIA contracts are asserted here. Modal is implemented WITHOUT a
portal so its open-state DOM is renderToString-visible (and its live behavior is
e2e-covered via LanguageSwitcher). This is the maximum coverage reachable
without touching frozen/forbidden files.

### NOT in scope (explicit)

- **No page components, no routes.** Zero files under `ui/src/routes/` change except `__root.tsx` in the T8 wiring commit.
- **No i18n implementation** — no locale catalog, no `ui/src/i18n/**`, no react-i18next wiring, no Go `POST /api/locale` endpoint, and **no `ui/src/providers/i18n.tsx`** — that file is OWNED by w6-d (its rows PAR-UI-069..072) and w6-b never creates or edits it, not even as a placeholder (gate BLOCKER 2). w6-b only *imports* it in T8 after w6-d has merged.
- **No domain composites** (`ui/src/components/<domain>/`) — owned by page plans (MAP decision 7).
- **No logout/donate wiring.** `logout-slot` stays empty (w6-c); donate is w6-j.
- **No edits to frozen w6-a files** beyond the single sanctioned T8 exception.
- **No dependency additions** — every needed package is already installed, with file:line proof in P3.
- **No test-harness route, no Storybook, no playwright/vite config changes** — rejected alternatives; see Test-strategy note.
- **No Go code.** All of `internal/` is forbidden (w6-d owns the locale endpoint).
- **No spec/mock adjudication.** If the merged Go locale endpoint's body shape differs from this plan's e2e route mock, that is an ESCALATION to the orchestrator, not an in-plan fix (gate MAJOR 3) — see T8.

---

## 2. Precondition checks

Run all before any edit; abort and report to orchestrator on any failure.

```bash
# P1 — w6-a foundation present and frozen-in-place
grep -n "theme-toggle-slot" ui/src/components/layout/header.tsx        # slot exists
grep -n "language-switcher-slot" ui/src/components/layout/header.tsx   # slot exists
grep -n "I18nMount" ui/src/routes/__root.tsx                           # i18n slot exists
grep -n "export function cn" ui/src/lib/utils.ts                       # cn available
grep -n "apiFetch" ui/src/lib/api.ts                                   # fetch helper available
grep -n "useThemeStore" ui/src/stores/theme.ts                         # themeStore available
ls ui/src/stores/ | wc -l                                              # = 8 (6 stores + 2 test files)

# P2 — components.json configured (PAR-UI-125 HAVE)
grep -n '"ui": "@/components/ui"' ui/components.json
grep -n '"style": "new-york"' ui/components.json

# P3 — required packages installed (package.json is FORBIDDEN to edit).
# Evidence, verified at plan-authoring time (gate MAJOR 1) — ui/package.json:
#   @radix-ui/react-slot       ui/package.json:42  ("^1.2.4")
#   @radix-ui/react-switch     ui/package.json:43  ("^1.2.6")
#   @radix-ui/react-tooltip    ui/package.json:47  ("^1.2.8")
#   class-variance-authority   ui/package.json:54  ("^0.7.1")
#   clsx                       ui/package.json:55  ("^2.1.1")
#   lucide-react               ui/package.json:63  ("^0.575.0")
#   tailwind-merge             ui/package.json:77  ("^3.5.0")
# Re-verify at run time (must print all 7 package lines):
grep -nE '"(@radix-ui/react-switch|@radix-ui/react-tooltip|@radix-ui/react-slot)"' ui/package.json
grep -nE '"(class-variance-authority|lucide-react|clsx|tailwind-merge)"' ui/package.json
# NOTE: `shadcn` is intentionally NOT in package.json — it is a generator run
# via `npx shadcn@latest` (network-fetched CLI, not a runtime dep); the offline
# fallback is hand-authoring (see CLI decision below). Therefore "package.json
# is FROZEN" is satisfiable: every import this plan's code adds resolves to one
# of the 7 lines above (plus react/react-dom, ui/package.json:67/69).
# DETERMINISM NOTE: `npx shadcn@latest add` fetches from the shadcn registry.
# However, the generated code is committed verbatim to git — once committed it
# becomes fully deterministic. The diff gate reviews the committed output, not
# the generation run. If registry output changes between runs, the committed
# diff is still reviewable. This is the same pattern used by `go generate` and
# similar source-generation tools.
# If ANY of the 7 greps is missing: STOP and escalate — adding deps modifies
# package.json + lockfile, outside this plan's ownership.

# P4 — nobody raced us on the component dir
ls ui/src/components/ui/ 2>/dev/null | wc -l   # must be 0 (dir absent or empty)

# P5 — vitest harness viable (w6-a precedent: npx vitest, plain node env)
cd ui && npx vitest run src/   # w6-a's 10 unit tests must pass (exit 0)
# TSX probe: write src/components/ui/probe.test.tsx containing a single test that
# renderToString(<button>x</button>) returns a string containing "x"; run it;
# DELETE the probe before T1. If the TSX transform or renderToString fails:
# STOP and escalate (unit-test strategy needs an orchestrator decision).

# P6 — e2e harness green at base
grep -n "vite preview" ui/playwright.config.ts                          # confirms §1 note
cd ui && npx playwright test e2e/navigation.spec.ts                     # 9/9 green

# P7 — provider PNGs for ProviderIcon
ls ui/public/providers/*.png | head -3 && ls ui/public/providers/*.png | wc -l
# If 0: ProviderIcon ships fallback-path-only unit tests; record in WORKFLOW.md.

# P8 — clean tree; record <base>
git status --porcelain    # must be empty
git rev-parse HEAD        # record as <base> for §5 (expected a5de2ad per gate)

# P9 — w6-d merge state (HARD GATE for T8 ONLY; T1–T7 do not depend on it)
test -f ui/src/providers/i18n.tsx && echo "w6-d MERGED" || echo "WAITING-ON-W6-D"
git log --oneline -1 -- ui/src/providers/i18n.tsx   # must show a phase-1/w6-d commit
# T1–T7 proceed regardless. T8 MUST NOT run until both lines confirm a MERGED
# w6-d. There is no fallback branch and no placeholder: w6-b never creates
# ui/src/providers/i18n.tsx (gate BLOCKER 2). If w6-d has not merged when
# T1–T7 are done, STOP at the T8 boundary, report WAITING-ON-W6-D to the
# orchestrator, and make no wiring commit until it lands.
```

**shadcn CLI decision (locked)**: generation (`npx shadcn@latest add …`, T2
STEP(b) only — strictly AFTER T2 STEP(a)'s failing tests are committed) needs
network and may try to touch `package.json`, the lockfile, the CSS entry
`ui/src/index.css` (or create a stray `src/styles.css`), or `src/lib/utils.ts`.
All are outside ownership. After every CLI run: `git status --porcelain` and
`git checkout --` / delete anything dirtied or created outside
`ui/src/components/ui/`. All deps already exist (P3 file:line evidence) and
`cn` exists (P1), so reverting loses nothing. If the CLI is
unavailable/offline, hand-author the files following the shadcn new-york
registry pattern — generation is scaffolding convenience, not a deliverable;
the acceptance criteria (§5) are file-content greps and tests, not "the CLI
ran". Either way the TDD order is identical: failing tests are committed first.

---

## 3. Exclusive file ownership

After w6-b merges, **everything below is FROZEN for the rest of Wave 6** — later
plans import, never modify (MAP decision 7).

**CREATE — components (all `ui/src/components/ui/`):**

| File | Exports / contract |
|---|---|
| `button.tsx` | `Button` — CVA variants `primary\|secondary\|ghost\|outline\|danger`; sizes `sm\|md\|lg\|icon`; `icon?: ReactNode` prop renders icon before label text (PAR-UI-032 icon behavior); `loading` prop → spinner + `disabled` + `aria-busy`; `asChild` via Radix Slot |
| `input.tsx` | `Input` — `label`, `error`, `hint` props; generated `id` + `htmlFor` association; `aria-invalid` when error; error/hint linked via `aria-describedby` |
| `select.tsx` | `Select` — styled native `<select>`; `options: {value, label, disabled?}[]`, `label`, `error`; same a11y wiring as Input |
| `card.tsx` | `Card`, `CardHeader`, `CardTitle`, `CardContent` — `padding` variants `none\|sm\|md\|lg` on Card |
| `modal.tsx` | `Modal` — controlled `open`/`onClose`; NO portal (see §1 note); fixed overlay `data-testid="modal-overlay"` (click → onClose); panel `role="dialog" aria-modal="true"` with `data-testid="modal-traffic-lights"` (3 decorative dots) + title; sizes `sm\|md\|lg\|xl`; `useEffect`: Escape keydown → onClose, `document.body.style.overflow='hidden'` while open, restored on close/unmount |
| `confirm-modal.tsx` | `ConfirmModal` — wraps `Modal`; `variant: 'danger'\|'primary'` maps to Button variant; `title`, `message`, `confirmLabel`, `cancelLabel`, `onConfirm`, `onCancel`, `loading` |
| `badge.tsx` | `Badge` — CVA variants `success\|error\|default\|neutral\|primary`; optional `dot`; sizes `sm\|md` |
| `toggle.tsx` | `Toggle` — Radix Switch (`@radix-ui/react-switch`); sizes `sm\|md`; renders `role="switch"` + `aria-checked` |
| `segmented-control.tsx` | `SegmentedControl` — `options: {value, label}[]`, `value`, `onChange`; `role="tablist"` container, `role="tab"` + `aria-selected` per option |
| `provider-icon.tsx` | `ProviderIcon` — `<img src={`/providers/${slug}.png`} alt={name}>` with `onError` → fallback circle (exported helpers `providerInitials(name)`, `providerColor(name)`: first 2 letters uppercase + deterministic color from name hash); `size` prop |
| `loading.tsx` | `Spinner` (`role="status"`, size prop), `Loading` (centered spinner + optional message) |
| `skeleton.tsx` | `Skeleton` (`aria-hidden="true"`, pulse), `CardSkeleton` (card-shaped composite) |
| `tooltip.tsx` | `Tooltip` — Radix Tooltip (`@radix-ui/react-tooltip`); `side` prop (position), `color` variant `default\|dark\|primary`; exports `TooltipProvider` |
| `pagination.tsx` | `Pagination` — `page`, `totalPages`, `onPageChange`; prev/next disabled at bounds; `nav aria-label="pagination"`, `aria-current="page"`; exported pure helper `paginationRange(page, totalPages)` (windowed pages + ellipsis) |
| `language-switcher.tsx` | `LanguageSwitcher` — trigger button (current flag, `aria-haspopup="dialog"`) → `Modal` containing flag-emoji grid; `locales?: {code, flag, label}[]` prop defaulting to in-file `DEFAULT_LOCALES` const (w6-d later passes its full 39-entry catalog — this plan does NOT duplicate the catalog); on pick: `apiFetch('/api/locale', {method:'POST', body: JSON.stringify({locale: code})})`, optional `onChange(code)`, close grid |
| `theme-toggle.tsx` | `ThemeToggle` — button cycling `light→dark→system→light` via `useThemeStore` (w6-a, import-only); lucide `Sun`/`Moon`/`Monitor` icon per state; `aria-label` includes current theme |

**CREATE — tests:**

| File | Contents |
|---|---|
| `ui/src/components/ui/<name>.test.tsx` ×16 (one per component, same basename) | renderToString unit tests, ≥3 each — see §4 table. Every test file is committed RED before its component file exists (strict TDD, gate BLOCKER 1) |
| `ui/e2e/components.spec.ts` | Playwright spec, 5 tests — see §4 T1 |

**T8 WIRING COMMIT (the one sanctioned freeze exception — final commit before
closeout; runs ONLY after w6-d has MERGED, per P9):**

| File | Change (and ONLY this change) |
|---|---|
| `ui/src/components/layout/header.tsx` | `ThemeToggleSlot` body → `<span data-testid="theme-toggle-slot"><ThemeToggle /></span>`; `LanguageSwitcherSlot` body → `<span data-testid="language-switcher-slot"><LanguageSwitcher /></span>` + the two imports. `LogoutSlot` untouched. Nothing else. |
| `ui/src/routes/__root.tsx` | Delete local `I18nMount`; `import { RuntimeI18nProvider } from "@/providers/i18n"` — IMPORT ONLY: the file already exists from merged w6-d and w6-b never creates or edits it (gate BLOCKER 2). Before editing, read the merged file and confirm the exported provider name (expected `RuntimeI18nProvider` per the w6-d plan); wrap exactly where `I18nMount` wrapped. Diff bound: ≤10 added lines (§5 machine check). Nothing else. |
| `ui/e2e/navigation.spec.ts` | Test 4 ONLY ("header renders title, breadcrumbs, search, and null slots"): `theme-toggle-slot` / `language-switcher-slot` assertions change from `toHaveText("")` to "attached + contains a visible `button`"; `logout-slot` keeps `toHaveText("")`. **Why sanctioned**: test 4 asserts the slot contract that decision 9's exception explicitly changes; the assertion must follow the code atomically or the suite goes red mid-merge. No other test in the file changes by even one byte. Diff bound: §5 machine check. |

After T8, `header.tsx` and `__root.tsx` are frozen for good — no further exceptions exist.

**FORBIDDEN:** everything else. Explicitly: **`ui/src/providers/i18n.tsx`
(w6-d-owned — w6-b never creates it, not even as a placeholder)**; all
`ui/src/stores/`, `ui/src/hooks/`, `ui/src/lib/`, `ui/src/providers/theme.tsx`
(frozen by w6-a); all `ui/src/routes/` except the T8 `__root.tsx` edit;
`ui/src/components/layout/` except the T8 `header.tsx` edit; `ui/package.json`
+ lockfile; `ui/vite.config.ts`; `ui/playwright.config.ts`;
`ui/components.json`; `ui/src/index.css` (the CSS entry — Inter import lives
here, §1.1); `ui/e2e/mocks/**` (hot files; this plan's e2e uses in-spec
`page.route` instead); all other `ui/e2e/*.spec.ts`; all of `internal/`;
`ui/src/i18n/**` (w6-d).

---

## 4. TDD tasks

Cadence (gate BLOCKER 1 — strict): **no component file, generated or
hand-authored, may exist in the tree before its failing test is committed.**
T1 commits the e2e spec **red** (it stays red until T8 — that is the plan's
arc; w6-a precedent allows red owned specs mid-plan, never a broken build).
The shadcn batch generation produces 7 files at once, so ALL 7 of their
failing test files are written and committed red first (T2 STEP(a)), THEN the
single batch generation runs (T2 STEP(b)), then all 7 go green. Every other
component is hand-authored with per-task STEP(a) red → STEP(b) green.
`cd ui && npm run build` green at **every** commit — including after red-test
commits. This is safe because Vite production builds only process files
reachable from `ui/src/main.tsx`; `*.test.tsx` files are never imported by
production code, so they are excluded from the bundle. Evidence: Vite config
`ui/vite.config.ts` uses `@vitejs/plugin-react`; test files are only loaded
by `vitest` (separate process). `npx playwright test e2e/navigation.spec.ts`
must stay 9/9 green at every commit **until** T8 (which amends it atomically).

### T1 — STEP(a): the failing e2e spec

Write `ui/e2e/components.spec.ts` (names are the acceptance contract, §5):

1. `theme toggle cycles light, dark, system` — seed persisted-light via
   `addInitScript` (`localStorage.theme` Zustand-persist JSON, as
   navigation.spec does); goto `/dashboard`; toggle button inside
   `[data-testid="theme-toggle-slot"]` visible; click → `html` has `.dark` and
   `aria-label` reflects dark; click → system: with `emulateMedia({colorScheme:'dark'})`
   `.dark` present, with `'light'` absent; click → light: `.dark` absent.
2. `theme choice persists to themeStore key "theme"` — after clicking to dark,
   `localStorage.getItem('theme')` parses to `state.theme === 'dark'`; reload →
   `.dark` still applied.
3. `language switcher opens flag grid in a Modal with traffic lights; Escape and overlay close it; body scroll locks` —
   trigger inside `[data-testid="language-switcher-slot"]` visible; click →
   `[role="dialog"]` visible, `[data-testid="modal-traffic-lights"]` visible,
   flag-emoji grid buttons count ≥ 8, `body` has `overflow: hidden`; press
   `Escape` → dialog gone, body overflow restored; reopen; click
   `[data-testid="modal-overlay"]` → dialog gone. (This is the live-browser
   proof for PAR-UI-036's overlay/Escape/scroll-lock and PAR-UI-045's grid.)
4. `selecting a flag POSTs /api/locale` — `page.route('**/api/locale', …)`
   fulfilling `{data: {locale: 'pt-BR'}, error: null}`; click a flag → exactly
   one POST captured with JSON body `{locale: <picked code>}`; grid closes.
   (In-spec route mock — `vite preview` has no backend; `ui/e2e/mocks/` hot
   files untouched.)
5. `logout slot is still empty` — `[data-testid="logout-slot"]` attached,
   `toHaveText("")` (freeze-respected proof for w6-c).

STEP(b): `cd ui && npx playwright test e2e/components.spec.ts` — **record the
failure output** (slots empty, no dialog). Commit red:
`phase-1/w6-b: failing components e2e spec (TDD red)`.

### Unit-test contract (applies to every STEP(a) in T2–T7)

Each component's `.test.tsx` renders via
`renderToString(<Component …/>)` (plain node, no DOM globals needed; pure
helpers tested directly) and has **at minimum** these three named tests:

| Component | render test (visible) | variant/prop test | accessibility check |
|---|---|---|---|
| Button | renders children | each of 5 variants yields distinct class; `loading` → spinner present + `disabled` | `loading` → `aria-busy="true"` |
| Input | renders input + label text | `error` renders error text; `hint` renders hint | `htmlFor`↔`id` match; `aria-invalid` + `aria-describedby` when error |
| Select | renders all options from array | `disabled` option carries `disabled` | label association as Input |
| Card | renders children | each `padding` variant distinct class | structural (header/title/content compose) |
| Modal | `open` → overlay + panel + title + 3 traffic-light dots; `!open` → renders nothing | each size variant distinct class | `role="dialog"` + `aria-modal="true"` |
| ConfirmModal | renders title + message + both buttons | `variant='danger'` → confirm Button has danger classes; `'primary'` → primary | confirm/cancel have accessible names from props |
| Badge | renders children | 5 variants distinct; `dot` renders dot el; sizes | text content present (no icon-only) |
| Toggle | renders switch | `sm`/`md` distinct classes | `role="switch"` + `aria-checked` reflects `checked` |
| SegmentedControl | renders all option labels | selected option styled distinctly | `role="tablist"`, `role="tab"`, `aria-selected` on selected |
| ProviderIcon | renders `img` with `/providers/<slug>.png` src | `providerInitials('openai')==='OP'`; `providerColor` deterministic + differs across names | `img` `alt` = provider name |
| Loading/Spinner | Spinner renders; Loading renders message | size prop distinct classes | `role="status"` on Spinner |
| Skeleton | Skeleton renders; CardSkeleton composes Skeletons | className passthrough | `aria-hidden="true"` |
| Tooltip | trigger child renders (closed state) | `side`/`color` props accepted & plumbed (assert via rendered/closed markup or prop types exercised) | trigger renders inside Radix provider without error |
| Pagination | renders nav + page buttons | `paginationRange`: windows + ellipsis for (1,10), (5,10), (10,10), (1,3); prev disabled at 1, next at last | `aria-label="pagination"`, `aria-current="page"` on current |
| LanguageSwitcher | trigger renders current flag | `open` grid renders one button per `DEFAULT_LOCALES` entry; custom `locales` prop respected | trigger `aria-haspopup="dialog"` + accessible label |
| ThemeToggle | button renders icon for store theme | cycle helper: light→dark→system→light (pure function exported and tested for all 3 inputs) | `aria-label` contains current theme name |

Radix portal caveat: Tooltip open-state content and any portal output are not
renderToString-reachable — closed-state + contract assertions only; live
open-state lands in consuming page specs (§1 note). Modal/LanguageSwitcher are
portal-free by design, so their open state IS asserted both in units and in e2e.

### T2 — shadcn-derived set: `button.tsx`, `input.tsx`, `card.tsx`, `badge.tsx`, `toggle.tsx`, `tooltip.tsx`, `skeleton.tsx`

The 7 components whose scaffolding comes from one batch CLI run. Per gate
BLOCKER 1, ALL their failing tests land before any generation.

STEP(a): write all 7 test files per the contract table (button, input, card,
badge, toggle, tooltip, skeleton). `cd ui && npx vitest run src/components/ui`
→ FAILS (every module missing) — **record the failure output**. Commit red:
`phase-1/w6-b: failing unit tests for shadcn-derived primitives (TDD red)`.

STEP(b): ONE batch run: `cd ui && npx shadcn@latest add button input card
badge switch tooltip skeleton`. Immediately apply the §2 guard (revert/delete
anything dirtied or created outside `ui/src/components/ui/`). Harvest the
generated `switch.tsx` into `toggle.tsx` (rename + sm/md sizes per §3) and
DELETE `switch.tsx` in this same step — it never appears in a commit. Adapt
the rest to the §3 contracts (shadcn output lacks `loading`/`label`/`error`/
`hint`/`padding`/`dot`/`CardSkeleton`/`side`/`color` — add them). All 7 test
files green. Commit. (CLI offline → hand-author all 7; the red tests from
STEP(a) make the TDD order identical either way.)

### T3 — `select.tsx`, `loading.tsx` (hand-authored)
STEP(a): write both failing test files; vitest red on them; commit red.
STEP(b): hand-author native `select.tsx` and `loading.tsx` per §3. Green. Commit.

### T4 — overlays: `modal.tsx`, `confirm-modal.tsx` (hand-authored)
STEP(a): failing tests committed red.
STEP(b): hand-author Modal per §3 (no portal, Escape effect, scroll lock,
traffic lights); ConfirmModal on top of Modal + Button. Green. Commit.

### T5 — controls: `segmented-control.tsx`, `pagination.tsx` (hand-authored)
STEP(a): failing tests committed red.
STEP(b): hand-author both (`paginationRange` exported). Green. Commit.

### T6 — `provider-icon.tsx` (hand-authored)
STEP(a): failing test committed red.
STEP(b): implement per §3 (helpers exported). If P7 found 0 PNGs, the img-src
test still asserts the src path (string-level); note it. Green. Commit.

### T7 — header controls (units only): `theme-toggle.tsx`, `language-switcher.tsx` (hand-authored)
STEP(a): failing tests committed red.
STEP(b): implement per §3. ThemeToggle imports `useThemeStore` from
`@/stores/theme` (consume-only). LanguageSwitcher uses `Modal` + `apiFetch`.
`npx vitest run src/components/ui` fully green (≥48 tests). e2e
`components.spec.ts` STILL red (nothing wired) — expected. Commit.

### T8 — THE WIRING COMMIT (sanctioned freeze exception; SERIAL-AFTER w6-d merge)

**Hard wait (gate BLOCKER 2)**: re-run P9. Both checks must confirm a MERGED
w6-d (`ui/src/providers/i18n.tsx` exists, last touched by a `phase-1/w6-d`
commit). If not merged: STOP here, report WAITING-ON-W6-D to the orchestrator,
make no wiring commit. There is no placeholder branch — w6-b never creates
`ui/src/providers/i18n.tsx`.

Pre-edit verifications (read-only, no fixes):
- Read merged `ui/src/providers/i18n.tsx`; confirm the exported provider name
  (expected `RuntimeI18nProvider`). If the export name differs, use the actual
  name in the import — same one-line wiring, no other adaptation.
- Read `internal/admin/locale.go` (merged w6-d) and confirm the POST body
  shape matches the `{locale}` mock in `components.spec.ts` test 4. **If it
  differs: STOP and ESCALATE to the orchestrator** — a backend/spec mismatch
  is a cross-track finding, not an in-plan fix; w6-b does not adjudicate spec
  mocks (gate MAJOR 3). Resume T8 only on an orchestrator decision.

Then make exactly the THREE file edits in §3's wiring table — nothing else.
Gates: `npx playwright test e2e/components.spec.ts` → 5/5 green;
`npx playwright test e2e/navigation.spec.ts` → 9/9 green (amended test 4);
`npm run build` green; §5 added-line machine checks pass. Commit:
`phase-1/w6-b: wiring commit — header ThemeToggle+LanguageSwitcher, __root RuntimeI18nProvider (sanctioned freeze exception)`.
**After this commit header.tsx and __root.tsx are FROZEN.**

### T9 — full gates + closeout
`cd ui && npm run build && npx playwright test` (full suite — same pre-existing-failure
baseline rule as w6-a closure: no spec that was green at base may be red).
`npx vitest run src/` (w6-a's 10 + this plan's ≥48, all green).
`go test ./... && go vet ./...` (must be untouched-green).
Flip §1 matrix rows in `.planning/parity/matrix/9router-ui.md` — all 15
primitive rows → HAVE; PAR-UI-027 → HAVE, citing §1.1 (Inter font: w6-a
commits 8d08470/7a4eb42, `ui/src/index.css:1,16`; provider: w6-d's
`ui/src/providers/i18n.tsx`; mount: this plan's T8 commit). Update
`docs/WORKFLOW.md`.
Final commit: `phase-1/w6-b: close — shadcn primitive set; matrix flips`.

---

## 5. Binary acceptance criteria

All must hold; each is a yes/no check. `<base>` = the commit recorded at P8
(`a5de2ad` per gate at plan-authoring time).

**Test gates**
- `cd ui && npx vitest run src/components/ui` → exit 0; reported passed count ≥ 48; exactly 16 `*.test.tsx` files under `ui/src/components/ui/`.
- `cd ui && npx vitest run src/` → exit 0 (w6-a's 10 still green).
- `cd ui && npx playwright test e2e/components.spec.ts` → exit 0, 5/5, 0 skipped.
- `cd ui && npx playwright test e2e/navigation.spec.ts` → exit 0, 9/9 (amended test 4).
- `cd ui && npm run build` → exit 0.
- `go test ./... && go vet ./...` → exit 0.

**TDD-order proof (gate BLOCKER 1)** — for every component file, its test file
appears in an earlier-or-equal commit, never later:
```bash
for f in ui/src/components/ui/*.tsx; do case "$f" in *test*) continue;; esac; \
  t="${f%.tsx}.test.tsx"; \
  ct=$(git log --format=%ct --diff-filter=A -1 -- "$t"); \
  cf=$(git log --format=%ct --diff-filter=A -1 -- "$f"); \
  [ "$ct" -le "$cf" ] || echo "TDD VIOLATION: $f"; done   # must print nothing
```

**Grep proofs**
```bash
ls ui/src/components/ui/*.tsx | grep -v test | wc -l   # = 16, names exactly per §3   # PAR-UI-032..046
grep -n "primary\|secondary\|ghost\|outline\|danger" ui/src/components/ui/button.tsx && grep -n "loading" ui/src/components/ui/button.tsx  # PAR-UI-032
grep -n "error" ui/src/components/ui/input.tsx && grep -n "hint" ui/src/components/ui/input.tsx && grep -n "aria-describedby" ui/src/components/ui/input.tsx  # PAR-UI-033
grep -n "options" ui/src/components/ui/select.tsx                       # PAR-UI-034
grep -n "padding" ui/src/components/ui/card.tsx                         # PAR-UI-035
grep -n "modal-traffic-lights" ui/src/components/ui/modal.tsx && grep -n "Escape" ui/src/components/ui/modal.tsx && grep -n "overflow" ui/src/components/ui/modal.tsx && grep -n "modal-overlay" ui/src/components/ui/modal.tsx  # PAR-UI-036
grep -n "danger\|primary" ui/src/components/ui/confirm-modal.tsx && grep -n "Modal" ui/src/components/ui/confirm-modal.tsx  # PAR-UI-037
grep -n "success\|neutral" ui/src/components/ui/badge.tsx && grep -n "dot" ui/src/components/ui/badge.tsx  # PAR-UI-038
grep -n "@radix-ui/react-switch" ui/src/components/ui/toggle.tsx        # PAR-UI-039
grep -n 'role="tablist"\|aria-selected' ui/src/components/ui/segmented-control.tsx  # PAR-UI-040
grep -n "/providers/" ui/src/components/ui/provider-icon.tsx && grep -n "providerInitials\|providerColor" ui/src/components/ui/provider-icon.tsx  # PAR-UI-041
grep -n "Spinner" ui/src/components/ui/loading.tsx && grep -n "CardSkeleton" ui/src/components/ui/skeleton.tsx  # PAR-UI-042
grep -n "@radix-ui/react-tooltip" ui/src/components/ui/tooltip.tsx && grep -n "side" ui/src/components/ui/tooltip.tsx  # PAR-UI-043
grep -n "paginationRange" ui/src/components/ui/pagination.tsx && grep -n "aria-current" ui/src/components/ui/pagination.tsx  # PAR-UI-044
grep -n "/api/locale" ui/src/components/ui/language-switcher.tsx && grep -n "DEFAULT_LOCALES" ui/src/components/ui/language-switcher.tsx  # PAR-UI-045
grep -n "useThemeStore" ui/src/components/ui/theme-toggle.tsx && grep -n "system" ui/src/components/ui/theme-toggle.tsx  # PAR-UI-046
grep -n "ThemeToggle" ui/src/components/layout/header.tsx && grep -n "LanguageSwitcher" ui/src/components/layout/header.tsx  # T8 wiring
grep -n "RuntimeI18nProvider" ui/src/routes/__root.tsx && ! grep -q "I18nMount" ui/src/routes/__root.tsx && echo OK  # PAR-UI-027 mount
grep -n "Inter" ui/src/index.css   # PAR-UI-027 font — lines 1 and 16, untouched (w6-a evidence, §1.1)
```

**Negative proofs (freeze + scope) — use w6-b commit-range, not <base>..HEAD (see §7)**
```bash
# Use <first-w6-b>^..<last-w6-b> range (excludes w6-d commits on same branch)
git diff <first-w6-b>^..<last-w6-b> --name-only -- ui/src/stores/ ui/src/hooks/ ui/src/lib/ | wc -l   # = 0 (w6-a freeze intact)
git diff <first-w6-b>^..<last-w6-b> --name-only -- ui/src/providers/ | grep -v 'i18n\.tsx' | wc -l  # = 0 (w6-b never touches providers/ files other than importing i18n.tsx)
git diff <base>..HEAD --name-only -- internal/ ui/package.json ui/package-lock.json ui/vite.config.ts ui/playwright.config.ts ui/components.json ui/src/index.css | wc -l   # = 0
git diff <base>..HEAD --name-only -- 'ui/src/routes/' | grep -v '__root.tsx' | wc -l   # = 0 (no routes touched)
git diff <base>..HEAD --name-only -- ui/e2e/ | grep -v 'components.spec.ts\|navigation.spec.ts' | wc -l   # = 0 (mocks + other specs untouched)
git diff <base>..HEAD -- ui/src/routes/__root.tsx | grep "^+" | wc -l               # must be ≤ 10 (import + wrap only; count includes the +++ header line)
git diff <base>..HEAD -- ui/src/components/layout/header.tsx | grep "^+" | wc -l    # must be ≤ 12 (2 imports + 2 slot fills; count includes the +++ header line)
git diff <base>..HEAD -- ui/e2e/navigation.spec.ts | grep "^+" | wc -l              # must be ≤ 10 (test-4 slot assertions only; count includes the +++ header line)
ls ui/src/components/ui/switch.tsx 2>/dev/null | wc -l   # = 0 (T2 STEP(b) harvested it into toggle.tsx and deleted it pre-commit)
grep -rn "i18next" ui/src/components/ui/ | wc -l          # = 0 (no i18n impl here)
git log --oneline <base>..HEAD -- ui/src/components/layout/header.tsx ui/src/routes/__root.tsx | wc -l   # = 1 (exactly ONE commit touches the frozen pair)
```

---

## 6. Out of scope (restated, binding)

No pages or routes; no domain composites; no i18n catalog/runtime/Go endpoint
and no `ui/src/providers/i18n.tsx` in any form — creation, placeholder, or
edit (w6-d); no logout/donate wiring (w6-c/w6-j); no store/hook/lib changes;
no dependency additions; no playwright/vite/components.json config changes; no
`ui/src/index.css` changes; no mocks-dir edits; no test-harness route or
Storybook; no Go changes; no spec-mock adjudication (backend mismatch →
escalate, T8). Tooltip/ConfirmModal open-state browser assertions are
deliberately deferred to consuming page specs (§1 Test-strategy note) —
absence here is a recorded decision, not a gap to "fix" mid-plan.

## 7. Diff-gate scope

**IMPORTANT — parallel-main scoping**: Since w6-d merges on main before T8,
a broad `git diff <base>..HEAD` range sweeps in w6-d commits and their files.
The diff gate MUST be scoped to w6-b's OWN commits only. The orchestrator
identifies w6-b commits with:
`git log --oneline main | grep "^[0-9a-f]* phase-1/w6-b:" | awk '{print $1}'`
and runs: `git diff <first-w6-b-commit>^..<last-w6-b-commit> -- [file list]`
(same commit-range scoping as w5-f split gate; see w5-f plan disposition).

`git diff <first-w6-b>^..<last-w6-b> --name-only` must be exactly a subset of:

```
ui/src/components/ui/button.tsx          ui/src/components/ui/button.test.tsx
ui/src/components/ui/input.tsx           ui/src/components/ui/input.test.tsx
ui/src/components/ui/select.tsx          ui/src/components/ui/select.test.tsx
ui/src/components/ui/card.tsx            ui/src/components/ui/card.test.tsx
ui/src/components/ui/modal.tsx           ui/src/components/ui/modal.test.tsx
ui/src/components/ui/confirm-modal.tsx   ui/src/components/ui/confirm-modal.test.tsx
ui/src/components/ui/badge.tsx           ui/src/components/ui/badge.test.tsx
ui/src/components/ui/toggle.tsx          ui/src/components/ui/toggle.test.tsx
ui/src/components/ui/segmented-control.tsx  ui/src/components/ui/segmented-control.test.tsx
ui/src/components/ui/provider-icon.tsx   ui/src/components/ui/provider-icon.test.tsx
ui/src/components/ui/loading.tsx         ui/src/components/ui/loading.test.tsx
ui/src/components/ui/skeleton.tsx        ui/src/components/ui/skeleton.test.tsx
ui/src/components/ui/tooltip.tsx         ui/src/components/ui/tooltip.test.tsx
ui/src/components/ui/pagination.tsx      ui/src/components/ui/pagination.test.tsx
ui/src/components/ui/language-switcher.tsx  ui/src/components/ui/language-switcher.test.tsx
ui/src/components/ui/theme-toggle.tsx    ui/src/components/ui/theme-toggle.test.tsx
ui/e2e/components.spec.ts
ui/src/components/layout/header.tsx      (T8 wiring commit only)
ui/src/routes/__root.tsx                 (T8 wiring commit only)
ui/e2e/navigation.spec.ts                (T8 wiring commit only; test-4 hunk only)
.planning/parity/matrix/9router-ui.md
docs/WORKFLOW.md
```

`ui/src/providers/i18n.tsx` is deliberately ABSENT from this list — it is
w6-d-owned and any w6-b diff touching it is an automatic review REJECT (gate
BLOCKER 2). Any other file outside this list in the diff is likewise an
automatic review REJECT. The three T8 files must appear in exactly one commit
(the §5 `git log … | wc -l` = 1 proof), and that commit may only land after
P9 confirms w6-d has merged. After merge, all of `ui/src/components/ui/` is
frozen for Wave 6 — every later plan consumes, never edits (MAP decision 7) —
and the w6-a foundation freeze is total: the decision-9 exception is now SPENT.

## Plan gate disposition (closed by decision after 3 cycles — 2026-06-12)

**Cycle 1 REJECT** — REAL: shadcn generation ran before failing tests (fixed: strict TDD,
all test files committed red before any component); T8 wiring created i18n.tsx owned by
w6-d (fixed: w6-b imports only — never creates — i18n.tsx); PAR-UI-027 HAVE claimed without
evidence that Inter font was already delivered (fixed: w6-a CSS evidence cited). Also fixed:
Button icon prop added; build-excludes-test-files citation added; spec-mock adjudication
removed.

**Cycle 2 REJECT** — REAL: diff gate negative proof for providers/ would fail after w6-d
merge because grep pattern was too broad (fixed: grep -v 'i18n\.tsx' added). FALSE: MAJOR 3
(Tooltip deferred behavior) — open-state browser assertions are a known recorded deferral
per WAVE-6-MAP architecture; Radix Tooltip handles position/color internally; page spec
coverage is the correct exercise point.

**Cycle 3 REJECT** — REAL: diff gate still used `<base>..HEAD` range which sweeps in w6-d
commits (fixed: §7 now mandates w6-b commit-range scoping using `git log | grep phase-1/w6-b:`
to isolate w6-b commits, matching w5-f split-gate precedent); shadcn non-determinism concern
(fixed: determinism note added — generated code is committed verbatim; reviewed artifact is
the committed diff). FALSE: BLOCKERs 1+2 (ownership ambiguity) — w6-b ownership is explicit
throughout (§3 names every file; FORBIDDEN lists i18n.tsx explicitly); the ambiguity was only
in the diff gate range, now resolved. FALSE: MAJOR 1 (T1 red spec violates binary acceptance)
— T1 writes a failing spec which is TDD by definition; the binary acceptance §5 covers the
FINAL state (all tests green after T8), not each individual commit; same pattern as w6-a's
navigation.spec.ts TDD red start. FALSE: MAJOR 3 again (Tooltip deferred) — same rebuttal as
cycle 2.

Plan is actionable for kimi dispatch after w6-a and w6-d are merged.

## Runtime dispositions (orchestrator, 2026-06-13)

Preconditions P1–P9 all PASS at dispatch time. Recorded substitutions (binding,
per plan header §line 13-15 and T8 prose):

- **`<base>` = `bb072fa`** (current `git rev-parse HEAD`; main advanced past the
  authored `a5de2ad` because w6-d merged). Use `bb072fa` everywhere §5 references
  `<base>`.
- **w6-d i18n provider export is `I18nProvider`** (NOT `RuntimeI18nProvider`).
  Verified in merged `ui/src/providers/i18n.tsx:28`. Per T8 prose, T8 imports the
  ACTUAL name: `import { I18nProvider } from "@/providers/i18n"`. The §5 grep at
  line 457 (`grep "RuntimeI18nProvider"`) is stale text — the binding acceptance
  is that `__root.tsx` imports `I18nProvider` and no longer references `I18nMount`.
- **Locale endpoint matches the e2e mock** — no T8 escalation. `internal/admin/locale.go`
  POST `/api/locale` accepts `{locale}` and returns `{data:{locale},error:null}`;
  cookie `locale=<code>; Path=/; SameSite=Lax`. Test 4's `{locale:<code>}` body +
  `{data:{locale:'pt-BR'},error:null}` mock are consistent.
- P8 clean-tree: untracked local artifacts (`.serena/`, `.claude/scheduled_tasks.lock`)
  were added to `.gitignore`; worker MUST use explicit `git add <file>` (never
  `git add -A`) so no tooling artifact is swept into a commit.
