# Brief: Phase 12B — DDD & Architecture Refactor (whole project)

**Problem:** The existing codebase mixes two styles — a 44 KB `api/server.go` god file with path-switch routing and business logic in handlers, plus concrete `*store.Store` coupling everywhere. New stage-13-19 code would land on an inconsistent foundation.

**Success criteria:**
- Routing, lifecycle, and wiring split out of `api/server.go`; route inventory provably identical.
- Consumers depend on narrow repository interfaces they define; `*store.Store` satisfies them implicitly.
- Business logic (usage aggregation, inference dispatch decisions) lives in domain packages; handlers are transport-only.
- Architecture conformance test enforces inward dependency direction.

**Non-goals:**
- No JSON field / route / CLI / env / DB schema renames; zero behavior change.
- No new features, no test deletion, no "while I'm here" cleanups.
- `internal/store` stays one flat package (decoupling via consumer interfaces, not a split).

**Constraints:** snake_case JSON + `{data,error}` envelope contracts preserved; coverage ≥ 95.0%; per-task `go test -race ./...`; tests re-pointed, never rewritten; `git mv` to preserve history.

**Verification:** `go test ./... -count=1 && go vet ./... && go build ./cmd/g0router && go test -race ./...` all green with coverage ≥ 95.0%; route-table snapshot test asserts method+path inventory unchanged.

**QA criteria:**
```yaml
qa_skip: type-only-refactor
qa_skip_rationale: "Pure refactor, zero behavior change — no JSON shape/route/CLI change. Existing Go unit + 48 KB integration suite is the net; coverage gate ≥95% guards it."
scenarios: []
manual_smoke: none
```

**Linked artifacts:** architect-plan: ./architect-plan.md; orchestration: ./orchestration.jsonl
