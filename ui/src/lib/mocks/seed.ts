import type {
  AlertChannel,
  Alias,
  ApiKey,
  AuditLog,
  ChatSession,
  Combo,
  Connection,
  FeatureFlag,
  Guardrails,
  McpAccount,
  McpInstance,
  McpTool,
  McpToolGroup,
  MitmTool,
  Model,
  ModelLimit,
  PricingOverride,
  PromptTemplate,
  Provider,
  ProxyPool,
  Quota,
  RoutingRule,
  Settings,
  Skill,
  Team,
  Tunnel,
  UsageLog,
  User,
  VirtualKey,
} from "./types";

const rand = (min: number, max: number) =>
  Math.floor(Math.random() * (max - min + 1)) + min;
const pick = <T>(arr: T[]): T => arr[Math.floor(Math.random() * arr.length)];
const id = () => Math.random().toString(36).slice(2, 10);
const now = () => new Date().toISOString();
const daysAgo = (n: number) =>
  new Date(Date.now() - n * 86400000).toISOString();
const daysFromNow = (n: number) =>
  new Date(Date.now() + n * 86400000).toISOString();

export const PROVIDER_CATALOG: Provider[] = [
  {
    id: "openai",
    name: "openai",
    display_name: "OpenAI",
    description: "GPT-4, GPT-4o, o1, o3 and embeddings.",
    auth_types: ["api_key", "oauth"],
    capabilities: ["chat", "embeddings", "vision", "tools", "audio"],
    connection_count: 3,
    status: "active",
  },
  {
    id: "anthropic",
    name: "anthropic",
    display_name: "Anthropic",
    description: "Claude Sonnet, Opus and Haiku family.",
    auth_types: ["api_key", "oauth"],
    capabilities: ["chat", "vision", "tools"],
    connection_count: 2,
    status: "active",
  },
  {
    id: "google",
    name: "google",
    display_name: "Google",
    description: "Gemini 1.5, 2.0 and 2.5 models via AI Studio / Vertex.",
    auth_types: ["api_key", "oauth"],
    capabilities: ["chat", "vision", "tools", "audio"],
    connection_count: 2,
    status: "active",
  },
  {
    id: "mistral",
    name: "mistral",
    display_name: "Mistral",
    description: "Mistral Large, Codestral, Mixtral.",
    auth_types: ["api_key"],
    capabilities: ["chat", "tools"],
    connection_count: 1,
    status: "needs_reauth",
  },
  {
    id: "groq",
    name: "groq",
    display_name: "Groq",
    description: "Ultra-low-latency Llama and Mixtral inference.",
    auth_types: ["api_key"],
    capabilities: ["chat"],
    connection_count: 1,
    status: "active",
  },
  {
    id: "openrouter",
    name: "openrouter",
    display_name: "OpenRouter",
    description: "Routed access to 300+ third-party models.",
    auth_types: ["api_key"],
    capabilities: ["chat", "tools"],
    connection_count: 1,
    status: "active",
  },
  {
    id: "cohere",
    name: "cohere",
    display_name: "Cohere",
    description: "Command, Embed and Rerank.",
    auth_types: ["api_key"],
    capabilities: ["chat", "embeddings", "rerank"],
    connection_count: 1,
    status: "active",
  },
  {
    id: "xai",
    name: "xai",
    display_name: "xAI",
    description: "Grok and Grok-Vision models.",
    auth_types: ["api_key"],
    capabilities: ["chat", "vision"],
    connection_count: 1,
    status: "error",
  },
  {
    id: "together",
    name: "together",
    display_name: "Together AI",
    description: "Open-source models served on Together's GPU cloud.",
    auth_types: ["api_key"],
    capabilities: ["chat", "embeddings"],
    connection_count: 1,
    status: "active",
  },
  {
    id: "deepseek",
    name: "deepseek",
    display_name: "DeepSeek",
    description: "DeepSeek V3, Coder and R1 reasoning.",
    auth_types: ["api_key"],
    capabilities: ["chat", "tools"],
    connection_count: 1,
    status: "active",
  },
  {
    id: "ollama",
    name: "ollama",
    display_name: "Ollama",
    description: "Local model runtime (no auth).",
    auth_types: ["noauth"],
    capabilities: ["chat", "embeddings"],
    connection_count: 1,
    status: "active",
  },
  {
    id: "perplexity",
    name: "perplexity",
    display_name: "Perplexity",
    description: "Sonar models with built-in web search.",
    auth_types: ["api_key"],
    capabilities: ["chat", "search"],
    connection_count: 1,
    status: "active",
  },
  {
    id: "fireworks",
    name: "fireworks",
    display_name: "Fireworks",
    description: "Fast OSS model hosting.",
    auth_types: ["api_key"],
    capabilities: ["chat"],
    connection_count: 0,
    status: "inactive",
  },
  {
    id: "azure",
    name: "azure",
    display_name: "Azure OpenAI",
    description: "Azure-hosted OpenAI deployments.",
    auth_types: ["api_key", "custom"],
    capabilities: ["chat", "embeddings"],
    connection_count: 0,
    status: "inactive",
  },
  {
    id: "bedrock",
    name: "bedrock",
    display_name: "AWS Bedrock",
    description: "Claude, Llama and Titan via AWS.",
    auth_types: ["custom"],
    capabilities: ["chat", "embeddings"],
    connection_count: 0,
    status: "inactive",
  },
  {
    id: "vertex",
    name: "vertex",
    display_name: "Google Vertex AI",
    description: "Gemini and PaLM on GCP.",
    auth_types: ["custom"],
    capabilities: ["chat", "embeddings"],
    connection_count: 0,
    status: "inactive",
  },
];

const MODEL_CATALOG: Record<string, { name: string; in: number; out: number; ctx: number }[]> = {
  openai: [
    { name: "gpt-4o", in: 2.5, out: 10, ctx: 128000 },
    { name: "gpt-4o-mini", in: 0.15, out: 0.6, ctx: 128000 },
    { name: "gpt-4-turbo", in: 10, out: 30, ctx: 128000 },
    { name: "o1-preview", in: 15, out: 60, ctx: 128000 },
    { name: "o3-mini", in: 1.1, out: 4.4, ctx: 200000 },
  ],
  anthropic: [
    { name: "claude-sonnet-4", in: 3, out: 15, ctx: 200000 },
    { name: "claude-opus-4", in: 15, out: 75, ctx: 200000 },
    { name: "claude-haiku-3.5", in: 0.8, out: 4, ctx: 200000 },
  ],
  google: [
    { name: "gemini-2.5-pro", in: 1.25, out: 5, ctx: 2000000 },
    { name: "gemini-2.5-flash", in: 0.075, out: 0.3, ctx: 1000000 },
    { name: "gemini-2.0-flash", in: 0.05, out: 0.2, ctx: 1000000 },
  ],
  mistral: [
    { name: "mistral-large", in: 2, out: 6, ctx: 128000 },
    { name: "codestral", in: 0.3, out: 0.9, ctx: 32000 },
  ],
  groq: [
    { name: "llama-3.3-70b", in: 0.59, out: 0.79, ctx: 128000 },
    { name: "mixtral-8x7b", in: 0.24, out: 0.24, ctx: 32000 },
  ],
  openrouter: [{ name: "auto", in: 1, out: 3, ctx: 200000 }],
  cohere: [{ name: "command-r-plus", in: 2.5, out: 10, ctx: 128000 }],
  xai: [{ name: "grok-2", in: 2, out: 10, ctx: 128000 }],
  together: [{ name: "llama-3.1-405b", in: 3.5, out: 3.5, ctx: 128000 }],
  deepseek: [
    { name: "deepseek-v3", in: 0.27, out: 1.1, ctx: 64000 },
    { name: "deepseek-r1", in: 0.55, out: 2.19, ctx: 64000 },
  ],
  ollama: [{ name: "llama3.2", in: 0, out: 0, ctx: 8192 }],
  perplexity: [{ name: "sonar-large", in: 1, out: 1, ctx: 128000 }],
  fireworks: [],
  azure: [],
  bedrock: [],
  vertex: [],
};

export function seedAll() {
  const providers = [...PROVIDER_CATALOG];

  const connections: Connection[] = [];
  providers.forEach((p) => {
    for (let i = 0; i < p.connection_count; i++) {
      connections.push({
        id: id(),
        provider: p.id,
        name: `${p.display_name} #${i + 1}`,
        auth_type: pick(p.auth_types) as "oauth" | "api_key" | "noauth",
        is_active: p.status === "active" || (p.status === "needs_reauth" && i === 0),
        models: (MODEL_CATALOG[p.id] ?? []).map((m) => m.name),
        priority: i,
        last_error: p.status === "error" ? "401 Unauthorized" : undefined,
        needs_reauth: p.status === "needs_reauth",
        expires_at: daysFromNow(rand(7, 90)),
      });
    }
  });

  const models: Model[] = providers.flatMap((p) =>
    (MODEL_CATALOG[p.id] ?? []).map((m) => ({
      id: `${p.id}/${m.name}`,
      provider: p.id,
      name: m.name,
      input_cost: m.in,
      output_cost: m.out,
      context_window: m.ctx,
      is_disabled: false,
      is_custom: false,
    })),
  );

  const keys: ApiKey[] = Array.from({ length: 4 }).map((_, i) => {
    const fullKey = `sk-${id()}${id()}${id()}`;
    return {
      id: id(),
      name: ["Production", "Staging", "Dev", "Mobile App"][i],
      prefix: fullKey.slice(0, 10),
      full_key: fullKey,
      scopes: ["chat", "embeddings"],
      expires_at: i === 0 ? undefined : daysFromNow(rand(30, 365)),
      rpm_limit: [1000, 500, 100, 200][i],
      tpm_limit: [200000, 100000, 50000, 80000][i],
      daily_spend_cap: [50, 20, 5, 10][i],
      is_active: i !== 2,
      created_at: daysAgo(rand(1, 90)),
    };
  });

  const teams: Team[] = [
    { id: id(), name: "Engineering", budget_usd: 500, budget_used_usd: 234.5, keys_count: 5, members: 8 },
    { id: id(), name: "Data Science", budget_usd: 1200, budget_used_usd: 980.2, keys_count: 3, members: 4 },
    { id: id(), name: "Support", budget_usd: 100, budget_used_usd: 12.3, keys_count: 2, members: 3 },
  ];

  const vkeys: VirtualKey[] = Array.from({ length: 6 }).map((_, i) => ({
    id: id(),
    name: `vk-${["alpha", "beta", "gamma", "delta", "epsilon", "zeta"][i]}`,
    prefix: `vk-${id().slice(0, 6)}`,
    budget_usd: pick([10, 25, 50, 100, 200]),
    budget_used_usd: rand(2, 80) + Math.random(),
    budget_period: pick(["daily", "weekly", "monthly"] as const),
    rate_limit_rpm: 200,
    rate_limit_tpm: 50000,
    team_id: i % 2 === 0 ? teams[0].id : teams[1].id,
    is_active: i !== 5,
  }));

  const combos: Combo[] = [
    {
      id: id(),
      name: "default-chat",
      strategy: "fallback",
      steps: [
        { provider: "openai", model: "gpt-4o" },
        { provider: "anthropic", model: "claude-sonnet-4" },
        { provider: "google", model: "gemini-2.5-pro" },
      ],
      is_active: true,
    },
    {
      id: id(),
      name: "fast-cheap",
      strategy: "cheapest",
      steps: [
        { provider: "google", model: "gemini-2.5-flash" },
        { provider: "openai", model: "gpt-4o-mini" },
        { provider: "groq", model: "llama-3.3-70b" },
      ],
      is_active: true,
    },
    {
      id: id(),
      name: "code-rotation",
      strategy: "round_robin",
      steps: [
        { provider: "anthropic", model: "claude-sonnet-4" },
        { provider: "deepseek", model: "deepseek-v3" },
        { provider: "mistral", model: "codestral" },
      ],
      sticky_limit: 10,
      is_active: true,
    },
  ];

  const routingRules: RoutingRule[] = [
    {
      id: id(),
      name: "Route GPT-4 to combo",
      priority: 1,
      condition: { field: "model", operator: "starts_with", value: "gpt-4" },
      target_provider: "openai",
      target_model: "gpt-4o",
      is_active: true,
    },
    {
      id: id(),
      name: "Cheap fallback for unknown",
      priority: 10,
      condition: { field: "model", operator: "contains", value: "unknown" },
      target_provider: "google",
      target_model: "gemini-2.5-flash",
      is_active: true,
    },
  ];

  const aliases: Alias[] = [
    { id: id(), alias: "fast", provider: "google", model: "gemini-2.5-flash", created_at: daysAgo(5) },
    { id: id(), alias: "smart", provider: "anthropic", model: "claude-sonnet-4", created_at: daysAgo(10) },
    { id: id(), alias: "cheap", provider: "openai", model: "gpt-4o-mini", created_at: daysAgo(2) },
  ];

  const pricing: PricingOverride[] = [
    { id: id(), provider: "openai", model: "gpt-4o", input_cost: 2.0, output_cost: 8.0 },
  ];

  const usageLogs: UsageLog[] = Array.from({ length: 120 }).map(() => {
    const p = pick(providers.filter((x) => x.connection_count > 0));
    const m = pick(MODEL_CATALOG[p.id] ?? [{ name: "default", in: 1, out: 1, ctx: 0 }]);
    const k = pick(keys);
    const pt = rand(100, 4000);
    const ct = rand(50, 2000);
    const ok = Math.random() > 0.08;
    return {
      id: id(),
      timestamp: new Date(Date.now() - rand(0, 60 * 86400000)).toISOString(),
      provider: p.id,
      model: m.name,
      api_key_id: k.id,
      api_key_name: k.name,
      status: ok ? "success" : "error",
      status_code: ok ? 200 : pick([400, 401, 429, 500]),
      prompt_tokens: pt,
      completion_tokens: ct,
      total_tokens: pt + ct,
      cost_usd: ((pt * m.in + ct * m.out) / 1_000_000),
      latency_ms: rand(120, 3500),
      rtk_enabled: Math.random() > 0.5,
      caveman_enabled: Math.random() > 0.7,
      combo_name: Math.random() > 0.5 ? pick(combos).name : undefined,
    };
  });

  // Build several quota rows per quota-eligible provider so the cards have
  // meaningful per-account breakdowns like 9router's tracker.
  const quotaProviders: Array<{ id: string; plan: "free" | "pro" | "ultra" | "enterprise"; accounts: number }> = [
    { id: "anthropic", plan: "pro", accounts: 3 },
    { id: "openai", plan: "ultra", accounts: 2 },
    { id: "google", plan: "free", accounts: 4 },
    { id: "groq", plan: "free", accounts: 2 },
    { id: "mistral", plan: "pro", accounts: 1 },
    { id: "openrouter", plan: "enterprise", accounts: 5 },
    { id: "cohere", plan: "free", accounts: 1 },
  ];
  const quotas: Quota[] = [];
  quotaProviders.forEach(({ id: pid, plan, accounts }) => {
    const conns = connections.filter((c) => c.provider === pid);
    if (conns.length === 0) return;
    for (let i = 0; i < accounts; i++) {
      const c = conns[i % conns.length];
      const lim = pick([0, 500, 1000, 5000, 25000, 100000, 500000]);
      const used = lim === 0 ? rand(100, 5000) : Math.round(lim * (Math.random() ** 1.5));
      quotas.push({
        connection_id: `${c.id}-acct-${i}`,
        provider: pid,
        connection_name: c.name,
        account_label: i === 0 ? c.name : `${pid}-${i + 1}@example.com`,
        plan,
        used,
        limit: lim,
        unit: pid === "openai" ? "tokens" : "requests",
        reset_at: daysFromNow(rand(0, 14)),
        is_active: c.is_active,
      });
    }
  });
  const xaiConn = connections.find((c) => c.provider === "xai");
  if (xaiConn) {
    quotas.push({
      connection_id: `${xaiConn.id}-info`,
      provider: "xai",
      connection_name: xaiConn.name,
      used: 0,
      limit: 0,
      reset_at: daysFromNow(30),
      is_active: xaiConn.is_active,
      message: "Quota API not exposed by this provider — view usage on the provider dashboard.",
    });
  }

  const sessions: ChatSession[] = [
    {
      id: id(),
      title: "Brainstorm landing page copy",
      provider: "anthropic",
      model: "claude-sonnet-4",
      messages: [
        { role: "user", content: "Give me 3 hero headlines for an LLM router product." },
        {
          role: "assistant",
          content:
            "Here are three options:\n\n1. **One endpoint. Every model.**\n2. **Stop juggling AI providers.**\n3. **Your LLM traffic, finally under control.**",
        },
      ],
      created_at: daysAgo(2),
      updated_at: daysAgo(1),
    },
    {
      id: id(),
      title: "Refactor auth middleware",
      provider: "openai",
      model: "gpt-4o",
      messages: [{ role: "user", content: "Show me a JWT verifier in Go." }],
      created_at: daysAgo(5),
      updated_at: daysAgo(5),
    },
  ];

  const proxyPools: ProxyPool[] = Array.from({ length: 4 }).map((_, i) => ({
    id: id(),
    name: `pool-${i + 1}`,
    protocol: pick(["http", "https", "socks5"] as const),
    host: `proxy-${i + 1}.example.com`,
    port: pick([8080, 8443, 1080]),
    username: i % 2 === 0 ? "user" : undefined,
    is_active: i !== 3,
    last_check_at: now(),
    last_check_status: i === 3 ? "timeout" : "ok",
  }));

  const tunnels: Tunnel[] = [
    { type: "cloudflare", is_enabled: false, status: "inactive" },
    { type: "tailscale", is_enabled: false, status: "inactive" },
  ];

  const mcpInstances: McpInstance[] = [
    { id: id(), name: "filesystem", command: "npx -y @modelcontextprotocol/server-filesystem /tmp", type: "stdio", status: "running", health: "healthy", tools_count: 8 },
    { id: id(), name: "github", command: "npx -y @modelcontextprotocol/server-github", type: "stdio", status: "running", health: "healthy", tools_count: 14 },
    { id: id(), name: "slack-sse", command: "https://slack.mcp/sse", type: "sse", status: "stopped", health: "unhealthy", tools_count: 0 },
  ];

  const mcpAccounts: McpAccount[] = [
    { id: id(), account: "alex@example.com", provider: "github", status: "linked", linked_to: "github" },
    { id: id(), account: "team-bot", provider: "slack", status: "unlinked" },
  ];

  const mcpTools: McpTool[] = [
    { name: "read_file", client: "filesystem", description: "Read a file from the local filesystem", parameters: { path: "string" } },
    { name: "write_file", client: "filesystem", description: "Write content to a local file", parameters: { path: "string", content: "string" } },
    { name: "list_directory", client: "filesystem", description: "List entries in a directory", parameters: { path: "string" } },
    { name: "create_issue", client: "github", description: "Open a GitHub issue", parameters: { repo: "string", title: "string", body: "string" } },
    { name: "list_pull_requests", client: "github", description: "List open pull requests", parameters: { repo: "string" } },
    { name: "merge_pr", client: "github", description: "Merge a pull request", parameters: { repo: "string", number: "number" } },
  ];

  const mcpToolGroups: McpToolGroup[] = [
    { id: id(), name: "fs-readonly", tools: ["read_file", "list_directory"] },
    { id: id(), name: "github-write", tools: ["create_issue", "merge_pr"] },
  ];

  const mitm: MitmTool[] = [
    { id: "antigravity", name: "Antigravity", enabled: false, dns_override: "antigravity.example.com", status: "inactive" },
    { id: "copilot", name: "GitHub Copilot", enabled: true, dns_override: "api.githubcopilot.com", status: "active" },
    { id: "cursor", name: "Cursor", enabled: false, dns_override: "api.cursor.sh", status: "inactive" },
    { id: "kiro", name: "Kiro", enabled: false, dns_override: "api.kiro.ai", status: "inactive" },
  ];

  const guardrails: Guardrails = {
    enabled: true,
    blocklist: ["spam", "malicious-prompt", "credit_card_full"],
    pii_redaction: true,
    pii_types: ["Email", "Phone", "SSN", "Credit Card"],
  };

  const modelLimits: ModelLimit[] = [
    { id: id(), model: "gpt-4o", max_tokens: 8000, max_requests_per_min: 200, allowed_keys: [keys[0].id], time_window_seconds: 60 },
  ];

  const prompts: PromptTemplate[] = [
    {
      id: id(),
      name: "Summarize document",
      description: "Three-bullet summary of any text.",
      system_prompt: "You are a concise summarizer. Output exactly 3 bullets.",
      user_prompt_template: "Summarize:\n\n{{document}}",
      variables: ["document"],
    },
    {
      id: id(),
      name: "Translate to language",
      user_prompt_template: "Translate the following to {{language}}:\n\n{{text}}",
      variables: ["language", "text"],
    },
  ];

  const alerts: AlertChannel[] = [
    { id: id(), name: "Ops webhook", channel_type: "webhook", config: { url: "https://hooks.example.com/ops" }, is_active: true },
    { id: id(), name: "Eng Discord", channel_type: "discord", config: { webhook_url: "https://discord.com/api/webhooks/…" }, is_active: true },
  ];

  const flags: FeatureFlag[] = [
    { id: id(), key: "adaptive_routing", enabled: true, description: "Use latency-aware combo selection" },
    { id: id(), key: "semantic_cache", enabled: true, description: "Cache by embedding similarity" },
    { id: id(), key: "websocket_chat", enabled: true, description: "Use WS for chat streaming" },
    { id: id(), key: "guardrails", enabled: true, description: "Block/redact via guardrails" },
    { id: id(), key: "pii_redaction", enabled: true, description: "Redact PII before sending" },
    { id: id(), key: "rtk", enabled: false, description: "Real-time token compression" },
    { id: id(), key: "caveman_mode", enabled: false, description: "Compress prompts aggressively" },
  ];

  const settings: Settings = {
    require_api_key: true,
    require_login: true,
    rtk_enabled: false,
    caveman_enabled: false,
    caveman_level: "lite",
    enable_request_logs: true,
    log_retention_days: 14,
    cache_enabled: true,
    cache_ttl_seconds: 3600,
    proxy_url: "",
    notify_webhook_url: "",
    notify_on_reauth: true,
    allowed_sources: ["local", "lan"],
    tunnel_dashboard_access: false,
    theme: "system",
    language: "en",
    inject_errors: false,
  };

  const audit: AuditLog[] = Array.from({ length: 25 }).map(() => ({
    id: id(),
    timestamp: new Date(Date.now() - rand(0, 30 * 86400000)).toISOString(),
    actor: pick(["admin", "alex", "sam", "system"]),
    action: pick(["login", "create_key", "delete_combo", "update_settings", "test_connection"]),
    target: pick(["api_key:prod", "combo:default-chat", "connection:openai#1", "settings"]),
    details: undefined,
  }));

  const skills: Skill[] = [
    { name: "openai-compatible", category: "Endpoint Skills", description: "Use g0router as OpenAI-compatible endpoint.", url: "https://github.com/example/skills/openai-compatible" },
    { name: "anthropic-bridge", category: "Endpoint Skills", description: "Bridge to Anthropic API shape.", url: "https://github.com/example/skills/anthropic-bridge" },
    { name: "cli-claude", category: "Entry Skills", description: "Wire Claude Code to g0router.", url: "https://github.com/example/skills/cli-claude" },
    { name: "cli-cursor", category: "Entry Skills", description: "Wire Cursor to g0router.", url: "https://github.com/example/skills/cli-cursor" },
    { name: "ext-guardrails", category: "Extension Skills", description: "Custom guardrails plugin.", url: "https://github.com/example/skills/ext-guardrails" },
  ];

  const users: User[] = [
    { id: id(), username: "admin", display_name: "Admin", role: "admin", password: "admin" },
  ];

  return {
    users,
    session_user_id: users[0].id, // start logged in for demo
    providers,
    connections,
    keys,
    vkeys,
    teams,
    combos,
    routingRules,
    models,
    aliases,
    pricing,
    usageLogs,
    quotas,
    sessions,
    proxyPools,
    tunnels,
    mcpInstances,
    mcpAccounts,
    mcpTools,
    mcpToolGroups,
    mitm,
    guardrails,
    modelLimits,
    prompts,
    alerts,
    flags,
    settings,
    audit,
    skills,
  };
}

export type Store = ReturnType<typeof seedAll>;
