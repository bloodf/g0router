# Closeout fix micro-plan — w1-d r6 / w1-e r5 / w1-f r7 remaining items (2026-06-10)

Author: Fable 5. Implementer: kimi-for-coding. This is the FINAL batch for the
w1-c..f gate loops (see ESCALATION note in WORKFLOW.md): it fixes every
remaining finding with substance; pure style nibbles are swept once here.

## Task 1 (w1-d) — raw image block test

`openai_claude_request_test.go`: extend `TestOpenAIClaudeImageBlocks` with a
part `{type:"image", source:{type:"base64", media_type:"image/png", data:"abc"}}`
asserting passthrough to a claude `image` block with the same source
(ref openai-to-claude.js:248-250 raw image passthrough).

## Task 2 (w1-d) — assert buffered tool arguments

`openai_claude_response_test.go`: in `TestClaudeOpenAIToolUseStartAndArgs`,
after the two `input_json_delta` events assert
`state.ClaudeBlockTools[1].Arguments == "{\"a\":1"` (the accumulated buffer) —
verify first whether the ref consumes its accumulated arguments at finish; if
the ref only accumulates (state bookkeeping), the test documents the buffer as
the row-042 contract; note which in the report.

## Task 3 (w1-e) — marshal-path test + helper direct tests

1. `gemini_openai_response_test.go`: test the wrapped marshal error using a
   non-marshalable functionCall args value (e.g. inject `math.Inf(1)` /
   a channel via the map seam if reachable; if the decoded-JSON input space
   cannot produce non-marshalable values, write the test proving the happy
   path returns nil error and state why the error branch is defensive —
   in a test comment, not prose).
2. `gemini_helpers_test.go`: add `TestExtractTextContent` (string parts,
   text-block arrays, empty) and `TestTryParseJSON` (valid object, invalid →
   raw string passthrough) — direct helper coverage the micro-plan listed.

## Task 4 (w1-f) — vertex signature literal + plan-owned test file

1. `openai_vertex_request_test.go`: assert `thoughtSignature` equals the
   VERBATIM literal from the frozen ref `defaultThinkingSignature.js` /
   vertex signature constant source (copy the string into the test, with the
   ref file:line cited in a comment) — never compare against the production
   constant.
2. Create `internal/translation/openai_claude_antigravity_test.go` and MOVE the
   claude-antigravity tests currently folded into other test files into it
   (pure move, no behavior change), restoring the plan's declared file
   ownership.

## Task 5 (all) — single comment sweep

Across `internal/translation/` files owned by w1-d/e/f (per diff-scopes.json
paths): delete comments that narrate obvious control flow ("// Text content",
"// user \"hello\"", "// Normalize tools", "// Skip empty non-finish chunks",
etc.). KEEP: ref citations (file:line), PAR/PR row references, JS-semantics
rationale (truthiness, iteration order), fallback/error-path intent. When in
doubt, keep the comment.

## Acceptance (binary)

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `ls internal/translation/openai_claude_antigravity_test.go` exists.
- `grep -c 'thoughtSignature' internal/translation/openai_vertex_request_test.go` ≥ 1 with a string literal (not `defaultThinkingVertexSignature`) as the expected value.
- Do NOT run git commit.
