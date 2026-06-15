# Bifrost OpenAI-Compatible Surface — Parity Matrix

Reference: `/Users/heitor/Developer/github.com/bloodf/_refs/bifrost` @ `ca21298`
Target: `/Users/heitor/Developer/github.com/bloodf/g0router`

---

## Endpoint Inventory

| ID | Behavior | Evidence (file:line) | g0router status | Notes |
|---|---|---|---|---|
| PAR-BF-OAI-001 | Route `POST /v1/chat/completions` registered with fasthttp | `transports/bifrost-http/handlers/inference.go:724` | HAVE | `internal/server/routes_openai.go:15` |
| PAR-BF-OAI-002 | Route `POST /v1/completions` registered with fasthttp | `transports/bifrost-http/handlers/inference.go:723` | HAVE | Route `r.POST("/v1/completions", completions.Handle)` (`internal/server/routes_openai.go:102`); handler `CompletionsHandler.Handle` (`internal/api/completions.go:53`) dispatches to `provider.TextCompletion`, returns the bare OpenAI `TextCompletionResponse`; tests `internal/api/completions_test.go` (bf-openai-1) |
| PAR-BF-OAI-003 | Route `POST /v1/responses` registered with fasthttp | `transports/bifrost-http/handlers/inference.go:725` | MISSING | Schema exists (`internal/schemas/responses.go:4-23`) but no route registered |
| PAR-BF-OAI-004 | Route `POST /v1/responses/input_tokens` (count tokens) registered | `transports/bifrost-http/handlers/inference.go:732` | MISSING | `CountTokens` stubbed in OpenAI provider (`internal/providers/openai/stubs.go:93-95`) |
| PAR-BF-OAI-005 | Route `POST /v1/responses/compact` (compaction) registered | `transports/bifrost-http/handlers/inference.go:733` | MISSING | No schema or route for compaction |
| PAR-BF-OAI-006 | Route `POST /v1/embeddings` registered with fasthttp | `transports/bifrost-http/handlers/inference.go:726` | HAVE | `internal/server/routes_openai.go:16` |
| PAR-BF-OAI-007 | Route `POST /v1/audio/speech` registered with fasthttp | `transports/bifrost-http/handlers/inference.go:729` | HAVE | bf-openai-2: route `internal/server/routes_openai.go` + handler `internal/api/audio.go` (`AudioHandler.Speech`, raw audio bytes + upstream Content-Type, no envelope); openai provider impl `internal/providers/openai/audio.go` (`Speech`) over the former stub |
| PAR-BF-OAI-008 | Route `POST /v1/audio/transcriptions` registered with fasthttp | `transports/bifrost-http/handlers/inference.go:730` | HAVE | bf-openai-2: route `internal/server/routes_openai.go` + handler `internal/api/audio.go` (`AudioHandler.Transcription`, multipart parse → bare `TranscriptionResponse`); openai provider impl `internal/providers/openai/audio.go` (`Transcription`, multipart out) |
| PAR-BF-OAI-009 | Route `POST /v1/images/generations` registered with fasthttp | `transports/bifrost-http/handlers/inference.go:731` | HAVE | bf-openai-2: route `internal/server/routes_openai.go` + handler `internal/api/images.go` (`ImagesHandler.Generations`, bare `ImageGenerationResponse`); openai provider impl `internal/providers/openai/images.go` (`ImageGeneration`) |
| PAR-BF-OAI-010 | Route `POST /v1/images/edits` registered with fasthttp | `transports/bifrost-http/handlers/inference.go:734` | HAVE | bf-openai-2: route `internal/server/routes_openai.go` + handler `internal/api/images.go` (`ImagesHandler.Edits`, multipart image+mask → bare `ImageGenerationResponse`); openai provider impl `internal/providers/openai/images.go` (`ImageEdit`, multipart out) |
| PAR-BF-OAI-011 | Route `POST /v1/images/variations` registered with fasthttp | `transports/bifrost-http/handlers/inference.go:735` | HAVE | bf-openai-2: route `internal/server/routes_openai.go` + handler `internal/api/images.go` (`ImagesHandler.Variations`, multipart image → bare `ImageGenerationResponse`); openai provider impl `internal/providers/openai/images.go` (`ImageVariation`, multipart out) |
| PAR-BF-OAI-012 | Route `POST /v1/videos` (generation) registered with fasthttp | `transports/bifrost-http/handlers/inference.go:736` | MISSING | No video schema or route in g0router |
| PAR-BF-OAI-013 | Route `GET /v1/videos/{video_id}` (retrieve) registered | `transports/bifrost-http/handlers/inference.go:745` | MISSING | No video schema or route in g0router |
| PAR-BF-OAI-014 | Route `GET /v1/videos/{video_id}/content` (download) registered | `transports/bifrost-http/handlers/inference.go:746` | MISSING | No video schema or route in g0router |
| PAR-BF-OAI-015 | Route `DELETE /v1/videos/{video_id}` registered | `transports/bifrost-http/handlers/inference.go:747` | MISSING | No video schema or route in g0router |
| PAR-BF-OAI-016 | Route `POST /v1/videos/{video_id}/remix` registered | `transports/bifrost-http/handlers/inference.go:748` | MISSING | No video schema or route in g0router |
| PAR-BF-OAI-017 | Route `GET /v1/videos` (list) registered | `transports/bifrost-http/handlers/inference.go:744` | MISSING | No video schema or route in g0router |
| PAR-BF-OAI-018 | Route `GET /v1/models` registered with fasthttp | `transports/bifrost-http/handlers/inference.go:720` | HAVE | `internal/server/routes_openai.go:17` |
| PAR-BF-OAI-019 | Route `GET /v1/models/{id}` registered with fasthttp | `transports/bifrost-http/integrations/openai.go:1351` | HAVE | Route `r.GET("/v1/models/{param}", models.GetOrByKind)` (`internal/server/routes_openai.go:101`); `GetOrByKind` (`internal/api/models.go:387`) dispatches non-kind params to `Get` (`internal/api/models.go:449-482`) which filters to ONE model by id and 404s on miss; prior PARTIAL note was stale (claimed no filtering); regression pinned by `TestModelsGetByID_RegressionSingleAndMiss` (`internal/api/models_test.go`, bf-openai-1) |
| PAR-BF-OAI-020 | Route `POST /v1/batches` registered with fasthttp | `transports/bifrost-http/handlers/inference.go:757` | HAVE | Route `r.POST("/v1/batches", batches.Create)` (`internal/server/routes_openai.go`); handler `BatchesHandler.Create` (`internal/api/batches.go`) → `provider.BatchCreate` (`internal/providers/openai/batches.go`), bare `*Batch` JSON (no admin envelope); Option A stateless passthrough; bf-openai-3 |
| PAR-BF-OAI-021 | Route `GET /v1/batches` registered with fasthttp | `transports/bifrost-http/handlers/inference.go:758` | HAVE | Route `r.GET("/v1/batches", batches.List)` (`internal/server/routes_openai.go`); handler `BatchesHandler.List` (`internal/api/batches.go`) → `provider.BatchList` (`internal/providers/openai/batches.go`), bare `*BatchListResponse`; bf-openai-3 |
| PAR-BF-OAI-022 | Route `GET /v1/batches/{batch_id}` registered with fasthttp | `transports/bifrost-http/handlers/inference.go:759` | HAVE | Route `r.GET("/v1/batches/{batch_id}", batches.Retrieve)` (`internal/server/routes_openai.go`); handler `BatchesHandler.Retrieve` reads `{batch_id}` param, empty→400 (`internal/api/batches.go`) → `provider.BatchRetrieve` (`internal/providers/openai/batches.go`); bf-openai-3 |
| PAR-BF-OAI-023 | Route `POST /v1/batches/{batch_id}/cancel` registered | `transports/bifrost-http/handlers/inference.go:760` | HAVE | Route `r.POST("/v1/batches/{batch_id}/cancel", batches.Cancel)` (`internal/server/routes_openai.go`); handler `BatchesHandler.Cancel` reads `{batch_id}` param (`internal/api/batches.go`) → `provider.BatchCancel` POST `/v1/batches/{id}/cancel` (`internal/providers/openai/batches.go`); bf-openai-3 |
| PAR-BF-OAI-024 | Route `GET /v1/batches/{batch_id}/results` registered | `transports/bifrost-http/handlers/inference.go:761` | MISSING | ESCALATED (bf-openai-3 §8 ESC-BATCH-RESULTS): NO provider interface method + NO schema (`internal/schemas/provider.go:102-105` declares only Create/List/Retrieve/Cancel; no `BatchResults`/`batch_results` symbol anywhere). A literal `/results` route requires a non-additive interface change touching all 43 providers — OUT of the buildable-additive bifrost phase. Capability IS reachable: OpenAI exposes batch results via the batch's `output_file_id` fetched through `GET /v1/files/{output_file_id}/content` = **PAR-BF-OAI-029** (HAVE, this plan). STAYS MISSING; see open-questions.md ESC-BATCH-RESULTS |
| PAR-BF-OAI-025 | Route `POST /v1/files` (upload) registered with fasthttp | `transports/bifrost-http/handlers/inference.go:770` | HAVE | Route `r.POST("/v1/files", files.Upload)` (`internal/server/routes_openai.go`); handler `FilesHandler.Upload` parses multipart/form-data (file+purpose, reuses SHIPPED `isMultipart`/`readMultipartFile`/`formValue`) (`internal/api/files.go`) → `provider.FileUpload` multipart-out (`internal/providers/openai/files.go`), bare `*FileObject`; bf-openai-3 |
| PAR-BF-OAI-026 | Route `GET /v1/files` (list) registered with fasthttp | `transports/bifrost-http/handlers/inference.go:771` | HAVE | Route `r.GET("/v1/files", files.List)` (`internal/server/routes_openai.go`); handler `FilesHandler.List` (`internal/api/files.go`) → `provider.FileList` (`internal/providers/openai/files.go`), bare `*FileListResponse`; bf-openai-3 |
| PAR-BF-OAI-027 | Route `GET /v1/files/{file_id}` (retrieve) registered | `transports/bifrost-http/handlers/inference.go:772` | HAVE | Route `r.GET("/v1/files/{file_id}", files.Retrieve)` (`internal/server/routes_openai.go`); handler `FilesHandler.Retrieve` reads `{file_id}` param, empty→400 (`internal/api/files.go`) → `provider.FileRetrieve` (`internal/providers/openai/files.go`), bare `*FileObject`; bf-openai-3 |
| PAR-BF-OAI-028 | Route `DELETE /v1/files/{file_id}` registered | `transports/bifrost-http/handlers/inference.go:773` | HAVE | Route `r.DELETE("/v1/files/{file_id}", files.Delete)` (`internal/server/routes_openai.go`); handler `FilesHandler.Delete` reads `{file_id}` param (`internal/api/files.go`) → `provider.FileDelete` DELETE `/v1/files/{id}` (`internal/providers/openai/files.go`), bare `*FileDeleteResponse`; bf-openai-3 |
| PAR-BF-OAI-029 | Route `GET /v1/files/{file_id}/content` registered | `transports/bifrost-http/handlers/inference.go:774` | HAVE | Route `r.GET("/v1/files/{file_id}/content", files.Content)` (`internal/server/routes_openai.go`); handler `FilesHandler.Content` reads `{file_id}` param (`internal/api/files.go`) → `provider.FileContent` (`internal/providers/openai/files.go`) which clones upstream `resp.Body()`; writes RAW bytes + `Content-Type: application/octet-stream` (NOT JSON, ESC-FILE-CONTENT-BYTES); also serves batch results via a batch's `output_file_id` (cross-ref 024); bf-openai-3 |
| PAR-BF-OAI-030 | Route `POST /v1/containers` registered with fasthttp | `transports/bifrost-http/handlers/inference.go:782` | MISSING | No container schema or route in g0router |
| PAR-BF-OAI-031 | Route `GET /v1/containers` registered with fasthttp | `transports/bifrost-http/handlers/inference.go:783` | MISSING | No container schema or route in g0router |
| PAR-BF-OAI-032 | Route `GET /v1/containers/{container_id}` registered | `transports/bifrost-http/handlers/inference.go:784` | MISSING | No container schema or route in g0router |
| PAR-BF-OAI-033 | Route `DELETE /v1/containers/{container_id}` registered | `transports/bifrost-http/handlers/inference.go:785` | MISSING | No container schema or route in g0router |
| PAR-BF-OAI-034 | Route `POST /v1/containers/{container_id}/files` registered | `transports/bifrost-http/handlers/inference.go:794` | MISSING | No container file schema or route in g0router |
| PAR-BF-OAI-035 | Route `GET /v1/containers/{container_id}/files` registered | `transports/bifrost-http/handlers/inference.go:795` | MISSING | No container file schema or route in g0router |
| PAR-BF-OAI-036 | Route `GET /v1/containers/{container_id}/files/{file_id}` registered | `transports/bifrost-http/handlers/inference.go:796` | MISSING | No container file schema or route in g0router |
| PAR-BF-OAI-037 | Route `GET /v1/containers/{container_id}/files/{file_id}/content` registered | `transports/bifrost-http/handlers/inference.go:797` | MISSING | No container file schema or route in g0router |
| PAR-BF-OAI-038 | Route `DELETE /v1/containers/{container_id}/files/{file_id}` registered | `transports/bifrost-http/handlers/inference.go:798` | MISSING | No container file schema or route in g0router |
| PAR-BF-OAI-039 | Route `POST /v1/rerank` registered with fasthttp | `transports/bifrost-http/handlers/inference.go:727` | MISSING | No rerank schema or route in g0router |
| PAR-BF-OAI-040 | Route `POST /v1/ocr` registered with fasthttp | `transports/bifrost-http/handlers/inference.go:728` | MISSING | No OCR schema or route in g0router |
| PAR-BF-OAI-041 | Alias routes without `/v1` prefix for completions, chat, embeddings, etc. | `transports/bifrost-http/integrations/openai.go:273-342` | MISSING | g0router only registers `/v1/*` paths |
| PAR-BF-OAI-042 | Azure wildcard route `POST /openai/openai/deployments/{deploymentPath:*}` | `transports/bifrost-http/integrations/openai.go:276` | MISSING | No Azure deployment path parsing in g0router |
| PAR-BF-OAI-043 | Async mirror routes under `/v1/async/*` | `transports/bifrost-http/handlers/asyncinference.go:65` | MISSING | No async job system in g0router |
| PAR-BF-OAI-044 | WebSocket Responses API route `GET /v1/responses` | `transports/bifrost-http/handlers/wsresponses.go:76` | MISSING | No WebSocket upgrade handler in g0router |

---

## Request/Response Normalization

| ID | Behavior | Evidence (file:line) | g0router status | Notes |
|---|---|---|---|---|
| PAR-BF-OAI-101 | Chat request strips `cache_control` unless provider is OpenRouter | `core/providers/openai/types.go:135-146` | MISSING | g0router passes `ChatRequest` directly to provider without field stripping |
| PAR-BF-OAI-102 | Chat request drops Anthropic server-tool shapes (`Function==nil && Custom==nil`) | `core/providers/openai/types.go:206-208` | MISSING | g0router has no Anthropic-specific normalization |
| PAR-BF-OAI-103 | Chat request strips Anthropic-only tool flags | `core/providers/openai/types.go:208-209` | MISSING | g0router `Tool` struct (`schemas/chat.go:51-54`) lacks these fields entirely |
| PAR-BF-OAI-104 | Chat request maps `reasoning.effort` to top-level `reasoning_effort` | `core/providers/openai/types.go:135` | MISSING | g0router `ChatRequest` has no `reasoning` field |
| PAR-BF-OAI-105 | Responses request drops `web_fetch` and `memory` tool types | `core/providers/openai/types.go:772-774` | MISSING | g0router `ResponsesRequest` has no tool-type filtering |
| PAR-BF-OAI-106 | Responses request strips `CacheControl` and Anthropic-only flags from `ResponsesTool` | `core/providers/openai/types.go:776-806` | MISSING | g0router `ResponsesRequest` has no such filtering |
| PAR-BF-OAI-107 | Responses request sets `reasoning.max_tokens` to `nil` before marshaling | `core/providers/openai/types.go:826` | MISSING | g0router `ReasoningConfig` (`schemas/responses.go:26-29`) lacks `max_tokens` |
| PAR-BF-OAI-108 | Batch ID normalization for Gemini (`batches/` ↔ `batches-`) | `transports/bifrost-http/integrations/openai.go:1415,1435,1565,1623,1678` | MISSING | g0router has no batch route and no ID normalization |
| PAR-BF-OAI-109 | Batch ID normalization for Bedrock (base64 encode/decode ARNs) | `transports/bifrost-http/integrations/openai.go:1420,1438,1570,1626,1680` | MISSING | g0router has no batch route and no ID normalization |
| PAR-BF-OAI-110 | File ID normalization for Gemini (`files/` ↔ `files-`) | `transports/bifrost-http/integrations/openai.go:1729,1790,1829,1893` | MISSING | g0router has no file route and no ID normalization |
| PAR-BF-OAI-111 | File ID normalization for Bedrock (base64 encode/decode) | `transports/bifrost-http/integrations/openai.go:1731,1795,1844,1896` | MISSING | g0router has no file route and no ID normalization |
| PAR-BF-OAI-112 | Video ID parsing uses `provider:id` format | `transports/bifrost-http/integrations/openai.go:2029-2073` | MISSING | g0router has no video endpoints |
| PAR-BF-OAI-113 | Container ops force `SendBackRawRequest`, `SendBackRawResponse`, `StoreRawRequestResponse` to `true` | `transports/bifrost-http/handlers/inference.go:608` | MISSING | g0router has no container endpoints |
| PAR-BF-OAI-114 | ExtraParams passthrough: unknown JSON keys collected into `ExtraParams` | `transports/bifrost-http/handlers/inference.go:637` | MISSING | g0router unmarshals directly into struct; unknown fields are discarded |
| PAR-BF-OAI-115 | ExtraParams from multipart forms manually iterated | `transports/bifrost-http/handlers/inference.go:3008-3095` | MISSING | g0router has no multipart handlers |
| PAR-BF-OAI-116 | Large-payload pre-hook back-fills `model` and `stream` when body skipped | `transports/bifrost-http/integrations/openai.go:60-100` | MISSING | g0router has no large-payload mode |
| PAR-BF-OAI-117 | Azure endpoint pre-hook parses deployment path and sets `azure/<deploymentID>` model | `transports/bifrost-http/integrations/openai.go:533` | MISSING | g0router `router.Resolve` (`internal/inference/router.go:14-63`) has no Azure deployment path logic |
| PAR-BF-OAI-118 | Azure SDK detection via `User-Agent` substring `AzureOpenAI` | `transports/bifrost-http/integrations/openai.go:56-58` | MISSING | g0router has no Azure SDK detection |
| PAR-BF-OAI-119 | `isModelBlockedByList` logic moved into `BlackList.IsBlocked` | `core/schemas/blacklist.go` (ref SHA ca21298) | EXTRA | g0router has no blacklist / model-block feature |

---

## Streaming

| ID | Behavior | Evidence (file:line) | g0router status | Notes |
|---|---|---|---|---|
| PAR-BF-OAI-201 | SSE headers set **after** stream setup so provider errors return JSON | `transports/bifrost-http/handlers/inference.go:1652-1666` | MISSING | g0router sets SSE headers before calling provider (`internal/api/chat.go:42-44`), so provider errors return `text/event-stream` with JSON body mismatch |
| PAR-BF-OAI-202 | SSE reader bypasses fasthttp internal pipe for one event per chunk | `transports/bifrost-http/handlers/inference.go:1698` (`lib.NewSSEStreamReader`) | MISSING | g0router writes directly to `ctx` without a streaming reader abstraction |
| PAR-BF-OAI-203 | SSE event format includes `event: <type>` for Responses and image-gen streams | `transports/bifrost-http/handlers/inference.go:1774,1834` | MISSING | g0router always emits plain `data: <json>\n\n` with no `event:` prefix (`internal/api/chat.go:54-56`) |
| PAR-BF-OAI-204 | `[DONE]` marker skipped when `includeEventType=true` or `skipDoneMarker=true` | `transports/bifrost-http/handlers/inference.go:1852` | MISSING | g0router always emits `data: [DONE]\n\n` (`internal/api/chat.go:58`) |
| PAR-BF-OAI-205 | Per-converter raw passthrough when `Provider == OpenAI && RawResponse != nil` | `transports/bifrost-http/integrations/openai.go:474,617,737` | MISSING | g0router provider always unmarshals and remarshals; no raw passthrough |
| PAR-BF-OAI-206 | Chat completion streaming supported end-to-end | `transports/bifrost-http/handlers/inference.go:724` | HAVE | g0router chat streaming works (`internal/api/chat.go:41-59`) |
| PAR-BF-OAI-207 | Text completion streaming supported | `transports/bifrost-http/handlers/inference.go:723` | HAVE | openai `Provider.TextCompletionStream` implemented (`internal/providers/openai/completions.go:74`): SSE drain via `NewSSEScanner`, `[DONE]` terminator, malformed-chunk abort (AUD-045), post-hook honored (AUD-047); handler stream path `internal/api/completions.go` sets `text/event-stream` on `stream:true` and frames via `writeSSEStream`; tests `internal/providers/openai/completions_test.go` + `internal/api/completions_test.go` (bf-openai-1) |
| PAR-BF-OAI-208 | Responses streaming supported | `transports/bifrost-http/handlers/inference.go:725` | MISSING | Provider stubbed (`stubs.go:21-23`) |
| PAR-BF-OAI-209 | Speech streaming supported | `transports/bifrost-http/handlers/inference.go:729` | HAVE | bf-openai-2 (Option A SSE-drain): `AudioHandler.Speech` `stream:true` → `provider.SpeechStream` (`internal/providers/openai/audio.go`), SSE-framed via `writeSSEStream` + `[DONE]`; hermetic test passes (upstream SSE frames pass through) |
| PAR-BF-OAI-210 | Transcription streaming supported | `transports/bifrost-http/handlers/inference.go:730` | HAVE | bf-openai-2 (Option A SSE-drain): `AudioHandler.Transcription` `stream` form-field → `provider.TranscriptionStream` (multipart body w/ `stream=true`), SSE-framed + `[DONE]`; malformed-chunk abort (AUD-045) tested |
| PAR-BF-OAI-211 | Image generation streaming supported | `transports/bifrost-http/handlers/inference.go:731` | HAVE | bf-openai-2 (Option A SSE-drain): `ImagesHandler.Generations` `stream:true` → `provider.ImageGenerationStream` (`internal/providers/openai/images.go`), SSE-framed + `[DONE]`; malformed-chunk abort (AUD-045) tested |

---

## Error Envelope

| ID | Behavior | Evidence (file:line) | g0router status | Notes |
|---|---|---|---|---|
| PAR-BF-OAI-301 | `BifrostError` struct with `event_id`, `type`, `is_bifrost_error`, `status_code`, `error` | `core/schemas/bifrost.go:1687-1696` | MISSING | g0router uses flat `error: {message, type, code}` envelope (`internal/api/errors.go:10-24`) |
| PAR-BF-OAI-302 | `ErrorField` struct with `Type`, `Code`, `Message`, `Error`, `Param`, `EventID` | `core/schemas/bifrost.go:1776-1783` | MISSING | g0router `APIError` (`internal/schemas/errors.go:4-9`) lacks `Param` and `EventID` fields |
| PAR-BF-OAI-303 | Status-code fallback: explicit → 400 for `!IsBifrostError` → 500 default | `transports/bifrost-http/handlers/utils.go:113-119` | PARTIAL | g0router `writeError` sets explicit status but has no `IsBifrostError` discriminator (`internal/api/errors.go:10-24`) |
| PAR-BF-OAI-304 | SSE error frames send `event: error` or `data: {error}` depending on stream type | `transports/bifrost-http/handlers/utils.go:144-158` | MISSING | g0router has no SSE-specific error formatting; streaming errors are written as plain JSON after headers set |
| PAR-BF-OAI-305 | Provider error passthrough preserves upstream `type`, `code`, `param`, `message` | `core/providers/openai/types.go` (error converters) | HAVE | g0router `ProviderError` → `writeError` preserves `type`, `message`, `code` (`internal/providers/openai/errors.go:19-37`) |

---

## Capability Flags

| ID | Behavior | Evidence (file:line) | g0router status | Notes |
|---|---|---|---|---|
| PAR-BF-OAI-401 | `AllowedRequests` struct gates 50+ operations per provider key | `core/schemas/provider.go:317-374` | MISSING | g0router `Provider` interface exists but has no per-key capability flags |
| PAR-BF-OAI-402 | `ProviderFeatureSupport` struct defines 30+ feature flags per provider | `core/providers/anthropic/types.go:104-139` | MISSING | g0router has no provider feature support matrix |
| PAR-BF-OAI-403 | `ProviderFeatures` map instantiates flags for Anthropic, Vertex, Bedrock, Azure | `core/providers/anthropic/types.go:146-260` | MISSING | g0router has no provider feature support matrix |
| PAR-BF-OAI-404 | Beta headers gated per provider via `ProviderFeatures` | `core/providers/anthropic/types.go:104-139` | MISSING | g0router has no beta header filtering |
| PAR-BF-OAI-405 | WebSocket capability interface `WebSocketCapableProvider` | `core/schemas/provider.go:714` | MISSING | g0router has no WebSocket capability abstraction |

---

## Async & WebSocket

| ID | Behavior | Evidence (file:line) | g0router status | Notes |
|---|---|---|---|---|
| PAR-BF-OAI-501 | Async handler mirrors 11 endpoints under `/v1/async/*` | `transports/bifrost-http/handlers/asyncinference.go:27-39,65` | MISSING | g0router has no async job subsystem |
| PAR-BF-OAI-502 | Async endpoints reject `stream=true` with 400 | `transports/bifrost-http/handlers/asyncinference.go:109,147,183` | MISSING | g0router has no async endpoints |
| PAR-BF-OAI-503 | Async endpoints return 202 with job status payload | `transports/bifrost-http/handlers/asyncinference.go:135,172,210` | MISSING | g0router has no async endpoints |
| PAR-BF-OAI-504 | WebSocket Responses API upgrades `GET /v1/responses` | `transports/bifrost-http/handlers/wsresponses.go:76,87` | MISSING | g0router has no WebSocket handler |
| PAR-BF-OAI-505 | WebSocket event loop supports `response.create` only | `transports/bifrost-http/handlers/wsresponses.go:137` | MISSING | g0router has no WebSocket handler |
| PAR-BF-OAI-506 | WebSocket forces `store=true` unless provider config has `DisableStore` | `transports/bifrost-http/handlers/wsresponses.go:180-191` | MISSING | g0router has no WebSocket handler |
| PAR-BF-OAI-507 | WebSocket tries native provider WS then falls back to HTTP bridge | `transports/bifrost-http/handlers/wsresponses.go:233,239` | MISSING | g0router has no WebSocket handler |

---

## Data models

### Bifrost request union (reference)
- `BifrostRequest` (`core/schemas/bifrost.go:457`) embeds every sub-request via tagged union.

### Chat completions
**Bifrost:**
- `BifrostChatRequest` (`core/schemas/chatcompletions.go:14-21`): `Provider`, `Model`, `Input []ChatMessage`, `Params *ChatParameters`, `Fallbacks`, `RawRequestBody`
- `BifrostChatResponse` (`core/schemas/chatcompletions.go:36-50`): `ID`, `Choices []BifrostResponseChoice`, `Created`, `Model`, `Object`, `ServiceTier`, `SystemFingerprint`, `Usage *BifrostLLMUsage`, `ExtraFields`, `ExtraParams`, `SearchResults`, `Videos`

**g0router:**
- `ChatRequest` (`internal/schemas/chat.go:4-21`): `Model`, `Messages []Message`, `Temperature`, `MaxTokens`, `TopP`, `N`, `Stream`, `Stop`, `PresencePenalty`, `FrequencyPenalty`, `LogitBias`, `User`, `Tools`, `ToolChoice`, `ResponseFormat`, `Seed`
- `ChatResponse` (`internal/schemas/chat.go:24-31`): `ID`, `Object`, `Created`, `Model`, `Choices []Choice`, `Usage *Usage`
- `StreamChunk` (`internal/schemas/chat.go:138-145`): `ID`, `Object`, `Created`, `Model`, `Choices []StreamChoice`, `Usage *Usage`

### Text completions
**Bifrost:**
- `BifrostTextCompletionRequest` (`core/schemas/textcompletions.go:8-15`): `Provider`, `Model`, `Input *TextCompletionInput`, `Params *TextCompletionParameters`, `Fallbacks`, `RawRequestBody`
- `BifrostTextCompletionResponse` (`core/schemas/textcompletions.go:67-75`): `ID`, `Choices []BifrostResponseChoice`, `Model`, `Object`, `SystemFingerprint`, `Usage`, `ExtraFields`

**g0router:**
- `TextCompletionRequest` (`internal/schemas/completions.go:4-21`): `Model`, `Prompt`, `Suffix`, `MaxTokens`, `Temperature`, `TopP`, `N`, `Stream`, `Logprobs`, `Echo`, `Stop`, `PresencePenalty`, `FrequencyPenalty`, `BestOf`, `LogitBias`, `User`
- `TextCompletionResponse` (`internal/schemas/completions.go:24-31`): `ID`, `Object`, `Created`, `Model`, `Choices []TextCompletionChoice`, `Usage *Usage`

### Embeddings
**Bifrost:**
- `BifrostEmbeddingRequest` (`core/schemas/embedding.go:8-15`): `Provider`, `Model`, `Input *EmbeddingInput`, `Params *EmbeddingParameters`, `Fallbacks`, `RawRequestBody`
- `BifrostEmbeddingResponse` (`core/schemas/embedding.go:21-27`): `Data []EmbeddingData`, `Model`, `Object`, `Usage`, `ExtraFields`
- `EmbeddingInput` (`core/schemas/embedding.go:39-45`): `Text`, `Texts`, `Embedding`, `Embeddings` (one-of)

**g0router:**
- `EmbeddingRequest` (`internal/schemas/embedding.go:4-10`): `Input any`, `Model`, `EncodingFormat`, `Dimensions`, `User`
- `EmbeddingResponse` (`internal/schemas/embedding.go:13-18`): `Object`, `Data []Embedding`, `Model`, `Usage *Usage`
- `Embedding` (`internal/schemas/embedding.go:21-25`): `Object`, `Embedding []float64`, `Index`

### Responses API
**Bifrost:**
- `BifrostResponsesRequest` (`core/schemas/responses.go:37-44`): `Provider`, `Model`, `Input []ResponsesMessage`, `Params *ResponsesParameters`, `Fallbacks`, `RawRequestBody`
- `BifrostResponsesResponse` (`core/schemas/responses.go:105+`): `ID`, `Object`, `CreatedAt`, `Model`, `Output []ResponsesOutputItem`, `Status`, `Usage`, `Error`, `IncompleteDetails`, etc.
- `BifrostCompactionRequest` (`core/schemas/responses.go:52-64`): `Provider`, `Model`, `Input`, `Instructions`, `PreviousResponseID`, `PromptCacheKey`, `PromptCacheRetention`, `ServiceTier`, `Fallbacks`, `ExtraParams`, `RawRequestBody`
- `BifrostCompactionResponse` (`core/schemas/responses.go:72-79`): `ID`, `Object`, `Model`, `CreatedAt`, `Output []ResponsesMessage`, `Usage`, `ExtraFields`
- `BifrostCountTokensResponse` (`core/schemas/count_tokens.go:4-14`): `Object`, `Model`, `InputTokens`, `InputTokensDetails`, `Tokens`, `TokenStrings`, `OutputTokens`, `TotalTokens`, `ExtraFields`

**g0router:**
- `ResponsesRequest` (`internal/schemas/responses.go:4-23`): `Model`, `Input any`, `Include`, `Instructions`, `MaxOutputTokens`, `Metadata`, `ParallelToolCalls`, `PreviousResponseID`, `Reasoning`, `Store`, `Stream`, `Temperature`, `Text`, `ToolChoice`, `Tools`, `TopP`, `Truncation`, `User`
- `ResponsesResponse` (`internal/schemas/responses.go:37-60`): `ID`, `Object`, `CreatedAt`, `Model`, `Output`, `Status`, `Usage`, `Error`, `IncompleteDetails`, etc.
- No `CompactionRequest`, `CompactionResponse`, or `CountTokensResponse` schemas.

### Audio
**Bifrost:**
- `BifrostSpeechRequest` (`core/schemas/speech.go:8-15`): `Provider`, `Model`, `Input *SpeechInput`, `Params *SpeechParameters`, `Fallbacks`, `RawRequestBody`
- `BifrostSpeechResponse` (`core/schemas/speech.go:21-28`): `Audio []byte`, `Usage *SpeechUsage`, `Alignment`, `NormalizedAlignment`, `AudioBase64`, `ExtraFields`
- `BifrostTranscriptionRequest` (`core/schemas/transcriptions.go:3-10`): `Provider`, `Model`, `Input *TranscriptionInput`, `Params *TranscriptionParameters`, `Fallbacks`, `RawRequestBody`
- `BifrostTranscriptionResponse` (`core/schemas/transcriptions.go:16-27`): `Duration`, `Language`, `LogProbs`, `Segments`, `Task`, `Text`, `Usage`, `Words`, `ResponseFormat`, `ExtraFields`

**g0router:**
- `SpeechRequest` (`internal/schemas/audio.go:4-10`): `Model`, `Input`, `Voice`, `ResponseFormat`, `Speed`
- `SpeechResponse` (`internal/schemas/audio.go:13-16`): `Audio []byte`, `ContentType string`
- `TranscriptionRequest` (`internal/schemas/audio.go:18-27`): `File []byte`, `Model`, `Language`, `Prompt`, `ResponseFormat`, `Temperature`, `TimestampGranularities`
- `TranscriptionResponse` (`internal/schemas/audio.go:29-37`): `Text`, `Task`, `Language`, `Duration`, `Words`, `Segments`

### Images
**Bifrost:**
- `BifrostImageGenerationRequest` (`core/schemas/images.go:15-22`): `Provider`, `Model`, `Input *ImageGenerationInput`, `Params *ImageGenerationParameters`, `Fallbacks`, `RawRequestBody`
- `ImageGenerationParameters` (`core/schemas/images.go:33-50`): `N`, `Background`, `Moderation`, `PartialImages`, `Size`, `Quality`, `OutputCompression`, `OutputFormat`, `Style`, `ResponseFormat`, `Seed`, `NegativePrompt`, `NumInferenceSteps`, `User`, `InputImages`, `AspectRatio`, `ExtraParams`

**g0router:**
- `ImageGenerationRequest` (`internal/schemas/images.go:4-13`): `Prompt`, `Model`, `N`, `Quality`, `ResponseFormat`, `Size`, `Style`, `User`
- `ImageGenerationResponse` (`internal/schemas/images.go:15-19`): `Created`, `Data []ImageData`
- `ImageData` (`internal/schemas/images.go:22-26`): `URL`, `B64JSON`, `RevisedPrompt`
- `ImageEditRequest` (`internal/schemas/images.go:28-38`): `Image []byte`, `Mask []byte`, `Prompt`, `Model`, `N`, `Size`, `ResponseFormat`, `User`
- `ImageVariationRequest` (`internal/schemas/images.go:40-48`): `Image []byte`, `Model`, `N`, `Size`, `ResponseFormat`, `User`

### Files
**Bifrost:**
- `FileObject` (`core/schemas/files.go:41-52`): `ID`, `Object`, `Bytes int64`, `CreatedAt`, `UpdatedAt`, `Filename`, `Purpose FilePurpose`, `Status`, `StatusDetails`, `ExpiresAt`
- `BifrostFileUploadRequest` (`core/schemas/files.go:55-73`): `Provider`, `Model`, `File []byte`, `Filename`, `Purpose FilePurpose`, `ContentType`, `StorageConfig`, `ExpiresAfter`, `ExtraParams`
- `FilePurpose` enum (`core/schemas/files.go:5-16`): `batch`, `assistants`, `fine-tune`, `vision`, `batch_output`, `user_data`, `responses`, `evals`
- `FileStatus` enum (`core/schemas/files.go:19-28`): `uploaded`, `processed`, `processing`, `pending_upload`, `error`, `deleted`

**g0router:**
- `FileObject` (`internal/schemas/files.go:4-13`): `ID`, `Object`, `Bytes int`, `CreatedAt`, `Filename`, `Purpose`, `Status`, `StatusDetails`
- `FileUploadRequest` (`internal/schemas/files.go:15-20`): `File []byte`, `Filename string`, `Purpose string`
- `FileListResponse` (`internal/schemas/files.go:22-26`): `Object`, `Data []FileObject`
- `FileDeleteResponse` (`internal/schemas/files.go:28-33`): `ID`, `Object`, `Deleted`
- No `FilePurpose` or `FileStatus` typed enums.

### Batch
**Bifrost:**
- `BatchStatus` enum (`core/schemas/batch.go:5-18`): `validating`, `failed`, `in_progress`, `finalizing`, `completed`, `expired`, `cancelling`, `cancelled`, `ended`, `deleted`
- `BatchEndpoint` enum (`core/schemas/batch.go:21-29`): `/v1/chat/completions`, `/v1/embeddings`, `/v1/completions`, `/v1/responses`, `/v1/messages`
- `BifrostBatchCreateRequest` (`core/schemas/batch.go:66-80`): `Provider`, `Model`, `RawRequestBody`, `InputFileID`, `Requests []BatchRequestItem`, `InputBlob`, `OutputFolder`
- `BatchRequestCounts` (`core/schemas/batch.go:41-49`): `Total`, `Completed`, `Failed`, `Succeeded`, `Expired`, `Canceled`, `Pending`

**g0router:**
- `Batch` (`internal/schemas/batch.go:4-22`): `ID`, `Object`, `Endpoint`, `Errors`, `InputFileID`, `CompletionWindow`, `Status`, `OutputFileID`, `ErrorFileID`, `CreatedAt`, `InProgressAt`, `CompletedAt`, `ExpiredAt`, `CancellingAt`, `CancelledAt`, `RequestCounts`, `Metadata`
- `BatchCreateRequest` (`internal/schemas/batch.go:45-51`): `InputFileID`, `Endpoint`, `CompletionWindow`, `Metadata`
- `BatchListResponse` (`internal/schemas/batch.go:53-57`): `Object`, `Data []Batch`
- `BatchErrors` (`internal/schemas/batch.go:25-28`): `Object`, `Data []BatchError`
- `BatchError` (`internal/schemas/batch.go:30-36`): `Line`, `Message`, `Param`, `Code`
- `BatchRequestCounts` (`internal/schemas/batch.go:38-43`): `Total`, `Completed`, `Failed`
- No typed enums for `BatchStatus` or `BatchEndpoint`. No `Requests []BatchRequestItem` inline batching.

---

## Edge cases and quirks

1. **SSE header timing:** Bifrost calls `getStream()` before setting `Content-Type: text/event-stream` so that provider-setup errors return proper JSON with the correct HTTP status code (`transports/bifrost-http/handlers/inference.go:1652-1666`). g0router sets SSE headers first (`internal/api/chat.go:42-44`), so a provider error after header write produces a malformed response.

2. **fasthttp pipe bypass:** Bifrost uses `lib.NewSSEStreamReader` with `ctx.Response.SetBodyStream(reader, -1)` to bypass fasthttp's `PipeConns` which batches multiple SSE events into one TCP segment (`transports/bifrost-http/handlers/inference.go:1698-1699`). g0router writes each chunk directly to `ctx`, relying on fasthttp default buffering.

3. **Raw response passthrough:** When `resp.ExtraFields.Provider == schemas.OpenAI && resp.ExtraFields.RawResponse != nil`, Bifrost returns the raw upstream bytes directly without re-marshaling (`transports/bifrost-http/integrations/openai.go:474,617,737`). g0router always unmarshals into `StreamChunk` and re-marshals, which can drop unknown fields.

4. **Azure SDK UA detection:** Bifrost detects Azure SDK requests by checking `User-Agent` for `AzureOpenAI` substring (`transports/bifrost-http/integrations/openai.go:56-58`). This gates behavior before body parsing. g0router has no Azure-specific path.

5. **Multipart parsing for image edit/variation/video:** Bifrost uses custom multipart parsers (`parseOpenAIImageEditMultipartRequest`, `parseOpenAIImageVariationMultipartRequest`, `parseOpenAIVideoGenerationMultipartRequest`) that manually iterate form values/files to build request structs and `ExtraParams` (`transports/bifrost-http/handlers/inference.go:1042-1100,3008-3095`). g0router has no multipart handlers.

6. **Batch ID encoding (Bedrock):** Bedrock batch IDs are ARNs; Bifrost base64-encodes them in OpenAI responses and decodes on inbound routes so that path params remain URL-safe (`transports/bifrost-http/integrations/openai.go:1420,1438,1570,1626,1680`). g0router has no batch routes.

7. **Batch ID prefix swap (Gemini):** Gemini uses `batches-` prefix; Bifrost swaps to `batches/` for OpenAI compatibility (`transports/bifrost-http/integrations/openai.go:1415,1435,1565,1623,1678`). g0router has no batch routes.

8. **File ID prefix swap (Gemini):** Similar to batch IDs, Gemini file IDs use `files-`; Bifrost swaps to `files/` (`transports/bifrost-http/integrations/openai.go:1729,1790,1829,1893`). g0router has no file routes.

9. **Container raw passthrough:** All container ops force `SendBackRawRequest`, `SendBackRawResponse`, `StoreRawRequestResponse` to `true` (`transports/bifrost-http/handlers/inference.go:608`). g0router has no container endpoints.

10. **Async stream rejection:** Async mirrors reject `stream=true` with HTTP 400 (`transports/bifrost-http/handlers/asyncinference.go:109,147,183`). g0router has no async subsystem.

11. **WebSocket store override:** WebSocket Responses API forces `store=true` unless provider config disables it (`transports/bifrost-http/handlers/wsresponses.go:180-191`). g0router has no WebSocket handler.

12. **Large payload metadata hydration:** When large-payload mode is active, Bifrost back-fills `model` and `stream` from metadata because the body is not parsed (`transports/bifrost-http/integrations/openai.go:60-100`). g0router has no large-payload mode.

13. **Error `Param` type:** Bifrost `ErrorField.Param` is `interface{}` (`core/schemas/bifrost.go:1781`). g0router `APIError.Param` is `*string` (`internal/schemas/errors.go:7`), which cannot represent non-string params.

14. **Error `EventID`:** Bifrost errors include `event_id` for tracing (`core/schemas/bifrost.go:1688`). g0router errors have no event ID field.

15. **g0router `ModelsHandler.Get` does not filter:** `GET /v1/models/{id}` delegates to `List` and returns the full unfiltered list (`internal/api/models.go:51-53`). The route is registered but the behavior is broken for the intended use case.

---

## Go-port considerations

1. Bifrost's `AllowedRequests` capability matrix (`provider.go:317-374`) and `ProviderFeatureSupport` (`anthropic/types.go:104-139`) are large typed structs. A g0router port could start with a smaller boolean map.
2. The SSE streaming reader abstraction (`lib.NewSSEStreamReader`) is worth porting to fix the header-timing bug in `internal/api/chat.go`.
3. Normalization logic (field stripping, ID encoding) lives in transport-layer converters. If g0router adds multi-provider support, move this into provider-specific request builders rather than the HTTP handler.
4. The `BifrostError` envelope with `IsBifrostError` discriminator supports richer fallback control. g0router's flat `ErrorResponse` is simpler but lacks this.
5. Bifrost uses `RawRequestBody []byte` passthrough on every request struct. g0router could add this to bypass JSON round-tripping for provider-native forwarding.
6. Multipart parsing for images/audio/video is non-trivial and should be implemented per-endpoint with explicit form field whitelists.
7. Container, video, and async endpoints are Bifrost-specific extensions. A minimal g0router port can defer these.
