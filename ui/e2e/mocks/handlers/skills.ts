import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json } from "./utils";

export function registerSkillsHandlers(page: Page, store: MockStore) {
  page.route("/api/skills", async (route) => {
    if (route.request().method() === "GET") return json(route, store.skills);
    return route.continue();
  });
}
