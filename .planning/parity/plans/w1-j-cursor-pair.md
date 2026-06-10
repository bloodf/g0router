# w1-j — cursor translator pair

Rows: **PAR-TRANS-064 (request), PAR-TRANS-065 (response passthrough)** (added 2026-06-10; cross-ref PAR-PROV-023). Registration parity basis: PAR-TRANS-001. Protobuf/checksum utils are EXCLUDED: `cursorProtobuf.js` and `cursorChecksum.js` are consumed only by `executors/cursor.js` (Wave-2 executor scope) — see WAVE-MAP correction 2026-06-10.

Frozen ref (@ 827e5c3), read whole: `open-sse/translator/request/openai-to-cursor.js` (1-184), `open-sse/translator/response/cursor-to-openai.js` (1-31).

## Preconditions (a "0 hits" grep exits 1 — that IS the pass)

- `grep -n 'FormatCursor' internal/translation/formats.go` → present
- `grep -rn 'buildCursorRequest\|cursorToOpenAIResponse' internal/translation/` → 0 hits

## Exclusive file ownership

NEW: `internal/translation/openai_cursor_request.go` + `_test.go`, `cursor_openai_response.go` + `_test.go`.
TOUCH-ONLY: `registry.go` (2 Register calls), `registry_test.go` (wiring test).

## Tasks

1. **Row 064: openai→cursor request** (`openai_cursor_request.go`).
   STEP (a) — write the named tests below; see them fail. STEP (b) — port `openai-to-cursor.js:12-181`:
   - `extractContent` (:12-24 — string passthrough; arrays keep only text blocks, joined with NO separator).
   - `sanitizeToolResultText` (:26-29 — strip control chars U+0000-0008, 000B, 000C, 000E-001F, 007F), `escapeXml` (:31-33 — `&`,`<`,`>` only, in that order), `buildToolResultBlock` (:35-44 — exact 5-line XML shape, tool name default "tool"), `normalizeToolCallId` (:46-48 — first line of id).
   - `convertMessages` (:50-168): meta map from assistant tool_calls AND tool_use blocks, remembering both raw and newline-normalized ids (:53-77); system → user `[System Instructions]\n<content>` (:82-88); tool role → user message with XML block, name precedence `msg.name` → meta → "tool" (:90-100); user array content → text parts + tool_result XML blocks joined "\n", emitted only when non-empty (:103-126); assistant with tool_calls → content default "" + tool_calls with `index` key STRIPPED (:130-136); assistant array content → tool_use blocks extracted to `{id, type:function, function:{name default "tool", arguments:marshal(input)}}` keeping only entries with non-empty id (:137-158); plain messages emitted only when content non-empty (:159-163).
   - `buildCursorRequest` (:170-181): drop `user`/`metadata`/`tool_choice`/`stream_options`/`system` keys, spread the rest, replace messages, `max_tokens` hardcoded 32000.
   Tests: `TestCursorExtractContent`, `TestCursorToolResultBlockXML` (escaping + control-char strip + exact shape), `TestCursorSystemToUserInstructions`, `TestCursorToolRoleNamePrecedence` (msg.name → meta map → "tool"; normalized-id fallback), `TestCursorAssistantToolCallsIndexStripped`, `TestCursorToolUseExtraction` (id-less entries dropped), `TestCursorFieldStrippingAndMaxTokens` (five keys gone, others spread, 32000).

2. **Row 065: cursor→openai response** (`cursor_openai_response.go`).
   STEP (a) — write `TestCursorOpenAIPassthrough` (chunk object, completion object, and unknown map all returned as-is; nil → nil); see it fail. STEP (b) — port `cursor-to-openai.js:13-28` verbatim passthrough (the protobuf→SSE transform lives in the Wave-2 executor; comment citing :3,:10-11).

3. **Registration** — parity basis PAR-TRANS-001. `Register(FormatOpenAI, FormatCursor, buildCursorRequest, nil)` (ref `:183`); `Register(FormatCursor, FormatOpenAI, nil, cursorToOpenAIResponse)` (ref `:30`).
   STEP (a) — `TestNewRegistryWiresCursorPair` (presence + reflect identity) first.

## Binary acceptance criteria

- `go test ./...` exits 0; `go vet ./...` exits 0.
- `grep -rn 'func init(\|panic(' internal/translation/openai_cursor_request.go internal/translation/cursor_openai_response.go` → 0 hits.
- `grep -c '32000' internal/translation/openai_cursor_request.go` ≥ 1.
- `TestCursorToolResultBlockXML` pins the exact `<tool_result>` block shape (covered by go test).

## Out of scope

`cursorProtobuf.js`, `cursorChecksum.js`, `executors/cursor.js` (Wave 2, PAR-PROV-023). kiro (w1-i). gemini-format clients + claude helpers (w1-k).
