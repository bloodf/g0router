# Bifrost Core Architecture Parity Matrix

Reference: `/Users/heitor/Developer/github.com/bloodf/_refs/bifrost` @ `ca21298`
Target: `/Users/heitor/Developer/github.com/bloodf/g0router`

## Behavior Matrix

| ID | Behavior | Evidence (file:line) | g0router status | Notes |
|---|---|---|---|---|
| PAR-BF-CORE-001 | Provider interface defines 30+ methods covering chat, completion, embedding, image, video, audio, file, batch, cached content, container, and passthrough APIs | `core/schemas/provider.go:590-707` | PARTIAL | `internal/schemas/provider.go:68` has 16 methods; missing Rerank, OCR, Video*, Container*, CachedContent*, Passthrough, PassthroughStream, Compaction |
| PAR-BF-CORE-002 | Provider interface accepts `postHookRunner PostHookRunner` and `postHookSpanFinalizer func(context.Context)` on every streaming method | `core/schemas/provider.go:598-606` | HAVE | `internal/schemas/provider.go:75` carries `postHookRunner PostHookRunner` on stream methods; no `postHookSpanFinalizer` equivalent |
| PAR-BF-CORE-003 | Optional `WebSocketCapableProvider` interface for native WS upstream | `core/schemas/provider.go:714-721` | MISSING | No WS-capable provider abstraction in g0router |
| PAR-BF-CORE-004 | `BasePlugin` interface with `GetName()` and `Cleanup()` | `core/schemas/plugin.go:208-215` | MISSING | No plugin system in g0router |
| PAR-BF-CORE-005 | `HTTPTransportPlugin` interface: PreHook, PostHook, StreamChunkHook | `core/schemas/plugin.go:218-265` | MISSING | g0router has only basic fasthttp middleware (`internal/server/middleware.go:10-18`) |
| PAR-BF-CORE-006 | `LLMPlugin` interface: PreRequestHook, PreLLMHook, PostLLMHook | `core/schemas/plugin.go:267-288` | MISSING | No LLM hook pipeline |
| PAR-BF-CORE-007 | `MCPPlugin` interface: PreMCPHook, PostMCPHook | `core/schemas/plugin.go:290-295` | MISSING | No MCP hook pipeline |
| PAR-BF-CORE-008 | `MCPConnectionPlugin` extension for Connect events | `core/schemas/plugin.go:297-319` | MISSING | No MCP connection hooks |
| PAR-BF-CORE-009 | `ObservabilityPlugin` interface for async trace injection | `core/schemas/plugin.go:378-411` | MISSING | No observability plugin interface |
| PAR-BF-CORE-010 | `ConfigMarshallerPlugin` interface for custom config serialization | `core/schemas/plugin.go:360-376` | MISSING | No plugin config marshalling |
| PAR-BF-CORE-011 | Plugin execution order: HTTPTransportPreHook → PreRequestHook → PreLLMHook → provider → PostLLMHook → HTTPTransportPostHook | `core/schemas/plugin.go:169-179` | MISSING | No ordered hook pipeline |
| PAR-BF-CORE-012 | Per-request vs per-attempt hook semantics documented and enforced | `core/schemas/plugin.go:181-188` | MISSING | g0router inference path has no plugin interception |
| PAR-BF-CORE-013 | `LLMPluginShortCircuit` and `MCPPluginShortCircuit` types for early return | `core/schemas/bifrost.go:1683-1700` | MISSING | No short-circuit mechanism |
| PAR-BF-CORE-014 | `PluginPlacement` constants (pre_builtin, post_builtin, builtin) for ordering | `core/schemas/plugin.go:338-346` | MISSING | No plugin placement system |
| PAR-BF-CORE-015 | `PluginPipeline` struct with pre/post hook error aggregation and streaming timing accumulators | `core/bifrost.go:164-189` | MISSING | No pipeline abstraction |
| PAR-BF-CORE-016 | Plugin pipeline object pooling via `sync.Pool` | `core/bifrost.go:281-302` | MISSING | No plugin pipeline pooling |
| PAR-BF-CORE-017 | `RunPreRequestHooks` runs once per top-level request, mutates provider/model/fallbacks | `core/bifrost.go:4179-4195` | MISSING | No PreRequestHook phase |
| PAR-BF-CORE-018 | `RunLLMPreHooks` / `RunPostLLMHooks` run once per provider attempt (primary + fallbacks) | `core/bifrost.go:4239-4304` | MISSING | No per-attempt hooks |
| PAR-BF-CORE-019 | Fallback chain with `shouldTryFallbacks` and `prepareFallbackRequest` | `core/bifrost.go:4487-4540` | MISSING | `internal/inference/router.go:34` has TODO for fallbacks |
| PAR-BF-CORE-020 | `AllowFallbacks` field on `BifrostError` controls fallback behavior | `core/schemas/bifrost.go:1683-1699` | MISSING | `internal/schemas/errors.go:26` has no `AllowFallbacks` field |
| PAR-BF-CORE-021 | Channel-based async request routing with `ProviderQueue` | `core/bifrost.go:54-60` | MISSING | g0router uses direct synchronous calls |
| PAR-BF-CORE-022 | `ProviderQueue` lifecycle: atomic closing flag + `sync.Once` + never-close-channel pattern | `core/bifrost.go:126-156` | MISSING | No queue-based provider lifecycle |
| PAR-BF-CORE-023 | Object pooling: ChannelMessage, response/error/stream channels, BifrostRequest, PluginPipeline | `core/bifrost.go:77-82` | MISSING | No object pooling in g0router core |
| PAR-BF-CORE-024 | `dropExcessRequests` atomic bool to drop requests when queue full | `core/bifrost.go:88` | MISSING | No queue-dropping mechanism |
| PAR-BF-CORE-025 | `KeySelector` function type for custom API key selection | `core/schemas/bifrost.go:15` | MISSING | g0router returns hardcoded keys (`internal/inference/router.go:38-52`) |
| PAR-BF-CORE-026 | `KVStore` interface for clustering/session stickiness | `core/schemas/bifrost.go:32` | MISSING | No KV store abstraction |
| PAR-BF-CORE-027 | Request-type constants: 20+ distinct request types | `core/schemas/bifrost.go:686-706` | PARTIAL | g0router handles fewer request categories |
| PAR-BF-CORE-028 | `BifrostContext` with typed key-value storage and request metadata | `core/schemas/bifrost.go:280-286` | PARTIAL | `internal/schemas/provider.go:25` has minimal `GatewayContext` with only `RequestID` |
| PAR-BF-CORE-029 | Routing engine with CEL expression evaluation and scope precedence | `plugins/governance/routing.go:48-74` | MISSING | `internal/inference/router.go:14` is a hardcoded prefix router |
| PAR-BF-CORE-030 | Routing rule chain evaluation with max depth (default 10) and cycle detection | `plugins/governance/routing.go:84-134` | MISSING | No rule engine |
| PAR-BF-CORE-031 | Adaptive load balancing: error penalty (50%), latency score (20%), utilization (5%), momentum bias | `docs/enterprise/adaptive-load-balancing.mdx:83-100` | MISSING | No adaptive scoring |
| PAR-BF-CORE-032 | Route health states: Healthy, Degraded, Failed, Recovering with automatic transitions | `docs/enterprise/adaptive-load-balancing.mdx:136` | MISSING | No health state machine |
| PAR-BF-CORE-033 | Weighted random key selection with 5% jitter and 25% exploration probability | `docs/enterprise/adaptive-load-balancing.mdx:148` | MISSING | No weighted selection |
| PAR-BF-CORE-034 | Semantic cache plugin with dual-path lookup: direct hash + semantic similarity | `plugins/semanticcache/main.go:135-164` | MISSING | No semantic cache |
| PAR-BF-CORE-035 | Semantic cache config: Provider, EmbeddingModel, TTL, Threshold, Dimension | `plugins/semanticcache/main.go:28-45` | MISSING | No cache config |
| PAR-BF-CORE-036 | Streaming response accumulation with background reaper and TTL bookkeeping | `plugins/semanticcache/main.go:99-127` | MISSING | No stream accumulation |
| PAR-BF-CORE-037 | `VectorStore` abstraction with Ping, CreateNamespace, GetNearest, Add, Delete | `framework/vectorstore/store.go:82-109` | MISSING | No vector store interface |
| PAR-BF-CORE-038 | Vector store backends: Weaviate, Redis, Qdrant, Pinecone | `framework/vectorstore/store.go:14-19` | MISSING | No vector backends |
| PAR-BF-CORE-039 | Cluster mode: memberlist gossip (port 10101) + gRPC sync (port 10102) | `docs/enterprise/clustering.mdx:50-55` | MISSING | No cluster mode |
| PAR-BF-CORE-040 | Cluster service discovery: K8s, Consul, etcd, DNS, UDP, mDNS | `docs/enterprise/clustering.mdx:28` | MISSING | No discovery |
| PAR-BF-CORE-041 | Cluster leader election: cluster-wide + per-region, deterministic lexicographic | `docs/enterprise/clustering.mdx:74-83` | MISSING | No leader election |
| PAR-BF-CORE-042 | Cluster replication: 30+ entity types with dedup (5-min TTL) | `docs/enterprise/clustering.mdx:57-64` | MISSING | No entity replication |
| PAR-BF-CORE-043 | OTEL plugin with multi-profile support (HTTP/gRPC protocols) | `plugins/otel/main.go:57-117` | MISSING | No OTEL integration |
| PAR-BF-CORE-044 | OTEL metrics: counters, histograms with custom bucket boundaries | `plugins/otel/metrics.go:38-66` | MISSING | No metrics exporter |
| PAR-BF-CORE-045 | Metrics include upstream latency, TTFT, inter-token latency, request retries, cost | `plugins/otel/metrics.go:56-61` | MISSING | No latency histograms |
| PAR-BF-CORE-046 | `Tracer` interface with StartSpan, EndSpan, SetAttribute, trace store | `framework/tracing/tracer.go:17-41` | MISSING | No tracing interface |
| PAR-BF-CORE-047 | Trace streaming accumulator with chunk aggregation | `framework/tracing/tracer.go:21` | MISSING | No trace accumulation |
| PAR-BF-CORE-048 | Request header pattern capture for observability plugins | `framework/tracing/tracer.go:44-69` | MISSING | No header capture |
| PAR-BF-CORE-049 | `HTTPRequest` / `HTTPResponse` pooled types for transport plugins | `core/schemas/plugin.go:41-163` | MISSING | No pooled HTTP transport types |
| PAR-BF-CORE-050 | Case-insensitive header/query/path-param lookup helpers | `core/schemas/plugin.go:51-87` | MISSING | No case-insensitive helpers |

## Data Models

### Bifrost Provider Interface
```go
type Provider interface {
    GetProviderKey() ModelProvider
    ListModels(ctx *BifrostContext, keys []Key, request *BifrostListModelsRequest) (*BifrostListModelsResponse, *BifrostError)
    TextCompletion(ctx *BifrostContext, key Key, request *BifrostTextCompletionRequest) (*BifrostTextCompletionResponse, *BifrostError)
    TextCompletionStream(ctx *BifrostContext, postHookRunner PostHookRunner, postHookSpanFinalizer func(context.Context), key Key, request *BifrostTextCompletionRequest) (chan *BifrostStreamChunk, *BifrostError)
    ChatCompletion(ctx *BifrostContext, key Key, request *BifrostChatRequest) (*BifrostChatResponse, *BifrostError)
    ChatCompletionStream(ctx *BifrostContext, postHookRunner PostHookRunner, postHookSpanFinalizer func(context.Context), key Key, request *BifrostChatRequest) (chan *BifrostStreamChunk, *BifrostError)
    Responses(ctx *BifrostContext, key Key, request *BifrostResponsesRequest) (*BifrostResponsesResponse, *BifrostError)
    ResponsesStream(ctx *BifrostContext, postHookRunner PostHookRunner, postHookSpanFinalizer func(context.Context), key Key, request *BifrostResponsesRequest) (chan *BifrostStreamChunk, *BifrostError)
    CountTokens(ctx *BifrostContext, key Key, request *BifrostResponsesRequest) (*BifrostCountTokensResponse, *BifrostError)
    Compaction(ctx *BifrostContext, key Key, request *BifrostCompactionRequest) (*BifrostCompactionResponse, *BifrostError)
    Embedding(ctx *BifrostContext, key Key, request *BifrostEmbeddingRequest) (*BifrostEmbeddingResponse, *BifrostError)
    Rerank(ctx *BifrostContext, key Key, request *BifrostRerankRequest) (*BifrostRerankResponse, *BifrostError)
    OCR(ctx *BifrostContext, key Key, request *BifrostOCRRequest) (*BifrostOCRResponse, *BifrostError)
    Speech(ctx *BifrostContext, key Key, request *BifrostSpeechRequest) (*BifrostSpeechResponse, *BifrostError)
    SpeechStream(ctx *BifrostContext, postHookRunner PostHookRunner, postHookSpanFinalizer func(context.Context), key Key, request *BifrostSpeechRequest) (chan *BifrostStreamChunk, *BifrostError)
    Transcription(ctx *BifrostContext, key Key, request *BifrostTranscriptionRequest) (*BifrostTranscriptionResponse, *BifrostError)
    TranscriptionStream(ctx *BifrostContext, postHookRunner PostHookRunner, postHookSpanFinalizer func(context.Context), key Key, request *BifrostTranscriptionRequest) (chan *BifrostStreamChunk, *BifrostError)
    ImageGeneration(ctx *BifrostContext, key Key, request *BifrostImageGenerationRequest) (*BifrostImageGenerationResponse, *BifrostError)
    ImageGenerationStream(ctx *BifrostContext, postHookRunner PostHookRunner, postHookSpanFinalizer func(context.Context), key Key, request *BifrostImageGenerationRequest) (chan *BifrostStreamChunk, *BifrostError)
    ImageEdit(ctx *BifrostContext, key Key, request *BifrostImageEditRequest) (*BifrostImageGenerationResponse, *BifrostError)
    ImageEditStream(ctx *BifrostContext, postHookRunner PostHookRunner, postHookSpanFinalizer func(context.Context), key Key, request *BifrostImageEditRequest) (chan *BifrostStreamChunk, *BifrostError)
    ImageVariation(ctx *BifrostContext, key Key, request *BifrostImageVariationRequest) (*BifrostImageGenerationResponse, *BifrostError)
    VideoGeneration(ctx *BifrostContext, key Key, request *BifrostVideoGenerationRequest) (*BifrostVideoGenerationResponse, *BifrostError)
    VideoRetrieve(ctx *BifrostContext, key Key, request *BifrostVideoRetrieveRequest) (*BifrostVideoGenerationResponse, *BifrostError)
    VideoDownload(ctx *BifrostContext, key Key, request *BifrostVideoDownloadRequest) (*BifrostVideoDownloadResponse, *BifrostError)
    VideoDelete(ctx *BifrostContext, key Key, request *BifrostVideoDeleteRequest) (*BifrostVideoDeleteResponse, *BifrostError)
    VideoList(ctx *BifrostContext, key Key, request *BifrostVideoListRequest) (*BifrostVideoListResponse, *BifrostError)
    VideoRemix(ctx *BifrostContext, key Key, request *BifrostVideoRemixRequest) (*BifrostVideoGenerationResponse, *BifrostError)
    BatchCreate(ctx *BifrostContext, key Key, request *BifrostBatchCreateRequest) (*BifrostBatchCreateResponse, *BifrostError)
    BatchList(ctx *BifrostContext, keys []Key, request *BifrostBatchListRequest) (*BifrostBatchListResponse, *BifrostError)
    BatchRetrieve(ctx *BifrostContext, keys []Key, request *BifrostBatchRetrieveRequest) (*BifrostBatchRetrieveResponse, *BifrostError)
    BatchCancel(ctx *BifrostContext, keys []Key, request *BifrostBatchCancelRequest) (*BifrostBatchCancelResponse, *BifrostError)
    BatchDelete(ctx *BifrostContext, keys []Key, request *BifrostBatchDeleteRequest) (*BifrostBatchDeleteResponse, *BifrostError)
    BatchResults(ctx *BifrostContext, keys []Key, request *BifrostBatchResultsRequest) (*BifrostBatchResultsResponse, *BifrostError)
    FileUpload(ctx *BifrostContext, key Key, request *BifrostFileUploadRequest) (*BifrostFileUploadResponse, *BifrostError)
    FileList(ctx *BifrostContext, keys []Key, request *BifrostFileListRequest) (*BifrostFileListResponse, *BifrostError)
    FileRetrieve(ctx *BifrostContext, keys []Key, request *BifrostFileRetrieveRequest) (*BifrostFileRetrieveResponse, *BifrostError)
    FileDelete(ctx *BifrostContext, keys []Key, request *BifrostFileDeleteRequest) (*BifrostFileDeleteResponse, *BifrostError)
    FileContent(ctx *BifrostContext, keys []Key, request *BifrostFileContentRequest) (*BifrostFileContentResponse, *BifrostError)
    CachedContentCreate(ctx *BifrostContext, key Key, request *BifrostCachedContentCreateRequest) (*BifrostCachedContentCreateResponse, *BifrostError)
    CachedContentList(ctx *BifrostContext, keys []Key, request *BifrostCachedContentListRequest) (*BifrostCachedContentListResponse, *BifrostError)
    CachedContentRetrieve(ctx *BifrostContext, keys []Key, request *BifrostCachedContentRetrieveRequest) (*BifrostCachedContentRetrieveResponse, *BifrostError)
    CachedContentUpdate(ctx *BifrostContext, keys []Key, request *BifrostCachedContentUpdateRequest) (*BifrostCachedContentUpdateResponse, *BifrostError)
    CachedContentDelete(ctx *BifrostContext, keys []Key, request *BifrostCachedContentDeleteRequest) (*BifrostCachedContentDeleteResponse, *BifrostError)
    ContainerCreate(ctx *BifrostContext, key Key, request *BifrostContainerCreateRequest) (*BifrostContainerCreateResponse, *BifrostError)
    ContainerList(ctx *BifrostContext, keys []Key, request *BifrostContainerListRequest) (*BifrostContainerListResponse, *BifrostError)
    ContainerRetrieve(ctx *BifrostContext, keys []Key, request *BifrostContainerRetrieveRequest) (*BifrostContainerRetrieveResponse, *BifrostError)
    ContainerDelete(ctx *BifrostContext, keys []Key, request *BifrostContainerDeleteRequest) (*BifrostContainerDeleteResponse, *BifrostError)
    ContainerFileCreate(ctx *BifrostContext, key Key, request *BifrostContainerFileCreateRequest) (*BifrostContainerFileCreateResponse, *BifrostError)
    ContainerFileList(ctx *BifrostContext, keys []Key, request *BifrostContainerFileListRequest) (*BifrostContainerFileListResponse, *BifrostError)
    ContainerFileRetrieve(ctx *BifrostContext, keys []Key, request *BifrostContainerFileRetrieveRequest) (*BifrostContainerFileRetrieveResponse, *BifrostError)
    ContainerFileContent(ctx *BifrostContext, keys []Key, request *BifrostContainerFileContentRequest) (*BifrostContainerFileContentResponse, *BifrostError)
    ContainerFileDelete(ctx *BifrostContext, keys []Key, request *BifrostContainerFileDeleteRequest) (*BifrostContainerFileDeleteResponse, *BifrostError)
    Passthrough(ctx *BifrostContext, key Key, req *BifrostPassthroughRequest) (*BifrostPassthroughResponse, *BifrostError)
    PassthroughStream(ctx *BifrostContext, postHookRunner PostHookRunner, postHookSpanFinalizer func(context.Context), key Key, req *BifrostPassthroughRequest) (chan *BifrostStreamChunk, *BifrostError)
}
```

### Bifrost Plugin Interfaces
```go
type BasePlugin interface {
    GetName() string
    Cleanup() error
}

type HTTPTransportPlugin interface {
    BasePlugin
    HTTPTransportPreHook(ctx *BifrostContext, req *HTTPRequest) (*HTTPResponse, error)
    HTTPTransportPostHook(ctx *BifrostContext, req *HTTPRequest, resp *HTTPResponse) error
    HTTPTransportStreamChunkHook(ctx *BifrostContext, req *HTTPRequest, chunk *BifrostStreamChunk) (*BifrostStreamChunk, error)
}

type LLMPlugin interface {
    BasePlugin
    PreRequestHook(ctx *BifrostContext, req *BifrostRequest) error
    PreLLMHook(ctx *BifrostContext, req *BifrostRequest) (*BifrostRequest, *LLMPluginShortCircuit, error)
    PostLLMHook(ctx *BifrostContext, resp *BifrostResponse, bifrostErr *BifrostError) (*BifrostResponse, *BifrostError, error)
}

type MCPPlugin interface {
    BasePlugin
    PreMCPHook(ctx *BifrostContext, req *BifrostMCPRequest) (*BifrostMCPRequest, *MCPPluginShortCircuit, error)
    PostMCPHook(ctx *BifrostContext, resp *BifrostMCPResponse, bifrostErr *BifrostError) (*BifrostMCPResponse, *BifrostError, error)
}

type ObservabilityPlugin interface {
    BasePlugin
    Inject(ctx context.Context, trace *Trace) error
}
```

### Bifrost VectorStore Interface
```go
type VectorStore interface {
    Ping(ctx context.Context) error
    CreateNamespace(ctx context.Context, namespace string, dimension int, properties map[string]VectorStoreProperties) error
    DeleteNamespace(ctx context.Context, namespace string) error
    GetChunk(ctx context.Context, namespace string, id string) (SearchResult, error)
    GetChunks(ctx context.Context, namespace string, ids []string) ([]SearchResult, error)
    GetAll(ctx context.Context, namespace string, queries []Query, selectFields []string, cursor *string, limit int64) ([]SearchResult, *string, error)
    GetNearest(ctx context.Context, namespace string, vector []float32, queries []Query, selectFields []string, threshold float64, limit int64) ([]SearchResult, error)
    RequiresVectors() bool
    Add(ctx context.Context, namespace string, id string, embedding []float32, metadata map[string]interface{}) error
    Delete(ctx context.Context, namespace string, id string) error
    DeleteAll(ctx context.Context, namespace string, queries []Query) ([]DeleteResult, error)
    Close(ctx context.Context, namespace string) error
}
```

### g0router Provider Interface
```go
type Provider interface {
    GetProvider() ModelProvider
    SetNetworkConfig(config NetworkConfig)
    ListModels(ctx *GatewayContext, key Key) (*ListModelsResponse, *ProviderError)
    ChatCompletion(ctx *GatewayContext, key Key, request *ChatRequest) (*ChatResponse, *ProviderError)
    ChatCompletionStream(ctx *GatewayContext, postHookRunner PostHookRunner, key Key, request *ChatRequest) (chan *StreamChunk, *ProviderError)
    TextCompletion(ctx *GatewayContext, key Key, request *TextCompletionRequest) (*TextCompletionResponse, *ProviderError)
    TextCompletionStream(ctx *GatewayContext, postHookRunner PostHookRunner, key Key, request *TextCompletionRequest) (chan *StreamChunk, *ProviderError)
    Responses(ctx *GatewayContext, key Key, request *ResponsesRequest) (*ResponsesResponse, *ProviderError)
    ResponsesStream(ctx *GatewayContext, postHookRunner PostHookRunner, key Key, request *ResponsesRequest) (chan *StreamChunk, *ProviderError)
    Embedding(ctx *GatewayContext, key Key, request *EmbeddingRequest) (*EmbeddingResponse, *ProviderError)
    ImageGeneration(ctx *GatewayContext, key Key, request *ImageGenerationRequest) (*ImageGenerationResponse, *ProviderError)
    ImageGenerationStream(ctx *GatewayContext, postHookRunner PostHookRunner, key Key, request *ImageGenerationRequest) (chan *StreamChunk, *ProviderError)
    ImageEdit(ctx *GatewayContext, key Key, request *ImageEditRequest) (*ImageGenerationResponse, *ProviderError)
    ImageVariation(ctx *GatewayContext, key Key, request *ImageVariationRequest) (*ImageGenerationResponse, *ProviderError)
    Speech(ctx *GatewayContext, key Key, request *SpeechRequest) (*SpeechResponse, *ProviderError)
    SpeechStream(ctx *GatewayContext, postHookRunner PostHookRunner, key Key, request *SpeechRequest) (chan *StreamChunk, *ProviderError)
    Transcription(ctx *GatewayContext, key Key, request *TranscriptionRequest) (*TranscriptionResponse, *ProviderError)
    TranscriptionStream(ctx *GatewayContext, postHookRunner PostHookRunner, key Key, request *TranscriptionRequest) (chan *StreamChunk, *ProviderError)
    FileUpload(ctx *GatewayContext, key Key, request *FileUploadRequest) (*FileObject, *ProviderError)
    FileList(ctx *GatewayContext, key Key) (*FileListResponse, *ProviderError)
    FileRetrieve(ctx *GatewayContext, key Key, fileID string) (*FileObject, *ProviderError)
    FileDelete(ctx *GatewayContext, key Key, fileID string) (*FileDeleteResponse, *ProviderError)
    FileContent(ctx *GatewayContext, key Key, fileID string) ([]byte, *ProviderError)
    BatchCreate(ctx *GatewayContext, key Key, request *BatchCreateRequest) (*Batch, *ProviderError)
    BatchList(ctx *GatewayContext, key Key) (*BatchListResponse, *ProviderError)
    BatchRetrieve(ctx *GatewayContext, key Key, batchID string) (*Batch, *ProviderError)
    BatchCancel(ctx *GatewayContext, key Key, batchID string) (*Batch, *ProviderError)
    CountTokens(ctx *GatewayContext, key Key, request *ChatRequest) (*TokenCountResponse, *ProviderError)
}
```

## Edge Cases and Quirks

1. **ProviderQueue never closes its channel**: `core/bifrost.go:126` documents that `queue` is never closed to avoid "send on closed channel" panics. Shutdown signals via `done` channel closed by `signalOnce.Do`.

2. **Plugin short-circuit symmetry**: For every `PreLLMHook` executed, the corresponding `PostLLMHook` runs in reverse order, even on short-circuit paths (`core/schemas/plugin.go:197`).

3. **PreRequestHook errors are non-blocking**: A non-nil error from `PreRequestHook` is logged as a warning and the pipeline continues; plugins cannot abort at this phase (`core/schemas/plugin.go:277-280`).

4. **Fallback shallow-copy visibility**: Mutations a `PreLLMHook` makes carry to later fallbacks only where `prepareFallbackRequest` happens to share pointers; `PreRequestHook` mutations are the explicit committed phase (`core/schemas/plugin.go:185-188`).

5. **Semantic cache direct-only mode**: Set `Provider=""` and `Dimension=1` to disable semantic search entirely; lookup goes through deterministic hash path only (`plugins/semanticcache/main.go:25-27`).

6. **OTEL profile defaults**: `Insecure` defaults to `true` when omitted so `http://` collectors work out-of-the-box (`plugins/otel/main.go:106-109`).

7. **Routing rule cycle prevention**: Visited rule IDs are tracked per chain to prevent a rule from matching more than once; self-looping rules fire once then yield to subsequent rules (`plugins/governance/routing.go:98`).

8. **Trace pool reuse race**: `ObservabilityPlugin.Inject` must not retain the `*Trace` pointer after returning; the caller releases it to a `sync.Pool` immediately (`core/schemas/plugin.go:406-410`).

## Go-Port Considerations

1. g0router's `Provider` interface is close but omits 14 method groups; expand incrementally.
2. Plugin system is the largest gap; design `LLMPlugin` and `HTTPTransportPlugin` interfaces first, then wire a `PluginPipeline` with `sync.Pool`.
3. Channel-based `ProviderQueue` can be ported directly; the atomic-closing pattern is self-contained.
4. Vector store abstraction is clean; port `VectorStore` interface before semantic cache plugin.
5. Clustering relies on memberlist (HashiCorp) and gRPC; both have Go libraries. Start with gossip membership, then add entity replication.
