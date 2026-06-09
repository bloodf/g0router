import type { ModelLimit } from "../../src/lib/types";

export function seedModelLimits(): ModelLimit[] {
  return [
    { id: 1, model: "gpt-4o", max_tokens: 128000, max_rpm: 1000, allowed_key_ids: ["key-1"], created_at: new Date(Date.now() - 86400000).toISOString() },
    { id: 2, model: "claude-sonnet-4", max_tokens: 200000, max_rpm: 500, allowed_key_ids: ["key-1", "key-2"], created_at: new Date(Date.now() - 86400000).toISOString() },
  ];
}
