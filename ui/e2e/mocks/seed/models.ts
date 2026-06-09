import type { Model } from "../../src/lib/types";

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
