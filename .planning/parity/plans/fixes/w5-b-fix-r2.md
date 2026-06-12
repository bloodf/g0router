# w5-b fix micro-plan — diff-gate round 2 (Fable 5, 2026-06-12)

Source: `artifacts/w5-b-usage-write-path-diff-scoped-gpt.txt` (cycle 2, REJECT).
Per-finding triage (verified against live tree 2026-06-12):

## Finding 1 (MAJOR) — "ring.go:33 — Ring.Init mutates r.items without r.mu while
Push/Snapshot use the mutex; lazy init racing with live pushes will fail -race."
REAL (verified: the initOnce.Do closure appends to r.items with no lock; a Push
concurrent with first Init is a data race). FIX: acquire `r.mu` inside the
initOnce.Do closure around the items mutation (initOnce already serializes Init vs
Init; the lock serializes Init vs Push/Snapshot). Test FIRST:
`TestRingInitPushConcurrent` — start Init (lister sleeps briefly via injected
function) and Push from another goroutine; run under `-race`; fails/races before the
fix, clean after.

## Finding 2 (MAJOR) — "ring.go:90 — ConnNameCache.Get returns (map, error),
breaking the plan's `Get() map[string]string` API."
REAL deviation — and the (map, error) shape is ALSO ref-unfaithful: the ref swallows
lister errors and returns the last cached map (`usageRepo.js:88-96` — catch{} then
`return connCache.map`). FIX: change signature to `Get() map[string]string`; on
lister error return the previous cached map (possibly empty, never nil); keep the
TTL/refresh logic. Update all callers/tests in owned files. Test FIRST: extend
`TestConnNameCacheTTL` with a failing-lister case → returns prior map, no panic.

## Finding 3 (MAJOR) — "requestlog.go:141 — silently discards json.Unmarshal errors
for tokens" / Finding 4 (MAJOR) — same for meta.
FALSE POSITIVE — REF-FAITHFUL BY DESIGN: the ref reads these blobs through
`parseJson(value, default)` which explicitly tolerates corrupt JSON and substitutes
the default (`usageRepo.js:108` `parseJson(r.tokens, {})`; helper
`src/lib/db/helpers/jsonCol.js`). Failing the whole listing for one corrupt row
would DIVERGE from ref behavior (ref renders the row with empty tokens). The Go port
`_ = json.Unmarshal(...)` leaves the zero map on failure = the same default
semantics. No change. (Recorded rebuttal; the repo's errors-are-values convention
governs error PROPAGATION paths, not deliberate default-on-corrupt reads that the
reference specifies.)

## Ownership
`internal/usage/ring.go`(+test) only (+ caller updates inside owned w5-b test files
if any consume Get()).

## Binary acceptance
- `go build ./... && go vet ./... && go test ./...` green; `go test -race ./internal/usage/ ./internal/store/` green.
- `grep -c 'func (c \*ConnNameCache) Get() map\[string\]string' internal/usage/ring.go` = 1.
- TestRingInitPushConcurrent passes under -race; extended TestConnNameCacheTTL passes.
