import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json, error } from "./utils";

export function registerProxyPoolsHandlers(page: Page, store: MockStore) {
  page.route("/api/proxy-pools", async (route) => {
    const method = route.request().method();
    if (method === "GET") return json(route, Array.from(store.proxyPools.values()));
    if (method === "POST") {
      const body = await route.request().postDataJSON();
      const pool = { id: store.nextId(), last_check_at: new Date().toISOString(), ...body };
      store.proxyPools.set(pool.id, pool);
      return json(route, pool);
    }
    return route.continue();
  });
  page.route("/api/proxy-pools/batch", async (route) => {
    if (route.request().method() === "POST") {
      const body = await route.request().postDataJSON();
      for (const item of body.items || []) {
        const pool = { id: store.nextId(), last_check_at: new Date().toISOString(), ...item };
        store.proxyPools.set(pool.id, pool);
      }
      return json(route, {});
    }
    return route.continue();
  });
  page.route(/\/api\/proxy-pools\/[^/]+$/, async (route) => {
    const id = route.request().url().split("/").pop()!;
    const method = route.request().method();
    if (method === "GET") {
      const p = store.proxyPools.get(id);
      return p ? json(route, p) : error(route, "Not found", 404);
    }
    if (method === "PUT") {
      const body = await route.request().postDataJSON();
      const existing = store.proxyPools.get(id);
      if (!existing) return error(route, "Not found", 404);
      const updated = { ...existing, ...body };
      store.proxyPools.set(id, updated);
      return json(route, updated);
    }
    if (method === "DELETE") {
      store.proxyPools.delete(id);
      return json(route, {});
    }
    return route.continue();
  });
  page.route(/\/api\/proxy-pools\/[^/]+\/test$/, async (route) => {
    if (route.request().method() === "POST") return json(route, { ok: true, latency_ms: Math.floor(Math.random() * 300) + 50 });
    return route.continue();
  });
}
