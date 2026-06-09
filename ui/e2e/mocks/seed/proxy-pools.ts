import type { ProxyPool } from "../../src/lib/types";

export function seedProxyPools(): ProxyPool[] {
  return [
    {
      id: "proxy-1",
      name: "US East",
      protocol: "https",
      host: "us-east.proxy.example.com",
      port: 8080,
      username: "user1",
      is_active: true,
      last_check_at: new Date(Date.now() - 3600000).toISOString(),
      last_check_status: "ok",
    },
  ];
}
