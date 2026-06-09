import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json } from "./utils";

export function registerVersionHandlers(page: Page, _store: MockStore) {
  page.route("/api/version", async (route) => {
    if (route.request().method() === "GET") return json(route, { version: "0.9.0-mock", build_date: "2024-01-01" });
    return route.continue();
  });
  page.route("/healthz", async (route) => {
    if (route.request().method() === "GET") return json(route, { status: "ok" });
    return route.continue();
  });
}
