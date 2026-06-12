# w5-d fix micro-plan — diff-gate round 2 (Fable 5, 2026-06-12)

Source: `artifacts/w5-d-usage-read-apis-diff-scoped-gpt.txt` (cycle 2, REJECT).

## Finding 1 (BLOCKER) — "routes_admin.go:80-81 registers /api/usage/stream and
/api/usage/{connectionId}, out of scope for w5-d."
FALSE POSITIVE — commit-range pollution, PROVEN by blame: those two registrations
were committed by `a6f2f12` ("phase-1/w5-e: connection usage route with refresh and
retry") and gated/merged under w5-e's own diff gate (closed by decision 2026-06-12).
They appear in this diff only because w5-d's commit range spans w5-e's interleaved
serial edits to the same file. Recorded gate artifact (same class as w5-a cycle-2a).
NO CHANGE.

## Finding 2 (MAJOR) — "fillTrackerFields reads s.tracker.byModel without holding
tracker.mu."
REAL (verified: `stats.go:407` iterates byModel BEFORE the `s.tracker.mu.Lock()` at
:412; only the byAccount loop is guarded). FIX: acquire `s.tracker.mu` before the
byModel/Pending loop and release after byAccount (one guarded section). Test FIRST:
`TestStatsTrackerConcurrent` — run Stats() concurrently with tracker Start/End under
`-race`; fails (race report) before the fix, clean after.

## Finding 3 (MAJOR) — "LoadDailyRange hardcodes time.Now(), test is date-dependent."
REAL (verified `requestlog.go:176`; repo convention is injected clocks). FIX: change
signature to `LoadDailyRange(maxDays int, now time.Time)` (caller `internal/usage/
stats.go` passes its injected clock; production wiring already holds one). Update
`TestLoadDailyRange` to a fixed `now` with deterministic dateKey fixtures.

## Ownership
`internal/usage/stats.go`(+test), `internal/store/requestlog.go`(+test), plus the
single LoadDailyRange call-site update in `internal/usage/stats.go`. NOTHING else.
A concurrent job owns internal/api, cmd/g0router/main.go, and
internal/server/usage_smoke_test.go — NEVER touch them. ABSOLUTE PROHIBITION:
never `git checkout`/`restore`/`stash` ANY unowned path, even temporarily; verify
with package-scoped `go test ./internal/usage/ ./internal/store/` if the full tree
is mid-edit; index.lock retry 5×10s.

## Binary acceptance
- `go build ./internal/usage/... ./internal/store/...` + vet green; `go test ./internal/usage/ ./internal/store/` green; `go test -race ./internal/usage/ ./internal/store/` green (incl. the new concurrent test).
- `grep -n 'time.Now' internal/store/requestlog.go` → no hits inside LoadDailyRange.
- TestStatsTrackerConcurrent passes under -race.
