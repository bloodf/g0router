# Design: g0router as a Go 1:1 Implementation of 9router, with BiFrost's OpenAI-Compatible Layer

**Date:** 2026-06-08  
**Status:** Draft — awaiting review  
**Scope:** Full clean-slate rebuild of g0router backend and dashboard. Keep only project metadata, build toolchain, and deployment scaffolding. Replace all feature code with a Go implementation that ports 9router's behavior and adapts 9router's WebUI, using BiFrost's proven Go patterns for the OpenAI-compatible API layer.

---

## Background

The current g0router codebase has drifted too far from a working, OpenAI-compatible gateway. The decision is to start fresh:

1. **9router** (`decolua/9router`) is the functional target. It already has the features we want: RTK compression, Caveman mode, 3-tier fallback, combos, multi-account per provider, OAuth auto-refresh, quota tracking, format translation, request logging, cloud sync, provider nodes, proxy pools, MCP gateway, and translator debug UI.
2. **BiFrost** (`maximhq/bifrost`) is the architectural reference for the OpenAI-compatible layer. It is written in Go, exposes `/v1/*` endpoints that are drop-in replacements for OpenAI, and has a proven provider interface + request normalization + streaming + error mapping design.
3. g0router will combine the two: **9router's features and WebUI** + **BiFrost's Go architecture for the OpenAI API surface**.

The existing `api/`, `internal/`, and `ui/src/` code will be deleted and rebuilt. The retained skeleton is: `go.mod`, `cmd/g0router/`, `embed.go`, `Dockerfile`, `deploy/`, `ui/package.json` + build toolchain, `ui/public/providers/`, and project metadata files.

---

## Goals

1. Expose a 100% OpenAI-compatible API (`/v1/*`) that works as a drop-in replacement for OpenAI, Anthropic, and Google SDKs.
2. Port all 9router management features behind `/api/*` with the same behavior and WebUI pages.
3. Support 20+ providers out of the box, using BiFrost's provider interface + converter pattern.
4. Deliver a single Go binary with an embedded React dashboard.
5. Maintain full Playwright E2E coverage via a mocked API layer that is a 1:1 copy of the real backend contract.
6. Pass `go test ./...`, `go vet ./...`, `npm run build`, and `npx playwright test` on every commit.

## Non-Goals

- Reimplement 9router in TypeScript/Node. The backend is Go-only.
- Support every BiFrost enterprise feature on day one (adaptive load balancing, semantic caching, guardrails, clustering).
- Maintain backward compatibility with the current g0router API surface.
- Add new providers before the core architecture is stable.

---

## Reference Architectures

### BiFrost (maximhq/bifrost)

BiFrost's Go architecture is the reference for the OpenAI-compatible layer:

```
core/
  schemas/           # Shared Go types (41 files)
    provider.go      # Provider interface with 30+ methods
    chatcompletions.go
    responses.go
    embedding.go
    images.go
    audio.go
    batch.go
    files.go
    errors.go
  providers/
    openai/          # Reference provider implementation
    anthropic/
    gemini/
    bedrock/
    groq/
    utils/           # Shared HTTP client, SSE parsing, error handling
  inference.go       # Routing, fallbacks, streaming dispatch
  bifrost.go         # Main struct, request queuing, provider lifecycle
framework/
  modelcatalog/      # Pricing + model registry
  streaming/         # Accumulator, delta copy, response marshaling
  configstore/       # Persistence backends
transports/bifrost-http/
  handlers/
    inference.go     # /v1/* endpoints
    governance.go    # Virtual keys, budgets, routing
    providers.go     # Provider CRUD
```

Key patterns we adopt:

- **Provider interface** with explicit methods per capability (`ChatCompletion`, `ChatCompletionStream`, `Embedding`, `Speech`, `ImageGeneration`, `ListModels`, etc.).
- **Feature-based converter files** (`chat.go`, `embedding.go`, `speech.go`) with pure `To[Provider]XRequest` / `ToBifrostXResponse` functions.
- **Per-provider `ErrorConverter`** that maps provider HTTP errors to a uniform `BifrostError` schema carrying provider/model/request-type metadata.
- **Model catalog** with in-memory pricing cache, cross-provider model resolution, and tiered pricing.
- **Governance/virtual keys** using `provider_configs` with `allowed_models`, `weight`, and `key_ids`.
- **Request flow**: FastHTTP transport → SDK integration layer → middleware chain → pre-hooks → provider queue → worker → key selection → upstream API call → response / SSE stream.
- **Streaming**: `chan *StreamChunk` + SSE standardization + streaming accumulator for post-hooks.

### 9router (decolua/9router)

9router is the feature target. Its WebUI and management concepts are ported directly:

- Provider management with multiple accounts/keys per provider.
- OAuth flows with auto-refresh for providers that support it.
- RTK token compression for cost reduction.
- Caveman mode (simplified/minimal request path).
- 3-tier fallback: model-level, provider-level, network-level.
- Combos: predefined provider+model combinations with aliases.
- Quota tracking and rate limiting per key.
- Format translation between OpenAI, Anthropic, and Gemini shapes.
- Request logging and usage analytics.
- Cloud sync for configuration backup/restore.
- Provider nodes: remote node registration and routing.
- Proxy pools for rotating egress IPs.
- MCP gateway with tool registration and execution.
- Translator debug UI for inspecting request/response transformations.

---

## Proposed Directory Structure

```
cmd/g0router/
  main.go                         # Entry point: init server, catalog, providers

internal/
  schemas/                        # Shared Go types (BiFrost-style)
    provider.go                   # Provider capability interface
    chat.go                       # Chat completion request/response
    responses.go                  # OpenAI Responses API types
    embedding.go
    images.go
    audio.go
    batch.go
    files.go
    errors.go                     # Uniform error type + OpenAI error envelope
    catalog.go                    # Model + pricing types
    governance.go                 # Virtual key + provider config types
    mcp.go

  server/                         # FastHTTP server + route registration
    server.go
    middleware.go                 # Auth, CORS, request ID, logging
    routes_openai.go              # /v1/* handlers
    routes_admin.go               # /api/* handlers

  api/                            # OpenAI-compatible API handlers
    chat.go
    completions.go
    embeddings.go
    images.go
    audio.go
    responses.go
    models.go
    files.go
    batch.go

  admin/                          # Management API handlers (9router features)
    auth.go
    settings.go
    providers.go
    connections.go                # Per-provider keys / accounts
    keys.go
    models.go                     # Aliases, limits, custom pricing
    combos.go
    routing.go
    usage.go
    logs.go
    proxy_pools.go
    nodes.go
    mcp.go
    sync.go
    translator.go

  providers/                      # Provider implementations
    openai/                       # Reference implementation
      provider.go
      chat.go
      embedding.go
      images.go
      audio.go
      models.go
      errors.go
    anthropic/
    gemini/
    groq/
    mistral/
    cohere/
    fireworks/
    together/
    deepseek/
    minimax/
    ollama/
    bedrock/
    vertex/
    utils/                        # Shared fasthttp client, SSE scanner pools

  inference/                      # Routing, fallbacks, key selection
    router.go
    fallback.go
    keyselector.go
    queue.go
    stream.go

  catalog/                        # Model catalog + pricing
    catalog.go
    pricing.go
    sync.go

  governance/                     # Virtual keys, provider configs, quotas
    virtualkeys.go
    quotas.go
    rbac.go

  auth/                           # Sessions, OAuth, API key auth
    session.go
    oauth.go
    apikey.go

  store/                          # SQLite persistence
    db.go
    migrations.go

  logging/                        # Request log + audit
    logger.go
    requestlog.go

  mcp/                            # MCP gateway
    server.go
    client.go
    tools.go

  config/                         # Runtime config loading
    config.go

  platform/                       # 9router-specific features
    rtk.go
    caveman.go
    combos.go
    translator.go
    sync.go
    nodes.go
    proxypool.go

ui/
  src/
    routes/                       # Ported/adapted from 9router WebUI
    components/
    lib/
    hooks/
  e2e/
    mocks/                        # 1:1 mock API layer
    specs/
```

---

## API Surface

### OpenAI-compatible surface (P0 — must ship first)

These endpoints must be drop-in compatible with the OpenAI SDK and pass standard client tests:

- `POST /v1/chat/completions` (streaming + non-streaming)
- `POST /v1/completions` (legacy)
- `POST /v1/embeddings`
- `GET /v1/models`
- `GET /v1/models/:id`
- `POST /v1/responses` (streaming + non-streaming)
- `POST /v1/images/generations`
- `POST /v1/images/edits`
- `POST /v1/images/variations`
- `POST /v1/audio/speech`
- `POST /v1/audio/transcriptions`
- `POST /v1/audio/translations`
- `POST /v1/files`
- `GET /v1/files`
- `GET /v1/files/:id`
- `DELETE /v1/files/:id`
- `GET /v1/files/:id/content`
- `POST /v1/batches`
- `GET /v1/batches`
- `GET /v1/batches/:id`
- `POST /v1/batches/:id/cancel`

### Management surface (9router features)

- `POST /api/login`, `POST /api/logout`, `GET /api/me`
- `GET /api/settings`, `PUT /api/settings`
- `GET /api/providers`, `POST /api/providers`, `GET /api/providers/:id`, `PUT /api/providers/:id`, `DELETE /api/providers/:id`
- `GET /api/providers/:id/models`, `GET /api/providers/:id/suggested-models`
- `GET /api/connections`, `POST /api/connections`, `PUT /api/connections/:id`, `DELETE /api/connections/:id`
- `GET /api/keys`, `POST /api/keys`, `POST /api/keys/:id/regenerate`, `DELETE /api/keys/:id`
- `GET /api/models`, `POST /api/models/aliases`, `PUT /api/models/aliases/:id`, `DELETE /api/models/aliases/:id`
- `GET /api/combos`, `POST /api/combos`, `PUT /api/combos/:id`, `DELETE /api/combos/:id`
- `GET /api/routing-rules`, `POST /api/routing-rules`, `PUT /api/routing-rules/:id`, `DELETE /api/routing-rules/:id`
- `GET /api/usage`, `GET /api/logs`
- `GET /api/proxy-pools`, `POST /api/proxy-pools`, ...
- `GET /api/nodes`, ...
- `GET /api/mcp/clients`, `POST /api/mcp/instances`, ...
- `POST /api/sync/export`, `POST /api/sync/import`
- `GET /api/translator/debug`, `POST /api/translator/test`
- `GET /api/oauth/:provider/start`, `GET /api/oauth/:provider/callback`, `POST /api/oauth/:provider/refresh`
- `GET /api/version`, `GET /api/health`

---

## Provider Interface

We adopt BiFrost's explicit capability interface. Every provider package implements the full interface and returns "not supported" errors for unsupported operations.

```go
package schemas

type Provider interface {
    // Lifecycle
    GetProvider() ModelProvider
    SetNetworkConfig(config NetworkConfig)

    // Models
    ListModels(ctx *GatewayContext, key Key) (*ListModelsResponse, *ProviderError)

    // Chat
    ChatCompletion(ctx *GatewayContext, key Key, request *ChatRequest) (*ChatResponse, *ProviderError)
    ChatCompletionStream(ctx *GatewayContext, postHookRunner PostHookRunner, key Key, request *ChatRequest) (chan *StreamChunk, *ProviderError)

    // Text completions
    TextCompletion(ctx *GatewayContext, key Key, request *TextCompletionRequest) (*TextCompletionResponse, *ProviderError)
    TextCompletionStream(ctx *GatewayContext, postHookRunner PostHookRunner, key Key, request *TextCompletionRequest) (chan *StreamChunk, *ProviderError)

    // Responses API
    Responses(ctx *GatewayContext, key Key, request *ResponsesRequest) (*ResponsesResponse, *ProviderError)
    ResponsesStream(ctx *GatewayContext, postHookRunner PostHookRunner, key Key, request *ResponsesRequest) (chan *StreamChunk, *ProviderError)

    // Embeddings
    Embedding(ctx *GatewayContext, key Key, request *EmbeddingRequest) (*EmbeddingResponse, *ProviderError)

    // Images
    ImageGeneration(ctx *GatewayContext, key Key, request *ImageGenerationRequest) (*ImageGenerationResponse, *ProviderError)
    ImageGenerationStream(ctx *GatewayContext, postHookRunner PostHookRunner, key Key, request *ImageGenerationRequest) (chan *StreamChunk, *ProviderError)
    ImageEdit(ctx *GatewayContext, key Key, request *ImageEditRequest) (*ImageGenerationResponse, *ProviderError)
    ImageVariation(ctx *GatewayContext, key Key, request *ImageVariationRequest) (*ImageGenerationResponse, *ProviderError)

    // Audio
    Speech(ctx *GatewayContext, key Key, request *SpeechRequest) (*SpeechResponse, *ProviderError)
    SpeechStream(ctx *GatewayContext, postHookRunner PostHookRunner, key Key, request *SpeechRequest) (chan *StreamChunk, *ProviderError)
    Transcription(ctx *GatewayContext, key Key, request *TranscriptionRequest) (*TranscriptionResponse, *ProviderError)
    TranscriptionStream(ctx *GatewayContext, postHookRunner PostHookRunner, key Key, request *TranscriptionRequest) (chan *StreamChunk, *ProviderError)

    // Files + batch
    FileUpload(ctx *GatewayContext, key Key, request *FileUploadRequest) (*FileObject, *ProviderError)
    FileList(ctx *GatewayContext, key Key) (*FileListResponse, *ProviderError)
    FileRetrieve(ctx *GatewayContext, key Key, fileID string) (*FileObject, *ProviderError)
    FileDelete(ctx *GatewayContext, key Key, fileID string) (*FileDeleteResponse, *ProviderError)
    FileContent(ctx *GatewayContext, key Key, fileID string) ([]byte, *ProviderError)
    BatchCreate(ctx *GatewayContext, key Key, request *BatchCreateRequest) (*Batch, *ProviderError)
    BatchList(ctx *GatewayContext, key Key) (*BatchListResponse, *ProviderError)
    BatchRetrieve(ctx *GatewayContext, key Key, batchID string) (*Batch, *ProviderError)
    BatchCancel(ctx *GatewayContext, key Key, batchID string) (*Batch, *ProviderError)

    // Utilities
    CountTokens(ctx *GatewayContext, key Key, request *ChatRequest) (*TokenCountResponse, *ProviderError)
}
```

Each provider package contains:

- `provider.go` — controller that implements the interface, owns the `fasthttp.Client`, and wires HTTP calls.
- `chat.go`, `embedding.go`, `speech.go`, etc. — pure converter functions with no side effects.
- `errors.go` — `ErrorConverter` that maps provider HTTP errors to the uniform error schema.

---

## Model Catalog

The catalog is a centralized, thread-safe registry of models, capabilities, and pricing.

```go
type Catalog interface {
    // Lookup
    Lookup(provider, model string, mode RequestType) (PricingEntry, bool)
    GetModelsForProvider(provider string) []string
    GetProvidersForModel(model string) []string
    IsModelAllowedForProvider(provider, model string, allowedModels []string) bool

    // Cost
    CalculateCost(provider, model string, mode RequestType, usage Usage) Cost

    // Sync
    Sync(ctx context.Context) error
}
```

Behavior:

- Pricing is loaded from a built-in seed file on first startup, then updated from an upstream pricing sheet when network is available.
- Lookups use a fallback chain: `model|provider|mode` → provider aliases (Gemini→Vertex, Bedrock `anthropic.` prefix, Vertex prefix stripping) → Responses→Chat mode fallback.
- Cross-provider resolution allows a request for `claude-3-5-sonnet` to resolve to Anthropic, Vertex, Bedrock, or OpenRouter depending on configured provider configs.
- Custom pricing overrides are stored in SQLite and layered on top of the catalog.

---

## Governance and Routing

Virtual keys are the primary routing boundary. Each virtual key has `provider_configs`:

```go
type ProviderConfig struct {
    Provider      string   `json:"provider"`
    AllowedModels []string `json:"allowed_models"` // ["*"] = catalog-validated allow-all; [] = deny-all
    KeyIDs        []string `json:"key_ids"`        // ["*"] = any key; [] = deny-all
    Weight        *float64 `json:"weight"`         // nil = not in weighted selection
}

type VirtualKey struct {
    ID              string           `json:"id"`
    Name            string           `json:"name"`
    ProviderConfigs []ProviderConfig `json:"provider_configs"`
    Budget          *Budget          `json:"budget,omitempty"`
    RateLimitRPM    *int             `json:"rate_limit_rpm,omitempty"`
}
```

Routing rules:

1. Parse `model` from request. If it is `provider/model`, lock to that provider.
2. If bare model name, use `GetProvidersForModel` to find candidates.
3. Filter candidates by the virtual key's `provider_configs`:
   - Provider must be listed.
   - Model must be in `allowed_models` (or `allowed_models: ["*"]` with catalog validation).
   - At least one allowed key must exist.
4. Weighted selection among remaining candidates. Weights are normalized to sum 1.0.
5. Automatic fallback chain is generated from candidates sorted by weight (highest first), unless the request includes an explicit `fallbacks` array.

Key selection:

- After provider selection, choose a key from the allowed set using weighted round-robin.
- Track per-key quota and rate-limit state; skip keys that are exhausted.

---

## Streaming and Error Handling

### Streaming

- All streaming methods return `chan *StreamChunk`.
- The HTTP handler flushes SSE chunks as they arrive.
- Chat completion SSE uses `data: {...}\n\n` and terminates with `data: [DONE]\n\n`.
- Responses API SSE uses `event: type` lines and does **not** emit `[DONE]`.
- A streaming accumulator collects chunks so post-hooks can observe the full response.

### Errors

All provider errors are normalized to:

```go
type APIError struct {
    Message string  `json:"message"`
    Type    string  `json:"type"`
    Param   *string `json:"param,omitempty"`
    Code    *string `json:"code,omitempty"`
}

type ErrorResponse struct {
    Error APIError `json:"error"`
}
```

Additional metadata is attached for internal logging and dashboards:

```go
type ErrorMeta struct {
    Provider       string
    ModelRequested string
    RequestType    string
    StatusCode     int
    RawBody        []byte
}
```

---

## WebUI

The dashboard is rebuilt in the existing Vite + React 19 + Tailwind 4 + shadcn/ui stack.

- **Source of truth:** 9router's WebUI pages and navigation structure.
- **Adaptation:** replace 9router branding with g0router branding and apply g0router's color palette.
- **Pages to port first:**
  1. Login + dashboard shell
  2. Providers list + provider detail + connections
  3. Models + aliases + combos
  4. API keys + virtual keys
  5. Routing rules
  6. Usage + logs
  7. Settings
  8. MCP (clients, instances, tools, tool groups)
  9. Proxy pools + nodes
  10. Translator debug
  11. Cloud sync

Each page talks to the management API (`/api/*`) using TanStack Query. Forms use existing shadcn components and `react-hook-form` where needed.

---

## Testing Strategy

### Backend tests

- Every package gets `_test.go` files before implementation (TDD).
- Provider converters are tested with table-driven tests against real provider JSON fixtures.
- Routing and key selection use fakes, not mocks.
- Integration tests use `httptest` against the full server with an in-memory SQLite database.

### E2E tests

- A mocked API layer in `ui/e2e/mocks/` implements the full `/api/*` and `/v1/*` contract using Playwright's `route` API.
- The mock layer shares TypeScript types with the real frontend so the contract stays synchronized.
- Tests cover every page, form, table action, modal, and dialog.
- CI runs: `go test ./...`, `go vet ./...`, `npm run build`, `npx playwright test`.

---

## Phased Rollout (GSD)

The implementation is too large for a single plan. We decompose into GSD phases:

| Phase | Focus | Deliverable |
|-------|-------|-------------|
| 1 | Scaffolding | Delete old code, set up new directory structure, CI passes empty build. |
| 2 | Schemas + Catalog | Define all shared Go types, model catalog, pricing lookup, seed data. |
| 3 | OpenAI provider | Full OpenAI provider implementation: chat, embeddings, models, streaming. |
| 4 | OpenAI API handlers | `/v1/chat/completions`, `/v1/embeddings`, `/v1/models` with tests. |
| 5 | Anthropic + Gemini providers | Converter-based providers for the two most important non-OpenAI APIs. |
| 6 | Management API foundation | Auth, settings, provider CRUD, connection CRUD. |
| 7 | Dashboard shell + providers UI | Login, layout, providers list/detail, connections. |
| 8 | Keys + virtual keys + routing | API keys, virtual keys, weighted routing, fallback chains. |
| 9 | Models + aliases + combos | Catalog-backed model management, aliases, combos. |
| 10 | Usage + logs | Request logging, cost calculation, usage dashboard, logs page. |
| 11 | Audio + images | `/v1/audio/*` and `/v1/images/*` with provider support. |
| 12 | Responses API + batch + files | OpenAI Responses API, batch, file management. |
| 13 | 9router feature port: RTK + caveman + translator | RTK compression, caveman mode, format translation debug UI. |
| 14 | MCP gateway | MCP client management, tool registration, execution. |
| 15 | Proxy pools + provider nodes | Proxy rotation and remote node registration. |
| 16 | Cloud sync | Configuration export/import and cloud sync UI. |
| 17 | Additional providers | Groq, Mistral, Cohere, DeepSeek, MiniMax, Fireworks, Together, Ollama, Bedrock, Vertex. |
| 18 | E2E hardening | Mock API layer, Playwright coverage of all pages and flows. |
| 19 | Polish + docs | Final QA, docs, deployment verification. |

---

## 9router PRs / Fixes to Port

The 9router changelog and issue tracker contain fixes that must be carried forward into the Go port. These are not net-new features; they are correctness fixes that should be designed in from the start.

### OpenAI / Format Translation Fixes

| PR | Topic | How We Port It |
|----|-------|----------------|
| #1343 | `json_schema` fallback for providers without native Structured Output | Provider converters detect `response_format: { type: "json_schema" }` and fall back to `json_object` + schema-in-system-prompt when the upstream does not support native JSON schema. |
| #536 | Strip `temperature` for `gpt-5.4` model | Catalog capability flags include `no_temperature`; router strips the parameter before forwarding. |
| #623 | Strip `thinking` / `reasoning_effort` for GitHub Copilot chat completions | Translator layer filters reasoning fields when target provider is Copilot-compatible. |

### Anthropic / Claude Translator Fixes

| PR | Topic | How We Port It |
|----|-------|----------------|
| #1144 | Sanitize `Read` tool args to prevent retry loops from non-Anthropic models | Tool-schema sanitizer runs before translator; strips/renames fields that Anthropic accepts but other providers reject. |
| #1354 | Strip empty `Read.pages` argument in OpenAI-to-Claude translator | Translator normalizes empty collections to omitted fields. |

### Gemini Fixes

| PR | Topic | How We Port It |
|----|-------|----------------|
| #1366 | Forward Gemini output dimensions for embeddings | Gemini embedding converter maps `dimensions` parameter explicitly; do not drop it. |
| #1271, #1428 | Reuse stored OAuth project IDs for quota checks; clearer setup guidance when project missing | OAuth session store persists `project_id`; quota fetcher uses it; dashboard surfaces setup hints. |

### Codex / GitHub Copilot Fixes

| PR | Topic | How We Port It |
|----|-------|----------------|
| #575 | Codex image URL fetches must await before sending upstream | Image fetch helper in translator is async/promise-equivalent in Go (wait for download before building provider request). |
| #591 | Enable Codex Apply/Reset buttons when CLI is installed | Dashboard state reflects CLI detection; keep 9router UI behavior. |

### Proxy / Tunnel / Provider Node Fixes

| PR | Topic | How We Port It |
|----|-------|----------------|
| #1360 | Cloudflare Workers proxy deployer and pool integration | Proxy-pools implementation supports Cloudflare Workers as a deploy target. |
| #1437 | Deno Deploy relays support and improved proxy pools dashboard layout | Proxy pools UI uses 9router layout; Deno Deploy relay added as deploy target. |
| #1300 | Tailscale connection status on Windows | Tunnel manager handles Windows-specific Tailscale path detection. |

### Tool Schema / Provider-Specific Fixes

| PR | Topic | How We Port It |
|----|-------|----------------|
| #566 | Strip `enumDescriptions` from tool schema in antigravity-to-openai | Tool-schema sanitizer drops non-standard `enumDescriptions` before sending to OpenAI-compatible providers. |
| #521 | Multi-model support for Factory Droid CLI tool | Combo system supports multi-model combos natively. |
| #580 | Add GLM-5 and MiniMax-M2.5 models to Kiro provider | Catalog seed data includes these models under the Kiro provider entry. |

### Dashboard / UI Fixes

| PR | Topic | How We Port It |
|----|-------|----------------|
| #1362 | Resolve `setState`-in-effect errors in dashboard components | Follow React best practices in ported components; validate with E2E tests. |

### Additional Open PRs to Evaluate

| PR | Topic | Initial Assessment |
|----|-------|-------------------|
| #1752 | Thought separation (merged recently) | **Must-have.** Design the translator so reasoning/thinking content is stored and exposed separately from final assistant content across OpenAI, Anthropic, and Gemini formats. |
| #1680 | 60-second timeout override | Add per-provider and per-request timeout overrides with a ceiling of 60s (or user-configured max). Store override in provider config and connection settings. |
| #1683 | Per-account model whitelist | Extend `AllowedModels` to support per-account/per-key whitelist in addition to virtual-key-level whitelist. Account whitelist takes precedence if stricter. |

### Future PR Research

Before Phase 3 begins, a focused research pass should review the following 9router PR areas for additional fixes to port:

- Codex streaming / prompt caching stability (v0.4.62 changelog)
- Token refresh in-flight deduplication (v0.4.63 changelog)
- Stream stall / pipe error handling on client disconnect
- OAuth Windows flow fixes
- Kiro provider translation and reasoning content support

---

## Open Questions / Decisions

1. **Pricing source:** Use BiFrost-style upstream pricing sheet with local SQLite cache, plus built-in seed fallback for offline/air-gapped deployments.
2. **Model naming:** Support both `provider/model` and bare model names, with catalog-based resolution.
3. **Virtual key header:** Use `x-g0-vk` (g0router convention) rather than BiFrost's `x-bf-vk`.
4. **OAuth storage:** Continue using encrypted `*_enc` columns in SQLite per existing g0router decision.
5. **Streaming library:** Use fasthttp + manual SSE framing instead of adding a heavy streaming dependency.
6. **Go version:** Keep current Go version in `go.mod`. Only bump if a specific language feature or dependency requires it; do not chase BiFrost's Go workspace version by default.

---

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Massive scope leads to never-ending rewrite | Strict GSD phasing; P0 OpenAI API ships before any UI or management features. |
| Provider API drift breaks converters | Table-driven fixture tests; nightly optional integration tests against live providers. |
| Catalog pricing becomes stale | Background sync + built-in seed fallback + custom pricing overrides. |
| WebUI port diverges from 9router behavior | Keep 9router's page structure but validate every interaction with Playwright against mock API. |
| Performance regressions vs old code | fasthttp, sync.Pool for buffers, per-provider goroutine queues, benchmark in phase 4. |
| Single-binary size bloat | Embed only compressed dashboard build; audit with `go build -ldflags`. |

---

## Success Criteria

- `curl` against `/v1/chat/completions` with an OpenAI SDK produces identical behavior to OpenAI for supported parameters.
- Dashboard supports full CRUD for providers, keys, models, combos, routing rules, usage, and logs.
- `go test ./...`, `go vet ./...`, `npm run build`, and `npx playwright test` pass on every phase commit.
- Single binary runs with `go run ./cmd/g0router` and serves both API and embedded UI.
