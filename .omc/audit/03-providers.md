# Provider Parity Audit — g0router

READ-ONLY source audit. Grumpy principal engineer hat on. Source-over-docs. Every claim has `file:line`.

## TL;DR

The matrix in `internal/provider/matrix.go` is unusually honest — it self-labels stubs as `auth_only` and documents non-streaming/non-catalog gaps. Cross-checked against actual registration (`internal/cli/provider_runtime.go`), the adapter interface (`internal/providers/interface.go`), the dispatch gate (`internal/proxy/engine.go`), the OAuth flow registry (`internal/cli/auth.go:257`), and the price catalog (`internal/modelcatalog/catalog.go`). Findings below distinguish what the matrix *advertises* from what the *source* actually does.

## How dispatch actually works (the gate)

1. Every adapter implements the same 3-method interface: `ChatCompletion`, `ChatCompletionStream`, `ListModels` — `internal/providers/interface.go:5`.
2. Registration is hardcoded in `newDefaultInferenceEngine` — `internal/cli/provider_runtime.go:28-53`, plus the OpenAI-compatible fan-out `registerOpenAICompatible` — `internal/cli/provider_runtime.go:68-100`.
3. Routing resolves alias → catalog → provider-qualified-dynamic → `gpt-*/claude-*` prefix — `internal/proxy/engine.go:292-317`.
4. Dispatch is **matrix-gated**: a route is rejected if the matrix `Inference` flag is false — `routableModelRoute` at `internal/proxy/engine.go:348-354`. So `auth_only` providers genuinely cannot dispatch.
5. Dynamic (catalog-less) routing is **allowlisted** to 16 providers in `providerQualifiedDynamicRoute` — `internal/proxy/engine.go:577-597`. A provider NOT in catalog AND NOT in this allowlist cannot route at all.

## FULL native API vs lowest-common-denominator

- **FULL native API**: `anthropic` (native Messages API, `internal/providers/anthropic/anthropic.go`), `gemini` (native `/v1beta`, native SSE — `internal/providers/gemini/gemini.go:94,128`), `vertex` (native publisher models — `internal/providers/vertex/vertex.go:99,136`), `openai` (native + Responses API — `internal/providers/openai/responses.go`), `bedrock` (native Converse — `internal/providers/bedrock/bedrock.go:77`), `ollama-cloud` (native `/api/chat`, `/api/tags` — `internal/providers/ollamacloud/ollamacloud.go:101,155`), `replicate` (native predictions create/poll — `internal/providers/replicate/replicate.go:77`), `xiaomi` (native Anthropic-compatible via the anthropic adapter — `internal/providers/xiaomi/xiaomi.go:29`), `gitlab-duo` (native Duo direct-access token exchange + AI Gateway proxy — `internal/providers/gitlabduo/gitlabduo.go:105,179`).
- **Lowest-common-denominator (OpenAI-compatible `/v1/chat/completions`)**: the entire `openaicompat` fleet — 24 providers fanned out from one implementation (`internal/providers/openaicompat/provider.go:96,120,152`) configured by base URL only (`internal/providers/openaicompat/registry.go:9-36`). This includes groq, cerebras, perplexity, fireworks, together, nvidia, deepseek, openrouter, huggingface, kimi, nebius, minimax, qwen, xai, vercel-ai-gateway, github-copilot, alibaba, qianfan, zhipu, litellm, vllm, lm-studio, opencode, kilo. `cohere` and `mistral` use their own `NewDefault()` wrappers that are also OpenAI-compatible shells. `cloudflare-ai-gateway` is a thin wrapper that builds an account-scoped `openaicompat.Provider` per request (`internal/providers/cloudflare/cloudflare.go:56-69`).

## OAuth / refresh reality

OAuth flows registered at `internal/cli/auth.go:257-282`: alibaba, anthropic, antigravity, codex, cursor, deepseek, gemini, github-copilot, gitlab-duo, kimi, kiro, minimax, qianfan, xai, xiaomi, zhipu.

`RefreshableFlow` requires a `Refresh()` method (`internal/provider/oauth/types.go:105-108`). Flows that actually implement `Refresh`: codex, gemini, gitlab, kiro, kimi, anthropic, xai, deepseek, xiaomi, cursor, antigravity, github (grep of `internal/provider/oauth/*.go`).

**Gap**: `alibaba`, `minimax`, `qianfan`, `zhipu` have OAuth login flows but **no `Refresh`** — matrix correctly marks their `Refresh=false` (e.g. matrix.go:71-73), so once the OAuth token expires the user must re-login. Honest but worth flagging.

## Status table

`auth` = credential capture exists. `models` = ListModels returns real data. `dispatch` = chat works & routing reaches it. `stream` = real streaming. `refresh` = OAuth refresh implemented (n/a if api-key only).

| provider | auth | models | dispatch | stream | refresh | status | evidence (file:line) |
|---|---|---|---|---|---|---|---|
| anthropic | yes | yes | yes | yes | yes | complete | anthropic.go:66,95,130; matrix.go:44 |
| openai (codex) | yes | yes | yes | yes | yes | complete | openai.go; responses.go; matrix.go:43; auth.go:262 |
| gemini | yes | yes | yes | yes | yes | complete | gemini.go:64,94,128; matrix.go:51 |
| vertex | yes | yes | yes | yes | yes(gemini flow) | complete | vertex.go:66,99,136; matrix.go:62 |
| azure | yes | yes | yes | yes | n/a | complete | azure.go:76,100,132; dynamic allowlist engine.go:579 |
| groq | yes | yes | yes | yes | n/a | complete | openaicompat registry.go:11; catalog.go:48 |
| cerebras | yes | yes | yes | yes | n/a | complete | registry.go:12; catalog.go:90 |
| perplexity | yes | yes | yes | yes | n/a | complete | registry.go:13; catalog.go:75 |
| fireworks | yes | yes | yes | yes | n/a | complete | registry.go:14; catalog.go:96 |
| together | yes | yes | yes | yes | n/a | complete | registry.go:15; catalog.go:101 |
| nvidia | yes | yes | yes | yes | n/a | complete | registry.go:16; catalog.go:64 |
| deepseek | yes | yes | yes | yes | yes | complete | registry.go:17; catalog.go:71; auth.go:264 |
| openrouter | yes | yes | yes | yes | n/a | complete (has quota) | registry.go:18; catalog.go:67; matrix.go:58 |
| huggingface | yes | yes | yes | yes | n/a | complete | registry.go:19; catalog.go:52 |
| kimi | yes | yes | yes | yes | yes | complete | registry.go:20; dynamic engine.go:584; auth.go:272 |
| nebius | yes | yes | yes | yes | n/a | complete | registry.go:21; catalog.go:61 |
| minimax | yes | yes | yes | yes | NO (oauth no refresh) | complete | registry.go:22; catalog.go:80; auth.go:274 |
| qwen | yes | yes | yes | yes | n/a | complete | registry.go:23; catalog.go:83 |
| xai | yes | yes | yes | yes | yes | complete | registry.go:24; catalog.go:87; auth.go:278 |
| vercel-ai-gateway | yes | yes | yes | yes | n/a | complete | registry.go:25; catalog.go:113 |
| github-copilot | yes | yes | yes | yes | yes | complete | registry.go:26; dynamic engine.go:581; auth.go:266 |
| alibaba | yes | yes | yes | yes | NO (oauth no refresh) | complete | registry.go:27; dynamic engine.go:578; auth.go:259 |
| qianfan | yes | yes(generic) | yes | yes | NO | complete | registry.go:28; dynamic engine.go:589; auth.go:275 |
| zhipu | yes | yes(generic) | yes | yes | NO | complete | registry.go:29; dynamic engine.go:593; auth.go:280 |
| litellm | yes | yes | yes | yes | n/a | complete (self-hosted) | registry.go:30; dynamic engine.go:585 |
| vllm | yes | yes | yes | yes | n/a | complete (self-hosted) | registry.go:31; dynamic engine.go:591 |
| lm-studio | yes | yes | yes | yes | n/a | complete (self-hosted) | registry.go:32; dynamic engine.go:586 |
| opencode | yes | NO (no list) | yes | yes | n/a | partial | registry.go:33; matrix.go:202; engine.go:588 |
| kilo | yes | NO (no list) | yes | yes | n/a | partial | registry.go:34; matrix.go:208; engine.go:583 |
| mistral | yes | yes | yes | yes | n/a | complete | provider_runtime.go:40; catalog.go:55 |
| cohere | yes | yes | yes | yes | n/a | complete | provider_runtime.go:43; catalog.go:93 |
| ollama | none | yes | yes | yes | n/a | complete (local) | provider_runtime.go:46; catalog.go:105 |
| ollama-cloud | yes | yes (native /api/tags) | yes | yes | n/a | complete | ollamacloud.go:101,113,155; engine.go:587 |
| replicate | yes | NO (stub) | yes (non-stream only) | NO (stub) | n/a | partial | replicate.go:114-120; engine.go:590 |
| bedrock | yes | yes | yes (non-stream only) | NO (stub) | n/a | partial | bedrock.go:109-111; matrix.go:46 |
| cloudflare-ai-gateway | yes | NO (no catalog) | yes (needs account_id) | yes | n/a | partial | cloudflare.go:56-60; matrix.go:190 |
| gitlab-duo | yes | yes (alias list) | yes | yes | yes | complete | gitlabduo.go:105,135,165; auth.go:269 |
| xiaomi | yes | yes (via anthropic) | yes | yes | yes | complete | xiaomi.go:29,42; auth.go:279 |
| antigravity | yes | NO | NO (gated off) | NO | yes | stub (auth_only) | matrix.go:63; auth.go:261; engine.go:350 |
| cursor | yes | NO | NO (gated off) | NO | yes | stub (auth_only) | matrix.go:65; cursor.go:78; auth.go:263 |
| kiro | yes | NO | NO (gated off) | NO | yes | stub (auth_only) | matrix.go:68; kiro.go:65; auth.go:273 |
| kagi | yes (api key) | NO | NO | NO | n/a | stub (auth_only, MCP tool only) | matrix.go:75 |
| tavily | yes (api key) | NO | NO | NO | n/a | stub (auth_only, MCP tool only) | matrix.go:83 |

## Notes / discrepancies worth a grumble

1. **`replicate` is advertised `supported` but is two-thirds stub.** `ChatCompletionStream` returns `ErrStreamingUnsupported` (`replicate.go:115`) and `ListModels` returns a bare `"replicate list models unsupported"` error (`replicate.go:119`). Only non-streaming predict-and-poll works. The matrix note (matrix.go:228) admits this, but the headline status is still `supported`, which oversells it.
2. **`bedrock` streaming is a hard stub** (`bedrock.go:110` returns `ErrStreamingUnsupported`) yet status is `supported`. Any client requesting a stream from Bedrock gets an error, not a fallback. Matrix note documents it (matrix.go:158); status header does not.
3. **`opencode` and `kilo` have no model listing** (`ListModels` not wired; matrix `ListModels=false` at matrix.go:203,209). Dispatch works via dynamic routing only, so a user has zero discovery — must know the exact `opencode/<model>` string.
4. **`cloudflare-ai-gateway` silently requires `account_id` on the stored connection** (`cloudflare.go:58-60`). Without it every call fails with `"cloudflare account id is required"`. No catalog, no list — pure blind dynamic dispatch.
5. **OAuth-without-refresh trap**: `alibaba`, `minimax`, `qianfan`, `zhipu` accept OAuth login but cannot refresh (no `Refresh` method). Tokens silently expire; re-login required. Matrix `Refresh=false` is correct but the UX implication is buried.
6. **`kagi`/`tavily` are not inference providers at all** — they back MCP search tools only (matrix.go:75,83). Correctly `auth_only`, but they pad the advertised provider count.
7. **Mistral/Cohere are OpenAI-compatible shells**, not native adapters, despite having dedicated packages (`provider_runtime.go:40,43`). Functionally fine; "native Mistral/Cohere API" is not what's happening.
8. Every dynamic-routed provider depends on the hardcoded allowlist `providerQualifiedDynamicRoute` (engine.go:577). Adding a provider to the matrix without adding it here = silently non-routable. Fragile coupling between two files.

## Counts

- Total entries in matrix: **43** (`internal/provider/matrix.go:42-86`).
- Fully functional (auth+models+dispatch+stream, allowing n/a refresh): **~30**.
- Partial (missing stream and/or model-listing, but dispatch works): **6** — replicate, bedrock, cloudflare-ai-gateway, opencode, kilo, (xiaomi/ollama-cloud are complete).
- Stub / auth-only (no dispatch): **5** — antigravity, cursor, kiro, kagi, tavily.
