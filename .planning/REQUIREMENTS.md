# Milestone v2.0 Requirements

**Milestone:** v2.0 9router + BiFrost Clean Slate Port  
**Source:** `docs/superpowers/specs/2026-06-08-9router-bifrost-port-design.md`

---

## Active Requirements

### OpenAI-Compatible API (OPENAI)

- [ ] **OPENAI-01**: `POST /v1/chat/completions` supports non-streaming requests with standard OpenAI parameters and returns OpenAI-shaped responses.
- [ ] **OPENAI-02**: `POST /v1/chat/completions` supports streaming via SSE and terminates with `data: [DONE]`.
- [ ] **OPENAI-03**: `POST /v1/completions` supports legacy text completions (streaming + non-streaming).
- [ ] **OPENAI-04**: `POST /v1/embeddings` returns OpenAI-compatible embedding responses.
- [ ] **OPENAI-05**: `GET /v1/models` and `GET /v1/models/:id` return available models including catalog models, aliases, and combos.
- [ ] **OPENAI-06**: `POST /v1/responses` supports the OpenAI Responses API (streaming + non-streaming) with event-based SSE.
- [ ] **OPENAI-07**: `POST /v1/images/generations`, `/edits`, `/variations` return OpenAI-compatible image responses.
- [ ] **OPENAI-08**: `POST /v1/audio/speech` and `/transcriptions` (and `/translations`) return OpenAI-compatible audio responses.
- [ ] **OPENAI-09**: `POST /v1/files`, `GET /v1/files`, `GET /v1/files/:id`, `DELETE /v1/files/:id`, `GET /v1/files/:id/content` implement file management.
- [ ] **OPENAI-10**: `POST /v1/batches`, `GET /v1/batches`, `GET /v1/batches/:id`, `POST /v1/batches/:id/cancel` implement batch operations.
- [ ] **OPENAI-11**: All OpenAI-compatible errors return the standard `{"error":{"message","type","param","code"}}` envelope.
- [ ] **OPENAI-12**: Streaming standardization uses SSE with proper headers, chunk framing, and provider-specific termination semantics.

### Providers (PROV)

- [ ] **PROV-01**: A `Provider` interface defines explicit capability methods for chat, completions, responses, embeddings, images, audio, files, batch, and models.
- [ ] **PROV-02**: Each provider package contains a controller (`provider.go`), feature converters (`chat.go`, `embedding.go`, etc.), and an `ErrorConverter` (`errors.go`).
- [ ] **PROV-03**: OpenAI provider is the reference implementation and supports all chat, embeddings, models, images, audio, files, and batch operations.
- [ ] **PROV-04**: Anthropic provider supports chat (streaming + non-streaming), models, and embeddings with OpenAI↔Anthropic format translation.
- [ ] **PROV-05**: Gemini provider supports chat, embeddings, and audio with OpenAI↔Gemini format translation.
- [ ] **PROV-06**: Groq, Mistral, Cohere, DeepSeek, MiniMax, Fireworks, Together, and Ollama providers support chat and models (OpenAI-compatible passthrough where applicable).
- [ ] **PROV-07**: Bedrock and Vertex providers support chat and embeddings with cloud-specific authentication and model naming.
- [ ] **PROV-08**: Provider errors include metadata (provider name, requested model, request type, status code) for dashboards and logging.
- [ ] **PROV-09**: Per-account/per-key timeout overrides are supported (see GOV-06).
- [ ] **PROV-10**: Per-account model whitelists are supported (see GOV-07).

### Model Catalog (CATALOG)

- [ ] **CATALOG-01**: Catalog loads built-in pricing/model seed data on first startup and supports offline operation.
- [ ] **CATALOG-02**: Catalog syncs pricing from an upstream source when network is available, with a 24-hour refresh interval.
- [ ] **CATALOG-03**: Lookup supports `model|provider|mode` keys with fallback chains (Gemini→Vertex, Bedrock prefix, Vertex prefix strip, Responses→Chat).
- [ ] **CATALOG-04**: `GetProvidersForModel` implements cross-provider resolution (e.g., `claude-3-5-sonnet` resolves to Anthropic, Vertex, Bedrock, OpenRouter).
- [ ] **CATALOG-05**: `IsModelAllowedForProvider` respects `allowed_models: ["*"]` (catalog-validated), explicit whitelists, and deny-by-default empty lists.
- [ ] **CATALOG-06**: Cost calculation handles token-based, image-based, audio-based, and tiered pricing (>128k, >200k tokens).
- [ ] **CATALOG-07**: Custom pricing overrides are persisted in SQLite and layered on top of catalog prices.
- [ ] **CATALOG-08**: Catalog exposes model capabilities (context window, modalities, tool support, reasoning support) for routing decisions.

### Governance and Routing (GOV)

- [ ] **GOV-01**: Virtual keys contain `provider_configs` with `provider`, `allowed_models`, `key_ids`, and optional `weight`.
- [ ] **GOV-02**: Requests with bare model names route via catalog-resolved candidates filtered by the virtual key's provider configs.
- [ ] **GOV-03**: Requests with `provider/model` lock to that provider after validation.
- [ ] **GOV-04**: Weighted selection normalizes weights across eligible candidates and distributes traffic proportionally.
- [ ] **GOV-05**: Automatic fallback chains are generated from candidates sorted by weight unless the request supplies explicit `fallbacks`.
- [ ] **GOV-06**: Per-provider and per-request timeout overrides are supported, with a configurable maximum (default 60s).
- [ ] **GOV-07**: Per-account/per-key model whitelists are layered on top of virtual-key whitelists; stricter whitelist wins.
- [ ] **GOV-08**: Per-key quotas and rate limits are tracked; exhausted keys are skipped during key selection.
- [ ] **GOV-09**: Virtual key authentication uses the `x-g0-vk` header.

### Management API (MGMT)

- [ ] **MGMT-01**: Auth endpoints support login, logout, session validation, and OAuth start/callback/refresh per provider.
- [ ] **MGMT-02**: Settings endpoints support reading and updating system configuration.
- [ ] **MGMT-03**: Provider endpoints support CRUD for providers and suggested-model fetching.
- [ ] **MGMT-04**: Connection endpoints support CRUD for per-provider API keys and OAuth accounts.
- [ ] **MGMT-05**: Key endpoints support CRUD for API keys, including regenerate and scoped usage.
- [ ] **MGMT-06**: Model alias endpoints support CRUD for model aliases and custom pricing overrides.
- [ ] **MGMT-07**: Combo endpoints support CRUD for predefined provider+model combinations.
- [ ] **MGMT-08**: Routing-rule endpoints support CRUD for conditional routing rules.
- [ ] **MGMT-09**: Usage and log endpoints support querying request history, costs, and aggregated usage.
- [ ] **MGMT-10**: Proxy-pool endpoints support CRUD, batch operations, and health tests.
- [ ] **MGMT-11**: Provider-node endpoints support registration, heartbeat, and routing to remote nodes.
- [ ] **MGMT-12**: MCP endpoints support client registration, instance management, tool discovery, tool-group management, and tool execution.
- [ ] **MGMT-13**: Sync endpoints support configuration export/import and optional cloud sync.
- [ ] **MGMT-14**: Translator debug endpoints support capturing request/response transformations for inspection.

### Dashboard (UI)

- [ ] **UI-01**: Login page authenticates against `/api/login` with no hard-coded demo credentials.
- [ ] **UI-02**: Dashboard shell includes navigation, version badge from `/api/version`, and responsive layout.
- [ ] **UI-03**: Providers page supports list, create, edit, delete, and detail views with connections table.
- [ ] **UI-04**: Provider detail shows suggested models and allows loading them into the catalog.
- [ ] **UI-05**: Connections page supports CRUD for per-provider keys and OAuth accounts.
- [ ] **UI-06**: Models page supports catalog browsing, aliases, custom pricing, and disabled-model toggles.
- [ ] **UI-07**: Combos page supports CRUD for multi-model fallback combinations.
- [ ] **UI-08**: Virtual keys page supports CRUD with provider configs, allowed models, weights, and key restrictions.
- [ ] **UI-09**: Routing rules page supports conditional rule CRUD.
- [ ] **UI-10**: Usage page shows charts and tables with provider/model/token/cost breakdowns.
- [ ] **UI-11**: Logs page supports filtering by provider, status, date range, and model.
- [ ] **UI-12**: Proxy pools page supports pool CRUD, deployment to Cloudflare Workers / Deno Deploy, and health tests.
- [ ] **UI-13**: Provider nodes page supports node registration and health status.
- [ ] **UI-14**: MCP pages support clients, instances, accounts, tools, and tool groups.
- [ ] **UI-15**: Translator debug page shows request/response transformations.
- [ ] **UI-16**: Cloud sync page supports export/import and sync status.
- [ ] **UI-17**: Settings page supports endpoint, RTK, caveman, and locale configuration.
- [ ] **UI-18**: All UI uses g0router branding and color palette; 9router references are replaced.

### Platform Features (PLAT)

- [ ] **PLAT-01**: RTK token compression runs before format translation and supports git-diff, grep, find, ls, tree, dedup-log, smart-truncate, read-numbered, and search-list filters.
- [ ] **PLAT-02**: RTK is safe by design: failed filters fall back to the original text.
- [ ] **PLAT-03**: Caveman mode injects a caveman-speak prompt to reduce output token length.
- [ ] **PLAT-04**: 3-tier fallback supports model-level, provider-level, and network-level retries.
- [ ] **PLAT-05**: Combos define ordered fallback chains with subscription → cheap → free logic.
- [ ] **PLAT-06**: Quota tracking fetches per-provider quota where APIs support it and displays reset countdowns.
- [ ] **PLAT-07**: Format translator supports OpenAI ↔ Anthropic ↔ Gemini ↔ Cursor ↔ Kiro ↔ Vertex transformations.
- [ ] **PLAT-08**: Thought separation stores reasoning/thinking content separately from final assistant content.
- [ ] **PLAT-09**: Tool schema sanitizer strips provider-incompatible fields (e.g., `enumDescriptions`, empty `pages`) before translation.
- [ ] **PLAT-10**: OAuth token refresh uses in-flight deduplication to prevent `refresh_token_reused` errors.
- [ ] **PLAT-11**: MCP gateway supports tool registration, discovery, filtering, and execution.
- [ ] **PLAT-12**: Proxy pools support Cloudflare Workers and Deno Deploy relays.
- [ ] **PLAT-13**: Provider nodes support remote registration and request routing.
- [ ] **PLAT-14**: Cloud sync supports encrypted configuration export/import.

### Testing (TEST)

- [ ] **TEST-01**: Every Go package has `_test.go` files written before implementation (TDD).
- [ ] **TEST-02**: Provider converters use table-driven tests with real provider JSON fixtures.
- [ ] **TEST-03**: Routing, key selection, and governance use fakes (interfaces), not mocks.
- [ ] **TEST-04**: Integration tests exercise the full HTTP server with an in-memory SQLite database.
- [ ] **TEST-05**: Playwright E2E tests cover every dashboard page, form, table action, modal, and dialog.
- [ ] **TEST-06**: The E2E mock layer implements the full `/api/*` and `/v1/*` contract as a 1:1 copy of the real backend.
- [ ] **TEST-07**: CI passes `go test ./...`, `go vet ./...`, `npm run build`, and `npx playwright test` on every commit.

### Reliability and Observability (REL)

- [ ] **REL-01**: Request logging records provider, model, key, tokens, cost, latency, status, and error metadata.
- [ ] **REL-02**: SQLite migrations are additive-only; no destructive schema changes.
- [ ] **REL-03**: Secrets are encrypted at rest using reversible `*_enc` columns.
- [ ] **REL-04**: Health and version endpoints are exposed for monitoring.
- [ ] **REL-05**: Per-provider goroutine queues isolate failures and prevent cascading errors.
- [ ] **REL-06**: Streaming handlers gracefully handle client disconnects without leaking goroutines.

---

## Future Requirements (Deferred)

- Semantic caching (BiFrost enterprise feature).
- Adaptive load balancing based on real-time provider performance metrics.
- Guardrails for input/output filtering.
- Cluster mode for horizontal scaling.
- OpenTelemetry integration.
- Native mobile dashboard.

---

## Out of Scope

- Backward compatibility with the previous g0router API surface (this is a clean-slate pivot).
- TypeScript/Node backend runtime.
- Managed cloud service implementation (cloud sync orchestration only; no cloud backend code).

---

## Traceability

_Requirements are mapped to phases in `.planning/ROADMAP.md`._
