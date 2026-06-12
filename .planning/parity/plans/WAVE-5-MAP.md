# Wave 5 — Usage, cost, request logging, virtual keys: micro-plan index (Stage-1 scope)

Author: Fable 5. Orchestrator: Sonnet/Fable session. Implementers: kimi. Gates: gpt-5.5.
**Non-authorizing INDEX** (like WAVE-2/3/4-MAP). Frozen ref @ 827e5c3. Depends on
Waves 0–4 — COMPLETE. Matrices: `matrix/9router-usage.md` (40 rows, all MISSING),
plus deferred rows PAR-ROUTE-030/031/054, PAR-TRANS-046 (usage clause),
PAR-AUTH-017/018, and the W4 carry-forward debts.

## Architectural decision (AGENTS.md)

**"Usage data lives in the `request_log` table."** The 9router `usageHistory` table
ports as g0router `request_log` (snake_case columns, additive migration). Companions:
`usage_daily` (per-day rollup JSON, PAR-USAGE-002), `request_details` (observability
records, PAR-USAGE-003), `kv` (scope='pricing' user overrides, PAR-USAGE-004).
PAR-ROUTE-054 (request logging with attribution) is satisfied by `request_log` rows +
the derived-log read path (PAR-USAGE-023 — ref derives logs from usageHistory,
`usageRepo.js:698-731`); no separate log store exists in the ref either.

## Stage-1 scope decisions

- IN (36 of 40 PAR-USAGE rows): 001-031, 034, 035, 038, 039, 040.
- PARTIAL Stage-1 (2): 032/033 provider-quota API — ref dispatcher
  (`open-sse/services/usage.js:60-101`) covers gh/gemini/antigravity/claude/codex/
  kiro/glm/minimax; Stage-1 ships the dispatcher + **claude** (`usage.js:497-614`,
  OAuth-first → legacy fallback) and **gemini** (`usage.js:225-342`, + subscription
  info) fetchers — the only Stage-1 providers with OAuth flows (W3 shipped
  anthropic/gemini/xai; xai has no usage endpoint in the ref). gh/antigravity/codex/
  kiro/glm/minimax fetchers defer to Stage 2 with their providers/OAuth.
- DEFERRED → Wave 6 (UI wave): 036 (UsageStats component), 037 (RequestLogger
  component) — both are dashboard React components; their APIs land here.
- Deferred-in rows landing now: PAR-ROUTE-054 (W4 ledger), PAR-ROUTE-030/031
  (W4 ledger; 032 schema already HAVE), PAR-TRANS-046 usage clause (W1 ledger),
  PAR-AUTH-017/018 (W3 ledger — sanitizeHeaders ports with request_details;
  debug-log prod gate ports with the logging utility).
- W4 carry-forward debts closed here (w5-pre): SetCredentialRefresher production
  caller (real OAuth token-exchange `RefreshCredentials(connectionID)`), and
  ErrModelTransient production wrapping of 502/503/504 (w4-e deferral).
- Still deferred: GetTestByKind live model pinging → Wave 6 (admin/dashboard wave,
  per w4-f disposition). MITM proxy half → S2/W7. Tunnels → W7.

## Micro-plan index (7 plans)

| Plan | Scope | Rows | Key ref evidence | Depends |
|---|---|---|---|---|
| **w5-pre** | Carry-forward debts: production `CredentialRefresher` (OAuth token-exchange via `OAuthFlow.Refresh` + persist rotated tokens to connections; wire `SetCredentialRefresher` in server startup); production runner wraps 502/503/504 in `ErrModelTransient` | debts (w4-e/w4-f dispositions) | in-tree `internal/api/chat.go:97-118,143-165`, `internal/auth/oauth.go:221-264`, `internal/inference/combo.go:21-23,190-191` | — (FIRST, alone) |
| **w5-a** | Schema + pricing engine: `request_log`/`usage_daily`/`request_details`/`kv` tables in migrate.go; `internal/usage` pricing tables (MODEL/PROVIDER/PATTERN), glob matcher, 3-step resolution, 5-category cost calc, token-field normalization, kv-backed user overrides merged view + 5s cache | 001-004 (tables), 005,006,007,008,009,010,040 | `src/shared/constants/pricing.js:12-303`, `src/lib/db/repos/pricingRepo.js:5-108`, `src/lib/db/schema.js:96-150` | w5-pre |
| **w5-b** | Usage write path + live trackers: transactional save (history insert + daily upsert + lifetime counter), cost-at-save, daily aggregation entry shape (byProvider/byModel/byAccount/byApiKey/byEndpoint with meta), recent-request dedup, pending tracker (60s timeout), ring buffer (50), connection-name cache (30s TTL), stats-update event hook for SSE | 001,002 (write semantics), 011,012,018,019,020,038 | `src/lib/db/repos/usageRepo.js:6-287` | w5-a |
| **w5-c** | Observability writer: buffered request-details writes (batch flush at batchSize, timer flush, shutdown flush), config from settings+env (enabled/maxRecords/batchSize/flushIntervalMs/maxJsonSize), header sanitizer (=PAR-AUTH-017), JSON truncation >maxJsonSize, retention delete-oldest, filtered+paginated query; debug-log production gate (PAR-AUTH-018) in `internal/logging` | 003 (write semantics), 024,025,026,027,028 + AUTH-017,018 | `src/lib/db/repos/requestDetailsRepo.js:4-200`, `debugLog.js:3` | w5-a (∥ w5-b) |
| **w5-d** | Usage read APIs: `/api/usage/stats` (periods today/24h/7d/30d/60d/all; daily-summary path >24h vs live-history path; totals + 5 breakdowns; last10Minutes 1-min buckets; lastUsed overlay), `/api/usage/chart` (24 hourly / N daily buckets + labels), `/api/usage/logs` (derived format), `/api/usage/request-details` (filters+pagination), `/api/pricing` GET/PATCH(validate non-negative known fields)/DELETE(per-provider/per-model/all) | 013,014,015,016,017,021,022,023,024 (route),029,030,031,039 | `usageRepo.js:319-731`, `src/app/api/pricing/route.js:9-117`, `src/app/api/usage/*/route.js` | w5-b, w5-c |
| **w5-e** | Usage SSE + provider quota: `/api/usage/stream` (full stats on update event, lightweight pending betweens, 25s keepalive ping), `/api/usage/{connectionId}` provider-quota dispatcher + claude/gemini fetchers with OAuth auto-refresh before fetch + retry-once on auth expiry (uses w5-pre refresher) | 032 (Stage-1 half), 033, 034, 035 | `src/app/api/usage/stream/route.js:10-78`, `src/app/api/usage/[connectionId]/route.js:16-188`, `open-sse/services/usage.js:60-101,225-342,497-614` | w5-b, w5-pre; serial-on-routes_admin.go after w5-d |
| **w5-f** | Pipeline usage glue (ONLY internal/api editor): pending start/end around dispatch, usage extraction from non-stream responses + stream accumulation, estimate-on-finish when no valid usage + TTFT/content-length tracking (= PAR-TRANS-046 usage clause), save request_log row with model/provider/connection/apiKey/endpoint attribution (= PAR-ROUTE-054), request-details capture on success/error paths | TRANS-046 (PARTIAL→HAVE), ROUTE-054 + wiring halves of 003/012/018 | `open-sse/handlers/chatCore.js:135-140,196,242`, `open-sse/handlers/chatCore/requestDetail.js:75-103`, `open-sse/utils/stream.js:147-160,290-335`, `usageTracking.js:115-305` | w5-b, w5-c (∥ d,e) |
| **w5-g** | Virtual keys: `virtual_keys` table + store, `x-g0-vk` header routing resolution, per-key quota tracking (budget spend from request_log cost attribution, RPM window), admin CRUD routes | ROUTE-030, ROUTE-031 (032 schema HAVE) | `internal/schemas/governance.go:4-25`, g0router phase-8 plan; spend source = request_log | w5-b, w5-f (serial-on-internal/api), w5-d (serial-on-routes_admin.go) |

## Ownership tracks (W3/W4 lesson: NO shared files across live jobs)

- w5-pre: `internal/auth/refresher*.go` (new), `internal/server/server.go` (wiring),
  `internal/inference/runner*.go` (transient wrap) — runs ALONE first.
- w5-a: `internal/store/migrate.go` + `internal/store/kv*.go` +
  `internal/usage/pricing*.go` (new package) — ALONE (migrate.go is hot).
- w5-b: `internal/store/requestlog*.go`, `internal/usage/tracker*.go`.
- w5-c: `internal/store/requestdetails*.go`, `internal/usage/observability*.go`,
  `internal/logging/*` — disjoint from w5-b.
- w5-d: `internal/admin/usage*.go`, `internal/admin/pricing*.go`,
  `internal/server/routes_admin.go`.
- w5-e: `internal/admin/usagestream*.go`, `internal/usage/providerusage*.go`;
  routes_admin.go additions AFTER w5-d merges.
- w5-f: `internal/api/*` — the ONLY plan touching internal/api (except w5-g after).
- w5-g: `internal/store/virtualkeys*.go`, `internal/governance/*`, api hook +
  admin routes — LAST (serializes on both api and admin route files).

## Impl order

w5-pre ALONE → w5-a ALONE → (w5-b ∥ w5-c) → (w5-d → w5-e) ∥ w5-f → w5-g LAST.

## Protocol (unchanged)

Plan → gpt-5.5 plan gate (≤3 cycles → decide) → kimi TDD impl → go test/vet/-race →
scoped diff gate (commit-bounded; live-tree verification before closure; remember the
known gate artifact: diff-only analysis flags pre-existing imports — go build is
ground truth) → merge → flip rows → WORKFLOW.md. Commits: `phase-1/w5-X: <desc>`.

## Out of Wave-5 scope (explicit)

UI components (036/037 → W6). Stage-2 provider quota fetchers (gh/antigravity/codex/
kiro/glm/minimax). Live model pinging for GetTestByKind (W6). MITM/proxy pools
(S2/W7). Tunnels (W7). SQLite JSON-function rewrite of daily rollups (port the
ref's app-side JSON blobs; optimization is post-parity backlog).
