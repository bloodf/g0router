# Phase 11: Packaging, Deployment + Polish

> **Depends on**: All previous phases  
> **Unlocks**: PROJECT COMPLETE  
> **Checkpoint**: `PHASE_11_COMPLETE`

---

## Prerequisites

- [ ] All previous phases complete
- [ ] `go test ./...` passes
- [ ] `go vet ./...` passes

---

## Task 11.1: Makefile

### TODO

- [ ] Write Makefile with build, test, lint, ui, docker, install targets
- [ ] Verify `make build` + `make test`
- [ ] Commit: `phase-11/task-1: makefile`

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

### TODO

- [ ] Write `internal/cli/install_test.go` — test FIRST
- [ ] Write `deploy/g0router.service`
- [ ] Write `deploy/g0router.default`
- [ ] Implement install/uninstall commands
- [ ] Commit: `phase-11/task-2: systemd service and install command`

### Pre-conditions

- Task 11.1 complete (or independent)

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
phase-11/task-2: systemd service and install command
```

---

## Task 11.3: Docker Support

### TODO

- [ ] Write `Dockerfile` (multi-stage)
- [ ] Write `docker-compose.yml`
- [ ] Write `.dockerignore`
- [ ] Verify `docker build` succeeds
- [ ] Commit: `phase-11/task-3: docker support`

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

### TODO

- [ ] Verify all docs are accurate
- [ ] Update README with tested examples
- [ ] Commit: `phase-11/task-4: documentation polish`

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

### TODO

- [ ] Write E2E test (server start → login → request → usage)
- [ ] Run with real API keys
- [ ] Commit: `phase-11/task-5: end-to-end integration tests`

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

- [ ] Task 11.1 complete (Makefile)
- [ ] Task 11.2 complete (systemd + Install CLI)
- [ ] Task 11.3 complete (Docker Support)
- [ ] Task 11.4 complete (.env.example + README)
- [ ] Task 11.5 complete (Final Integration Test Suite)
- [ ] All tests pass: `go test ./...`
- [ ] Vet clean: `go vet ./...`
- [ ] Build succeeds: `go build ./cmd/g0router`
- [ ] All commits follow `phase-11/task-N: description` format
- [ ] Update `docs/WORKFLOW.md`: phase_11.status → `DONE`
- [ ] **PHASE_11_COMPLETE** → **PROJECT COMPLETE**
