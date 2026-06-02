# Phase 10: Dashboard UI

> **Depends on**: Phase 2  
> **Unlocks**: Phase 11  
> **Checkpoint**: `PHASE_10_COMPLETE`

---

## Prerequisites

- [ ] Phase 2 complete
- [ ] `go test ./...` passes
- [ ] `go vet ./...` passes

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

### TODO

- [ ] Initialize Vite + React + Tailwind
- [ ] Verify `npm run build` succeeds
- [ ] Commit: `phase-10/task-1: ui scaffold`

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

### TODO

- [ ] Build overview page
- [ ] Commit: `phase-10/task-2: dashboard overview page`

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

### TODO

- [ ] Build API key + RTK + caveman controls
- [ ] Commit: `phase-10/task-3: endpoint configuration page`

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

### TODO

- [ ] Build provider grid + connect flow
- [ ] Commit: `phase-10/task-4: providers management page`

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

### TODO

- [ ] Build charts + request table
- [ ] Commit: `phase-10/task-5: usage analytics page`

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

### TODO

- [ ] Build quota bars
- [ ] Commit: `phase-10/task-6: quota display page`

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

### TODO

- [ ] Build combos, MCP, settings pages
- [ ] Commit: `phase-10/task-7: combos mcp settings pages`

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

### TODO

- [ ] Write `embed.go`
- [ ] Update server to serve embedded files
- [ ] Verify `go build` includes UI
- [ ] Commit: `phase-10/task-8: embed ui in go binary`

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

- [ ] Task 10.1 complete (UI Scaffold)
- [ ] Task 10.2 complete (Dashboard Page)
- [ ] Task 10.3 complete (Endpoint Page)
- [ ] Task 10.4 complete (Providers Page)
- [ ] Task 10.5 complete (Usage Page)
- [ ] Task 10.6 complete (Quota Page)
- [ ] Task 10.7 complete (Remaining Pages)
- [ ] Task 10.8 complete (Embed UI in Go Binary)
- [ ] All tests pass: `go test ./...`
- [ ] Vet clean: `go vet ./...`
- [ ] Build succeeds: `go build ./cmd/g0router`
- [ ] All commits follow `phase-10/task-N: description` format
- [ ] Update `docs/WORKFLOW.md`: phase_10.status → `DONE`
- [ ] **PHASE_10_COMPLETE**
