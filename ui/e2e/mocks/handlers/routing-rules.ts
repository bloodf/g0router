import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json, error } from "./utils";

export function registerRoutingRulesHandlers(page: Page, store: MockStore) {
  page.route("/api/routing-rules", async (route) => {
    const method = route.request().method();
    if (method === "GET") return json(route, Array.from(store.routingRules.values()));
    if (method === "POST") {
      const body = await route.request().postDataJSON();
      const rule = { id: store.nextId(), is_active: true, created_at: new Date().toISOString(), ...body };
      store.routingRules.set(rule.id, rule);
      return json(route, rule);
    }
    return route.continue();
  });
  page.route(/\/api\/routing-rules\/[^/]+$/, async (route) => {
    const id = route.request().url().split("/").pop()!;
    const method = route.request().method();
    if (method === "GET") {
      const r = store.routingRules.get(id);
      return r ? json(route, r) : error(route, "Not found", 404);
    }
    if (method === "PUT") {
      const body = await route.request().postDataJSON();
      const existing = store.routingRules.get(id);
      if (!existing) return error(route, "Not found", 404);
      const updated = { ...existing, ...body };
      store.routingRules.set(id, updated);
      return json(route, updated);
    }
    if (method === "DELETE") {
      store.routingRules.delete(id);
      return json(route, { message: "Routing rule deleted successfully" });
    }
    return route.continue();
  });
}
