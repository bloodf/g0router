import type { Alias } from "../../src/lib/types";

export function seedAliases(): Alias[] {
  return [
    { id: "alias-1", alias: "gpt4", provider: "openai", model: "gpt-4o" },
    { id: "alias-2", alias: "claude", provider: "anthropic", model: "claude-sonnet-4" },
    { id: "alias-3", alias: "gemini", provider: "google", model: "gemini-2.5-pro" },
  ];
}
