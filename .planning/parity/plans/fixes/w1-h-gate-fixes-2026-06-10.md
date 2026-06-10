# Fix micro-plan — w1-h diff-gate findings (2026-06-10)

Author: Fable 5. Implementer: kimi. Dispatch after w1-i merges (shares no files
with w1-i, but serialize kimi). Authorizing artifact:
`artifacts/w1-h-ollama-commandcode-diff-scoped-gpt.txt`. All three REAL.

## Task 1 — ollama dataURIPattern: drop global + MustCompile (BLOCKER)

`openai_ollama_request.go:250` — package-level `var dataURIPattern = regexp.MustCompile(...)`
violates no-global-state + no-panic. Fix per the w1-d precedent
(`openai_claude_request.go` image branch): compile locally with
`regexp.Compile` inside the extract function, ignore the error (static pattern)
or fall through to no-match. No package-level var.
TDD: existing image tests must still pass; add `TestOllamaImageExtractionNoGlobalState` only if a behavior assertion is missing.

## Task 2 — ollama tool_call index reads the wrong field (MAJOR, faithful fix ≠ critic suggestion)

`openai_ollama_request.go:155` reads `fn["index"]` (function.index). The ref
(`openai-to-ollama.js:105`) reads `tc.index || 0` — the TOP-LEVEL tool_call
index, not `function.index`. Fix: read `tc["index"]` (the tool_call map),
default 0, and write it to the ollama `function.index` output.
REBUTTAL of the critic's suggested fix: "preserve the call position/index"
(i.e. use the array loop index `i`) would DIVERGE from the ref — 9router uses
`tc.index || 0`, which is 0 for non-streaming history tool calls. Match the ref,
do not invent positional indices.
TDD: `TestOllamaToolCallIndexFromToolCall` — a tool_call with top-level
`index: 2` → ollama `function.index == 2`; a tool_call with no index → 0;
a tool_call whose `function` carries an `index` key is IGNORED (only `tc.index`
counts).

## Task 3 — CommandCode usage omitted when absent (MAJOR)

`registry.go:113` and `commandcode_openai_response.go:271` initialize
`CommandCodeUsage` to a non-nil empty map, so the `finish` branch
(`commandcode_openai_response.go:193`) sees a non-nil `totalUsage` and emits a
zero-token usage object. The ref (`commandcode-to-openai.js:169-176`):
`const totalUsage = event.totalUsage || state.usage; if (totalUsage) {...}` —
omits usage entirely when both are falsy. Fix: do NOT pre-initialize
`CommandCodeUsage` (remove it from `NewStreamState` init and the :271 reset, or
set to nil); in the finish branch attach usage only when `totalUsage != nil`
AND non-empty (`len > 0`). A `finish-step` with usage still populates it.
TDD: `TestCommandCodeFinishWithoutUsageOmitsUsage` — `finish` event, no prior
`finish-step` usage, no `totalUsage` → final chunk has NO `usage` key;
`TestCommandCodeFinishWithStepUsage` — `finish-step` usage then `finish` →
usage present with those tokens.

## Acceptance (binary)

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `grep -rn 'regexp.MustCompile\|var dataURIPattern' internal/translation/openai_ollama_request.go` → 0 hits.
- `grep -c 'CommandCodeUsage:' internal/translation/registry.go` → 0 (not pre-initialized).
- The three new tests pass.
- Files touched ONLY: `openai_ollama_request.go`(+test), `commandcode_openai_response.go`(+test), `registry.go`. Do NOT git commit.

## Out of scope

Any other w1-h behavior (rest of the pair is gate-clean). The critic's
positional-index suggestion (rebutted — non-parity).
