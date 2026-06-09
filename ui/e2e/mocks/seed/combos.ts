import type { Combo } from "../../src/lib/types";

export function seedCombos(): Combo[] {
  return [
    { id: "combo-1", name: "Fast + Cheap", strategy: "fallback", steps: [{ provider: "groq", model: "llama-3-70b" }, { provider: "openai", model: "gpt-4o-mini" }], is_active: true },
    { id: "combo-2", name: "Best Quality", strategy: "fallback", steps: [{ provider: "openai", model: "gpt-4o" }, { provider: "anthropic", model: "claude-sonnet-4" }], is_active: true },
  ];
}
