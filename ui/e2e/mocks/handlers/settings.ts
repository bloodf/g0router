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
  // DB-info panel (PAR-UI-101) — mock-only; no Go endpoint today (plan §1.4/§8 ESC-3).
  page.route("/api/settings/database", async (route) => {
    if (route.request().method() === "GET")
      return json(route, {
        path: "/var/lib/g0router/g0router.db",
        size_bytes: 1048576,
        tables: [
          { name: "settings", rows: 14 },
          { name: "request_log", rows: 2048 },
          { name: "api_keys", rows: 3 },
        ],
      });
    return route.continue();
  });
}
