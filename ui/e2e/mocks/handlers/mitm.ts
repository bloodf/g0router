import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json, error } from "./utils";

export function registerMitmHandlers(page: Page, store: MockStore) {
  page.route("/api/mitm/status", async (route) => {
    if (route.request().method() === "GET") return json(route, { enabled: store.mitmEnabled, tools: store.mitmTools });
    return route.continue();
  });
  page.route("/api/mitm/toggle", async (route) => {
    if (route.request().method() === "POST") {
      store.mitmEnabled = !store.mitmEnabled;
      return json(route, { enabled: store.mitmEnabled });
    }
    return route.continue();
  });
  page.route("/api/mitm/ca-cert", async (route) => {
    if (route.request().method() === "GET") {
      return route.fulfill({
        status: 200,
        headers: { "Content-Type": "application/x-pem-file" },
        body: "-----BEGIN CERTIFICATE-----\nMIIBkTCB+wIJAKHBfpE\n-----END CERTIFICATE-----",
      });
    }
    return route.continue();
  });
  page.route(/\/api\/mitm\/tools\/[^/]+$/, async (route) => {
    if (route.request().method() === "POST") {
      const id = route.request().url().split("/").pop()!;
      const tool = store.mitmTools.find((t) => t.id === id);
      if (!tool) return error(route, "Tool not found", 404);
      tool.enabled = !tool.enabled;
      tool.status = tool.enabled ? "active" : "inactive";
      return json(route, tool);
    }
    return route.continue();
  });
}
