# w5-f — Pipeline usage glue (ONLY internal/api editor of the concurrent phase)

PAR rows: PAR-TRANS-046 (usage clause — flips PARTIAL→HAVE), PAR-ROUTE-054, plus the
WIRING halves of PAR-USAGE-003 (detail capture call-sites), PAR-USAGE-012
(saveRequestUsage invoked after provider response), PAR-USAGE-018 (pending start/end
around dispatch). Deferral provenance: PAR-ROUTE-054 deferred from W4
(`WAVE-4-MAP.md` §Stage-1 scope: "054 request-log attribution → Wave 5");
PAR-TRANS-046 usage clause deferred from W1 (`WAVE-MAP.md:55`).
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
   flag + buffer).
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
   STEP (b): in `internal/api`: small consumer interfaces (api imports neither store
   nor usage — w4-e ComboLister precedent): `UsageRecorder`, `PendingTracker`,
   `DetailCapture`; setters on Chat/Messages/Embeddings/Responses handlers; glue:
   Start before dispatch; on completion End + Record (+ Detail) with endpoint =
   route path; on error End(error=true) + Record(status=error) + Detail. Server
   wiring: construct Recorder/Tracker/DetailWriter (w5-b/c) in `internal/server`
   (routes_openai.go bootstrap — produced by w5-pre's signature change) and inject;
   also call `DetailWriter.Close()` on server shutdown (the hook w5-c deferred
   here).

## Preconditions (each states its own pass condition)
- `grep -c 'func (r \*Recorder) Record\|func (.*Recorder) Record' internal/usage/recorder.go` ≥ 1 (w5-b merged).
- `grep -c 'func (w \*DetailWriter) Save\|func (.*DetailWriter) Save' internal/usage/detailwriter.go` ≥ 1 (w5-c merged).
- `grep -rc 'Recorder\|Tracker' internal/api/chat.go` outputs `0` (glue is the gap).
- `grep -c 'ContentLen' internal/translation/stream.go` outputs `0` (046 usage clause gap).

## Exclusive file ownership
NEW: `internal/translation/usage_tracking.go`(+test). TOUCH:
`internal/translation/stream.go`(+test),
`internal/api/{chat,messages,responses,embeddings}.go`(+tests),
`internal/server/{server,routes_openai}.go`(+tests — wiring only). NO
routes_admin.go (w5-d/e), NO store or usage files (consumed via interfaces/calls).

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
