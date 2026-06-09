import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json } from "./utils";

export function registerGuardrailsHandlers(page: Page, store: MockStore) {
  page.route("/api/guardrails", async (route) => {
    const method = route.request().method();
    if (method === "GET") return json(route, store.guardrails);
    if (method === "PUT") {
      const body = await route.request().postDataJSON();
      store.guardrails = { ...store.guardrails, ...body };
      return json(route, store.guardrails);
    }
    return route.continue();
  });
  page.route("/api/guardrails/test", async (route) => {
    if (route.request().method() === "POST") {
      const body = await route.request().postDataJSON();
      const prompt = body.prompt || "";
      const blocked = store.guardrails.guardrails_enabled && store.guardrails.guardrails_blocklist.some((w) => prompt.toLowerCase().includes(w.toLowerCase()));
      return json(route, { blocked, reason: blocked ? "Contains blocked term" : undefined });
    }
    return route.continue();
  });
}
