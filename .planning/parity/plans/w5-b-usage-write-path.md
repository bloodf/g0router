# w5-b — Usage write path + live trackers

PAR rows: PAR-USAGE-001/002 (write semantics — tables landed in w5-a), PAR-USAGE-011,
PAR-USAGE-012, PAR-USAGE-018, PAR-USAGE-019, PAR-USAGE-020, PAR-USAGE-038.
NOT in scope: read APIs (PAR-USAGE-013..017/021..023 → w5-d), observability
(PAR-USAGE-024..028 → w5-c), SSE (PAR-USAGE-034/035 → w5-e), handler glue
(PAR-ROUTE-054, PAR-TRANS-046 → w5-f).
Frozen ref @ 827e5c3. Depends: w5-a merged. Runs ∥ w5-c (disjoint files).

## Layering decision
`internal/store` stays a leaf (imports no internal packages — verified: zero
`bloodf/g0router` imports in non-test store files). The daily-rollup JSON blob is a
persistence format, so its aggregation helper lives INSIDE store (private), mirroring
the ref where aggregation is repo-internal (`usageRepo.js:44-77`). Cost is computed in
the DOMAIN (`internal/usage.Recorder`) before persisting — port of
`usageRepo.js:248` `entry.cost = await calculateCost(...)`.

## Tasks

1. **Transactional usage save (PAR-USAGE-011, PAR-USAGE-001/002 write)** — evidence:
   `usageRepo.js:243-287`: ONE transaction = history INSERT + daily upsert
   (read-modify-write of `usage_daily.data` JSON) + lifetime counter increment;
   `usageRepo.js:254-255` comments the atomicity requirement.
   STEP (a): `TestSaveUsageTransactional` (save → request_log row present, usage_daily
   day JSON has requests=1 + correct byProvider/byModel sums, lifetime counter=1; second
   save same day → day accumulates, counter=2) and `TestSaveUsageRollsBackTogether`
   (inject failure via closed tx / constraint → NO partial writes); run — fail.
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
   (`usageRepo.js:121-122` ports as w5-a `NormalizeTokens`).
   STEP (a): `TestRecorderComputesCost` (fake UsageStore capturing the entry; Recorder
   over w5-a Resolver with known model pricing → entry.Cost matches golden value;
   missing pricing → Cost 0; timestamp filled when empty); run — fails.
   STEP (b): NEW `internal/usage/recorder.go`: `UsageStore` interface
   (`SaveUsage(*store-shaped entry) error` — defined locally with a small entry struct
   to keep usage→store decoupled; server wiring adapts), `Recorder{resolver, store,
   clock}` with `Record(entry) error`: normalize tokens → CostFor → SaveUsage.

4. **Recent-request dedup (PAR-USAGE-038)** — evidence: `usageRepo.js:217-237` and
   duplicate logic `:345-365`: drop zero-token entries; dedupe key
   `model|provider|promptTokens|completionTokens|minute` (minute = timestamp[:16]);
   newest-first; cap 20.
   STEP (a): table-driven `TestDedupeRecent` (zero-token dropped; same-minute duplicate
   collapsed; different-minute kept; cap at 20; sort newest-first); run — fails.
   STEP (b): `internal/usage/recent.go`: `RecentRequest` struct +
   `DedupeRecent([]RecentRequest) []RecentRequest` (pure).

5. **Pending tracker (PAR-USAGE-018)** — evidence: `usageRepo.js:6,153-196`:
   byModel counts keyed `model (provider)`; byAccount nested connectionId→modelKey;
   START increments / END decrements clamped ≥0 with map cleanup; 60s timer per
   `connectionId|modelKey` zeroes counts and emits; END clears the timer; error END
   records lastErrorProvider (lowercased) with 10s read window (`:188-191,239`);
   emits a stats event on every change (SSE hook for w5-e).
   STEP (a): `TestTrackerStartEnd` (counts, clamp, cleanup), `TestTrackerTimeout`
   (injected timer-factory fires → counts zeroed, event emitted),
   `TestTrackerErrorProvider` (10s window via injected clock), `TestTrackerConcurrent`
   (parallel Start/End under -race); run — fail.
   STEP (b): NEW `internal/usage/tracker.go`: `Tracker` (mutex-guarded maps; injected
   `clock func() time.Time` + `timerFactory func(d, fn) stopFn` — production
   `time.AfterFunc`; `Subscribe(chan struct{})`-style or callback list for events —
   pick the simplest seam w5-e can consume). No globals, no init().

6. **Ring buffer + connection-name cache (PAR-USAGE-019, PAR-USAGE-020)** — evidence:
   `usageRepo.js:7,79-111` (ring cap 50, lazily initialized once from last-50 history
   rows, push appends + truncates) and `:8,86-97` (conn map id→name|email|id, 30s TTL).
   STEP (a): `TestRingInitOnceFromStore` (fake last-N lister called exactly once; later
   pushes append; cap 50), `TestConnNameCacheTTL` (injected clock: within TTL no
   re-list; after TTL re-lists; fallback chain name→email→id); run — fail.
   STEP (b): in `internal/usage/tracker.go` (or `ring.go`): `Ring` with
   `Init(lister)` / `Push` / `Snapshot`; `ConnNameCache{lister, ttl, clock}` with
   `Get() map[string]string`. Store side: `(s *Store) ListRecentRequestLogs(limit int)
   ([]*RequestLogEntry, error)` (ORDER BY id DESC LIMIT ?) in requestlog.go —
   ring init + w5-d recent-requests both consume it.

## Preconditions (each states its own pass condition)
- `grep -c 'request_log' internal/store/migrate.go` ≥ 1 (w5-a merged — tables exist).
- `grep -c 'func MatchPattern\|func.*CostFor' internal/usage/*.go` ≥ 1 (w5-a pricing engine present).
- `ls internal/store/requestlog.go 2>/dev/null | wc -l` outputs `0` (write path is the gap).
- `grep -rc 'bloodf/g0router' internal/store/store.go internal/store/connections.go | grep -v ':0' | wc -l` outputs `0` (store leaf invariant to preserve).

## Exclusive file ownership
NEW: `internal/store/requestlog.go`(+test), `internal/usage/{recorder,recent,tracker}.go`
(+tests; `ring.go` optional split). TOUCHES NO file owned by w5-c
(`internal/store/requestdetails*.go`, `internal/usage/observability*.go`,
`internal/logging/*`) — the two run concurrently.

## Binary acceptance
- `go build ./... && go vet ./...` clean; `go test ./...` green; `go test -race ./internal/store/ ./internal/usage/` green.
- `sqlite3` smoke on a migrated DB after one SaveUsage: `SELECT COUNT(*) FROM request_log` = 1; `SELECT COUNT(*) FROM usage_daily` = 1; kv meta counter = '1'.
- `grep -rc 'bloodf/g0router/internal' internal/store/requestlog.go` → `:0` (leaf preserved).
- TestSaveUsageTransactional, TestAggregateEntryToDay, TestRecorderComputesCost,
  TestDedupeRecent, TestTrackerTimeout, TestTrackerConcurrent, TestRingInitOnceFromStore,
  TestConnNameCacheTTL all pass.

## Out of scope
getUsageStats/chart/logs readers (w5-d). Observability writer (w5-c). SSE emit
consumption (w5-e — the tracker only EXPOSES the event seam). Handler wiring of
Recorder/Tracker into chat/messages/embeddings (w5-f). Virtual keys (w5-g).
