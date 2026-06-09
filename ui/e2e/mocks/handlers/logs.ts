import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json } from "./utils";

export function registerLogsHandlers(page: Page, store: MockStore) {
  page.route("/api/logs", async (route) => {
    if (route.request().method() === "GET") {
      const url = new URL(route.request().url());
      const limit = parseInt(url.searchParams.get("limit") || "100", 10);
      return json(route, { items: store.usageLogs.slice(0, limit), total: store.usageLogs.length });
    }
    return route.continue();
  });
}
