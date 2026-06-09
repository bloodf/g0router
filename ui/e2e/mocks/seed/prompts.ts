import type { PromptTemplate } from "../../src/lib/types";

export function seedPromptTemplates(): PromptTemplate[] {
  return [
    { id: 1, name: "Code Review", system_prompt: "You are a senior code reviewer. Be concise and actionable.", models: ["gpt-4o", "claude-sonnet-4"], is_active: true, created_at: new Date(Date.now() - 86400000 * 14).toISOString() },
    { id: 2, name: "Documentation", system_prompt: "You write clear technical documentation.", models: ["gpt-4o-mini"], is_active: true, created_at: new Date(Date.now() - 86400000 * 7).toISOString() },
  ];
}
