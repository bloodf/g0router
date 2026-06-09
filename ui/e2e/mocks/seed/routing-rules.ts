import type { RoutingRule } from "../../src/lib/types";

export function seedRoutingRules(): RoutingRule[] {
  return [
    { id: "rule-1", name: "Route GPT-4 to OpenAI", priority: 1, cond_field: "model", cond_operator: "equals", cond_value: "gpt-4o", target_provider: "openai", is_active: true, created_at: new Date(Date.now() - 86400000).toISOString() },
    { id: "rule-2", name: "Route Claude to Anthropic", priority: 2, cond_field: "model", cond_operator: "equals", cond_value: "claude-sonnet-4", target_provider: "anthropic", is_active: true, created_at: new Date(Date.now() - 172800000).toISOString() },
  ];
}
