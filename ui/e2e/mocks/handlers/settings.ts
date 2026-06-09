import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json } from "./utils";

export function registerSettingsHandlers(page: Page, store: MockStore) {
  page.route("/api/settings", async (route) => {
    const method = route.request().method();
    if (method === "GET") return json(route, store.settings);
    if (method === "PUT") {
      const body = await route.request().postDataJSON();
      store.settings = { ...store.settings, ...body };
      return json(route, store.settings);
    }
    return route.continue();
  });
}
