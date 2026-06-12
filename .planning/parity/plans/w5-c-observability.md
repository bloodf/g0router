# w5-c — Observability writer: buffered request details + sanitization + debug gate

PAR rows: PAR-USAGE-003 (write semantics — table landed in w5-a), PAR-USAGE-024,
PAR-USAGE-025, PAR-USAGE-026, PAR-USAGE-027, PAR-USAGE-028, PAR-AUTH-017,
PAR-AUTH-018. NOT in scope: the `/api/usage/request-details` HTTP route (w5-d serves
it over this plan's query), usage stats/charts (w5-d), handler capture call-sites
(w5-f), SSE (w5-e).
Frozen ref @ 827e5c3. Depends: w5-a merged. Runs ∥ w5-b (disjoint files).

Deferral provenance: PAR-AUTH-017/018 were deferred from Wave 3 to Wave 5 in
`WAVE-3-MAP.md` §Deferred ("Port `sanitizeHeaders` (`requestDetailsRepo.js:46-54`)
with the Wave-5 logging plan"; "the prod gate (`debugLog.js:3`) ports together with
the utility").

## Tasks

1. **Observability config (PAR-USAGE-025)** — evidence:
   `src/lib/db/repos/requestDetailsRepo.js:4-40`: defaults maxRecords=200,
   batchSize=20, flushIntervalMs=5000, maxJsonSize=5KB; enabled = settings
   `enableObservability` bool if present else env `OBSERVABILITY_ENABLED != "false"`;
   each field overridable via settings then env; config cached 5s.
   STEP (a): table-driven `TestObservabilityConfig` (defaults; settings override; env
   override; enabled precedence settings-bool > env; 5s cache via injected clock) — fails.
   STEP (b): NEW `internal/usage/observability.go`: `ObsConfig` struct +
   `ObsConfigLoader{settings SettingsReader, getenv func(string) string, clock}` with
   `Load() ObsConfig`. SettingsReader is a small interface (store satisfies it).

2. **Header sanitizer (PAR-USAGE-027 = PAR-AUTH-017)** — evidence:
   `requestDetailsRepo.js:46-54`: case-insensitive SUBSTRING match against
   [authorization, x-api-key, cookie, token, api-key]; matching keys DELETED (not
   masked); non-map input → empty map.
   STEP (a): table-driven `TestSanitizeHeaders` (Authorization, X-Api-Key, Cookie,
   X-Auth-Token, api-key variants removed; Content-Type kept; substring semantics:
   `x-csrf-token` removed because it contains "token"; nil → empty) — fails.
   STEP (b): `SanitizeHeaders(map[string]string) map[string]string` in
   `internal/usage/observability.go` (pure).

3. **JSON truncation** (quirk under PAR-USAGE-026) — evidence:
   `requestDetailsRepo.js:63-69`: serialized field > maxJsonSize → replaced by
   `{_truncated: true, _originalSize: N, _preview: first-200-chars}`.
   STEP (a): `TestTruncateField` (small passes through; oversized replaced with the
   three marker fields, preview exactly 200 chars) — fails.
   STEP (b): `TruncateField(v any, maxSize int) any` (pure, same file).

4. **Buffered writer with batch flush + retention (PAR-USAGE-026, PAR-USAGE-028,
   PAR-USAGE-003)** — evidence: `requestDetailsRepo.js:42-142,183-200`: in-memory
   buffer; flush when len ≥ batchSize (immediate, cancels timer) else timer at
   flushIntervalMs; flush drains the whole buffer in a loop, single-flight
   (isFlushing); per-item: id generated `timestamp-random6-modelSlug`
   (`:56-61`), timestamp defaulted, request headers sanitized, latency/tokens/
   request/providerRequest/providerResponse/response truncated and stored as one
   JSON `data` blob, INSERT OR UPDATE by id; retention: when COUNT > maxRecords
   delete oldest-by-timestamp overflow IN the same tx (`:109-115`); shutdown flush
   drains remainder (`:183-186` — Go port: explicit `Close()` called from server
   shutdown is OUT of this plan's files; expose Close and test it directly).
   Disabled config → drop writes (`:126-127`).
   STEP (a): `TestWriterFlushAtBatchSize` (batchSize=2: first write buffers, second
   triggers flush → both rows in request_details), `TestWriterTimerFlush` (injected
   timer fires → row persisted), `TestWriterRetention` (maxRecords=3, insert 5 →
   COUNT=3, oldest gone), `TestWriterDisabledDrops` (enabled=false → no rows),
   `TestWriterCloseFlushes` (buffered item + Close → persisted),
   `TestWriterConcurrent` (parallel saves under -race, single-flight flush) — fail.
   STEP (b): `internal/usage/detailwriter.go`: `DetailWriter{store DetailStore,
   config loader, mu, buffer, timerFactory, randRead seam}` with `Save(detail
   RequestDetail)` + `Close()`. Store side NEW `internal/store/requestdetails.go`:
   `(s *Store) SaveRequestDetails(items []*RequestDetailRow) error` (one tx: upserts
   + retention delete given maxRecords param) — keep store a leaf; sanitize/truncate
   happen in the domain writer before rows reach the store.

5. **Filtered + paginated query (PAR-USAGE-024)** — evidence:
   `requestDetailsRepo.js:144-181`: filters provider/model/connectionId/status/
   startDate/endDate (ISO string comparison); COUNT for pagination; page/pageSize
   (default 1/50) → details DESC by timestamp + pagination
   {page,pageSize,totalItems,totalPages,hasNext,hasPrev}; get-by-id returns the
   decoded data blob or nil.
   STEP (a): `TestRequestDetailsQuery` (seed 6 rows across 2 providers/status; filter
   combos; page 2 of pageSize 2 → correct slice + pagination math) and
   `TestRequestDetailByID` — fail.
   STEP (b): in `internal/store/requestdetails.go`: `RequestDetailsFilter` struct,
   `(s *Store) QueryRequestDetails(f) (rows, Pagination, error)`,
   `(s *Store) GetRequestDetailByID(id string) ([]byte, error)`.

6. **Debug log production gate (PAR-AUTH-018)** — evidence: `debugLog.js:1-15`:
   `isDev = NODE_ENV !== "production"`; `dbg(tag,msg)` no-ops in production; output
   format `[HH:MM:SS] 🐛 [DBG:tag] msg`. Go adaptation (recorded decision): gate on
   env `G0ROUTER_ENV != "production"` (no NODE_ENV in Go), constructor-injected —
   no init(), no global mutable state; package `internal/logging` (currently a
   placeholder — `internal/logging/doc.go` says request-log/audit arrive later;
   this plan adds ONLY the debug gate, not the audit trail).
   STEP (a): `TestDebugLogProductionGate` (production env → writer receives nothing;
   dev → tagged line written; format contains `[DBG:tag]`) — fails (placeholder
   package).
   STEP (b): `internal/logging/debug.go`: `Debug struct{enabled bool, out io.Writer,
   clock}` + `NewDebug(getenv func(string) string, out io.Writer) *Debug` +
   `(d *Debug) Logf(tag, format string, args ...any)`.

## Preconditions (each states its own pass condition)
- `grep -c 'request_details' internal/store/migrate.go` ≥ 1 (w5-a merged).
- `ls internal/store/requestdetails.go 2>/dev/null | wc -l` outputs `0` (gap).
- `grep -c 'func ' internal/logging/doc.go internal/logging/logging_test.go | grep -v ':0' | wc -l` ≤ 1 (logging is still the placeholder).
- `grep -c 'SanitizeHeaders' -r internal/ --include='*.go'` outputs `0` (gap; flips ≥1).

## Exclusive file ownership
NEW: `internal/store/requestdetails.go`(+test),
`internal/usage/{observability,detailwriter}.go`(+tests),
`internal/logging/debug.go`(+test). TOUCHES NO w5-b file
(`internal/store/requestlog*.go`, `internal/usage/{recorder,recent,tracker}*.go`) —
the two run concurrently.

## Binary acceptance
- `go build ./... && go vet ./...` clean; `go test ./...` green; `go test -race ./internal/store/ ./internal/usage/ ./internal/logging/` green.
- `grep -c 'authorization' internal/usage/observability.go` ≥ 1 (sanitizer list ported).
- `grep -rc 'bloodf/g0router/internal' internal/store/requestdetails.go` → `:0` (store leaf preserved).
- `grep -c 'init()' internal/logging/debug.go internal/usage/observability.go internal/usage/detailwriter.go` → 0 each.
- TestObservabilityConfig, TestSanitizeHeaders, TestTruncateField, TestWriterFlushAtBatchSize, TestWriterRetention, TestWriterCloseFlushes, TestWriterConcurrent, TestRequestDetailsQuery, TestDebugLogProductionGate all pass.

## Out of scope
HTTP routes serving these queries (w5-d). Wiring DetailWriter.Save/Close into
handlers and server shutdown (w5-f wires Save call-sites; the server Close hook
ships with w5-f's glue). Usage stats/charts/logs (w5-d). SSE (w5-e). The audit
trail mentioned in `internal/logging/doc.go` (not a Wave-5 row).
