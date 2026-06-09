# GSD State

## Current Position

**Milestone:** v2.0 9router + BiFrost Clean Slate Port
**Phase:** 6 (Management API Foundation) — COMPLETE
**Plan:** `.planning/phases/06-management-api-foundation/PLAN.md`
**Status:** Ready for next phase

**Last activity:** 2026-06-09 — Phase 6 Management API Foundation complete. SQLite store (WAL, additive-only ensureColumn migrations, AES-256-GCM encrypted *_enc secret columns, auto-generated key file), auth package (PBKDF2 hashing, session manager, PKCE OAuth flow for Anthropic), admin handlers (login/logout/me, settings, provider CRUD, connection CRUD with masked secrets, OAuth start/callback/refresh), all wired behind RequireSession middleware at /api/*. Default admin seeded on first run. All gates pass.

---

## Phase 1 — Scaffolding — Deliverables

| Commit | Subject |
|---|---|
| `6338148` | phase-01/task-1: remove obsolete api/, internal/, and root e2e tests |
| `63124ba` | phase-01/task-2: scaffold internal/ package layout with placeholder tests |
| `c900b55` | phase-01/task-3: rewrite cmd/g0router/main.go as minimal fasthttp skeleton |
| `e36a19c` | phase-01/task-4: go mod tidy |
| `79db515` | phase-01/task-1: scaffold minimal UI placeholder (main.tsx, App.tsx, index.css) |

---

## Phase 2 — Schemas + Catalog — Deliverables

| Commit | Subject |
|---|---|
| `cde0e77` | phase-02/task-1: core OpenAI-compatible schema types (chat, completions, embeddings) + round-trip tests |
| `f7cc17f` | phase-02/task-2: extended schema types (images, audio, files, batch, responses, errors) + round-trip tests |
| `d1f2cee` | phase-02/task-3: provider interface, governance, catalog, and MCP stub types + compile check |

---

## Phase 3 — OpenAI Provider — Deliverables

| Commit | Subject |
|---|---|
| `ee8c48a` | phase-03/task-1: OpenAI provider (chat, embeddings, models, streaming) + utils + tests |

---

## Phase 4 — OpenAI API Handlers — Deliverables

| Commit | Subject |
|---|---|
| (to be added on commit) | phase-04/task-1: fasthttp server with CORS, request ID, chat/embeddings/models handlers |

---

## Phase 5 — Anthropic + Gemini Providers — Deliverables

| Commit | Subject |
|---|---|
| (to be added on commit) | phase-05/task-1: Anthropic provider with chat converter, streaming, error converter + tests |
| (to be added on commit) | phase-05/task-2: Gemini provider with chat/embedding converters, streaming, error converter + tests |
| (to be added on commit) | phase-05/task-3: update router with prefix-based Anthropic/Gemini resolution + env key helpers |

---

## Phase 6 — Management API Foundation — Deliverables

| Commit | Subject |
|---|---|
| `cbad6f80` | phase-06/task-1: SQLite store with WAL, additive migrations, AES-GCM cipher, and repositories (users, sessions, settings, providers, connections, oauth sessions) |
| `e1a4a869` | phase-06/task-2: auth package with PBKDF2 password hashing, session manager, and PKCE OAuth flow (Anthropic) |
| `31d0f43c` | phase-06/task-3: admin handlers — login/logout/me, settings, provider CRUD, connection CRUD with masked secrets, OAuth start/callback/refresh |
| `35618972` | phase-06/task-4: wire admin routes into fasthttp server; main.go opens store, seeds default admin, serves management API |
| `7c5d4a82` | phase-06/task-5: end-to-end management API integration test (login, settings, provider/connection CRUD, encrypted secrets, OAuth flow, logout) |

---

## Accumulated Context

- Clean-slate pivot from previous g0router architecture. Phase 1 wipes the v1 code (`api/`, `internal/`, `ui/src/`) and the v1-era root `e2e_*.go` files.
- Design spec approved: `docs/superpowers/specs/2026-06-08-9router-bifrost-port-design.md`
- BiFrost patterns adopted for OpenAI-compatible layer.
- 9router features targeted for management layer and dashboard.
- 19 phases grouped into 6 execution waves.
- Phase 1 leaves `go.mod` with a single direct dep (`github.com/valyala/fasthttp v1.71.0`).
- Phase 2 fills `internal/schemas/` with all shared wire-format types.
- Phase 3 implements the reference OpenAI provider in `internal/providers/openai/` with fasthttp, SSE streaming, and shared utilities.
- Phase 4 exposes `/v1/chat/completions`, `/v1/embeddings`, and `/v1/models` via fasthttp handlers.
- Phase 5 adds converter-based Anthropic and Gemini providers with format translation.
- Phase 6 adds the management API: SQLite store (modernc.org/sqlite, WAL, additive-only migrations, encrypted `*_enc` columns), session auth, settings/provider/connection CRUD, PKCE OAuth (Anthropic). Auth routes live at `/api/auth/*`. Encryption key auto-generated at `<datadir>/secret.key`; data dir defaults to `~/.g0router` (override `G0ROUTER_DATA`). Default admin `admin`/`123456` seeded on first run only.

---

## Active Blockers

_None._

---

## Next Step

Continue **Wave 2: Core Providers + Admin** with **Phase 7: Dashboard Shell + Providers UI**.

React dashboard: login page, shell layout, providers list/detail, connections page. Management API client targeting `/api/auth/login`, `/api/settings`, `/api/providers`, `/api/connections`, `/api/oauth/{provider}/*`.

### What "next phase" should keep in mind
- `go test ./...` and `go vet ./...` must pass green at every commit; `cd ui && npm run build` for UI work.
- Commit format: `phase-07/task-N: <description>`.
- Connection responses mask secrets (`secret_set` / `access_token_set` booleans) — the UI never echoes secrets; empty secret fields on update preserve stored values.
- Default admin is `admin`/`123456`; a password-change endpoint still needs to be added.
