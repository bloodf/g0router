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
project_status: ACTIVE_REMEDIATION
current_stage: 8
current_wave: "8.S"
last_updated: "2026-06-04T09:31:33Z"
last_agent: "orchestrator"
```

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

**Checkpoint**: Stage 8 remains active; continue auditing docs-defined gaps and run evaluator before release lock.

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
evaluation: "PENDING external evaluator run"
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

**Checkpoint**: Wave 8.S fixes the auth/runtime split for Vertex. `vertex` now resolves to the Gemini OAuth flow while persisted CLI and HTTP OAuth connections keep runtime provider `vertex` with `oauth_provider=gemini`, so provider-qualified Vertex dispatch can find usable credentials. External evaluation remains pending.

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

### Wave 7.E — Real 9Router-style dispatch pipeline

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
3. Read WORKFLOW.md → find IN_PROGRESS task
4. Fix failing tests before proceeding
5. Never skip a broken test — fix or revert
