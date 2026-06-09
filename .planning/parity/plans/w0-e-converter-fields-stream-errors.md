# w0-e — converter field coverage + stream error propagation (rev 3)

Rows: AUD-032, AUD-033, AUD-034, AUD-036, AUD-037, AUD-046, AUD-047, AUD-081, AUD-085. Behavior contract: PARITY.md §2 Bundle E acceptance column (PARITY.md approved per `.planning/parity/GATE-RESOLUTION.md`).
Runs LAST in Wave 0 — starts only after w0-a AND w0-c merge. Sequencing evidence: `WAVE-MAP.md` Wave 0 row + the time-boxed ownership windows declared in `w0-a-rand-marshal-errors.md` ("File ownership (time-boxed exclusivity)") and `w0-c-converter-data-loss.md`; once those plans merge, their windows close and no other Wave 0 plan lists the files below.
Worker: M3. Reviewer: Kimi diff gate.

## File ownership (exclusive once started)
ALL files under `internal/providers/anthropic/`, `internal/providers/gemini/`, and `internal/providers/openai/chat.go` + `internal/providers/openai/openai_test.go`. Unconditional: since w0-e is the only in-flight plan once w0-a and w0-c merge (WAVE-MAP Wave 0 sequencing), whole-package ownership removes any ambiguity about test files w0-c may or may not have created.

## Field disposition (parity-evidenced — Stage 1 matches 9router)
Gemini evidence: the reference maps only temperature/top_p/top_k/max_tokens (`_refs/9router/open-sse/translator/request/openai-to-gemini.js:48-58`); no candidateCount/penalty/seed mapping exists in that file.
Anthropic evidence (reproducible): `grep -n "presence_penalty\|frequency_penalty\|logit_bias\|seed\|response_format" _refs/9router/open-sse/translator/request/openai-to-claude.js` returns exactly 3 hits, all `response_format` at lines 108-110 (PAR-TRANS-017, Wave 1). No other listed field appears anywhere in the 401-line file.
Richer mappings are NOT ported because the reference does not have them.

| Field | Anthropic | Gemini chat |
|---|---|---|
| N, PresencePenalty, FrequencyPenalty, LogitBias, User, ResponseFormat, Seed | UNSUPPORTED (documented) | UNSUPPORTED (documented) |
| Message.Name | UNSUPPORTED (documented) | UNSUPPORTED (documented) |
| Stream (AUD-036) | n/a | already expressed by method split — add test asserting both paths set it correctly, close the row |

Gemini embeddings (AUD-081): `Dimensions` → `outputDimensionality` — evidence `_refs/9router/open-sse/handlers/embeddingProviders/gemini.js:18-25`. `EncodingFormat`, `User` → UNSUPPORTED (documented; reference does not map them).

UNSUPPORTED mechanism (test-only, no production artifact): per Bundle E acceptance "unit test maps each field or documents unsupported", documentation lives in the TEST file — a `var unsupportedFields = []string{...}` declared in `_test.go` + one test per converter asserting (a) a request populating every listed field serializes to an upstream payload containing none of them, (b) every field named by the AUD rows appears either in a mapping test or this list. Zero production-code surface is added for documentation. ResponseFormat-via-system-prompt is PAR-TRANS-017 (Wave 1) — listed UNSUPPORTED here as interim, superseded by that row.

## Tasks (TDD order)

1. **AUD-032/033/034** Anthropic: disposition tests per table; add `unsupportedFields` + consistency test.
2. **AUD-036/037** Gemini chat: same; plus the Stream-paths test.
3. **AUD-081** Gemini embeddings: `Dimensions` → `outputDimensionality` mapping test + fix; UNSUPPORTED entries for the rest.
4. **AUD-085** Anthropic `ConvertStreamEventToChunk`: test — two `input_json_delta` events produce `Delta.ToolCalls[0].Function.Arguments` fragments that concatenate to the original JSON, and `Delta.Content` stays empty (Bundle E acceptance: "input_json_delta maps to tool-call Delta"). Fix the mapping.
5. **AUD-046** each provider `chat.go`: test — scanner yielding a non-EOF error makes the stream function return that error (Bundle E acceptance: "scanner error is propagated to caller"). Fix: capture `scanner.Err()` after the loop and return it through the error return that w0-a task 6 (AUD-045) adds — a hard dependency this plan inherits by running after w0-a merges. No new signature changes in this plan.
6. **AUD-047** each provider: test — `postHookRunner.Run` returning an error is surfaced via the same error return (Bundle E acceptance: "hook errors are surfaced instead of discarded"). Fix all three call sites.

## Acceptance (binary — one per row)
- AUD-032/033/034/036/037/081: per-converter disposition tests pass and consistency test proves the UNSUPPORTED list matches actual serialization.
- AUD-085: concatenation test passes with empty `Delta.Content`.
- AUD-046: per-provider scanner-error tests pass. AUD-047: per-provider hook-error tests pass.
- `go test ./...` green; `go vet ./...` clean.

## Out of scope
New field mappings absent from 9router (deferred until a PAR row demands them). OpenAI converter (native pass-through). Translation-layer json_schema injection (Wave 1, PAR-TRANS-017). `internal/api` changes. Retry logic.
