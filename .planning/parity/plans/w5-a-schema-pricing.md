# w5-a — Usage schema + pricing engine

PAR rows: PAR-USAGE-001/002/003/004 (tables only — write/read semantics land in
w5-b/c/d), 005, 006, 007, 008, 009, 010, 040. NOT in scope: saveRequestUsage
transaction (011/012 → w5-b), observability writer (024-028 → w5-c), all usage/pricing
HTTP routes (013-017/021-023/029-031 → w5-d/e), pricing mutation/reset repo ops
(`updatePricing`/`resetPricing`/`resetAllPricing`, `pricingRepo.js:60-108` — port in
w5-d with PAR-USAGE-030/031; w5-a exposes only the `Invalidate()` hook they will call),
handler glue (→ w5-f).
Frozen ref @ 827e5c3. Depends: w5-pre merged. Runs ALONE (owns `migrate.go`, hot file).

## Architectural decisions
- `AGENTS.md:27` "Usage data lives in the `request_log` table" — 9router `usageHistory`
  (`src/lib/db/schema.js:105-127`) ports as `request_log`, snake_case columns.
- Timestamps in `request_log`/`request_details` are ISO-8601 TEXT (not epoch INTEGER
  like older tables): the ref's read paths depend on lexicographic string ranges and
  minute-prefix slicing (`usageRepo.js:231,359`, `requestDetailsRepo.js:112`); porting
  behavior, not storage style.
- Pricing engine lives in NEW package `internal/usage` (domain layer, no store import);
  user overrides reach it through a small interface implemented by the store's kv table.
  Mandated by `AGENTS.md:24` "Layered DDD architecture (transport→domain→repository)":
  domain (usage) may not import repository (store), and the in-repo precedent is the
  interface-seam pattern of `internal/api/models.go` (`ComboLister`/`DisabledChecker`,
  w4-c/e) — the interface is the MINIMAL mechanism, not new infrastructure.

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

2. **kv store accessors (READ path of PAR-USAGE-004 only)** — evidence:
   PAR-USAGE-004 "kv table with scope='pricing' stores user pricing overrides"
   (`src/lib/db/schema.js:96-104`); the read path consumes them via
   `pricingRepo.js:14-16` (`pricingKv.getAll()`); storing requires an upsert
   (`pricingRepo.js:71` INSERT ... ON CONFLICT — cited here ONLY to fix the upsert
   SQL shape, not to port mutation flows).
   STEP (a): `TestKVRoundTrip` (SetKV/GetKV/ListKV by scope; unknown key behaves
   exactly like the neighbor convention `internal/store/settings.go:33-40`
   `GetSetting` — `sql.ErrNoRows` maps to ("", nil), missing is not an error) — fails.
   STEP (b): NEW `internal/store/kv.go`: `SetKV(scope, key, value string)` (upsert),
   `GetKV(scope, key)`, `ListKV(scope) (map[string]string, error)`. NO DeleteKV /
   ClearKVScope — those exist only for pricing reset (PAR-USAGE-031) and port in
   w5-d. Match neighbor style (`internal/store/settings.go`).

3. **Pricing data tables** — evidence: `src/shared/constants/pricing.js:12-117`
   (MODEL_PRICING, exactly **83** entries — counted against the frozen ref), `:124-129`
   (PROVIDER_PRICING: 1 provider, gh/gpt-5.3-codex override), `:136-207`
   (PATTERN_PRICING, exactly **49** ordered patterns, first match wins).
   STEP (a): `TestPricingDataParity` asserts EXACTLY these ten entries (full 5-field
   golden values from the frozen ref) — `claude-opus-4-6` {5.00, 25.00, 0.50, 25.00,
   6.25}; `claude-sonnet-4-6` {3.00, 15.00, 0.30, 15.00, 3.75}; `gpt-4o-mini` {0.15,
   0.60, 0.075, 0.90, 0.15}; `gpt-5.3-codex-spark` {3.00, 12.00, 0.30, 12.00, 3.00};
   `gemini-2.5-flash-lite` {0.15, 1.25, 0.015, 1.875, 0.15}; `deepseek-chat` {0.14,
   0.28, 0.0028, 0.28, 0.14}; `kimi-k2.5-thinking` {1.80, 7.20, 0.90, 10.80, 1.80};
   `glm-4.6v` {0.75, 3.00, 0.375, 4.50, 0.75}; `MiniMax-M3` {0.30, 1.20, 0.06, 1.80,
   0.30}; `auto` {2.00, 8.00, 1.00, 12.00, 2.00} — plus provider override
   `ProviderPricing["gh"]["gpt-5.3-codex"]` {1.75, 14.00, 0.175, 14.00, 1.75}, plus
   first/last pattern anchors `PatternPricing[0].Pattern == "*-codex-xhigh"` (input
   10.00) and `PatternPricing[48].Pattern == "grok-*"` (input 0.50), AND the binary
   counts `len(ModelPricing) == 83`, `len(PatternPricing) == 49`,
   `len(ProviderPricing) == 1` — fails.
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
   `ResolvePricing(provider, model string) (Pricing, bool)` (pure, constants-only —
   this alone closes PAR-USAGE-008's 3-step chain). The user-override-FIRST step is
   row evidence too: PAR-USAGE-004's overrides are consumed at resolution by
   `pricingRepo.js:51-56` (`getPricingForModel`: user kv hit returns before the
   constants chain) — so a `Resolver` struct holds the override source behind an
   `OverrideStore` interface (`UserPricing() (map[string]map[string]map[string]float64,
   error)` — provider → model → rate-name → value; present keys only) and exposes
   `PricingForModel(provider, model)` = user override → ResolvePricing. The interface
   is the layering-mandated seam (`AGENTS.md:24`; precedent `internal/api/models.go`
   ComboLister), not new infrastructure. A per-model user override is returned
   VERBATIM, not overlaid onto canonical (`pricingRepo.js:51-56` returns the raw user
   object); absent rate keys resolve to 0 in Go (the ref's undefined-arithmetic NaN is
   a JS artifact with no parity value — recorded adaptation).

5. **Merged pricing view + 5s cache (040, 004 — READ side only)** — evidence:
   `pricingRepo.js:6-49` (cache TTL 5000ms; merge PROVIDER_PRICING with user kv per
   provider per model — user fields overlay field-wise via JS object spread `:30-32`;
   user-only providers included `:37-45`). Mutation/reset (`updatePricing`,
   `resetPricing`, `resetAllPricing`, `pricingRepo.js:60-108`) belongs to
   PAR-USAGE-030/031 and ports in **w5-d** with the pricing routes; w5-a ships only
   the read path plus an exported `Invalidate()` hook w5-d's mutations will call.
   Field-absence semantics (binary): user override rows are stored/parsed as
   `map[string]float64` keyed by rate name — a PRESENT key overrides that field, an
   ABSENT key inherits the canonical value (exactly the JS spread semantics; a Go
   zero-value cannot be conflated with absent because absent keys never enter the map).
   STEP (a): `TestMergedPricingAndCache` (injected clock: second call within TTL does
   not re-read store — count reads via fake OverrideStore; after TTL expiry re-reads;
   `Invalidate()` forces re-read; merge: user kv sets ONLY `input` on gh/gpt-5.3-codex
   → merged input is the user value AND output stays 14.00 canonical; user kv adds a
   provider absent from constants → it appears) — fails.
   STEP (b): implement on `Resolver`: `Merged() (map[string]map[string]Pricing, error)`
   + `Invalidate()`; cache `{value, expiresAt}` guarded by mutex, clock injected via
   struct field (no global state). Store-side read: kv rows scope='pricing',
   key=provider, value=JSON model→map[rate]float64.

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

## Plan-gate disposition (cycle 3, Fable 5, 2026-06-12) — CLOSED BY DECISION
Three substantive cycles complete. Cycle-1 findings FIXED (mutation ops moved to w5-d
with Invalidate() hook retained; kv sentinel tied to settings.go:33-40 convention;
exact counts 83/49/1; overlay field-absence semantics defined as present-key maps).
Cycle-2 findings FIXED (kv scoped to Set/Get/List, Delete/Clear → w5-d; ten golden
entries + first/last pattern anchors enumerated; layering cited to AGENTS.md:24 +
models.go ComboLister precedent). Cycle-3 residual findings triaged:

BLOCKER "SetKV is write behavior beyond read-path scope": FALSE POSITIVE.
PAR-USAGE-004 reads "kv table … STORES user pricing overrides" — storage requires the
write primitive. SetKV is a single-row upsert with no pricing semantics (no merge, no
validation, no route); the pricing MUTATION FLOWS (updatePricing/resetPricing/
resetAllPricing, pricingRepo.js:60-108) remain in w5-d. Cross-plan necessity: w5-b's
lifetime counter (PAR-USAGE-011, usageRepo.js:276-279 `_meta` upsert) writes kv
scope='meta' inside its save transaction — the primitive must exist before w5-b.

MAJOR "Resolver/user-override exceeds PAR-USAGE-008": FALSE POSITIVE. The
user-override-first read (pricingRepo.js:51-56) is the consumption half of
PAR-USAGE-004's evidence — a stored override that is never consulted is dead storage,
and no other w5 plan owns resolution (w5-d owns routes+mutations only; w5-b's
PAR-USAGE-012 calls `calculateCost(provider, model, tokens)` which resolves through
pricingRepo.getPricingForModel — the exact chain this task ports). Omitting it from
w5-a would make w5-b's cost-at-save unimplementable.

MAJOR "Merged view maps to PAR-USAGE-029": FALSE POSITIVE (half). PAR-USAGE-040
("Pricing cache TTL 5s … to avoid repeated DB reads", pricingRepo.js:6-12) caches
EXACTLY the merged view built by getPricing (pricingRepo.js:18-49) — the cache cannot
be ported without the function it caches. PAR-USAGE-029 is the HTTP ROUTE serving
this view; the route stays in w5-d and will consume w5-a's Merged().

Kimi diff gate at implementation (full source context) is the binding check.
APPROVED BY DECISION for dispatch after w5-pre merges.

## Diff-gate cycle-2 artifact note (Fable 5, 2026-06-12)
The first cycle-2 run was DISCARDED as a harness-scoping artifact: the path filter
`internal/usage/` swept w5-b's interleaved commits (recorder/events/recent/ring/
tracker — gated separately under w5-b) into the w5-a diff, and every "finding" was
about those files. Re-run with exact w5-a file paths only (migrate, kv, doc,
pricingdata, pricing, tokens, cost + tests). Cycle-1 REAL findings were fixed in
fixes/w5-a-fix-r1.md (commit b6089ab): Store.UserPricing() kv reader, wrapped
resolver errors, exact-match user overrides.
