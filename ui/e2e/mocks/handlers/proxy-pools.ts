import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json, error } from "./utils";

// toPoolDTO mirrors the real Go proxyPoolDTO: the 9 canonical snake_case fields
// plus password_set (the password is NEVER echoed). Defaults match the Go
// CreateProxyPool path: is_active defaults true, protocol defaults "http",
// last_check_status/last_check_at start empty (a fresh pool has not been tested).
function toPoolDTO(id: string, body: Record<string, unknown>) {
  const { password, ...rest } = body as Record<string, unknown>;
  return {
    id,
    name: (rest.name as string) ?? "",
    protocol: (rest.protocol as string) ?? "http",
    host: (rest.host as string) ?? "",
    port: (rest.port as number) ?? 0,
    username: (rest.username as string) ?? "",
    password_set: typeof password === "string" && password.length > 0,
    is_active: rest.is_active === undefined ? true : (rest.is_active as boolean),
    last_check_status: (rest.last_check_status as string) ?? "",
    last_check_at: (rest.last_check_at as string) ?? "",
  };
}

export function registerProxyPoolsHandlers(page: Page, store: MockStore) {
  page.route("/api/proxy-pools", async (route) => {
    const method = route.request().method();
    if (method === "GET") return json(route, Array.from(store.proxyPools.values()));
    if (method === "POST") {
      const body = await route.request().postDataJSON();
      const pool = toPoolDTO(store.nextId(), body);
      store.proxyPools.set(pool.id, pool);
      return json(route, pool, 201);
    }
    return route.continue();
  });
  page.route("/api/proxy-pools/batch", async (route) => {
    if (route.request().method() === "POST") {
      const body = await route.request().postDataJSON();
      let created = 0;
      for (const item of body.items || []) {
        const pool = toPoolDTO(store.nextId(), item);
        store.proxyPools.set(pool.id, pool);
        created++;
      }
      return json(route, { created });
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
      const updated = { ...existing, ...toPoolDTO(id, body) };
      store.proxyPools.set(id, updated);
      return json(route, updated);
    }
    if (method === "DELETE") {
      store.proxyPools.delete(id);
      return json(route, { message: "Proxy pool deleted successfully" });
    }
    return route.continue();
  });
  page.route(/\/api\/proxy-pools\/[^/]+\/test$/, async (route) => {
    if (route.request().method() === "POST") {
      return json(route, { ok: true, latency_ms: Math.floor(Math.random() * 300) + 50, status: "ok" });
    }
    return route.continue();
  });
}
