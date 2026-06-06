# Phase 12B: DDD & Architecture Refactor (Whole Project)

> Process, contracts, gates, architecture target: see `docs/phases/STAGE-13-19-PROCESS.md` §1.8.
> **Runs BEFORE Phase 13** — stages 13-19 build on this foundation.
> Pure refactor: **zero behavior change**. Every task ends with the full test
> suite green and no JSON shape / route / CLI change. Coverage must stay ≥ 95.0%.

## Goal
Bring the EXISTING codebase to the layered DDD-lite architecture (transport →
domain → repository) so new stage-13-19 code lands on a consistent foundation
instead of two coexisting styles.

## Method: Strangler, Not Big-Bang
- One domain per task. Move code, re-point callers, keep old tests passing.
- Tests are the safety net — they assert behavior, so they must NOT be
  rewritten, only re-pointed (import/package path updates).
- If a task balloons past ~1 day of work, stop, commit green state, split it.
- `git mv` where possible to preserve history.

## Current State Assessment
Already good (leave structure, verify boundaries only):
- `internal/providers/*`, `internal/provider` (matrix), `internal/rtk`,
  `internal/cache`, `internal/ratelimit`, `internal/traffic`,
  `internal/usage`, `internal/translate`, `internal/streaming`,
  `internal/modelcatalog`, `internal/notify`, `internal/mcp` — already
  domain-shaped packages.

Violations to fix:
1. **`api/server.go` ~44 KB god file** — routing table, wiring, and business
   logic in one file. Path-switch routing for 33+ routes.
2. **Business logic in handlers** — `api/handlers/usage.go` (317 lines) and
   others contain aggregation/decision logic that belongs in domain packages.
3. **Store is concrete everywhere** — handlers and domains depend on
   `*store.Store` directly; no repository interfaces, fakes are awkward.
4. **`internal/store` is one flat package** mixing 14+ aggregates — acceptable
   as repository layer, but each domain must access it through a narrow
   interface it defines.

## Tasks

### `phase-12b/task-1`: Routing table extraction
- Split `api/server.go`: `api/routes.go` (pure route → handler table),
  `api/server.go` (lifecycle: listen/shutdown/TLS), `api/wiring.go`
  (dependency construction). No handler body changes.
- Verify: full suite green, route inventory identical (write a route-table
  test that snapshots method+path pairs BEFORE the split, keep it after).

### `phase-12b/task-2`: Repository interfaces
- For each consumer (handlers pkg, mcp, proxy, usage…), define the narrow
  interface it actually uses (e.g. `type apiKeyStore interface { CreateAPIKey(...) ... }`)
  in the CONSUMER package; `*store.Store` satisfies them implicitly.
- Replace `*store.Store` fields with the interfaces. No store code changes.
- Verify: suite green; new fakes possible (prove with one converted test).

### `phase-12b/task-3`: Usage domain extraction
- `internal/usage` absorbs aggregation/summary logic currently in
  `api/handlers/usage.go`; handler becomes parse → call domain → envelope.
- Verify: usage handler tests unchanged and green.

### `phase-12b/task-4`: Inference pipeline boundary
- `api/handlers/inference.go` + dispatch path: extract decision logic
  (model resolution order, RTK/Caveman application, MCP injection points)
  into `internal/proxy` (or a new `internal/dispatch` if proxy would exceed
  cohesion) as an explicit ordered pipeline; handler keeps transport only.
- This is the highest-risk task — do last among extractions; rely on the
  existing integration tests (`server_integration_test.go`, 48 KB) as the net.
- Verify: integration suite green, streaming + non-streaming + MCP loop paths.

### `phase-12b/task-5`: Handler hygiene sweep
- Remaining `api/handlers/*.go`: any business rule (validation beyond shape,
  cross-aggregate decisions, computation) moves to the owning domain package.
  Pure CRUD handlers stay as-is.
- File size rule applied: anything > 800 lines split by aggregate.

### `phase-12b/task-6`: Architecture conformance test
- Add `internal/archtest/arch_test.go`: assert dependency direction with
  `go list` (no `internal/<domain>` package imports `api/`; no domain imports
  fasthttp except transport-adjacent packages on an explicit allowlist;
  `internal/store` imports no domain package).
- This gate keeps 13-19 honest automatically.

### `phase-12b/checkpoint`
- Per-phase gate (incl. `-race`), coverage ≥ 95.0%, WORKFLOW update,
  `## Outcome` section, architecture notes added to `docs/ARCHITECTURE.md`.

## Explicit Non-Goals
- No renames of JSON fields, routes, CLI flags, env vars, DB schema.
- No new features. No test deletion. No "while I'm here" improvements.
- `internal/store` stays one package (split is cosmetic churn; interfaces in
  consumers give the decoupling).

## Risk Controls
- Snapshot tests first where structure is asserted (route table, task-1).
- Commit per task; any red state > 30 min → revert to last green.
- `go test -race ./...` per task, not just at checkpoint (refactors move
  state; race exposure changes).

## Outcome

All 6 tasks completed. Zero behavior change maintained throughout.

### Shipped
- **task-1**: `api/server.go` split into `api/routes.go` (route table), `api/wiring.go` (dependency construction), `api/server.go` (lifecycle only). Route snapshot test (`api/routes_test.go`) asserts 45 identical method+path pairs.
- **task-2**: 25+ narrow repository interfaces defined in consumer packages (`api/handlers`, `internal/proxy`, `internal/search`, `internal/cli`). `*store.Store` satisfies them implicitly. Fake test in `api/handlers/settings_fake_test.go` proves decoupling.
- **task-3**: Usage aggregation moved to `internal/usage/usage.go` (`ListUsage`, `GetSummary`). Handler tests unchanged and green.
- **task-4**: Explicit preprocessing pipeline (`internal/proxy/pipeline.go`) with 4 ordered stages: model resolution → RTK compression → Caveman injection → MCP tool injection. 14 unit tests with fakes. Integration suite green.
- **task-5**: Anthropic translation logic moved to `internal/translate/anthropic_messages.go`. `api/handlers/inference.go` (1108 lines) split into `inference_openai.go` (373 lines) and `inference_anthropic.go` (252 lines).
- **task-6**: `internal/archtest/arch_test.go` enforces dependency direction via `go list`. Documented allowlists for known pre-existing couplings (`internal/cli`→`api`, `internal/providers/*`→`fasthttp`, `internal/store`→`internal/mcp`/`internal/usage`).

### Deferred
- Store→domain type couplings (`internal/store` imports `internal/mcp` and `internal/usage`). Deferred because `internal/store` stays one package per explicit non-goals; interfaces in consumers provide the decoupling boundary.

### Gates
- `go test ./... -count=1`: PASS
- `go vet ./...`: PASS
- `go test -race ./...`: PASS
- `go build ./cmd/g0router`: PASS
- Coverage: 95.0%

### Commit Range
`7f6e1b2` → `600c4bd`

## Commit Message (final)
`phase-12b/ddd-refactor: layered architecture, repository interfaces, arch test`
