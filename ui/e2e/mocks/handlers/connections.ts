import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json, error } from "./utils";

export function registerConnectionsHandlers(page: Page, store: MockStore) {
  page.route("/api/connections", async (route) => {
    const method = route.request().method();
    if (method === "GET") return json(route, Array.from(store.connections.values()));
    if (method === "POST") {
      const body = await route.request().postDataJSON();
      const conn = { id: store.nextId(), is_active: true, needs_reauth: false, ...body };
      store.connections.set(conn.id, conn);
      return json(route, conn);
    }
    return route.continue();
  });
  page.route(/\/api\/connections\/[^/]+$/, async (route) => {
    const id = route.request().url().split("/").pop()!;
    const method = route.request().method();
    if (method === "GET") {
      const c = store.connections.get(id);
      return c ? json(route, c) : error(route, "Not found", 404);
    }
    if (method === "PUT") {
      const body = await route.request().postDataJSON();
      const existing = store.connections.get(id);
      if (!existing) return error(route, "Not found", 404);
      const updated = { ...existing, ...body };
      store.connections.set(id, updated);
      return json(route, updated);
    }
    if (method === "DELETE") {
      store.connections.delete(id);
      return json(route, {});
    }
    return route.continue();
  });
  page.route(/\/api\/connections\/[^/]+\/test$/, async (route) => {
    if (route.request().method() === "POST") return json(route, { ok: true, latency_ms: Math.floor(Math.random() * 300) + 50 });
    return route.continue();
  });
  page.route("/api/connections/bulk-enable", async (route) => {
    if (route.request().method() === "POST") {
      for (const c of store.connections.values()) c.is_active = true;
      return json(route, {});
    }
    return route.continue();
  });
  page.route("/api/connections/bulk-disable", async (route) => {
    if (route.request().method() === "POST") {
      for (const c of store.connections.values()) c.is_active = false;
      return json(route, {});
    }
    return route.continue();
  });
}
