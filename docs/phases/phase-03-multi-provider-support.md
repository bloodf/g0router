# Phase 3: Multi-Provider Support

> **Depends on**: Phase 2  
> **Unlocks**: Phase 6, Phase 11  
> **Checkpoint**: `PHASE_3_COMPLETE`

---

## Prerequisites

- [ ] Phase 2 complete
- [ ] `go test ./...` passes
- [ ] `go vet ./...` passes

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Translation direction | OpenAI as canonical internal format | Most clients (Claude Code, Codex, Cursor) send OpenAI format |
| OpenAI-compatible providers | Single `openaicompat` package with config | 13 providers use identical wire format, only base URL differs |
| Anthropic types | Separate wire types | Messages API differs significantly (system field, content blocks, tool_result) |
| Gemini types | Separate wire types | `generateContent` format is structurally different |
| Thinking blocks | Preserve as-is | Claude's thinking/reasoning blocks pass through without modification |
| Format detection | Heuristic from request body | If `messages` field + no `system` at top level → OpenAI; `system` as string → Anthropic; `contents` → Gemini |

---

## Task 3.1: Anthropic Provider

### TODO

- [ ] Write `internal/providers/anthropic/anthropic_test.go` — test FIRST
- [ ] Run tests → RED (types don't exist)
- [ ] Write `internal/providers/anthropic/anthropic.go`
- [ ] Write `internal/providers/anthropic/types.go`
- [ ] Write `internal/providers/anthropic/errors.go`
- [ ] Run tests → GREEN
- [ ] Commit: `phase-3/task-1: anthropic provider with Messages API`

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
phase-3/task-1: anthropic provider with Messages API
```

---

## Task 3.2: Format Translation Engine

### TODO

- [ ] Write `internal/translate/detect_test.go` — test FIRST
- [ ] Write `internal/translate/anthropic_test.go` — test FIRST
- [ ] Run tests → RED
- [ ] Write `internal/translate/detect.go`
- [ ] Write `internal/translate/openai.go`
- [ ] Write `internal/translate/anthropic.go`
- [ ] Run tests → GREEN
- [ ] Commit: `phase-3/task-2: format translation engine`

### Pre-conditions

- Task 3.1 complete (or independent)

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
phase-3/task-2: format translation engine
```

---

## Task 3.3: OpenAI-Compatible Providers (batch)

### TODO

- [ ] Write `internal/providers/openaicompat/provider_test.go` — parameterized test FIRST
- [ ] Run tests → RED
- [ ] Write `internal/providers/openaicompat/provider.go`
- [ ] Write `internal/providers/openaicompat/registry.go`
- [ ] Run tests → GREEN
- [ ] Commit: `phase-3/task-3: openai-compatible provider factory`

### Pre-conditions

- Task 3.2 complete (or independent)

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
phase-3/task-3: openai-compatible provider factory
```

---

## Task 3.4: Gemini Provider

### TODO

- [ ] Write `internal/providers/gemini/gemini_test.go` — test FIRST
- [ ] Run tests → RED
- [ ] Write `internal/providers/gemini/gemini.go` + types.go
- [ ] Run tests → GREEN
- [ ] Commit: `phase-3/task-4: gemini provider with generateContent`

### Pre-conditions

- Task 3.3 complete (or independent)

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
phase-3/task-4: gemini provider with generateContent
```

---

## Task 3.5: Gemini Format Translation

### TODO

- [ ] Write `internal/translate/gemini_test.go` — test FIRST
- [ ] Run tests → RED
- [ ] Write `internal/translate/gemini.go`
- [ ] Run tests → GREEN
- [ ] Commit: `phase-3/task-5: openai to gemini format translation`

### Pre-conditions

- Task 3.4 complete (or independent)

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
phase-3/task-5: openai to gemini format translation
```

---

## Task 3.6: Vertex AI Provider

### TODO

- [ ] Write `internal/providers/vertex/vertex_test.go` — test FIRST
- [ ] Implement
- [ ] Commit: `phase-3/task-6: vertex ai provider`

### Pre-conditions

- Task 3.5 complete (or independent)

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
phase-3/task-6: vertex ai provider
```

---

## Task 3.7: AWS Bedrock Provider

### TODO

- [ ] Write `internal/providers/bedrock/bedrock_test.go` — test FIRST
- [ ] Implement with SigV4 signing
- [ ] Commit: `phase-3/task-7: aws bedrock provider with sigv4`

### Pre-conditions

- Task 3.6 complete (or independent)

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
phase-3/task-7: aws bedrock provider with sigv4
```

---

## Task 3.8: Azure OpenAI Provider

### TODO

- [ ] Write `internal/providers/azure/azure_test.go` — test FIRST
- [ ] Implement
- [ ] Commit: `phase-3/task-8: azure openai provider`

### Pre-conditions

- Task 3.7 complete (or independent)

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
phase-3/task-8: azure openai provider
```

---

## Task 3.9: Mistral, Ollama, Cohere, Replicate

### TODO

- [ ] Write tests FIRST for each
- [ ] Implement each
- [ ] Commit: `phase-3/task-9: additional providers`

### Pre-conditions

- Task 3.8 complete (or independent)

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
phase-3/task-9: additional providers
```

---

## Task 3.10: Responses API Support

### TODO

- [ ] Write `internal/providers/openai/responses_test.go` — test FIRST
- [ ] Implement
- [ ] Commit: `phase-3/task-10: openai responses api support`

### Pre-conditions

- Task 3.9 complete (or independent)

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
phase-3/task-10: openai responses api support
```

---

## Phase Gate

```bash
go test ./... -count=1    # ALL tests pass
go vet ./...              # Clean
go build ./cmd/g0router   # Binary builds
```

## Phase Checklist

- [ ] Task 3.1 complete (Anthropic Provider)
- [ ] Task 3.2 complete (Format Translation Engine)
- [ ] Task 3.3 complete (OpenAI-Compatible Providers (batch))
- [ ] Task 3.4 complete (Gemini Provider)
- [ ] Task 3.5 complete (Gemini Format Translation)
- [ ] Task 3.6 complete (Vertex AI Provider)
- [ ] Task 3.7 complete (AWS Bedrock Provider)
- [ ] Task 3.8 complete (Azure OpenAI Provider)
- [ ] Task 3.9 complete (Mistral, Ollama, Cohere, Replicate)
- [ ] Task 3.10 complete (Responses API Support)
- [ ] All tests pass: `go test ./...`
- [ ] Vet clean: `go vet ./...`
- [ ] Build succeeds: `go build ./cmd/g0router`
- [ ] All commits follow `phase-3/task-N: description` format
- [ ] Update `docs/WORKFLOW.md`: phase_3.status → `DONE`
- [ ] **PHASE_3_COMPLETE**
