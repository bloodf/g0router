import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json, error } from "./utils";

export function registerModelLimitsHandlers(page: Page, store: MockStore) {
  page.route("/api/model-limits", async (route) => {
    const method = route.request().method();
    if (method === "GET") return json(route, Array.from(store.modelLimits.values()));
    if (method === "POST") {
      const body = await route.request().postDataJSON();
      const ml = { id: Date.now(), created_at: new Date().toISOString(), ...body };
      store.modelLimits.set(String(ml.id), ml);
      return json(route, ml);
    }
    return route.continue();
  });
  page.route(/\/api\/model-limits\/[^/]+$/, async (route) => {
    const id = route.request().url().split("/").pop()!;
    const method = route.request().method();
    if (method === "GET") {
      const ml = store.modelLimits.get(id);
      return ml ? json(route, ml) : error(route, "Not found", 404);
    }
    if (method === "PUT") {
      const body = await route.request().postDataJSON();
      const existing = store.modelLimits.get(id);
      if (!existing) return error(route, "Not found", 404);
      const updated = { ...existing, ...body };
      store.modelLimits.set(id, updated);
      return json(route, updated);
    }
    if (method === "DELETE") {
      store.modelLimits.delete(id);
      return json(route, {});
    }
    return route.continue();
  });
}
