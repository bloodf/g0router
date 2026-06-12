# w5-b — Usage write path + live trackers

PAR rows: PAR-USAGE-001/002 (write semantics — tables landed in w5-a), PAR-USAGE-011,
PAR-USAGE-012, PAR-USAGE-018, PAR-USAGE-019, PAR-USAGE-020, PAR-USAGE-038.
NOT in scope: read APIs (PAR-USAGE-013..017/021..023 → w5-d), observability
(PAR-USAGE-024..028 → w5-c), SSE (PAR-USAGE-034/035 → w5-e), handler glue
(PAR-ROUTE-054, PAR-TRANS-046 → w5-f).
Frozen ref @ 827e5c3. Depends: w5-a merged. Runs ∥ w5-c (disjoint files).

## Layering decision
`internal/store` stays a leaf — the §Preconditions grep below is the binding evidence
(`grep -rh 'bloodf/g0router' internal/store/*.go | grep -v _test | wc -l` → `0`).
The daily-rollup JSON blob is a persistence format, so its aggregation helper lives
INSIDE store (private), mirroring the ref where aggregation is repo-internal
(`usageRepo.js:44-77`). Cost is computed in the DOMAIN (`internal/usage.Recorder`)
before persisting — port of `usageRepo.js:248` `entry.cost = await calculateCost(...)`.
Domain→repository imports are the established in-repo pattern per `AGENTS.md:24`
(transport→domain→repository dependency direction) and the live precedent
`internal/inference/combo.go:12` + `internal/inference/selection.go:11` (both import
`internal/store`): therefore `internal/usage/recorder.go` and `tracker.go` use
`*store.RequestLogEntry`/store types DIRECTLY — no duplicate entry struct, no adapter.
(w5-a's "usage does not import store" acceptance bound w5-a's pure pricing files —
`pricingdata/pricing/tokens/cost.go` stay store-free; the recorder/tracker files this
plan adds follow the inference precedent.)

## Tasks

1. **Transactional usage save (PAR-USAGE-011, PAR-USAGE-001/002 write)** — evidence:
   `usageRepo.js:243-287`: ONE transaction = history INSERT + daily upsert
   (read-modify-write of `usage_daily.data` JSON) + lifetime counter increment;
   `usageRepo.js:254-255` comments the atomicity requirement.
   STEP (a): `TestSaveUsageTransactional` (save → request_log row present, usage_daily
   day JSON has requests=1 + correct byProvider/byModel sums, lifetime counter=1; second
   save same day → day accumulates, counter=2) and `TestSaveUsageRollsBackTogether`
   with a DETERMINISTIC failure seam: the test drops the `kv` table
   (`DROP TABLE kv`) before calling SaveUsage — the third write (lifetime counter
   upsert) then fails inside the tx → assert SaveUsage returns an error AND
   `SELECT COUNT(*) FROM request_log` = 0 AND `usage_daily` is empty (the first two
   writes rolled back); run — fail.
   STEP (b): NEW `internal/store/requestlog.go`: `RequestLogEntry` struct (Timestamp
   ISO-8601 string, Provider, Model, ConnectionID, APIKey, Endpoint, PromptTokens,
   CompletionTokens int64, Cost float64, Status string, Tokens map[string]int64, Meta
   map[string]string); `(s *Store) SaveUsage(e *RequestLogEntry) error` running all
   three writes in `db.Begin()`/`Commit`. Lifetime counter = kv row scope='meta',
   key='total_requests_lifetime' (kv table from w5-a), incremented IN the same tx
   (ref `usageRepo.js:276-279` uses `_meta`).

2. **Daily aggregation shape (PAR-USAGE-002)** — evidence: `usageRepo.js:30-77`:
   local date key YYYY-MM-DD; day totals (requests/promptTokens/completionTokens/cost);
   byProvider keyed by provider; byModel key `model|provider` with meta
   {rawModel, provider}; byAccount keyed connectionId with meta; byApiKey key
   `apiKey|model|provider` with `local-no-key` fallback (`:70-72`); byEndpoint key
   `endpoint|model|provider` with `Unknown` endpoint fallback (`:74-76`); counter
   metas shallow-assigned.
   STEP (a): table-driven `TestAggregateEntryToDay` (entry with/without provider,
   connectionId, apiKey, endpoint → exact key shapes + accumulation across two
   entries + meta fields preserved); run — fails.
   STEP (b): private `aggregateEntryToDay(day map[string]any, e *RequestLogEntry)` in
   `internal/store/requestlog.go`, JSON-roundtrip-safe (camelCase JSON field names
   matching the ref blob so w5-d's reader and any 9router data importer agree:
   byProvider/byModel/byAccount/byApiKey/byEndpoint, promptTokens, completionTokens).

3. **Cost-at-save Recorder (PAR-USAGE-012)** — evidence: `usageRepo.js:243-252`:
   timestamp defaulted, cost computed from provider+model+tokens via pricing
   resolution, prompt/completion extracted via synonym normalization
   (`usageRepo.js:121-122` ports as w5-a `NormalizeTokens`). Emits the "update" event
   after a successful save — this is INSIDE PAR-USAGE-011's cited port range:
   the row's evidence is `usageRepo.js:243-287` (`saveRequestUsage`), and line 283
   `statsEmitter.emit("update")` is part of that function's behavior; omitting it
   would be an incomplete port of the row w5-b owns. (PAR-USAGE-034, out of scope
   here, is the SSE CONSUMER of this event — w5-e.)
   STEP (a): `TestRecorderComputesCost` (fake UsageStore capturing the entry; Recorder
   over w5-a Resolver with known model pricing → entry.Cost matches golden value;
   missing pricing → Cost 0; timestamp filled when empty) and `TestRecorderEmitsUpdate`
   (registered callback receives kind "update" exactly once per successful Record);
   run — fail.
   STEP (b): NEW `internal/usage/recorder.go`: `UsageStore` interface typed DIRECTLY
   on the store type (`SaveUsage(*store.RequestLogEntry) error` — §Layering decision;
   inference precedent), `Recorder{resolver, store, clock, events *Events}` with
   `Record(entry) error`: normalize tokens → CostFor → SaveUsage → emit "update".

4. **Recent-request dedup (PAR-USAGE-038)** — evidence: `usageRepo.js:217-237` and
   duplicate logic `:345-365`: drop zero-token entries; dedupe key
   `model|provider|promptTokens|completionTokens|minute` (minute = timestamp[:16]);
   newest-first; cap 20.
   STEP (a): table-driven `TestDedupeRecent` (zero-token dropped; same-minute duplicate
   collapsed; different-minute kept; cap at 20; sort newest-first); run — fails.
   STEP (b): `internal/usage/recent.go`: `RecentRequest` struct +
   `DedupeRecent([]RecentRequest) []RecentRequest` (pure).

5. **Pending tracker + event emission (PAR-USAGE-018)** — evidence:
   `usageRepo.js:6,153-196`: byModel counts keyed `model (provider)`; byAccount nested
   connectionId→modelKey; START increments / END decrements clamped ≥0 with map
   cleanup; 60s timer per `connectionId|modelKey` zeroes counts and emits; END clears
   the timer; error END records lastErrorProvider (lowercased) with 10s read window
   (`:188-191,239`). Event emission is IN the cited evidence, not w5-e scope: the
   tracker itself emits "pending" on every change (`usageRepo.js:181,195`
   `statsEmitter.emit("pending")` — both lines inside the PAR-USAGE-018 evidence range
   153-196; the emitter global is `:14-17`). w5-e only CONSUMES the events.
   STEP (a): `TestTrackerStartEnd` (counts, clamp, cleanup), `TestTrackerTimeout`
   (injected timer-factory fires → counts zeroed, "pending" emitted),
   `TestTrackerErrorProvider` (10s window via injected clock), `TestTrackerEmitsPending`
   (one callback invocation kind "pending" per Start/End), `TestTrackerConcurrent`
   (parallel Start/End under -race); run — fail.
   STEP (b): NEW `internal/usage/events.go`: `Events` — the Go translation of the
   ref's `statsEmitter` EventEmitter, which is itself part of the files/rows w5-b
   ports (`usageRepo.js:14-17` declares it; every emit site lies inside w5-b's cited
   ranges: :181,:195 in PAR-USAGE-018's 153-196, :283 in PAR-USAGE-011's 243-287).
   Mutex-guarded callback registry, EXACTLY this API:
   `(e *Events) OnEvent(fn func(kind string))` + `(e *Events) Emit(kind string)`
   (synchronous fan-out; kinds used: "pending", "update" — the two the ref emits). NEW `internal/usage/tracker.go`: `Tracker`
   (mutex-guarded maps; injected `clock func() time.Time` + `timerFactory
   func(time.Duration, func()) (stop func())` — production wraps `time.AfterFunc`;
   holds *Events and emits "pending"). No globals, no init().

6. **Ring buffer + connection-name cache (PAR-USAGE-019, PAR-USAGE-020)** — evidence:
   `usageRepo.js:7,79-111` (ring cap 50, lazily initialized once from last-50 history
   rows, push appends + truncates) and `:8,86-97` (conn map id→name|email|id, 30s TTL).
   STEP (a): `TestRingInitOnceFromStore` (fake last-N lister called exactly once; later
   pushes append; cap 50), `TestConnNameCacheTTL` (injected clock: within TTL no
   re-list; after TTL re-lists; fallback chain name→email→id); run — fail.
   STEP (b): in NEW `internal/usage/ring.go` (deterministic — not optional): `Ring` with
   `Init(lister)` / `Push` / `Snapshot`; `ConnNameCache{lister, ttl, clock}` with
   `Get() map[string]string`. Store side: `(s *Store) ListRecentRequestLogs(limit int)
   ([]*RequestLogEntry, error)` (ORDER BY id DESC LIMIT ?) in requestlog.go —
   ring init + w5-d recent-requests both consume it.

## Preconditions (each states its own pass condition)
- `grep -c 'request_log' internal/store/migrate.go` ≥ 1 (w5-a merged — tables exist).
- `grep -c 'func MatchPattern\|func.*CostFor' internal/usage/*.go` ≥ 1 (w5-a pricing engine present).
- `ls internal/store/requestlog.go 2>/dev/null | wc -l` outputs `0` (write path is the gap).
- `grep -rh 'bloodf/g0router' internal/store/*.go | grep -v _test | wc -l` outputs `0` (store leaf invariant — evidence for §Layering decision; preserved by this plan).

## Exclusive file ownership
NEW (exact, deterministic): `internal/store/requestlog.go`(+test),
`internal/usage/{recorder,recent,tracker,events,ring}.go`(+tests).
w5-c's ownership is the binding cross-reference: `w5-c-observability.md`
§Exclusive file ownership (plan-gate PASS 2026-06-12) lists
`internal/store/requestdetails.go`, `internal/usage/{observability,detailwriter}.go`,
`internal/logging/debug.go` — zero overlap with this plan's list.
TOUCHES NO file owned by w5-c
(`internal/store/requestdetails*.go`, `internal/usage/observability*.go`,
`internal/logging/*`) — the two run concurrently.

## Binary acceptance
- `go build ./... && go vet ./...` clean; `go test ./...` green; `go test -race ./internal/store/ ./internal/usage/` green.
- `sqlite3` smoke on a migrated DB after one SaveUsage: `SELECT COUNT(*) FROM request_log` = 1; `SELECT COUNT(*) FROM usage_daily` = 1; kv meta counter = '1'.
- `grep -rc 'bloodf/g0router/internal' internal/store/requestlog.go` → `:0` (leaf preserved).
- TestSaveUsageTransactional, TestAggregateEntryToDay, TestRecorderComputesCost,
  TestRecorderEmitsUpdate, TestDedupeRecent, TestTrackerTimeout, TestTrackerEmitsPending,
  TestTrackerConcurrent, TestRingInitOnceFromStore, TestConnNameCacheTTL all pass.

## Out of scope
getUsageStats/chart/logs readers (w5-d). Observability writer (w5-c). SSE emit
consumption (w5-e — the tracker only EXPOSES the event seam). Handler wiring of
Recorder/Tracker into chat/messages/embeddings (w5-f). Virtual keys (w5-g).

## Plan-gate disposition (cycle 3, Fable 5, 2026-06-12) — CLOSED BY DECISION
Three substantive cycles complete. Cycle-1 findings FIXED (layering claim → binding
precondition grep; event seam tied to in-range emit sites usageRepo.js:181/195/283 +
exact Events API specified; store-import question settled by inference precedent
combo.go:12/selection.go:11). Cycle-2 findings FIXED (update-emit anchored inside
PAR-USAGE-011's evidence range 243-287; Events justified as the statsEmitter port
declared at usageRepo.js:14-17 within owned files; ring.go made deterministic).
Cycle-3 residual findings triaged:
- MAJOR rollback-seam: REAL → FIXED in-plan (deterministic DROP TABLE kv seam; binary
  assertions on request_log/usage_daily emptiness).
- MAJOR w5-c ownership evidence: REAL → FIXED in-plan (binding cross-reference to
  w5-c-observability.md §Exclusive file ownership, plan-gate PASS same day; zero
  file overlap enumerated).
- MINOR events scope: FALSE POSITIVE — emit("update") at :283 is inside
  saveRequestUsage, the exact function PAR-USAGE-011 cites (243-287); an omission
  would be an incomplete port. PAR-USAGE-034 (SSE consumer) remains w5-e.
Kimi diff gate at implementation is the binding check. APPROVED BY DECISION for
dispatch after w5-a merges.

## Diff-gate disposition (cycle 3, Fable 5, 2026-06-12) — CLOSED BY DECISION
Three substantive cycles complete. Cycle-1: 4 REAL findings FIXED (fix-r1, 80679a9:
NullString scans, ring init cap, emit-after-unlock, exact aggregation key tests).
Cycle-2: 2 REAL FIXED (fix-r2, cdefa31: ring init under mutex, ref-faithful
`Get() map[string]string` returning stale map on lister error) + 2 rebutted
(tokens/meta parse tolerance = ref's parseJson(value, default), usageRepo.js:108).
Cycle-3 residual triage — three findings contradict the frozen reference itself:
- MAJOR "missing-provider byModel should be model|provider": FALSE POSITIVE —
  `usageRepo.js:63` `entry.provider ? `${model}|${provider}` : entry.model` — bare
  model IS the ref shape (and fix-r1's Finding 4 instructed exactly this).
- MAJOR "'unknown' provider segment breaks compatibility": FALSE POSITIVE — the ref
  writes `${provider || "unknown"}` in BOTH byApiKey (`:71`) and byEndpoint (`:75`)
  composites; "unknown" IS the importer-compatible shape.
- MAJOR "timeout corrupts byModel for other connections": FALSE POSITIVE — ref
  timeout behavior verbatim: `usageRepo.js:176-181` zeroes the GLOBAL
  byModel[modelKey] (and the timer connection's account entry) on a single
  connection's 60s timeout. A 9router quirk, ported faithfully (parity program
  ports behavior, including quirks — recorded matrix §Edge cases notes the timer
  semantics).
- MAJOR "no ORDER BY id DESC test": residual test nit accepted — the ordering is
  exercised through TestRingInitOnceFromStore (newest-first reversal contract);
  follow-up coverage lands with w5-d's TestRecentLogsFormat which asserts exact
  ordered lines over the same query.
Build/vet/test/-race green post-fix-r2 (verified live). MERGED.
Rows flip: PAR-USAGE-001/002 (write semantics complete), 011, 012, 018, 019, 020,
038 → HAVE.
