# Phase 4: Persistence + Provider Registry

> **Depends on**: Phase 1  
> **Unlocks**: Phase 6  
> **Checkpoint**: `PHASE_4_COMPLETE`

---

## Prerequisites

- [ ] Phase 1 complete
- [ ] `go test ./...` passes
- [ ] `go vet ./...` passes

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

- [ ] Write `internal/provider/registry_test.go` — test FIRST
- [ ] Run tests → RED
- [ ] Write `internal/provider/registry.go`
- [ ] Run tests → GREEN
- [ ] Commit: `phase-4/task-1: provider registry with model resolution`

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

- [ ] Write `internal/provider/connection_test.go` — test FIRST
- [ ] Run tests → RED
- [ ] Write `internal/provider/connection.go`
- [ ] Run tests → GREEN
- [ ] Commit: `phase-4/task-2: round-robin connection selection`

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

- [ ] Write `internal/store/combos_test.go` — test FIRST
- [ ] Run tests → RED
- [ ] Write `internal/store/combos.go` + `internal/proxy/combo.go`
- [ ] Run tests → GREEN
- [ ] Commit: `phase-4/task-3: combo model store and resolver`

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

- [ ] Write tests FIRST
- [ ] Implement `internal/store/aliases.go` + `internal/store/pricing.go`
- [ ] Commit: `phase-4/task-4: model aliases and pricing overrides`

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

- [ ] Write handler tests FIRST
- [ ] Implement all CRUD endpoints
- [ ] Commit: `phase-4/task-5: management API endpoints`

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

- [ ] Task 4.1 complete (Provider Registry)
- [ ] Task 4.2 complete (Connection Management with Round-Robin)
- [ ] Task 4.3 complete (Combos Store + Resolver)
- [ ] Task 4.4 complete (Model Aliases + Pricing Overrides)
- [ ] Task 4.5 complete (Management API Handlers)
- [ ] All tests pass: `go test ./...`
- [ ] Vet clean: `go vet ./...`
- [ ] Build succeeds: `go build ./cmd/g0router`
- [ ] All commits follow `phase-4/task-N: description` format
- [ ] Update `docs/WORKFLOW.md`: phase_4.status → `DONE`
- [ ] **PHASE_4_COMPLETE**
