import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json, error } from "./utils";

export function registerAlertChannelsHandlers(page: Page, store: MockStore) {
  page.route("/api/alert-channels", async (route) => {
    const method = route.request().method();
    if (method === "GET") return json(route, Array.from(store.alertChannels.values()));
    if (method === "POST") {
      const body = await route.request().postDataJSON();
      const ac = { id: Date.now(), created_at: new Date().toISOString(), ...body };
      store.alertChannels.set(String(ac.id), ac);
      return json(route, ac);
    }
    return route.continue();
  });
  page.route(/\/api\/alert-channels\/[^/]+$/, async (route) => {
    const id = route.request().url().split("/").pop()!;
    const method = route.request().method();
    if (method === "GET") {
      const ac = store.alertChannels.get(id);
      return ac ? json(route, ac) : error(route, "Not found", 404);
    }
    if (method === "PUT") {
      const body = await route.request().postDataJSON();
      const existing = store.alertChannels.get(id);
      if (!existing) return error(route, "Not found", 404);
      const updated = { ...existing, ...body };
      store.alertChannels.set(id, updated);
      return json(route, updated);
    }
    if (method === "DELETE") {
      store.alertChannels.delete(id);
      return json(route, {});
    }
    return route.continue();
  });
  page.route(/\/api\/alert-channels\/[^/]+\/test$/, async (route) => {
    if (route.request().method() === "POST") return json(route, { ok: true, message: "Test notification sent" });
    return route.continue();
  });
}
