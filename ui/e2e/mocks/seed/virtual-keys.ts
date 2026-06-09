import type { VirtualKey } from "../../src/lib/types";

export function seedVirtualKeys(): VirtualKey[] {
  return [
    { id: "vk-1", name: "Team Alpha", prefix: "vk-alpha", budget_usd: 500, budget_used_usd: 127.5, budget_period: "monthly", rate_limit_rpm: 500, is_active: true },
    { id: "vk-2", name: "Team Beta", prefix: "vk-beta", budget_usd: 200, budget_used_usd: 45.0, budget_period: "monthly", rate_limit_rpm: 200, is_active: true },
  ];
}
