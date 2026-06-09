import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json } from "./utils";

export function registerLocaleHandlers(page: Page, store: MockStore) {
  page.route("/api/locale", async (route) => {
    const method = route.request().method();
    if (method === "GET") return json(route, store.locale);
    if (method === "PUT") {
      const body = await route.request().postDataJSON();
      store.locale = { ...store.locale, ...body };
      return json(route, store.locale);
    }
    return route.continue();
  });
}
