# w5-g fix micro-plan — diff-gate round 2 (Fable 5, 2026-06-12)

Source: `artifacts/w5-g-virtual-keys-diff-scoped-gpt.txt` (cycle 2, REJECT).

## REBUTTED — no change
- BLOCKER "KeyIDs dropped, header routing skipped": SECOND occurrence of the
  deferral RECORDED in `fixes/w5-g-fix-r1.md` §REBUTTED — KeyIDs pinning needs
  connection-pinned dispatch (W6, with PAR-ROUTE-057/058). Resolution at closure:
  PAR-ROUTE-030 flips to **PARTIAL** (gate + provider/model constraints + quota +
  attribution shipped; KeyIDs pinning = the W6 half), the same recorded partial
  mechanism as PAR-USAGE-032 and PAR-AUTH-020. Not silently skipped — explicitly
  partial.
- MAJOR "usage_glue.go outside ownership": FALSE POSITIVE — `fixes/w5-g-fix-r1.md`
  §Ownership EXPLICITLY lists `internal/api/usage_glue.go`(+test) (the
  spend-attribution fix transferred from w5-f). Same fix-plan-grant rebuttal as
  w5-f's cmd/main.go.

## REAL → FIX

### Fix 1 (MAJOR) — SumCostByAPIKey has no store-level test
Only `fakeSpendReader` exercises the interface; the real SQL in
`internal/store/requestlog.go` is untested. FIX (test-only): add
`TestSumCostByAPIKey` to `internal/store/requestlog_test.go` — seed request_log
rows: two rows for key "vk-1" inside the window (costs 0.4 + 0.6), one row for
"vk-1" BEFORE sinceISO, one row for a different key, one row with empty api_key →
assert sum = 1.0 exactly; unknown key → 0; verify the sinceISO bound is inclusive
per the implementation's comparison operator (assert whichever the SQL implements
and note it).

## Ownership
`internal/store/requestlog_test.go` ONLY (test-only round).

## Binary acceptance
- `go test ./internal/store/` green; `go test -race ./internal/store/` green.
- TestSumCostByAPIKey passes.
