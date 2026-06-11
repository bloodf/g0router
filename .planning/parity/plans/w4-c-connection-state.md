# w4-c — Connection/account state: locks, cooldown, disabled models

Rows: PAR-ROUTE-012 (per-model locks `modelLock_${model}`, `open-sse/services/accountFallback.js:106-114`, `src/sse/services/auth.js:203-241`), PAR-ROUTE-013 (account-level `modelLock___all`), PAR-ROUTE-014 (exponential backoff cooldown `backoffLevel`, `accountFallback.js:9-13,30+` — read whole), PAR-ROUTE-015 (success resets lock+backoff), PAR-ROUTE-025 (disabled-model tracking per provider alias), PAR-ROUTE-026 (disabled excluded from /v1/models), PAR-ROUTE-049 (group lock: all accounts locked → return earliest retry). Frozen ref @ 827e5c3. Depends: w4-pre MERGED. Parallel-safe with w4-a/b.

Go-port consideration (verbatim): "Replace flat `modelLock_*` fields with a dedicated `connection_model_locks` table (connection_id, model, expires_at) for queryability." Matrix g0router note: connections table has no is_active/last_error/backoff_level/rate_limited_until columns (`migrate.go:43-55`) — added here additively.

## Tasks (tests FIRST each)
1. Store (`internal/store/connlocks.go` NEW + migrate TOUCH): `connection_model_locks` table (connection_id, model — "__all" sentinel for account-level, expires_at, backoff_level); additive connection columns (last_error, rate_limited_until); CRUD: `LockModel`, `LockAccount`, `ClearLocks(connID)`, `ActiveLocks(connID)`, `EarliestExpiry(providerID, model)`. Tests: `TestModelLockCRUD`, `TestAccountLockSentinel`, `TestMigrationAdditiveRerun`, `TestEarliestExpiryAcrossConnections`.
2. Cooldown engine (`internal/inference/accounts.go` NEW): backoff schedule port (exact steps from `accountFallback.js:9-13`), `MarkUnavailable(conn, model, classifierVerdict)` (uses w4-b classes when merged — NOT a dependency: takes a verdict enum param), `MarkSuccess` clears (PAR-ROUTE-015), group-lock check returns earliest retry-after (PAR-ROUTE-049). Injected clock. Tests: `TestBackoffSchedule` (exact step values), `TestSuccessResets`, `TestGroupLockEarliestRetry`, `-race` on concurrent mark/clear.
3. Disabled models (`internal/store/` settings-or-table per ref `src/.../auth.js` disabled tracking — read evidence; pick the ref's storage shape) + `/v1/models` exclusion hook: catalog list filtered by disabled set (PAR-ROUTE-026; `internal/api/models.go` TOUCH — coordinate: w4-pre owns models.go FIRST; this lands after). Tests: `TestDisabledModelTracking`, `TestModelsListExcludesDisabled`.

## Preconditions
- `grep -rn 'connection_model_locks\|connlocks' internal/` → 0 hits (new).
- w4-pre merged: `grep -c 'TestModelsGetByID' internal/api/models_test.go` ≥ 1.

## Exclusive file ownership
NEW: `internal/store/connlocks.go`+test, `internal/inference/accounts.go`+test. TOUCH: `internal/store/migrate.go`, `internal/api/models.go`+test (exclusion filter only, AFTER w4-pre). NOT: selection strategies (w4-d), retry/classifier internals (w4-b), alias/factory (w4-a).

## Binary acceptance
- `go test ./... && go vet ./... && go test -race ./internal/inference/ ./internal/store/` green.
- `TestBackoffSchedule` pins exact ref steps; `TestGroupLockEarliestRetry`, `TestModelsListExcludesDisabled`, `TestMigrationAdditiveRerun` pass.

## Out of scope
Strategy selection (w4-d). Combo cooldown (w4-e). Classifier rules (w4-b). Request-log attribution (W5).
