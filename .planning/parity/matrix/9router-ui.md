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
| PAR-UI-005 | Route `/dashboard` alias to `/dashboard/endpoint` | `src/app/(dashboard)/dashboard/page.js` | MISSING | g0router e2e expects `/dashboard` |
| PAR-UI-006 | Route `/dashboard/endpoint` shows API endpoint config + API key management | `src/app/(dashboard)/dashboard/endpoint/page.js:1-7` | MISSING | g0router e2e has `/endpoint` route |
| PAR-UI-007 | Route `/dashboard/providers` lists providers in card grid (OAuth, Free, API Key, Compatible) | `ui/src/routes/providers.tsx`, `ui/src/components/providers/provider-card.tsx` | HAVE | w6-e (variant §1.5): flat `/providers` route, grouped sections (OAuth/API-Key/Free/Compatible) of `card-elev` cards from `GET /api/providers/catalog` |
| PAR-UI-008 | Route `/dashboard/providers/new` adds new provider | `ui/src/routes/providers.tsx`, `ui/src/components/providers/manual-config-modal.tsx` | HAVE | w6-e (variant §1.5): in-page "add connection" flow (modal), not a nested `/new` route |
| PAR-UI-009 | Route `/dashboard/providers/[id]` shows provider detail with connections + models | `ui/src/components/providers/provider-detail-panel.tsx` | HAVE | w6-e (variant §1.5): in-page detail panel loading `/api/providers/{id}/connections` + `/models`, not a nested `/[id]` route |
| PAR-UI-010 | Route `/dashboard/combos` lists combos with DnD reordering | `src/app/(dashboard)/dashboard/combos/page.js:15-214` | MISSING | g0router e2e expects `/combos` |
| PAR-UI-011 | Route `/dashboard/usage` has overview/logs/details tabs with period selector | `src/app/(dashboard)/dashboard/usage/page.js:16-74` | MISSING | g0router splits this into `/usage`, `/logs`, `/traffic` |
| PAR-UI-012 | Route `/dashboard/quota` shows provider limits | `src/app/(dashboard)/dashboard/quota/page.js:5-10` | MISSING | g0router e2e expects `/quota` |
| PAR-UI-013 | Route `/dashboard/mitm` MITM proxy config | `src/app/(dashboard)/dashboard/mitm/page.js:1-5` | MISSING | g0router e2e expects `/mitm` |
| PAR-UI-014 | Route `/dashboard/cli-tools` CLI tool integrations | `src/app/(dashboard)/dashboard/cli-tools/page.js:1-7` | MISSING | Not in g0router e2e |
| PAR-UI-015 | Route `/dashboard/cli-tools/[toolId]` per-tool detail | `src/app/(dashboard)/dashboard/cli-tools/[toolId]/page.js` | MISSING | Not in g0router e2e |
| PAR-UI-016 | Route `/dashboard/basic-chat` hidden chat UI | `src/app/(dashboard)/dashboard/basic-chat/page.js` | MISSING | g0router e2e has `/chat` instead |
| PAR-UI-017 | Route `/dashboard/console-log` live server console | `src/app/(dashboard)/dashboard/console-log/page.js` | MISSING | g0router e2e expects `/console` |
| PAR-UI-018 | Route `/dashboard/translator` debug translation flow | `src/app/(dashboard)/dashboard/translator/page.js` | MISSING | Not in g0router e2e |
| PAR-UI-019 | Route `/dashboard/proxy-pools` proxy pool management with bulk ops | `src/app/(dashboard)/dashboard/proxy-pools/page.js:30-1063` | MISSING | g0router e2e expects `/proxy-pools` |
| PAR-UI-020 | Route `/dashboard/skills` agent skills links | `src/app/(dashboard)/dashboard/skills/page.js` | MISSING | g0router e2e expects `/skills` |
| PAR-UI-021 | Route `/dashboard/profile` settings (theme, lang, OIDC, password, DB) | `src/app/(dashboard)/dashboard/profile/page.js:23-1140` | MISSING | g0router e2e expects `/settings` |
| PAR-UI-022 | Route `/dashboard/media-providers/[kind]` media provider kind list | `src/app/(dashboard)/dashboard/media-providers/[kind]/page.js` | MISSING | Not in g0router e2e |
| PAR-UI-023 | Route `/dashboard/media-providers/[kind]/[id]` media provider detail | `src/app/(dashboard)/dashboard/media-providers/[kind]/[id]/page.js` | MISSING | Not in g0router e2e |
| PAR-UI-024 | Route `/dashboard/media-providers/web` web search/fetch combos | `src/app/(dashboard)/dashboard/media-providers/web/page.js` | MISSING | Not in g0router e2e |
| PAR-UI-025 | Route `/dashboard/settings/pricing` pricing management | `src/app/dashboard/settings/pricing/page.js` | MISSING | g0router e2e expects `/pricing` |
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
| PAR-UI-047 | UsageStats component: period selector, overview cards, provider topology, usage table, SSE updates | `src/shared/components/UsageStats.js:192-505` | MISSING | g0router e2e expects dashboard metrics |
| PAR-UI-048 | RequestLogger auto-refreshing table (3s poll) | `src/shared/components/RequestLogger.js` | MISSING | g0router e2e expects `/logs` table |
| PAR-UI-049 | ModelSelectModal hierarchical model picker with combos + custom models | `src/shared/components/ModelSelectModal.js` | MISSING | g0router e2e expects model selection dialogs |
| PAR-UI-050 | ComboFormModal with DnD model list (create/edit) | `src/shared/components/ComboFormModal.js` | MISSING | g0router `package.json` has `@dnd-kit/core` |
| PAR-UI-051 | OAuthModal generic OAuth login with local proxy | `ui/src/components/providers/oauth-modal.tsx`, `ui/src/lib/oauth-popup.ts` | HAVE | w6-e: OAuth popup via `GET /api/oauth/{provider}/start` + the w6-c `/callback` relay (BroadcastChannel/postMessage/storage), finalized at `POST /api/oauth/{provider}/callback` |
| PAR-UI-052 | EditConnectionModal edit provider connection | `ui/src/components/providers/edit-connection-modal.tsx` | HAVE | w6-e: edits name/active + secret rotation via `PUT /api/connections/{id}` |
| PAR-UI-053 | ManualConfigModal manual API key entry | `ui/src/components/providers/manual-config-modal.tsx` | HAVE | w6-e: manual key/token entry via `POST /api/connections` |
| PAR-UI-054 | McpMarketplaceModal MCP marketplace | `src/shared/components/McpMarketplaceModal.js` | MISSING | g0router e2e expects `/mcp` routes |
| PAR-UI-055 | ChangelogModal fetched from GitHub raw CHANGELOG.md | `src/shared/components/ChangelogModal.js` | MISSING | Not in g0router e2e |
| PAR-UI-056 | DonateModal donation CTA | `src/shared/components/DonateModal.js` | MISSING | Not in g0router e2e |
| PAR-UI-057 | PricingModal pricing config | `src/shared/components/PricingModal.js` | MISSING | g0router e2e expects `/pricing` CRUD |
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
| PAR-UI-082 | Real-time: SSE `EventSource` for usage stats at `/api/usage/stream` | `src/shared/components/UsageStats.js:255-278` | MISSING | g0router e2e mocks SSE with `MockEventSource` |
| PAR-UI-083 | Real-time: SSE `EventSource` for console logs at `/api/translator/console-logs/stream` | `src/app/(dashboard)/dashboard/console-log/page.js` | MISSING | g0router e2e mocks SSE for `/api/console-logs/stream` |
| PAR-UI-084 | Drag & Drop: `@dnd-kit/core` + `@dnd-kit/sortable` in combo builder | `src/app/(dashboard)/dashboard/combos/page.js:4-7` | HAVE | g0router `package.json` has `@dnd-kit/core`, `@dnd-kit/sortable` |
| PAR-UI-085 | React Flow for provider topology visualization | `src/app/(dashboard)/dashboard/usage/components/ProviderTopology.js` | HAVE | g0router `package.json` has `@xyflow/react` |
| PAR-UI-086 | Monaco Editor for translator debug page | `src/app/(dashboard)/dashboard/translator/page.js` | MISSING | Not in g0router `package.json` |
| PAR-UI-087 | API endpoint: `GET /api/providers` lists connections | `internal/admin/providers_catalog.go` (`ListProviderCatalog`) | HAVE | w6-e (variant §1.6/§8 ESCALATION-1): provider-shaped read overlay `GET /api/providers/catalog` (display_name/auth_types/capabilities/connection_count/status); existing `GET /api/providers` CRUD left untouched |
| PAR-UI-088 | API endpoint: `POST /api/providers` creates connection | `internal/admin/providers_catalog.go` (`GetProviderCatalog`/`GetProviderConnections`) | HAVE | w6-e (variant §1.6/§8 ESCALATION-2): `GET /api/providers/{id}/catalog` + `/{id}/connections` (UI-shaped, secrets masked); connection creation stays on existing `POST /api/connections` |
| PAR-UI-089 | API endpoint: `PUT /api/providers/${id}` toggles active | `internal/admin/providers_catalog.go` (`GetProviderModels`/`GetProviderSuggestedModels`) | HAVE | w6-e (variant §1.6): `GET /api/providers/{id}/models` + `/{id}/suggested-models` from static catalog metadata; active-toggle stays on existing `PUT /api/connections/{id}` |
| PAR-UI-090 | API endpoint: `POST /api/providers/test-batch` batch test | `internal/admin/providers_catalog.go` (`TestProvidersBatch`) | HAVE | w6-e: `POST /api/providers/test-batch` returns `{results:[{provider,ok,latency_ms}]}` (ok = provider has a connection) |
| PAR-UI-091 | API endpoint: `GET /api/combos` list combos | `src/app/(dashboard)/dashboard/combos/page.js:32` | MISSING | g0router e2e mocks `GET /api/combos` |
| PAR-UI-092 | API endpoint: `POST /api/combos` create combo | `src/app/(dashboard)/dashboard/combos/page.js:55-70` | MISSING | g0router e2e mocks `POST /api/combos` |
| PAR-UI-093 | API endpoint: `PUT /api/combos/${id}` update combo | `src/app/(dashboard)/dashboard/combos/page.js:72-89` | MISSING | g0router e2e mocks `PUT /api/combos/:id` |
| PAR-UI-094 | API endpoint: `DELETE /api/combos/${id}` delete combo | `src/app/(dashboard)/dashboard/combos/page.js:91-107` | MISSING | g0router e2e mocks `DELETE /api/combos/:id` |
| PAR-UI-095 | API endpoint: `GET /api/usage/stats?period=` usage statistics | `src/shared/components/UsageStats.js:242` | MISSING | g0router uses `GET /api/usage/summary` and `GET /api/usage/chart` |
| PAR-UI-096 | API endpoint: `GET /api/usage/request-logs` request logs | `src/shared/components/RequestLogger.js` | MISSING | g0router uses `GET /api/logs` |
| PAR-UI-097 | API endpoint: `GET /api/settings` get settings | `src/app/(dashboard)/dashboard/profile/page.js:66` | MISSING | g0router e2e mocks `GET /api/settings` |
| PAR-UI-098 | API endpoint: `PATCH /api/settings` patch settings | `src/app/(dashboard)/dashboard/profile/page.js:105` | MISSING | g0router e2e mocks `PUT /api/settings` |
| PAR-UI-099 | API endpoint: `POST /api/settings/proxy-test` test outbound proxy | `src/app/(dashboard)/dashboard/profile/page.js:141` | MISSING | Not in g0router e2e |
| PAR-UI-100 | API endpoint: `GET /api/settings/database` export DB | `src/app/(dashboard)/dashboard/profile/page.js:478` | MISSING | Not in g0router e2e |
| PAR-UI-101 | API endpoint: `POST /api/settings/database` import DB | `src/app/(dashboard)/dashboard/profile/page.js:516` | MISSING | Not in g0router e2e |
| PAR-UI-102 | API endpoint: `GET /api/version` check for updates | `src/shared/components/Sidebar.js:64` | MISSING | g0router e2e mocks `GET /api/version` |
| PAR-UI-103 | API endpoint: `POST /api/version/shutdown` shutdown server | `src/shared/components/Sidebar.js:94` | MISSING | Not in g0router e2e |
| PAR-UI-104 | API endpoint: `GET /api/proxy-pools?includeUsage=true` list pools | `src/app/(dashboard)/dashboard/proxy-pools/page.js:71` | MISSING | g0router e2e mocks `GET /api/proxy-pools` |
| PAR-UI-105 | API endpoint: `POST /api/proxy-pools` create pool | `src/app/(dashboard)/dashboard/proxy-pools/page.js:122` | MISSING | g0router e2e mocks `POST /api/proxy-pools` |
| PAR-UI-106 | API endpoint: `POST /api/proxy-pools/vercel-deploy` deploy relay | `src/app/(dashboard)/dashboard/proxy-pools/page.js:379` | MISSING | Not in g0router e2e |
| PAR-UI-107 | API endpoint: `POST /api/proxy-pools/cloudflare-deploy` deploy relay | `src/app/(dashboard)/dashboard/proxy-pools/page.js:404` | MISSING | Not in g0router e2e |
| PAR-UI-108 | API endpoint: `POST /api/proxy-pools/deno-deploy` deploy relay | `src/app/(dashboard)/dashboard/proxy-pools/page.js:429` | MISSING | Not in g0router e2e |
| PAR-UI-109 | API endpoint: `GET /api/provider-nodes` custom compatible nodes | `src/app/(dashboard)/dashboard/providers/page.js:148` | MISSING | Not in g0router e2e |
| PAR-UI-110 | API endpoint: `POST /api/provider-nodes` create node | `src/app/(dashboard)/dashboard/providers/page.js:876` | MISSING | Not in g0router e2e |
| PAR-UI-111 | API endpoint: `POST /api/provider-nodes/validate` validate endpoint | `src/app/(dashboard)/dashboard/providers/page.js:909` | MISSING | Not in g0router e2e |
| PAR-UI-112 | API endpoint: `GET /api/tunnel/status` tunnel status | `src/app/(dashboard)/dashboard/endpoint/EndpointPageClient.js` | MISSING | g0router e2e mocks `GET /api/tunnel/status` |
| PAR-UI-113 | API endpoint: `POST /api/tunnel/enable` / `disable` Cloudflare tunnel | `src/app/(dashboard)/dashboard/endpoint/EndpointPageClient.js` | MISSING | g0router e2e has `/tunnels` page |
| PAR-UI-114 | API endpoint: `POST /api/tunnel/tailscale-enable` / `disable` Tailscale | `src/app/(dashboard)/dashboard/endpoint/EndpointPageClient.js` | MISSING | g0router e2e has `/tunnels` page |
| PAR-UI-115 | API endpoint: `POST /api/keys` create API key | `src/app/(dashboard)/dashboard/endpoint/EndpointPageClient.js` | MISSING | g0router e2e mocks `POST /api/keys` |
| PAR-UI-116 | API endpoint: `GET /api/models/alias` model aliases | `src/app/(dashboard)/dashboard/combos/page.js:418` | MISSING | g0router e2e has `/aliases` page |
| PAR-UI-117 | API endpoint: `GET /api/models/custom` custom models | `src/shared/components/ModelSelectModal.js` | MISSING | g0router e2e mocks `GET /api/models/custom` |
| PAR-UI-118 | API endpoint: `GET /api/models/disabled` disabled models | `src/shared/components/ModelSelectModal.js` | MISSING | g0router e2e mocks `GET /api/models/disabled` |
| PAR-UI-119 | API endpoint: `POST /api/models/test` test model inference | `src/shared/components/ModelSelectModal.js` | MISSING | Not in g0router e2e |
| PAR-UI-120 | API endpoint: `GET /api/models/availability` model availability | `src/app/(dashboard)/dashboard/providers/components/ModelAvailabilityBadge.js` | MISSING | Not in g0router e2e |
| PAR-UI-121 | Next.js App Router with `output: "standalone"` | `next.config.mjs` | MISSING | g0router uses Vite + TanStack Router |
| PAR-UI-122 | Vite build with React 19, TypeScript, path aliases | `ui/vite.config.ts`, `ui/tsconfig.json` | HAVE | g0router stack is different but modern |
| PAR-UI-123 | TanStack Router file-based routing | `ui/src/routes/__root.tsx`, `ui/routeTree.gen.ts` | HAVE | Auto-generated, only `__root__` registered |
| PAR-UI-124 | TanStack Query for server state | `ui/package.json:49` | HAVE | Listed in dependencies, unused in src |
| PAR-UI-125 | shadcn/ui components via `components.json` | `ui/components.json` | HAVE | Configured but no components generated |
| PAR-UI-126 | Recharts for charts | `ui/package.json:74` | HAVE | Listed in dependencies, unused |
| PAR-UI-127 | React Hook Form + Zod validation | `ui/package.json:70`, `ui/package.json:81` | HAVE | Listed, unused |
| PAR-UI-128 | AI SDK React for chat | `ui/package.json:15`, `ui/package.json:53` | HAVE | Listed, unused |
| PAR-UI-129 | Playwright e2e tests defining full app specification | `ui/e2e/*.spec.ts` (41 files) | EXTRA | g0router e2e suite is larger than 9router's `tests/` |
| PAR-UI-130 | g0router-specific routes not in 9router: `/connections`, `/virtual-keys`, `/routing-rules`, `/teams`, `/audit`, `/feature-flags`, `/guardrails`, `/prompts`, `/model-limits`, `/alerts`, `/mcp`, `/mcp/tools`, `/endpoint` | `ui/src/routes/connections.tsx` (`/connections`); `ui/e2e/connections.spec.ts`, `ui/e2e/virtual-keys.spec.ts`, etc. | EXTRA (`/connections` subset HAVE) | g0router backend has these entities; w6-e ships the `/connections` page (rows render, toggle/edit/test/delete/bulk via existing connection CRUD); remaining routes tracked by later plans |
| PAR-UI-131 | g0router-specific API: `GET /api/connections`, `GET /api/virtual-keys`, `GET /api/routing-rules`, `GET /api/teams`, `GET /api/audit`, `GET /api/feature-flags`, `GET /api/guardrails`, `GET /api/prompt-templates`, `GET /api/model-limits`, `GET /api/alert-channels`, `GET /api/mcp/*` | `ui/e2e/mocks/handlers/*.ts` | EXTRA | Backend APIs exist; e2e mocks define contracts |
| PAR-UI-132 | g0router-specific auth: `POST /api/auth/setup` first-user creation, `PUT /api/auth/password`, `GET/POST /api/auth/users` | `ui/e2e/mocks/handlers/auth.ts` | EXTRA | g0router has user management not in 9router |
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
