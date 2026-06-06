# Handoff: g0router — Stage 12B-19 Backend Execution

## Context

You are a new agent taking over g0router. Planning is complete. **Your job:
execute Phase 12B (architecture refactor) then phases 13-19 (backend
features) while the user works with Lovable on the UI in parallel.**

## Read These, In Order, Before Any Code

1. `AGENTS.md` — behavioral rules (TDD, no mocks, no init(), errors as values)
2. `docs/phases/STAGE-13-19-PROCESS.md` — **execution process, cross-cutting
   contracts, gates, checkpoint protocol. The single source of truth for HOW.**
3. `docs/WORKFLOW.md` — Stage 12B-19 section: phase table + current status
4. The phase doc for the phase you are starting

## What Was Done Before You

1. **Feature evaluation** (complete): 84 features from 9Router, 89 from
   g0router, 95 from Bifrost → unified comparison → ~55 to implement,
   rest skipped/deferred (deferrals listed in WORKFLOW + phase docs).
2. **Lovable UI prompt** (complete — user handles generation):
   `docs/lovable-prompt.md`. UI integration is phases 20-21, NOT yours.
3. **Phase documents** (complete — your roadmap):
   `docs/phases/phase-12b-*` and `phase-13-*` … `phase-19-*`. Each has
   architecture notes, tables, endpoints, task breakdown with TDD steps,
   minimum test requirements, commit messages.
4. **Process doc** (complete): `docs/phases/STAGE-13-19-PROCESS.md`.

## The Big Picture

**g0router**: single-binary Go LLM gateway — 43+ providers, OAuth flows,
6 routing strategies, MCP gateway, RTK + Caveman compression, Prometheus
metrics, SQLite (WAL) store, embedded React UI, rate limiting, connection
health tracking. ~28K src LOC, 41 packages, ~95% coverage, 2700+ tests.

**Goal**: layered DDD architecture + 9Router dashboard parity + selected
Bifrost governance features.

## Architecture You Must Preserve

- **HTTP** (`api/`): `valyala/fasthttp`, custom routing. Routes: `/healthz`,
  `/metrics`, `/v1/*` (inference), `/api/*` (management). Middleware chain:
  CORS → source policy → auth → request ID. UI via `embed.FS`.
  (Note: health route is `/healthz`, NOT `/api/healthz`.)
- **Store** (`internal/store/`): SQLite WAL, additive migrations via
  `ensureColumn`. Usage data lives in **`request_log`**.
- **Providers** (`internal/providers/`): 16 native adapters + OpenAI-compat
  registry. Matrix in `internal/provider/matrix.go`.
- **Routing** (`internal/proxy/`): fallback chains, combos (6 strategies incl.
  existing `auto` classifier), alias TTL cache, provider-qualified routes.
- **Inference** (`api/handlers/inference.go`): `/v1/chat/completions`,
  `/v1/messages`, `/v1/responses`; SSE streaming; MCP tool injection +
  RTK/Caveman applied before dispatch.

Do not break: source IP policy, RTK/Caveman wiring, MCP agent loop, existing
bearer/X-API-Key auth on `/api/*` (sessions COEXIST with it — phase 13 defines
the precedence and exempt routes).

### Default admin credential (fact check)
First boot auto-creates a control-plane **API key** named `admin` with raw
value `123456` (banner in `internal/cli/root.go`). It is an API key, NOT a
dashboard username/password. Dashboard users start empty; first one is created
via `POST /api/auth/setup` (phase 13).

## Execution Order

```
12B (refactor, zero behavior change)
 → 13 (auth foundation — everything later assumes its middleware)
 → 14 → 15 → 16 → 17 (reorderable if blocked)
 → 18 (sub-stages 18A→18D, checkpoint-lite between each)
 → 19 (highest-risk items last: MITM final task)
```

Checkpoint protocol at every phase end — process doc §4. A phase with failing
gates or missing tests is BLOCKED, not DONE. Never start the next phase from
a BLOCKED one.

## Gates (memorize)

Per-commit:
```bash
go test ./... -count=1
go vet ./...
go build ./cmd/g0router
```
Per-phase checkpoint adds: `go test -race ./...` + coverage ≥ 95.0%.
UI gates only if you touched `ui/` (you should not).
**Run `go test` from project root** — known failure mode when run from `ui/`.

## Cross-Cutting Contracts (full versions in process doc §1)

- All new JSON: `snake_case`, `{data, error}` envelope. Lovable prompt types
  are the contract.
- Every mutating endpoint: audit_log row.
- Reversible secrets encrypted at rest (oauthsessions.go pattern); passwords
  bcrypt; keys sha256.
- Flagged features check `feature_flags` and no-op when disabled.
- New business logic in `internal/<domain>` packages; handlers thin;
  repository interfaces defined by consumers. Arch conformance test from 12B
  enforces this.

## Commit Pattern
```
phase-12b/task-1: routing table extraction
phase-13/task-3: auth handlers with login rate limit
phase-N/checkpoint: workflow + outcome notes
```

## When to Stop

Stop after Phase 19 checkpoint passes (all gates + security passes recorded).
Do NOT start UI integration (20-21) — user handles Lovable.

Final verification:
```bash
go test ./... -count=1 && go vet ./... && go test -race ./... && go build ./cmd/g0router
make e2e-binary
```

## Questions?

If a phase doc contradicts the codebase, the codebase wins — update the doc,
note it at checkpoint. If genuinely ambiguous, ask the user. Do not guess.
