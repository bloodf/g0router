# w4-e — Combo chains

Rows: PAR-ROUTE-001 (ordered-model fallback strategy, `open-sse/services/combo.js:108-198,151-170`), PAR-ROUTE-002 (round-robin + sticky, `combo.js:36-65`; rotation state is an in-memory Map reset on restart — NOT TTL), PAR-ROUTE-003 (per-combo strategy override), PAR-ROUTE-004 (combo name validation, API-route behavior), PAR-ROUTE-011 (recursive resolution protection), PAR-ROUTE-024 (transient 502/503/504 cooldown ≤5s, `combo.js:161-165`), PAR-ROUTE-046 (earliest retry-after across combo models), PAR-ROUTE-047 (combo names first in /v1/models) + PAR-PR-648 (`PARITY.md` "Reset models state on combo prop change"). Frozen ref @ 827e5c3. Depends: w4-a,b,c,d MERGED.

Storage model (per ref): combos themselves = data `{name, models[]}` (`combo.js:87-89`). Strategy/sticky live in SETTINGS, NOT on the combo row: `settings["comboStrategy"]` (default "fallback"|"round-robin"), `settings["comboStrategies"]` (per-combo override map), `settings["comboStickyRoundRobinLimit"]` (default 1 via `normalizeStickyLimit`, `combo.js:14-17`).

## Tasks (STEP (a) failing tests FIRST; STEP (b) implement)
1. **Store** (`internal/store/combos.go` NEW + migrate). (a) `TestComboCRUD` ({name, models[]} only — NO strategy column), `TestMigrationAdditiveRerun2`. (b) combos table {name UNIQUE, models JSON}; CRUD repository-only (validation is task 3's API layer).
2. **Combo engine** (`internal/inference/combo.go` NEW). (a) `TestComboFallbackOrder`, `TestComboRoundRobinSticky` (sticky from settings; `normalizeStickyLimit` default 1 for invalid), `TestComboRecursionGuard` (011, visited-set), `TestComboTransientCooldownCap5s` (024), `TestComboEarliestRetryAfter` (046), `TestComboStateResetOnChange` (PR-648, keyed by definition hash), `-race`. (b) `ResolveCombo(name)` ordered targets w/ recursion guard; strategies read from settings keys above; fallback uses w4-d `WithAccountFallback` per model; round-robin rotation = in-memory `map[string]int` reset on restart (matches ref Map, NO TTL); transient cooldown wait ≤5s; earliest retry-after aggregation; rotation state reset when the combo's models/definition hash changes (PR-648).
3. **Name validation + /v1/models promotion** (`internal/admin/combos.go` handler + `internal/api/models.go` TOUCH). (a) `TestComboNameValidation` (valid/invalid sets — the API-route rule PAR-ROUTE-004: regex `/^[a-zA-Z0-9_.\-]+$/` — alphanumeric, underscore, DOT, hyphen (`src/app/api/combos/route.js:7`)), `TestModelsListCombosFirst` (047). (b) combo CRUD API validates names against `^[a-zA-Z0-9_.\-]+$`; /v1/models lists combo names first.

## Preconditions
- `grep -c 'func SelectConnection\|func WithAccountFallback' internal/inference/selection.go` ≥ 1 (w4-d merged).
- `grep -rn 'combo' internal/ --include='*.go'` (non-test) → 0 hits.

## Exclusive file ownership
NEW: `internal/store/combos.go`+test, `internal/inference/combo.go`+test, `internal/admin/combos.go`+test. TOUCH: `internal/store/migrate.go`, `internal/store/settings.go` (combo* keys — AFTER w4-d merges, serial on settings.go), `internal/api/models.go`+test (promotion only), `internal/server/routes_admin.go` (combo routes).

## Binary acceptance
- `go test ./... && go vet ./... && go test -race ./internal/inference/` green.
- TestComboRecursionGuard, TestComboStateResetOnChange, TestComboTransientCooldownCap5s, TestModelsListCombosFirst pass; sticky default-1 + settings-based strategy storage asserted; combos table has NO strategy column.

## Out of scope
Combo UI + PAR-PR-339 (Wave 6). Per-key quotas (W5). Handler dispatch of combos into the chat path (w4-f).


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
