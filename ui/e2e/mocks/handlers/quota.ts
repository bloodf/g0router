import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json } from "./utils";

export function registerQuotaHandlers(page: Page, store: MockStore) {
  page.route("/api/quota", async (route) => {
    if (route.request().method() === "GET") return json(route, store.quotas);
    return route.continue();
  });
}
