# Parallel Agent Orchestration

> Master coordination document for multi-agent execution of g0router.
> Any orchestrator (human, OMC team, Claude Code, Codex) reads this to dispatch work.

---

## Execution Model

Work is organized into **Stages** (sequential barriers) containing **Waves** (parallel groups).
Within a wave, every task can run on its own agent simultaneously — zero shared writes.

**Concurrency budget**: Up to 8 agents at once is the sweet spot. More risks merge churn.

---

## Stage / Wave / Task Breakdown

### STAGE 0 — Bootstrap (sequential, 1 agent)

| Wave | Task | Agent Scope | Files (exclusive write) |
|------|------|-------------|------------------------|
| 0.A | 0.1 Go module + docs | Agent-0 | `go.mod`, `cmd/g0router/main.go`, `.gitignore`, `.env.example`, all `docs/**` |

**Gate**: `go build ./cmd/g0router && go vet ./...`

---

### STAGE 1 — Core Foundation (parallel, up to 3 agents)

Phase 1 tasks split into two waves. Wave 1.A has no intra-dependencies. Wave 1.B depends on 1.2 (SQLite store).

#### Wave 1.A — Independent foundation (3 agents)

| Task | Agent | Files (exclusive write) | Depends on |
|------|-------|------------------------|------------|
| 1.1 Core types | Agent-1 | `internal/providers/types.go`, `types_test.go`, `interface.go` | — |
| 1.2 SQLite store | Agent-2 | `internal/store/sqlite.go`, `sqlite_test.go` | — |
| 1.6 Config loading | Agent-3 | `internal/config/config.go`, `config_test.go` | — |

**Gate**: merge all 3 → `go test ./... && go vet ./...`

#### Wave 1.B — Store CRUD (3 agents, after 1.A merge)

| Task | Agent | Files (exclusive write) | Depends on |
|------|-------|------------------------|------------|
| 1.3 Connection CRUD | Agent-1 | `internal/store/connections.go`, `connections_test.go`, `errors.go` | 1.2 |
| 1.4 Settings + API keys | Agent-2 | `internal/store/settings.go`, `settings_test.go`, `apikeys.go`, `apikeys_test.go` | 1.2 |
| 1.5 Usage log store | Agent-3 | `internal/store/usage.go`, `usage_test.go` | 1.2 |

**Gate**: merge all 3 → `go test ./... && go vet ./...` → **PHASE_1_COMPLETE**

---

### STAGE 2 — Server + Parallel Streams (up to 8 agents)

After Phase 1, five independent streams can start simultaneously.

#### Wave 2.A — HTTP foundations + Phase 7/8 starts (up to 8 agents)

| Task | Agent | Files (exclusive write) | Depends on |
|------|-------|------------------------|------------|
| 2.1 Server skeleton | Agent-1 | `api/server.go`, `server_test.go`, `api/handlers/health.go` | Phase 1 |
| 2.3 Proxy engine | Agent-2 | `internal/proxy/engine.go`, `pool.go`, `engine_test.go` | Phase 1 |
| 2.4 OpenAI provider | Agent-3 | `internal/providers/openai/*.go` | Phase 1 |
| 2.5 Provider utils | Agent-4 | `internal/providers/utils/*.go` | Phase 1 |
| 2.6 Streaming accum | Agent-5 | `internal/streaming/*.go` | Phase 1 |
| 5.1 OAuth types | Agent-6 | `internal/provider/oauth/types.go`, `types_test.go` | Phase 1 |
| 7.1 RTK autodetect | Agent-7 | `internal/rtk/autodetect.go`, `autodetect_test.go`, `constants.go` | Phase 1 |
| 8.1+8.2 Usage+pricing | Agent-8 | `internal/usage/tracker.go`, `tracker_test.go`, `internal/modelcatalog/*.go` | Phase 1 |

**Gate**: merge all → `go test ./... && go vet ./...`

#### Wave 2.B — Middleware + dependent tasks (up to 8 agents)

| Task | Agent | Files (exclusive write) | Depends on |
|------|-------|------------------------|------------|
| 2.2 Middleware | Agent-1 | `api/middleware.go`, `middleware_test.go` | 2.1 |
| 4.1 Provider registry | Agent-2 | `internal/provider/registry.go`, `registry_test.go` | Phase 1 |
| 4.2 Connection mgmt | Agent-3 | `internal/provider/connection.go`, `connection_test.go` | Phase 1 |
| 7.2 RTK filters (11) | Agent-4 | `internal/rtk/filters/*.go` | 7.1 |
| 7.4 Caveman injection | Agent-5 | `internal/rtk/caveman.go`, `caveman_test.go`, `prompts.go` | Phase 1 |
| 8.3 Cost calculation | Agent-6 | `internal/usage/cost.go`, `cost_test.go` | 8.1, 8.2 |
| 5.2 Anthropic OAuth | Agent-7 | `internal/provider/oauth/anthropic.go`, test | 5.1 |
| 5.3 Codex OAuth | Agent-8 | `internal/provider/oauth/codex.go`, test | 5.1 |

**Gate**: merge all → `go test ./... && go vet ./...`

#### Wave 2.C — Integration + more parallel work (up to 8 agents)

| Task | Agent | Files (exclusive write) | Depends on |
|------|-------|------------------------|------------|
| 2.7 Inference handler | Agent-1 | `api/handlers/inference.go`, `inference_test.go`, `models.go` | 2.1, 2.2, 2.3, 2.5, 2.6 |
| 7.3 RTK compression | Agent-2 | `internal/rtk/rtk.go`, `rtk_test.go` | 7.1, 7.2 |
| 8.4 Quota fetchers | Agent-3 | `internal/usage/quota.go`, test | Phase 1 |
| 8.5 Request logging | Agent-4 | `internal/logging/*.go` | Phase 1 |
| 5.4 GitHub OAuth | Agent-5 | `internal/provider/oauth/github.go`, test | 5.1 |
| 5.5 Cursor OAuth | Agent-6 | `internal/provider/oauth/cursor.go`, test | 5.1 |
| 5.6 Google OAuth | Agent-7 | `internal/provider/oauth/gemini.go`, `antigravity.go`, tests | 5.1 |
| 4.3 Combos store | Agent-8 | `internal/store/combos.go`, `combos_test.go`, `internal/proxy/combo.go` | Phase 1 |

**Gate**: merge all → `go test ./... && go vet ./...` → **PHASE_2_COMPLETE**, **PHASE_7_COMPLETE**

---

### STAGE 3 — Providers + Registry + OAuth + MCP + UI (up to 8 agents)

#### Wave 3.A — Provider implementations (up to 8 agents)

| Task | Agent | Files (exclusive write) | Depends on |
|------|-------|------------------------|------------|
| 3.1 Anthropic provider | Agent-1 | `internal/providers/anthropic/*.go` | Phase 2 |
| 3.2 Translation engine | Agent-2 | `internal/translate/detect.go`, `openai.go`, `anthropic.go`, tests | Phase 2 |
| 3.3 OpenAI-compat batch | Agent-3 | `internal/providers/openaicompat/*.go` | Phase 2 |
| 3.4 Gemini provider | Agent-4 | `internal/providers/gemini/*.go` | Phase 2 |
| 3.7 Bedrock provider | Agent-5 | `internal/providers/bedrock/*.go` | Phase 2 |
| 3.8 Azure provider | Agent-6 | `internal/providers/azure/*.go` | Phase 2 |
| 9.1+9.2 MCP client+tool | Agent-7 | `internal/mcp/clientmanager.go`, `toolmanager.go`, tests | Phase 2 |
| 10.1 UI scaffold | Agent-8 | `ui/**` | Phase 2 |

**Gate**: merge → test

#### Wave 3.B — Translation + remaining providers + OAuth + management (up to 8 agents)

| Task | Agent | Files (exclusive write) | Depends on |
|------|-------|------------------------|------------|
| 3.5 Gemini translation | Agent-1 | `internal/translate/gemini.go`, `gemini_test.go` | 3.2, 3.4 |
| 3.6 Vertex provider | Agent-2 | `internal/providers/vertex/*.go` | 3.4 |
| 3.9 Mistral+Ollama+etc | Agent-3 | respective provider dirs | Phase 2 |
| 4.4 Aliases + pricing | Agent-4 | `internal/store/aliases.go`, `pricing.go`, tests | Phase 1 |
| 5.7 xAI+DeepSeek+etc | Agent-5 | `internal/provider/oauth/xai.go`, `deepseek.go`, etc | 5.1 |
| 5.8 Chinese providers | Agent-6 | `internal/provider/oauth/kimi.go`, `minimax.go`, etc | 5.1 |
| 5.9 Token refresh | Agent-7 | `internal/provider/refresh.go`, `refresh_test.go` | 5.1 |
| 10.2-10.7 UI pages | Agent-8 | `ui/src/pages/*.tsx` | 10.1 |

**Gate**: merge → test → **PHASE_3_COMPLETE** (when 3.1–3.10 done)

---

### STAGE 4 — Integration + Final Features (up to 6 agents)

#### Wave 4.A — Handler integration + remaining tasks

| Task | Agent | Files (exclusive write) | Depends on |
|------|-------|------------------------|------------|
| 4.5 Management API | Agent-1 | `api/handlers/providers.go`, `connections.go`, `settings.go`, `apikeys.go`, `combos.go` | 4.1–4.4 |
| 5.10 OAuth endpoints | Agent-2 | `api/handlers/oauth.go`, test | 5.1–5.9 |
| 5.11 CLI (cobra) | Agent-3 | `internal/cli/*.go`, `cmd/g0router/main.go` | Phase 1 |
| 6.1 Fallback engine | Agent-4 | `internal/provider/fallback.go`, `fallback_test.go` | Phase 2, 4.2 |
| 6.2 Combo resolution | Agent-5 | `internal/proxy/combo.go` (extend), test | Phase 2, 4.3 |
| 8.6 Usage API handlers | Agent-6 | `api/handlers/usage.go`, `logging.go`, tests | 8.1–8.5 |

**Gate**: merge → test → **PHASE_4_COMPLETE**, **PHASE_5_COMPLETE**, **PHASE_6_COMPLETE**, **PHASE_8_COMPLETE**

#### Wave 4.B — MCP completion + Responses API + UI embed

| Task | Agent | Files (exclusive write) | Depends on |
|------|-------|------------------------|------------|
| 3.10 Responses API | Agent-1 | `internal/providers/openai/responses.go`, `internal/streaming/responses.go`, `internal/translate/responses.go` | Phase 2 |
| 9.3 MCP discovery | Agent-2 | `internal/mcp/discovery.go`, test | 9.1, 9.2 |
| 9.4 MCP agent loop | Agent-3 | `internal/mcp/agent.go`, test | 9.1, 9.2 |
| 9.5 MCP health | Agent-4 | `internal/mcp/healthmonitor.go`, test | 9.1 |
| 9.6 MCP handlers | Agent-5 | `api/handlers/mcp.go`, `internal/store/mcpclients.go`, tests | 9.1–9.5 |
| 10.8 Embed UI | Agent-6 | `embed.go`, update `api/server.go` | 10.1–10.7 |

**Gate**: merge → test → **PHASE_9_COMPLETE**, **PHASE_10_COMPLETE**

---

### STAGE 5 — Packaging (sequential or 3 agents)

| Task | Agent | Files (exclusive write) | Depends on |
|------|-------|------------------------|------------|
| 11.1 Makefile | Agent-1 | `Makefile` | all |
| 11.2 systemd + install | Agent-2 | `deploy/*`, `internal/cli/install.go`, test | all |
| 11.3 Docker | Agent-3 | `Dockerfile`, `docker-compose.yml`, `.dockerignore` | all |
| 11.4 README polish | Agent-1 | `README.md`, `.env.example`, `docs/DEPLOYMENT.md` | all |
| 11.5 E2E tests | Agent-2 | E2E test file | all |

**Gate**: `make test && make build && make docker` → **PHASE_11_COMPLETE** → advance to Stage 6

---

### STAGE 6 — Advanced MCP Gateway (sequential, 1 agent)

Runs only after all existing phases are complete. This stage upgrades the Phase 9 MCP base into a production MCP gateway with configurable instances, docker/npx/http launches, OAuth accounts, and pasted callback URL completion.

| Task | Agent | Files (exclusive write) | Depends on |
|------|-------|------------------------|------------|
| 12.1 MCP instance model + store | Agent-1 | `internal/store/mcpinstances.go`, `internal/store/mcpinstances_test.go`, `internal/mcp/instances.go`, `internal/mcp/instances_test.go` | Phase 11 |
| 12.2 MCP launcher matrix | Agent-1 | `internal/mcp/launcher.go`, `internal/mcp/launcher_test.go`, `internal/mcp/process.go`, `internal/mcp/http.go` | 12.1 |
| 12.3 MCP OAuth account engine | Agent-1 | `internal/mcp/oauth.go`, `internal/mcp/oauth_test.go`, `internal/store/mcpoauth.go`, `internal/store/mcpoauth_test.go` | 12.1 |
| 12.4 MCP OAuth callback completion | Agent-1 | `api/handlers/mcpoauth.go`, `api/handlers/mcpoauth_test.go`, `internal/cli/mcp_auth.go`, CLI tests | 12.3 |
| 12.5 MCP management surfaces | Agent-1 | `api/handlers/mcp.go`, MCP API tests, `internal/cli/mcp.go`, `ui/src/pages/*`, UI MCP components | 12.1-12.4 |
| 12.6 Advanced MCP integration docs | Agent-1 | MCP integration tests, `docs/SCHEMA.md`, `docs/CONFIG.md`, `docs/DEPLOYMENT.md`, `README.md` | 12.2-12.5 |

**Gate**: `go test ./... && go vet ./... && go build ./cmd/g0router` → **PHASE_12_COMPLETE** → **PROJECT COMPLETE**

---

## Conflict Prevention Rules

### Rule 1: Exclusive File Ownership

Every file is owned by exactly ONE task in each wave. If the ownership map above shows a file, only that agent writes it. All others may read it.

### Rule 2: Package Boundary

Each agent's work compiles independently within its package. Tests for package X only import package X's public API. No circular imports.

### Rule 3: Interface Contracts

When Agent-A produces an interface and Agent-B consumes it:
- Agent-A writes the interface FIRST (Wave N)
- Agent-B implements against it LATER (Wave N+1 or later)
- Interface files are read-only once merged

Key interfaces defined in Wave 1.A:
- `providers.Provider` — consumed by all provider implementations
- `store.Store` methods — consumed by handlers, engine, CLI

### Rule 4: Test Isolation

Each `_test.go` file uses `t.TempDir()` for any filesystem state. No shared test databases. No hardcoded ports (use `:0` and `ln.Addr()`).

### Rule 5: Merge Order

Within a wave, task branches merge in task-number order (1.1, 1.2, 1.3...) to make conflicts deterministic. The orchestrator resolves any conflicts before proceeding.

---

## Agent Dispatch Template

When spawning an agent, provide this context:

```
## Your Assignment

Task: {task_id} — {task_name}
Phase doc: docs/phases/phase-{NN}-{slug}.md
Wave: {wave_id}

## Files You Own (exclusive write)
{file_list}

## Files You May Read
{read_list}

## Pre-conditions
{what_must_exist_before_you_start}

## Deliverables
1. Test file(s) — written FIRST, must fail initially
2. Implementation file(s) — minimum to pass tests
3. `go test ./... && go vet ./...` must pass
4. One commit: `phase-{N}/task-{M}: {description}`

## Constraints
- Do NOT modify files outside your ownership list
- Do NOT add dependencies not listed in the phase doc
- Do NOT import packages from tasks in the same wave that haven't merged yet
- Use interfaces from already-merged waves only
```

---

## Orchestrator Checklist (per wave)

```
- [x] All prerequisite waves merged and green
- [x] Stage branch up to date
- [x] Dispatch agents with task assignments
- [x] Monitor: all agents report completion
- [x] Merge task branches in order: task/N.1, task/N.2, ...
- [x] Run: go test ./... && go vet ./... && go build ./cmd/g0router
- [x] If RED: identify conflict file, assign fix agent
- [x] If GREEN: update WORKFLOW.md statuses
- [x] Advance to next wave
```

---

## Timeline Summary

| Stage | Waves | Max parallel agents | Tasks completed |
|-------|-------|--------------------|----|
| 0 | 1 | 1 | 1 |
| 1 | 2 | 3 | 6 |
| 2 | 3 | 8 | 22 |
| 3 | 2 | 8 | 16 |
| 4 | 2 | 6 | 12 |
| 5 | 2 | 3 | 5 |
| 6 | 1 | 1 | 6 |
| 7 | 13 | 8 | remediation Waves 7.A–7.M |
| 8 | 40 | 8 | completion hardening Waves 8.A–8.AN |
| **Total** | **66 waves** | — | **77 original tasks + remediation + completion hardening** |

With 8 agents, the original 77 tasks compressed into ~13 sequential merge
points instead of 77. Stage 7 then ran the principal-engineer remediation, and
Stage 8 records completion hardening, integration coverage, optional live smoke,
follow-up audit remediation, dashboard route hardening, public route integration
coverage, MCP OAuth parity, connection mutation integration, no-auth provider
runtime dispatch, selected MCP OAuth account binding, MCP OAuth client
credential propagation, dashboard MCP instance launch fields, streamable HTTP
MCP initialize params, and dashboard MCP OAuth resource discovery through Wave
8.AN.
