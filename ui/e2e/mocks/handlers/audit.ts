import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json } from "./utils";

export function registerAuditHandlers(page: Page, store: MockStore) {
  // Mirrors the real Go read-only public surface: GET /api/audit?limit=N →
  // { data:{ items, total } }. Audit writes are internal-only (server-side),
  // so the mock exposes no POST.
  page.route("/api/audit", async (route) => {
    const method = route.request().method();
    if (method === "GET") {
      const url = new URL(route.request().url());
      const limit = parseInt(url.searchParams.get("limit") || "100", 10);
      return json(route, { items: store.auditLogs.slice(0, limit), total: store.auditLogs.length });
    }
    return route.continue();
  });
}
