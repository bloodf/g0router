# Requirements

<!-- Operator-owned. Drafted from existing docs (WORKFLOW.md stage 12B-19 plan) — review and edit; agents read but never write. -->

## Functional Requirements

- OpenAI-compatible inference endpoint with translation to native provider formats (16 native adapters; adapter-only providers tracked separately)
- Connection management: API keys + OAuth flows per provider, multiple connections per provider, automatic account fallback on rate limit
- Combos: named model aliases with `fallback` / `round-robin` / `least-used` / `auto` strategies
- Dashboard auth: first-run setup, session cookies, CSRF protection, user CRUD (phase 13)
- Proxy pools and per-connection proxy assignment (phase 14)
- Tunnels: cloudflared (pinned binary) + tailscale (preinstalled) exposure (phase 15)
- Chat console with streaming, image attachments, server console log view (phase 16)
- Usage analytics from `request_log`: time-series charts, cost attribution, bulk model disable (phase 17)
- Governance: teams, virtual keys with budgets, routing rules, model limits, guardrails, alerts, prompt templates, backup/restore (phase 18)
- Advanced: semantic cache, self-update with checksum verification, WebSocket chat, MITM proxy CA (phase 19)
- RTK/Caveman prompt compression; MCP gateway; embedded React dashboard; CLI control plane

## Non-Functional Requirements

- Single static binary, SQLite (WAL) storage, additive-only migrations (`ensureColumn`)
- ≥95% Go test coverage; `go test ./...`, `go vet`, race detector green at every commit
- Secrets encrypted at rest (reversible `*_enc` columns); passwords bcrypt-hashed
- Layered DDD architecture: transport (api/handlers) → domain (internal/<domain>) → repository (internal/store); enforced by arch conformance test (phase 12B)
- API contract: snake_case JSON, `{data, error}` envelope, audit log on mutations, feature-flag gating for risky features
- No telemetry; fully self-hosted

## Out of Scope

- WebRTC transport, 33-locale i18n (en + pt-BR only), adaptive routing beyond the existing `auto` classifier, OpenTelemetry export (deferred — see docs/WORKFLOW.md deferral list)
- Hosted/multi-tenant SaaS operation
