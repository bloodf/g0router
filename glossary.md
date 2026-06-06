# Glossary

The Ubiquitous Language for this project - the domain terms the team and any LLM agents working in this repo use to describe the system. Keep this list current as the domain evolves; agents prefer existing terms over inventing synonyms.

<!-- Format: each entry is a term followed by a one-to-two sentence definition. -->
<!-- Group related terms under H2 headings as the glossary grows. -->

- **Provider** - an upstream LLM API (Anthropic, OpenAI, etc.). 16 native adapters live in `internal/providers/`.
- **Connection** - a stored credential (API key or OAuth) binding the gateway to one provider account. Multiple connections per provider enable account fallback.
- **Combo** - a named model alias that resolves to an ordered list of provider/model pairs with a strategy (`fallback`, `round-robin`, `least-used`, `auto`).
- **Adapter** - the per-provider translation layer converting OpenAI-format requests to the provider's native format and back.
- **Compression (RTK/Caveman)** - token-reduction middleware applied to prompts before forwarding upstream.
- **MCP Gateway** - the Model Context Protocol proxy surface exposing configured MCP servers through the gateway.
- **Traffic broker** - the in-process pub/sub ring (`internal/traffic`) that mirrors request events to SSE subscribers for the live topology view.
- **Request log** - the `request_log` SQLite table; the single source of usage/analytics data (there is no `usage_logs` table).
