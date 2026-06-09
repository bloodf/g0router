import type { ConsoleLogEntry } from "../../src/lib/types";

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
