# Provider Catalog

Complete reference for all upstream LLM providers supported by g0router.

## Wire Format Groups

g0router translates between these wire formats transparently:

| Format | Endpoint Pattern | Content-Type | Streaming |
|--------|-----------------|--------------|-----------|
| **OpenAI Chat** | `POST /v1/chat/completions` | `application/json` | SSE: `data: {chunk}\n\ndata: [DONE]\n\n` |
| **OpenAI Responses** | `POST /v1/responses` | `application/json` | SSE with typed events |
| **Anthropic Messages** | `POST /v1/messages` | `application/json` | SSE: `event: type\ndata: {chunk}\n\n` |
| **Gemini** | `POST /v1beta/models/{model}:generateContent` | `application/json` | SSE via `?alt=sse` |
| **Bedrock** | `POST /model/{id}/converse` | `application/json` | Chunk-based streaming |

## Provider Details

### Tier 1: Primary Providers (custom implementations)

#### OpenAI
- **Base URL**: `https://api.openai.com`
- **Auth**: `Authorization: Bearer sk-...` (API key) or OAuth token (Codex)
- **Wire format**: OpenAI Chat + Responses API
- **Models**: gpt-4o, gpt-4o-mini, gpt-4.1, gpt-4.1-mini, gpt-4.1-nano, o3, o3-mini, o4-mini
- **OAuth**: Device-code flow (Codex CLI)
  - Client ID: `DQ1Ij3iIOC1S0aQCBk5KFj9m4gQZLrIf`
  - Token URL: `https://auth.openai.com/oauth/token`
- **Special**: Supports `stream_options.include_usage` for streaming usage

#### Anthropic
- **Base URL**: `https://api.anthropic.com`
- **Auth**: `x-api-key: sk-ant-...` (API key) or `Authorization: Bearer ...` (OAuth)
- **Wire format**: Anthropic Messages API
- **Models**: claude-sonnet-4-20250514, claude-3-5-haiku-20241022, claude-3-5-sonnet-20241022, claude-3-opus-20240229
- **OAuth**: PKCE + callback
  - Client ID: `9d6bc642-e7b0-4445-8dab-bd2d0804e37c`
  - Auth URL: `https://console.anthropic.com/oauth/authorize`
  - Token URL: `https://api.anthropic.com/v1/oauth/token`
  - Callback: `http://localhost:54545/oauth/callback`
  - Scope: `org:create_api_key`
- **Special**: System as top-level field, thinking blocks, cache tokens, content blocks

#### Gemini
- **Base URL**: `https://generativelanguage.googleapis.com`
- **Auth**: `?key=AIza...` (API key) or OAuth token
- **Wire format**: Gemini generateContent
- **Models**: gemini-2.5-pro, gemini-2.5-flash, gemini-2.0-flash, gemini-1.5-pro
- **OAuth**: OAuth2 + callback (Gemini CLI)
  - Auth URL: `https://accounts.google.com/o/oauth2/v2/auth`
  - Token URL: `https://oauth2.googleapis.com/token`
- **Special**: `system_instruction` field, `parts[]` content model

### Tier 2: OpenAI-Compatible Providers

All use identical OpenAI wire format. Only base URL, auth, and available models differ.

| Provider | ID | Base URL | Auth | Notable Models |
|----------|-----|----------|------|----------------|
| Groq | `groq` | `https://api.groq.com/openai` | Bearer | llama-3.3-70b-versatile, mixtral-8x7b |
| Cerebras | `cerebras` | `https://api.cerebras.ai` | Bearer | llama3.1-70b, llama3.1-8b |
| Perplexity | `perplexity` | `https://api.perplexity.ai` | Bearer | sonar-pro, sonar |
| Fireworks | `fireworks` | `https://api.fireworks.ai/inference` | Bearer | llama-v3p3-70b-instruct |
| Together | `together` | `https://api.together.xyz` | Bearer | meta-llama/Llama-3.3-70B |
| NVIDIA | `nvidia` | `https://integrate.api.nvidia.com` | Bearer | nvidia/llama-3.1-nemotron-70b |
| DeepSeek | `deepseek` | `https://api.deepseek.com` | Bearer | deepseek-chat, deepseek-reasoner |
| OpenRouter | `openrouter` | `https://openrouter.ai/api` | Bearer | Any (aggregator) |
| HuggingFace | `huggingface` | `https://api-inference.huggingface.co` | Bearer | Model-specific URL |
| Nebius | `nebius` | `https://api.studio.nebius.ai` | Bearer | Various |
| vLLM | `vllm` | User-configured | Bearer/none | Self-hosted |
| SGL | `sgl` | User-configured | Bearer/none | Self-hosted |
| Parasail | `parasail` | `https://api.parasail.io` | Bearer | Various |

### Tier 3: Cloud Platform Providers

| Provider | ID | Auth | Notes |
|----------|-----|------|-------|
| Vertex AI | `vertex` | GCP OAuth/service account | Same wire format as Gemini, different URL |
| AWS Bedrock | `bedrock` | SigV4 (AWS credentials) | `converse` API, model-specific routing |
| Azure OpenAI | `azure` | `api-key` header | Deployment-based URL, API version parameter |

### Tier 4: Additional Providers

| Provider | ID | Auth | Notes |
|----------|-----|------|-------|
| Mistral | `mistral` | Bearer | OpenAI-compatible, `api.mistral.ai` |
| Ollama | `ollama` | None | OpenAI-compatible, `localhost:11434` |
| Cohere | `cohere` | Bearer | Dedicated `/v2/chat` format |
| Replicate | `replicate` | Bearer | Predictions API with polling |
| xAI | `xai` | Bearer / OAuth | `api.x.ai`, OAuth2 flow |

### Tier 5: Chinese / Regional Providers

| Provider | ID | Auth | OAuth Flow |
|----------|-----|------|------------|
| Qwen | `qwen` | OAuth | Provider-specific |
| Kimi (Moonshot) | `kimi` | OAuth | Device-code, custom polling |
| MiniMax | `minimax` | API key / OAuth | API key or OAuth |
| Alibaba | `alibaba` | API key | Direct key |
| Zhipu (GLM) | `zhipu` | API key | Direct key |
| Xiaomi | `xiaomi` | OAuth | Standard OAuth2 |

### Tier 6: IDE Proxy Providers

| Provider | ID | Auth | Notes |
|----------|-----|------|-------|
| GitHub Copilot | `github-copilot` | OAuth | Device-code, session token refresh every 30min |
| Cursor | `cursor` | OAuth | PKCE + polling, dual token (accessToken + authToken) |
| GitLab Duo | `gitlab` | OAuth | Standard OAuth2 |
| Kiro | `kiro` | OAuth | AWS-backed |

## OAuth Flow Summary

| Flow Type | Providers | User Experience |
|-----------|-----------|-----------------|
| **PKCE + Callback** | Anthropic, Cursor, Perplexity | Browser opens → authorize → redirect to localhost |
| **Device Code** | OpenAI/Codex, GitHub Copilot, Kimi | Terminal shows code → user enters at URL → polls |
| **OAuth2 + Callback** | Gemini, Antigravity, xAI, GitLab, Xiaomi | Browser opens → authorize → redirect to localhost |
| **Password** | DeepSeek | Terminal prompts email + password |
| **API Key** | Most providers | User pastes key from provider dashboard |

## Combo Models

User-defined fallback chains (stored in SQLite `combos` table):

```json
{
    "name": "fast-fallback",
    "steps": [
        {"provider": "groq", "model": "llama-3.3-70b-versatile"},
        {"provider": "cerebras", "model": "llama3.1-70b"},
        {"provider": "openai", "model": "gpt-4o-mini"}
    ]
}
```

Request to `model: "combo/fast-fallback"`:
1. Try Groq → if 429/error → try Cerebras → if error → try OpenAI
2. First success wins. Last error returned if all fail.

## Model Aliases

SQLite `model_aliases` table maps shorthand names:

```sql
INSERT INTO model_aliases VALUES ('fast', 'groq', 'llama-3.3-70b-versatile');
INSERT INTO model_aliases VALUES ('smart', 'anthropic', 'claude-sonnet-4-20250514');
INSERT INTO model_aliases VALUES ('cheap', 'openai', 'gpt-4o-mini');
```

Client sends `model: "fast"` → resolves to `groq/llama-3.3-70b-versatile`.
