import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json } from "./utils";

export function registerAuditHandlers(page: Page, store: MockStore) {
  page.route("/api/audit", async (route) => {
    const method = route.request().method();
    if (method === "GET") {
      const url = new URL(route.request().url());
      const limit = parseInt(url.searchParams.get("limit") || "100", 10);
      return json(route, { items: store.auditLogs.slice(0, limit), total: store.auditLogs.length });
    }
    if (method === "POST") {
      const body = await route.request().postDataJSON();
      const log = { id: store.nextId(), timestamp: new Date().toISOString(), actor: store.auth.username || "admin", ...body };
      store.auditLogs.unshift(log);
      return json(route, log);
    }
    return route.continue();
  });
}
