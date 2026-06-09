import type { Tunnel } from "../../src/lib/types";

export function seedTunnels(): Tunnel[] {
  return [
    { type: "cloudflare", is_enabled: false, url: "https://g0router-demo.trycloudflare.com", status: "inactive" },
    { type: "tailscale", is_enabled: false, url: "http://g0router.tailnet.ts.net", status: "inactive" },
  ];
}
