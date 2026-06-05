# Directory Structure

Target repository layout after all phases are complete. Files organized by Go package convention.

```
g0router/
│
├── cmd/g0router/
│   └── main.go                          # Entry point: cobra root command
│
├── api/                                  # HTTP server package
│   ├── server.go                         # Server struct, route registration, Start/Stop
│   ├── server_test.go                    # Health, routing, shutdown tests
│   ├── middleware.go                     # CORS, request ID, API key auth, timing
│   ├── middleware_test.go
│   ├── handlers/                         # HTTP request handlers
│   │   ├── health.go                     # GET /healthz
│   │   ├── inference.go                  # POST /v1/chat/completions, /v1/messages
│   │   ├── inference_test.go             # Tests with fake engine
│   │   ├── models.go                     # GET /v1/models
│   │   ├── providers.go                  # GET /api/providers, /api/providers/:id/models
│   │   ├── connections.go                # CRUD /api/connections
│   │   ├── settings.go                   # GET/PUT /api/settings
│   │   ├── apikeys.go                    # CRUD /api/keys
│   │   ├── combos.go                     # CRUD /api/combos
│   │   ├── oauth.go                      # /api/oauth/:provider/authorize, /poll, /callback
│   │   ├── usage.go                      # GET /api/usage, /api/usage/summary, /api/usage/quota
│   │   ├── logging.go                    # GET /api/logs
│   │   └── mcp.go                        # CRUD /api/mcp/clients, /api/mcp/tools
│
├── internal/                             # Private packages
│   ├── cli/                              # Cobra CLI commands
│   │   ├── root.go                       # All CLI commands (serve, login, logout, keys, providers, status, mcp, install, etc.)
│   │   ├── install.go                    # `g0router install [--user]`
│   │   └── install_test.go
│   │
│   ├── config/                           # Configuration loading
│   │   ├── config.go                     # Load() from env vars, defaults, validation
│   │   └── config_test.go
│   │
│   ├── store/                            # SQLite persistence layer
│   │   ├── sqlite.go                     # Store struct, NewStore, Close, migrate
│   │   ├── sqlite_test.go               # Migration, idempotency tests
│   │   ├── connections.go                # Connection CRUD
│   │   ├── connections_test.go
│   │   ├── settings.go                   # Settings get/update
│   │   ├── settings_test.go
│   │   ├── apikeys.go                    # API key CRUD + HMAC validation
│   │   ├── apikeys_test.go
│   │   ├── usage.go                      # Request log + usage summary
│   │   ├── usage_test.go
│   │   ├── combos.go                     # Combo model chains CRUD
│   │   ├── combos_test.go
│   │   ├── aliases.go                    # Model alias CRUD
│   │   ├── pricing.go                    # Pricing override CRUD
│   │   ├── mcpclients.go                 # MCP client config + manifest storage
│   │   └── errors.go                     # ErrNotFound sentinel
│   │
│   ├── providers/                        # Provider implementations
│   │   ├── types.go                      # ChatRequest, ChatResponse, StreamChunk, Key, Model
│   │   ├── types_test.go                 # JSON round-trip tests
│   │   ├── interface.go                  # Provider interface definition
│   │   ├── openai/                       # OpenAI provider
│   │   │   ├── openai.go                 # ChatCompletion, ChatCompletionStream, ListModels
│   │   │   ├── types.go                  # OpenAI-specific wire types (if needed)
│   │   │   ├── responses.go              # Responses API support
│   │   │   ├── errors.go                 # Error response parsing
│   │   │   └── openai_test.go            # Request building, response parsing, SSE tests
│   │   ├── anthropic/                    # Anthropic Messages API
│   │   │   ├── anthropic.go
│   │   │   ├── types.go                  # AnthropicRequest, AnthropicResponse, SSE events
│   │   │   ├── errors.go
│   │   │   └── anthropic_test.go
│   │   ├── gemini/                       # Gemini generateContent
│   │   │   ├── gemini.go
│   │   │   ├── types.go
│   │   │   └── gemini_test.go
│   │   ├── openaicompat/                 # Config-driven OpenAI-compatible providers
│   │   │   ├── provider.go              # Generic implementation (URL + headers differ)
│   │   │   ├── registry.go              # Pre-built configs for Groq, Cerebras, etc.
│   │   │   └── provider_test.go
│   │   ├── bedrock/                      # AWS Bedrock with SigV4
│   │   │   ├── bedrock.go
│   │   │   └── bedrock_test.go
│   │   ├── azure/                        # Azure OpenAI (deployment URL, api-key header)
│   │   │   ├── azure.go
│   │   │   └── azure_test.go
│   │   ├── vertex/                       # Vertex AI (GCP auth + Gemini format)
│   │   │   ├── vertex.go
│   │   │   └── vertex_test.go
│   │   └── utils/                        # Shared provider utilities
│   │       ├── http.go                   # fasthttp client wrapper, retry logic
│   │       ├── http_test.go
│   │       ├── sse.go                    # SSE parser (data: lines, [DONE])
│   │       ├── sse_test.go
│   │       └── errors.go                # ProviderError, ErrAuth, ErrRateLimit, etc.
│   │
│   ├── proxy/                            # Proxy engine
│   │   ├── engine.go                     # Engine struct, Dispatch, DispatchStream
│   │   ├── engine_test.go
│   │   ├── pool.go                       # sync.Pool for ChatRequest/Response
│   │   └── combo.go                      # Combo model sequential fallback
│   │
│   ├── provider/                         # Provider management (registry, connections, OAuth)
│   │   ├── registry.go                   # Provider registry, model resolution
│   │   ├── registry_test.go
│   │   ├── connection.go                 # Round-robin connection selection
│   │   ├── connection_test.go
│   │   ├── fallback.go                   # Exponential backoff, per-model locks
│   │   ├── fallback_test.go
│   │   ├── refresh.go                    # Token refresh with singleflight dedup
│   │   ├── refresh_test.go
│   │   └── oauth/                        # Per-provider OAuth implementations
│   │       ├── types.go                  # OAuthProvider interface, OAuthCredentials
│   │       ├── types_test.go
│   │       ├── anthropic.go              # PKCE + callback
│   │       ├── codex.go                  # Device-code + callback
│   │       ├── github.go                 # Device-code (Copilot)
│   │       ├── cursor.go                 # PKCE + polling
│   │       ├── gemini.go                 # OAuth2 + callback
│   │       ├── antigravity.go            # OAuth2 + callback
│   │       ├── xai.go                    # OAuth2
│   │       ├── deepseek.go              # Password login
│   │       ├── gitlab.go                 # OAuth2
│   │       ├── kimi.go                   # Device-code
│   │       ├── minimax.go               # API key
│   │       ├── alibaba.go               # API key
│   │       ├── zhipu.go                  # API key
│   │       └── xiaomi.go                 # OAuth2
│   │
│   ├── translate/                        # Format translation engine
│   │   ├── detect.go                     # DetectFormat (OpenAI/Anthropic/Gemini heuristic)
│   │   ├── detect_test.go
│   │   ├── openai.go                     # OpenAI canonical helpers
│   │   ├── anthropic.go                  # OpenAI ↔ Anthropic translation
│   │   ├── anthropic_test.go
│   │   ├── gemini.go                     # OpenAI ↔ Gemini translation
│   │   ├── gemini_test.go
│   │   └── responses.go                  # Responses API translation
│   │
│   ├── rtk/                              # Response Token Kompression
│   │   ├── autodetect.go                 # Content format detection (first 1KB)
│   │   ├── autodetect_test.go
│   │   ├── rtk.go                        # CompressMessages (entry point)
│   │   ├── rtk_test.go
│   │   ├── caveman.go                    # Caveman prompt injection
│   │   ├── caveman_test.go
│   │   ├── prompts.go                    # Caveman prompt text (lite/full/ultra)
│   │   ├── constants.go                  # Thresholds, limits
│   │   └── filters/                      # Compression filters
│   │       ├── filters.go                # All 11 filter implementations
│   │       └── filters_test.go
│   │
│   ├── streaming/                        # Stream accumulation
│   │   ├── accumulator.go               # Collects chunks → complete response
│   │   ├── accumulator_test.go
│   │   ├── chat.go                       # Chat-specific accumulation helpers
│   │   └── responses.go                  # Responses API streaming
│   │
│   ├── usage/                            # Usage tracking
│   │   ├── tracker.go                    # Extract usage from provider responses
│   │   ├── tracker_test.go
│   │   ├── cost.go                       # Calculate cost from usage + pricing
│   │   ├── cost_test.go
│   │   └── quota.go                      # Per-provider quota API fetchers
│   │
│   ├── modelcatalog/                     # Model + pricing catalog
│   │   ├── pricing.go                    # DefaultPricing map (100+ models)
│   │   ├── catalog.go                    # Model lists per provider
│   │   └── pricing_test.go
│   │
│   ├── mcp/                              # MCP gateway
│   │   ├── clientmanager.go              # Client lifecycle (connect/disconnect/reconnect)
│   │   ├── clientmanager_test.go
│   │   ├── toolmanager.go               # Tool registration + lookup
│   │   ├── toolmanager_test.go
│   │   ├── discovery.go                  # Compact manifest generation + TTL cache
│   │   ├── agent.go                      # Multi-turn tool execution loop
│   │   ├── agent_test.go
│   │   └── healthmonitor.go             # Periodic ping + auto-reconnect
│   │
│   └── logging/                          # Request/response logging
│       ├── logger.go                     # RequestLogger with toggle
│       └── requestlog.go                # Client detection (User-Agent → tool name)
│
├── ui/                                   # React dashboard (Vite + Tailwind)
│   ├── package.json
│   ├── vite.config.ts
│   ├── tailwind.config.ts
│   ├── tsconfig.json
│   ├── index.html
│   └── src/
│       ├── main.tsx
│       ├── App.tsx
│       ├── api/client.ts                 # Typed fetch wrappers
│       ├── components/                   # Shared UI components
│       └── pages/                        # Dashboard, Providers, Usage, etc.
│
├── deploy/                               # Deployment artifacts
│   ├── g0router.service                  # systemd unit file
│   ├── g0router.default                  # /etc/default/g0router env template
│
│
├── docs/                                 # All documentation
│   ├── README.md                         # Documentation hub
│   ├── ARCHITECTURE.md                   # System design + diagrams
│   ├── PLAN.md                           # Master plan index
│   ├── WORKFLOW.md                       # Agent handoff protocol + task status
│   ├── SCHEMA.md                         # SQLite schema + API contracts
│   ├── REFERENCES.md                     # Historical source mapping (migration complete)
│   ├── DEPLOYMENT.md                     # systemd, Docker, nginx
│   ├── CONFIG.md                         # Environment variables reference
│   ├── PROVIDERS.md                      # Provider catalog
│   ├── DIRECTORY_STRUCTURE.md            # This file
│   └── phases/                           # Per-phase implementation guides
│       ├── phase-00-project-bootstrap.md
│       ├── phase-01-core-types-sqlite-store.md
│       ├── phase-02-http-server-proxy-engine.md
│       ├── phase-03-multi-provider-support.md
│       ├── phase-04-persistence-provider-registry.md
│       ├── phase-05-oauth-flows-cli.md
│       ├── phase-06-account-fallback-combos.md
│       ├── phase-07-rtk-caveman.md
│       ├── phase-08-usage-tracking-cost-logging.md
│       ├── phase-09-mcp-gateway.md
│       ├── phase-10-dashboard-ui.md
│       ├── phase-11-packaging-deployment-polish.md
│       └── phase-12-advanced-mcp-gateway.md
│
├── embed.go                              # //go:embed ui/dist/* (production build)
├── Makefile                              # build, test, lint, ui, docker, install
├── Dockerfile                            # Multi-stage: node → go → distroless
├── .dockerignore
├── go.mod
├── go.sum
├── .gitignore
├── .env.example
├── CLAUDE.md                             # AI agent guidelines + project rules
└── README.md                             # Project overview
```

## Package Count Summary

| Area | Packages | Files (est.) |
|------|----------|-------------|
| Commands | 2 (`cmd/g0router`, `internal/cli`) | ~12 |
| HTTP | 2 (`api`, `api/handlers`) | ~15 |
| Providers | 8+ (`providers/*`, `openaicompat`) | ~25 |
| Core | 7 (`store`, `config`, `proxy`, `provider`, `translate`, `streaming`, `logging`) | ~30 |
| RTK | 2 (`rtk`, `rtk/filters`) | ~25 |
| Usage | 2 (`usage`, `modelcatalog`) | ~8 |
| MCP | 1 (`mcp`) | ~8 |
| UI | — (React, not Go) | ~15 |
| **Total** | ~24 Go packages | ~140 Go files + ~15 TS files |
