# Workflow — AI Agent Handoff Protocol

## How to Use This File

1. Read `CLAUDE.md` for behavioral rules.
2. Read `docs/ORCHESTRATION.md` for the parallel execution model.
3. Check the current state below before doing any work.
4. If `project_status` is `COMPLETE`, do not infer a next task from old wave history.
5. If new work is explicitly added later, create a new wave before implementation.
6. Complete new tasks following TDD.
7. Run the relevant gates before committing.
8. Commit: `phase-N/task-M: <description>`.
9. Update status to `DONE`.
10. When ALL tasks in a new wave are `DONE`, orchestrator merges, verifies, and records evaluator results.

## Status Values

| Status | Meaning |
|--------|---------|
| `PENDING` | Not started |
| `IN_PROGRESS` | Agent is actively working |
| `BLOCKED` | Cannot proceed — reason in notes |
| `DONE` | Complete, tests pass, committed |
| `SKIPPED` | Deliberately not doing (reason in notes) |

---

## Current State

```yaml
project_status: IN_PROGRESS
current_stage: "Parity Stage 1 Wave 1"
current_wave: "Translation engine — w1-a..f implemented; diff gates pending; w1-g+ not planned"
last_completed_plan: "w1-f cloud envelope (Kimi, tasks 0-7, commit 5d629345)"
last_updated: "2026-06-10T14:45:00Z"
orchestrator: "Claude Code (VPS) — see .planning/harness/HANDOFF.md"
planner: "Fable 5"
implementer: "Kimi"
notes: |
  Harness handoff ready for VPS: HANDOFF.md, VPS-SETUP.md, diff-scopes.json,
  run-diff-scoped.sh, templates/ for job dispatch.
  w1-c/d/e/f need scoped diff gate re-run + blocker fixes (see HANDOFF open work).
  Next: Fable drafts w1-g (Responses API), then w1-h..j.
  go test ./... green at HEAD.
```

---

## Parity Program — Stage 0

### A0 — CLI harness + frozen reference sources

```yaml
task: parity-a0
status: DONE
summary: |
  Cloned frozen reference repos: decolua/9router @ 827e5c3 (MIT, v0.4.71)
  and maximhq/bifrost @ ca21298 (Apache-2.0) into ~/Developer/github.com/
  bloodf/_refs/. SHAs + license decisions recorded in
  .planning/parity/SOURCES.md. Scaffolded .planning/harness/ with
  run-worker.sh, run-critic.sh, run-gates.sh, parse-verdict.sh, prompt
  templates (analyzer-base, critic-plan, critic-diff with stop-slop rules
  inlined), and README.md pinning model invocations. Smoke-tested all four
  lanes: kimi -p (works; lingers after completion — timeout wrapper, exit
  124 != failure; file writes confirmed), pi MiniMax-M3, pi
  MiniMax-M2.7-highspeed with read/bash tools, pi gpt-5.5 no-tools critic
  (VERDICT: PASS round-trip through run-worker.sh + parse-verdict.sh).
  Kimi quirk: -y/--auto cannot combine with -p.
completed_at: "2026-06-09T21:25:00Z"
verdict: PASS
```

---

## v2.0 Milestone — 9router + BiFrost Clean Slate Port

Wave 2 of 6. 19 phases total in `.planning/ROADMAP.md`.

### Phase 6 — Management API Foundation

```yaml
phase: 6
status: DONE
summary: |
  Built the admin API foundation with strict TDD (RED before GREEN per task).
  internal/store: SQLite via modernc.org/sqlite (pure Go), WAL mode,
  additive-only ensureColumn migrations, AES-256-GCM cipher with key
  auto-generated at <datadir>/secret.key (0600, no env vars), repositories
  for users/sessions/settings/providers/connections/oauth_sessions with
  secrets at rest in *_enc columns. internal/auth: PBKDF2-SHA256 password
  hashing (210k iters), session manager with seeding, PKCE (S256) OAuth flow
  with single-use server-side state (Anthropic config included).
  internal/admin: {data, error} snake_case envelope handlers — login/logout/
  me, settings GET/PUT, provider CRUD, connection CRUD with masked secrets
  (*_set booleans; empty update fields preserve stored secrets), OAuth
  start/callback/refresh, RequireSession middleware (Bearer or g0_session
  cookie). internal/server/routes_admin.go wires it all; main.go opens the
  store under ~/.g0router (G0ROUTER_DATA override) and seeds admin/123456 on
  first run only. End-to-end integration test drives the full surface over
  an in-memory listener with a fake OAuth token endpoint.
completed_at: "2026-06-09T19:10:00Z"
verdict: PASS
gates:
  - { command: "go test ./...", status: PASS, notes: "30 packages ok" }
  - { command: "go vet ./...", status: PASS }
  - { command: "go build ./...", status: PASS }
  - { command: "manual curl smoke", status: PASS, notes: "real binary: login, provider+connection create, 401 unauthenticated" }
tasks:
  - "task-1: SQLite store — WAL, migrations, cipher, repositories (DONE)"
  - "task-2: auth — password hashing, sessions, PKCE OAuth flow (DONE)"
  - "task-3: admin handlers — auth/settings/providers/connections/oauth (DONE)"
  - "task-4: wire admin routes + main.go store/seed bootstrap (DONE)"
  - "task-5: end-to-end management API integration test (DONE)"
  - "task-6: gates + docs + summary (DONE)"
commits:
  - "cbad6f80 phase-06/task-1: SQLite store with WAL, additive migrations, AES-GCM cipher, and repositories"
  - "e1a4a869 phase-06/task-2: auth package with PBKDF2 password hashing, session manager, and PKCE OAuth flow (Anthropic)"
  - "31d0f43c phase-06/task-3: admin handlers — login/logout/me, settings, provider CRUD, connection CRUD with masked secrets, OAuth start/callback/refresh"
  - "35618972 phase-06/task-4: wire admin routes into fasthttp server; main.go opens store, seeds default admin, serves management API"
  - "7c5d4a82 phase-06/task-5: end-to-end management API integration test"
caveats:
  - "Auth routes are /api/auth/* (execution brief) rather than PLAN.md's /api/login — Phase 7 UI must target /api/auth/*."
  - "No env-based default password (AGENTS.md forbids secret env vars): seeded admin/123456 on first run, change-password endpoint deferred to Phase 7."
  - "Provider 'suggested models' endpoint deferred — no catalog data yet (Phase 9)."
  - "DeleteExpiredSessions exists but is not yet scheduled periodically."
next: "Phase 7 — Dashboard Shell + Providers UI"
```

### Phase 1 — Scaffolding

```yaml
phase: 1
status: DONE
summary: |
  Wiped v1 code (api/, internal/, ui/src/, root e2e_*.go) and scaffolded the
  new 14+14 internal package layout with placeholder tests. Wrote a minimal
  fasthttp cmd/g0router/main.go that serves GET /api/health and a SPA-style
  catch-all from the embedded UI. Wrote a minimal Vite+React 19 UI
  placeholder (main.tsx + App.tsx + index.css + routes/__root.tsx) so
  `npm run build` ships a real bundle with the required <div id="root"></div>.
  go.mod tidied to a single direct dep (fasthttp). The new architecture is
  documented in docs/superpowers/specs/2026-06-08-9router-bifrost-port-design.md.
completed_at: "2026-06-09T15:00:00Z"
plan: plan_63b4da91
verdict: PASS
gates:
  - { command: "go test ./...", status: PASS, notes: "30 ok / 1 ignored (ui/node_modules pre-existing)" }
  - { command: "go vet ./...", status: PASS }
  - { command: "cd ui && npm run build", status: PASS, notes: "193.87 kB JS, 477ms" }
  - { command: "go build ./cmd/g0router", status: PASS, notes: "9.5 MB binary" }
  - { command: "npx playwright test --list", status: PASS, notes: "79 tests in 30 files, no crash" }
adversarial_probes:
  - "Started binary, hit /api/health, /, /dashboard, /assets/*.js — all 200 OK"
  - "No leftover internal/cli imports"
  - "No leftover github.com/bloodf/g0router/api/ imports"
  - "All 28 internal packages contain only doc.go + _test.go (no half-implementations)"
  - "go.mod is minimal (1 direct dep)"
  - "All 28 placeholder _test.go files have exactly 1 func Test each"
  - "No TODO/FIXME/panic in placeholders"
  - "embed_test.go is a real integration test, not a placeholder"
commits:
  - "6338148 phase-01/task-1: remove obsolete api/, internal/, and root e2e tests"
  - "63124ba phase-01/task-2: scaffold internal/ package layout with placeholder tests"
  - "c900b55 phase-01/task-3: rewrite cmd/g0router/main.go as minimal fasthttp skeleton"
  - "e36a19c phase-01/task-4: go mod tidy"
  - "79db515 phase-01/task-1: scaffold minimal UI placeholder (main.tsx, App.tsx, index.css)"

caveats:
  - "Naming collision: phase-01/task-1 prefix used by both Go and UI tracks. Producers worked on disjoint files; no conflict. Future parallel plans: prefix UI tasks differently (e.g. phase-N/ui-task-N)."
  - "Optional cleanup: rm -rf api on macOS hosts drops a stray gitignored .DS_Store that the OS re-created in the post-deletion empty folder. Not tracked, not in any gate. Defer to a later phase."
  - "Dirty pre-existing WIP at handoff: ui/e2e/mocks/{fixture,handlers,handlers/catalog}.ts and docs/superpowers/specs/2026-06-08-9router-bifrost-port-design.md. Phase 1 left them as-is; commit or clean in Phase 2+."

deliverable: "deliverable.md (335 lines, full final-gate report)"

next: "Phase 2 — Schemas + Catalog (internal/schemas/ + internal/catalog/)"
```

---

## Wave E2E — Comprehensive end-to-end test suite

```yaml
wave: "E2E"
status: DONE
summary: "Built comprehensive e2e tests covering all 38+ API endpoints and 64+ UI screens/flows. Fixed UI bugs found during testing (aliases created_at, audit empty SelectItem, combos null steps). Added startup ASCII art banner. Added Makefile e2e orchestration target."
completed_at: "2026-06-07T20:56:00Z"
```

**Gate Results:**
- `go test ./... -count=1`: PASS (all packages green)
- `go vet ./...`: PASS
- `go build ./cmd/g0router`: PASS
- `npm --prefix ui run build`: PASS
- `make e2e`: PASS (38 Go API tests + 64 UI Playwright tests)

**Tasks:**
- task-1: Comprehensive Go API e2e tests — 38 endpoint tests (`e2e_api_comprehensive_test.go`)
- task-2: Comprehensive UI Playwright e2e tests — 21 spec files, 64 tests (`ui/e2e/*.spec.ts`)
- task-3: Fixed Aliases page — removed Created column using non-existent backend field
- task-4: Fixed Audit page — filtered empty-string actor values from Select dropdown
- task-5: Fixed Combos page — guarded `combo.steps` against undefined/null from backend
- task-6: Fixed apiFetch response.ok check + backend envelope unwrapping
- task-7: Fixed `listResponse[T].MarshalJSON` to convert nil slices to `[]`
- task-8: Added unique per-run suffixes to comprehensive CRUD test names to avoid UNIQUE constraint failures
- task-9: Set Playwright `workers: 1` to prevent SQLite concurrency issues
- task-10: Added `make e2e` Makefile target orchestrating build + Go API tests + UI tests
- task-11: Added startup ASCII art banner with version, port, and links (`api/server.go`)

---

## Wave UI-2 — Zero-config startup + default admin

```yaml
wave: "UI-2"
status: DONE
summary: "Removed API_KEY_SECRET env requirement at startup. Server now auto-generates and stores the secret in DB if missing. Creates default admin user (admin/123456) on first startup. Added CLI secret command and full Settings page in UI with api_key_secret field."
completed_at: "2026-06-07T07:18:52Z"
```

**Gate Results:**
- `go test ./... -count=1`: PASS (all packages green)
- `go vet ./...`: PASS
- `go build ./cmd/g0router`: PASS
- `npm --prefix ui run build`: PASS
- Coverage: 95.0%

**Tasks:**
- task-1: Removed `API_KEY_SECRET` hard requirement from `internal/config/config.go`
- task-2: Added `GetAPIKeySecret()` / `SetAPIKeySecret()` to store (`internal/store/settings.go`)
- task-3: Auto-generate 64-hex `API_KEY_SECRET` on startup if env + DB are empty (`internal/cli/root.go`)
- task-4: Auto-create default admin user `admin` / `123456` on first startup if no users exist
- task-5: Added `SeedDefaultAdminUser` store method bypassing 8-char password minimum
- task-6: Updated settings handler to accept `api_key_secret` in PUT body (`api/handlers/settings.go`)
- task-7: Added CLI `g0router secret` command to view stored secret
- task-8: Built full Settings page in UI (`ui/src/routes/_app.settings.tsx`) with all backend fields + secret field

---

## Wave UI-1 — g0-route-guard Dashboard Integration

```yaml
wave: "UI-1"
status: DONE
summary: "Replaced the existing ui/ dashboard with the new g0-route-guard template, converted from TanStack Start SSR to Vite SPA, wired apiFetch to real backend endpoints, added missing backend endpoints, and fixed response shape mismatches."
completed_at: "2026-06-07T07:18:52Z"
```

**Gate Results:**
- `go test ./... -count=1`: PASS (all packages green, excluding ui/node_modules)
- `go vet ./...`: PASS
- `go build ./cmd/g0router`: PASS
- `npm --prefix ui run build`: PASS (static SPA output in ui/dist/)
- Coverage: 95.0%

**Tasks:**
- task-1: Copied new dashboard from g0-route-guard template into ui/
- task-2: Converted TanStack Start → Vite SPA (replaced vite.config.ts, added index.html + main.tsx, removed server.ts/start.ts, fixed __root.tsx shellComponent)
- task-3: Replaced mock API client with real fetch to /api/* (ui/src/lib/api/client.ts)
- task-4: Backend gap analysis — documented 60 mock endpoints vs backend routes (docs/technical/ui-api-gap-analysis.md)
- task-5: Fixed apiFetch response envelope handling (wrapped {data,error} vs raw objects)
- task-6: Fixed list response shapes for /api/usage and /api/audit (normalizeListResponse helper)
- task-7: Added missing backend endpoint GET /api/models (api/handlers/models_admin.go)
- task-8: Added missing backend endpoint POST /api/keys/:id/regenerate (api/handlers/apikeys.go + store method)
- task-9: Added missing backend endpoint GET /api/quota (api/handlers/usage.go QuotaAggregate)
- task-10: Fixed Pricing page compatibility — backend now returns pricingOverrideResponse with id, input_cost, output_cost; accepts UI field aliases
- task-11: Fixed audit page actor field mismatch (actor → actor_api_key_id)

**Known follow-ups (non-blocking):**
- Settings field mismatches (theme, language, tunnel_dashboard_access) — UI mock-only fields
- Connections models[] field — backend uses ModelLocks map instead
- MITM endpoints — pages are ComingSoon, mock routes unused
- Diagnostics endpoint — page is ComingSoon

---

## Phase 12B — DDD & Architecture Refactor

```yaml
phase: 12B
status: DONE
summary: "Layered DDD-lite refactor: routing extraction, repository interfaces, usage domain extraction, inference pipeline boundary, handler hygiene sweep, architecture conformance test."
commit_range: "7f6e1b2..600c4bd"
completed_at: "2026-06-06T01:24:55Z"
```

**Gate Results:**
- `go test ./... -count=1`: PASS (all 44 packages green)
- `go vet ./...`: PASS
- `go test -race ./...`: PASS
- `go build ./cmd/g0router`: PASS
- Coverage: 95.0%

**Tasks:**
- task-1: Routing table extraction — `api/routes.go`, `api/wiring.go`, `api/routes_test.go`
- task-2: Repository interfaces — 25+ consumer-defined narrow interfaces
- task-3: Usage domain extraction — `internal/usage/usage.go`, `usage_test.go`
- task-4: Inference pipeline boundary — `internal/proxy/pipeline.go`, `pipeline_test.go`
- task-5: Handler hygiene sweep — split `inference.go`, moved Anthropic translation to `internal/translate`
- task-6: Architecture conformance test — `internal/archtest/arch_test.go`

**Next:** Phase 13 — Auth & Core Infrastructure

---

## Phase 13 — Auth & Core Infrastructure

```yaml
phase: 13
status: DONE
summary: "Password-based dashboard authentication with server-side sessions, CSRF protection, first-run setup, login rate limiting, and minimal user management. Coexists with existing bearer/X-API-Key auth."
commit_range: "401c4df..287d336"
completed_at: "2026-06-06T07:18:52Z"
```

**Gate Results:**
- `go test ./... -count=1`: PASS (all 45 packages green)
- `go vet ./...`: PASS
- `go test -race ./...`: PASS
- `go build ./cmd/g0router`: PASS
- Coverage: 95.0%

**Tasks:**
- task-1: Store — `dashboard_users.go` CRUD + bcrypt (`internal/store/dashboard_users.go`, `dashboard_users_test.go`)
- task-2: Store — `dashboard_sessions.go` CRUD + SHA-256 (`internal/store/dashboard_sessions.go`, `dashboard_sessions_test.go`)
- task-3: Handlers — setup/login/logout/status (`api/handlers/auth.go`, `auth_test.go`)
- task-4: Middleware — session validation + coexistence + exempt routes + Origin CSRF check (`api/middleware.go`, `middleware_session_test.go`)
- task-5: Handlers — password change + users CRUD + last-admin guard (`api/handlers/auth.go`)
- task-6: Settings — `require_login`, `trust_proxy_headers` + lockout guard (`internal/store/settings.go`, `api/handlers/settings.go`)
- task-coverage: Auth handler error branch + audit handler coverage (`api/handlers/auth_test.go`, `audit_test.go`)

**Security Review (mandatory):**
- Input validation: ✅ All auth endpoints validate JSON and password length (min 8 chars)
- Authn/authz: ✅ Every new route has correct auth — exempt routes public, admin routes require admin role, coexistence with bearer keys
- Secrets at rest: ✅ Passwords hashed with bcrypt, session tokens SHA-256 hashed in DB
- Secrets in logs: ✅ Passwords and tokens never logged; audit entries exclude sensitive data
- Supply-chain: ✅ No external downloads; `golang.org/x/crypto/bcrypt` standard library
- Privilege requirements: ✅ Admin role required for user management; last-admin guard prevents lockout; require_login lockout guard

**Next:** Phase 14 — Providers & Testing

---

## Phase 14 — Providers & Testing

```yaml
phase: 14
status: DONE
summary: "Provider detail APIs, model testing (single + batch SSE), proxy pools with encrypted credentials, disabled/custom model management, and proxy wiring into provider HTTP clients."
commit_range: "90c63b7..e567421"
completed_at: "2026-06-06T09:57:00Z"
```

**Gate Results:**
- `go test ./... -count=1`: PASS (all 46 packages green)
- `go vet ./...`: PASS
- `go test -race ./...`: PASS
- `go build ./cmd/g0router`: PASS
- Coverage: 95.0%

**Tasks:**
- task-1: Store — `proxy_pools` CRUD with AES-GCM encrypted password (`internal/store/proxypools.go`, `proxypools_test.go`)
- task-2: Store — `disabled_models` + `custom_models` CRUD (`internal/store/models_mgmt.go`, `models_mgmt_test.go`)
- task-3: Handlers — proxy pools CRUD/test/batch import (`api/handlers/proxypools.go`, `proxypools_test.go`)
- task-4: Proxy wiring — HTTP/HTTPS/SOCKS5 proxy into provider clients (`internal/providers/utils/proxy.go`, `proxy_test.go`)
- task-5: Handlers — provider detail/connections/suggested-models (`api/handlers/providers.go`, `providers_test.go`)
- task-6: Handlers — model test single + batch SSE (`api/handlers/modeltest.go`, `modeltest_test.go`)
- task-7: Disabled/custom model filtering in listings and routing (`api/handlers/models_mgmt.go`, `internal/proxy/engine.go`)
- task-coverage: Error branch coverage for all new Phase 14 code

**Next:** Phase 15 — Tunnels & Network

---

## Phase 15 — Tunnels & Network

```yaml
phase: 15
status: DONE
summary: "Cloudflare and Tailscale tunnel management with checksum-verified binary downloads, process supervision, tunnel health checks, proxy connectivity testing, and proxy pool auto health checks."
commit_range: "f8e2943..39aa6d7"
completed_at: "2026-06-06T09:57:00Z"
```

**Gate Results:**
- `go test ./... -count=1`: PASS (all 47 packages green)
- `go vet ./...`: PASS
- `go test -race ./...`: PASS
- `go build ./cmd/g0router`: PASS
- Coverage: 95.0%

**Tasks:**
- task-1: Store — `tunnel_config` CRUD with encrypted config (`internal/store/tunnels.go`, `tunnels_test.go`)
- task-2: Tunnel package — checksum-verified download, process supervisor, tunnel manager (`internal/tunnel/download.go`, `supervisor.go`, `tunnel.go`)
- task-3: Handlers — tunnel CRUD/health + proxy-test (`api/handlers/tunnels.go`, `tunnels_test.go`)
- task-4: Background health loops — tunnel (60s) + proxy pool (5min) (`api/server.go`, `server_health_test.go`)
- task-coverage: Tunnel package error branch coverage

**Security Review (mandatory):**
- Input validation: ✅ Tunnel name `[a-z0-9-]{1,63}`; port validation; no shell interpolation
- Authn/authz: ✅ All tunnel mutations require admin-session or bearer auth
- Secrets at rest: ✅ Tunnel config encrypted with AES-GCM
- Secrets in logs: ✅ CLI stderr discarded; tokens never logged
- Supply-chain: ✅ Cloudflared download pinned version + SHA-256 per OS/arch; checksum verified before chmod+exec; HTTPS from GitHub releases only
- Privilege requirements: ✅ Tailscale not downloaded by g0router; requires preinstalled binary on PATH; returns 409 with instructions if absent

---

## Phase 16 — Chat & Console

```yaml
phase: 16
status: DONE
summary: "SQLite-backed chat sessions with base64 image attachment validation, live console log streaming via SSE, ring buffer broker, slog tee handler, and log clearing endpoint."
commit_range: "81999c6..e5b9ce1"
completed_at: "2026-06-06T11:20:00Z"
```

**Gate Results:**
- `go test ./... -count=1`: PASS (all packages green)
- `go vet ./...`: PASS
- `go test -race ./...`: PASS
- `go build ./cmd/g0router`: PASS
- Coverage: 95.0%

**Tasks:**
- task-1: `internal/console/` — ring buffer + slog handler + broker + tests
- task-2: store — `chat_sessions` repository + `messages_json` validation + tests
- task-3: handlers — chat sessions CRUD + tests
- task-4: handlers — console SSE stream + clear + startup wiring + tests
- task-coverage: nil-store guards, levelString, dead code removal, proxy SOCKS5, encryption empty key

**Next:** Phase 17 — Usage & Analytics

---

## Phase 17 — Usage & Analytics

```yaml
phase: 17
status: DONE
summary: "Backend-bucketed time-series chart aggregation and bulk connection quota actions. SQLite strftime GROUP BY with Go zero-fill; bulk disable/enable via quota_limit/quota_remaining columns."
commit_range: "e5b9ce1..9cc6c93"
completed_at: "2026-06-06T14:34:00Z"
```

**Gate Results:**
- `go test ./... -count=1`: PASS (all packages green)
- `go vet ./...`: PASS
- `go test -race ./...`: PASS
- `go build ./cmd/g0router`: PASS
- Coverage: 95.0%

**Tasks:**
- task-1: store — `GetUsageChart` with strftime bucketing (day/hour), Go zero-fill, period range (`today/24h/7d/30d/60d`) + tests (`internal/store/usage.go`, `usage_test.go`)
- task-2: handler — `GET /api/usage/chart` param validation, default granularity logic, time-based validation + tests (`api/handlers/usage.go`, `usage_test.go`, `api/routes.go`)
- task-3: store + handlers — bulk disable/enable connections with `quota_limit`/`quota_remaining` columns, `BulkDisableConnectionsByThreshold`, `BulkEnableConnectionsWithQuota`, audit logging + tests (`internal/store/connections.go`, `connections_test.go`, `api/handlers/connections.go`, `connections_test.go`, `api/routes.go`, `internal/store/sqlite.go`)
- task-coverage: error branch coverage for nil-store, missing period, store error, audit error paths; chartTimeRange all periods; closed-DB query error paths

**Next:** Phase 18B — TBD

---

## Phase 18A — Virtual Keys, Teams, Governance

```yaml
phase: 18A
status: DONE
summary: "Virtual keys with hashed storage, team-based budget/rate-limit grouping, governance domain with budget reset rollover, and middleware integration for virtual key auth on inference endpoints."
commit_range: "9cc6c93..3f80d9c"
completed_at: "2026-06-06T16:32:00Z"
```

**Gate Results:**
- `go test ./... -count=1`: PASS (all packages green)
- `go vet ./...`: PASS
- `go test -race ./...`: PASS
- `go build ./cmd/g0router`: PASS
- Coverage: 95.0%

**Tasks:**
- task-1: store — `teams` + `virtual_keys` CRUD with budget tracking (`internal/store/governance.go`, `governance_test.go`)
- task-2: domain — governance check with lazy budget reset, rate limits, team inheritance (`internal/governance/governance.go`, `budget.go`, `governance_test.go`)
- task-3: handlers — `Teams` + `VirtualKeys` CRUD handlers (`api/handlers/teams.go`, `virtualkeys.go`, `teams_test.go`, `virtualkeys_test.go`)
- task-4: middleware — `validVirtualKey` integrated into `validAPIKey` chain (`api/middleware.go`, `middleware_virtualkey_test.go`)
- task-5: policy — `recordVirtualKeyUsage` wired into inference logging (`api/policy.go`)
- task-coverage: error branch coverage for nil store, store DB errors, validation errors, middleware rejections, governance edge cases (`*_coverage_test.go` files)

---

## Phase 18B — Routing Rules, Model Limits

```yaml
phase: 18B
status: DONE
summary: "TTL-cached routing rules with priority-ordered evaluation, model limits with RPM tracking and key allowlists, wired into proxy dispatch before alias/combo resolution."
commit_range: "8545f54..464500d"
completed_at: "2026-06-06T14:50:00Z"
```

**Gate Results:**
- `go test ./... -count=1`: PASS (all packages green)
- `go vet ./...`: PASS
- `go test -race ./...`: PASS
- `go build ./cmd/g0router`: PASS
- Coverage: 95.0%

**Tasks:**
- task-4: store — `routing_rules` + `model_limits` CRUD with migrations (`internal/store/routing.go`, `routing_test.go`)
- task-5: proxy — TTL-cached rule evaluator, model limit checker with RPM tracker, dispatch wiring (`internal/proxy/routing.go`, `dispatch.go`, `routing_test.go`, `dispatch_test.go`)
- task-coverage: MethodNotAllowed handlers, store error paths, previewResolveProvider fallback, loadRules store error (`*_coverage_test.go` files)

---

## Phase 18C — Guardrails, PII, Prompts, MCP Tool Groups

```yaml
phase: 18C
status: DONE
summary: "Blocklist + PII redaction guardrails wired into dispatch pipeline, prompt templates with {{var}} extraction, MCP tool groups for filtering injected tool sets."
commit_range: "e268c4b..ac55b59"
completed_at: "2026-06-06T15:20:00Z"
```

**Gate Results:**
- `go test ./... -count=1`: PASS (all packages green)
- `go vet ./...`: PASS
- `go test -race ./...`: PASS
- `go build ./cmd/g0router`: PASS
- Coverage: 95.0%

**Tasks:**
- task-6: guardrails — blocklist + PII redaction domain, settings-backed config, dispatch wiring, test endpoint (`internal/guardrails/`, `api/handlers/guardrails.go`)
- task-7: prompt templates — store + CRUD handlers + `{{var}}` extraction (`internal/store/prompttemplates.go`, `api/handlers/prompttemplates.go`)
- task-8: mcp tool groups — store + CRUD handlers + injection filtering (`internal/store/mcptoolgroups.go`, `api/handlers/mcptoolgroups.go`, `internal/mcp/inject.go`)
- task-coverage: guardrails error branches, PII types, prompt template validation, tool group resolution, pipeline wiring (`*_coverage_test.go` files)

---

## Phase 18D — Alerts, Feature Flags, Backup/Restore

```yaml
phase: 18D
status: DONE
summary: "Alert channels with encrypted config and webhook/discord/telegram dispatch, feature flags with seeded defaults, backup/restore with secret redaction and schema validation."
commit_range: "bc17c60..e6dd251"
completed_at: "2026-06-06T16:45:00Z"
```

**Gate Results:**
- `go test ./... -count=1`: PASS (all packages green)
- `go vet ./...`: PASS
- `go test -race ./...`: PASS
- `go build ./cmd/g0router`: PASS
- Coverage: 95.0%

**Tasks:**
- task-9: alert channels — store with `config_enc` encryption, alerts domain dispatch, CRUD + test handlers (`internal/store/alertchannels.go`, `internal/alerts/alerts.go`, `api/handlers/alertchannels.go`)
- task-10: feature flags — store with seeded defaults, toggle-only handlers (`internal/store/featureflags.go`, `api/handlers/featureflags.go`)
- task-11: backup/restore — export with secret redaction (`redacted_fields` manifest), restore with schema validation and null-placeholder secret preservation (`api/handlers/backup.go`, `internal/store/backup.go`)
- task-coverage: alert dispatch errors, flag edge cases, backup/restore error branches (`*_coverage_test.go` files)

**Security Review (mandatory):**
- Input validation: ✅ Alert channel names validated; backup schema version checked; restore rejects unknown shapes
- Authn/authz: ✅ All mutations require admin-session or bearer auth; backup/restore audited
- Secrets at rest: ✅ Alert config encrypted with AES-GCM (`config_enc`); backup exports null placeholders for secrets
- Secrets in logs: ✅ No tokens/passwords in alert dispatch logs; redacted backup manifest
- Supply-chain: ✅ No external downloads in this phase
- Privilege requirements: ✅ Backup/restore admin-only; feature flags read-only for non-admin

---

## Phase 19 — Advanced Features

```yaml
phase: 19
status: DONE
summary: "Semantic cache with cosine similarity, version check + auto-update with checksum verification, locale persistence, WebSocket chat protocol v1, MITM proxy with CA generation and per-host leaf certs, skills catalog endpoint."
commit_range: "aad2a50..2dd429a"
completed_at: "2026-06-06T21:30:00Z"
```

**Gate Results:**
- `go test ./... -count=1`: PASS (all packages green)
- `go vet ./...`: PASS
- `go test -race ./...`: PASS
- `go build ./cmd/g0router`: PASS
- Coverage: 95.0%

**Tasks:**
- task-1: version/update-check + locale + skills endpoints + tests (`internal/update/`, `api/handlers/version.go`, `api/handlers/locale.go`, `api/handlers/skills.go`)
- task-2: `internal/semcache/` domain + store + tests (`internal/semcache/semcache.go`, `internal/store/semcache.go`)
- task-3: semantic cache dispatch wiring + handlers + tests (`internal/proxy/dispatch.go`, `api/handlers/cache.go`)
- task-4: WebSocket chat endpoint + protocol v1 (`api/handlers/ws.go`)
- task-5: MITM proxy + CA generation + cert minting + handlers (`internal/mitm/`, `api/handlers/mitm.go`)
- task-coverage: mitm error branches, websocket auth rejection, semcache store errors, update download/checksum errors (`*_coverage_test.go` files)

**Security Review (mandatory):**
- Auto-updater: ✅ Checksum verification against `checksums.txt`; staged swap at `DATA_DIR/update/g0router.new`; admin-only; audited
- MITM CA: ✅ ECDSA P-256, 10y validity; `ca.key` file mode 0600; per-host leaf certs minted on demand; non-tool hosts tunneled through untouched
- Semantic cache: ✅ Embedding via existing provider connection; never blocks requests; flag-gated; expired entries not served

---

## Session S2 — Full release-readiness audit

```yaml
wave: S2
status: DONE
summary: "Adversarial full-project audit per FULL-PROJECT-REVIEW-HANDOFF.md. Verified all gates, fixed doc drift, tightened error redaction, coverage maintained at 95.0%."
work:
  - "Gate suite: go vet, gitleaks, go test ./..., go test -race ./..., coverage ≥95%, make verify, make e2e-binary, docker smoke"
  - "Verified fixes from prior audit: B1-B6 (Anthropic streaming tools, refresh race, streaming success timing, bedrock/replicate streaming, /v1/models resilience, error redaction)"
  - "Fixed PROVIDERS.md false advertising: bedrock + replicate streaming claims corrected"
  - "Fixed SCHEMA.md: added 7 missing routes (embeddings, images, audio, metrics, audit, traffic)"
  - "Fixed DEPLOYMENT.md: Dockerfile Go version aligned to go.mod (1.24)"
  - "Fixed DIRECTORY_STRUCTURE.md: removed phantom files (deploy/docker-compose.yml, 11 filter files)"
  - "Fixed ARCHITECTURE.md: removed phantom internal/cli/login.go reference"
  - "Tightened error redaction: apikeys.go DB errors → 500 static; usage.go quota unsupported → static message"
  - "Added store.ErrInvalidPolicy sentinel to distinguish validation (400) from DB (500) errors"
  - "Added coverage tests for new error-handling branches"
gates:
  - { command: "gitleaks detect", status: PASS, notes: "no leaks, 476 commits" }
  - { command: "go vet ./...", status: PASS }
  - { command: "go test ./... -count=1", status: PASS, notes: "~2658 tests; coverage 95.0%" }
  - { command: "go test -race ./...", status: PASS, notes: "41 packages, zero warnings" }
  - { command: "go test -tags e2ebin -run TestE2EBinary", status: PASS }
  - { command: "make verify", status: PASS, notes: "go+ui+playwright+git-diff green" }
  - { command: "docker build + smoke", status: PASS, notes: "/healthz 200 OK" }
```

---

## Session S1 — Dashboard wiring, traffic topology, coverage gate

```yaml
wave: S1
status: DONE
summary: "Lifted Go coverage to the 95% gate and wired every Round 1-5 backend
  feature into the dashboard, then added a live traffic topology view."
work:
  - "coverage -> 95.0%: multimodal handlers, metrics, cache, audit, server helpers, traffic broker"
  - "UI wave b: per-key policy form, notify+cache settings, fastest/cheapest combo strategies, audit page"
  - "UI wave c: provider health page, usage summary + daily time-series charts, strategy hints, one-click reauth"
  - "traffic topology: internal/traffic broker + SSE GET /api/traffic/stream + animated SVG graph"
  - "fix: API-key wire shape normalized snake_case->dashboard; keyed policy-row fragment"
gates:
  - { command: "gitleaks detect", status: PASS, notes: "no leaks, 473 commits" }
  - { command: "go vet ./...", status: PASS }
  - { command: "go test ./... -count=1", status: PASS, notes: "2701 tests; coverage 95.0%" }
  - { command: "go test -race ./api/ ./internal/traffic/", status: PASS }
  - { command: "go test -tags e2ebin -run TestE2EBinary", status: PASS }
  - { command: "npm --prefix ui test", status: PASS, notes: "147 tests" }
  - { command: "npm --prefix ui run build", status: PASS }
  - { command: "npm --prefix ui run e2e", status: PASS, notes: "33 passed, 1 skipped" }
  - { command: "docker build + run + /healthz (OrbStack)", status: PASS, notes: "healthz 200; API_KEY_SECRET required at runtime" }
  - { command: "git diff --check", status: PASS }
```

---

## Stage 9 — Principal Engineer Audit Remediation (Waves R1–R15)

A full line-by-line audit (security, backend/API, provider parity, routing/runtime,
MCP, dashboard, docs) plus an independent Kimi CLI review drove waves R1–R15.
All findings fixed under TDD; gates green at every commit.

- **R1** strip legacy upstream brand references from source + UI
- **R2** translate tool calls in Anthropic streaming egress (`/v1/messages`)
- **R3** map races (refreshers/quota/pool), streaming backoff timing, `/v1/models` resilience
- **R4** redact internal errors from client responses (~38 sites)
- **R5** stop leaking the pooled `*fasthttp.RequestCtx` (use-after-recycle data race, found via `-race`)
- **R6** strip brands from docs; fix false claims (23→43 providers, phantom files/pkgs, SCHEMA)
- **R7** clamp negative input cost; plumb cache-write tokens
- **R8** MCP: stdio deadlock, cancellation notifications, call-after-close refcount, session teardown
- **R9** preserve Anthropic tool ids on ingress; bound stream timeouts; honest capability tests
- **R10** validate SQL identifiers; expire MCP OAuth flows; cache settings
- **R11** UI: multi-step combos; origin-relative endpoint URLs
- **R12–R13** unit coverage 76% → **95.0%** (real-behavior tests, no mocks)
- **R14** container binds `0.0.0.0`; opt-in real-binary smoke (`make e2e-binary`)
- **R15** full Anthropic tool loop (translate tool definitions + tool_choice on ingress)

Gates: `go test ./...` 2098+ pass, `-race` clean, `go vet` clean, UI 100 tests,
Playwright e2e 23/24 (1 skipped), `make verify` green, gitleaks clean (420 commits),
real-binary + OrbStack container smoke verified. Coverage **95.0%**.

### Wave L — Full request-logging system

- Configurable retention via the Web UI (`log_retention_days`: 5/15/30/60/90/180,
  keep-forever, or custom) with an hourly background cleanup that prunes logs past
  the window. Default 30 days. Negative values rejected.
- `GET /api/logs` rich query: `provider`, `model`, `auth_type`, `source_format`,
  `status_class` (success/client_error/server_error), `search`, `start`/`end`
  (RFC3339), `limit`/`offset`; response carries `total` for pagination.
- Log viewer page: kind/provider/model/source-format/date filters, debounced
  search, pagination with totals, expandable per-row detail.
- Operational fields now populated: `client_tool` (X-Client-Tool / User-Agent),
  `rtk_bytes_saved` (RTK compression delta), `combo_name` (active-combo routing).
- Scope note: only inference request logs are persisted; "Kind" filters by HTTP
  status class. No separate MCP/access/system log streams (not in scope).

Final gates after Wave L: `go test ./... -race` **2181 pass**, `go vet` clean,
coverage **95.0%**, UI **104 tests**, Playwright e2e **27 pass**, `make verify`
green, `make e2e-binary` green, gitleaks clean (history), real-binary + OrbStack
container smoke verified, working tree clean.

---

## STAGE 8 — Completion Hardening

### Wave 8.L — API/Auth Integration Hardening

```yaml
wave: "8.L"
status: DONE
max_agents: 1
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build"
completed_at: "2026-06-04T07:19:56Z"
evaluator_prompt: "docs/evaluations/wave-8L-evaluator-prompt.md"

tasks:
  - id: "8.L.1"
    name: "Real-server management mutation integration coverage"
    status: DONE
    agent: "orchestrator"
    commit: "7022836"
    files_owned:
      - api/server_integration_test.go

  - id: "8.L.2"
    name: "Real-server MCP instance OAuth integration coverage"
    status: DONE
    agent: "orchestrator"
    commit: "7633953"
    files_owned:
      - api/server_integration_test.go

  - id: "8.L.3"
    name: "CLI API-key login persistence"
    status: DONE
    agent: "orchestrator"
    commit: "009117f"
    files_owned:
      - internal/cli/auth.go
      - internal/cli/auth_test.go
```

**Checkpoint**: Wave 8.L is complete and retained as historical gate evidence.

### Wave 8.M — Optional Live Provider Smoke Gate

```yaml
wave: "8.M"
status: DONE
max_agents: 1
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build"
completed_at: "2026-06-04T07:25:36Z"
evaluator_prompt: "docs/evaluations/wave-8M-evaluator-prompt.md"

tasks:
  - id: "8.M.1"
    name: "Opt-in MiniMax live provider smoke test"
    status: DONE
    agent: "orchestrator"
    commit: "f83addd"
    files_owned:
      - internal/providers/openaicompat/live_minimax_test.go
      - docs/CONFIG.md
```

**Checkpoint**: Live-provider checks are optional and skipped by default; release gates remain deterministic without external network credentials.

### Wave 8.N — Principal Audit Remediation

```yaml
wave: "8.N"
status: DONE
max_agents: 8
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build"
completed_at: "2026-06-04T08:35:02Z"
evaluator_prompt: "docs/evaluations/wave-8N-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e91bf-e9d8-7ca1-a054-b4b674de04ad at commit 66818e8"
gate_results:
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS"
  - "make build: PASS"

tasks:
  - id: "8.N.1"
    name: "Dashboard provider connection management"
    status: DONE
    agent: "orchestrator"
    commit: "09d68ac"
    files_owned:
      - ui/src/api.ts
      - ui/src/pages/ProvidersPage.tsx
      - ui/src/pages/ProvidersPage.test.tsx
      - ui/e2e/dashboard.e2e.ts
      - ui/dist

  - id: "8.N.2"
    name: "Provider matrix quota truth"
    status: DONE
    agent: "orchestrator"
    commit: "f83ca6d"
    files_owned:
      - internal/provider/matrix.go
      - internal/provider/matrix_test.go
      - api/handlers/providers_test.go
      - docs/PROVIDERS.md
      - ui/e2e/dashboard.e2e.ts

  - id: "8.N.3"
    name: "OpenAI-compatible base URL normalization"
    status: DONE
    agent: "orchestrator"
    commit: "9d98320"
    files_owned:
      - internal/providers/openaicompat/provider.go
      - internal/providers/openaicompat/provider_test.go

  - id: "8.N.4"
    name: "OAuth exchange failure sanitization"
    status: DONE
    agent: "orchestrator"
    commit: "d13892d"
    files_owned:
      - api/handlers/oauth.go
      - api/handlers/oauth_test.go

  - id: "8.N.5"
    name: "Docker Compose auth configuration alignment"
    status: DONE
    agent: "orchestrator"
    commit: "743e581"
    files_owned:
      - docker-compose.yml
      - .env.example

  - id: "8.N.6"
    name: "Dashboard MCP OAuth, tools, and deletion actions"
    status: DONE
    agent: "orchestrator"
    commit: "a005601"
    files_owned:
      - ui/src/api.ts
      - ui/src/pages/McpPage.tsx
      - ui/src/pages/McpPage.test.tsx
      - ui/e2e/dashboard.e2e.ts
      - ui/dist

  - id: "8.N.7"
    name: "Anthropic live upstream streaming"
    status: DONE
    agent: "orchestrator"
    commit: "8ce739f"
    files_owned:
      - internal/providers/anthropic/anthropic.go
      - internal/providers/anthropic/anthropic_test.go

  - id: "8.N.8"
    name: "Unsupported native streaming classification"
    status: DONE
    agent: "orchestrator"
    commit: "f8c3910"
    files_owned:
      - internal/proxy/errors.go
      - api/handlers/inference_test.go

  - id: "8.N.9"
    name: "Quota API uses active stored provider credentials"
    status: DONE
    agent: "orchestrator"
    commit: "e674de4"
    files_owned:
      - api/handlers/usage.go
      - api/handlers/usage_test.go
      - api/server.go

  - id: "8.N.10"
    name: "Malformed SSE errors are surfaced and sanitized"
    status: DONE
    agent: "orchestrator"
    commit: "f98638b"
    files_owned:
      - api/handlers/inference.go
      - api/handlers/inference_test.go
      - internal/providers/types.go
      - internal/providers/openai/openai.go
      - internal/providers/openai/openai_test.go
      - internal/providers/azure/azure.go
      - internal/providers/azure/azure_test.go
      - internal/providers/openaicompat/provider.go
      - internal/providers/openaicompat/provider_test.go

  - id: "8.N.11"
    name: "Provider test command and provider model API truth"
    status: DONE
    agent: "orchestrator"
    commit: "e34491d"
    files_owned:
      - internal/cli/root.go
      - internal/cli/providers_test.go
      - api/handlers/providers.go
      - api/handlers/providers_test.go

  - id: "8.N.12"
    name: "Anthropic stream error events are surfaced and sanitized"
    status: DONE
    agent: "orchestrator"
    commit: "b2f6fe2"
    files_owned:
      - internal/providers/anthropic/anthropic.go
      - internal/providers/anthropic/anthropic_test.go
      - internal/providers/anthropic/types.go

  - id: "8.N.13"
    name: "MCP OAuth completion errors are sanitized"
    status: DONE
    agent: "orchestrator"
    commit: "36c2463"
    files_owned:
      - api/handlers/mcpoauth.go
      - api/handlers/mcpoauth_test.go
```

**Checkpoint**: Wave 8.N deterministic gates passed from `main` at `2026-06-04T08:35:02Z` after MCP OAuth sanitization commit `36c2463`; external evaluator thread `019e91bf-e9d8-7ca1-a054-b4b674de04ad` returned PASS at commit `66818e8` with no blocking findings.

### Wave 8.O — Gateway Provider Parity

```yaml
wave: "8.O"
status: DONE
max_agents: 4
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build"
completed_at: "2026-06-04T08:51:20Z"
evaluator_prompt: "docs/evaluations/wave-8O-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e91d6-1def-7be0-8dc6-67b537725536 at commit 099e3f3"
gate_results:
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS"
  - "make build: PASS"

tasks:
  - id: "8.O.1"
    name: "OpenAI-compatible gateway adapter coverage"
    status: DONE
    agent: "orchestrator"
    commit: "d14b736"
    files_owned:
      - internal/providers/types.go
      - internal/providers/openaicompat/registry.go
      - internal/providers/openaicompat/provider_test.go
      - internal/cli/provider_runtime.go
      - internal/cli/providers_test.go
      - internal/cli/root_test.go
      - internal/provider/matrix.go
      - internal/provider/matrix_test.go
      - internal/modelcatalog/catalog.go
      - internal/modelcatalog/pricing_test.go
      - api/handlers/providers_test.go
      - docs/PROVIDERS.md
      - docs/evaluations/wave-8O-evaluator-prompt.md
```

**Checkpoint**: Wave 8.O adds real OpenAI-compatible adapter coverage for Vercel AI Gateway, LiteLLM, vLLM, and LM Studio without advertising instance-defined local gateway providers as public direct-dispatch surfaces; external evaluator thread `019e91d6-1def-7be0-8dc6-67b537725536` returned PASS at commit `099e3f3` with no blocking findings.

### Wave 8.P — NVIDIA Direct Routing

```yaml
wave: "8.P"
status: DONE
max_agents: 2
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build"
completed_at: "2026-06-04T08:59:35Z"
evaluator_prompt: "docs/evaluations/wave-8P-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e91dc-656d-7952-b293-2292fda400cb at commit c996f13"
gate_results:
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS"
  - "make build: PASS"

tasks:
  - id: "8.P.1"
    name: "NVIDIA catalog-backed public routing"
    status: DONE
    agent: "orchestrator"
    commit: "d079d50"
    files_owned:
      - internal/provider/matrix.go
      - internal/provider/matrix_test.go
      - internal/modelcatalog/catalog.go
      - internal/modelcatalog/pricing_test.go
      - api/handlers/providers_test.go
      - internal/cli/providers_test.go
      - internal/cli/root_test.go
      - docs/PROVIDERS.md
      - docs/evaluations/wave-8P-evaluator-prompt.md
```

**Checkpoint**: Wave 8.P promotes the already registered NVIDIA OpenAI-compatible adapter to public direct dispatch via a catalog-backed `meta/llama-3.1-8b-instruct` route; external evaluator thread `019e91dc-656d-7952-b293-2292fda400cb` returned PASS at commit `c996f13` with no blocking findings.

### Wave 8.Q — Vertex Direct Routing

```yaml
wave: "8.Q"
status: DONE
max_agents: 2
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build"
completed_at: "2026-06-04T09:06:30Z"
evaluator_prompt: "docs/evaluations/wave-8Q-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e91e3-4405-7a60-a649-e10c70492a79 at commit f36c0da"
gate_results:
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS"
  - "make build: PASS"

tasks:
  - id: "8.Q.1"
    name: "Vertex catalog-backed public routing"
    status: DONE
    agent: "orchestrator"
    commit: "1891a0c"
    files_owned:
      - .env.example
      - docs/CONFIG.md
      - docs/PROVIDERS.md
      - internal/cli/provider_runtime.go
      - internal/cli/providers_test.go
      - internal/cli/root_test.go
      - internal/provider/matrix.go
      - internal/provider/matrix_test.go
      - internal/providers/vertex/vertex.go
      - internal/providers/vertex/vertex_test.go
      - api/handlers/providers_test.go
      - docs/evaluations/wave-8Q-evaluator-prompt.md
```

**Checkpoint**: Wave 8.Q promotes the native Vertex adapter to public direct dispatch for cataloged Gemini models when `VERTEX_PROJECT_ID` and `VERTEX_LOCATION` are configured; streaming and quota remain explicitly unsupported, and external evaluator thread `019e91e3-4405-7a60-a649-e10c70492a79` returned PASS at commit `f36c0da` with no blocking findings.

### Wave 8.R — Provider-Qualified Vertex Routing

```yaml
wave: "8.R"
status: DONE
max_agents: 2
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build"
completed_at: "2026-06-04T09:23:41Z"
evaluator_prompt: "docs/evaluations/wave-8R-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e91f5-46ee-7bd2-bb34-2589de0e3107 at commit e729177"
gate_results:
  - "go test ./api ./internal/modelcatalog ./internal/proxy -count=1: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS"
  - "make build: PASS"

tasks:
  - id: "8.R.1"
    name: "Provider-qualified Vertex catalog routing"
    status: DONE
    agent: "orchestrator"
    commit: "22c24f2"
    files_owned:
      - api/server.go
      - api/server_test.go
      - docs/CONFIG.md
      - docs/PROVIDERS.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8R-evaluator-prompt.md
      - internal/modelcatalog/catalog.go
      - internal/modelcatalog/pricing_test.go
      - internal/proxy/engine.go
      - internal/proxy/engine_test.go
```

**Checkpoint**: Wave 8.R fixes the post-8.Q routing gap where unqualified Gemini catalog IDs made Vertex direct dispatch unreachable. Vertex public routes now use provider-qualified `vertex/gemini-*` IDs, dispatch rewrites them to upstream Gemini model IDs, and request logging preserves the public model for cost lookup. External evaluator thread `019e91f5-46ee-7bd2-bb34-2589de0e3107` returned PASS at commit `e729177` with no blocking findings.

### Wave 8.S — Vertex OAuth Binding

```yaml
wave: "8.S"
status: DONE
max_agents: 2
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build"
completed_at: "2026-06-04T09:31:33Z"
evaluator_prompt: "docs/evaluations/wave-8S-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e91fb-fb7a-7af3-b6f1-d0ae7eeef3d5 at commit ac08662"
gate_results:
  - "go test ./internal/provider/oauth ./internal/provider ./internal/cli ./api/handlers -run 'TestCanonical|TestOAuthFlowAccepts|TestLoginDevicePersistsVertex|TestOAuthExchangeStoresVertex|TestOAuthStartStores|TestOAuthCallbackUses|TestOAuthPoll' -count=1: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS"
  - "make build: PASS"

tasks:
  - id: "8.S.1"
    name: "Bind Vertex auth to Gemini OAuth flow"
    status: DONE
    agent: "orchestrator"
    commit: "4fc4f27"
    files_owned:
      - api/handlers/oauth.go
      - api/handlers/oauth_test.go
      - docs/PROVIDERS.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8S-evaluator-prompt.md
      - internal/cli/auth.go
      - internal/cli/auth_test.go
      - internal/provider/credentials.go
      - internal/provider/oauth/types.go
      - internal/provider/oauth/types_test.go
```

**Checkpoint**: Wave 8.S fixes the auth/runtime split for Vertex. `vertex` now resolves to the Gemini OAuth flow while persisted CLI and HTTP OAuth connections keep runtime provider `vertex` with `oauth_provider=gemini`, so provider-qualified Vertex dispatch can find usable credentials. External evaluator thread `019e91fb-fb7a-7af3-b6f1-d0ae7eeef3d5` returned PASS at commit `ac08662` with no source blockers.

### Wave 8.T — RTK And Caveman Dispatch Wiring

```yaml
wave: "8.T"
status: DONE
max_agents: 2
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build"
completed_at: "2026-06-04T09:39:34Z"
evaluator_prompt: "docs/evaluations/wave-8T-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e9204-c4d4-74a0-a5e4-3a62c22a5533 at commit c985c16"
gate_results:
  - "go test ./api -run TestInferenceAppliesRTKAndCavemanSettingsBeforeDispatch -count=1: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS"
  - "make build: PASS"

tasks:
  - id: "8.T.1"
    name: "Apply RTK and caveman settings before dispatch"
    status: DONE
    agent: "orchestrator"
    commit: "5baa4c6"
    files_owned:
      - api/server.go
      - api/server_test.go
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8T-evaluator-prompt.md
```

**Checkpoint**: Wave 8.T wires runtime settings into normal `/v1/*` dispatch. Requests pass through RTK compression and caveman prompt injection before the inference engine, and request logs now record source/target format plus RTK/caveman enabled flags. External evaluator thread `019e9204-c4d4-74a0-a5e4-3a62c22a5533` returned PASS at commit `c985c16`; its non-blocking note about coarse source-format values was addressed in Wave 8.U.

### Wave 8.U — MCP Tool Injection And Route Format Logging

```yaml
wave: "8.U"
status: DONE
max_agents: 2
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build"
completed_at: "2026-06-04T09:47:03Z"
evaluator_prompt: "docs/evaluations/wave-8U-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e920a-6b79-75b2-ada9-615a3ef5060a at commit 3a0ed83"
gate_results:
  - "go test ./api -run 'TestInferenceLoggingRecordsMessagesRouteWhenEnabled|TestInferenceLoggingRecordsResponsesRouteWhenEnabled|TestInferenceAddsRegisteredMCPToolsBeforeDispatch|TestInferenceAppliesRTKAndCavemanSettingsBeforeDispatch' -count=1: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS"
  - "make build: PASS"

tasks:
  - id: "8.U.1"
    name: "Attach registered MCP tools and log route source formats"
    status: DONE
    agent: "orchestrator"
    commit: "694caae"
    files_owned:
      - api/server.go
      - api/server_test.go
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8U-evaluator-prompt.md
```

**Checkpoint**: Wave 8.U makes registered MCP tools visible to normal `/v1/chat/completions`, `/v1/messages`, and `/v1/responses` inference requests when callers do not provide their own tools, while preserving caller-supplied tools. Request logs now record `openai`, `anthropic`, or `responses` as the source format according to the public route. External evaluator thread `019e920a-6b79-75b2-ada9-615a3ef5060a` returned PASS at commit `3a0ed83` with no blocking findings.

### Wave 8.V — MCP Agent Execution In Gateway Dispatch

```yaml
wave: "8.V"
status: DONE
max_agents: 2
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build"
completed_at: "2026-06-04T09:54:33Z"
evaluator_prompt: "docs/evaluations/wave-8V-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e9210-dc8f-75d0-9e09-f0362b35127d at commit 7d6909c"
gate_results:
  - "go test ./internal/proxy -run TestDispatchRunsMCPAgentToolLoopWhenToolsConfigured -count=1: PASS"
  - "go test ./internal/mcp -run TestAgent -count=1: PASS"
  - "go test ./internal/cli -run TestDefaultServerConfigWiresWave7BRuntime -count=1: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS"
  - "make build: PASS"

tasks:
  - id: "8.V.1"
    name: "Run MCP agent loop from proxy dispatch"
    status: DONE
    agent: "orchestrator"
    commit: "902b91f"
    files_owned:
      - internal/proxy/engine.go
      - internal/proxy/engine_test.go
      - internal/mcp/agent.go
      - internal/cli/root.go
      - internal/cli/root_test.go
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8V-evaluator-prompt.md
```

**Checkpoint**: Wave 8.V wires the existing MCP agent loop into non-streaming proxy dispatch when a registered MCP tool manager is present, executes tool calls through `mcp.ToolManager`, feeds tool results back to the provider, and gives normal `g0router serve` startup the same runtime tool manager as the API control plane. External evaluator thread `019e9210-dc8f-75d0-9e09-f0362b35127d` returned PASS at commit `7d6909c` with no blocking findings.

### Wave 8.W — Dashboard Models Page

```yaml
wave: "8.W"
status: DONE
max_agents: 2
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build"
completed_at: "2026-06-04T10:01:03Z"
evaluator_prompt: "docs/evaluations/wave-8W-evaluator-prompt.md"
evaluation: "FAIL external evaluator thread 019e9216-7b90-79a1-95fd-6d849442edd1; missing Models page loading and non-auth error test coverage, remediated in Wave 8.Z"
gate_results:
  - "npm --prefix ui test -- --run ModelsPage App: PASS"
  - "npm --prefix ui run e2e -- dashboard.e2e.ts: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS"
  - "make build: PASS"

tasks:
  - id: "8.W.1"
    name: "Add dashboard provider models page"
    status: DONE
    agent: "orchestrator"
    commit: "69e3d41"
    files_owned:
      - ui/src/App.tsx
      - ui/src/App.test.tsx
      - ui/src/pages/ModelsPage.tsx
      - ui/src/pages/ModelsPage.test.tsx
      - ui/e2e/dashboard.e2e.ts
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8W-evaluator-prompt.md
```

**Checkpoint**: Wave 8.W adds a real Models dashboard route backed by `/api/providers` and `/api/providers/{provider}/models`, with unit and E2E coverage for loading, provider switching, empty state, and auth-expired state.

### Wave 8.X — Documentation Status Reconciliation

```yaml
wave: "8.X"
status: DONE
max_agents: 1
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build"
completed_at: "2026-06-04T10:03:52Z"
evaluator_prompt: "docs/evaluations/wave-8X-evaluator-prompt.md"
evaluation: "FAIL external evaluator thread 019e9219-8aaa-7b20-a827-9863248eabfc; docs/PLAN.md and docs/ORCHESTRATION.md still described Stage 8 as Waves 8.A-8.N, remediated in Wave 8.Z"
gate_results:
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS"
  - "make build: PASS"

tasks:
  - id: "8.X.1"
    name: "Refresh README Stage 8 status"
    status: DONE
    agent: "orchestrator"
    commit: "2b486b3"
    files_owned:
      - docs/README.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8X-evaluator-prompt.md
```

**Checkpoint**: Wave 8.X updates the documentation landing page so it no longer implies Stage 8 is merely future or secondary to Stage 7; `docs/WORKFLOW.md` remains the source of truth for active remediation status.

### Wave 8.Y — Dashboard API Keys Page

```yaml
wave: "8.Y"
status: DONE
max_agents: 2
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build"
completed_at: "2026-06-04T10:11:28Z"
evaluator_prompt: "docs/evaluations/wave-8Y-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e9223-bb90-78d1-bc2e-711e556feff7 at commit c938513"
gate_results:
  - "npm --prefix ui test -- --run APIKeysPage EndpointPage App: PASS"
  - "npm --prefix ui run e2e -- dashboard.e2e.ts: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS"
  - "make build: PASS"

tasks:
  - id: "8.Y.1"
    name: "Add dedicated dashboard API keys page"
    status: DONE
    agent: "orchestrator"
    commit: "85c95d5"
    files_owned:
      - ui/src/App.tsx
      - ui/src/App.test.tsx
      - ui/src/pages/APIKeysPage.tsx
      - ui/src/pages/APIKeysPage.test.tsx
      - ui/src/pages/EndpointPage.tsx
      - ui/e2e/dashboard.e2e.ts
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8Y-evaluator-prompt.md
```

**Checkpoint**: Wave 8.Y separates API key management into its own dashboard route backed by the existing `/api/keys` contract while leaving endpoint-copy controls only on Endpoint Setup.

### Wave 8.Z — Evaluator Remediation For 8.W And 8.X

```yaml
wave: "8.Z"
status: DONE
max_agents: 2
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build"
completed_at: "2026-06-04T10:16:00Z"
evaluator_prompt: "docs/evaluations/wave-8Z-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e9223-edad-7972-ba44-a4cdf97da9e6 at commit c938513"
gate_results:
  - "npm --prefix ui test -- --run ModelsPage: PASS"
  - "rg -n \"8\\.A.*8\\.N|14 \\| 8|40 waves|8\\.Y\" docs/PLAN.md docs/ORCHESTRATION.md docs/WORKFLOW.md: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS"
  - "make build: PASS"

tasks:
  - id: "8.Z.1"
    name: "Remediate Wave 8.W and 8.X evaluator findings"
    status: DONE
    agent: "orchestrator"
    commit: "db4eda5"
    files_owned:
      - ui/src/pages/ModelsPage.test.tsx
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8Z-evaluator-prompt.md
```

**Checkpoint**: Wave 8.Z fixes evaluator-found gaps from Wave 8.W and 8.X by adding focused Models page loading/empty/error coverage and aligning Stage 8 planning/orchestration docs to the current Wave 8.Y scope.

### Wave 8.AA — Dashboard Connections/Auth Page

```yaml
wave: "8.AA"
status: DONE
max_agents: 2
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build"
completed_at: "2026-06-04T10:21:41Z"
evaluator_prompt: "docs/evaluations/wave-8AA-evaluator-prompt.md"
evaluation: "FAIL external evaluator thread 019e9229-d19f-7e50-9339-a72aa72658b2; Playwright navigation matched MCP split labels ambiguously, remediated in Wave 8.AB"
gate_results:
  - "npm --prefix ui test -- --run ConnectionsAuthPage ProvidersPage App: PASS"
  - "npm --prefix ui run e2e -- dashboard.e2e.ts: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS"
  - "make build: PASS"

tasks:
  - id: "8.AA.1"
    name: "Add dedicated dashboard Connections/Auth page"
    status: DONE
    agent: "orchestrator"
    commit: "7f633a5"
    files_owned:
      - ui/src/App.tsx
      - ui/src/App.test.tsx
      - ui/src/pages/ConnectionsAuthPage.tsx
      - ui/src/pages/ConnectionsAuthPage.test.tsx
      - ui/src/pages/ProvidersPage.tsx
      - ui/e2e/dashboard.e2e.ts
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8AA-evaluator-prompt.md
```

**Checkpoint**: Wave 8.AA separates provider account and auth management into a dedicated Connections/Auth dashboard route while keeping provider matrix contract details on the Providers route.

### Wave 8.AB — Dashboard MCP Split Pages

```yaml
wave: "8.AB"
status: DONE
max_agents: 2
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build"
completed_at: "2026-06-04T10:30:05Z"
evaluator_prompt: "docs/evaluations/wave-8AB-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e9230-f77d-7f63-8d6f-c063799f9e39 at commit 93de90e"
gate_results:
  - "npm --prefix ui test -- --run McpSplitPages McpPage App: PASS"
  - "npm --prefix ui run e2e -- dashboard.e2e.ts: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS"
  - "make build: PASS"

tasks:
  - id: "8.AB.1"
    name: "Add dedicated dashboard MCP instance, account, and tool pages"
    status: DONE
    agent: "orchestrator"
    commit: "a68f6c7"
    files_owned:
      - ui/src/App.tsx
      - ui/src/App.test.tsx
      - ui/src/pages/McpPage.tsx
      - ui/src/pages/McpInstancesPage.tsx
      - ui/src/pages/McpAccountsPage.tsx
      - ui/src/pages/McpToolsPage.tsx
      - ui/src/pages/McpSplitPages.test.tsx
      - ui/e2e/dashboard.e2e.ts
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8AB-evaluator-prompt.md
```

**Checkpoint**: Wave 8.AB splits the dashboard MCP surface into dedicated Instances, Accounts, and Tools routes while preserving the combined MCP route and credential redaction.

### Wave 8.AC — Dashboard Settings/Security Page

```yaml
wave: "8.AC"
status: DONE
max_agents: 1
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build"
completed_at: "2026-06-04T10:35:19Z"
evaluator_prompt: "docs/evaluations/wave-8AC-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e9236-2327-7263-b659-909e4ac0abf9 at commit 8254a07"
gate_results:
  - "npm --prefix ui test -- --run SettingsSecurityPage SettingsPage App: PASS"
  - "npm --prefix ui run e2e -- dashboard.e2e.ts: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS"
  - "make build: PASS"

tasks:
  - id: "8.AC.1"
    name: "Add dedicated dashboard Settings/Security page"
    status: DONE
    agent: "orchestrator"
    commit: "f707219"
    files_owned:
      - ui/src/App.tsx
      - ui/src/App.test.tsx
      - ui/src/pages/SettingsPage.tsx
      - ui/src/pages/SettingsSecurityPage.tsx
      - ui/src/pages/SettingsSecurityPage.test.tsx
      - ui/e2e/dashboard.e2e.ts
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8AC-evaluator-prompt.md
```

**Checkpoint**: Wave 8.AC adds a dedicated Settings/Security dashboard route backed by the real settings API and covers control-plane protection plus request logging controls.

### Wave 8.AD — Dashboard Route Name Reconciliation

```yaml
wave: "8.AD"
status: DONE
max_agents: 1
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build"
completed_at: "2026-06-04T10:43:03Z"
evaluator_prompt: "docs/evaluations/wave-8AD-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e923e-23b6-7b90-a1ba-b41d1a856f42 at commit ed9dca9"
gate_results:
  - "npm --prefix ui test -- --run App: FAIL before implementation, missing Endpoint Setup label"
  - "npm --prefix ui test -- --run App: PASS"
  - "npm --prefix ui run e2e -- dashboard.e2e.ts: PASS"
  - "go test ./... -count=1: PASS after one transient api/TestRequestIDUnique rerun"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS"
  - "make build: PASS"

tasks:
  - id: "8.AD.1"
    name: "Align dashboard route names with documented management pages"
    status: DONE
    agent: "orchestrator"
    commit: "e806fe8"
    files_owned:
      - ui/src/App.tsx
      - ui/src/App.test.tsx
      - ui/e2e/dashboard.e2e.ts
      - ui/dist/assets/index.js
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8AD-evaluator-prompt.md
```

**Checkpoint**: Wave 8.AD aligns the dashboard navigation with the documented `Endpoint Setup` and `Combos/Routing` management page names and proves those exact labels in unit and Playwright E2E coverage.

### Wave 8.AE — MCP OAuth Metadata Discovery

```yaml
wave: "8.AE"
status: DONE
max_agents: 1
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build"
completed_at: "2026-06-04T10:51:03Z"
evaluator_prompt: "docs/evaluations/wave-8AE-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e9242-efbd-7211-bb7e-b32ea56ea2ef at commit 32c2131"
gate_results:
  - "go test ./internal/mcp -run 'TestOAuthEngine(DiscoversTokenEndpointFromAuthorizationServerMetadata|RequiresRealTokenEndpoint)' -count=1: FAIL before implementation, metadata token endpoint unavailable"
  - "go test ./internal/mcp -run 'TestOAuthEngine(DiscoversTokenEndpointFromAuthorizationServerMetadata|RequiresRealTokenEndpoint|CompletesCallbackForMatchingInstance|RejectsRedirectingTokenEndpointWithoutFollowing)' -count=1: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS"
  - "make build: PASS"

tasks:
  - id: "8.AE.1"
    name: "Discover MCP OAuth token endpoint from authorization server metadata"
    status: DONE
    agent: "orchestrator"
    commit: "60a0e41"
    files_owned:
      - internal/mcp/oauth.go
      - internal/mcp/oauth_test.go
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8AE-evaluator-prompt.md
```

**Checkpoint**: Wave 8.AE replaces MCP OAuth token endpoint fabrication for non-`/authorize` flows with read-only authorization-server metadata discovery, while preserving the existing `/authorize` convention and no-redirect token behavior.

### Wave 8.AF — Public Route Integration Coverage

```yaml
wave: "8.AF"
status: DONE
max_agents: 1
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build"
completed_at: "2026-06-04T10:59:19Z"
evaluator_prompt: "docs/evaluations/wave-8AF-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e924a-8c74-7373-b0a3-d22ee5fa7428 at commit d26d69f"
gate_results:
  - "go test ./api -run TestIntegrationAuthenticatedAPIServerWithFakeUpstream -count=1: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS"
  - "make build: PASS"

tasks:
  - id: "8.AF.1"
    name: "Cover /v1/messages and /v1/responses in real-server integration"
    status: DONE
    agent: "orchestrator"
    commit: "32c2131"
    files_owned:
      - api/server_integration_test.go
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8AF-evaluator-prompt.md
```

**Checkpoint**: Wave 8.AF extends the authenticated real-server integration test to prove `/v1/messages` and `/v1/responses` dispatch through the same configured gateway and fake OpenAI-compatible upstream as `/v1/chat/completions`, including response-shape and usage mapping assertions.

### Wave 8.AG — MCP OAuth Protected Resource Discovery

```yaml
wave: "8.AG"
status: DONE
max_agents: 1
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build"
completed_at: "2026-06-04T11:08:31Z"
evaluator_prompt: "docs/evaluations/wave-8AG-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e9253-0223-75a0-aac8-fed3f2ab8cf9 at commit dccbf70"
gate_results:
  - "go test ./internal/mcp ./api/handlers ./internal/cli -run 'TestDiscoverOAuthAuthorizationURLFromProtectedResourceMetadata|TestMCPOAuthStartDiscoversAuthorizationURLFromResourceMetadata|TestMCPOAuthStartCommandDiscoversAuthorizationURL' -count=1: FAIL before implementation for missing discovery/helper wiring, then PASS"
  - "go test ./internal/mcp ./api/handlers ./internal/cli -count=1: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS"
  - "make build: PASS"

tasks:
  - id: "8.AG.1"
    name: "Discover MCP OAuth authorization endpoint from protected resource metadata"
    status: DONE
    agent: "orchestrator"
    commit: "168feb5"
    files_owned:
      - internal/mcp/oauth.go
      - internal/mcp/oauth_test.go
      - api/handlers/mcp.go
      - api/handlers/mcp_test.go
      - internal/cli/mcp_auth.go
      - internal/cli/mcp_auth_test.go
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8AG-evaluator-prompt.md
```

**Checkpoint**: Wave 8.AG lets HTTP API and CLI MCP OAuth start flows omit `authorization_url` when `resource_uri` exposes OAuth protected-resource metadata. Discovery follows the resource `WWW-Authenticate` `resource_metadata` URL, reads `authorization_servers`, reads authorization-server metadata, then stores the normal PKCE flow without test-only handlers or external network.

### Wave 8.AH — Connection Mutation Integration Coverage

```yaml
wave: "8.AH"
status: DONE
max_agents: 1
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build"
completed_at: "2026-06-04T11:20:39Z"
evaluator_prompt: "docs/evaluations/wave-8AH-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e9267-f213-7f33-b494-bf3555a25133 at commit b0ee89e after remediation commit b0ee89e"
gate_results:
  - "go test ./api -run TestIntegrationManagementMutationsRoundTripThroughAuthenticatedServer -count=1: PASS after adding missing local test helper"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS"
  - "make build: PASS"

tasks:
  - id: "8.AH.1"
    name: "Cover connection mutation redaction through authenticated real server"
    status: DONE
    agent: "orchestrator"
    commit: "1623081"
    files_owned:
      - api/server_integration_test.go
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8AH-evaluator-prompt.md
```

**Checkpoint**: Wave 8.AH extends the authenticated real-server management integration suite to create, test, list, update, and delete provider connections through `/api/connections`, while asserting access tokens, refresh tokens, API keys, and nested provider secrets are persisted but never serialized back to management API responses.

### Wave 8.AI — No-Auth Provider Runtime Dispatch

```yaml
wave: "8.AI"
status: DONE
max_agents: 1
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build"
completed_at: "2026-06-04T11:29:41Z"
evaluator_prompt: "docs/evaluations/wave-8AI-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e926d-1a74-79d0-94ab-0cd5ce3c35fd at commit 6ce428c"
gate_results:
  - "go test ./internal/proxy -run 'TestDispatchUsesNoAuthProviderWithoutStoredConnection|TestDispatchStreamUsesNoAuthProviderWithoutStoredConnection|TestDispatchUsesCatalogForOllamaNoAuthProvider' -count=1: FAIL before implementation with Dispatch/DispatchStream reporting no active connections"
  - "go test ./internal/proxy -run 'TestDispatchUsesNoAuthProviderWithoutStoredConnection|TestDispatchStreamUsesNoAuthProviderWithoutStoredConnection|TestDispatchUsesCatalogForOllamaNoAuthProvider' -count=1: PASS"
  - "go test ./internal/proxy -count=1: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS"
  - "make build: PASS"

tasks:
  - id: "8.AI.1"
    name: "Dispatch no-auth providers without stored connections"
    status: DONE
    agent: "orchestrator"
    commit: "5dab3ec"
    files_owned:
      - internal/proxy/engine.go
      - internal/proxy/engine_test.go
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8AI-evaluator-prompt.md
```

**Checkpoint**: Wave 8.AI lets catalog-supported no-auth providers such as Ollama dispatch direct and streaming requests without requiring a persisted provider connection, while preserving stored no-auth connection behavior and continuing to reject providers that require credentials when no active connection exists.

### Wave 8.AJ — MCP OAuth Selected Account Label Binding

```yaml
wave: "8.AJ"
status: DONE
max_agents: 1
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router"
completed_at: "2026-06-04T11:50:04Z"
evaluator_prompt: "docs/evaluations/wave-8AJ-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e9279-a261-7470-9e41-0c7fba48cd0b at commit 336aba9"
gate_results:
  - "go test ./internal/mcp -run 'TestOAuthEnginePrefersSelectedInstanceAccountLabelOverTokenAccountLabel|TestOAuthEngineUsesSelectedInstanceAccountLabel' -count=1: FAIL before implementation with account label token-work, want selected-work"
  - "go test ./internal/mcp -run 'TestOAuthEnginePrefersSelectedInstanceAccountLabelOverTokenAccountLabel|TestOAuthEngineUsesSelectedInstanceAccountLabel' -count=1: PASS"
  - "go test ./internal/mcp -count=1: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"

tasks:
  - id: "8.AJ.1"
    name: "Bind MCP OAuth completion to selected instance account label"
    status: DONE
    agent: "019e9272-f0e3-76b3-ae4c-5210c3978311"
    commit: "3b8aa81"
    files_owned:
      - internal/mcp/oauth.go
      - internal/mcp/oauth_test.go
      - api/server_integration_test.go
```

**Checkpoint**: Wave 8.AJ makes an MCP instance-selected account label authoritative during OAuth completion, so token endpoint `account_label` values cannot orphan persisted accounts from runtime account selection. Existing token-derived fallback labels still apply when no selected instance label exists.

### Wave 8.AK — MCP OAuth Client Credentials

```yaml
wave: "8.AK"
status: DONE
max_agents: 1
gate: "go test ./internal/mcp -run 'TestOAuthStartIncludesClientID|TestOAuthEnginePostsClientCredentialsWhenFlowProvidesThem' -count=1 && go test ./internal/store -run TestMCPOAuthFlow -count=1 && go test ./api/handlers -run TestMCPOAuthStart -count=1 && go test ./internal/cli -run TestMCPOAuthStartCommand -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router"
completed_at: "2026-06-04T12:19:00Z"
evaluator_prompt: "docs/evaluations/wave-8AK-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e9284-6fe9-79c1-a73f-88a2a2f2336d after remediation commits 3a4a38b and 0fa8a28; initial migration and API suite gate failures resolved"
gate_results:
  - "go test ./internal/mcp -run 'TestOAuthStartIncludesClientID|TestOAuthEnginePostsClientCredentialsWhenFlowProvidesThem' -count=1: PASS"
  - "go test ./internal/store -run TestMCPOAuthFlow -count=1: PASS"
  - "go test ./api/handlers -run TestMCPOAuthStart -count=1: PASS"
  - "go test ./internal/cli -run TestMCPOAuthStartCommand -count=1: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "go test ./api -count=20 after API stabilization: PASS"
  - "go test ./api -run 'TestInferenceAddsRegisteredMCPToolsBeforeDispatch|TestManagementRoutesRequireAPIKey' -count=50 after API stabilization: PASS"

tasks:
  - id: "8.AK.1"
    name: "MCP OAuth user-provided client credentials"
    status: DONE
    agent: "subagent 019e927a-aba7-7d22-8bc6-a0a342e32172"
    commit: "e57c6d6"
    merge_commit: "e751673"
    remediation_commits:
      - "3a4a38b"
      - "0fa8a28"
    remediation_merge_commit: "4d9b03e"
    files_owned:
      - internal/mcp/oauth.go
      - internal/mcp/oauth_test.go
      - internal/store/mcpoauth.go
      - internal/store/mcpoauth_test.go
      - internal/store/sqlite.go
      - api/handlers/mcp.go
      - api/handlers/mcp_test.go
      - internal/cli/mcp_auth.go
      - internal/cli/mcp_auth_test.go
```

**Checkpoint**: Wave 8.AK lets API and CLI MCP OAuth start flows accept optional client credentials, persist them only in the short-lived OAuth flow, include `client_id` in the authorization URL, and post both credentials during token exchange without returning or printing `client_secret`.

### Wave 8.AL — Dashboard MCP Instance Launch Fields

```yaml
wave: "8.AL"
status: DONE
max_agents: 1
gate: "npm --prefix ui test -- --run McpSplitPages && npm --prefix ui run e2e -- --grep \"MCP\" && npm --prefix ui test -- --run && npm --prefix ui run build"
completed_at: "2026-06-04T12:28:00Z"
evaluator_prompt: "docs/evaluations/wave-8AL-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e9288-03af-7012-96f3-438cf4382954 after npm ci dependency setup"
gate_results:
  - "npm --prefix ui test -- --run McpSplitPages: PASS"
  - "npm --prefix ui run e2e -- --grep \"MCP\": PASS"
  - "npm --prefix ui test -- --run: PASS after post-merge stabilization commit 566ab53"
  - "npm --prefix ui run build: PASS"

tasks:
  - id: "8.AL.1"
    name: "Dashboard MCP instance advanced launch fields"
    status: DONE
    agent: "subagent 019e927c-30e9-71b2-a839-e1a7378b8217"
    commit: "c1c5b3a"
    merge_commit: "42762e9"
    followup_commit: "566ab53"
    files_owned:
      - ui/src/pages/McpPage.tsx
      - ui/src/pages/McpSplitPages.test.tsx
      - ui/e2e/dashboard.e2e.ts
```

**Checkpoint**: Wave 8.AL adds dashboard form support for MCP instance `args`, `headers`, `env`, and `cwd`, validates JSON before POST, omits empty values, keeps secret values out of rendered instance rows, and proves the flow in both unit and mocked Playwright E2E coverage.

### Wave 8.AM — Streamable HTTP MCP Initialize Params

```yaml
wave: "8.AM"
status: DONE
max_agents: 1
gate: "go test ./internal/mcp -run 'TestHTTPLauncherStoresStreamableSessionID|TestStreamableHTTPClientListsAndCallsTools' -count=1 && go test ./internal/mcp -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router"
completed_at: "2026-06-04T12:40:00Z"
evaluator_prompt: "docs/evaluations/wave-8AM-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e9294-e5ff-75c1-8d66-1f3474d97ff5; follow-up docs removed nonexistent focused test name"
gate_results:
  - "go test ./internal/mcp -run 'TestHTTPLauncherStoresStreamableSessionID|TestStreamableHTTPClientListsAndCallsTools' -count=1: PASS"
  - "go test ./internal/mcp -count=1: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"

tasks:
  - id: "8.AM.1"
    name: "Streamable HTTP launcher initialize params"
    status: DONE
    agent: "subagent 019e928c-61f2-7882-a108-f54251c29863"
    commit: "ba401d8"
    merge_commit: "f98302a"
    files_owned:
      - internal/mcp/http.go
      - internal/mcp/launcher_test.go
```

**Checkpoint**: Wave 8.AM makes the legacy streamable HTTP launcher send MCP `initialize` params with protocol version, capabilities, and `clientInfo`, matching the runtime streamable HTTP client while preserving protocol headers, session capture, and initialized notification behavior.

### Wave 8.AN — Dashboard MCP OAuth Resource Discovery

```yaml
wave: "8.AN"
status: DONE
max_agents: 1
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e"
completed_at: "2026-06-04T12:39:47Z"
evaluator_prompt: "docs/evaluations/wave-8AN-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e92b6-2143-7f42-94f4-4662ceb729a9 after API listener stabilization; prompt gate-name cleanup committed in 875d894"
gate_results:
  - "npm --prefix ui test -- --run McpSplitPages --reporter=dot: PASS in worker before merge"
  - "npm --prefix ui run e2e -- --grep 'MCP': PASS in worker before merge"
  - "npm --prefix ui test -- --run: PASS on main"
  - "npm --prefix ui run build: PASS on main"
  - "npm --prefix ui run e2e: PASS on main"
  - "go test ./api -run 'TestPublicRoutesBypassAuth|TestInferenceLoggingRecordsFailedRequestWhenEnabled' -count=100: PASS on main"
  - "go test ./api -count=20 -shuffle=on: RECORDED PASS on main before external evaluator reproduced intermittent API listener failure"
  - "go test ./api -run 'TestInferenceLoggingRecordsStreamingUsageWhenEnabled|TestInferenceLoggingUsesPublicCatalogModelForProviderQualifiedRoute|TestManagementRoutesDispatchThroughServer' -count=50 -shuffle=on: PASS on codex/wave-8an-api-shuffle-stabilization"
  - "go test ./api -count=20 -shuffle=on: PASS on codex/wave-8an-api-shuffle-stabilization"
  - "go test ./... -count=1: PASS on codex/wave-8an-api-shuffle-stabilization"
  - "go vet ./...: PASS on codex/wave-8an-api-shuffle-stabilization"
  - "go build ./cmd/g0router: PASS on codex/wave-8an-api-shuffle-stabilization"
  - "go test ./... -count=1: PASS on main after rerun without concurrent UI server"
  - "go vet ./...: PASS on main"
  - "go build ./cmd/g0router: PASS on main"
  - "initial concurrent Go/UI gate attempt: go test ./... failed once in api with plain 403 while UI dev server was also active; isolated/repeated API and sequential full Go gates passed"

tasks:
  - id: "8.AN.1"
    name: "Dashboard MCP OAuth resource discovery start"
    status: DONE
    agent: "subagent 019e9296-f7c7-7443-a9bc-050d23ee55cd with orchestrator finish"
    branch: "codex/wave-8an-dashboard-mcp-oauth-resource-discovery"
    commit: "1323d51"
    merge_commit: "325b248"
    files_owned:
      - ui/src/pages/McpPage.tsx
      - ui/src/pages/McpSplitPages.test.tsx
      - ui/e2e/dashboard.e2e.ts
```

**Checkpoint**: Wave 8.AN lets the dashboard start MCP OAuth from a Resource URI alone, matching the backend protected-resource discovery flow added in Wave 8.AG. The UI now performs explicit alternate-field validation instead of blocking on required browser fields, and unit plus mocked Playwright coverage prove the blank `authorization_url` request body.

---

### Wave 8.AO — Dashboard Provider OAuth Connect Flow

```yaml
wave: "8.AO"
status: DONE
max_agents: 1
gate: "npm --prefix ui test -- --run ProvidersPage ConnectionsAuthPage && npm --prefix ui run e2e -- --grep 'OAuth' && npm --prefix ui test -- --run && npm --prefix ui run build"
completed_at: "2026-06-04T13:03:52Z"
evaluator_prompt: "docs/evaluations/wave-8AO-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e92bd-04dd-7943-ab26-3c0b02c85544 with no blocking findings; evaluator prompt commit refs clarified in 7d2c5c0"
gate_results:
  - "npm --prefix ui test -- --run ProvidersPage ConnectionsAuthPage: PASS on main, 13 tests"
  - "npm --prefix ui run e2e -- --grep 'OAuth': PASS on main, 4 tests"
  - "npm --prefix ui test -- --run: PASS on main, 83 tests"
  - "npm --prefix ui run build: PASS on main"
  - "generated ui/test-results removed and tracked ui/dist rewrites restored after build verification"

tasks:
  - id: "8.AO.1"
    name: "Dashboard provider OAuth connect flow"
    status: DONE
    agent: "subagent 019e92a8-50fc-7511-ad86-bdd0f528e991"
    branch: "codex/wave-8ao-dashboard-provider-oauth-connect"
    commit: "95a6f22"
    merge_commit: "d2a2972"
    files_owned:
      - ui/src/api.ts
      - ui/src/pages/ProvidersPage.tsx
      - ui/src/pages/ProvidersPage.test.tsx
      - ui/src/pages/ConnectionsAuthPage.test.tsx
      - ui/e2e/dashboard.e2e.ts
```

**Checkpoint**: Wave 8.AO adds a dashboard provider OAuth start/exchange flow to the shared Providers and Connections/Auth control plane. OAuth controls render only for providers advertising `oauth`, start requests send the account label to `/api/oauth/{provider}/authorize`, exchange requests complete through `/api/oauth/{provider}/exchange`, and tests prove redacted connection display without rendering access tokens, refresh tokens, or API keys.

---

### Wave 8.AP — Dashboard Quotas Naming Reconciliation

```yaml
wave: "8.AP"
status: DONE
max_agents: 1
gate: "npm --prefix ui test -- --run App QuotaPage && npm --prefix ui run e2e -- --grep 'Quotas' && npm --prefix ui run e2e && npm --prefix ui test -- --run && npm --prefix ui run build"
completed_at: "2026-06-04T13:19:03Z"
evaluator_prompt: "docs/evaluations/wave-8AP-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e8e4c-d356-7150-990a-2a84fdb825e4 with no blocking findings; full UI gate suite passed with generated artifacts clean after content refresh"
gate_results:
  - "npm --prefix ui test -- --run App QuotaPage: PASS on main, 11 tests"
  - "npm --prefix ui run e2e -- --grep 'Quotas': PASS on main, 2 tests"
  - "npm --prefix ui run e2e: PASS on main, 20 tests"
  - "npm --prefix ui test -- --run: PASS on main, 84 tests"
  - "npm --prefix ui run build: PASS on main"
  - "generated ui/test-results removed and tracked ui/dist rewrites restored after build verification"

tasks:
  - id: "8.AP.1"
    name: "Dashboard Quotas naming reconciliation"
    status: DONE
    agent: "subagent 019e92c2-44b4-7362-b59f-bb1964134aba"
    branch: "codex/wave-8ap-dashboard-quotas-label"
    commit: "7e0830b"
    merge_commit: "15da585"
    files_owned:
      - ui/src/App.tsx
      - ui/src/App.test.tsx
      - ui/src/pages/QuotaPage.tsx
      - ui/src/pages/QuotaPage.test.tsx
      - ui/e2e/dashboard.e2e.ts
```

**Checkpoint**: Wave 8.AP reconciles the dashboard user-facing quota page label with the Stage 8 dashboard scope by rendering navigation, route title, page heading, and E2E navigation as `Quotas` while preserving the stable `quota` route id and `/api/usage/quota/{provider}` backend contract.

---

### Wave 8.AQ — Phase Documentation Completion Wording

```yaml
wave: "8.AQ"
status: DONE
max_agents: 1
gate: "rg -n '^### TODO$|Create the test file referenced in TODO|implementation does not exist|implementation doesn''t exist' docs/phases --glob '*.md' && false || true; rg -n -- '- \\[ \\]' docs/phases docs/WORKFLOW.md docs/PLAN.md docs/ORCHESTRATION.md docs/README.md && false || true"
completed_at: "2026-06-04T13:23:16Z"
evaluator_prompt: "docs/evaluations/wave-8AQ-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e9189-4e7f-7ce3-9c7f-1930e571b5b3 after metadata remediation commit ffc9419"
gate_results:
  - "rg -n '^### TODO$|Create the test file referenced in TODO|implementation does not exist|implementation doesn''t exist' docs/phases --glob '*.md': PASS, no stale phase TODO wording"
  - "rg -n -- '- \\[ \\]' docs/phases docs/WORKFLOW.md docs/PLAN.md docs/ORCHESTRATION.md docs/README.md: PASS, no unchecked task boxes"

tasks:
  - id: "8.AQ.1"
    name: "Reconcile stale phase TODO wording"
    status: DONE
    agent: "orchestrator"
    commit: "65eeb01"
    files_owned:
      - docs/WORKFLOW.md
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/evaluations/wave-8AQ-evaluator-prompt.md
      - docs/phases/phase-00-project-bootstrap.md
      - docs/phases/phase-01-core-types-sqlite-store.md
      - docs/phases/phase-02-http-server-proxy-engine.md
      - docs/phases/phase-03-multi-provider-support.md
      - docs/phases/phase-04-persistence-provider-registry.md
      - docs/phases/phase-05-oauth-flows-cli.md
      - docs/phases/phase-06-account-fallback-combos.md
      - docs/phases/phase-07-rtk-caveman.md
      - docs/phases/phase-08-usage-tracking-cost-logging.md
      - docs/phases/phase-09-mcp-gateway.md
      - docs/phases/phase-10-dashboard-ui.md
      - docs/phases/phase-11-packaging-deployment-polish.md
      - docs/phases/phase-12-advanced-mcp-gateway.md
```

**Checkpoint**: Wave 8.AQ removes stale future-tense TODO wording from completed phase documents. Task checklists remain intact and completed, while the repeated TDD boilerplate now reads as implementation evidence instead of claiming the implementation still does not exist.

---

### Wave 8.AR — MCP Unsupported Launch Transport Guard

```yaml
wave: "8.AR"
status: DONE
max_agents: 1
gate: "go test ./internal/cli -run TestMCPLauncherConnectorRejectsUnsupportedLaunchTransport -count=1 && go test ./internal/cli -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router"
completed_at: "2026-06-04T13:30:32Z"
evaluator_prompt: "docs/evaluations/wave-8AR-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e9189-70a4-72d2-a2ec-aa14e286a882 with no blocking findings; all Go gates passed"
gate_results:
  - "go test ./internal/cli -run TestMCPLauncherConnectorRejectsUnsupportedLaunchTransport -count=1: PASS"
  - "go test ./internal/cli -count=1: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"

tasks:
  - id: "8.AR.1"
    name: "Remove silent MCP fallback client for unsupported launch transports"
    status: DONE
    agent: "orchestrator"
    commit: "199d4d3"
    files_owned:
      - internal/cli/mcp_runtime.go
      - internal/cli/mcp_runtime_test.go
      - docs/WORKFLOW.md
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/evaluations/wave-8AR-evaluator-prompt.md
```

**Checkpoint**: Wave 8.AR removes the unreachable MCP fallback client that silently returned no tools and tool-not-found for unknown launch transports. The connector now rejects unsupported launcher transports with `mcp.ErrInvalidClientConfig` and closes any launched process before returning the error.

---

### Wave 8.AS — Phase 12 Completed Data Model Wording

```yaml
wave: "8.AS"
status: DONE
max_agents: 1
gate: "rg -n 'Data Model Plan|Phase 12 should add or migrate' docs/phases/phase-12-advanced-mcp-gateway.md && false || true; rg -n -- '- \\[ \\]' docs/phases docs/WORKFLOW.md docs/PLAN.md docs/ORCHESTRATION.md docs/README.md && false || true"
completed_at: "2026-06-04T13:38:25Z"
evaluator_prompt: "docs/evaluations/wave-8AS-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e92dc-320d-75c1-9b05-74938b8339e8 after metadata fix 4da4d73; no blocking findings"
gate_results:
  - "rg -n 'Data Model Plan|Phase 12 should add or migrate' docs/phases/phase-12-advanced-mcp-gateway.md: PASS, no matches"
  - "rg -n -- '- \\[ \\]' docs/phases docs/WORKFLOW.md docs/PLAN.md docs/ORCHESTRATION.md docs/README.md: PASS, no matches"
  - "git diff --check: PASS"

tasks:
  - id: "8.AS.1"
    name: "Reconcile Phase 12 completed data model wording"
    status: DONE
    agent: "orchestrator"
    commit: "0da6717"
    files_owned:
      - docs/phases/phase-12-advanced-mcp-gateway.md
      - docs/WORKFLOW.md
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/evaluations/wave-8AS-evaluator-prompt.md
```

**Checkpoint**: Wave 8.AS removes the last future-facing Phase 12 data model heading and wording so the completed phase document no longer describes the instance-oriented MCP schema as future work.

---

### Wave 8.AT — Release Lock Verification

```yaml
wave: "8.AT"
status: DONE
max_agents: 1
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build"
completed_at: "2026-06-04T13:44:19Z"
evaluator_prompt: "docs/evaluations/wave-8AT-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e930c-4870-73a2-ba64-b7c97d9de3a5 after metadata remediation commit 3e88212; no blocking findings"
gate_results:
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS, 20 files and 84 tests"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS, 20 tests"
  - "make build: initially FAIL under production npm config because dev build tooling was omitted; fixed by forcing npm ci --include=dev in Makefile"
  - "make build after Makefile fix: PASS"
  - "artifact cleanup after release gates: PASS, generated binary, UI dist rewrites, and Playwright test-results removed/restored"
  - "secret scan for leaked MiniMax/API token patterns, excluding the evaluator prompt that contains the scan expression itself: PASS"
  - "unmerged branch audit: PASS with known stale conflicting branch codex/wave-8an-dashboard-mcp-oauth-resource-discovery intentionally unmerged"
  - "git diff --check: PASS"
  - "git status --short: PASS, only protected local dirt .DS_Store, docs/.DS_Store, .pi/, and AGENTS.md"

tasks:
  - id: "8.AT.1"
    name: "Run final release-lock verification"
    status: DONE
    agent: "orchestrator"
    commit: "e966841"
    files_owned:
      - docs/WORKFLOW.md
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/evaluations/wave-8AT-evaluator-prompt.md
  - id: "8.AT.2"
    name: "Fix clean npm install release build gate"
    status: DONE
    agent: "orchestrator"
    commit: "3b11b46"
    files_owned:
      - Makefile
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8AT-evaluator-prompt.md
```

**Checkpoint**: Wave 8.AT records a full release-lock gate pass from the main checkout after fixing `make build` to install UI build tooling even when npm is configured to omit development dependencies. Generated artifacts from the gate were cleaned afterward; only protected local dirt remains in the worktree.

---

### Wave 8.AU — Gemini Streaming Parity

```yaml
wave: "8.AU"
status: DONE
max_agents: 1
gate: "go test ./internal/providers/gemini ./internal/provider -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router"
completed_at: "2026-06-04T17:01:59Z"
evaluator_prompt: "docs/evaluations/wave-8AU-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e9399-3449-76b0-83c1-b545c3a564aa; no blocking findings"
gate_results:
  - "go test ./internal/providers/gemini -run 'TestChatCompletionStreamMapsGeminiSSEChunks|TestChatCompletionStreamWithOAuthUsesBearerAndAltSSE' -count=1: RED before implementation, ChatCompletionStream returned gemini unsupported operation"
  - "go test ./internal/providers/gemini -run 'TestChatCompletionStreamMapsGeminiSSEChunks|TestChatCompletionStreamWithOAuthUsesBearerAndAltSSE' -count=1: PASS"
  - "go test ./internal/providers/gemini ./internal/provider -count=1: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"

tasks:
  - id: "8.AU.1"
    name: "Implement native Gemini SSE streaming"
    status: DONE
    agent: "orchestrator"
    commit: "4cc160d"
    files_owned:
      - internal/providers/gemini/gemini.go
      - internal/providers/gemini/gemini_test.go
      - internal/provider/matrix.go
      - internal/provider/matrix_test.go
      - docs/PROVIDERS.md
      - docs/WORKFLOW.md
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/evaluations/wave-8AU-evaluator-prompt.md
```

**Checkpoint**: Wave 8.AU removes a concrete provider parity gap by replacing Gemini's unsupported streaming stub with a native `streamGenerateContent?alt=sse` implementation. Tests prove API-key and OAuth request behavior against a local SSE server, text/tool-call/finish/usage chunk mapping, and updated provider matrix streaming capability.

---

### Wave 8.AV — Vertex Streaming Parity

```yaml
wave: "8.AV"
status: DONE
max_agents: 1
gate: "go test ./internal/providers/vertex ./internal/provider -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router"
completed_at: "2026-06-04T17:12:53Z"
evaluator_prompt: "docs/evaluations/wave-8AV-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e93a2-ce23-7f22-a974-07f393f52bad; no blocking findings"
gate_results:
  - "go test ./internal/providers/vertex -run 'TestChatCompletionStreamMapsVertexSSEChunks|TestChatCompletionStreamMalformedSSEEmitsErrorChunk' -count=1: RED before implementation, ChatCompletionStream returned vertex unsupported operation"
  - "go test ./internal/providers/vertex -run 'TestChatCompletionStreamMapsVertexSSEChunks|TestChatCompletionStreamMalformedSSEEmitsErrorChunk' -count=1: PASS"
  - "go test ./internal/providers/vertex ./internal/provider -count=1: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"

tasks:
  - id: "8.AV.1"
    name: "Implement native Vertex SSE streaming"
    status: DONE
    agent: "orchestrator"
    commit: "8be47fd"
    files_owned:
      - internal/providers/vertex/vertex.go
      - internal/providers/vertex/vertex_test.go
      - internal/provider/matrix.go
      - internal/provider/matrix_test.go
      - docs/PROVIDERS.md
      - docs/WORKFLOW.md
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/evaluations/wave-8AU-evaluator-prompt.md
      - docs/evaluations/wave-8AV-evaluator-prompt.md
```

**Checkpoint**: Wave 8.AV removes Vertex's unsupported streaming stub and adds native Vertex `streamGenerateContent?alt=sse` support for configured project/location routing. Tests prove bearer-auth request behavior against a local SSE server, text/finish/usage chunk mapping, malformed stream error chunks, and updated provider matrix streaming capability.

---

### Wave 8.AW — Bedrock Model Listing Parity

```yaml
wave: "8.AW"
status: DONE
max_agents: 1
gate: "go test ./internal/providers/bedrock ./internal/provider ./api/handlers -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router"
completed_at: "2026-06-04T17:26:04Z"
evaluator_prompt: "docs/evaluations/wave-8AW-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e93ae-f9e1-74c1-8adb-41ffb0550d39 after endpoint remediation; no blocking findings"
gate_results:
  - "go test ./internal/providers/bedrock -run TestListModelsSignsAndParsesFoundationModels -count=1: RED before implementation, ListModels returned bedrock list models: unsupported"
  - "go test ./internal/providers/bedrock -run TestListModelsSignsAndParsesFoundationModels -count=1: PASS"
  - "go test ./internal/providers/bedrock -run TestListModelsUsesBedrockControlPlaneEndpointByDefault -count=1: RED before endpoint fix, default ListModels used bedrock-runtime instead of bedrock control-plane endpoint"
  - "go test ./internal/providers/bedrock -run 'TestListModelsUsesBedrockControlPlaneEndpointByDefault|TestListModelsSignsAndParsesFoundationModels' -count=1: PASS"
  - "go test ./internal/providers/bedrock ./internal/provider ./api/handlers -count=1: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS"
  - "make build: PASS"

tasks:
  - id: "8.AW.1"
    name: "Implement signed Bedrock foundation model listing"
    status: DONE
    agent: "orchestrator"
    commit: "d088f8f"
    fix_commit: "498ffaa"
    files_owned:
      - internal/providers/bedrock/bedrock.go
      - internal/providers/bedrock/bedrock_test.go
      - internal/provider/matrix.go
      - internal/provider/matrix_test.go
      - api/handlers/providers_test.go
      - docs/PROVIDERS.md
      - docs/WORKFLOW.md
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/evaluations/wave-8AW-evaluator-prompt.md
```

**Checkpoint**: Wave 8.AW removes one Bedrock adapter gap by replacing the unsupported `ListModels` stub with a signed `GET /foundation-models` implementation. Tests prove SigV4 headers, session-token propagation, response parsing, and provider matrix/API exposure while keeping Bedrock non-public for streaming, catalog routing, quota, and direct dispatch.

---

### Wave 8.AX — Bedrock Converse Adapter Parity

```yaml
wave: "8.AX"
status: DONE
max_agents: 1
gate: "go test ./internal/providers/bedrock ./internal/provider ./api/handlers ./internal/proxy -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router"
completed_at: "2026-06-04T17:45:48Z"
evaluator_prompt: "docs/evaluations/wave-8AX-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e93c1-50db-7ae1-8fb3-ee9d6fdeea12 after stopSequences remediation commit 3c0ef3e; no blocking findings"
gate_results:
  - "go test ./internal/providers/bedrock -run 'TestChatCompletionSignsConverseRequest|TestChatCompletionParsesBedrockResponse' -count=1: RED before implementation, missing Converse request contract"
  - "go test ./internal/providers/bedrock -run 'TestChatCompletionSignsConverseRequest|TestChatCompletionParsesBedrockResponse' -count=1: PASS"
  - "go test ./internal/providers/bedrock -run 'TestChatCompletionNormalizesStopArray|TestChatCompletionRejectsUnsupportedStopShape' -count=1: RED after evaluator, []any stop was dropped and unsupported stop reached the network path"
  - "go test ./internal/providers/bedrock -run 'TestChatCompletionSignsConverseRequest|TestChatCompletionNormalizesStopArray|TestChatCompletionRejectsUnsupportedStopShape|TestChatCompletionParsesBedrockResponse' -count=1: PASS after stopSequences remediation"
  - "go test ./internal/providers/bedrock -count=1: PASS after stopSequences remediation"
  - "go test ./internal/providers/bedrock ./internal/provider ./api/handlers ./internal/proxy -run 'TestChatCompletionSignsConverseRequest|TestChatCompletionParsesBedrockResponse|TestProviderMatrixKeepsBedrockAdapterOnlyAfterConverseSupport|TestProvidersListKnownProviders|TestDispatchUsesBedrockAliasThroughAdapterOnlyInference|TestComboDispatchUsesBedrockAdapterOnlyStep' -count=1: PASS"
  - "go test ./... -count=1: PASS after stopSequences remediation"
  - "go vet ./...: PASS after stopSequences remediation"
  - "go build ./cmd/g0router: PASS after stopSequences remediation"
  - "npm --prefix ui test -- --run: PASS after stopSequences remediation, 20 files and 84 tests"
  - "make build: PASS after stopSequences remediation"
  - "npm --prefix ui run build: PASS after stopSequences remediation"
  - "npm --prefix ui run e2e: PASS after stopSequences remediation, 20 tests"

tasks:
  - id: "8.AX.1"
    name: "Implement non-streaming Bedrock Converse dispatch"
    status: DONE
    agent: "orchestrator"
    commit: "e9c8a78"
    fix_commit: "3c0ef3e"
    files_owned:
      - internal/providers/bedrock/bedrock.go
      - internal/providers/bedrock/bedrock_test.go
      - internal/providers/bedrock/types.go
      - internal/provider/matrix.go
      - internal/provider/matrix_test.go
      - api/handlers/providers_test.go
      - internal/proxy/engine_test.go
      - internal/proxy/combo_test.go
      - docs/PROVIDERS.md
      - docs/WORKFLOW.md
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/evaluations/wave-8AX-evaluator-prompt.md
```

**Checkpoint**: Wave 8.AX replaces the Bedrock Anthropic-native invoke request path with the AWS Bedrock Converse API for non-streaming chat completions. Explicit aliases and combo steps can route to Bedrock adapter-only inference. Wave 8.AY promotes a cataloged Bedrock Converse model to public direct dispatch; streaming and quota remain disabled.

### Wave 8.AY — Bedrock Catalog Direct Dispatch

```yaml
wave: "8.AY"
status: DONE
max_agents: 1
gate: "go test ./internal/modelcatalog ./internal/provider ./internal/proxy ./api/handlers ./internal/cli -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build"
completed_at: "2026-06-04T18:14:41Z"
evaluator_prompt: "docs/evaluations/wave-8AY-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e93dc-087a-73e0-866f-846ff956d150 after generated UI artifact remediation commit 3699aa4; no blocking findings"
gate_results:
  - "go test ./internal/modelcatalog -run 'TestCatalogRouteForBedrockConverseModel|TestCatalogIncludesRepresentativeWave7IProviderCoverage|TestCatalogHostedModelsHaveExplicitNonZeroRates|TestCatalogOmitsProvidersWithoutDefensibleEmbeddedPricing' -count=1: RED before implementation, Bedrock catalog route and pricing missing"
  - "go test ./internal/provider -run 'TestProviderMatrix.*Bedrock|TestPublicInferenceProvidersExcludeUnsupportedAndAuthOnlyEntries|TestPublicProvidersDoNotClaimQuotaSupport' -count=1: RED before implementation, Bedrock still adapter_only"
  - "go test ./internal/proxy -run TestDispatchUsesCatalogForBedrockConverseModel -count=1: RED before implementation, provider not found"
  - "go test ./api/handlers -run TestProvidersListKnownProviders -count=1: RED before implementation, Bedrock was not supported public inference"
  - "go test ./internal/cli -run 'TestProvidersListShowsKnownProviders|TestProvidersListShowsSupportedInferenceProvidersOnly' -count=1: RED before implementation, Bedrock absent from providers list"
  - "focused Bedrock catalog/matrix/proxy/API/CLI tests: PASS after implementation"
  - "go test ./internal/modelcatalog -run 'TestCatalogRouteForBedrockConverseModel|TestCatalogIncludesRepresentativeWave7IProviderCoverage|TestCatalogHostedModelsHaveExplicitNonZeroRates|TestCatalogOmitsProvidersWithoutDefensibleEmbeddedPricing' -count=1: PASS"
  - "go test ./internal/provider -run 'TestProviderMatrix.*Bedrock|TestPublicInferenceProvidersExcludeUnsupportedAndAuthOnlyEntries|TestPublicProvidersDoNotClaimQuotaSupport' -count=1: PASS"
  - "go test ./internal/proxy -run 'TestDispatchUsesCatalogForBedrockConverseModel|TestDispatchUsesBedrockAliasThroughAdapterOnlyInference|TestComboDispatchUsesBedrockAdapterOnlyStep' -count=1: PASS"
  - "go test ./api/handlers -run TestProvidersListKnownProviders -count=1: PASS"
  - "go test ./internal/cli -run 'TestProvidersListShowsKnownProviders|TestProvidersListShowsSupportedInferenceProvidersOnly' -count=1: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS, 20 files and 84 tests"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS, 20 tests"
  - "make build: PASS"
  - "external evaluator thread 019e93dc-087a-73e0-866f-846ff956d150: FAIL because npm/make build rewrote tracked ui/dist assets and Playwright created untracked ui/test-results"
  - "npm --prefix ui run build: PASS after artifact remediation"
  - "npm --prefix ui run e2e: PASS after artifact remediation, 20 tests"
  - "make build: PASS after artifact remediation"
  - "go test ./internal/modelcatalog -run 'TestCatalogRouteForBedrockConverseModel|TestCatalogIncludesRepresentativeWave7IProviderCoverage|TestCatalogHostedModelsHaveExplicitNonZeroRates|TestCatalogOmitsProvidersWithoutDefensibleEmbeddedPricing' -count=1: PASS after artifact remediation"
  - "go test ./internal/provider -run 'TestProviderMatrix.*Bedrock|TestPublicInferenceProvidersExcludeUnsupportedAndAuthOnlyEntries|TestPublicProvidersDoNotClaimQuotaSupport' -count=1: PASS after artifact remediation"
  - "go test ./internal/proxy -run 'TestDispatchUsesCatalogForBedrockConverseModel|TestDispatchUsesBedrockAliasThroughAdapterOnlyInference|TestComboDispatchUsesBedrockAdapterOnlyStep' -count=1: PASS after artifact remediation"
  - "go test ./api/handlers -run TestProvidersListKnownProviders -count=1: PASS after artifact remediation"
  - "go test ./internal/cli -run 'TestProvidersListShowsKnownProviders|TestProvidersListShowsSupportedInferenceProvidersOnly' -count=1: PASS after artifact remediation"
  - "go test ./... -count=1: PASS after artifact remediation"
  - "git diff --check: PASS after artifact remediation"
  - "git status --short: only protected local dirt plus intended .gitignore and ui/dist assets before commit"
  - "external evaluator thread 019e93dc-087a-73e0-866f-846ff956d150 re-evaluation: PASS after remediation commit 3699aa4; no ui/dist drift and ui/test-results ignored"

tasks:
  - id: "8.AY.1"
    name: "Promote cataloged Bedrock Converse model to public direct dispatch"
    status: DONE
    agent: "orchestrator"
    files_owned:
      - internal/modelcatalog/catalog.go
      - internal/modelcatalog/pricing_test.go
      - internal/provider/matrix.go
      - internal/provider/matrix_test.go
      - internal/proxy/engine_test.go
      - api/handlers/providers_test.go
      - internal/cli/providers_test.go
      - internal/cli/root_test.go
      - docs/PROVIDERS.md
      - docs/WORKFLOW.md
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/evaluations/wave-8AY-evaluator-prompt.md
  - id: "8.AY.2"
    name: "Resolve evaluator generated UI artifact hygiene failure"
    status: DONE
    agent: "orchestrator"
    files_owned:
      - .gitignore
      - ui/dist/assets/index.css
      - ui/dist/assets/index.js
      - docs/WORKFLOW.md
```

**Checkpoint**: Wave 8.AY adds catalog-backed direct dispatch for Bedrock model `anthropic.claude-3-5-haiku-20241022-v1:0` through the existing non-streaming Converse adapter. Bedrock remains non-streaming and quota remains unsupported. Evaluator thread `019e93dc-087a-73e0-866f-846ff956d150` initially found generated UI artifact drift after the required gates; task 8.AY.2 commits regenerated `ui/dist` assets and ignores Playwright `ui/test-results/`, and the evaluator re-run passed after remediation commit `3699aa4`.

### Wave 8.AZ — Provider-Qualified Dynamic Adapter Routing

```yaml
wave: "8.AZ"
status: DONE
max_agents: 1
gate: "go test ./internal/proxy ./internal/provider ./api/handlers ./internal/cli -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build"
completed_at: "2026-06-04T18:53:35Z"
evaluator_prompt: "docs/evaluations/wave-8AZ-evaluator-prompt.md"
evaluation: "PASS subagent evaluator 019e93cf-c0d7-7942-b5af-ee590191ff16 at commit f7c0cbd; no blocking findings"
gate_results:
  - "go test ./internal/proxy -run 'TestDispatchUsesProviderQualifiedDynamicRouteForDeploymentDefinedProviders|TestDispatchPrefersExactCatalogBeforeProviderQualifiedDynamicRoute' -count=1: RED before implementation, provider-qualified dynamic models returned provider not found"
  - "go test ./internal/provider -run 'TestProviderMatrixMarksDeploymentDefinedAdaptersAsDynamicPublicRoutes|TestPublicInferenceProvidersExcludeUnsupportedAndAuthOnlyEntries|TestOpenAICompatibleGatewayProvidersUseDynamicPublicRoutesWithoutFakeCatalogs' -count=1: RED before implementation, deployment-defined adapters remained adapter_only"
  - "go test ./api/handlers -run TestProvidersListKnownProviders -count=1: RED before implementation, provider metadata still reported adapter_only"
  - "go test ./internal/proxy -run 'TestDispatchUsesProviderQualifiedDynamicRouteForDeploymentDefinedProviders|TestDispatchPrefersExactCatalogBeforeProviderQualifiedDynamicRoute|TestDispatchStreamUsesProviderQualifiedDynamicRouteForDeploymentDefinedProviders|TestDispatchRejectsInvalidProviderQualifiedDynamicRoutes' -count=1: PASS"
  - "go test ./internal/provider -run 'TestProviderMatrixMarksDeploymentDefinedAdaptersAsDynamicPublicRoutes|TestReplicateRemainsAdapterOnlyUntilPublicSemanticsAreProven|TestPublicInferenceProvidersExcludeUnsupportedAndAuthOnlyEntries|TestOpenAICompatibleGatewayProvidersUseDynamicPublicRoutesWithoutFakeCatalogs|TestPublicProvidersDoNotClaimQuotaSupport' -count=1: PASS"
  - "go test ./api/handlers -run TestProvidersListKnownProviders -count=1 && go test ./internal/cli -run 'TestProvidersListShowsKnownProviders|TestProvidersListShowsSupportedInferenceProvidersOnly' -count=1: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS, 20 files and 84 tests"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS, 20 tests"
  - "make build: PASS"
  - "git diff --check: PASS"
  - "subagent evaluator 019e93cf-c0d7-7942-b5af-ee590191ff16 at commit f7c0cbd: PASS with no blocking findings; only protected local dirt noted"

tasks:
  - id: "8.AZ.1"
    name: "Promote deployment-defined adapters through provider-qualified dynamic routing"
    status: DONE
    agent: "orchestrator"
    files_owned:
      - internal/proxy/engine.go
      - internal/proxy/engine_test.go
      - internal/provider/matrix.go
      - internal/provider/matrix_test.go
      - api/handlers/providers_test.go
      - internal/cli/providers_test.go
      - internal/cli/root_test.go
      - docs/PROVIDERS.md
      - docs/WORKFLOW.md
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/evaluations/wave-8AZ-evaluator-prompt.md
```

**Checkpoint**: Wave 8.AZ adds provider-qualified dynamic public routing for deployment-defined registered adapters: `azure/<deployment>`, `litellm/<model>`, `lm-studio/<loaded-model>`, and `vllm/<served-model>`. Exact catalog matches still win before dynamic prefix routing, so catalog-owned slash models such as OpenRouter models are not hijacked. These providers remain without static catalog pricing or quota fetchers. Replicate stays adapter-only until its public API semantics are proven.

---

### Wave 8.BA — GitHub Copilot Runtime Routing

```yaml
wave: "8.BA"
status: DONE
max_agents: 1
gate: "go test ./internal/providers/openaicompat ./internal/proxy ./internal/provider ./api/handlers ./internal/cli -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build"
completed_at: "2026-06-04T19:34:00Z"
evaluator_prompt: "docs/evaluations/wave-8BA-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e93cf-c0d7-7942-b5af-ee590191ff16 at commit acd3249; no blocking findings"
gate_results:
  - "go test ./internal/providers/openaicompat -run 'TestConfiguredProvidersUseOpenAICompatibleEndpoints|TestGitHubCopilotDefaultProviderSendsOMPHeaders|TestDefaultConfigsAreRegistered' -count=1: RED before implementation, Config.Headers and GitHub Copilot default config were missing"
  - "go test ./internal/proxy -run TestDispatchUsesProviderQualifiedDynamicRouteForDeploymentDefinedProviders -count=1: RED before implementation, github-copilot/gpt-4o returned provider not found"
  - "go test ./internal/providers/openaicompat -run 'TestConfiguredProvidersUseOpenAICompatibleEndpoints|TestGitHubCopilotDefaultProviderSendsOMPHeaders|TestDefaultConfigsAreRegistered' -count=1: PASS"
  - "go test ./internal/proxy -run 'TestDispatchUsesProviderQualifiedDynamicRouteForDeploymentDefinedProviders|TestDispatchStreamUsesProviderQualifiedDynamicRouteForDeploymentDefinedProviders' -count=1: PASS"
  - "go test ./internal/provider -run 'TestProviderMatrixMarksAuthOnlyProvidersExplicitly|TestProviderMatrixMarksDeploymentDefinedAdaptersAsDynamicPublicRoutes|TestPublicInferenceProvidersExcludeUnsupportedAndAuthOnlyEntries|TestPublicProvidersDoNotClaimQuotaSupport|TestOpenAICompatibleGatewayProvidersUseDynamicPublicRoutesWithoutFakeCatalogs|TestProviderMatrixSupportedEntriesHaveUsableSurface' -count=1: PASS"
  - "go test ./api/handlers -run TestProvidersListKnownProviders -count=1 && go test ./internal/cli -run 'TestProvidersListShowsKnownProviders|TestProvidersListShowsSupportedInferenceProvidersOnly|TestProvidersTestReportsAuthOnlyProvider' -count=1: PASS"
  - "go test ./... -count=1: PASS after updating stale auth-only provider-list negative test from github to cursor"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS, 20 files and 84 tests"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS, 20 tests"
  - "make build: PASS"
  - "git diff --check: PASS"
  - "secret scan excluding the historical evaluator prompt that contains the scan expression itself: PASS"
  - "external evaluator thread 019e93cf-c0d7-7942-b5af-ee590191ff16 at commit acd3249: PASS with no blocking findings; only protected local dirt noted"

tasks:
  - id: "8.BA.1"
    name: "Promote GitHub Copilot through OpenAI-compatible runtime routing"
    status: DONE
    agent: "orchestrator"
    files_owned:
      - internal/providers/openaicompat/provider.go
      - internal/providers/openaicompat/registry.go
      - internal/providers/openaicompat/provider_test.go
      - internal/proxy/engine.go
      - internal/proxy/engine_test.go
      - internal/provider/matrix.go
      - internal/provider/matrix_test.go
      - api/handlers/providers_test.go
      - internal/cli/provider_runtime.go
      - internal/cli/providers_test.go
      - internal/cli/root_test.go
      - docs/PROVIDERS.md
      - docs/WORKFLOW.md
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/evaluations/wave-8BA-evaluator-prompt.md
```

**Checkpoint**: Wave 8.BA promotes GitHub Copilot from auth-only to public provider-qualified runtime routing through the existing OpenAI-compatible adapter. The adapter sends `User-Agent: opencode/1.3.15`, strips `github-copilot/` before upstream dispatch, and keeps Copilot without a fake static model catalog or quota fetcher.

---

### Wave 8.BB — Cursor Auth Parity

```yaml
wave: "8.BB"
status: DONE
max_agents: 1
gate: "go test ./internal/provider/oauth -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build"
completed_at: "2026-06-04T20:12:00Z"
evaluator_prompt: "docs/evaluations/wave-8BB-evaluator-prompt.md"
evaluation: "FAIL external evaluator thread 019e9425-54b8-7861-9a92-ba1349918371; OAuthPoll could not restore the stored Cursor PKCE verifier from the public session id, remediated in Wave 8.BC"
gate_results:
  - "go test ./internal/provider/oauth -run 'TestCursorFlow(StartBuildsOMPLoginDeepControlURL|PollPendingOn404|PollCompleteStoresAccessRefreshAndExpiry|RefreshUsesOMPExchangeUserAPIKey|ExchangeUnsupported)' -count=1: RED before implementation, CursorConfig lacked LoginURL/PollURL/RefreshURL/NewUUID and Poll was unsupported"
  - "go test ./internal/provider/oauth -run 'TestCursorFlow(StartBuildsOMPLoginDeepControlURL|PollPendingOn404|PollCompleteStoresAccessRefreshAndExpiry|RefreshUsesOMPExchangeUserAPIKey|ExchangeUnsupported)' -count=1: PASS"
  - "go test ./internal/provider/oauth -count=1: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: initial full-suite run had a transient Connections/Auth loading-state failure; focused rerun of src/pages/ConnectionsAuthPage.test.tsx passed"
  - "npm --prefix ui test -- --run: PASS on full rerun, 20 files and 84 tests"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS, 20 tests"
  - "make build: PASS"
  - "git diff --check: PASS"

tasks:
  - id: "8.BB.1"
    name: "Align Cursor OAuth with loginDeepControl polling"
    status: DONE
    agent: "orchestrator"
    files_owned:
      - internal/provider/oauth/cursor.go
      - internal/provider/oauth/cursor_test.go
      - internal/provider/matrix.go
      - docs/PROVIDERS.md
      - docs/WORKFLOW.md
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/evaluations/wave-8BB-evaluator-prompt.md
```

**Checkpoint**: Wave 8.BB replaces the old Cursor callback-PKCE implementation with `loginDeepControl` auth: start creates a PKCE challenge plus UUID login URL, poll checks `api2.cursor.sh/auth/poll` with UUID and verifier, 404 remains pending, complete polls persist access/refresh tokens, and refresh uses `api2.cursor.sh/auth/exchange_user_api_key`. Cursor remains `auth_only` until a real Cursor inference adapter is implemented.

---

### Wave 8.BC — Cursor OAuth Poll Completion Remediation

```yaml
wave: "8.BC"
status: DONE
max_agents: 1
gate: "go test ./internal/store ./api/handlers -run 'TestOAuthSessionCanBeReadBeforeSingleUseConsume|TestOAuthPollUsesStoredVerifierAndAccountLabel|TestOAuthPollUsesSessionFromQuery|TestOAuthPollAcceptsGitHubAlias' -count=1 && npm --prefix ui test -- --run src/api.test.ts src/pages/ProvidersPage.test.tsx && npm --prefix ui run e2e -- --grep 'provider OAuth' && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build"
completed_at: "2026-06-04T20:48:00Z"
evaluator_prompt: "docs/evaluations/wave-8BC-evaluator-prompt.md"
evaluation: "FAIL external evaluator agent 019e946b-f591-7790-9306-108ca36743ae; OAuthPoll serialized raw poll errors that could contain tokens or PKCE verifier material, remediated in Wave 8.BD"
gate_results:
  - "external evaluator thread 019e9425-54b8-7861-9a92-ba1349918371 identified that OAuthStart returned only the public Cursor uuid/session state, while OAuthPoll needed the stored PKCE verifier to call the Cursor flow"
  - "go test ./internal/store -run TestOAuthSessionCanBeReadBeforeSingleUseConsume -count=1: PASS"
  - "go test ./api/handlers -run 'TestOAuthPollUsesStoredVerifierAndAccountLabel|TestOAuthPollUsesSessionFromQuery|TestOAuthPollAcceptsGitHubAlias' -count=1: PASS"
  - "npm --prefix ui test -- --run src/api.test.ts src/pages/ProvidersPage.test.tsx: PASS, 2 files and 19 tests"
  - "npm --prefix ui run e2e -- --grep 'provider OAuth': PASS, 4 tests"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS, 20 files and 85 tests"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS, 22 tests"
  - "make build: PASS"
  - "git diff --check: PASS"

tasks:
  - id: "8.BC.1"
    name: "Restore stored verifier during provider OAuth polling and expose dashboard poll completion"
    status: DONE
    agent: "orchestrator"
    files_owned:
      - api/handlers/oauth.go
      - api/handlers/oauth_test.go
      - internal/store/oauthsessions.go
      - internal/store/oauthsessions_test.go
      - ui/src/api.ts
      - ui/src/api.test.ts
      - ui/src/pages/ProvidersPage.tsx
      - ui/src/pages/ProvidersPage.test.tsx
      - ui/e2e/dashboard.e2e.ts
      - docs/WORKFLOW.md
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/evaluations/wave-8BC-evaluator-prompt.md
```

**Checkpoint**: Wave 8.BC fixes the Cursor OAuth completion path exposed by Wave 8.BB. Provider OAuth poll now looks up the stored verifier without consuming pending sessions, restores `state.verifier` before calling the provider flow, consumes the stored session only after a complete poll, and persists the account label into the resulting connection. The dashboard can now complete device/poll provider OAuth sessions with a `Poll OAuth` action while still supporting callback exchange flows.

---

### Wave 8.BD — OAuth Poll Error Sanitization

```yaml
wave: "8.BD"
status: DONE
max_agents: 1
gate: "go test ./api/handlers -run 'TestOAuthPoll|TestOAuthHandlers' -count=1 && go test ./internal/store -run TestOAuthSessionCanBeReadBeforeSingleUseConsume -count=1 && go test ./api/handlers -run 'TestOAuthPollUsesStoredVerifierAndAccountLabel|TestOAuthPollUsesSessionFromQuery|TestOAuthPollAcceptsGitHubAlias' -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && git diff --check"
completed_at: "2026-06-04T21:18:00Z"
evaluator_prompt: "docs/evaluations/wave-8BD-evaluator-prompt.md"
evaluation: "PASS external evaluator agent 019e9478-42c3-7d83-9610-c99c6e09ef77 at commit df1308f after metadata fix; no remaining blockers"
gate_results:
  - "go test ./api/handlers -run TestOAuthHandlersSanitizePollFlowErrors -count=1: RED before implementation, response serialized raw poll error containing access-token, refresh-token, callback-code, and cursor-verifier"
  - "go test ./api/handlers -run 'TestOAuthPoll|TestOAuthHandlers' -count=1: PASS"
  - "go test ./internal/store -run TestOAuthSessionCanBeReadBeforeSingleUseConsume -count=1: PASS"
  - "go test ./api/handlers -run 'TestOAuthPollUsesStoredVerifierAndAccountLabel|TestOAuthPollUsesSessionFromQuery|TestOAuthPollAcceptsGitHubAlias' -count=1: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "git diff --check: PASS"

tasks:
  - id: "8.BD.1"
    name: "Sanitize provider OAuth poll errors"
    status: DONE
    agent: "orchestrator"
    files_owned:
      - api/handlers/oauth.go
      - api/handlers/oauth_test.go
      - docs/WORKFLOW.md
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/evaluations/wave-8BD-evaluator-prompt.md
```

**Checkpoint**: Wave 8.BD fixes the Wave 8.BC evaluator-found leak by returning a stable `oauth poll failed` response when provider polling fails. Regression coverage proves access tokens, refresh tokens, callback codes, and Cursor PKCE verifier material from raw poll errors are not serialized to API/UI clients.

---

### Wave 8.BE — Real-Server Control Plane Integration Expansion

```yaml
wave: "8.BE"
status: DONE
max_agents: 1
gate: "go test ./api -run TestIntegrationUsageQuotaLogsAndProviderOAuthThroughAuthenticatedServer -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && git diff --check"
completed_at: "2026-06-04T21:45:00Z"
evaluator_prompt: "docs/evaluations/wave-8BE-evaluator-prompt.md"
evaluation: "PASS external evaluator agent 019e9482-6da6-77c1-a329-a617a73fd0db at commit d0713ad; no blocking findings"
gate_results:
  - "go test ./api -run TestIntegrationUsageQuotaLogsAndProviderOAuthThroughAuthenticatedServer -count=1: RED before assertions were complete, provider OAuth exchange rejected an unstored manual session"
  - "go test ./api -run TestIntegrationUsageQuotaLogsAndProviderOAuthThroughAuthenticatedServer -count=1: PASS"
  - "go test ./api -run TestIntegration -count=1: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "git diff --check: PASS"

tasks:
  - id: "8.BE.1"
    name: "Expand real-server control-plane integration coverage"
    status: DONE
    agent: "orchestrator"
    files_owned:
      - api/server_integration_test.go
      - docs/WORKFLOW.md
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/evaluations/wave-8BE-evaluator-prompt.md
```

**Checkpoint**: Wave 8.BE extends authenticated real-server integration coverage beyond route smoke tests for usage, logs, quota, and provider OAuth authorize/poll/callback/exchange. The test seeds real SQLite request logs, exercises quota with active stored provider credentials, verifies usage/log/summary responses through server middleware, and proves provider OAuth completion persists redacted connections without bypassing stored session state.

---

### Wave 8.BF — Real Dashboard Server E2E Smoke

```yaml
wave: "8.BF"
status: DONE
max_agents: 1
gate: "npm --prefix ui run e2e -- real-server.e2e.ts && npm --prefix ui test -- --run && npm --prefix ui run e2e && npm --prefix ui run build && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && git diff --check"
completed_at: "2026-06-04T22:05:00Z"
evaluator_prompt: "docs/evaluations/wave-8BF-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e948a-033d-73e3-9b0a-63646b099d63 at commit 3c30a42"
gate_results:
  - "npm --prefix ui run e2e -- real-server.e2e.ts: RED before implementation fixes, invalid Playwright skip predicate and ambiguous/broad selectors"
  - "npm --prefix ui run e2e -- real-server.e2e.ts: PASS, chromium real-server smoke passed and mobile project intentionally skipped"
  - "npm --prefix ui test -- --run: PASS, 20 files and 85 tests"
  - "npm --prefix ui run e2e: PASS, 23 tests passed and 1 real-server mobile skip"
  - "npm --prefix ui run build: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "git diff --check: PASS"
  - "external evaluator 019e948a-033d-73e3-9b0a-63646b099d63: PASS, no blocking findings"

tasks:
  - id: "8.BF.1"
    name: "Add embedded dashboard real-server E2E smoke"
    status: DONE
    agent: "orchestrator"
    files_owned:
      - ui/e2e/real-server.e2e.ts
      - docs/WORKFLOW.md
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/evaluations/wave-8BF-evaluator-prompt.md
```

**Checkpoint**: Wave 8.BF adds a Playwright smoke path that creates a temp data dir, mints a real gateway API key through the CLI, starts `g0router serve` on a random loopback port, loads the embedded dashboard from that server, saves the control-plane key, reads real settings, creates a real API key through the dashboard, and verifies it appears in the live API key table. Mocked dashboard E2E remains the broad deterministic page/action suite.

---

### Wave 8.BG — Alibaba and Zhipu Dynamic Runtime Routing

```yaml
wave: "8.BG"
status: DONE
max_agents: 1
gate: "go test ./internal/providers/openaicompat ./internal/provider ./internal/proxy ./internal/cli ./api/handlers -run 'TestConfiguredProvidersUseOpenAICompatibleEndpoints|TestZhipuDefaultProviderUsesDocumentedPaaSPathWithoutV1Prefix|TestDefaultConfigsAreRegistered|TestProviderMatrixMarksAuthOnlyProvidersExplicitly|TestProviderMatrixMarksDeploymentDefinedAdaptersAsDynamicPublicRoutes|TestPublicInferenceProvidersExcludeUnsupportedAndAuthOnlyEntries|TestPublicProvidersDoNotClaimQuotaSupport|TestOpenAICompatibleGatewayProvidersUseDynamicPublicRoutesWithoutFakeCatalogs|TestDispatchUsesProviderQualifiedDynamicRouteForDeploymentDefinedProviders|TestProvidersListShowsKnownProviders|TestProvidersListShowsSupportedInferenceProvidersOnly|TestProvidersListKnownProviders' -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build && git diff --check"
completed_at: "2026-06-04T22:32:00Z"
evaluator_prompt: "docs/evaluations/wave-8BG-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e949d-02e5-7c02-96a1-964ad923ca93 at commit 02a63ae"
gate_results:
  - "focused provider/API/CLI tests: RED before implementation, ProviderAlibaba/ProviderZhipu and Zhipu chat path override were missing; Alibaba/Zhipu remained auth_only"
  - "focused provider/API/CLI tests: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS, 20 files and 85 tests"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS, 23 tests passed and 1 real-server mobile skip"
  - "make build: PASS"
  - "git diff --check: PASS"
  - "external evaluator 019e949d-02e5-7c02-96a1-964ad923ca93: PASS, no blocking findings"

tasks:
  - id: "8.BG.1"
    name: "Promote Alibaba and Zhipu to dynamic OpenAI-compatible runtime routing"
    status: DONE
    agent: "orchestrator"
    files_owned:
      - internal/providers/types.go
      - internal/providers/openaicompat/provider.go
      - internal/providers/openaicompat/registry.go
      - internal/providers/openaicompat/provider_test.go
      - internal/provider/matrix.go
      - internal/provider/matrix_test.go
      - internal/proxy/engine.go
      - internal/proxy/engine_test.go
      - internal/cli/provider_runtime.go
      - internal/cli/providers_test.go
      - internal/cli/root_test.go
      - api/handlers/providers_test.go
      - docs/PROVIDERS.md
      - docs/WORKFLOW.md
      - docs/PLAN.md
      - docs/evaluations/wave-8BG-evaluator-prompt.md
```

**Checkpoint**: Wave 8.BG removes two concrete `auth_only` provider gaps. Alibaba now registers the OpenAI-compatible DashScope adapter and routes provider-qualified models such as `alibaba/qwen3-max-2026-01-23`. Zhipu now registers the OpenAI-compatible Z.AI adapter with the documented `/api/paas/v4/chat/completions` path and routes provider-qualified models such as `zhipu/glm-5.1`. Both providers remain intentionally non-catalog and non-quota to avoid fake pricing or unsupported quota claims.

---

### Wave 8.BH — Qianfan Dynamic Runtime Routing

```yaml
wave: "8.BH"
status: DONE
max_agents: 1
gate: "go test ./internal/provider/oauth ./internal/providers/openaicompat ./internal/provider ./internal/proxy ./internal/cli ./api/handlers -run 'TestQianfanFlow|TestConfiguredProvidersUseOpenAICompatibleEndpoints|TestDefaultConfigsAreRegistered|TestProviderMatrixMarksDeploymentDefinedAdaptersAsDynamicPublicRoutes|TestPublicInferenceProvidersExcludeUnsupportedAndAuthOnlyEntries|TestPublicProvidersDoNotClaimQuotaSupport|TestOpenAICompatibleGatewayProvidersUseDynamicPublicRoutesWithoutFakeCatalogs|TestDispatchUsesProviderQualifiedDynamicRouteForDeploymentDefinedProviders|TestProvidersListShowsKnownProviders|TestProvidersListShowsSupportedInferenceProvidersOnly|TestProvidersListKnownProviders' -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build && git diff --check"
completed_at: "2026-06-04T23:02:00Z"
evaluator_prompt: "docs/evaluations/wave-8BH-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e94ac-f7d0-7531-9780-a7fd6213b41a at commit 426ed8c"
gate_results:
  - "focused provider/API/CLI tests: RED before implementation, NewQianfanFlow and ProviderQianfan were missing; Qianfan remained unsupported"
  - "focused provider/API/CLI tests: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS, 20 files and 85 tests"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS, 23 tests passed and 1 real-server mobile skip"
  - "make build: PASS"
  - "git diff --check: PASS"
  - "external evaluator 019e94ac-f7d0-7531-9780-a7fd6213b41a: PASS, no blocking findings"

tasks:
  - id: "8.BH.1"
    name: "Promote Qianfan to dynamic OpenAI-compatible runtime routing"
    status: DONE
    agent: "orchestrator"
    files_owned:
      - internal/provider/oauth/qianfan.go
      - internal/provider/oauth/qianfan_test.go
      - internal/providers/types.go
      - internal/providers/openaicompat/registry.go
      - internal/providers/openaicompat/provider_test.go
      - internal/provider/matrix.go
      - internal/provider/matrix_test.go
      - internal/proxy/engine.go
      - internal/proxy/engine_test.go
      - internal/cli/auth.go
      - internal/cli/provider_runtime.go
      - internal/cli/providers_test.go
      - internal/cli/root_test.go
      - api/handlers/providers_test.go
      - docs/PROVIDERS.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8BH-evaluator-prompt.md
```

**Checkpoint**: Wave 8.BH removes one concrete `unsupported` provider gap. Qianfan now has direct API-key credential capture, registers the OpenAI-compatible Qianfan adapter, and routes provider-qualified models such as `qianfan/deepseek-v3.1-250821`. Qianfan remains intentionally non-catalog and non-quota to avoid fake pricing or unsupported quota claims.

---

### Wave 8.BI — Cloudflare AI Gateway Dynamic Runtime Routing

```yaml
wave: "8.BI"
status: DONE
max_agents: 1
gate: "go test ./internal/providers ./internal/providers/cloudflare ./internal/provider ./internal/proxy ./internal/cli ./api/handlers -run 'TestKeyCarriesProviderAccountID|TestChatCompletionUsesAccountScopedCloudflareOpenAIEndpoint|TestChatCompletionRequiresAccountID|TestProviderMatrixMarksDeploymentDefinedAdaptersAsDynamicPublicRoutes|TestPublicInferenceProvidersExcludeUnsupportedAndAuthOnlyEntries|TestPublicProvidersDoNotClaimQuotaSupport|TestOpenAICompatibleGatewayProvidersUseDynamicPublicRoutesWithoutFakeCatalogs|TestDispatchUsesProviderQualifiedDynamicRouteForDeploymentDefinedProviders|TestProvidersListShowsKnownProviders|TestProvidersListShowsSupportedInferenceProvidersOnly|TestProvidersListKnownProviders' -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build && git diff --check"
completed_at: "2026-06-04T23:28:00Z"
evaluator_prompt: "docs/evaluations/wave-8BI-evaluator-prompt.md"
evaluation: "FAIL external evaluator thread 019e94bb-a183-7523-a8cb-8b9176e2d7b3; runtime provider surface passed, dashboard account_id blocker remediated by Wave 8.BJ commit f9fd56b"
gate_results:
  - "focused provider/API/CLI tests: RED before implementation, providers.Key lacked AccountID, Cloudflare provider package was missing, and Cloudflare remained unsupported"
  - "focused provider/API/CLI tests: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS, 20 files and 85 tests"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS, 23 tests passed and 1 real-server mobile skip"
  - "make build: PASS"
  - "git diff --check: PASS"
  - "external evaluator 019e94bb-a183-7523-a8cb-8b9176e2d7b3: FAIL, UI account_id blocker fixed by Wave 8.BJ"

tasks:
  - id: "8.BI.1"
    name: "Promote Cloudflare AI Gateway to account-scoped dynamic runtime routing"
    status: DONE
    agent: "orchestrator"
    files_owned:
      - internal/providers/types.go
      - internal/providers/types_test.go
      - internal/providers/cloudflare/cloudflare.go
      - internal/providers/cloudflare/cloudflare_test.go
      - internal/provider/matrix.go
      - internal/provider/matrix_test.go
      - internal/proxy/engine.go
      - internal/proxy/engine_test.go
      - internal/cli/provider_runtime.go
      - internal/cli/providers_test.go
      - internal/cli/root_test.go
      - api/handlers/providers_test.go
      - docs/PROVIDERS.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8BI-evaluator-prompt.md
```

**Checkpoint**: Wave 8.BI removes one concrete `unsupported` gateway provider gap. Cloudflare AI Gateway now has a native account-scoped adapter that delegates to the shared OpenAI-compatible runtime at `/accounts/{account_id}/ai/v1/chat/completions`, propagates stored connection `account_id` into provider keys, and routes provider-qualified models such as `cloudflare-ai-gateway/openai/gpt-4.1`. Cloudflare remains intentionally non-catalog, non-listing, and non-quota until those public contracts are implemented.

---

### Wave 8.BJ — Dashboard Connection Account Metadata

```yaml
wave: "8.BJ"
status: DONE
max_agents: 1
gate: "npm --prefix ui test -- --run ProvidersPage.test.tsx -t 'creates Cloudflare AI Gateway connections with account ID metadata' && npm --prefix ui test -- --run ProvidersPage.test.tsx && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && make build && git diff --check"
completed_at: "2026-06-04T23:34:00Z"
evaluator_prompt: "docs/evaluations/wave-8BJ-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e94c3-e3d8-7ea1-8e66-c7ffcf624620 at commit f9fd56b"
gate_results:
  - "npm --prefix ui test -- --run ProvidersPage.test.tsx -t 'creates Cloudflare AI Gateway connections with account ID metadata': RED before implementation, Cloudflare account ID input was missing"
  - "npm --prefix ui test -- --run ProvidersPage.test.tsx -t 'creates Cloudflare AI Gateway connections with account ID metadata': PASS"
  - "npm --prefix ui test -- --run ProvidersPage.test.tsx -t 'Cloudflare': PASS, 2 tests"
  - "npm --prefix ui test -- --run ProvidersPage.test.tsx: PASS, 14 tests"
  - "npm --prefix ui test -- --run: PASS, 20 files and 87 tests"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS, 23 tests passed and 1 real-server mobile skip"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "make build: PASS"
  - "git diff --check: PASS"
  - "external evaluator 019e94c3-e3d8-7ea1-8e66-c7ffcf624620: PASS, no blocking findings"

tasks:
  - id: "8.BJ.1"
    name: "Expose Cloudflare account ID in dashboard connection creation"
    status: DONE
    agent: "orchestrator"
    files_owned:
      - ui/src/api.ts
      - ui/src/pages/ProvidersPage.tsx
      - ui/src/pages/ProvidersPage.test.tsx
      - ui/dist/assets/index.css
      - ui/dist/assets/index.js
      - docs/PLAN.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8BJ-evaluator-prompt.md
```

**Checkpoint**: Wave 8.BJ closes the dashboard management gap introduced by account-scoped Cloudflare routing. The provider connection form now shows a Cloudflare account ID field only for `cloudflare-ai-gateway`, requires it before submission, sends it as `account_id`, clears it after creation/provider changes, and continues to avoid rendering provider credentials.

---

### Wave 8.BK — Kimi Dynamic Runtime Routing

```yaml
wave: "8.BK"
status: DONE
max_agents: 1
gate: "go test ./internal/provider ./internal/providers/openaicompat ./internal/proxy ./internal/cli ./api/handlers -run 'TestProviderMatrixMarksAuthOnlyProvidersExplicitly|TestProviderMatrixMarksDeploymentDefinedAdaptersAsDynamicPublicRoutes|TestPublicInferenceProvidersExcludeUnsupportedAndAuthOnlyEntries|TestPublicProvidersDoNotClaimQuotaSupport|TestOpenAICompatibleGatewayProvidersUseDynamicPublicRoutesWithoutFakeCatalogs|TestConfiguredProvidersUseOpenAICompatibleEndpoints|TestDefaultConfigsAreRegistered|TestDispatchUsesProviderQualifiedDynamicRouteForDeploymentDefinedProviders|TestProvidersListKnownProviders|TestProvidersListShowsKnownProviders|TestProvidersListShowsSupportedInferenceProvidersOnly' -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build && git diff --check"
completed_at: "2026-06-05T00:05:00Z"
evaluator_prompt: "docs/evaluations/wave-8BK-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e94cd-fa4e-7422-b488-d74336a913ac at commit c33a578; implementation and gates passed, workflow pending metadata remediated by follow-up record commit"
gate_results:
  - "focused provider/API/CLI tests: RED before implementation, ProviderKimi was undefined and Kimi remained auth_only"
  - "focused provider/API/CLI tests: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS, 20 files and 87 tests"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS, 23 tests passed and 1 real-server mobile skip"
  - "make build: PASS"
  - "git diff --check: PASS"
  - "external evaluator 019e94cd-fa4e-7422-b488-d74336a913ac: implementation and gates PASS; no code blockers"

tasks:
  - id: "8.BK.1"
    name: "Promote Kimi to OpenAI-compatible provider-qualified runtime routing"
    status: DONE
    agent: "orchestrator"
    files_owned:
      - internal/providers/types.go
      - internal/providers/openaicompat/registry.go
      - internal/providers/openaicompat/provider_test.go
      - internal/provider/matrix.go
      - internal/provider/matrix_test.go
      - internal/proxy/engine.go
      - internal/proxy/engine_test.go
      - internal/cli/provider_runtime.go
      - internal/cli/providers_test.go
      - internal/cli/root_test.go
      - api/handlers/providers_test.go
      - docs/PROVIDERS.md
      - docs/PLAN.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8BK-evaluator-prompt.md
```

**Checkpoint**: Wave 8.BK removes the Kimi `auth_only` runtime gap. Kimi now registers through the shared OpenAI-compatible adapter at `https://api.moonshot.ai/v1`, appears in public provider/API/CLI surfaces, and routes provider-qualified models such as `kimi/kimi-k2.6` upstream as `kimi-k2.6`. Static catalog, embedded pricing, and quota fetchers remain intentionally absent until those contracts are implemented.

---

### Wave 8.BL — Xiaomi Anthropic-Compatible Runtime Routing

```yaml
wave: "8.BL"
status: DONE
max_agents: 1
gate: "go test ./internal/providers/xiaomi ./internal/provider ./internal/proxy ./internal/cli -run 'TestProviderRoutesStandardKeysToXiaomiAnthropicEndpoint|TestProviderRoutesTokenPlanKeysToTokenPlanEndpoint|TestProviderMatrixMarksDeploymentDefinedAdaptersAsDynamicPublicRoutes|TestPublicInferenceProvidersExcludeUnsupportedAndAuthOnlyEntries|TestPublicProvidersDoNotClaimQuotaSupport|TestDispatchUsesProviderQualifiedDynamicRouteForDeploymentDefinedProviders|TestProvidersListShowsKnownProviders|TestProvidersListShowsSupportedInferenceProvidersOnly' -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build && git diff --check"
completed_at: "2026-06-05T00:49:00Z"
evaluator_prompt: "docs/evaluations/wave-8BL-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e94df-ce53-7bf1-aed2-ff270ccba771 at commit c11b8e4"
gate_results:
  - "focused Xiaomi provider/matrix/proxy/CLI tests: RED before implementation, xiaomi provider package was undefined, matrix remained auth_only, dynamic route returned provider not found, and CLI did not list xiaomi"
  - "focused Xiaomi provider/matrix/proxy/CLI tests: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS, 20 files and 87 tests"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS, 23 tests passed and 1 real-server mobile skip"
  - "make build: PASS"
  - "git diff --check: PASS"
  - "external evaluator 019e94df-ce53-7bf1-aed2-ff270ccba771: PASS, no blocking findings; workflow pending marker replaced by this record commit"

tasks:
  - id: "8.BL.1"
    name: "Promote Xiaomi to Anthropic-compatible dynamic runtime routing"
    status: DONE
    agent: "orchestrator"
    files_owned:
      - internal/providers/types.go
      - internal/providers/anthropic/anthropic.go
      - internal/providers/xiaomi/xiaomi.go
      - internal/providers/xiaomi/xiaomi_test.go
      - internal/provider/matrix.go
      - internal/provider/matrix_test.go
      - internal/proxy/engine.go
      - internal/proxy/engine_test.go
      - internal/cli/provider_runtime.go
      - internal/cli/providers_test.go
      - internal/cli/root_test.go
      - docs/PROVIDERS.md
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8BL-evaluator-prompt.md
```

**Checkpoint**: Wave 8.BL removes the Xiaomi `auth_only` runtime gap. Xiaomi now registers as an Anthropic-compatible runtime provider, routes provider-qualified models such as `xiaomi/claude-sonnet-4` upstream as `claude-sonnet-4`, and selects the token-plan endpoint for `tp-` keys. Static catalog, model listing, embedded pricing, and quota fetchers remain intentionally absent until those contracts are implemented.

---

### Wave 8.BM — OpenCode Zen Dynamic Runtime Routing

```yaml
wave: "8.BM"
status: DONE
max_agents: 1
gate: "go test ./internal/providers/openaicompat ./internal/provider ./internal/proxy ./internal/cli ./api/handlers -run 'TestOpenCodeDefaultConfigUsesZenOpenAICompatibleEndpoint|TestConfiguredProvidersUseOpenAICompatibleEndpoints|TestDefaultConfigsAreRegistered|TestProviderMatrixMarksDeploymentDefinedAdaptersAsDynamicPublicRoutes|TestPublicInferenceProvidersExcludeUnsupportedAndAuthOnlyEntries|TestPublicProvidersDoNotClaimQuotaSupport|TestDeploymentDefinedPublicProvidersExposeDynamicRouting|TestDispatchUsesProviderQualifiedDynamicRouteForDeploymentDefinedProviders|TestProvidersListShowsKnownProviders|TestProvidersListKnownProviders' -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build && git diff --check"
completed_at: "2026-06-05T01:10:00Z"
evaluator_prompt: "docs/evaluations/wave-8BM-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e94ed-0635-7652-ac17-fb067e107e68 at commit c5f59d0"
gate_results:
  - "focused OpenCode provider/matrix/proxy/CLI/API tests: RED before implementation, ProviderOpenCode was undefined, matrix remained unsupported, public lists omitted opencode, and dynamic route support was absent"
  - "focused OpenCode provider/matrix/proxy/CLI/API tests: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS, 20 files and 87 tests"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS, 23 tests passed and 1 real-server mobile skip"
  - "make build: PASS"
  - "git diff --check: PASS"
  - "external evaluator 019e94ed-0635-7652-ac17-fb067e107e68: PASS, no blocking or non-blocking findings; workflow pending marker replaced by this record commit"

tasks:
  - id: "8.BM.1"
    name: "Promote OpenCode Zen to OpenAI-compatible dynamic runtime routing"
    status: DONE
    agent: "orchestrator"
    files_owned:
      - internal/providers/types.go
      - internal/providers/openaicompat/registry.go
      - internal/providers/openaicompat/provider_test.go
      - internal/provider/matrix.go
      - internal/provider/matrix_test.go
      - internal/proxy/engine.go
      - internal/proxy/engine_test.go
      - internal/cli/provider_runtime.go
      - internal/cli/providers_test.go
      - api/handlers/providers_test.go
      - docs/PROVIDERS.md
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8BM-evaluator-prompt.md
```

**Checkpoint**: Wave 8.BM removes the OpenCode `unsupported` runtime gap for OpenCode Zen only. OpenCode now registers through the shared OpenAI-compatible adapter at `https://opencode.ai/zen/v1`, appears in public provider/API/CLI surfaces, and routes provider-qualified models such as `opencode/anthropic/claude-sonnet-4` upstream as `anthropic/claude-sonnet-4`. OpenCode Go remains explicitly not wired, and static catalog, model listing, embedded pricing, and quota fetchers remain intentionally absent until those contracts are implemented.

---

### Wave 8.BN — Kilo Gateway Dynamic Runtime Routing

```yaml
wave: "8.BN"
status: DONE
max_agents: 1
gate: "go test ./internal/providers/openaicompat ./internal/provider ./internal/proxy ./internal/cli ./api/handlers -run 'TestKiloDefaultConfigUsesGatewayEndpoint|TestConfiguredProvidersUseOpenAICompatibleEndpoints|TestDefaultConfigsAreRegistered|TestProviderMatrixMarksDeploymentDefinedAdaptersAsDynamicPublicRoutes|TestPublicInferenceProvidersExcludeUnsupportedAndAuthOnlyEntries|TestPublicProvidersDoNotClaimQuotaSupport|TestDeploymentDefinedPublicProvidersExposeDynamicRouting|TestDispatchUsesProviderQualifiedDynamicRouteForDeploymentDefinedProviders|TestProvidersListShowsKnownProviders|TestProvidersListShowsSupportedProvidersOnly|TestProvidersListKnownProviders|TestProvidersLoginListsSupportedProvidersOnly|TestDefaultInferenceEngineRegistersKiloProvider' -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build && git diff --check"
completed_at: "2026-06-04T23:26:02Z"
evaluator_prompt: "docs/evaluations/wave-8BN-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e94fa-cb88-7520-8170-8c5bea4dfc86 at commit 9776094"
gate_results:
  - "focused Kilo provider/matrix/proxy/CLI/API tests: RED before implementation, ProviderKilo was undefined, matrix remained unsupported, public lists omitted kilo, and dynamic route support was absent"
  - "focused Kilo provider/matrix/proxy/CLI/API tests: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS, 20 files and 87 tests"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS, 23 tests passed and 1 real-server mobile skip"
  - "make build: PASS"
  - "git diff --check: PASS"
  - "external evaluator 019e94fa-cb88-7520-8170-8c5bea4dfc86: PASS, no blocking findings; workflow pending marker replaced by this record commit"

tasks:
  - id: "8.BN.1"
    name: "Promote Kilo Gateway to OpenAI-compatible dynamic runtime routing"
    status: DONE
    agent: "orchestrator after subagent 019e94ee-2178-7362-884b-e703af0c10f5 stalled before docs/full gates"
    files_owned:
      - internal/providers/types.go
      - internal/providers/openaicompat/registry.go
      - internal/providers/openaicompat/provider_test.go
      - internal/provider/matrix.go
      - internal/provider/matrix_test.go
      - internal/proxy/engine.go
      - internal/proxy/engine_test.go
      - internal/cli/provider_runtime.go
      - internal/cli/providers_test.go
      - internal/cli/root_test.go
      - api/handlers/providers_test.go
      - docs/PROVIDERS.md
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8BN-evaluator-prompt.md
```

**Checkpoint**: Wave 8.BN removes the Kilo `unsupported` runtime gap. Kilo now registers through the shared OpenAI-compatible adapter at `https://api.kilo.ai/api/gateway`, appears in public provider/API/CLI surfaces, and routes provider-qualified models such as `kilo/anthropic/claude-sonnet-4.5` upstream as `anthropic/claude-sonnet-4.5`. Kiro remains a distinct auth-only provider, and static catalog, model listing, embedded pricing, and quota fetchers remain intentionally absent until those contracts are implemented.

---

### Wave 8.BO — Kagi and Tavily Search Credential Parity

```yaml
wave: "8.BO"
status: DONE
max_agents: 1
gate: "go test ./internal/provider ./internal/cli ./api/handlers -run 'TestProviderMatrixMarksSearchCredentialsAuthOnly|TestAuthListShowsSupportedProviders|TestLoginCommandPersistsSearchProviderAPIKeyConnection|TestProvidersListKnownProviders' -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build && git diff --check"
completed_at: "2026-06-04T23:58:00Z"
evaluator_prompt: "docs/evaluations/wave-8BO-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e9508-1b01-72f0-b14b-2a0d1a64a739 at commit 6685aae"
gate_results:
  - "focused Kagi/Tavily provider/API/CLI tests: RED before implementation, matrix status was unsupported, auth list omitted kagi/tavily, API-key login rejected them, and provider API reported unsupported"
  - "focused Kagi/Tavily provider/API/CLI tests: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS, 20 files and 87 tests"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS, 23 tests passed and 1 real-server mobile skip"
  - "make build: PASS"
  - "git diff --check: PASS"
  - "external evaluator 019e9508-1b01-72f0-b14b-2a0d1a64a739: PASS, no blocking findings; workflow pending marker replaced by Wave 8.BP docs update"

tasks:
  - id: "8.BO.1"
    name: "Promote Kagi and Tavily to API-key auth-only search providers"
    status: DONE
    agent: "orchestrator"
    files_owned:
      - internal/provider/matrix.go
      - internal/provider/matrix_test.go
      - internal/cli/auth_test.go
      - api/handlers/providers_test.go
      - docs/PROVIDERS.md
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8BO-evaluator-prompt.md
```

**Checkpoint**: Wave 8.BO removes the Kagi and Tavily `unsupported` credential gap only. They now appear in auth-capable provider surfaces and accept stored API-key connections. Wave 8.BW later uses those credentials for built-in MCP search tools while preserving `auth_only` provider status: no inference adapter, public dispatch, static catalog, model listing, streaming, pricing, or quota support is advertised.

---

### Wave 8.BP — Ollama Cloud Native Runtime Routing

```yaml
wave: "8.BP"
status: DONE
max_agents: 1
gate: "go test ./internal/providers/ollamacloud ./internal/provider ./internal/cli ./api/handlers ./internal/proxy -run 'Test(ChatCompletionUsesNativeOllamaCloudChat|ListModelsUsesNativeTagsEndpoint|NewDefaultUsesOllamaCloudProvider|OllamaCloudPublicNativeProvider|ProviderMatrixMarksDeploymentDefinedAdaptersAsDynamicPublicRoutes|PublicInferenceProvidersExcludeUnsupportedAndAuthOnlyEntries|PublicProvidersDoNotClaimQuotaSupport|ProvidersListShowsKnownProviders|DefaultInferenceEngineRegistersOllamaCloudProvider|ProvidersListKnownProviders|DispatchUsesProviderQualifiedDynamicRouteForDeploymentDefinedProviders)' -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build && git diff --check"
completed_at: "2026-06-05T01:28:00Z"
evaluator_prompt: "docs/evaluations/wave-8BP-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e9515-e2e4-7f93-a1ee-7293925a05ed at commit 0fc9797"
gate_results:
  - "focused Ollama Cloud provider/matrix/proxy/CLI/API tests: RED before implementation, ollama-cloud provider package was missing, matrix remained unsupported, runtime registration omitted it, public lists omitted it, and provider-qualified dispatch did not route it"
  - "focused Ollama Cloud provider/matrix/proxy/CLI/API tests: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS, 20 files and 87 tests"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS, 23 tests passed and 1 real-server mobile skip"
  - "make build: PASS"
  - "git diff --check: PASS"
  - "external evaluator 019e9515-e2e4-7f93-a1ee-7293925a05ed: PASS, no blocking findings; workflow pending marker replaced by this record commit"

tasks:
  - id: "8.BP.1"
    name: "Promote Ollama Cloud to native provider-qualified runtime routing"
    status: DONE
    agent: "orchestrator"
    files_owned:
      - internal/providers/types.go
      - internal/providers/ollamacloud/ollamacloud.go
      - internal/providers/ollamacloud/ollamacloud_test.go
      - internal/provider/matrix.go
      - internal/provider/matrix_test.go
      - internal/proxy/engine.go
      - internal/proxy/engine_test.go
      - internal/cli/provider_runtime.go
      - internal/cli/providers_test.go
      - internal/cli/root_test.go
      - api/handlers/providers_test.go
      - docs/PROVIDERS.md
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8BP-evaluator-prompt.md
```

**Checkpoint**: Wave 8.BP removes the Ollama Cloud `unsupported` runtime gap with a native Ollama `/api/chat` and `/api/tags` provider. Ollama Cloud now accepts API-key credentials, appears in public provider/API/CLI surfaces, registers in normal server startup, and routes provider-qualified models such as `ollama-cloud/gpt-oss:120b` upstream as `gpt-oss:120b`. Static catalog, pricing, and quota fetchers remain intentionally absent.

---

### Wave 8.BQ — GitLab Duo Auth Identity Normalization

```yaml
wave: "8.BQ"
status: DONE
max_agents: 1
gate: "go test ./api/handlers ./internal/provider ./internal/provider/oauth ./internal/cli -run 'Test(OAuthExchangeAcceptsGitLabAliasAndStoresGitLabDuoConnection|CanonicalProviderIDNormalizesRuntimeAliases|ProviderAliasesIncludeLegacyIDs|CanonicalFlowProviderIDNormalizesAuthAliases|CanonicalProviderIDKeepsVertexRuntimeProvider|GitLabFlowStartBuildsPKCEAuthURL|GitLabFlowExchangePostsAuthorizationCode|GitLabFlowPollUnsupported|ConnectionFromOAuthTokenNormalizesGitLabToGitLabDuo|ProviderMatrixCoversRemediationParityTiers|ProviderMatrixMarksOAuthOnlyProvidersExplicitly|PublicInferenceProvidersExcludeUnsupportedAndAuthOnlyEntries|AuthListShowsSupportedProviders|OAuthFlowAcceptsCanonicalProviderAliases)' -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build && git diff --check"
completed_at: "2026-06-05T02:00:00Z"
evaluator_prompt: "docs/evaluations/wave-8BQ-evaluator-prompt.md"
evaluation: "PASS external evaluator subagent 019e952a-7b90-7120-bd6b-b54ccb3eff27 at commit 38c2ab7"
gate_results:
  - "focused GitLab Duo provider/API/CLI/OAuth tests: RED before implementation, gitlab did not canonicalize to gitlab-duo, GitLab OAuth emitted provider gitlab, matrix omitted gitlab-duo, auth list showed gitlab, and /api/oauth/gitlab exchange did not have dedicated canonical persistence coverage"
  - "focused GitLab Duo provider/API/CLI/OAuth tests: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS, 20 files and 87 tests"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS, 23 tests passed and 1 real-server mobile skip"
  - "make build: PASS"
  - "git diff --check: PASS"
  - "external evaluator 019e952a-7b90-7120-bd6b-b54ccb3eff27: PASS, no blocking findings; workflow pending marker replaced by this record commit"

tasks:
  - id: "8.BQ.1"
    name: "Normalize GitLab OAuth identity to gitlab-duo"
    status: DONE
    agent: "orchestrator"
    files_owned:
      - api/handlers/oauth_test.go
      - internal/provider/ids.go
      - internal/provider/ids_test.go
      - internal/provider/credentials_test.go
      - internal/provider/matrix.go
      - internal/provider/matrix_test.go
      - internal/provider/oauth/types.go
      - internal/provider/oauth/types_test.go
      - internal/provider/oauth/gitlab.go
      - internal/provider/oauth/gitlab_test.go
      - internal/cli/auth.go
      - internal/cli/auth_test.go
      - docs/PROVIDERS.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8BQ-evaluator-prompt.md
```

**Checkpoint**: Wave 8.BQ establishes `gitlab-duo` as the canonical g0router provider ID for GitLab before runtime work. Legacy `gitlab` auth/API aliases normalize to `gitlab-duo`, GitLab OAuth uses bundled client defaults, scope `api`, and callback `http://localhost:8080/callback`, and persisted OAuth connections use runtime provider `gitlab-duo`. GitLab Duo remains `auth_only`; the direct-access token and GitLab AI Gateway runtime adapter are intentionally deferred.

---

### Wave 8.BR — Replicate Runtime Truthfulness

```yaml
wave: "8.BR"
status: DONE
max_agents: 1
gate: "go test ./internal/provider ./api/handlers ./internal/cli -run 'Test(ReplicateRemainsAuthOnlyUntilPredictionRuntimeIsImplemented|ProvidersListKnownProviders|AuthListShowsSupportedProviders|LoginCommandPersistsSearchProviderAPIKeyConnection|ProvidersTestReportsAuthOnlyProvider)' -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build && git diff --check"
completed_at: "2026-06-05T02:35:00Z"
evaluator_prompt: "docs/evaluations/wave-8BR-evaluator-prompt.md"
evaluation: "PASS external evaluator subagent 019e9531-47a8-7903-9971-981e280fe605 at commit ed78ce9"
gate_results:
  - "focused Replicate provider/API/CLI tests: RED before implementation, matrix reported adapter_only, API reported a registered inference adapter, and providers test returned adapter_only instead of auth_only"
  - "focused Replicate provider/API/CLI tests: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS, 20 files and 87 tests"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS, 23 tests passed and 1 real-server mobile skip"
  - "make build: PASS"
  - "git diff --check: PASS"
  - "external evaluator 019e9531-47a8-7903-9971-981e280fe605: PASS, no blocking findings; workflow pending marker replaced by this record commit"
  - "direct CLI spot checks: auth list includes replicate, providers list excludes replicate, providers test replicate reports auth_only"

tasks:
  - id: "8.BR.1"
    name: "Demote Replicate to API-key auth-only until prediction runtime exists"
    status: DONE
    agent: "orchestrator"
    files_owned:
      - api/handlers/providers_test.go
      - internal/cli/auth_test.go
      - internal/cli/provider_runtime.go
      - internal/cli/providers_test.go
      - internal/provider/matrix.go
      - internal/provider/matrix_test.go
      - internal/providers/replicate/replicate.go
      - internal/providers/replicate/replicate_test.go
      - docs/PROVIDERS.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8BR-evaluator-prompt.md
```

**Checkpoint**: Wave 8.BR removes the unproven Replicate OpenAI-compatible wrapper from normal startup and marks Replicate as API-key `auth_only`. This keeps credential capture available while refusing to advertise adapter, inference, streaming, listing, catalog, quota, or public dispatch support until a real Replicate prediction-backed runtime is implemented and tested against local fake prediction endpoints.

---

### Wave 8.BS — GitLab Duo Runtime Dispatch

```yaml
wave: "8.BS"
status: DONE
max_agents: 1
gate: "go test ./internal/providers/anthropic ./internal/providers/gitlabduo ./internal/provider ./internal/cli ./api/handlers -run 'Test(NewForProviderWithHeadersAddsProviderHeaders|ChatCompletionExchangesDirectAccessAndRoutesOpenAIModel|ChatCompletionRoutesAnthropicModelWithDirectAccessHeaders|ChatCompletionCachesDirectAccessToken|ChatCompletionRejectsUnsupportedModel|ListModelsReturnsDuoAliasesDeterministically|GitLabDuoPublicDynamicProvider|ProviderMatrixMarksDeploymentDefinedAdaptersAsDynamicPublicRoutes|PublicInferenceProvidersExcludeUnsupportedAndAuthOnlyEntries|ProvidersListShowsKnownProviders|ProvidersListKnownProviders)' -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build && git diff --check"
completed_at: "2026-06-05T02:55:00Z"
evaluator_prompt: "docs/evaluations/wave-8BS-evaluator-prompt.md"
evaluation: "PASS external evaluator subagent 019e9545-3401-7ea0-b7e6-76685927b8ba at commit 619e704"
gate_results:
  - "focused GitLab Duo runtime tests: RED before implementation, provider package and ProviderGitLabDuo constant did not exist, Anthropic provider could not inject GitLab direct-access headers, matrix/API/CLI still treated gitlab-duo as auth_only, and provider-qualified routing did not accept gitlab-duo"
  - "focused GitLab Duo runtime tests: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS, 20 files and 87 tests"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS, 23 tests passed and 1 real-server mobile skip"
  - "make build: PASS"
  - "git diff --check: PASS"
  - "external evaluator 019e9545-3401-7ea0-b7e6-76685927b8ba: PASS, no blocking findings; non-blocking note to make the Duo alias table immutable-by-convention later"

tasks:
  - id: "8.BS.1"
    name: "Promote GitLab Duo to provider-qualified runtime dispatch"
    status: DONE
    agent: "orchestrator"
    files_owned:
      - api/handlers/providers_test.go
      - internal/cli/provider_runtime.go
      - internal/cli/providers_test.go
      - internal/provider/matrix.go
      - internal/provider/matrix_test.go
      - internal/providers/anthropic/anthropic.go
      - internal/providers/anthropic/anthropic_test.go
      - internal/providers/gitlabduo/gitlabduo.go
      - internal/providers/gitlabduo/gitlabduo_test.go
      - internal/providers/types.go
      - internal/proxy/engine.go
      - docs/PROVIDERS.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8BS-evaluator-prompt.md
```

**Checkpoint**: Wave 8.BS turns canonical `gitlab-duo` credentials into real provider-qualified dispatch. The runtime exchanges stored GitLab OAuth tokens for GitLab Duo direct-access tokens, caches those direct-access credentials briefly, forwards GitLab-provided gateway headers, maps Duo aliases to GitLab AI Gateway OpenAI and Anthropic proxy models, exposes deterministic Duo alias model listing, and advertises `gitlab-duo` through provider matrix, CLI, API, and normal server startup. Static priced catalog and quota fetchers remain intentionally absent.

---

### Wave 8.BT — GitLab Duo Alias Table Hardening

```yaml
wave: "8.BT"
status: DONE
max_agents: 1
gate: "go test ./internal/providers/gitlabduo -run 'Test(MappedRequestUsesFixedAliasTable|ListModelsReturnsDuoAliasesDeterministically)' -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build && git diff --check"
completed_at: "2026-06-05T03:15:00Z"
evaluator_prompt: "docs/evaluations/wave-8BT-evaluator-prompt.md"
evaluation: "PASS external evaluator subagent 019e954c-2b8c-7ae0-9b8c-f09409b6c50b at commit 278cf2a"
gate_results:
  - "focused GitLab Duo alias table test: RED before implementation, mutating the package-level modelMappings map changed routing for duo-chat-gpt-5-1"
  - "focused GitLab Duo alias table test: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS, 20 files and 87 tests"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS, 23 tests passed and 1 real-server mobile skip"
  - "make build: PASS"
  - "git diff --check: PASS"
  - "external evaluator 019e954c-2b8c-7ae0-9b8c-f09409b6c50b: PASS, no blocking findings; noted modelAliases remains an unexported package var array but no mutable map or production mutator remains"

tasks:
  - id: "8.BT.1"
    name: "Replace GitLab Duo mutable alias map with a fixed alias table"
    status: DONE
    agent: "orchestrator"
    files_owned:
      - internal/providers/gitlabduo/gitlabduo.go
      - internal/providers/gitlabduo/gitlabduo_test.go
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8BT-evaluator-prompt.md
```

**Checkpoint**: Wave 8.BT resolves the Wave 8.BS evaluator's non-blocking mutable-map finding. GitLab Duo model aliases are held in a fixed alias table, routing scans that table instead of reading from a mutable map, model listing still returns deterministic aliases, and direct-access runtime behavior remains unchanged.

---

### Wave 8.BU — Replicate Prediction Runtime

```yaml
wave: "8.BU"
status: DONE
max_agents: 1
gate: "go test ./internal/providers/replicate ./internal/proxy ./internal/provider ./api/handlers ./internal/cli -run 'Test(ChatCompletionCreatesAndPollsPrediction|ChatCompletionMapsStringPredictionOutput|ChatCompletionReportsFailedPrediction|ChatCompletionTimesOutPendingPrediction|ChatCompletionStreamUnsupported|ListModelsUnsupported|DispatchUsesProviderQualifiedDynamicRouteForDeploymentDefinedProviders|ReplicatePromotesToPredictionBackedInferenceProvider|ProvidersListShowsKnownProviders|ProvidersTestRequiresActiveConnectionForCredentialProvider|ProvidersListKnownProviders)' -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build && git diff --check"
completed_at: "2026-06-05T03:35:00Z"
evaluator_prompt: "docs/evaluations/wave-8BU-evaluator-prompt.md"
evaluation: "PASS external evaluator subagent 019e955b-0964-75d1-93fd-caccb5e1ae78 at commit 763e2af"
gate_results:
  - "focused Replicate runtime tests: RED before implementation, replicate provider package was missing, provider-qualified route returned provider not found, matrix/API/CLI still treated replicate as auth_only"
  - "focused Replicate runtime tests: PASS"
  - "go test ./... -count=1: PASS after updating stale root/provider public-list negative tests"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS, 20 files and 87 tests"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS, 23 tests passed and 1 real-server mobile skip"
  - "make build: PASS"
  - "git diff --check: PASS"
  - "external evaluator 019e955b-0964-75d1-93fd-caccb5e1ae78: PASS, no blocking findings; non-blocking note to add a dedicated default-engine registration test for Replicate later"

tasks:
  - id: "8.BU.1"
    name: "Promote Replicate to non-streaming prediction-backed provider-qualified dispatch"
    status: DONE
    agent: "orchestrator"
    files_owned:
      - api/handlers/providers_test.go
      - internal/cli/provider_runtime.go
      - internal/cli/providers_test.go
      - internal/provider/matrix.go
      - internal/provider/matrix_test.go
      - internal/providers/replicate/replicate.go
      - internal/providers/replicate/replicate_test.go
      - internal/proxy/engine.go
      - internal/proxy/engine_test.go
      - docs/PROVIDERS.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8BU-evaluator-prompt.md
```

**Checkpoint**: Wave 8.BU replaces Replicate credential-only status with a real prediction-backed runtime. Provider-qualified models such as `replicate/owner/model` create Replicate predictions, poll to terminal status, and map string outputs into OpenAI-compatible chat responses. Streaming, model listing, static catalog, and quota fetchers remain intentionally unsupported rather than fabricated.

---

### Wave 8.BV — Replicate Startup Registration Guard

```yaml
wave: "8.BV"
status: DONE
max_agents: 1
gate: "go test ./internal/cli -run TestDefaultInferenceEngineRegistersReplicateProvider -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build && git diff --check"
completed_at: "2026-06-05T03:55:00Z"
evaluator_prompt: "docs/evaluations/wave-8BV-evaluator-prompt.md"
evaluation: "PASS external evaluator subagent 019e9560-caf7-72b2-b9ae-5cd6c6bcc88b at commit f8ee844"
gate_results:
  - "go test ./internal/cli -run TestDefaultInferenceEngineRegistersReplicateProvider -count=1: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS, 20 files and 87 tests"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS, 23 tests passed and 1 real-server mobile skip"
  - "make build: PASS"
  - "git diff --check: PASS"
  - "external evaluator 019e9560-caf7-72b2-b9ae-5cd6c6bcc88b: PASS, no blocking findings"

tasks:
  - id: "8.BV.1"
    name: "Add default-engine regression coverage for Replicate startup registration"
    status: DONE
    agent: "orchestrator"
    files_owned:
      - internal/cli/root_test.go
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8BV-evaluator-prompt.md
```

**Checkpoint**: Wave 8.BV resolves the Wave 8.BU evaluator's non-blocking registration-test note. The default inference engine now has direct regression coverage that Replicate remains registered in normal startup.

---

### Wave 8.BW — Kagi and Tavily Built-In MCP Search Tools

```yaml
wave: "8.BW"
status: DONE
max_agents: 2
gate: "go test ./internal/search ./internal/cli ./internal/provider ./api/handlers -run 'Test(KagiSearchTool|TavilySearchTool|SearchToolRequiresActiveAPIKey|SearchToolErrorsAreSanitized|BuiltInSearchTools|DefaultServerConfigRegistersBuiltInSearchTools|ProviderMatrixKeepsSearchProvidersAuthOnly|ProviderMatrixMarksSearchCredentialsAuthOnly)' -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build && git diff --check"
completed_at: "2026-06-05T04:40:00Z"
evaluator_prompt: "docs/evaluations/wave-8BW-evaluator-prompt.md"
evaluation: "PASS external evaluator subagent 019e9571-7c63-73c0-8793-d6f2d5893758 at commit 079a576"
gate_results:
  - "focused internal/search tests: RED before implementation, search package API and startup registration did not exist"
  - "focused internal/search and internal/cli startup registration tests: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS, 20 files and 87 tests"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS, 23 tests passed and 1 real-server mobile skip"
  - "make build: PASS"
  - "git diff --check: PASS"
  - "external evaluator 019e9571-7c63-73c0-8793-d6f2d5893758: PASS, no blocking findings"

tasks:
  - id: "8.BW.1"
    name: "Register stored Kagi and Tavily API keys as built-in MCP search tools"
    status: DONE
    agent: "orchestrator with read-only scout 019e9566-e03c-7101-ada7-74bd7e7a8dd3"
    files_owned:
      - internal/search/search.go
      - internal/search/search_test.go
      - internal/cli/root.go
      - internal/cli/root_test.go
      - internal/provider/matrix.go
      - docs/PROVIDERS.md
      - docs/CONFIG.md
      - docs/SCHEMA.md
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8BW-evaluator-prompt.md
```

**Checkpoint**: Wave 8.BW turns the Wave 8.BO Kagi/Tavily API-key credential capture into usable built-in MCP search tools. Active stored API-key connections register `kagi__search` and `tavily__search` during normal server startup and through the same MCP `ToolManager` surface used by dashboard tool execution and inference tool injection. Kagi and Tavily remain `auth_only` providers: no inference adapter, `/v1` dispatch, `/api/search` route, model catalog, streaming, pricing, or quota support is advertised.

---

### Wave 8.BX — Responses API Streaming Translation

```yaml
wave: "8.BX"
status: DONE
max_agents: 1
gate: "go test ./api/handlers -run 'TestResponsesStreamingTranslatesChatStream|Test(StreamInference|Responses|Inference)' -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build && git diff --check"
completed_at: "2026-06-05T05:10:00Z"
evaluator_prompt: "docs/evaluations/wave-8BX-evaluator-prompt.md"
evaluation: "PASS external evaluator subagent 019e9580-a5ce-7f93-a936-5547d594bf15 at commit 87a4767"
gate_results:
  - "go test ./api/handlers -run TestResponsesStreamingTranslatesChatStream -count=1: RED before implementation, /v1/responses stream:true returned 501 responses streaming unavailable"
  - "go test ./api/handlers -run 'Test(StreamInference|ResponsesStreamingTranslatesChatStream|Responses|Inference)' -count=1: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS, 20 files and 87 tests"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS, 23 tests passed and 1 real-server mobile skip"
  - "make build: PASS"
  - "git diff --check: PASS"
  - "external evaluator 019e9580-a5ce-7f93-a936-5547d594bf15: PASS, no blocking findings; non-blocking coverage note addressed by adding response.output_text.done assertion"

tasks:
  - id: "8.BX.1"
    name: "Translate chat stream chunks into Responses API SSE events"
    status: DONE
    agent: "orchestrator"
    files_owned:
      - api/handlers/inference.go
      - api/handlers/inference_test.go
      - docs/SCHEMA.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8BX-evaluator-prompt.md
```

**Checkpoint**: Wave 8.BX removes the hard 501 for `/v1/responses` streaming when the request can be translated to the existing OpenAI-compatible chat request shape. The handler now dispatches through `DispatchStream`, emits Responses-style SSE events for `response.output_text.delta`, `response.output_text.done`, and `response.completed`, and preserves explicit rejection of unsupported native Responses input items. `/v1/messages` streaming and richer native Responses item support remain separate gaps.

---

### Wave 8.BY — Messages API Streaming Translation

```yaml
wave: "8.BY"
status: DONE
max_agents: 1
gate: "go test ./api/handlers -run 'TestMessagesStreamingTranslatesChatStream|TestMessagesResponsePreservesToolUseBlocks|TestResponsesStreamingTranslatesChatStream|TestStreamInferenceWritesSanitizedStreamError' -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build && git diff --check"
completed_at: "2026-06-05T05:30:00Z"
evaluator_prompt: "docs/evaluations/wave-8BY-evaluator-prompt.md"
evaluation: "PASS external evaluator subagent 019e958c-36dd-71a2-bf94-6e899159d8ef at commit 17f0cb8"
gate_results:
  - "go test ./api/handlers -run TestMessagesStreamingTranslatesChatStream -count=1: RED before implementation, /v1/messages stream:true returned 501 messages streaming unavailable"
  - "go test ./api/handlers -run 'TestMessagesStreamingTranslatesChatStream|TestMessagesResponsePreservesToolUseBlocks|TestResponsesStreamingTranslatesChatStream|TestStreamInferenceWritesSanitizedStreamError' -count=1: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS, 20 files and 87 tests"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS, 23 tests passed and 1 real-server mobile skip"
  - "make build: PASS"
  - "git diff --check: PASS"
  - "external evaluator 019e958c-36dd-71a2-bf94-6e899159d8ef: PASS, no blocking findings"

tasks:
  - id: "8.BY.1"
    name: "Translate chat stream chunks into Anthropic Messages SSE events"
    status: DONE
    agent: "orchestrator"
    files_owned:
      - api/handlers/inference.go
      - api/handlers/inference_test.go
      - docs/SCHEMA.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8BY-evaluator-prompt.md
```

**Checkpoint**: Wave 8.BY removes the hard 501 for `/v1/messages` streaming when the request can be represented by the existing OpenAI-compatible chat request shape. The handler now dispatches through `DispatchStream` and emits Anthropic-style SSE events (`message_start`, `content_block_start`, `content_block_delta`, `content_block_stop`, `message_delta`, and `message_stop`) without an OpenAI `[DONE]` sentinel. Native Anthropic tool input/result blocks remain explicitly rejected before dispatch instead of being silently dropped.

---

### Wave 8.BZ — Unsupported Streaming Error Classification

```yaml
wave: "8.BZ"
status: DONE
max_agents: 1
gate: "go test ./internal/providers/bedrock ./internal/providers/replicate ./api/handlers -run 'TestUnsupportedMethodsReturnErrors|TestChatCompletionStreamUnsupported|TestStreamInferenceUnsupportedProviderUsesStableError' -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build && git diff --check"
completed_at: "2026-06-05T05:50:00Z"
evaluator_prompt: "docs/evaluations/wave-8BZ-evaluator-prompt.md"
evaluation: "PASS external evaluator subagent 019e9597-fb38-7b50-beaa-7b05c999855b at commit b4385f2"
gate_results:
  - "go test ./internal/providers/bedrock ./internal/providers/replicate ./api/handlers -run 'TestUnsupportedMethodsReturnErrors|TestChatCompletionStreamUnsupported|TestStreamInferenceUnsupportedProviderUsesStableError' -count=1: RED before implementation, providers.ErrStreamingUnsupported was undefined"
  - "go test ./internal/providers/bedrock ./internal/providers/replicate ./api/handlers -run 'TestUnsupportedMethodsReturnErrors|TestChatCompletionStreamUnsupported|TestStreamInferenceUnsupportedProviderUsesStableError' -count=1: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS, 20 files and 87 tests"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS, 23 tests passed and 1 real-server mobile skip"
  - "make build: PASS"
  - "git diff --check: PASS"
  - "external evaluator 019e9597-fb38-7b50-beaa-7b05c999855b: FAIL at commit 833204a on stale Stage 8 range in PLAN/ORCHESTRATION; PASS after docs remediation commit b4385f2"

tasks:
  - id: "8.BZ.1"
    name: "Classify provider unsupported streaming through a shared sentinel"
    status: DONE
    agent: "orchestrator with read-only scout 019e9590-53de-7432-a8de-d711fafad853"
    files_owned:
      - api/handlers/inference_test.go
      - internal/providers/types.go
      - internal/providers/bedrock/bedrock.go
      - internal/providers/bedrock/bedrock_test.go
      - internal/providers/replicate/replicate.go
      - internal/providers/replicate/replicate_test.go
      - internal/proxy/errors.go
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8BZ-evaluator-prompt.md
```

**Checkpoint**: Wave 8.BZ keeps Bedrock and Replicate streaming truthfully unsupported while making that unsupported capability machine-readable. Providers now wrap a shared `providers.ErrStreamingUnsupported` sentinel, and dispatch error classification maps it to the stable sanitized OpenAI-compatible `501 streaming_unsupported` response instead of a generic upstream failure.

---

### Wave 8.CA — Quota Capability Truth Hardening

```yaml
wave: "8.CA"
status: DONE
max_agents: 1
gate: "go test ./internal/cli -run TestDefaultQuotaFetchersReturnUnsupportedForQuotaFalseProviders -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build && git diff --check"
completed_at: "2026-06-05T06:10:00Z"
evaluator_prompt: "docs/evaluations/wave-8CA-evaluator-prompt.md"
evaluation: "PASS external evaluator subagent 019e95a3-cdab-78c2-b0d2-267cf8cf64fc at commit 2b774c6"
gate_results:
  - "go test ./internal/cli -run TestDefaultQuotaFetchersReturnUnsupportedForQuotaFalseProviders -count=1: RED before final scope, auth-only providers do not register default quota fetchers"
  - "go test ./internal/cli -run TestDefaultQuotaFetchersReturnUnsupportedForQuotaFalseProviders -count=1: PASS after scoping the contract to public inference providers"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS, 20 files and 87 tests"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS, 23 tests passed and 1 real-server mobile skip"
  - "make build: PASS"
  - "git diff --check: PASS"
  - "external evaluator 019e95a3-cdab-78c2-b0d2-267cf8cf64fc: PASS, no blocking findings"

tasks:
  - id: "8.CA.1"
    name: "Clarify quota endpoint capability gating and default unsupported behavior"
    status: DONE
    agent: "orchestrator with read-only scout 019e9590-82fd-7ea2-8861-b9e6514d6565"
    files_owned:
      - internal/cli/root_test.go
      - docs/SCHEMA.md
      - docs/phases/phase-08-usage-tracking-cost-logging.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8CA-evaluator-prompt.md
```

**Checkpoint**: Wave 8.CA makes the provider quota contract harder to misread. The API schema and Phase 8 docs now state that quota is capability-gated and unsupported by default unless a provider-specific fetcher exists, and startup has regression coverage that every public inference provider with `quota=false` returns `usage.ErrQuotaUnsupported` through its default cached quota fetcher.

---

### Wave 8.CB — OpenRouter Quota Support

```yaml
wave: "8.CB"
status: DONE
max_agents: 1
gate: "go test ./internal/usage ./internal/cli ./internal/provider ./api/handlers ./internal/proxy -run 'TestOpenRouterQuotaFetcher|TestDefaultServerConfigRegistersOpenRouterQuotaFetcher|TestPublicProvidersOnlyClaimImplementedQuotaSupport|TestProvidersListKnownProviders|TestDispatchUnlimitedQuotaAllowsProviderCall' -count=1 && npm --prefix ui test -- --run QuotaPage && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build && git diff --check"
completed_at: "2026-06-05T06:45:00Z"
evaluator_prompt: "docs/evaluations/wave-8CB-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e95b9-0966-7db1-a04c-56f608b366c4 at commit 2327865"
gate_results:
  - "go test ./internal/usage ./internal/cli ./internal/provider ./api/handlers ./internal/proxy -run 'TestOpenRouterQuotaFetcher|TestDefaultServerConfigRegistersOpenRouterQuotaFetcher|TestPublicProvidersOnlyClaimImplementedQuotaSupport|TestProvidersListKnownProviders|TestDispatchUnlimitedQuotaAllowsProviderCall' -count=1: PASS"
  - "npm --prefix ui test -- --run QuotaPage: PASS, 1 file and 7 tests"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS, 20 files and 88 tests"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS, 23 tests passed and 1 real-server mobile skip"
  - "make build: PASS"
  - "git diff --check: PASS"
  - "external evaluator 019e95b9-0966-7db1-a04c-56f608b366c4: PASS, no blocking findings"

tasks:
  - id: "8.CB.1"
    name: "Implement real OpenRouter quota fetcher and unlimited quota UI"
    status: DONE
    agent: "orchestrator with evaluator 019e95b1-75e4-75e2-9efe-9e1378eed041"
    commit: "ee96357"
    files_owned:
      - internal/usage/quota.go
      - internal/usage/quota_test.go
      - internal/cli/root.go
      - internal/cli/root_test.go
      - internal/provider/matrix.go
      - internal/provider/matrix_test.go
      - internal/proxy/engine.go
      - internal/proxy/engine_test.go
      - api/handlers/providers_test.go
      - ui/src/api.ts
      - ui/src/pages/QuotaPage.tsx
      - ui/src/pages/QuotaPage.test.tsx
      - ui/dist/assets/index.js
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/PROVIDERS.md
      - docs/SCHEMA.md
      - docs/phases/phase-08-usage-tracking-cost-logging.md
      - docs/evaluations/wave-8CB-evaluator-prompt.md
      - docs/WORKFLOW.md
```

**Checkpoint**: Wave 8.CB promotes only OpenRouter to quota-capable status. The default server wiring uses the OpenRouter current API key credits endpoint with bearer auth, dashboard quota rendering understands unlimited credit responses, dispatch does not block explicit unlimited quota, and all other providers with `quota=false` remain explicitly unsupported by default fetchers.

---

### Wave 8.CC — Provider Matrix Quota Documentation

```yaml
wave: "8.CC"
status: DONE
max_agents: 1
gate: "go test ./internal/provider -run TestProviderDocsExposeQuotaColumnMatchingMatrix -count=1 && go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && npm --prefix ui run e2e && make build && git diff --check"
completed_at: "2026-06-05T07:05:00Z"
evaluator_prompt: "docs/evaluations/wave-8CC-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e95c4-074e-7d01-9d91-06d1f981eca3 at commit eb4cf81"
gate_results:
  - "go test ./internal/provider -run TestProviderDocsExposeQuotaColumnMatchingMatrix -count=1: FAIL before docs update, missing explicit Quota column"
  - "go test ./internal/provider -run TestProviderDocsExposeQuotaColumnMatchingMatrix -count=1: PASS"
  - "go test ./... -count=1: PASS"
  - "go vet ./...: PASS"
  - "go build ./cmd/g0router: PASS"
  - "npm --prefix ui test -- --run: PASS, 20 files and 88 tests"
  - "npm --prefix ui run build: PASS"
  - "npm --prefix ui run e2e: PASS, 23 tests passed and 1 real-server mobile skip"
  - "make build: PASS"
  - "git diff --check: PASS"
  - "external evaluator 019e95c4-074e-7d01-9d91-06d1f981eca3: PASS, no blocking findings; UI dependencies were bootstrapped with make build before rerunning UI gates"

tasks:
  - id: "8.CC.1"
    name: "Expose quota capability in provider matrix docs"
    status: DONE
    agent: "orchestrator"
    commit: "fd80d7e"
    files_owned:
      - internal/provider/matrix_test.go
      - docs/PROVIDERS.md
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8CC-evaluator-prompt.md
```

**Checkpoint**: Wave 8.CC makes quota capability explicit in the human provider matrix. `docs/PROVIDERS.md` now has a `Quota` column matching `internal/provider/matrix.go`, with OpenRouter as the only `yes` row, and a regression test keeps the docs table aligned with the provider matrix.

---

### Wave 8.CD — Clean-Checkout Release Gate Bootstrap

```yaml
wave: "8.CD"
status: DONE
max_agents: 1
gate: "make verify"
completed_at: "2026-06-05T07:35:00Z"
evaluator_prompt: "docs/evaluations/wave-8CD-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e95d4-470f-7180-9a69-7e118eabecc1 at commit 37ab57d"
gate_results:
  - "clean checkout raw npm --prefix ui test -- --run McpSplitPages before dependency bootstrap: FAIL, vitest command not found"
  - "make build in clean checkout before implementation: PASS, but bootstraps after raw UI gates in historical gate order"
  - "clean checkout npm --prefix ui test -- --run after make build bootstrap: PASS"
  - "npm --prefix ui test -- --run McpSplitPages: PASS"
  - "npm --prefix ui test -- --run: PASS, 20 files and 88 tests"
  - "focused provider/API/UI checks: PASS"
  - "make verify: PASS; bootstrapped UI deps, passed go test, go vet, go build, UI unit/build/E2E, make build, and git diff --check"
  - "git diff --check: PASS"
  - "external evaluator 019e95d4-470f-7180-9a69-7e118eabecc1: PASS, no blocking findings"

tasks:
  - id: "8.CD.1"
    name: "Add bootstrapped clean-checkout release verification target"
    status: DONE
    agent: "orchestrator with audit subagents 019e95c8-6d0c-7c23-be7a-1f4f724b5c09, 019e95c8-a5d7-76b3-b264-59f1a87eeb37, 019e95c8-f512-7fe2-b08b-a7066c1a35f6"
    commit: "82d8485"
    files_owned:
      - Makefile
      - README.md
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8CD-evaluator-prompt.md
```

**Checkpoint**: Wave 8.CD fixes the clean-checkout release gate ordering found by the docs/release audit. `make verify` now installs UI development dependencies before raw UI unit/build/E2E gates, then runs the Go gates, binary build, `make build`, and whitespace check as one documented release verification command.

---

### Wave 8.CE — Dashboard Update Controls And Negative-State Coverage

```yaml
wave: "8.CE"
status: DONE
max_agents: 1
gate: "npm --prefix ui test -- --run src/api.test.ts src/pages/AliasesPage.test.tsx src/pages/CombosPage.test.tsx src/pages/PricingPage.test.tsx src/pages/ProvidersPage.test.tsx src/pages/LogsPage.test.tsx src/pages/DiagnosticsPage.test.tsx && npm --prefix ui run e2e && make verify"
completed_at: "2026-06-05T08:05:00Z"
evaluator_prompt: "docs/evaluations/wave-8CE-evaluator-prompt.md"
evaluation: "PASS replacement external evaluator thread 019e95ea-0676-7761-8373-a07074aaeeaf at commit 87d674c; original evaluator 019e95e3-eb96-7f91-beda-b104451480cb stalled after successful gates"
gate_results:
  - "focused dashboard update/state tests before implementation: RED, updateConnection helper missing and Edit/Deactivate controls absent"
  - "focused dashboard update/state tests after implementation: PASS, 7 files and 41 tests"
  - "npm --prefix ui run e2e: PASS, 23 tests passed and 1 real-server mobile skip"
  - "make verify: PASS; bootstrapped UI deps, passed go test, go vet, go build, UI unit/build/E2E, make build, and git diff --check"
  - "replacement external evaluator 019e95ea-0676-7761-8373-a07074aaeeaf: focused UI tests PASS, E2E PASS, make verify PASS, standalone git diff --check PASS; no blocking findings surfaced before final report stall"

tasks:
  - id: "8.CE.1"
    name: "Add dashboard update controls and complete state coverage"
    status: DONE
    agent: "orchestrator"
    commit: "2ba7073"
    files_owned:
      - ui/src/api.ts
      - ui/src/api.test.ts
      - ui/src/pages/AliasesPage.tsx
      - ui/src/pages/AliasesPage.test.tsx
      - ui/src/pages/CombosPage.tsx
      - ui/src/pages/CombosPage.test.tsx
      - ui/src/pages/PricingPage.tsx
      - ui/src/pages/PricingPage.test.tsx
      - ui/src/pages/ProvidersPage.tsx
      - ui/src/pages/ProvidersPage.test.tsx
      - ui/src/pages/LogsPage.test.tsx
      - ui/src/pages/DiagnosticsPage.test.tsx
      - ui/e2e/dashboard.e2e.ts
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8CE-evaluator-prompt.md
```

**Checkpoint**: Wave 8.CE closes the dashboard audit gaps for documented update endpoints and state coverage. Connections can be activated/deactivated through `PUT /api/connections/:id`, aliases, combos, and pricing overrides can be edited through their documented PUT routes, connection metadata is scrubbed before dashboard PUT serialization, Pricing/Logs/Diagnostics have explicit negative-state tests, and mocked dashboard E2E covers update actions plus auth-expired coverage for Pricing, Usage, Logs, Quotas, and Diagnostics.

---

### Wave 8.CF — Backend And Docs Audit Contract Hardening

```yaml
wave: "8.CF"
status: DONE
max_agents: 1
gate: "go test ./api/handlers ./internal/provider -run 'TestUsageQuotaRawJSONContract|TestProvidersListModelsForDynamicProvider|TestOAuthStartDoesNotLeakFlowErrorSecrets|TestOAuthPhaseDocsDescribeCursorOMPFlow' -count=1 && make verify"
completed_at: "2026-06-05T04:04:11Z"
evaluator_prompt: "docs/evaluations/wave-8CF-evaluator-prompt.md"
evaluation: "PASS replacement external evaluator thread 019e95f8-117e-71b3-a762-6dcfbbce47b8 at commit 38e715c; original evaluator 019e95f6-64a0-7083-b15d-5c8dd4c07bf9 stalled before gate output and was archived"
gate_results:
  - "focused backend/docs tests before implementation: RED, OAuthStart leaked raw flow error details, provider dynamic test asserted internal-only provider field, and phase-05 docs still described Cursor as PKCE OAuth"
  - "focused backend/docs tests after implementation: PASS"
  - "make verify: PASS; bootstrapped UI deps, passed go test, go vet, go build, UI unit/build/E2E, make build, and git diff --check"
  - "replacement external evaluator 019e95f8-117e-71b3-a762-6dcfbbce47b8: targeted Go gate PASS, make verify PASS, standalone git diff --check PASS, no blocking findings"

tasks:
  - id: "8.CF.1"
    name: "Harden backend API contract coverage and Cursor OAuth docs truth"
    status: DONE
    agent: "orchestrator"
    commit: "bff6624"
    files_owned:
      - api/handlers/oauth.go
      - api/handlers/oauth_test.go
      - api/handlers/providers_test.go
      - api/handlers/usage_test.go
      - internal/provider/matrix_test.go
      - docs/phases/phase-05-oauth-flows-cli.md
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8CF-evaluator-prompt.md
```

**Checkpoint**: Wave 8.CF closes the remaining small audit gaps around raw quota JSON shape, dynamic provider model-list coverage, OAuth start-path error redaction, and stale Cursor OAuth phase wording.

---

### Wave 8.CG — Final Docs And Workflow Closure

```yaml
wave: "8.CG"
status: DONE
max_agents: 1
gate: "rg -n 'status: (PENDING|IN_PROGRESS|BLOCKED)|evaluation: \"PENDING|\\[ \\]|TODO|FIXME|implementation does not exist|implementation doesn''t exist' docs && false || true; git diff --check"
completed_at: "2026-06-05T04:18:07Z"
evaluator_prompt: "docs/evaluations/wave-8CG-evaluator-prompt.md"
evaluation: "PASS external evaluator thread 019e9602-c8f2-7601-b681-13296280746a at commit 03685bc"
gate_results:
  - "final docs/workflow completion audit 019e95fb-59d2-7352-8485-0a0247daff83: FAIL before implementation, README still said Stage 8 remained active and WORKFLOW front matter still reported PARITY_HARDENING/current 8.CF"
  - "final code/test gap audit 019e95fb-9beb-7a51-b9cb-bb88e33c4bed: PASS, go test ./... -count=1 PASS, npm --prefix ui run e2e PASS with 23 passed and 1 skipped, no code/test blockers"
  - "external evaluator 019e9602-c8f2-7601-b681-13296280746a: PASS, completion-status rg PASS, stale README/Phase 0 rg PASS, Stage 8 range rg PASS, git diff --check PASS"

tasks:
  - id: "8.CG.1"
    name: "Close final docs and workflow status"
    status: DONE
    agent: "orchestrator"
    commit: "99eaef4"
    files_owned:
      - docs/README.md
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/WORKFLOW.md
      - docs/phases/phase-00-project-bootstrap.md
      - docs/evaluations/wave-8CG-evaluator-prompt.md
```

**Checkpoint**: Wave 8.CG closes the final docs-only audit findings. The workflow now marks the project complete, README no longer points agents at nonexistent pending work, and the historical Phase 0 bootstrap sample no longer says `serve` is unimplemented.

---

### Wave 8.CH — Stale Closure-Language Cleanup

```yaml
wave: "8.CH"
status: DONE
max_agents: 1
gate: "rg -n 'remains active|next PENDING|PENDING task|current wave|find IN_PROGRESS task|not yet implemented' docs/README.md docs/WORKFLOW.md docs/PLAN.md docs/ORCHESTRATION.md docs/phases --glob '*.md' | rg -v 'gate:|fresh completion audit' && false || true; git diff --check"
completed_at: "2026-06-05T04:25:40Z"
evaluator_prompt: "docs/evaluations/wave-8CH-evaluator-prompt.md"
evaluation: "PASS re-evaluator thread 019e960b-4215-7721-bcfe-afbbf25f015b at commit f329eee; initial evaluator 019e9609-6b02-7a30-9aea-5d1762c79830 failed on stale README 8.CG wording before fix"
gate_results:
  - "fresh completion audit after Wave 8.CG: FAIL before implementation, docs/WORKFLOW.md still said Stage 8 remains active and PLAN/WORKFLOW navigation still pointed agents to current-wave pending tasks"
  - "initial external evaluator 019e9609-6b02-7a30-9aea-5d1762c79830: FAIL, docs/README.md still said complete through Stage 8.CG while WORKFLOW had 8.CH"
  - "re-evaluator 019e960b-4215-7721-bcfe-afbbf25f015b: PASS, completion metadata gate PASS, stale active-work language scan PASS, Stage 8 range gate PASS, git diff --check PASS, README says Wave 8.CH"

tasks:
  - id: "8.CH.1"
    name: "Remove stale current-wave and pending-task closure language"
    status: DONE
    agent: "orchestrator"
    commit: "1288133"
    files_owned:
      - docs/README.md
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8CH-evaluator-prompt.md
```

**Checkpoint**: Wave 8.CH removes the remaining stale completion-language blockers found after Wave 8.CG. Historical wave names that include "remaining" are left intact when they describe old batch contents, not current unfinished work.

---

### Wave 8.CI — MCP Accounts Test Stability

```yaml
wave: "8.CI"
status: DONE
max_agents: 1
gate: "npm --prefix ui test -- --run src/pages/McpSplitPages.test.tsx && make verify"
completed_at: "2026-06-05T04:34:56Z"
evaluator_prompt: "docs/evaluations/wave-8CI-evaluator-prompt.md"
evaluation: "PASS re-evaluator 019e961a-77e6-7721-b584-e20f7a310ef9 at commit 5cf6748; initial evaluator 019e9612-1d32-7d20-ad63-1ff9ab3da303 failed because make verify hit a Playwright MCP create-flow race before fix"
gate_results:
  - "npm --prefix ui test -- --run src/pages/McpSplitPages.test.tsx: PASS, 1 file and 6 tests"
  - "initial external evaluator 019e9612-1d32-7d20-ad63-1ff9ab3da303: FAIL before E2E fix; make verify failed in Playwright at dashboard control plane > creates MCP instances with advanced launch fields while waiting for the MCP instances table"
  - "npm --prefix ui run e2e -- --project=chromium -g 'creates MCP instances with advanced launch fields': PASS after fix, 1 test"
  - "make verify: PASS after fix, includes npm ci --prefix ui --include=dev; go test ./... -count=1; go vet ./...; go build ./cmd/g0router; npm --prefix ui test -- --run with 20 files and 97 tests; npm --prefix ui run build; npm --prefix ui run e2e with 23 passed and 1 skipped; make build; git diff --check"
  - "external re-evaluator 019e961a-77e6-7721-b584-e20f7a310ef9: PASS, focused MCP split-page unit gate PASS, make verify PASS, standalone git diff --check PASS"

tasks:
  - id: "8.CI.1"
    name: "Stabilize MCP accounts split-page async control assertions"
    status: DONE
    agent: "orchestrator"
    commit: "ff51b3e"
    fix_commit: "6d02935"
    files_owned:
      - ui/src/pages/McpSplitPages.test.tsx
      - ui/e2e/dashboard.e2e.ts
      - docs/README.md
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
      - docs/WORKFLOW.md
      - docs/evaluations/wave-8CI-evaluator-prompt.md
```

**Checkpoint**: Wave 8.CI fixes the release-gate UI race found after Wave 8.CH by waiting for loaded MCP account controls before interacting with OAuth fields/buttons. The full `make verify` gate passed after the fix.

---

## STAGE 0 — Bootstrap

### Wave 0.A

```yaml
wave: "0.A"
status: DONE
max_agents: 1
gate: "go build ./cmd/g0router && go vet ./..."

tasks:
  - id: "0.1"
    name: "Initialize Go module and directory structure"
    status: DONE
    agent: "orchestrator"
    started_at: "2026-06-02T17:13:28Z"
    completed_at: "2026-06-02T17:13:28Z"
    files_owned:
      - go.mod
      - cmd/g0router/main.go
      - .gitignore
      - .env.example
    phase_doc: "docs/phases/phase-00-project-bootstrap.md"
```

**Checkpoint**: `PHASE_0_COMPLETE` → advance to STAGE 1

---

## STAGE 1 — Core Foundation

### Wave 1.A — Independent foundation (3 agents)

```yaml
wave: "1.A"
status: DONE
max_agents: 3
depends_on: ["0.A"]
gate: "go test ./... && go vet ./..."

tasks:
  - id: "1.1"
    name: "Define core types"
    status: DONE
    agent: "Mendel"
    completed_at: "2026-06-02T17:22:56Z"
    files_owned:
      - internal/providers/types.go
      - internal/providers/types_test.go
      - internal/providers/interface.go
    phase_doc: "docs/phases/phase-01-core-types-sqlite-store.md"

  - id: "1.2"
    name: "SQLite store foundation"
    status: DONE
    agent: "Carver"
    completed_at: "2026-06-02T17:22:56Z"
    files_owned:
      - internal/store/sqlite.go
      - internal/store/sqlite_test.go
    phase_doc: "docs/phases/phase-01-core-types-sqlite-store.md"

  - id: "1.6"
    name: "Config loading"
    status: DONE
    agent: "Feynman"
    completed_at: "2026-06-02T17:22:56Z"
    files_owned:
      - internal/config/config.go
      - internal/config/config_test.go
    phase_doc: "docs/phases/phase-01-core-types-sqlite-store.md"
```

### Wave 1.B — Store CRUD (3 agents)

```yaml
wave: "1.B"
status: DONE
max_agents: 3
depends_on: ["1.A"]
gate: "go test ./... && go vet ./..."

tasks:
  - id: "1.3"
    name: "Connection CRUD"
    status: DONE
    agent: "Dirac"
    completed_at: "2026-06-02T17:30:15Z"
    depends_on_tasks: ["1.2"]
    files_owned:
      - internal/store/connections.go
      - internal/store/connections_test.go
      - internal/store/errors.go
    phase_doc: "docs/phases/phase-01-core-types-sqlite-store.md"

  - id: "1.4"
    name: "Settings + API keys store"
    status: DONE
    agent: "Arendt"
    completed_at: "2026-06-02T17:30:15Z"
    depends_on_tasks: ["1.2"]
    files_owned:
      - internal/store/settings.go
      - internal/store/settings_test.go
      - internal/store/apikeys.go
      - internal/store/apikeys_test.go
    phase_doc: "docs/phases/phase-01-core-types-sqlite-store.md"

  - id: "1.5"
    name: "Usage log store"
    status: DONE
    agent: "Pascal"
    completed_at: "2026-06-02T17:30:15Z"
    depends_on_tasks: ["1.2"]
    files_owned:
      - internal/store/usage.go
      - internal/store/usage_test.go
    phase_doc: "docs/phases/phase-01-core-types-sqlite-store.md"
```

**Checkpoint**: `PHASE_1_COMPLETE` → advance to STAGE 2

---

## STAGE 2 — Server + Parallel Streams

### Wave 2.A — HTTP foundations + independent streams (up to 8 agents)

```yaml
wave: "2.A"
status: DONE
max_agents: 8
depends_on: ["1.B"]
gate: "go test ./... && go vet ./..."

tasks:
  - id: "2.1"
    name: "fasthttp server skeleton"
    status: DONE
    agent: "Erdos"
    completed_at: "2026-06-02T17:44:41Z"
    files_owned:
      - go.mod
      - go.sum
      - api/server.go
      - api/server_test.go
      - api/handlers/health.go
    phase_doc: "docs/phases/phase-02-http-server-proxy-engine.md"

  - id: "2.3"
    name: "Proxy engine core"
    status: DONE
    agent: "Raman"
    completed_at: "2026-06-02T17:44:41Z"
    files_owned:
      - internal/proxy/engine.go
      - internal/proxy/pool.go
      - internal/proxy/engine_test.go
    phase_doc: "docs/phases/phase-02-http-server-proxy-engine.md"

  - id: "2.4"
    name: "OpenAI provider implementation"
    status: DONE
    agent: "Socrates"
    completed_at: "2026-06-02T17:44:41Z"
    files_owned:
      - internal/providers/openai/openai.go
      - internal/providers/openai/types.go
      - internal/providers/openai/errors.go
      - internal/providers/openai/openai_test.go
    phase_doc: "docs/phases/phase-02-http-server-proxy-engine.md"

  - id: "2.5"
    name: "Shared provider utilities"
    status: DONE
    agent: "Noether"
    completed_at: "2026-06-02T17:44:41Z"
    files_owned:
      - internal/providers/utils/http.go
      - internal/providers/utils/http_test.go
      - internal/providers/utils/sse.go
      - internal/providers/utils/sse_test.go
      - internal/providers/utils/errors.go
    phase_doc: "docs/phases/phase-02-http-server-proxy-engine.md"

  - id: "2.6"
    name: "Streaming accumulator"
    status: DONE
    agent: "Aristotle"
    completed_at: "2026-06-02T17:44:41Z"
    files_owned:
      - internal/streaming/accumulator.go
      - internal/streaming/chat.go
      - internal/streaming/accumulator_test.go
    phase_doc: "docs/phases/phase-02-http-server-proxy-engine.md"

  - id: "5.1"
    name: "OAuth types and interface"
    status: DONE
    agent: "Gibbs"
    completed_at: "2026-06-02T17:44:41Z"
    files_owned:
      - internal/provider/oauth/types.go
      - internal/provider/oauth/types_test.go
    phase_doc: "docs/phases/phase-05-oauth-flows-cli.md"

  - id: "7.1"
    name: "RTK autodetect"
    status: DONE
    agent: "Cicero"
    completed_at: "2026-06-02T17:44:41Z"
    files_owned:
      - internal/rtk/autodetect.go
      - internal/rtk/autodetect_test.go
      - internal/rtk/constants.go
    phase_doc: "docs/phases/phase-07-rtk-caveman.md"

  - id: "8.1+8.2"
    name: "Usage extraction + pricing catalog"
    status: DONE
    agent: "Franklin"
    completed_at: "2026-06-02T17:44:41Z"
    files_owned:
      - internal/usage/tracker.go
      - internal/usage/tracker_test.go
      - internal/modelcatalog/pricing.go
      - internal/modelcatalog/catalog.go
      - internal/modelcatalog/pricing_test.go
    phase_doc: "docs/phases/phase-08-usage-tracking-cost-logging.md"
```

### Wave 2.B — Middleware + dependent tasks (up to 8 agents)

```yaml
wave: "2.B"
status: DONE
max_agents: 8
depends_on: ["2.A"]
gate: "go test ./... && go vet ./..."

tasks:
  - id: "2.2"
    name: "Middleware (CORS, auth, request ID)"
    status: DONE
    agent: "Hilbert"
    completed_at: "2026-06-02T17:59:59Z"
    depends_on_tasks: ["2.1"]
    files_owned:
      - api/server.go
      - api/middleware.go
      - api/middleware_test.go
    phase_doc: "docs/phases/phase-02-http-server-proxy-engine.md"

  - id: "4.1"
    name: "Provider registry"
    status: DONE
    agent: "Darwin"
    completed_at: "2026-06-02T17:59:59Z"
    files_owned:
      - internal/provider/registry.go
      - internal/provider/registry_test.go
    phase_doc: "docs/phases/phase-04-persistence-provider-registry.md"

  - id: "4.2"
    name: "Connection management with round-robin"
    status: DONE
    agent: "Jason"
    completed_at: "2026-06-02T17:59:59Z"
    files_owned:
      - internal/provider/connection.go
      - internal/provider/connection_test.go
    phase_doc: "docs/phases/phase-04-persistence-provider-registry.md"

  - id: "7.2"
    name: "RTK filters (11 total)"
    status: DONE
    agent: "Bohr"
    completed_at: "2026-06-02T17:59:59Z"
    depends_on_tasks: ["7.1"]
    files_owned:
      - internal/rtk/filters/*.go
    phase_doc: "docs/phases/phase-07-rtk-caveman.md"

  - id: "7.4"
    name: "Caveman prompt injection"
    status: DONE
    agent: "Pauli"
    completed_at: "2026-06-02T17:59:59Z"
    files_owned:
      - internal/rtk/caveman.go
      - internal/rtk/caveman_test.go
      - internal/rtk/prompts.go
    phase_doc: "docs/phases/phase-07-rtk-caveman.md"

  - id: "8.3"
    name: "Cost calculation"
    status: DONE
    agent: "Singer"
    completed_at: "2026-06-02T17:59:59Z"
    depends_on_tasks: ["8.1+8.2"]
    files_owned:
      - internal/usage/cost.go
      - internal/usage/cost_test.go
    phase_doc: "docs/phases/phase-08-usage-tracking-cost-logging.md"

  - id: "5.2"
    name: "Anthropic OAuth (Claude Code)"
    status: DONE
    agent: "Averroes"
    completed_at: "2026-06-02T17:59:59Z"
    depends_on_tasks: ["5.1"]
    files_owned:
      - internal/provider/oauth/anthropic.go
      - internal/provider/oauth/anthropic_test.go
    phase_doc: "docs/phases/phase-05-oauth-flows-cli.md"

  - id: "5.3"
    name: "OpenAI Codex OAuth"
    status: DONE
    agent: "Goodall"
    completed_at: "2026-06-02T17:59:59Z"
    depends_on_tasks: ["5.1"]
    files_owned:
      - internal/provider/oauth/codex.go
      - internal/provider/oauth/codex_test.go
    phase_doc: "docs/phases/phase-05-oauth-flows-cli.md"
```

### Wave 2.C — Integration + more OAuth (up to 8 agents)

```yaml
wave: "2.C"
status: DONE
max_agents: 8
depends_on: ["2.B"]
gate: "go test ./... && go vet ./..."

tasks:
  - id: "2.7"
    name: "Inference handler"
    status: DONE
    agent: "Lovelace"
    completed_at: "2026-06-02T18:41:49Z"
    depends_on_tasks: ["2.1", "2.2", "2.3", "2.5", "2.6"]
    files_owned:
      - api/server.go
      - api/handlers/inference.go
      - api/handlers/inference_test.go
      - api/handlers/models.go
    phase_doc: "docs/phases/phase-02-http-server-proxy-engine.md"

  - id: "7.3"
    name: "RTK message compression"
    status: DONE
    agent: "Chandrasekhar"
    completed_at: "2026-06-02T18:41:49Z"
    depends_on_tasks: ["7.1", "7.2"]
    files_owned:
      - internal/rtk/rtk.go
      - internal/rtk/rtk_test.go
    phase_doc: "docs/phases/phase-07-rtk-caveman.md"

  - id: "8.4"
    name: "Provider quota fetchers"
    status: DONE
    agent: "Harvey"
    completed_at: "2026-06-02T18:41:49Z"
    files_owned:
      - internal/usage/quota.go
      - internal/usage/quota_test.go
    phase_doc: "docs/phases/phase-08-usage-tracking-cost-logging.md"

  - id: "8.5"
    name: "Request/response logging"
    status: DONE
    agent: "Faraday"
    completed_at: "2026-06-02T18:41:49Z"
    files_owned:
      - internal/logging/logger.go
      - internal/logging/requestlog.go
      - internal/logging/logger_test.go
    phase_doc: "docs/phases/phase-08-usage-tracking-cost-logging.md"

  - id: "5.4"
    name: "GitHub Copilot OAuth"
    status: DONE
    agent: "Maxwell"
    completed_at: "2026-06-02T18:41:49Z"
    depends_on_tasks: ["5.1"]
    files_owned:
      - internal/provider/oauth/github.go
      - internal/provider/oauth/github_test.go
    phase_doc: "docs/phases/phase-05-oauth-flows-cli.md"

  - id: "5.5"
    name: "Cursor PKCE OAuth"
    status: DONE
    agent: "Rawls"
    completed_at: "2026-06-02T18:41:49Z"
    depends_on_tasks: ["5.1"]
    files_owned:
      - internal/provider/oauth/cursor.go
      - internal/provider/oauth/cursor_test.go
    phase_doc: "docs/phases/phase-05-oauth-flows-cli.md"

  - id: "5.6"
    name: "Google OAuth (Gemini CLI, Antigravity)"
    status: DONE
    agent: "Hubble"
    completed_at: "2026-06-02T18:41:49Z"
    depends_on_tasks: ["5.1"]
    files_owned:
      - internal/provider/oauth/gemini.go
      - internal/provider/oauth/antigravity.go
      - internal/provider/oauth/gemini_test.go
      - internal/provider/oauth/antigravity_test.go
    phase_doc: "docs/phases/phase-05-oauth-flows-cli.md"

  - id: "4.3"
    name: "Combos store + resolver"
    status: DONE
    agent: "Copernicus"
    completed_at: "2026-06-02T18:41:49Z"
    files_owned:
      - internal/store/combos.go
      - internal/store/combos_test.go
      - internal/proxy/combo.go
    phase_doc: "docs/phases/phase-04-persistence-provider-registry.md"
    notes: "Owns combo resolver in Wave 2.C. Later Task 6.2 must serialize or split any additional internal/proxy/combo.go changes."
```

**Checkpoint**: `PHASE_2_COMPLETE`, `PHASE_7_COMPLETE` → advance to STAGE 3

---

## STAGE 3 — Providers + MCP + UI

### Wave 3.A — Provider implementations (up to 8 agents)

```yaml
wave: "3.A"
status: DONE
max_agents: 8
depends_on: ["2.C"]
gate: "go test ./... && go vet ./..."

tasks:
  - id: "3.1"
    name: "Anthropic provider"
    status: DONE
    agent: "Lagrange"
    completed_at: "2026-06-02T19:07:47Z"
    files_owned: ["internal/providers/anthropic/*"]

  - id: "3.2"
    name: "Format translation engine"
    status: DONE
    agent: "Tesla"
    completed_at: "2026-06-02T19:07:47Z"
    files_owned: ["internal/translate/detect.go", "internal/translate/openai.go", "internal/translate/anthropic.go", "internal/translate/detect_test.go", "internal/translate/anthropic_test.go"]

  - id: "3.3"
    name: "OpenAI-compatible providers (batch)"
    status: DONE
    agent: "Bernoulli"
    completed_at: "2026-06-02T19:07:47Z"
    files_owned: ["internal/providers/openaicompat/*"]

  - id: "3.4"
    name: "Gemini provider"
    status: DONE
    agent: "Kant"
    completed_at: "2026-06-02T19:07:47Z"
    files_owned: ["internal/providers/gemini/*"]

  - id: "3.7"
    name: "AWS Bedrock provider"
    status: DONE
    agent: "Anscombe"
    completed_at: "2026-06-02T19:07:47Z"
    files_owned: ["internal/providers/bedrock/*"]

  - id: "3.8"
    name: "Azure OpenAI provider"
    status: DONE
    agent: "Boyle"
    completed_at: "2026-06-02T19:07:47Z"
    files_owned: ["internal/providers/azure/*"]

  - id: "9.1+9.2"
    name: "MCP client manager + tool manager"
    status: DONE
    agent: "Plato"
    completed_at: "2026-06-02T19:07:47Z"
    files_owned: ["internal/mcp/clientmanager.go", "internal/mcp/clientmanager_test.go", "internal/mcp/toolmanager.go", "internal/mcp/toolmanager_test.go"]

  - id: "10.1"
    name: "UI scaffold (Vite + React + Tailwind)"
    status: DONE
    agent: "Descartes"
    completed_at: "2026-06-02T19:07:47Z"
    files_owned: ["ui/**"]
```

### Wave 3.B — Translation + remaining providers + OAuth + management (up to 8 agents)

```yaml
wave: "3.B"
status: DONE
max_agents: 8
depends_on: ["3.A"]
gate: "go test ./... && go vet ./..."

tasks:
  - id: "3.5"
    name: "Gemini format translation"
    status: DONE
    agent: "Boole"
    completed_at: "2026-06-02T19:31:59Z"
    depends_on_tasks: ["3.2", "3.4"]
    files_owned: ["internal/translate/gemini.go", "internal/translate/gemini_test.go"]

  - id: "3.6"
    name: "Vertex AI provider"
    status: DONE
    agent: "Popper"
    completed_at: "2026-06-02T19:31:59Z"
    depends_on_tasks: ["3.4"]
    files_owned: ["internal/providers/vertex/*"]

  - id: "3.9"
    name: "Mistral, Ollama, Cohere, Replicate"
    status: DONE
    agent: "Confucius"
    completed_at: "2026-06-02T19:31:59Z"
    files_owned: ["internal/providers/mistral/*", "internal/providers/ollama/*", "internal/providers/cohere/*", "internal/providers/replicate/*"]

  - id: "4.4"
    name: "Model aliases + pricing overrides"
    status: DONE
    agent: "Mill"
    completed_at: "2026-06-02T19:31:59Z"
    files_owned: ["internal/store/aliases.go", "internal/store/aliases_test.go", "internal/store/pricing.go", "internal/store/pricing_test.go"]

  - id: "5.7"
    name: "xAI, DeepSeek, GitLab, Kiro OAuth"
    status: DONE
    agent: "James"
    completed_at: "2026-06-02T19:31:59Z"
    depends_on_tasks: ["5.1"]
    files_owned: ["internal/provider/oauth/xai.go", "internal/provider/oauth/deepseek.go", "internal/provider/oauth/gitlab.go", "internal/provider/oauth/kiro.go"]

  - id: "5.8"
    name: "Chinese provider OAuth"
    status: DONE
    agent: "Hooke"
    completed_at: "2026-06-02T19:31:59Z"
    depends_on_tasks: ["5.1"]
    files_owned: ["internal/provider/oauth/kimi.go", "internal/provider/oauth/minimax.go", "internal/provider/oauth/alibaba.go", "internal/provider/oauth/zhipu.go", "internal/provider/oauth/xiaomi.go"]

  - id: "5.9"
    name: "Token refresh with dedup"
    status: DONE
    agent: "Ampere"
    completed_at: "2026-06-02T19:31:59Z"
    depends_on_tasks: ["5.1"]
    files_owned: ["internal/provider/refresh.go", "internal/provider/refresh_test.go"]

  - id: "10.2-10.7"
    name: "UI pages (Dashboard, Endpoint, Providers, Usage, Quota, etc.)"
    status: DONE
    agent: "Turing"
    completed_at: "2026-06-02T19:31:59Z"
    depends_on_tasks: ["10.1"]
    files_owned: ["ui/src/pages/*", "ui/src/components/*", "ui/src/api/*"]
```

**Checkpoint**: Wave 3.B complete → advance to STAGE 4. `PHASE_3_COMPLETE` waits for Task 3.10 in Wave 4.B.

---

## STAGE 4 — Integration + Final Features

### Wave 4.A — Handlers + CLI + fallback (up to 6 agents)

```yaml
wave: "4.A"
status: DONE
max_agents: 6
depends_on: ["3.B"]
gate: "go test ./... && go vet ./..."

tasks:
  - id: "4.5"
    name: "Management API handlers"
    status: DONE
    agent: "Galileo"
    completed_at: "2026-06-02T19:54:04Z"
    depends_on_tasks: ["4.1", "4.2", "4.3", "4.4"]
    files_owned: ["api/handlers/providers.go", "api/handlers/connections.go", "api/handlers/settings.go", "api/handlers/apikeys.go", "api/handlers/combos.go"]

  - id: "5.10"
    name: "OAuth HTTP endpoints"
    status: DONE
    agent: "Schrodinger"
    completed_at: "2026-06-02T19:54:04Z"
    files_owned: ["api/handlers/oauth.go", "api/handlers/oauth_test.go"]

  - id: "5.11"
    name: "CLI commands (cobra)"
    status: DONE
    agent: "Ptolemy"
    completed_at: "2026-06-02T19:54:04Z"
    files_owned: ["internal/cli/*.go", "cmd/g0router/main.go"]

  - id: "6.1"
    name: "Account fallback engine"
    status: DONE
    agent: "Leibniz"
    completed_at: "2026-06-02T19:54:04Z"
    depends_on_tasks: ["4.2"]
    files_owned: ["internal/provider/fallback.go", "internal/provider/fallback_test.go"]

  - id: "6.2"
    name: "Combo model resolution"
    status: DONE
    agent: "Halley"
    completed_at: "2026-06-02T19:54:04Z"
    depends_on_tasks: ["4.3"]
    files_owned: ["internal/proxy/combo.go", "internal/proxy/combo_test.go"]

  - id: "8.6"
    name: "Usage + logging API handlers"
    status: DONE
    agent: "Peirce"
    completed_at: "2026-06-02T19:54:04Z"
    depends_on_tasks: ["8.1+8.2", "8.3", "8.4", "8.5"]
    files_owned: ["api/handlers/usage.go", "api/handlers/logging.go"]
```

### Wave 4.B — MCP completion + Responses + UI embed (up to 6 agents)

```yaml
wave: "4.B"
status: DONE
max_agents: 6
depends_on: ["4.A"]
gate: "go test ./... && go vet ./..."

tasks:
  - id: "3.10"
    name: "Responses API support"
    status: DONE
    agent: "Aquinas"
    completed_at: "2026-06-02T20:44:29Z"
    files_owned: ["internal/providers/openai/responses.go", "internal/streaming/responses.go", "internal/translate/responses.go"]

  - id: "9.3"
    name: "MCP tool discovery (compact manifests)"
    status: DONE
    agent: "Euler"
    completed_at: "2026-06-02T20:44:29Z"
    depends_on_tasks: ["9.1+9.2"]
    files_owned: ["internal/mcp/discovery.go", "internal/mcp/discovery_test.go"]

  - id: "9.4"
    name: "MCP agent loop"
    status: DONE
    agent: "Wegener"
    completed_at: "2026-06-02T20:44:29Z"
    depends_on_tasks: ["9.1+9.2"]
    files_owned: ["internal/mcp/agent.go", "internal/mcp/agent_test.go"]

  - id: "9.5"
    name: "MCP health monitor"
    status: DONE
    agent: "Archimedes"
    completed_at: "2026-06-02T20:44:29Z"
    depends_on_tasks: ["9.1+9.2"]
    files_owned: ["internal/mcp/healthmonitor.go", "internal/mcp/healthmonitor_test.go"]

  - id: "9.6"
    name: "MCP API handlers + store"
    status: DONE
    agent: "Volta"
    completed_at: "2026-06-02T20:44:29Z"
    depends_on_tasks: ["9.1+9.2", "9.3", "9.4", "9.5"]
    files_owned: ["api/handlers/mcp.go", "internal/store/mcpclients.go", "internal/store/mcpclients_test.go"]

  - id: "10.8"
    name: "Embed UI in Go binary"
    status: DONE
    agent: "Hegel"
    completed_at: "2026-06-02T20:44:29Z"
    depends_on_tasks: ["10.2-10.7"]
    files_owned: ["embed.go"]
```

**Checkpoint**: `PHASE_4_COMPLETE`, `PHASE_5_COMPLETE`, `PHASE_6_COMPLETE`, `PHASE_8_COMPLETE`, `PHASE_9_COMPLETE`, `PHASE_10_COMPLETE`

---

## STAGE 5 — Packaging

### Wave 5.A — Final packaging (3 agents)

```yaml
wave: "5.A"
status: DONE
max_agents: 3
depends_on: ["4.B"]
gate: "make test && make build"

tasks:
  - id: "11.1"
    name: "Makefile"
    status: DONE
    agent: "orchestrator"
    completed_at: "2026-06-02T20:59:58Z"
    files_owned: ["Makefile"]

  - id: "11.2"
    name: "systemd service + install CLI"
    status: DONE
    agent: "orchestrator"
    completed_at: "2026-06-02T20:59:58Z"
    files_owned: ["deploy/g0router.service", "deploy/g0router.default", "internal/cli/install.go", "internal/cli/install_test.go"]

  - id: "11.3"
    name: "Docker support"
    status: DONE
    agent: "orchestrator"
    completed_at: "2026-06-02T20:59:58Z"
    files_owned: ["Dockerfile", "docker-compose.yml", ".dockerignore"]
```

### Wave 5.B — Polish + E2E (2 agents)

```yaml
wave: "5.B"
status: DONE
max_agents: 2
depends_on: ["5.A"]
gate: "make test && make build && make docker"

tasks:
  - id: "11.4"
    name: ".env.example + README polish"
    status: DONE
    agent: "orchestrator"
    completed_at: "2026-06-02T21:15:40Z"
    files_owned: ["README.md", ".env.example", "docs/DEPLOYMENT.md"]

  - id: "11.5"
    name: "Final integration test suite"
    status: DONE
    agent: "orchestrator"
    completed_at: "2026-06-02T21:28:21Z"
    files_owned: ["e2e_test.go"]
```

**Checkpoint**: `PHASE_11_COMPLETE` → advance to STAGE 6

---

## STAGE 6 — Advanced MCP Gateway

### Wave 6.A — Future MCP instance + OAuth gateway (1 agent, sequential)

```yaml
wave: "6.A"
status: DONE
max_agents: 1
depends_on: ["5.B"]
gate: "go test ./... && go vet ./... && go build ./cmd/g0router"

tasks:
  - id: "12.1"
    name: "MCP instance model + store"
    status: DONE
    agent: "orchestrator"
    completed_at: "2026-06-02T22:17:31Z"
    files_owned: ["internal/store/mcpinstances.go", "internal/store/mcpinstances_test.go", "internal/mcp/instances.go", "internal/mcp/instances_test.go"]
    phase_doc: "docs/phases/phase-12-advanced-mcp-gateway.md"

  - id: "12.2"
    name: "MCP launchers for command, npx, docker, and HTTP"
    status: DONE
    agent: "orchestrator"
    completed_at: "2026-06-02T22:20:03Z"
    depends_on_tasks: ["12.1"]
    files_owned: ["internal/mcp/launcher.go", "internal/mcp/launcher_test.go", "internal/mcp/process.go", "internal/mcp/http.go"]
    phase_doc: "docs/phases/phase-12-advanced-mcp-gateway.md"

  - id: "12.3"
    name: "MCP OAuth account engine"
    status: DONE
    agent: "orchestrator"
    completed_at: "2026-06-02T22:22:31Z"
    depends_on_tasks: ["12.1"]
    files_owned: ["internal/mcp/oauth.go", "internal/mcp/oauth_test.go", "internal/store/mcpoauth.go", "internal/store/mcpoauth_test.go"]
    phase_doc: "docs/phases/phase-12-advanced-mcp-gateway.md"

  - id: "12.4"
    name: "MCP OAuth callback URL completion"
    status: DONE
    agent: "orchestrator"
    completed_at: "2026-06-02T22:25:41Z"
    depends_on_tasks: ["12.3"]
    files_owned: ["api/handlers/mcpoauth.go", "api/handlers/mcpoauth_test.go", "internal/cli/mcp_auth.go"]
    phase_doc: "docs/phases/phase-12-advanced-mcp-gateway.md"

  - id: "12.5"
    name: "MCP management surfaces"
    status: DONE
    agent: "orchestrator"
    completed_at: "2026-06-02T22:29:46Z"
    depends_on_tasks: ["12.1", "12.2", "12.3", "12.4"]
    files_owned: ["api/handlers/mcp.go", "internal/cli/mcp.go", "ui/src/pages/*", "ui/src/components/*"]
    phase_doc: "docs/phases/phase-12-advanced-mcp-gateway.md"

  - id: "12.6"
    name: "Advanced MCP integration tests + docs"
    status: DONE
    agent: "orchestrator"
    completed_at: "2026-06-02T22:32:54Z"
    depends_on_tasks: ["12.2", "12.3", "12.4", "12.5"]
    files_owned: ["internal/mcp/*integration*_test.go", "docs/SCHEMA.md", "docs/CONFIG.md", "docs/DEPLOYMENT.md", "README.md"]
    phase_doc: "docs/phases/phase-12-advanced-mcp-gateway.md"
```

**Checkpoint**: `PHASE_12_COMPLETE` → **PROJECT COMPLETE**

---

## STAGE 7 — Principal Engineer Remediation

Stage 7 exists because the green tree still left release-blocking runtime and security gaps. It follows the same wave model as earlier stages, with evaluator prompts after each completed wave.

### Wave 7.A — Stop The Bleeding

```yaml
wave: "7.A"
status: DONE
max_agents: 2
depends_on: ["6.A"]
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && make build"

tasks:
  - id: "7.A.1"
    name: "Protect management API, tighten CORS, redact connection credentials"
    status: DONE
    agent: "Linnaeus"
    completed_at: "2026-06-02T23:05:26Z"
    files_owned:
      - api/middleware.go
      - api/middleware_test.go
      - api/handlers/connections.go
      - api/handlers/connections_test.go

  - id: "7.A.2"
    name: "Validated serve config and localhost-default binding"
    status: DONE
    agent: "Herschel"
    completed_at: "2026-06-02T23:05:26Z"
    files_owned:
      - internal/config/config.go
      - internal/config/config_test.go
      - internal/cli/root.go
      - internal/cli/root_test.go
      - docker-compose.yml
      - docs/CONFIG.md

  - id: "7.A.3"
    name: "Orchestrator integration fixes and evaluator prompt"
    status: DONE
    agent: "orchestrator"
    completed_at: "2026-06-02T23:05:26Z"
    files_owned:
      - e2e_test.go
      - docs/WORKFLOW.md
      - docs/evaluations/wave-7A-evaluator-prompt.md

  - id: "7.A.4"
    name: "Evaluator clean-checkout gate fix"
    status: DONE
    agent: "orchestrator"
    completed_at: "2026-06-02T23:14:52Z"
    files_owned:
      - .gitignore
      - ui/dist/**
      - internal/cli/root.go
      - internal/cli/keys_test.go
      - README.md
      - docs/DEPLOYMENT.md
      - docs/WORKFLOW.md
```

**Checkpoint**: Wave 7.A complete → advance to Wave 7.B.

### Wave 7.B — Make `g0router serve` A Real Gateway

```yaml
wave: "7.B"
status: DONE
max_agents: 3
depends_on: ["7.A"]
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && make build"

tasks:
  - id: "7.B.1"
    name: "Wire real inference engine and provider registry in serve startup"
    status: DONE
    agent: "orchestrator"
    completed_at: "2026-06-03T01:27:07Z"
    files_owned:
      - internal/cli/root.go
      - internal/cli/provider_runtime.go
      - internal/cli/root_test.go
      - internal/proxy/engine.go
      - internal/proxy/engine_test.go
      - internal/proxy/pool.go
  - id: "7.B.2"
    name: "Wire MCP runtime managers in serve startup"
    status: DONE
    agent: "orchestrator"
    completed_at: "2026-06-03T01:27:07Z"
    files_owned:
      - internal/cli/root.go
      - internal/cli/mcp_runtime.go
      - internal/cli/root_test.go
      - internal/mcp/launcher.go
  - id: "7.B.3"
    name: "Propagate request contexts through inference, models, and MCP handlers"
    status: DONE
    agent: "orchestrator"
    completed_at: "2026-06-03T01:27:07Z"
    files_owned:
      - api/handlers/context.go
      - api/handlers/inference.go
      - api/handlers/inference_test.go
      - api/handlers/models.go
      - api/handlers/providers.go
      - api/handlers/usage.go
      - api/handlers/usage_test.go
      - api/handlers/mcp.go
  - id: "7.B.4"
    name: "Wave 7.B integration verification and evaluator prompt"
    status: DONE
    agent: "orchestrator"
    completed_at: "2026-06-03T01:27:07Z"
    files_owned:
      - internal/cli/root_test.go
      - docs/WORKFLOW.md
      - docs/evaluations/wave-7B-evaluator-prompt.md
  - id: "7.B.5"
    name: "Evaluator fix: register implemented Vertex provider"
    status: DONE
    agent: "orchestrator"
    completed_at: "2026-06-03T01:35:45Z"
    files_owned:
      - internal/cli/provider_runtime.go
      - internal/cli/root_test.go
      - internal/proxy/engine.go
      - docs/WORKFLOW.md
```

**Checkpoint**: Wave 7.B complete → advance to Wave 7.C.

### Wave 7.C — OAuth and credential lifecycle

```yaml
wave: "7.C"
status: DONE
max_agents: 3
depends_on: ["7.B"]
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && make build"

tasks:
  - id: "7.C.1"
    name: "Store OAuth callback sessions and persist HTTP OAuth completions"
    status: DONE
    agent: "orchestrator"
    completed_at: "2026-06-03T01:49:58Z"
    files_owned:
      - api/handlers/oauth.go
      - api/handlers/oauth_test.go
      - api/server.go
      - internal/provider/oauth/types.go
      - internal/store/oauthsessions.go
      - internal/store/oauthsessions_test.go
      - internal/store/sqlite.go
      - docs/WORKFLOW.md
  - id: "7.C.2"
    name: "Make CLI login complete supported flows and persist connections"
    status: DONE
    agent: "orchestrator"
    completed_at: "2026-06-03T01:55:43Z"
    files_owned:
      - api/handlers/oauth.go
      - internal/cli/auth.go
      - internal/cli/auth_test.go
      - internal/cli/root.go
      - internal/provider/credentials.go
      - internal/provider/credentials_test.go
      - docs/WORKFLOW.md
  - id: "7.C.3"
    name: "Refresh OAuth credentials before dispatch when near expiry"
    status: DONE
    agent: "orchestrator"
    completed_at: "2026-06-03T02:06:42Z"
    files_owned:
      - internal/cli/provider_runtime.go
      - internal/provider/oauth/anthropic.go
      - internal/provider/oauth/anthropic_test.go
      - internal/provider/oauth/antigravity.go
      - internal/provider/oauth/codex.go
      - internal/provider/oauth/codex_test.go
      - internal/provider/oauth/cursor.go
      - internal/provider/oauth/deepseek.go
      - internal/provider/oauth/gemini.go
      - internal/provider/oauth/gemini_test.go
      - internal/provider/oauth/github.go
      - internal/provider/oauth/gitlab.go
      - internal/provider/oauth/kimi.go
      - internal/provider/oauth/kiro.go
      - internal/provider/oauth/refresh.go
      - internal/provider/oauth/types.go
      - internal/provider/oauth/xai.go
      - internal/provider/oauth/xiaomi.go
      - internal/proxy/combo.go
      - internal/proxy/combo_test.go
      - internal/proxy/engine.go
      - internal/proxy/engine_test.go
      - internal/store/connections.go
      - internal/store/connections_test.go
      - docs/WORKFLOW.md
  - id: "7.C.4"
    name: "Normalize provider IDs across auth, routing, docs, and store rows"
    status: DONE
    agent: "orchestrator"
    completed_at: "2026-06-03T02:14:06Z"
    files_owned:
      - api/handlers/connections.go
      - api/handlers/connections_test.go
      - api/handlers/oauth.go
      - api/handlers/oauth_test.go
      - internal/cli/auth.go
      - internal/cli/auth_test.go
      - internal/provider/credentials.go
      - internal/provider/ids.go
      - internal/provider/ids_test.go
      - internal/provider/oauth/types.go
      - internal/provider/oauth/types_test.go
      - internal/proxy/combo.go
      - internal/proxy/combo_test.go
      - internal/proxy/engine.go
      - internal/proxy/engine_test.go
      - internal/store/connections.go
      - docs/WORKFLOW.md
  - id: "7.C.5"
    name: "Wave 7.C evaluator prompt"
    status: DONE
    agent: "orchestrator"
    completed_at: "2026-06-03T02:15:51Z"
    files_owned:
      - docs/evaluations/wave-7C-evaluator-prompt.md
      - docs/WORKFLOW.md
```

### Wave 7.D — Provider and model parity matrix

```yaml
wave: "7.D"
status: DONE
max_agents: 3
depends_on: ["7.C"]
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && make build"

tasks:
  - id: "7.D.1"
    name: "Provider matrix source of truth and public surface wiring"
    status: DONE
    agent: "orchestrator"
    completed_at: "2026-06-03T02:39:45Z"
    files_owned:
      - internal/provider/matrix.go
      - internal/provider/matrix_test.go
      - api/handlers/providers.go
      - api/handlers/providers_test.go
      - internal/cli/root.go
      - internal/cli/root_test.go
      - internal/cli/providers_test.go
  - id: "7.D.2"
    name: "Provider parity documentation cleanup"
    status: DONE
    agent: "orchestrator"
    completed_at: "2026-06-03T02:39:45Z"
    files_owned:
      - README.md
      - docs/README.md
      - docs/PROVIDERS.md
      - docs/SCHEMA.md
      - docs/WORKFLOW.md
  - id: "7.D.3"
    name: "Wave 7.D evaluator prompt"
    status: DONE
    agent: "orchestrator"
    completed_at: "2026-06-03T02:39:45Z"
    files_owned:
      - docs/evaluations/wave-7D-evaluator-prompt.md
      - docs/WORKFLOW.md
```

### Wave 7.E — Real dispatch pipeline

```yaml
wave: "7.E"
status: DONE
max_agents: 3
depends_on: ["7.D"]
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && make build"

tasks:
  - id: "7.E.1"
    name: "Alias and catalog-driven model resolution"
    status: DONE
    agent: "orchestrator"
    branch: "codex/wave-7e-routing"
    completed_at: "2026-06-03T02:51:40Z"
    files_owned:
      - internal/proxy/engine.go
      - internal/proxy/engine_test.go
      - internal/modelcatalog/catalog.go
      - internal/modelcatalog/pricing_test.go
      - docs/PROVIDERS.md
      - docs/WORKFLOW.md
  - id: "7.E.2"
    name: "Request logging and cost wiring"
    status: DONE
    agent: "orchestrator"
    branch: "codex/wave-7e-logging"
    completed_at: "2026-06-03T02:55:09Z"
    files_owned:
      - e2e_test.go
      - api/server.go
      - api/server_test.go
      - docs/WORKFLOW.md
  - id: "7.E.3"
    name: "Documented /v1/messages and /v1/responses route availability"
    status: DONE
    agent: "orchestrator"
    branch: "codex/wave-7e-routes"
    completed_at: "2026-06-03T02:59:10Z"
    files_owned:
      - api/server.go
      - api/server_test.go
      - api/handlers/inference.go
      - api/handlers/inference_test.go
      - internal/translate/*
      - internal/providers/openai/responses.go
      - docs/WORKFLOW.md
  - id: "7.E.4"
    name: "Tool-call preservation across provider adapters"
    status: DONE
    agent: "orchestrator"
    branch: "codex/wave-7e-tools"
    completed_at: "2026-06-03T03:09:08Z"
    files_owned:
      - internal/providers/anthropic/*
      - internal/providers/gemini/*
      - internal/translate/*
      - docs/WORKFLOW.md
  - id: "7.E.5"
    name: "Combo dispatch hardening, fallback/backoff, and quota gates"
    status: DONE
    agent: "orchestrator"
    branch: "codex/wave-7e-combo"
    completed_at: "2026-06-03T03:20:34Z"
    files_owned:
      - internal/proxy/combo.go
      - internal/proxy/combo_test.go
      - internal/proxy/engine.go
      - internal/proxy/engine_test.go
      - internal/provider/fallback.go
      - internal/provider/fallback_test.go
      - api/server.go
      - api/server_test.go
      - docs/WORKFLOW.md
  - id: "7.E.6"
    name: "Wave 7.E evaluator prompt"
    status: DONE
    agent: "orchestrator"
    completed_at: "2026-06-03T03:22:01Z"
    files_owned:
      - docs/evaluations/wave-7E-evaluator-prompt.md
      - docs/WORKFLOW.md
  - id: "7.E.7"
    name: "Wave 7.E evaluator remediation"
    status: DONE
    agent: "orchestrator"
    branch: "codex/wave-7e-eval-fix"
    completed_at: "2026-06-03T03:40:53Z"
    files_owned:
      - api/handlers/inference.go
      - api/handlers/inference_test.go
      - api/server.go
      - api/server_test.go
      - internal/providers/types.go
      - internal/proxy/engine.go
      - internal/proxy/engine_test.go
      - internal/translate/responses.go
      - internal/translate/responses_test.go
      - docs/PROVIDERS.md
      - docs/evaluations/wave-7E-remediation-evaluator-prompt.md
      - docs/WORKFLOW.md
```

### Wave 7.F — Provider correctness

```yaml
wave: "7.F"
status: DONE
max_agents: 3
depends_on: ["7.E"]
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && make build"

tasks:
  - id: "7.F.1"
    name: "Live upstream streaming for OpenAI, Azure, and OpenAI-compatible providers"
    status: DONE
    agent: "orchestrator"
    branch: "codex/wave-7f-live-streaming"
    completed_at: "2026-06-03T03:56:11Z"
    files_owned:
      - internal/providers/openai/openai.go
      - internal/providers/openai/openai_test.go
      - internal/providers/azure/azure.go
      - internal/providers/azure/azure_test.go
      - internal/providers/openaicompat/provider.go
      - internal/providers/openaicompat/provider_test.go
      - docs/WORKFLOW.md
  - id: "7.F.2"
    name: "Stable sanitized provider error responses"
    status: DONE
    agent: "orchestrator"
    branch: "codex/wave-7f-errors"
    completed_at: "2026-06-03T03:59:22Z"
    files_owned:
      - api/handlers/inference.go
      - api/handlers/inference_test.go
      - internal/cli/root_test.go
      - docs/WORKFLOW.md
  - id: "7.F.3"
    name: "Bedrock Converse downgrade or implementation accuracy"
    status: DONE
    agent: "orchestrator"
    branch: "codex/wave-7f-bedrock-status"
    completed_at: "2026-06-03T04:04:28Z"
    files_owned:
      - api/handlers/providers.go
      - api/handlers/providers_test.go
      - internal/provider/matrix.go
      - internal/provider/matrix_test.go
      - docs/PROVIDERS.md
      - docs/WORKFLOW.md
  - id: "7.F.4"
    name: "Wave 7.F evaluator prompt"
    status: DONE
    agent: "orchestrator"
    branch: "codex/wave-7f-evaluator"
    completed_at: "2026-06-03T04:06:47Z"
    files_owned:
      - docs/evaluations/wave-7F-evaluator-prompt.md
      - docs/WORKFLOW.md
  - id: "7.F.5"
    name: "Wave 7.F evaluator remediation"
    status: DONE
    agent: "orchestrator"
    branch: "codex/wave-7f-eval-remediation"
    completed_at: "2026-06-03T04:19:16Z"
    files_owned:
      - api/handlers/inference.go
      - api/handlers/inference_test.go
      - internal/proxy/errors.go
      - internal/proxy/engine.go
      - internal/proxy/engine_test.go
      - internal/proxy/combo_test.go
      - docs/PROVIDERS.md
      - docs/evaluations/wave-7F-remediation-evaluator-prompt.md
      - docs/WORKFLOW.md
```

### Wave 7.G — Real MCP runtime

```yaml
wave: "7.G"
status: DONE
max_agents: 3
depends_on: ["7.F"]
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && make build"

tasks:
  - id: "7.G.1"
    name: "Real stdio MCP JSON-RPC client"
    status: DONE
    agent: "orchestrator"
    branch: "codex/wave-7g-stdio"
    completed_at: "2026-06-03T04:30:20Z"
    files_owned:
      - internal/mcp/jsonrpc.go
      - internal/mcp/stdio.go
      - internal/mcp/stdio_test.go
      - internal/mcp/process.go
      - internal/mcp/launcher_test.go
      - internal/mcp/advanced_integration_test.go
      - internal/cli/mcp_runtime.go
      - internal/cli/mcp_runtime_test.go
      - docs/WORKFLOW.md
  - id: "7.G.2"
    name: "Real MCP HTTP OAuth token exchange and account selection"
    status: DONE
    agent: "Dirac the 2nd"
    branch: "codex/wave-7g-oauth"
    commit: "a95807a"
    completed_at: "2026-06-03T04:41:43Z"
    files_owned:
      - internal/mcp/oauth.go
      - internal/mcp/oauth_test.go
      - internal/store/mcpoauth.go
      - internal/store/mcpoauth_test.go
      - api/handlers/mcp.go
      - api/handlers/mcp_test.go
      - api/handlers/mcpoauth.go
      - api/handlers/mcpoauth_test.go
      - internal/cli/mcp_auth.go
      - internal/cli/mcp_auth_test.go
  - id: "7.G.3"
    name: "Streamable HTTP and SSE MCP JSON-RPC clients"
    status: DONE
    agent: "orchestrator"
    branch: "codex/wave-7g-http"
    completed_at: "2026-06-03T04:37:11Z"
    files_owned:
      - internal/mcp/http.go
      - internal/mcp/httpclient.go
      - internal/mcp/httpclient_test.go
      - internal/mcp/launcher.go
      - internal/mcp/launcher_test.go
      - internal/cli/mcp_runtime.go
      - internal/cli/mcp_http_runtime_test.go
      - docs/WORKFLOW.md
  - id: "7.G.4"
    name: "Tool manager concurrency, schema validation, and request filtering"
    status: DONE
    agent: "Anscombe the 2nd"
    branch: "codex/wave-7g-tools"
    started_at: "2026-06-03T04:43:40Z"
    commit: "8be8705"
    completed_at: "2026-06-03T04:53:16Z"
    files_owned:
      - internal/mcp/toolmanager.go
      - internal/mcp/toolmanager_test.go
      - internal/mcp/agent.go
      - internal/mcp/agent_test.go
      - api/handlers/mcp.go
      - api/handlers/mcp_test.go
  - id: "7.G.5"
    name: "Startup rehydration, tool sync, health persistence, and evaluator prompt"
    status: DONE
    agent: "orchestrator"
    branch: "codex/wave-7g-rehydrate"
    started_at: "2026-06-03T04:53:16Z"
    completed_at: "2026-06-03T04:53:59Z"
    files_owned:
      - internal/cli/mcp_runtime.go
      - internal/cli/root.go
      - internal/cli/root_test.go
      - internal/mcp/healthmonitor.go
      - internal/mcp/healthmonitor_test.go
      - internal/store/mcpinstances.go
      - internal/store/mcpinstances_test.go
      - docs/evaluations/wave-7G-evaluator-prompt.md
      - docs/WORKFLOW.md
  - id: "7.G.6"
    name: "Wave 7.G evaluator remediation"
    status: DONE
    agent: "orchestrator"
    branch: "codex/wave-7g-eval-fix"
    completed_at: "2026-06-03T05:06:12Z"
    files_owned:
      - api/handlers/mcp.go
      - api/handlers/mcp_test.go
      - internal/cli/root.go
      - internal/cli/root_test.go
      - internal/mcp/oauth.go
      - internal/mcp/oauth_test.go
      - internal/mcp/advanced_integration_test.go
      - docs/evaluations/wave-7G-remediation-evaluator-prompt.md
      - docs/WORKFLOW.md
  - id: "7.G.7"
    name: "Wave 7.G OAuth redirect remediation"
    status: DONE
    agent: "orchestrator"
    branch: "codex/wave-7g-redirect-fix"
    completed_at: "2026-06-03T05:14:58Z"
    files_owned:
      - internal/mcp/oauth.go
      - internal/mcp/oauth_test.go
      - docs/WORKFLOW.md
```

### Wave 7.H — Real dashboard

```yaml
wave: "7.H"
status: DONE
max_agents: 4
depends_on: ["7.G"]
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && make build"

tasks:
  - id: "7.H.0"
    name: "Plan dashboard work slices and ownership"
    status: DONE
    agent: "orchestrator"
    completed_at: "2026-06-03T05:21:34Z"
    files_owned:
      - docs/WORKFLOW.md
  - id: "7.H.1"
    name: "Dashboard API client contracts and shared async states"
    status: DONE
    agent: "orchestrator"
    branch: "codex/wave-7h-api"
    completed_at: "2026-06-03T05:26:23Z"
    files_owned:
      - ui/src/api.ts
      - ui/src/api.test.ts
      - ui/src/components/Primitives.tsx
      - ui/src/components/Primitives.test.tsx
      - ui/src/setupTests.ts
      - ui/src/App.test.tsx
      - ui/dist/**
  - id: "7.H.2"
    name: "Providers and endpoint pages use real API contracts"
    status: DONE
    agent: "Hume the 2nd"
    branch: "codex/wave-7h-providers-endpoint"
    started_at: "2026-06-03T05:29:55Z"
    commit: "973e9a9"
    completed_at: "2026-06-03T05:38:42Z"
    depends_on_tasks: ["7.H.1"]
    files_owned:
      - ui/src/pages/ProvidersPage.tsx
      - ui/src/pages/ProvidersPage.test.tsx
      - ui/src/pages/EndpointPage.tsx
      - ui/src/pages/EndpointPage.test.tsx
  - id: "7.H.3"
    name: "Usage, quota, logs, and overview pages use real API contracts"
    status: DONE
    agent: "Mencius the 2nd"
    branch: "codex/wave-7h-usage-quota-logs"
    started_at: "2026-06-03T05:29:55Z"
    commit: "9c375b1"
    completed_at: "2026-06-03T05:39:39Z"
    depends_on_tasks: ["7.H.1"]
    files_owned:
      - ui/src/pages/UsagePage.tsx
      - ui/src/pages/UsagePage.test.tsx
      - ui/src/pages/QuotaPage.tsx
      - ui/src/pages/QuotaPage.test.tsx
      - ui/src/pages/DashboardPage.tsx
      - ui/src/pages/DashboardPage.test.tsx
  - id: "7.H.4"
    name: "Combos and settings pages use real API contracts"
    status: DONE
    agent: "Peirce the 2nd"
    branch: "codex/wave-7h-combos-settings"
    started_at: "2026-06-03T05:29:55Z"
    commit: "4704dae"
    completed_at: "2026-06-03T05:43:00Z"
    depends_on_tasks: ["7.H.1"]
    files_owned:
      - ui/src/pages/CombosPage.tsx
      - ui/src/pages/CombosPage.test.tsx
      - ui/src/pages/SettingsPage.tsx
      - ui/src/pages/SettingsPage.test.tsx
  - id: "7.H.5"
    name: "MCP dashboard page uses real API contracts without exposing credentials"
    status: DONE
    agent: "Newton the 2nd"
    branch: "codex/wave-7h-mcp-page"
    started_at: "2026-06-03T05:29:55Z"
    commit: "83cce34"
    completed_at: "2026-06-03T05:44:27Z"
    depends_on_tasks: ["7.H.1"]
    files_owned:
      - ui/src/pages/McpPage.tsx
      - ui/src/pages/McpPage.test.tsx
  - id: "7.H.6"
    name: "Dashboard integration, mobile overflow remediation, workflow completion, and evaluator prompt"
    status: DONE
    agent: "orchestrator"
    completed_at: "2026-06-03T05:51:31Z"
    depends_on_tasks: ["7.H.2", "7.H.3", "7.H.4", "7.H.5"]
    files_owned:
      - ui/src/App.tsx
      - ui/src/App.test.tsx
      - ui/src/pages/McpPage.tsx
      - ui/src/pages/McpPage.test.tsx
      - docs/evaluations/wave-7H-evaluator-prompt.md
      - docs/WORKFLOW.md
      - ui/dist/**
  - id: "7.H.7"
    name: "Wave 7.H evaluator remediation"
    status: DONE
    agent: "orchestrator"
    completed_at: "2026-06-03T06:09:00Z"
    depends_on_tasks: ["7.H.6"]
    files_owned:
      - api/handlers/connections.go
      - api/handlers/connections_test.go
      - api/handlers/providers.go
      - api/handlers/providers_test.go
      - ui/src/api.ts
      - ui/src/pages/ProvidersPage.tsx
      - ui/src/pages/ProvidersPage.test.tsx
      - ui/src/pages/EndpointPage.tsx
      - ui/src/pages/EndpointPage.test.tsx
      - ui/src/pages/UsagePage.tsx
      - ui/src/pages/UsagePage.test.tsx
      - ui/src/pages/CombosPage.tsx
      - ui/src/pages/CombosPage.test.tsx
      - docs/evaluations/wave-7H-remediation-evaluator-prompt.md
      - docs/WORKFLOW.md
      - ui/dist/**
```

### Wave 7.I — Usage, cost, logs, and quotas

```yaml
wave: "7.I"
status: DONE
max_agents: 3
depends_on: ["7.H"]
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && make build"

tasks:
  - id: "7.I.1"
    name: "Honor ENABLE_REQUEST_LOGS and log complete request metadata"
    status: DONE
    branch: "codex/wave-7i-logging"
    files_owned:
      - api/middleware.go
      - api/middleware_test.go
      - api/server.go
      - api/server_test.go
      - internal/cli/root.go
      - internal/cli/root_test.go
      - internal/logging/requestlog.go
      - internal/logging/logger_test.go
    phase_doc: "docs/phases/phase-08-usage-tracking-cost-logging.md"
  - id: "7.I.2"
    name: "Expand pricing and model catalog coverage"
    status: DONE
    branch: "codex/wave-7i-catalog"
    files_owned:
      - internal/modelcatalog/catalog.go
      - internal/modelcatalog/pricing.go
      - internal/modelcatalog/pricing_test.go
      - docs/PROVIDERS.md
    phase_doc: "docs/phases/phase-08-usage-tracking-cost-logging.md"
  - id: "7.I.3"
    name: "Enforce quotas across direct models, aliases, fallback, and combos"
    status: DONE
    branch: "codex/wave-7i-quotas"
    files_owned:
      - internal/proxy/engine.go
      - internal/proxy/engine_test.go
      - internal/proxy/combo_test.go
    phase_doc: "docs/phases/phase-08-usage-tracking-cost-logging.md"
  - id: "7.I.4"
    name: "Wave 7.I evaluator prompt"
    status: DONE
    branch: "codex/wave-7i-evaluator"
    depends_on_tasks: ["7.I.1", "7.I.2", "7.I.3"]
    files_owned:
      - docs/evaluations/wave-7I-evaluator-prompt.md
      - docs/WORKFLOW.md
    phase_doc: "docs/phases/phase-08-usage-tracking-cost-logging.md"
  - id: "7.I.5"
    name: "Wave 7.I quota evaluator remediation"
    status: DONE
    branch: "codex/wave-7i-quota-remediation"
    depends_on_tasks: ["7.I.4"]
    files_owned:
      - internal/proxy/engine.go
      - internal/proxy/engine_test.go
      - internal/proxy/combo.go
      - internal/proxy/combo_test.go
      - docs/evaluations/wave-7I-remediation-evaluator-prompt.md
      - docs/WORKFLOW.md
    phase_doc: "docs/phases/phase-08-usage-tracking-cost-logging.md"
```

---

### Wave 7.J — Release readiness hardening

```yaml
wave: "7.J"
status: DONE
max_agents: 3
depends_on: ["7.I"]
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && make build"

tasks:
  - id: "7.J.1"
    name: "Dashboard control-plane authentication"
    status: DONE
    agent: "orchestrator"
    branch: "codex/wave-7j-dashboard-auth"
    commit: "13db10d"
    merged_commit: "3c20d48"
    completed_at: "2026-06-03T17:41:09Z"
    files_owned:
      - ui/src/api.ts
      - ui/src/api.test.ts
      - ui/src/App.tsx
      - ui/src/App.test.tsx
      - ui/src/pages/*.test.tsx
    phase_doc: "docs/phases/phase-10-dashboard-ui.md"
  - id: "7.J.2"
    name: "Self-contained installer and service bootstrap"
    status: DONE
    agent: "orchestrator"
    branch: "codex/wave-7j-installer"
    commit: "d8de874"
    merged_commit: "d7782a0"
    completed_at: "2026-06-03T17:41:09Z"
    files_owned:
      - internal/cli/install.go
      - internal/cli/install_test.go
      - deploy/g0router.default
      - deploy/g0router.service
    phase_doc: "docs/phases/phase-11-packaging-deployment-polish.md"
  - id: "7.J.3"
    name: "Docker release bootstrap and writable data"
    status: DONE
    agent: "orchestrator"
    branch: "codex/wave-7j-docker"
    commit: "677b6ff"
    merged_commit: "35ac82c"
    completed_at: "2026-06-03T17:41:09Z"
    files_owned:
      - Dockerfile
      - docker-compose.yml
      - .dockerignore
      - docs/DEPLOYMENT.md
      - README.md
    phase_doc: "docs/phases/phase-11-packaging-deployment-polish.md"
  - id: "7.J.4"
    name: "Live MCP instance and OAuth lifecycle"
    status: DONE
    agent: "orchestrator"
    branch: "codex/wave-7j-mcp-runtime"
    commit: "407933c"
    merged_commit: "4728044"
    completed_at: "2026-06-03T17:41:09Z"
    files_owned:
      - api/server.go
      - api/server_test.go
      - api/handlers/mcp.go
      - api/handlers/mcp_test.go
      - api/handlers/mcpoauth.go
      - api/handlers/mcpoauth_test.go
      - internal/mcp/*.go
      - internal/store/mcp*.go
    phase_doc: "docs/phases/phase-12-advanced-mcp-gateway.md"
  - id: "7.J.5"
    name: "Wave 7.J evaluator prompt and workflow closure"
    status: DONE
    agent: "orchestrator"
    branch: "codex/wave-7j-evaluator"
    completed_at: "2026-06-03T17:41:09Z"
    depends_on_tasks: ["7.J.1", "7.J.2", "7.J.3", "7.J.4"]
    files_owned:
      - docs/evaluations/wave-7J-evaluator-prompt.md
      - docs/WORKFLOW.md
      - docs/PLAN.md
      - docs/ORCHESTRATION.md
    phase_doc: "docs/phases/phase-11-packaging-deployment-polish.md"

evaluation:
  status: PASS
  completed_at: "2026-06-03T17:53:22Z"
  evaluator_agent: "Wegener"
  non_blocking_findings:
    - "Frontend build output churns tracked ui/dist assets; evaluate deterministic generated output or a cleaner embed build path."
    - "Operator docs should more clearly distinguish dashboard/API-key auth from JWT secret requirements."
    - "MCP instance delete closes runtime before store delete; a store-delete failure can leave a stale row without live tools."
```

---

### Wave 7.K — Release hygiene remediation

```yaml
wave: "7.K"
status: DONE
max_agents: 1
depends_on: ["7.J"]
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && make build && JWT_SECRET=test-jwt API_KEY_SECRET=test-api docker compose config && docker build -t g0router:wave-7k-hygiene-test ."

tasks:
  - id: "7.K.1"
    name: "Resolve Wave 7.J evaluator hygiene findings"
    status: DONE
    agent: "orchestrator"
    branch: "codex/wave-7k-release-hygiene"
    completed_at: "2026-06-03T18:18:36Z"
    files_owned:
      - api/handlers/mcp.go
      - api/handlers/mcp_test.go
      - ui/vite.config.ts
      - ui/dist/**
      - README.md
      - docs/DEPLOYMENT.md
  - id: "7.K.2"
    name: "Wave 7.K evaluator prompt"
    status: DONE
    agent: "orchestrator"
    branch: "codex/wave-7k-release-hygiene"
    completed_at: "2026-06-03T18:18:36Z"
    depends_on_tasks: ["7.K.1"]
    files_owned:
      - docs/evaluations/wave-7K-evaluator-prompt.md
      - docs/WORKFLOW.md

evaluation:
  status: PASS
  completed_at: "2026-06-03T18:26:14Z"
  evaluator_agent: "James"
  prompt: "docs/evaluations/wave-7K-evaluator-prompt.md"
  non_blocking_findings:
    - "MCP instance delete returns 500 if runtime close fails after successful store delete; decide whether to ignore/log runtime-close failures after persistence succeeds."
```

---

### Wave 7.L — Final MCP delete semantics

```yaml
wave: "7.L"
status: DONE
max_agents: 1
depends_on: ["7.K"]
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && make build"

tasks:
  - id: "7.L.1"
    name: "Make post-delete MCP runtime close best-effort"
    status: DONE
    agent: "orchestrator"
    branch: "codex/wave-7l-mcp-delete-close"
    completed_at: "2026-06-03T18:27:33Z"
    files_owned:
      - api/handlers/mcp.go
      - api/handlers/mcp_test.go
  - id: "7.L.2"
    name: "Wave 7.L evaluator prompt"
    status: DONE
    agent: "orchestrator"
    branch: "codex/wave-7l-mcp-delete-close"
    completed_at: "2026-06-03T18:27:33Z"
    depends_on_tasks: ["7.L.1"]
    files_owned:
      - docs/evaluations/wave-7L-evaluator-prompt.md
      - docs/WORKFLOW.md

evaluation:
  status: PASS
  completed_at: "2026-06-03T18:35:15Z"
  evaluator_agent: "Leibniz"
  prompt: "docs/evaluations/wave-7L-evaluator-prompt.md"
  non_blocking_findings:
    - "Node emitted deprecation/experimental warnings during UI test/build; commands exited 0."
```

---

### Wave 7.M — Completion audit remediation and docs reconciliation

```yaml
wave: "7.M"
status: DONE
max_agents: 5
depends_on: ["7.L"]
gate: "go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && npm --prefix ui test -- --run && npm --prefix ui run build && make build"

tasks:
  - id: "7.M.1"
    name: "Alias TTL cache"
    status: DONE
    agent: "Cicero"
    branch: "codex/wave-7m-alias-cache-ttl"
    completed_at: "2026-06-03T19:20:00Z"
    files_owned:
      - internal/proxy/alias_cache.go
      - internal/proxy/alias_cache_test.go
      - internal/proxy/engine.go
    commit: "1dc2159c7ea9e3b293cf47b6b31530e36bc70153"
  - id: "7.M.2"
    name: "Pricing override cost integration"
    status: DONE
    agent: "orchestrator"
    branch: "codex/wave-7m-pricing-overrides"
    completed_at: "2026-06-03T19:31:00Z"
    files_owned:
      - internal/usage/cost.go
      - internal/usage/cost_test.go
      - internal/store/pricing.go
      - api/server.go
      - api/server_test.go
    commit: "07b63b3647a647d603d02ab7222c81d972686b93"
  - id: "7.M.3"
    name: "Quota fetch cache"
    status: DONE
    agent: "Dalton"
    branch: "codex/wave-7m-quota-cache"
    completed_at: "2026-06-03T19:25:00Z"
    files_owned:
      - internal/usage/quota.go
      - internal/usage/quota_test.go
      - internal/cli/root.go
      - internal/cli/root_test.go
    commit: "39f575aa4af07e553f784474ad7a8e17d817b93c"
  - id: "7.M.4"
    name: "Periodic MCP health checks"
    status: DONE
    agent: "Helmholtz"
    branch: "codex/wave-7m-mcp-health-monitor"
    completed_at: "2026-06-03T19:28:00Z"
    files_owned:
      - internal/mcp/healthmonitor.go
      - internal/mcp/healthmonitor_test.go
      - internal/mcp/toolmanager.go
    commit: "4ae4de4706d80e389ce711d38a47db4749bf4732"
  - id: "7.M.5"
    name: "Alias, pricing, and connection-test management APIs"
    status: DONE
    agent: "orchestrator"
    branch: "codex/wave-7m-management-api"
    completed_at: "2026-06-03T19:16:00Z"
    files_owned:
      - api/handlers/aliases.go
      - api/handlers/aliases_test.go
      - api/handlers/pricing.go
      - api/handlers/pricing_test.go
      - api/handlers/connections.go
      - api/server.go
      - api/server_test.go
    commit: "d40f7d0e30cae1e86b66eb2895be2716228ef2f3"
  - id: "7.M.6"
    name: "Docs completion reconciliation and evaluator prompt"
    status: DONE
    agent: "orchestrator"
    branch: "codex/wave-7m-docs-reconcile"
    completed_at: "2026-06-03T19:40:00Z"
    depends_on_tasks: ["7.M.1", "7.M.2", "7.M.3", "7.M.4", "7.M.5"]
    files_owned:
      - docs/README.md
      - docs/PLAN.md
      - docs/SCHEMA.md
      - docs/CONFIG.md
      - docs/DEPLOYMENT.md
      - docs/ORCHESTRATION.md
      - docs/WORKFLOW.md
      - docs/phases/*.md
      - docs/evaluations/wave-7M-evaluator-prompt.md

gate_results:
  - command: "go test ./... -count=1"
    status: PASS
  - command: "go vet ./..."
    status: PASS
  - command: "go build ./cmd/g0router"
    status: PASS
  - command: "npm --prefix ui test -- --run"
    status: PASS
    notes: "Node emitted deprecation/experimental warnings; exit code 0."
  - command: "npm --prefix ui run build"
    status: PASS
    notes: "Node emitted deprecation warning; exit code 0."
  - command: "make build"
    status: PASS

evaluation:
  status: PASS
  completed_at: "2026-06-03T19:58:00Z"
  evaluator_agent: "McClintock"
  prompt: "docs/evaluations/wave-7M-evaluator-prompt.md"
  non_blocking_findings:
    - "Node emitted deprecation/experimental localStorage warnings during UI test/build; commands exited 0."
    - "POST /api/connections/:id/test is a stored-row/is_active health check, not a live upstream credential probe."
```

---

## Verification Protocol

Before marking any task DONE:

```bash
go test ./... -count=1    # All tests pass
go vet ./...              # Clean
go build ./cmd/g0router   # Binary builds
```

## Recovery Protocol

If project is in a broken state:

1. `go test ./...` → identify failures
2. `git log --oneline -10` → last good commit
3. Read WORKFLOW.md → identify the active wave/task if one exists
4. Fix failing tests before proceeding
5. Never skip a broken test — fix or revert


---

## Stage 12B-19 — DDD Refactor + UI Overhaul: 9Router + Bifrost Feature Integration

> **Status**: PLANNING COMPLETE — phase docs + process doc + Lovable prompt written. Agentic-engineering framework adopted (strict profile); Plan-tier artifacts per phase under `docs/planning/<phase-slug>/` (Skeptic-reviewed, sign-off granted). Next: execute Phase 12B per `docs/planning/phase-12b-ddd-architecture-refactor/`.
> **Started**: 2026-06-05
> **Goal**: Refactor project to layered DDD architecture, then implement ~55 new backend features from 9Router and Bifrost. Lovable generates the new UI in parallel; integration deferred to phases 20-21.

### Process
Single source of truth for execution process, cross-cutting contracts
(snake_case, response envelope, audit, secrets-at-rest, feature flags,
architecture layers), gates, and the per-phase checkpoint protocol:

**`docs/phases/STAGE-13-19-PROCESS.md`** — read it before any phase work.

### Phase Plan

| Phase | Name | Doc | Status |
|-------|------|-----|--------|
| 12B | DDD & Architecture Refactor (whole project) | `phase-12b-ddd-architecture-refactor.md` | PENDING |
| 13 | Auth & Core Infrastructure | `phase-13-auth-core-infrastructure.md` | PENDING |
| 14 | Providers & Testing | `phase-14-providers-testing.md` | PENDING |
| 15 | Tunnels & Network | `phase-15-tunnels-network.md` | PENDING |
| 16 | Chat & Console | `phase-16-chat-console.md` | PENDING |
| 17 | Usage & Analytics | `phase-17-usage-analytics.md` | PENDING |
| 18 | Bifrost Features (sub-stages 18A-18D) | `phase-18-bifrost-features.md` | PENDING |
| 19 | Advanced Features | `phase-19-advanced-features.md` | IN_PROGRESS |
| 20 | Lovable UI Generation | prompt: `docs/lovable-prompt.md` (DONE) — generation PENDING (user-driven) | PENDING |
| 21 | UI Integration & Gates | TBD after Lovable output | PENDING |

Execution order is strict: 12B before 13; 13 before all others (auth
foundation); 14-17 may reorder if needed; 18 before 19. Checkpoint protocol
(gates, WORKFLOW update, `## Outcome` section) at every phase end — see
process doc §4.

### Deferred (decided during planning — see phase docs for rationale)
- Adaptive routing heuristic (duplicate of existing `auto` strategy classifier)
- OTel distributed tracing
- WebRTC realtime (WebSocket only)
- 33-locale i18n (en + pt-BR real; rest fall back)
- Automatic /etc/hosts editing for MITM (manual instructions instead)
- Tailscale binary auto-install (drives preinstalled binary only)

### New DB Tables
- `dashboard_users`, `dashboard_sessions` (Phase 13)
- `proxy_pools`, `disabled_models`, `custom_models` (Phase 14)
- `tunnel_config` (Phase 15)
- `chat_sessions` (Phase 16)
- `teams`, `virtual_keys` (Phase 18A)
- `routing_rules`, `model_limits` (Phase 18B)
- `prompt_templates`, `mcp_tool_groups` (Phase 18C; guardrails config lives in settings)
- `alert_channels`, `feature_flags` (Phase 18D)
- `semantic_cache` (Phase 19)

### Gates
Per-commit: `go test ./... -count=1 && go vet ./... && go build ./cmd/g0router`
Per-phase (checkpoint): adds `go test -race ./...` + coverage ≥ 95.0%.
UI gates (`npm --prefix ui test -- --run`, `npm --prefix ui run build`) only
when `ui/` touched. Stage exit: `make e2e-binary` + gitleaks clean.

### Commit Pattern
`phase-12b/task-1: routing table extraction`
`phase-13/task-1: dashboard users store with bcrypt`


---

## Hotfix — API/UI contract alignment

```yaml
wave: "hotfix"
status: DONE
summary: "Aligned backend response contracts with React dashboard TypeScript types. Fixed providers list blank screen, connections shape, admin models envelope, API keys full_key exposure, virtual key id/prefix types, and seeded default tunnel configs on first run."
completed_at: "2026-06-08T21:15:00Z"
```

**Files changed:**
- `api/handlers/providers.go` — return UI-facing `providerListItem` with `display_name`, `description`, `capabilities`, `connection_count`, `status`; detail response embeds same fields plus `matrix_info`.
- `api/handlers/connections.go` — snake_case JSON tags; derive `models` from `ModelLocks`; add `priority`, `proxy_id`, `last_error`, ISO `expires_at`/`unavailable_until`.
- `api/handlers/models_admin.go` — wrap response in `{data: ...}` envelope.
- `api/handlers/apikeys.go` — return flat `apiKeyView` with `full_key` on create/put/regenerate; accept `rpm_limit`/`tpm_limit`/`daily_spend_cap` UI field names with legacy aliases.
- `api/handlers/virtualkeys.go` — return `id` as string, `prefix` instead of `key_prefix`, `team_id` as string.
- `internal/cli/root.go` — seed default `cloudflare` and `tailscale` tunnel configs on first run.
- Tests updated across `api/handlers/*_test.go`, `api/server_integration_test.go`, `e2e_api_comprehensive_test.go`.

**Gate results:** `go test ./...` and `go build ./cmd/g0router` pass.

---

## Hotfix — Provider icons from 9Router

```yaml
wave: "hotfix"
status: DONE
summary: "Borrowed 34 provider PNG icons from 9Router (MIT-licensed) and wired them into the dashboard. ProviderIcon now renders actual logos with gradient+initials fallback. Backend /api/providers and /api/providers/:id return icon_url."
completed_at: "2026-06-08T21:30:00Z"
```

**Files changed:**
- `ui/public/providers/` — downloaded 34 PNG icon assets from https://github.com/decolua/9router/tree/master/public/providers (MIT License).
- `ui/public/providers/LICENSE-9Router-icons.txt` — attribution and full MIT license text.
- `ui/src/components/common/ProviderIcon.tsx` — render `<img>` from `iconUrl` prop or local mapping; `onError` falls back to existing gradient + initials.
- `ui/src/routes/_app.providers.index.tsx` — pass `iconUrl={p.icon_url}` to `ProviderIcon`.
- `ui/src/routes/_app.providers.$id.tsx` — pass `iconUrl={provider.icon_url}` to `ProviderIcon`.
- `api/handlers/providers.go` — added `icon_url` to `providerListItem` and `providerDetailResponse`; added `providerIconPaths` mapping g0router provider IDs to `/providers/*.png` assets.

**Gate results:** `go test ./...`, `go vet ./...`, `go build -o g0router ./cmd/g0router`, and `npm run build` in `ui/` all pass. Server restarted at http://127.0.0.1:20128 with icons verified (`/providers/openai.png` serves correctly; `/api/providers` returns `"icon_url":"/providers/openai.png"`).

---

## Hotfix — New connection button on provider detail

```yaml
wave: "hotfix"
status: DONE
summary: "The 'New connection' button on the provider detail page was a no-op. Replaced it with a working dialog that POSTs to /api/connections and supports name, auth-type selection, credential input, and active toggle."
completed_at: "2026-06-08T21:35:00Z"
```

**Files changed:**
- `ui/src/routes/_app.providers.$id.tsx` — added `CreateConnectionDialog` component using Radix Dialog, Select, Switch, and Input; wired 'New connection' button `onClick` to open it. On success the dialog invalidates provider + connection queries so the list refreshes.

**Gate results:** `npm run build` in `ui/` passes; `go build -o g0router ./cmd/g0router` passes. Server restarted at http://127.0.0.1:20128. Verified `POST /api/connections` with the same payload returns a created connection.

---

## Hotfix — Remove all placeholders, mocks, and no-ops from the dashboard

```yaml
wave: "hotfix"
status: DONE
summary: "Audited the entire React dashboard for placeholders and no-op interactions, then wired every screen, button, form, and dialog to real backend APIs via parallel subagents. Replaced 14 `<ComingSoon />` routes with working pages, fixed chat auth and session persistence, added connection edit/delete, fixed routing-rules conditions, added key regenerate and missing form fields, wired MITM routes, and implemented a real Diagnostics page."
completed_at: "2026-06-08T22:15:00Z"
```

**Files changed (high-level):**
- `ui/src/routes/login.tsx` — removed prefilled demo credentials and demo banner.
- `ui/src/components/layout/Header.tsx` — removed no-op notifications bell.
- `ui/src/components/layout/Sidebar.tsx` — live version from `GET /api/version`.
- `ui/src/routes/_app.chat.tsx` — removed hard-coded provider/model, fixed auth to use `full_key`, wired recent sessions and new-chat persistence.
- `ui/src/routes/_app.providers.$id.tsx` — added connection edit/delete, suggested models.
- `ui/src/routes/_app.connections.tsx` — added connection edit action.
- `ui/src/routes/_app.routing-rules.tsx` — added `cond_field`, `cond_operator`, `cond_value` fields aligned with backend.
- `ui/src/routes/_app.keys.tsx` — added regenerate action, scopes, and expires_at fields.
- `ui/src/routes/_app.teams.tsx` — added budget_period and rate_limit_rpm, removed blank columns.
- `ui/src/routes/_app.endpoint.tsx` — removed broken client-side audit, dynamic curl sample.
- `ui/src/routes/_app.logs.tsx` — switched from `/api/usage` to `/api/logs`.
- `ui/src/routes/_app.alerts.tsx`, `_app.guardrails.tsx`, `_app.model-limits.tsx`, `_app.prompts.tsx`, `_app.feature-flags.tsx`, `_app.proxy-pools.tsx`, `_app.mcp.*.tsx`, `_app.skills.tsx` — replaced `<ComingSoon />` with working CRUD/feature pages.
- `ui/src/routes/_app.mitm.tsx` + `api/routes.go` + `api/server.go` — wired MITM backend routes and implemented MITM control page.
- `ui/src/routes/_app.diagnostics.tsx` — implemented a real diagnostics page with version, API connectivity, auth status, browser info, provider/connection counts.
- `ui/src/lib/types.ts` — aligned types with backend contracts, removed mock comments.
- `ui/src/lib/lovable-error-reporting.ts` — deleted; import removed from `__root.tsx`.
- `ui/src/routes/_app.dashboard.tsx.bak` — deleted.
- Supporting changes in `ui/src/components/common/CrudPage.tsx`, `ui/src/components/connections/EditConnectionDialog.tsx`, `ui/e2e/mocks/seed.ts`, `api/routes_test.go`, `e2e_api_comprehensive_test.go`.

**Gate results:** `go test ./...`, `go vet ./...`, `go build -o g0router ./cmd/g0router`, and `npm run build` in `ui/` all pass. Zero `<ComingSoon />` occurrences remain in `ui/src/routes/`. Server verified at http://127.0.0.1:20128.

## Wave 1 (w1-a..f) — diff-gate closure decision

```yaml
wave: "parity-w1"
status: GATES-CLOSED-BY-DECISION
summary: "Translation core (w1-a..f) merged with all behavioral diff-gate findings fixed across ~20 gpt-5.5 review rounds. Gate loops for w1-c/d/e/f closed by documented orchestrator/planner decision after findings decayed to style nibbles and ref-contradicting false positives; per-round artifacts and rebuttals retained."
completed_at: "2026-06-10T23:30:00Z"
```

**Behavioral defects found by gates and fixed** (none re-flagged after fix):
role-assert panic; message_stop cache-token loss; `AdjustMaxTokens` early
returns skipping tools-floor/budget-bump sequence (2 rounds); enum `String()`
parity (`jsString`); nondeterministic antigravity tool order; ignored
`json.Marshal` errors (×3 sites); `uuid.Must`/globals/`math/rand` removal;
enabled-only temperature drop (PAR-PR-1264); filter `tool_calls` truthiness +
bare-name tool guard (openaiHelper.js:20,88); vacuous Azure-strip tests
replaced with presence-in/absence-out (schema fields ratified deviation);
SSE scanner EOF contract; ~25 test-coverage additions.

**Standing rebuttals** (ref-verified false positives, do not re-litigate):
`plans/fixes/w1-def-gate-fixes-r3-2026-06-10.md` Appendix A + w1-c v3 footer.

**Closure rationale**: harness rule caps reject loops at 3 cycles before
escalation; loops ran 5-7 cycles each. Final-round verdicts rejected on test
comments ("// user \"hello\"") and a claim contradicting the frozen ref
(`hasValuableContent` default-true — streamHelpers.js:39,61). Escalated to
operator 2026-06-10; closure proceeded per announced default. Matrix rows
flipped: 33 PAR-TRANS rows → HAVE; 002/046 PARTIAL (clauses owned by
w1-g..j / Wave 5).

**Artifacts**: `.planning/harness/artifacts/*-diff-scoped-gpt.txt`, fix plans
`plans/fixes/*2026-06-10.md`, impl reports `artifacts/*-impl-report.md`.

## w1-h ollama+commandcode — diff gate CLEAN (2026-06-11)

```yaml
wave: "parity-w1"
plan: "w1-h-ollama-commandcode"
status: MERGED
rows: "PAR-TRANS-058,059,060,061 → HAVE"
commits: "14c971b (impl), bc6358d (gate fixes)"
```
Diff gate: 3 real findings (dataURIPattern global, tc.index field, CommandCode
usage omission) all fixed in bc6358d, none re-flagged. Final re-run's sole
"BLOCKER" was a false positive — it cited a non-existent `parity-w0/` commit
format; the commit correctly uses `parity-w1/w1-h:` per AGENTS.md, and
commit-message format is outside diff-gate scope. Accepted by decision.

## w1-g Responses API — diff gate PASS (2026-06-11)

```yaml
wave: "parity-w1"
plan: "w1-g-responses-api"
status: MERGED
rows: "PAR-TRANS-031..038 → HAVE"
commits: "6640b33 (impl), ca8274e (gate fixes)"
```
Both responses↔chat translator directions, event-based SSE with sequence
numbers, reasoning buffering, tool lifecycle, usage extraction. Diff gate: dead
code + test gap fixed in ca8274e; hosted-tool finding rebutted ref-faithful
(request/openai-responses.js:156-176 filters nameless only). PASS.

## w1-i kiro pair — diff gate CLEAN (2026-06-11)

```yaml
wave: "parity-w1"
plan: "w1-i-kiro-pair"
status: MERGED
rows: "PAR-TRANS-062,063 → HAVE"
commits: "0347b41 (impl, two-job), 956b09c (gate fixes)"
```
Diff gate: 4 real findings (assistantResponseEvent.content vs textDelta,
tool-input stringify, uuid test, test Printf) all fixed in 956b09c, none
re-flagged. Final finding was the recurring `parity-w0/` false positive —
ROOT CAUSE FIXED: critic-diff.md:10 hardcoded `parity-w0/` as the required
commit format; corrected to make commit-message format out-of-scope for the
diff gate (the orchestrator owns it; project format is `parity-w1/...`).
Accepted by decision.

## w1-j cursor pair — diff gate PASS (2026-06-11)

```yaml
wave: "parity-w1"
plan: "w1-j-cursor-pair"
status: MERGED
rows: "PAR-TRANS-064,065 → HAVE"
commits: "82e88bd (impl), d51064d + 7d04e10 (gate-fix tests)"
```
Diff gate: BLOCKER #2 (tool-role precedence test coverage) closed across two
test-only fixes; BLOCKER #1 (inline normalize) and the passthrough-branch MINOR
rebutted ref-faithful (dual-key meta map; ref-verbatim passthrough branches).
PASS. (Protobuf/checksum = Wave-2 executor scope.)

## w1-k gemini client request — diff gate PASS (2026-06-11)

```yaml
wave: "parity-w1"
plan: "w1-k-gemini-client"
status: MERGED
rows: "PAR-TRANS-066 → HAVE"
commits: "f94a58a (impl), c108609+4228da2+a41cdc5+640c505 (gate fixes)"
```
Diff gate: 4 rounds on functionResponse truthiness — tool-call id format, registry
wiring assertions, then the full result||response||{} chain with jsTruthy matching
JS Boolean (empty {} and [] truthy). One round corrected a planner error (fix3
wrongly made empty collections falsey). PASS. (strip/dedupe/inject helpers → Wave 4.)

## w1-l claude pipeline helpers — diff gate CLOSED BY DECISION (2026-06-11)

```yaml
wave: "parity-w1"
plan: "w1-l-claude-pipeline-helpers"
status: MERGED
rows: "PAR-TRANS-022,054,055 → HAVE"
commits: "c11a2fd (impl), c0f42d9 + 238c02c + cfb1730 (gate fixes)"
```
Rounds 1-3 fixed all real findings: bypass source-format responses, escaped
naming JSON (json.Marshal), translation-error propagation (no silent OpenAI
downgrade), reg==nil error for non-OpenAI source, comma-ok assertions, +tests.
Round 4 was entirely FALSE POSITIVES proposing frozen-ref divergence — all three
rebutted with ref citations:
- cloakClaudeTools suffixes ALL tool_use names unconditionally per claudeCloaking.js:54-58 (comment: "all client tools get suffix").
- applyCloaking string-system branch has no skip-check per claudeCloaking.js:142-143 (a string is never a prior injection, which is always an array).
- bypass title pattern is array-indexed content[0].text=="{" per bypassHandler.js:29 (a string "{" does not match — by design).
Closed by decision (4 rounds; substance sound; round-4 contradicts the ref).

## Wave 1 — Translation engine COMPLETE (2026-06-11)

```yaml
wave: "parity-w1"
status: COMPLETE
summary: "Full 9router translation engine ported to Go: 12 wire formats, openai-intermediate pipeline, SSE stream processor (translate+passthrough), cloud-code envelopes, claude pipeline helpers, /v1/responses endpoint. All translator/format/endpoint PAR-TRANS rows HAVE; remaining 6 rows deferred to later waves by design."
plans_done: [w1-a,w1-b,w1-c,w1-d,w1-e,w1-f,w1-g,w1-g2,w1-h,w1-i,w1-j,w1-k,w1-l]
rows_have: 61
gates: "go test ./... + go vet ./... green; build OK"
```

**Per-plan close**: w1-a/b PASS-early; w1-c/d/e/f gates closed by decision after all
behavioral findings fixed; w1-g(031-038), w1-h(058-061), w1-i(062-063),
w1-j(064-065), w1-k(066), w1-g2(050,031-route) diff-gate PASS; w1-l(022/054/055)
closed by decision (round-4 false positives rebutted ref-cited).

**Deferred (not Wave-1 scope)**:
- PAR-TRANS-006 / 051 / 052 / 053 → Wave 4 (`w4-pre`): stripContentTypes,
  injectReasoningContent, deepseek-v4-pro alias, dedupeTools — request-pipeline
  preprocessing; ship WITH their routing integration.
- PAR-TRANS-050b → Wave 2: passthrough-mode response.failed (needs the Responses
  passthrough provider path).
- PAR-TRANS-046 usage-estimation clause → Wave 5 (PAR-USAGE accounting stack).

**Process notes**: ~3 matrix-vs-ref corrections caught (PAR-TRANS-033 clamp
direction, `_cc`→`_ide` suffix, CC_DEFAULT_TOOLS 26→20) + 1 critic-prompt bug
fixed (parity-w0 hallucination). Granular rows PAR-TRANS-058..066 added for
per-translator tracking. Helpers 006/051/052/053 reassigned to Wave 4.

## Wave 2 — Provider adapters (Stage-1 scope) COMPLETE (2026-06-11)

Stage-1 "Include now" providers per matrix ranking (10): deepseek, groq, mistral,
together, fireworks, cohere, xai (API-key path), openrouter, perplexity, ollama.
Plans_done: w2-a (catalog+model catalogs), w2-b (generic OpenAI-compatible adapter,
collapses 9 providers), w2-c (ollama adapter, NDJSON), w2-d (catalog-driven router +
factory + /v1/models aggregation). Rows_have: PAR-PROV-004..010, 014, 027, 029 (10).
All diff-gates clean (w2-a/b/c closed-by-decision after substantive findings fixed;
w2-d PASS). Suite green incl. `go test -race ./internal/inference/`.

Key gate catches: dropped PAR-PR-664 (max_completion_tokens — github/qoder/codex only,
Stage-2); xai OAuth + providerSpecificData override deferred to Wave 3; deterministic
provider-precedence for dup model IDs; lazy-cache concurrency fix (sync.RWMutex).

Deferred to later (recorded in WAVE-2-MAP "Deferred to Stage 2+"):
- /v1/models ranges catalog.Providers — fine now (catalog holds only the 10 Stage-1
  providers); when Stage-2 providers are ADDED to the catalog, /v1/models needs a
  "wired/enabled" filter so it doesn't expose unrouted providers. (Wave-future note.)
- All Stage-2+ provider classes: OAuth, custom-format/reverse-engineered, GCP,
  Chinese ecosystems, media specialists, free-tier, no-op.

## Wave 3 — OAuth + auth hardening (Stage-1 scope) COMPLETE (2026-06-11)

Plans_done: w3-a (login hardening: limiter/lockout/default-pw/auth-mode/reset-CLI),
w3-b (central dashboard guard + local-only gate + tunnel toggles), w3-c (OIDC login
PKCE + probe), w3-d (API keys + /v1 gating + CLI token), w3-e (outbound env-proxy),
w3-f (provider OAuth: anthropic/gemini/xai + credential refresh + key resolution).
Rows_have: PAR-AUTH-002,005-015,019,021-023,026-029 (20 → HAVE); 020 → PARTIAL
(env-proxy half; MITM-DNS-bypass deferred to Stage-2/W7). All diff-gates resolved
(PASS or closed-by-decision after fixes). Suite + `go test -race` green.

Security holes caught & fixed by the diff gate (highest-value catches of the program):
- w3-c: public OIDC secret-probe endpoint was reading the SERVER's stored
  oidc_client_secret → restricted to caller-provided values only.
- w3-d: x-9r-cli-token granted remote /v1 LLM access → narrowed to loopback-or-API-key
  (CLI token valid for /api protected routes only).
Other real fixes: limiter remaining-count semantics, single-flight refresh race,
xai scope %20 encoding, Router.keyResolver race, credentials metadata-error handling,
proxy dialer host:port, ParseAPIKey shape enforcement. GitHub push-protection block on
the Gemini public client-secret resolved (env-overridable, runtime-assembled).

Deferred (recorded in WAVE-3-MAP §Deferred): PAR-AUTH-017/018 → Wave 5 (request_log +
debug-log substrates land there); PAR-AUTH-003 closed by decision 2 (opaque tokens, no
JWT); PR-1711 closed by decision 2; ~11 Stage-2 provider OAuth handlers + PR-717/641/
1388/1458/1004/665 → Stage 2 with their adapters.

## Wave 4 — Routing (Stage-1 scope) IN PROGRESS (2026-06-12)

### w4-pre — Audit wiring fixes + Wave-1 deferred pipeline helpers COMPLETE (2026-06-12)

Plans_done: w4-pre (audit G1-G6 fixes + PAR-TRANS-006/051/052/053 helpers).
Rows_flipped: PAR-TRANS-051/052/053 → HAVE; PAR-TRANS-006 → PARTIAL (function wired,
Stage-1 schema has string Content so no-op until Stage-2 multi-part content support).
Fixes: G1/G2 (credential resolver + gemini/xai flows wired into server.go), G3 (models/{id}
filter + 404), G4 (randomUUID returns error not placeholder), G5 (stream abort ctx.Done
select in ProcessPassthroughStream), G6 (stale phase comments removed).
Diff-gate: CLOSED BY DECISION after 2 cycles (architectural constraints: server.New
creates infRouter without injectable seam; Stage-1 schema has string Content only; stream
abort landed in translation/stream.go which was the correct minimal change). Suite +
`go test -race` GREEN.

Next: w4-a (aliases) ∥ w4-b (errors/retry) ∥ w4-c (connection-state) — launching parallel.

### w4-a — Model & provider aliases + prefix resolution COMPLETE (2026-06-12)

Plans_done: w4-a (PAR-ROUTE-005/006/007/008/010 + PAR-PR-485).
Rows_flipped: PAR-ROUTE-005/006/008/010 MISSING→HAVE; PAR-ROUTE-007 PARTIAL→HAVE.
Commits: 8637945 (prefix+inference+PR-485), 119e41e (apiKeyGenerator wiring), b2d1f93
(aliasStore wiring), 2a23f85 (unexport ProviderAliases + accessors), b977370 (DFS cycle
detection), ca6fd72 (isBuiltinProvider guard), 6b57543 (InferProvider sort longest-first).
Diff-gate: CLOSED BY DECISION after 4 cycles. Real bugs fixed: aliasStore wiring (cycle 1
BLOCKER), ProviderAliases unexport (cycle 2), DFS ResolveChain + cc→claude guard (cycle 3),
InferProvider nondeterminism (cycle 4). Residual cycle-4 findings are architectural
constraints (router wiring is the fix, error passthrough is intentional, cc→claude is
Stage-2 scope). Suite + go vet GREEN.

### w4-b — Error classification + retry middleware COMPLETE (2026-06-12)

Plans_done: w4-b (PAR-ROUTE-020/021/022/044/045/048 + PAR-PR-1626).
Rows_flipped: PAR-ROUTE-020/021/022/044/045/048 MISSING→HAVE.
Commits: e01e4ef (error classifier), 05be5c5 (retry middleware), 325d2a2 (PR-1626
token-param auto-learn), 8fa6e6e (ClassUnsupportedParam), b0a63fd (kiro retry override),
790dc85 (SetSetting error propagation), 2c4a944 (remove mutable globals), c361b18 (store
integration test), 0199af5 (TestErrorClassFixture verbatim), 1e4d02b (catalog test),
12ad3b1 (classificationRules function), 511bd60 (fixture order + TestErrorClassRuleOrder),
cd8f997 (remove dead fmt.Stringer).
Diff-gate: CLOSED BY DECISION after 4 cycles. Real bugs fixed: ClassUnsupportedParam +
mutable global (cycle 1), fixture order (cycle 2), SetSetting propagation (cycle 3), dead
fmt.Stringer code (cycle 4). Residual: connect-timeout fasthttp port constraint; GetSetting
swallowing intentional design. Suite + go vet GREEN.

Next: w4-c (connection-state) — NOW UNBLOCKED (migrate.go free; w4-a merged).

### w4-c — Connection/account state: locks, backoff, disabled models COMPLETE (2026-06-12)

Plans_done: w4-c (PAR-ROUTE-012/013/014/015/025/026/049).
Rows_flipped: PAR-ROUTE-012/013/014/015/025/026/049 MISSING→HAVE.
Commits: 70a3bc4 (initial), c0c813a (FIX1: providerID scope + rate_limited_until + snake_case),
794dce4 (FIX2: migrate.go provider_id + admin tests), d312de8 (FIX3: GroupRetryAfter error propagation).
Diff-gate: CLOSED BY DECISION after 3 cycles. Real bugs fixed: providerID scoping in EarliestExpiry +
rate_limited_until write + IsDisabled error propagation + DisabledChecker wiring + snake_case API
(cycle 2), migrate.go provider_id commit + admin disabledmodels tests (cycle 2 carry-over),
GroupRetryAfter error swallowing (cycle 3). Residual cycle-3 finding: backoff 2s/5min vs plan
comment 1s/4min — REBUTTAL: plan comment is a transcription error; actual accountFallback.js:9-13
uses base=2000ms and max=5*60*1000ms; implementation matches ref. Suite + go vet + go test -race GREEN.

### w4-d — Selection engine + account fallback loop COMPLETE (2026-06-12)

Plans_done: w4-d (PAR-ROUTE-016/017/018/019/050/051 + PAR-PR-640).
Rows_flipped: PAR-ROUTE-016/017/018/019/050/051 MISSING→HAVE. PAR-PR-640 tracked in PARITY.md.
Commits: 41ab54c (initial selection engine + tests), bc1ee23 (FIX1: error propagation +
atomic mutex test), 025f1ec (FIX2: exhaustion test coverage), 2af418a (FIX3: precise
retry-after assertion).
Diff-gate: CLOSED BY DECISION after 3 cycles. Real bugs fixed: resolveStrategy GetSetting error
swallowing + atomic mutex serialization test (cycle 2), precise retry-after substring assertion
(cycle 3). Residual: ErrAllUnavailable wrapping satisfies contract (both sentinel + retry time);
accountStrategy key doesn't exist in frozen ref. Suite + go vet + go test -race GREEN.

Next: w4-e (combos).

### w4-e — Combo chains COMPLETE (2026-06-12)

Plans_done: w4-e (PAR-ROUTE-001/002/003/004/011/024/046/047 + PAR-PR-648).
Rows_flipped: PAR-ROUTE-001/002/003/004/011/024/046/047 MISSING→HAVE. PAR-PR-648 tracked in PARITY.md.
Commits: 4e549c6 (initial), bb6a324 (routes wiring), 6f29db8 (FIX1), 85eaef7 (FIX2).
Diff-gate: CLOSED BY DECISION after cycle 3.
  Cycle-1 FIXED: production wiring, transient cooldown semantics, regex.
  Cycle-2 FIXED: ComboLister layering (api→store removed), ErrModelTransient sentinel,
    sticky-limit default-1 test, per-combo comboStrategies override test.
  Cycle-3 BLOCKER rebutted: ErrModelTransient production wiring (real runner wrapping
    502/503/504) is intentionally deferred to w4-f pipeline glue wave. Combo engine
    contract is fully defined and tested; interface boundary is the correct split point.

Next: w4-f (pipeline glue).

## w4-f: Pipeline Glue (2026-06-12)
Plan: .planning/parity/plans/w4-f-pipeline-glue.md
PAR rows: PAR-ROUTE-023/052 (refresh-retry), PAR-ROUTE-033 (format auto-detect),
  PAR-ROUTE-034 (bypass wiring), PAR-ROUTE-037 (kind routes), PAR-ROUTE-038 (model-test-by-kind),
  PAR-ROUTE-041 (native passthrough), PAR-ROUTE-042 (thinking override),
  PAR-ROUTE-043 (stream decision), PAR-TRANS-050b (DetectFormat), PAR-028/029/035/036 (verify-flips).
Rows_flipped: PAR-ROUTE-023/033/034/036/037/038/041/042/043/052 MISSING→HAVE.
  PAR-ROUTE-035 MISSING→PARTIAL (single URL, no fallback). PAR-028/029 MISSING→HAVE.
  PAR-TRANS-050b: remains MISSING (Wave 2 — responses passthrough executor deferred).
Commits: 8f18271 (initial), 5d504d4 (FIX1: model-test route, native passthrough before translate,
  scope fix), eb950ae (FIX2: refresh semantics, catalog type assertions, cycle-2 rebuttal),
  3edad62 (cycle-3 close-by-decision).
Diff-gate: CLOSED BY DECISION after cycle 3.
  Cycle-1 FIXED: model-test-by-kind route, native passthrough restructured to resolve
    provider before translation, bypass_handler.go scope reverted (DetectFormat inlined).
  Cycle-2 FIXED: refresh-retry to 3 dispatch cycles, TestRefreshRetryUpTo3On401 tightened,
    TestModelsByKind catalog-type assertions added.
  Cycle-3 BLOCKER rebutted: SetCredentialRefresher production wiring deferred to OAuth wave
    (requires external HTTP token exchange; nil guard provides graceful degradation).
  MAJOR rebuttals: routing regression (false — routing only uses req.Model, unchanged by
    translation); GetTestByKind metadata (architectural adaptation, pingModelByKind is BFF).

Next: Wave-4 complete. Next phase per HANDOFF.md.

## Wave 5: Planning + w5-pre/w5-a (2026-06-12)
Map: .planning/parity/plans/WAVE-5-MAP.md — 8 micro-plans (w5-pre, a-g).
Scope: PAR-USAGE 36 full + 032/033 Stage-1 partial (036/037→W6); PAR-ROUTE-030/031/054;
  PAR-TRANS-046 usage clause; PAR-AUTH-017/018; W4 carry-forward debts.
Plan gates: w5-pre PASS (c3), w5-c PASS (c3), w5-a/b/d/e/f/g closed-by-decision after
  3 cycles each (per-finding triage appended to each plan).
Impl order: w5-pre ALONE → w5-a ALONE → (w5-b ∥ w5-c) → (w5-d → w5-e) ∥ w5-f → w5-g.

## w5-pre: Debt Closure (2026-06-12)
Plan: .planning/parity/plans/w5-pre-debt-closure.md
Closes: SetCredentialRefresher production caller (RefreshCredentials by connection ID,
  wired in routes_openai), production AccountRunner (ErrModelTransient wrap 502/503/504),
  combo dispatch glue in chat path (Cooldown→Selection→Runner→ComboEngine chain in server.New).
Rows_flipped: none (makes PAR-ROUTE-023 + combo rows real in production).
Commits: 94b9812, 878aa64, 53cd537. Tests+race green.
Diff-gate: cycle-1 REJECT → fix micro-plan fixes/w5-pre-fix-r1.md (production bridge test,
  RegisterOpenAIRoutes plumb test, refresher compile assertion) — dispatch queued.

## w5-a: Schema + Pricing Engine (2026-06-12)
Plan: .planning/parity/plans/w5-a-schema-pricing.md
Delivered: request_log/usage_daily/request_details/kv tables (+8 indexes), kv accessors,
  pricing data (83 models/1 provider/49 patterns, golden-tested), glob matcher + 3-step
  resolution + user-override-first, merged view + 5s cache + Invalidate, token
  normalization + 5-category cost calc.
Rows pending flip (after fix-r1 + gate close): PAR-USAGE-004..010, 040 (+001/002/003 tables).
Commits: 2f28678, eaa00a5, 7e5b2ea, 76d6c9d, dfd09a7, 353bc98. Tests+race green.
Diff-gate: cycle-1 REJECT — REAL: missing store UserPricing() reader, unwrapped errors,
  user-override baseModel over-match → fix micro-plan fixes/w5-a-fix-r1.md queued.
