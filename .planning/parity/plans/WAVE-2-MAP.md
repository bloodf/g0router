# Wave 2 — Provider adapters: micro-plan index (Stage-1 scope)

Author: Fable 5 (planner). Orchestrator: Sonnet. Implementers: kimi/M3. Gates: gpt-5.5.
**This is a non-authorizing INDEX** (like `WAVE-MAP.md`): it does not authorize
implementation. Each `w2-<slug>.md` micro-plan carries its own cited rows,
TDD-ordered tasks, binary acceptance, and goes through the gpt-5.5 plan gate before
dispatch. Frozen ref @ 827e5c3. Depends on Wave 1 (translators) — COMPLETE.

## Stage-1 scope is the matrix's "Include now" ranking — NOT all 67 providers

`matrix/9router-providers.md` §"Stage 1 Go-port ranking" lists exactly the providers
Stage 1 ports; the rest are **Defer to Stage 2+** (quoted there). Wave 2 implements
ONLY the "Include now" set:

| PAR-PROV | Provider | Format | Auth | providers.js |
|---|---|---|---|---|
| PAR-PROV-005 | deepseek | openai | api-key | :257-259 |
| PAR-PROV-004 | groq | openai | api-key | :269-271 |
| PAR-PROV-006 | mistral | openai | api-key | :281-283 |
| PAR-PROV-008 | together | openai | api-key | :289-291 |
| PAR-PROV-009 | fireworks | openai | api-key | :293-295 |
| PAR-PROV-007 | cohere | openai | api-key | :301-303 |
| PAR-PROV-027 | xai (API-key path only; OAuth=Stage 2) | openai | api-key | :273-280 |
| PAR-PROV-014 | openrouter | openai | api-key | :115-122 |
| PAR-PROV-029 | perplexity | openai | api-key | :285-287 |
| PAR-PROV-010 | ollama / ollama-local | ollama | none | :333-340 |

Nine are **pure OpenAI-format API-key** providers; per Go-port consideration #2
("DefaultExecutor covers ~80% via OpenAI-compatible passthrough; a generic Go
executor … would collapse many adapters into one struct") they collapse into ONE
config-driven generic adapter. ollama is ollama-native (uses the w1-h openai↔ollama
translators) and gets its own thin adapter.

## Deferred to Stage 2+ (per matrix ranking — NOT Wave-2 plans)

Recorded so the index is exhaustive; each is out of Wave-2 scope:
- **OAuth providers**: claude, codex, gemini-cli, qwen, iflow, antigravity, github,
  kiro, cursor, kimi-coding, cline, kilocode, xai-OAuth (→ Wave 3 PAR-AUTH + their executors).
- **Custom format / reverse-engineered**: cursor, kiro, commandcode, qoder, grok-web,
  perplexity-web, azure (Go-port considerations #4, #5).
- **GCP/enterprise**: vertex, vertex-partner, cloudflare-ai, bedrock.
- **Chinese ecosystems**: glm, glm-cn, kimi, alicode(-intl), volcengine-ark, byteplus,
  xiaomi-mimo, xiaomi-tokenplan, siliconflow.
- **Media/STT/TTS/embedding specialists** (consideration #6): fal-ai, stability-ai,
  black-forest-labs, recraft, runwayml, sdwebui, comfyui, huggingface, deepgram,
  assemblyai, nanobanana, voyage-ai, nvidia.
- **Free-tier / experimental** (29 OmniRoute): agentrouter, aimlapi, novita, … blackbox, chutes.
- **No-op/stub**: opencode, opencode-go, gitlab, codebuddy, vercel-ai-gateway.

## Micro-plan index (4 plans, dependency-ordered)

| Plan | Scope | PAR-PROV rows | Ref surface | Depends |
|---|---|---|---|---|
| **w2-a** | Provider config catalog: Go struct `{baseURL, format, headers, authHeader, noAuth}` + entries for the 10 Stage-1 providers + lookup; `providerModels.js` static catalogs for those 10 → model→provider map | 004,005,006,007,008,009,010,014,027,029 (config substrate) | `config/providers.js:115-122,257-259,269-271,273-280,281-283,285-287,289-291,293-295,301-303,333-340`, `config/providerModels.js` (those provider blocks) | W1 |
| **w2-b** | Generic OpenAI-compatible adapter: one config-driven `Provider` (chat + stream + embeddings) replacing the per-dir stubs for the 9 openai-format Stage-1 providers; bearer auth; PAR-PR-664 (`max_tokens`→`max_completion_tokens` for openai-compatible) | 004,005,006,007,008,009,014,027,029 + PAR-PR-664 | `executors/base.js`, `executors/default.js`; existing `internal/providers/openai/chat.go` as the pattern | w2-a |
| **w2-c** | Ollama adapter: ollama-native chat+stream via w1-h openai↔ollama translators; no-auth; local (`ollama-local`) + cloud host resolution (`resolveOllamaLocalHost`) | 010 | `executors/ollama-local.js`, `config/providers.js:333-340,442-445`, w1-h translators | w2-a |
| **w2-d** | Provider registry + catalog-driven router: replace prefix `Router` (`internal/inference/router.go`) with catalog/model-catalog resolution wiring all 10 Stage-1 providers; `/v1/models` aggregation over their catalogs | routing for 004-010,014,027,029 | `services/provider.js`, `internal/inference/router.go` | w2-a,b,c |

## Ownership & order

Strict serial: w2-a (catalog/data) → w2-b (generic adapter) + w2-c (ollama) [disjoint
adapter files, both depend only on a] → w2-d (router wiring, depends on all). w2-b and
w2-c may be authored in parallel but implemented either order; w2-d last (it imports both).

## Per-micro-plan protocol (same as Wave 1)

Fable 5 writes plan → gpt-5.5 plan gate (max 3 cycles) → kimi/M3 impl verbatim →
`go test ./... && go vet ./...` → gpt-5.5 scoped diff gate → merge → flip PAR-PROV
rows HAVE → update WORKFLOW.md.

## Out of Wave-2 scope (explicit)

OAuth acquisition/refresh (Wave 3 — the generic adapter accepts a key, never mints
it). Combo/fallback/rate-limit/alias routing (Wave 4 — w2-d does single-provider
catalog resolution only). Usage/cost (Wave 5). Bifrost non-core capabilities stay
typed not-implemented stubs (decision 9). All Stage-2+ providers above.
