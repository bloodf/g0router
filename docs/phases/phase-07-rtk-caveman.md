# Phase 7: RTK + Caveman

> **Depends on**: Phase 1  
> **Unlocks**: Phase 11  
> **Checkpoint**: `PHASE_7_COMPLETE`

---

## Prerequisites

- [x] Phase 1 complete
- [x] `go test ./...` passes
- [x] `go vet ./...` passes

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

- [x] Write `internal/rtk/autodetect_test.go` — test FIRST with real samples
- [x] Run tests → RED
- [x] Write `internal/rtk/autodetect.go`
- [x] Run tests → GREEN
- [x] Commit: `phase-7/task-1: rtk content format autodetection`

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

- [x] For EACH filter: write test FIRST with real-world sample
- [x] Implement filter
- [x] Run tests → GREEN
- [x] Commit per filter or batch: `phase-7/task-2: rtk filters`

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

- [x] Write `internal/rtk/rtk_test.go` — test FIRST
- [x] Run tests → RED
- [x] Write `internal/rtk/rtk.go`
- [x] Run tests → GREEN
- [x] Commit: `phase-7/task-3: rtk message compression`

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

- [x] Write `internal/rtk/caveman_test.go` — test FIRST
- [x] Run tests → RED
- [x] Write `internal/rtk/caveman.go` + prompts.go
- [x] Run tests → GREEN
- [x] Commit: `phase-7/task-4: caveman prompt injection`

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

- [x] Task 7.1 complete (RTK Autodetect)
- [x] Task 7.2 complete (RTK Filters (11 total))
- [x] Task 7.3 complete (RTK Message Compression)
- [x] Task 7.4 complete (Caveman Prompt Injection)
- [x] All tests pass: `go test ./...`
- [x] Vet clean: `go vet ./...`
- [x] Build succeeds: `go build ./cmd/g0router`
- [x] All commits follow `phase-7/task-N: description` format
- [x] Update `docs/WORKFLOW.md`: phase_7.status → `DONE`
- [x] **PHASE_7_COMPLETE**
