import type { Page } from "@playwright/test";
import type { MockStore } from "../store";
import { json } from "./utils";

// Mock body mirrors the real Go request-logs path (plan w6-g §1.4):
//   GET /api/usage/request-logs -> internal/admin/usage.go:139
//     (GetUsageRequestLogs, alias /api/usage/logs at routes_admin.go:92-93).
// The real Go returns recent logs (200). It formats them as pipe-delimited
// display strings (internal/usage/logs.go:9-44); the mock serves the richer
// structured UsageLog[] (the seed / UI type) so the e2e renders full columns,
// and RequestLogger is tolerant of BOTH shapes (string[] from real Go OR
// UsageLog[] from the mock) — same client-side normalization precedent as
// connections.tsx (§8). NO seed/index edit; body only.
export function registerLogsHandlers(page: Page, store: MockStore) {
  page.route(/\/api\/usage\/request-logs(\?.*)?$/, async (route) => {
    if (route.request().method() === "GET") {
      return json(route, store.usageLogs.slice(0, 200));
    }
    return route.continue();
  });
  page.route(/\/api\/usage\/logs(\?.*)?$/, async (route) => {
    if (route.request().method() === "GET") {
      return json(route, store.usageLogs.slice(0, 200));
    }
    return route.continue();
  });
}
