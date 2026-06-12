# w5-a — Usage schema + pricing engine

PAR rows: PAR-USAGE-001/002/003/004 (tables only — write/read semantics land in
w5-b/c/d), 005, 006, 007, 008, 009, 010, 040. NOT in scope: saveRequestUsage
transaction (011/012 → w5-b), observability writer (024-028 → w5-c), all usage/pricing
HTTP routes (013-017/021-023/029-031 → w5-d/e), handler glue (→ w5-f).
Frozen ref @ 827e5c3. Depends: w5-pre merged. Runs ALONE (owns `migrate.go`, hot file).

## Architectural decisions
- AGENTS.md: **usage data lives in `request_log`** — 9router `usageHistory`
  (`src/lib/db/schema.js:105-127`) ports as `request_log`, snake_case columns.
- Timestamps in `request_log`/`request_details` are ISO-8601 TEXT (not epoch INTEGER
  like older tables): the ref's read paths depend on lexicographic string ranges and
  minute-prefix slicing (`usageRepo.js:231,359`, `requestDetailsRepo.js:112`); porting
  behavior, not storage style.
- Pricing engine lives in NEW package `internal/usage` (domain layer, no store import);
  user overrides reach it through a small interface implemented by the store's kv table
  (DDD layering per arch test).

## Tasks

1. **Migrations** — evidence: `src/lib/db/schema.js:96-150` (kv, usageHistory,
   usageDaily, requestDetails + indexes); in-tree pattern `internal/store/migrate.go:15-107`.
   STEP (a): extend `TestMigrate` (or add `TestMigrateUsageTables`) asserting
   `request_log`, `usage_daily`, `request_details`, `kv` exist with expected columns
   (PRAGMA table_info) and indexes (`idx_request_log_timestamp` DESC, `_provider`,
   `_model`, `_connection_id`; same four for `request_details`); run — fails.
   STEP (b): append to the `tables` slice (additive-only):
   `request_log` (id INTEGER PRIMARY KEY AUTOINCREMENT, timestamp TEXT NOT NULL,
   provider TEXT, model TEXT, connection_id TEXT, api_key TEXT, endpoint TEXT,
   prompt_tokens INTEGER NOT NULL DEFAULT 0, completion_tokens INTEGER NOT NULL DEFAULT 0,
   cost REAL NOT NULL DEFAULT 0, status TEXT, tokens TEXT NOT NULL DEFAULT '{}',
   meta TEXT NOT NULL DEFAULT '{}');
   `usage_daily` (date_key TEXT PRIMARY KEY, data TEXT NOT NULL);
   `request_details` (id TEXT PRIMARY KEY, timestamp TEXT NOT NULL, provider TEXT,
   model TEXT, connection_id TEXT, status TEXT, data TEXT NOT NULL);
   `kv` (scope TEXT NOT NULL, key TEXT NOT NULL, value TEXT NOT NULL,
   PRIMARY KEY (scope, key)); plus the 8 indexes.

2. **kv store accessors** — evidence: `src/lib/db/helpers/kvStore.js` usage via
   `pricingRepo.js:5` (`makeKv("pricing")` → getAll/clear) and direct kv SQL at
   `pricingRepo.js:64-98`.
   STEP (a): `TestKVRoundTrip` (Set/Get/List by scope/Delete/ClearScope; unknown key →
   sql.ErrNoRows-mapped sentinel) — fails.
   STEP (b): NEW `internal/store/kv.go`: `SetKV(scope, key, value string)` (upsert),
   `GetKV(scope, key)`, `ListKV(scope) (map[string]string, error)`, `DeleteKV(scope, key)`,
   `ClearKVScope(scope)`. Match neighbor style (`internal/store/settings.go`).

3. **Pricing data tables** — evidence: `src/shared/constants/pricing.js:12-117`
   (MODEL_PRICING ~100 entries), `:124-129` (PROVIDER_PRICING: gh/gpt-5.3-codex
   override), `:136-207` (PATTERN_PRICING ~45 ordered patterns, first match wins).
   STEP (a): `TestPricingDataParity` spot-checks ≥10 known entries across families
   (claude-opus-4-6 input=5.00/cache_creation=6.25; deepseek-chat cached=0.0028;
   gh override gpt-5.3-codex input=1.75; pattern "*-codex-xhigh" input=10.00; counts:
   len(ModelPricing) and len(PatternPricing) equal the ref's entry counts) — fails.
   STEP (b): NEW `internal/usage/pricingdata.go`: `Pricing{Input, Output, Cached,
   Reasoning, CacheCreation float64}`; `ModelPricing map[string]Pricing`,
   `ProviderPricing map[string]map[string]Pricing`, `PatternPricing []PatternPrice`
   ported VERBATIM in ref order (package-level vars, no init()).

4. **Glob matcher + 3-step resolution** — evidence: `pricing.js:212-248` (matchPattern:
   `*`→`.*`, regex-quote the rest, anchored ^$; resolution: provider override → canonical
   by baseModel (strip `vendor/` prefix via last `/` segment) then full model → first
   pattern matching baseModel OR model; nil when nothing matches), `pricingRepo.js:51-57`
   (user kv override checked BEFORE constants).
   STEP (a): table-driven `TestMatchPattern` (`*-codex-xhigh` matches
   `gpt-5.3-codex-xhigh`, not `gpt-5.3-codex-high`; `gemini-*-flash-lite` ordering;
   literal dots not regex-active) and `TestResolvePricing` (provider override wins;
   `deepseek/deepseek-chat` → strips to `deepseek-chat`; pattern fallback order; unknown
   → nil,false) — fails.
   STEP (b): `internal/usage/pricing.go`: `MatchPattern(pattern, model string) bool`,
   `ResolvePricing(provider, model string) (Pricing, bool)`; `Resolver` struct holding an
   `OverrideStore` interface (`UserPricing() (map[string]map[string]Pricing, error)`)
   with method `PricingForModel(provider, model)` = user override → ResolvePricing.

5. **Merged pricing view + 5s cache (040, 004)** — evidence: `pricingRepo.js:6-49`
   (cache TTL 5000ms; merge PROVIDER_PRICING with user kv per provider per model —
   user fields overlay shallowly; user-only providers included), `:60-108`
   (updatePricing per-provider read-modify-write in one tx; resetPricing per-provider /
   per-model / all; every mutation invalidates the cache).
   STEP (a): `TestMergedPricingAndCache` (injected clock: second call within TTL hits
   cache — count store reads via fake; update invalidates; merge semantics: user override
   on gh model overlays field-wise, new provider appears) — fails.
   STEP (b): implement on `Resolver`: `Merged() (map[string]map[string]Pricing, error)`,
   `Update(data map[string]map[string]Pricing)`, `Reset(provider, model string)`,
   `ResetAll()`; cache `{value, expiresAt}` guarded by mutex, clock injected via struct
   field (no global state). Store-side: kv rows scope='pricing', key=provider,
   value=JSON model→Pricing map.

6. **Token normalization + cost calculation (009, 010)** — evidence:
   `pricing.js:274-303` / `usageRepo.js:113-151`: synonyms prompt_tokens|input_tokens,
   completion_tokens|output_tokens, cached_tokens|cache_read_input_tokens; cached
   subtracted from input then billed at cached-or-input rate; reasoning falls back to
   output rate; cache_creation_input_tokens falls back to input rate; all rates per 1M;
   zero/absent pricing → cost 0.
   STEP (a): golden-value `TestCalculateCost` (e.g. claude-sonnet-4-6: 1M in, 200k cached,
   100k out, 50k reasoning, 10k cache_creation → hand-computed dollars; synonym-field
   inputs produce identical cost; nil tokens → 0) — fails.
   STEP (b): `internal/usage/tokens.go`: `TokenSet` struct + `NormalizeTokens(map[string]
   int64) TokenSet` accepting both synonym sets; `internal/usage/cost.go`:
   `CalculateCost(tokens TokenSet, p Pricing) float64` and
   `(r *Resolver) CostFor(provider, model string, tokens TokenSet) float64` (0 when no
   pricing resolves, mirroring `usageRepo.js:114,118`).

## Preconditions (each states its own pass condition)
- `grep -c 'request_log' internal/store/migrate.go` outputs `0` (the gap; acceptance flips ≥1).
- `ls internal/usage/ 2>/dev/null | wc -l` outputs `0` (package is new).
- `grep -c 'ensureColumn' internal/store/migrate.go` ≥ 1 (additive pattern exists; follow it).

## Exclusive file ownership
TOUCH: `internal/store/migrate.go`(+test). NEW: `internal/store/kv.go`(+test),
`internal/usage/{doc.go,pricingdata.go,pricing.go,tokens.go,cost.go}`(+tests).
Runs ALONE (migrate.go is hot; w5-b/c both add store files next).

## Binary acceptance
- `go build ./... && go vet ./...` clean; `go test ./...` green; `go test -race ./internal/store/ ./internal/usage/` green.
- `sqlite3` against a freshly-migrated DB: `.tables` lists request_log, usage_daily, request_details, kv.
- `grep -c 'init()' internal/usage/*.go` → 0; `grep -rc 'bloodf/g0router/internal/store' internal/usage/*.go | grep -v ':0'` → empty (layering: usage does not import store).
- TestPricingDataParity, TestResolvePricing, TestMergedPricingAndCache, TestCalculateCost pass.

## Out of scope
saveRequestUsage/daily rollup writes (w5-b). Buffered request-details writer (w5-c).
All admin routes incl. /api/pricing (w5-d). SSE (w5-e). Handler glue (w5-f). VK (w5-g).
SQLite JSON-function rollups (post-parity backlog).
