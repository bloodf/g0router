# Handoff: Start Stage 12B–19 Execution

> Paste this entire document into a new AI agent session to begin implementation.
> All planning artifacts are A+ grade verified. Do NOT re-plan — execute.

---

## 1. Project Context (g0router)

Single-binary Go LLM gateway/proxy with 43+ providers, OAuth flows, RTK compression, MCP gateway, embedded React dashboard. Stage 0–8 complete. Stage 12B–19 pending.

**Repo**: `/Users/heitor/Developer/github.com/bloodf/g0router`
**Tech**: Go 1.22+, SQLite (WAL), fasthttp, embedded React UI (out of scope until Phase 20)
**Architecture**: Layered DDD-lite enforced by arch test (§1.8 below)

---

## 2. Execution Strategy: Subagent-Driven Development

Use `Agent(subagent_type="coder")` to dispatch one implementer subagent per task. After each task completes, run spec-compliance review, then code-quality review. **Never** dispatch multiple implementers in parallel — they will conflict on shared files (`internal/store/sqlite.go`, `api/server.go`).

### Per-Task Workflow

1. **Extract** the task from the phase doc + orchestration.jsonl.
2. **Dispatch implementer** with full context (read 3 analogous files first, include their paths).
3. **Implementer** writes `_test.go` first, sees it fail, writes minimum code, commits.
4. **Spec reviewer** (fresh subagent): confirms code matches phase doc + STAGE-13-19-PROCESS.md contracts.
5. **Code quality reviewer** (fresh subagent): checks style, error wrapping, no globals, no `init()`, interface design.
6. Fix loops until both reviews pass.
7. Mark task done in TodoWrite, proceed to next task.

### Model Selection
- **Implementer**: standard model (clear spec, 1–4 files)
- **Reviewers**: standard model
- **Integration/debugging**: most capable model

---

## 3. Non-Negotiable Rules (from AGENTS.md + STAGE-13-19-PROCESS.md)

### TDD
- Write `_test.go` **first**. Run it. See it fail. Then implement.
- No mocks — interfaces and fakes. Temp SQLite DBs in tests.
- `httptest`-style handler tests via fasthttp `serveCtx` helpers in `api/*_test.go`.
- `go test ./... -count=1` + `go vet ./...` green before every commit.

### DDD-lite Layers (dependency direction inward)
```
api/handlers/        → HTTP transport ONLY: parse, validate, envelope, status.
                       No business rules. Delegate to domain.
internal/<domain>/   → domain packages own ALL business logic.
                       Pure Go, no fasthttp imports.
internal/store/      → repository layer: persistence only, no business decisions.
                       One file per aggregate.
```
- New business logic NEVER lives in a handler or store.
- Domain packages define repository interfaces they need; `*store.Store` satisfies them.
- Handlers depend on domain packages, not reverse.

### Cross-Cutting Contracts
- **All JSON** field names `snake_case`.
- **Response envelope**: `{ "data": <payload>, "error": null }` (failure: `data: null, error: "msg"`).
- **Audit logging**: every mutating endpoint writes `audit_log` row (actor, action, target, details).
- **Secrets at rest**: reversible via `*_enc` columns (read `internal/store/oauthsessions.go`); irreversible hashed (bcrypt/SHA-256).
- **Feature flags**: check at middleware/dispatch boundary, no-op when disabled.

### Gates
- **Per-commit**: `go test ./... -count=1 && go vet ./... && go build ./cmd/g0router`
- **Per-phase**: above + `go test -race ./...` + coverage ≥ 95.0%
- **UI gates** only if `ui/` touched (backend phases should not touch `ui/`)

### Commit Format
```
phase-N/task-M: <description>
```
Direct push to `main` (no PRs). Run gates locally before every push.

---

## 4. Phase Execution Order

| Phase | Name | Status | Start Here? |
|-------|------|--------|-------------|
| 12B | DDD & Architecture Refactor | **PENDING** | ✅ YES — start here |
| 13 | Auth & Core Infrastructure | PENDING | |
| 14 | Providers & Testing | PENDING | |
| 15 | Tunnels & Network | PENDING | |
| 16 | Chat & Console | PENDING | |
| 17 | Usage & Analytics | PENDING | |
| 18 | Bifrost Features (18A–18D) | PENDING | |
| 19 | Advanced Features | PENDING | |
| 20 | Lovable UI Generation | PENDING | user-driven, out of scope |

**Strict order**: 12B → 13 → (14–17 may reorder) → 18 → 19.

---

## 5. Starting Point: Phase 12B

Read these files **in this order** before touching code:

1. `docs/phases/phase-12b-ddd-architecture-refactor.md` — canonical spec
2. `docs/planning/phase-12b-ddd-architecture-refactor/architect-plan.md` — implementation plan
3. `docs/planning/phase-12b-ddd-architecture-refactor/orchestration.jsonl` — task list with files + verification
4. `docs/phases/STAGE-13-19-PROCESS.md` — cross-cutting contracts, gates, checkpoint protocol

### Phase 12B Task List (from orchestration.jsonl)

```
task-1: Routing table extraction
  files: api/server.go, api/routes.go, api/wiring.go, api/routes_test.go
  → Route-table snapshot test asserts identical method+path pairs before/after split

task-2: Repository interfaces
  files: api/handlers/*.go, internal/mcp/*.go, internal/proxy/*.go, internal/usage/*.go
  → Consumer-defined interfaces replacing *store.Store fields; one fake-proves-possible test

task-3: Usage domain extraction
  files: api/handlers/usage.go, internal/usage/usage.go, internal/usage/usage_test.go
  → Handler tests unchanged and green after aggregation logic moves to internal/usage

task-4: Inference pipeline boundary (TDD-FIRST — write pipeline_test.go first, see fail, then extract)
  files: api/handlers/inference.go, internal/proxy/pipeline.go, internal/proxy/pipeline_test.go,
         internal/dispatch/dispatch.go, internal/dispatch/dispatch_test.go
  → Pipeline unit tests pass first (model resolution, RTK/Caveman, MCP injection stages with fakes)

task-5: Handler hygiene sweep
  files: api/handlers/*.go
  → Business rules moved to owning domains, pure CRUD untouched, files >800 lines split

task-6: Architecture conformance test
  files: internal/archtest/arch_test.go
  → go list-based arch test: no internal/<domain> imports api/, domains avoid fasthttp, store imports no domain

checkpoint: WORKFLOW update + ## Outcome + ARCHITECTURE.md notes
```

### Before Task 1: Read 3 Existing Files
Read these to understand existing patterns:
- `api/server.go` — how routes are currently registered
- `api/handlers/apikeys.go` — handler pattern (envelope, snake_case, audit)
- `internal/store/oauthsessions.go` — store pattern (ensureColumn migrations, encryption)

---

## 6. Key Domain Packages to Create (across phases)

| Phase | Domain Package | Responsibility |
|-------|---------------|----------------|
| 12B | `internal/usage/` | Usage aggregation, chart zero-fill |
| 12B | `internal/proxy/` | Pipeline stages (model resolution, RTK, Caveman, MCP injection) |
| 12B | `internal/dispatch/` | Dispatch orchestration |
| 13 | `internal/auth/` | Session validation, CSRF, coexistence resolver, rate limiter |
| 15 | `internal/tunnel/` | Checksum-verified download, process supervisor, health loops |
| 16 | `internal/console/` | Ring buffer, slog tee, subscriber broker |
| 18A | `internal/governance/` | Virtual keys, budgets, hierarchical limits |
| 18C | `internal/guardrails/` | Blocklist, PII redaction |
| 18D | `internal/alerts/` | Webhook/discord/telegram dispatch, retry |
| 19 | `internal/semcache/` | Semantic cache, cosine similarity |
| 19 | `internal/update/` | Checksum verify, staged swap |
| 19 | `internal/mitm/` | CA generation, cert minting, proxy |

---

## 7. Planning Artifact Locations

| Artifact | Path |
|----------|------|
| Phase docs | `docs/phases/phase-NN-*.md` |
| Process doc | `docs/phases/STAGE-13-19-PROCESS.md` |
| Architect plans | `docs/planning/phase-NN-*/architect-plan.md` |
| Orchestration | `docs/planning/phase-NN-*/orchestration.jsonl` |
| Risk registers | `docs/planning/phase-NN-*/risk-register.md` |
| Rollback plans | `docs/planning/phase-NN-*/rollback.md` |
| Verification gates | `docs/planning/phase-NN-*/verification-gate.md` |
| WORKFLOW tracker | `docs/WORKFLOW.md` |

---

## 8. Security Review Triggers (mandatory at checkpoint)

- Phase 13 (auth/sessions/CSRF)
- Phase 15 (binary downloads, CLI shelling)
- Phase 18 (backup/restore secret export, budget enforcement)
- Phase 19 (MITM CA, auto-updater self-replace)

Checklist per pass: input validation, authn/authz on every new route, secrets at rest, secrets in logs, supply-chain (downloads pinned + checksummed), privilege requirements documented.

---

## 9. Checkpoint Protocol (end of every phase)

1. Run per-phase gate (3.2 in STAGE-13-19-PROCESS.md). All green or phase is NOT done.
2. Update `docs/WORKFLOW.md`: status → `DONE`, add commit range.
3. Update phase doc: append `## Outcome` section (shipped, deferred, why).
4. Commit docs: `phase-N/checkpoint: workflow + outcome notes`.
5. **Stop and reassess**: re-read next phase doc against current codebase. Fix assumptions before starting.

---

## 10. Recovery

If tests fail mid-phase:
1. `go test ./...` → identify failures
2. `git log --oneline -10` → last good commit
3. Read `docs/WORKFLOW.md` → active phase/task
4. Fix failing tests before proceeding. Never skip — fix or revert.

---

## 11. Immediate First Action

Do this now:

1. Read `docs/phases/phase-12b-ddd-architecture-refactor.md`
2. Read `docs/planning/phase-12b-ddd-architecture-refactor/orchestration.jsonl`
3. Read `docs/phases/STAGE-13-19-PROCESS.md`
4. Read 3 existing files: `api/server.go`, `api/handlers/apikeys.go`, `internal/store/oauthsessions.go`
5. Create TodoWrite with all 6 Phase 12B tasks + checkpoint
6. Dispatch implementer subagent for **task-1** (routing table extraction)

**Do NOT re-plan. Do NOT modify phase docs. Execute the plan as written.**
