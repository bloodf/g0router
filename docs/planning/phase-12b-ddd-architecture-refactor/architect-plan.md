# Architect Plan: Phase 12B — DDD & Architecture Refactor

Canonical spec: [`docs/phases/phase-12b-ddd-architecture-refactor.md`](../../phases/phase-12b-ddd-architecture-refactor.md)

## Summary
- Approach: **strangler, not big-bang** — one domain per task, move code, re-point callers, keep old tests passing; `git mv` to preserve history.
- Target: layered DDD-lite (transport → domain → repository) so stage-13-19 code lands on one consistent foundation, not two coexisting styles.
- Already-good packages (`internal/providers`, `provider`, `rtk`, `cache`, `ratelimit`, `traffic`, `usage`, `translate`, `streaming`, `modelcatalog`, `notify`, `mcp`) are left structurally intact; only boundaries verified.
- task-1: split `api/server.go` god file → `api/routes.go` (route→handler table), `api/server.go` (lifecycle: listen/shutdown/TLS), `api/wiring.go` (deps). Snapshot route table first.
- task-2: each consumer defines the narrow repository interface it uses; replace `*store.Store` fields with interfaces; no store code changes; prove one fake.
- task-3: move usage aggregation/summary logic from `api/handlers/usage.go` into `internal/usage`; handler becomes parse → call domain → envelope.
- task-4 (highest risk, done last): extract inference decision logic (model resolution order, RTK/Caveman, MCP injection) into `internal/proxy`/`internal/dispatch` as an ordered pipeline; integration tests are the net.
- task-5: handler hygiene sweep — remaining business rules move to owning domain; pure CRUD stays; split files > 800 lines by aggregate.
- task-6: add `internal/archtest/arch_test.go` asserting dependency direction via `go list`.
- checkpoint: per-phase gate (incl. `-race`), coverage ≥ 95.0%, WORKFLOW + `## Outcome` + `docs/ARCHITECTURE.md` notes.

## Layer/architecture notes
- Respects §1.8 direction (strictly inward): `api/handlers/` transport-only after the sweep; domain packages own business logic; `internal/store` stays persistence-only.
- §1.8 repository rule: interfaces are defined in the CONSUMER package, not in store; `*store.Store` satisfies them implicitly so no store edits are needed (task-2).
- §1.8 fasthttp boundary: domain packages must not import fasthttp; task-6's arch test mechanically enforces this and the no-`api/`-import and store-imports-no-domain rules.
- §1.8 file-size rule (200-400 typical, 800 max, split by aggregate) is applied in task-1 (server god file) and task-5 (oversized handlers).
