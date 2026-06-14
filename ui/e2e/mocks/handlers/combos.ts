import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json, error } from "./utils";

export function registerCombosHandlers(page: Page, store: MockStore) {
  page.route("/api/combos", async (route) => {
    const method = route.request().method();
    if (method === "GET") return json(route, Array.from(store.combos.values()));
    if (method === "POST") {
      const body = await route.request().postDataJSON();
      // Mirror the real Go combos-admin create DTO: strategy defaults to
      // "fallback", is_active defaults true (w7-route-a §1.7 / ESC-COMBOS).
      const combo = { id: store.nextId(), strategy: "fallback", is_active: true, ...body };
      store.combos.set(combo.id, combo);
      return json(route, combo);
    }
    return route.continue();
  });
  page.route(/\/api\/combos\/[^/]+$/, async (route) => {
    const id = route.request().url().split("/").pop()!;
    const method = route.request().method();
    if (method === "GET") {
      const c = store.combos.get(id);
      return c ? json(route, c) : error(route, "Not found", 404);
    }
    if (method === "PUT") {
      const body = await route.request().postDataJSON();
      const existing = store.combos.get(id);
      if (!existing) return error(route, "Not found", 404);
      const updated = { ...existing, ...body };
      store.combos.set(id, updated);
      return json(route, updated);
    }
    if (method === "DELETE") {
      store.combos.delete(id);
      return json(route, { message: "Combo deleted successfully" });
    }
    return route.continue();
  });
}
