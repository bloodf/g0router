# Workflow — AI Agent Handoff Protocol

## How to Use This File

Any AI agent (Claude, Codex, Gemini, or human) starting work on g0router:

1. Read `CLAUDE.md` for behavioral rules.
2. Read this file to find current state.
3. Pick up the next `PENDING` task.
4. Update status to `IN_PROGRESS` with your agent ID and timestamp.
5. Complete the task following TDD (test first).
6. Run `go test ./...` + `go vet ./...`.
7. Commit: `phase-N/task-M: <description>`.
8. Update status to `DONE` with timestamp.
9. If a phase checkpoint is reached, update phase status.
10. Move to next task.

## Status Values

| Status | Meaning |
|--------|---------|
| `PENDING` | Not started |
| `IN_PROGRESS` | Agent is actively working on it |
| `BLOCKED` | Cannot proceed — reason documented |
| `DONE` | Complete, tests pass, committed |
| `SKIPPED` | Deliberately not doing (with reason) |

## Current State

```yaml
project_status: DOCS_COMPLETE
current_phase: null
last_updated: 2026-06-02T16:15:00Z
last_agent: composer
notes: "22 markdown docs: README, CLAUDE, 9 docs/, 12 phase docs. Implementation not started."

phases:
  phase_0:
    status: PENDING
    name: "Project Bootstrap"
    checkpoint: PHASE_0_COMPLETE
    tasks:
      - id: "0.1"
        name: "Initialize Go module and directory structure"
        status: PENDING
        agent: null
        started_at: null
        completed_at: null
        notes: null

  phase_1:
    status: PENDING
    name: "Core Types + SQLite Store"
    checkpoint: PHASE_1_COMPLETE
    depends_on: [phase_0]
    tasks:
      - id: "1.1"
        name: "Define core types"
        status: PENDING
      - id: "1.2"
        name: "SQLite store foundation"
        status: PENDING
      - id: "1.3"
        name: "Connection CRUD"
        status: PENDING
        depends_on: ["1.2"]
      - id: "1.4"
        name: "Settings + API keys store"
        status: PENDING
        depends_on: ["1.2"]
      - id: "1.5"
        name: "Usage log store"
        status: PENDING
        depends_on: ["1.2"]
      - id: "1.6"
        name: "Config loading"
        status: PENDING

  phase_2:
    status: PENDING
    name: "HTTP Server + Proxy Engine"
    checkpoint: PHASE_2_COMPLETE
    depends_on: [phase_1]
    tasks:
      - id: "2.1"
        name: "fasthttp server skeleton"
        status: PENDING
      - id: "2.2"
        name: "Middleware (CORS, auth, request ID)"
        status: PENDING
        depends_on: ["2.1"]
      - id: "2.3"
        name: "Proxy engine core"
        status: PENDING
      - id: "2.4"
        name: "OpenAI provider implementation"
        status: PENDING
      - id: "2.5"
        name: "Shared provider utilities"
        status: PENDING
      - id: "2.6"
        name: "Streaming accumulator"
        status: PENDING
      - id: "2.7"
        name: "Inference handler"
        status: PENDING
        depends_on: ["2.1", "2.2", "2.3", "2.4", "2.5", "2.6"]

  phase_3:
    status: PENDING
    name: "Multi-Provider Support"
    checkpoint: PHASE_3_COMPLETE
    depends_on: [phase_2]
    tasks:
      - id: "3.1"
        name: "Anthropic provider"
        status: PENDING
      - id: "3.2"
        name: "Format translation engine"
        status: PENDING
      - id: "3.3"
        name: "OpenAI-compatible providers (13 total)"
        status: PENDING
      - id: "3.4"
        name: "Gemini provider"
        status: PENDING
      - id: "3.5"
        name: "Gemini format translation"
        status: PENDING
        depends_on: ["3.2", "3.4"]
      - id: "3.6"
        name: "Vertex AI provider"
        status: PENDING
      - id: "3.7"
        name: "AWS Bedrock provider"
        status: PENDING
      - id: "3.8"
        name: "Azure OpenAI provider"
        status: PENDING
      - id: "3.9"
        name: "Mistral, Ollama, Cohere, Replicate"
        status: PENDING
      - id: "3.10"
        name: "Responses API support"
        status: PENDING

  phase_4:
    status: PENDING
    name: "Persistence + Provider Registry"
    checkpoint: PHASE_4_COMPLETE
    depends_on: [phase_1]
    parallel_with: [phase_3, phase_5]
    tasks:
      - id: "4.1"
        name: "Provider registry"
        status: PENDING
      - id: "4.2"
        name: "Connection management with round-robin"
        status: PENDING
      - id: "4.3"
        name: "Combos store + resolver"
        status: PENDING
      - id: "4.4"
        name: "Model aliases + pricing overrides"
        status: PENDING
      - id: "4.5"
        name: "Management API handlers"
        status: PENDING
        depends_on: ["4.1", "4.2", "4.3", "4.4"]

  phase_5:
    status: PENDING
    name: "OAuth Flows + CLI"
    checkpoint: PHASE_5_COMPLETE
    depends_on: [phase_1]
    parallel_with: [phase_3, phase_4]
    tasks:
      - id: "5.1"
        name: "OAuth types and interface"
        status: PENDING
      - id: "5.2"
        name: "Anthropic OAuth (Claude Code)"
        status: PENDING
        depends_on: ["5.1"]
      - id: "5.3"
        name: "OpenAI Codex OAuth"
        status: PENDING
        depends_on: ["5.1"]
      - id: "5.4"
        name: "GitHub Copilot OAuth"
        status: PENDING
        depends_on: ["5.1"]
      - id: "5.5"
        name: "Cursor PKCE OAuth"
        status: PENDING
        depends_on: ["5.1"]
      - id: "5.6"
        name: "Google OAuth (Gemini CLI, Antigravity)"
        status: PENDING
        depends_on: ["5.1"]
      - id: "5.7"
        name: "xAI, DeepSeek, GitLab, Kiro OAuth"
        status: PENDING
        depends_on: ["5.1"]
      - id: "5.8"
        name: "Chinese provider OAuth (Qwen, Kimi, MiniMax, etc.)"
        status: PENDING
        depends_on: ["5.1"]
      - id: "5.9"
        name: "Token refresh with dedup"
        status: PENDING
        depends_on: ["5.1"]
      - id: "5.10"
        name: "OAuth HTTP endpoints"
        status: PENDING
      - id: "5.11"
        name: "CLI commands (cobra)"
        status: PENDING

  phase_6:
    status: PENDING
    name: "Account Fallback + Combos"
    checkpoint: PHASE_6_COMPLETE
    depends_on: [phase_2, phase_4]
    parallel_with: [phase_7, phase_8]
    tasks:
      - id: "6.1"
        name: "Account fallback engine"
        status: PENDING
      - id: "6.2"
        name: "Combo model resolution"
        status: PENDING

  phase_7:
    status: PENDING
    name: "RTK + Caveman"
    checkpoint: PHASE_7_COMPLETE
    depends_on: [phase_1]
    parallel_with: [phase_6, phase_8]
    tasks:
      - id: "7.1"
        name: "RTK autodetect"
        status: PENDING
      - id: "7.2"
        name: "RTK filters (11 total)"
        status: PENDING
      - id: "7.3"
        name: "RTK message compression"
        status: PENDING
        depends_on: ["7.1", "7.2"]
      - id: "7.4"
        name: "Caveman prompt injection"
        status: PENDING

  phase_8:
    status: PENDING
    name: "Usage Tracking + Cost + Logging"
    checkpoint: PHASE_8_COMPLETE
    depends_on: [phase_1]
    parallel_with: [phase_6, phase_7]
    tasks:
      - id: "8.1"
        name: "Usage extraction from responses"
        status: PENDING
      - id: "8.2"
        name: "Model pricing catalog"
        status: PENDING
      - id: "8.3"
        name: "Cost calculation"
        status: PENDING
        depends_on: ["8.1", "8.2"]
      - id: "8.4"
        name: "Provider quota fetchers"
        status: PENDING
      - id: "8.5"
        name: "Request/response logging"
        status: PENDING
      - id: "8.6"
        name: "Usage + logging API handlers"
        status: PENDING
        depends_on: ["8.1", "8.2", "8.3", "8.4", "8.5"]

  phase_9:
    status: PENDING
    name: "MCP Gateway"
    checkpoint: PHASE_9_COMPLETE
    depends_on: [phase_2]
    tasks:
      - id: "9.1"
        name: "MCP client manager"
        status: PENDING
      - id: "9.2"
        name: "MCP tool manager"
        status: PENDING
      - id: "9.3"
        name: "MCP tool discovery (compact manifests)"
        status: PENDING
        depends_on: ["9.1", "9.2"]
      - id: "9.4"
        name: "MCP agent loop"
        status: PENDING
        depends_on: ["9.1", "9.2"]
      - id: "9.5"
        name: "MCP health monitor"
        status: PENDING
        depends_on: ["9.1"]
      - id: "9.6"
        name: "MCP API handlers + store"
        status: PENDING
        depends_on: ["9.1", "9.2", "9.3", "9.4", "9.5"]

  phase_10:
    status: PENDING
    name: "Dashboard UI"
    checkpoint: PHASE_10_COMPLETE
    depends_on: [phase_2]
    parallel_with: [phase_3, phase_4, phase_5, phase_6, phase_7, phase_8, phase_9]
    tasks:
      - id: "10.1"
        name: "UI scaffold"
        status: PENDING
      - id: "10.2"
        name: "Dashboard overview page"
        status: PENDING
      - id: "10.3"
        name: "Endpoint page"
        status: PENDING
      - id: "10.4"
        name: "Providers page"
        status: PENDING
      - id: "10.5"
        name: "Usage page"
        status: PENDING
      - id: "10.6"
        name: "Quota page"
        status: PENDING
      - id: "10.7"
        name: "Combos, MCP, Settings, Profile pages"
        status: PENDING
      - id: "10.8"
        name: "Embed UI in Go binary"
        status: PENDING
        depends_on: ["10.1", "10.2", "10.3", "10.4", "10.5", "10.6", "10.7"]

  phase_11:
    status: PENDING
    name: "Packaging, Deployment + Polish"
    checkpoint: PHASE_11_COMPLETE
    depends_on: [phase_0, phase_1, phase_2, phase_3, phase_4, phase_5, phase_6, phase_7, phase_8, phase_9, phase_10]
    tasks:
      - id: "11.1"
        name: "Makefile (build, test, install, docker)"
        status: PENDING
      - id: "11.2"
        name: "systemd service unit + install/uninstall CLI"
        status: PENDING
      - id: "11.3"
        name: "Docker + docker-compose"
        status: PENDING
      - id: "11.4"
        name: ".env.example + README + DEPLOYMENT.md"
        status: PENDING
      - id: "11.5"
        name: "Final integration test suite"
        status: PENDING
```

## Parallelization Map

```
Phase 0 ──► Phase 1 ──► Phase 2 ──┬──► Phase 3
                    │              ├──► Phase 9
                    │              ├──► Phase 10 (UI, independent)
                    ├──► Phase 4 ──┤
                    ├──► Phase 5   ├──► Phase 6
                    ├──► Phase 7   │
                    └──► Phase 8   │
                                   └──► Phase 11 (after all)
```

## Verification Protocol

Before marking any task DONE:

```bash
# 1. Tests pass
go test ./...

# 2. Vet passes
go vet ./...

# 3. Build succeeds
go build ./cmd/g0router

# 4. New tests exist for new code
# (check that _test.go files were created BEFORE implementation)
```

## Recovery Protocol

If you find the project in a broken state:

1. Run `go test ./...` to identify failures.
2. Check `git log --oneline -10` for last successful commit.
3. Read this WORKFLOW.md to find which task is IN_PROGRESS.
4. Fix the failing tests before proceeding.
5. Never skip a broken test — fix it or revert the change.
