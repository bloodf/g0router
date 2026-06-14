# Wave 7 — Backend (Go) parity to 100%: strategic micro-plan MAP

Author: Planner (g0router parity lead). Orchestrator: per `CLI_ORCHESTRATOR.md`
(W6+ rule; HANDOFF.md no longer governs). Frozen ref @ `827e5c3` (same 9router
checkout as W2–W6). Depends on Waves 0–6 — **Wave 6 COMPLETE**.

**Non-authorizing INDEX** (like WAVE-2/3/4/5/6-MAP). This MAP sequences the work
and assigns row ownership so per-plan micro-plans (`w7-a.md`, …) can be authored
afterward. Binary acceptance criteria, exact task steps, precondition greps, and
stop conditions live in the individual micro-plan files, written before each
plan's dispatch per `CLI_ORCHESTRATOR.md`.

## Goal

Drive the **9router** parity matrices to 100% by building the **Go backend** that
Wave 6 deferred (every "Serial Go follow-up" logged in
`.planning/parity/plans/open-questions.md`) or that was never built. Wave 6 shipped
the dashboard UI against e2e **mocks**; Wave 7 builds the real Go so those rows
flip from variant-HAVE/PARTIAL to **true HAVE** and the mocks become reality-mirrors
or are replaced by integration tests against real handlers.

**Scope boundary (user-confirmed):** finish **9router** fully BEFORE the
`bifrost-*` program. **Bifrost is OUT of scope for Wave 7** — the
`matrix/bifrost-*.md` and `g0router-audit.md` matrices are not addressed here.

This is the first predominantly-**Go** wave since W5. The TDD unit is the Go
`_test.go` file (test first → see it fail → minimum handler/store to pass; no
mocks, fakes via interfaces). `go test ./... && go vet ./... && go build ./...`
is the per-commit gate; `cd ui && npm run build` + scoped Playwright remain green
gates only for the handful of plans that touch UI wiring or correct a mock body.

---

## Architectural decisions (cross-cutting)

1. **Real Go wins; mocks become mirrors (the binding reconciliation rule).** Every
   Wave-6 surface that shipped variant-HAVE/PARTIAL against an e2e mock
   (`ui/e2e/mocks/handlers/*`) now gets a real Go handler. The binding rule
   (extends W6 decision 4): when the real Go DTO and the W6 mock body disagree,
   **the Go API wins** and the mock body + seed are corrected in the SAME plan to
   mirror the real handler's `{data,error}` snake_case shape. Where a route is a
   pure backend (no UI to keep green), the plan ALSO adds a Go-side handler
   integration test; the Playwright spec stays mock-backed (mock now mirrors Go)
   OR is switched to hit the real handler under `vite preview` if the harness
   permits (plan decides at authoring time; default = corrected mock + Go
   integration test, lowest risk). No plan invents a new UI contract — it makes
   the existing W6 contract real.

2. **Additive migrations only.** All new tables/columns land via additive
   `ensureColumn`/`ensureTable` in `internal/store/migrate.go` (WAL store,
   additive-only — `AGENTS.md` decision). No destructive DDL, no column renames.
   New domain tables (mcp_*, proxy_pools, tunnels, teams, audit_log,
   feature_flags, guardrails, prompt_templates, alert_channels, aliases admin,
   routing_rules, model_limits, api_keys-already-exist) follow the existing
   `*_enc` secret-at-rest precedent (`internal/store/oauthsessions.go`) for any
   secret fields (OIDC client secret, MCP OAuth tokens, sudo password, tunnel
   tokens, proxy credentials).

3. **`internal/server/routes_admin.go` is the serial hot file.** Exactly ONE
   in-flight plan may hold an unmerged edit to it at a time (W3/W4/W5/W6 lesson).
   Wave 7 REOPENS the serial chain (W6's chain closed on w6-j). The W7 serial
   order is defined in **Impl order** below. Every plan that adds `/api/*` admin
   routes takes the slot in turn; route additions are additive appends only.
   Static-segment routes must be registered before `{id}`/`{param}` routes
   (fasthttp/router precedence — see existing `/api/providers/catalog` vs
   `/api/providers/{id}` at `routes_admin.go:54-61`).

4. **Layered DDD is enforced** (arch test, phase 12B): transport
   (`internal/server`, `internal/admin`) → domain (`internal/governance`,
   `internal/mcp`, `internal/platform`, `internal/inference`) → repository
   (`internal/store`). New backends create a domain package + a store layer +
   an `internal/admin/<domain>.go` transport handler. No `init()`, errors-as-
   values (`fmt.Errorf("ctx: %w", err)`), no new global state — dependencies via
   struct fields / constructor params. Read 3 existing files before writing a new
   one (e.g. `internal/admin/virtualkeys.go` + `internal/store/virtualkeys.go` +
   `internal/governance/quota.go` are the canonical CRUD-domain templates).

5. **Provider parity ≈ catalog config, NOT N new adapters (the big de-risk).**
   `internal/providers/generic/` is a complete OpenAI-format adapter and
   `internal/inference/factory.go:104-109` already routes *any* catalog provider
   through `generic.New()`. So the ~40 `format:"openai"` provider MISSING rows are
   **catalog + model-catalog entries** (`internal/providers/catalog/catalog.go`,
   `catalog/models.go`, `catalog/aliases.go`), NOT new Go packages. Only the
   **specialized-format** providers need genuinely new converter/adapter code:
   `kiro` (AWS eventstream — catalog entry exists but no eventstream decoder),
   `cursor` (connect+proto protobuf), `antigravity` (multi-backend routing),
   `commandcode` (custom JSON), `vertex` (service-account + dynamic URL),
   `perplexity-web`/`grok-web` (cookie auth, reverse-engineered),
   `azure` (resource-specific URL), `cloudflare-ai` (`{accountId}` template).
   This split drives the w7-prov-* sub-plan partition.

6. **MCP gateway scope is explicit, not hand-waved (the hardest chunk).**
   `internal/mcp/` is a Phase-1 placeholder (`doc.go` + a no-op
   `TestPackageCompiles`); `schemas/mcp.go` defines `MCPClient/MCPInstance/
   MCPTool/MCPToolGroup` but nothing consumes them. For Wave-7 **parity**, the
   MCP gateway means concretely (mapped to PAR-MCP rows): (a) **store layer** —
   mcp_instances, mcp_clients, mcp_oauth_accounts, mcp_oauth_flows tables; (b)
   **launcher** — stdio child-process spawn with command allowlist + HTTP/SSE
   client modes (`os/exec` StdinPipe/StdoutPipe; `sync.RWMutex`-guarded bridge
   map); (c) **stdio↔SSE bridge** — JSON-RPC frame broadcast to SSE sessions,
   smart text filter, exit/stderr handling; (d) **probe** — initialize +
   notifications/initialized + tools/list handshake, session-id replay, 8s
   timeout, requiresAuth on 401/403; (e) **registry client** — Anthropic
   mcp-registry pagination + 1h cache + direct-connect filter; (f) **OAuth
   account engine** — PKCE, protected-resource-metadata, token storage/refresh;
   (g) **admin transport** — `/api/mcp/clients|instances|tools|tool-groups`
   handlers backing the w6-l mock; (h) **discovery/health/agent loop** — tools
   cache + compact injection, periodic ping, multi-turn tool execution. This is
   too large for one plan; it splits into **w7-mcp-1 (store+launcher+bridge),
   w7-mcp-2 (probe+registry+OAuth), w7-mcp-3 (admin transport + tools/tool-groups
   + discovery/health)** with serial-slot discipline on the transport plan only.
   Out of Wave-7 MCP scope (deferred, recorded as escalations, not parity
   blockers for the dashboard): Cursor MCP protobuf encoders (PAR-MCP-026..029 —
   ride with the w7-prov cursor adapter if/when built), Cowork Claude-Desktop 3p
   config writer + toolPolicy generator (PAR-MCP-018..023, 045..050 — 9router
   feature is globally disabled in ref, low parity value), antigravity hardcoded
   unavailable tool (PAR-MCP-060 — rides w7-prov antigravity).

7. **Platform backends flip w6-m PARTIAL→HAVE.** mitm / proxy-pools / tunnels Go
   engines land here (`internal/platform/` is a placeholder doc.go today). Scope
   is pragmatic Go ports per `9router-platform.md` Go-port considerations: proxy-
   pool CRUD + connectivity test (defer vercel/cloudflare/deno *deploy* relays —
   PAR-PLAT-006/007/008 + PAR-UI-106/107/108 — to a tracked escalation, niche
   value, heavy external-API surface); tunnels = cloudflared + tailscale process
   management; MITM = CA-cert serving + per-tool config (full hosts-file patching
   / system-trust-store install is OS-privileged — scope CA generation + serving
   + config, defer auto-install to escalation).

8. **Wave-6 UI is FROZEN / consume-only.** Wave 7 is backend-focused. UI changes
   are limited to: (a) correcting a mock handler/seed body to mirror real Go
   (decision 1); (b) the rare wiring needed when a row genuinely requires a UI
   touch (e.g. switching a page from a mock path to the real path if the path
   changed). No new pages, no component redesign, no route files. The frozen
   sets from w6-a/w6-b stay consume-only.

9. **No new global state; reuse the `Handlers` injection pattern.**
   `internal/admin/handlers.go` already exposes additive setters
   (`SetUsageServices`, `SetVersionInfo`, `SetShutdownFunc`). New domains follow
   the same shape: a domain service constructed in `NewAdminHandlers`
   (`routes_admin.go:19-28`) and injected, keeping `admin.New(...)` signature
   churn additive.

---

## Row coverage ledger (every MISSING/PARTIAL row → exactly one plan)

| Matrix | MISSING | PARTIAL | W7 disposition |
|---|---|---|---|
| `9router-mcp.md` | 57 (PAR-MCP-001..029, 032..041, 043..060) | 0 | w7-mcp-1/2/3 (PAR-MCP-030/031 already HAVE; 042 EXTRA) |
| `9router-platform.md` | 46 (PAR-PLAT-001..044, 049, 050) | 2 (047, 048) | w7-plat-1 (proxy-pools), w7-plat-2 (tunnels), w7-plat-3 (mitm), w7-prov-oauth (047 OAuth providers), w7-routes-admin (048 route registration rolls up across plans) |
| `9router-providers.md` | 53 (012,013,015–026,028,030–051,053–067) | 0 | w7-prov-openai (config providers), w7-prov-special (specialized adapters), w7-prov-oauth (OAuth providers), w7-prov-media (image/stt/tts/embed — scope call) |
| `9router-routing.md` | 9 (009,027,039,040,053,055,056,059,060) | 1 (035) | w7-route (provider-nodes routing 009/040 land with w7-platnodes; 027/035/039/053/055/056/059/060 grouped) |
| `9router-ui.md` | 11 | 11 | backend-blocked: 013/019/104/105/112/113/114 → w7-plat-*; 106/107/108 → deploy ESCALATION; genuine-UI/SKIP: 004/014/015/022/023/024/061/121/028/029/070/071 → NOT W7 (see UI disposition) |
| `9router-auth.md` | 1 (003) | 1 (020) | 003 JWT → variant decision (NOT built; recorded); 020 SSRF/outbound-proxy → w7-plat-1 (proxy subsystem) |
| `9router-translation.md` | 1 (050b) | 1 (006) | both schema/Stage-2 blocked → ESCALATION (multi-part content schema); NOT W7 unless schema work scheduled |
| `9router-usage.md` | 0 | 1 (032) | w7-usage-quota (remaining 6 provider quota fetchers) |

**Plus** the Wave-6-deferred Go backends from `open-questions.md` (each "Serial Go
follow-up"): governance ×6 (teams/audit/feature-flags/guardrails/prompts/alerts),
user-management auth (`/api/auth/setup,password,users`), aliases admin, routing-
rules, model-limits, combos DTO reconciliation, `/api/quota` aggregation, console-
logs SSE, chat-sessions (optional), translator backend, models test/availability/
custom. Each is mapped below.

---

## Micro-plan index

| Plan | Scope | PAR rows it closes | Key existing files / integration points (file:line) | New Go files | routes_admin.go? | e2e impact | Depends |
|---|---|---|---|---|---|---|---|
| **w7-gov-1** | **Governance backends A**: teams (+ user-management auth: `/api/auth/setup,password,users[/{id}]`), audit-log. Store + domain + admin CRUD; password hashing reuses `internal/auth/password.go`. | open-questions w6-k ESC-1a/1b/2 (teams, audit, user-mgmt PAR-UI-132); flips those rows mock→true-HAVE | `internal/admin/auth.go:1` (login/logout/me only — extend), `internal/admin/virtualkeys.go` (CRUD template), `internal/store/virtualkeys.go`, `internal/store/users.go`, `internal/governance/quota.go` (domain template) | `internal/store/teams.go`, `internal/store/auditlog.go`, `internal/governance/teams.go`, `internal/admin/teams.go`, `internal/admin/audit.go`, `internal/admin/usermgmt.go` (+ `_test.go` each) | YES (serial slot) | corrects `mocks/handlers/{teams,audit,auth}.ts` + seeds to mirror Go; w6-k user-panel consumes real `auth.ts` | — |
| **w7-gov-2** | **Governance backends B**: feature-flags (GET+PUT toggle), prompt-templates CRUD. | open-questions w6-k ESC-1c/1e (feature-flags, prompts) | same governance templates as w7-gov-1; `internal/store/kv.go` (flag-store option) | `internal/store/featureflags.go`, `internal/store/prompttemplates.go`, `internal/governance/featureflags.go`, `internal/admin/featureflags.go`, `internal/admin/prompttemplates.go` (+tests) | YES (serial slot) | corrects `mocks/handlers/{feature-flags,prompts}.ts` to mirror Go | — (∥ w7-gov-1; disjoint files) |
| **w7-gov-3** | **Governance backends C**: guardrails (config + blocklist/PII test endpoint over request pipeline), alert-channels (CRUD + real test-notification). | open-questions w6-k ESC-1d/1f (guardrails, alerts) | `internal/inference/` (pipeline hook for guardrail test), governance templates | `internal/store/guardrails.go`, `internal/store/alertchannels.go`, `internal/governance/guardrails.go`, `internal/governance/alertchannels.go`, `internal/admin/guardrails.go`, `internal/admin/alerts.go` (+tests) | YES (serial slot) | corrects `mocks/{handlers,seed}/guardrails.ts` (the `blocked:true` tester contract) + `alert-channels.ts` | — (∥ gov-1/2) |
| **w7-route** | **Routing admin backends + dynamic routing**: aliases admin CRUD over existing `store.ListAliases()`; routing-rules store+CRUD; model-limits store+CRUD; combos DTO reconciliation (real Go `{name,models[]}` ↔ UI `Combo`); `/api/quota` aggregation; weighted provider selection (PAR-ROUTE-027); free no-auth virtual connection (PAR-ROUTE-039); proxy-pool resolution per connection (PAR-ROUTE-055); live model catalog override (PAR-ROUTE-056); web search/fetch pseudo-models (PAR-ROUTE-059); upstream-connection detection (PAR-ROUTE-060); project-ID cold-miss resolution (PAR-ROUTE-053). | PAR-ROUTE-027, 039, 053, 055, 056, 059, 060, 035(→HAVE multi-URL fallback); open-questions w6-h ESC-1/2/3a/3b (combos DTO, aliases, routing-rules, model-limits), w6-g ESC-1c (`/api/quota`) | `internal/store/aliases.go:64` (ListAliases), `internal/admin/combos.go:15-21,65-91` (DTO divergence), `internal/inference/selection.go` (weighted/free-conn/proxy hooks), `internal/inference/factory.go:104` (live catalog), `internal/admin/connectionusage.go` (quota source) | `internal/store/routingrules.go`, `internal/store/modellimits.go`, `internal/admin/aliases.go`, `internal/admin/routingrules.go`, `internal/admin/modellimits.go`, `internal/admin/quota.go` (+tests); selection.go edits (additive) | YES (serial slot — largest holder) | corrects `mocks/handlers/{aliases,routing-rules,model-limits,combos,quota}.ts` to mirror Go | depends on **w7-platnodes** (provider-nodes — PAR-ROUTE-009/040 prefix routing) |
| **w7-platnodes** | **Provider-node system** (the routing prerequisite): dynamic provider nodes (openai-/anthropic-/custom-embedding) with prefix routing that overrides static alias resolution; node CRUD + baseUrl sanitization + cascade-to-connections + `/models`→`/chat/completions` validation. Note: w6-f shipped a *thin* `internal/admin/nodes.go` over the providers table (no prefix/cascade) — this plan builds the real node prefix-routing engine. | PAR-ROUTE-009, PAR-ROUTE-040, PAR-PLAT-010, PAR-PLAT-011, PAR-PLAT-012, PAR-PLAT-013, PAR-PLAT-014 | `internal/admin/nodes.go` (w6-f thin version — extend), `internal/store/providers.go`, `internal/inference/alias.go` (prefix override), `internal/inference/factory.go:37` (providerForModel) | `internal/store/providernodes.go`, `internal/platform/providernodes.go` (domain), node prefix resolver in `internal/inference/` (+tests) | YES (serial slot — FIRST, unblocks w7-route) | extends `mocks/handlers/nodes.ts` to mirror real node DTO | — (FIRST serial holder) |
| **w7-prov-openai** | **Config-only providers** (generic adapter, no new packages): all `format:"openai"` API-key providers — add catalog + model-catalog + alias entries. Families: Chinese (glm, glm-cn, kimi, kimi-coding-apikey, alicode/-intl, volcengine-ark, byteplus, siliconflow, xiaomi-mimo, minimax/-cn), Western (nvidia, cerebras, nebius, hyperbolic, gitlab, codebuddy, vercel-ai-gateway, chutes, blackbox), free-tier bundle (PAR-PROV-067: 29 providers). | PAR-PROV-013, 014, 029, 034, 035, 036, 037, 038, 039, 041, 042, 043, 044, 045, 046, 048, 049, 050, 051, 052, 056, 057, 067 | `internal/providers/catalog/catalog.go:28` (Providers map), `catalog/models.go:19` (model map), `catalog/aliases.go`, `internal/inference/factory.go:104-109` (generic routing — NO change needed) | catalog/models/aliases entries only (NO new package); `_test.go` count-asserts + golden per family | NO | none (catalog-only; no UI contract) | — (∥ everything; catalog is disjoint) |
| **w7-prov-special** | **Specialized-format adapters** (genuinely new converter code): kiro (AWS eventstream decoder — catalog entry exists, no stream decoder), cursor (connect+proto protobuf), antigravity (multi-backend routing + PAR-MCP-060), commandcode (custom JSON), azure (resource URL), cloudflare-ai (`{accountId}` template), vertex (service-account + dynamic URL build). | PAR-PROV-012(vertex), 020(antigravity), 022(kiro), 023(cursor), 028(qoder), 030(perplexity-web), 031(grok-web), 032(azure), 033(cloudflare-ai), 040(commandcode), 047(xiaomi-tokenplan region) | `internal/providers/generic/chat.go` (format-dispatch pattern), `internal/translation/` (converters), `internal/providers/kiro|cursor|vertex` (dirs exist as doc.go stubs for some) | `internal/providers/{kiro,cursor,antigravity,commandcode,azure,vertex}/*.go` converters + adapters (+tests); reverse-engineered web (perplexity-web/grok-web) cookie-auth = ESCALATION (fragile, defer) | NO (factory.go switch edit — but factory is NOT routes_admin.go; coordinate as a second micro-serial if concurrent) | none (catalog/adapter; no UI contract) | may depend on **w7-prov-oauth** for OAuth-gated specialized providers |
| **w7-prov-oauth** | **OAuth provider flows**: device-code + PKCE + refresh for claude(cc), codex(cx), gemini-cli(gc), qwen(qw), iflow(if), github(gh), kilocode(kc), cline(cl). Reuses `internal/auth/oauth.go` PKCE engine (anthropic today) + `internal/store/oauthsessions.go` `*_enc` pattern. | PAR-PROV-015, 016, 017, 018, 019, 021, 024, 025, 026, 027(xai OAuth path); PAR-PLAT-047 (OAuth flows beyond anthropic); PAR-AUTH-019 (extend to ~15 providers) | `internal/auth/oauth.go:33-160` (anthropic PKCE — generalize), `internal/admin/oauth.go:34-87`, `routes_admin.go:85-86` (`/api/oauth/{provider}`) | `internal/auth/oauth_<provider>.go` flow configs, refresh logic; `internal/store` token columns (additive) (+tests) | YES (serial slot — `/api/oauth/{provider}` registration is dynamic, but new flows register in `NewAdminHandlers` flows map `routes_admin.go:21-23`) | none (OAuth is API-flow; provider-OAuth modals already shipped in w6-e against mocks — correct mock if path diverges) | — (∥ catalog plans) |
| **w7-prov-media** | **Media/embedding specialist providers** (scope call — see escalations): image (nanobanana, fal-ai, stability-ai, black-forest-labs, recraft, sdwebui, comfyui), video (runwayml), stt (deepgram, assemblyai), embedding (voyage-ai), multimodal (huggingface). These map to non-chat `Provider` interface methods (ImageGeneration/Speech/Transcription/Embedding). | PAR-PROV-053, 054, 055, 058, 059, 060, 061, 062, 063, 064, 065, 066 | `internal/schemas/provider.go:68-107` (Image/Speech/Transcription/Embedding methods), `internal/providers/openai/embedding.go` (embedding template) | catalog entries + `internal/providers/<media>/` adapters for non-OpenAI-shaped APIs (+tests); MAY be deferred wholesale to a media escalation if Stage-1 chat-only ranking holds | NO | none (no UI page — media-providers UI was W6-deferred PAR-UI-022..024) | — |
| **w7-mcp-1** | **MCP foundation**: store layer (mcp_instances, mcp_clients, mcp_oauth_accounts, mcp_oauth_flows tables) + launcher (stdio spawn w/ command allowlist + HTTP/SSE client modes) + stdio↔SSE bridge (JSON-RPC broadcast, smart text filter, exit/stderr handling, isRunning). | PAR-MCP-003, 004, 005, 006, 007, 008, 032, 033(store), 035, 036, 043, 044, 051, 052, 053, 054 | `internal/mcp/doc.go` (placeholder), `schemas/mcp.go:4` (MCPClient/Instance/Tool/ToolGroup), `internal/store/migrate.go:11` (additive tables), `internal/store/oauthsessions.go` (`*_enc` pattern for OAuth tokens) | `internal/store/mcpinstances.go`, `internal/store/mcpoauth.go`, `internal/mcp/launcher.go`, `internal/mcp/process.go`, `internal/mcp/bridge.go`, `internal/mcp/filter.go` (+tests) | NO | none (foundation; no route yet) | — (∥ governance/providers; disjoint `internal/mcp`) |
| **w7-mcp-2** | **MCP client + OAuth engine**: SSE/message HTTP endpoints (the bridge transport), probe (initialize+notifications/initialized+tools/list handshake, session-id replay, 8s timeout, requiresAuth), registry client (Anthropic mcp-registry pagination + 1h cache + direct-connect filter + URL dedupe), OAuth account engine (PKCE, protected-resource-metadata, token storage/refresh, health). | PAR-MCP-001, 002, 009, 010, 011, 012, 013, 014, 015, 016, 017, 037, 038, 039, 055, 056, 057, 058, 059 | w7-mcp-1 store+bridge, `internal/auth/oauth.go` (PKCE engine reuse), `internal/mcp/bridge.go` | `internal/mcp/sse.go`, `internal/mcp/probe.go`, `internal/mcp/registry.go`, `internal/mcp/oauth.go`, `internal/mcp/healthmonitor.go`, `internal/mcp/discovery.go` (+tests) | NO (SSE/message routes are `/api/mcp/{plugin}/sse|message` — register in mcp transport plan w7-mcp-3 OR here; assign to w7-mcp-3 for single serial holder) | none | **w7-mcp-1** |
| **w7-mcp-3** | **MCP admin transport + tools**: `/api/mcp/clients`, `/api/mcp/instances` (CRUD + OAuth `…/auth/start`), `/api/mcp/tools` (+ per-tool execute), `/api/mcp/tool-groups` (CRUD + is_active), `/api/skills` (catalog/store), SSE/message bridge routes; agent loop (multi-turn tool execution). Backs the w6-l mocks. | PAR-MCP-018(subset), 019(subset), 022, 040, 041(UI exists), 045, 046, 047, 048, 049, 050, 060(antigravity unavailable tool ride-along); open-questions w6-l ESC-1a/1b/1c (mcp clients/instances, tools/tool-groups, skills) | `routes_admin.go` (new `/api/mcp/*` + `/api/skills`), w7-mcp-1/2 domain, `internal/server/guard.go:46` (LOCAL_ONLY_PATHS mcp entry) | `internal/admin/mcp.go`, `internal/admin/skills.go`, `internal/mcp/agent.go` (+tests) | YES (serial slot) | corrects `mocks/handlers/mcp.ts` + `mocks/{handlers,seed}/skills.ts` to mirror Go | **w7-mcp-1, w7-mcp-2** |
| **w7-plat-1** | **Proxy-pools backend + outbound proxy / SSRF**: proxy_pools table (JSON data col, additive), CRUD + isActive/includeUsage filter, bound-connection guard (409), connectivity-test (HTTP ProxyAgent HEAD / relay probe, writes testStatus/lastTestedAt), per-connection proxy resolution wiring. Closes PAR-AUTH-020 (outbound proxy + SSRF mitigation). | PAR-PLAT-001, 002, 003, 004, 005, 009; PAR-AUTH-020(→HAVE); PAR-UI-019(→HAVE), PAR-UI-104(→HAVE), PAR-UI-105(→HAVE); open-questions w6-m ESC-1b | `internal/platform/doc.go` (placeholder), `internal/store/migrate.go`, `internal/inference/selection.go` (per-conn proxy hook — also touched by w7-route, COORDINATE), `routes_admin.go` | `internal/store/proxypools.go`, `internal/platform/proxypools.go`, `internal/platform/outboundproxy.go`, `internal/admin/proxypools.go` (+tests) | YES (serial slot) | corrects `mocks/handlers/proxy-pools.ts` to mirror Go; flips w6-m PARTIAL→HAVE | coordinate selection.go edit window with **w7-route** (serialize) |
| **w7-plat-2** | **Tunnels backend**: cloudflared (download+magic-byte validate, `tunnel run --token`, quick-tunnel URL extract, kill) + tailscale (install, daemon userspace/TUN, login poll, funnel, cert) process management; tunnel status/enable/disable + health; tunnel token `*_enc`. | PAR-PLAT-015, 016, 017, 018, 019, 020, 021, 022, 023; PAR-UI-112(→HAVE), PAR-UI-113(→HAVE), PAR-UI-114(→HAVE); open-questions w6-m ESC-1c | `internal/server/guard.go:135-141` (tunnelDashboardAccess guard — consume, don't edit), `internal/platform/doc.go`, `routes_admin.go` | `internal/store/tunnels.go`, `internal/platform/tunnel/cloudflared.go`, `internal/platform/tunnel/tailscale.go`, `internal/admin/tunnels.go` (+tests) | YES (serial slot) | corrects `mocks/handlers/tunnels.ts` to mirror Go; flips w6-m PARTIAL→HAVE | — (∥ plat-1/3; disjoint files except routes_admin serial) |
| **w7-plat-3** | **MITM backend**: Root CA generation + CA-cert serving (raw PEM), HTTPS MITM reverse proxy (SNI cert cache, ALPN), per-tool config + enable/toggle, restart backoff. Defer system-trust-store auto-install + hosts-file patching (OS-privileged) to escalation. | PAR-PLAT-024, 025, 028; PAR-UI-013(→HAVE); open-questions w6-m ESC-1a | `internal/platform/doc.go`, `crypto/tls`+`net/http` (Go-port consideration), `routes_admin.go` | `internal/store/mitm.go`, `internal/platform/mitm/ca.go`, `internal/platform/mitm/proxy.go`, `internal/admin/mitm.go` (+tests) | YES (serial slot) | corrects `mocks/handlers/mitm.ts` to mirror Go (CA-cert raw-PEM path); flips w6-m PARTIAL→HAVE | — (∥ plat-1/2) |
| **w7-misc** | **Remaining W6-deferred small backends**: console-logs SSE (`GET /api/console-logs/stream` over server log pipeline), translator backend (`/api/translator/{load,save,translate,send}` over request-logger transformation), models test/availability/custom (`/api/models/{test,availability,custom}`), version/shutdown OIDC-secret-at-rest (`oidc_secret_enc`), chat-sessions admin CRUD (OPTIONAL — 9router keeps in localStorage; scope call). | open-questions w6-i ESC-1/1a/2 (console-logs, chat-sessions, translator), w6-f ESC-3 (models test/availability/custom), w6-j ESC-4 (OIDC secret-at-rest) | `internal/admin/version.go` (OIDC secret), `internal/admin/nodes.go` (models endpoints), `internal/logging/` (console stream), `routes_admin.go` | `internal/admin/console.go`, `internal/admin/translator.go`, `internal/admin/models.go` (+tests); `internal/store` oidc_secret_enc additive | YES (serial slot — LAST) | corrects `mocks/handlers/{logs,translator,nodes,version}.ts` to mirror Go | — |
| **w7-usage-quota** | **Provider quota fetchers (Stage-2 half)**: remaining 6 provider usage adapters (GitHub, Antigravity, Codex, Kiro, GLM, MiniMax) behind the existing `/api/usage/{connectionId}` dispatcher (w5-e shipped claude+gemini). | PAR-USAGE-032(→HAVE) | `internal/admin/connectionusage.go` (w5-e dispatcher + claude/gemini fetchers — extend), `internal/usage/` | `internal/usage/quota_<provider>.go` fetchers (+tests) | NO (dispatcher already routed at `connectionusage.go`) | none (provider quota is API-flow; quota page consumes existing endpoint) | may depend on **w7-prov-oauth** (Codex/Kiro/GitHub OAuth tokens) |

---

## Concurrency waves + impl order

`internal/` domains are largely disjoint, so most plans run concurrently. The
single shared hot file is `internal/server/routes_admin.go` — its holders run in
a defined **serial chain**; a secondary micro-serial governs `internal/inference/
factory.go` (w7-prov-special) and `internal/inference/selection.go` (w7-route ↔
w7-plat-1, which both add hooks). Orchestrator caps at ≤4 concurrent jobs.

```
Catalog track (no routes_admin, no shared files — start immediately, run throughout):
  w7-prov-openai  ──▶  (catalog/models/aliases only; disjoint)
  w7-prov-oauth   ──▶  (auth/oauth + flows map; near-disjoint)
  w7-prov-special ──▶  (factory.go micro-serial; specialized adapters)
  w7-prov-media   ──▶  (scope-call; may defer to escalation)
  w7-usage-quota  ──▶  (connectionusage dispatcher; after prov-oauth tokens)

MCP track (disjoint internal/mcp — start immediately):
  w7-mcp-1 ──▶ w7-mcp-2 ──▶ w7-mcp-3 (mcp-3 takes routes_admin serial slot)

Platform track (disjoint internal/platform except routes_admin serial):
  w7-plat-1 ∥ w7-plat-2 ∥ w7-plat-3   (serialize their routes_admin slots;
                                        plat-1 serialize selection.go vs w7-route)

Governance + routing track (routes_admin serial holders):
  w7-gov-1 ∥ w7-gov-2 ∥ w7-gov-3      (disjoint domain/store files;
                                        serialize routes_admin slots)
  w7-platnodes ──▶ w7-route           (platnodes is the routing prerequisite)

routes_admin.go SERIAL CHAIN (one unmerged holder at a time), ordered:
  w7-platnodes → w7-route → w7-gov-1 → w7-gov-2 → w7-gov-3
               → w7-mcp-3 → w7-plat-1 → w7-plat-2 → w7-plat-3 → w7-misc
  (w7-prov-oauth registers in the flows map, not the route table; it takes a
   brief slot only if it appends an /api/oauth route — sequence it after w7-misc
   or coordinate as a trivial append.)
```

Reasons:
- **w7-platnodes FIRST in the serial chain**: provider-node prefix routing is the
  prerequisite for w7-route (PAR-ROUTE-009/040) and touches the
  inference resolver; landing it early unblocks routing.
- **Catalog track parallel to everything**: `internal/providers/catalog/*` is a
  pure config surface with no route registration — fully disjoint, runs the whole
  wave. w7-prov-openai is the cheapest high-row-count plan; start it day one.
- **MCP track is independent** (`internal/mcp` is greenfield + disjoint) but
  internally serial (foundation → client → transport); only w7-mcp-3 touches
  routes_admin.
- **Platform plats parallel by domain** but serialize routes_admin slots; w7-plat-1
  and w7-route both edit `selection.go` (proxy hook / weighted selection) — these
  two serialize that file via the orchestrator.
- **Governance gov-1/2/3 parallel** (disjoint store/domain/admin files); they
  queue for the routes_admin slot in order.

---

## e2e / mock reconciliation (binding)

Wave 6 left a registered mock for every deferred surface (`ui/e2e/mocks/handlers/
*` + seeds). Wave 7's rule (decision 1): **real Go wins, mock body corrected to
mirror it, in the same plan**. Concretely per plan:
- Governance/routing/platform/MCP plans that own a W6 mock MUST, as a closing
  step, diff the real Go `{data,error}` snake_case response against the mock body
  + seed and correct the mock to mirror reality (never the reverse). The owning
  W6 spec stays green against the corrected mock; the plan additionally lands a
  Go handler integration test (`internal/admin/<domain>_test.go`) as the
  authoritative proof of the real behavior.
- Shared mock-handler bodies (`mocks/handlers/index.ts`, seeds) are hot —
  orchestrator-ordered trivial appends/edits, same as W6.
- Where a W6 page consumes a path that the real Go diverges from (e.g. combos
  `{id,strategy,steps[]}` mock vs Go `{name,models[]}`), the plan decides: adapt
  the page to the real DTO (preferred, frozen-file serial follow-up via
  orchestrator) OR keep a compatibility shape in the Go DTO. Default: real Go DTO
  is canonical; page adaptation is a tracked serial follow-up, NOT an in-plan UI
  redesign (decision 8).

---

## Freeze rules

- **Wave-6 UI is frozen / consume-only.** No new pages, components, route files,
  stores, or layout edits. The only sanctioned UI touches are (a) mock/seed body
  corrections to mirror real Go, (b) a page path swap when Go diverges from the
  mock path (orchestrator serial follow-up).
- **w6-a/w6-b frozen sets** (root, layout, stores, lib, ui primitives) remain
  consume-only.
- **`internal/server/routes_admin.go`**: one unmerged holder at a time, in the
  serial chain order above. Additive appends only.
- **`internal/inference/factory.go` + `selection.go`**: micro-serial for the two
  plans that edit them (w7-prov-special on factory; w7-route ↔ w7-plat-1 on
  selection); additive only.
- **Migrations**: additive `ensureColumn`/`ensureTable` only; no destructive DDL.

---

## Escalations / blockers / unknowns (explicit)

1. **Deploy relays (PAR-PLAT-006/007/008 + PAR-UI-106/107/108)** — Vercel /
   Cloudflare-Workers / Deno-Deploy relay deployment. Heavy external-API surface
   (multipart worker upload, deployment polling, SSO toggling), niche value.
   RECOMMEND: defer to a post-W7 escalation; record the 6 rows as
   explicitly-deferred (NOT counted in W7 100%-of-feasible) unless the operator
   wants them. **DECISION NEEDED.**
2. **Cookie-auth reverse-engineered providers (PAR-PROV-030 perplexity-web,
   PAR-PROV-031 grok-web)** — fragile, reverse-engineered web endpoints. Matrix
   Go-port note: "defer until after GA." RECOMMEND defer (escalation), exclude
   from w7-prov-special's committed set. **DECISION NEEDED.**
3. **Translation Stage-2 rows (PAR-TRANS-006 strip content-types, PAR-TRANS-050b
   responses-passthrough stream)** — both blocked on the multi-part `Message.
   Content` schema (Stage-1 is `Content string`). These require a schema change
   (NOT additive-trivial). RECOMMEND: a dedicated schema-evolution plan OR defer
   to the responses-passthrough wave; NOT in the current W7 backend set.
   **DECISION NEEDED.**
4. **PAR-AUTH-003 (HS256 JWT sessions)** — g0router deliberately uses opaque DB
   tokens (more secure, revocable). This is a **variant decision**, not a gap.
   RECOMMEND: record as variant (do NOT build JWT); it stays MISSING-by-design
   with rationale, or flip to a "variant-HAVE" note. **DECISION NEEDED** on
   matrix labeling.
5. **MCP Cowork / Claude-Desktop 3p integration (PAR-MCP-018..023, 045..050) and
   Cursor MCP protobuf (PAR-MCP-026..029)** — the Cowork feature is *globally
   disabled* in the 9router reference (PAR-MCP-042 EXTRA); the protobuf encoders
   only matter if the cursor provider adapter is built. RECOMMEND: scope the
   3p-config writer + toolPolicy generator into w7-mcp-3 ONLY if cheap; otherwise
   defer (low parity value given ref disables it). Cursor protobuf rides
   w7-prov-special's cursor adapter or is deferred with it.
6. **Media-provider adapters (w7-prov-media, 12 rows)** — image/video/stt/tts/
   embedding specialists map to non-chat `Provider` interface methods with no UI
   page (media-providers UI was W6-deferred). The provider matrix Stage-1 ranking
   defers all of these. RECOMMEND: scope catalog entries (cheap) but defer the
   actual non-OpenAI-shaped adapters to a media escalation unless the operator
   wants media parity in W7. **DECISION NEEDED.**
7. **Platform OS-privileged operations** — MITM system-trust-store auto-install
   (PAR-PLAT-025 partial), hosts-file patching (PAR-PLAT-026/027), tray/auto-
   start/auto-update (PAR-PLAT-030..036), CLI-tools auto-config (PAR-PLAT-037/038),
   IDE token auto-import (PAR-PLAT-039..043). These need elevated privileges /
   are desktop-app concerns ill-suited to a server binary. RECOMMEND: scope CA
   generation + serving + MITM proxy + config (the dashboard-controllable half);
   defer trust-store install, hosts patching, tray, auto-update, CLI auto-config,
   and IDE token import to a "desktop/agent" escalation (per matrix Go-port note:
   "drop for server binary; add `g0router service install`"). **DECISION NEEDED.**

---

## Out of Wave-7 scope (explicit)

- **bifrost-* program** entirely (`matrix/bifrost-core.md`, `bifrost-governance.md`,
  `bifrost-mcp.md`, `bifrost-openai.md`, `g0router-audit.md`) — user scope: finish
  9router first.
- **Genuine-UI / SKIP UI rows** (NOT backend, handled in W6 or permanently
  skipped): PAR-UI-004 `/landing` (SKIP), PAR-UI-014/015 cli-tools pages,
  PAR-UI-022/023/024 media-providers pages (no contract — only relevant if
  w7-prov-media + a UI wave land), PAR-UI-061 NineRemotePromoModal (SKIP),
  PAR-UI-121 Next.js App Router (SKIP — Vite+TanStack is the standing decision),
  PAR-UI-028/029 (sidebar media-accordion / header donate slots — W6 PARTIAL,
  cosmetic UI follow-ups, not backend), PAR-UI-070/071 (i18n DOM-translation —
  permanently replaced by react-i18next variant).
- Deploy relays, cookie-auth web providers, translation Stage-2, media adapters,
  OS-privileged platform ops — pending escalation decisions 1/2/3/6/7 above.

---

## Protocol (W6 protocol, Go-weighted)

Per micro-plan: plan → plan gate (≤3 cycles → decide) → TDD impl (Go `_test.go`
first, see fail, minimum code to pass; no mocks, fakes via interfaces) → gates →
scoped diff gate (commit-bounded; live-tree verification before closure) → merge
→ flip matrix rows → mock-mirror correction → `docs/WORKFLOW.md` update.
Commits: `phase-1/w7-X: <description>` (per `AGENTS.md` convention; push direct
to main, quality gates local).

Per-commit gates (every commit): `go test ./... && go vet ./...` green,
`go build ./...` green. For plans that correct a mock or touch UI wiring:
`cd ui && npm run build` green + the affected scoped `npx playwright test
e2e/<specs>` green. Full `npx playwright test` at each plan that corrects a mock,
and at every wave boundary.
