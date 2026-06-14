import type { VirtualKey } from "../../src/lib/types";

// The mock seed mirrors the REAL Go virtualKeyDTO (internal/admin/virtualkeys.go:13-22):
// {id, key, name, provider_configs[], budget{limit,period,used}?, rate_limit_rpm?,
// is_active, created_at, updated_at} where provider_configs[] is
// schemas.ProviderConfig{provider, allowed_models[], key_ids[], weight?}. The frozen
// UI VirtualKey type's flat fields are display-optional, so the seed is cast to it
// (plan §1.4 / §1.6 / §8 ESC-2: mocks mirror reality; key_ids is the pinning field).
export function seedVirtualKeys(): VirtualKey[] {
  const now = Math.floor(Date.now() / 1000);
  return [
    {
      id: "vk-1",
      key: "vk-alpha-1234567890",
      name: "Team Alpha",
      provider_configs: [
        {
          provider: "openai",
          allowed_models: ["gpt-4o", "gpt-4o-mini"],
          key_ids: ["conn-1"],
        },
      ],
      budget: { limit: 500, period: "monthly", used: 127.5 },
      rate_limit_rpm: 500,
      is_active: true,
      created_at: now - 86400 * 30,
      updated_at: now - 86400 * 2,
    } as unknown as VirtualKey,
    {
      id: "vk-2",
      key: "vk-beta-0987654321",
      name: "Team Beta",
      provider_configs: [
        {
          provider: "anthropic",
          allowed_models: ["claude-sonnet-4"],
          key_ids: ["conn-2"],
        },
      ],
      budget: { limit: 200, period: "monthly", used: 45.0 },
      rate_limit_rpm: 200,
      is_active: true,
      created_at: now - 86400 * 14,
      updated_at: now - 86400,
    } as unknown as VirtualKey,
  ];
}
