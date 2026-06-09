import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json, error } from "./utils";

export function registerTunnelsHandlers(page: Page, store: MockStore) {
  page.route("/api/tunnels", async (route) => {
    if (route.request().method() === "GET") return json(route, Array.from(store.tunnels.values()));
    return route.continue();
  });
  page.route("/api/tunnels/health", async (route) => {
    if (route.request().method() === "GET") return json(route, { healthy: true });
    return route.continue();
  });
  page.route(/\/api\/tunnels\/[^/]+$/, async (route) => {
    const type = route.request().url().split("/").pop() as "cloudflare" | "tailscale";
    const method = route.request().method();
    if (method === "POST") {
      const t = store.tunnels.get(type);
      if (t) { t.is_enabled = true; t.status = "active"; }
      return json(route, t || {});
    }
    if (method === "DELETE") {
      const t = store.tunnels.get(type);
      if (t) { t.is_enabled = false; t.status = "inactive"; }
      return json(route, {});
    }
    return route.continue();
  });
}
