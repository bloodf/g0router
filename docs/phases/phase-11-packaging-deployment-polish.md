# Phase 11: Packaging, Deployment + Polish

> **Depends on**: All previous phases  
> **Unlocks**: Phase 12
> **Checkpoint**: `PHASE_11_COMPLETE`

---

## Prerequisites

- [x] All previous phases complete
- [x] `go test ./...` passes
- [x] `go vet ./...` passes

---

## Task 11.1: Makefile

### Completed Work

- [x] Write Makefile with build, test, lint, ui, docker, install targets
- [x] Verify `make build` + `make test`
- [x] Commit: `phase-11/task-1: makefile`

### Pre-conditions

- Previous phase or dependency complete

### TDD Cycle

#### RED: Write Failing Tests First

Write tests for the new functionality before implementing.

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
phase-11/task-1: makefile
```

---

## Task 11.2: systemd + Install CLI

### Completed Work

- [x] Write `internal/cli/install_test.go` — test FIRST
- [x] Write `deploy/g0router.service`
- [x] Write `deploy/g0router.default`
- [x] Implement install/uninstall commands
- [x] Commit: `phase-11/task-2: systemd service and install command`

### Pre-conditions

- Task 11.1 complete (or independent)

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
phase-11/task-2: systemd service and install command
```

---

## Task 11.3: Docker Support

### Completed Work

- [x] Write `Dockerfile` (multi-stage)
- [x] Write `docker-compose.yml`
- [x] Write `.dockerignore`
- [x] Verify `docker build` succeeds
- [x] Commit: `phase-11/task-3: docker support`

### Pre-conditions

- Task 11.2 complete (or independent)

### TDD Cycle

#### RED: Write Failing Tests First

Write tests for the new functionality before implementing.

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
phase-11/task-3: docker support
```

---

## Task 11.4: .env.example + README

### Completed Work

- [x] Verify all docs are accurate
- [x] Update README with tested examples
- [x] Commit: `phase-11/task-4: documentation polish`

### Pre-conditions

- Task 11.3 complete (or independent)

### TDD Cycle

#### RED: Write Failing Tests First

Write tests for the new functionality before implementing.

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
phase-11/task-4: documentation polish
```

---

## Task 11.5: Final Integration Test Suite

### Completed Work

- [x] Write E2E test (server start → login → request → usage)
- [x] Run with real API keys
- [x] Commit: `phase-11/task-5: end-to-end integration tests`

### Pre-conditions

- Task 11.4 complete (or independent)

### TDD Cycle

#### RED: Write Failing Tests First

Write tests for the new functionality before implementing.

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
phase-11/task-5: end-to-end integration tests
```

---

## Phase Gate

```bash
go test ./... -count=1    # ALL tests pass
go vet ./...              # Clean
go build ./cmd/g0router   # Binary builds
```

## Phase Checklist

- [x] Task 11.1 complete (Makefile)
- [x] Task 11.2 complete (systemd + Install CLI)
- [x] Task 11.3 complete (Docker Support)
- [x] Task 11.4 complete (.env.example + README)
- [x] Task 11.5 complete (Final Integration Test Suite)
- [x] All tests pass: `go test ./...`
- [x] Vet clean: `go vet ./...`
- [x] Build succeeds: `go build ./cmd/g0router`
- [x] All commits follow `phase-11/task-N: description` format
- [x] Update `docs/WORKFLOW.md`: phase_11.status → `DONE`
- [x] **PHASE_11_COMPLETE** -> advance to Phase 12
