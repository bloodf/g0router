# ROUTE parity matrix: 9router → g0router

Reference: `/Users/heitor/Developer/github.com/bloodf/_refs/9router` (SHA `827e5c3`)

## Behavior rows

| ID | Behavior | Evidence (file:line) | g0router status | Notes |
|---|---|---|---|---|
| PAR-ROUTE-001 | Combo fallback strategy: ordered model list, try next on transient/error | `open-sse/services/combo.js:108-198`, `open-sse/services/combo.js:151-170` |  HAVE  |  `ComboEngine.ExecuteCombo`: ordered-model fallback iterates combo.Models in order, falling back on per-model failure. (w4-e)  |
| PAR-ROUTE-002 | Combo round-robin strategy with sticky limit (requests per model before switching) | `open-sse/services/combo.js:36-65`, `tests/unit/combo-routing.test.js:25-40` |  HAVE  |  `ComboEngine` round-robin + sticky: in-memory `rrStates` map, no TTL (resets on restart), `normalizeStickyLimit` default 1. (w4-e)  |
| PAR-ROUTE-003 | Combo per-combo strategy override vs global default | `src/sse/handlers/chat.js:96-99`, `src/sse/handlers/chat.js:128-134` |  HAVE  |  Per-combo strategy override via `comboStrategies[name].fallbackStrategy` JSON setting key. (w4-e)  |
| PAR-ROUTE-004 | Combo name validation (alphanumeric, hyphens, underscores, dots) | `src/app/api/combos/route.js:7`, `src/app/api/combos/route.js:31-33` |  HAVE  |  Combo name validation: `^[a-zA-Z0-9_.\-]+$` per src/app/api/combos/route.js:7, enforced in CreateCombo handler. (w4-e)  |
| PAR-ROUTE-005 | Model alias resolution from localDb aliases map | `src/sse/services/model.js:22-25`, `open-sse/services/model.js:182-208` |  HAVE  |  Chain resolution: `internal/store/aliases.go` ResolveChain (DFS visited-set) + `internal/inference/alias.go` ResolveModelAlias. (w4-a) |
| PAR-ROUTE-006 | Provider alias to ID mapping (~140 aliases) | `open-sse/services/model.js:1-143` |  HAVE  |  ~140 aliases ported verbatim to `internal/providers/catalog/aliases.go`; ResolveProviderAlias + ForEachProviderAlias accessors. (w4-a) |
| PAR-ROUTE-007 | Provider prefix parsing (`provider/model` or `alias/model`) | `open-sse/services/model.js:155-176` |  HAVE  |  ParseModelPrefix + InferProvider (longest-alias-first) + catalog alias resolution wired into factory.go providerForModel. (w4-a) |
| PAR-ROUTE-008 | Model name prefix inference when no alias matches | `open-sse/services/model.js:248-259` |  HAVE  |  InferProvider in `internal/inference/alias.go`: sorted longest-alias-first over catalog aliases. (w4-a) |
| PAR-ROUTE-009 | Provider node prefix matching (openai-compatible, anthropic-compatible, custom-embedding) | `src/sse/services/model.js:35-51`, `src/app/api/provider-nodes/route.js:32-104` | HAVE | w7-platnodes: `Router.Resolve` step-0 consults `NodeResolver.ResolveByPrefix` BEFORE static alias/catalog (`internal/inference/router.go`, `noderesolve.go`); a model `mn/x` whose prefix `mn` is a registered node routes to that node's provider+base URL, overriding static alias resolution (`internal/platform/providernodes.go` ResolveByPrefix). |
| PAR-ROUTE-010 | Circular alias loop validation | `docs/ARCHITECTURE.md:17` (mentioned), `.planning/phases/09-models-aliases-combos/PLAN.md:64` |  HAVE  |  DFS cycle detection in store.ResolveChain visited-set; CreateAlias rejects cycles at write time. g0router-defensive (ref has no impl). (w4-a) |
| PAR-ROUTE-011 | Combo recursive resolution protection | `src/sse/handlers/chat.js:122-147` |  HAVE  |  Recursive resolution guard: `visited` set passed through executeCombo; detects cycles and returns ErrComboRecursion immediately. (w4-e)  |
| PAR-ROUTE-012 | Per-model account locks (`modelLock_${model}`) | `open-sse/services/accountFallback.js:106-114`, `src/sse/services/auth.js:203-241` |  HAVE  |  `connection_model_locks` table + `LockModel`/`LockAccount`; `CooldownEngine.MarkUnavailable` in `internal/inference/accounts.go`. (w4-c)  |
| PAR-ROUTE-013 | Account-level lock (`modelLock___all`) | `open-sse/services/accountFallback.js:109`, `open-sse/services/accountFallback.js:120-125` |  HAVE  |  `LockAccount` writes model='__all' sentinel; `EarliestExpiry` includes `OR model='__all'` in query. (w4-c)  |
| PAR-ROUTE-014 | Account cooldown with exponential backoff (`backoffLevel`) | `open-sse/services/accountFallback.js:9-13`, `open-sse/services/accountFallback.js:31-34`, `open-sse/config/errorConfig.js:32-35` |  HAVE  |  Exponential backoff in `quotaCooldown` (base=2s, max=5min per actual ref); `backoff_level` additive column on connections. (w4-c)  |
| PAR-ROUTE-015 | Account state reset on success (clear model lock + backoff) | `src/sse/services/auth.js:252-285`, `open-sse/services/accountFallback.js:184-193` |  HAVE  |  `CooldownEngine.MarkSuccess` clears all model locks + resets backoff_level=0. (w4-c)  |
| PAR-ROUTE-016 | Provider account fallback loop (`excludeConnectionIds`) | `src/sse/handlers/chat.js:162-245` |  HAVE  |  `SelectionEngine.WithAccountFallback` fallback loop with growing excludeConnectionIds; terminates when all excluded (PR-640). (w4-d)  |
| PAR-ROUTE-017 | Account selection mutex (prevents race conditions) | `src/sse/services/auth.js:9`, `src/sse/services/auth.js:24-30` |  HAVE  |  Single package-level `selectionMu sync.Mutex` serializes all `SelectConnection` calls (faithful port of auth.js global promise mutex). (w4-d)  |
| PAR-ROUTE-018 | Account selection strategies (`fill-first`, `round-robin`) | `src/sse/services/auth.js:102-157` |  HAVE  |  `SelectionEngine.SelectConnection`: fill-first (default) and round-robin strategies read from settings; mirrors auth.js:102-157. (w4-d)  |
| PAR-ROUTE-019 | Sticky round-robin limit for accounts | `src/sse/services/auth.js:116`, `src/sse/services/auth.js:129-136` |  HAVE  |  Sticky round-robin limit (`stickyRoundRobinLimit` setting, default 3): stay with current connection until limit reached, then rotate. (w4-d)  |
| PAR-ROUTE-020 | Per-URL retry with configurable attempts/delay by status code | `open-sse/executors/base.js:98-174`, `open-sse/config/runtimeConfig.js:52-57` |  HAVE  |  newDefaultRetryConfig() in `internal/inference/retry.go`: 429→0, 502→3/3s, 503→3/2s, 504→2/3s. (w4-b) |
| PAR-ROUTE-021 | Provider-specific retry config override | `open-sse/executors/base.js:105`, `open-sse/config/providers.js:197` |  HAVE  |  Catalog `Retry map[int]int` field; kiro override {429:2} applied per-provider in retry middleware. (w4-b) |
| PAR-ROUTE-022 | Connect timeout abort and retry mapping to 502 | `open-sse/executors/base.js:125-128`, `open-sse/executors/base.js:156-163` |  HAVE  |  net.Error.Timeout() at fasthttp boundary → 502, not retried. ARCH: fasthttp has no distinct dial-timeout type; Timeout()+!Temporary() is the Go/fasthttp idiom. (w4-b) |
| PAR-ROUTE-023 | Token refresh on 401/403 with retry | `open-sse/handlers/chatCore.js:216-235` |  HAVE  |  `retryWithRefresh` in `ChatHandler` performs up to 3 refresh+dispatch cycles; `CredentialRefresher` interface + nil guard for graceful degradation; production OAuth wiring deferred to credential-management wave. (w4-f) |
| PAR-ROUTE-024 | Combo transient error cooldown (503/502/504 waits up to 5s before next model) | `open-sse/services/combo.js:161-165` |  HAVE  |  Transient cooldown ≤5s: if ModelRetryAfter ≤5s, sleep and retry same model once; if >5s, skip to next model. (w4-e)  |
| PAR-ROUTE-025 | Disabled model tracking per provider alias | `src/app/api/models/disabled/route.js:1-50`, `src/app/api/v1/models/route.js:190-191` |  HAVE  |  `disabled_models` table + `DisableModels`/`EnableModels`/`ListDisabledModels`/`IsDisabled` in `internal/store/disabledmodels.go`; `/api/models/disabled` CRUD in admin. (w4-c)  |
| PAR-ROUTE-026 | Disabled models excluded from `/v1/models` | `src/app/api/v1/models/route.js:354`, `src/app/api/v1/models/route.js:225` |  HAVE  |  `ModelsHandler.List` filters via `DisabledChecker.IsDisabled`; wired via `RegisterOpenAIRoutes` in `internal/server/routes_openai.go`. (w4-c)  |
| PAR-ROUTE-027 | Weighted provider selection | `.planning/phases/08-keys-virtualkeys-routing/PLAN.md:23` | MISSING | 9router has no explicit weighted selection for providers. g0router Phase 8 plans weighted provider selection but it is not implemented. |
| PAR-ROUTE-028 | API key validation (`requireApiKey` setting) | `src/sse/handlers/chat.js:68-80` |  HAVE  |  `NewAPIKeyValidator` at `internal/auth/apikey.go:205`; wired in `guard.go`. (w4-f verify-flip) |
| PAR-ROUTE-029 | API key extraction (`Bearer`, `x-api-key`) | `src/sse/services/auth.js:290-304` |  HAVE  |  `TestGuardV1RemoteValidKey` at `internal/server/guard_test.go:233`; Bearer extraction in `auth/apikey.go`. (w4-f verify-flip) |
| PAR-ROUTE-030 | Virtual key routing via `x-g0-vk` header | `.planning/phases/08-keys-virtualkeys-routing/PLAN.md:46` | HAVE | Full: x-g0-vk gate + model constraints + quota + attribution (w5-g) + KeyIDs pinning via VKPinnedKeyResolver → SelectConnection preferredConnID (w6-pre) |
| PAR-ROUTE-031 | Per-key quota tracking | `.planning/phases/08-keys-virtualkeys-routing/PLAN.md:25` | HAVE | QuotaEngine: budget windows daily/weekly/monthly + RPM minute window + SumCostByAPIKey real spend, race-tested (w5-g) |
| PAR-ROUTE-032 | Virtual key budget and rate limit RPM schema | `internal/schemas/governance.go:4-25` | HAVE | `VirtualKey` struct defines `Budget` and `RateLimitRPM`. 9router has no virtual key schema. |
| PAR-ROUTE-033 | Request format auto-detection (OpenAI, Claude, Gemini, Antigravity, Responses) | `open-sse/services/provider.js:49-126` |  HAVE  |  `DetectFormat` in `internal/api/detect.go`; inlined from `detectBypassSourceFormat`; covers all 5 formats; `TestFormatAutoDetect` green. (w4-f) |
| PAR-ROUTE-034 | Bypass patterns for Claude CLI (warmup, count, title, skip, naming) | `open-sse/utils/bypassHandler.js:11-91` |  HAVE  |  `HandleBypassRequest` wired in `api/chat.go` + `api/messages.go`; `TestBypassWarmupShortCircuits` + `TestBypassTitleSkip` green. (w4-f) |
| PAR-ROUTE-035 | Provider-specific URL building with fallback URLs | `open-sse/services/provider.js:155-209`, `open-sse/executors/base.js:20-42` | PARTIAL | Single `chatURL()` at `internal/providers/generic/chat.go:20`; no index-based fallback URL list. Multi-URL fallback is out of scope for Stage 1. (w4-f verify-flip) |
| PAR-ROUTE-036 | Provider-specific header building (auth, spoofing, streaming) | `open-sse/services/provider.js:212-320` |  HAVE  |  `TestGenericChatCustomHeaders` at `internal/providers/generic/chat_test.go:55`; per-provider auth headers in generic + anthropic providers. (w4-f verify-flip) |
| PAR-ROUTE-037 | Model kind routing (`/v1/models/{kind}`) | `src/app/api/v1/models/[kind]/route.js:1-55` |  HAVE  |  `GetByKind` + `GetOrByKind` dispatcher; `kindSlugMap` (6 kinds); `TestModelsByKind` with catalog-type assertions; route `/v1/models/{param}` in `routes_openai.go`. (w4-f) |
| PAR-ROUTE-038 | Model test routing by kind (llm, image, embedding, stt) | `src/app/api/models/test/route.js:1-14`, `tests/unit/model-test-routing.test.js:47-166` |  HAVE  |  `GetTestByKind` returns kind→endpoint metadata; `kindTestEndpoints` map; `TestModelTestRoutesByKind` (7 kinds + 404); live pinging is BFF/admin-layer concern deferred to dashboard wave. (w4-f) |
| PAR-ROUTE-039 | Free provider no-auth virtual connection injection | `src/sse/services/auth.js:36-53`, `src/shared/constants/providers.js:14` | MISSING | Injects synthetic connection for `noAuth` providers (e.g. opencode). g0router has no free-provider virtual connection logic. |
| PAR-ROUTE-040 | OpenAI-compatible and Anthropic-compatible provider node routing | `src/sse/services/model.js:35-51`, `src/app/api/provider-nodes/route.js:32-104` | HAVE | w7-platnodes: a node's `api_type` is carried through `ResolveByPrefix`; the prefix-stripped bare model is routed to the node's base URL via a generic OpenAI-compatible adapter pointed at that base URL (`internal/inference/noderesolve.go` buildNodeProvider → `generic.NewNode`). Anthropic-compatible nodes are served through the same adapter pointed at the node base URL (adapter-by-api_type is a tracked follow-up, see open-questions). |
| PAR-ROUTE-041 | Native passthrough detection (same ecosystem, skip translation) | `open-sse/handlers/chatCore.js:86-103` |  HAVE  |  `MessagesHandler.Handle`: provider resolved before translation; `NativeFormat()` interface checked; direct body unmarshal skips `TranslateRequest`; `TestNativePassthroughSkipsTranslation` green. (w4-f) |
| PAR-ROUTE-042 | Provider thinking config override injection | `open-sse/handlers/chatCore.js:48-58` |  HAVE  |  `applyThinkingOverride` in `chat.go`; `ThinkingMode()` interface (on/off/effort string); injects `ThinkingConfig{enabled,10000}` or `ReasoningEffort`; `TestThinkingOverrideInjected` green. (w4-f) |
| PAR-ROUTE-043 | Streaming vs non-streaming decision logic | `open-sse/handlers/chatCore.js:60-77` |  HAVE  |  `useStream` decision in `ChatHandler.Handle`: `RequiresStreaming()` interface, deepseek-tui UA, `Accept: application/json`; `TestStreamDecision` (3 subtests) green. (w4-f) |
| PAR-ROUTE-044 | Error classification rules (text-first then status) | `open-sse/config/errorConfig.js:59-76` |  HAVE  |  classificationRules() in `internal/inference/errorclass.go`: 9 text rules then 5 status rules, exact errorConfig.js sequence. (w4-b) |
| PAR-ROUTE-045 | Precise cooldown override from provider-reported `resetsAtMs` | `src/sse/services/auth.js:211-214`, `open-sse/services/accountFallback.js:49` |  HAVE  |  extractResetsAt: resets_at (sec/ms) + resets_in_seconds; hard-capped at maxRateLimitCooldown (30min). (w4-b) |
| PAR-ROUTE-046 | Earliest retry-after tracking across combo models | `open-sse/services/combo.js:113`, `open-sse/services/combo.js:141-143`, `open-sse/services/combo.js:187-191` |  HAVE  |  EarliestRetryAfter aggregates ModelRetryAfter across all combo models; returns earliest non-expired time. (w4-e)  |
| PAR-ROUTE-047 | Model promotion rules (combo names appear first in `/v1/models`) | `src/app/api/v1/models/route.js:202-213` |  HAVE  |  /v1/models response: combo names prepended (owned_by=combo) before sorted provider models. (w4-e)  |
| PAR-ROUTE-048 | Quota window parsing (unix timestamps in seconds or milliseconds) | `open-sse/services/usage.js:103-131` |  HAVE  |  parseTimestamp: <1e12 treated as seconds (×1000), ≥1e12 as milliseconds. (w4-b) |
| PAR-ROUTE-049 | Group lock semantics (all accounts locked → return earliest expiry) | `src/sse/services/auth.js:80-98` |  HAVE  |  `CooldownEngine.GroupRetryAfter(providerID, model)` returns earliest expiry across all locked connections; `__all` sentinel included. (w4-c)  |
| PAR-ROUTE-050 | Per-provider strategy override (`providerStrategies`) | `src/sse/services/auth.js:101-103` |  HAVE  |  Per-provider strategy override via `providerStrategies[providerId].fallbackStrategy` JSON setting key. (w4-d)  |
| PAR-ROUTE-051 | Pinned connection preference | `src/sse/services/auth.js:106-112` |  HAVE  |  Pinned connection preference: if `preferredConnID` is eligible, return it before applying strategy. (w4-d)  |
| PAR-ROUTE-052 | Provider credentials refresh before dispatch | `src/sse/handlers/chat.js:188-198` |  HAVE  |  `CredentialRefresher` interface + `SetCredentialRefresher`; `retryWithRefresh` covers both chat and stream paths; nil guard for graceful degradation. (w4-f) |
| PAR-ROUTE-053 | Project ID cold-miss resolution (antigravity, gemini-cli) | `src/sse/handlers/chat.js:191-198` | MISSING | Fetches project ID if missing and persists to DB. g0router has no project ID resolution. |
| PAR-ROUTE-054 | Request logging with model/provider/connection attribution | `open-sse/handlers/chatCore.js:79-82`, `open-sse/handlers/chatCore.js:135-140` | HAVE | request_log rows with model/provider/connection/endpoint attribution on all four inference endpoints; pending start/end; detail capture (w5-f) |
| PAR-ROUTE-055 | Proxy pool resolution per connection (`connectionProxyEnabled`, `vercelRelayUrl`) | `open-sse/handlers/chatCore.js:151-182` | MISSING | Resolves proxy config from connection `providerSpecificData`. g0router has no proxy pool logic. |
| PAR-ROUTE-056 | Live model catalog override (Kiro, Qoder dynamic models) | `src/app/api/v1/models/route.js:16-38`, `src/app/api/v1/models/route.js:289-299` | MISSING | Live resolvers fetch per-account model lists. g0router lists static models from providers. |
| PAR-ROUTE-057 | Custom model merging (customModels + aliasModelIds + static models) | `src/app/api/v1/models/route.js:316-348` | HAVE | `customModelsAdapter` + `aliasModelsAdapter` in `/v1/models`; order combos→catalog→custom→alias per ref:358 seen-set; `TestModelsList_MergesCustomModels`/`_MergesAliasModels`/`_DedupCustomVsCatalog` green (w6-pre) |
| PAR-ROUTE-058 | Sub-config model exposure (TTS/embedding models from provider config) | `src/app/api/v1/models/route.js:364-383` | HAVE | `subConfigModelsAdapter` reads `providerSpecificData.ttsConfig.models`/`.embeddingConfig.models`; `TestModelsList_IncludesSubConfigModels` green (w6-pre) |
| PAR-ROUTE-059 | Web search/fetch model exposure (`{alias}/search`, `{alias}/fetch`) | `src/app/api/v1/models/route.js:386-401` | MISSING | Exposes search/fetch as pseudo-models when provider has config. g0router has no search/fetch endpoints. |
| PAR-ROUTE-060 | Upstream connection detection (UUID suffix) | `src/app/api/v1/models/route.js:46`, `src/app/api/v1/models/route.js:282-284` | MISSING | `UPSTREAM_CONNECTION_RE` skips live fetch for upstream connections. g0router has no upstream connection concept. |

## Data models

### 9router (SQLite / JSON blobs)

**`combos`**
- `name TEXT PRIMARY KEY`
- `models TEXT NOT NULL` — JSON array of model strings
- `kind TEXT` — service kind (llm, webSearch, webFetch, etc.)

**`providerConnections`**
- `id TEXT PRIMARY KEY`
- `provider TEXT NOT NULL`
- `authType TEXT NOT NULL`
- `name TEXT`, `email TEXT`, `priority INTEGER`, `isActive INTEGER DEFAULT 1`
- `data TEXT NOT NULL` — JSON blob with secrets in plaintext
- Flat fields for locks: `modelLock_${model}` (ISO timestamp), `modelLock___all`
- `backoffLevel INTEGER`, `rateLimitedUntil TEXT`, `lastError TEXT`, `errorCode INTEGER`
- `lastUsedAt TEXT`, `consecutiveUseCount INTEGER`, `testStatus TEXT`

**`modelAliases`**
- `alias TEXT PRIMARY KEY`
- `model TEXT NOT NULL` — resolved value in `provider/model` format

**`providerNodes`**
- `id TEXT PRIMARY KEY`
- `type TEXT` — `openai-compatible` | `anthropic-compatible` | `custom-embedding`
- `prefix TEXT NOT NULL`
- `baseUrl TEXT`
- `apiType TEXT` — `chat` | `responses` (for openai-compatible)

**`disabledModels`**
- `providerAlias TEXT PRIMARY KEY`
- `ids TEXT NOT NULL` — JSON array of disabled model IDs

**`settings`** (single-row JSON blob)
- `comboStrategy TEXT` — global default (`fallback` or `round-robin`)
- `comboStrategies OBJECT` — per-combo overrides
- `comboStickyRoundRobinLimit INTEGER`
- `fallbackStrategy TEXT` — account fallback strategy (`fill-first` or `round-robin`)
- `providerStrategies OBJECT` — per-provider overrides
- `stickyRoundRobinLimit INTEGER` — account sticky limit
- `requireApiKey INTEGER`

### g0router (SQLite)

**`providers`**
- `id TEXT PRIMARY KEY`
- `name TEXT NOT NULL`
- `type TEXT NOT NULL`
- `base_url TEXT NOT NULL DEFAULT ''`
- `enabled INTEGER NOT NULL DEFAULT 1`
- `created_at INTEGER NOT NULL`
- `updated_at INTEGER NOT NULL`

**`connections`**
- `id TEXT PRIMARY KEY`
- `provider_id TEXT NOT NULL`
- `name TEXT NOT NULL`
- `kind TEXT NOT NULL`
- `secret_enc TEXT NOT NULL DEFAULT ''`
- `access_token_enc TEXT NOT NULL DEFAULT ''`
- `refresh_token_enc TEXT NOT NULL DEFAULT ''`
- `expires_at INTEGER NOT NULL DEFAULT 0`
- `metadata TEXT NOT NULL DEFAULT ''`
- `created_at INTEGER NOT NULL`
- `updated_at INTEGER NOT NULL`

**`VirtualKey`** (Go struct only, no table yet)
- `ID string`
- `Name string`
- `ProviderConfigs []ProviderConfig`
- `Budget *Budget`
- `RateLimitRPM *int`

**`ProviderConfig`** (Go struct only)
- `Provider string`
- `AllowedModels []string`
- `KeyIDs []string`
- `Weight *float64`

## Edge cases and quirks

### 9router

- **Combo rotation state is in-memory only**: `comboRotationState` is a `Map` that resets on server restart (`open-sse/services/combo.js:12`). This means round-robin position is lost across deploys.
- **Combo bypass before rotation**: `handleBypassRequest` runs before `getComboModels` to avoid wasting rotation slots on warmup/naming requests (`src/sse/handlers/chat.js:88-91`).
- **Sticky limit normalization**: `normalizeStickyLimit` defaults to 1 if invalid (`open-sse/services/combo.js:14-17`).
- **Account mutex blocks all concurrent selection**: `selectionMutex` is a single global promise chain; all `getProviderCredentials` calls serialize (`src/sse/services/auth.js:9-30`).
- **Model lock uses flat fields, not relational**: `modelLock_gpt-4` is a column name on the connection row, making schema changes unnecessary but querying expensive (`open-sse/services/accountFallback.js:112-114`).
- **Error rules are text-first, status-second**: A 200 response with body containing `"rate limit"` would trigger backoff because text rules precede status rules (`open-sse/config/errorConfig.js:59-76`).
- **Max rate limit cooldown hard cap**: Provider-reported `resetsAtMs` is capped at 30 minutes even if the provider says 6 hours (`open-sse/config/errorConfig.js:42`).
- **Connect timeout vs abort signal**: `BaseExecutor.execute` uses two `AbortController`s — one for connect timeout, one for client disconnect. It distinguishes connect timeout from client abort by checking `connectCtrl.signal.aborted` (`open-sse/executors/base.js:125-128`, `156-160`).
- **Free provider virtual connection lacks ID**: Injected no-auth connections use `id: "noauth"`, which `markAccountUnavailable` skips (`src/sse/services/auth.js:204`).
- **Kiro live model resolver can fail silently**: If `resolveKiroModels` throws, the catch block logs and falls back to static models (`src/app/api/v1/models/route.js:296-298`).
- **Model kind inference from ID heuristics**: When per-model type metadata is absent, `inferKindFromUnknownModelId` uses regex on the model ID string (`src/app/api/v1/models/route.js:68-74`).
- **Disabled models checked against both output and static alias**: `isDisabled` tests both `outputAlias` and `staticAlias` because provider node prefix may differ from static alias (`src/app/api/v1/models/route.js:354`).

### g0router

- **Router returns empty key values**: `Resolve` returns `schemas.Key{Value: ""}` for all providers (`internal/inference/router.go:37-54`). Empty keys yield provider auth errors until Phase 6+ wires the key store.
- **No error classification layer**: Provider errors return directly as `ProviderError` with no retry/fallback logic (`internal/api/chat.go:48-69`).
- **Stream chunk loop has no abort check**: `ChatHandler.Handle` ranges over `ch` with no select on context cancellation (`internal/api/chat.go:52-58`).
- **Models list ignores the `id` path parameter**: `ModelsHandler.Get` delegates to `List` with no filtering (`internal/api/models.go:51-54`).
- **No connection status tracking**: `connections` table has no `is_active`, `last_error`, `backoff_level`, or `rate_limited_until` columns (`internal/store/migrate.go:43-55`).
- **Settings table is key-value only**: No structured JSON for `providerStrategies`, `comboStrategies`, etc. (`internal/store/migrate.go:29-33`).
- **Virtual key schema is Go-only**: `VirtualKey` and `ProviderConfig` structs exist in `internal/schemas/governance.go` but no migration creates the table.

## Go-port considerations

- Implement combo rotation state in Redis or in-memory with TTL; SQLite is too slow for per-request sticky counter updates.
- Replace flat `modelLock_*` fields with a dedicated `connection_model_locks` table (connection_id, model, expires_at) for queryability.
- Add `internal/inference/` retry middleware between router and provider executors; keep retry config per-provider.
- Centralize error classification in `internal/inference/` using the ordered rule pattern from 9router.
- Use `fasthttp` pipeline for connect timeout instead of `AbortController` pattern.
- Implement alias resolution as a catalog lookup cache with cycle detection (DFS on alias graph at write time).
- Add `kind` filtering to `/v1/models` and introduce `/v1/models/{kind}` before Phase 9 dashboard work.
