# Fix micro-plan 2 — w1-l residual diff-gate findings (2026-06-11)

Author: Fable 5. Implementer: kimi. Dispatch after w1-k fix3 merges.
Authorizing artifact: `artifacts/w1-l-claude-pipeline-helpers-diff-scoped-gpt.txt` (round 2).

## Rebuttal — MAJOR #1 (PrepareClaudeRequest doesn't call cloakClaudeTools) is a FALSE POSITIVE

The ref calls `cloakClaudeTools` in `translator/index.js:129` (the request
PIPELINE), NOT inside `prepareClaudeRequest` (`claudeHelper.js` has zero
`cloakClaudeTools` references — verified). So the Go `PrepareClaudeRequest`
correctly does NOT call it. `cloakClaudeTools` is the exported, unit-tested
helper (PAR-TRANS-022) whose caller is the request pipeline (Wave-2/4 routing),
exactly like the other w1-l pipeline helpers. No code change; rebut in the gate prompt.

## Task 1 — naming bypass must JSON-escape the title (MAJOR #3, real)

`bypass_handler.go:252` builds `{"isNewTopic":true,"title":"%s"}` via
`fmt.Sprintf` with the raw user-derived title — quotes/backslashes/newlines
produce invalid JSON. The ref uses `JSON.stringify({isNewTopic:true, title})`
(`bypassHandler.js`). Fix: build `map[string]any{"isNewTopic":true,"title":title}`
and `json.Marshal` it (wrap the error). TDD: title containing `"` and `\`
produces valid JSON that round-trips via `json.Unmarshal`.

## Task 2 — bypass must not silently downgrade source-format on translation error (MAJOR #2, real)

`bypass_handler.go:299-301`: when `reg.TranslateResponse(OPENAI, sourceFormat,...)`
errors, the code returns the OpenAI chunk — silently giving a Claude client an
OpenAI-shaped body, the exact divergence the source-format fix was for. Fix:
propagate the error — change the build helpers to return `([]map[string]any, error)`
and have `HandleBypassRequest` return that error to its caller (the handler
decides; do not mask). When `sourceFormat == FormatOpenAI`, no translation, no error.
TDD: a stubbed registry returning an error → the build func returns that error
(not an OpenAI-shaped chunk).

## Task 3 — checked type assertions in claude_prepare.go (MAJOR #4, real)

`claude_prepare.go:223` (`filtered[i].(map[string]any)`) and `:234`
(`content[j].(map[string]any)`) are unchecked and panic on malformed blocks,
violating the no-panic convention. Fix: comma-ok both (skip the element on
`!ok`). Scan the file for any other unchecked `.(map[string]any)`/`.([]any)` in
the message/loops and make them comma-ok too. TDD: a message with a non-map
content element and a non-map block element → no panic, element skipped.

## Acceptance (binary)

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `grep -c 'fmt.Sprintf(`{"isNewTopic"' internal/translation/bypass_handler.go` → 0 (uses json.Marshal).
- `grep -nE 'filtered\[i\]\.\(map\[string\]any\)|content\[j\]\.\(map\[string\]any\)' internal/translation/claude_prepare.go` → 0 unchecked (all comma-ok).
- The three TDD cases pass.
- Files touched ONLY: `bypass_handler.go`(+test), `claude_prepare.go`(+test). Do NOT git commit.

## Out of scope

cloakClaudeTools wiring (rebutted — index.js:129 pipeline, Wave-2/4). cloaking
and prepare core behavior (gate-clean).
