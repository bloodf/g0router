import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json, error } from "./utils";

export function registerKeysHandlers(page: Page, store: MockStore) {
  page.route("/api/keys", async (route) => {
    const method = route.request().method();
    if (method === "GET") return json(route, Array.from(store.keys.values()));
    if (method === "POST") {
      const body = await route.request().postDataJSON();
      const key = { id: store.nextId(), prefix: `sk-${Math.random().toString(36).slice(2, 8)}`, full_key: `sk-${Math.random().toString(36).slice(2, 12)}`, scopes: [], is_active: true, created_at: new Date().toISOString(), ...body };
      store.keys.set(key.id, key);
      return json(route, key);
    }
    return route.continue();
  });
  page.route(/\/api\/keys\/[^/]+$/, async (route) => {
    const id = route.request().url().split("/").pop()!;
    const method = route.request().method();
    if (method === "GET") {
      const k = store.keys.get(id);
      return k ? json(route, k) : error(route, "Not found", 404);
    }
    if (method === "PUT") {
      const body = await route.request().postDataJSON();
      const existing = store.keys.get(id);
      if (!existing) return error(route, "Not found", 404);
      const updated = { ...existing, ...body };
      store.keys.set(id, updated);
      return json(route, updated);
    }
    if (method === "DELETE") {
      store.keys.delete(id);
      return json(route, {});
    }
    return route.continue();
  });
  page.route(/\/api\/keys\/[^/]+\/regenerate$/, async (route) => {
    if (route.request().method() === "POST") {
      const id = route.request().url().split("/")[3];
      const existing = store.keys.get(id);
      if (!existing) return error(route, "Not found", 404);
      existing.full_key = `sk-${Math.random().toString(36).slice(2, 12)}`;
      return json(route, existing);
    }
    return route.continue();
  });
}
