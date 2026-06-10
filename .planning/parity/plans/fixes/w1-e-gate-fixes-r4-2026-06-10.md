# Fix micro-plan — w1-d r5 + w1-e r4 + w1-f r6 residual findings (2026-06-10)

Author: Fable 5. Implementer: kimi-for-coding via run-worker.sh.
Authorizing artifact: `artifacts/w1-e-gemini-pair-diff-scoped-gpt.txt` (round 3
verdict; zero BLOCKERs — these three MAJORs are in the original w1-e
implementation).

## Task 1 — test defaultSafetySettings (TDD)

`internal/translation/gemini_helpers_test.go`: add `TestDefaultSafetySettings`
asserting the slice contains exactly five entries whose `category` values are
the five from the ref (`open-sse/translator/request/openai-to-gemini.js` /
`geminiHelper.js` — read the ref definition and cite it in the test comment)
and that every entry has `threshold:"OFF"`. Test first (it should PASS against
correct existing data — if it FAILS, the data diverges from the ref: fix the
data to match the ref, never the test to match the data).

## Task 2 — wrap the ignored Marshal error (TDD)

`internal/translation/gemini_openai_response.go:279` area: `json.Marshal(fcArgs)`
error is discarded. Mirror the repo convention (see
`internal/translation/antigravity_openai_request.go:264-267`): return
`fmt.Errorf("marshal functionCall args: %w", err)` through the enclosing
function's error path (it already returns `([]map[string]any, error)` as a
ResponseTranslator). Add/extend a test only if an existing test seam can reach
the path with a non-marshalable value; otherwise the wrapped return + green
suite satisfies the convention (note this in the report).

## Task 3 — strip padded comments in w1-e-owned files

Remove obvious section comments ("// Convert tools.", "// Convert messages.",
etc.) from `internal/translation/openai_gemini_request.go`,
`gemini_openai_response.go`, `gemini_helpers.go` ONLY. Keep every comment that
carries non-obvious intent: ref citations (file:line), parity rationale,
PAR/PR row references, fallback semantics. No code changes in this task.

## Task 4 — (w1-f-owned, commit separately) cloud_code.go padded comments

Delete obvious control-flow comments in `internal/translation/cloud_code.go`
(e.g. "// Build tool_use id -> name map" at :183); keep intent/ref-citation
comments. No code changes. (w1-f r6 verdict, sole remaining finding.)

## Task 5 — (w1-d-owned, commit separately) registry lookup test on NewRegistry

Context: the pre-w1-d `TestRegistryRegisterLookup` used `NewRegistry()` and
asserted `ResponseTranslatorFor(FormatClaude, FormatOpenAI) == nil` — an
assertion w1-d itself invalidated by wiring `claudeToOpenAIResponse`. The
implementer's hand-built registry preserved mechanics coverage but dropped the
wired-registry override coverage. Restore BOTH:
1. Rename the current hand-built test to `TestRegistryRegisterLookupMechanics`
   (unchanged body).
2. Re-add `TestRegistryRegisterLookup` using `NewRegistry()`: register an
   override request translator for `FormatClaude→FormatOpenAI`, assert the
   override (not the wired translator) is invoked via a `called` flag; assert
   `ResponseTranslatorFor(FormatKiro, FormatOpenAI) == nil` (an unwired pair)
   for the nil-lookup branch.

## Task 6 — (w1-d-owned, same commit as Task 5) padded comments

Delete narrating comments in `internal/translation/openai_claude_request.go`
("// Temperature passthrough.", "// Messages and system extraction.",
"// Tools conversion." and similar obvious section markers). Keep ref
citations, PAR/PR row references, and non-obvious intent comments
(e.g. the PAR-PR-1264 guard comment, the toolNameMap signature comment).

## Acceptance (binary)

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `grep -n 'json.Marshal' internal/translation/gemini_openai_response.go` shows no `_ =` or comma-blank discards.
- `TestDefaultSafetySettings` exists and passes.
- Only the listed files are modified. Do NOT run git commit.

## Out of scope

Any behavior change beyond Task 2's error propagation. Other plans' files.
