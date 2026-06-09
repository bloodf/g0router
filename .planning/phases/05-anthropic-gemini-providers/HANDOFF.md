# Handoff: Continue Autonomous Development to Working V1.0

**Date:** 2026-06-09
**Current Phase:** 5 COMPLETE
**Next Phase:** 6 (Management API Foundation)
**Milestone:** v2.0 9router + BiFrost Clean Slate Port

---

## What Is Done (Phases 1–5)

### Phase 1: Scaffolding ✅
- Clean-slate directory structure under `internal/`
- Minimal `cmd/g0router/main.go` with fasthttp skeleton
- Empty placeholder tests in all packages

### Phase 2: Schemas + Catalog ✅
- `internal/schemas/` — 14 files covering all OpenAI wire types (chat, completions, embeddings, images, audio, files, batch, responses, errors, provider interface, governance, catalog, MCP stubs)
- JSON round-trip tests for all schema types
- `Provider` interface with 25+ methods

### Phase 3: OpenAI Provider ✅
- `internal/providers/openai/` — reference provider
- Chat (non-streaming + streaming SSE), embeddings, list models
- `ErrorConverter` for OpenAI error envelope
- Shared utilities: `ClientPool`, `SSEScanner`, JSON helpers in `internal/providers/utils/`
- `notImplemented` stubs for all future methods

### Phase 4: OpenAI API Handlers ✅
- fasthttp server with `httprouter`, CORS, request ID middleware
- `POST /v1/chat/completions`, `POST /v1/embeddings`, `GET /v1/models`
- Stream branching with SSE output and `[DONE]` terminator

### Phase 5: Anthropic + Gemini Providers ✅
- `internal/providers/anthropic/` — Messages API converter, streaming, error converter + tests
- `internal/providers/gemini/` — generateContent/embedContent converters, streaming, error converter + tests
- Router updated with prefix-based resolution (`anthropic/`, `gemini/`, `claude-*`, `gemini-*`)
- **Key architectural decision:** API keys flow from the router/WebUI key store. No env var injection. Empty keys pass through and yield provider auth errors until the management layer is wired.

---

## Architecture Patterns (MUST FOLLOW)

### Provider Pattern
Each provider lives in `internal/providers/{name}/` with:
- `provider.go` — struct, constructor, `GetProvider()`, `SetNetworkConfig()`
- `chat.go` — `ChatCompletion` + `ChatCompletionStream`
- `converter.go` — provider-native request/response types + `ConvertRequest`/`ConvertResponse`
- `errors.go` — `ErrorConverter` with provider-specific error envelope parsing
- `stubs.go` — all unimplemented Provider interface methods return `notImplemented` errors
- `{name}_test.go` — TDD tests for converters, stubs, error handling

### HTTP Layer
- fasthttp + `github.com/fasthttp/router`
- Handlers in `internal/api/` (chat, embeddings, models)
- Route registration in `internal/server/routes_openai.go`
- Middleware chain: `RequestIDMiddleware` → `CORSMiddleware`

### Router
- `internal/inference/router.go` — resolves model string → `(Provider, Key, error)`
- Phase 5: prefix-based resolution. Phase 8: virtual-key routing with weighted selection and fallbacks.

### Keys
- `schemas.Key{ID, Provider, Value}` — Value comes from the WebUI/key store
- No env var injection. Phase 6 builds the SQLite-backed key store.

---

## Development Conventions (STRICT)

Read `AGENTS.md` for full details. Key rules:

1. **TDD always.** Write the test first. Watch it fail. Write minimal code to pass. Never skip the RED phase.
2. **Every package gets `_test.go` files before implementation.**
3. **`go test ./...` and `go vet ./...` must pass green at every commit.**
4. **No mocks** — use interfaces and fakes; test real behavior.
5. **No `init()` functions** — explicit initialization via constructors.
6. **Errors are values** — return `error`, never panic; wrap with `fmt.Errorf("context: %w", err)`.
7. **No global state** — pass dependencies via struct fields or function params.
8. **Naming:** `camelCase` locals, `PascalCase` exports; package names lowercase singular nouns.
9. **Commit format:** `phase-NN/task-N: <description>`
10. **No PR workflow** — commit and push directly to `main`; quality gates run locally before every push.
11. **Update `docs/WORKFLOW.md`** after completing any task.
12. **Update `.planning/STATE.md`** at phase boundaries.

---

## Remaining Phases (6–19)

### Wave 2: Core Providers + Admin

**Phase 6: Management API Foundation**
- Auth (login/session), settings CRUD, provider CRUD, connection CRUD
- SQLite persistence with encrypted secrets (`*_enc` columns)
- OAuth start/callback/refresh for at least one provider
- Target: `internal/admin/` or `internal/api/admin/` with protected routes

**Phase 7: Dashboard Shell + Providers UI**
- React dashboard: login page, shell layout, providers list/detail, connections page
- Port from 9router WebUI patterns; adapt to existing Vite/React/Tailwind stack in `ui/`
- Management API client in the frontend

**Phase 8: Keys + Virtual Keys + Routing**
- API key CRUD, virtual key CRUD with provider configs/weights/key IDs
- `x-g0-vk` header routing with weighted selection and automatic fallback chains
- Update router to use the key store instead of prefix-only resolution

### Wave 3: Catalog + Usage

**Phase 9: Models + Aliases + Combos**
- Model alias CRUD, custom pricing overrides, combo CRUD for fallback chains
- Dashboard models page and combos page

**Phase 10: Usage + Logs**
- Request logging (`request_log` table), cost calculation, usage aggregation
- Dashboard usage charts and logs filtering

### Wave 4: Advanced API Surface

**Phase 11: Audio + Images**
- `POST /v1/audio/speech`, `/v1/audio/transcriptions`
- `POST /v1/images/generations`
- Provider implementations for OpenAI + Gemini

**Phase 12: Responses API + Batch + Files**
- `POST /v1/responses` (OpenAI Responses API)
- File upload/list/retrieve/delete/content
- Batch create/list/retrieve/cancel

### Wave 5: 9router Features

**Phase 13: RTK + Caveman + Translator**
- RTK compression for tool results, Caveman mode, translator debug UI

**Phase 14: MCP Gateway**
- MCP client registration, instance management (STDIO/SSE), tool groups

**Phase 15: Proxy Pools + Provider Nodes**
- Proxy pool CRUD, provider node registration/heartbeat

**Phase 16: Cloud Sync**
- Config export/import with encryption

**Phase 17: Additional Providers**
- Groq, Mistral, Cohere, DeepSeek, MiniMax, Fireworks, Together, Ollama, Bedrock, Vertex
- Most are OpenAI-compatible passthroughs; Bedrock/Vertex need custom converters

### Wave 6: Hardening + Ship

**Phase 18: E2E Hardening**
- Mock API layer in `ui/e2e/mocks/`
- Playwright coverage for every dashboard page/action
- E2E tests run without real provider keys

**Phase 19: Polish + Docs**
- Final QA, documentation updates, deployment verification
- Docker, systemd configs validated
- Single binary starts and serves API + embedded UI

---

## Immediate Next Steps (Phase 6)

1. Read `.planning/phases/06-management-api-foundation/` — create PLAN.md if missing
2. Design SQLite schema for: users/sessions, settings, providers, connections (with `*_enc` secret columns)
3. Implement auth endpoints: `POST /api/auth/login`, `POST /api/auth/logout`, session middleware
4. Implement settings CRUD: `GET/PUT /api/settings`
5. Implement provider CRUD: `GET/POST/PUT/DELETE /api/providers`
6. Implement connection CRUD: `GET/POST/PUT/DELETE /api/connections` with encrypted secrets
7. Add admin route registration in `internal/server/`
8. All with TDD, tests first

---

## Critical Context

### GSD Framework
This project uses the GSD (Get-Shit-Done) framework. Key files:
- `.planning/STATE.md` — current phase and status
- `.planning/ROADMAP.md` — all 19 phases with goals and success criteria
- `.planning/phases/NN-name/PLAN.md` — per-phase plan
- `.planning/phases/NN-name/SUMMARY.md` — per-phase summary (created at completion)
- `docs/WORKFLOW.md` — task log

Use the GSD skills when working:
- `$gsd-progress` — check current state
- `$gsd-plan-phase {N}` — plan the next phase
- `$gsd-execute-phase {N}` — execute all plans in a phase
- `$gsd-next` — auto-advance to next logical step

### Provider Interface
Located in `internal/schemas/provider.go`. All providers must implement every method. Unimplemented methods return `notImplemented` errors (501). This allows incremental provider development.

### SQLite Store
Pattern: additive-only `ensureColumn` migrations. Secrets encrypted at rest via reversible `*_enc` columns. See `internal/store/oauthsessions.go` for the encryption pattern.

### Dashboard Stack
- `ui/` — Vite + React 19 + Tailwind 4 + shadcn/ui
- Has its own `AGENTS.md` with deeper context
- Embedded into the Go binary via `embed.FS` at build time

### Testing
- Go: `go test ./...` must pass
- Go vet: `go vet ./...` must pass
- UI build: `cd ui && npm run build` must pass (when working on UI phases)
- Playwright: `cd ui && npx playwright test` must pass (Phase 18+)

---

## Success Criteria for V1.0

A "working V1.0" means:

1. **OpenAI-compatible API works** — `/v1/chat/completions`, `/v1/embeddings`, `/v1/models` with OpenAI, Anthropic, and Gemini providers
2. **Management API works** — auth, settings, providers, connections, keys, virtual keys
3. **Dashboard is functional** — login, providers, connections, keys, models, usage pages
4. **Routing works** — virtual keys route to correct providers with weights and fallbacks
5. **Single binary runs** — `go build ./cmd/g0router` produces one binary that serves API + UI
6. **All gates pass** — `go test ./...`, `go vet ./...`, `npm run build`, Playwright tests
7. **No env var dependencies** — all configuration and keys via WebUI/SQLite

---

## Start Command

```bash
# Verify current state
cd /Users/heitor/Developer/github.com/bloodf/g0router
go test ./... && go vet ./...

# Check GSD progress
gsd-progress

# Start Phase 6
gsd-plan-phase 6
gsd-execute-phase 6
```

---

*This handoff was generated after Phase 5 completion. All phases 1–5 are committed to `main` and all gates pass.*
