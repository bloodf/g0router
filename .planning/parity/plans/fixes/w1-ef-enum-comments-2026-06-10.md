# Fix micro-plan — w1-e enum String() parity + w1-f comment cleanup (2026-06-10)

Author: Fable 5 (planner). Implementer: kimi-for-coding via run-worker.sh.
Authorizing artifacts: w1-e gate verdict (MAJOR, gemini_helpers.go enum
stringification) and w1-f gate verdict (MINOR, padded comments) of 2026-06-10.

## Task 1 — w1-e: enum values must stringify like JS `String()` (TDD)

The ref (`geminiHelper.js:168`) maps enum values with `String(v)`. The Go port
uses `fmt.Sprintf("%v", v)`, which diverges for null and composite values:
JS `String(null)`="null", Go `%v` of nil="<nil>"; JS `String({})`="[object
Object]"; JS `String([1,2])`="1,2".

1. FIRST extend `TestCleanSchemaEnumToStrings` (gemini_helpers_test.go) with
   enum values: `nil` → "null", `map[string]any{"a":1}` → "[object Object]",
   `[]any{float64(1), float64(2)}` → "1,2", `true` → "true", `float64(2)` →
   "2", `float64(2.5)` → "2.5". Run; see it fail.
2. Add unexported helper `jsString(v any) string` in gemini_helpers.go:
   nil → "null"; string → itself; bool → "true"/"false"; float64 →
   `strconv.FormatFloat(v, 'f', -1, 64)`; `[]any` → elements via jsString
   joined with ","  (JS Array.prototype.toString); `map[string]any` →
   "[object Object]"; anything else → `fmt.Sprintf("%v", v)`.
3. Use `jsString` in `convertEnumValuesToStrings`. Tests green.

## Task 2 — w1-f: remove padded obvious comments

In `internal/translation/antigravity_openai_request.go` remove these
line-comments only (no code changes): "// System instruction",
"// Convert contents to messages", "// Tools". Keep comments that state
non-obvious intent (ref citations, fallback semantics).

## Acceptance (binary)

- `go test ./...` and `go vet ./...` green.
- `grep -c 'Sprintf("%v", v)' internal/translation/gemini_helpers.go` → 0 in
  convertEnumValuesToStrings (helper may still use %v as last-resort branch).
- The three padded comments are gone; ref-citation comments remain.

## Out of scope

Any other file. Any behavior change beyond enum stringification.
Do NOT run git commit — the orchestrator commits per plan-id.
