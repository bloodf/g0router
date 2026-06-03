# Phase 4: Persistence + Provider Registry

> **Depends on**: Phase 1  
> **Unlocks**: Phase 6  
> **Checkpoint**: `PHASE_4_COMPLETE`

---

## Prerequisites

- [x] Phase 1 complete
- [x] `go test ./...` passes
- [x] `go vet ./...` passes

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Registry | In-memory map rebuilt from DB on startup | Fast lookups; DB is source of truth |
| Round-robin | Atomic counter per provider | Lock-free, fair distribution |
| Combo resolution | Sequential fallback | Simple, predictable; first success wins |
| Alias resolution | DB lookup → cache with TTL | Aliases change rarely; avoid DB hit per request |

---

## Task 4.1: Provider Registry

### TODO

- [x] Write `internal/provider/registry_test.go` — test FIRST
- [x] Run tests → RED
- [x] Write `internal/provider/registry.go`
- [x] Run tests → GREEN
- [x] Commit: `phase-4/task-1: provider registry with model resolution`

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
phase-4/task-1: provider registry with model resolution
```

---

## Task 4.2: Connection Management with Round-Robin

### TODO

- [x] Write `internal/provider/connection_test.go` — test FIRST
- [x] Run tests → RED
- [x] Write `internal/provider/connection.go`
- [x] Run tests → GREEN
- [x] Commit: `phase-4/task-2: round-robin connection selection`

### Pre-conditions

- Task 4.1 complete (or independent)

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
phase-4/task-2: round-robin connection selection
```

---

## Task 4.3: Combos Store + Resolver

### TODO

- [x] Write `internal/store/combos_test.go` — test FIRST
- [x] Run tests → RED
- [x] Write `internal/store/combos.go` + `internal/proxy/combo.go`
- [x] Run tests → GREEN
- [x] Commit: `phase-4/task-3: combo model store and resolver`

### Pre-conditions

- Task 4.2 complete (or independent)

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
phase-4/task-3: combo model store and resolver
```

---

## Task 4.4: Model Aliases + Pricing Overrides

### TODO

- [x] Write tests FIRST
- [x] Implement `internal/store/aliases.go` + `internal/store/pricing.go`
- [x] Commit: `phase-4/task-4: model aliases and pricing overrides`

### Pre-conditions

- Task 4.3 complete (or independent)

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
phase-4/task-4: model aliases and pricing overrides
```

---

## Task 4.5: Management API Handlers

### TODO

- [x] Write handler tests FIRST
- [x] Implement all CRUD endpoints
- [x] Commit: `phase-4/task-5: management API endpoints`

### Pre-conditions

- Task 4.4 complete (or independent)

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
phase-4/task-5: management API endpoints
```

---

## Phase Gate

```bash
go test ./... -count=1    # ALL tests pass
go vet ./...              # Clean
go build ./cmd/g0router   # Binary builds
```

## Phase Checklist

- [x] Task 4.1 complete (Provider Registry)
- [x] Task 4.2 complete (Connection Management with Round-Robin)
- [x] Task 4.3 complete (Combos Store + Resolver)
- [x] Task 4.4 complete (Model Aliases + Pricing Overrides)
- [x] Task 4.5 complete (Management API Handlers)
- [x] All tests pass: `go test ./...`
- [x] Vet clean: `go vet ./...`
- [x] Build succeeds: `go build ./cmd/g0router`
- [x] All commits follow `phase-4/task-N: description` format
- [x] Update `docs/WORKFLOW.md`: phase_4.status → `DONE`
- [x] **PHASE_4_COMPLETE**
