import type { Model, Provider } from "../../src/lib/types";

// Mock catalog fallback mirroring internal/modelcatalog/catalog.go.
// Provides default models and pricing per provider before any connection is added.

export type CatalogModel = Omit<Model, "id"> & { id: string };

const providerCatalog: Record<string, CatalogModel[]> = {
  openai: [
    { id: "gpt-4o", provider: "openai", name: "gpt-4o", input_cost: 2.5, output_cost: 10.0, context_window: 128000, is_disabled: false, is_custom: false },
    { id: "gpt-4o-mini", provider: "openai", name: "gpt-4o-mini", input_cost: 0.15, output_cost: 0.6, context_window: 128000, is_disabled: false, is_custom: false },
  ],
  anthropic: [
    { id: "claude-sonnet-4", provider: "anthropic", name: "claude-sonnet-4", input_cost: 3.0, output_cost: 15.0, context_window: 200000, is_disabled: false, is_custom: false },
    { id: "claude-opus-4", provider: "anthropic", name: "claude-opus-4", input_cost: 15.0, output_cost: 75.0, context_window: 200000, is_disabled: false, is_custom: false },
    { id: "claude-3-5-haiku-20241022", provider: "anthropic", name: "claude-3-5-haiku-20241022", input_cost: 0.8, output_cost: 4.0, context_window: 200000, is_disabled: false, is_custom: false },
  ],
  gemini: [
    { id: "gemini-2.5-flash", provider: "gemini", name: "gemini-2.5-flash", input_cost: 0.3, output_cost: 2.5, context_window: 1000000, is_disabled: false, is_custom: false },
    { id: "gemini-2.5-flash-lite", provider: "gemini", name: "gemini-2.5-flash-lite", input_cost: 0.1, output_cost: 0.4, context_window: 1000000, is_disabled: false, is_custom: false },
  ],
  groq: [
    { id: "llama-3.3-70b-versatile", provider: "groq", name: "llama-3.3-70b-versatile", input_cost: 0.59, output_cost: 0.79, context_window: 128000, is_disabled: false, is_custom: false },
    { id: "llama-3.1-8b-instant", provider: "groq", name: "llama-3.1-8b-instant", input_cost: 0.05, output_cost: 0.08, context_window: 128000, is_disabled: false, is_custom: false },
  ],
  mistral: [
    { id: "mistral-large-latest", provider: "mistral", name: "mistral-large-latest", input_cost: 2.0, output_cost: 6.0, context_window: 128000, is_disabled: false, is_custom: false },
    { id: "mistral-small-latest", provider: "mistral", name: "mistral-small-latest", input_cost: 0.1, output_cost: 0.3, context_window: 128000, is_disabled: false, is_custom: false },
    { id: "magistral-small-latest", provider: "mistral", name: "magistral-small-latest", input_cost: 0.5, output_cost: 1.5, context_window: 128000, is_disabled: false, is_custom: false },
    { id: "ministral-8b-latest", provider: "mistral", name: "ministral-8b-latest", input_cost: 0.15, output_cost: 0.15, context_window: 128000, is_disabled: false, is_custom: false },
  ],
  nebius: [
    { id: "meta-llama/Llama-3.3-70B-Instruct", provider: "nebius", name: "meta-llama/Llama-3.3-70B-Instruct", input_cost: 0.13, output_cost: 0.4, context_window: 128000, is_disabled: false, is_custom: false },
  ],
  nvidia: [
    { id: "meta/llama-3.1-8b-instruct", provider: "nvidia", name: "meta/llama-3.1-8b-instruct", input_cost: 0.1, output_cost: 0.1, context_window: 128000, is_disabled: false, is_custom: false },
  ],
  openrouter: [
    { id: "openai/gpt-4o", provider: "openrouter", name: "openai/gpt-4o", input_cost: 2.5, output_cost: 10.0, context_window: 128000, is_disabled: false, is_custom: false },
    { id: "openai/gpt-4o-mini", provider: "openrouter", name: "openai/gpt-4o-mini", input_cost: 0.15, output_cost: 0.6, context_window: 128000, is_disabled: false, is_custom: false },
  ],
  deepseek: [
    { id: "deepseek-chat", provider: "deepseek", name: "deepseek-chat", input_cost: 0.27, output_cost: 1.1, context_window: 64000, is_disabled: false, is_custom: false },
    { id: "deepseek-reasoner", provider: "deepseek", name: "deepseek-reasoner", input_cost: 0.55, output_cost: 2.19, context_window: 64000, is_disabled: false, is_custom: false },
  ],
  perplexity: [
    { id: "sonar", provider: "perplexity", name: "sonar", input_cost: 1.0, output_cost: 1.0, context_window: 128000, is_disabled: false, is_custom: false },
    { id: "sonar-pro", provider: "perplexity", name: "sonar-pro", input_cost: 3.0, output_cost: 15.0, context_window: 128000, is_disabled: false, is_custom: false },
    { id: "sonar-reasoning-pro", provider: "perplexity", name: "sonar-reasoning-pro", input_cost: 2.0, output_cost: 8.0, context_window: 128000, is_disabled: false, is_custom: false },
  ],
  minimax: [
    { id: "MiniMax-M3", provider: "minimax", name: "MiniMax-M3", input_cost: 0.3, output_cost: 1.2, context_window: 128000, is_disabled: false, is_custom: false },
  ],
  qwen: [
    { id: "qwen3.7-max", provider: "qwen", name: "qwen3.7-max", input_cost: 2.5, output_cost: 7.5, context_window: 128000, is_disabled: false, is_custom: false },
    { id: "qwen3.6-plus", provider: "qwen", name: "qwen3.6-plus", input_cost: 0.5, output_cost: 3.0, context_window: 128000, is_disabled: false, is_custom: false },
  ],
  xai: [
    { id: "grok-4.3", provider: "xai", name: "grok-4.3", input_cost: 1.25, output_cost: 2.5, context_window: 128000, is_disabled: false, is_custom: false },
  ],
  cerebras: [
    { id: "llama3.1-8b", provider: "cerebras", name: "llama3.1-8b", input_cost: 0.1, output_cost: 0.1, context_window: 128000, is_disabled: false, is_custom: false },
  ],
  cohere: [
    { id: "command-r-08-2024", provider: "cohere", name: "command-r-08-2024", input_cost: 0.15, output_cost: 0.6, context_window: 128000, is_disabled: false, is_custom: false },
  ],
  fireworks: [
    { id: "accounts/fireworks/models/deepseek-v4-flash", provider: "fireworks", name: "accounts/fireworks/models/deepseek-v4-flash", input_cost: 0.14, output_cost: 0.28, context_window: 128000, is_disabled: false, is_custom: false },
    { id: "accounts/fireworks/models/llama-v3p1-70b-instruct", provider: "fireworks", name: "accounts/fireworks/models/llama-v3p1-70b-instruct", input_cost: 0.3, output_cost: 1.2, context_window: 128000, is_disabled: false, is_custom: false },
    { id: "accounts/fireworks/models/llama-v3p3-70b-instruct", provider: "fireworks", name: "accounts/fireworks/models/llama-v3p3-70b-instruct", input_cost: 0.9, output_cost: 0.9, context_window: 128000, is_disabled: false, is_custom: false },
  ],
  together: [
    { id: "meta-llama/Llama-3.3-70B-Instruct-Turbo", provider: "together", name: "meta-llama/Llama-3.3-70B-Instruct-Turbo", input_cost: 1.04, output_cost: 1.04, context_window: 128000, is_disabled: false, is_custom: false },
    { id: "meta-llama/Meta-Llama-3-8B-Instruct-Lite", provider: "together", name: "meta-llama/Meta-Llama-3-8B-Instruct-Lite", input_cost: 0.1, output_cost: 0.1, context_window: 128000, is_disabled: false, is_custom: false },
  ],
  ollama: [
    { id: "llama3.1:8b", provider: "ollama", name: "llama3.1:8b", input_cost: 0, output_cost: 0, context_window: 128000, is_disabled: false, is_custom: false },
    { id: "llama3.3:70b", provider: "ollama", name: "llama3.3:70b", input_cost: 0, output_cost: 0, context_window: 128000, is_disabled: false, is_custom: false },
  ],
  vertex: [
    { id: "vertex/gemini-2.5-flash", provider: "vertex", name: "vertex/gemini-2.5-flash", input_cost: 0.3, output_cost: 2.5, context_window: 1000000, is_disabled: false, is_custom: false },
    { id: "vertex/gemini-2.5-flash-lite", provider: "vertex", name: "vertex/gemini-2.5-flash-lite", input_cost: 0.1, output_cost: 0.4, context_window: 1000000, is_disabled: false, is_custom: false },
  ],
  "vercel-ai-gateway": [
    { id: "anthropic/claude-sonnet-4.5", provider: "vercel-ai-gateway", name: "anthropic/claude-sonnet-4.5", input_cost: 3.0, output_cost: 15.0, context_window: 200000, is_disabled: false, is_custom: false },
  ],
  bedrock: [
    { id: "anthropic.claude-3-5-haiku-20241022-v1:0", provider: "bedrock", name: "anthropic.claude-3-5-haiku-20241022-v1:0", input_cost: 0.8, output_cost: 4.0, context_window: 200000, is_disabled: false, is_custom: false },
  ],
  huggingface: [
    { id: "meta-llama/Llama-3.3-70B-Instruct:groq", provider: "huggingface", name: "meta-llama/Llama-3.3-70B-Instruct:groq", input_cost: 0.59, output_cost: 0.79, context_window: 128000, is_disabled: false, is_custom: false },
  ],
};

// Known providers matching the backend matrix (providerinfo.ProviderMatrix).
export const knownProviders: Provider[] = [
  { id: "openai", name: "openai", display_name: "OpenAI", description: "GPT-4, GPT-3.5, DALL-E, Whisper", auth_types: ["api_key"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "anthropic", name: "anthropic", display_name: "Anthropic", description: "Claude 3.5 Sonnet, Opus, Haiku", auth_types: ["api_key"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "gemini", name: "gemini", display_name: "Google AI", description: "Gemini Pro, Flash, Ultra", auth_types: ["api_key", "oauth"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "azure", name: "azure", display_name: "Azure OpenAI", description: "Enterprise GPT-4 via Azure", auth_types: ["api_key"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "bedrock", name: "bedrock", display_name: "AWS Bedrock", description: "Amazon Claude, Llama, Titan", auth_types: ["api_key", "custom"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "cerebras", name: "cerebras", display_name: "Cerebras", description: "Fast inference on Cerebras hardware", auth_types: ["api_key"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "cohere", name: "cohere", display_name: "Cohere", description: "Command, Embed, Rerank", auth_types: ["api_key"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "deepseek", name: "deepseek", display_name: "DeepSeek", description: "DeepSeek V3, Coder", auth_types: ["api_key"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "fireworks", name: "fireworks", display_name: "Fireworks AI", description: "Fast open-source inference", auth_types: ["api_key"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "gemini", name: "gemini", display_name: "Gemini", description: "Google Gemini models", auth_types: ["api_key", "oauth"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "groq", name: "groq", display_name: "Groq", description: "Ultra-fast LLM inference", auth_types: ["api_key"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "huggingface", name: "huggingface", display_name: "Hugging Face", description: "Inference API and endpoints", auth_types: ["api_key"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "minimax", name: "minimax", display_name: "MiniMax", description: "MiniMax M3 and multi-modal models", auth_types: ["api_key"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "mistral", name: "mistral", display_name: "Mistral AI", description: "Mistral Large, Medium, Small", auth_types: ["api_key"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "nebius", name: "nebius", display_name: "Nebius", description: "Nebius AI inference", auth_types: ["api_key"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "nvidia", name: "nvidia", display_name: "NVIDIA", description: "NVIDIA NIM inference", auth_types: ["api_key"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "ollama", name: "ollama", display_name: "Ollama", description: "Local open-source models", auth_types: ["noauth"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "openrouter", name: "openrouter", display_name: "OpenRouter", description: "Unified API for 100+ models", auth_types: ["api_key"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "perplexity", name: "perplexity", display_name: "Perplexity", description: "Search-augmented LLMs", auth_types: ["api_key"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "qwen", name: "qwen", display_name: "Qwen", description: "Alibaba Qwen models", auth_types: ["api_key"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "together", name: "together", display_name: "Together AI", description: "Open-source model hub", auth_types: ["api_key"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "vertex", name: "vertex", display_name: "Google Vertex", description: "Gemini on GCP", auth_types: ["oauth"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "xai", name: "xai", display_name: "xAI", description: "Grok models", auth_types: ["api_key"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "alibaba", name: "alibaba", display_name: "Alibaba", description: "Qwen and Tongyi models", auth_types: ["api_key"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "github-copilot", name: "github-copilot", display_name: "GitHub Copilot", description: "GitHub Copilot Chat", auth_types: ["oauth"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "kimi", name: "kimi", display_name: "Kimi", description: "Moonshot Kimi models", auth_types: ["api_key"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "zhipu", name: "zhipu", display_name: "Zhipu", description: "GLM models", auth_types: ["api_key"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "cloudflare-ai-gateway", name: "cloudflare-ai-gateway", display_name: "Cloudflare AI Gateway", description: "Cloudflare AI Gateway", auth_types: ["api_key"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "kagi", name: "kagi", display_name: "Kagi", description: "Kagi search and summarization", auth_types: ["api_key"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "litellm", name: "litellm", display_name: "LiteLLM", description: "LiteLLM proxy", auth_types: ["api_key"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "lm-studio", name: "lm-studio", display_name: "LM Studio", description: "Local LM Studio server", auth_types: ["noauth"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "ollama-cloud", name: "ollama-cloud", display_name: "Ollama Cloud", description: "Ollama Cloud inference", auth_types: ["api_key"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "opencode", name: "opencode", display_name: "Opencode", description: "Opencode models", auth_types: ["api_key"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "replicate", name: "replicate", display_name: "Replicate", description: "Run any open-source model", auth_types: ["api_key"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "tavily", name: "tavily", display_name: "Tavily", description: "Tavily search API", auth_types: ["api_key"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
  { id: "vllm", name: "vllm", display_name: "vLLM", description: "vLLM OpenAI-compatible server", auth_types: ["noauth"], capabilities: ["inference", "streaming", "model_catalog", "list_models", "public_inference", "direct_dispatch"], connection_count: 0, status: "inactive" },
];

export function getCatalogModels(providerId: string): CatalogModel[] {
  return providerCatalog[providerId] ?? [];
}

export function getAllCatalogModels(): CatalogModel[] {
  return Object.values(providerCatalog).flat();
}

export function getKnownProvider(id: string): Provider | undefined {
  return knownProviders.find((p) => p.id === id);
}
