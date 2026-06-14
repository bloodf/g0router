# UI Parity Matrix: 9Router Reference → g0router

Reference: `/Users/heitor/Developer/github.com/bloodf/_refs/9router` (SHA `827e5c3`)
Target: `/Users/heitor/Developer/github.com/bloodf/g0router/ui`

---

## Row Table

| ID | Behavior | Evidence (file:line) | g0router status | Notes |
|---|---|---|---|---|
| PAR-UI-001 | Route `/` redirects to `/dashboard` | `ui/src/routes/__root.tsx` | HAVE | `beforeLoad` throws `redirect({ to: '/dashboard' })` |
| PAR-UI-002 | Route `/login` renders password + OIDC login form | `ui/src/routes/login.tsx`, `ui/src/components/auth/login-form.tsx` | HAVE | w6-c: status-driven password + OIDC login form (`#username`/`#password`, OIDC button when `auth_mode ∈ {oidc,both}`) |
| PAR-UI-003 | Route `/callback` handles OAuth callback via postMessage, BroadcastChannel, localStorage | `ui/src/routes/callback.tsx`, `ui/src/lib/auth.ts` | HAVE | w6-c: `relayOAuthCallback` (postMessage origin-allowlist + BroadcastChannel `oauth_callback` + localStorage) + manual-copy fallback |
| PAR-UI-004 | Route `/landing` marketing page | `src/app/landing/page.js` | MISSING | Not referenced in g0router e2e |
| PAR-UI-005 | Route `/dashboard` alias to `/dashboard/endpoint` | `ui/src/routes/dashboard.tsx` | HAVE (variant) | w6-g §1.5 — `/dashboard` IS the overview page (UsageStats summary + RequestLogger preview); MAP scope decision |
| PAR-UI-006 | Route `/dashboard/endpoint` shows API endpoint config + API key management | `src/app/(dashboard)/dashboard/endpoint/page.js:1-7` | HAVE (variant — w6-f) | flat `/endpoint` route (`ui/src/routes/endpoint.tsx`): base-url panel (origin + `/v1`, copy, sample curl) + compact `<ApiKeysPanel>` against REAL `/api/keys` Go CRUD + custom provider-node modal (plan §1.3/§1.5; `ui/e2e/endpoint.spec.ts`). Go `apiKeyDTO` shape honored, UI extras display-optional (§8 ESC-2) |
| PAR-UI-007 | Route `/dashboard/providers` lists providers in card grid (OAuth, Free, API Key, Compatible) | `ui/src/routes/providers.tsx`, `ui/src/components/providers/provider-card.tsx` | HAVE | w6-e (variant §1.5): flat `/providers` route, grouped sections (OAuth/API-Key/Free/Compatible) of `card-elev` cards from `GET /api/providers/catalog` |
| PAR-UI-008 | Route `/dashboard/providers/new` adds new provider | `ui/src/routes/providers.tsx`, `ui/src/components/providers/manual-config-modal.tsx` | HAVE | w6-e (variant §1.5): in-page "add connection" flow (modal), not a nested `/new` route |
| PAR-UI-009 | Route `/dashboard/providers/[id]` shows provider detail with connections + models | `ui/src/components/providers/provider-detail-panel.tsx` | HAVE | w6-e (variant §1.5): in-page detail panel loading `/api/providers/{id}/connections` + `/models`, not a nested `/[id]` route |
| PAR-UI-010 | Route `/dashboard/combos` lists combos with DnD reordering | `ui/src/routes/combos.tsx`, `ui/src/components/combos/combo-list.tsx`, `ui/src/components/combos/combo-form-modal.tsx` | HAVE (variant) | w6-h §1.3/§1.5 — flat `/combos` route; combo list (PAR-PR-339: shows model names not raw IDs) + ComboFormModal with @dnd-kit member reorder delegating to pure `moveStep` (`ui/src/lib/combo-order.ts`, unit-tested); reorder e2e proof = persisted-order PUT-body intercept |
| PAR-UI-011 | Route `/dashboard/usage` has overview/logs/details tabs with period selector | `ui/src/routes/usage.tsx` | HAVE (variant) | w6-g §1.5 — `/usage` has overview/logs/details tabs + period selector; g0router also splits standalone `/logs` + `/traffic` routes |
| PAR-UI-012 | Route `/dashboard/quota` shows provider limits | `ui/src/routes/quota.tsx` | HAVE (variant) | w6-g §1.4 — provider-limits view (ProviderLimits) from `/api/quota` MOCK; runtime Go aggregation (no `GET /api/quota`) is a serial follow-up (open-questions ESCALATION-1c) |
| PAR-UI-013 | Route `/dashboard/mitm` MITM proxy config | `ui/src/routes/mitm.tsx` | PARTIAL | w6-m §1.3/§8 ESC-1a — flat `/mitm` route: status panel + global enable `Toggle` (POST `/api/mitm/toggle`), per-tool toggle (POST `/api/mitm/tools/{id}`), CA-cert download via plain `fetch`/anchor (GET `/api/mitm/ca-cert` raw PEM, not `{data}`); UI half done vs the registered `/api/mitm/*` MOCK; NO Go backend (W7 follow-up → HAVE in W7) |
| PAR-UI-014 | Route `/dashboard/cli-tools` CLI tool integrations | `src/app/(dashboard)/dashboard/cli-tools/page.js:1-7` | MISSING | Not in g0router e2e |
| PAR-UI-015 | Route `/dashboard/cli-tools/[toolId]` per-tool detail | `src/app/(dashboard)/dashboard/cli-tools/[toolId]/page.js` | MISSING | Not in g0router e2e |
| PAR-UI-016 | Route `/dashboard/basic-chat` hidden chat UI | `ui/src/routes/chat.tsx`, `ui/src/components/chat/chat-window.tsx`, `ui/src/components/chat/chat-message.tsx` | HAVE (variant) | w6-i §1.3/§1.4 — flat `/chat` route; chat page POSTs the REAL gateway route `/v1/chat/completions` and streams the assistant turn via a plain-fetch ReadableStream reader (`streamChatCompletion`, unit-tested); the e2e `inference.ts` mock streams the reply (mock body corrected to JSON.stringify chunks, §1.4) |
| PAR-UI-017 | Route `/dashboard/console-log` live server console | `ui/src/routes/console.tsx`, `ui/src/components/console/console-log-viewer.tsx` | HAVE (variant) | w6-i §1.5/§8 ESCALATION-1 — flat `/console` route; opens `EventSource("/api/console-logs/stream")` via pure `subscribeConsoleLogs` (unit-tested SSE proof) driven by the fixture `MockEventSource`; NO Go backend (mock+fixture surface; runtime SSE endpoint is a serial follow-up) |
| PAR-UI-018 | Route `/dashboard/translator` debug translation flow | `ui/src/routes/translator.tsx`, `ui/src/components/translator/translator-step.tsx`, `ui/src/lib/translator-format.ts` | HAVE (variant) | w6-i §1.6/§1.7/§8 ESCALATION-2 — NEW `/translator` route (regenerates routeTree.gen.ts); 7-step textarea inspector against a NEW self-contained mock (`/api/translator/load`+`/translate`); NO Go backend (serial follow-up) |
| PAR-UI-019 | Route `/dashboard/proxy-pools` proxy pool management with bulk ops | `ui/src/routes/proxy-pools.tsx`, `ui/src/components/platform/proxy-pool-form-modal.tsx`, `ui/src/lib/proxy-pool-form.ts` | PARTIAL | w6-m §1.4/§8 ESC-1b — flat `/proxy-pools` route: list + create/edit `ProxyPoolFormModal` (POST/PUT, pure `toProxyPoolPayload` helper unit-tested), per-pool Test (POST `/api/proxy-pools/{id}/test`), Delete (ConfirmModal DELETE); UI half done vs the registered `/api/proxy-pools*` MOCK; NO Go backend (W7 follow-up → HAVE in W7) |
| PAR-UI-020 | Route `/dashboard/skills` agent skills links | `ui/src/routes/skills.tsx`, `ui/src/lib/skills-format.ts` | HAVE (variant) | w6-l §1.3/§1.7/§8 ESC-1c — NEW `/skills` route (regenerates routeTree.gen.ts); reads the registered `/api/skills` MOCK, groups by `category` via pure `groupSkillsByCategory` (unit-tested), per-skill copy-to-clipboard (`navigator.clipboard.writeText`); NO Go backend (`/api/skills` absent — serial follow-up) |
| PAR-UI-021 | Route `/dashboard/profile` settings (theme, lang, OIDC, password, DB) | `src/app/(dashboard)/dashboard/profile/page.js:23-1140` | HAVE | w6-j variant: `/settings` page with theme/lang/OIDC/password/DB panels; sidebar update-badge data source via `use-version-check.ts` → frozen `setUpdateInfo` (plan §1.3/§1.6) |
| PAR-UI-022 | Route `/dashboard/media-providers/[kind]` media provider kind list | `src/app/(dashboard)/dashboard/media-providers/[kind]/page.js` | MISSING | Not in g0router e2e |
| PAR-UI-023 | Route `/dashboard/media-providers/[kind]/[id]` media provider detail | `src/app/(dashboard)/dashboard/media-providers/[kind]/[id]/page.js` | MISSING | Not in g0router e2e |
| PAR-UI-024 | Route `/dashboard/media-providers/web` web search/fetch combos | `src/app/(dashboard)/dashboard/media-providers/web/page.js` | MISSING | Not in g0router e2e |
| PAR-UI-025 | Route `/dashboard/settings/pricing` pricing management | `ui/src/routes/pricing.tsx` | HAVE | w6-g — `/pricing` table + PricingModal (GET/PATCH/DELETE `/api/pricing`) |
| PAR-UI-026 | Dashboard layout wraps all routes with sidebar + header + toasts | `ui/src/routes/__root.tsx` | HAVE | `ThemeProvider > I18nMount > flex shell [Sidebar | MobileSidebar | Header + Outlet] + Toaster` |
| PAR-UI-027 | Root layout loads Inter font, ThemeProvider, RuntimeI18nProvider | `ui/src/routes/__root.tsx` | HAVE | Inter font (w6-a `ui/src/index.css:1,16`); `ThemeProvider > I18nProvider` shell; `I18nProvider` from w6-d mounted by w6-b T8 |
| PAR-UI-028 | Sidebar renders traffic lights, logo, nav items, media accordion, update checker | `ui/src/components/layout/sidebar.tsx` | PARTIAL | Traffic lights, logo, 29 nav items, update badge present; media accordion + live update-check deferred |
| PAR-UI-029 | Header renders breadcrumbs, page title, search bar, auth badge, donate, theme/lang toggles, logout | `ui/src/components/layout/header.tsx` | PARTIAL | Breadcrumbs, title, search bound to store, auth badge present; theme/lang toggles + logout + donate slots null (w6-b/c/j) |
| PAR-UI-030 | Toast notifications via Zustand store with auto-dismiss | `ui/src/stores/notification.ts`, `ui/src/components/layout/toaster.tsx` | HAVE | Zustand `notificationStore` with auto-dismiss; sonner bridge via `AppToaster` |
| PAR-UI-031 | Mobile sidebar with overlay and slide-in animation | `ui/src/components/layout/mobile-sidebar.tsx` | HAVE | Reuses `NAV_ITEMS`, overlay closes panel, hamburger-driven |
| PAR-UI-032 | Button component with variants (primary, secondary, ghost, outline, danger), sizes, icon, loading | `ui/src/components/ui/button.tsx` | HAVE | w6-b: CVA 5 variants, sizes sm/md/lg/icon, `icon`/`loading` (spinner + aria-busy), Radix Slot `asChild` |
| PAR-UI-033 | Input component with label, error, hint | `ui/src/components/ui/input.tsx` | HAVE | w6-b: label/error/hint, generated id + htmlFor, aria-invalid + aria-describedby |
| PAR-UI-034 | Select component with options array | `ui/src/components/ui/select.tsx` | HAVE | w6-b: styled native `<select>`, `options` array, label/error a11y wiring (variant-HAVE) |
| PAR-UI-035 | Card component with padding variants | `ui/src/components/ui/card.tsx` | HAVE | w6-b: `Card`/`CardHeader`/`CardTitle`/`CardContent`, padding none/sm/md/lg |
| PAR-UI-036 | Modal component with traffic lights, sizes, overlay click, Escape key, body scroll lock | `ui/src/components/ui/modal.tsx` | HAVE | w6-b: portal-free, traffic lights, sizes sm/md/lg/xl, overlay click + Escape close, body scroll lock |
| PAR-UI-037 | ConfirmModal wrapper with danger/primary variants | `ui/src/components/ui/confirm-modal.tsx` | HAVE | w6-b: wraps Modal + Button, danger/primary variant mapping |
| PAR-UI-038 | Badge component with success/error/default/neutral/primary variants, dot, size | `ui/src/components/ui/badge.tsx` | HAVE | w6-b: CVA 5 variants, optional dot, sizes sm/md |
| PAR-UI-039 | Toggle component with sm/md sizes | `ui/src/components/ui/toggle.tsx` | HAVE | w6-b: Radix Switch (`@radix-ui/react-switch`), sizes sm/md, role=switch + aria-checked |
| PAR-UI-040 | SegmentedControl component for tab-like selection | `ui/src/components/ui/segmented-control.tsx` | HAVE | w6-b: role=tablist + role=tab + aria-selected, options array |
| PAR-UI-041 | ProviderIcon component with fallback text/color | `ui/src/components/ui/provider-icon.tsx` | HAVE | w6-b: `/providers/<slug>.png` img + onError fallback, `providerInitials`/`providerColor` helpers |
| PAR-UI-042 | Loading/Spinner/Skeleton/CardSkeleton components | `ui/src/components/ui/loading.tsx`, `ui/src/components/ui/skeleton.tsx` | HAVE | w6-b: `Spinner`/`Loading` (role=status), `Skeleton`/`CardSkeleton` (aria-hidden pulse) |
| PAR-UI-043 | Tooltip component with position and color | `ui/src/components/ui/tooltip.tsx` | HAVE | w6-b: Radix Tooltip (`@radix-ui/react-tooltip`), `side` position + `color` variants, exports `TooltipProvider` |
| PAR-UI-044 | Pagination component | `ui/src/components/ui/pagination.tsx` | HAVE | w6-b: prev/next bounds-disabled, nav aria-label + aria-current, exported `paginationRange` helper |
| PAR-UI-045 | LanguageSwitcher shows flag emoji grid, POSTs to `/api/locale` | `ui/src/components/ui/language-switcher.tsx` | HAVE | w6-b: trigger → Modal flag-emoji grid, POST `/api/locale` via `apiFetch`, `DEFAULT_LOCALES` |
| PAR-UI-046 | ThemeToggle cycles light/dark/system | `ui/src/components/ui/theme-toggle.tsx` | HAVE | w6-b: cycles light→dark→system via w6-a `useThemeStore`, Sun/Moon/Monitor icon, aria-label names theme |
| PAR-UI-047 | UsageStats component: period selector, overview cards, provider topology, usage table, SSE updates | `ui/src/components/usage/usage-stats.tsx` | HAVE (variant) | w6-g §1.3 — REST cards/topology(@xyflow)/table + additive SSE overlay; SSE strategy variant |
| PAR-UI-048 | RequestLogger auto-refreshing table (3s poll) | `ui/src/components/usage/request-logger.tsx` | HAVE | w6-g — 3s setInterval poll + refresh toggle; tolerant of real Go string[] + mock UsageLog[] |
| PAR-UI-049 | ModelSelectModal hierarchical model picker with combos + custom models | `src/shared/components/ModelSelectModal.js` | HAVE (w6-f) | `ui/src/components/keys/model-select-modal.tsx`: combos (`/api/combos`) + per-provider models (`/api/models`, disabled hidden via `/api/models/disabled`) + custom (`/api/models/custom`) segmented picker (plan §1.4) |
| PAR-UI-050 | ComboFormModal with DnD model list (create/edit) | `ui/src/components/combos/combo-form-modal.tsx` | HAVE | w6-h §1.3 — in-page ComboFormModal consuming frozen `Modal`/`Input`/`Select`; member list reorderable via @dnd-kit (`DndContext`/`SortableContext`/`useSortable`/`verticalListSortingStrategy` + modifiers), `onDragEnd` delegates to pure `moveStep`; save POST/PUT with member order |
| PAR-UI-051 | OAuthModal generic OAuth login with local proxy | `ui/src/components/providers/oauth-modal.tsx`, `ui/src/lib/oauth-popup.ts` | HAVE | w6-e: OAuth popup via `GET /api/oauth/{provider}/start` + the w6-c `/callback` relay (BroadcastChannel/postMessage/storage), finalized at `POST /api/oauth/{provider}/callback` |
| PAR-UI-052 | EditConnectionModal edit provider connection | `ui/src/components/providers/edit-connection-modal.tsx` | HAVE | w6-e: edits name/active + secret rotation via `PUT /api/connections/{id}` |
| PAR-UI-053 | ManualConfigModal manual API key entry | `ui/src/components/providers/manual-config-modal.tsx` | HAVE | w6-e: manual key/token entry via `POST /api/connections` |
| PAR-UI-054 | McpMarketplaceModal MCP marketplace | `ui/src/components/mcp/mcp-marketplace-modal.tsx`, `ui/src/lib/mcp-install.ts` | HAVE (variant) | w6-l §1.6/§8 ESC-1a — ref's `/api/cli-tools/cowork-mcp-*` registry endpoints absent in g0router; browse/install REMAPPED to the in-tree mcp mock (browse `GET /api/mcp/clients`, install `POST /api/mcp/instances` via pure `toInstancePayload`, unit-tested); mounted from the `/mcp` page; NO Go backend (serial follow-up) |
| PAR-UI-055 | ChangelogModal fetched from GitHub raw CHANGELOG.md | `src/shared/components/ChangelogModal.js` | HAVE | w6-j: `changelog-modal.tsx` (frozen `Modal` + installed `react-markdown`); mounted from settings about-block; source mock-route `/api/version/changelog` (plan §1.7b) |
| PAR-UI-056 | DonateModal donation CTA | `src/shared/components/DonateModal.js` | HAVE | w6-j: `donate-modal.tsx` (frozen `Modal`); mounted from settings about-block; source mock-route `/api/version/donate` (plan §1.7b) |
| PAR-UI-057 | PricingModal pricing config | `ui/src/components/usage/pricing-modal.tsx` | HAVE | w6-g — Modal + input/output/cached/reasoning/cache_creation fields; PATCH save / DELETE reset |
| PAR-UI-058 | CursorAuthModal / KiroAuthModal / KiroSocialOAuthModal IDE auth flows | `ui/src/components/providers/cursor-auth-modal.tsx`, `ui/src/components/providers/kiro-auth-modal.tsx` | HAVE | w6-e: Cursor session-token + Kiro access-token modals creating connections via `POST /api/connections` |
| PAR-UI-059 | IFlowCookieModal iFlow cookie auth | `ui/src/components/providers/iflow-cookie-modal.tsx` | HAVE | w6-e: iFlow session-cookie modal → `POST /api/connections` |
| PAR-UI-060 | GitLabAuthModal GitLab PAT import | `ui/src/components/providers/gitlab-auth-modal.tsx` | HAVE | w6-e: GitLab PAT modal → `POST /api/connections` |
| PAR-UI-061 | NineRemotePromoModal remote access promo | `src/shared/components/NineRemotePromoModal.js` | MISSING | Not in g0router e2e |
| PAR-UI-062 | AddCustomEmbeddingModal custom embedding provider | `ui/src/components/providers/add-custom-embedding-modal.tsx` | HAVE | w6-e: custom model/embedding registration via `POST /api/models/custom` |
| PAR-UI-063 | NoAuthProxyCard proxy pool selector for no-auth mode | `ui/src/components/providers/no-auth-proxy-card.tsx` | HAVE | w6-e: no-auth/local-proxy provider card (port of `NoAuthProxyCard.js`) |
| PAR-UI-064 | ProviderInfoCard provider info display | `ui/src/components/providers/provider-info-card.tsx`, `ui/src/components/providers/no-auth-proxy-card.tsx` | HAVE | w6-e: provider identity/capabilities card (port of `ProviderInfoCard.js`) |
| PAR-UI-065 | Auth: password login POST `/api/auth/login` with bcrypt, rate limit, retry countdown | `ui/src/lib/auth.ts`, `ui/src/routes/login.tsx` | HAVE (variant) | w6-c §1.4: `loginWithPassword` POSTs `/api/auth/login`; 429 reads `error.retry_after` (g0router envelope) → 1s `setInterval` "Wait {n}s" countdown disabling submit; bcrypt is Go-side |
| PAR-UI-066 | Auth: OIDC flow with PKCE, state, nonce, JWKS verification | `ui/src/lib/auth.ts`, `ui/src/routes/callback.tsx` | HAVE (variant) | w6-c §1.4/§1.3: UI entry = `startOidc` navigation to Go `/api/auth/oidc/start`; `/callback` covers the provider-OAuth relay half; PKCE/state/nonce/JWKS are correctly server-owned (Go) |
| PAR-UI-067 | Auth: `GET /api/auth/status` on mount to check requireLogin, authMode, oidcConfigured | `ui/src/lib/auth.ts`, `ui/src/routes/login.tsx` | HAVE (variant) | w6-c §1.4: `getAuthStatus` on mount drives `authMode` visibility; real Go status returns only `auth_mode` (auth.go:177-179), so `oidc_configured`/`require_login` are absent and UI degrades to static OIDC label |
| PAR-UI-068 | Auth: logout POST `/api/auth/logout` clears cookies, redirects to `/login` | `ui/src/components/auth/logout-button.tsx`, `ui/src/lib/auth.ts` | HAVE | w6-c: `LogoutButton` (header LogoutSlot) → `logout()` POSTs `/api/auth/logout` → `useUserStore.clear()` → navigate `/login` |
| PAR-UI-069 | i18n: 33 locales configured in `LOCALES` array | `src/i18n/config.js:1` | HAVE | `ui/src/i18n/locales.ts` mirrors ref exactly (33 codes) |
| PAR-UI-070 | i18n: runtime DOM translation via MutationObserver, stores `_originalText` per node | `src/i18n/runtime.js` | PARTIAL | variant: `react-i18next` hook-based init in `ui/src/i18n/index.ts`; DOM scan not ported |
| PAR-UI-071 | i18n: `RuntimeI18nProvider` re-processes DOM on route change (double RAF) | `src/i18n/RuntimeI18nProvider.js:7-27` | PARTIAL | variant: `I18nProvider` subscribes to `router.subscribe('onResolved', ...)`; mounted by w6-b |
| PAR-UI-072 | i18n: locale cookie name `locale`, POST `/api/locale` to set server-side | `src/shared/components/LanguageSwitcher.js` | HAVE | `POST /api/locale` sets non-HttpOnly `locale` cookie; `I18nProvider.setLocale` uses `apiFetch` |
| PAR-UI-073 | Theming: Tailwind CSS v4 with `@theme inline` and semantic tokens | `ui/src/index.css` | HAVE | `@theme inline` with primary, background, foreground, muted, border, ring tokens + dark overrides |
| PAR-UI-074 | Theming: brand color `#E56A4A`, light `#FDFAF6`, dark `#1a1a1a` | `ui/src/index.css` | HAVE | `--color-primary: #e56a4a; --color-bg-light: #fdfaf6; --color-bg-dark: #1a1a1a` |
| PAR-UI-075 | Theming: Zustand themeStore with `persist` middleware, key `"theme"` | `ui/src/stores/theme.ts` | HAVE | `create(persist(..., { name: 'theme' }))` with light/dark/system |
| PAR-UI-076 | Theming: `useTheme` hook syncs with `prefers-color-scheme` via `useSyncExternalStore` | `ui/src/hooks/use-theme.ts` | HAVE | `useSyncExternalStore` over `matchMedia('(prefers-color-scheme: dark)')` |
| PAR-UI-077 | Theming: `ThemeProvider` calls `initTheme()` on mount | `ui/src/providers/theme.tsx` | HAVE | `useEffect` calls `initTheme()` on mount; re-applies on theme/system change |
| PAR-UI-078 | Theming: `.dark` class toggled on `<html>` | `ui/src/stores/theme.ts` | HAVE | `document.documentElement.classList.toggle('dark', ...)` in `applyTheme` |
| PAR-UI-079 | Icons: Material Symbols Outlined font with `fill-1` class | `ui/src/index.css` | HAVE | `@import 'material-symbols/outlined.css'` + `.fill-1 { font-variation-settings: 'FILL' 1; }` |
| PAR-UI-080 | State: Zustand stores (themeStore, userStore, providerStore, settingsStore, notificationStore, headerSearchStore) | `ui/src/stores/*.ts` | HAVE | All six stores implemented; theme/settings use `persist` |
| PAR-UI-081 | Data fetching: raw `fetch()` with local state, no React Query/SWR | `ui/src/lib/api.ts` | HAVE (variant) | `apiFetch` unwraps Go `{data,error}` envelope; serves as TanStack Query `queryFn` adapter |
| PAR-UI-082 | Real-time: SSE `EventSource` for usage stats at `/api/usage/stream` | `ui/src/components/usage/usage-stats.tsx` | HAVE (variant) | w6-g §1.3 — additive `EventSource("/api/usage/stream")`; merges active/recent/pending/error_provider; onerror no-op; proven by unit test (e2e stays REST-deterministic; MockEventSource idles for the usage url) |
| PAR-UI-083 | Real-time: SSE `EventSource` for console logs at `/api/translator/console-logs/stream` | `ui/src/components/console/console-log-viewer.tsx`, `ui/src/components/chat/chat-window.tsx` | HAVE (variant) | w6-i §1.4/§1.5 — console SSE consumes `EventSource("/api/console-logs/stream")` via pure `subscribeConsoleLogs` (the fixture MockEventSource path renamed from the 9router `/api/translator/console-logs/stream`); the chat send/receive turn streams the REAL `/v1/chat/completions` via `streamChatCompletion`; both mock/fixture-intercepted; runtime Go SSE endpoint deferred (ESCALATION-1) |
| PAR-UI-084 | Drag & Drop: `@dnd-kit/core` + `@dnd-kit/sortable` in combo builder | `src/app/(dashboard)/dashboard/combos/page.js:4-7` | HAVE | g0router `package.json` has `@dnd-kit/core`, `@dnd-kit/sortable` |
| PAR-UI-085 | React Flow for provider topology visualization | `src/app/(dashboard)/dashboard/usage/components/ProviderTopology.js` | HAVE | g0router `package.json` has `@xyflow/react` |
| PAR-UI-086 | Monaco Editor for translator debug page | `ui/src/components/translator/translator-step.tsx` | HAVE (variant) | w6-i §1.6/§8 ESCALATION-2 — textarea variant: editor surface is a plain monospaced `<textarea>` per step (NO Monaco/CodeMirror — neither installed, NO dep added); optional upgrade to a real editor is a serial follow-up if a Monaco/CodeMirror dep is later sanctioned |
| PAR-UI-087 | API endpoint: `GET /api/providers` lists connections | `internal/admin/providers_catalog.go` (`ListProviderCatalog`) | HAVE | w6-e (variant §1.6/§8 ESCALATION-1): provider-shaped read overlay `GET /api/providers/catalog` (display_name/auth_types/capabilities/connection_count/status); existing `GET /api/providers` CRUD left untouched |
| PAR-UI-088 | API endpoint: `POST /api/providers` creates connection | `internal/admin/providers_catalog.go` (`GetProviderCatalog`/`GetProviderConnections`) | HAVE | w6-e (variant §1.6/§8 ESCALATION-2): `GET /api/providers/{id}/catalog` + `/{id}/connections` (UI-shaped, secrets masked); connection creation stays on existing `POST /api/connections` |
| PAR-UI-089 | API endpoint: `PUT /api/providers/${id}` toggles active | `internal/admin/providers_catalog.go` (`GetProviderModels`/`GetProviderSuggestedModels`) | HAVE | w6-e (variant §1.6): `GET /api/providers/{id}/models` + `/{id}/suggested-models` from static catalog metadata; active-toggle stays on existing `PUT /api/connections/{id}` |
| PAR-UI-090 | API endpoint: `POST /api/providers/test-batch` batch test | `internal/admin/providers_catalog.go` (`TestProvidersBatch`) | HAVE | w6-e: `POST /api/providers/test-batch` returns `{results:[{provider,ok,latency_ms}]}` (ok = provider has a connection) |
| PAR-UI-091 | API endpoint: `GET /api/combos` list combos | `ui/src/routes/combos.tsx` | HAVE (variant) | w6-h §1.2/§8 ESCALATION-1 — page consumes `GET /api/combos` MOCK; real Go exists (`routes_admin.go:85`) but serves divergent DTO `{name,models:[]string}` keyed by `name` (no id/strategy/is_active/structured steps); DTO reconcile is a serial Go follow-up |
| PAR-UI-092 | API endpoint: `POST /api/combos` create combo | `ui/src/components/combos/combo-form-modal.tsx` | HAVE (variant) | w6-h §1.2/§8 ESCALATION-1 — page POSTs to `/api/combos` MOCK; real Go `routes_admin.go:86` body is `{name,models:[]string}`; DTO reconcile serial Go follow-up |
| PAR-UI-093 | API endpoint: `PUT /api/combos/${id}` update combo | `ui/src/components/combos/combo-form-modal.tsx` | HAVE (variant) | w6-h §1.2/§8 ESCALATION-1 — page PUTs `/api/combos/{id}` MOCK; real Go keys by `{name}` (`routes_admin.go:87`); key/DTO reconcile serial Go follow-up |
| PAR-UI-094 | API endpoint: `DELETE /api/combos/${id}` delete combo | `ui/src/routes/combos.tsx` | HAVE (variant) | w6-h §1.2/§8 ESCALATION-1 — page DELETEs `/api/combos/{id}` MOCK; real Go keys by `{name}` (`routes_admin.go:88`); key reconcile serial Go follow-up |
| PAR-UI-095 | API endpoint: `GET /api/usage/stats?period=` usage statistics | `internal/admin/usage.go:101` | HAVE (variant) | w6-g §1.4 — page + corrected mock now call the real Go `GET /api/usage/stats?period=` (was mock-only `/api/usage/summary`) |
| PAR-UI-096 | API endpoint: `GET /api/usage/request-logs` request logs | `internal/admin/usage.go:139` | HAVE (variant) | w6-g §1.4 — page + corrected mock call the real Go `GET /api/usage/request-logs` (was mock-only `/api/logs`); component tolerant of real string[] + structured shapes |
| PAR-UI-097 | API endpoint: `GET /api/settings` get settings | `src/app/(dashboard)/dashboard/profile/page.js:66` | HAVE | Real Go `internal/admin/settings.go` `GetSettings`; consumed by `general-settings-panel.tsx`/`oidc-config-panel.tsx` (plan §1.2) |
| PAR-UI-098 | API endpoint: `PATCH /api/settings` patch settings | `src/app/(dashboard)/dashboard/profile/page.js:105` | HAVE | Real Go `PutSettings` (`PUT /api/settings`, flat `map[string]string`); General/OIDC panels persist via it (plan §1.2) |
| PAR-UI-099 | API endpoint: `POST /api/settings/proxy-test` test outbound proxy | `src/app/(dashboard)/dashboard/profile/page.js:141` | HAVE | w6-j variant: OIDC config panel persists `oidc_*` keys via real `PUT /api/settings`; tests via real `POST /api/auth/oidc/test` (plan §1.4/§8 ESC-4) |
| PAR-UI-100 | API endpoint: `GET /api/settings/database` export DB | `src/app/(dashboard)/dashboard/profile/page.js:478` | HAVE | w6-j variant: `password-panel.tsx` (PAR-UI-100 = password change) ships mock-only via existing `PUT /api/auth/password` handler; no real Go yet (plan §1.4/§8 ESC-2) |
| PAR-UI-101 | API endpoint: `POST /api/settings/database` import DB | `src/app/(dashboard)/dashboard/profile/page.js:516` | HAVE | w6-j variant: `db-info-panel.tsx` ships mock-only via `GET /api/settings/database`; no real Go yet (plan §1.4/§8 ESC-3) |
| PAR-UI-102 | API endpoint: `GET /api/version` check for updates | `src/shared/components/Sidebar.js:64` | HAVE | Go (NEW) `internal/admin/version.go` `GetVersion`; injected version/build_date via `SetVersionInfo` (plan §1.5) |
| PAR-UI-103 | API endpoint: `POST /api/version/shutdown` shutdown server | `src/shared/components/Sidebar.js:94` | HAVE | Go (NEW) `internal/admin/version.go` `Shutdown`; injectable nil-safe `SetShutdownFunc`, response-first async, never exits in handler (plan §1.5b) |
| PAR-UI-104 | API endpoint: `GET /api/proxy-pools?includeUsage=true` list pools | `ui/src/routes/proxy-pools.tsx`, `ui/e2e/mocks/handlers/proxy-pools.ts` | PARTIAL | w6-m §1.4/§8 ESC-1b — UI consumes the registered `GET /api/proxy-pools` MOCK (mock ignores `?includeUsage`); NO Go endpoint (W7 follow-up → HAVE in W7) |
| PAR-UI-105 | API endpoint: `POST /api/proxy-pools` create pool | `ui/src/components/platform/proxy-pool-form-modal.tsx`, `ui/e2e/mocks/handlers/proxy-pools.ts` | PARTIAL | w6-m §1.4/§8 ESC-1b — UI consumes the registered `POST /api/proxy-pools` MOCK; NO Go endpoint (W7 follow-up → HAVE in W7) |
| PAR-UI-106 | API endpoint: `POST /api/proxy-pools/vercel-deploy` deploy relay | `src/app/(dashboard)/dashboard/proxy-pools/page.js:379` | MISSING | Not in g0router e2e |
| PAR-UI-107 | API endpoint: `POST /api/proxy-pools/cloudflare-deploy` deploy relay | `src/app/(dashboard)/dashboard/proxy-pools/page.js:404` | MISSING | Not in g0router e2e |
| PAR-UI-108 | API endpoint: `POST /api/proxy-pools/deno-deploy` deploy relay | `src/app/(dashboard)/dashboard/proxy-pools/page.js:429` | MISSING | Not in g0router e2e |
| PAR-UI-109 | API endpoint: `GET /api/provider-nodes` custom compatible nodes | `src/app/(dashboard)/dashboard/providers/page.js:148` | HAVE (Go — w6-f) | `internal/admin/nodes.go` `ListProviderNodes` — providers filtered to `type=="openai-compatible"` (plan §1.6b); route `routes_admin.go`; mock `ui/e2e/mocks/handlers/nodes.ts` |
| PAR-UI-110 | API endpoint: `POST /api/provider-nodes` create node | `src/app/(dashboard)/dashboard/providers/page.js:876` | HAVE (Go — w6-f) | `internal/admin/nodes.go` `CreateProviderNode` — creates a `providers` row `type=openai-compatible` (camelCase/snake_case body); no schema change (plan §1.6b) |
| PAR-UI-111 | API endpoint: `POST /api/provider-nodes/validate` validate endpoint | `src/app/(dashboard)/dashboard/providers/page.js:909` | HAVE (Go — w6-f) | `internal/admin/nodes.go` `ValidateProviderNode` — deterministic URL well-formedness; `api_key` NEVER persisted; route precedence proven by `TestNodesRouteDisambiguation` (plan §1.6b/§8 ESC-4) |
| PAR-UI-112 | API endpoint: `GET /api/tunnel/status` tunnel status | `ui/src/routes/tunnels.tsx`, `ui/e2e/mocks/handlers/tunnels.ts` | PARTIAL | w6-m §1.5/§8 ESC-1c — UI consumes the registered `GET /api/tunnels` (+ `/api/tunnels/health`) MOCK via plain REST poll (path REMAPPED from ref `/api/tunnel/status`; NO SSE/EventSource); NO Go endpoint (W7 follow-up → HAVE in W7) |
| PAR-UI-113 | API endpoint: `POST /api/tunnel/enable` / `disable` Cloudflare tunnel | `ui/src/routes/tunnels.tsx`, `ui/e2e/mocks/handlers/tunnels.ts` | PARTIAL | w6-m §1.5/§8 ESC-1c — UI consumes `POST /api/tunnels/cloudflare` enable / `DELETE /api/tunnels/cloudflare` disable MOCK (path REMAPPED); NO Go endpoint (W7 follow-up → HAVE in W7) |
| PAR-UI-114 | API endpoint: `POST /api/tunnel/tailscale-enable` / `disable` Tailscale | `ui/src/routes/tunnels.tsx`, `ui/e2e/mocks/handlers/tunnels.ts` | PARTIAL | w6-m §1.5/§8 ESC-1c — UI consumes `POST /api/tunnels/tailscale` enable / `DELETE /api/tunnels/tailscale` disable MOCK (path REMAPPED) + the g0router-grouped `/tunnels` page; NO Go endpoint (W7 follow-up → HAVE in W7) |
| PAR-UI-115 | API endpoint: `POST /api/keys` create API key | `src/app/(dashboard)/dashboard/endpoint/EndpointPageClient.js` | HAVE (real Go) | REAL `internal/admin/apikeys.go` `CreateAPIKey` (body `{name}` → `{key,name,id,machine_id}`); consumed by `<ApiKeysPanel>` (plan §1.2; mock body corrected to Go DTO §8 ESC-2) |
| PAR-UI-116 | API endpoint: `GET /api/models/alias` model aliases | `ui/src/routes/aliases.tsx`, `ui/src/components/routing/alias-modal.tsx` | HAVE (variant) | w6-h §1.2/§8 ESCALATION-2 — `/aliases` page (list + AliasModal CRUD) consumes `/api/aliases` MOCK; `store.ListAliases()` exists (`internal/store/aliases.go:64`) but there is NO admin `/api/aliases` endpoint; admin Go endpoint is a serial follow-up |
| PAR-UI-117 | API endpoint: `GET /api/models/custom` custom models | `src/shared/components/ModelSelectModal.js` | HAVE (variant — mock-only, w6-f) | consumed by ModelSelectModal via w6-e-owned `ui/e2e/mocks/handlers/models.ts`; NO Go — serial follow-up (plan §1.4/§8 ESC-3, open-questions) |
| PAR-UI-118 | API endpoint: `GET /api/models/disabled` disabled models | `src/shared/components/ModelSelectModal.js` | HAVE (real Go) | REAL `internal/admin/disabledmodels.go` `GetDisabledModels`; consumed by ModelSelectModal (plan §1.2) |
| PAR-UI-119 | API endpoint: `POST /api/models/test` test model inference | `src/shared/components/ModelSelectModal.js` | HAVE (variant — mock-only, w6-f) | NEW mock body `ui/e2e/mocks/handlers/nodes.ts` → `{ok,latency_ms}`; NO Go — serial follow-up (plan §1.4/§8 ESC-3, open-questions) |
| PAR-UI-120 | API endpoint: `GET /api/models/availability` model availability | `src/app/(dashboard)/dashboard/providers/components/ModelAvailabilityBadge.js` | HAVE (variant — mock-only, w6-f) | NEW mock body `ui/e2e/mocks/handlers/nodes.ts` → `{available:[...]}`; NO Go — serial follow-up (plan §1.4/§8 ESC-3, open-questions) |
| PAR-UI-121 | Next.js App Router with `output: "standalone"` | `next.config.mjs` | MISSING | g0router uses Vite + TanStack Router |
| PAR-UI-122 | Vite build with React 19, TypeScript, path aliases | `ui/vite.config.ts`, `ui/tsconfig.json` | HAVE | g0router stack is different but modern |
| PAR-UI-123 | TanStack Router file-based routing | `ui/src/routes/__root.tsx`, `ui/routeTree.gen.ts` | HAVE | Auto-generated, only `__root__` registered |
| PAR-UI-124 | TanStack Query for server state | `ui/package.json:49` | HAVE | Listed in dependencies, unused in src |
| PAR-UI-125 | shadcn/ui components via `components.json` | `ui/components.json` | HAVE | Configured but no components generated |
| PAR-UI-126 | Recharts for charts | `ui/package.json:74` | HAVE | Listed in dependencies, unused |
| PAR-UI-127 | React Hook Form + Zod validation | `ui/package.json:70`, `ui/package.json:81` | HAVE | Listed, unused |
| PAR-UI-128 | AI SDK React for chat | `ui/package.json:15`, `ui/package.json:53` | HAVE | Listed, unused |
| PAR-UI-129 | Playwright e2e tests defining full app specification | `ui/e2e/*.spec.ts` (41 files) | EXTRA | g0router e2e suite is larger than 9router's `tests/` |
| PAR-UI-130 | g0router-specific routes not in 9router: `/connections`, `/virtual-keys`, `/routing-rules`, `/teams`, `/audit`, `/feature-flags`, `/guardrails`, `/prompts`, `/model-limits`, `/alerts`, `/mcp`, `/mcp/tools`, `/endpoint` | `ui/src/routes/connections.tsx` (`/connections`); `ui/src/routes/routing-rules.tsx`, `ui/src/routes/model-limits.tsx` (w6-h); `ui/e2e/connections.spec.ts`, etc. | EXTRA (`/connections`, `/routing-rules`, `/model-limits`, `/virtual-keys`, `/endpoint` subset HAVE) | g0router backend has these entities; w6-e ships `/connections`; w6-h ships `/routing-rules` + `/model-limits` (variant — list + modal CRUD against the `/api/routing-rules` and `/api/model-limits` MOCKS; NO Go backend for either — serial follow-ups §8 ESCALATION-3a/3b, open-questions); w6-f ships `/virtual-keys` (list + form modal with KeyIDs editor against REAL w5-g VK `provider_configs[].key_ids` CRUD) + `/endpoint` (base-url + ApiKeysPanel); **w6-k ships `/teams`, `/audit`, `/feature-flags`, `/guardrails` (+ prompt tester), `/prompts`, `/alerts` HAVE (variant — list/config + modal CRUD/toggle against the registered `/api/{teams,audit,feature-flags,guardrails,prompt-templates,alert-channels}` MOCKS; NO Go backend for ANY of the six — the MAP "phases 13-19 complete" claim is VERIFIED FALSE, w6-k §1.2; serial Go follow-ups §8 ESCALATION-1a..1f, open-questions)**; **w6-l ships `/mcp` + `/mcp/tools` HAVE (variant — REWRITE existing stubs; `/mcp` lists clients/instances + marketplace install (`mcp-marketplace-modal.tsx`), `/mcp/tools` lists tools/execute + tool-groups CRUD, all against the registered `/api/mcp/{clients,instances,tools,tool-groups}` MOCKS; reads PascalCase client/instance + snake_case tool-group casing; NO Go backend — `internal/mcp/` is a Phase-1 placeholder, the MAP "MCP gateway backend in-tree" claim is VERIFIED FALSE, w6-l §1.2; serial Go follow-ups §8 ESC-1a/1b, open-questions)**; **w7-gov-1 UPDATE: `/teams` + `/audit` flip variant-HAVE→HAVE (real Go backend — `internal/admin/{teams,audit}.go`; mocks corrected to mirror, w7-gov-1 §1.4/§1.5); the w6-k Users panel on `/teams` is now real-Go-backed (PAR-UI-132). The other four governance routes (`/feature-flags`,`/guardrails`,`/prompts`,`/alerts`) remain mock-variant pending w7-gov-2/3.** **w7-gov-2 UPDATE: `/feature-flags` + `/prompts` flip variant-HAVE→HAVE (real Go backend — `internal/admin/{featureflags,prompttemplates}.go` over `internal/store/{featureflags,prompttemplates}.go`; INTEGER-PK numeric ids per ESC-IDTYPE; feature-flags is GET-list+PUT-toggle only, prompt-templates is full CRUD + `/test`; mocks corrected to mirror, w7-gov-2 §1.4/§1.5). The remaining two governance routes (`/guardrails`,`/alerts`) stay mock-variant pending w7-gov-3.** **w7-gov-3 UPDATE: `/guardrails` + `/alerts` flip variant-HAVE→true-HAVE (real Go backend — `internal/admin/{guardrails,alerts}.go` over `internal/store/{guardrails,alertchannels}.go` + the `internal/governance/{guardrails,alertchannels}.go` evaluator/dispatcher seams; guardrails is a SINGLETON config (GET/PUT) + standalone `/test` evaluator, alert-channels is full CRUD + per-channel `/test`; alert config encrypted at rest via `config_enc`, never echoed in the test-notification; mocks verified to already mirror the Go DTOs — no body change, w7-gov-3 §1.4/§1.5). With this the ENTIRE w6-k governance cluster (`/teams`,`/audit`,`/feature-flags`,`/guardrails`,`/prompts`,`/alerts`) is real-Go-backed.** |
| PAR-UI-131 | g0router-specific API: `GET /api/connections`, `GET /api/virtual-keys`, `GET /api/routing-rules`, `GET /api/teams`, `GET /api/audit`, `GET /api/feature-flags`, `GET /api/guardrails`, `GET /api/prompt-templates`, `GET /api/model-limits`, `GET /api/alert-channels`, `GET /api/mcp/*` | `ui/e2e/mocks/handlers/*.ts` | EXTRA (governance GET subset HAVE — variant) | Backend APIs exist; e2e mocks define contracts. **w6-k governance subset (`GET /api/{teams,audit,feature-flags,guardrails,prompt-templates,alert-channels}`): HAVE (variant — served by the registered e2e MOCKS; NO Go endpoints exist, w6-k §1.2; serial Go follow-ups §8 ESCALATION-1, open-questions).** **w7-gov-1 UPDATE: `GET /api/teams` (full CRUD) + `GET /api/audit` are now REAL Go (`internal/admin/{teams,audit}.go` over `internal/store/{teams,auditlog}.go`); their mocks were corrected to mirror the real DTOs → mock→true-HAVE for teams+audit. The remaining four (feature-flags/guardrails/prompt-templates/alert-channels) stay mock-variant pending w7-gov-2/3.** **w7-gov-2 UPDATE: `GET/PUT /api/feature-flags[/{id}]` (list + toggle, no POST/DELETE) + `GET/POST/PUT/DELETE /api/prompt-templates[/{id}]` + `POST /api/prompt-templates/test` are now REAL Go (`internal/admin/{featureflags,prompttemplates}.go` over `internal/store/{featureflags,prompttemplates}.go`, INTEGER-PK numeric ids); mocks corrected to mirror (prompts POST/PUT drop `updated_at`) → mock→true-HAVE for feature-flags+prompt-templates. The remaining two (guardrails/alert-channels) stay mock-variant pending w7-gov-3.** **w7-gov-3 UPDATE: `GET/PUT /api/guardrails` (singleton config) + `POST /api/guardrails/test` (standalone blocklist/PII evaluator) + `GET/POST /api/alert-channels` + `GET/PUT/DELETE /api/alert-channels/{id}` + `POST /api/alert-channels/{id}/test` are now REAL Go (`internal/admin/{guardrails,alerts}.go` over `internal/store/{guardrails,alertchannels}.go`, INTEGER-PK numeric alert ids; alert `config` encrypted at rest via `config_enc`, never echoed in the test-notification response); mocks verified to already mirror the Go DTOs → mock→true-HAVE for guardrails+alert-channels. The full w6-k governance GET subset is now real-Go-backed.** |
| PAR-UI-132 | g0router-specific auth: `POST /api/auth/setup` first-user creation, `PUT /api/auth/password`, `GET/POST /api/auth/users` | `internal/admin/usermgmt.go`; `internal/store/users.go`; `ui/e2e/mocks/handlers/auth.ts`; `ui/src/components/governance/users-panel.tsx` (w6-k) | HAVE (real Go — w7-gov-1) | g0router has user management not in 9router. **w7-gov-1 ships the REAL Go backend (`POST /api/auth/setup` public first-user onboarding self-guarding on `CountUsers()==0`; `PUT /api/auth/password` verifying the current password; `GET/POST /api/auth/users`; `DELETE /api/auth/users/{id}` with last-user-delete guard) in `internal/admin/usermgmt.go` over `internal/store/users.go` (+additive `display_name`/`role` columns + `CreateUserFull`), reusing `auth.HashPassword`/`VerifyPassword`; password/hash never echoed (runtime no-leak tests). The w6-k Users panel now consumes real Go; the `auth.ts` users-route mocks were corrected to mirror the Go `{data}`+`userDTO{id,username,display_name,role}`. Flipped mock→true-HAVE by w7-gov-1 §1.6 (resolves open-questions w6-k ESCALATION-2).** |
| PAR-UI-133 | g0router-specific chat via `/v1/chat/completions` streaming | `ui/e2e/chat.spec.ts` | EXTRA | 9router has hidden `/dashboard/basic-chat` |

---

## Data Models

### 9Router Settings (fetched from `GET /api/settings`)

Fields observed in `src/app/(dashboard)/dashboard/profile/page.js`:
- `requireLogin` (boolean)
- `hasPassword` (boolean)
- `authMode` ("password" | "oidc" | "both")
- `oidcIssuerUrl` (string)
- `oidcClientId` (string)
- `oidcScopes` (string, default "openid profile email")
- `oidcLoginLabel` (string)
- `oidcConfigured` (boolean)
- `fallbackStrategy` ("fill-first" | "round-robin")
- `stickyRoundRobinLimit` (number, default 3)
- `comboStrategy` ("fallback" | "round-robin")
- `comboStickyRoundRobinLimit` (number, default 1)
- `outboundProxyEnabled` (boolean)
- `outboundProxyUrl` (string)
- `outboundNoProxy` (string)
- `enableObservability` (boolean)
- `enableTranslator` (boolean)
- `comboStrategies` (object map)

### 9Router Connection (from `GET /api/providers`)

Fields observed in `src/app/(dashboard)/dashboard/providers/page.js`:
- `id` (string)
- `provider` (string)
- `authType` ("oauth" | "apikey" | "free" | "compatible")
- `isActive` (boolean)
- `testStatus` ("active" | "success" | "error" | "expired" | "unavailable")
- `lastError` (string)
- `lastErrorType` (string: runtime_error, upstream_auth_error, etc.)
- `errorCode` (string/number)
- `lastErrorAt` (ISO timestamp)
- `modelLock_*` fields (timestamps for cooldown)

### 9Router Combo (from `GET /api/combos`)

Fields observed in `src/app/(dashboard)/dashboard/combos/page.js`:
- `id` (string)
- `name` (string, regex `/^[a-zA-Z0-9_.\-]+$/`)
- `models` (array of model ID strings)
- `kind` (optional string, null for LLM combos)

### 9Router Proxy Pool (from `GET /api/proxy-pools`)

Fields observed in `src/app/(dashboard)/dashboard/proxy-pools/page.js`:
- `id` (string)
- `name` (string)
- `proxyUrl` (string)
- `noProxy` (string)
- `isActive` (boolean)
- `strictProxy` (boolean)
- `testStatus` (string)
- `lastError` (string)
- `lastTestedAt` (ISO timestamp)
- `boundConnectionCount` (number)
- `type` ("vercel" | "cloudflare" | undefined)

### g0router API Contracts (from e2e mocks)

The g0router e2e mock handlers (`ui/e2e/mocks/handlers/*.ts`) define these backend contracts:
- All responses use `{data, error}` envelope
- Auth: `POST /api/auth/login` returns `{token: string}`; `GET /api/auth/status` returns `{require_login, has_users, authenticated, username, display_name, role}`
- Settings: `GET /api/settings` returns a flat object with nested sections (general, logging, features, network, notifications, security)
- CRUD resources (keys, virtual-keys, teams, connections, combos, aliases, pricing, routing-rules, proxy-pools, model-limits, alert-channels, prompt-templates) share a pattern: `GET /api/<resource>` lists, `POST /api/<resource>` creates, `GET/PUT/DELETE /api/<resource>/:id` manages individual items
- Usage: `GET /api/usage/summary` (aggregated), `GET /api/usage/chart` (7-day buckets), `GET /api/usage` (paginated logs), `GET /api/logs` (request logs)
- Streams: `GET /api/traffic/stream` (SSE), `GET /api/console-logs/stream` (SSE)
- MCP: `GET /api/mcp/clients`, `GET /api/mcp/tools`, `POST /api/mcp/tools/:name/execute`

---

## Edge Cases and Quirks

- **Login rate limit**: `src/app/login/page.js:20-24` implements a `retryAfter` countdown with `setInterval`. Server returns `retryAfter` field.
- **Default password fallback**: `src/app/login/page.js:181-182` displays default password `123456` when no custom password is set.
- **Auth status safe fallback**: `src/app/login/page.js:50-56` sets `hasPassword = true` on non-OK response to avoid infinite loading.
- **Provider toggle optimistic update**: `src/app/(dashboard)/dashboard/providers/page.js:208-228` updates local state before firing `PUT` requests, then uses `Promise.allSettled`.
- **Combo name validation**: `src/app/(dashboard)/dashboard/combos/page.js:13` regex `/^[a-zA-Z0-9_.\-]+$/` enforced client-side.
- **Combo edit inline**: `src/app/(dashboard)/dashboard/combos/page.js:290-385` models in combo modal are inline-editable with Enter/Escape handlers.
- **DnD index-based IDs**: `src/app/(dashboard)/dashboard/combos/page.js:403` uses `uid: "item-${i}"` to handle duplicate model names.
- **Usage stats SSE merge strategy**: `src/shared/components/UsageStats.js:255-278` SSE only overwrites `activeRequests`, `recentRequests`, `errorProvider`, `pending` — never full stats from REST.
- **Proxy pool batch health check**: `src/app/(dashboard)/dashboard/proxy-pools/page.js:267-328` runs with `CONCURRENCY = 10` workers, then offers to disable dead proxies via confirm modal.
- **Proxy pool delete 409 handling**: `src/app/(dashboard)/dashboard/proxy-pools/page.js:157-162` shows warning with `boundConnectionCount` when pool has bound connections.
- **Sidebar update checker**: `src/shared/components/Sidebar.js:63-68` fetches `/api/version` on mount, shows install command with countdown shutdown.
- **i18n DOM skip**: `src/app/(dashboard)/dashboard/profile/page.js:653` uses `data-i18n-skip="true"` on language button to prevent translation.
- **Callback origin allowlist**: `src/app/callback/page.js:34-37` restricts `postMessage` to `window.location.origin` and `"http://localhost:1455"` (Codex port).
- **Modal body scroll lock**: `src/shared/components/Modal.js:27-34` sets `document.body.style.overflow = "hidden"` when open.
- **Modal Escape close**: `src/shared/components/Modal.js:36-42` attaches global keydown listener.
- **Header search store**: `src/shared/components/Header.js:327-358` global search query shared across pages via Zustand.

---

## Go-Port Considerations

1. g0router backend already implements the API contracts defined in e2e mocks; the UI only needs consumers.
2. Use TanStack Query (`@tanstack/react-query`) for all data fetching instead of raw `fetch()` with local state.
3. Use TanStack Router file-based routes under `ui/src/routes/`; the plugin is already configured in `vite.config.ts`.
4. Use shadcn/ui primitives (already configured in `components.json`) instead of custom Button/Input/Modal components.
5. The g0router e2e suite (`ui/e2e/`) is the functional specification; build pages to make these tests pass.
6. g0router has 30+ routes not in 9router; do not limit the port to 9router's subset — implement all e2e-specified routes.
7. Theme implementation should use the already-configured Tailwind v4 dark mode; Zustand persist store pattern from 9router ports directly.
