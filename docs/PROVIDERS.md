# Provider Matrix

`internal/provider/matrix.go` is the source of truth for provider status. Public APIs and CLI provider lists must derive from that matrix instead of maintaining separate handwritten lists.

Status meanings:

- `supported`: public inference works through normal `g0router serve` routing today.
- `adapter_only`: an adapter is registered in startup, but public support is still limited by missing model catalog entries, provider capability routing, authentication, streaming, tooling, or other documented gaps.
- `auth_only`: credential capture exists, but no inference adapter is wired.
- `unsupported`: explicitly not implemented; do not advertise as usable.

Current public direct-dispatch providers are `openai`, `anthropic`, `azure`, `bedrock`, `cerebras`, `cohere`, `deepseek`, `fireworks`, `gemini`, `github-copilot`, `groq`, `huggingface`, `litellm`, `lm-studio`, `mistral`, `minimax`, `nebius`, `nvidia`, `ollama`, `openrouter`, `perplexity`, `qwen`, `together`, `vercel-ai-gateway`, `vertex`, `vllm`, and `xai`. Deployment-defined providers without a static catalog use provider-qualified model names such as `azure/<deployment>`, `github-copilot/<model>`, or `vllm/<served-model>`.

## Public Surfaces

- `GET /api/providers` returns the full matrix with status and capability fields.
- `g0router providers list` prints only public direct-dispatch providers.
- `g0router auth list` prints providers with an auth flow, including `auth_only` providers.
- `/api/connections` lists stored credentials for every provider, including auth-only rows, but that does not imply inference support.

## Matrix

| g0router ID | OMP ID | 9Router ID | Bifrost ID | Status | Auth | Refresh | Adapter | Public inference | Streaming | Catalog | List models | Notes |
|-------------|--------|------------|------------|--------|------|---------|---------|------------------|-----------|---------|-------------|-------|
| `alibaba` | `alibaba` | `alibaba` | `alibaba` | `auth_only` | API key | no | no | no | no | no | no | Direct key capture exists; no Alibaba inference adapter. |
| `anthropic` | `anthropic` | `anthropic` | `anthropic` | `supported` | API key, OAuth | yes | yes | yes | yes | yes | yes | Claude adapter is public-routable; quota fetcher is not implemented. |
| `antigravity` | `antigravity` | `antigravity` | `antigravity` | `auth_only` | OAuth | yes | no | no | no | no | no | Google OAuth credential flow exists; dispatch is not a separate Antigravity provider. |
| `azure` | `azure` | `azure` | `azure` | `supported` | API key | no | yes | yes | yes | no | yes | Provider-qualified model IDs such as `azure/<deployment>` route through the Azure adapter; quota fetcher is not implemented. |
| `bedrock` | `bedrock` | `bedrock` | `bedrock` | `supported` | API key/AWS material | no | yes | yes | no | yes | yes | `anthropic.claude-3-5-haiku-20241022-v1:0` routes through non-streaming Bedrock Converse; streaming and quota are not implemented. |
| `cerebras` | `cerebras` | `cerebras` | `cerebras` | `supported` | API key | no | yes | yes | yes | yes | yes | Cataloged model IDs route through the OpenAI-compatible adapter; quota fetcher is not implemented. |
| `cloudflare-ai-gateway` | `cloudflare-ai-gateway` | `cloudflare-ai-gateway` | `cloudflare-ai-gateway` | `unsupported` | none | no | no | no | no | no | no | No gateway adapter. |
| `cohere` | `cohere` | `cohere` | `cohere` | `supported` | API key | no | yes | yes | yes | yes | yes | `command-r-08-2024` routes through the OpenAI-compatible adapter; quota fetcher is not implemented. |
| `cursor` | `cursor` | `cursor` | `cursor` | `auth_only` | OAuth | yes | no | no | no | no | no | OAuth exists; no Cursor inference adapter. |
| `deepseek` | `deepseek` | `deepseek` | `deepseek` | `supported` | API key, OAuth | yes | yes | yes | yes | yes | yes | Cataloged model IDs route through the OpenAI-compatible adapter; quota fetcher is not implemented. |
| `fireworks` | `fireworks` | `fireworks` | `fireworks` | `supported` | API key | no | yes | yes | yes | yes | yes | Cataloged model IDs route through the OpenAI-compatible adapter; quota fetcher is not implemented. |
| `gemini` | `gemini` | `gemini` | `gemini` | `supported` | API key, OAuth | yes | yes | yes | yes | yes | yes | Cataloged Gemini model IDs route through the native Gemini adapter, including native SSE streaming. |
| `github-copilot` | `github-copilot` | `github-copilot` | `github-copilot` | `supported` | OAuth | yes | yes | yes | yes | no | yes | OMP-style OpenAI-compatible dispatch works through provider-qualified model IDs such as `github-copilot/gpt-4o`; static catalog and quota fetcher are not implemented. |
| `gitlab` | `gitlab` | `gitlab` | `gitlab` | `auth_only` | OAuth | yes | no | no | no | no | no | GitLab-style OAuth exists; no inference adapter. |
| `groq` | `groq` | `groq` | `groq` | `supported` | API key | no | yes | yes | yes | yes | yes | Cataloged model IDs route through the OpenAI-compatible adapter; quota fetcher is not implemented. |
| `huggingface` | `huggingface` | `huggingface` | `huggingface` | `supported` | API key | no | yes | yes | yes | yes | yes | Provider-suffixed Hugging Face router model IDs route through the OpenAI-compatible adapter; quota fetcher is not implemented. |
| `kagi` | `kagi` | `kagi` | `kagi` | `unsupported` | none | no | no | no | no | no | no | No Kagi tool/search integration. |
| `kilo` | `kilo` | `kilo` | `kilo` | `unsupported` | none | no | no | no | no | no | no | No Kilo provider integration. |
| `kimi` | `kimi` | `kimi` | `kimi` | `auth_only` | OAuth | yes | no | no | no | no | no | Device-code OAuth exists; no Moonshot/Kimi inference adapter. |
| `kiro` | `kiro` | `kiro` | `kiro` | `auth_only` | OAuth | yes | no | no | no | no | no | OAuth exists; no Kiro inference adapter. |
| `litellm` | `litellm` | `litellm` | `litellm` | `supported` | API key | no | yes | yes | yes | no | yes | Provider-qualified model IDs such as `litellm/<model>` route through the OpenAI-compatible adapter; quota fetcher is not implemented. |
| `lm-studio` | `lm-studio` | `lm-studio` | `lm-studio` | `supported` | API key | no | yes | yes | yes | no | yes | Provider-qualified model IDs such as `lm-studio/<loaded-model>` route through the OpenAI-compatible adapter; quota fetcher is not implemented. |
| `minimax` | `minimax` | `minimax` | `minimax` | `supported` | API key | no | yes | yes | yes | yes | yes | `MiniMax-M3` routes through the OpenAI-compatible adapter; quota fetcher is not implemented. |
| `mistral` | `mistral` | `mistral` | `mistral` | `supported` | API key | no | yes | yes | yes | yes | yes | Cataloged model IDs route through the OpenAI-compatible adapter; quota fetcher is not implemented. |
| `nebius` | `nebius` | `nebius` | `nebius` | `supported` | API key | no | yes | yes | yes | yes | yes | `meta-llama/Llama-3.3-70B-Instruct` routes through the OpenAI-compatible Token Factory adapter; quota fetcher is not implemented. |
| `nvidia` | `nvidia` | `nvidia` | `nvidia` | `supported` | API key | no | yes | yes | yes | yes | yes | `meta/llama-3.1-8b-instruct` routes through the OpenAI-compatible NVIDIA NIM adapter; quota fetcher is not implemented. |
| `ollama` | `ollama` | `ollama` | `ollama` | `supported` | none | no | yes | yes | yes | yes | yes | Local no-auth cataloged model IDs route through the OpenAI-compatible adapter; hosted quota does not apply. |
| `ollama-cloud` | `ollama-cloud` | `ollama-cloud` | `ollama-cloud` | `unsupported` | none | no | no | no | no | no | no | Only local Ollama adapter exists. |
| `opencode` | `opencode` | `opencode` | `opencode` | `unsupported` | none | no | no | no | no | no | no | No OpenCode provider integration. |
| `openai` | `openai/codex` | `openai` | `openai` | `supported` | API key, OAuth | yes | yes | yes | yes | yes | yes | Codex OAuth stores as runtime provider `openai`; OpenAI is public-routable for `gpt-*` models; quota fetcher is not implemented. |
| `openrouter` | `openrouter` | `openrouter` | `openrouter` | `supported` | API key | no | yes | yes | yes | yes | yes | Cataloged model IDs route through the OpenAI-compatible adapter; quota fetcher is not implemented. |
| `perplexity` | `perplexity` | `perplexity` | `perplexity` | `supported` | API key | no | yes | yes | yes | yes | yes | Cataloged model IDs route through the OpenAI-compatible adapter; quota fetcher is not implemented. |
| `qianfan` | `qianfan` | `qianfan` | `qianfan` | `unsupported` | none | no | no | no | no | no | no | No Baidu Qianfan auth or inference adapter. |
| `qwen` | `qwen` | `qwen` | `qwen` | `supported` | API key | no | yes | yes | yes | yes | yes | Cataloged Qwen Cloud model IDs route through the OpenAI-compatible adapter; quota fetcher is not implemented. |
| `replicate` | `replicate` | `replicate` | `replicate` | `adapter_only` | API key | no | yes | no | yes | no | yes | OpenAI-compatible wrapper exists; public routing stays disabled until Replicate public semantics are proven against the expected API behavior. |
| `tavily` | `tavily` | `tavily` | `tavily` | `unsupported` | none | no | no | no | no | no | no | No Tavily tool/search integration. |
| `together` | `together` | `together` | `together` | `supported` | API key | no | yes | yes | yes | yes | yes | Cataloged model IDs route through the OpenAI-compatible adapter; quota fetcher is not implemented. |
| `vercel-ai-gateway` | `vercel-ai-gateway` | `vercel-ai-gateway` | `vercel-ai-gateway` | `supported` | API key | no | yes | yes | yes | yes | yes | `anthropic/claude-sonnet-4.5` routes through the OpenAI-compatible AI Gateway adapter; quota fetcher is not implemented. |
| `vertex` | `vertex` | `vertex` | `vertex` | `supported` | OAuth/service account | yes | yes | yes | yes | yes | yes | Provider-qualified catalog IDs such as `vertex/gemini-2.5-flash` route through the native Vertex adapter when `VERTEX_PROJECT_ID` and `VERTEX_LOCATION` are configured; Vertex auth uses the Gemini OAuth flow but persists runtime provider `vertex`; quota fetcher is not implemented. |
| `vllm` | `vllm` | `vllm` | `vllm` | `supported` | API key | no | yes | yes | yes | no | yes | Provider-qualified model IDs such as `vllm/<served-model>` route through the OpenAI-compatible adapter; quota fetcher is not implemented. |
| `xai` | `xai` | `xai` | `xai` | `supported` | API key, OAuth | yes | yes | yes | yes | yes | yes | Cataloged Grok model IDs route through the OpenAI-compatible adapter; quota fetcher is not implemented. |
| `xiaomi` | `xiaomi` | `xiaomi` | `xiaomi` | `auth_only` | OAuth | yes | no | no | no | no | no | OAuth exists; no Xiaomi inference adapter. |
| `zhipu` | `zhipu` | `zhipu` | `zhipu` | `auth_only` | API key | no | no | no | no | no | no | API-key capture exists; no ZAI/Zhipu inference adapter. |

## Model Routing Caveat

Current dispatch resolves stored model aliases first, then exact model catalog matches, then provider-qualified dynamic routes for deployment-defined providers, then legacy `gpt-*` and `claude-*` prefixes. Exact catalog matches provide public routing for OpenAI, Anthropic, Bedrock, Cerebras, Cohere, DeepSeek, Fireworks, Gemini, Groq, Hugging Face, Mistral, MiniMax, Nebius, NVIDIA, Ollama, OpenRouter, Perplexity, Qwen, Together, Vercel AI Gateway, Vertex, and xAI model IDs when a matching active connection exists. Provider-qualified dynamic routes strip the provider prefix before upstream dispatch for Azure, GitHub Copilot, LiteLLM, LM Studio, and vLLM, so `azure/gpt-4o-prod` reaches Azure as deployment `gpt-4o-prod` and `github-copilot/gpt-4o` reaches Copilot as `gpt-4o` without adding fake static pricing. Bedrock catalog routing is limited to documented non-streaming Converse models such as `anthropic.claude-3-5-haiku-20241022-v1:0`. Vertex catalog routes are provider-qualified (`vertex/gemini-*`) and are rewritten to upstream Gemini model IDs before dispatch so unqualified `gemini-*` remains owned by the Gemini adapter. Explicit aliases can target registered adapter providers only when the provider matrix marks inference capability true, `combo/*` routes use the same dispatch path, and request logging uses dispatch metadata when available. Quota fetchers are intentionally unsupported for providers whose matrix `quota` field is `false`; those routes fail open rather than fabricating provider quota data.
