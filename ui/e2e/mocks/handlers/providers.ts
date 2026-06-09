import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json, error } from "./utils";

export function registerProvidersHandlers(page: Page, store: MockStore) {
  page.route("/api/providers", async (route) => {
    if (route.request().method() === "GET") {
      const list = Array.from(store.providers.values()).map((p) => ({
        ...p,
        connection_count: Array.from(store.connections.values()).filter((c) => c.provider === p.id).length,
      }));
      return json(route, list);
    }
    return route.continue();
  });
  page.route(/\/api\/providers\/[^/]+$/, async (route) => {
    if (route.request().method() === "GET") {
      const id = route.request().url().split("/").pop()!;
      const p = store.providers.get(id);
      if (!p) return error(route, "Provider not found", 404);
      return json(route, { ...p, connection_count: Array.from(store.connections.values()).filter((c) => c.provider === id).length });
    }
    return route.continue();
  });
  page.route(/\/api\/providers\/[^/]+\/connections$/, async (route) => {
    if (route.request().method() === "GET") {
      const parts = route.request().url().split("/");
      const id = parts[parts.length - 2];
      return json(route, Array.from(store.connections.values()).filter((c) => c.provider === id));
    }
    return route.continue();
  });
  page.route(/\/api\/providers\/[^/]+\/models$/, async (route) => {
    if (route.request().method() === "GET") {
      const parts = route.request().url().split("/");
      const id = parts[parts.length - 2];
      return json(route, Array.from(store.models.values()).filter((m) => m.provider === id));
    }
    return route.continue();
  });
  page.route(/\/api\/providers\/[^/]+\/suggested-models$/, async (route) => {
    if (route.request().method() === "GET") {
      const parts = route.request().url().split("/");
      const id = parts[parts.length - 2];
      const list = Array.from(store.models.values()).filter((m) => m.provider === id).slice(0, 5);
      return json(route, list.map((m) => m.id));
    }
    return route.continue();
  });
  page.route("/api/providers/test-batch", async (route) => {
    if (route.request().method() === "POST") {
      const results = Array.from(store.providers.values()).map((p) => ({ provider: p.id, ok: p.status === "active", latency_ms: Math.floor(Math.random() * 500) + 50 }));
      return json(route, { results });
    }
    return route.continue();
  });
}
