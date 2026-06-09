# Milestone v2.0 Roadmap

**Milestone:** v2.0 9router + BiFrost Clean Slate Port  
**Phases:** 19  
**Requirements mapped:** 78  
**Source:** `docs/superpowers/specs/2026-06-08-9router-bifrost-port-design.md`

---

## Overview

| Phase | Name | Goal | Requirements | Success Criteria |
|-------|------|------|--------------|------------------|
| 1 | Scaffolding | Delete old code and set up new directory structure. | TEST-07 | 4 |
| 2 | Schemas + Catalog | Define shared Go types and model catalog. | CATALOG-01..08, OPENAI-11 | 5 |
| 3 | OpenAI Provider | Implement the reference OpenAI provider. | PROV-01..03, OPENAI-01..02 | 5 |
| 4 | OpenAI API Handlers | Expose `/v1/chat/completions`, `/v1/embeddings`, `/v1/models`. | OPENAI-01..05, OPENAI-11..12 | 5 |
| 5 | Anthropic + Gemini Providers | Add converter-based providers. | PROV-04..05, PLAT-07..09 | 5 |
| 6 | Management API Foundation | Auth, settings, provider/connection CRUD. | MGMT-01..04 | 5 |
| 7 | Dashboard Shell + Providers UI | Login, layout, providers list/detail. | UI-01..05 | 5 |
| 8 | Keys + Virtual Keys + Routing | API keys, virtual keys, weighted routing, fallbacks. | GOV-01..05, MGMT-05, MGMT-08, UI-08..09 | 5 |
| 9 | Models + Aliases + Combos | Catalog-backed model management. | CATALOG-05..07, MGMT-06..07, UI-06..07 | 5 |
| 10 | Usage + Logs | Request logging, cost calculation, usage dashboard. | MGMT-09, UI-10..11, REL-01 | 5 |
| 11 | Audio + Images | `/v1/audio/*` and `/v1/images/*`. | OPENAI-07..08 | 4 |
| 12 | Responses API + Batch + Files | OpenAI Responses API, batch, file management. | OPENAI-06, OPENAI-09..10 | 5 |
| 13 | RTK + Caveman + Translator | 9router platform features. | PLAT-01..09, MGMT-14, UI-15..16 | 5 |
| 14 | MCP Gateway | MCP client, tools, tool groups. | MGMT-12, UI-14, PLAT-11 | 5 |
| 15 | Proxy Pools + Provider Nodes | Proxy rotation and remote node routing. | MGMT-10..11, UI-12..13, PLAT-12..13 | 5 |
| 16 | Cloud Sync | Config export/import and sync UI. | MGMT-13, UI-16, PLAT-14 | 4 |
| 17 | Additional Providers | Groq, Mistral, Cohere, DeepSeek, MiniMax, Fireworks, Together, Ollama, Bedrock, Vertex. | PROV-06..07 | 4 |
| 18 | E2E Hardening | Mock API layer, Playwright coverage. | TEST-05..07 | 4 |
| 19 | Polish + Docs | Final QA, docs, deployment verification. | REL-02..06, UI-18 | 5 |

---

## Phase Details

### Phase 1: Scaffolding

**Goal:** Delete old feature code and set up the new directory structure so CI passes an empty build.

**Requirements:** TEST-07

**Success criteria:**
1. `api/`, `internal/`, and `ui/src/` old feature code is removed.
2. New directory structure from the design doc exists under `internal/` and `cmd/g0router/`.
3. `go.mod` is cleaned of unused dependencies.
4. `go test ./...`, `go vet ./...`, and `npm run build` pass (empty or minimal).

**Depends on:** —

---

### Phase 2: Schemas + Catalog

**Goal:** Define all shared Go types and build the model catalog with pricing lookup.

**Requirements:** CATALOG-01..08, OPENAI-11

**Success criteria:**
1. `internal/schemas/` contains all request/response/error types.
2. `internal/catalog/` loads built-in seed data and supports offline operation.
3. Catalog lookup resolves `model|provider|mode` with documented fallback chain.
4. Cross-provider resolution returns correct candidates for `claude-3-5-sonnet` and `gpt-4o`.
5. Cost calculation produces nonzero costs for known models from fixture usage data.

**Depends on:** Phase 1

---

### Phase 3: OpenAI Provider

**Goal:** Implement the reference OpenAI provider with chat, embeddings, models, and streaming.

**Requirements:** PROV-01..03, OPENAI-01..02

**Success criteria:**
1. `internal/providers/openai/` implements the `Provider` interface.
2. Chat completion (non-streaming) returns a correct OpenAI-shaped response against a recorded fixture.
3. Chat completion streaming emits SSE chunks and terminates with `[DONE]`.
4. Embeddings and list-models endpoints return valid responses.
5. Error converter maps OpenAI HTTP errors to the uniform error schema.

**Depends on:** Phase 2

---

### Phase 4: OpenAI API Handlers

**Goal:** Expose the first three `/v1/*` endpoints and validate drop-in OpenAI SDK compatibility.

**Requirements:** OPENAI-01..05, OPENAI-11..12

**Success criteria:**
1. `POST /v1/chat/completions` works with the OpenAI Python SDK pointed at the local server.
2. `POST /v1/embeddings` works with the OpenAI Python SDK.
3. `GET /v1/models` returns models from the catalog and combos.
4. Errors return the standard OpenAI error envelope.
5. Integration tests pass with an in-memory SQLite database.

**Depends on:** Phase 3

---

### Phase 5: Anthropic + Gemini Providers

**Goal:** Add converter-based providers for Anthropic and Gemini with format translation.

**Requirements:** PROV-04..05, PLAT-07..09

**Success criteria:**
1. Anthropic chat completion maps OpenAI requests to Anthropic's Messages API and back.
2. Gemini chat completion maps OpenAI requests to Gemini format and back.
3. Tool schemas are sanitized before translation.
4. Reasoning/thinking content is handled separately from final content where applicable.
5. Streaming works for both providers with correct SSE output.

**Depends on:** Phase 4

---

### Phase 6: Management API Foundation

**Goal:** Build the admin API foundation: auth, settings, providers, and connections.

**Requirements:** MGMT-01..04

**Success criteria:**
1. Login endpoint issues a session cookie/token and protects admin routes.
2. Settings endpoints read and update system configuration.
3. Provider CRUD endpoints persist providers to SQLite.
4. Connection CRUD endpoints persist API keys and OAuth accounts with encrypted secrets.
5. OAuth start/callback/refresh flows work for at least one provider.

**Depends on:** Phase 1

---

### Phase 7: Dashboard Shell + Providers UI

**Goal:** Port the dashboard shell and providers/connections pages from 9router's WebUI.

**Requirements:** UI-01..05

**Success criteria:**
1. Login page authenticates without hard-coded credentials.
2. Dashboard shell shows navigation and version from `/api/version`.
3. Providers list page supports create/edit/delete.
4. Provider detail page shows overview stats, connections table, and models table.
5. Connections page supports CRUD with edit/delete actions.

**Depends on:** Phase 6

---

### Phase 8: Keys + Virtual Keys + Routing

**Goal:** Implement API keys, virtual keys, weighted routing, and automatic fallback chains.

**Requirements:** GOV-01..05, MGMT-05, MGMT-08, UI-08..09

**Success criteria:**
1. API key CRUD and regenerate endpoints work.
2. Virtual key CRUD supports provider configs with allowed models, weights, and key IDs.
3. Requests with `x-g0-vk` route according to virtual key rules.
4. Weighted routing distributes traffic proportionally across eligible providers.
5. Automatic fallback retries on provider failure.

**Depends on:** Phase 4, Phase 6

---

### Phase 9: Models + Aliases + Combos

**Goal:** Build catalog-backed model management with aliases and combos.

**Requirements:** CATALOG-05..07, MGMT-06..07, UI-06..07

**Success criteria:**
1. Model alias CRUD endpoints map user-friendly names to provider-specific model IDs.
2. Custom pricing overrides are persisted and applied in cost calculation.
3. Combo CRUD endpoints define ordered fallback chains.
4. Dashboard models page shows catalog, aliases, and disabled-model toggles.
5. Dashboard combos page supports create/edit/delete of combos.

**Depends on:** Phase 2, Phase 7

---

### Phase 10: Usage + Logs

**Goal:** Implement request logging, cost calculation, and usage/log dashboards.

**Requirements:** MGMT-09, UI-10..11, REL-01

**Success criteria:**
1. Every inference request is logged with provider, model, tokens, cost, and latency.
2. Usage endpoint returns aggregated metrics by provider/model/time range.
3. Logs endpoint supports filtering by provider, status, date range, and model.
4. Dashboard usage page shows charts and tables.
5. Dashboard logs page supports all filters.

**Depends on:** Phase 4, Phase 8

---

### Phase 11: Audio + Images

**Goal:** Implement `/v1/audio/*` and `/v1/images/*` with provider support.

**Requirements:** OPENAI-07..08

**Success criteria:**
1. `POST /v1/audio/speech` returns audio data.
2. `POST /v1/audio/transcriptions` returns transcription JSON.
3. `POST /v1/images/generations` returns image URLs or b64 data.
4. Streaming audio (TTS) emits SSE chunks.
5. At least OpenAI and Gemini providers support these endpoints.

**Depends on:** Phase 3, Phase 5

---

### Phase 12: Responses API + Batch + Files

**Goal:** Implement OpenAI Responses API, file management, and batch operations.

**Requirements:** OPENAI-06, OPENAI-09..10

**Success criteria:**
1. `POST /v1/responses` returns OpenAI Responses API-shaped output.
2. Responses API streaming uses `event:` types and no `[DONE]` marker.
3. File upload/list/retrieve/delete/content endpoints work.
4. Batch create/list/retrieve/cancel endpoints work.
5. Integration tests cover all new endpoints.

**Depends on:** Phase 4

---

### Phase 13: RTK + Caveman + Translator

**Goal:** Port the 9router platform features: RTK compression, Caveman mode, and translator debug.

**Requirements:** PLAT-01..09, MGMT-14, UI-15..16

**Success criteria:**
1. RTK detects and compresses git diff, grep, find, ls, tree, and log tool results.
2. Failed RTK filters silently fall back to original text.
3. Caveman mode reduces output tokens when enabled.
4. Translator debug UI captures and displays request/response transformations.
5. Thought separation is preserved across OpenAI/Anthropic/Gemini formats.

**Depends on:** Phase 5, Phase 7

---

### Phase 14: MCP Gateway

**Goal:** Implement the MCP gateway with clients, instances, tools, and tool groups.

**Requirements:** MGMT-12, UI-14, PLAT-11

**Success criteria:**
1. MCP client registration endpoints work.
2. MCP instance management supports STDIO and SSE connections.
3. Tool discovery and tool-group CRUD endpoints work.
4. Tool execution endpoint invokes registered tools and returns results.
5. Dashboard MCP pages support all CRUD operations.

**Depends on:** Phase 7

---

### Phase 15: Proxy Pools + Provider Nodes

**Goal:** Implement proxy rotation and remote provider node registration.

**Requirements:** MGMT-10..11, UI-12..13, PLAT-12..13

**Success criteria:**
1. Proxy pool CRUD supports HTTP/SOCKS proxies and Cloudflare Workers deployer.
2. Proxy health tests verify proxy reachability.
3. Provider node registration and heartbeat endpoints work.
4. Requests can route through registered remote nodes.
5. Dashboard proxy pools and nodes pages support all operations.

**Depends on:** Phase 7, Phase 8

---

### Phase 16: Cloud Sync

**Goal:** Implement configuration export/import and cloud sync UI.

**Requirements:** MGMT-13, UI-16, PLAT-14

**Success criteria:**
1. Sync export endpoint returns encrypted configuration bundle.
2. Sync import endpoint validates and restores configuration bundle.
3. Optional cloud sync endpoint orchestrates upload/download.
4. Dashboard cloud sync page supports export/import and status display.
5. Sensitive fields are encrypted inside the sync bundle.

**Depends on:** Phase 7

---

### Phase 17: Additional Providers

**Goal:** Add the remaining providers: Groq, Mistral, Cohere, DeepSeek, MiniMax, Fireworks, Together, Ollama, Bedrock, Vertex.

**Requirements:** PROV-06..07

**Success criteria:**
1. Each provider package implements chat completion and list models.
2. Bedrock and Vertex implement embeddings.
3. OpenAI-compatible providers (Groq, Mistral, Fireworks, Together) use minimal converter passthrough.
4. Each provider has fixture-based converter tests.
5. At least one integration test per provider shape passes.

**Depends on:** Phase 4, Phase 5

---

### Phase 18: E2E Hardening

**Goal:** Build the mocked API layer and achieve full Playwright coverage.

**Requirements:** TEST-05..07

**Success criteria:**
1. `ui/e2e/mocks/` implements the full `/api/*` and `/v1/*` contract.
2. Mock layer shares TypeScript types with the real frontend.
3. Playwright tests cover every dashboard page, form, table action, modal, and dialog.
4. E2E tests run against the mock layer without requiring real provider keys.
5. `npx playwright test` passes with zero failures.

**Depends on:** Phase 7, Phase 10

---

### Phase 19: Polish + Docs

**Goal:** Final QA, documentation updates, and deployment verification.

**Requirements:** REL-02..06, UI-18

**Success criteria:**
1. All gates pass: `go test ./...`, `go vet ./...`, `npm run build`, `npx playwright test`.
2. Documentation reflects the new architecture and API surface.
3. Deployment configs (Docker, systemd) work with the new binary.
4. Single binary starts and serves both API and embedded UI.
5. Branding and color palette are consistently applied across the dashboard.

**Depends on:** All previous phases

---

## Dependency Graph

```
Phase 1 ─┬─► Phase 2 ──► Phase 3 ──► Phase 4 ─┬─► Phase 5 ──► Phase 11
         │                                    │
         └─► Phase 6 ──► Phase 7 ─┬─► Phase 8 ─┘
                                  │
                                  ├─► Phase 9
                                  │
                                  ├─► Phase 10 ──► Phase 18
                                  │
                                  ├─► Phase 13
                                  │
                                  ├─► Phase 14
                                  │
                                  ├─► Phase 15
                                  │
                                  ├─► Phase 16
                                  │
                                  └─► Phase 17

Phase 4,5,11,12,17 feed into Phase 19
Phase 18 feeds into Phase 19
```

---

## Execution Waves

### Wave 1: Foundation (Phases 1–4)
Goal: Have a working OpenAI-compatible `/v1/chat/completions` endpoint with the OpenAI provider.

### Wave 2: Core Providers + Admin (Phases 5–8)
Goal: Add Anthropic/Gemini, management API foundation, dashboard shell, keys, and virtual-key routing.

### Wave 3: Catalog + Usage (Phases 9–10)
Goal: Models, aliases, combos, usage tracking, and logs dashboard.

### Wave 4: Advanced API Surface (Phases 11–12)
Goal: Audio, images, Responses API, files, and batch operations.

### Wave 5: 9router Features (Phases 13–17)
Goal: RTK, caveman, translator, MCP, proxy pools, nodes, cloud sync, and additional providers.

### Wave 6: Hardening + Ship (Phases 18–19)
Goal: Playwright E2E coverage, docs, deployment verification, and final gates.
