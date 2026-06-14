import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json, error } from "./utils";

export function registerAliasesHandlers(page: Page, store: MockStore) {
  page.route("/api/aliases", async (route) => {
    const method = route.request().method();
    if (method === "GET") return json(route, Array.from(store.aliases.values()));
    if (method === "POST") {
      const body = await route.request().postDataJSON();
      const alias = { id: store.nextId(), ...body };
      store.aliases.set(alias.id, alias);
      return json(route, alias);
    }
    return route.continue();
  });
  page.route(/\/api\/aliases\/[^/]+$/, async (route) => {
    const id = route.request().url().split("/").pop()!;
    const method = route.request().method();
    if (method === "GET") {
      const a = store.aliases.get(id);
      return a ? json(route, a) : error(route, "Not found", 404);
    }
    if (method === "PUT") {
      const body = await route.request().postDataJSON();
      const existing = store.aliases.get(id);
      if (!existing) return error(route, "Not found", 404);
      const updated = { ...existing, ...body };
      store.aliases.set(id, updated);
      return json(route, updated);
    }
    if (method === "DELETE") {
      store.aliases.delete(id);
      return json(route, { message: "Alias deleted successfully" });
    }
    return route.continue();
  });
}
