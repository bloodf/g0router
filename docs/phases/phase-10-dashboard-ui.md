# Phase 10: Dashboard UI

> **Depends on**: Phase 2  
> **Unlocks**: Phase 11  
> **Checkpoint**: `PHASE_10_COMPLETE`

---

## Prerequisites

- [x] Phase 2 complete
- [x] `go test ./...` passes
- [x] `go vet ./...` passes

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Framework | React 19 + Vite | Fast build, widely known |
| Styling | TailwindCSS v4 | Utility-first, small bundle |
| Charts | Recharts or lightweight SVG | No heavy chart library |
| State | React Query for server state | Cache + refetch; no Redux overhead |
| Embedding | `embed.FS` in Go | Single binary; no separate UI server |
| API client | fetch + typed wrappers | No axios dependency |

---

## Task 10.1: UI Scaffold

### Completed Work

- [x] Initialize Vite + React + Tailwind
- [x] Verify `npm run build` succeeds
- [x] Commit: `phase-10/task-1: ui scaffold`

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
phase-10/task-1: ui scaffold
```

---

## Task 10.2: Dashboard Page

### Completed Work

- [x] Build overview page
- [x] Commit: `phase-10/task-2: dashboard overview page`

### Pre-conditions

- Task 10.1 complete (or independent)

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
phase-10/task-2: dashboard overview page
```

---

## Task 10.3: Endpoint Page

### Completed Work

- [x] Build API key + RTK + caveman controls
- [x] Commit: `phase-10/task-3: endpoint configuration page`

### Pre-conditions

- Task 10.2 complete (or independent)

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
phase-10/task-3: endpoint configuration page
```

---

## Task 10.4: Providers Page

### Completed Work

- [x] Build provider grid + connect flow
- [x] Commit: `phase-10/task-4: providers management page`

### Pre-conditions

- Task 10.3 complete (or independent)

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
phase-10/task-4: providers management page
```

---

## Task 10.5: Usage Page

### Completed Work

- [x] Build charts + request table
- [x] Commit: `phase-10/task-5: usage analytics page`

### Pre-conditions

- Task 10.4 complete (or independent)

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
phase-10/task-5: usage analytics page
```

---

## Task 10.6: Quota Page

### Completed Work

- [x] Build quota bars
- [x] Commit: `phase-10/task-6: quota display page`

### Pre-conditions

- Task 10.5 complete (or independent)

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
phase-10/task-6: quota display page
```

---

## Task 10.7: Remaining Pages

### Completed Work

- [x] Build combos, MCP, settings pages
- [x] Commit: `phase-10/task-7: combos mcp settings pages`

### Pre-conditions

- Task 10.6 complete (or independent)

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
phase-10/task-7: combos mcp settings pages
```

---

## Task 10.8: Embed UI in Go Binary

### Completed Work

- [x] Write `embed.go`
- [x] Update server to serve embedded files
- [x] Verify `go build` includes UI
- [x] Commit: `phase-10/task-8: embed ui in go binary`

### Pre-conditions

- Task 10.7 complete (or independent)

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
phase-10/task-8: embed ui in go binary
```

---

## Phase Gate

```bash
go test ./... -count=1    # ALL tests pass
go vet ./...              # Clean
go build ./cmd/g0router   # Binary builds
```

## Phase Checklist

- [x] Task 10.1 complete (UI Scaffold)
- [x] Task 10.2 complete (Dashboard Page)
- [x] Task 10.3 complete (Endpoint Page)
- [x] Task 10.4 complete (Providers Page)
- [x] Task 10.5 complete (Usage Page)
- [x] Task 10.6 complete (Quota Page)
- [x] Task 10.7 complete (Remaining Pages)
- [x] Task 10.8 complete (Embed UI in Go Binary)
- [x] All tests pass: `go test ./...`
- [x] Vet clean: `go vet ./...`
- [x] Build succeeds: `go build ./cmd/g0router`
- [x] All commits follow `phase-10/task-N: description` format
- [x] Update `docs/WORKFLOW.md`: phase_10.status → `DONE`
- [x] **PHASE_10_COMPLETE**
