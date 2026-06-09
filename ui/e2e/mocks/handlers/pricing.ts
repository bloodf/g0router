import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json, error } from "./utils";

export function registerPricingHandlers(page: Page, store: MockStore) {
  page.route("/api/pricing", async (route) => {
    const method = route.request().method();
    if (method === "GET") return json(route, Array.from(store.pricing.values()));
    if (method === "POST") {
      const body = await route.request().postDataJSON();
      const price = { id: store.nextId(), ...body };
      store.pricing.set(price.id, price);
      return json(route, price);
    }
    return route.continue();
  });
  page.route(/\/api\/pricing\/[^/]+\/[^/]+$/, async (route) => {
    const parts = route.request().url().split("/");
    const provider = parts[parts.length - 2];
    const model = parts[parts.length - 1];
    const method = route.request().method();
    if (method === "GET") {
      const p = Array.from(store.pricing.values()).find((pr) => pr.provider === provider && pr.model === model);
      return p ? json(route, p) : error(route, "Not found", 404);
    }
    if (method === "PUT") {
      const body = await route.request().postDataJSON();
      const existing = Array.from(store.pricing.values()).find((pr) => pr.provider === provider && pr.model === model);
      if (!existing) return error(route, "Not found", 404);
      const updated = { ...existing, ...body };
      store.pricing.set(updated.id, updated);
      return json(route, updated);
    }
    if (method === "DELETE") {
      const existing = Array.from(store.pricing.values()).find((pr) => pr.provider === provider && pr.model === model);
      if (existing) store.pricing.delete(existing.id);
      return json(route, {});
    }
    return route.continue();
  });
}
