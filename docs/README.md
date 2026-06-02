# g0router Documentation

Start here to understand the project, then dive into specific areas.

## Reading Order

1. **[ARCHITECTURE.md](ARCHITECTURE.md)** — System overview, request pipeline, key interfaces
2. **[PLAN.md](PLAN.md)** — Master implementation plan with links to all 13 phase documents
3. **[SCHEMA.md](SCHEMA.md)** — SQLite tables and API endpoint contracts
4. **[CONFIG.md](CONFIG.md)** — Environment variables and settings
5. **[PROVIDERS.md](PROVIDERS.md)** — Provider catalog with auth details and wire formats

## Reference

| Document | When to Read |
|----------|-------------|
| [WORKFLOW.md](WORKFLOW.md) | Before starting any implementation work — shows current status |
| [REFERENCES.md](REFERENCES.md) | When porting code from bifrost, 9router, or oh-my-pi |
| [DEPLOYMENT.md](DEPLOYMENT.md) | When packaging for production (systemd, Docker, nginx) |
| [DIRECTORY_STRUCTURE.md](DIRECTORY_STRUCTURE.md) | When creating new files — shows where everything goes |

## Phase Guides

Each phase is a self-contained implementation guide with design decisions, type definitions, algorithm descriptions, test scenarios, and acceptance criteria:

| Phase | Focus | Key Content |
|-------|-------|-------------|
| [Phase 0](phases/phase-00-project-bootstrap.md) | Bootstrap | Go module setup, docs scaffolding |
| [Phase 1](phases/phase-01-core-types-sqlite-store.md) | Core types + DB | Full type definitions, SQLite CRUD, config |
| [Phase 2](phases/phase-02-http-server-proxy-engine.md) | HTTP + proxy | fasthttp server, middleware, SSE streaming |
| [Phase 3](phases/phase-03-multi-provider-support.md) | Multi-provider | Anthropic/Gemini wire formats, translation engine |
| [Phase 4](phases/phase-04-persistence-provider-registry.md) | Registry | Round-robin, combos, aliases, management API |
| [Phase 5](phases/phase-05-oauth-flows-cli.md) | OAuth + CLI | 15+ OAuth flows, PKCE/device-code, cobra CLI |
| [Phase 6](phases/phase-06-account-fallback-combos.md) | Fallback | Backoff algorithm, per-model locks, combo chains |
| [Phase 7](phases/phase-07-rtk-caveman.md) | RTK + Caveman | 11 filters, autodetect algorithm, prompt injection |
| [Phase 8](phases/phase-08-usage-tracking-cost-logging.md) | Usage + cost | Token extraction, pricing catalog, quota APIs |
| [Phase 9](phases/phase-09-mcp-gateway.md) | MCP | Client manager, compact manifests, agent loop |
| [Phase 10](phases/phase-10-dashboard-ui.md) | Dashboard | React + Vite + Tailwind, embedded in binary |
| [Phase 11](phases/phase-11-packaging-deployment-polish.md) | Deploy | Makefile, systemd, Docker, E2E tests |
| [Phase 12](phases/phase-12-advanced-mcp-gateway.md) | Advanced MCP | Docker/npx/http instances, OAuth, multi-account |

## AI Agent Protocol

If you're an AI agent working on this project:

1. Read [CLAUDE.md](../CLAUDE.md) for behavioral rules
2. Read [WORKFLOW.md](WORKFLOW.md) for current task status
3. Pick up the next `PENDING` task
4. Follow TDD: write test → see fail → write code → see pass
5. Run `go test ./... && go vet ./...` before committing
6. Commit: `phase-N/task-M: <description>`
7. Update WORKFLOW.md status
