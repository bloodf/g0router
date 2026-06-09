import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json } from "./utils";

export function registerUsageHandlers(page: Page, store: MockStore) {
  page.route("/api/usage/summary", async (route) => {
    if (route.request().method() === "GET") {
      return json(route, {
        total_requests: store.usageLogs.length,
        total_tokens: store.usageLogs.reduce((s, l) => s + l.total_tokens, 0),
        total_cost: store.usageLogs.reduce((s, l) => s + l.cost_usd, 0),
        avg_latency_ms: Math.floor(store.usageLogs.reduce((s, l) => s + l.latency_ms, 0) / Math.max(store.usageLogs.length, 1)),
      });
    }
    return route.continue();
  });
  page.route("/api/usage/chart", async (route) => {
    if (route.request().method() === "GET") {
      const buckets = Array.from({ length: 7 }, (_, i) => new Date(Date.now() - (6 - i) * 86400000).toISOString().slice(0, 10));
      return json(route, {
        buckets,
        tokens_input: buckets.map(() => Math.floor(Math.random() * 50000)),
        tokens_output: buckets.map(() => Math.floor(Math.random() * 20000)),
        costs: buckets.map(() => Math.random() * 10),
        requests: buckets.map(() => Math.floor(Math.random() * 500)),
      });
    }
    return route.continue();
  });
  page.route("/api/usage", async (route) => {
    if (route.request().method() === "GET") {
      const url = new URL(route.request().url());
      const limit = parseInt(url.searchParams.get("limit") || "100", 10);
      return json(route, { items: store.usageLogs.slice(0, limit), total: store.usageLogs.length });
    }
    return route.continue();
  });
}
