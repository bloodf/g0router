# CLAUDE.md

Behavioral guidelines to reduce common LLM coding mistakes. Merge with project-specific instructions as needed.

**Tradeoff:** These guidelines bias toward caution over speed. For trivial tasks, use judgment.

## 1. Think Before Coding

**Don't assume. Don't hide confusion. Surface tradeoffs.**

Before implementing:
- State your assumptions explicitly. If uncertain, ask.
- If multiple interpretations exist, present them - don't pick silently.
- If a simpler approach exists, say so. Push back when warranted.
- If something is unclear, stop. Name what's confusing. Ask.

## 2. Simplicity First

**Minimum code that solves the problem. Nothing speculative.**

- No features beyond what was asked.
- No abstractions for single-use code.
- No "flexibility" or "configurability" that wasn't requested.
- No error handling for impossible scenarios.
- If you write 200 lines and it could be 50, rewrite it.

Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## 3. Surgical Changes

**Touch only what you must. Clean up only your own mess.**

When editing existing code:
- Don't "improve" adjacent code, comments, or formatting.
- Don't refactor things that aren't broken.
- Match existing style, even if you'd do it differently.
- If you notice unrelated dead code, mention it - don't delete it.

When your changes create orphans:
- Remove imports/variables/functions that YOUR changes made unused.
- Don't remove pre-existing dead code unless asked.

The test: Every changed line should trace directly to the user's request.

## 4. Goal-Driven Execution

**Define success criteria. Loop until verified.**

Transform tasks into verifiable goals:
- "Add validation" → "Write tests for invalid inputs, then make them pass"
- "Fix the bug" → "Write a test that reproduces it, then make it pass"
- "Refactor X" → "Ensure tests pass before and after"

For multi-step tasks, state a brief plan:
```
1. [Step] → verify: [check]
2. [Step] → verify: [check]
3. [Step] → verify: [check]
```

Strong success criteria let you loop independently. Weak criteria ("make it work") require constant clarification.

---

**These guidelines are working if:** fewer unnecessary changes in diffs, fewer rewrites due to overcomplication, and clarifying questions come before implementation rather than after mistakes.

---

## Project: g0router

**What**: Go LLM gateway/proxy combining 9router + bifrost + oh-my-pi patterns.
**Single binary**. No plugin architecture. CLI + Web UI control plane.

### Development Rules

1. **TDD always.** Write test first, see it fail, write minimum code to pass.
2. **Every package gets `_test.go` files before implementation.** No exceptions.
3. **`go test ./...` must pass at every commit.** Red tree = blocked.
4. **`go vet ./...` must pass at every commit.**
5. **Match existing patterns.** Read 3 existing files before writing a new one.
6. **No mocks.** Use interfaces and fakes. Test real behavior.
7. **No `init()` functions.** Explicit initialization via constructors.
8. **Errors are values.** Return `error`, don't panic. Wrap with `fmt.Errorf("context: %w", err)`.
9. **No global state.** Pass dependencies via struct fields or function params.
10. **Naming**: `camelCase` locals, `PascalCase` exports. Package names are lowercase singular nouns.

### Workflow Awareness

- Read `docs/README.md` for the doc index, then `docs/WORKFLOW.md` for current task.
- Update `docs/WORKFLOW.md` status after completing any task.
- Run `go test ./...` before and after every change.
- Commit after each green phase. Message: `phase-N/task-M: <description>`.
