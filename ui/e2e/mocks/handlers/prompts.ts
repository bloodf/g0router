import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json, error } from "./utils";

export function registerPromptsHandlers(page: Page, store: MockStore) {
  page.route("/api/prompt-templates", async (route) => {
    const method = route.request().method();
    if (method === "GET") return json(route, Array.from(store.promptTemplates.values()));
    if (method === "POST") {
      const body = await route.request().postDataJSON();
      const pt = { id: Date.now(), created_at: new Date().toISOString(), ...body };
      store.promptTemplates.set(String(pt.id), pt);
      return json(route, pt);
    }
    return route.continue();
  });
  page.route(/\/api\/prompt-templates\/[^/]+$/, async (route) => {
    const id = route.request().url().split("/").pop()!;
    const method = route.request().method();
    if (method === "GET") {
      const pt = store.promptTemplates.get(id);
      return pt ? json(route, pt) : error(route, "Not found", 404);
    }
    if (method === "PUT") {
      const body = await route.request().postDataJSON();
      const existing = store.promptTemplates.get(id);
      if (!existing) return error(route, "Not found", 404);
      const updated = { ...existing, ...body };
      store.promptTemplates.set(id, updated);
      return json(route, updated);
    }
    if (method === "DELETE") {
      store.promptTemplates.delete(id);
      return json(route, {});
    }
    return route.continue();
  });
  page.route("/api/prompt-templates/test", async (route) => {
    if (route.request().method() === "POST") return json(route, { rendered: "Mock rendered prompt" });
    return route.continue();
  });
}
