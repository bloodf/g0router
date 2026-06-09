import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json, error } from "./utils";

export function registerVirtualKeysHandlers(page: Page, store: MockStore) {
  page.route("/api/virtual-keys", async (route) => {
    const method = route.request().method();
    if (method === "GET") return json(route, Array.from(store.virtualKeys.values()));
    if (method === "POST") {
      const body = await route.request().postDataJSON();
      const vk = { id: store.nextId(), prefix: `vk-${Math.random().toString(36).slice(2, 6)}`, budget_used_usd: 0, is_active: true, ...body };
      store.virtualKeys.set(vk.id, vk);
      return json(route, vk);
    }
    return route.continue();
  });
  page.route(/\/api\/virtual-keys\/[^/]+$/, async (route) => {
    const id = route.request().url().split("/").pop()!;
    const method = route.request().method();
    if (method === "GET") {
      const vk = store.virtualKeys.get(id);
      return vk ? json(route, vk) : error(route, "Not found", 404);
    }
    if (method === "PUT") {
      const body = await route.request().postDataJSON();
      const existing = store.virtualKeys.get(id);
      if (!existing) return error(route, "Not found", 404);
      const updated = { ...existing, ...body };
      store.virtualKeys.set(id, updated);
      return json(route, updated);
    }
    if (method === "DELETE") {
      store.virtualKeys.delete(id);
      return json(route, {});
    }
    return route.continue();
  });
}
