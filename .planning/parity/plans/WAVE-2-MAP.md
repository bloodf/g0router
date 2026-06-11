# Wave 2 — Provider adapters & executors: micro-plan breakdown

Author: Fable 5 (planner). Orchestrator: Sonnet. Implementers: kimi/M3. Gates: gpt-5.5.
Source: PARITY.md PAR-PROV-001..067 (+ ~25 PR ports) + WAVE-MAP.md row 2 + checkpoint decisions.
Frozen ref @ 827e5c3. Wave 2 depends on Wave 1 (translators) — COMPLETE.

## Scope boundary (read first)

Wave 2 builds the **provider config catalog, the executor layer, model catalogs, and
the catalog-driven router** so requests reach all 43+ upstreams. Wave 2 consumes
credentials (`apiKey`/`accessToken`) but does NOT mint them — **OAuth device-code /
token-refresh flows are Wave 3 (PAR-AUTH)**. Each Wave-2 executor accepts a
credentials struct and uses whatever token is present; the acquisition handlers,
refresh loops, and session hardening land in Wave 3.

The Bifrost-size `Provider` interface already exists (`internal/schemas/provider.go:68-107`,
decision 9) — chat/text/responses/embeddings/images/speech/transcription/files/batch/
count-tokens. Wave 2's job is real implementations for the core capabilities each
provider supports + **typed not-implemented stubs** for the rest (the `stubs.go`
pattern already in `internal/providers/openai/stubs.go`).

## Structural model (how the ref is organized — port this shape)

- **Config catalog** (`config/providers.js:50-438`, ~70 entries): per-provider
  `{baseUrl|baseUrls, format, headers, authHeader?, noAuth?, retry?, clientId/tokenUrl
  (Wave-3 handoff), special URL fields}`. `format` ∈ {openai, claude, gemini,
  gemini-cli, vertex, antigravity, openai-responses, kiro, cursor, ollama,
  commandcode, grok-web, perplexity-web} → maps 1:1 to Wave-1 translators.
- **Executor layer** (`executors/`): `BaseExecutor` (buildUrl/buildHeaders/auth/retry,
  `base.js`), `DefaultExecutor` (OpenAI-compatible chat+stream, `default.js`), and ~18
  specialized executors dispatched by `getExecutor(provider)` (`executors/index.js:1-52`).
  The DefaultExecutor's `refresh*` methods are Wave-3 token refresh — STUB in Wave 2
  (return "refresh is Wave-3" error), wire real in Wave 3.
- **Model catalogs** (`config/providerModels.js`, `models.js`, `ollamaModels.js`):
  alias/id → provider + model list; feeds `/v1/models` and routing.

## Micro-plan index (12 plans, dependency-ordered)

| Plan | Scope | PAR-PROV rows | Ref surface | Depends |
|---|---|---|---|---|
| **w2-a** | Provider config catalog: port `providers.js` → `internal/providers/catalog` (baseUrl/baseUrls/format/headers/authHeader/noAuth/retry/special-URL/oauth-handoff fields) + lookup + OS/arch stainless header helpers + ollama-local/xiaomi-region resolvers | substrate for 001-067 | `config/providers.js:1-457` | W1 |
| **w2-b** | Executor base + DefaultExecutor (OpenAI-compatible chat + stream through W1 openai path; buildUrl/buildHeaders/bearer+x-api-key/noAuth; retry config; refresh*=Wave-3 stub) | 001,004-009,014,029,041-057 (openai-format core) | `executors/base.js`, `executors/default.js` | w2-a |
| **w2-c** | Provider registry + catalog-driven router: replace prefix `Router` with catalog+model-catalog resolution; `getExecutor` dispatch; provider factory; Bifrost stubs wired for default-executor providers | 001 (routing), registry | `executors/index.js:1-55`, `services/provider.js` | w2-a, w2-b |
| **w2-d** | Model catalogs: port `providerModels.js`/`models.js`/`ollamaModels.js` → model→provider map; `/v1/models` aggregation; alias→provider table (cc/cx/gc/qw/if/ag/gh/kr/cu/kmc/cl/kc/qd) for Wave-3 | catalog substrate | `config/providerModels.js`, `models.js`, `ollamaModels.js` | w2-a, w2-c |
| **w2-e** | Claude-format providers via default executor + claude translator + prepareClaudeRequest (w1-l) + anthropic/x-api-key headers | 002,015,024,034,036,067-claude (anthropic,claude,glm,kimi,minimax,minimax-cn,agentrouter,kimi-coding) | `config/providers.js` claude entries, `base.js` anthropic-compatible | w2-a,b,c + w1-d/l |
| **w2-f** | Google family executors: gemini (API), gemini-cli (cloud-code envelope w1-f), vertex + vertex-partner (SA-JSON dynamic URL) | 003,012,017 | `executors/gemini-cli.js`, `executors/vertex.js` | w2-a,b,c + w1-e/f |
| **w2-g** | Responses-format executors: codex (chatgpt backend, openai-responses w1-g) + github copilot (responsesUrl, copilot headers) | 016,021 | `executors/codex.js`, `executors/github.js` | w2-a,b,c + w1-g |
| **w2-h** | Event-stream/binary executors: kiro (AWS eventstream parse, w1-i) + cursor (connect-rpc protobuf + checksum — the `cursorProtobuf.js`/`cursorChecksum.js` deferred from w1-j land HERE) | 022,023 | `executors/kiro.js`, `executors/cursor.js`, `utils/cursorProtobuf.js`, `utils/cursorChecksum.js` | w2-a,b,c + w1-i/j |
| **w2-i** | Ollama + commandcode + antigravity executors (NDJSON / `/alpha/generate` / cloud-code envelope w1-f); ollama-local host resolution | 010,020,040 | `executors/ollama-local.js`, `executors/commandcode.js`, `executors/antigravity.js` | w2-a,b,c + w1-h/f |
| **w2-j** | Niche/reverse-engineered executors: azure (dynamic URL), qwen/iflow/qoder/opencode/opencode-go/xiaomi-tokenplan, grok-web/perplexity-web (cookie auth); decision-5 smoke-test hooks scaffolded | 018,019,028,030,031,032,047,048,049 | `executors/{azure,qwen,iflow,qoder,opencode,opencode-go,xiaomi-tokenplan,grok-web,perplexity-web}.js` | w2-a,b,c + w1-j |
| **w2-k** | Static-catalog media/embedding providers + Bifrost stub audit: image/audio/embedding providers as static catalogs; verify typed not-implemented stubs across ALL providers (decision 9) | 058-066 (fal-ai,stability-ai,black-forest-labs,recraft,runwayml,sdwebui,comfyui,huggingface,voyage-ai) | g0router-audit static catalogs | w2-a,c,d |
| **w2-l** | Free-tier + remaining config providers + validation route: ~30 free-tier entries (default executor; authHeader variants: enally x-api-key, uncloseai/opencode noAuth), `vertex-partner`/`cloudflare-ai`/`gitlab`/`codebuddy` config; provider validation endpoint | 011,013,025-027,033,035,037-039,050-052,067(free-tier) | `config/providers.js` free-tier block :404-437 | w2-a,b,c,d |

## Ownership & concurrency

- w2-a/b/c/d are the **serial foundation** (each depends on the prior; all touch
  `internal/providers/catalog` + `internal/inference/router`). Author + gate + implement
  in order a→b→c→d.
- w2-e..w2-l are **provider-family plans** with disjoint executor files; they share
  `router`/`registry` wiring → use the Wave-1 dispatch-order-gate pattern (each impl
  job verifies prior family registrations are merged before editing the registry).
- PR ports (~25): absorbed per-family in the plan that owns that provider; each plan's
  Rows header lists the PAR-PR rows it ports.

## Plan factory protocol (per micro-plan — same as Wave 1)

1. Fable 5 writes `plans/w2-<slug>.md`: cited PAR-PROV/PR rows + frozen-ref file:line,
   precondition greps, exclusive file ownership, TDD tasks (failing test first),
   binary acceptance, out-of-scope, Wave-3/4/5 handoff notes.
2. gpt-5.5 plan gate (`run-critic.sh plan`); max 3 reject cycles then decide.
3. kimi/M3 implements verbatim; deviations → plan amendment.
4. Gates (`go test ./...`, `go vet ./...`).
5. gpt-5.5 scoped diff gate; REJECT → fix loop or rebut (ref-cited).
6. Merge to main, flip PAR-PROV rows HAVE, update WORKFLOW.md.

## Out of Wave-2 scope (explicit)

- OAuth device-code/token-refresh/session hardening → **Wave 3** (the executor
  `refresh*` paths are stubbed here).
- Combo chains / fallback / rate-limit rotation / model aliases routing logic →
  **Wave 4** (w2-c does single-provider catalog resolution only).
- Usage accounting / cost / token counting → **Wave 5**.
- The w4-pre request-pipeline helpers (stripContentTypes/dedupeTools/
  injectReasoningContent — PAR-TRANS-006/051/052/053) stay in Wave 4.
- Live smoke CI execution → **Wave 8** (w2-j only scaffolds the hooks).

## Sizing

67 PAR-PROV rows + ~25 PR ports across 12 micro-plans. Foundation (a-d) is ~4 plans;
provider families (e-l) ~8. Plans authored in dependency order so each absorbs prior
learnings (catalog shape settles in w2-a, executor contract in w2-b).
