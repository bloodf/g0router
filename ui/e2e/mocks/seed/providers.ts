import type { Provider } from "../../src/lib/types";

export function seedProviders(): Provider[] {
  return [
    { id: "openai", name: "openai", display_name: "OpenAI", description: "GPT-4, GPT-3.5, DALL-E, Whisper", auth_types: ["api_key"], capabilities: ["chat", "images", "audio", "embeddings"], connection_count: 2, status: "active" },
    { id: "anthropic", name: "anthropic", display_name: "Anthropic", description: "Claude 3.5 Sonnet, Opus, Haiku", auth_types: ["api_key"], capabilities: ["chat", "vision"], connection_count: 1, status: "active" },
    { id: "google", name: "google", display_name: "Google AI", description: "Gemini Pro, Flash, Ultra", auth_types: ["api_key", "oauth"], capabilities: ["chat", "vision", "embeddings"], connection_count: 1, status: "active" },
    { id: "azure", name: "azure", display_name: "Azure OpenAI", description: "Enterprise GPT-4 via Azure", auth_types: ["api_key"], capabilities: ["chat", "embeddings"], connection_count: 0, status: "inactive" },
    { id: "cohere", name: "cohere", display_name: "Cohere", description: "Command, Embed, Rerank", auth_types: ["api_key"], capabilities: ["chat", "embeddings"], connection_count: 0, status: "inactive" },
    { id: "mistral", name: "mistral", display_name: "Mistral AI", description: "Mistral Large, Medium, Small", auth_types: ["api_key"], capabilities: ["chat"], connection_count: 0, status: "inactive" },
    { id: "groq", name: "groq", display_name: "Groq", description: "Ultra-fast LLM inference", auth_types: ["api_key"], capabilities: ["chat"], connection_count: 1, status: "active" },
    { id: "ollama", name: "ollama", display_name: "Ollama", description: "Local open-source models", auth_types: ["noauth"], capabilities: ["chat", "embeddings"], connection_count: 0, status: "inactive" },
    { id: "bedrock", name: "bedrock", display_name: "AWS Bedrock", description: "Amazon Claude, Llama, Titan", auth_types: ["api_key", "custom"], capabilities: ["chat", "embeddings"], connection_count: 0, status: "inactive" },
    { id: "vertex", name: "vertex", display_name: "Google Vertex", description: "Gemini on GCP", auth_types: ["oauth"], capabilities: ["chat", "vision"], connection_count: 0, status: "inactive" },
    { id: "deepseek", name: "deepseek", display_name: "DeepSeek", description: "DeepSeek V3, Coder", auth_types: ["api_key"], capabilities: ["chat"], connection_count: 0, status: "inactive" },
    { id: "perplexity", name: "perplexity", display_name: "Perplexity", description: "Search-augmented LLMs", auth_types: ["api_key"], capabilities: ["chat"], connection_count: 0, status: "inactive" },
    { id: "fireworks", name: "fireworks", display_name: "Fireworks AI", description: "Fast open-source inference", auth_types: ["api_key"], capabilities: ["chat", "embeddings"], connection_count: 0, status: "inactive" },
    { id: "together", name: "together", display_name: "Together AI", description: "Open-source model hub", auth_types: ["api_key"], capabilities: ["chat", "images"], connection_count: 0, status: "inactive" },
    { id: "replicate", name: "replicate", display_name: "Replicate", description: "Run any open-source model", auth_types: ["api_key"], capabilities: ["chat", "images", "audio"], connection_count: 0, status: "inactive" },
    { id: "huggingface", name: "huggingface", display_name: "Hugging Face", description: "Inference API and endpoints", auth_types: ["api_key"], capabilities: ["chat", "embeddings"], connection_count: 0, status: "inactive" },
    { id: "ai21", name: "ai21", display_name: "AI21 Labs", description: "Jamba, Jurassic models", auth_types: ["api_key"], capabilities: ["chat"], connection_count: 0, status: "inactive" },
    { id: "moonshot", name: "moonshot", display_name: "Moonshot AI", description: "Kimi K1.5 long-context models", auth_types: ["api_key"], capabilities: ["chat"], connection_count: 0, status: "inactive" },
    { id: "xai", name: "xai", display_name: "xAI", description: "Grok models", auth_types: ["api_key"], capabilities: ["chat", "vision"], connection_count: 0, status: "inactive" },
    { id: "openrouter", name: "openrouter", display_name: "OpenRouter", description: "Unified API for 100+ models", auth_types: ["api_key"], capabilities: ["chat", "images"], connection_count: 1, status: "active" },
  ];
}
