import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json, error } from "./utils";

export function registerTeamsHandlers(page: Page, store: MockStore) {
  page.route("/api/teams", async (route) => {
    const method = route.request().method();
    if (method === "GET") return json(route, Array.from(store.teams.values()));
    if (method === "POST") {
      const body = await route.request().postDataJSON();
      // Mirror the real Go teamDTO: 6 canonical fields, budget_used_usd defaults 0.
      const team = { id: store.nextId(), budget_used_usd: 0, ...body };
      store.teams.set(team.id, team);
      return json(route, team);
    }
    return route.continue();
  });
  page.route(/\/api\/teams\/[^/]+$/, async (route) => {
    const id = route.request().url().split("/").pop()!;
    const method = route.request().method();
    if (method === "GET") {
      const t = store.teams.get(id);
      return t ? json(route, t) : error(route, "Not found", 404);
    }
    if (method === "PUT") {
      const body = await route.request().postDataJSON();
      const existing = store.teams.get(id);
      if (!existing) return error(route, "Not found", 404);
      const updated = { ...existing, ...body };
      store.teams.set(id, updated);
      return json(route, updated);
    }
    if (method === "DELETE") {
      store.teams.delete(id);
      return json(route, {});
    }
    return route.continue();
  });
}
