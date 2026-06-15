# Bifrost Governance Parity Matrix (BF-GOV)

Reference: `/Users/heitor/Developer/github.com/bloodf/_refs/bifrost` @ `ca21298`
Target: `/Users/heitor/Developer/github.com/bloodf/g0router`

---

## Row Table

| ID | Behavior | Evidence (file:line) | g0router status | Notes |
|---|---|---|---|---|
| PAR-BF-GOV-001 | Virtual key schema: ID, Name, Value, IsActive, TeamID/CustomerID (mutually exclusive), RateLimitID, CalendarAligned, Budgets, ProviderConfigs, MCPConfigs | `framework/configstore/tables/virtualkey.go:208-241` | PARTIAL | bf-gov-1: VK→Team link HAVE — `team_id TEXT NOT NULL DEFAULT ''` additive col (`migrate.go`) + `store.VirtualKey.TeamID`/`schemas.VirtualKey.TeamID` (D4); `IsActive` reconciled to NOT-NULL-DEFAULT-1 col (D1, VAR=nil-means-active). CustomerID/RateLimitID/CalendarAligned/MCPConfigs subset ESC. g0router base: `internal/schemas/governance.go:4-10` |
| PAR-BF-GOV-002 | VK provider config: Provider, Weight, AllowedModels (schemas.WhiteList), BlacklistedModels (schemas.BlackList), AllowAllKeys, Keys (many2many join table), Budgets, RateLimit | `framework/configstore/tables/virtualkey.go:26-40` | PARTIAL | bf-gov-1: `AllowAllKeys` HAVE — `schemas.ProviderConfig.AllowAllKeys` + `api.VKProviderConfig.AllowAllKeys` consumed in `AllowVK` keyIDs branch (true ⇒ no pin / fall-through, D5). BlacklistedModels → bf-gov-2; many2many Keys join ESC (g0router pins via `KeyIDs []string`). g0router base: `internal/schemas/governance.go:13-19` |
| PAR-BF-GOV-003 | VK MCP config: VirtualKeyID, MCPClientID, ToolsToExecute (schemas.WhiteList) | `framework/configstore/tables/virtualkey.go:159-175` | MISSING | No MCP config on VK in g0router |
| PAR-BF-GOV-004 | VK in-memory lookup via `sync.Map` (lock-free, value-hash indexed) | `plugins/governance/store.go:858` | MISSING | No VK store or lookup exists in g0router |
| PAR-BF-GOV-005 | VK `CalendarAligned` propagated to owned budgets and rate limits via `AfterFind` | `framework/configstore/tables/virtualkey.go:291-318` | MISSING | No calendar-aligned logic in g0router |
| PAR-BF-GOV-006 | VK value encrypted at rest: SHA-256 hash for lookup, vault or AES encryption in `BeforeSave` | `framework/configstore/tables/virtualkey.go:260-284` | MISSING | g0router has no VK table or encryption logic |
| PAR-BF-GOV-007 | Team schema: ID, Name, CustomerID, RateLimitID, Budgets, RateLimit, VirtualKeys, VirtualKeyCount (computed), CalendarAligned | `framework/configstore/tables/team.go:12-45` | HAVE | bf-gov-1: Team budget+RPM subset HAVE — `teams` table + `store.Team` pre-existing (CRUD complete); bf-gov-1 CONSUMES it for the 2-level hierarchy (`GetTeamByID` read-only in `storeVKToAPI`). CalendarAligned (bf-gov-3) / CustomerID (Customer ESC) excluded by design. `internal/store/teams.go:11-20` |
| PAR-BF-GOV-008 | Customer schema: ID, Name, RateLimitID, Budgets, RateLimit, Teams, VirtualKeys, CalendarAligned | `framework/configstore/tables/customer.go:10-32` | MISSING | No Customer type or table in g0router Go backend |
| PAR-BF-GOV-009 | VK ownership mutual exclusion enforced in `BeforeSave`: error if both TeamID and CustomerID non-nil | `framework/configstore/tables/virtualkey.go:260-264` | HAVE | bf-gov-1: HAVE-by-design (D6) — with CustomerID ESC, the only VK owner is TeamID, so TeamID-XOR-CustomerID is satisfied by construction (no second owner field). GORM BeforeSave mechanism ESC; a future Customer-tier plan owns reinstating the XOR. |
| PAR-BF-GOV-010 | Hierarchical evaluation order: Provider/Model → User → VK → Team → Customer | `plugins/governance/resolver.go:83-330` | PARTIAL | bf-gov-1: VK→Team slice HAVE — `QuotaEngine.Allow` runs VK budget+RPM then Team budget+RPM, both must pass, fail-closed at first failing level (D3 precedence). Provider/Model/User/Customer tiers ESC. `internal/governance/quota.go` (Allow + checkTeamBudget/checkTeamRPM) |
| PAR-BF-GOV-011 | Budget schema: ID, MaxLimit, ResetDuration, LastReset, CurrentUsage; single-owner FKs (TeamID, VirtualKeyID, ProviderConfigID, ModelConfigID, CustomerID) | `framework/configstore/tables/budget.go:11-44` | PARTIAL | g0router has `Budget` struct with Limit, Period, Used only: `internal/schemas/governance.go:21-25` |
| PAR-BF-GOV-012 | Budget single-owner enforcement in `BeforeSave`: error if more than one owner FK is non-nil | `framework/configstore/tables/budget.go:50-70` | HAVE | bf-gov-1: inline single-owner validation HAVE (D2) — `governance.ValidateBudgetOwner(BudgetOwner{VirtualKeyID,TeamID})` errors when >1 of {VK,Team} owner is set; consumed live in the VK admin create/update path (`internal/admin/virtualkeys.go:93`). GORM BeforeSave mechanism ESC; behavior built inline (errors-as-values). |
| PAR-BF-GOV-013 | Budget atomic CAS increment via `BumpBudgetUsage`: auto-resets if ResetDuration expired, retries on CAS failure | `plugins/governance/store.go:375-410` | MISSING | No budget accumulation logic in g0router |
| PAR-BF-GOV-014 | Budget hierarchy evaluation: VK-level budgets → Team-level budgets → Customer-level budgets via `collectBudgetsFromHierarchy` | `plugins/governance/store.go:977-998` | PARTIAL | bf-gov-1: VK→Team budget hierarchy HAVE — Team budget enforced via the LIVE `SumCostByTeam` aggregate over `request_log` joined through `virtual_keys.team_id` vs team `budget_usd` (D8, NOT the display-only `budget_used_usd`). Customer ESC. `internal/store/requestlog.go` SumCostByTeam + `quota.go` checkTeamBudget |
| PAR-BF-GOV-015 | Budget calendar-aligned reset: `IsCalendarAligned` derived from owner; reset path reads stamped value | `framework/configstore/tables/budget.go:36` | MISSING | g0router spec mentions lazy reset: `docs/phases/phase-18-bifrost-features.md:72-73` |
| PAR-BF-GOV-016 | Budget DB sync every 10s: `DumpBudgets` writes in-memory `CurrentUsage` to database | `plugins/governance/store.go:1968` | MISSING | No DB sync for budgets |
| PAR-BF-GOV-017 | Rate limit schema: dual token/request limits with MaxLimit, ResetDuration, CurrentUsage, LastReset per dimension | `framework/configstore/tables/ratelimit.go:11-42` | PARTIAL | g0router has `RateLimitRPM *int` on VirtualKey only: `internal/schemas/governance.go:9` |
| PAR-BF-GOV-018 | Rate limit atomic CAS increment via `BumpRateLimitUsage`: auto-resets expired counters, retries on CAS failure | `plugins/governance/store.go:412-456` | MISSING | No rate-limit tracking in g0router |
| PAR-BF-GOV-019 | Rate limit calendar-aligned reset support via `GetCalendarPeriodStart` | `plugins/governance/store.go:1750` (referenced by reset worker) | MISSING | No calendar-aligned reset logic |
| PAR-BF-GOV-020 | Rate limit hierarchy evaluation: VK-level → provider-config-level → Team-level → Customer-level | `plugins/governance/store.go:1522-1538` | PARTIAL | bf-gov-1: VK→Team RPM hierarchy HAVE — Team RPM from `teams.rate_limit_rpm` enforced via the engine's in-memory rpm window keyed by synthetic `team:<id>` after the VK RPM check (D3). Provider-config-level RPM + Customer ESC. `internal/governance/quota.go` checkTeamRPM |
| PAR-BF-GOV-021 | `UsageUpdate` struct with streaming-aware fields: IsStreaming, IsFinalChunk, HasUsageData | `plugins/governance/tracker.go:17-31` | MISSING | No UsageUpdate type in g0router |
| PAR-BF-GOV-022 | `UpdateUsage` order: global provider/model → user-level → per-user scoped model → VK-level → per-VK scoped model | `plugins/governance/tracker.go:70-173` | MISSING | No usage update pipeline in g0router |
| PAR-BF-GOV-023 | Background reset worker: every 10s (`workerInterval`) resets expired rate limits and budgets, dumps to DB | `plugins/governance/tracker.go:49,199` | MISSING | No background workers for governance |
| PAR-BF-GOV-024 | Log schema with denormalized governance fields: VirtualKeyID, VirtualKeyName, TeamID, TeamName, CustomerID, CustomerName, BudgetIDs, RateLimitIDs | `framework/logstore/tables.go:127-269` | MISSING | `request_log` table design exists in docs only: `docs/SCHEMA.md:85-119` |
| PAR-BF-GOV-025 | `CollectApplicableGovernanceIDs` returns every budget and rate-limit ID charged for a request; stamped on log row for ghost-node reconciliation | `plugins/governance/store.go:200` | MISSING | No ghost-node reconciliation in g0router |
| PAR-BF-GOV-026 | `WhiteList` semantics: `["*"]` allows all values; empty list denies all; non-empty list without `*` allows only listed values | `core/schemas/account.go:22-30` | MISSING | g0router uses `[]string` for AllowedModels with no semantics enforced |
| PAR-BF-GOV-027 | `BlackList` semantics: `["*"]` blocks all values; empty list blocks none; non-empty list without `*` blocks only listed values | `core/schemas/account.go:80-106` | MISSING | No BlackList type in g0router |
| PAR-BF-GOV-028 | Blacklist wins over allowlist: two-pass check (blacklist scan first, then allowlist scan) in `isModelAllowed` | `plugins/governance/resolver.go:358-390` | MISSING | No model filtering logic in g0router |
| PAR-BF-GOV-029 | Provider key-level blacklists: `TableKey.BlacklistedModels` runtime field persisted as `BlacklistedModelsJSON` | `framework/configstore/tables/key.go:24,83` | MISSING | g0router `schemas.Key` has no blacklisted_models field |
| PAR-BF-GOV-030 | VK provider config-level blacklists: `TableVirtualKeyProviderConfig.BlacklistedModels` persisted as JSON | `framework/configstore/tables/virtualkey.go:32` | MISSING | g0router `ProviderConfig` has no blacklisted_models field |
| PAR-BF-GOV-031 | Model catalog integration for cross-provider allowlist matching: `IsModelAllowedForProvider` delegates to catalog | `plugins/governance/main.go:467-479` | MISSING | No model catalog integration in g0router |
| PAR-BF-GOV-032 | `EvaluateGovernanceRequest` stamps VK/Team/Customer IDs and names into `BifrostContext` for downstream logging | `plugins/governance/main.go:763-799` | MISSING | No governance context stamping in g0router |
| PAR-BF-GOV-033 | `loadBalanceProvider` performs weighted random selection across eligible provider configs; blacklist pre-pass excludes blocked providers | `plugins/governance/main.go:405-479` | MISSING | g0router router is prefix-based only: `internal/inference/router.go:33-54` |
| PAR-BF-GOV-034 | Virtual key mandatory mode: rejects requests without `x-bf-vk` header when `isVkMandatory` is true | `plugins/governance/main.go:783-797` | MISSING | No VK mandatory mode in g0router |
| PAR-BF-GOV-035 | `Decision` enum for governance outcomes: Allow, VirtualKeyNotFound, VirtualKeyBlocked, RateLimited, BudgetExceeded, TokenLimited, RequestLimited, ModelBlocked, ProviderBlocked, MCPToolBlocked | `plugins/governance/resolver.go:14-27` | MISSING | No Decision enum in g0router |
| PAR-BF-GOV-036 | `EvaluationResult` carries Decision, Reason, VirtualKey, RateLimitInfo, BudgetInfo, UsageInfo | `plugins/governance/resolver.go:37-62` | MISSING | No evaluation result type in g0router |
| PAR-BF-GOV-037 | `BeforeSave` on `TableVirtualKeyProviderConfig` validates `AllowedModels` and `BlacklistedModels` via `Validate()` | `framework/configstore/tables/virtualkey.go:82-90` | MISSING | No validation hooks in g0router |
| PAR-BF-GOV-038 | `BeforeSave` on `TableRateLimit` validates reset duration format and requires reset duration when max limit is set | `framework/configstore/tables/ratelimit.go:48-87` | MISSING | No rate limit validation in g0router |
| PAR-BF-GOV-039 | `CheckBudget` treats expired budgets (rolling window elapsed) as reset by skipping the check | `plugins/governance/store.go:946-974` | MISSING | No budget check logic in g0router |
| PAR-BF-GOV-040 | `CheckRateLimit` checks token and request limits against local usage + remote baseline for multi-node clusters | `plugins/governance/store.go:870` | MISSING | No rate limit check logic in g0router |
| PAR-BF-GOV-041 | `UpdateVirtualKeyBudgetUsageInMemory` walks hierarchy and bumps every applicable budget atomically | `plugins/governance/store.go:1541-1554` | MISSING | No budget usage update in g0router |
| PAR-BF-GOV-042 | Provider-level budgets and rate limits checked before VK-level in `EvaluateModelAndProviderRequest` | `plugins/governance/resolver.go:83-127` | PARTIAL | bf-gov-1: VK→Team ownership precedence HAVE — the VK→Team owner slice is evaluated deterministically (VK before Team, D3). The Provider/Model-level budget/RPM tier ESC (not in the phase-18 2-level design). `internal/governance/quota.go` Allow precedence |
| PAR-BF-GOV-043 | Scoped model config budgets/rate limits (VK-scoped and user-scoped) aggregate with global model checks | `plugins/governance/resolver.go:314-329` | MISSING | No scoped model configs in g0router |
| PAR-BF-GOV-044 | `UsageTracker.PerformStartupResets` checks ALL virtual keys (active and inactive) for expired rate limits on startup | `plugins/governance/tracker.go:224-350` | MISSING | No startup reset logic in g0router |
| PAR-BF-GOV-045 | `UsageTracker.Cleanup` flushes in-memory deltas to DB before shutdown | `plugins/governance/tracker.go:354-376` | MISSING | No graceful shutdown flush for governance |
| PAR-BF-GOV-046 | `TableTeam.AfterFind` propagates `CalendarAligned` down to owned budgets and rate limit | `framework/configstore/tables/team.go:94-116` | MISSING | No team-level calendar alignment in g0router |
| PAR-BF-GOV-047 | `TableCustomer.AfterFind` propagates `CalendarAligned` down to owned budgets and rate limit | `framework/configstore/tables/customer.go:39-46` | MISSING | No customer-level calendar alignment in g0router |
| PAR-BF-GOV-048 | `TableKey.AfterFind` deserializes `BlacklistedModelsJSON` into `BlacklistedModels` runtime field | `framework/configstore/tables/key.go:662-666` | MISSING | No key-level blacklist deserialization in g0router |
| PAR-BF-GOV-049 | `LocalGovernanceStore` uses `sync.Map` for lock-free reads on virtualKeys, teams, customers, budgets, rateLimits, modelConfigs, providers, routingRules | `plugins/governance/store.go:27-35` | MISSING | No in-memory governance store in g0router |
| PAR-BF-GOV-050 | `GovernanceStore` interface defines 40+ methods for CRUD, checks, updates, dumps, and routing rule CEL compilation | `plugins/governance/store.go:90-200` | MISSING | No governance store interface in g0router |

---

## Data Models

### Bifrost (Reference)

**TableVirtualKey** (`framework/configstore/tables/virtualkey.go:208-241`)
- `ID string` (PK)
- `Name string` (unique, not null)
- `Value string` (not null, encrypted at rest)
- `IsActive *bool` (nil = true)
- `TeamID *string` (FK, mutually exclusive with CustomerID)
- `CustomerID *string` (FK)
- `RateLimitID *string` (FK)
- `CalendarAligned bool`
- `ProviderConfigs []TableVirtualKeyProviderConfig`
- `MCPConfigs []TableVirtualKeyMCPConfig`
- `Budgets []TableBudget`
- `ConfigHash string`
- `CreatedByUserID *string`
- `CreatedAt time.Time`
- `UpdatedAt time.Time`

**TableVirtualKeyProviderConfig** (`framework/configstore/tables/virtualkey.go:26-40`)
- `ID uint` (PK, autoIncrement)
- `VirtualKeyID string`
- `Provider string`
- `Weight *float64`
- `Allowed schemas.WhiteList` (JSON)
- `BlacklistedModels schemas.BlackList` (JSON)
- `AllowAllKeys bool`
- `RateLimitID *string`
- `Budgets []TableBudget`
- `Keys []TableKey` (many2many via `governance_virtual_key_provider_config_keys`)

**TableTeam** (`framework/configstore/tables/team.go:12-45`)
- `ID string` (PK)
- `Name string` (unique, not null)
- `CustomerID *string` (FK)
- `RateLimitID *string`
- `Budgets []TableBudget`
- `RateLimit *TableRateLimit`
- `VirtualKeys []TableVirtualKey`
- `VirtualKeyCount int64` (computed)
- `CalendarAligned bool`

**TableCustomer** (`framework/configstore/tables/customer.go:10-32`)
- `ID string` (PK)
- `Name string` (not null)
- `RateLimitID *string`
- `Budgets []TableBudget`
- `RateLimit *TableRateLimit`
- `Teams []TableTeam`
- `VirtualKeys []TableVirtualKey`
- `CalendarAligned bool`

**TableBudget** (`framework/configstore/tables/budget.go:11-44`)
- `ID string` (PK)
- `MaxLimit float64` (not null)
- `ResetDuration string` (e.g. "1h", "1d", "1M")
- `LastReset time.Time`
- `CurrentUsage float64` (default 0)
- `TeamID *string`
- `VirtualKeyID *string`
- `ProviderConfigID *uint`
- `ModelConfigID *string`
- `CustomerID *string`
- `IsCalendarAligned bool` (derived, not persisted)

**TableRateLimit** (`framework/configstore/tables/ratelimit.go:11-42`)
- `ID string` (PK)
- `TokenMaxLimit *int64`
- `TokenResetDuration *string`
- `TokenCurrentUsage int64`
- `TokenLastReset time.Time`
- `RequestMaxLimit *int64`
- `RequestResetDuration *string`
- `RequestCurrentUsage int64`
- `RequestLastReset time.Time`
- `IsCalendarAligned bool` (derived, not persisted)

**TableKey** (`framework/configstore/tables/key.go:16-92`)
- `ID uint` (PK)
- `Name string` (unique)
- `ProviderID uint`
- `KeyID string` (unique UUID)
- `Value schemas.EnvVar`
- `Models schemas.WhiteList` (runtime)
- `BlacklistedModels schemas.BlackList` (runtime, persisted as `BlacklistedModelsJSON`)
- `Weight *float64`
- `Enabled *bool`

### g0router (Target)

**VirtualKey** (`internal/schemas/governance.go:4-10`)
- `ID string`
- `Name string`
- `ProviderConfigs []ProviderConfig`
- `Budget *Budget`
- `RateLimitRPM *int`

**ProviderConfig** (`internal/schemas/governance.go:13-18`)
- `Provider string`
- `AllowedModels []string`
- `KeyIDs []string`
- `Weight *float64`

**Budget** (`internal/schemas/governance.go:21-25`)
- `Limit float64`
- `Period string`
- `Used float64`

No Team, Customer, RateLimit, or Key schema with blacklists exists in the Go backend.

---

## Edge Cases and Quirks

1. **Nil IsActive means true**: `TableVirtualKey.IsActive` is a `*bool`; nil resolves to true via `IsActiveValue()`: `framework/configstore/tables/virtualkey.go:247-255`.

2. **Deny-by-default on empty ProviderConfigs**: Empty `ProviderConfigs` on a VK means no providers and no models are allowed: `plugins/governance/resolver.go:359-362,393-397`.

3. **Empty Keys with AllowAllKeys=false means no keys allowed**: `framework/configstore/tables/virtualkey.go:33` comment and `plugins/governance/resolver.go:335-343` enforce this.

4. **Budget owner mutual exclusion**: A budget cannot have more than one of TeamID, VirtualKeyID, ProviderConfigID, ModelConfigID, CustomerID set. `BeforeSave` counts owners and errors if >1: `framework/configstore/tables/budget.go:50-70`.

5. **Rate limit requires reset duration when max limit is set**: `BeforeSave` errors if `TokenMaxLimit != nil && TokenResetDuration == nil`: `framework/configstore/tables/ratelimit.go:68-74`.

6. **Streaming-aware usage tracking**: Only increments tokens on chunks with `HasUsageData=true`; only increments requests on `IsFinalChunk=true`: `plugins/governance/tracker.go:78-80`.

7. **CAS retry loop never drops increments**: `BumpBudgetUsage` and `BumpRateLimitUsage` spin on `CompareAndSwap` until success: `plugins/governance/store.go:385-410,418-456`.

8. **Expired budgets are skipped (not reset) during check**: `CheckBudget` continues without enforcing a budget whose rolling window has elapsed; actual reset happens later in the background worker: `plugins/governance/store.go:949-957`.

9. **Model catalog overrides simple string matching for allowlists**: When `modelCatalog` and `inMemoryStore` are present, `isModelAllowed` delegates to `IsModelAllowedForProvider` to handle cross-provider model names (e.g. OpenRouter, Vertex): `plugins/governance/resolver.go:374-384`.

10. **Ghost-node reconciliation via log-stamped governance IDs**: `CollectApplicableGovernanceIDs` stamps every budget and rate-limit ID onto the log row so cluster leaders can re-attribute usage from dead nodes: `plugins/governance/store.go:198-200`.

11. **VK value hash computed before encryption**: `BeforeSave` hashes plaintext before encrypting, so the hash column indexes the original value: `framework/configstore/tables/virtualkey.go:267-269`.

12. **Calendar alignment is owner-propagated, not persisted on budget/rate-limit rows**: `IsCalendarAligned` is set in `AfterFind` hooks on VK/Team/Customer and re-stamped on every in-memory update: `framework/configstore/tables/virtualkey.go:302-316`, `framework/configstore/tables/team.go:110-115`, `framework/configstore/tables/customer.go:40-44`.

---

## Go-Port Considerations

1. Replace GORM with `database/sql` + `ensureColumn` additive migrations.
2. Replace `sync.Map` with `sync.RWMutex` + typed maps for stronger typing.
3. Implement `WhiteList` and `BlackList` as first-class types with `Validate()`, `IsAllowed()`, `IsBlocked()` methods.
4. Port the CAS-retry budget/rate-limit bump pattern exactly; it prevents lost increments under concurrency.
5. g0router uses SQLite; Bifrost uses PostgreSQL. JSON columns become `TEXT` with `json.Marshal`/`Unmarshal`.
6. The `EvaluationResult` → `BifrostError` mapping in `EvaluateGovernanceRequest` should return g0router's existing error envelope format (`{data, error}`).
7. Bifrost's `UsageTracker` background worker (10s ticker) maps to a g0router `time.Ticker` in the server lifecycle.
8. g0router lacks a `modelCatalog`; the cross-provider allowlist check in `isModelAllowed` needs a substitute or simplification.
