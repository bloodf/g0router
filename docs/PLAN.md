# Master Implementation Plan

> **g0router** — Go LLM gateway combining 9router + bifrost + oh-my-pi.  
> Each phase is a standalone implementation guide in `docs/phases/`.

## Methodology

- **TDD**: Every task starts with tests. Tests fail first (red), minimum code to pass (green), then refactor.
- **Phase gates**: `go test ./...` + `go vet ./...` must pass before moving to next phase.
- **Atomic commits**: One commit per completed task. Message format: `phase-N/task-M: <description>`.
- **No speculative code**: Only implement what the current task requires.

## Phase Index

| Phase | Document | Tasks | Focus |
|-------|----------|-------|-------|
| 0 | [Project Bootstrap](phases/phase-00-project-bootstrap.md) | 1 | Go module, docs, .gitignore, .env.example |
| 1 | [Core Types + SQLite Store](phases/phase-01-core-types-sqlite-store.md) | 6 | OpenAI types, Provider interface, SQLite CRUD, config |
| 2 | [HTTP Server + Proxy Engine](phases/phase-02-http-server-proxy-engine.md) | 7 | fasthttp, middleware, proxy engine, OpenAI provider, SSE streaming |
| 3 | [Multi-Provider Support](phases/phase-03-multi-provider-support.md) | 10 | Anthropic, Gemini, 13 OpenAI-compat, format translation |
| 4 | [Persistence + Provider Registry](phases/phase-04-persistence-provider-registry.md) | 5 | Registry, round-robin, combos, aliases, management API |
| 5 | [OAuth Flows + CLI](phases/phase-05-oauth-flows-cli.md) | 11 | 15+ OAuth providers, PKCE/device-code, cobra CLI |
| 6 | [Account Fallback + Combos](phases/phase-06-account-fallback-combos.md) | 2 | Exponential backoff, per-model locks, combo chains |
| 7 | [RTK + Caveman](phases/phase-07-rtk-caveman.md) | 4 | 11 compression filters, autodetect, caveman injection |
| 8 | [Usage Tracking + Cost + Logging](phases/phase-08-usage-tracking-cost-logging.md) | 6 | Token extraction, pricing catalog, quota, request logging |
| 9 | [MCP Gateway](phases/phase-09-mcp-gateway.md) | 6 | Client manager, tool discovery, compact manifests, agent loop |
| 10 | [Dashboard UI](phases/phase-10-dashboard-ui.md) | 8 | React + Vite + Tailwind, embedded in Go binary |
| 11 | [Packaging + Deployment](phases/phase-11-packaging-deployment-polish.md) | 5 | Makefile, systemd, Docker, E2E tests |

**Total**: 71 tasks across 12 phases.

## Parallelization Map

```
Phase 0 ──► Phase 1 ──┬──► Phase 2 ──┬──► Phase 3
                       │              ├──► Phase 9 (MCP)
                       │              ├──► Phase 10 (UI)
                       ├──► Phase 4 ──┤
                       ├──► Phase 5   ├──► Phase 6
                       ├──► Phase 7   │
                       └──► Phase 8   │
                                      └──► Phase 11 (after all)
```

**Critical path**: 0 → 1 → 2 → 3 → 11  
**Parallel streams after Phase 1**: Phases 4, 5, 7, 8 can all start independently.

## Phase Contents Guide

Each phase document contains:

| Section | Description |
|---------|-------------|
| **Goal** | What the phase achieves; what works after completion |
| **Design Decisions** | Key choices with rationale (table format) |
| **Task N.M** | Detailed task with TDD approach |
| ├─ Types/Structs | Full Go type definitions with field explanations |
| ├─ Methods/Functions | Function signatures with behavior specification |
| ├─ Algorithm | Pseudocode or step-by-step logic for non-trivial operations |
| ├─ Wire Format | JSON examples for API-facing types |
| ├─ Test Scenarios | Table of (input, expected) covering happy path + edge cases |
| └─ Files | Exact file paths to create |
| **Phase Gate** | Verification commands that must pass |

## Related Documentation

| Document | Purpose |
|----------|---------|
| [ARCHITECTURE.md](ARCHITECTURE.md) | System diagrams, request pipeline, key interfaces |
| [SCHEMA.md](SCHEMA.md) | SQLite DDL, API endpoint contracts, CLI commands |
| [REFERENCES.md](REFERENCES.md) | File-level mapping from bifrost/9router/oh-my-pi |
| [WORKFLOW.md](WORKFLOW.md) | Task status YAML, agent handoff protocol |
| [DEPLOYMENT.md](DEPLOYMENT.md) | systemd, Docker, nginx reverse proxy |
| [CONFIG.md](CONFIG.md) | Environment variables, defaults, validation |
| [PROVIDERS.md](PROVIDERS.md) | Provider catalog with auth types and wire formats |
| [DIRECTORY_STRUCTURE.md](DIRECTORY_STRUCTURE.md) | Target file layout with package descriptions |
