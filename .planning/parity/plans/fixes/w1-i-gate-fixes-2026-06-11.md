# Fix micro-plan — w1-i diff-gate findings (2026-06-11)

Author: Fable 5. Implementer: kimi. Authorizing artifact:
`artifacts/w1-i-kiro-pair-diff-scoped-gpt.txt`. All four REAL (verified vs ref).

## Task 1 — assistantResponseEvent reads wrong field (BLOCKER)

`kiro_openai_response.go` `handleAssistantResponseEvent` reads
`data["textDelta"]`. The ref (`kiro-to-openai.js:66`) reads
`data.assistantResponseEvent?.content || data.content || ""`. Kiro's
assistantResponseEvent carries `content`, never `textDelta`, so content is
currently dropped. Fix: read `content` from the nested
`data["assistantResponseEvent"]` map first, then fall back to top-level
`data["content"]`; empty → nil (per ref :67).
TDD: extend `TestKiroOpenAIContentAndRoleInjection` to feed
`{assistantResponseEvent:{content:"hi"}}` AND the wrapped/top-level
`{content:"hi", _eventType:"assistantResponseEvent"}` form; both → content delta "hi".

## Task 2 — toolUseEvent must JSON-stringify input always (MAJOR)

`kiro_openai_response.go` tool-use branch special-cases string `input` and
emits it raw. The ref (`kiro-to-openai.js:141`) is
`arguments: JSON.stringify(toolInput)` for ALL input types — a string input
becomes a JSON-quoted string. Fix: remove the `case string` shortcut; always
`json.Marshal(input)` (default `{}`/`map[string]any{}` when input absent, per
ref `toolUse.input || {}` :124). Wrap marshal errors.
TDD: `TestKiroToolUseStringInputStringified` — `input:"raw"` → arguments ==
`"\"raw\""`; `input:{a:1}` → arguments == `{"a":1}`; absent input → `{}`.

## Task 3 — actually exercise the uuid fallback (MAJOR)

`openai_kiro_request_test.go` `TestKiroAssistantToolUses` supplies an id, so the
`toolUseId: tc.id || uuidv4()` fallback (ref `openai-to-kiro.js:372,378`) is
untested. Fix: add a sub-case with a tool_call/​tool_use that has NO id; assert
the emitted `toolUseId` is non-empty (a generated uuid — assert format/length,
not exact value).

## Task 4 — remove test stdout noise (MINOR)

`kiro_constants_test.go:213` uses `fmt.Printf`. Delete it (tests assert, they do
not print). If it was a derivation aid for the sha256, move the derivation to a
code comment.

## Acceptance (binary)

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `grep -c 'textDelta' internal/translation/kiro_openai_response.go` → 0.
- `grep -c 'fmt.Printf' internal/translation/kiro_constants_test.go` → 0.
- The three new/extended tests pass.
- Files touched ONLY: `kiro_openai_response.go`(+test), `openai_kiro_request_test.go`, `kiro_constants_test.go`. Do NOT git commit.

## Out of scope

Any other w1-i behavior (rest is gate-clean).
