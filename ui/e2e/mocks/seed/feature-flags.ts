import type { FeatureFlag } from "../../src/lib/types";

export function seedFeatureFlags(): FeatureFlag[] {
  return [
    { id: 1, key: "mcp_gateway", enabled: true, description: "Enable MCP gateway", created_at: new Date(Date.now() - 86400000 * 30).toISOString() },
    { id: 2, key: "rtk_compression", enabled: false, description: "Enable RTK compression", created_at: new Date(Date.now() - 86400000 * 20).toISOString() },
    { id: 3, key: "new_dashboard", enabled: false, description: "New React dashboard", created_at: new Date(Date.now() - 86400000 * 10).toISOString() },
  ];
}
