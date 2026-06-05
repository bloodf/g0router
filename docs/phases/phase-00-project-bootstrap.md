# Phase 0: Project Bootstrap

> **Depends on**: nothing  
> **Unlocks**: Phase 1  
> **Checkpoint**: `PHASE_0_COMPLETE`

---

## Prerequisites

- [x] Go 1.24+ installed (`go version`)
- [x] Git initialized (`git init` if needed)
- [x] Working directory is repo root

---

## Task 0.1: Initialize Go Module and Directory Structure

### Completed Work

- [x] Create `go.mod`
- [x] Create `cmd/g0router/main.go`
- [x] Create `.gitignore`
- [x] Create `.env.example`
- [x] Create `CLAUDE.md`
- [x] Create `README.md`
- [x] Create `docs/` directory with all documentation files
- [x] Create `docs/phases/` directory with all phase files
- [x] Verify `go build ./cmd/g0router` succeeds
- [x] Verify `./g0router --version` prints `0.1.0-dev`
- [x] Verify `go vet ./...` passes
- [x] Commit: `phase-0/task-1: initialize go module and project documentation`

### Pre-conditions

- Empty repository (or only docs/config files)
- No existing `go.mod`

### TDD Cycle

**This task is scaffolding — no TDD cycle.** There's no logic to test yet. The first TDD cycle begins in Phase 1.

### Step-by-Step Implementation

#### Step 1: Go module

```bash
go mod init github.com/bloodf/g0router
```

Edit `go.mod` to set Go version:
```
module github.com/bloodf/g0router

go 1.24
```

#### Step 2: Entry point

Create `cmd/g0router/main.go`:

```go
package main

import (
    "flag"
    "fmt"
    "os"
)

var version = "0.1.0-dev"

func main() {
    showVersion := flag.Bool("version", false, "print version and exit")
    flag.Parse()

    if *showVersion {
        fmt.Println(version)
        return
    }

    fmt.Fprintf(os.Stderr, "g0router %s\n", version)
    fmt.Fprintln(os.Stderr, "Use 'g0router serve' to start the server")
    os.Exit(1)
}
```

**Why bare `flag` instead of cobra?** Cobra is added in Phase 5 when we need subcommands. Until then, we have one binary with one flag. No premature dependencies.

**Why `os.Exit(1)` on no args?** Running with no args is a user error at this stage. Explicit failure is better than silent nothing.

#### Step 3: .gitignore

```
# Build
/g0router
*.exe

# Data
data/
*.db
*.db-journal
*.db-wal
*.db-shm

# UI
ui/dist/
ui/node_modules/

# Environment
.env
.env.local

# IDE
.idea/
.vscode/
*.swp
*~

# OS
.DS_Store
Thumbs.db
```

#### Step 4: .env.example

```bash
# g0router configuration — copy to .env and edit

# Server
PORT=20128
DATA_DIR=~/.g0router

# Security — REQUIRED in production
# Generate with: openssl rand -hex 32
JWT_SECRET=
API_KEY_SECRET=

# Access control
REQUIRE_API_KEY=true

# Logging
ENABLE_REQUEST_LOGS=false

# RTK (Response Token Kompression)
RTK_ENABLED=true

# Caveman mode
CAVEMAN_ENABLED=false
CAVEMAN_LEVEL=full   # lite | full | ultra

# Network
# HTTPS_PROXY=http://proxy:8080
```

#### Step 5: Documentation files

Create all files listed in `docs/` and `docs/phases/`. Content already defined in the blueprint plan artifact.

### Verification

```bash
# All must pass:
go build ./cmd/g0router              # exit 0, binary created
./g0router --version                  # prints "0.1.0-dev"
./g0router 2>&1; echo "exit: $?"     # prints usage hint, exit: 1
go vet ./...                          # exit 0
go test ./...                         # exit 0 (no test files = ok)

# All doc files exist and are non-empty:
test -s CLAUDE.md
test -s README.md
test -s docs/ARCHITECTURE.md
test -s docs/PLAN.md
test -s docs/WORKFLOW.md
test -s docs/REFERENCES.md
test -s docs/SCHEMA.md
test -s docs/DEPLOYMENT.md
test -s docs/CONFIG.md
test -s docs/PROVIDERS.md
test -s docs/DIRECTORY_STRUCTURE.md
```

### Post-conditions

- [x] `go.mod` exists with module `github.com/bloodf/g0router`
- [x] `go build ./cmd/g0router` produces binary
- [x] `./g0router --version` outputs version string
- [x] `go vet ./...` passes
- [x] All documentation files exist and are non-empty
- [x] `.gitignore` covers build artifacts, data, UI, env files
- [x] `.env.example` documents all env vars with comments

### Commit

```
phase-0/task-1: initialize go module and project documentation
```

---

## Phase Gate

```bash
go build ./cmd/g0router  && echo "✓ build"
./g0router --version     && echo "✓ version"
go vet ./...             && echo "✓ vet"
go test ./...            && echo "✓ test"
```

All four must succeed.

## Phase Checklist

- [x] Task 0.1 complete
- [x] All verification commands pass
- [x] Committed with `phase-0/task-1: ...`
- [x] Update `docs/WORKFLOW.md`: phase_0.status → `DONE`
- [x] Update `docs/WORKFLOW.md`: current_phase → `1`
- [x] **PHASE_0_COMPLETE**
