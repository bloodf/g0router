# Fix micro-plan — w1-l diff-gate findings (2026-06-11)

Author: Fable 5. Implementer: kimi. Dispatch after w1-k merges (no file overlap;
serialize kimi). Authorizing artifact:
`artifacts/w1-l-claude-pipeline-helpers-diff-scoped-gpt.txt`.

## Rebuttal — BLOCKERs #1 & #2 (ccDefaultTools count) are PLAN ERRORS, not code defects

The plan said `ccDefaultTools` has 26 entries and acceptance pinned `==26`. That
was a planning miscount — the frozen ref `CC_DEFAULT_TOOLS`
(`appConstants.js:79-106`) has EXACTLY 20 members (verified by extracting the
Set body). Kimi implemented 20 and set `TestCCDefaultToolsCount` to `==20`,
which is CORRECT per the ref. The plan has been corrected (26→20). NO code or
test change — the implementation already matches the frozen ref. Do not re-flag.

## Task 1 — bypass must build SOURCE-FORMAT responses (MAJOR #3 + #4, real)

`bypass_handler.go` leaves `detectBypassSourceFormat` unused and always emits an
OpenAI-chat-shaped bypass response. The ref `createStreamingResponse`/
`createNonStreamingResponse` (`bypassHandler.js:182-195` and the non-streaming
sibling) build the response in the DETECTED source format: create an OpenAI
response, then `translateResponse(OPENAI, sourceFormat, chunk, state)` per chunk.
A claude-cli client hitting `/v1/messages` (claude source) must get a
claude-shaped bypass response, not OpenAI.
STEP (a) tests first, STEP (b):
- Wire `detectBypassSourceFormat(body)` (ref `services/provider.js:49` body-shape
  detector) to determine the source format.
- `createStreamingResponse(reg *Registry, sourceFormat Format, model, text string) []map[string]any` and `createNonStreamingResponse(...)`: build the OpenAI
  response/chunks, then translate each to `sourceFormat` via
  `reg.TranslateResponse(FormatOpenAI, sourceFormat, chunk, state)` (the registry
  is already the engine; pass it in — `HandleBypassRequest` gains a `*Registry`
  param, or the bypass funcs take one). When `sourceFormat == FormatOpenAI`, no
  translation (passthrough).
- Keep the 5 bypass-trigger patterns and naming synthesis unchanged.
Tests: `TestBypassClaudeSourceFormatResponse` (claude-shaped body/`/v1/messages`
detection → claude-shaped bypass chunks, e.g. `type:message_start`...),
`TestBypassOpenAISourceFormatResponse` (openai → openai chunks, no translation),
`TestBypassNamingSourceFormat` (naming bypass also source-format-shaped).

## Acceptance (binary)

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `grep -c 'detectBypassSourceFormat' internal/translation/bypass_handler.go` ≥ 2 (defined AND used).
- `TestBypassClaudeSourceFormatResponse` asserts a non-OpenAI (claude) chunk shape (covered by go test).
- `TestCCDefaultToolsCount` stays `==20` (already correct — do NOT change).
- Files touched ONLY: `bypass_handler.go`(+test). Do NOT git commit.

## Out of scope

ccDefaultTools count (rebutted — plan corrected, code already right). cloaking
and prepareClaudeRequest (gate-clean). Wiring bypass into the live request
handler (Wave-2/4 routing — the helper + tests ship here per row 054).
