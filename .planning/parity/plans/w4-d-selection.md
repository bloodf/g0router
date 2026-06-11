# w4-d — Account selection strategies + fallback loop

Rows: PAR-ROUTE-016 (fallback loop with excludeConnectionIds, `src/sse/handlers/chat.js:162-245`), PAR-ROUTE-017 (selection mutex — ref serializes ALL selection via a global promise chain, `src/sse/services/auth.js:9-30`; Go: a per-provider mutex is the faithful-or-better equivalent, documented), PAR-ROUTE-018 (fill-first/round-robin strategies, `auth.js:102-157`), PAR-ROUTE-019 (sticky round-robin limit), PAR-ROUTE-027 (weighted selection), PAR-ROUTE-050 (per-provider strategy override `providerStrategies`), PAR-ROUTE-051 (pinned connection preference) + PAR-PR-640 (prevent infinite retry when all accounts error). Frozen ref @ 827e5c3. Depends: w4-c MERGED (locks/cooldown state).

## Tasks (tests FIRST each)
1. Strategy engine (`internal/inference/selection.go` NEW): `SelectConnection(providerID, model, exclude []string) (*store.Connection, error)` honoring: pinned first (051), then strategy fill-first|round-robin|weighted (018/027; sticky counter w/ limit 019 — in-memory with TTL per Go-port consideration "combo rotation state in-memory with TTL; SQLite too slow"), per-provider override from settings `providerStrategies` (050; settings JSON key), skipping locked/cooled connections (w4-c `ActiveLocks`), per-provider `sync.Mutex` (017). Injected clock + rand seam. Tests: `TestPinnedPreferred`, `TestFillFirst`, `TestRoundRobinSticky` (limit honored, counter resets), `TestWeighted` (deterministic via injected rand), `TestStrategyOverridePerProvider`, `TestSkipsLockedConnections`, `TestSelectionConcurrent` (-race).
2. Fallback loop (`internal/inference/selection.go`): `WithAccountFallback(providerID, model, fn)` — try selected connection; on classifier-class failure mark unavailable (w4-c) and retry NEXT connection with exclude-list (016); TERMINATES when all tried (PR-640 — no infinite loop; returns the group-lock earliest-retry error from w4-c when exhausted). Tests: `TestFallbackAdvancesOnFailure`, `TestFallbackTerminatesAllExcluded` (PR-640), `TestFallbackSuccessMarksReset`.

## Preconditions
- `grep -c 'func.*ActiveLocks' internal/store/connlocks.go` ≥ 1 (w4-c merged).
- `grep -rn 'selection.go' internal/inference/` → 0 hits (new).

## Exclusive file ownership
NEW: `internal/inference/selection.go`+test. NOT: accounts.go internals (w4-c, consumed), retry.go (w4-b), router/factory/alias, api/.

## Binary acceptance
- `go test ./... && go vet ./... && go test -race ./internal/inference/` green.
- `TestRoundRobinSticky`, `TestFallbackTerminatesAllExcluded`, `TestSelectionConcurrent` pass; strategy defaults match ref (`auth.js:102-157` order pinned in a fixture test).

## Out of scope
Combo-level model fallback (w4-e — this is ACCOUNT fallback within one provider/model). Handler wiring (w4-f). Free-tier no-auth injection (Stage 2).
