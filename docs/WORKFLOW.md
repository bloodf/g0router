# Workflow — AI Agent Handoff Protocol

## How to Use This File

1. Read `CLAUDE.md` for behavioral rules.
2. Read `docs/ORCHESTRATION.md` for the parallel execution model.
3. Find current stage and wave below.
4. Pick up the next `PENDING` task **in the current wave**.
5. Update status to `IN_PROGRESS` with your agent ID.
6. Complete the task following TDD.
7. Run `go test ./... && go vet ./...`.
8. Commit: `phase-N/task-M: <description>`.
9. Update status to `DONE`.
10. When ALL tasks in a wave are `DONE`, orchestrator merges and advances.

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
current_stage: 4
current_wave: "4.B"
last_updated: "2026-06-02T19:54:04Z"
last_agent: "orchestrator"
```

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
status: PENDING
max_agents: 6
depends_on: ["4.A"]
gate: "go test ./... && go vet ./..."

tasks:
  - id: "3.10"
    name: "Responses API support"
    status: PENDING
    files_owned: ["internal/providers/openai/responses.go", "internal/streaming/responses.go", "internal/translate/responses.go"]

  - id: "9.3"
    name: "MCP tool discovery (compact manifests)"
    status: PENDING
    depends_on_tasks: ["9.1+9.2"]
    files_owned: ["internal/mcp/discovery.go", "internal/mcp/discovery_test.go"]

  - id: "9.4"
    name: "MCP agent loop"
    status: PENDING
    depends_on_tasks: ["9.1+9.2"]
    files_owned: ["internal/mcp/agent.go", "internal/mcp/agent_test.go"]

  - id: "9.5"
    name: "MCP health monitor"
    status: PENDING
    depends_on_tasks: ["9.1+9.2"]
    files_owned: ["internal/mcp/healthmonitor.go", "internal/mcp/healthmonitor_test.go"]

  - id: "9.6"
    name: "MCP API handlers + store"
    status: PENDING
    depends_on_tasks: ["9.1+9.2", "9.3", "9.4", "9.5"]
    files_owned: ["api/handlers/mcp.go", "internal/store/mcpclients.go", "internal/store/mcpclients_test.go"]

  - id: "10.8"
    name: "Embed UI in Go binary"
    status: PENDING
    depends_on_tasks: ["10.2-10.7"]
    files_owned: ["embed.go"]
```

**Checkpoint**: `PHASE_4_COMPLETE`, `PHASE_5_COMPLETE`, `PHASE_6_COMPLETE`, `PHASE_8_COMPLETE`, `PHASE_9_COMPLETE`, `PHASE_10_COMPLETE`

---

## STAGE 5 — Packaging

### Wave 5.A — Final packaging (3 agents)

```yaml
wave: "5.A"
status: PENDING
max_agents: 3
depends_on: ["4.B"]
gate: "make test && make build"

tasks:
  - id: "11.1"
    name: "Makefile"
    status: PENDING
    files_owned: ["Makefile"]

  - id: "11.2"
    name: "systemd service + install CLI"
    status: PENDING
    files_owned: ["deploy/g0router.service", "deploy/g0router.default", "internal/cli/install.go", "internal/cli/install_test.go"]

  - id: "11.3"
    name: "Docker support"
    status: PENDING
    files_owned: ["Dockerfile", "docker-compose.yml", ".dockerignore"]
```

### Wave 5.B — Polish + E2E (2 agents)

```yaml
wave: "5.B"
status: PENDING
max_agents: 2
depends_on: ["5.A"]
gate: "make test && make build && make docker"

tasks:
  - id: "11.4"
    name: ".env.example + README polish"
    status: PENDING
    files_owned: ["README.md", ".env.example", "docs/DEPLOYMENT.md"]

  - id: "11.5"
    name: "Final integration test suite"
    status: PENDING
    files_owned: ["e2e_test.go"]
```

**Checkpoint**: `PHASE_11_COMPLETE` → **PROJECT COMPLETE**

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
3. Read WORKFLOW.md → find IN_PROGRESS task
4. Fix failing tests before proceeding
5. Never skip a broken test — fix or revert
