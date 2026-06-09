import type { Connection } from "../../src/lib/types";

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
