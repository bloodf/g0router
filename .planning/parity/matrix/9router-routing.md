# ROUTE parity matrix: 9router → g0router

Reference: `/Users/heitor/Developer/github.com/bloodf/_refs/9router` (SHA `827e5c3`)

## Behavior rows

| ID | Behavior | Evidence (file:line) | g0router status | Notes |
|---|---|---|---|---|
| PAR-ROUTE-001 | Combo fallback strategy: ordered model list, try next on transient/error | `open-sse/services/combo.js:108-198`, `open-sse/services/combo.js:151-170` | MISSING | g0router has no combo system. Planned in Phase 9 (`09-models-aliases-combos/PLAN.md:21`). |
| PAR-ROUTE-002 | Combo round-robin strategy with sticky limit (requests per model before switching) | `open-sse/services/combo.js:36-65`, `tests/unit/combo-routing.test.js:25-40` | MISSING | Rotation state stored in-memory `Map` per combo name. g0router has no equivalent. |
| PAR-ROUTE-003 | Combo per-combo strategy override vs global default | `src/sse/handlers/chat.js:96-99`, `src/sse/handlers/chat.js:128-134` | MISSING | 9router checks `comboStrategies[modelStr]?.fallbackStrategy` then falls back to `settings.comboStrategy`. |
| PAR-ROUTE-004 | Combo name validation (alphanumeric, hyphens, underscores, dots) | `src/app/api/combos/route.js:7`, `src/app/api/combos/route.js:31-33` | MISSING | Regex `/^[a-zA-Z0-9_.\-]+$/`. g0router has no combo CRUD endpoint. |
| PAR-ROUTE-005 | Model alias resolution from localDb aliases map | `src/sse/services/model.js:22-25`, `open-sse/services/model.js:182-208` |  HAVE  |  Chain resolution: `internal/store/aliases.go` ResolveChain (DFS visited-set) + `internal/inference/alias.go` ResolveModelAlias. (w4-a) |
| PAR-ROUTE-006 | Provider alias to ID mapping (~140 aliases) | `open-sse/services/model.js:1-143` |  HAVE  |  ~140 aliases ported verbatim to `internal/providers/catalog/aliases.go`; ResolveProviderAlias + ForEachProviderAlias accessors. (w4-a) |
| PAR-ROUTE-007 | Provider prefix parsing (`provider/model` or `alias/model`) | `open-sse/services/model.js:155-176` |  HAVE  |  ParseModelPrefix + InferProvider (longest-alias-first) + catalog alias resolution wired into factory.go providerForModel. (w4-a) |
| PAR-ROUTE-008 | Model name prefix inference when no alias matches | `open-sse/services/model.js:248-259` |  HAVE  |  InferProvider in `internal/inference/alias.go`: sorted longest-alias-first over catalog aliases. (w4-a) |
| PAR-ROUTE-009 | Provider node prefix matching (openai-compatible, anthropic-compatible, custom-embedding) | `src/sse/services/model.js:35-51`, `src/app/api/provider-nodes/route.js:32-104` | MISSING | Dynamic nodes with prefix override static alias resolution. g0router has no provider node system. |
| PAR-ROUTE-010 | Circular alias loop validation | `docs/ARCHITECTURE.md:17` (mentioned), `.planning/phases/09-models-aliases-combos/PLAN.md:64` |  HAVE  |  DFS cycle detection in store.ResolveChain visited-set; CreateAlias rejects cycles at write time. g0router-defensive (ref has no impl). (w4-a) |
| PAR-ROUTE-011 | Combo recursive resolution protection | `src/sse/handlers/chat.js:122-147` | MISSING | 9router `handleSingleModelChat` re-checks `getComboModels` when `modelInfo.provider` is null to prevent infinite recursion. g0router has no combo handling. |
| PAR-ROUTE-012 | Per-model account locks (`modelLock_${model}`) | `open-sse/services/accountFallback.js:106-114`, `src/sse/services/auth.js:203-241` | MISSING | Flat field `modelLock_${model}` on connection record. g0router connection schema has no model lock columns (`internal/store/migrate.go:43-55`). |
| PAR-ROUTE-013 | Account-level lock (`modelLock___all`) | `open-sse/services/accountFallback.js:109`, `open-sse/services/accountFallback.js:120-125` | MISSING | Special key when no model is known. g0router has no equivalent. |
| PAR-ROUTE-014 | Account cooldown with exponential backoff (`backoffLevel`) | `open-sse/services/accountFallback.js:9-13`, `open-sse/services/accountFallback.js:31-34`, `open-sse/config/errorConfig.js:32-35` | MISSING | Level 1→1s, 2→2s, 3→4s… capped at 4 min. g0router has no backoff state. |
| PAR-ROUTE-015 | Account state reset on success (clear model lock + backoff) | `src/sse/services/auth.js:252-285`, `open-sse/services/accountFallback.js:184-193` | MISSING | On success clears `modelLock_*`, resets `backoffLevel` to 0, `testStatus` to active. g0router has no error state tracking on connections. |
| PAR-ROUTE-016 | Provider account fallback loop (`excludeConnectionIds`) | `src/sse/handlers/chat.js:162-245` | MISSING | 9router loops over accounts, excluding failed ones. g0router resolves a single provider+key (`internal/inference/router.go:33-54`). |
| PAR-ROUTE-017 | Account selection mutex (prevents race conditions) | `src/sse/services/auth.js:9`, `src/sse/services/auth.js:24-30` | MISSING | Promise-chain mutex guards `getProviderCredentials`. g0router has no account selection mutex. |
| PAR-ROUTE-018 | Account selection strategies (`fill-first`, `round-robin`) | `src/sse/services/auth.js:102-157` | MISSING | `fill-first` uses priority sorting; `round-robin` uses `lastUsedAt` + `consecutiveUseCount`. g0router has no strategy selection. |
| PAR-ROUTE-019 | Sticky round-robin limit for accounts | `src/sse/services/auth.js:116`, `src/sse/services/auth.js:129-136` | MISSING | Configurable `stickyRoundRobinLimit` (default 3). g0router has no sticky limit. |
| PAR-ROUTE-020 | Per-URL retry with configurable attempts/delay by status code | `open-sse/executors/base.js:98-174`, `open-sse/config/runtimeConfig.js:52-57` |  HAVE  |  newDefaultRetryConfig() in `internal/inference/retry.go`: 429→0, 502→3/3s, 503→3/2s, 504→2/3s. (w4-b) |
| PAR-ROUTE-021 | Provider-specific retry config override | `open-sse/executors/base.js:105`, `open-sse/config/providers.js:197` |  HAVE  |  Catalog `Retry map[int]int` field; kiro override {429:2} applied per-provider in retry middleware. (w4-b) |
| PAR-ROUTE-022 | Connect timeout abort and retry mapping to 502 | `open-sse/executors/base.js:125-128`, `open-sse/executors/base.js:156-163` |  HAVE  |  net.Error.Timeout() at fasthttp boundary → 502, not retried. ARCH: fasthttp has no distinct dial-timeout type; Timeout()+!Temporary() is the Go/fasthttp idiom. (w4-b) |
| PAR-ROUTE-023 | Token refresh on 401/403 with retry | `open-sse/handlers/chatCore.js:216-235` | MISSING | `refreshWithRetry` up to 3 attempts. g0router has no token refresh in the inference path. |
| PAR-ROUTE-024 | Combo transient error cooldown (503/502/504 waits up to 5s before next model) | `open-sse/services/combo.js:161-165` | MISSING | Waits `cooldownMs` when `status ∈ {502,503,504}` and `cooldownMs ≤ 5000`. g0router has no combo handling. |
| PAR-ROUTE-025 | Disabled model tracking per provider alias | `src/app/api/models/disabled/route.js:1-50`, `src/app/api/v1/models/route.js:190-191` | MISSING | `disabledModelsDb` stores `{providerAlias: [modelId]}`. g0router has no disabled model tracking. |
| PAR-ROUTE-026 | Disabled models excluded from `/v1/models` | `src/app/api/v1/models/route.js:354`, `src/app/api/v1/models/route.js:225` | MISSING | `isDisabled(alias, modelId)` filters list. g0router lists all models from provider (`internal/api/models.go:23-48`). |
| PAR-ROUTE-027 | Weighted provider selection | `.planning/phases/08-keys-virtualkeys-routing/PLAN.md:23` | MISSING | 9router has no explicit weighted selection for providers. g0router Phase 8 plans weighted provider selection but it is not implemented. |
| PAR-ROUTE-028 | API key validation (`requireApiKey` setting) | `src/sse/handlers/chat.js:68-80` | MISSING | 9router validates `Bearer` or `x-api-key` against SQLite when `requireApiKey=true`. g0router `/v1` routes are public with no key check (`internal/server/routes_openai.go:15-18`). |
| PAR-ROUTE-029 | API key extraction (`Bearer`, `x-api-key`) | `src/sse/services/auth.js:290-304` | MISSING | Checks `Authorization: Bearer` first, then `x-api-key`. g0router has no API key extraction. |
| PAR-ROUTE-030 | Virtual key routing via `x-g0-vk` header | `.planning/phases/08-keys-virtualkeys-routing/PLAN.md:46` | MISSING | 9router has no virtual key header routing. g0router Phase 8 plans `x-g0-vk` header but it is not implemented. |
| PAR-ROUTE-031 | Per-key quota tracking | `.planning/phases/08-keys-virtualkeys-routing/PLAN.md:25` | MISSING | 9router has no per-key quota. g0router Phase 8 plans per-key quota but it is not implemented. |
| PAR-ROUTE-032 | Virtual key budget and rate limit RPM schema | `internal/schemas/governance.go:4-25` | HAVE | `VirtualKey` struct defines `Budget` and `RateLimitRPM`. 9router has no virtual key schema. |
| PAR-ROUTE-033 | Request format auto-detection (OpenAI, Claude, Gemini, Antigravity, Responses) | `open-sse/services/provider.js:49-126` | MISSING | 9router inspects body shape to detect format. g0router assumes OpenAI format for chat (`internal/api/chat.go:24-28`). |
| PAR-ROUTE-034 | Bypass patterns for Claude CLI (warmup, count, title, skip, naming) | `open-sse/utils/bypassHandler.js:11-91` | MISSING | Returns fake responses for CLI patterns without calling provider. g0router has no bypass handler. |
| PAR-ROUTE-035 | Provider-specific URL building with fallback URLs | `open-sse/services/provider.js:155-209`, `open-sse/executors/base.js:20-42` | MISSING | Multiple `baseUrls` per provider; index-based fallback. g0router providers have single hard-coded URLs. |
| PAR-ROUTE-036 | Provider-specific header building (auth, spoofing, streaming) | `open-sse/services/provider.js:212-320` | MISSING | Per-provider auth headers (Bearer, x-api-key, x-goog-api-key, GitHub spoof, Claude spoof). g0router uses generic headers per provider package. |
| PAR-ROUTE-037 | Model kind routing (`/v1/models/{kind}`) | `src/app/api/v1/models/[kind]/route.js:1-55` | MISSING | Supports `image`, `tts`, `stt`, `embedding`, `image-to-text`, `web`. g0router has `GET /v1/models` only (`internal/server/routes_openai.go:17`). |
| PAR-ROUTE-038 | Model test routing by kind (llm, image, embedding, stt) | `src/app/api/models/test/route.js:1-14`, `tests/unit/model-test-routing.test.js:47-166` | MISSING | Routes to internal `/v1/images/generations`, `/v1/embeddings`, `/v1/audio/transcriptions`. g0router has no model test endpoint. |
| PAR-ROUTE-039 | Free provider no-auth virtual connection injection | `src/sse/services/auth.js:36-53`, `src/shared/constants/providers.js:14` | MISSING | Injects synthetic connection for `noAuth` providers (e.g. opencode). g0router has no free-provider virtual connection logic. |
| PAR-ROUTE-040 | OpenAI-compatible and Anthropic-compatible provider node routing | `src/sse/services/model.js:35-51`, `src/app/api/provider-nodes/route.js:32-104` | MISSING | Dynamic nodes with prefix-based routing. g0router has no provider node system. |
| PAR-ROUTE-041 | Native passthrough detection (same ecosystem, skip translation) | `open-sse/handlers/chatCore.js:86-103` | MISSING | Detects when client tool and provider are same ecosystem. g0router has no passthrough logic. |
| PAR-ROUTE-042 | Provider thinking config override injection | `open-sse/handlers/chatCore.js:48-58` | MISSING | Injects `thinking` or `reasoning_effort` based on provider-level config when client omits it. g0router has no thinking override. |
| PAR-ROUTE-043 | Streaming vs non-streaming decision logic | `open-sse/handlers/chatCore.js:60-77` | MISSING | Considers `providerRequiresStreaming`, `clientPrefersJson`, `deepseek-tui` detection. g0router uses `req.Stream` boolean directly (`internal/api/chat.go:41`). |
| PAR-ROUTE-044 | Error classification rules (text-first then status) | `open-sse/config/errorConfig.js:59-76` |  HAVE  |  classificationRules() in `internal/inference/errorclass.go`: 9 text rules then 5 status rules, exact errorConfig.js sequence. (w4-b) |
| PAR-ROUTE-045 | Precise cooldown override from provider-reported `resetsAtMs` | `src/sse/services/auth.js:211-214`, `open-sse/services/accountFallback.js:49` |  HAVE  |  extractResetsAt: resets_at (sec/ms) + resets_in_seconds; hard-capped at maxRateLimitCooldown (30min). (w4-b) |
| PAR-ROUTE-046 | Earliest retry-after tracking across combo models | `open-sse/services/combo.js:113`, `open-sse/services/combo.js:141-143`, `open-sse/services/combo.js:187-191` | MISSING | Tracks earliest `retryAfter` across all combo models for consolidated error response. g0router has no combo handling. |
| PAR-ROUTE-047 | Model promotion rules (combo names appear first in `/v1/models`) | `src/app/api/v1/models/route.js:202-213` | MISSING | Combos listed before provider models. g0router has no combo concept. |
| PAR-ROUTE-048 | Quota window parsing (unix timestamps in seconds or milliseconds) | `open-sse/services/usage.js:103-131` |  HAVE  |  parseTimestamp: <1e12 treated as seconds (×1000), ≥1e12 as milliseconds. (w4-b) |
| PAR-ROUTE-049 | Group lock semantics (all accounts locked → return earliest expiry) | `src/sse/services/auth.js:80-98` | MISSING | When all accounts locked, returns `allRateLimited=true` with `retryAfterHuman`. g0router has single account per provider. |
| PAR-ROUTE-050 | Per-provider strategy override (`providerStrategies`) | `src/sse/services/auth.js:101-103` | MISSING | `settings.providerStrategies[providerId].fallbackStrategy` overrides global. g0router has no provider strategy config. |
| PAR-ROUTE-051 | Pinned connection preference | `src/sse/services/auth.js:106-112` | MISSING | `preferredConnectionId` option skips strategy. g0router has no connection pinning. |
| PAR-ROUTE-052 | Provider credentials refresh before dispatch | `src/sse/handlers/chat.js:188-198` | MISSING | `checkAndRefreshToken` called before `handleChatCore`. g0router has no refresh in inference path. |
| PAR-ROUTE-053 | Project ID cold-miss resolution (antigravity, gemini-cli) | `src/sse/handlers/chat.js:191-198` | MISSING | Fetches project ID if missing and persists to DB. g0router has no project ID resolution. |
| PAR-ROUTE-054 | Request logging with model/provider/connection attribution | `open-sse/handlers/chatCore.js:79-82`, `open-sse/handlers/chatCore.js:135-140` | MISSING | Logs model, provider, connection, proxy config. g0router has no request logger. |
| PAR-ROUTE-055 | Proxy pool resolution per connection (`connectionProxyEnabled`, `vercelRelayUrl`) | `open-sse/handlers/chatCore.js:151-182` | MISSING | Resolves proxy config from connection `providerSpecificData`. g0router has no proxy pool logic. |
| PAR-ROUTE-056 | Live model catalog override (Kiro, Qoder dynamic models) | `src/app/api/v1/models/route.js:16-38`, `src/app/api/v1/models/route.js:289-299` | MISSING | Live resolvers fetch per-account model lists. g0router lists static models from providers. |
| PAR-ROUTE-057 | Custom model merging (customModels + aliasModelIds + static models) | `src/app/api/v1/models/route.js:316-348` | MISSING | Merges three sources with deduplication. g0router has no custom model or alias support. |
| PAR-ROUTE-058 | Sub-config model exposure (TTS/embedding models from provider config) | `src/app/api/v1/models/route.js:364-383` | MISSING | Exposes `ttsConfig.models` and `embeddingConfig.models`. g0router providers expose only chat models. |
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
