# Architecture

## Overview

g0router is a single-binary Go LLM gateway with 43+ providers, fasthttp, OAuth flows, RTK compression, caveman, cost tracking, combo models, MCP gateway, and a React dashboard.

```
                    ┌─────────────────────────────────────┐
                    │            g0router binary           │
                    │                                      │
  CLI commands ────►│  ┌──────────┐   ┌────────────────┐  │
                    │  │  cobra   │   │  fasthttp      │  │◄──── HTTP clients
  g0router login   │  │  CLI     │   │  server        │  │      (Claude Code,
  g0router status  │  │          │   │                │  │       Codex, curl,
  g0router keys    │  └────┬─────┘   └───────┬────────┘  │       SDKs)
                    │       │                 │            │
                    │       ▼                 ▼            │
                    │  ┌─────────────────────────────────┐│
                    │  │         Internal API Layer       ││
                    │  │  (shared by CLI + HTTP handlers) ││
                    │  └──────────────┬──────────────────┘│
                    │                 │                    │
                    │    ┌────────────┼────────────┐      │
                    │    ▼            ▼            ▼      │
                    │  ┌──────┐ ┌─────────┐ ┌─────────┐  │
                    │  │Auth  │ │ Proxy   │ │ Usage   │  │
                    │  │OAuth │ │ Engine  │ │ Tracker │  │
                    │  │Keys  │ │ Fallback│ │ Cost    │  │
                    │  │Refresh│ │ Combos │ │ Quota   │  │
                    │  └────┬───┘ └────┬────┘ └────┬────┘  │
                    │     │          │           │        │
                    │     ▼          ▼           ▼        │
                    │  ┌─────────────────────────────────┐│
                    │  │        SQLite Store              ││
                    │  │  connections, settings, usage,   ││
                    │  │  apikeys, combos, pricing, logs  ││
                    │  └─────────────────────────────────┘│
                    │                 │                    │
                    │    ┌────────────┼────────────┐      │
                    │    ▼            ▼            ▼      │
                    │  ┌──────┐ ┌─────────┐ ┌─────────┐  │
                    │  │RTK   │ │Translate│ │MCP      │  │
                    │  │Filters│ │Format  │ │Gateway  │  │
                    │  │Caveman│ │Convert │ │Discovery│  │
                    │  └──────┘ └─────────┘ └─────────┘  │
                    │                 │                    │
                    │                 ▼                    │
                    │  ┌─────────────────────────────────┐│
                    │  │       Provider Implementations   ││
                    │  │  OpenAI, Anthropic, Gemini,      ││
                    │  │  Bedrock, Azure, Groq, Mistral,  ││
                    │  │  Ollama, OpenRouter, xAI, ...    ││
                    │  └─────────────────────────────────┘│
                    │                 │                    │
                    │                 ▼                    │
                    │           Upstream APIs              │
                    └─────────────────────────────────────┘
```

## Request Pipeline

```
HTTP Request → Middleware (auth, CORS, request ID, logging)
            → Route Handler (inference, models, oauth, etc.)
            → [RTK compression on tool_result content]
            → [Caveman injection on system message]
            → Format Translation (source → target format)
            → Provider Selection (registry → fallback → combo)
            → [OAuth token refresh if needed]
            → Provider.ChatCompletion() or .ChatCompletionStream()
            → [Usage extraction from response]
            → [Cost calculation from pricing catalog]
            → [Request log write to SQLite]
            → [Format Translation (target → source format)]
            → HTTP Response (SSE stream or JSON)
```

## Package Dependency Graph

```
cmd/g0router/main.go
  └── internal/cli/           (cobra commands)
        ├── internal/config/  (app config)
        ├── internal/store/   (SQLite persistence)
        └── internal/provider/ (registry, oauth, fallback)

api/server.go
  ├── api/middleware.go        (auth, CORS)
  ├── api/handlers/            (HTTP handlers)
  │     ├── internal/proxy/    (engine, inference, combo)
  │     ├── internal/rtk/      (compression, caveman)
  │     ├── internal/translate/ (format conversion)
  │     ├── internal/usage/    (tracker, cost, quota)
  │     ├── internal/mcp/      (gateway, discovery)
  │     └── internal/provider/ (registry, oauth)
  └── api/handlers/            (HTTP handlers)

internal/providers/            (provider implementations)
  ├── internal/providers/utils/ (shared HTTP, SSE)
  └── internal/streaming/       (accumulator)
```

## Key Interfaces

```go
// Provider — the core abstraction. Each upstream (OpenAI, Anthropic, etc.)
// implements this. Simplified to LLM essentials.
type Provider interface {
    Name() ModelProvider
    ChatCompletion(ctx context.Context, key Key, req *ChatRequest) (*ChatResponse, error)
    ChatCompletionStream(ctx context.Context, key Key, req *ChatRequest) (<-chan StreamChunk, error)
    ListModels(ctx context.Context, key Key) ([]Model, error)
}

// OAuthProvider — implements a specific OAuth flow (device-code, PKCE, callback).
type OAuthProvider interface {
    ID() string
    Name() string
    StartLogin(ctrl OAuthController) (*OAuthAuthInfo, error)
    PollLogin(params map[string]string) (*OAuthCredentials, error)
    RefreshToken(creds *OAuthCredentials) (*OAuthCredentials, error)
}

// Store — persistence layer. All data in one SQLite database.
type Store interface {
    // Connections
    GetConnections(provider string) ([]Connection, error)
    CreateConnection(conn *Connection) error
    UpdateConnection(conn *Connection) error
    DeleteConnection(id string) error
    // Settings
    GetSettings() (*Settings, error)
    UpdateSettings(s *Settings) error
    // Usage
    LogRequest(entry *RequestLogEntry) error
    GetUsage(filter UsageFilter) ([]RequestLogEntry, error)
    GetUsageSummary(filter UsageFilter) (*UsageSummary, error)
    // ... (combos, apikeys, aliases, pricing, mcp clients)
}

// RTKFilter — a pure function that compresses tool output content.
type RTKFilter func(input string) string
```

## MCP Discovery Protocol

```
Phase 1: Connect
  Client connects → g0router fetches tools/list → stores full schemas in DB

Phase 2: Inject (per-request)
  Request has tools=true → inject compact manifests:
    [{name: "read_file", description: "Read a file from disk"}]
  NOT the full JSON Schema for parameters. ~90% token savings.

Phase 3: Execute
  Model selects tool "read_file" → g0router looks up full schema →
  validates args → executes via MCP client → returns result

Phase 4: Health
  Periodic health checks → on failure, re-fetch manifests → update DB
```

## Data Flow: OAuth Login

```
CLI: g0router login anthropic
  → internal/cli/auth.go
  → internal/provider/oauth/anthropic.go::StartLogin()
  → Opens browser to claude.ai/oauth/authorize
  → Starts local callback server on :54545
  → User authorizes → callback receives code
  → Exchange code for tokens via anthropic.com/v1/oauth/token
  → internal/provider/oauth/anthropic.go::ExchangeCode()
  → internal/store/connections.go::CreateConnection()
  → Stored in SQLite as {provider: "anthropic", auth_type: "oauth", ...}
  → "✓ Connected to Anthropic (claude-user@example.com)"
```

## Data Flow: Proxied Request

```
POST /v1/chat/completions {model: "claude-sonnet-4-20250514", messages: [...]}
  → middleware: validate API key or session
  → handler: parse request, detect source format (openai)
  → provider registry: resolve "claude-sonnet-4-20250514" → anthropic provider
  → connection selection: round-robin among active anthropic connections
  → [if OAuth: check token expiry, refresh if needed]
  → RTK: scan tool_result content, auto-detect filter, compress
  → caveman: if enabled, inject system prompt prefix
  → translate: openai format → anthropic format
  → provider.ChatCompletionStream(key, request)
  → stream chunks: translate back anthropic → openai format
  → usage: extract tokens from response, calculate cost
  → log: write request_log row to SQLite
  → SSE stream to client
```
