# Phase 8: Usage Tracking + Cost + Logging

> **Depends on**: Phase 1  
> **Unlocks**: Phase 11  
> **Checkpoint**: `PHASE_8_COMPLETE`

---

## Prerequisites

- [ ] Phase 1 complete
- [ ] `go test ./...` passes
- [ ] `go vet ./...` passes

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Pricing source | Embedded Go map + DB overrides | No external API dependency; DB overrides for custom pricing |
| Cost precision | float64 USD | Sufficient for per-request tracking; aggregate in DB |
| Quota fetching | On-demand with 5-min cache | Don't poll providers constantly |
| Logging toggle | Per-request via settings | Full body logging is expensive; off by default |

---

## Task 8.1: Usage Extraction

### TODO

- [ ] Write `internal/usage/tracker_test.go` — test FIRST
- [ ] Implement
- [ ] Commit: `phase-8/task-1: usage extraction from provider responses`

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
phase-8/task-1: usage extraction from provider responses
```

---

## Task 8.2: Model Pricing Catalog

### TODO

- [ ] Write `internal/modelcatalog/pricing_test.go` — test FIRST
- [ ] Implement
- [ ] Commit: `phase-8/task-2: model pricing catalog`

### Pre-conditions

- Task 8.1 complete (or independent)

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
phase-8/task-2: model pricing catalog
```

---

## Task 8.3: Cost Calculation

### TODO

- [ ] Write `internal/usage/cost_test.go` — test FIRST
- [ ] Implement
- [ ] Commit: `phase-8/task-3: cost calculation`

### Pre-conditions

- Task 8.2 complete (or independent)

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
phase-8/task-3: cost calculation
```

---

## Task 8.4: Provider Quota Fetchers

### TODO

- [ ] Write `internal/usage/quota_test.go` — test FIRST
- [ ] Implement per-provider fetchers
- [ ] Commit: `phase-8/task-4: provider quota fetchers`

### Pre-conditions

- Task 8.3 complete (or independent)

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
phase-8/task-4: provider quota fetchers
```

---

## Task 8.5: Request/Response Logging

### TODO

- [ ] Write `internal/logging/logger_test.go` — test FIRST
- [ ] Implement
- [ ] Commit: `phase-8/task-5: request response logging`

### Pre-conditions

- Task 8.4 complete (or independent)

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
phase-8/task-5: request response logging
```

---

## Task 8.6: Usage + Logging API Handlers

### TODO

- [ ] Write handler tests FIRST
- [ ] Implement endpoints
- [ ] Commit: `phase-8/task-6: usage and logging api handlers`

### Pre-conditions

- Task 8.5 complete (or independent)

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
phase-8/task-6: usage and logging api handlers
```

---

## Phase Gate

```bash
go test ./... -count=1    # ALL tests pass
go vet ./...              # Clean
go build ./cmd/g0router   # Binary builds
```

## Phase Checklist

- [ ] Task 8.1 complete (Usage Extraction)
- [ ] Task 8.2 complete (Model Pricing Catalog)
- [ ] Task 8.3 complete (Cost Calculation)
- [ ] Task 8.4 complete (Provider Quota Fetchers)
- [ ] Task 8.5 complete (Request/Response Logging)
- [ ] Task 8.6 complete (Usage + Logging API Handlers)
- [ ] All tests pass: `go test ./...`
- [ ] Vet clean: `go vet ./...`
- [ ] Build succeeds: `go build ./cmd/g0router`
- [ ] All commits follow `phase-8/task-N: description` format
- [ ] Update `docs/WORKFLOW.md`: phase_8.status → `DONE`
- [ ] **PHASE_8_COMPLETE**
