# Phase 9: MCP Gateway

> **Depends on**: Phase 2  
> **Unlocks**: Phase 11  
> **Checkpoint**: `PHASE_9_COMPLETE`

---

## Prerequisites

- [x] Phase 2 complete
- [x] `go test ./...` passes
- [x] `go vet ./...` passes

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Transport | Support stdio, SSE, streamable-HTTP | Covers all MCP transport options |
| Manifest caching | SQLite `mcp_clients.tool_manifest` | Survives restarts; TTL-based refresh |
| Compact injection | Name + description only | ~90% token savings vs full JSON Schema |
| Full schema | On-demand lookup for tool execution | Only fetched when LLM actually calls the tool |
| Health checks | Periodic ping with auto-reconnect | Detect dead servers early |

---

## Task 9.1: MCP Client Manager

### TODO

- [x] Write `internal/mcp/clientmanager_test.go` — test FIRST
- [x] Implement
- [x] Commit: `phase-9/task-1: mcp client manager`

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
phase-9/task-1: mcp client manager
```

---

## Task 9.2: MCP Tool Manager

### TODO

- [x] Write `internal/mcp/toolmanager_test.go` — test FIRST
- [x] Implement
- [x] Commit: `phase-9/task-2: mcp tool manager`

### Pre-conditions

- Task 9.1 complete (or independent)

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
phase-9/task-2: mcp tool manager
```

---

## Task 9.3: MCP Tool Discovery

### TODO

- [x] Write `internal/mcp/discovery_test.go` — test FIRST
- [x] Implement compact manifests
- [x] Commit: `phase-9/task-3: mcp compact tool discovery`

### Pre-conditions

- Task 9.2 complete (or independent)

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
phase-9/task-3: mcp compact tool discovery
```

---

## Task 9.4: MCP Agent Loop

### TODO

- [x] Write `internal/mcp/agent_test.go` — test FIRST
- [x] Implement multi-turn execution
- [x] Commit: `phase-9/task-4: mcp agent loop`

### Pre-conditions

- Task 9.3 complete (or independent)

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
phase-9/task-4: mcp agent loop
```

---

## Task 9.5: MCP Health Monitor

### TODO

- [x] Write `internal/mcp/healthmonitor_test.go` — test FIRST
- [x] Implement
- [x] Commit: `phase-9/task-5: mcp health monitor`

### Pre-conditions

- Task 9.4 complete (or independent)

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
phase-9/task-5: mcp health monitor
```

---

## Task 9.6: MCP API Handlers + Store

### TODO

- [x] Write tests FIRST
- [x] Implement
- [x] Commit: `phase-9/task-6: mcp api handlers and store`

### Pre-conditions

- Task 9.5 complete (or independent)

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
phase-9/task-6: mcp api handlers and store
```

---

## Phase Gate

```bash
go test ./... -count=1    # ALL tests pass
go vet ./...              # Clean
go build ./cmd/g0router   # Binary builds
```

## Phase Checklist

- [x] Task 9.1 complete (MCP Client Manager)
- [x] Task 9.2 complete (MCP Tool Manager)
- [x] Task 9.3 complete (MCP Tool Discovery)
- [x] Task 9.4 complete (MCP Agent Loop)
- [x] Task 9.5 complete (MCP Health Monitor)
- [x] Task 9.6 complete (MCP API Handlers + Store)
- [x] All tests pass: `go test ./...`
- [x] Vet clean: `go vet ./...`
- [x] Build succeeds: `go build ./cmd/g0router`
- [x] All commits follow `phase-9/task-N: description` format
- [x] Update `docs/WORKFLOW.md`: phase_9.status → `DONE`
- [x] **PHASE_9_COMPLETE**
