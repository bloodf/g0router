# w5-b fix micro-plan — diff-gate round 1 (Fable 5, 2026-06-12)

Source: `artifacts/w5-b-usage-write-path-diff-scoped-gpt.txt` (cycle 1, REJECT).
All four findings verified REAL against the live tree (2026-06-12). Per-finding:

## Finding 1 (MAJOR) — "requestlog.go:137 — ListRecentRequestLogs scans nullable
columns into string; rows with NULL provider/model fail. Fix: sql.NullString."
REAL (verified: `rows.Scan(&e.Timestamp, &e.Provider, ...)` with plain strings;
schema columns provider/model/connection_id/api_key/endpoint/status are nullable —
rows imported from a 9router DB carry NULLs even though Go-written rows do not).
FIX: scan `sql.NullString` for the six nullable TEXT columns and assign `.String`.
Test FIRST: `TestListRecentRequestLogsNullColumns` — raw-SQL insert a row with NULL
provider/model/connection_id/api_key/endpoint/status → ListRecentRequestLogs returns
it with empty strings, no error; run failing → fix. Apply the same NullString
treatment to any other reader in requestlog.go scanning those columns.

## Finding 2 (MAJOR) — "ring.go:35 — Ring.Init does not enforce capacity on initial
load."
REAL (verified: Init appends all lister items without truncation). FIX: after the
reverse-append loop, truncate to the last `cap` items (keep NEWEST — consistent with
Push semantics). Test FIRST: `TestRingInitEnforcesCap` (NewRing(3), lister returns 5
→ Snapshot has 3, the 3 newest); run failing → fix.

## Finding 3 (MAJOR) — "tracker.go:58 — Start emits while holding Tracker.mu;
synchronous callbacks that inspect tracker state deadlock. Same in End/timeout.
Emit after unlocking."
REAL (verified: `defer t.mu.Unlock()` + `t.events.Emit("pending")` inside the
locked region in Start; same shape in End and zeroOnTimeout). This WILL deadlock
w5-e's SSE consumer, whose "pending" callback reads tracker state. FIX: restructure
Start/End/zeroOnTimeout to release the mutex before Emit (drop defer where needed
or collect-and-emit after an explicit Unlock). Test FIRST:
`TestTrackerEmitReentrant` — register a callback that calls `Snapshot()` (or the
exported state reader) on the SAME tracker; Start/End/timeout each complete within
1s (guard with a timeout channel); run — deadlocks/fails → fix. Keep
TestTrackerConcurrent green under -race.

## Finding 4 (MAJOR) — "TestAggregateEntryToDay does not cover the required
without-provider/without-connection exact key shapes or meta preservation."
REAL (plan §Task 2 STEP (a) required "entry with/without provider, connectionId,
apiKey, endpoint → exact key shapes + meta fields preserved"). FIX (test-only):
extend the table with cases — (a) entry WITHOUT provider → byModel key is bare
`model` (no `|provider` suffix, per usageRepo.js:63), byApiKey/byEndpoint keys use
`unknown` provider segment (`:71,75`); (b) entry WITHOUT connectionId → no byAccount
entry; (c) entry WITHOUT apiKey → byApiKey key `local-no-key|model|provider`... NOTE
verify against ref `usageRepo.js:70-72`: apiKeyVal falls back to "local-no-key" and
the composite is `${apiKeyVal}|${model}|${provider||"unknown"}`; (d) meta fields
(rawModel/provider/apiKey/endpoint/accountName) present on the respective counters.
Assert EXACT key strings.

## Ownership
`internal/store/requestlog.go`(+test), `internal/usage/ring.go`(+test),
`internal/usage/tracker.go`(+test). No other files.

## Binary acceptance
- `go build ./... && go vet ./... && go test ./...` green; `go test -race ./internal/store/ ./internal/usage/` green.
- TestListRecentRequestLogsNullColumns, TestRingInitEnforcesCap,
  TestTrackerEmitReentrant pass; extended TestAggregateEntryToDay passes.
- `grep -c 'sql.NullString' internal/store/requestlog.go` ≥ 1.
