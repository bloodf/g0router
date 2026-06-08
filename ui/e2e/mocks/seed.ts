import type {
  Provider, Model, Settings, Tunnel, Connection, ApiKey, UsageLog,
  AuditLog, Quota, ChatSession, Combo, Alias, PricingOverride,
  RoutingRule, Team, VirtualKey, ConsoleLogEntry, User,
} from "../../src/lib/types";

export function seedUsers(): User[] {
  return [
    { id: "user-1", username: "admin", display_name: "Administrator", role: "admin", password: "123456" },
  ];
}

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

export function seedModels(): Model[] {
  return [
    { id: "gpt-4o", provider: "openai", name: "gpt-4o", input_cost: 2.5, output_cost: 10.0, context_window: 128000, is_disabled: false, is_custom: false },
    { id: "gpt-4o-mini", provider: "openai", name: "gpt-4o-mini", input_cost: 0.15, output_cost: 0.6, context_window: 128000, is_disabled: false, is_custom: false },
    { id: "gpt-4-turbo", provider: "openai", name: "gpt-4-turbo", input_cost: 10.0, output_cost: 30.0, context_window: 128000, is_disabled: false, is_custom: false },
    { id: "claude-sonnet-4", provider: "anthropic", name: "claude-3-5-sonnet-20241022", input_cost: 3.0, output_cost: 15.0, context_window: 200000, is_disabled: false, is_custom: false },
    { id: "claude-haiku", provider: "anthropic", name: "claude-3-haiku-20240307", input_cost: 0.25, output_cost: 1.25, context_window: 200000, is_disabled: false, is_custom: false },
    { id: "gemini-2.5-pro", provider: "google", name: "gemini-2.5-pro-preview-03-25", input_cost: 1.25, output_cost: 10.0, context_window: 1000000, is_disabled: false, is_custom: false },
    { id: "gemini-2.5-flash", provider: "google", name: "gemini-2.5-flash-preview-04-17", input_cost: 0.15, output_cost: 0.6, context_window: 1000000, is_disabled: false, is_custom: false },
    { id: "llama-3-70b", provider: "groq", name: "llama-3.1-70b-versatile", input_cost: 0.59, output_cost: 0.79, context_window: 128000, is_disabled: false, is_custom: false },
    { id: "mixtral-8x22b", provider: "groq", name: "mixtral-8x22b-instruct", input_cost: 0.9, output_cost: 0.9, context_window: 64000, is_disabled: false, is_custom: false },
    { id: "mistral-large", provider: "mistral", name: "mistral-large-latest", input_cost: 2.0, output_cost: 6.0, context_window: 128000, is_disabled: false, is_custom: false },
    { id: "command-r", provider: "cohere", name: "command-r", input_cost: 0.5, output_cost: 1.5, context_window: 128000, is_disabled: false, is_custom: false },
    { id: "deepseek-chat", provider: "deepseek", name: "deepseek-chat", input_cost: 0.14, output_cost: 0.28, context_window: 64000, is_disabled: false, is_custom: false },
    { id: "grok-2", provider: "xai", name: "grok-2", input_cost: 5.0, output_cost: 15.0, context_window: 128000, is_disabled: false, is_custom: false },
    { id: "kimi-k1.5", provider: "moonshot", name: "kimi-k1.5", input_cost: 0.5, output_cost: 2.0, context_window: 256000, is_disabled: false, is_custom: false },
    { id: "jamba-1.5", provider: "ai21", name: "jamba-1.5-large", input_cost: 2.0, output_cost: 8.0, context_window: 256000, is_disabled: false, is_custom: false },
    { id: "openrouter-gpt-4o", provider: "openrouter", name: "openai/gpt-4o", input_cost: 5.0, output_cost: 15.0, context_window: 128000, is_disabled: false, is_custom: false },
    { id: "openrouter-claude", provider: "openrouter", name: "anthropic/claude-3.5-sonnet", input_cost: 3.0, output_cost: 15.0, context_window: 200000, is_disabled: false, is_custom: false },
    { id: "azure-gpt-4o", provider: "azure", name: "gpt-4o", input_cost: 5.0, output_cost: 15.0, context_window: 128000, is_disabled: false, is_custom: false },
    { id: "llama-3.1-8b", provider: "ollama", name: "llama3.1", input_cost: 0, output_cost: 0, context_window: 128000, is_disabled: false, is_custom: false },
    { id: "dall-e-3", provider: "openai", name: "dall-e-3", input_cost: 0.04, output_cost: 0.08, context_window: 0, is_disabled: false, is_custom: false },
  ];
}

export function seedSettings(): Settings {
  return {
    require_api_key: false,
    require_login: true,
    rtk_enabled: false,
    caveman_enabled: false,
    caveman_level: "lite",
    enable_request_logs: true,
    log_retention_days: 30,
    cache_enabled: false,
    cache_ttl_seconds: 3600,
    proxy_url: "",
    notify_webhook_url: "",
    notify_on_reauth: false,
    allowed_sources: ["local", "lan"],
    tunnel_dashboard_access: false,
    theme: "system",
    language: "en",
  };
}

export function seedTunnels(): Tunnel[] {
  return [
    { type: "cloudflare", is_enabled: false, url: "https://g0router-demo.trycloudflare.com", status: "inactive" },
    { type: "tailscale", is_enabled: false, url: "http://g0router.tailnet.ts.net", status: "inactive" },
  ];
}

export function seedConnections(): Connection[] {
  return [
    { id: "conn-1", provider: "openai", name: "OpenAI Prod", auth_type: "api_key", is_active: true, models: ["gpt-4o", "gpt-4o-mini"], priority: 1, needs_reauth: false },
    { id: "conn-2", provider: "openai", name: "OpenAI Dev", auth_type: "api_key", is_active: true, models: ["gpt-4o-mini"], priority: 2, needs_reauth: false },
    { id: "conn-3", provider: "anthropic", name: "Anthropic Main", auth_type: "api_key", is_active: true, models: ["claude-sonnet-4", "claude-haiku"], priority: 1, needs_reauth: false },
    { id: "conn-4", provider: "google", name: "Google AI", auth_type: "api_key", is_active: true, models: ["gemini-2.5-pro", "gemini-2.5-flash"], priority: 1, needs_reauth: false },
    { id: "conn-5", provider: "groq", name: "Groq Fast", auth_type: "api_key", is_active: true, models: ["llama-3-70b", "mixtral-8x22b"], priority: 1, needs_reauth: false },
    { id: "conn-6", provider: "openrouter", name: "OpenRouter", auth_type: "api_key", is_active: true, models: ["openrouter-gpt-4o", "openrouter-claude"], priority: 1, needs_reauth: false },
  ];
}

export function seedKeys(): ApiKey[] {
  return [
    { id: "key-1", name: "Default Key", prefix: "sk-g0def", full_key: "sk-g0def-1234567890abcdef", scopes: ["chat", "embeddings"], rpm_limit: 1000, tpm_limit: 1000000, daily_spend_cap: 100, is_active: true, created_at: new Date(Date.now() - 86400000 * 7).toISOString() },
    { id: "key-2", name: "Staging Key", prefix: "sk-g0stg", scopes: ["chat"], rpm_limit: 100, is_active: true, created_at: new Date(Date.now() - 86400000 * 3).toISOString() },
  ];
}

export function seedVirtualKeys(): VirtualKey[] {
  return [
    { id: "vk-1", name: "Team Alpha", prefix: "vk-alpha", budget_usd: 500, budget_used_usd: 127.5, budget_period: "monthly", rate_limit_rpm: 500, is_active: true },
    { id: "vk-2", name: "Team Beta", prefix: "vk-beta", budget_usd: 200, budget_used_usd: 45.0, budget_period: "monthly", rate_limit_rpm: 200, is_active: true },
  ];
}

export function seedTeams(): Team[] {
  return [
    { id: "team-1", name: "Engineering", budget_usd: 2000, budget_used_usd: 850, keys_count: 5, members: 12 },
    { id: "team-2", name: "Data Science", budget_usd: 1500, budget_used_usd: 420, keys_count: 3, members: 8 },
  ];
}

export function seedCombos(): Combo[] {
  return [
    { id: "combo-1", name: "Fast + Cheap", strategy: "fallback", steps: [{ provider: "groq", model: "llama-3-70b" }, { provider: "openai", model: "gpt-4o-mini" }], is_active: true },
    { id: "combo-2", name: "Best Quality", strategy: "fallback", steps: [{ provider: "openai", model: "gpt-4o" }, { provider: "anthropic", model: "claude-sonnet-4" }], is_active: true },
  ];
}

export function seedAliases(): Alias[] {
  return [
    { id: "alias-1", alias: "gpt4", provider: "openai", model: "gpt-4o" },
    { id: "alias-2", alias: "claude", provider: "anthropic", model: "claude-sonnet-4" },
    { id: "alias-3", alias: "gemini", provider: "google", model: "gemini-2.5-pro" },
  ];
}

export function seedPricing(): PricingOverride[] {
  return [
    { id: "price-1", provider: "openai", model: "gpt-4o", input_cost: 2.0, output_cost: 8.0 },
    { id: "price-2", provider: "anthropic", model: "claude-sonnet-4", input_cost: 2.5, output_cost: 12.0 },
  ];
}

export function seedRoutingRules(): RoutingRule[] {
  return [
    { id: "rule-1", name: "Route GPT-4 to OpenAI", priority: 1, condition: { field: "model", operator: "equals", value: "gpt-4o" }, target_provider: "openai", is_active: true },
    { id: "rule-2", name: "Route Claude to Anthropic", priority: 2, condition: { field: "model", operator: "equals", value: "claude-sonnet-4" }, target_provider: "anthropic", is_active: true },
  ];
}

export function seedUsageLogs(): UsageLog[] {
  const providers = ["openai", "anthropic", "google", "groq"];
  const models = ["gpt-4o", "claude-sonnet-4", "gemini-2.5-pro", "llama-3-70b"];
  const logs: UsageLog[] = [];
  for (let i = 0; i < 25; i++) {
    const ok = Math.random() > 0.1;
    logs.push({
      id: `log-${i}`,
      timestamp: new Date(Date.now() - Math.random() * 86400000 * 7).toISOString(),
      provider: providers[i % providers.length],
      model: models[i % models.length],
      api_key_id: "key-1",
      api_key_name: "Default Key",
      status: ok ? "success" : "error",
      status_code: ok ? 200 : 429,
      prompt_tokens: Math.floor(Math.random() * 2000) + 50,
      completion_tokens: Math.floor(Math.random() * 1000) + 20,
      total_tokens: 0,
      cost_usd: Math.random() * 0.5,
      latency_ms: Math.floor(Math.random() * 2000) + 100,
      rtk_enabled: false,
      caveman_enabled: false,
    });
    logs[i].total_tokens = logs[i].prompt_tokens + logs[i].completion_tokens;
  }
  return logs;
}

export function seedAuditLogs(): AuditLog[] {
  return [
    { id: "audit-1", timestamp: new Date(Date.now() - 3600000).toISOString(), actor: "admin", action: "create_key", target: "key-1", details: "Created Default Key" },
    { id: "audit-2", timestamp: new Date(Date.now() - 7200000).toISOString(), actor: "admin", action: "copy_key", target: "key-1" },
    { id: "audit-3", timestamp: new Date(Date.now() - 86400000).toISOString(), actor: "admin", action: "enable_key", target: "key-2" },
    { id: "audit-4", timestamp: new Date(Date.now() - 172800000).toISOString(), actor: "admin", action: "export_keys", target: "all", details: "Exported 2 keys" },
    { id: "audit-5", timestamp: new Date(Date.now() - 259200000).toISOString(), actor: "admin", action: "regenerate_key", target: "key-1" },
  ];
}

export function seedQuota(): Quota[] {
  return [
    { connection_id: "conn-1", provider: "openai", connection_name: "OpenAI Prod", account_label: "org-123", plan: "pro", used: 45000, limit: 100000, unit: "tokens", reset_at: new Date(Date.now() + 86400000).toISOString(), is_active: true },
    { connection_id: "conn-3", provider: "anthropic", connection_name: "Anthropic Main", account_label: "team-alpha", plan: "pro", used: 23000, limit: 50000, unit: "tokens", reset_at: new Date(Date.now() + 86400000).toISOString(), is_active: true },
    { connection_id: "conn-4", provider: "google", connection_name: "Google AI", plan: "free", used: 12000, limit: 15000, unit: "tokens", reset_at: new Date(Date.now() + 86400000).toISOString(), is_active: true },
    { connection_id: "conn-5", provider: "groq", connection_name: "Groq Fast", plan: "free", used: 8000, limit: 0, unit: "tokens", reset_at: new Date(Date.now() + 86400000).toISOString(), is_active: true },
  ];
}

export function seedChatSessions(): ChatSession[] {
  return [
    { id: "chat-1", title: "Python helper", model: "gpt-4o", provider: "openai", messages: [{ role: "user", content: "How do I parse JSON?" }, { role: "assistant", content: "Use JSON.parse()..." }], created_at: new Date(Date.now() - 86400000).toISOString(), updated_at: new Date(Date.now() - 3600000).toISOString() },
    { id: "chat-2", title: "Code review", model: "claude-sonnet-4", provider: "anthropic", messages: [{ role: "user", content: "Review this function" }], created_at: new Date(Date.now() - 172800000).toISOString(), updated_at: new Date(Date.now() - 172800000).toISOString() },
  ];
}

export function seedConsoleLogs(): ConsoleLogEntry[] {
  return [
    { timestamp: new Date().toISOString(), level: "INFO", message: "Server started on port 20128" },
    { timestamp: new Date(Date.now() - 5000).toISOString(), level: "INFO", message: "Connected to OpenAI (2 models)" },
    { timestamp: new Date(Date.now() - 10000).toISOString(), level: "INFO", message: "Connected to Anthropic (2 models)" },
    { timestamp: new Date(Date.now() - 15000).toISOString(), level: "WARN", message: "Provider ollama unreachable, falling back to catalog" },
    { timestamp: new Date(Date.now() - 20000).toISOString(), level: "INFO", message: "Loaded 20 providers from catalog" },
    { timestamp: new Date(Date.now() - 30000).toISOString(), level: "DEBUG", message: "Cache miss for embedding model list" },
  ];
}
