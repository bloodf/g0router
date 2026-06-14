import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json } from "./utils";

// Mock bodies mirror the real Go usage API (plan w6-g §1.4):
//   GET /api/usage/stats?period=  -> internal/admin/usage.go:101 (GetUsageStats)
//     returns the usage.Stats map (stats.go:26-41): total_requests,
//     total_prompt_tokens, total_completion_tokens, total_cost, by_provider,
//     by_model, active_requests, recent_requests, pending, error_provider.
//   GET /api/usage/chart?period=  -> usage.go:120 (GetUsageChart) (path AGREES).
//   GET /api/usage/request-details -> usage.go:145 (paginated {data,pagination}).
export function registerUsageHandlers(page: Page, store: MockStore) {
  page.route("/api/usage/stats", async (route) => {
    if (route.request().method() === "GET") {
      const logs = store.usageLogs;
      const byProvider: Record<string, { requests: number; prompt_tokens: number; completion_tokens: number; cost: number }> = {};
      const byModel: Record<string, { requests: number; prompt_tokens: number; completion_tokens: number; cost: number; raw_model: string; provider: string; last_used: string }> = {};
      for (const l of logs) {
        const p = (byProvider[l.provider] ??= { requests: 0, prompt_tokens: 0, completion_tokens: 0, cost: 0 });
        p.requests += 1;
        p.prompt_tokens += l.prompt_tokens;
        p.completion_tokens += l.completion_tokens;
        p.cost += l.cost_usd;
        const mk = `${l.provider}/${l.model}`;
        const m = (byModel[mk] ??= { requests: 0, prompt_tokens: 0, completion_tokens: 0, cost: 0, raw_model: l.model, provider: l.provider, last_used: l.timestamp });
        m.requests += 1;
        m.prompt_tokens += l.prompt_tokens;
        m.completion_tokens += l.completion_tokens;
        m.cost += l.cost_usd;
      }
      return json(route, {
        total_requests: logs.length,
        total_prompt_tokens: logs.reduce((s, l) => s + l.prompt_tokens, 0),
        total_completion_tokens: logs.reduce((s, l) => s + l.completion_tokens, 0),
        total_cost: logs.reduce((s, l) => s + l.cost_usd, 0),
        by_provider: byProvider,
        by_model: byModel,
        by_account: {},
        by_api_key: {},
        by_endpoint: {},
        last_10_minutes: [],
        pending: {},
        active_requests: [],
        recent_requests: logs.slice(0, 10).map((l) => ({
          timestamp: l.timestamp,
          model: l.model,
          provider: l.provider,
          prompt_tokens: l.prompt_tokens,
          completion_tokens: l.completion_tokens,
          status: l.status,
        })),
        error_provider: "",
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
  page.route(/\/api\/usage\/request-details(\?.*)?$/, async (route) => {
    if (route.request().method() === "GET") {
      const url = new URL(route.request().url());
      const pageNum = parseInt(url.searchParams.get("page") || "1", 10);
      const pageSize = parseInt(url.searchParams.get("pageSize") || "20", 10);
      const start = (pageNum - 1) * pageSize;
      const rows = store.usageLogs.slice(start, start + pageSize);
      return json(route, {
        data: rows,
        pagination: {
          page: pageNum,
          page_size: pageSize,
          total: store.usageLogs.length,
          total_pages: Math.max(1, Math.ceil(store.usageLogs.length / pageSize)),
        },
      });
    }
    return route.continue();
  });
}
