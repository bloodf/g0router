# w5-d — Usage read APIs + pricing routes

PAR rows: PAR-USAGE-013, 014, 015, 016, 017, 021, 022, 023, 024 (route half — query
landed in w5-c), 029, 030, 031, 039. NOT in scope: SSE stream (034/035 → w5-e),
provider quota API (032/033 → w5-e), INFERENCE-pipeline usage capture
(chat/messages/embeddings glue → w5-f — distinct from the ADMIN HTTP routes, which
are this plan's explicit deliverable: every row above lists its route half), UI
components (036/037 → W6).
Frozen ref @ 827e5c3. Depends: w5-a + w5-b + w5-c merged. Serial: w5-e edits
`internal/server/routes_admin.go` AFTER this plan merges.

Ref route inventory (verified against frozen tree, `src/app/api/`):
`/api/usage/stats`, `/api/usage/chart`, `/api/usage/request-logs` (getRecentLogs 200,
`request-logs/route.js:4-13` — the path PAR-USAGE-037 cites), `/api/usage/logs`
(IDENTICAL body — also getRecentLogs(200), `logs/route.js:4-12`; REQUIRED by the
matrix itself: `matrix/9router-usage.md:125` Go-port considerations list
`/api/usage/logs` among the admin routes to add — both names register the one
handler), `/api/usage/request-details` (filters+pagination, pageSize cap 100),
`/api/pricing` (GET/PATCH/DELETE). EXCLUDED with evidence: `/api/usage/history`
(`history/route.js:4-12` is a bare getUsageStats() duplicate of the stats route with
no PAR row and no matrix Go-port mention — not ported); `/api/usage/providers`
(`providers/route.js:1-40` builds a provider-dropdown list for the RequestDetails UI
panel from request-details rows — a UI filter helper with no PAR row; ships with the
W6 UI wave that renders that panel); `/api/usage/{connectionId}` (PAR-USAGE-032/033
→ w5-e).

## Tasks

1. **Stats service: period selection + dual read path (PAR-USAGE-013, 014, 015, 016)**
   — evidence: `src/app/api/usage/stats/route.js:4` (periods today/24h/7d/30d/60d/all);
   `usageRepo.js:418` (`useDailySummary = period !== "24h" && period !== "today"`);
   daily path `usageRepo.js:420-503` (sums day JSON; byModel key `rawModel (provider)`,
   byAccount key `model (provider - accountName)`, byApiKey keyed
   `apiKey|model|provider` with keyName fallback `apiKey[:8]...` / "Local (No API
   Key)", byEndpoint `endpoint|model|provider`; provider display names mapped through
   provider-node names); live path `usageRepo.js:531-614` (24h/today reads request_log
   rows, same key shapes, lastUsed = max timestamp); totals `usageRepo.js:367-376,616`
   (totalRequests = Σ byProvider.requests). The stats payload's
   pending/activeRequests/recentRequests/errorProvider fields are PART OF
   PAR-USAGE-015's cited evidence (the stats literal at `usageRepo.js:367-376`,
   lines 372-375) — this plan CONSUMES them read-only through w5-b's existing
   `Tracker` snapshot / `Ring` + `DedupeRecent` APIs; the PAR-USAGE-018/019
   implementations remain w5-b's, untouched here.
   STEP (a): store-level FIRST — `TestLoadDailyRange` (seed 4 usage_daily rows;
   maxDays=2 → only rows at/after the dateKey cutoff; nil maxDays → all) and
   `TestRangeRequestLogs` (ISO window bounds inclusive per `usageRepo.js:403-405`)
   in `internal/store/requestlog_test.go`; then service-level
   `TestUsageStatsDailyPath` (seed usage_daily 2 days + request_log overlay
   rows → 7d stats: totals, all five breakdowns' key shapes, provider display-name
   mapping) and `TestUsageStatsLivePath` (today/24h reads request_log only; cutoff =
   start-of-day vs now-24h) — all fail before implementation.
   STEP (b): NEW `internal/usage/stats.go`: `StatsService` over interfaces
   `UsageReader` (daily range + log range reads — implemented by store),
   `NameSource` (connection map — w5-b ConnNameCache; provider id→name; api-key
   key→name), and the w5-b `Tracker`/`Ring`. Store side (serial AFTER w5-b — same
   file, no live concurrency): add to `internal/store/requestlog.go`:
   `LoadDailyRange(maxDays int)` (dateKey cutoff per `usageRepo.js:309-317`),
   `RangeRequestLogs(sinceISO, untilISO string)`.

2. **last10Minutes buckets (PAR-USAGE-017)** — evidence: `usageRepo.js:393-416`:
   10 one-minute buckets ending at the current minute start; rows bucketed by
   floor(ts/60s); requests/promptTokens/completionTokens/cost per bucket.
   STEP (a): `TestLast10MinuteBuckets` (injected clock; rows at -1m, -9m, -11m → the
   -11m row excluded, others in correct buckets) — fails.
   STEP (b): bucket builder in `stats.go` using the injected clock + a window query.

3. **lastUsed overlay (PAR-USAGE-039)** — evidence: `usageRepo.js:505-530`: after
   daily aggregation, request_log rows within the period overlay PRECISE lastUsed
   timestamps onto byModel/byAccount/byApiKey/byEndpoint entries (only when newer).
   STEP (a): `TestLastUsedOverlay` (daily rows give date-key lastUsed; a request_log
   row same day 14:33 → byModel lastUsed becomes the precise ISO timestamp) — fails.
   STEP (b): overlay pass in `stats.go` daily path.

4. **Chart data (PAR-USAGE-021, 022)** — evidence: `usageRepo.js:620-691`: today →
   24 hourly buckets from start-of-day (rows outside [start, start+24h) skipped);
   24h → 24 hourly buckets ending now (idx clamped to last); 7d/30d/60d → N daily
   buckets from usage_daily, missing days zero-filled; labels: hours "15:04"-style
   en-US 24h clock, days short "Jan 2" (`:631,653,673` — Go `time.Format("15:04")` /
   `("Jan 2")`, recorded as the locale-string adaptation); tokens = prompt+completion.
   STEP (a): `TestChartToday`, `TestChart24hClamp`, `TestChartDailyZeroFill` (injected
   clock; golden labels + bucket sums) — fail.
   STEP (b): `internal/usage/chart.go`: `ChartData(period)` on StatsService deps.

5. **Derived request logs (PAR-USAGE-023)** — evidence: `usageRepo.js:693-731`:
   reads last N (route passes 200) request_log rows DESC; line format
   `DD-MM-YYYY HH:MM:SS | model | PROVIDER | account | sent | received | status`
   with connection-name map and `connectionId[:8]` fallback, "-" placeholders.
   STEP (a): `TestRecentLogsFormat` (seeded rows → exact golden lines incl. fallbacks)
   — fails.
   STEP (b): `internal/usage/logs.go`: `RecentLogs(limit) []string` over
   `ListRecentRequestLogs` (w5-b) + conn map.

6. **Pricing mutations (PAR-USAGE-030/031 repo half)** — evidence:
   `pricingRepo.js:60-78` (updatePricing: per-provider read-modify-write merging
   model entries, upsert kv, invalidate cache), `:80-102` (resetPricing: no model →
   delete provider key; with model → remove model from provider JSON, delete key when
   empty), `:104-108` (resetAllPricing: clear scope). Store needs the w5-d-only
   additions `DeleteKV(scope, key)` + `ClearKVScope(scope)` (explicitly deferred out
   of w5-a by its cycle-2 disposition). Adaptation (recorded): per-provider
   read-modify-write runs under a Resolver-level mutex instead of a multi-statement
   SQLite tx — the ref's tx exists for multi-provider atomicity in a sync driver;
   single-upsert-per-provider is already atomic, the mutex serializes the read-modify
   step.
   STEP (a): `TestUpdatePricingMergesPerProvider` (existing user model kept, new model
   added, cache invalidated → next Merged() reflects), `TestResetPricing` (provider /
   provider+model / empties→delete key), `TestResetAllPricing`, `TestKVDeleteAndClear`
   (store level) — fail.
   STEP (b): add `DeleteKV`/`ClearKVScope` to `internal/store/kv.go`; add
   `Update(map[string]map[string]map[string]float64) error`, `Reset(provider, model
   string) error`, `ResetAll() error` to `internal/usage` Resolver (mutex +
   `Invalidate()` from w5-a).

7. **HTTP routes** (route halves of 013-017/021-023/024/029/030/031) — evidence:
   `src/app/api/usage/stats/route.js:4-13` (`?period=` default "all"),
   `src/app/api/usage/chart/route.js:4-13` (`?period=` default "7d"),
   `src/app/api/usage/request-logs/route.js:4-13` (fixed limit 200),
   `src/app/api/usage/request-details/route.js:8-60` (page ≥1, pageSize 1-100
   → 400 otherwise; passes 6 filters), `src/app/api/pricing/route.js:9-20` (GET
   merged user+default), `:27-83` (PATCH body {provider: {model: {rates}}}; rejects
   non-object, unknown rate fields, negative/non-number values → 400 with message;
   returns updated user pricing), `:91-117` (DELETE ?provider=&model=; no provider →
   reset all).
   STEP (a): handler tests `TestUsageStatsRoute` (period plumbed; envelope shape),
   `TestRequestDetailsRouteValidation` (page=0 → 400; pageSize=101 → 400),
   `TestPricingPatchValidation` (unknown field "foo" → 400; negative → 400; valid →
   200 + persisted), `TestPricingDelete` (provider, provider+model, all) — fail.
   STEP (b): NEW `internal/admin/usage.go` + `internal/admin/pricing.go` handlers on
   `*Handlers` using the `{data, error}` envelope (`internal/admin/respond.go:10-17`)
   and snake_case JSON (AGENTS.md); register in `internal/server/routes_admin.go`
   under `RequireSession`: GET `/api/usage/stats`, `/api/usage/chart`,
   `/api/usage/request-logs` AND `/api/usage/logs` (one handler, two registrations —
   `matrix/9router-usage.md:125` requires `/api/usage/logs`; PAR-USAGE-037 cites
   `request-logs`), `/api/usage/request-details`; GET+PATCH+DELETE `/api/pricing`. Wire StatsService/Resolver construction in
   `internal/server/routes_admin.go`'s handler bootstrap (follow `NewAdminHandlers`,
   `routes_admin.go:15-23`).

## Preconditions (each states its own pass condition)
- `grep -c 'func (s \*Store) SaveUsage' internal/store/requestlog.go` ≥ 1 (w5-b merged).
- `grep -c 'func (s \*Store) QueryRequestDetails' internal/store/requestdetails.go` ≥ 1 (w5-c merged).
- `grep -c 'Invalidate' internal/usage/pricing.go` ≥ 1 (w5-a hook present).
- `grep -rc '/api/usage' internal/server/routes_admin.go` outputs `0` (routes are the gap; flips ≥1).

## Exclusive file ownership
NEW: `internal/usage/{stats,chart,logs}.go`(+tests), `internal/admin/usage.go`(+test),
`internal/admin/pricing.go`(+test). TOUCH: `internal/store/requestlog.go`(+test — read
queries; serial after w5-b), `internal/store/kv.go`(+test — Delete/Clear; serial after
w5-a), `internal/usage/pricing.go`(+test — mutations; serial after w5-a),
`internal/server/routes_admin.go`(+test). w5-e waits for this merge before touching
routes_admin.go; w5-f owns internal/api (disjoint).

## Binary acceptance
- `go build ./... && go vet ./...` clean; `go test ./...` green; `go test -race ./internal/usage/ ./internal/admin/ ./internal/store/` green.
- `grep -c '/api/usage/stats\|/api/usage/chart\|/api/usage/request-logs\|/api/usage/logs\|/api/usage/request-details' internal/server/routes_admin.go` ≥ 5; method-explicit registration checks: `grep -c 'r.GET("/api/pricing"' internal/server/routes_admin.go` = 1, `grep -c 'r.PATCH("/api/pricing"' ...` = 1, `grep -c 'r.DELETE("/api/pricing"' ...` = 1.
- ALL named tests pass (complete list): TestLoadDailyRange, TestRangeRequestLogs,
  TestUsageStatsDailyPath, TestUsageStatsLivePath, TestLast10MinuteBuckets,
  TestLastUsedOverlay, TestChartToday, TestChart24hClamp, TestChartDailyZeroFill,
  TestRecentLogsFormat, TestUpdatePricingMergesPerProvider, TestResetPricing,
  TestResetAllPricing, TestKVDeleteAndClear, TestUsageStatsRoute,
  TestRequestDetailsRouteValidation, TestPricingPatchValidation, TestPricingDelete.

## Out of scope
SSE `/api/usage/stream` + per-connection provider quota (w5-e). Wiring usage capture
into chat/messages handlers (w5-f). UI (W6). `/api/usage/history` and
`/api/usage/providers` (excluded with evidence in §Ref route inventory; NOTE
`/api/usage/logs` IS in scope — required by `matrix/9router-usage.md:125`).

## Plan-gate disposition (cycle 3, Fable 5, 2026-06-12) — CLOSED BY DECISION
Three substantive cycles complete. Cycle-1 findings FIXED (scope terminology
"inference-pipeline glue" vs admin routes; stats live-fields anchored to
PAR-USAGE-015's evidence lines 372-375; route exclusions evidenced; /api/usage/logs
included per matrix Go-port directive 9router-usage.md:125). Cycle-2 findings FIXED
(Out-of-scope contradiction removed; history alias dropped; store-level test-first
queries added; method-explicit pricing-route acceptance). Cycle-3 residual triage:
- BLOCKER "/api/usage/request-logs not authorized": FALSE POSITIVE. PAR-USAGE-037
  defers the W6 REACT COMPONENT, not the API it polls; w5-d's charter is exactly the
  route halves whose UI consumers come later (same split as every row in this plan).
  The ref serves BOTH paths with one function (request-logs/route.js:4-13 ≡
  logs/route.js:4-12); registering both names of one handler adds zero scope. Note
  the cycle-2/cycle-3 findings are mutually contradictory (cycle 2 attacked the
  logs alias as unauthorized; cycle 3 attacks request-logs while accepting logs) —
  a gate-stability artifact, not a plan defect.
- MAJOR ownership crossing: FIXED at source — WAVE-5-MAP §Ownership now enumerates
  the w5-d serial grants (kv Delete/Clear, pricing mutations, requestlog read
  queries) recorded in w5-a's cycle-2 disposition. All serial-after-merge.
- MAJOR missing named tests: REAL → FIXED (complete 18-test list in acceptance).
Kimi diff gate at implementation is the binding check. APPROVED BY DECISION for
dispatch after w5-b + w5-c merge.
