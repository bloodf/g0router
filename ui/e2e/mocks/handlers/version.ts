import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json } from "./utils";

export function registerVersionHandlers(page: Page, _store: MockStore) {
  page.route("/api/version", async (route) => {
    if (route.request().method() === "GET")
      return json(route, {
        version: "0.9.0-mock",
        build_date: "2024-01-01",
        update_available: true,
        latest_version: "v9.9.9",
      });
    return route.continue();
  });
  page.route("/api/version/shutdown", async (route) => {
    if (route.request().method() === "POST") return json(route, { ok: true });
    return route.continue();
  });
  page.route("/api/version/changelog", async (route) => {
    if (route.request().method() === "GET")
      return json(route, {
        changelog: "# Changelog\n\n## v9.9.9\n- Mock release notes for e2e.\n",
      });
    return route.continue();
  });
  page.route("/api/version/donate", async (route) => {
    if (route.request().method() === "GET")
      return json(route, {
        title: "Support g0router",
        message: "Donate to support development.",
        links: [{ label: "GitHub Sponsors", url: "https://example.com/donate" }],
      });
    return route.continue();
  });
  page.route("/healthz", async (route) => {
    if (route.request().method() === "GET") return json(route, { status: "ok" });
    return route.continue();
  });
}
