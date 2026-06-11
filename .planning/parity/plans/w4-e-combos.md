# w4-e — Combo chains

Rows: PAR-ROUTE-001 (ordered-model fallback strategy, `open-sse/services/combo.js:108-198,151-170`), PAR-ROUTE-002 (round-robin + sticky limit, `combo.js:36-65`), PAR-ROUTE-003 (per-combo strategy override), PAR-ROUTE-004 (name validation alphanum/hyphen/underscore), PAR-ROUTE-011 (recursive resolution protection), PAR-ROUTE-024 (transient 502/503/504 cooldown wait ≤5s, `combo.js:161-165`), PAR-ROUTE-046 (earliest retry-after across combo models), PAR-ROUTE-047 (combo names first in /v1/models) + PAR-PR-648 (reset combo rotation state on combo definition change). Frozen ref @ 827e5c3. Depends: w4-a (alias resolution), w4-b (classifier), w4-c (locks), w4-d (selection) ALL MERGED. Sticky normalization note (matrix): `normalizeStickyLimit` defaults to 1 if invalid (`combo.js:14-17`).

## Tasks (tests FIRST each)
1. Store (`internal/store/combos.go` NEW + migrate): combos table (name UNIQUE w/ validation PAR-ROUTE-004 regex from ref, ordered model list JSON, strategy, sticky_limit). CRUD + name-validation. Tests: `TestComboCRUD`, `TestComboNameValidation` (valid/invalid sets from ref), `TestMigrationAdditiveRerun2`.
2. Combo engine (`internal/inference/combo.go` NEW): `ResolveCombo(name)` returns ordered targets; RECURSION protection (combo→combo refs resolved with visited-set, PAR-ROUTE-011); fallback strategy = try models in order, on failure-class advance (uses w4-d account-fallback per model); round-robin strategy w/ sticky counter (in-memory TTL state; `normalizeStickyLimit` default 1); transient-error cooldown wait ≤5s before next (024); earliest retry-after aggregation across models (046); rotation state RESET when combo definition changes (PR-648 — keyed by a definition hash). Injected clock. Tests: `TestComboFallbackOrder`, `TestComboRoundRobinSticky` (limit + normalization default 1), `TestComboRecursionGuard`, `TestComboTransientCooldownCap5s`, `TestComboEarliestRetryAfter`, `TestComboStateResetOnChange` (PR-648), `-race` on concurrent combo selection.
3. /v1/models promotion (047): combo names listed FIRST in the models list (`internal/api/models.go` TOUCH — after w4-c's filter; coordinate serial). Test: `TestModelsListCombosFirst`.

## Preconditions
- `grep -c 'func SelectConnection\|func WithAccountFallback' internal/inference/selection.go` ≥ 1 (w4-d merged).
- `grep -rn 'combo' internal/ --include='*.go'` (non-test) → 0 hits (new).

## Exclusive file ownership
NEW: `internal/store/combos.go`+test, `internal/inference/combo.go`+test. TOUCH: `internal/store/migrate.go`, `internal/api/models.go`+test (promotion only). NOT: selection/accounts internals, handler dispatch (w4-f wires combos into the chat path).

## Binary acceptance
- `go test ./... && go vet ./... && go test -race ./internal/inference/` green.
- `TestComboRecursionGuard`, `TestComboStateResetOnChange`, `TestComboTransientCooldownCap5s`, `TestModelsListCombosFirst` pass; sticky default-1 pinned.

## Out of scope
Combo UI + PAR-PR-339 (Wave 6). Per-key quotas (W5). Handler wiring (w4-f).
