import type { UsageLog, Quota } from "../../src/lib/types";

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

export function seedQuota(): Quota[] {
  return [
    { connection_id: "conn-1", provider: "openai", connection_name: "OpenAI Prod", account_label: "org-123", plan: "pro", used: 45000, limit: 100000, unit: "tokens", reset_at: new Date(Date.now() + 86400000).toISOString(), is_active: true },
    { connection_id: "conn-3", provider: "anthropic", connection_name: "Anthropic Main", account_label: "team-alpha", plan: "pro", used: 23000, limit: 50000, unit: "tokens", reset_at: new Date(Date.now() + 86400000).toISOString(), is_active: true },
    { connection_id: "conn-4", provider: "google", connection_name: "Google AI", plan: "free", used: 12000, limit: 15000, unit: "tokens", reset_at: new Date(Date.now() + 86400000).toISOString(), is_active: true },
    { connection_id: "conn-5", provider: "groq", connection_name: "Groq Fast", plan: "free", used: 8000, limit: 0, unit: "tokens", reset_at: new Date(Date.now() + 86400000).toISOString(), is_active: true },
  ];
}
