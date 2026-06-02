# Master Implementation Plan

> **g0router** — Go LLM gateway combining 9router + bifrost + oh-my-pi.
> Each phase is a standalone implementation guide in `docs/phases/`.
> Parallel execution is coordinated via `docs/ORCHESTRATION.md` and `docs/WORKFLOW.md`.

## Methodology

- **TDD**: Every task starts with tests. Tests fail first (red), minimum code to pass (green), then refactor.
- **Phase gates**: `go test ./...` + `go vet ./...` must pass before advancing.
- **Atomic commits**: One commit per task. Format: `phase-N/task-M: <description>`.
- **No speculative code**: Only implement what the current task requires.
- **Parallel-safe**: Each task owns exclusive files. No shared writes within a wave.

## Execution Summary

| Metric | Value |
|--------|-------|
| Phases | 12 (0–11) |
| Tasks | 71 |
| Stages | 6 (sequential barriers) |
| Waves | 11 (parallel batches within stages) |
| Max parallel agents | 8 |
| Critical path | Stage 0 → 1 → 2 → 3 → 4 → 5 |

With 8 agents, 71 tasks compress into **11 merge points** instead of 71 sequential steps.

## Stage → Wave → Task Map

```
STAGE 0 ─── Wave 0.A [1 task]   ──► bootstrap
STAGE 1 ─┬─ Wave 1.A [3 tasks]  ──► types, sqlite, config
          └─ Wave 1.B [3 tasks]  ──► connection, settings, usage CRUD
STAGE 2 ─┬─ Wave 2.A [8 tasks]  ──► server, engine, providers, OAuth types, RTK, usage
          ├─ Wave 2.B [8 tasks]  ──► middleware, registry, filters, caveman, cost, OAuth×2
          └─ Wave 2.C [8 tasks]  ──► inference handler, RTK compress, logging, OAuth×3, combos
STAGE 3 ─┬─ Wave 3.A [8 tasks]  ──► all providers, translation, MCP, UI scaffold
          └─ Wave 3.B [8 tasks]  ──► gemini xlat, vertex, remaining providers, OAuth, refresh, UI pages
STAGE 4 ─┬─ Wave 4.A [6 tasks]  ──► mgmt API, OAuth endpoints, CLI, fallback, combos, usage API
          └─ Wave 4.B [6 tasks]  ──► responses API, MCP discovery/agent/health/handlers, UI embed
STAGE 5 ─┬─ Wave 5.A [3 tasks]  ──► Makefile, systemd, Docker
          └─ Wave 5.B [2 tasks]  ──► README polish, E2E tests
```

## Phase Index

| Phase | Document | Tasks | Focus | Waves |
|-------|----------|-------|-------|-------|
| 0 | [Phase 0: Bootstrap](phases/phase-00-project-bootstrap.md) | 1 | Go module, docs | 0.A |
| 1 | [Phase 1: Core Types + SQLite](phases/phase-01-core-types-sqlite-store.md) | 6 | Types, Store, Config | 1.A, 1.B |
| 2 | [Phase 2: HTTP Server + Proxy](phases/phase-02-http-server-proxy-engine.md) | 7 | fasthttp, middleware, engine, OpenAI, SSE | 2.A–2.C |
| 3 | [Phase 3: Multi-Provider](phases/phase-03-multi-provider-support.md) | 10 | Anthropic, Gemini, 13 compat, translation | 3.A–3.B, 4.B |
| 4 | [Phase 4: Registry](phases/phase-04-persistence-provider-registry.md) | 5 | Registry, round-robin, combos, aliases, API | 2.B, 2.C, 3.B, 4.A |
| 5 | [Phase 5: OAuth + CLI](phases/phase-05-oauth-flows-cli.md) | 11 | 15+ OAuth flows, cobra CLI | 2.A–4.A |
| 6 | [Phase 6: Fallback](phases/phase-06-account-fallback-combos.md) | 2 | Backoff, per-model locks, combo chains | 4.A |
| 7 | [Phase 7: RTK + Caveman](phases/phase-07-rtk-caveman.md) | 4 | 11 filters, autodetect, caveman | 2.A–2.C |
| 8 | [Phase 8: Usage + Cost](phases/phase-08-usage-tracking-cost-logging.md) | 6 | Token extraction, pricing, quota, logging | 2.A–4.A |
| 9 | [Phase 9: MCP Gateway](phases/phase-09-mcp-gateway.md) | 6 | Client/tool manager, discovery, agent loop | 3.A, 4.B |
| 10 | [Phase 10: Dashboard UI](phases/phase-10-dashboard-ui.md) | 8 | React + Vite + Tailwind, embedded | 3.A–4.B |
| 11 | [Phase 11: Packaging](phases/phase-11-packaging-deployment-polish.md) | 5 | Makefile, systemd, Docker, E2E | 5.A–5.B |

## Coordination Documents

| Document | Purpose |
|----------|---------|
| **[ORCHESTRATION.md](ORCHESTRATION.md)** | Stage/wave definitions, file ownership, agent protocol, merge rules |
| **[WORKFLOW.md](WORKFLOW.md)** | Live task status (YAML), agent assignments, wave progress |
| [ARCHITECTURE.md](ARCHITECTURE.md) | System diagrams, request pipeline, key interfaces |
| [SCHEMA.md](SCHEMA.md) | SQLite DDL, API contracts, CLI commands |
| [REFERENCES.md](REFERENCES.md) | Source mapping from bifrost/9router/oh-my-pi |
| [DEPLOYMENT.md](DEPLOYMENT.md) | systemd, Docker, nginx |
| [CONFIG.md](CONFIG.md) | Environment variables, defaults, validation |
| [PROVIDERS.md](PROVIDERS.md) | Provider catalog with auth details |
| [DIRECTORY_STRUCTURE.md](DIRECTORY_STRUCTURE.md) | Target file layout (~140 Go files) |

## How to Start

### Single agent
1. Read `CLAUDE.md` → `WORKFLOW.md` → pick next PENDING task in current wave
2. Follow the phase doc's TDD process
3. Commit, update WORKFLOW.md, move to next task

### Multi-agent orchestration
1. Read `ORCHESTRATION.md` for the full parallel model
2. Dispatch one agent per task in the current wave
3. Merge when all wave tasks report DONE
4. Run gate verification, advance to next wave
