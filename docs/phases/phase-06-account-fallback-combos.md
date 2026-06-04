# Phase 6: Account Fallback + Combos

> **Depends on**: Phase 2, Phase 4  
> **Unlocks**: Phase 11  
> **Checkpoint**: `PHASE_6_COMPLETE`

---

## Prerequisites

- [x] Phase 2, Phase 4 complete
- [x] `go test ./...` passes
- [x] `go vet ./...` passes

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Backoff strategy | Exponential: 1s → 2s → 4s → ... → max 4min | Standard; matches 9router behavior |
| Per-model locks | Map in connection struct | Model A rate-limited doesn't block model B on same connection |
| Cooldown recovery | Time-based expiry check | Simple; no background goroutine needed |
| Fallback ordering | Round-robin with skip | Fair distribution; skips unavailable connections |

---

## Task 6.1: Account Fallback Engine

### Completed Work

- [x] Write `internal/provider/fallback_test.go` — test FIRST
- [x] Run tests → RED
- [x] Write `internal/provider/fallback.go`
- [x] Run tests → GREEN
- [x] Commit: `phase-6/task-1: account fallback with exponential backoff`

### Pre-conditions

- Previous phase or dependency complete

### TDD Cycle

#### RED: Write Failing Tests First

Implementation evidence: tests for this task were expected to be written before code and to exercise the public API before implementation.

```bash
go test ./...  # Expected: FAIL — new types/functions don't exist
```

#### GREEN: Write Minimum Implementation

Implement only enough code to make the tests pass. No extra features, no premature optimization.

```bash
go test ./...  # Expected: PASS — all tests green
```

#### REFACTOR

- Remove duplication
- Verify no unused imports
- Run `go vet ./...` — must be clean

### Verification

```bash
go test ./... -count=1   # All tests pass
go vet ./...              # Clean
```

### Commit

```
phase-6/task-1: account fallback with exponential backoff
```

---

## Task 6.2: Combo Model Resolution

### Completed Work

- [x] Write `internal/proxy/combo_test.go` — test FIRST
- [x] Run tests → RED
- [x] Write `internal/proxy/combo.go`
- [x] Run tests → GREEN
- [x] Commit: `phase-6/task-2: combo model sequential fallback`

### Pre-conditions

- Task 6.1 complete (or independent)

### TDD Cycle

#### RED: Write Failing Tests First

Implementation evidence: tests for this task were expected to be written before code and to exercise the public API before implementation.

```bash
go test ./...  # Expected: FAIL — new types/functions don't exist
```

#### GREEN: Write Minimum Implementation

Implement only enough code to make the tests pass. No extra features, no premature optimization.

```bash
go test ./...  # Expected: PASS — all tests green
```

#### REFACTOR

- Remove duplication
- Verify no unused imports
- Run `go vet ./...` — must be clean

### Verification

```bash
go test ./... -count=1   # All tests pass
go vet ./...              # Clean
```

### Commit

```
phase-6/task-2: combo model sequential fallback
```

---

## Phase Gate

```bash
go test ./... -count=1    # ALL tests pass
go vet ./...              # Clean
go build ./cmd/g0router   # Binary builds
```

## Phase Checklist

- [x] Task 6.1 complete (Account Fallback Engine)
- [x] Task 6.2 complete (Combo Model Resolution)
- [x] All tests pass: `go test ./...`
- [x] Vet clean: `go vet ./...`
- [x] Build succeeds: `go build ./cmd/g0router`
- [x] All commits follow `phase-6/task-N: description` format
- [x] Update `docs/WORKFLOW.md`: phase_6.status → `DONE`
- [x] **PHASE_6_COMPLETE**
