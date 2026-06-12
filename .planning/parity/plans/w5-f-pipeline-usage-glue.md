# w5-f — Pipeline usage glue (ONLY internal/api editor of the concurrent phase)

Ownership authority: WAVE-5-MAP §Ownership tracks (amended 2026-06-12) assigns w5-f
`internal/api/*` PLUS `internal/translation/{usage_tracking.go,stream.go}` (the
PAR-TRANS-046 stream processor lives in translation, not api) and wiring-only
touches to `internal/server/{server,routes_openai}.go` — and records that no
concurrent plan owns any of those files (w5-d/e own internal/admin +
routes_admin.go only).

PAR rows: PAR-TRANS-046 (usage clause — flips PARTIAL→HAVE; row text,
`matrix/9router-translation.md:55`: "Central stream processor … tracks TTFT,
accumulates content/thinking lengths, ESTIMATES USAGE ON FINISH … g0router
`internal/api/chat.go` writes raw chunks; NO ACCUMULATION OR USAGE ESTIMATION" —
the estimation machinery is exactly what Tasks 1-2 port), PAR-ROUTE-054, plus the
HANDLER-WIRING gaps the usage matrix itself names: PAR-USAGE-012 (row NOTE:
"Chat/embed handlers do not extract usage" — that note IS this plan's gap) and the
matrix Go-port directive `matrix/9router-usage.md:124` ("Wire usage extraction into
chat/embeddings handlers after provider response"); PAR-USAGE-018 wiring half
(pending start/end around dispatch — tracker built in w5-b, call-sites here);
PAR-USAGE-026 production call-sites (the buffered writer built in w5-c is dead code
until the handlers call Save — `chatCore.js:196,242` are those call-sites in the
ref). Deferral provenance: PAR-ROUTE-054 deferred from W4 (`WAVE-4-MAP.md` §Stage-1
scope: "054 request-log attribution → Wave 5"); PAR-TRANS-046 usage clause deferred
from W1 (`WAVE-MAP.md:55`).
NOT in scope: any admin route (w5-d/e), store/domain write logic (w5-a/b/c — consumed
here, not modified), virtual keys (w5-g).
Frozen ref @ 827e5c3. Depends: w5-b + w5-c merged. Runs ∥ w5-d/e (api vs admin files).

## Tasks

1. **Usage tracking helpers (translation layer)** — evidence:
   `open-sse/utils/usageTracking.js`: `normalizeUsage` (`:115-150` — numeric
   coercion of 7 token fields + preserved details objects; empty → nil),
   `hasValidUsage` (`:150-170` — any of prompt/completion/total/input/output/
   promptTokenCount/candidatesTokenCount > 0), `extractUsage` (`:172-238` — Claude
   message_delta / OpenAI chunk.usage / Gemini usageMetadata / Responses
   response.usage shapes), `addBufferToUsage` (`:19,31-55` — BUFFER_TOKENS=2000 added
   to input_tokens AND/OR prompt_tokens; total recomputed), `filterUsageForFormat`
   (`:57-113`), `estimateInputTokens` (`:240-255` — ceil(len(JSON(body))/4)),
   `estimateOutputTokens` (`:259-262` — max(1, floor(contentLength/4))),
   `formatUsage`/`estimateUsage` (`:270-305` — claude vs openai shape + estimated
   flag + buffer). EVERY helper is demanded by the PAR-TRANS-046 finish-clause
   call-sites in `stream.js` (none is speculative): `hasValidUsage` at :151/:292,
   `estimateUsage` at :152/:293/:330, `filterUsageForFormat` at :153/:158/:294/:299,
   `addBufferToUsage` at :158/:298, `extractUsage` at :147/:255, `normalizeUsage` is
   extractUsage's output normalizer (usageTracking.js:172+ delegates),
   `formatUsage` is estimateUsage's body (`:295-305`), `estimateInputTokens`/
   `estimateOutputTokens` are formatUsage's inputs.
   STEP (a): table-driven `TestNormalizeUsage`, `TestHasValidUsage`,
   `TestExtractUsageFormats` (one fixture per source format), `TestAddBufferToUsage`
   (2000 + total recompute), `TestEstimateUsage` (golden: body of known JSON length →
   ceil/4; contentLength 10 → 2 output tokens; claude vs openai shape; estimated
   flag) — run — fail.
   STEP (b): NEW `internal/translation/usage_tracking.go` porting the eight helpers
   over `map[string]any` usage payloads (format-aware translation concern; same
   package as the stream processor that consumes them).

2. **Stream usage accumulation + estimate-on-finish (PAR-TRANS-046 usage clause)** —
   evidence: `open-sse/utils/stream.js:147-160` (TRANSLATE finish chunk: no valid
   usage → inject `estimateUsage(body, totalContentLength)` filtered for the client
   format; valid usage → `addBufferToUsage` + filter; keep original in state for
   logging) and `:290-335` (same for PASSTHROUGH + final summary usage for the
   logging callback); in-tree `internal/translation/stream.go:18-46` already tracks
   TTFT in `StreamSummary` (w1-c) but accumulates no content length and extracts no
   usage — the PARTIAL gap named by the matrix row.
   STEP (a): `TestStreamAccumulatesUsage` (chunks carrying usage → summary.Usage =
   extracted, client-bound finish chunk gets buffered+filtered usage),
   `TestStreamEstimatesOnFinish` (no usage in any chunk, content present → finish
   chunk + summary carry estimated usage with `estimated` flag),
   `TestPassthroughSummaryUsage` (same for ProcessPassthroughStream) — run — fail.
   STEP (b): extend `StreamSummary` with `ContentLen int` + `Usage map[string]any`;
   thread the request body (for input estimation) into `ProcessTranslateStream`/
   `ProcessPassthroughStream` via a new optional `EstimateSource` param (body bytes +
   client format); apply the ref's finish-chunk logic.

3. **Handler glue: pending + record + details (PAR-ROUTE-054 + wiring halves)** —
   evidence: `open-sse/handlers/chatCore.js:135-140` (trackPendingRequest(start)
   before executor + appendRequestLog PENDING), `:196,242` (saveRequestDetail on
   error and non-stream success), `open-sse/handlers/chatCore/requestDetail.js:75-103`
   (`saveUsageStats`: skip zero-token; normalize to prompt/completion; calls
   saveRequestUsage with provider/model/tokens/connectionId/apiKey/endpoint),
   `open-sse/utils/stream.js:329-340` (stream finish → estimate-if-needed →
   saveUsageStats path), `open-sse/handlers/chatCore/streamingHandler.js:54,87` +
   `nonStreamingHandler.js:207` + `sseToJsonHandler.js:128,202` (detail capture on
   every outcome path). PAR-ROUTE-054's attribution columns
   (model/provider/connection) land in the same `request_log` row (AGENTS.md:27;
   `chatCore.js:79-82` request logger attribution).
   STEP (a): `TestChatRecordsUsageNonStream` (fake Recorder/Tracker injected via
   setters; POST chat → tracker Start+End called, Recorder.Record got
   provider/model/connection/endpoint="/v1/chat/completions" + tokens from the
   provider response), `TestChatRecordsUsageStream` (stream finishes → Record with
   accumulated/estimated usage), `TestChatRecordsErrorStatus` (provider error →
   tracker End(error=true), entry status != "ok"), `TestChatCapturesRequestDetail`
   (fake DetailWriter receives sanitized capture on success AND error paths),
   `TestMessagesRecordsUsage`, `TestEmbeddingsRecordsUsage` (endpoint attribution
   varies per route) — run — fail.
   STEP (b): in `internal/api`: small consumer interfaces — NOT a stylistic choice:
   `AGENTS.md:24` ("Layered DDD architecture (transport→domain→repository) enforced
   by arch test") forbids the transport layer importing domain/repository packages
   directly, so an api-side interface + setter is the ONLY compliant way to hand the
   handlers a Recorder/Tracker/Writer (the mechanism every prior wave used:
   `internal/api/models.go:17-19` ComboLister/DisabledChecker (w4-c/e),
   `internal/api/chat.go` CredentialRefresher + ComboDispatcher (w4-f/w5-pre)):
   `UsageRecorder`, `PendingTracker`,
   `DetailCapture`; setters on Chat/Messages/Embeddings/Responses handlers; glue:
   Start before dispatch; on completion End + Record (+ Detail) with endpoint =
   route path; on error End(error=true) + Record(status=error) + Detail. Server
   wiring: construct Recorder/Tracker/DetailWriter (w5-b/c) in `internal/server` and
   inject through `RegisterOpenAIRoutes` — whose dependency-injection signature
   ALREADY exists post-w5-pre (`internal/server/routes_openai.go:11` now takes
   `refresher api.CredentialRefresher, comboDisp api.ComboDispatcher`; this plan
   extends the same pattern). Also call `DetailWriter.Close()` on server shutdown —
   REF EVIDENCE `requestDetailsRepo.js:183-200` (`_shutdownHandler` flushes the
   buffer on beforeExit/SIGINT/SIGTERM/exit — the shutdown flush is the ref's own
   PAR-USAGE-026 behavior, not an invention); the call-SITE placement here is the
   recorded w5-c deferral (`w5-c-observability.md` §Task 4: "explicit `Close()`
   called from server shutdown is OUT of this plan's files" and §Out of scope: "the
   server Close hook ships with w5-f's glue").

## Preconditions (each states its own pass condition)
- `grep -c 'func (r \*Recorder) Record\|func (.*Recorder) Record' internal/usage/recorder.go` ≥ 1 (w5-b merged).
- `grep -c 'func (w \*DetailWriter) Save\|func (.*DetailWriter) Save' internal/usage/detailwriter.go` ≥ 1 (w5-c merged).
- `grep -rc 'Recorder\|Tracker' internal/api/chat.go` outputs `0` (glue is the gap).
- `grep -c 'ContentLen' internal/translation/stream.go` outputs `0` (046 usage clause gap).

## Exclusive file ownership
NEW: `internal/translation/usage_tracking.go`(+test). TOUCH:
`internal/translation/stream.go`(+test),
`internal/api/{chat,messages,responses,embeddings}.go`(+tests),
`internal/server/{server,routes_openai}.go`(+tests — wiring only). NO concurrency
conflict: the plans running in the same phase touch DISJOINT files — w5-d's
ownership is `internal/admin/* + internal/server/routes_admin.go + store read
queries` and w5-e's is `internal/admin/* + routes_admin.go` (their §Exclusive file
ownership sections); NEITHER touches server.go/routes_openai.go/internal/api. NO
routes_admin.go here, NO store or usage files (consumed via interfaces/calls).

## Binary acceptance
- `go build ./... && go vet ./...` clean; `go test ./...` green; `go test -race ./internal/api/ ./internal/translation/ ./internal/server/` green.
- `grep -c 'ContentLen' internal/translation/stream.go` ≥ 1.
- `grep -rc 'bloodf/g0router/internal/store\|bloodf/g0router/internal/usage' internal/api/chat.go` → `:0` (api layering held).
- End-to-end smoke (test or script): one fake-provider chat request → `SELECT COUNT(*) FROM request_log` = 1 with non-empty provider/model attribution (PAR-ROUTE-054 binding check).
- TestStreamEstimatesOnFinish, TestChatRecordsUsageNonStream, TestChatRecordsUsageStream, TestChatRecordsErrorStatus, TestChatCapturesRequestDetail, TestMessagesRecordsUsage, TestEmbeddingsRecordsUsage all pass.

## Out of scope
Admin routes (w5-d/e). Virtual keys + per-key quota counting (w5-g). RTK/caveman
token savers (Stage-2 features adjacent in chatCore). Combo-path usage attribution
beyond what flows naturally through the wired Recorder (the combo branch dispatches
through the same handlers — no special-casing in this plan).

## Plan-gate disposition (cycle 3, Fable 5, 2026-06-12) — CLOSED BY DECISION
Three substantive cycles complete. Cycle-1/2 findings FIXED (046 row text anchored to
translation matrix; helper-by-helper mapping to stream.js call-sites; Close() tied to
ref evidence requestDetailsRepo.js:183-200; ownership resolved in amended WAVE-5-MAP;
seam mandated by AGENTS.md:24). Cycle-3 residual triage:
- BLOCKER "messages.go glue not row-backed" + BLOCKER "responses.go not row-backed":
  FALSE POSITIVE. (a) PAR-USAGE-016 requires the byEndpoint breakdown
  (`endpoint|model|provider` keys) — an attribution dimension that is structurally
  empty if only one endpoint records usage; (b) PAR-ROUTE-054's evidence
  (`chatCore.js:79-82,135-140`) is the SHARED core that serves /v1/chat/completions,
  /v1/messages AND /v1/responses in the ref (every endpoint handler delegates to
  chatCore — usage recording is endpoint-agnostic there); the per-endpoint setters
  here reproduce what the ref gets for free from its shared core. (c) The usage
  matrix's own Go-port directive (9router-usage.md:124) says "chat/embeddings
  handlers" — embeddings is named explicitly, refuting the chat-only reading.
- MAJOR "Close() wiring beyond rows": rebutted in-plan with ref evidence
  requestDetailsRepo.js:183-200 (shutdown flush IS PAR-USAGE-026 behavior).
- MAJOR "EstimateSource signature change without caller migration": callers of
  ProcessTranslateStream/ProcessPassthroughStream live in internal/api handlers —
  ALL inside this plan's ownership (verified: `grep -rln 'ProcessTranslateStream\|
  ProcessPassthroughStream' internal/ --include='*.go' | grep -v _test` →
  internal/api only + translation itself). No unowned caller exists.
APPROVED BY DECISION for dispatch after w5-b + w5-c merge.

## Diff-gate disposition (cycle 3, both halves, Fable 5, 2026-06-12) — CLOSED BY DECISION
Three substantive cycles per half. Cycle-1 REAL FIXED (fix-r1, 4b337b2: production
shutdown wiring via NewWithShutdown + SIGINT/SIGTERM in main.go, persistence-
asserting smoke tests, error-path + sanitized detail capture, deduped record glue).
Cycle-2 REAL FIXED (fix-r2, feb76c3: passthrough valid-usage test, isArrayish stub
deleted, Claude token synonyms in glue, logged-not-discarded write failures).
Cycle-3 residual triage:
- A-BLOCKER "NormalizeUsage omits input_tokens/output_tokens/promptTokenCount":
  FALSE POSITIVE — ref-verbatim: normalizeUsage (`usageTracking.js:115-148`)
  assigns ONLY the OpenAI-shaped keys because extractUsage MAPS claude/gemini/
  responses shapes INTO prompt_tokens/completion_tokens BEFORE normalizing
  (`:177-180` claude, `:189-196` responses, `:216-221` gemini). The Go port
  mirrors both halves of that pipeline.
- A-MAJOR buffer-total increment: THIRD occurrence of a rebutted finding (ref
  `:46-51`). A-MAJOR FormatUsage default-OpenAI + filter: SECOND occurrence (ref
  `:283` comment; estimated-only payloads for gemini/responses clients are the
  ref's own behavior). Gate non-convergence documented.
- B-BLOCKER cmd/main.go scope: SECOND occurrence — fix-r1's explicit OWNERSHIP
  GRANT closed the gate's OWN cycle-1 blocker (shutdown flush unreachable).
- B-MAJOR "TestServerWiresUsageGlue only checks table": REBUTTED BY SIBLINGS — the
  fix-r1-strengthened smokes (chat/messages/embeddings PersistsRequestLog) prove
  the full wiring end-to-end (HTTP request → request_log row with attribution);
  the named test is a redundant precondition check, not the wiring proof.
- B-MAJOR "APIKey attribution never populated": REAL → TRANSFERRED TO w5-g
  (recorded follow-up): the machine/virtual key value enters the request at the
  guard/VK layer; w5-g's per-key quota engine is the CONSUMER of request_log
  api_key attribution (its SumCostByAPIKey reads it), so the plumbing (guard/VK
  context → UsageEntry.APIKey) lands with w5-g and is checked at w5-g's diff gate.
  Until then byApiKey stats attribute to local-no-key — matching the ref's
  behavior for keyless local traffic.
Full gates green (build/vet/test/-race verified live). MERGED.
Rows flip: PAR-TRANS-046 PARTIAL→HAVE; PAR-ROUTE-054 MISSING→HAVE.
