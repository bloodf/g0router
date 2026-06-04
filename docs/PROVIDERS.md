# Provider Matrix

`internal/provider/matrix.go` is the source of truth for provider status. Public APIs and CLI provider lists must derive from that matrix instead of maintaining separate handwritten lists.

Status meanings:

- `supported`: public inference works through normal `g0router serve` routing today.
- `adapter_only`: an adapter is registered in startup, but public support is still limited by missing model catalog entries, provider capability routing, authentication, streaming, tooling, or other documented gaps.
- `auth_only`: credential capture exists, but no inference adapter is wired.
- `unsupported`: explicitly not implemented; do not advertise as usable.

Current public direct-dispatch providers are `openai`, `anthropic`, `cerebras`, `deepseek`, `fireworks`, `groq`, `mistral`, `minimax`, `ollama`, `openrouter`, `perplexity`, `qwen`, `together`, and `xai`. Adapter-only providers with matrix `Inference=true` may be reached only through explicit aliases or `combo/*` routes; providers with `Inference=false`, including `bedrock`, cannot be routed.

## Public Surfaces

- `GET /api/providers` returns the full matrix with status and capability fields.
- `g0router providers list` prints only public direct-dispatch providers.
- `g0router auth list` prints providers with an auth flow, including `auth_only` providers.
- `/api/connections` lists stored credentials for every provider, including auth-only rows, but that does not imply inference support.

## Matrix

| g0router ID | OMP ID | 9Router ID | Bifrost ID | Status | Auth | Refresh | Adapter | Public inference | Streaming | Catalog | List models | Notes |
|-------------|--------|------------|------------|--------|------|---------|---------|------------------|-----------|---------|-------------|-------|
| `alibaba` | `alibaba` | `alibaba` | `alibaba` | `auth_only` | API key | no | no | no | no | no | no | Direct key capture exists; no Alibaba inference adapter. |
| `anthropic` | `anthropic` | `anthropic` | `anthropic` | `supported` | API key, OAuth | yes | yes | yes | yes | yes | yes | Claude adapter is public-routable. |
| `antigravity` | `antigravity` | `antigravity` | `antigravity` | `auth_only` | OAuth | yes | no | no | no | no | no | Google OAuth credential flow exists; dispatch is not a separate Antigravity provider. |
| `azure` | `azure` | `azure` | `azure` | `adapter_only` | API key | no | yes | no | yes | no | yes | Registered adapter, but no ordinary model-name routing yet. |
| `bedrock` | `bedrock` | `bedrock` | `bedrock` | `adapter_only` | API key/AWS material | no | yes | no | no | no | no | Registered adapter does not implement Bedrock Converse, streaming, model catalog/ListModels, quota, or public direct dispatch. |
| `cerebras` | `cerebras` | `cerebras` | `cerebras` | `supported` | API key | no | yes | yes | yes | yes | yes | Cataloged model IDs route through the OpenAI-compatible adapter; quota fetcher is not implemented. |
| `cloudflare-ai-gateway` | `cloudflare-ai-gateway` | `cloudflare-ai-gateway` | `cloudflare-ai-gateway` | `unsupported` | none | no | no | no | no | no | no | No gateway adapter. |
| `cohere` | `cohere` | `cohere` | `cohere` | `adapter_only` | API key | no | yes | no | yes | no | yes | OpenAI-compatible wrapper exists; not public-routable yet. |
| `cursor` | `cursor` | `cursor` | `cursor` | `auth_only` | OAuth | yes | no | no | no | no | no | OAuth exists; no Cursor inference adapter. |
| `deepseek` | `deepseek` | `deepseek` | `deepseek` | `supported` | API key, OAuth | yes | yes | yes | yes | yes | yes | Cataloged model IDs route through the OpenAI-compatible adapter; quota fetcher is not implemented. |
| `fireworks` | `fireworks` | `fireworks` | `fireworks` | `supported` | API key | no | yes | yes | yes | yes | yes | Cataloged model IDs route through the OpenAI-compatible adapter; quota fetcher is not implemented. |
| `gemini` | `gemini` | `gemini` | `gemini` | `adapter_only` | API key, OAuth | yes | yes | no | no | no | yes | Gemini adapter exists; no streaming or public routing yet. |
| `github-copilot` | `github-copilot` | `github-copilot` | `github-copilot` | `auth_only` | OAuth | yes | no | no | no | no | no | Device-code OAuth exists; no Copilot inference adapter. |
| `gitlab` | `gitlab` | `gitlab` | `gitlab` | `auth_only` | OAuth | yes | no | no | no | no | no | GitLab-style OAuth exists; no inference adapter. |
| `groq` | `groq` | `groq` | `groq` | `supported` | API key | no | yes | yes | yes | yes | yes | Cataloged model IDs route through the OpenAI-compatible adapter; quota fetcher is not implemented. |
| `huggingface` | `huggingface` | `huggingface` | `huggingface` | `adapter_only` | API key | no | yes | no | yes | no | yes | OpenAI-compatible adapter, but no public routing yet. |
| `kagi` | `kagi` | `kagi` | `kagi` | `unsupported` | none | no | no | no | no | no | no | No Kagi tool/search integration. |
| `kilo` | `kilo` | `kilo` | `kilo` | `unsupported` | none | no | no | no | no | no | no | No Kilo provider integration. |
| `kimi` | `kimi` | `kimi` | `kimi` | `auth_only` | OAuth | yes | no | no | no | no | no | Device-code OAuth exists; no Moonshot/Kimi inference adapter. |
| `kiro` | `kiro` | `kiro` | `kiro` | `auth_only` | OAuth | yes | no | no | no | no | no | OAuth exists; no Kiro inference adapter. |
| `litellm` | `litellm` | `litellm` | `litellm` | `unsupported` | none | no | no | no | no | no | no | No LiteLLM gateway adapter. |
| `lm-studio` | `lm-studio` | `lm-studio` | `lm-studio` | `unsupported` | none | no | no | no | no | no | no | No LM Studio adapter. |
| `minimax` | `minimax` | `minimax` | `minimax` | `supported` | API key | no | yes | yes | yes | yes | yes | `MiniMax-M3` routes through the OpenAI-compatible adapter; quota fetcher is not implemented. |
| `mistral` | `mistral` | `mistral` | `mistral` | `supported` | API key | no | yes | yes | yes | yes | yes | Cataloged model IDs route through the OpenAI-compatible adapter; quota fetcher is not implemented. |
| `nebius` | `nebius` | `nebius` | `nebius` | `adapter_only` | API key | no | yes | no | yes | no | yes | OpenAI-compatible adapter, but no public routing yet. |
| `nvidia` | `nvidia` | `nvidia` | `nvidia` | `adapter_only` | API key | no | yes | no | yes | no | yes | OpenAI-compatible adapter, but no public routing yet. |
| `ollama` | `ollama` | `ollama` | `ollama` | `supported` | none | no | yes | yes | yes | yes | yes | Local no-auth cataloged model IDs route through the OpenAI-compatible adapter; hosted quota does not apply. |
| `ollama-cloud` | `ollama-cloud` | `ollama-cloud` | `ollama-cloud` | `unsupported` | none | no | no | no | no | no | no | Only local Ollama adapter exists. |
| `opencode` | `opencode` | `opencode` | `opencode` | `unsupported` | none | no | no | no | no | no | no | No OpenCode provider integration. |
| `openai` | `openai/codex` | `openai` | `openai` | `supported` | API key, OAuth | yes | yes | yes | yes | yes | yes | Codex OAuth stores as runtime provider `openai`; OpenAI is public-routable for `gpt-*` models. |
| `openrouter` | `openrouter` | `openrouter` | `openrouter` | `supported` | API key | no | yes | yes | yes | yes | yes | Cataloged model IDs route through the OpenAI-compatible adapter; quota fetcher is not implemented. |
| `perplexity` | `perplexity` | `perplexity` | `perplexity` | `supported` | API key | no | yes | yes | yes | yes | yes | Cataloged model IDs route through the OpenAI-compatible adapter; quota fetcher is not implemented. |
| `qianfan` | `qianfan` | `qianfan` | `qianfan` | `unsupported` | none | no | no | no | no | no | no | No Baidu Qianfan auth or inference adapter. |
| `qwen` | `qwen` | `qwen` | `qwen` | `supported` | API key | no | yes | yes | yes | yes | yes | Cataloged Qwen Cloud model IDs route through the OpenAI-compatible adapter; quota fetcher is not implemented. |
| `replicate` | `replicate` | `replicate` | `replicate` | `adapter_only` | API key | no | yes | no | yes | no | yes | OpenAI-compatible wrapper exists; not public-routable yet. |
| `tavily` | `tavily` | `tavily` | `tavily` | `unsupported` | none | no | no | no | no | no | no | No Tavily tool/search integration. |
| `together` | `together` | `together` | `together` | `supported` | API key | no | yes | yes | yes | yes | yes | Cataloged model IDs route through the OpenAI-compatible adapter; quota fetcher is not implemented. |
| `vercel-ai-gateway` | `vercel-ai-gateway` | `vercel-ai-gateway` | `vercel-ai-gateway` | `unsupported` | none | no | no | no | no | no | no | No Vercel AI Gateway adapter. |
| `vertex` | `vertex` | `vertex` | `vertex` | `adapter_only` | OAuth/service account | yes | yes | no | no | no | yes | Vertex adapter exists; no streaming or public routing yet. |
| `vllm` | `vllm` | `vllm` | `vllm` | `unsupported` | none | no | no | no | no | no | no | No configurable self-hosted vLLM adapter. |
| `xai` | `xai` | `xai` | `xai` | `supported` | API key, OAuth | yes | yes | yes | yes | yes | yes | Cataloged Grok model IDs route through the OpenAI-compatible adapter; quota fetcher is not implemented. |
| `xiaomi` | `xiaomi` | `xiaomi` | `xiaomi` | `auth_only` | OAuth | yes | no | no | no | no | no | OAuth exists; no Xiaomi inference adapter. |
| `zhipu` | `zhipu` | `zhipu` | `zhipu` | `auth_only` | API key | no | no | no | no | no | no | API-key capture exists; no ZAI/Zhipu inference adapter. |

## Model Routing Caveat

Current dispatch resolves stored model aliases first, then exact model catalog matches, then legacy `gpt-*` and `claude-*` prefixes. Exact catalog matches provide public routing for OpenAI, Anthropic, Cerebras, DeepSeek, Fireworks, Groq, Mistral, MiniMax, Ollama, OpenRouter, Perplexity, Qwen, Together, and xAI model IDs when a matching active connection exists. Explicit aliases can target registered adapter providers only when the provider matrix marks inference capability true, `combo/*` routes use the same dispatch path, and request logging uses dispatch metadata when available. Broader provider capability routing and expanded quota/cost coverage remain Wave 7.I work.
