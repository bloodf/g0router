# Phase 7: RTK + Caveman

> **Depends on**: Phase 1  
> **Unlocks**: Phase 11  
> **Checkpoint**: `PHASE_7_COMPLETE`

---

## Prerequisites

- [ ] Phase 1 complete
- [ ] `go test ./...` passes
- [ ] `go vet ./...` passes

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Detection | First 1KB inspection | Fast, sufficient for format identification |
| Filter purity | `func(string) string` | No side effects, easily testable, composable |
| Message scanning | Only `tool` role / `tool_result` blocks | Don't compress user text or assistant responses |
| Caveman injection | Prepend to system message | Works with all providers; format-aware |
| Config | Per-request toggle via settings | Can enable/disable without restart |

---

## Task 7.1: RTK Autodetect

### TODO

- [ ] Write `internal/rtk/autodetect_test.go` — test FIRST with real samples
- [ ] Run tests → RED
- [ ] Write `internal/rtk/autodetect.go`
- [ ] Run tests → GREEN
- [ ] Commit: `phase-7/task-1: rtk content format autodetection`

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
phase-7/task-1: rtk content format autodetection
```

---

## Task 7.2: RTK Filters (11 total)

### TODO

- [ ] For EACH filter: write test FIRST with real-world sample
- [ ] Implement filter
- [ ] Run tests → GREEN
- [ ] Commit per filter or batch: `phase-7/task-2: rtk filters`

### Pre-conditions

- Task 7.1 complete (or independent)

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

---

## Task 7.3: RTK Message Compression

### TODO

- [ ] Write `internal/rtk/rtk_test.go` — test FIRST
- [ ] Run tests → RED
- [ ] Write `internal/rtk/rtk.go`
- [ ] Run tests → GREEN
- [ ] Commit: `phase-7/task-3: rtk message compression`

### Pre-conditions

- Task 7.2 complete (or independent)

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
phase-7/task-3: rtk message compression
```

---

## Task 7.4: Caveman Prompt Injection

### TODO

- [ ] Write `internal/rtk/caveman_test.go` — test FIRST
- [ ] Run tests → RED
- [ ] Write `internal/rtk/caveman.go` + prompts.go
- [ ] Run tests → GREEN
- [ ] Commit: `phase-7/task-4: caveman prompt injection`

### Pre-conditions

- Task 7.3 complete (or independent)

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
phase-7/task-4: caveman prompt injection
```

---

## Phase Gate

```bash
go test ./... -count=1    # ALL tests pass
go vet ./...              # Clean
go build ./cmd/g0router   # Binary builds
```

## Phase Checklist

- [ ] Task 7.1 complete (RTK Autodetect)
- [ ] Task 7.2 complete (RTK Filters (11 total))
- [ ] Task 7.3 complete (RTK Message Compression)
- [ ] Task 7.4 complete (Caveman Prompt Injection)
- [ ] All tests pass: `go test ./...`
- [ ] Vet clean: `go vet ./...`
- [ ] Build succeeds: `go build ./cmd/g0router`
- [ ] All commits follow `phase-7/task-N: description` format
- [ ] Update `docs/WORKFLOW.md`: phase_7.status → `DONE`
- [ ] **PHASE_7_COMPLETE**
