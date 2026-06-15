# Bifrost parity program — strategic micro-plan MAP

Author: Planner (g0router parity lead). Orchestrator: per `CLI_ORCHESTRATOR.md`
(W6+ rule; `HANDOFF.md` no longer governs). The 9router parity program is
**COMPLETE** (HAVE=384, 61 documented escalations; Waves 0–7 merged per
`docs/WORKFLOW.md`). This is the **next** program: parity against **Bifrost**, a
*different* LLM-gateway product, across four matrices (~257 MISSING/PARTIAL).

**Non-authorizing INDEX** (same status as WAVE-2..7-MAP). This MAP sequences the
work and assigns row ownership so per-plan micro-plans (`bf-core-1.md`, …) can be
authored afterward. Binary acceptance criteria, exact task steps, precondition
greps, and stop conditions live in the individual micro-plan files, written
before each plan's dispatch per `CLI_ORCHESTRATOR.md` §5/§9.3.

---

## Goal

Assess and (where buildable) close the four **Bifrost** parity matrices:

- `matrix/bifrost-core.md` — 1 HAVE, 3 PARTIAL, 46 MISSING (50 rows)
- `matrix/bifrost-governance.md` — 0 HAVE, 2 PARTIAL, 48 MISSING (50 rows)
- `matrix/bifrost-mcp.md` — 0 HAVE, 0 PARTIAL, 80 MISSING (80 rows)
- `matrix/bifrost-openai.md` — 5 HAVE, 3 PARTIAL, 81 MISSING (89 rows, incl. 1 EXTRA)

Total scope = **269 rows** (the brief's "~257 MISSING" + the PARTIAL/HAVE rows
that still need verification or a labeling flip).

The dominant early finding is the **three-way split**, because Bifrost is a
*second gateway* and g0router shipped a large feature set in Waves 0–7:

1. **Already-satisfied** — a g0router feature (often built for 9router parity)
   already covers the bifrost behavior; the row just needs **matrix
   verification + a HAVE/PARTIAL flip** (cheap, no code).
2. **Buildable** — a real feature g0router lacks that fits its architecture
   (SQLite WAL + `database/sql`, fasthttp, layered DDD), buildable as additive
   work.
3. **Escalation** — a Bifrost feature that is out-of-scope, architecturally
   divergent (GORM/Postgres/cluster/plugin-pipeline), or not-applicable to a
   single-binary SQLite gateway (the bifrost analogue of 9router's
   deploy-relays / cookie-auth-web-providers escalation set).

This is a **strategic map only**. No micro-plans, no code.

---

## BINDING BLOCKER (read first) — ESC-REF-ABSENT

**The frozen Bifrost reference is NOT on this host.**
`/Users/heitor/Developer/github.com/bloodf/_refs/bifrost` (the `@ca21298`
checkout every matrix cites) is **absent** on this Linux machine — verified
2026-06-15. This is the *same* blocker that constrained `w7-usage-quota`
(`docs/WORKFLOW.md`: ESC-REF-ABSENT, where the 9router ref was likewise absent
and no endpoint was fabricated).

Consequences, binding on every bifrost plan:

1. The **matrix rows are the only ground truth.** No bifrost behavior may be
   built from an unverifiable source claim. If a row's evidence cite
   (`core/...`, `plugins/...`, `framework/...`, `transports/...`) is needed at
   implementation detail level and the matrix note does not capture it, the plan
   STOPS and escalates (§3) rather than fabricating Bifrost internals.
2. Plans build to the **documented matrix behavior + g0router's own
   conventions**, never to a guessed Bifrost wire format.
3. Restoring the ref (clone `bifrost @ ca21298` to the cited path, or relocate to
   a host that has it) is the highest-leverage unblock for the *buildable*
   protocol-detail rows (semantic cache, MCP server-mode JSON-RPC framing,
   responses/compaction shapes). **DECISION NEEDED** — see Escalations §1.

---

## Architectural decisions (cross-cutting)

1. **g0router is a single-binary SQLite gateway; Bifrost is a clustered
   Postgres/GORM gateway with a plugin pipeline.** This divergence is the
   spine of the escalation set. Bifrost's `ProviderQueue` channel routing,
   `sync.Pool` object pooling, GORM hooks (`BeforeSave`/`AfterFind`),
   memberlist+gRPC clustering, and the `LLMPlugin`/`HTTPTransportPlugin`
   pipeline are **architecturally divergent** from g0router's direct synchronous
   `database/sql` + fasthttp design (`AGENTS.md`: "SQLite WAL store with
   additive-only `ensureColumn` migrations"; "Layered DDD architecture
   (transport→domain→repository)"; "No global state"). Porting them wholesale is
   a rewrite, not parity — they escalate unless an operator explicitly funds the
   re-architecture.

2. **Additive migrations only** (`AGENTS.md`). New bifrost tables/columns land
   via additive `ensureColumn`/`ensureTable` in `internal/store/migrate.go`.
   Secret fields use the `*_enc` precedent (`internal/store/oauthsessions.go`,
   `internal/store/crypto.go`). No destructive DDL.

3. **`internal/server/routes_admin.go` and `routes_openai.go` are serial hot
   files.** Exactly ONE in-flight plan may hold an unmerged edit to each at a
   time (W3–W7 lesson). The serial chain is defined in **Impl order**. Static
   routes register before `{param}` routes (fasthttp precedence — see
   `routes_admin.go:210` `/api/providers/catalog` before `:212`
   `/api/providers/{id}`, and `routes_openai.go:100-101` `/v1/models/test/{kind}`
   before `/v1/models/{param}`).

4. **Layered DDD is enforced** (arch test, phase 12B). New backends = domain
   package (`internal/governance|mcp|platform|inference`) + store layer
   (`internal/store`) + transport handler (`internal/admin` or
   `internal/server`). No `init()`, errors-as-values (`fmt.Errorf("ctx: %w")`),
   no new global state. Read 3 existing files first. Canonical CRUD templates:
   `internal/admin/virtualkeys.go` + `internal/store/virtualkeys.go` +
   `internal/governance/quota.go`.

5. **The OpenAI surface is mostly "wire up the missing route over an existing
   stubbed provider method", not new subsystems.** `internal/server/
   routes_openai.go:95-101` already registers chat/messages/responses/embeddings/
   models. The `Provider` interface (`internal/schemas/provider.go:69-108`)
   already declares TextCompletion, Speech, Transcription, ImageGeneration/Edit/
   Variation, File*, Batch*, CountTokens — currently **stubbed** in
   `internal/providers/openai/stubs.go`. So the bifrost-openai MISSING routes
   split into: (a) **route + DTO wiring over an existing interface method**
   (completions, audio, images, files, batches — buildable), (b) **g0router
   variant already satisfies** (chat/embeddings/models HAVE; error-envelope is
   g0router's `{data,error}` by design), and (c) **Bifrost-specific extension
   surface** (video, containers, rerank, OCR, async `/v1/async/*`, WebSocket
   responses — escalation). This split drives the `bf-openai-*` partition.

6. **g0router's MCP is CLIENT-mode and rich; Bifrost-mcp is mostly SERVER-mode
   and per-user — almost entirely new.** Wave-7 shipped a full MCP *client*
   gateway: `internal/mcp/{launcher,process,bridge,probe,registry,oauth,agent,
   discovery,healthmonitor,toolpolicy,filter,allowlist}.go`, stores
   (`internal/store/mcp{instances,oauth,toolgroups}.go`), and admin routes
   (`routes_admin.go:178-195`: clients/instances/tools/tool-groups + instance
   OAuth `auth/start`). **None of bifrost-mcp's 80 rows is HAVE** because
   bifrost-mcp is a *different feature*: (a) MCP **server-mode** — exposing
   g0router itself AS an MCP server over `/mcp` JSON-RPC+SSE (PAR-BF-MCP-002/003/
   004, 052-054, 075); (b) **per-virtual-key** scoped MCP servers (004, 019, 020,
   033); (c) **per-user** OAuth/header credential flows + sessions API (012-016,
   028-031, 040-047, 058-059, 064-065); (d) `mark3labs/mcp-go` dependency (075,
   076). g0router's existing client-mode launcher/bridge/probe/oauth are
   *adjacent infrastructure that can be reused*, but the server-mode + per-VK +
   per-user surface is genuinely new. Several rows where g0router's client-mode
   already covers the *concept* (STDIO/HTTP/SSE transports, OAuth account engine,
   tool filtering, health monitor, agent loop) are **already-satisfied for the
   client direction** and should be flipped/annotated, not rebuilt.

7. **g0router's governance is FLAT; Bifrost-governance is HIERARCHICAL with
   CAS + calendar alignment + CEL routing.** g0router has VK CRUD
   (`internal/store/virtualkeys.go`), a flat `Budget{Limit,Period,Used}` +
   `RateLimitRPM` (`internal/schemas/governance.go`), a teams table NOT linked to
   VKs (`internal/store/teams.go`), and a simple quota engine
   (`internal/governance/quota.go`: per-key budget window + RPM). Bifrost wants
   VK→Team→Customer ownership with mutual exclusion, single-owner budgets,
   dual-dimension (token+request) rate limits, atomic CAS bumps, calendar-aligned
   resets, `WhiteList`/`BlackList` `["*"]` semantics, blacklist-wins-over-allowlist,
   a 10s DB-sync background worker, and CEL-expression routing rules. g0router's
   `phase-18-bifrost-features.md` already chose the FLAT design deliberately
   ("hierarchical: key limit AND team limit must both pass" is the *only*
   hierarchy planned). So bifrost-governance splits into: (a) **already-satisfied
   by g0router's flat variant** (basic VK/budget/RPM/teams CRUD), (b) **buildable
   incremental upgrades** that fit SQLite (VK↔team linkage + 2-level hierarchical
   check, `WhiteList`/`BlackList` typed semantics + blacklist filtering,
   dual-dimension rate limit, calendar-aligned lazy reset), and (c) **escalations**
   (Customer tier, `sync.Map` lock-free store, CAS-spin bumps, 10s DB-sync
   worker, CEL routing, ghost-node cluster reconciliation, adaptive load
   balancing). This split drives the `bf-gov-*` partition.

8. **Bifrost-core is mostly a different runtime architecture — heaviest
   escalation density.** Plugin pipeline (PAR-BF-CORE-004..018), `ProviderQueue`
   + object pooling (021-024), `KeySelector`/`KVStore` (025-026), CEL routing +
   adaptive load balancing + health states (029-033), semantic cache + vector
   store (034-038), clustering (039-042), OTEL + tracing (043-048), pooled HTTP
   transport types (049-050). g0router has **none** of these (verified: no
   `memberlist`/`gossip`/`VectorStore`/`semanticcache`/`OTEL`/`PluginPipeline`/
   `PreLLMHook` anywhere in `internal/`). Most are architecturally divergent
   escalations. The few **buildable** cores: the `Provider` interface 14-method
   expansion (001 — but those methods are mostly the same stubs as bf-openai;
   reconcile), `AllowFallbacks` on the error type (020 — small, but g0router
   already has account-level fallback via `inference.WithAccountFallback`,
   `selection.go:334`, so this is a *variant* not a gap), and **semantic cache**
   (034-036) which is **already planned** in `phase-19-advanced-features.md`
   (flag-gated `internal/semcache/`, cosine-in-Go) but **not yet built** — that
   is g0router's own roadmap item, buildable as a g0router-shaped variant (SQLite
   + Go cosine, NOT Bifrost's Weaviate/Redis/Qdrant/Pinecone vector backends).

9. **No new global state; reuse the `Handlers` injection pattern.**
   `internal/admin/handlers.go` exposes additive setters; new domains follow the
   same shape (domain service constructed in `NewAdminHandlers`, injected). Keep
   `admin.New(...)` / `RegisterOpenAIRoutes(...)` signatures additive.

10. **"Bifrost parity" is defined per-area, concretely:**
    - **bifrost-openai parity** = the OpenAI-compatible *route surface* g0router
      can serve over its existing provider interface (completions/audio/images/
      files/batches), PLUS the SSE-correctness fixes (header timing, `event:`
      typing) — NOT Bifrost's video/container/async/WS extensions.
    - **bifrost-governance parity** = g0router's flat governance upgraded to
      2-level hierarchy + typed allow/block-list semantics + dual-dimension rate
      limits + calendar-aligned reset — NOT the Customer tier / cluster / CAS /
      CEL machinery.
    - **bifrost-mcp parity** = g0router gains MCP **server-mode** (`/mcp`
      JSON-RPC+SSE exposing its tool catalog) + per-VK tool scoping, reusing the
      existing client-mode infra — NOT the full per-user-OAuth/header credential
      sessions enterprise surface (that escalates).
    - **bifrost-core parity** = the semantic-cache (g0router-shaped) + the
      provider-interface method reconciliation — NOT the plugin pipeline /
      queue / clustering / OTEL runtime.

---

## Row-coverage ledger (every row → exactly one plan OR escalation)

Disposition keys: **SAT** = already-satisfied (matrix flip, no code) ·
**BUILD** = buildable in a plan · **ESC** = escalation (out-of-scope /
divergent / not-applicable) · **VAR** = g0router variant-by-design (label, don't
build).

### bifrost-openai.md (89 rows)

| Rows | Disposition | Plan / reason |
|---|---|---|
| 001, 006, 018, 206, 305 | **SAT** (5) | chat/embeddings/models routes + chat streaming + provider-error passthrough already HAVE (`routes_openai.go:95-99`, `internal/api/chat.go`, `internal/providers/openai/errors.go`) — matrix already marks HAVE; verify only. |
| 019 | **BUILD** | `bf-openai-1` — fix `GET /v1/models/{id}` to filter to one model (currently delegates to List, `internal/api/models.go`). PARTIAL→HAVE. |
| 002, 207 | **BUILD** | `bf-openai-1` — register `POST /v1/completions` over the stubbed `TextCompletion`/`TextCompletionStream` (schema `completions.go` exists). |
| 007, 008, 209, 210 | **BUILD** | `bf-openai-2` — `POST /v1/audio/speech` + `/v1/audio/transcriptions` (+ stream) over stubbed Speech/Transcription; multipart parse for transcription. |
| 009, 010, 011, 211 | **BUILD** | `bf-openai-2` — `POST /v1/images/{generations,edits,variations}` over stubbed ImageGeneration/Edit/Variation; multipart for edits/variations. |
| 025, 026, 027, 028, 029 | **BUILD** | `bf-openai-3` — `/v1/files` CRUD over stubbed File* methods. |
| 020, 021, 022, 023, 024 | **BUILD** | `bf-openai-3` — `/v1/batches` CRUD over stubbed Batch* methods. |
| 003, 208 | **SAT/BUILD** | `/v1/responses` route already registered (`routes_openai.go:97`) and responses streaming exists — verify HAVE; matrix says MISSING but route is present (flip). |
| 004, 005 | **BUILD** | `bf-openai-4` — `POST /v1/responses/input_tokens` (count tokens; `CountTokens` exists on interface) + `/v1/responses/compact` (needs Compaction schema — small additive). |
| 201, 202, 203, 204, 304 | **BUILD** | `bf-openai-4` — SSE correctness: set headers AFTER stream setup so provider-setup errors return JSON (real bug, `internal/api/chat.go:78-85`); `event:` typing for responses/image streams; `[DONE]` skip when typed; SSE error frames. (PAR-BF-OAI-202 fasthttp pipe bypass = optional perf, ride or ESC.) |
| 301, 302, 303 | **VAR** | g0router uses `{data,error}` snake_case envelope by design (`AGENTS.md`: "All API responses use snake_case JSON with a `{data, error}` envelope"). Bifrost's `BifrostError`/`is_bifrost_error`/`event_id`/`Param interface{}` is a different contract. Record as variant; optionally add `param`+`event_id` fields (tiny, ride `bf-openai-4`) — **DECISION NEEDED** (label vs augment). |
| 101, 102, 103, 104, 105, 106, 107 | **ESC** | Anthropic/OpenRouter request-normalization field-stripping (`cache_control`, server-tool shapes, `reasoning.effort`→`reasoning_effort`, `reasoning.max_tokens=nil`). g0router has no multi-provider transport-normalization layer and several target fields don't exist on its schemas. Divergent; defer to a normalization escalation. |
| 108, 109, 110, 111, 112 | **ESC** | Batch/file/video ID normalization for Gemini/Bedrock (prefix swap, base64 ARN, `provider:id`). Rides only if the corresponding batch/file routes ship AND multi-provider id-encoding is funded; defer. |
| 113, 114, 115, 116, 117, 118 | **ESC** | Container raw-passthrough, ExtraParams passthrough, multipart ExtraParams, large-payload mode, Azure deployment-path + UA detection. Bifrost-specific transport features; defer. |
| 401, 402, 403, 404 | **ESC** | `AllowedRequests` 50-op capability matrix + `ProviderFeatureSupport` 30-flag matrix + beta-header gating. Large typed matrices; g0router has none. Defer (Go-port note suggests a smaller bool map later). |
| 405, 044, 504, 505, 506, 507 | **ESC** | WebSocket Responses API (`WebSocketCapableProvider`, WS upgrade, store override, native-WS-then-HTTP-bridge). No WS handler in g0router; phase-19 plans a *chat* WS, not a responses-WS. Defer. |
| 012, 013, 014, 015, 016, 017, 112(video) | **ESC** | `/v1/videos*` generation/retrieve/download/delete/remix/list. No video schema/provider in g0router. Bifrost extension; defer. |
| 030, 031, 032, 033, 034, 035, 036, 037, 038 | **ESC** | `/v1/containers*` + container files. No container concept in g0router. Bifrost extension; defer. |
| 039, 040 | **ESC** | `/v1/rerank` + `/v1/ocr`. No Rerank/OCR provider methods or schemas. Defer (could BUILD if a reranker provider is funded — note as deferred-buildable). |
| 041, 042 | **ESC** | Non-`/v1`-prefixed alias routes + Azure wildcard `/openai/openai/deployments/{*}`. g0router serves `/v1/*` only by design. Defer. |
| 043, 501, 502, 503 | **ESC** | Async mirror `/v1/async/*` (11 endpoints, 202+job, reject stream). No async job subsystem; large. Defer. |
| 205 | **ESC** | Raw upstream-bytes passthrough when `Provider==OpenAI && RawResponse!=nil`. g0router always unmarshals/remarshals; adding `RawRequestBody`/`RawResponse` passthrough is a cross-cutting transport change. Defer (Go-port note #5). |
| 119 | **n/a** | EXTRA in matrix (g0router has no blacklist) — folds into `bf-gov-2` WhiteList/BlackList work; no separate disposition. |

### bifrost-governance.md (50 rows)

| Rows | Disposition | Plan / reason |
|---|---|---|
| 001(part), 002(part), 011(part), 017(part) | **SAT** | g0router VK has ID/Name/ProviderConfigs/Budget/RateLimitRPM; ProviderConfig has Provider/AllowedModels/KeyIDs/Weight; Budget has Limit/Period/Used; teams table exists. The *basic* shape is satisfied — these stay PARTIAL with the upgrades below. |
| 001, 002, 007, 009 | **BUILD** | `bf-gov-1` — VK↔Team linkage (add `team_id` to VK per `phase-18` table), `IsActive *bool` (nil=true) semantics, VK provider-config `AllowAllKeys`. (Customer tier = ESC.) |
| 010, 014, 020, 042 | **BUILD** | `bf-gov-1` — 2-level hierarchical evaluation (VK-level AND Team-level budget + RPM must both pass), per `phase-18-bifrost-features.md:64-69`. (Provider/Model→User→VK→Team→Customer full chain = ESC; build VK→Team only.) |
| 026, 027, 028, 029, 030, 037, 048, 119(oai) | **BUILD** | `bf-gov-2` — typed `WhiteList`/`BlackList` with `["*"]`/empty/listed semantics + `IsAllowed`/`IsBlocked`; blacklist-wins-over-allowlist 2-pass; key-level + provider-config-level blacklists (additive JSON cols). |
| 017, 038 | **BUILD** | `bf-gov-3` — dual-dimension rate limit (token + request limits, each with max/reset/usage/lastReset); reset-duration validation. |
| 015, 019, 039 | **BUILD** | `bf-gov-3` — calendar-aligned lazy reset (`phase-18:72-73` already specifies lazy reset; add calendar-period-start). |
| 021, 022, 023, 044, 045 | **BUILD/ESC** | `bf-gov-3` — streaming-aware `UsageUpdate` (IsStreaming/IsFinalChunk/HasUsageData) + startup-reset + graceful-flush. BUILD the streaming-aware accrual + startup reset over g0router's existing usage path; the **10s DB-sync background worker** (016, 023, 045) is BUILDable as a `time.Ticker` (Go-port note #7) — keep in `bf-gov-3`. |
| 035, 036 | **BUILD** | `bf-gov-3` — `Decision` enum + `EvaluationResult` (Allow/VKNotFound/RateLimited/BudgetExceeded/ModelBlocked/...) mapped to g0router's `{data,error}` envelope (Go-port note #6). |
| 034 | **VAR/BUILD** | VK-mandatory mode (`x-bf-vk` header). g0router uses `x-g0-vk` (`routes_openai.go:75`). BUILD an optional mandatory-VK setting under g0router's header name; record header-name as variant. |
| 008, 019(customer), 046, 047 | **ESC** | **Customer tier** (Customer schema + AfterFind calendar propagation + customer-level budgets/rate-limits). g0router's `phase-18` design is VK+Team only — adding a third tier is an explicit scope expansion. Defer. |
| 004, 049, 050 | **ESC** | `sync.Map` lock-free in-memory governance store + 40-method `GovernanceStore` interface. g0router uses `database/sql` reads, not an in-memory mirror store (Go-port note #2 says replace `sync.Map`); the *behaviors* are covered by the BUILD rows above — the in-memory-store *architecture* is divergent. Defer the architecture. |
| 013, 018, 041 | **ESC** | Atomic **CAS-spin** budget/rate-limit bumps (`CompareAndSwap` retry loop). Presupposes the in-memory atomic store (004/049). g0router accrues via SQL writes; CAS-on-in-memory is divergent. Defer (the *accrual correctness* is covered by bf-gov-3's SQL path). |
| 003, 005, 006 | **ESC** | VK MCP-config join (ToolsToExecute per VK↔MCPClient), VK `CalendarAligned` AfterFind propagation, VK value SHA-256-hash-for-lookup + AES-at-rest. (006 partially overlaps g0router's `g0vk-` keys which are already random+stored; the hash-indexed-encrypted scheme is a divergent rework — defer. 003 rides bifrost-mcp per-VK scoping if that ships.) |
| 005, 046, 047 | **ESC** | Calendar-alignment owner-propagation via GORM `AfterFind` (VK/Team/Customer). g0router has no GORM hooks; bf-gov-3 builds calendar reset directly. The *AfterFind propagation mechanism* is GORM-specific — n/a. |
| 024, 025, 031, 032, 033, 040, 043 | **ESC** | Log-denormalized governance fields + ghost-node reconciliation + model-catalog cross-provider allowlist + governance-context stamping + cluster remote-baseline rate check + load-balance-provider weighted selection + scoped-model-config budgets. Cluster/observability-coupled or catalog-coupled; g0router has no model-catalog or cluster. Defer (033 weighted selection is *already satisfied* by `inference.SelectionEngine` smooth-WRR, `selection.go:252` — flip 033 to SAT/VAR). |
| 012 | **ESC** | Budget single-owner mutual-exclusion `BeforeSave`. GORM-hook-specific; g0router validates in handler. Build inline if cheap inside bf-gov-1, else n/a. |

### bifrost-mcp.md (80 rows)

| Rows | Disposition | Plan / reason |
|---|---|---|
| 001, 005, 006, 007, 008, 009, 010, 011, 021, 024, 025, 026, 027, 066, 067, 076, 080 | **SAT/VAR** | g0router's **client-mode** MCP already covers: HTTP/SSE/STDIO transports (`internal/mcp/launcher.go`, `process.go`), STDIO command/args/env, none/headers/oauth auth concepts (`oauth.go`), ping-vs-listTools health (`healthmonitor.go`, `probe.go`), agent loop (`agent.go`), TLS, connection-state, mcp-go-equivalent (g0router rolls its own bridge — VAR, no `mark3labs/mcp-go` dep needed), EnvVar-encrypted connection string (`*_enc` via `crypto.go`). Flip these to HAVE/PARTIAL **for the client direction** with a note that server-mode/per-user variants are separate. |
| 002, 003, 075 | **BUILD** | `bf-mcp-1` — MCP **server-mode**: expose g0router as an MCP server over `POST/GET /mcp` (JSON-RPC + SSE), global un-scoped tool surface. Reuse `internal/mcp/bridge.go` JSON-RPC framing + `sse.go`. (New transport: register `/mcp` in `routes_admin.go` or a new server route file.) |
| 052, 053, 054 | **BUILD** | `bf-mcp-1` — VK resolution for MCP server (`x-g0-vk` > Bearer > x-api-key, g0router header), SSE heartbeat `: ping` every 15s, deferred trace completion for SSE. |
| 004, 019, 020, 033 | **BUILD** | `bf-mcp-2` — per-virtual-key scoped MCP server (lazy creation) + per-VK `executeOnlyTools` wildcard filtering + `AllowOnAllVirtualKeys` + VK↔MCP assignment table (additive). Reuses `internal/store/mcptoolgroups.go` + tool-group filtering. |
| 017, 018, 049, 057, 071, 077, 078, 079 | **BUILD** | `bf-mcp-2` — tool filtering ToolsToExecute/ToolsToAutoExecute subset validation, disable-auto-inject flag, allowed-extra-headers whitelist, tool annotations mapping, code-mode + config-hash flags (flags only; code-mode VFS execution = ESC). |
| 034, 035, 036, 037, 038, 048 | **BUILD/SAT** | client CRUD API (`/api/mcp/client[s]` + reconnect + tool/execute). g0router already has `/api/mcp/clients` (list/get), `/api/mcp/instances` (CRUD), `/api/mcp/tools/{name}/execute` (`routes_admin.go:178-187`). Bifrost's create/update/delete/reconnect on *clients* map to g0router's *instances* — **largely SAT via the instances surface**; build only the genuine gaps (POST/PUT/DELETE client semantics if the instances model doesn't cover them, reconnect). Reconcile naming (bifrost "client" ≈ g0router "instance"). |
| 012, 013, 014, 015, 016, 027, 028, 029, 030, 031, 040, 041, 042, 043, 044, 045, 046, 047, 058, 059, 064, 065 | **ESC** | **Per-user OAuth + per-user header credential flows + sessions API** (per_user_oauth/per_user_headers auth types, identity dimensions user/vk/session, header-credential tables with TTL, oauth_user_tokens/sessions tables, `/api/mcp/sessions*`, `/api/oauth/per-user/flows*`, `/api/mcp/per-user-headers/*`, credential sweep worker, temp-token minting, needs-update state machine, reconciliation hooks). This is Bifrost's *enterprise multi-tenant* MCP surface. g0router has server-level OAuth (`internal/mcp/oauth.go`) but no per-user identity model. Large, multi-tenant; defer as the bifrost-mcp escalation core. |
| 022, 023, 050, 051, 055, 056, 060, 061, 062, 063, 068, 069, 070, 072, 073, 074 | **ESC** | Flexible duration parsing quirks, canonicalization, tool-pricing catalog (`MCPCatalog` "server/tool"), code-mode VFS binding, plugin-pipeline-for-nested-tools, DB+memory two-phase create/update rollback, discovered-tool key migration on rename, retry-on-in-flight-reconnect loops, header redaction/merge, OAuth-rotation-disabled guard, sentinel error types. Implementation-quirk + plugin-pipeline-coupled rows; most presuppose Bifrost's handler architecture (and the absent ref). Defer; cherry-pick cheap ones (e.g. error sentinels, validation) into bf-mcp-2 only if matrix-evidenced. |
| 032, 039 | **BUILD** | `bf-mcp-1` — MCP client config table with encryption at rest + complete-oauth endpoint. g0router has `mcpoauth.go` + `mcpinstances.go`; align the table to bifrost's documented columns where it adds server-mode needs. |

### bifrost-core.md (50 rows)

| Rows | Disposition | Plan / reason |
|---|---|---|
| 002 | **SAT** | `postHookRunner` already on stream methods (`internal/schemas/provider.go:76`); HAVE. `postHookSpanFinalizer` = part of tracing ESC. |
| 001, 027, 028 | **BUILD/reconcile** | `bf-core-1` — Provider-interface method expansion (Rerank/OCR/Video*/Container*/CachedContent*/Passthrough/Compaction) + request-type constants + richer `GatewayContext`. **Reconcile with bf-openai**: only add interface methods whose routes are actually being built (Compaction yes; Video/Container = ESC, don't add dead interface methods per §3 no-leftovers). So bf-core-1 = `GatewayContext` typed KV + Compaction + CountTokens reconciliation only; the rest ride their route plans or ESC. |
| 019, 020 | **VAR** | Fallback chain + `AllowFallbacks`. g0router HAS account/connection-level fallback (`inference.WithAccountFallback`, `selection.go:334`; weighted WRR `selection.go:252`) — a *different mechanism* (no per-attempt PreLLMHook). Record as variant-HAVE; optionally add an `AllowFallbacks`-equivalent flag to `ProviderError` (small, ride bf-core-1) — **DECISION NEEDED**. |
| 034, 035, 036 | **BUILD** | `bf-core-2` — **Semantic cache**, g0router-shaped per `phase-19-advanced-features.md:60-69` (already-planned, NOT-yet-built: exact-key hash hit → cosine-in-Go over ≤500 candidates, threshold 0.95, SQLite `semantic_cache` table, `[flag: semantic_cache]`, non-streaming chat only, after guardrails). This is g0router's OWN roadmap, satisfying the bifrost *concept* without Bifrost's vector backends. |
| 037, 038 | **ESC** | `VectorStore` interface + Weaviate/Redis/Qdrant/Pinecone backends. g0router's semantic cache uses SQLite + Go cosine by design (`phase-19:61`). External vector backends are out-of-scope. Defer. |
| 004, 005, 006, 007, 008, 009, 010, 011, 012, 013, 014, 015, 016, 017, 018, 049, 050 | **ESC** | **Plugin system** (BasePlugin/HTTPTransportPlugin/LLMPlugin/MCPPlugin/Observability/ConfigMarshaller interfaces, ordered pipeline, short-circuit, placement, pooling, pooled HTTP types, case-insensitive helpers). g0router has fasthttp middleware only (`internal/server/middleware.go`); a full plugin pipeline is a runtime re-architecture. Defer (Go-port note #2 calls it "the largest gap"). |
| 003, 405(oai) | **ESC** | `WebSocketCapableProvider`. No WS provider abstraction; see bf-openai WS ESC. Defer. |
| 021, 022, 023, 024 | **ESC** | `ProviderQueue` channel routing + lifecycle + object pooling + `dropExcessRequests`. g0router uses direct synchronous calls by design. Divergent runtime. Defer. |
| 025, 026 | **ESC** | `KeySelector` func type + `KVStore` clustering interface. g0router resolves keys via `KeyResolver`/`SelectionEngine` (already-satisfied for selection — VAR); KVStore presupposes clustering. Defer KVStore; 025 ≈ SAT via existing selection. |
| 029, 030, 031, 032, 033 | **ESC** | CEL routing engine + rule-chain cycle detection + adaptive load balancing (error/latency/utilization scoring) + route health state machine + weighted-random key selection. g0router has prefix routing (`router.go`) + smooth-WRR (`selection.go`) + retry/cooldown (`retry.go`) — covers the *practical* selection need (VAR for 033); CEL + adaptive scoring + health-states are a divergent routing engine. Defer; `phase-18:22-23` already deferred "adaptive routing" as a duplicate of the `auto` combo classifier. |
| 039, 040, 041, 042 | **ESC** | **Clustering** (memberlist gossip + gRPC sync + service discovery + leader election + 30-entity replication). Single-binary SQLite gateway; clustering is a product-category change. Defer (hard escalation). |
| 043, 044, 045, 046, 047, 048 | **ESC** | OTEL plugin + metrics exporters + `Tracer` interface + trace streaming accumulator + header capture. `phase-18:24` already deferred OTEL ("optional infra, no UI dependency. Skip"). Defer. |

---

## Micro-plan index

| Plan | Scope | Bifrost rows it closes | Key existing files / integration points (file:line) | New Go files | routes? | e2e impact | Depends |
|---|---|---|---|---|---|---|---|
| **bf-openai-1** | **Completions + models-get fix**: register `POST /v1/completions` (+stream) over the stubbed `TextCompletion`/`TextCompletionStream`; fix `GET /v1/models/{id}` to filter to one model. | PAR-BF-OAI-002, 207, 019 | `internal/server/routes_openai.go:95-101`, `internal/providers/openai/stubs.go:13-15`, `internal/api/models.go` (Get→filter), `internal/schemas/completions.go` | none (handler in `internal/api/completions.go`) + tests | YES (`routes_openai.go` serial) | adds Go handler integration tests; no UI | — |
| **bf-openai-2** | **Audio + images routes**: `/v1/audio/{speech,transcriptions}` (+stream) and `/v1/images/{generations,edits,variations}` (+stream) over stubbed provider methods; multipart parse for transcription/edits/variations. | PAR-BF-OAI-007, 008, 009, 010, 011, 209, 210, 211 | `internal/providers/openai/stubs.go:25-55`, `internal/schemas/{audio,images}.go`, multipart helpers | `internal/api/{audio,images}.go` + tests | YES (serial) | Go integration tests | — (∥ bf-openai-1) |
| **bf-openai-3** | **Files + batches routes**: `/v1/files` CRUD + `/v1/batches` CRUD over stubbed File*/Batch* methods. | PAR-BF-OAI-020..029 | `internal/providers/openai/stubs.go:57-91`, `internal/schemas/{files,batch}.go` | `internal/api/{files,batches}.go` + tests | YES (serial) | Go integration tests | — (∥) |
| **bf-openai-4** | **Responses extras + SSE correctness + error fields**: `/v1/responses/input_tokens` (CountTokens) + `/v1/responses/compact` (new Compaction schema); SSE header-after-setup fix, `event:` typing, typed `[DONE]` skip, SSE error frames; optional `param`/`event_id` on error envelope. | PAR-BF-OAI-004, 005, 201, 203, 204, 304, (302/303 augment) | `internal/api/chat.go:78-85` (SSE bug), `internal/api/responses.go`, `internal/schemas/{responses,errors}.go`, `internal/providers/openai/stubs.go:93-95` | `internal/schemas/compaction.go` + tests | YES (serial) | Go integration tests; chat SSE spec stays green | — (∥) |
| **bf-gov-1** | **VK↔Team hierarchy**: add `team_id` to VK (additive), `IsActive *bool` nil=true, `AllowAllKeys`; 2-level hierarchical budget+RPM check (VK AND Team must pass); budget single-owner validation inline. | PAR-BF-GOV-001, 002, 007, 009, 010, 014, 020, 042(VK→Team only), 012 | `internal/store/{virtualkeys,teams}.go`, `internal/governance/quota.go`, `internal/schemas/governance.go`, `phase-18-bifrost-features.md:64-69` | quota-engine hierarchy edits + store cols + tests | NO (CRUD routes exist) | corrects governance handler tests | — |
| **bf-gov-2** | **Typed allow/block-list semantics**: `WhiteList`/`BlackList` types with `["*"]`/empty/listed semantics + `IsAllowed`/`IsBlocked`; blacklist-wins 2-pass model filtering; key-level + provider-config-level blacklists (additive JSON). | PAR-BF-GOV-026, 027, 028, 029, 030, 037, 048; PAR-BF-OAI-119 | `internal/schemas/governance.go`, `internal/governance/`, `internal/store/{virtualkeys,apikeys}.go` | `internal/schemas/lists.go` (WhiteList/BlackList) + filter logic + tests | NO | governance tests | — (∥ bf-gov-1) |
| **bf-gov-3** | **Dual-dimension rate limit + calendar reset + streaming accrual + decision enum + 10s sync worker**: token+request limits; calendar-aligned lazy reset; streaming-aware `UsageUpdate`; `Decision`/`EvaluationResult`→`{data,error}`; startup reset + graceful flush + `time.Ticker` DB sync. | PAR-BF-GOV-015, 017, 018(SQL-accrual variant), 019, 021, 022, 023, 035, 036, 038, 039, 044, 045, 016 | `internal/governance/quota.go`, server lifecycle (`internal/server/server.go`), usage path | rate-limit + reset + worker + tests | NO | governance tests | bf-gov-1 (hierarchy shape) |
| **bf-mcp-1** | **MCP server-mode foundation**: expose g0router as an MCP server over `POST/GET /mcp` (JSON-RPC + SSE), global tool surface; VK resolution; SSE heartbeat; deferred trace completion; client-config table alignment + complete-oauth. | PAR-BF-MCP-002, 003, 032, 039, 052, 053, 054, 075 | `internal/mcp/{bridge,sse,probe,oauth}.go` (reuse), `internal/store/mcp{instances,oauth}.go`, `routes_admin.go` or new `routes_mcp.go` | `internal/mcp/server.go`, `internal/server/routes_mcp.go` + tests | YES (`/mcp` registration — serial) | Go integration tests | — (∥ openai/gov; reuses mcp pkg) |
| **bf-mcp-2** | **Per-VK MCP scoping + tool filtering**: per-VK lazy-created MCP server, `executeOnlyTools` wildcard filter, `AllowOnAllVirtualKeys`, VK↔MCP assignment table (additive); ToolsToExecute/AutoExecute subset validation; annotations; flags. | PAR-BF-MCP-004, 017, 018, 019, 020, 033, 049, 057, 071, 077, 078, 079 | `internal/mcp/{filter,allowlist,toolpolicy}.go` (reuse), `internal/store/mcptoolgroups.go`, `internal/admin/mcp.go` | VK-MCP store + scoping + tests | YES (serial — admin mcp routes) | corrects mcp handler tests | bf-mcp-1 |
| **bf-core-1** | **Provider-interface reconciliation**: richer typed `GatewayContext` (KV + metadata); Compaction + CountTokens interface/method reconciliation aligned with bf-openai-4; optional `AllowFallbacks` flag on `ProviderError`. (NO dead Video/Container methods — §3.) | PAR-BF-CORE-001(subset), 027, 028, 020(variant flag) | `internal/schemas/{provider,errors}.go`, `internal/inference/selection.go` (fallback flag honor) | schema edits + tests | NO | provider/inference tests | reconcile with bf-openai-4 |
| **bf-core-2** | **Semantic cache (g0router-shaped)**: `internal/semcache/` domain (exact-key hash hit → cosine-in-Go ≤500 candidates, threshold 0.95), SQLite `semantic_cache` table, `[flag: semantic_cache]`, non-streaming chat only, after guardrails, `GET/DELETE /api/cache/semantic`. Per `phase-19-advanced-features.md:60-69`. | PAR-BF-CORE-034, 035, 036 | `internal/governance/guardrails.go` (ordering), `internal/api/chat.go` (cache hook), `internal/store/migrate.go`, `phase-19:60-69` | `internal/semcache/*.go`, `internal/store/semcache.go`, `internal/admin/cache.go` + tests | YES (`/api/cache/*` serial) | corrects cache handler tests | — (∥; disjoint domain) |

**Escalation rows are NOT assigned a plan** — they are listed in the Escalation
section with reasons and counts.

---

## Concurrency waves + impl order

`internal/` domains are largely disjoint (openai-api / governance / mcp / semcache
are separate packages), so most plans run concurrently. Shared hot files:
`internal/server/routes_openai.go` (bf-openai-1..4), `routes_admin.go` /
`routes_mcp.go` (bf-mcp-1/2, bf-core-2), and `internal/governance/quota.go`
(bf-gov-1→3 internal serial). Orchestrator caps at ≤4 concurrent jobs (§ same as
WAVE-7).

```
OpenAI-surface track (routes_openai.go serial holders):
  bf-openai-1 → bf-openai-2 → bf-openai-3 → bf-openai-4
  (each appends /v1/* routes; static-before-{param} precedence;
   bf-openai-4 reconciles with bf-core-1 on schemas/errors.go + provider.go)

Governance track (disjoint store/domain; internal serial on quota.go):
  bf-gov-1 → bf-gov-3        (bf-gov-3 extends bf-gov-1's hierarchy)
  bf-gov-2 ∥ bf-gov-1        (lists.go is disjoint from quota.go)

MCP track (reuses internal/mcp; serial on the new /mcp + admin routes):
  bf-mcp-1 → bf-mcp-2        (per-VK scoping extends server-mode)

Core track:
  bf-core-1 ∥ everything     (schema reconcile; coordinate provider.go/errors.go
                              edit window with bf-openai-4 as a micro-serial)
  bf-core-2 ∥ everything     (semcache is greenfield disjoint; /api/cache route)

routes_openai.go SERIAL CHAIN (one unmerged holder at a time):
  bf-openai-1 → bf-openai-2 → bf-openai-3 → bf-openai-4

routes_admin.go / routes_mcp.go SERIAL CHAIN:
  bf-mcp-1 → bf-mcp-2 → bf-core-2
  (bf-mcp-1 registers /mcp; bf-mcp-2 touches admin mcp routes;
   bf-core-2 appends /api/cache/* — append-only, last)

provider.go / errors.go MICRO-SERIAL:
  bf-core-1 ↔ bf-openai-4   (both touch internal/schemas/{provider,errors}.go +
                              compaction; serialize that file via orchestrator)
```

Reasons:
- **OpenAI surface is the highest-value, lowest-risk track** — the schemas and
  stubbed provider methods already exist (`internal/providers/openai/stubs.go`),
  so these are route+DTO wiring, not new subsystems. Start bf-openai-1 day one.
- **Governance + lists are disjoint** (quota.go vs lists.go), parallel; gov-3
  serializes after gov-1 because it extends the hierarchy shape.
- **MCP track reuses the Wave-7 mcp package** but adds the genuinely-new
  server-mode + per-VK surface; serial because both touch the mcp admin/route
  registration.
- **Semantic cache is greenfield** (`internal/semcache/` does not exist) — fully
  disjoint, runs throughout; only its `/api/cache/*` route registration is hot.
- **bf-core-1 reconciles schemas** with bf-openai-4 (Compaction/CountTokens/
  errors) — micro-serial on those files to avoid the W3–W7 routes-file lesson.

---

## e2e / mock reconciliation (binding)

Bifrost is a backend-parity program; most plans are Go-only with handler
integration tests as the authoritative proof (TDD: test first → see fail →
minimum code → pass; no mocks, fakes via interfaces — `AGENTS.md`). Where a plan
touches a surface that has a Wave-6 UI mock (`ui/e2e/mocks/handlers/*`), the
Wave-7 binding rule carries over: **real Go wins; the mock body + seed are
corrected to mirror the real `{data,error}` snake_case response in the same
plan** (never the reverse). New bifrost surfaces with no UI page (semantic cache
admin, `/mcp` server-mode, `/v1/{files,batches,audio,images}` API routes) ship a
Go integration test and need no UI touch. `cd ui && npm run build` + scoped
Playwright stay green only for the rare plan that corrects a mock.

---

## Freeze rules

- **`internal/server/routes_openai.go`**: one unmerged holder at a time, in the
  bf-openai-1→4 chain order. Additive appends; static-before-`{param}`.
- **`internal/server/routes_admin.go` / `routes_mcp.go`**: one unmerged holder
  at a time (bf-mcp-1 → bf-mcp-2 → bf-core-2). Additive appends.
- **`internal/schemas/provider.go` + `errors.go`**: micro-serial for bf-core-1 ↔
  bf-openai-4; additive only. **No dead interface methods** — only add a provider
  method when its route/feature is actually being built (§3 no-leftovers; Wave-5
  `NewWithShutdown` dead-wiring lesson).
- **`internal/governance/quota.go`**: bf-gov-1 → bf-gov-3 internal serial.
- **Migrations**: additive `ensureColumn`/`ensureTable` only; no destructive DDL.
- **Wave-7 MCP client-mode is consume-only** for SAT rows — flip the matrix
  label (client direction), do not rebuild `internal/mcp/*`.
- **No reverse engineering of the absent Bifrost ref** — build to matrix +
  g0router conventions only (ESC-REF-ABSENT).

---

## Buildable-vs-escalation summary (counts)

Indicative dispositions over the 269 in-scope rows (exact per-row counts are in
the ledger; ranges reflect SAT-vs-VAR labeling decisions pending operator):

| Disposition | Approx rows | Where |
|---|---|---|
| **Already-satisfied / variant (SAT/VAR)** — flip label, no code | **~40** | bifrost-openai 5 HAVE + responses-route; bifrost-mcp ~17 client-mode rows; bifrost-gov ~4 (weighted-selection 033, basic VK/budget/team) ; bifrost-core ~4 (002 postHookRunner, 019/020/025/033 fallback+selection variants) |
| **Buildable** across **11 plans** | **~95** | bf-openai-1..4 (~32), bf-gov-1..3 (~28), bf-mcp-1..2 (~20), bf-core-1..2 (~15) |
| **Escalated** — out-of-scope / divergent / not-applicable | **~134** | plugin pipeline + queue + pooling (core ~17); clustering (core 4); OTEL+tracing (core 6); vector backends (core 2); CEL+adaptive routing+health (core 5); MCP per-user OAuth/header/sessions enterprise surface (mcp ~22); MCP quirk/code-mode/two-phase rows (mcp ~16); governance Customer-tier + in-memory-store + CAS + cluster-recon + model-catalog (gov ~18); openai video/containers/async/WS/rerank/OCR/alias/Azure/normalization extensions (openai ~37) |

**Headline:** like 9router's 61-escalation tail, a 269-row *second-gateway*
parity has a **large escalation set (~50%)** — driven by Bifrost being a
clustered, plugin-pipeline, Postgres/GORM, multi-tenant product. The
**high-value, low-risk early wins** are: (1) the **OpenAI route surface**
(schemas + stubbed methods already exist), (2) the **MCP client-mode SAT flips**
(cheap matrix verification), and (3) **semantic cache** (already on g0router's
own phase-19 roadmap).

---

## Escalations / blockers / decisions needed (explicit)

1. **ESC-REF-ABSENT (BINDING)** — the Bifrost ref `@ca21298` is absent on this
   host. Buildable protocol-detail rows (semantic cache shapes, MCP server-mode
   JSON-RPC framing, responses/compaction DTOs) need it for fidelity. **DECISION
   NEEDED:** clone `bifrost @ ca21298` to the cited path (or run from a host that
   has it) before dispatching bf-mcp-1 / bf-core-2 / bf-openai-4 protocol rows;
   until then those plans build to the matrix text only and STOP-escalate on any
   undocumented detail.
2. **Plugin pipeline (PAR-BF-CORE-004..018, 049, 050)** — a runtime
   re-architecture (ordered Pre/PostLLM hooks, short-circuit, pooling). RECOMMEND
   defer as a hard escalation; revisit only if g0router commits to a plugin
   model. **DECISION NEEDED** (fund re-architecture or record permanently-deferred).
3. **Clustering (PAR-BF-CORE-039..042) + in-memory governance store + CAS +
   ghost-node reconciliation (gov 004, 013, 025, 041, 049, 050)** — these
   presuppose a multi-node cluster. g0router is a single-binary SQLite gateway.
   RECOMMEND permanent escalation (product-category change).
4. **MCP per-user OAuth/header credential + sessions surface (mcp ~22 rows)** —
   Bifrost's enterprise multi-tenant MCP. Large. RECOMMEND defer as the
   bifrost-mcp escalation core; build server-mode + per-VK (bf-mcp-1/2) first.
   **DECISION NEEDED** if multi-tenant MCP is a product goal.
5. **OpenAI extension surface — video / containers / async / WebSocket-responses
   / rerank / OCR / non-`/v1` aliases / Azure-deployment (openai ~37 rows)** —
   Bifrost-specific endpoints with no g0router schema/provider. RECOMMEND defer;
   rerank/OCR are *deferred-buildable* if a reranker/OCR provider is funded.
   **DECISION NEEDED** on whether any extension is in scope.
6. **Request normalization layer (PAR-BF-OAI-101..118)** — Anthropic/OpenRouter/
   Azure transport field-stripping + ID encoding. g0router has no multi-provider
   transport-normalization layer and several target fields are absent from its
   schemas. RECOMMEND defer to a dedicated normalization plan if/when multi-format
   passthrough is funded.
7. **Customer governance tier (gov 008, 046, 047) + VK value hash-index+AES
   (gov 006)** — g0router's `phase-18` design is VK+Team only and uses `g0vk-`
   random keys. RECOMMEND defer the Customer tier; treat 006 as a variant
   (g0router's key scheme) unless hash-indexed-encrypted lookup is required.
   **DECISION NEEDED** on Customer tier.
8. **Error-envelope labeling (PAR-BF-OAI-301/302/303)** — g0router's `{data,error}`
   snake_case envelope is an `AGENTS.md` decision. RECOMMEND record as variant;
   optionally augment `APIError` with `param`/`event_id` (tiny, rides bf-openai-4).
   **DECISION NEEDED** (label-as-variant vs augment).
9. **`AllowFallbacks` / per-attempt fallback (core 019/020)** — g0router has
   account-level fallback (`WithAccountFallback`) but not Bifrost's per-attempt
   PreLLMHook chain. RECOMMEND record as variant-HAVE; optionally add an
   `AllowFallbacks`-equivalent flag (rides bf-core-1). **DECISION NEEDED.**

---

## Out of Bifrost-program scope (explicit)

- **9router program** — COMPLETE; not reopened here.
- All rows marked **ESC** above are out of the *buildable* program until their
  decision (Escalations §1–9) is resolved; they are NOT counted toward
  "bifrost parity of feasible".
- Any Bifrost behavior whose only evidence is an unreadable ref cite and whose
  detail is not captured in the matrix note (ESC-REF-ABSENT).

---

## Protocol (CLI_ORCHESTRATOR-governed, Go-weighted)

Per micro-plan: memory/context preflight (§9.1) → planning loop + micro-plan gate
(§9.2/9.3) → TDD impl (Go `_test.go` first, see fail, minimum code; no mocks,
fakes via interfaces — `AGENTS.md`) → validation loop (§9.5) → scoped diff gate
(commit-bounded; live-tree verification before closure) → shipping (§9.6) → flip
matrix rows → mock-mirror correction (if a mock exists) → `docs/WORKFLOW.md`
update. Required-but-unplanned work → STOP + escalate (§3). Commits:
`phase-N/bf-X: <description>` (`AGENTS.md` convention; push direct to main,
quality gates local).

Per-commit gates (every commit): `go test ./... && go vet ./...` green,
`go build ./...` green, **hermetic** (no net/sleep in tests — Wave-7 lesson). For
plans that correct a mock or touch UI wiring: `cd ui && npm run build` green +
the affected scoped Playwright spec green; full Playwright at any mock-correcting
plan and at the program boundary.
