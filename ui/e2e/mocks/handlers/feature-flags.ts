import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json, error } from "./utils";

export function registerFeatureFlagsHandlers(page: Page, store: MockStore) {
  page.route("/api/feature-flags", async (route) => {
    if (route.request().method() === "GET") return json(route, Array.from(store.featureFlags.values()));
    return route.continue();
  });
  page.route(/\/api\/feature-flags\/[^/]+$/, async (route) => {
    const id = route.request().url().split("/").pop()!;
    const method = route.request().method();
    if (method === "GET") {
      const ff = store.featureFlags.get(id);
      return ff ? json(route, ff) : error(route, "Not found", 404);
    }
    if (method === "PUT") {
      const body = await route.request().postDataJSON();
      const existing = store.featureFlags.get(id);
      if (!existing) return error(route, "Not found", 404);
      const updated = { ...existing, ...body };
      store.featureFlags.set(id, updated);
      return json(route, updated);
    }
    return route.continue();
  });
}
