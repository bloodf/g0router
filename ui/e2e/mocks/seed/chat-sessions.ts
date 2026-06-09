import type { ChatSession } from "../../src/lib/types";

export function seedChatSessions(): ChatSession[] {
  return [
    { id: "chat-1", title: "Python helper", model: "gpt-4o", provider: "openai", messages: [{ role: "user", content: "How do I parse JSON?" }, { role: "assistant", content: "Use JSON.parse()..." }], created_at: new Date(Date.now() - 86400000).toISOString(), updated_at: new Date(Date.now() - 3600000).toISOString() },
    { id: "chat-2", title: "Code review", model: "claude-sonnet-4", provider: "anthropic", messages: [{ role: "user", content: "Review this function" }], created_at: new Date(Date.now() - 172800000).toISOString(), updated_at: new Date(Date.now() - 172800000).toISOString() },
  ];
}
