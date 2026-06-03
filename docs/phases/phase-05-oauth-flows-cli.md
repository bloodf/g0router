# Phase 5: OAuth Flows + CLI

> **Depends on**: Phase 1  
> **Unlocks**: Phase 6  
> **Checkpoint**: `PHASE_5_COMPLETE`

---

## Prerequisites

- [x] Phase 1 complete
- [x] `go test ./...` passes
- [x] `go vet ./...` passes

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| CLI framework | cobra | Industry standard, subcommand support, built-in help |
| OAuth callback server | Local HTTP on port 54545 (configurable) | Same port as oh-my-pi; avoids random port issues with pre-registered redirect URIs |
| Browser opening | `open` (macOS), `xdg-open` (Linux), `start` (Windows) | Cross-platform, no dependency |
| Token storage | SQLite `connections` table | Already built in Phase 1; encrypted at-rest possible later |
| Device-code polling | Configurable interval (default 5s) | Respects provider `interval` field |
| Client IDs | Hardcoded per provider | Same as 9router/oh-my-pi; these are public OAuth client IDs |

---

## Task 5.1: OAuth Types and Interface

### TODO

- [x] Write `internal/provider/oauth/types_test.go` — test FIRST
- [x] Implement types.go
- [x] Commit: `phase-5/task-1: oauth types and interface`

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
phase-5/task-1: oauth types and interface
```

---

## Task 5.2: Anthropic OAuth (Claude Code)

### TODO

- [x] Write `internal/provider/oauth/anthropic_test.go` — test FIRST
- [x] Implement PKCE flow
- [x] Commit: `phase-5/task-2: anthropic oauth with pkce`

### Pre-conditions

- Task 5.1 complete (or independent)

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
phase-5/task-2: anthropic oauth with pkce
```

---

## Task 5.3: OpenAI Codex OAuth

### TODO

- [x] Write test FIRST
- [x] Implement device-code flow
- [x] Commit: `phase-5/task-3: openai codex device-code oauth`

### Pre-conditions

- Task 5.2 complete (or independent)

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
phase-5/task-3: openai codex device-code oauth
```

---

## Task 5.4: GitHub Copilot OAuth

### TODO

- [x] Write test FIRST
- [x] Implement
- [x] Commit: `phase-5/task-4: github copilot oauth`

### Pre-conditions

- Task 5.3 complete (or independent)

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
phase-5/task-4: github copilot oauth
```

---

## Task 5.5: Cursor PKCE OAuth

### TODO

- [x] Write test FIRST
- [x] Implement
- [x] Commit: `phase-5/task-5: cursor pkce oauth`

### Pre-conditions

- Task 5.4 complete (or independent)

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
phase-5/task-5: cursor pkce oauth
```

---

## Task 5.6: Google OAuth (Gemini CLI, Antigravity)

### TODO

- [x] Write tests FIRST
- [x] Implement both flows
- [x] Commit: `phase-5/task-6: google oauth flows`

### Pre-conditions

- Task 5.5 complete (or independent)

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
phase-5/task-6: google oauth flows
```

---

## Task 5.7: xAI, DeepSeek, GitLab, Kiro OAuth

### TODO

- [x] Write tests FIRST
- [x] Implement each
- [x] Commit: `phase-5/task-7: xai deepseek gitlab kiro oauth`

### Pre-conditions

- Task 5.6 complete (or independent)

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
phase-5/task-7: xai deepseek gitlab kiro oauth
```

---

## Task 5.8: Chinese Provider OAuth

### TODO

- [x] Write tests FIRST
- [x] Implement each
- [x] Commit: `phase-5/task-8: chinese provider oauth`

### Pre-conditions

- Task 5.7 complete (or independent)

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
phase-5/task-8: chinese provider oauth
```

---

## Task 5.9: Token Refresh with Dedup

### TODO

- [x] Write `internal/provider/refresh_test.go` — test FIRST
- [x] Implement singleflight dedup
- [x] Commit: `phase-5/task-9: token refresh with singleflight dedup`

### Pre-conditions

- Task 5.8 complete (or independent)

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
phase-5/task-9: token refresh with singleflight dedup
```

---

## Task 5.10: OAuth HTTP Endpoints

### TODO

- [x] Write handler tests FIRST
- [x] Implement authorize/poll/callback
- [x] Commit: `phase-5/task-10: oauth http endpoints`

### Pre-conditions

- Task 5.9 complete (or independent)

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
phase-5/task-10: oauth http endpoints
```

---

## Task 5.11: CLI Commands (Cobra)

### TODO

- [x] go get github.com/spf13/cobra
- [x] Write CLI tests FIRST
- [x] Implement all commands
- [x] Update cmd/g0router/main.go
- [x] Commit: `phase-5/task-11: cobra cli with all commands`

### Pre-conditions

- Task 5.10 complete (or independent)

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
phase-5/task-11: cobra cli with all commands
```

---

## Phase Gate

```bash
go test ./... -count=1    # ALL tests pass
go vet ./...              # Clean
go build ./cmd/g0router   # Binary builds
```

## Phase Checklist

- [x] Task 5.1 complete (OAuth Types and Interface)
- [x] Task 5.2 complete (Anthropic OAuth (Claude Code))
- [x] Task 5.3 complete (OpenAI Codex OAuth)
- [x] Task 5.4 complete (GitHub Copilot OAuth)
- [x] Task 5.5 complete (Cursor PKCE OAuth)
- [x] Task 5.6 complete (Google OAuth (Gemini CLI, Antigravity))
- [x] Task 5.7 complete (xAI, DeepSeek, GitLab, Kiro OAuth)
- [x] Task 5.8 complete (Chinese Provider OAuth)
- [x] Task 5.9 complete (Token Refresh with Dedup)
- [x] Task 5.10 complete (OAuth HTTP Endpoints)
- [x] Task 5.11 complete (CLI Commands (Cobra))
- [x] All tests pass: `go test ./...`
- [x] Vet clean: `go vet ./...`
- [x] Build succeeds: `go build ./cmd/g0router`
- [x] All commits follow `phase-5/task-N: description` format
- [x] Update `docs/WORKFLOW.md`: phase_5.status → `DONE`
- [x] **PHASE_5_COMPLETE**
