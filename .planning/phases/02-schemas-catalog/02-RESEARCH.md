# Phase 2: Schemas + Catalog — Research

**Researched:** 2026-06-09
**Status:** Complete

---

## 1. Schema Type Design

### Reference: BiFrost Go Types
BiFrost uses explicit wire-format structs in `core/schemas/` with ~41 files. For g0router we consolidate into a smaller surface because we control both producer and consumer (no external SDK consumers of these types — they are consumed by our own handlers and providers).

### Decisions
- Use `json:"field_name,omitempty"` tags on all struct fields for snake_case wire format per AGENTS.md convention.
- Use pointer types for optional fields (`*string`, `*int`, `*float64`) so `omitempty` works correctly.
- Define a single `Usage` struct reused across chat, embeddings, and responses.
- Define streaming chunk types once and reuse across all streaming-capable endpoints.
- Keep MCP types minimal in this phase — only structs needed by the Provider interface (none yet; MCP arrives in Phase 14).

### OpenAI API Spec Coverage
The following OpenAI endpoints need request/response types:
| Endpoint | Request | Response | Streaming |
|----------|---------|----------|-----------|
| POST /v1/chat/completions | ChatRequest | ChatResponse | ChatCompletionStream |
| POST /v1/completions | TextCompletionRequest | TextCompletionResponse | TextCompletionStream |
| POST /v1/embeddings | EmbeddingRequest | EmbeddingResponse | — |
| POST /v1/images/generations | ImageGenerationRequest | ImageGenerationResponse | ImageGenerationStream |
| POST /v1/audio/speech | SpeechRequest | SpeechResponse | SpeechStream |
| POST /v1/audio/transcriptions | TranscriptionRequest | TranscriptionResponse | TranscriptionStream |
| POST /v1/files | FileUploadRequest | FileObject | — |
| GET /v1/files | — | FileListResponse | — |
| GET /v1/files/:id | — | FileObject | — |
| DELETE /v1/files/:id | — | FileDeleteResponse | — |
| POST /v1/batches | BatchCreateRequest | Batch | — |
| GET /v1/batches | — | BatchListResponse | — |
| POST /v1/responses | ResponsesRequest | ResponsesResponse | ResponsesStream |

### Error Schema
Per OPENAI-11 and the design doc:
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
Internal provider errors carry metadata:
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

## 2. Provider Interface

### Design
Adopt BiFrost's explicit capability interface with Go idioms:
- One interface per provider package implements all methods.
- Unsupported operations return a typed `*ProviderError` with `Type: "not_supported"`.
- Streaming methods return `chan *StreamChunk` + `*ProviderError`.
- `GatewayContext` carries request-scoped values (request ID, virtual key, etc.).
- `PostHookRunner` is an interface passed to streaming methods for accumulator hooks.

### Key Types
```go
type Provider interface {
    GetProvider() ModelProvider
    SetNetworkConfig(config NetworkConfig)

    ListModels(ctx *GatewayContext, key Key) (*ListModelsResponse, *ProviderError)

    ChatCompletion(ctx *GatewayContext, key Key, request *ChatRequest) (*ChatResponse, *ProviderError)
    ChatCompletionStream(ctx *GatewayContext, postHookRunner PostHookRunner, key Key, request *ChatRequest) (chan *StreamChunk, *ProviderError)

    // ... other capabilities
}
```

`ModelProvider` is a string enum type (`openai`, `anthropic`, `gemini`, etc.).

---

## 3. Catalog Architecture

### In-Memory Registry
The catalog is an in-memory, thread-safe registry loaded at startup. Two layers:
1. **Seed data** (built-in JSON) — loaded first, guarantees offline operation.
2. **Upstream sync** (optional) — updates pricing when network is available.

### Data Model
```go
type PricingEntry struct {
    Provider        string
    Model           string
    Mode            RequestType
    InputPrice      float64 // per 1M tokens
    OutputPrice     float64 // per 1M tokens
    ImagePrice      float64 // per image (if applicable)
    AudioInputPrice float64 // per 1M tokens (if applicable)
    AudioOutputPrice float64 // per 1M tokens (if applicable)
    TieredPricing   []Tier  // e.g., >128k token rate
}

type ModelCapability struct {
    ContextWindow   int
    Modalities      []string // ["text", "image", "audio"]
    Tools           bool
    Reasoning       bool
    NoTemperature   bool     // for gpt-5.4 style models
}
```

### Lookup Fallback Chain
Per CATALOG-03:
1. Exact match: `model|provider|mode`
2. Provider alias: `gemini` → `vertex`
3. Bedrock prefix: `anthropic.claude-3-5-sonnet` → strip `anthropic.` prefix
4. Vertex prefix strip: `publishers/anthropic/models/claude-3-5-sonnet` → normalize
5. Responses → Chat mode fallback

### Cross-Provider Resolution
Per CATALOG-04:
- `GetProvidersForModel("claude-3-5-sonnet")` returns `[anthropic, vertex, bedrock, openrouter]`
- Driven by seed data entries, not hardcoded logic.

### Allowlist Logic
Per CATALOG-05:
- `allowed_models: ["*"]` → catalog-validated allow-all (model must exist in catalog)
- Explicit list → only those models
- Empty list `[]` → deny-all

### Cost Calculation
Per CATALOG-06:
- Token-based: `(inputTokens * inputPrice + outputTokens * outputPrice) / 1_000_000`
- Image-based: `imageCount * imagePrice`
- Audio-based: `(audioInputTokens * audioInputPrice + audioOutputTokens * audioOutputPrice) / 1_000_000`
- Tiered: if `totalTokens > tier.Threshold`, apply tier rate for tokens above threshold.

### Custom Pricing Overrides
Per CATALOG-07:
- Store overrides in SQLite (schema defined in Phase 6, interface stub in Phase 2).
- Layer: `override_price` replaces catalog price when present.

---

## 4. Seed Data Strategy

### Format
A single `seed.json` file embedded with `//go:embed` containing an array of model entries:
```json
[
  {
    "provider": "openai",
    "model": "gpt-4o",
    "modes": ["chat"],
    "capabilities": {
      "context_window": 128000,
      "modalities": ["text", "image"],
      "tools": true,
      "reasoning": false
    },
    "pricing": {
      "input_price": 2.50,
      "output_price": 10.00
    }
  }
]
```

### Coverage
Seed should include top models from: OpenAI, Anthropic, Gemini, Groq, Mistral, Cohere, DeepSeek, MiniMax, Fireworks, Together, Ollama, Bedrock, Vertex. Keep it to ~30-40 entries (enough for routing tests, not exhaustive — upstream sync fills gaps).

---

## 5. Background Sync

### Design
- `Catalog.Sync(ctx)` fetches upstream pricing JSON.
- HTTP client with 10s timeout.
- On failure: log warning, keep existing data.
- On success: atomic swap of in-memory pricing map.
- Refresh interval: 24 hours.
- Triggered at startup (non-blocking) and via background goroutine.

### Upstream Source
Use a public pricing JSON URL (e.g., OpenRouter pricing API or a static GitHub-hosted JSON). The URL is configurable via environment variable with a sensible default.

---

## 6. Testing Strategy

### Schema Tests
- Compile-time verification: `go test ./internal/schemas/...` passes.
- JSON round-trip tests for each type: marshal → unmarshal → compare.

### Catalog Tests (TDD)
- **Plan 04 (Lookup):** Table-driven tests for exact match, fallback chain, cross-provider resolution.
- **Plan 05 (Cost):** Fixture usage data → expected cost assertions.
- Use fakes for upstream sync (no mocks per AGENTS.md).

---

## 7. Dependencies

Phase 2 re-adds minimal dependencies to `go.mod`:
- `github.com/google/uuid` — for file/batch ID generation (used in schema types).
- No ORM needed yet (SQLite schema arrives in Phase 6).
- `encoding/json` and standard library only for schemas.

---

## RESEARCH COMPLETE
