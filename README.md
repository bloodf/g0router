# g0router

Single-binary Go LLM gateway that unifies multiple AI provider APIs behind one endpoint.

## What It Does

- **Unified API**: Send OpenAI-format requests → g0router routes today to OpenAI and Anthropic, with additional adapters tracked in the provider matrix
- **OAuth Login**: `g0router login anthropic` → browser OAuth → credentials stored
- **Format Translation**: Client sends OpenAI format → g0router translates supported provider formats and tracks adapter-only providers separately
- **Account Fallback**: Rate limited on one connection? Automatically tries the next
- **RTK Compression**: Tool outputs (git diffs, build logs) compressed 40–80% before sending to LLM
- **Cost Tracking**: Token usage + cost per request, per provider, per model
- **MCP Gateway**: Connect MCP servers, inject tools into requests with 90% token savings
- **Dashboard**: Web UI for managing providers, viewing usage, configuring settings

## Quick Start

```bash
# Build and test
make test
make build

# Login to a provider
./g0router login anthropic    # Opens browser for OAuth
./g0router login openai --key # Prompts for API key

# Generate gateway API key. Keep this same secret for serve.
export API_KEY_SECRET="$(openssl rand -hex 32)"
./g0router keys add default

# Start serving
./g0router serve

# Add MCP instances for separate accounts on the same MCP server
./g0router mcp add atlassian-a --server-key atlassian --launch-type http --transport streamable-http --url https://mcp.atlassian.com/mcp --account-label account-a
./g0router mcp add atlassian-b --server-key atlassian --launch-type http --transport streamable-http --url https://mcp.atlassian.com/mcp --account-label account-b
./g0router mcp auth start atlassian-a --authorization-url https://auth.example/authorize --resource https://mcp.atlassian.com --redirect-url http://localhost:20128/api/mcp/oauth/callback
./g0router mcp auth complete atlassian-a "http://localhost:20128/api/mcp/oauth/callback?code=...&state=..."

# Use it (OpenAI-compatible endpoint)
curl http://localhost:20128/v1/chat/completions \
  -H "Authorization: Bearer <your-gateway-key>" \
  -H "Content-Type: application/json" \
  -d '{"model":"claude-sonnet-4-20250514","messages":[{"role":"user","content":"Hello"}]}'
```

## Deployment

```bash
# systemd service
sudo ./g0router install
sudo systemctl status g0router

# Docker image
make docker
export JWT_SECRET="$(openssl rand -hex 32)"
export API_KEY_SECRET="$(openssl rand -hex 32)"
docker run --rm -p 127.0.0.1:20128:20128 \
  -e JWT_SECRET="${JWT_SECRET}" \
  -e API_KEY_SECRET="${API_KEY_SECRET}" \
  -e BIND_ADDRESS=0.0.0.0 \
  g0router:latest

# Docker Compose
JWT_SECRET="${JWT_SECRET}" API_KEY_SECRET="${API_KEY_SECRET}" docker compose up -d
```

Keep both secrets stable across restarts. `API_KEY_SECRET` validates gateway and dashboard control-plane API keys; `JWT_SECRET` is separate session-signing material for dashboard/admin flows. See [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) for systemd, Docker, API key bootstrap, logs, health checks, and upgrade steps.

## Supported Providers

Public direct-dispatch support is currently OpenAI and Anthropic. Registered adapters, auth-only providers, and unsupported parity targets are listed explicitly in [docs/PROVIDERS.md](docs/PROVIDERS.md); do not treat a provider name there as usable unless its status is `supported`.

## Documentation

| Document | Description |
|----------|-------------|
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | System design, request pipeline, interfaces |
| [docs/PLAN.md](docs/PLAN.md) | Implementation roadmap (12 phases, 71 tasks) |
| [docs/SCHEMA.md](docs/SCHEMA.md) | SQLite schema + API contracts |
| [docs/CONFIG.md](docs/CONFIG.md) | Environment variables reference |
| [docs/PROVIDERS.md](docs/PROVIDERS.md) | Provider catalog with auth details |
| [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) | systemd, Docker, nginx |
| [docs/REFERENCES.md](docs/REFERENCES.md) | Source mapping from 9router/bifrost/oh-my-pi |
| [docs/WORKFLOW.md](docs/WORKFLOW.md) | Development workflow + task status |

## Architecture

```
HTTP client → g0router → [auth] → [RTK compress] → [format translate] → Provider API
                                                                              ↓
HTTP client ← g0router ← [usage track] ← [format translate] ←───── Provider Response
```

Single binary. SQLite for persistence. No external dependencies at runtime.

## Development

```bash
# Prerequisites: Go 1.24+, Node 22+ (for UI)

make test    # Run all tests
make vet     # Run go vet
make build   # Build binary
make ui      # Build React dashboard
make docker  # Build Docker image
```

See [CLAUDE.md](CLAUDE.md) for development rules (TDD, commit conventions, code style).

## Lineage

g0router combines patterns from three projects:
- **[bifrost](https://github.com/maximhq/bifrost)** — Provider engine, fasthttp, object pooling, MCP
- **[9router](https://github.com/decolua/9router)** — OAuth flows, RTK, caveman, cost tracking, combos, UI
- **[oh-my-pi](https://github.com/can1357/oh-my-pi)** — OAuth catalog (50+ providers), credential storage
