# Wave 6 — Dashboard UI: micro-plan index (Stage-1 scope)

Author: Fable 5. Orchestrator: Sonnet (Claude Code session). Implementers: kimi.
Gates: gpt-5.5. **Non-authorizing INDEX** (like WAVE-2/3/4/5-MAP). Frozen ref
@ 827e5c3 (same 9router checkout as W5). Depends on Waves 0–5 — COMPLETE.
Governance: `CLI_ORCHESTRATOR.md` (W6+ rule; HANDOFF.md no longer governs).
Matrices: `matrix/9router-ui.md` (133 rows total: 128 from 9router source —
117 MISSING, 2 PARTIAL, 9 HAVE — plus 5 g0router-specific EXTRA rows
PAR-UI-129..133) plus W5 carry-forwards PAR-USAGE-036/037, PAR-ROUTE-030
(KeyIDs half), PAR-ROUTE-057/058, and PAR-PR-339 (combo list UI).

This is the largest wave. It is also the first predominantly-TypeScript wave:
the TDD unit is the **Playwright e2e spec** (mock-backed, `ui/e2e/mocks/`),
which plays the role `_test.go` files play on the Go side — spec first, see it
fail, minimum component code to pass. `npm run build` joins
`go test ./... && go vet ./...` as a per-commit gate.

**Non-authorizing index note**: This MAP is the dispatch index; it is not
a micro-plan. Binary acceptance criteria, exact task steps, precondition
greps, and stop conditions live in individual micro-plan files
(`w6-a.md`, `w6-b.md`, … `w6-m.md`), written by Fable 5 before each
plan's Pi Code dispatch per CLI_ORCHESTRATOR.md §9.3. See WAVE-5-MAP for
precedent — the pattern is identical.

## Architectural decisions

1. **Flat TanStack routes, not `/dashboard/*` nesting.** g0router's router is
   Vite + TanStack Router file-based (`ui/src/routes/*.tsx`, ~30 stubs already
   exist). 9router's `/dashboard/providers` maps to g0router `/providers`, etc.
   PAR-UI route rows flip as **variant-HAVE** with the flat path recorded.
   PAR-UI-121 (Next.js App Router) is SKIP by prior Phase-1 decision.
2. **Data layer = TanStack Query** (installed, PAR-UI-124 HAVE), not 9router's
   raw `fetch()` + local state. PAR-UI-081 flips as variant-HAVE. SSE surfaces
   (PAR-UI-082 usage stream, PAR-UI-083 console) stay raw `EventSource` — no
   Query wrapper around streams.
3. **i18n = react-i18next hook-based**, not 9router's runtime DOM
   MutationObserver (`runtime.js`). Port the 39-locale catalog (PAR-UI-069),
   locale cookie + `POST /api/locale` (PAR-UI-072). PAR-UI-070/071 flip
   PARTIAL (variant: provider-level re-render on language change replaces DOM
   re-processing on route change). Cleaner under React 19 concurrent rendering.
4. **The e2e mock harness is the app contract.** 30 spec files + the
   `ui/e2e/mocks/{handlers,seed}/` fixture set already define every page's API
   shape. Every micro-plan writes/extends its spec(s) BEFORE component code.
   Where the spec's mock handler and the real Go API disagree, the Go API wins
   and the mock is corrected in the same plan (mocks mirror reality, never
   lead it).
5. **Backend gaps land inline, in Go, TDD-first** (PAR-UI-087…120 subset).
   Each UI plan that needs a missing endpoint adds it in its own NEW
   `internal/admin/<domain>.go` (+ `_test.go` first) file. Route registration
   in `internal/server/routes_admin.go` is **merge-serialized**: it is a hot
   file; only one in-flight plan may hold an unmerged edit to it at any time
   (W3/W4/W5 lesson). No mocks on the Go side — fakes via interfaces, as ever.
6. **`routeTree.gen.ts` is generated — never hand-edited.** Plans that ADD a
   route file (w6-c `callback.tsx`, w6-i `translator.tsx`, w6-l `skills.tsx`)
   regenerate it via the Vite plugin; any merge conflict on it is resolved by
   regeneration, never by manual merge. At most one plan per concurrency wave
   adds a new route file (enforced in the impl order below).
7. **shadcn/ui generation**: `components.json` exists (PAR-UI-125 HAVE) but no
   components are generated. w6-b generates the primitive set into
   `ui/src/components/ui/` once; all later plans CONSUME those files and never
   edit them. Page-specific composites live under
   `ui/src/components/<domain>/`, owned by the page plan.
8. **Shared foundation freeze**: `__root.tsx`, layout components, all six
   Zustand stores, theming CSS, and the `ui/src/lib/` fetch/util helpers are
   created in w6-a and FROZEN afterward — later plans import, never modify.
   A needed change to a frozen file is a serial follow-up commit through the
   orchestrator, not an in-plan edit.
9. **Header slot / component ordering**: w6-a creates the header component
   with **import-ready skeleton slots** for LanguageSwitcher and ThemeToggle
   (rendered as `null` placeholders). w6-b (foundation wave, runs after w6-a)
   creates the real `LanguageSwitcher` and `ThemeToggle` components and wires
   them into those slots as its final commit step — this is the ONLY exception
   to the frozen-after-merge rule for the header file. After w6-b merges the
   wiring commit, header is frozen. No other plan touches it.

## Stage-1 scope decisions

- **IN — g0router EXTRA rows (PAR-UI-129…133)**: These are five g0router-
  specific matrix entries beyond 9router's source rows.
  - PAR-UI-129 (Playwright e2e suite): already HAVE/EXTRA — the suite exists.
  - PAR-UI-130 (g0router-specific routes): IN — routes `/connections`,
    `/virtual-keys`, `/routing-rules`, `/teams`, `/audit`, `/feature-flags`,
    `/guardrails`, `/prompts`, `/model-limits`, `/alerts`, `/mcp`, `/mcp/tools`,
    `/endpoint`, `/tunnels`. Distributed: w6-e (`/connections`), w6-f
    (`/virtual-keys`, `/endpoint`), w6-h (`/routing-rules`, `/model-limits`),
    w6-k (`/teams`, `/audit`, `/feature-flags`, `/guardrails`, `/prompts`,
    `/alerts`), w6-l (`/mcp`, `/mcp/tools`), w6-m (`/tunnels`).
  - PAR-UI-131 (g0router-specific API contracts): IN — APIs exercised by
    the page plans above; no separate implementation step required.
  - PAR-UI-132 (g0router auth endpoints): IN — Go endpoints exist from W3
    (`internal/auth/`, `internal/admin/` auth handlers); w6-c consumes them.
  - PAR-UI-133 (g0router chat behavior): IN — covered by w6-i chat page.
- **IN — PAR-UI-103** `POST /api/version/shutdown`: simple Go endpoint; no
  existing e2e spec but straightforward to add in w6-j alongside `GET /api/version`.
- **IN — MITM / proxy-pools / tunnels PAGES (UI halves only)**: PAR-UI-013,
  PAR-UI-019, plus the g0router tunnels page. Route stubs and mock-backed e2e
  (`mitm.spec.ts`, `proxy-pools.spec.ts`, `tunnels.spec.ts`,
  `mocks/handlers/{mitm,proxy-pools,tunnels}.ts`) already exist, so the UI is
  buildable and testable now. The backend engines remain W7 (unchanged W5
  disposition); these rows flip PARTIAL (UI half) in W6, HAVE in W7.
- **PARTIAL completions (2)**: PAR-UI-073 Tailwind v4 `@theme inline` semantic
  tokens; PAR-UI-079 Material Symbols integration — both close in w6-a.
- **Variant implementations**: PAR-UI-070/071 (i18n, decision 3 above);
  PAR-UI-081 (TanStack Query, decision 2); PAR-UI-086 (translator editor:
  **CodeMirror or textarea fallback — Monaco is NOT added to package.json**);
  PAR-UI-005 (`/dashboard` is itself the overview page; redirect semantics
  satisfied by `/` → `/dashboard`).
- **SKIP (no parity value)**: PAR-UI-004 `/landing` (marketing page, no e2e,
  no backend); PAR-UI-121 (Next.js App Router — Vite+TanStack is the Phase-1
  decision); PAR-UI-061 NineRemotePromoModal (9router self-promo, meaningless
  in g0router).
- **OUT → W7 (platform wave)**: PAR-UI-106/107/108 deploy endpoints
  (Vercel/Cloudflare/Deno) — same bucket as tunnel/MITM backends.
- **OUT → W7/S2 (no e2e contract, no route stub, no mock, no Stage-1
  backend)**: PAR-UI-014/015 cli-tools pages, PAR-UI-022/023/024
  media-providers pages. Fable 5 scope call: zero in-tree evidence of a
  contract; deferring is cheaper than inventing one mid-wave.
- **W5 carry-forwards land here**: PAR-USAGE-036 (UsageStats), PAR-USAGE-037
  (RequestLogger) in w6-g; PAR-ROUTE-057/058 + PAR-ROUTE-030 KeyIDs half in
  w6-pre (Go-only, parallel track); PAR-PR-339 combo list UI in w6-h.

## Micro-plan index (14 plans)

| Plan | Scope | Rows | Key ref/e2e evidence | Depends |
|---|---|---|---|---|
| **w6-pre** | **Go-only routing carry-forward**: settings-driven model catalog (dynamic node registration from DB settings), `GetTestByKind` live HTTP ping per model, VK `ProviderConfig.KeyIDs` threading into `SelectConnection` dispatch (preferredConnID path); admin catalog endpoints | PAR-ROUTE-057, PAR-ROUTE-058, PAR-ROUTE-030 (KeyIDs half → HAVE) | in-tree `internal/inference/selection.go`, `internal/store/settings.go`; w4-f disposition (GetTestByKind deferral); w5-g closure record (KeyIDs PARTIAL) | — (runs ∥ w6-a; Go track) |
| **w6-a** | **Shell + theming + stores (UI foundation)**: root layout (Inter, ThemeProvider, I18nProvider mount point), dashboard layout (sidebar+header+toasts), sidebar (logo, nav, update-check badge), header (breadcrumbs, search, auth badge, theme/lang toggles, logout button slot), mobile sidebar overlay, toast system (Zustand+sonner), Tailwind v4 `@theme inline` semantic tokens, brand colors (#E56A4A / #FDFAF6 / #1a1a1a), themeStore (persist, key "theme"), useTheme (prefers-color-scheme, useSyncExternalStore), ThemeProvider initTheme, `.dark` on `<html>`, Material Symbols, ALL six Zustand stores, `/` → `/dashboard` redirect, shared `ui/src/lib/` fetch+util helpers | PAR-UI-001, PAR-UI-026, PAR-UI-027, PAR-UI-028, PAR-UI-029, PAR-UI-030, PAR-UI-031, PAR-UI-073→HAVE, PAR-UI-074, PAR-UI-075, PAR-UI-076, PAR-UI-077, PAR-UI-078, PAR-UI-079→HAVE, PAR-UI-080 | `e2e/navigation.spec.ts`, `e2e/comprehensive.spec.ts` (shell assertions); ref `src/app/dashboard/layout.js`, `src/stores/*` | w6-pre N/A (disjoint); FIRST in UI track, ALONE |
| **w6-b** | **Core shared primitives** (generate shadcn set + custom): Button, Input, Select, Card, Modal, ConfirmModal, Badge, Toggle, SegmentedControl, ProviderIcon (PNGs in `public/providers/`), Loading/Spinner/Skeleton, Tooltip, Pagination, LanguageSwitcher, ThemeToggle | PAR-UI-032, PAR-UI-033, PAR-UI-034, PAR-UI-035, PAR-UI-036, PAR-UI-037, PAR-UI-038, PAR-UI-039, PAR-UI-040, PAR-UI-041, PAR-UI-042, PAR-UI-043, PAR-UI-044, PAR-UI-045, PAR-UI-046 (15 rows) | `components.json` (PAR-UI-125 HAVE); ref `src/components/ui/*`; consumed by every page spec | w6-a |
| **w6-c** | **Auth pages**: `/login` (password + OIDC form, rate-limit countdown), `/callback` (NEW route file; OAuth callback, postMessage + BroadcastChannel), `POST /api/auth/login`, OIDC PKCE/state/nonce, `GET /api/auth/status` on mount, `POST /api/auth/logout` wired to header button. Go auth endpoints already exist from W3 (`internal/auth/`, `internal/admin/` auth handlers) — this plan is UI-only. | PAR-UI-002, PAR-UI-003, PAR-UI-065, PAR-UI-066, PAR-UI-067, PAR-UI-068 | `e2e/auth.spec.ts`, `mocks/handlers/auth.ts`, `mocks/seed/auth.ts`; ref `src/app/login/page.js`, `src/app/callback/page.js` | **w6-b** (needs Button/Input components; page wave 1) |
| **w6-d** | **i18n**: 39-locale LOCALES catalog + resource files, react-i18next runtime wiring (variant per decision 3), locale cookie, Go `POST /api/locale` endpoint (new `internal/admin/locale.go`, TDD) | PAR-UI-069, PAR-UI-070→PARTIAL(variant), PAR-UI-071→PARTIAL(variant), PAR-UI-072 | `mocks/handlers/locale.ts`; ref `src/lib/i18n/locales.js` (39 entries), `runtime.js` (NOT ported) | w6-a (∥ w6-b, w6-c); serial slot on routes_admin.go after w6-pre merges |
| **w6-e** | **Providers + connections + models cluster**: provider cards page (OAuth/Free/API-Key/Compatible groups), provider new/detail flows, connections page, models page; modals: OAuthModal, EditConnectionModal, ManualConfigModal, CursorAuthModal/KiroAuthModal, IFlowCookieModal, GitLabAuthModal, AddCustomEmbeddingModal, NoAuthProxyCard, ProviderInfoCard; Go gaps: provider-shaped read API over connections (PAR-UI-087/088/089), batch connection test (PAR-UI-090), new `internal/admin/providers.go` | PAR-UI-007, PAR-UI-008, PAR-UI-009, PAR-UI-051, PAR-UI-052, PAR-UI-053, PAR-UI-058, PAR-UI-059, PAR-UI-060, PAR-UI-062, PAR-UI-063, PAR-UI-064, PAR-UI-087, PAR-UI-088, PAR-UI-089, PAR-UI-090 + PAR-UI-130 subset (`/connections`) | `e2e/providers.spec.ts`, `e2e/connections.spec.ts`, `e2e/models.spec.ts`, `mocks/handlers/{providers,connections,models}.ts` | w6-b, w6-c (OAuth popup contract); page wave 1 |
| **w6-f** | **Endpoint + keys + virtual-keys cluster**: endpoint config page (base URL, API key management), API keys page (PAR-UI-115 `POST /api/keys`), virtual-keys page (CRUD, budget/RPM display, **KeyIDs pinning editor** consuming w6-pre's catalog API), ModelSelectModal (PAR-UI-049 custom/disabled model lists PAR-UI-117/118/119/120); Go gaps: PAR-UI-109/110/111 (provider-nodes endpoints, new `internal/admin/nodes.go`) | PAR-UI-006, PAR-UI-049, PAR-UI-109, PAR-UI-110, PAR-UI-111, PAR-UI-115, PAR-UI-117, PAR-UI-118, PAR-UI-119, PAR-UI-120 + PAR-UI-130 subset (`/virtual-keys`, `/endpoint`) | `e2e/keys.spec.ts`, `e2e/virtual-keys.spec.ts`, `mocks/handlers/{keys,virtual-keys}.ts`; VK backend = w5-g | w6-b, **w6-pre** (catalog API for ModelSelectModal/KeyIDs); page wave 2 |
| **w6-g** | **Usage + logs + quota + pricing cluster** (W5 APIs, zero new Go): dashboard overview page, UsageStats component (SSE `EventSource` on `/api/usage/stream`, period selector, provider topology via @xyflow/react, usage table), RequestLogger (3s poll), usage page (overview/logs/details tabs), traffic page, quota page, pricing settings page + PricingModal, recharts charts | PAR-UI-005(variant), PAR-UI-011, PAR-UI-012, PAR-UI-025, PAR-UI-047, PAR-UI-048, PAR-UI-057, PAR-UI-081(variant→HAVE), PAR-UI-082, PAR-UI-095, PAR-UI-096, PAR-USAGE-036, PAR-USAGE-037 | `e2e/{dashboard,usage,traffic,quota,pricing}.spec.ts`, `mocks/handlers/{usage,quota,pricing,streams}.ts`; W5 APIs (w5-d/w5-e) | w6-b; page wave 1 |
| **w6-h** | **Combos + routing cluster**: combos page with @dnd-kit reordering, ComboFormModal (DnD member list), g0router combo list UI (PAR-PR-339), routing-rules page, model-limits page, aliases page; Go gaps: combo API endpoints (PAR-UI-091..094) already covered by w4-e backend — no new Go needed | PAR-UI-010, PAR-UI-050, PAR-UI-091, PAR-UI-092, PAR-UI-093, PAR-UI-094, PAR-UI-116, PAR-PR-339 + PAR-UI-130 subset (`/routing-rules`, `/model-limits`) | `e2e/{combos,routing-rules,model-limits,aliases}.spec.ts`, `mocks/handlers/{combos,routing-rules,model-limits,aliases}.ts`; combo engine = w4-f/w5-pre | w6-b; page wave 1 |
| **w6-i** | **Chat + console + translator cluster**: basic-chat page (@ai-sdk/react against gateway), console page (SSE live log via `EventSource`), translator debug page (NEW route file; CodeMirror/textarea variant, NO Monaco dep) | PAR-UI-016, PAR-UI-017, PAR-UI-018, PAR-UI-083, PAR-UI-086(variant) | `e2e/chat.spec.ts`, `e2e/console.spec.ts`, `mocks/handlers/{chat-sessions,inference,logs}.ts`, `mocks/seed/console-logs.ts` | w6-b; page wave 1 (only new-route-file plan in wave 1) |
| **w6-j** | **Settings/profile + version cluster**: settings page (theme, lang, OIDC config, password change, DB info), ChangelogModal, DonateModal, sidebar update-checker data source; Go gaps: `GET /api/version` (PAR-UI-102), `POST /api/version/shutdown` (PAR-UI-103), settings API endpoints (PAR-UI-097..101), new `internal/admin/version.go` | PAR-UI-021, PAR-UI-055, PAR-UI-056, PAR-UI-097, PAR-UI-098, PAR-UI-099, PAR-UI-100, PAR-UI-101, PAR-UI-102, PAR-UI-103 | `e2e/settings.spec.ts`, `mocks/handlers/{settings,version}.ts` | w6-b, w6-a (header/sidebar slots); page wave 2; serial slot on routes_admin.go |
| **w6-k** | **Governance pages (g0router EXTRA)**: teams, audit, feature-flags, guardrails, prompts, alerts — backend complete (phases 13–19), pure UI | PAR-UI-130 subset (`/teams`, `/audit`, `/feature-flags`, `/guardrails`, `/prompts`, `/alerts`) + PAR-UI-131, PAR-UI-132 (auth endpoints for user management) | `e2e/{teams,audit,feature-flags,guardrails,prompts,alerts}.spec.ts` + matching mocks/handlers + seeds | w6-b; page wave 2 |
| **w6-l** | **MCP + skills cluster**: mcp page, mcp/tools page, McpMarketplaceModal, skills page (NEW route file; mock handler+seed exist) | PAR-UI-020, PAR-UI-054 + PAR-UI-130 subset (`/mcp`, `/mcp/tools`) | `e2e/mcp.spec.ts`, `mocks/handlers/mcp.ts`, `mocks/{handlers,seed}/skills.ts`; MCP gateway backend in-tree | w6-b; page wave 2 (only new-route-file plan in wave 2) |
| **w6-m** | **Platform pages — UI client-side halves**: mitm page, proxy-pools page, tunnels page; UI wires up to mock-backed e2e for all API contracts (PAR-UI-104/105 proxy-pool list/create, PAR-UI-112..114 tunnel status/enable/disable); Go backends for ALL of these remain W7 — rows PAR-UI-013/019/104/105/112/113/114 flip **PARTIAL** in W6 (UI half done, mock-backed specs green), HAVE in W7 when Go backends land | PAR-UI-013→PARTIAL, PAR-UI-019→PARTIAL, PAR-UI-104→PARTIAL, PAR-UI-105→PARTIAL, PAR-UI-112→PARTIAL, PAR-UI-113→PARTIAL, PAR-UI-114→PARTIAL + tunnels page | `e2e/{mitm,proxy-pools,tunnels}.spec.ts`, `mocks/handlers/{mitm,proxy-pools,tunnels}.ts` | w6-b; page wave 2 |

## Ownership tracks (W3/W4/W5 lesson: NO shared files across live jobs)

Go track and UI track are fully disjoint (`internal/` vs `ui/`) and may always
run concurrently. Within each track:

- **w6-pre** (Go): `internal/inference/selection.go` (+`selection_test.go`),
  `internal/inference/catalog.go`/`catalog_test.go` (new),
  `internal/store/settings.go` (+test), `internal/admin/catalog.go` (+test,
  new), `internal/server/routes_admin.go` (FIRST holder of the serial slot).
- **w6-a** (UI, alone first): `ui/src/routes/__root.tsx`, `ui/src/styles/`
  (or `index.css` Tailwind v4 tokens), `ui/src/components/layout/`
  (`sidebar.tsx`, `header.tsx`, `mobile-sidebar.tsx`, `toaster.tsx`),
  `ui/src/stores/` (ALL: `theme.ts`, `user.ts`, `provider.ts`, `settings.ts`,
  `notification.ts`, `header-search.ts`), `ui/src/hooks/use-theme.ts`,
  `ui/src/providers/theme.tsx`, `ui/src/lib/` (`api.ts`, `utils.ts`),
  `ui/e2e/navigation.spec.ts`, `ui/e2e/comprehensive.spec.ts`. All FROZEN
  after merge (decision 8).
- **w6-b**: `ui/src/components/ui/*` (exact: `button.tsx`, `input.tsx`,
  `select.tsx`, `card.tsx`, `modal.tsx`, `confirm-modal.tsx`, `badge.tsx`,
  `toggle.tsx`, `segmented-control.tsx`, `provider-icon.tsx`, `loading.tsx`,
  `skeleton.tsx`, `tooltip.tsx`, `pagination.tsx`, `language-switcher.tsx`,
  `theme-toggle.tsx` + component tests). FROZEN after merge.
- **w6-c**: `ui/src/routes/login.tsx`, `ui/src/routes/callback.tsx` (NEW),
  `ui/src/lib/auth.ts` (new file — not w6-a's frozen `api.ts`),
  `ui/e2e/auth.spec.ts`, `ui/e2e/mocks/handlers/auth.ts` (corrections only).
- **w6-d**: `ui/src/i18n/**` (new dir: `index.ts`, `locales.ts`,
  `locales/*.json` ×39), `ui/src/providers/i18n.tsx`; Go:
  `internal/admin/locale.go` (+test, new) + serialized routes_admin.go slot.
- **w6-e**: `ui/src/routes/providers.tsx`, `ui/src/routes/connections.tsx`,
  `ui/src/routes/models.tsx`, `ui/src/components/providers/**` (all listed
  modals/cards), `ui/e2e/{providers,connections,models}.spec.ts` + their
  mocks; Go: `internal/admin/providers.go` (+test, new) + serialized
  routes_admin.go slot.
- **w6-f**: `ui/src/routes/endpoint.tsx`, `ui/src/routes/keys.tsx`,
  `ui/src/routes/virtual-keys.tsx`, `ui/src/components/keys/**` (incl.
  `model-select-modal.tsx`), `ui/e2e/{keys,virtual-keys}.spec.ts` + mocks.
- **w6-g**: `ui/src/routes/dashboard.tsx`, `ui/src/routes/usage.tsx`,
  `ui/src/routes/logs.tsx`, `ui/src/routes/traffic.tsx`,
  `ui/src/routes/quota.tsx`, `ui/src/routes/pricing.tsx`,
  `ui/src/components/usage/**` (`usage-stats.tsx`, `request-logger.tsx`,
  charts, topology, `pricing-modal.tsx`),
  `ui/e2e/{dashboard,usage,traffic,quota,pricing}.spec.ts` + mocks. No Go.
- **w6-h**: `ui/src/routes/combos.tsx`, `ui/src/routes/routing-rules.tsx`,
  `ui/src/routes/model-limits.tsx`, `ui/src/routes/aliases.tsx`,
  `ui/src/components/combos/**`, `ui/src/components/routing/**`,
  `ui/e2e/{combos,routing-rules,model-limits,aliases}.spec.ts` + mocks. No Go.
- **w6-i**: `ui/src/routes/chat.tsx`, `ui/src/routes/console.tsx`,
  `ui/src/routes/translator.tsx` (NEW), `ui/src/components/chat/**`,
  `ui/src/components/console/**`, `ui/e2e/{chat,console}.spec.ts` + new
  `ui/e2e/translator.spec.ts` + mocks.
- **w6-j**: `ui/src/routes/settings.tsx`, `ui/src/components/settings/**`
  (incl. `changelog-modal.tsx`, `donate-modal.tsx`),
  `ui/e2e/settings.spec.ts` + mocks; Go: `internal/admin/version.go` (+test,
  new) + serialized routes_admin.go slot.
- **w6-k**: `ui/src/routes/{teams,audit,feature-flags,guardrails,prompts,alerts}.tsx`,
  `ui/src/components/governance/**`,
  `ui/e2e/{teams,audit,feature-flags,guardrails,prompts,alerts}.spec.ts` +
  mocks. No Go.
- **w6-l**: `ui/src/routes/mcp.tsx`, `ui/src/routes/mcp.tools.tsx`,
  `ui/src/routes/skills.tsx` (NEW), `ui/src/components/mcp/**`,
  `ui/e2e/mcp.spec.ts` + new `ui/e2e/skills.spec.ts` + mocks.
- **w6-m**: `ui/src/routes/{mitm,proxy-pools,tunnels}.tsx`,
  `ui/src/components/platform/**`,
  `ui/e2e/{mitm,proxy-pools,tunnels}.spec.ts` + mocks. No Go.

Cross-cutting rules:
- `ui/src/routeTree.gen.ts`: generated only; conflicts resolved by regen
  (decision 6). New-route plans (c, i, l) sit in different concurrency waves.
- `ui/e2e/mocks/handlers/index.ts` and `mocks/seed/index.ts`: hot files —
  a plan may append its own registration line; merges are orchestrator-ordered
  (rebase-and-regen, trivial append conflicts only).
- `internal/server/routes_admin.go`: ONE unmerged holder at a time, in order
  w6-pre → w6-d → w6-e → w6-j.
- Frozen sets (w6-a, w6-b outputs): consume-only for all later plans.

## Impl order

```
Go track:  w6-pre ──────────────────────────────▶ (done early; unblocks w6-f)
UI track:  w6-a ALONE
           → w6-b ∥ w6-d                  (component + i18n wave)
           → w6-c ∥ w6-e ∥ w6-g ∥ w6-h ∥ w6-i   (page wave 1)
           → w6-f ∥ w6-j ∥ w6-k ∥ w6-l ∥ w6-m   (page wave 2)
```

Reasons:
- **w6-pre ∥ w6-a**: disjoint trees (`internal/` vs `ui/`); starting both
  immediately puts the catalog API in place well before w6-f needs it.
- **w6-a alone**: it owns the files every other UI plan imports (root, layout,
  stores, lib, theming). Nothing UI-side can write spec assertions about the
  shell until the shell contract exists.
- **Component + i18n wave (b∥d)**: w6-b generates the shared UI primitive set;
  w6-d wires i18n. These are disjoint (components/ui vs i18n/). w6-c is moved
  to page wave 1 because auth pages require Button/Input from w6-b — running
  w6-c in parallel with w6-b would mean auth pages can't import the components
  they depend on. w6-d has no component dependency so it runs here.
- **Page wave 1 (c∥e∥g∥h∥i)**: auth + the four biggest page clusters, fully
  disjoint route and component ownership; only w6-e holds a Go slot (after
  w6-d's merges); only w6-i adds a route file.
- **Page wave 2 (f∥j∥k∥l∥m)**: w6-f deliberately deferred to wave 2 so
  w6-pre's catalog endpoints are merged and exercised; only w6-j holds a Go
  slot; only w6-l adds a route file.
- Orchestrator may run waves at ≤4 concurrent jobs — split wave 1 as
  (c∥e∥g) → (h∥i) or wave 2 as (f∥j∥k) → (l∥m); ownership permits it.

## Protocol (W5 protocol + UI gates)

Per micro-plan: plan → gpt-5.5 plan gate (≤3 cycles → decide) → kimi TDD impl
→ gates → scoped diff gate (commit-bounded; live-tree verification before
closure; known gate artifact still applies: diff-only analysis flags
pre-existing imports — build is ground truth) → merge → flip matrix rows →
`docs/WORKFLOW.md` update. Commits: `phase-1/w6-X: <description>`.

TDD shape per side:
- **UI**: write/extend the plan's Playwright spec(s) + mock handlers FIRST,
  run them, see them fail, then minimum component code to green. Component
  unit tests where a spec can't reach (stores, hooks).
- **Go**: `_test.go` first in the new `internal/admin/<domain>.go` package
  file, see it fail, minimum handler to pass. No mocks — fakes via interfaces.

Per-commit gates (every commit, both halves):
`go test ./... && go vet ./...` green, `go build ./...` green,
`cd ui && npm run build` green, and the plan's scoped
`npx playwright test e2e/<plan-specs>` green. Full
`npx playwright test` runs at plan closure and at each wave boundary.

## Out of Wave-6 scope (explicit)

- PAR-UI-004 `/landing` — SKIP (no e2e, no backend, no parity value).
- PAR-UI-121 Next.js App Router — SKIP (Vite+TanStack is the standing
  Phase-1 decision).
- PAR-UI-061 NineRemotePromoModal — SKIP (9router self-promotion).
- PAR-UI-086 Monaco Editor — variant only; the Monaco dependency itself is
  never added.
- PAR-UI-106/107/108 deploy endpoints (Vercel/Cloudflare/Deno) — W7.
- MITM / tunnel / proxy-pool **backends** — W7 (UI halves ship here as
  PARTIAL; see w6-m).
- PAR-UI-014/015 cli-tools pages, PAR-UI-022/023/024 media-providers pages —
  W7/S2 (no route stub, no e2e, no mock, no Stage-1 backend).
- 9router runtime DOM-translation engine (`runtime.js` MutationObserver) —
  permanently replaced by the react-i18next variant (070/071 stay PARTIAL
  with the variant recorded; revisit only if parity audit demands literal
  behavior).
- Stage-2 provider quota fetchers and other W5-noted deferrals — unchanged.

## Plan gate disposition (closed by decision after 3 cycles — 2026-06-12)

**Cycle 1 REJECT** — REAL: row count wrong (128 vs 133), `/endpoint` missing from
extra-routes list, PAR-UI-047/048 missing from w6-g, PAR-UI-091..094 missing
from w6-h, PAR-UI-097..102 missing from w6-j, PAR-UI-103 justification wrong,
non-canonical row IDs, PAR-UI-112..115 not assigned. All fixed in revision.

**Cycle 2 REJECT** — REAL: PAR-UI-129..133 mislabeled as "routes", w6-f had
PAR-UI-116 duplicate, w6-m endpoint rows contradicted "UI halves only" language,
w6-a header slots ordered before w6-b components. All fixed. FALSE: BLOCKER 1
(no per-plan acceptance criteria — MAP is non-authorizing index; criteria in
individual micro-plan files, identical to WAVE-2/3/4/5-MAP). MAJOR 1 (w6-c auth
Go files missing — auth endpoints exist from W3, w6-c is UI-only per plan text
and per W3 HANDOFF record). MAJOR 3 (evidence claims — MAP is a non-authorizing
index, not a micro-plan; evidence lives in per-plan files per protocol).

**Cycle 3 REJECT** — REAL: w6-c ordering (auth needs w6-b components — fixed,
moved to page wave 1 after w6-b), tunnels page untracked in PAR-UI-130 (fixed,
added to PAR-UI-130 list), PAR-UI-129..133 labeling imprecise (fixed, each row
described individually). FALSE: MAJOR 2 ("30 spec files" vs new specs added —
existing harness is the contract for existing pages; new pages write specs
TDD-first, which is the pattern not a contradiction). MAJOR 3 (PAR-UI-099/100/101
"not in e2e" — these are sub-features of the in-scope settings page PAR-UI-021;
the deferral rule "no e2e, no stub, no mock, no backend" applies to standalone
features with none of those anchors; settings has all four). MINOR (prose padding
— four phrases trimmed in this revision).

Ground-truth verification: `ls ui/e2e/*.spec.ts | wc -l` → 41 files confirmed;
`grep -r "internal/auth" internal/admin/` confirms W3 auth handlers in-tree;
`grep "tunnels" ui/src/routes/tunnels.tsx` confirms stub exists. MAP is
actionable for Fable 5 to begin drafting individual micro-plan files.
