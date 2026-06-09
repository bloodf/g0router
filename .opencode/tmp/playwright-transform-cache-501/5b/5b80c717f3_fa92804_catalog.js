// Mock catalog fallback mirroring internal/modelcatalog/catalog.go.
// Provides default models and pricing per provider before any connection is added.

const providerCatalog = {
  openai: [{
    id: "gpt-4o",
    provider: "openai",
    name: "gpt-4o",
    input_cost: 2.5,
    output_cost: 10.0,
    context_window: 128000,
    is_disabled: false,
    is_custom: false
  }, {
    id: "gpt-4o-mini",
    provider: "openai",
    name: "gpt-4o-mini",
    input_cost: 0.15,
    output_cost: 0.6,
    context_window: 128000,
    is_disabled: false,
    is_custom: false
  }],
  anthropic: [{
    id: "claude-sonnet-4",
    provider: "anthropic",
    name: "claude-sonnet-4",
    input_cost: 3.0,
    output_cost: 15.0,
    context_window: 200000,
    is_disabled: false,
    is_custom: false
  }, {
    id: "claude-opus-4",
    provider: "anthropic",
    name: "claude-opus-4",
    input_cost: 15.0,
    output_cost: 75.0,
    context_window: 200000,
    is_disabled: false,
    is_custom: false
  }, {
    id: "claude-3-5-haiku-20241022",
    provider: "anthropic",
    name: "claude-3-5-haiku-20241022",
    input_cost: 0.8,
    output_cost: 4.0,
    context_window: 200000,
    is_disabled: false,
    is_custom: false
  }],
  gemini: [{
    id: "gemini-2.5-flash",
    provider: "gemini",
    name: "gemini-2.5-flash",
    input_cost: 0.3,
    output_cost: 2.5,
    context_window: 1000000,
    is_disabled: false,
    is_custom: false
  }, {
    id: "gemini-2.5-flash-lite",
    provider: "gemini",
    name: "gemini-2.5-flash-lite",
    input_cost: 0.1,
    output_cost: 0.4,
    context_window: 1000000,
    is_disabled: false,
    is_custom: false
  }],
  groq: [{
    id: "llama-3.3-70b-versatile",
    provider: "groq",
    name: "llama-3.3-70b-versatile",
    input_cost: 0.59,
    output_cost: 0.79,
    context_window: 128000,
    is_disabled: false,
    is_custom: false
  }, {
    id: "llama-3.1-8b-instant",
    provider: "groq",
    name: "llama-3.1-8b-instant",
    input_cost: 0.05,
    output_cost: 0.08,
    context_window: 128000,
    is_disabled: false,
    is_custom: false
  }],
  mistral: [{
    id: "mistral-large-latest",
    provider: "mistral",
    name: "mistral-large-latest",
    input_cost: 2.0,
    output_cost: 6.0,
    context_window: 128000,
    is_disabled: false,
    is_custom: false
  }, {
    id: "mistral-small-latest",
    provider: "mistral",
    name: "mistral-small-latest",
    input_cost: 0.1,
    output_cost: 0.3,
    context_window: 128000,
    is_disabled: false,
    is_custom: false
  }, {
    id: "magistral-small-latest",
    provider: "mistral",
    name: "magistral-small-latest",
    input_cost: 0.5,
    output_cost: 1.5,
    context_window: 128000,
    is_disabled: false,
    is_custom: false
  }, {
    id: "ministral-8b-latest",
    provider: "mistral",
    name: "ministral-8b-latest",
    input_cost: 0.15,
    output_cost: 0.15,
    context_window: 128000,
    is_disabled: false,
    is_custom: false
  }],
  nebius: [{
    id: "meta-llama/Llama-3.3-70B-Instruct",
    provider: "nebius",
    name: "meta-llama/Llama-3.3-70B-Instruct",
    input_cost: 0.13,
    output_cost: 0.4,
    context_window: 128000,
    is_disabled: false,
    is_custom: false
  }],
  nvidia: [{
    id: "meta/llama-3.1-8b-instruct",
    provider: "nvidia",
    name: "meta/llama-3.1-8b-instruct",
    input_cost: 0.1,
    output_cost: 0.1,
    context_window: 128000,
    is_disabled: false,
    is_custom: false
  }],
  openrouter: [{
    id: "openai/gpt-4o",
    provider: "openrouter",
    name: "openai/gpt-4o",
    input_cost: 2.5,
    output_cost: 10.0,
    context_window: 128000,
    is_disabled: false,
    is_custom: false
  }, {
    id: "openai/gpt-4o-mini",
    provider: "openrouter",
    name: "openai/gpt-4o-mini",
    input_cost: 0.15,
    output_cost: 0.6,
    context_window: 128000,
    is_disabled: false,
    is_custom: false
  }],
  deepseek: [{
    id: "deepseek-chat",
    provider: "deepseek",
    name: "deepseek-chat",
    input_cost: 0.27,
    output_cost: 1.1,
    context_window: 64000,
    is_disabled: false,
    is_custom: false
  }, {
    id: "deepseek-reasoner",
    provider: "deepseek",
    name: "deepseek-reasoner",
    input_cost: 0.55,
    output_cost: 2.19,
    context_window: 64000,
    is_disabled: false,
    is_custom: false
  }],
  perplexity: [{
    id: "sonar",
    provider: "perplexity",
    name: "sonar",
    input_cost: 1.0,
    output_cost: 1.0,
    context_window: 128000,
    is_disabled: false,
    is_custom: false
  }, {
    id: "sonar-pro",
    provider: "perplexity",
    name: "sonar-pro",
    input_cost: 3.0,
    output_cost: 15.0,
    context_window: 128000,
    is_disabled: false,
    is_custom: false
  }, {
    id: "sonar-reasoning-pro",
    provider: "perplexity",
    name: "sonar-reasoning-pro",
    input_cost: 2.0,
    output_cost: 8.0,
    context_window: 128000,
    is_disabled: false,
    is_custom: false
  }],
  minimax: [{
    id: "MiniMax-M3",
    provider: "minimax",
    name: "MiniMax-M3",
    input_cost: 0.3,
    output_cost: 1.2,
    context_window: 128000,
    is_disabled: false,
    is_custom: false
  }],
  qwen: [{
    id: "qwen3.7-max",
    provider: "qwen",
    name: "qwen3.7-max",
    input_cost: 2.5,
    output_cost: 7.5,
    context_window: 128000,
    is_disabled: false,
    is_custom: false
  }, {
    id: "qwen3.6-plus",
    provider: "qwen",
    name: "qwen3.6-plus",
    input_cost: 0.5,
    output_cost: 3.0,
    context_window: 128000,
    is_disabled: false,
    is_custom: false
  }],
  xai: [{
    id: "grok-4.3",
    provider: "xai",
    name: "grok-4.3",
    input_cost: 1.25,
    output_cost: 2.5,
    context_window: 128000,
    is_disabled: false,
    is_custom: false
  }],
  cerebras: [{
    id: "llama3.1-8b",
    provider: "cerebras",
    name: "llama3.1-8b",
    input_cost: 0.1,
    output_cost: 0.1,
    context_window: 128000,
    is_disabled: false,
    is_custom: false
  }],
  cohere: [{
    id: "command-r-08-2024",
    provider: "cohere",
    name: "command-r-08-2024",
    input_cost: 0.15,
    output_cost: 0.6,
    context_window: 128000,
    is_disabled: false,
    is_custom: false
  }],
  fireworks: [{
    id: "accounts/fireworks/models/deepseek-v4-flash",
    provider: "fireworks",
    name: "accounts/fireworks/models/deepseek-v4-flash",
    input_cost: 0.14,
    output_cost: 0.28,
    context_window: 128000,
    is_disabled: false,
    is_custom: false
  }, {
    id: "accounts/fireworks/models/llama-v3p1-70b-instruct",
    provider: "fireworks",
    name: "accounts/fireworks/models/llama-v3p1-70b-instruct",
    input_cost: 0.3,
    output_cost: 1.2,
    context_window: 128000,
    is_disabled: false,
    is_custom: false
  }, {
    id: "accounts/fireworks/models/llama-v3p3-70b-instruct",
    provider: "fireworks",
    name: "accounts/fireworks/models/llama-v3p3-70b-instruct",
    input_cost: 0.9,
    output_cost: 0.9,
    context_window: 128000,
    is_disabled: false,
    is_custom: false
  }],
  together: [{
    id: "meta-llama/Llama-3.3-70B-Instruct-Turbo",
    provider: "together",
    name: "meta-llama/Llama-3.3-70B-Instruct-Turbo",
    input_cost: 1.04,
    output_cost: 1.04,
    context_window: 128000,
    is_disabled: false,
    is_custom: false
  }, {
    id: "meta-llama/Meta-Llama-3-8B-Instruct-Lite",
    provider: "together",
    name: "meta-llama/Meta-Llama-3-8B-Instruct-Lite",
    input_cost: 0.1,
    output_cost: 0.1,
    context_window: 128000,
    is_disabled: false,
    is_custom: false
  }],
  ollama: [{
    id: "llama3.1:8b",
    provider: "ollama",
    name: "llama3.1:8b",
    input_cost: 0,
    output_cost: 0,
    context_window: 128000,
    is_disabled: false,
    is_custom: false
  }, {
    id: "llama3.3:70b",
    provider: "ollama",
    name: "llama3.3:70b",
    input_cost: 0,
    output_cost: 0,
    context_window: 128000,
    is_disabled: false,
    is_custom: false
  }],
  vertex: [{
    id: "vertex/gemini-2.5-flash",
    provider: "vertex",
    name: "vertex/gemini-2.5-flash",
    input_cost: 0.3,
    output_cost: 2.5,
    context_window: 1000000,
    is_disabled: false,
    is_custom: false
  }, {
    id: "vertex/gemini-2.5-flash-lite",
    provider: "vertex",
    name: "vertex/gemini-2.5-flash-lite",
    input_cost: 0.1,
    output_cost: 0.4,
    context_window: 1000000,
    is_disabled: false,
    is_custom: false
  }],
  "vercel-ai-gateway": [{
    id: "anthropic/claude-sonnet-4.5",
    provider: "vercel-ai-gateway",
    name: "anthropic/claude-sonnet-4.5",
    input_cost: 3.0,
    output_cost: 15.0,
    context_window: 200000,
    is_disabled: false,
    is_custom: false
  }],
  bedrock: [{
    id: "anthropic.claude-3-5-haiku-20241022-v1:0",
    provider: "bedrock",
    name: "anthropic.claude-3-5-haiku-20241022-v1:0",
    input_cost: 0.8,
    output_cost: 4.0,
    context_window: 200000,
    is_disabled: false,
    is_custom: false
  }],
  huggingface: [{
    id: "meta-llama/Llama-3.3-70B-Instruct:groq",
    provider: "huggingface",
    name: "meta-llama/Llama-3.3-70B-Instruct:groq",
    input_cost: 0.59,
    output_cost: 0.79,
    context_window: 128000,
    is_disabled: false,
    is_custom: false
  }]
};

// Known providers matching the backend matrix (providerinfo.ProviderMatrix).
export const knownProviders = [{
  id: "openai",
  name: "openai",
  display_name: "OpenAI",
  description: "GPT-4, GPT-3.5, DALL-E, Whisper",
  auth_types: ["api_key"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "anthropic",
  name: "anthropic",
  display_name: "Anthropic",
  description: "Claude 3.5 Sonnet, Opus, Haiku",
  auth_types: ["api_key"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "gemini",
  name: "gemini",
  display_name: "Google AI",
  description: "Gemini Pro, Flash, Ultra",
  auth_types: ["api_key", "oauth"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "azure",
  name: "azure",
  display_name: "Azure OpenAI",
  description: "Enterprise GPT-4 via Azure",
  auth_types: ["api_key"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "bedrock",
  name: "bedrock",
  display_name: "AWS Bedrock",
  description: "Amazon Claude, Llama, Titan",
  auth_types: ["api_key", "custom"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "cerebras",
  name: "cerebras",
  display_name: "Cerebras",
  description: "Fast inference on Cerebras hardware",
  auth_types: ["api_key"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "cohere",
  name: "cohere",
  display_name: "Cohere",
  description: "Command, Embed, Rerank",
  auth_types: ["api_key"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "deepseek",
  name: "deepseek",
  display_name: "DeepSeek",
  description: "DeepSeek V3, Coder",
  auth_types: ["api_key"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "fireworks",
  name: "fireworks",
  display_name: "Fireworks AI",
  description: "Fast open-source inference",
  auth_types: ["api_key"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "gemini",
  name: "gemini",
  display_name: "Gemini",
  description: "Google Gemini models",
  auth_types: ["api_key", "oauth"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "groq",
  name: "groq",
  display_name: "Groq",
  description: "Ultra-fast LLM inference",
  auth_types: ["api_key"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "huggingface",
  name: "huggingface",
  display_name: "Hugging Face",
  description: "Inference API and endpoints",
  auth_types: ["api_key"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "minimax",
  name: "minimax",
  display_name: "MiniMax",
  description: "MiniMax M3 and multi-modal models",
  auth_types: ["api_key"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "mistral",
  name: "mistral",
  display_name: "Mistral AI",
  description: "Mistral Large, Medium, Small",
  auth_types: ["api_key"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "nebius",
  name: "nebius",
  display_name: "Nebius",
  description: "Nebius AI inference",
  auth_types: ["api_key"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "nvidia",
  name: "nvidia",
  display_name: "NVIDIA",
  description: "NVIDIA NIM inference",
  auth_types: ["api_key"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "ollama",
  name: "ollama",
  display_name: "Ollama",
  description: "Local open-source models",
  auth_types: ["noauth"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "openrouter",
  name: "openrouter",
  display_name: "OpenRouter",
  description: "Unified API for 100+ models",
  auth_types: ["api_key"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "perplexity",
  name: "perplexity",
  display_name: "Perplexity",
  description: "Search-augmented LLMs",
  auth_types: ["api_key"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "qwen",
  name: "qwen",
  display_name: "Qwen",
  description: "Alibaba Qwen models",
  auth_types: ["api_key"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "together",
  name: "together",
  display_name: "Together AI",
  description: "Open-source model hub",
  auth_types: ["api_key"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "vertex",
  name: "vertex",
  display_name: "Google Vertex",
  description: "Gemini on GCP",
  auth_types: ["oauth"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "xai",
  name: "xai",
  display_name: "xAI",
  description: "Grok models",
  auth_types: ["api_key"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "alibaba",
  name: "alibaba",
  display_name: "Alibaba",
  description: "Qwen and Tongyi models",
  auth_types: ["api_key"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "github-copilot",
  name: "github-copilot",
  display_name: "GitHub Copilot",
  description: "GitHub Copilot Chat",
  auth_types: ["oauth"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "kimi",
  name: "kimi",
  display_name: "Kimi",
  description: "Moonshot Kimi models",
  auth_types: ["api_key"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "zhipu",
  name: "zhipu",
  display_name: "Zhipu",
  description: "GLM models",
  auth_types: ["api_key"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "cloudflare-ai-gateway",
  name: "cloudflare-ai-gateway",
  display_name: "Cloudflare AI Gateway",
  description: "Cloudflare AI Gateway",
  auth_types: ["api_key"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "kagi",
  name: "kagi",
  display_name: "Kagi",
  description: "Kagi search and summarization",
  auth_types: ["api_key"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "litellm",
  name: "litellm",
  display_name: "LiteLLM",
  description: "LiteLLM proxy",
  auth_types: ["api_key"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "lm-studio",
  name: "lm-studio",
  display_name: "LM Studio",
  description: "Local LM Studio server",
  auth_types: ["noauth"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "ollama-cloud",
  name: "ollama-cloud",
  display_name: "Ollama Cloud",
  description: "Ollama Cloud inference",
  auth_types: ["api_key"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "opencode",
  name: "opencode",
  display_name: "Opencode",
  description: "Opencode models",
  auth_types: ["api_key"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "replicate",
  name: "replicate",
  display_name: "Replicate",
  description: "Run any open-source model",
  auth_types: ["api_key"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "tavily",
  name: "tavily",
  display_name: "Tavily",
  description: "Tavily search API",
  auth_types: ["api_key"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}, {
  id: "vllm",
  name: "vllm",
  display_name: "vLLM",
  description: "vLLM OpenAI-compatible server",
  auth_types: ["noauth"],
  capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"],
  connection_count: 0,
  status: "inactive"
}];
export function getCatalogModels(providerId) {
  var _providerCatalog$prov;
  return (_providerCatalog$prov = providerCatalog[providerId]) !== null && _providerCatalog$prov !== void 0 ? _providerCatalog$prov : [];
}
export function getAllCatalogModels() {
  return Object.values(providerCatalog).flat();
}
export function getKnownProvider(id) {
  return knownProviders.find(p => p.id === id);
}
//# sourceMappingURL=data:application/json;charset=utf-8;base64,eyJ2ZXJzaW9uIjozLCJuYW1lcyI6WyJwcm92aWRlckNhdGFsb2ciLCJvcGVuYWkiLCJpZCIsInByb3ZpZGVyIiwibmFtZSIsImlucHV0X2Nvc3QiLCJvdXRwdXRfY29zdCIsImNvbnRleHRfd2luZG93IiwiaXNfZGlzYWJsZWQiLCJpc19jdXN0b20iLCJhbnRocm9waWMiLCJnZW1pbmkiLCJncm9xIiwibWlzdHJhbCIsIm5lYml1cyIsIm52aWRpYSIsIm9wZW5yb3V0ZXIiLCJkZWVwc2VlayIsInBlcnBsZXhpdHkiLCJtaW5pbWF4IiwicXdlbiIsInhhaSIsImNlcmVicmFzIiwiY29oZXJlIiwiZmlyZXdvcmtzIiwidG9nZXRoZXIiLCJvbGxhbWEiLCJ2ZXJ0ZXgiLCJiZWRyb2NrIiwiaHVnZ2luZ2ZhY2UiLCJrbm93blByb3ZpZGVycyIsImRpc3BsYXlfbmFtZSIsImRlc2NyaXB0aW9uIiwiYXV0aF90eXBlcyIsImNhcGFiaWxpdGllcyIsImNvbm5lY3Rpb25fY291bnQiLCJzdGF0dXMiLCJnZXRDYXRhbG9nTW9kZWxzIiwicHJvdmlkZXJJZCIsIl9wcm92aWRlckNhdGFsb2ckcHJvdiIsImdldEFsbENhdGFsb2dNb2RlbHMiLCJPYmplY3QiLCJ2YWx1ZXMiLCJmbGF0IiwiZ2V0S25vd25Qcm92aWRlciIsImZpbmQiLCJwIl0sInNvdXJjZXMiOlsiY2F0YWxvZy50cyJdLCJzb3VyY2VzQ29udGVudCI6WyJpbXBvcnQgdHlwZSB7IE1vZGVsLCBQcm92aWRlciB9IGZyb20gXCIuLi8uLi9zcmMvbGliL3R5cGVzXCI7XG5cbi8vIE1vY2sgY2F0YWxvZyBmYWxsYmFjayBtaXJyb3JpbmcgaW50ZXJuYWwvbW9kZWxjYXRhbG9nL2NhdGFsb2cuZ28uXG4vLyBQcm92aWRlcyBkZWZhdWx0IG1vZGVscyBhbmQgcHJpY2luZyBwZXIgcHJvdmlkZXIgYmVmb3JlIGFueSBjb25uZWN0aW9uIGlzIGFkZGVkLlxuXG5leHBvcnQgdHlwZSBDYXRhbG9nTW9kZWwgPSBPbWl0PE1vZGVsLCBcImlkXCI+ICYgeyBpZDogc3RyaW5nIH07XG5cbmNvbnN0IHByb3ZpZGVyQ2F0YWxvZzogUmVjb3JkPHN0cmluZywgQ2F0YWxvZ01vZGVsW10+ID0ge1xuICBvcGVuYWk6IFtcbiAgICB7IGlkOiBcImdwdC00b1wiLCBwcm92aWRlcjogXCJvcGVuYWlcIiwgbmFtZTogXCJncHQtNG9cIiwgaW5wdXRfY29zdDogMi41LCBvdXRwdXRfY29zdDogMTAuMCwgY29udGV4dF93aW5kb3c6IDEyODAwMCwgaXNfZGlzYWJsZWQ6IGZhbHNlLCBpc19jdXN0b206IGZhbHNlIH0sXG4gICAgeyBpZDogXCJncHQtNG8tbWluaVwiLCBwcm92aWRlcjogXCJvcGVuYWlcIiwgbmFtZTogXCJncHQtNG8tbWluaVwiLCBpbnB1dF9jb3N0OiAwLjE1LCBvdXRwdXRfY29zdDogMC42LCBjb250ZXh0X3dpbmRvdzogMTI4MDAwLCBpc19kaXNhYmxlZDogZmFsc2UsIGlzX2N1c3RvbTogZmFsc2UgfSxcbiAgXSxcbiAgYW50aHJvcGljOiBbXG4gICAgeyBpZDogXCJjbGF1ZGUtc29ubmV0LTRcIiwgcHJvdmlkZXI6IFwiYW50aHJvcGljXCIsIG5hbWU6IFwiY2xhdWRlLXNvbm5ldC00XCIsIGlucHV0X2Nvc3Q6IDMuMCwgb3V0cHV0X2Nvc3Q6IDE1LjAsIGNvbnRleHRfd2luZG93OiAyMDAwMDAsIGlzX2Rpc2FibGVkOiBmYWxzZSwgaXNfY3VzdG9tOiBmYWxzZSB9LFxuICAgIHsgaWQ6IFwiY2xhdWRlLW9wdXMtNFwiLCBwcm92aWRlcjogXCJhbnRocm9waWNcIiwgbmFtZTogXCJjbGF1ZGUtb3B1cy00XCIsIGlucHV0X2Nvc3Q6IDE1LjAsIG91dHB1dF9jb3N0OiA3NS4wLCBjb250ZXh0X3dpbmRvdzogMjAwMDAwLCBpc19kaXNhYmxlZDogZmFsc2UsIGlzX2N1c3RvbTogZmFsc2UgfSxcbiAgICB7IGlkOiBcImNsYXVkZS0zLTUtaGFpa3UtMjAyNDEwMjJcIiwgcHJvdmlkZXI6IFwiYW50aHJvcGljXCIsIG5hbWU6IFwiY2xhdWRlLTMtNS1oYWlrdS0yMDI0MTAyMlwiLCBpbnB1dF9jb3N0OiAwLjgsIG91dHB1dF9jb3N0OiA0LjAsIGNvbnRleHRfd2luZG93OiAyMDAwMDAsIGlzX2Rpc2FibGVkOiBmYWxzZSwgaXNfY3VzdG9tOiBmYWxzZSB9LFxuICBdLFxuICBnZW1pbmk6IFtcbiAgICB7IGlkOiBcImdlbWluaS0yLjUtZmxhc2hcIiwgcHJvdmlkZXI6IFwiZ2VtaW5pXCIsIG5hbWU6IFwiZ2VtaW5pLTIuNS1mbGFzaFwiLCBpbnB1dF9jb3N0OiAwLjMsIG91dHB1dF9jb3N0OiAyLjUsIGNvbnRleHRfd2luZG93OiAxMDAwMDAwLCBpc19kaXNhYmxlZDogZmFsc2UsIGlzX2N1c3RvbTogZmFsc2UgfSxcbiAgICB7IGlkOiBcImdlbWluaS0yLjUtZmxhc2gtbGl0ZVwiLCBwcm92aWRlcjogXCJnZW1pbmlcIiwgbmFtZTogXCJnZW1pbmktMi41LWZsYXNoLWxpdGVcIiwgaW5wdXRfY29zdDogMC4xLCBvdXRwdXRfY29zdDogMC40LCBjb250ZXh0X3dpbmRvdzogMTAwMDAwMCwgaXNfZGlzYWJsZWQ6IGZhbHNlLCBpc19jdXN0b206IGZhbHNlIH0sXG4gIF0sXG4gIGdyb3E6IFtcbiAgICB7IGlkOiBcImxsYW1hLTMuMy03MGItdmVyc2F0aWxlXCIsIHByb3ZpZGVyOiBcImdyb3FcIiwgbmFtZTogXCJsbGFtYS0zLjMtNzBiLXZlcnNhdGlsZVwiLCBpbnB1dF9jb3N0OiAwLjU5LCBvdXRwdXRfY29zdDogMC43OSwgY29udGV4dF93aW5kb3c6IDEyODAwMCwgaXNfZGlzYWJsZWQ6IGZhbHNlLCBpc19jdXN0b206IGZhbHNlIH0sXG4gICAgeyBpZDogXCJsbGFtYS0zLjEtOGItaW5zdGFudFwiLCBwcm92aWRlcjogXCJncm9xXCIsIG5hbWU6IFwibGxhbWEtMy4xLThiLWluc3RhbnRcIiwgaW5wdXRfY29zdDogMC4wNSwgb3V0cHV0X2Nvc3Q6IDAuMDgsIGNvbnRleHRfd2luZG93OiAxMjgwMDAsIGlzX2Rpc2FibGVkOiBmYWxzZSwgaXNfY3VzdG9tOiBmYWxzZSB9LFxuICBdLFxuICBtaXN0cmFsOiBbXG4gICAgeyBpZDogXCJtaXN0cmFsLWxhcmdlLWxhdGVzdFwiLCBwcm92aWRlcjogXCJtaXN0cmFsXCIsIG5hbWU6IFwibWlzdHJhbC1sYXJnZS1sYXRlc3RcIiwgaW5wdXRfY29zdDogMi4wLCBvdXRwdXRfY29zdDogNi4wLCBjb250ZXh0X3dpbmRvdzogMTI4MDAwLCBpc19kaXNhYmxlZDogZmFsc2UsIGlzX2N1c3RvbTogZmFsc2UgfSxcbiAgICB7IGlkOiBcIm1pc3RyYWwtc21hbGwtbGF0ZXN0XCIsIHByb3ZpZGVyOiBcIm1pc3RyYWxcIiwgbmFtZTogXCJtaXN0cmFsLXNtYWxsLWxhdGVzdFwiLCBpbnB1dF9jb3N0OiAwLjEsIG91dHB1dF9jb3N0OiAwLjMsIGNvbnRleHRfd2luZG93OiAxMjgwMDAsIGlzX2Rpc2FibGVkOiBmYWxzZSwgaXNfY3VzdG9tOiBmYWxzZSB9LFxuICAgIHsgaWQ6IFwibWFnaXN0cmFsLXNtYWxsLWxhdGVzdFwiLCBwcm92aWRlcjogXCJtaXN0cmFsXCIsIG5hbWU6IFwibWFnaXN0cmFsLXNtYWxsLWxhdGVzdFwiLCBpbnB1dF9jb3N0OiAwLjUsIG91dHB1dF9jb3N0OiAxLjUsIGNvbnRleHRfd2luZG93OiAxMjgwMDAsIGlzX2Rpc2FibGVkOiBmYWxzZSwgaXNfY3VzdG9tOiBmYWxzZSB9LFxuICAgIHsgaWQ6IFwibWluaXN0cmFsLThiLWxhdGVzdFwiLCBwcm92aWRlcjogXCJtaXN0cmFsXCIsIG5hbWU6IFwibWluaXN0cmFsLThiLWxhdGVzdFwiLCBpbnB1dF9jb3N0OiAwLjE1LCBvdXRwdXRfY29zdDogMC4xNSwgY29udGV4dF93aW5kb3c6IDEyODAwMCwgaXNfZGlzYWJsZWQ6IGZhbHNlLCBpc19jdXN0b206IGZhbHNlIH0sXG4gIF0sXG4gIG5lYml1czogW1xuICAgIHsgaWQ6IFwibWV0YS1sbGFtYS9MbGFtYS0zLjMtNzBCLUluc3RydWN0XCIsIHByb3ZpZGVyOiBcIm5lYml1c1wiLCBuYW1lOiBcIm1ldGEtbGxhbWEvTGxhbWEtMy4zLTcwQi1JbnN0cnVjdFwiLCBpbnB1dF9jb3N0OiAwLjEzLCBvdXRwdXRfY29zdDogMC40LCBjb250ZXh0X3dpbmRvdzogMTI4MDAwLCBpc19kaXNhYmxlZDogZmFsc2UsIGlzX2N1c3RvbTogZmFsc2UgfSxcbiAgXSxcbiAgbnZpZGlhOiBbXG4gICAgeyBpZDogXCJtZXRhL2xsYW1hLTMuMS04Yi1pbnN0cnVjdFwiLCBwcm92aWRlcjogXCJudmlkaWFcIiwgbmFtZTogXCJtZXRhL2xsYW1hLTMuMS04Yi1pbnN0cnVjdFwiLCBpbnB1dF9jb3N0OiAwLjEsIG91dHB1dF9jb3N0OiAwLjEsIGNvbnRleHRfd2luZG93OiAxMjgwMDAsIGlzX2Rpc2FibGVkOiBmYWxzZSwgaXNfY3VzdG9tOiBmYWxzZSB9LFxuICBdLFxuICBvcGVucm91dGVyOiBbXG4gICAgeyBpZDogXCJvcGVuYWkvZ3B0LTRvXCIsIHByb3ZpZGVyOiBcIm9wZW5yb3V0ZXJcIiwgbmFtZTogXCJvcGVuYWkvZ3B0LTRvXCIsIGlucHV0X2Nvc3Q6IDIuNSwgb3V0cHV0X2Nvc3Q6IDEwLjAsIGNvbnRleHRfd2luZG93OiAxMjgwMDAsIGlzX2Rpc2FibGVkOiBmYWxzZSwgaXNfY3VzdG9tOiBmYWxzZSB9LFxuICAgIHsgaWQ6IFwib3BlbmFpL2dwdC00by1taW5pXCIsIHByb3ZpZGVyOiBcIm9wZW5yb3V0ZXJcIiwgbmFtZTogXCJvcGVuYWkvZ3B0LTRvLW1pbmlcIiwgaW5wdXRfY29zdDogMC4xNSwgb3V0cHV0X2Nvc3Q6IDAuNiwgY29udGV4dF93aW5kb3c6IDEyODAwMCwgaXNfZGlzYWJsZWQ6IGZhbHNlLCBpc19jdXN0b206IGZhbHNlIH0sXG4gIF0sXG4gIGRlZXBzZWVrOiBbXG4gICAgeyBpZDogXCJkZWVwc2Vlay1jaGF0XCIsIHByb3ZpZGVyOiBcImRlZXBzZWVrXCIsIG5hbWU6IFwiZGVlcHNlZWstY2hhdFwiLCBpbnB1dF9jb3N0OiAwLjI3LCBvdXRwdXRfY29zdDogMS4xLCBjb250ZXh0X3dpbmRvdzogNjQwMDAsIGlzX2Rpc2FibGVkOiBmYWxzZSwgaXNfY3VzdG9tOiBmYWxzZSB9LFxuICAgIHsgaWQ6IFwiZGVlcHNlZWstcmVhc29uZXJcIiwgcHJvdmlkZXI6IFwiZGVlcHNlZWtcIiwgbmFtZTogXCJkZWVwc2Vlay1yZWFzb25lclwiLCBpbnB1dF9jb3N0OiAwLjU1LCBvdXRwdXRfY29zdDogMi4xOSwgY29udGV4dF93aW5kb3c6IDY0MDAwLCBpc19kaXNhYmxlZDogZmFsc2UsIGlzX2N1c3RvbTogZmFsc2UgfSxcbiAgXSxcbiAgcGVycGxleGl0eTogW1xuICAgIHsgaWQ6IFwic29uYXJcIiwgcHJvdmlkZXI6IFwicGVycGxleGl0eVwiLCBuYW1lOiBcInNvbmFyXCIsIGlucHV0X2Nvc3Q6IDEuMCwgb3V0cHV0X2Nvc3Q6IDEuMCwgY29udGV4dF93aW5kb3c6IDEyODAwMCwgaXNfZGlzYWJsZWQ6IGZhbHNlLCBpc19jdXN0b206IGZhbHNlIH0sXG4gICAgeyBpZDogXCJzb25hci1wcm9cIiwgcHJvdmlkZXI6IFwicGVycGxleGl0eVwiLCBuYW1lOiBcInNvbmFyLXByb1wiLCBpbnB1dF9jb3N0OiAzLjAsIG91dHB1dF9jb3N0OiAxNS4wLCBjb250ZXh0X3dpbmRvdzogMTI4MDAwLCBpc19kaXNhYmxlZDogZmFsc2UsIGlzX2N1c3RvbTogZmFsc2UgfSxcbiAgICB7IGlkOiBcInNvbmFyLXJlYXNvbmluZy1wcm9cIiwgcHJvdmlkZXI6IFwicGVycGxleGl0eVwiLCBuYW1lOiBcInNvbmFyLXJlYXNvbmluZy1wcm9cIiwgaW5wdXRfY29zdDogMi4wLCBvdXRwdXRfY29zdDogOC4wLCBjb250ZXh0X3dpbmRvdzogMTI4MDAwLCBpc19kaXNhYmxlZDogZmFsc2UsIGlzX2N1c3RvbTogZmFsc2UgfSxcbiAgXSxcbiAgbWluaW1heDogW1xuICAgIHsgaWQ6IFwiTWluaU1heC1NM1wiLCBwcm92aWRlcjogXCJtaW5pbWF4XCIsIG5hbWU6IFwiTWluaU1heC1NM1wiLCBpbnB1dF9jb3N0OiAwLjMsIG91dHB1dF9jb3N0OiAxLjIsIGNvbnRleHRfd2luZG93OiAxMjgwMDAsIGlzX2Rpc2FibGVkOiBmYWxzZSwgaXNfY3VzdG9tOiBmYWxzZSB9LFxuICBdLFxuICBxd2VuOiBbXG4gICAgeyBpZDogXCJxd2VuMy43LW1heFwiLCBwcm92aWRlcjogXCJxd2VuXCIsIG5hbWU6IFwicXdlbjMuNy1tYXhcIiwgaW5wdXRfY29zdDogMi41LCBvdXRwdXRfY29zdDogNy41LCBjb250ZXh0X3dpbmRvdzogMTI4MDAwLCBpc19kaXNhYmxlZDogZmFsc2UsIGlzX2N1c3RvbTogZmFsc2UgfSxcbiAgICB7IGlkOiBcInF3ZW4zLjYtcGx1c1wiLCBwcm92aWRlcjogXCJxd2VuXCIsIG5hbWU6IFwicXdlbjMuNi1wbHVzXCIsIGlucHV0X2Nvc3Q6IDAuNSwgb3V0cHV0X2Nvc3Q6IDMuMCwgY29udGV4dF93aW5kb3c6IDEyODAwMCwgaXNfZGlzYWJsZWQ6IGZhbHNlLCBpc19jdXN0b206IGZhbHNlIH0sXG4gIF0sXG4gIHhhaTogW1xuICAgIHsgaWQ6IFwiZ3Jvay00LjNcIiwgcHJvdmlkZXI6IFwieGFpXCIsIG5hbWU6IFwiZ3Jvay00LjNcIiwgaW5wdXRfY29zdDogMS4yNSwgb3V0cHV0X2Nvc3Q6IDIuNSwgY29udGV4dF93aW5kb3c6IDEyODAwMCwgaXNfZGlzYWJsZWQ6IGZhbHNlLCBpc19jdXN0b206IGZhbHNlIH0sXG4gIF0sXG4gIGNlcmVicmFzOiBbXG4gICAgeyBpZDogXCJsbGFtYTMuMS04YlwiLCBwcm92aWRlcjogXCJjZXJlYnJhc1wiLCBuYW1lOiBcImxsYW1hMy4xLThiXCIsIGlucHV0X2Nvc3Q6IDAuMSwgb3V0cHV0X2Nvc3Q6IDAuMSwgY29udGV4dF93aW5kb3c6IDEyODAwMCwgaXNfZGlzYWJsZWQ6IGZhbHNlLCBpc19jdXN0b206IGZhbHNlIH0sXG4gIF0sXG4gIGNvaGVyZTogW1xuICAgIHsgaWQ6IFwiY29tbWFuZC1yLTA4LTIwMjRcIiwgcHJvdmlkZXI6IFwiY29oZXJlXCIsIG5hbWU6IFwiY29tbWFuZC1yLTA4LTIwMjRcIiwgaW5wdXRfY29zdDogMC4xNSwgb3V0cHV0X2Nvc3Q6IDAuNiwgY29udGV4dF93aW5kb3c6IDEyODAwMCwgaXNfZGlzYWJsZWQ6IGZhbHNlLCBpc19jdXN0b206IGZhbHNlIH0sXG4gIF0sXG4gIGZpcmV3b3JrczogW1xuICAgIHsgaWQ6IFwiYWNjb3VudHMvZmlyZXdvcmtzL21vZGVscy9kZWVwc2Vlay12NC1mbGFzaFwiLCBwcm92aWRlcjogXCJmaXJld29ya3NcIiwgbmFtZTogXCJhY2NvdW50cy9maXJld29ya3MvbW9kZWxzL2RlZXBzZWVrLXY0LWZsYXNoXCIsIGlucHV0X2Nvc3Q6IDAuMTQsIG91dHB1dF9jb3N0OiAwLjI4LCBjb250ZXh0X3dpbmRvdzogMTI4MDAwLCBpc19kaXNhYmxlZDogZmFsc2UsIGlzX2N1c3RvbTogZmFsc2UgfSxcbiAgICB7IGlkOiBcImFjY291bnRzL2ZpcmV3b3Jrcy9tb2RlbHMvbGxhbWEtdjNwMS03MGItaW5zdHJ1Y3RcIiwgcHJvdmlkZXI6IFwiZmlyZXdvcmtzXCIsIG5hbWU6IFwiYWNjb3VudHMvZmlyZXdvcmtzL21vZGVscy9sbGFtYS12M3AxLTcwYi1pbnN0cnVjdFwiLCBpbnB1dF9jb3N0OiAwLjMsIG91dHB1dF9jb3N0OiAxLjIsIGNvbnRleHRfd2luZG93OiAxMjgwMDAsIGlzX2Rpc2FibGVkOiBmYWxzZSwgaXNfY3VzdG9tOiBmYWxzZSB9LFxuICAgIHsgaWQ6IFwiYWNjb3VudHMvZmlyZXdvcmtzL21vZGVscy9sbGFtYS12M3AzLTcwYi1pbnN0cnVjdFwiLCBwcm92aWRlcjogXCJmaXJld29ya3NcIiwgbmFtZTogXCJhY2NvdW50cy9maXJld29ya3MvbW9kZWxzL2xsYW1hLXYzcDMtNzBiLWluc3RydWN0XCIsIGlucHV0X2Nvc3Q6IDAuOSwgb3V0cHV0X2Nvc3Q6IDAuOSwgY29udGV4dF93aW5kb3c6IDEyODAwMCwgaXNfZGlzYWJsZWQ6IGZhbHNlLCBpc19jdXN0b206IGZhbHNlIH0sXG4gIF0sXG4gIHRvZ2V0aGVyOiBbXG4gICAgeyBpZDogXCJtZXRhLWxsYW1hL0xsYW1hLTMuMy03MEItSW5zdHJ1Y3QtVHVyYm9cIiwgcHJvdmlkZXI6IFwidG9nZXRoZXJcIiwgbmFtZTogXCJtZXRhLWxsYW1hL0xsYW1hLTMuMy03MEItSW5zdHJ1Y3QtVHVyYm9cIiwgaW5wdXRfY29zdDogMS4wNCwgb3V0cHV0X2Nvc3Q6IDEuMDQsIGNvbnRleHRfd2luZG93OiAxMjgwMDAsIGlzX2Rpc2FibGVkOiBmYWxzZSwgaXNfY3VzdG9tOiBmYWxzZSB9LFxuICAgIHsgaWQ6IFwibWV0YS1sbGFtYS9NZXRhLUxsYW1hLTMtOEItSW5zdHJ1Y3QtTGl0ZVwiLCBwcm92aWRlcjogXCJ0b2dldGhlclwiLCBuYW1lOiBcIm1ldGEtbGxhbWEvTWV0YS1MbGFtYS0zLThCLUluc3RydWN0LUxpdGVcIiwgaW5wdXRfY29zdDogMC4xLCBvdXRwdXRfY29zdDogMC4xLCBjb250ZXh0X3dpbmRvdzogMTI4MDAwLCBpc19kaXNhYmxlZDogZmFsc2UsIGlzX2N1c3RvbTogZmFsc2UgfSxcbiAgXSxcbiAgb2xsYW1hOiBbXG4gICAgeyBpZDogXCJsbGFtYTMuMTo4YlwiLCBwcm92aWRlcjogXCJvbGxhbWFcIiwgbmFtZTogXCJsbGFtYTMuMTo4YlwiLCBpbnB1dF9jb3N0OiAwLCBvdXRwdXRfY29zdDogMCwgY29udGV4dF93aW5kb3c6IDEyODAwMCwgaXNfZGlzYWJsZWQ6IGZhbHNlLCBpc19jdXN0b206IGZhbHNlIH0sXG4gICAgeyBpZDogXCJsbGFtYTMuMzo3MGJcIiwgcHJvdmlkZXI6IFwib2xsYW1hXCIsIG5hbWU6IFwibGxhbWEzLjM6NzBiXCIsIGlucHV0X2Nvc3Q6IDAsIG91dHB1dF9jb3N0OiAwLCBjb250ZXh0X3dpbmRvdzogMTI4MDAwLCBpc19kaXNhYmxlZDogZmFsc2UsIGlzX2N1c3RvbTogZmFsc2UgfSxcbiAgXSxcbiAgdmVydGV4OiBbXG4gICAgeyBpZDogXCJ2ZXJ0ZXgvZ2VtaW5pLTIuNS1mbGFzaFwiLCBwcm92aWRlcjogXCJ2ZXJ0ZXhcIiwgbmFtZTogXCJ2ZXJ0ZXgvZ2VtaW5pLTIuNS1mbGFzaFwiLCBpbnB1dF9jb3N0OiAwLjMsIG91dHB1dF9jb3N0OiAyLjUsIGNvbnRleHRfd2luZG93OiAxMDAwMDAwLCBpc19kaXNhYmxlZDogZmFsc2UsIGlzX2N1c3RvbTogZmFsc2UgfSxcbiAgICB7IGlkOiBcInZlcnRleC9nZW1pbmktMi41LWZsYXNoLWxpdGVcIiwgcHJvdmlkZXI6IFwidmVydGV4XCIsIG5hbWU6IFwidmVydGV4L2dlbWluaS0yLjUtZmxhc2gtbGl0ZVwiLCBpbnB1dF9jb3N0OiAwLjEsIG91dHB1dF9jb3N0OiAwLjQsIGNvbnRleHRfd2luZG93OiAxMDAwMDAwLCBpc19kaXNhYmxlZDogZmFsc2UsIGlzX2N1c3RvbTogZmFsc2UgfSxcbiAgXSxcbiAgXCJ2ZXJjZWwtYWktZ2F0ZXdheVwiOiBbXG4gICAgeyBpZDogXCJhbnRocm9waWMvY2xhdWRlLXNvbm5ldC00LjVcIiwgcHJvdmlkZXI6IFwidmVyY2VsLWFpLWdhdGV3YXlcIiwgbmFtZTogXCJhbnRocm9waWMvY2xhdWRlLXNvbm5ldC00LjVcIiwgaW5wdXRfY29zdDogMy4wLCBvdXRwdXRfY29zdDogMTUuMCwgY29udGV4dF93aW5kb3c6IDIwMDAwMCwgaXNfZGlzYWJsZWQ6IGZhbHNlLCBpc19jdXN0b206IGZhbHNlIH0sXG4gIF0sXG4gIGJlZHJvY2s6IFtcbiAgICB7IGlkOiBcImFudGhyb3BpYy5jbGF1ZGUtMy01LWhhaWt1LTIwMjQxMDIyLXYxOjBcIiwgcHJvdmlkZXI6IFwiYmVkcm9ja1wiLCBuYW1lOiBcImFudGhyb3BpYy5jbGF1ZGUtMy01LWhhaWt1LTIwMjQxMDIyLXYxOjBcIiwgaW5wdXRfY29zdDogMC44LCBvdXRwdXRfY29zdDogNC4wLCBjb250ZXh0X3dpbmRvdzogMjAwMDAwLCBpc19kaXNhYmxlZDogZmFsc2UsIGlzX2N1c3RvbTogZmFsc2UgfSxcbiAgXSxcbiAgaHVnZ2luZ2ZhY2U6IFtcbiAgICB7IGlkOiBcIm1ldGEtbGxhbWEvTGxhbWEtMy4zLTcwQi1JbnN0cnVjdDpncm9xXCIsIHByb3ZpZGVyOiBcImh1Z2dpbmdmYWNlXCIsIG5hbWU6IFwibWV0YS1sbGFtYS9MbGFtYS0zLjMtNzBCLUluc3RydWN0Omdyb3FcIiwgaW5wdXRfY29zdDogMC41OSwgb3V0cHV0X2Nvc3Q6IDAuNzksIGNvbnRleHRfd2luZG93OiAxMjgwMDAsIGlzX2Rpc2FibGVkOiBmYWxzZSwgaXNfY3VzdG9tOiBmYWxzZSB9LFxuICBdLFxufTtcblxuLy8gS25vd24gcHJvdmlkZXJzIG1hdGNoaW5nIHRoZSBiYWNrZW5kIG1hdHJpeCAocHJvdmlkZXJpbmZvLlByb3ZpZGVyTWF0cml4KS5cbmV4cG9ydCBjb25zdCBrbm93blByb3ZpZGVyczogUHJvdmlkZXJbXSA9IFtcbiAgeyBpZDogXCJvcGVuYWlcIiwgbmFtZTogXCJvcGVuYWlcIiwgZGlzcGxheV9uYW1lOiBcIk9wZW5BSVwiLCBkZXNjcmlwdGlvbjogXCJHUFQtNCwgR1BULTMuNSwgREFMTC1FLCBXaGlzcGVyXCIsIGF1dGhfdHlwZXM6IFtcImFwaV9rZXlcIl0sIGNhcGFiaWxpdGllczogW1wiaW5mZXJlbmNlXCIsIFwic3RyZWFtaW5nXCIsIFwibW9kZWxfY2F0YWxvZ1wiLCBcImxpc3RfbW9kZWxzXCIsIFwicHVibGljX2luZmVyZW5jZVwiLCBcImRpcmVjdF9kaXNwYXRjaFwiXSwgY29ubmVjdGlvbl9jb3VudDogMCwgc3RhdHVzOiBcImluYWN0aXZlXCIgfSxcbiAgeyBpZDogXCJhbnRocm9waWNcIiwgbmFtZTogXCJhbnRocm9waWNcIiwgZGlzcGxheV9uYW1lOiBcIkFudGhyb3BpY1wiLCBkZXNjcmlwdGlvbjogXCJDbGF1ZGUgMy41IFNvbm5ldCwgT3B1cywgSGFpa3VcIiwgYXV0aF90eXBlczogW1wiYXBpX2tleVwiXSwgY2FwYWJpbGl0aWVzOiBbXCJpbmZlcmVuY2VcIiwgXCJzdHJlYW1pbmdcIiwgXCJtb2RlbF9jYXRhbG9nXCIsIFwibGlzdF9tb2RlbHNcIiwgXCJwdWJsaWNfaW5mZXJlbmNlXCIsIFwiZGlyZWN0X2Rpc3BhdGNoXCJdLCBjb25uZWN0aW9uX2NvdW50OiAwLCBzdGF0dXM6IFwiaW5hY3RpdmVcIiB9LFxuICB7IGlkOiBcImdlbWluaVwiLCBuYW1lOiBcImdlbWluaVwiLCBkaXNwbGF5X25hbWU6IFwiR29vZ2xlIEFJXCIsIGRlc2NyaXB0aW9uOiBcIkdlbWluaSBQcm8sIEZsYXNoLCBVbHRyYVwiLCBhdXRoX3R5cGVzOiBbXCJhcGlfa2V5XCIsIFwib2F1dGhcIl0sIGNhcGFiaWxpdGllczogW1wiaW5mZXJlbmNlXCIsIFwic3RyZWFtaW5nXCIsIFwibW9kZWxfY2F0YWxvZ1wiLCBcImxpc3RfbW9kZWxzXCIsIFwicHVibGljX2luZmVyZW5jZVwiLCBcImRpcmVjdF9kaXNwYXRjaFwiXSwgY29ubmVjdGlvbl9jb3VudDogMCwgc3RhdHVzOiBcImluYWN0aXZlXCIgfSxcbiAgeyBpZDogXCJhenVyZVwiLCBuYW1lOiBcImF6dXJlXCIsIGRpc3BsYXlfbmFtZTogXCJBenVyZSBPcGVuQUlcIiwgZGVzY3JpcHRpb246IFwiRW50ZXJwcmlzZSBHUFQtNCB2aWEgQXp1cmVcIiwgYXV0aF90eXBlczogW1wiYXBpX2tleVwiXSwgY2FwYWJpbGl0aWVzOiBbXCJpbmZlcmVuY2VcIiwgXCJzdHJlYW1pbmdcIiwgXCJtb2RlbF9jYXRhbG9nXCIsIFwibGlzdF9tb2RlbHNcIiwgXCJwdWJsaWNfaW5mZXJlbmNlXCIsIFwiZGlyZWN0X2Rpc3BhdGNoXCJdLCBjb25uZWN0aW9uX2NvdW50OiAwLCBzdGF0dXM6IFwiaW5hY3RpdmVcIiB9LFxuICB7IGlkOiBcImJlZHJvY2tcIiwgbmFtZTogXCJiZWRyb2NrXCIsIGRpc3BsYXlfbmFtZTogXCJBV1MgQmVkcm9ja1wiLCBkZXNjcmlwdGlvbjogXCJBbWF6b24gQ2xhdWRlLCBMbGFtYSwgVGl0YW5cIiwgYXV0aF90eXBlczogW1wiYXBpX2tleVwiLCBcImN1c3RvbVwiXSwgY2FwYWJpbGl0aWVzOiBbXCJpbmZlcmVuY2VcIiwgXCJzdHJlYW1pbmdcIiwgXCJtb2RlbF9jYXRhbG9nXCIsIFwibGlzdF9tb2RlbHNcIiwgXCJwdWJsaWNfaW5mZXJlbmNlXCIsIFwiZGlyZWN0X2Rpc3BhdGNoXCJdLCBjb25uZWN0aW9uX2NvdW50OiAwLCBzdGF0dXM6IFwiaW5hY3RpdmVcIiB9LFxuICB7IGlkOiBcImNlcmVicmFzXCIsIG5hbWU6IFwiY2VyZWJyYXNcIiwgZGlzcGxheV9uYW1lOiBcIkNlcmVicmFzXCIsIGRlc2NyaXB0aW9uOiBcIkZhc3QgaW5mZXJlbmNlIG9uIENlcmVicmFzIGhhcmR3YXJlXCIsIGF1dGhfdHlwZXM6IFtcImFwaV9rZXlcIl0sIGNhcGFiaWxpdGllczogW1wiaW5mZXJlbmNlXCIsIFwic3RyZWFtaW5nXCIsIFwibW9kZWxfY2F0YWxvZ1wiLCBcImxpc3RfbW9kZWxzXCIsIFwicHVibGljX2luZmVyZW5jZVwiLCBcImRpcmVjdF9kaXNwYXRjaFwiXSwgY29ubmVjdGlvbl9jb3VudDogMCwgc3RhdHVzOiBcImluYWN0aXZlXCIgfSxcbiAgeyBpZDogXCJjb2hlcmVcIiwgbmFtZTogXCJjb2hlcmVcIiwgZGlzcGxheV9uYW1lOiBcIkNvaGVyZVwiLCBkZXNjcmlwdGlvbjogXCJDb21tYW5kLCBFbWJlZCwgUmVyYW5rXCIsIGF1dGhfdHlwZXM6IFtcImFwaV9rZXlcIl0sIGNhcGFiaWxpdGllczogW1wiaW5mZXJlbmNlXCIsIFwic3RyZWFtaW5nXCIsIFwibW9kZWxfY2F0YWxvZ1wiLCBcImxpc3RfbW9kZWxzXCIsIFwicHVibGljX2luZmVyZW5jZVwiLCBcImRpcmVjdF9kaXNwYXRjaFwiXSwgY29ubmVjdGlvbl9jb3VudDogMCwgc3RhdHVzOiBcImluYWN0aXZlXCIgfSxcbiAgeyBpZDogXCJkZWVwc2Vla1wiLCBuYW1lOiBcImRlZXBzZWVrXCIsIGRpc3BsYXlfbmFtZTogXCJEZWVwU2Vla1wiLCBkZXNjcmlwdGlvbjogXCJEZWVwU2VlayBWMywgQ29kZXJcIiwgYXV0aF90eXBlczogW1wiYXBpX2tleVwiXSwgY2FwYWJpbGl0aWVzOiBbXCJpbmZlcmVuY2VcIiwgXCJzdHJlYW1pbmdcIiwgXCJtb2RlbF9jYXRhbG9nXCIsIFwibGlzdF9tb2RlbHNcIiwgXCJwdWJsaWNfaW5mZXJlbmNlXCIsIFwiZGlyZWN0X2Rpc3BhdGNoXCJdLCBjb25uZWN0aW9uX2NvdW50OiAwLCBzdGF0dXM6IFwiaW5hY3RpdmVcIiB9LFxuICB7IGlkOiBcImZpcmV3b3Jrc1wiLCBuYW1lOiBcImZpcmV3b3Jrc1wiLCBkaXNwbGF5X25hbWU6IFwiRmlyZXdvcmtzIEFJXCIsIGRlc2NyaXB0aW9uOiBcIkZhc3Qgb3Blbi1zb3VyY2UgaW5mZXJlbmNlXCIsIGF1dGhfdHlwZXM6IFtcImFwaV9rZXlcIl0sIGNhcGFiaWxpdGllczogW1wiaW5mZXJlbmNlXCIsIFwic3RyZWFtaW5nXCIsIFwibW9kZWxfY2F0YWxvZ1wiLCBcImxpc3RfbW9kZWxzXCIsIFwicHVibGljX2luZmVyZW5jZVwiLCBcImRpcmVjdF9kaXNwYXRjaFwiXSwgY29ubmVjdGlvbl9jb3VudDogMCwgc3RhdHVzOiBcImluYWN0aXZlXCIgfSxcbiAgeyBpZDogXCJnZW1pbmlcIiwgbmFtZTogXCJnZW1pbmlcIiwgZGlzcGxheV9uYW1lOiBcIkdlbWluaVwiLCBkZXNjcmlwdGlvbjogXCJHb29nbGUgR2VtaW5pIG1vZGVsc1wiLCBhdXRoX3R5cGVzOiBbXCJhcGlfa2V5XCIsIFwib2F1dGhcIl0sIGNhcGFiaWxpdGllczogW1wiaW5mZXJlbmNlXCIsIFwic3RyZWFtaW5nXCIsIFwibW9kZWxfY2F0YWxvZ1wiLCBcImxpc3RfbW9kZWxzXCIsIFwicHVibGljX2luZmVyZW5jZVwiLCBcImRpcmVjdF9kaXNwYXRjaFwiXSwgY29ubmVjdGlvbl9jb3VudDogMCwgc3RhdHVzOiBcImluYWN0aXZlXCIgfSxcbiAgeyBpZDogXCJncm9xXCIsIG5hbWU6IFwiZ3JvcVwiLCBkaXNwbGF5X25hbWU6IFwiR3JvcVwiLCBkZXNjcmlwdGlvbjogXCJVbHRyYS1mYXN0IExMTSBpbmZlcmVuY2VcIiwgYXV0aF90eXBlczogW1wiYXBpX2tleVwiXSwgY2FwYWJpbGl0aWVzOiBbXCJpbmZlcmVuY2VcIiwgXCJzdHJlYW1pbmdcIiwgXCJtb2RlbF9jYXRhbG9nXCIsIFwibGlzdF9tb2RlbHNcIiwgXCJwdWJsaWNfaW5mZXJlbmNlXCIsIFwiZGlyZWN0X2Rpc3BhdGNoXCJdLCBjb25uZWN0aW9uX2NvdW50OiAwLCBzdGF0dXM6IFwiaW5hY3RpdmVcIiB9LFxuICB7IGlkOiBcImh1Z2dpbmdmYWNlXCIsIG5hbWU6IFwiaHVnZ2luZ2ZhY2VcIiwgZGlzcGxheV9uYW1lOiBcIkh1Z2dpbmcgRmFjZVwiLCBkZXNjcmlwdGlvbjogXCJJbmZlcmVuY2UgQVBJIGFuZCBlbmRwb2ludHNcIiwgYXV0aF90eXBlczogW1wiYXBpX2tleVwiXSwgY2FwYWJpbGl0aWVzOiBbXCJpbmZlcmVuY2VcIiwgXCJzdHJlYW1pbmdcIiwgXCJtb2RlbF9jYXRhbG9nXCIsIFwibGlzdF9tb2RlbHNcIiwgXCJwdWJsaWNfaW5mZXJlbmNlXCIsIFwiZGlyZWN0X2Rpc3BhdGNoXCJdLCBjb25uZWN0aW9uX2NvdW50OiAwLCBzdGF0dXM6IFwiaW5hY3RpdmVcIiB9LFxuICB7IGlkOiBcIm1pbmltYXhcIiwgbmFtZTogXCJtaW5pbWF4XCIsIGRpc3BsYXlfbmFtZTogXCJNaW5pTWF4XCIsIGRlc2NyaXB0aW9uOiBcIk1pbmlNYXggTTMgYW5kIG11bHRpLW1vZGFsIG1vZGVsc1wiLCBhdXRoX3R5cGVzOiBbXCJhcGlfa2V5XCJdLCBjYXBhYmlsaXRpZXM6IFtcImluZmVyZW5jZVwiLCBcInN0cmVhbWluZ1wiLCBcIm1vZGVsX2NhdGFsb2dcIiwgXCJsaXN0X21vZGVsc1wiLCBcInB1YmxpY19pbmZlcmVuY2VcIiwgXCJkaXJlY3RfZGlzcGF0Y2hcIl0sIGNvbm5lY3Rpb25fY291bnQ6IDAsIHN0YXR1czogXCJpbmFjdGl2ZVwiIH0sXG4gIHsgaWQ6IFwibWlzdHJhbFwiLCBuYW1lOiBcIm1pc3RyYWxcIiwgZGlzcGxheV9uYW1lOiBcIk1pc3RyYWwgQUlcIiwgZGVzY3JpcHRpb246IFwiTWlzdHJhbCBMYXJnZSwgTWVkaXVtLCBTbWFsbFwiLCBhdXRoX3R5cGVzOiBbXCJhcGlfa2V5XCJdLCBjYXBhYmlsaXRpZXM6IFtcImluZmVyZW5jZVwiLCBcInN0cmVhbWluZ1wiLCBcIm1vZGVsX2NhdGFsb2dcIiwgXCJsaXN0X21vZGVsc1wiLCBcInB1YmxpY19pbmZlcmVuY2VcIiwgXCJkaXJlY3RfZGlzcGF0Y2hcIl0sIGNvbm5lY3Rpb25fY291bnQ6IDAsIHN0YXR1czogXCJpbmFjdGl2ZVwiIH0sXG4gIHsgaWQ6IFwibmViaXVzXCIsIG5hbWU6IFwibmViaXVzXCIsIGRpc3BsYXlfbmFtZTogXCJOZWJpdXNcIiwgZGVzY3JpcHRpb246IFwiTmViaXVzIEFJIGluZmVyZW5jZVwiLCBhdXRoX3R5cGVzOiBbXCJhcGlfa2V5XCJdLCBjYXBhYmlsaXRpZXM6IFtcImluZmVyZW5jZVwiLCBcInN0cmVhbWluZ1wiLCBcIm1vZGVsX2NhdGFsb2dcIiwgXCJsaXN0X21vZGVsc1wiLCBcInB1YmxpY19pbmZlcmVuY2VcIiwgXCJkaXJlY3RfZGlzcGF0Y2hcIl0sIGNvbm5lY3Rpb25fY291bnQ6IDAsIHN0YXR1czogXCJpbmFjdGl2ZVwiIH0sXG4gIHsgaWQ6IFwibnZpZGlhXCIsIG5hbWU6IFwibnZpZGlhXCIsIGRpc3BsYXlfbmFtZTogXCJOVklESUFcIiwgZGVzY3JpcHRpb246IFwiTlZJRElBIE5JTSBpbmZlcmVuY2VcIiwgYXV0aF90eXBlczogW1wiYXBpX2tleVwiXSwgY2FwYWJpbGl0aWVzOiBbXCJpbmZlcmVuY2VcIiwgXCJzdHJlYW1pbmdcIiwgXCJtb2RlbF9jYXRhbG9nXCIsIFwibGlzdF9tb2RlbHNcIiwgXCJwdWJsaWNfaW5mZXJlbmNlXCIsIFwiZGlyZWN0X2Rpc3BhdGNoXCJdLCBjb25uZWN0aW9uX2NvdW50OiAwLCBzdGF0dXM6IFwiaW5hY3RpdmVcIiB9LFxuICB7IGlkOiBcIm9sbGFtYVwiLCBuYW1lOiBcIm9sbGFtYVwiLCBkaXNwbGF5X25hbWU6IFwiT2xsYW1hXCIsIGRlc2NyaXB0aW9uOiBcIkxvY2FsIG9wZW4tc291cmNlIG1vZGVsc1wiLCBhdXRoX3R5cGVzOiBbXCJub2F1dGhcIl0sIGNhcGFiaWxpdGllczogW1wiaW5mZXJlbmNlXCIsIFwic3RyZWFtaW5nXCIsIFwibW9kZWxfY2F0YWxvZ1wiLCBcImxpc3RfbW9kZWxzXCIsIFwicHVibGljX2luZmVyZW5jZVwiLCBcImRpcmVjdF9kaXNwYXRjaFwiXSwgY29ubmVjdGlvbl9jb3VudDogMCwgc3RhdHVzOiBcImluYWN0aXZlXCIgfSxcbiAgeyBpZDogXCJvcGVucm91dGVyXCIsIG5hbWU6IFwib3BlbnJvdXRlclwiLCBkaXNwbGF5X25hbWU6IFwiT3BlblJvdXRlclwiLCBkZXNjcmlwdGlvbjogXCJVbmlmaWVkIEFQSSBmb3IgMTAwKyBtb2RlbHNcIiwgYXV0aF90eXBlczogW1wiYXBpX2tleVwiXSwgY2FwYWJpbGl0aWVzOiBbXCJpbmZlcmVuY2VcIiwgXCJzdHJlYW1pbmdcIiwgXCJtb2RlbF9jYXRhbG9nXCIsIFwibGlzdF9tb2RlbHNcIiwgXCJwdWJsaWNfaW5mZXJlbmNlXCIsIFwiZGlyZWN0X2Rpc3BhdGNoXCJdLCBjb25uZWN0aW9uX2NvdW50OiAwLCBzdGF0dXM6IFwiaW5hY3RpdmVcIiB9LFxuICB7IGlkOiBcInBlcnBsZXhpdHlcIiwgbmFtZTogXCJwZXJwbGV4aXR5XCIsIGRpc3BsYXlfbmFtZTogXCJQZXJwbGV4aXR5XCIsIGRlc2NyaXB0aW9uOiBcIlNlYXJjaC1hdWdtZW50ZWQgTExNc1wiLCBhdXRoX3R5cGVzOiBbXCJhcGlfa2V5XCJdLCBjYXBhYmlsaXRpZXM6IFtcImluZmVyZW5jZVwiLCBcInN0cmVhbWluZ1wiLCBcIm1vZGVsX2NhdGFsb2dcIiwgXCJsaXN0X21vZGVsc1wiLCBcInB1YmxpY19pbmZlcmVuY2VcIiwgXCJkaXJlY3RfZGlzcGF0Y2hcIl0sIGNvbm5lY3Rpb25fY291bnQ6IDAsIHN0YXR1czogXCJpbmFjdGl2ZVwiIH0sXG4gIHsgaWQ6IFwicXdlblwiLCBuYW1lOiBcInF3ZW5cIiwgZGlzcGxheV9uYW1lOiBcIlF3ZW5cIiwgZGVzY3JpcHRpb246IFwiQWxpYmFiYSBRd2VuIG1vZGVsc1wiLCBhdXRoX3R5cGVzOiBbXCJhcGlfa2V5XCJdLCBjYXBhYmlsaXRpZXM6IFtcImluZmVyZW5jZVwiLCBcInN0cmVhbWluZ1wiLCBcIm1vZGVsX2NhdGFsb2dcIiwgXCJsaXN0X21vZGVsc1wiLCBcInB1YmxpY19pbmZlcmVuY2VcIiwgXCJkaXJlY3RfZGlzcGF0Y2hcIl0sIGNvbm5lY3Rpb25fY291bnQ6IDAsIHN0YXR1czogXCJpbmFjdGl2ZVwiIH0sXG4gIHsgaWQ6IFwidG9nZXRoZXJcIiwgbmFtZTogXCJ0b2dldGhlclwiLCBkaXNwbGF5X25hbWU6IFwiVG9nZXRoZXIgQUlcIiwgZGVzY3JpcHRpb246IFwiT3Blbi1zb3VyY2UgbW9kZWwgaHViXCIsIGF1dGhfdHlwZXM6IFtcImFwaV9rZXlcIl0sIGNhcGFiaWxpdGllczogW1wiaW5mZXJlbmNlXCIsIFwic3RyZWFtaW5nXCIsIFwibW9kZWxfY2F0YWxvZ1wiLCBcImxpc3RfbW9kZWxzXCIsIFwicHVibGljX2luZmVyZW5jZVwiLCBcImRpcmVjdF9kaXNwYXRjaFwiXSwgY29ubmVjdGlvbl9jb3VudDogMCwgc3RhdHVzOiBcImluYWN0aXZlXCIgfSxcbiAgeyBpZDogXCJ2ZXJ0ZXhcIiwgbmFtZTogXCJ2ZXJ0ZXhcIiwgZGlzcGxheV9uYW1lOiBcIkdvb2dsZSBWZXJ0ZXhcIiwgZGVzY3JpcHRpb246IFwiR2VtaW5pIG9uIEdDUFwiLCBhdXRoX3R5cGVzOiBbXCJvYXV0aFwiXSwgY2FwYWJpbGl0aWVzOiBbXCJpbmZlcmVuY2VcIiwgXCJzdHJlYW1pbmdcIiwgXCJtb2RlbF9jYXRhbG9nXCIsIFwibGlzdF9tb2RlbHNcIiwgXCJwdWJsaWNfaW5mZXJlbmNlXCIsIFwiZGlyZWN0X2Rpc3BhdGNoXCJdLCBjb25uZWN0aW9uX2NvdW50OiAwLCBzdGF0dXM6IFwiaW5hY3RpdmVcIiB9LFxuICB7IGlkOiBcInhhaVwiLCBuYW1lOiBcInhhaVwiLCBkaXNwbGF5X25hbWU6IFwieEFJXCIsIGRlc2NyaXB0aW9uOiBcIkdyb2sgbW9kZWxzXCIsIGF1dGhfdHlwZXM6IFtcImFwaV9rZXlcIl0sIGNhcGFiaWxpdGllczogW1wiaW5mZXJlbmNlXCIsIFwic3RyZWFtaW5nXCIsIFwibW9kZWxfY2F0YWxvZ1wiLCBcImxpc3RfbW9kZWxzXCIsIFwicHVibGljX2luZmVyZW5jZVwiLCBcImRpcmVjdF9kaXNwYXRjaFwiXSwgY29ubmVjdGlvbl9jb3VudDogMCwgc3RhdHVzOiBcImluYWN0aXZlXCIgfSxcbiAgeyBpZDogXCJhbGliYWJhXCIsIG5hbWU6IFwiYWxpYmFiYVwiLCBkaXNwbGF5X25hbWU6IFwiQWxpYmFiYVwiLCBkZXNjcmlwdGlvbjogXCJRd2VuIGFuZCBUb25neWkgbW9kZWxzXCIsIGF1dGhfdHlwZXM6IFtcImFwaV9rZXlcIl0sIGNhcGFiaWxpdGllczogW1wiaW5mZXJlbmNlXCIsIFwic3RyZWFtaW5nXCIsIFwibW9kZWxfY2F0YWxvZ1wiLCBcImxpc3RfbW9kZWxzXCIsIFwicHVibGljX2luZmVyZW5jZVwiLCBcImRpcmVjdF9kaXNwYXRjaFwiXSwgY29ubmVjdGlvbl9jb3VudDogMCwgc3RhdHVzOiBcImluYWN0aXZlXCIgfSxcbiAgeyBpZDogXCJnaXRodWItY29waWxvdFwiLCBuYW1lOiBcImdpdGh1Yi1jb3BpbG90XCIsIGRpc3BsYXlfbmFtZTogXCJHaXRIdWIgQ29waWxvdFwiLCBkZXNjcmlwdGlvbjogXCJHaXRIdWIgQ29waWxvdCBDaGF0XCIsIGF1dGhfdHlwZXM6IFtcIm9hdXRoXCJdLCBjYXBhYmlsaXRpZXM6IFtcImluZmVyZW5jZVwiLCBcInN0cmVhbWluZ1wiLCBcIm1vZGVsX2NhdGFsb2dcIiwgXCJsaXN0X21vZGVsc1wiLCBcInB1YmxpY19pbmZlcmVuY2VcIiwgXCJkaXJlY3RfZGlzcGF0Y2hcIl0sIGNvbm5lY3Rpb25fY291bnQ6IDAsIHN0YXR1czogXCJpbmFjdGl2ZVwiIH0sXG4gIHsgaWQ6IFwia2ltaVwiLCBuYW1lOiBcImtpbWlcIiwgZGlzcGxheV9uYW1lOiBcIktpbWlcIiwgZGVzY3JpcHRpb246IFwiTW9vbnNob3QgS2ltaSBtb2RlbHNcIiwgYXV0aF90eXBlczogW1wiYXBpX2tleVwiXSwgY2FwYWJpbGl0aWVzOiBbXCJpbmZlcmVuY2VcIiwgXCJzdHJlYW1pbmdcIiwgXCJtb2RlbF9jYXRhbG9nXCIsIFwibGlzdF9tb2RlbHNcIiwgXCJwdWJsaWNfaW5mZXJlbmNlXCIsIFwiZGlyZWN0X2Rpc3BhdGNoXCJdLCBjb25uZWN0aW9uX2NvdW50OiAwLCBzdGF0dXM6IFwiaW5hY3RpdmVcIiB9LFxuICB7IGlkOiBcInpoaXB1XCIsIG5hbWU6IFwiemhpcHVcIiwgZGlzcGxheV9uYW1lOiBcIlpoaXB1XCIsIGRlc2NyaXB0aW9uOiBcIkdMTSBtb2RlbHNcIiwgYXV0aF90eXBlczogW1wiYXBpX2tleVwiXSwgY2FwYWJpbGl0aWVzOiBbXCJpbmZlcmVuY2VcIiwgXCJzdHJlYW1pbmdcIiwgXCJtb2RlbF9jYXRhbG9nXCIsIFwibGlzdF9tb2RlbHNcIiwgXCJwdWJsaWNfaW5mZXJlbmNlXCIsIFwiZGlyZWN0X2Rpc3BhdGNoXCJdLCBjb25uZWN0aW9uX2NvdW50OiAwLCBzdGF0dXM6IFwiaW5hY3RpdmVcIiB9LFxuICB7IGlkOiBcImNsb3VkZmxhcmUtYWktZ2F0ZXdheVwiLCBuYW1lOiBcImNsb3VkZmxhcmUtYWktZ2F0ZXdheVwiLCBkaXNwbGF5X25hbWU6IFwiQ2xvdWRmbGFyZSBBSSBHYXRld2F5XCIsIGRlc2NyaXB0aW9uOiBcIkNsb3VkZmxhcmUgQUkgR2F0ZXdheVwiLCBhdXRoX3R5cGVzOiBbXCJhcGlfa2V5XCJdLCBjYXBhYmlsaXRpZXM6IFtcImluZmVyZW5jZVwiLCBcInN0cmVhbWluZ1wiLCBcIm1vZGVsX2NhdGFsb2dcIiwgXCJsaXN0X21vZGVsc1wiLCBcInB1YmxpY19pbmZlcmVuY2VcIiwgXCJkaXJlY3RfZGlzcGF0Y2hcIl0sIGNvbm5lY3Rpb25fY291bnQ6IDAsIHN0YXR1czogXCJpbmFjdGl2ZVwiIH0sXG4gIHsgaWQ6IFwia2FnaVwiLCBuYW1lOiBcImthZ2lcIiwgZGlzcGxheV9uYW1lOiBcIkthZ2lcIiwgZGVzY3JpcHRpb246IFwiS2FnaSBzZWFyY2ggYW5kIHN1bW1hcml6YXRpb25cIiwgYXV0aF90eXBlczogW1wiYXBpX2tleVwiXSwgY2FwYWJpbGl0aWVzOiBbXCJpbmZlcmVuY2VcIiwgXCJzdHJlYW1pbmdcIiwgXCJtb2RlbF9jYXRhbG9nXCIsIFwibGlzdF9tb2RlbHNcIiwgXCJwdWJsaWNfaW5mZXJlbmNlXCIsIFwiZGlyZWN0X2Rpc3BhdGNoXCJdLCBjb25uZWN0aW9uX2NvdW50OiAwLCBzdGF0dXM6IFwiaW5hY3RpdmVcIiB9LFxuICB7IGlkOiBcImxpdGVsbG1cIiwgbmFtZTogXCJsaXRlbGxtXCIsIGRpc3BsYXlfbmFtZTogXCJMaXRlTExNXCIsIGRlc2NyaXB0aW9uOiBcIkxpdGVMTE0gcHJveHlcIiwgYXV0aF90eXBlczogW1wiYXBpX2tleVwiXSwgY2FwYWJpbGl0aWVzOiBbXCJpbmZlcmVuY2VcIiwgXCJzdHJlYW1pbmdcIiwgXCJtb2RlbF9jYXRhbG9nXCIsIFwibGlzdF9tb2RlbHNcIiwgXCJwdWJsaWNfaW5mZXJlbmNlXCIsIFwiZGlyZWN0X2Rpc3BhdGNoXCJdLCBjb25uZWN0aW9uX2NvdW50OiAwLCBzdGF0dXM6IFwiaW5hY3RpdmVcIiB9LFxuICB7IGlkOiBcImxtLXN0dWRpb1wiLCBuYW1lOiBcImxtLXN0dWRpb1wiLCBkaXNwbGF5X25hbWU6IFwiTE0gU3R1ZGlvXCIsIGRlc2NyaXB0aW9uOiBcIkxvY2FsIExNIFN0dWRpbyBzZXJ2ZXJcIiwgYXV0aF90eXBlczogW1wibm9hdXRoXCJdLCBjYXBhYmlsaXRpZXM6IFtcImluZmVyZW5jZVwiLCBcInN0cmVhbWluZ1wiLCBcIm1vZGVsX2NhdGFsb2dcIiwgXCJsaXN0X21vZGVsc1wiLCBcInB1YmxpY19pbmZlcmVuY2VcIiwgXCJkaXJlY3RfZGlzcGF0Y2hcIl0sIGNvbm5lY3Rpb25fY291bnQ6IDAsIHN0YXR1czogXCJpbmFjdGl2ZVwiIH0sXG4gIHsgaWQ6IFwib2xsYW1hLWNsb3VkXCIsIG5hbWU6IFwib2xsYW1hLWNsb3VkXCIsIGRpc3BsYXlfbmFtZTogXCJPbGxhbWEgQ2xvdWRcIiwgZGVzY3JpcHRpb246IFwiT2xsYW1hIENsb3VkIGluZmVyZW5jZVwiLCBhdXRoX3R5cGVzOiBbXCJhcGlfa2V5XCJdLCBjYXBhYmlsaXRpZXM6IFtcImluZmVyZW5jZVwiLCBcInN0cmVhbWluZ1wiLCBcIm1vZGVsX2NhdGFsb2dcIiwgXCJsaXN0X21vZGVsc1wiLCBcInB1YmxpY19pbmZlcmVuY2VcIiwgXCJkaXJlY3RfZGlzcGF0Y2hcIl0sIGNvbm5lY3Rpb25fY291bnQ6IDAsIHN0YXR1czogXCJpbmFjdGl2ZVwiIH0sXG4gIHsgaWQ6IFwib3BlbmNvZGVcIiwgbmFtZTogXCJvcGVuY29kZVwiLCBkaXNwbGF5X25hbWU6IFwiT3BlbmNvZGVcIiwgZGVzY3JpcHRpb246IFwiT3BlbmNvZGUgbW9kZWxzXCIsIGF1dGhfdHlwZXM6IFtcImFwaV9rZXlcIl0sIGNhcGFiaWxpdGllczogW1wiaW5mZXJlbmNlXCIsIFwic3RyZWFtaW5nXCIsIFwibW9kZWxfY2F0YWxvZ1wiLCBcImxpc3RfbW9kZWxzXCIsIFwicHVibGljX2luZmVyZW5jZVwiLCBcImRpcmVjdF9kaXNwYXRjaFwiXSwgY29ubmVjdGlvbl9jb3VudDogMCwgc3RhdHVzOiBcImluYWN0aXZlXCIgfSxcbiAgeyBpZDogXCJyZXBsaWNhdGVcIiwgbmFtZTogXCJyZXBsaWNhdGVcIiwgZGlzcGxheV9uYW1lOiBcIlJlcGxpY2F0ZVwiLCBkZXNjcmlwdGlvbjogXCJSdW4gYW55IG9wZW4tc291cmNlIG1vZGVsXCIsIGF1dGhfdHlwZXM6IFtcImFwaV9rZXlcIl0sIGNhcGFiaWxpdGllczogW1wiaW5mZXJlbmNlXCIsIFwic3RyZWFtaW5nXCIsIFwibW9kZWxfY2F0YWxvZ1wiLCBcImxpc3RfbW9kZWxzXCIsIFwicHVibGljX2luZmVyZW5jZVwiLCBcImRpcmVjdF9kaXNwYXRjaFwiXSwgY29ubmVjdGlvbl9jb3VudDogMCwgc3RhdHVzOiBcImluYWN0aXZlXCIgfSxcbiAgeyBpZDogXCJ0YXZpbHlcIiwgbmFtZTogXCJ0YXZpbHlcIiwgZGlzcGxheV9uYW1lOiBcIlRhdmlseVwiLCBkZXNjcmlwdGlvbjogXCJUYXZpbHkgc2VhcmNoIEFQSVwiLCBhdXRoX3R5cGVzOiBbXCJhcGlfa2V5XCJdLCBjYXBhYmlsaXRpZXM6IFtcImluZmVyZW5jZVwiLCBcInN0cmVhbWluZ1wiLCBcIm1vZGVsX2NhdGFsb2dcIiwgXCJsaXN0X21vZGVsc1wiLCBcInB1YmxpY19pbmZlcmVuY2VcIiwgXCJkaXJlY3RfZGlzcGF0Y2hcIl0sIGNvbm5lY3Rpb25fY291bnQ6IDAsIHN0YXR1czogXCJpbmFjdGl2ZVwiIH0sXG4gIHsgaWQ6IFwidmxsbVwiLCBuYW1lOiBcInZsbG1cIiwgZGlzcGxheV9uYW1lOiBcInZMTE1cIiwgZGVzY3JpcHRpb246IFwidkxMTSBPcGVuQUktY29tcGF0aWJsZSBzZXJ2ZXJcIiwgYXV0aF90eXBlczogW1wibm9hdXRoXCJdLCBjYXBhYmlsaXRpZXM6IFtcImluZmVyZW5jZVwiLCBcInN0cmVhbWluZ1wiLCBcIm1vZGVsX2NhdGFsb2dcIiwgXCJsaXN0X21vZGVsc1wiLCBcInB1YmxpY19pbmZlcmVuY2VcIiwgXCJkaXJlY3RfZGlzcGF0Y2hcIl0sIGNvbm5lY3Rpb25fY291bnQ6IDAsIHN0YXR1czogXCJpbmFjdGl2ZVwiIH0sXG5dO1xuXG5leHBvcnQgZnVuY3Rpb24gZ2V0Q2F0YWxvZ01vZGVscyhwcm92aWRlcklkOiBzdHJpbmcpOiBDYXRhbG9nTW9kZWxbXSB7XG4gIHJldHVybiBwcm92aWRlckNhdGFsb2dbcHJvdmlkZXJJZF0gPz8gW107XG59XG5cbmV4cG9ydCBmdW5jdGlvbiBnZXRBbGxDYXRhbG9nTW9kZWxzKCk6IENhdGFsb2dNb2RlbFtdIHtcbiAgcmV0dXJuIE9iamVjdC52YWx1ZXMocHJvdmlkZXJDYXRhbG9nKS5mbGF0KCk7XG59XG5cbmV4cG9ydCBmdW5jdGlvbiBnZXRLbm93blByb3ZpZGVyKGlkOiBzdHJpbmcpOiBQcm92aWRlciB8IHVuZGVmaW5lZCB7XG4gIHJldHVybiBrbm93blByb3ZpZGVycy5maW5kKChwKSA9PiBwLmlkID09PSBpZCk7XG59XG4iXSwibWFwcGluZ3MiOiJBQUVBO0FBQ0E7O0FBSUEsTUFBTUEsZUFBK0MsR0FBRztFQUN0REMsTUFBTSxFQUFFLENBQ047SUFBRUMsRUFBRSxFQUFFLFFBQVE7SUFBRUMsUUFBUSxFQUFFLFFBQVE7SUFBRUMsSUFBSSxFQUFFLFFBQVE7SUFBRUMsVUFBVSxFQUFFLEdBQUc7SUFBRUMsV0FBVyxFQUFFLElBQUk7SUFBRUMsY0FBYyxFQUFFLE1BQU07SUFBRUMsV0FBVyxFQUFFLEtBQUs7SUFBRUMsU0FBUyxFQUFFO0VBQU0sQ0FBQyxFQUN0SjtJQUFFUCxFQUFFLEVBQUUsYUFBYTtJQUFFQyxRQUFRLEVBQUUsUUFBUTtJQUFFQyxJQUFJLEVBQUUsYUFBYTtJQUFFQyxVQUFVLEVBQUUsSUFBSTtJQUFFQyxXQUFXLEVBQUUsR0FBRztJQUFFQyxjQUFjLEVBQUUsTUFBTTtJQUFFQyxXQUFXLEVBQUUsS0FBSztJQUFFQyxTQUFTLEVBQUU7RUFBTSxDQUFDLENBQ2pLO0VBQ0RDLFNBQVMsRUFBRSxDQUNUO0lBQUVSLEVBQUUsRUFBRSxpQkFBaUI7SUFBRUMsUUFBUSxFQUFFLFdBQVc7SUFBRUMsSUFBSSxFQUFFLGlCQUFpQjtJQUFFQyxVQUFVLEVBQUUsR0FBRztJQUFFQyxXQUFXLEVBQUUsSUFBSTtJQUFFQyxjQUFjLEVBQUUsTUFBTTtJQUFFQyxXQUFXLEVBQUUsS0FBSztJQUFFQyxTQUFTLEVBQUU7RUFBTSxDQUFDLEVBQzNLO0lBQUVQLEVBQUUsRUFBRSxlQUFlO0lBQUVDLFFBQVEsRUFBRSxXQUFXO0lBQUVDLElBQUksRUFBRSxlQUFlO0lBQUVDLFVBQVUsRUFBRSxJQUFJO0lBQUVDLFdBQVcsRUFBRSxJQUFJO0lBQUVDLGNBQWMsRUFBRSxNQUFN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RSxHQUFHO0lBQUVDLFdBQVcsRUFBRSxHQUFHO0lBQUVDLGNBQWMsRUFBRSxNQUFNO0lBQUVDLFdBQVcsRUFBRSxLQUFLO0lBQUVDLFNBQVMsRUFBRTtFQUFNLENBQUMsQ0FDL0o7RUFDRFcsSUFBSSxFQUFFLENBQ0o7SUFBRWxCLEVBQUUsRUFBRSxhQUFhO0lBQUVDLFFBQVEsRUFBRSxNQUFNO0lBQUVDLElBQUksRUFBRSxhQUFhO0lBQUVDLFVBQVUsRUFBRSxHQUFHO0lBQUVDLFdBQVcsRUFBRSxHQUFHO0lBQUVDLGNBQWMsRUFBRSxNQUFNO0lBQUVDLFdBQVcsRUFBRSxLQUFLO0lBQUVDLFNBQVMsRUFBRTtFQUFNLENBQUMsRUFDN0o7SUFBRVAsRUFBRSxFQUFFLGNBQWM7SUFBRUMsUUFBUSxFQUFFLE1BQU07SUFBRUMsSUFBSSxFQUFFLGNBQWM7SUFBRUMsVUFBVSxFQUFFLEdBQUc7SUFBRUMsV0FBVyxFQUFFLEdBQUc7SUFBRUMsY0FBYyxFQUFFLE1BQU07SUFBRUMsV0FBVyxFQUFFLEtBQUs7SUFBRUMsU0FBUyxFQUFFO0VBQU0sQ0FBQyxDQUNoSztFQUNEWSxHQUFHLEVBQUUsQ0FDSDtJQUFFbkIsRUFBRSxFQUFFLFVBQVU7SUFBRUMsUUFBUSxFQUFFLEtBQUs7SUFBRUMsSUFBSSxFQUFFLFVBQVU7SUFBRUMsVUFBVSxFQUFFLElBQUk7SUFBRUMsV0FBVyxFQUFFLEdBQUc7SUFBRUMsY0FBYyxFQUFFLE1BQU07SUFBRUMsV0FBVyxFQUFFLEtBQUs7SUFBRUMsU0FBUyxFQUFFO0VBQU0sQ0FBQyxDQUN4SjtFQUNEYSxRQUFRLEVBQUUsQ0FDUjtJQUFFcEIsRUFBRSxFQUFFLGFBQWE7SUFBRUMsUUFBUSxFQUFFLFVBQVU7SUFBRUMsSUFBSSxFQUFFLGFBQWE7SUFBRUMsVUFBVSxFQUFFLEdBQUc7SUFBRUMsV0FBVyxFQUFFLEdBQUc7SUFBRUMsY0FBYyxFQUFFLE1BQU07SUFBRUMsV0FBVyxFQUFFLEtBQUs7SUFBRUMsU0FBUyxFQUFFO0VBQU0sQ0FBQyxDQUNsSztFQUNEYyxNQUFNLEVBQUUs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