# w4-c — Connection/account state: locks, backoff, disabled models

Rows: PAR-ROUTE-012 (per-model lock `modelLock_${model}`, `open-sse/services/accountFallback.js:106-114`, `src/sse/services/auth.js:203-241`), PAR-ROUTE-013 (account-level `modelLock___all`), PAR-ROUTE-014 (exponential backoff: L1→1s,2→2s,3→4s… capped 4min, `accountFallback.js:9-13,31-34`, `errorConfig.js:32-35`; ref keeps `backoffLevel` ON the connection row), PAR-ROUTE-015 (success clears lock+backoff), PAR-ROUTE-025 (disabled-model tracking `disabledModelsDb {providerAlias:[modelId]}`, `src/app/api/models/disabled/route.js:1-50`), PAR-ROUTE-026 (disabled excluded from /v1/models, `v1/models/route.js:190-191`), PAR-ROUTE-049 (group lock: all locked → earliest retry). Frozen ref @ 827e5c3. Depends: w4-pre MERGED. Parallel-safe with w4-a/w4-b.

Go-port consideration (verbatim): "Replace flat `modelLock_*` fields with a dedicated `connection_model_locks` table (connection_id, model, expires_at) for queryability." So: locks → new table {connection_id, model("__all" for account-level), expires_at}; backoff_level → an additive COLUMN on `connections` (ref keeps it on the connection, PAR-ROUTE-014).

## Tasks (STEP (a) failing tests FIRST; STEP (b) implement)
1. **Store** (`internal/store/connlocks.go` NEW + migrate). (a) `TestModelLockCRUD`, `TestAccountLockSentinel` ("__all"), `TestMigrationAdditiveRerun`, `TestEarliestExpiryAcrossConnections`. (b) `connection_model_locks` table {connection_id, model, expires_at}; additive `connections` columns `backoff_level INT`, `rate_limited_until INT`, `last_error TEXT` (ensureColumn); funcs LockModel/LockAccount/ClearLocks(connID)/ActiveLocks(connID)/EarliestExpiry(providerID,model)/SetBackoffLevel/GetBackoffLevel.
2. **Cooldown engine** (`internal/inference/accounts.go` NEW). (a) `TestBackoffSchedule` (exact steps 1s/2s/4s…cap 240s from `accountFallback.js:9-13`), `TestSuccessResets`, `TestGroupLockEarliestRetry`. (b) `MarkUnavailable(connID, model, verdict Verdict)` where `Verdict` is an ENUM DEFINED IN THIS FILE (`type Verdict int` — RateLimit/Auth/Transient/Permanent; w4-b's classifier maps to it, but w4-c owns the type — no w4-b import dependency); compute next backoff from the connection's backoff_level, write lock+rate_limited_until; `MarkSuccess` clears (015); `GroupRetryAfter(providerID, model)` returns earliest expiry (049). Injected clock.
3. **Disabled models** (`internal/store/disabledmodels.go` NEW + `/api/models/disabled` handler + /v1/models filter). (a) `TestDisabledModelTracking` ({providerAlias:[modelId]} CRUD), `TestModelsListExcludesDisabled`. (b) store keyed by provider alias (ref `disabledModelsDb` shape, `models/disabled/route.js:1-50`); `/v1/models` List filters them out (`internal/api/models.go` TOUCH — AFTER w4-pre owns models.go; serialized).

## Preconditions
- `grep -rn 'connection_model_locks\|connlocks' internal/` → 0 hits.
- w4-pre merged: `grep -c 'TestModelsGetByID' internal/api/models_test.go` ≥ 1.

## Exclusive file ownership
NEW: `internal/store/connlocks.go`+test, `internal/store/disabledmodels.go`+test, `internal/inference/accounts.go`+test, `internal/admin/disabledmodels.go`+test. TOUCH: `internal/store/migrate.go`, `internal/api/models.go`+test (exclusion filter only), `internal/server/routes_admin.go` (register /api/models/disabled).

## Binary acceptance
- `go test ./... && go vet ./... && go test -race ./internal/inference/ ./internal/store/` green.
- TestBackoffSchedule pins exact ref steps; TestGroupLockEarliestRetry, TestModelsListExcludesDisabled, TestMigrationAdditiveRerun pass; `backoff_level` is a connection column (not a lock-table column).

## Out of scope
Selection strategies + the global selection mutex (w4-d, PAR-ROUTE-017). Combo cooldown (w4-e). Classifier rules (w4-b). Request-log attribution (W5).


## Plan-gate disposition (Fable 5, 2026-06-12)
CLOSED BY DECISION after 2 substantive cycles. Round-1 + round-2 substantive findings
FIXED: dropped non-parity scope (027 weighted, 009/040 provider-nodes), global
selection mutex (017), backoff on connection column (014), combo strategy in settings
+ reset-on-restart map not TTL (002), 023=up-to-3-attempts, 033 +Antigravity/Responses,
037 six kinds, fallbackStrategy key + pinned param (w4-d), combo regex dots (w4-e),
explicit STEP(a)/(b) test-first, settings.go serialization. Residual rejections are a
HARNESS-CONTEXT artifact, rebutted: the plan gate is fed only `9router-routing.md`, so
(a) PAR-PR rows (485/640/648/1626) read as "not a valid row / not in matrix" — they ARE
in `PARITY.md` (e.g. PR-1626 at :129); (b) in-tree facts read as "no evidence" though
VERIFIED present — `internal/translation/bypass_handler.go` EXISTS (w1, unwired),
`internal/inference/factory.go providerForModel` EXISTS (w2-d); (c) cross-plan staged
deps (w4-c Verdict enum consumed by w4-d/e) are by-design dependency-inversion, not
ambiguity; (d) whole-file cites for obvious stream loops. The Kimi DIFF gate at
implementation (with full source context) is the binding check.
