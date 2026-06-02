# Phase 6: Account Fallback + Combos

> **Depends on**: Phase 2, Phase 4  
> **Unlocks**: Phase 11  
> **Checkpoint**: `PHASE_6_COMPLETE`

---

## Prerequisites

- [ ] Phase 2, Phase 4 complete
- [ ] `go test ./...` passes
- [ ] `go vet ./...` passes

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Backoff strategy | Exponential: 1s → 2s → 4s → ... → max 4min | Standard; matches 9router behavior |
| Per-model locks | Map in connection struct | Model A rate-limited doesn't block model B on same connection |
| Cooldown recovery | Time-based expiry check | Simple; no background goroutine needed |
| Fallback ordering | Round-robin with skip | Fair distribution; skips unavailable connections |

---

## Task 6.1: Account Fallback Engine

### TODO

- [ ] Write `internal/provider/fallback_test.go` — test FIRST
- [ ] Run tests → RED
- [ ] Write `internal/provider/fallback.go`
- [ ] Run tests → GREEN
- [ ] Commit: `phase-6/task-1: account fallback with exponential backoff`

### Pre-conditions

- Previous phase or dependency complete

### TDD Cycle

#### RED: Write Failing Tests First

Create the test file referenced in TODO. Write tests that exercise the public API of the new code. Tests must compile but FAIL because the implementation doesn't exist.

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

### TODO

- [ ] Write `internal/proxy/combo_test.go` — test FIRST
- [ ] Run tests → RED
- [ ] Write `internal/proxy/combo.go`
- [ ] Run tests → GREEN
- [ ] Commit: `phase-6/task-2: combo model sequential fallback`

### Pre-conditions

- Task 6.1 complete (or independent)

### TDD Cycle

#### RED: Write Failing Tests First

Create the test file referenced in TODO. Write tests that exercise the public API of the new code. Tests must compile but FAIL because the implementation doesn't exist.

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

- [ ] Task 6.1 complete (Account Fallback Engine)
- [ ] Task 6.2 complete (Combo Model Resolution)
- [ ] All tests pass: `go test ./...`
- [ ] Vet clean: `go vet ./...`
- [ ] Build succeeds: `go build ./cmd/g0router`
- [ ] All commits follow `phase-6/task-N: description` format
- [ ] Update `docs/WORKFLOW.md`: phase_6.status → `DONE`
- [ ] **PHASE_6_COMPLETE**
