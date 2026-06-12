# Parity Matrix: USAGE + COST

Reference: `/Users/heitor/Developer/github.com/bloodf/_refs/9router` (frozen SHA `827e5c3`)
Target: `/Users/heitor/Developer/github.com/bloodf/g0router`

---

## Behavior Matrix

| ID | Behavior | Evidence (file:line) | g0router status | Notes |
|---|---|---|---|---|
| PAR-USAGE-001 | `usageHistory` table stores per-request timestamp, provider, model, connectionId, apiKey, endpoint, promptTokens, completionTokens, cost, status, tokens JSON, meta JSON | `src/lib/db/schema.js:105-127` | HAVE | request_log table (w5-a) + transactional SaveUsage write path (w5-b) |
| PAR-USAGE-002 | `usageDaily` table stores dateKey + aggregated JSON (byProvider, byModel, byAccount, byApiKey, byEndpoint) | `src/lib/db/schema.js:128-133` | HAVE | usage_daily table (w5-a) + aggregateEntryToDay rollup upsert in same tx (w5-b) |
| PAR-USAGE-003 | `requestDetails` table stores observability records with id, timestamp, provider, model, connectionId, status, data JSON | `src/lib/db/schema.js:134-150` | HAVE | request_details table (w5-a) + buffered DetailWriter batch/timer flush, retention, Close (w5-c) |
| PAR-USAGE-004 | `kv` table with scope='pricing' stores user pricing overrides per provider | `src/lib/db/schema.js:96-104` | HAVE | kv table + Store.UserPricing() reader + SetKV/GetKV/ListKV (internal/store/kv.go); scope=pricing overrides consumed by usage.Resolver (w5-a) |
| PAR-USAGE-005 | Provider-specific pricing overrides merged with canonical model pricing and pattern pricing | `src/shared/constants/pricing.js:124-129` | HAVE | ProviderPricing override map ported; gh/gpt-5.3-codex golden-tested (internal/usage/pricingdata.go) (w5-a) |
| PAR-USAGE-006 | Canonical model pricing table covers Anthropic, OpenAI, Gemini, Qwen, Kimi, DeepSeek, GLM, MiniMax, Grok, etc. | `src/shared/constants/pricing.js:12-117` | HAVE | ModelPricing: 83 entries ported verbatim, count-asserted (w5-a) |
| PAR-USAGE-007 | Pattern-based pricing fallback using glob patterns (e.g. `*-codex-xhigh`, `claude-opus-*`) | `src/shared/constants/pricing.js:136-207` | HAVE | PatternPricing: 49 ordered glob patterns, first-match-wins, anchors tested (w5-a) |
| PAR-USAGE-008 | Three-step pricing resolution: PROVIDER_PRICING → MODEL_PRICING → PATTERN_PRICING | `src/shared/constants/pricing.js:227-248` | HAVE | 3-step resolution ResolvePricing + user-override-first PricingForModel (exact provider/model match per fix-r1) (w5-a) |
| PAR-USAGE-009 | Cost calculation supports input, output, cached, reasoning, cache_creation rates per 1M tokens | `src/shared/constants/pricing.js:274-303` | HAVE | CalculateCost: 5 rate categories per 1M, cached-subtraction, reasoning/cache_creation fallbacks, golden-tested (internal/usage/cost.go) (w5-a) |
| PAR-USAGE-010 | Token field normalization: prompt_tokens / input_tokens, completion_tokens / output_tokens, cached_tokens / cache_read_input_tokens | `src/lib/db/repos/usageRepo.js:121-122` | HAVE | NormalizeTokens accepts prompt|input, completion|output, cached|cache_read synonyms (internal/usage/tokens.go) (w5-a) |
| PAR-USAGE-011 | Usage entry saved atomically in transaction: insert history, upsert daily, increment lifetime counter in `_meta` | `src/lib/db/repos/usageRepo.js:243-287` | HAVE | SaveUsage: history insert + daily upsert + kv meta lifetime counter in ONE tx; rollback-tested (w5-b) |
| PAR-USAGE-012 | `saveRequestUsage` computes cost before persisting via `calculateCost(provider, model, tokens)` | `src/lib/db/repos/usageRepo.js:248` | HAVE | usage.Recorder computes cost via Resolver.CostFor before SaveUsage; emits update (w5-b) |
| PAR-USAGE-013 | `getUsageStats` supports periods: today, 24h, 7d, 30d, 60d, all | `src/app/api/usage/stats/route.js:4` | MISSING | No usage stats API in g0router |
| PAR-USAGE-014 | Daily-summary path used for periods >24h; live-history path used for today/24h | `src/lib/db/repos/usageRepo.js:418` | MISSING | No aggregation strategy exists |
| PAR-USAGE-015 | Stats include totalRequests, totalPromptTokens, totalCompletionTokens, totalCost | `src/lib/db/repos/usageRepo.js:367-370` | MISSING | No stats computation |
| PAR-USAGE-016 | Stats break down byProvider, byModel, byAccount, byApiKey, byEndpoint | `src/lib/db/repos/usageRepo.js:370` | MISSING | No multi-dimensional aggregation |
| PAR-USAGE-017 | last10Minutes bucket array computed from usageHistory with 1-minute buckets | `src/lib/db/repos/usageRepo.js:394-416` | MISSING | No minute-level bucketing |
| PAR-USAGE-018 | Active requests tracked in-memory with pending timeout (60s) and automatic cleanup | `src/lib/db/repos/usageRepo.js:6,153-196` | HAVE | usage.Tracker: 60s timer timeout, clamp, error-provider 10s window, reentrant-safe emit (w5-b) |
| PAR-USAGE-019 | Recent request ring buffer capped at 50 entries, initialized from DB on first access | `src/lib/db/repos/usageRepo.js:7,79-111` | HAVE | usage.Ring cap 50, lazy init-once from store, mutex-guarded (w5-b) |
| PAR-USAGE-020 | Connection name cache with 30s TTL for display labels | `src/lib/db/repos/usageRepo.js:8,86-97` | HAVE | ConnNameCache 30s TTL, name→email→id fallback, stale-on-error per ref (w5-b) |
| PAR-USAGE-021 | `getChartData` returns 24 hourly buckets for today/24h, daily buckets for 7d/30d/60d | `src/app/api/usage/chart/route.js:4` | MISSING | No chart data API |
| PAR-USAGE-022 | Chart bucket labels use locale time strings for hours, short date for days | `src/lib/db/repos/usageRepo.js:631,673` | MISSING | No chart formatting |
| PAR-USAGE-023 | `getRecentLogs` derives logs from usageHistory with formatted string output | `src/lib/db/repos/usageRepo.js:701-730` | MISSING | No log API |
| PAR-USAGE-024 | `getRequestDetails` supports filtering by provider, model, connectionId, status, startDate, endDate with pagination | `src/lib/db/repos/requestDetailsRepo.js:144-175` | HAVE | QueryRequestDetails: 6 filters + pagination math, raw data blobs, never-nil rows (w5-c) |
| PAR-USAGE-025 | Request details observability config: enabled flag, maxRecords, batchSize, flushIntervalMs, maxJsonSize | `src/lib/db/repos/requestDetailsRepo.js:13-40` | HAVE | ObsConfigLoader: settings>env precedence, enabled flag, 5s cache; KB*1024 per ref :27 (w5-c) |
| PAR-USAGE-026 | Request details write buffer with batch flush and shutdown handler | `src/lib/db/repos/requestDetailsRepo.js:42-142` | HAVE | DetailWriter: batch-size immediate flush + interval timer, single-flight drain, id timestamp-random6-modelSlug, JSON.stringify-parity serialization, Close flush (w5-c) |
| PAR-USAGE-027 | Request details sanitize headers (authorization, x-api-key, cookie, token, api-key) | `src/lib/db/repos/requestDetailsRepo.js:46-54` | HAVE | SanitizeHeaders: case-insensitive substring delete of authorization/x-api-key/cookie/token/api-key (w5-c) |
| PAR-USAGE-028 | Request details retention: delete oldest when count exceeds maxRecords | `src/lib/db/repos/requestDetailsRepo.js:109-115` | HAVE | Retention delete-oldest beyond maxRecords in same tx; oldest-gone tested (w5-c) |
| PAR-USAGE-029 | Pricing API GET returns merged user + default pricing | `src/app/api/pricing/route.js:9-20` | MISSING | No pricing endpoint in g0router admin routes |
| PAR-USAGE-030 | Pricing API PATCH validates fields input/output/cached/reasoning/cache_creation as non-negative numbers | `src/app/api/pricing/route.js:27-83` | MISSING | No pricing mutation API |
| PAR-USAGE-031 | Pricing API DELETE resets per-provider, per-model, or all pricing | `src/app/api/pricing/route.js:91-117` | MISSING | No pricing reset API |
| PAR-USAGE-032 | Provider usage API fetches external quotas for GitHub, Gemini, Antigravity, Claude, Codex, Kiro, GLM, MiniMax | `src/app/api/usage/[connectionId]/route.js:16,122-188` | PARTIAL | Stage-1 half (w5-e): dispatcher + claude (OAuth-first→legacy) + gemini (quota+subscription) fetchers; remaining six providers ship with Stage-2 adapters |
| PAR-USAGE-033 | Provider usage API auto-refreshes OAuth credentials before fetching, retries once on auth expiry | `src/app/api/usage/[connectionId]/route.js:155-180` | HAVE | Connection-scoped refresh-before-fetch + force-refresh retry-once on auth-expired message (w5-e fix-r2) |
| PAR-USAGE-034 | Usage SSE stream pushes full stats on update event, lightweight pending updates in between | `src/app/api/usage/stream/route.js:10-78` | HAVE | /api/usage/stream SSE: full stats on update, lightweight pending overlay with live active_requests (w5-e) |
| PAR-USAGE-035 | Usage SSE stream sends keepalive ping every 25s | `src/app/api/usage/stream/route.js:53-61` | HAVE | 25s keepalive ping comment, interval-injected (w5-e) |
| PAR-USAGE-036 | Dashboard UsageStats component fetches `/api/usage/stats?period=`, subscribes to `/api/usage/stream` | `src/shared/components/UsageStats.js:242-278` | MISSING | UI has no usage stats page |
| PAR-USAGE-037 | Dashboard RequestLogger polls `/api/usage/request-logs` every 3s with auto-refresh toggle | `src/shared/components/RequestLogger.js:15-23` | MISSING | UI has no request logger component |
| PAR-USAGE-038 | Usage history dedupes recent requests by (model + provider + promptTokens + completionTokens + minute) | `src/lib/db/repos/usageRepo.js:229-237` | HAVE | DedupeRecent: zero-token drop + minute composite key, cap 20 (w5-b) |
| PAR-USAGE-039 | Usage daily aggregation overlays precise lastUsed timestamps from history rows | `src/lib/db/repos/usageRepo.js:506-530` | MISSING | No lastUsed overlay |
| PAR-USAGE-040 | Pricing cache TTL 5s in memory to avoid repeated DB reads | `src/lib/db/repos/pricingRepo.js:6-12` | HAVE | Resolver.Merged() 5s TTL cache + Invalidate() hook, injected clock (w5-a) |

---

## Data models

### Reference: `usageHistory`
- `id` INTEGER PRIMARY KEY AUTOINCREMENT
- `timestamp` TEXT NOT NULL
- `provider` TEXT
- `model` TEXT
- `connectionId` TEXT
- `apiKey` TEXT
- `endpoint` TEXT
- `promptTokens` INTEGER DEFAULT 0
- `completionTokens` INTEGER DEFAULT 0
- `cost` REAL DEFAULT 0
- `status` TEXT
- `tokens` TEXT (JSON blob)
- `meta` TEXT (JSON blob)
- Indexes: timestamp DESC, provider, model, connectionId

### Reference: `usageDaily`
- `dateKey` TEXT PRIMARY KEY
- `data` TEXT NOT NULL (JSON with requests, promptTokens, completionTokens, cost, byProvider, byModel, byAccount, byApiKey, byEndpoint)

### Reference: `requestDetails`
- `id` TEXT PRIMARY KEY
- `timestamp` TEXT NOT NULL
- `provider` TEXT
- `model` TEXT
- `connectionId` TEXT
- `status` TEXT
- `data` TEXT NOT NULL (JSON with latency, tokens, request, providerRequest, providerResponse, response)
- Indexes: timestamp DESC, provider, model, connectionId

### Reference: Pricing kv
- `scope` TEXT NOT NULL, `key` TEXT NOT NULL, `value` TEXT NOT NULL
- Primary key: (scope, key)
- scope='pricing' stores per-provider JSON maps of model → {input, output, cached, reasoning, cache_creation}

### g0router: `schemas.Usage`
- `PromptTokens` int
- `CompletionTokens` int
- `TotalTokens` int
- `PromptTokensDetails` *TokensDetails (AudioTokens, CachedTokens, TextTokens, ImageTokens)
- `CompletionTokensDetails` *TokensDetails (same fields)
- Evidence: `internal/schemas/chat.go:120-135`

---

## Edge cases and quirks

- Token normalization accepts both `prompt_tokens` and `input_tokens` as synonyms; same for `completion_tokens` / `output_tokens` and `cached_tokens` / `cache_read_input_tokens`. Evidence: `src/lib/db/repos/usageRepo.js:121-122`.
- Cost calculation subtracts cached tokens from input tokens to compute non-cached input cost, then adds cached cost separately. Evidence: `src/lib/db/repos/usageRepo.js:123-129`.
- Reasoning tokens and cache_creation tokens fall back to output/input rate respectively if no dedicated rate exists. Evidence: `src/lib/db/repos/usageRepo.js:134-144`.
- Daily aggregation JSON stores nested counters with `rawModel`, `provider`, `apiKey`, `endpoint`, `accountName` metadata fields. Evidence: `src/lib/db/repos/usageRepo.js:55-77`.
- Pending request timeout is 60s; timer leaks are prevented by clearing on end. Evidence: `src/lib/db/repos/usageRepo.js:6,173-186`.
- Recent request deduplication drops entries with zero tokens and dedupes by minute-granularity composite key. Evidence: `src/lib/db/repos/usageRepo.js:229-237`.
- Observability defaults to disabled unless env `OBSERVABILITY_ENABLED` is not false or setting is true. Evidence: `src/lib/db/repos/requestDetailsRepo.js:18-21`.
- Request details truncate JSON payloads > maxJsonSize (default 5KB) with `_truncated`, `_originalSize`, `_preview` fields. Evidence: `src/lib/db/repos/requestDetailsRepo.js:63-69`.
- Pricing validation rejects unknown fields and negative values. Evidence: `src/app/api/pricing/route.js:57-71`.
- Provider usage API allows apikey-based providers (GLM, MiniMax) in addition to OAuth. Evidence: `src/app/api/usage/[connectionId]/route.js:135-139`.
- GitHub Copilot usage API uses accessToken (not copilotToken) in the open-sse version but copilotToken in the legacy src/lib version. Evidence: `open-sse/services/usage.js:144` vs `src/lib/usage/fetcher.js:41-48`.
- Claude usage tries OAuth endpoint first, falls back to legacy settings/org endpoint. Evidence: `open-sse/services/usage.js:497-611`.
- Kiro usage tries three endpoints (codewhisperer-get, codewhisperer-post, q-get) and returns differentiated error messages per auth method. Evidence: `open-sse/services/usage.js:755-873`.

---

## Go-port considerations

- Add `usageHistory`, `usageDaily`, `requestDetails`, and `kv` tables to `migrate.go`.
- Implement `calculateCost` with token field normalization and five rate categories.
- Wire usage extraction into chat/embeddings handlers after provider response.
- Add admin routes: `/api/usage/stats`, `/api/usage/chart`, `/api/usage/logs`, `/api/usage/request-details`, `/api/pricing`, `/api/usage/stream`.
- Consider SQLite JSON functions for daily aggregation instead of application-side JSON blobs.
