# Fix micro-plan 3 — w1-l residual diff-gate findings (2026-06-11)

Author: Fable 5. Implementer: kimi. Dispatch after w1-k fix4 merges.
Authorizing artifact: `artifacts/w1-l-claude-pipeline-helpers-diff-scoped-gpt.txt` (round 3).

## Rebuttal — MAJOR #2 (tool filter) is a FALSE POSITIVE

The finding claims empty-type Claude built-ins (Bash/Read) are "incorrectly
retained" for non-Claude providers. The ref does EXACTLY that: prepareClaudeRequest
filters `tools.filter(tool => !tool.type || tool.type === "function")`
(`claudeHelper.js:189`) — it KEEPS tools with no type OR type==="function", and
drops only typed non-function tools (e.g. `web_search_20250305`). The Go is
byte-faithful: `if toolType == "" || toolType == "function" { keep }`
(`claude_prepare.go:294`). Retaining empty-type tools is the ref's behavior, not
a defect. NO change; rebut in the gate prompt.

## Task 1 — reg==nil must not silently downgrade non-OpenAI source (MAJOR #1, real)

`bypass_handler.go:310` `if reg == nil || sourceFormat == FormatOpenAI { return
[chunk], nil }` collapses two distinct cases. Split them:
- `sourceFormat == FormatOpenAI` → return `[]map[string]any{openaiChunk}, nil`
  (no translation needed — correct).
- `reg == nil` AND `sourceFormat != FormatOpenAI` → return `nil, fmt.Errorf(...)`
  ("bypass: registry required to build %s source-format response", sourceFormat)
  — cannot translate without a registry; do NOT emit an OpenAI body for a
  non-OpenAI client.
TDD: `reg=nil, sourceFormat=FormatClaude` → error (not an OpenAI chunk);
`reg=nil, sourceFormat=FormatOpenAI` → chunk, nil.

## Task 2 — strengthen malformed-content test (MINOR #3, real)

`claude_prepare_test.go:468` only asserts no-panic. Strengthen: feed a message
whose `content` array has a non-map element among valid blocks; assert the result
SKIPS the non-map element (the valid blocks are processed, the bad one is absent
from output), not merely that it didn't panic.

## Acceptance (binary)

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `bypass_handler.go` reg==nil + non-OpenAI source returns an error (covered by go test).
- malformed-content test asserts element skipped (covered by go test).
- Files touched ONLY: `bypass_handler.go`(+test), `claude_prepare_test.go`. Do NOT git commit.

## Out of scope

Tool filter (rebutted — ref-faithful `claudeHelper.js:189`). Everything else (gate-clean).
