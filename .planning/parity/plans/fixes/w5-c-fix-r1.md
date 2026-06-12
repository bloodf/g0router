# w5-c fix micro-plan — diff-gate round 1 (Fable 5, 2026-06-12)

Source: `artifacts/w5-c-observability-diff-scoped-gpt.txt` (cycle 1, REJECT).
Per-finding triage (verified against live tree 2026-06-12):

## Finding 1 (BLOCKER) — "observability.go:154 — TruncateField slices str[:200];
values larger than maxSize but shorter than 200 bytes panic."
REAL as a defensive defect (production config cannot reach it — maxSize is KB*1024 ≥
1024 > 200 — but the function is exported and the repo convention is never-panic).
FIX: preview length = min(200, len(str)). Test FIRST: `TestTruncateFieldShortOversize`
(maxSize=10, value serializing to ~50 bytes → marker returned, preview = whole
string, NO panic); run failing (panics) → fix.

## Finding 2 (MAJOR) — "maxJsonSize overrides treated as KB and multiplied by 1024,
but the plan ports maxJsonSize as a byte limit."
FALSE POSITIVE — the KB×1024 treatment is the REF'S OWN BEHAVIOR, verbatim:
`requestDetailsRepo.js:27` `maxJsonSize: (settings.observabilityMaxJsonSize ||
parseInt(process.env.OBSERVABILITY_MAX_JSON_SIZE || "5", 10)) * 1024` — BOTH the
settings value and the env value are kilobytes multiplied to bytes. The plan's
"maxJsonSize=5KB default" describes exactly this. No change.

## Finding 3 (MAJOR) — "detailwriter.go:66 — Save takes *RequestDetail but the plan
requires Save(detail RequestDetail); permits nil-pointer panics."
REAL conformance drift. FIX: change signature to `Save(detail RequestDetail) error`
(value, per plan §Task 4); update all callers/tests in owned files. (This also
removes the nil-pointer class entirely.)

## Finding 4 (MAJOR) — "TestWriterRetention only checks count, not that the oldest
rows are gone."
REAL test gap (verified: asserts len==3 only; also all 5 saves share one injected
clock instant, so oldest-by-timestamp is ambiguous). FIX: advance the injected clock
per save (distinct timestamps), capture generated ids (or query by timestamp), and
assert the TWO OLDEST timestamps are absent and the three newest present.

## Finding 5 (MAJOR) — "TestRequestDetailsQuery omits model and connectionId filter
assertions."
REAL test gap (verified: subtests cover provider, provider+status, pagination, date
range only). FIX: add subtests `filter by model` (model=gpt-4o → d1,d5,d6) and
`filter by connectionId` (conn-2 → d3,d4) asserting exact ids via the data blob.

## Ownership
`internal/usage/observability.go`(+test), `internal/usage/detailwriter.go`(+test),
`internal/store/requestdetails_test.go` (test-only). No other files.

## Binary acceptance
- `go build ./... && go vet ./... && go test ./...` green; `go test -race ./internal/usage/ ./internal/store/` green.
- `grep -c 'func (w \*DetailWriter) Save(detail RequestDetail)' internal/usage/detailwriter.go` = 1.
- TestTruncateFieldShortOversize passes; extended TestWriterRetention asserts oldest-gone; model/connectionId filter subtests pass.
