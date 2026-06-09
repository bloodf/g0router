import type { ApiKey } from "../../src/lib/types";

export function seedKeys(): ApiKey[] {
  return [
    { id: "key-1", name: "Default Key", prefix: "sk-g0def", full_key: "sk-g0def-1234567890abcdef", scopes: ["chat", "embeddings"], rpm_limit: 1000, tpm_limit: 1000000, daily_spend_cap: 100, is_active: true, created_at: new Date(Date.now() - 86400000 * 7).toISOString() },
    { id: "key-2", name: "Staging Key", prefix: "sk-g0stg", scopes: ["chat"], rpm_limit: 100, is_active: true, created_at: new Date(Date.now() - 86400000 * 3).toISOString() },
  ];
}
